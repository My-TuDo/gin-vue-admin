package jxc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SaleService 销售单：业务规则 + 状态机，库存动作全部委托给 LedgerService。
//
// 与旧实现最重要的差别：**开单即预留**（Reserve），出库时才转为账本出库；
// 取消时释放预留。这样"可售量"永远等于 结存 - 有效预留，
// 而不是靠一个会被遗忘的 lock_quantity 计数器。
type SaleService struct {
	ledger *LedgerService
}

func NewSaleService() *SaleService {
	return &SaleService{ledger: new(LedgerService)}
}

// Create 开单（草稿 + 预留库存）。预留失败（可售不足）则整单不成立。
func (s *SaleService) Create(ctx context.Context, order *jxc.SaleOrder) (*jxc.SaleOrder, error) {
	if len(order.Items) == 0 {
		return nil, errors.New("销售单至少需要一条明细")
	}
	if order.WarehouseID == 0 {
		return nil, errors.New("请选择出库仓库")
	}
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureWarehouse(db, order.WarehouseID); err != nil {
		return nil, err
	}
	total, err := normalizeSaleItems(db, order.Items)
	if err != nil {
		return nil, err
	}
	order.TotalAmount = total
	order.Status = jxc.SaleStatusDraft

	err = db.Transaction(func(tx *gorm.DB) error {
		for attempt := 0; attempt < 3; attempt++ {
			no, e := nextOrderNo(tx, "sale_order", "order_no", "SO")
			if e != nil {
				return e
			}
			order.OrderNo = no
			e = tx.Create(order).Error
			if e == nil {
				break
			}
			if !isDuplicateKeyErr(e) {
				return e
			}
			order.ID = 0
			order.Items = nil
			if attempt == 2 {
				return errors.New("单号生成冲突，请重试")
			}
		}
		// 逐行预留：任何一行预留失败都会回滚整单
		for _, it := range order.Items {
			if err := s.ledger.Reserve(ctx, tx, MoveInput{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID, Qty: it.Qty,
				SourceType: jxc.ReserveSourceSaleOrder, SourceID: order.ID, SourceLineID: it.ID,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// Ship 出库（草稿 → 已出库）：产生负向账本行，并把该行的成本回填到明细上。
//
// 为什么要把成本回填到 sale_item：毛利 = 售价 - 出库成本。
// 账本行里已经冻结了成本，这里只是让明细页能直接显示，
// 避免每次查毛利都要 join 账本。
func (s *SaleService) Ship(ctx context.Context, id uint, operator string, occurredAt time.Time) error {
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.SaleOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return errors.New("销售单不存在")
		}
		if order.Status != jxc.SaleStatusDraft {
			return fmt.Errorf("仅草稿状态的销售单可以出库（当前状态 %d）", order.Status)
		}
		var items []jxc.SaleItem
		if err := tx.Where("sale_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("销售单没有明细")
		}

		// 先把本单的预留转出库，再记账。
		// 顺序不能反：Post 的可售校验会把"仍在活跃状态的预留"算作占用，
		// 若先记账，卖掉最后一件时会出现 after.Qty - reserved < 0 的假失败
		// （自己占着自己的库存）。这个 bug 由"卖光库存"的用例抓到。
		for _, it := range items {
			if err := s.ledger.ReleaseReservation(ctx, tx, jxc.ReserveSourceSaleOrder, order.ID, it.ID, jxc.ReservationConsumed); err != nil {
				return err
			}
		}

		moves := make([]MoveInput, 0, len(items))
		for _, it := range items {
			moves = append(moves, MoveInput{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID,
				Qty:        -it.Qty, // 出库：负数
				SourceType: jxc.MoveSourceSaleOut, SourceID: order.ID, SourceLineID: it.ID,
				OccurredAt: occurredAt, Operator: operator,
			})
		}
		if err := s.ledger.Post(ctx, tx, moves); err != nil {
			return err
		}

		// 回填明细成本（从刚写入的账本行取，保证与账本一致）
		var costTotal money.Cents
		for _, it := range items {
			var mv jxc.StockMove
			if err := tx.Where("source_type = ? AND source_id = ? AND source_line_id = ?",
				jxc.MoveSourceSaleOut, order.ID, it.ID).First(&mv).Error; err != nil {
				return err
			}
			lineCost := money.FromYuan(mv.Amount).Neg()
			costTotal = costTotal.Add(lineCost)
			if err := tx.Model(&jxc.SaleItem{}).Where("id = ?", it.ID).
				Update("cost_amt", lineCost.Yuan()).Error; err != nil {
				return err
			}
		}

		ok, err := advanceDocStatus(tx, &jxc.SaleOrder{}, id, jxc.SaleStatusDraft, jxc.SaleStatusShipped)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("单据状态已变更，请刷新后重试")
		}
		return tx.Model(&jxc.SaleOrder{}).Where("id = ?", id).
			Updates(map[string]interface{}{"posted_at": occurredAt, "cost_amount": costTotal.Yuan()}).Error
	})
}

// Cancel 取消（仅草稿）：释放预留，不动库存。
func (s *SaleService) Cancel(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.SaleOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return errors.New("销售单不存在")
		}
		if order.Status != jxc.SaleStatusDraft {
			return errors.New("仅草稿状态的销售单可以取消；已出库请开具退货单")
		}
		var items []jxc.SaleItem
		if err := tx.Where("sale_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		for _, it := range items {
			if err := s.ledger.ReleaseReservation(ctx, tx, jxc.ReserveSourceSaleOrder, order.ID, it.ID, jxc.ReservationReleased); err != nil {
				return err
			}
		}
		return tx.Model(&jxc.SaleOrder{}).Where("id = ?", id).Update("status", jxc.SaleStatusCanceled).Error
	})
}

// Get 详情
func (s *SaleService) Get(ctx context.Context, id uint) (*jxc.SaleOrder, error) {
	var order jxc.SaleOrder
	if err := global.GVA_DB.WithContext(ctx).Preload("Items").First(&order, id).Error; err != nil {
		return nil, errors.New("销售单不存在")
	}
	return &order, nil
}

// Page 列表
func (s *SaleService) Page(ctx context.Context, page, size int) ([]jxc.SaleOrder, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.SaleOrder{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []jxc.SaleOrder
	err := db.Preload("Items").Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

func normalizeSaleItems(db *gorm.DB, items []jxc.SaleItem) (float64, error) {
	var total money.Cents
	for i := range items {
		it := &items[i]
		if it.SkuID == 0 {
			return 0, errors.New("明细缺少 SKU")
		}
		if it.Qty <= 0 {
			return 0, errors.New("销售数量必须大于 0")
		}
		if it.Price < 0 {
			return 0, errors.New("销售单价不能为负")
		}
		if err := ensureSku(db, it.SkuID); err != nil {
			return 0, err
		}
		price := money.FromYuan(it.Price)
		amount := price.MulQty(it.Qty)
		it.Price = price.Yuan()
		it.Amount = amount.Yuan()
		total = total.Add(amount)
	}
	return total.Yuan(), nil
}

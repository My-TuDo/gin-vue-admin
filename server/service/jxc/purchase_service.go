package jxc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PurchaseService 采购单：负责业务规则与状态机，**不直接碰库存**。
// 落账一律交给 LedgerService，单据层只负责"组装 MoveInput"。
type PurchaseService struct {
	ledger *LedgerService
}

func NewPurchaseService() *PurchaseService {
	return &PurchaseService{ledger: new(LedgerService)}
}

// Create 创建采购单（草稿）。审核之前不动库存。
func (s *PurchaseService) Create(ctx context.Context, order *jxc.PurchaseOrder) (*jxc.PurchaseOrder, error) {
	if len(order.Items) == 0 {
		return nil, errors.New("采购单至少需要一条明细")
	}
	if order.WarehouseID == 0 {
		return nil, errors.New("请选择入库仓库")
	}
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureWarehouse(db, order.WarehouseID); err != nil {
		return nil, err
	}
	if order.SupplierID != nil && *order.SupplierID > 0 {
		var cnt int64
		if err := db.Model(&jxc.Partner{}).
			Where("id = ? AND kind = ?", *order.SupplierID, jxc.PartnerSupplier).Count(&cnt).Error; err != nil {
			return nil, err
		}
		if cnt == 0 {
			return nil, errors.New("供应商不存在")
		}
	}
	total, err := normalizePurchaseItems(db, order.Items)
	if err != nil {
		return nil, err
	}
	order.TotalAmount = total
	order.Status = jxc.PurchaseStatusDraft

	// 单号并发/删单重建时可能撞唯一键，重试几次
	for attempt := 0; attempt < 3; attempt++ {
		no, err := nextOrderNo(db, "purchase_order", "order_no", "PO")
		if err != nil {
			return nil, err
		}
		order.OrderNo = no
		err = db.Create(order).Error
		if err == nil {
			return order, nil
		}
		if !isDuplicateKeyErr(err) {
			return nil, err
		}
		order.ID = 0
		order.Items = nil
	}
	return nil, errors.New("单号生成冲突，请重试")
}

// Approve 审核（草稿 → 已审核）。仍不动库存。
func (s *PurchaseService) Approve(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ok, err := advanceDocStatus(tx, &jxc.PurchaseOrder{}, id,
			jxc.PurchaseStatusDraft, jxc.PurchaseStatusApproved)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("仅草稿状态的采购单可以审核")
		}
		return nil
	})
}

// Receive 入库（已审核 → 已入库）：**这里才产生账本行**。
//
// 事务内顺序：锁单据 → 校验状态 → 组装账本行 → 过账 → 推进状态。
// 任何一步失败整体回滚，不会出现"库存加了但单据没变"。
func (s *PurchaseService) Receive(ctx context.Context, id uint, operator string, occurredAt time.Time) error {
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.PurchaseOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return errors.New("采购单不存在")
		}
		if order.Status != jxc.PurchaseStatusApproved {
			return fmt.Errorf("仅已审核的采购单可以入库（当前状态 %d）", order.Status)
		}
		var items []jxc.PurchaseItem
		if err := tx.Where("purchase_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("采购单没有明细")
		}

		moves := make([]MoveInput, 0, len(items))
		for _, it := range items {
			moves = append(moves, MoveInput{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID,
				Qty: it.Qty, UnitCost: it.UnitCost,
				SourceType: jxc.MoveSourcePurchaseIn, SourceID: order.ID, SourceLineID: it.ID,
				OccurredAt: occurredAt, Operator: operator,
			})
		}
		if err := s.ledger.Post(ctx, tx, moves); err != nil {
			return err
		}

		ok, err := advanceDocStatus(tx, &jxc.PurchaseOrder{}, id,
			jxc.PurchaseStatusApproved, jxc.PurchaseStatusReceived)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("单据状态已变更，请刷新后重试")
		}
		return tx.Model(&jxc.PurchaseOrder{}).Where("id = ?", id).Update("posted_at", occurredAt).Error
	})
}

// Cancel 取消：草稿/已审核可取消；**已入库不可取消**（要用红冲，见账本设计）。
func (s *PurchaseService) Cancel(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.PurchaseOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return errors.New("采购单不存在")
		}
		switch order.Status {
		case jxc.PurchaseStatusDraft, jxc.PurchaseStatusApproved:
		case jxc.PurchaseStatusReceived:
			return errors.New("已入库的采购单不能取消；如需更正，请在当期开红冲单")
		default:
			return errors.New("当前状态不可取消")
		}
		return tx.Model(&jxc.PurchaseOrder{}).Where("id = ?", id).Update("status", jxc.PurchaseStatusCanceled).Error
	})
}

// Get 详情
func (s *PurchaseService) Get(ctx context.Context, id uint) (*jxc.PurchaseOrder, error) {
	var order jxc.PurchaseOrder
	err := global.GVA_DB.WithContext(ctx).Preload("Items").First(&order, id).Error
	if err != nil {
		return nil, errors.New("采购单不存在")
	}
	return &order, nil
}

// Page 列表
func (s *PurchaseService) Page(ctx context.Context, page, size int) ([]jxc.PurchaseOrder, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.PurchaseOrder{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []jxc.PurchaseOrder
	err := db.Preload("Items").Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// ============ 内部工具 ============

// normalizePurchaseItems 校验并归一化明细，返回总额（分制累加，落库 2 位小数）
func normalizePurchaseItems(db *gorm.DB, items []jxc.PurchaseItem) (float64, error) {
	var total money.Cents
	for i := range items {
		it := &items[i]
		if it.SkuID == 0 {
			return 0, errors.New("明细缺少 SKU")
		}
		if it.Qty <= 0 {
			return 0, errors.New("采购数量必须大于 0")
		}
		if it.UnitCost < 0 {
			return 0, errors.New("采购单价不能为负")
		}
		if err := ensureSku(db, it.SkuID); err != nil {
			return 0, err
		}
		unit := money.FromYuan(it.UnitCost)
		amount := unit.MulQty(it.Qty)
		it.UnitCost = unit.Yuan()
		it.Amount = amount.Yuan()
		total = total.Add(amount)
	}
	return total.Yuan(), nil
}

// nextOrderNo 生成 PO-YYYYMMDD-### 形式的单号（按当日最大序号 +1）
func nextOrderNo(db *gorm.DB, table, column, prefix string) (string, error) {
	date := time.Now().Format("20060102")
	like := prefix + "-" + date + "-%"
	var maxNo string
	if err := db.Table(table).Where(column+" LIKE ?", like).
		Order(column+" desc").Limit(1).Pluck(column, &maxNo).Error; err != nil {
		return "", err
	}
	seq := 0
	if l := len(maxNo); l >= 3 {
		seq, _ = strconv.Atoi(maxNo[l-3:])
	}
	return fmt.Sprintf("%s-%s-%03d", prefix, date, seq+1), nil
}

// advanceDocStatus 条件推进单据状态：并发下只有一个请求能成功。
func advanceDocStatus(tx *gorm.DB, model any, id uint, from, to int8) (bool, error) {
	res := tx.Model(model).Where("id = ? AND status = ?", id, from).Update("status", to)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func ensureWarehouse(db *gorm.DB, id uint) error {
	var cnt int64
	if err := db.Model(&jxc.Warehouse{}).Where("id = ?", id).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("仓库不存在：%d", id)
	}
	return nil
}

func ensureSku(db *gorm.DB, id uint) error {
	var cnt int64
	if err := db.Model(&jxc.GoodsSku{}).Where("id = ?", id).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("SKU 不存在：%d", id)
	}
	return nil
}

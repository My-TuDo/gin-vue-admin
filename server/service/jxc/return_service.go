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

// ReturnService 销售退货：客户把货退回来。
//
// 与红冲的区别（这是两件不同的事，不能合成一张"负销售单"）：
//   - 红冲是"单子录错了"，整单撤销，不留业务痕迹；
//   - 退货是"生意真的发生了"，可以只退一部分，财务上要退款，库存上真的多回来了。
//
// 成本取值是这里最关键的一条：退货必须按**原单出库时冻结的成本**回补，
// 不能用当前移动均价——用今天的成本退上个月的货，等于把上个月的毛利改了。
type ReturnService struct {
	ledger *LedgerService
}

func NewReturnService() *ReturnService {
	return &ReturnService{ledger: new(LedgerService)}
}

// Create 建退货单（草稿）。逐行校验不能超过可退数量。
func (s *ReturnService) Create(ctx context.Context, ret *jxc.SaleReturn) (*jxc.SaleReturn, error) {
	if ret.SaleID == 0 {
		return nil, errors.New("请选择原销售单")
	}
	if len(ret.Items) == 0 {
		return nil, errors.New("请至少选择一项退货商品")
	}

	// 忽略客户端传入的主键/单号/状态/时间，避免伪造历史字段
	ret.ID = 0
	ret.ReturnNo = ""
	ret.SaleNo = ""
	ret.Status = jxc.ReturnStatusDraft
	ret.PostedAt = nil
	for i := range ret.Items {
		ret.Items[i].ID = 0
		ret.Items[i].ReturnID = 0
	}

	db := global.GVA_DB.WithContext(ctx)
	var err error
	err = db.Transaction(func(tx *gorm.DB) error {
		var sale jxc.SaleOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&sale, ret.SaleID).Error; err != nil {
			return errors.New("原销售单不存在")
		}
		if sale.Status != jxc.SaleStatusShipped {
			return errors.New("只有已出库的销售单可以退货")
		}
		if sale.ReversedAt != nil {
			return errors.New("该销售单已整单红冲，不能退货")
		}

		var saleItems []jxc.SaleItem
		if err := tx.Where("sale_id = ?", sale.ID).Find(&saleItems).Error; err != nil {
			return err
		}
		sold, bySku := aggregateSaleItems(saleItems)
		returned, err := s.returnedQty(tx, sale.ID)
		if err != nil {
			return err
		}

		var amount, cost money.Cents
		for i := range ret.Items {
			it := &ret.Items[i]
			orig, ok := bySku[it.SkuID]
			if !ok {
				return fmt.Errorf("SKU %d 不在原销售单中", it.SkuID)
			}
			if it.Qty <= 0 {
				return errors.New("退货数量必须大于 0")
			}
			canReturn := sold[it.SkuID] - returned[it.SkuID]
			if it.Qty > canReturn {
				return fmt.Errorf("SKU %d 最多可退 %d 件（原单 %d，已退 %d）",
					it.SkuID, canReturn, sold[it.SkuID], returned[it.SkuID])
			}
			// 单价与成本都取原单，不接受前端传值
			it.Price = orig.Price
			it.Amount = money.FromYuan(orig.Price).MulQty(it.Qty).Yuan()
			if orig.Qty > 0 {
				unit := money.FromYuan(orig.CostAmt).Yuan() / float64(orig.Qty)
				it.UnitCost = money.FromYuan(unit).Yuan()
			}
			amount = amount.Add(money.FromYuan(it.Amount))
			cost = cost.Add(money.FromYuan(it.UnitCost).MulQty(it.Qty))
		}

		no, err := nextOrderNo(tx, "RT")
		if err != nil {
			return err
		}
		ret.ReturnNo = no
		ret.SaleNo = sale.OrderNo
		ret.WarehouseID = sale.WarehouseID
		ret.Amount = amount.Yuan()
		ret.CostAmount = cost.Yuan()
		return tx.Create(ret).Error
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// Post 退货入库（草稿 → 已入库）：按原单成本回补库存。
func (s *ReturnService) Post(ctx context.Context, id uint, operator string, occurredAt time.Time) error {
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ret jxc.SaleReturn
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&ret, id).Error; err != nil {
			return errors.New("退货单不存在")
		}
		if ret.Status != jxc.ReturnStatusDraft {
			return errors.New("只有草稿状态的退货单可以入库")
		}
		var items []jxc.SaleReturnItem
		if err := tx.Where("return_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("退货单没有明细")
		}

		// 建单后可能又发生了别的退货（或原单被红冲），入库前必须在同一把锁下重新校验。
		// 只靠建单时校验，等于允许"先建两张各退全部的草稿，再依次入库"。
		var sale jxc.SaleOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&sale, ret.SaleID).Error; err != nil {
			return errors.New("原销售单不存在")
		}
		if sale.Status != jxc.SaleStatusShipped {
			return errors.New("原销售单不是已出库状态，不能退货入库")
		}
		if sale.ReversedAt != nil {
			return errors.New("原销售单已整单红冲，不能退货入库")
		}
		var saleItems []jxc.SaleItem
		if err := tx.Where("sale_id = ?", sale.ID).Find(&saleItems).Error; err != nil {
			return err
		}
		sold, _ := aggregateSaleItems(saleItems)
		returned, err := s.returnedQty(tx, sale.ID)
		if err != nil {
			return err
		}
		// 同一张退货单里也可能出现同 SKU 多行，先合并再校验
		thisQty := map[uint]int{}
		for _, it := range items {
			if it.Qty <= 0 {
				return errors.New("退货数量必须大于 0")
			}
			if _, ok := sold[it.SkuID]; !ok {
				return fmt.Errorf("SKU %d 不在原销售单中", it.SkuID)
			}
			thisQty[it.SkuID] += it.Qty
		}
		for skuID, qty := range thisQty {
			canReturn := sold[skuID] - returned[skuID]
			if qty > canReturn {
				return fmt.Errorf("SKU %d 最多可退 %d 件（原单 %d，已退 %d），本次退货 %d 件",
					skuID, canReturn, sold[skuID], returned[skuID], qty)
			}
		}

		moves := make([]MoveInput, 0, len(items))
		for _, it := range items {
			moves = append(moves, MoveInput{
				WarehouseID: ret.WarehouseID, SkuID: it.SkuID,
				Qty: it.Qty, UnitCost: it.UnitCost, // 正数入库，成本取原单
				SourceType: jxc.MoveSourceSaleReturn, SourceID: ret.ID, SourceLineID: it.ID,
				OccurredAt: occurredAt, Operator: operator,
			})
		}
		if err := s.ledger.Post(ctx, tx, moves); err != nil {
			return err
		}

		return tx.Model(&jxc.SaleReturn{}).Where("id = ?", id).
			Updates(map[string]interface{}{"status": jxc.ReturnStatusPosted, "posted_at": occurredAt}).Error
	})
}

// Cancel 取消退货单（仅草稿）
func (s *ReturnService) Cancel(ctx context.Context, id uint) error {
	res := global.GVA_DB.WithContext(ctx).Model(&jxc.SaleReturn{}).
		Where("id = ? AND status = ?", id, jxc.ReturnStatusDraft).
		Update("status", jxc.ReturnStatusCanceled)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("只有草稿状态的退货单可以取消")
	}
	return nil
}

// Get 详情
func (s *ReturnService) Get(ctx context.Context, id uint) (*jxc.SaleReturn, error) {
	var ret jxc.SaleReturn
	if err := global.GVA_DB.WithContext(ctx).Preload("Items").First(&ret, id).Error; err != nil {
		return nil, errors.New("退货单不存在")
	}
	return &ret, nil
}

// Page 列表
func (s *ReturnService) Page(ctx context.Context, page, size int) ([]jxc.SaleReturn, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.SaleReturn{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []jxc.SaleReturn
	err := db.Preload("Items").Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// returnableQty 某张销售单每个 SKU 还能退多少（供前端展示与后端校验）
func (s *ReturnService) Returnable(ctx context.Context, saleID uint) ([]ReturnableLine, error) {
	db := global.GVA_DB.WithContext(ctx)
	var sale jxc.SaleOrder
	if err := db.First(&sale, saleID).Error; err != nil {
		return nil, errors.New("销售单不存在")
	}
	var items []jxc.SaleItem
	if err := db.Where("sale_id = ?", saleID).Find(&items).Error; err != nil {
		return nil, err
	}
	returned, err := s.returnedQty(db, saleID)
	if err != nil {
		return nil, err
	}
	sold, bySku := aggregateSaleItems(items)
	out := make([]ReturnableLine, 0, len(items))
	for _, it := range items {
		orig := bySku[it.SkuID]
		var unitCost float64
		if orig.Qty > 0 {
			unitCost = money.FromYuan(orig.CostAmt).Yuan() / float64(orig.Qty)
		}
		out = append(out, ReturnableLine{
			SkuID: it.SkuID, SoldQty: sold[it.SkuID], ReturnedQty: returned[it.SkuID],
			CanReturn: sold[it.SkuID] - returned[it.SkuID], Price: orig.Price, UnitCost: unitCost,
		})
	}
	return out, nil
}

// aggregateSaleItems 把同一 SKU 的多行合并成"原售总量"，同时保留一行用于
// 取价格与成本。销售单正常不会出现同 SKU 多行，但防御性处理可以避免
// "每行都按单行上限校验"导致超退。
func aggregateSaleItems(items []jxc.SaleItem) (map[uint]int, map[uint]jxc.SaleItem) {
	sold := map[uint]int{}
	bySku := map[uint]jxc.SaleItem{}
	for _, it := range items {
		sold[it.SkuID] += it.Qty
		bySku[it.SkuID] = it
	}
	return sold, bySku
}

// ReturnableLine 可退明细
type ReturnableLine struct {
	SkuID       uint    `json:"skuId"`
	SoldQty     int     `json:"soldQty"`
	ReturnedQty int     `json:"returnedQty"`
	CanReturn   int     `json:"canReturn"`
	Price       float64 `json:"price"`
	UnitCost    float64 `json:"unitCost"`
}

// returnedQty 已退数量（只统计已入库的退货单）
func (s *ReturnService) returnedQty(db *gorm.DB, saleID uint) (map[uint]int, error) {
	type row struct {
		SkuID uint
		Qty   int
	}
	var rows []row
	err := db.Table("sale_return_item i").
		Select("i.sku_id AS sku_id, SUM(i.qty) AS qty").
		Joins("JOIN sale_return r ON r.id = i.return_id").
		Where("r.sale_id = ? AND r.status = ?", saleID, jxc.ReturnStatusPosted).
		Group("i.sku_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[uint]int{}
	for _, r := range rows {
		out[r.SkuID] = r.Qty
	}
	return out, nil
}

package jxc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	common "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
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
		costBySku := saleCostBySku(saleItems)
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
			// 单价与成本都取原单，不接受前端传值。
			// 成本按该 SKU 的**原始总成本 / 原始总数量**求均价，而不是取某一行，
			// 同一 SKU 分多行销售时也能算对。
			it.Price = orig.Price
			it.Amount = money.FromYuan(orig.Price).MulQty(it.Qty).Yuan()
			if sold[it.SkuID] > 0 {
				unit := money.FromYuan(costBySku[it.SkuID].Yuan() / float64(sold[it.SkuID]))
				it.UnitCost = unit.Yuan()
				it.CostAmount = unit.MulQty(it.Qty).Yuan()
			}
			amount = amount.Add(money.FromYuan(it.Amount))
			cost = cost.Add(money.FromYuan(it.CostAmount))
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

		// 计算本次每一行实际回补的成本。
		// 若不是"最后一次退货"，按原单均价 × 数量；如果是最后一次，则把
		// 原单总成本减去历史已退成本后的**全部余额**放在这一行上，
		// 吸收"100.00 / 3 件 = 33.33"这类分位残差。否则退完三件只回来 99.99。
		costBySku := saleCostBySku(saleItems)
		returnedCost, err := s.returnedCost(tx, sale.ID)
		if err != nil {
			return err
		}
		lastIdx := make(map[uint]int, len(items))
		for i := range items {
			lastIdx[items[i].SkuID] = i
		}
		allocated := map[uint]money.Cents{}
		var totalCost money.Cents
		moves := make([]MoveInput, 0, len(items))
		for i := range items {
			it := &items[i]
			unit := money.FromYuan(it.UnitCost)
			lineCost := unit.MulQty(it.Qty)
			isLastLine := lastIdx[it.SkuID] == i
			isFinalReturn := returned[it.SkuID]+thisQty[it.SkuID] == sold[it.SkuID]
			if isFinalReturn && isLastLine {
				remaining := costBySku[it.SkuID].Sub(returnedCost[it.SkuID]).Sub(allocated[it.SkuID])
				if remaining < 0 {
					return fmt.Errorf("SKU %d 的退货成本异常（原单成本 %s，已退 %s，本次已分配 %s）",
						it.SkuID, costBySku[it.SkuID], returnedCost[it.SkuID], allocated[it.SkuID])
				}
				lineCost = remaining
			}
			allocated[it.SkuID] = allocated[it.SkuID].Add(lineCost)
			it.CostAmount = lineCost.Yuan()
			totalCost = totalCost.Add(lineCost)

			if err := tx.Model(&jxc.SaleReturnItem{}).Where("id = ?", it.ID).
				Update("cost_amount", lineCost.Yuan()).Error; err != nil {
				return err
			}
			// 金额按精确值入账；UnitCost 仅作为展示/参考，不再反乘出金额。
			exact := lineCost
			moves = append(moves, MoveInput{
				WarehouseID: ret.WarehouseID, SkuID: it.SkuID,
				Qty: it.Qty, UnitCost: it.UnitCost,
				AmountOverride: &exact,
				SourceType:     jxc.MoveSourceSaleReturn, SourceID: ret.ID, SourceLineID: it.ID,
				OccurredAt: occurredAt, Operator: operator,
			})
		}
		if err := s.ledger.Post(ctx, tx, moves); err != nil {
			return err
		}

		return tx.Model(&jxc.SaleReturn{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status": jxc.ReturnStatusPosted, "posted_at": occurredAt,
				"cost_amount": totalCost.Yuan(),
			}).Error
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

// Page 列表（兼容旧调用；新代码用 Search）
func (s *ReturnService) Page(ctx context.Context, page, size int) ([]jxc.SaleReturn, int64, error) {
	return s.Search(ctx, jxcReq.DocumentSearch{
		PageInfo: common.PageInfo{Page: page, PageSize: size},
	})
}

// Search 服务端筛选 + 分页。
// 销售退货没有直接的客户字段，关键词/客户筛选通过原销售单 sale_order 关联。
func (s *ReturnService) Search(ctx context.Context, info jxcReq.DocumentSearch) ([]jxc.SaleReturn, int64, error) {
	info.Normalize()
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.SaleReturn{}).
		Joins("JOIN sale_order so ON so.id = sale_return.sale_id").
		Joins("LEFT JOIN jxc_partner p ON p.id = so.customer_id")

	if info.Status != nil {
		db = db.Where("sale_return.status = ?", *info.Status)
	}
	if info.WarehouseID > 0 {
		db = db.Where("sale_return.warehouse_id = ?", info.WarehouseID)
	}
	if info.PartnerID > 0 {
		db = db.Where("so.customer_id = ?", info.PartnerID)
	}
	if kw := keywordPattern(info.Keyword); kw != "" {
		db = db.Where("(sale_return.return_no LIKE ? OR sale_return.sale_no LIKE ? OR sale_return.reason LIKE ? OR p.name LIKE ? OR p.code LIKE ?)",
			kw, kw, kw, kw, kw)
	}
	db, err := applyDocumentDateRange(db, info, "sale_return.created_at", "sale_return.posted_at")
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset := info.LimitOffset()
	var list []jxc.SaleReturn
	err = db.Preload("Items").Order("sale_return.id desc").Offset(offset).Limit(limit).Find(&list).Error
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
	costBySku := saleCostBySku(items)
	out := make([]ReturnableLine, 0, len(items))
	for _, it := range items {
		orig := bySku[it.SkuID]
		var unitCost float64
		if sold[it.SkuID] > 0 {
			unitCost = money.FromYuan(costBySku[it.SkuID].Yuan() / float64(sold[it.SkuID])).Yuan()
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

// saleCostBySku 原单每个 SKU 的冻结总成本。
// 退货成本的分母必须是"这个 SKU 的总数量"，而不是某一行的数量——
// 同一 SKU 分多行销售时，按某一行算会把成本摊错。
func saleCostBySku(items []jxc.SaleItem) map[uint]money.Cents {
	out := map[uint]money.Cents{}
	for _, it := range items {
		out[it.SkuID] = out[it.SkuID].Add(money.FromYuan(it.CostAmt))
	}
	return out
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

// returnedCost 已退成本（只统计已入库的退货单）。
//
// cost_amount 是本轮新增的"精确回补金额"；老库的历史退货行没有这一列，
// 用 qty × unit_cost 兜底，迁移后不会把已退成本算成 0。
func (s *ReturnService) returnedCost(db *gorm.DB, saleID uint) (map[uint]money.Cents, error) {
	type row struct {
		SkuID uint
		Cost  float64
	}
	var rows []row
	err := db.Table("sale_return_item i").
		Select("i.sku_id AS sku_id, COALESCE(SUM(CASE WHEN i.cost_amount <> 0 THEN i.cost_amount ELSE i.qty * i.unit_cost END),0) AS cost").
		Joins("JOIN sale_return r ON r.id = i.return_id").
		Where("r.sale_id = ? AND r.status = ?", saleID, jxc.ReturnStatusPosted).
		Group("i.sku_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[uint]money.Cents{}
	for _, r := range rows {
		out[r.SkuID] = money.FromYuan(r.Cost)
	}
	return out, nil
}

package jxc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	common "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PurchaseReturnService 采购退货：把已入库的货退给供应商。
//
// 与采购红冲的区别：红冲整单反向、要求库存足够；采购退货可以只退一部分，
// 是一笔真实业务，退款金额按**原采购单价**计算。
//
// 账本动作是一条负向移动（source_type=purchase_return），金额用原采购单价精确
// 出库，不走当前移动加权平均——否则供应商按原价退款、账上按今天的均价减成本，
// 差额会凭空出现。
type PurchaseReturnService struct {
	ledger *LedgerService
}

func NewPurchaseReturnService() *PurchaseReturnService {
	return &PurchaseReturnService{ledger: new(LedgerService)}
}

// Create 建采购退货单（草稿）。只接受 purchaseId + items[{skuId,qty}]，
// 单价、金额、仓库、供应商全部从原采购单带出，客户端传值无效。
func (s *PurchaseReturnService) Create(ctx context.Context, ret *jxc.PurchaseReturn) (*jxc.PurchaseReturn, error) {
	if ret.PurchaseID == 0 {
		return nil, errors.New("请选择原采购单")
	}
	if len(ret.Items) == 0 {
		return nil, errors.New("请至少选择一项退货商品")
	}

	// 归零客户端可伪造的字段
	ret.ID = 0
	ret.ReturnNo = ""
	ret.PurchaseNo = ""
	ret.SupplierID = nil
	ret.WarehouseID = 0
	ret.Status = jxc.PurchaseReturnStatusDraft
	ret.Amount = 0
	ret.PostedAt = nil
	for i := range ret.Items {
		ret.Items[i].ID = 0
		ret.Items[i].ReturnID = 0
		ret.Items[i].UnitCost = 0
		ret.Items[i].Amount = 0
	}

	db := global.GVA_DB.WithContext(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		var po jxc.PurchaseOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, ret.PurchaseID).Error; err != nil {
			return errors.New("原采购单不存在")
		}
		if po.Status != jxc.PurchaseStatusReceived {
			return errors.New("只有已入库的采购单可以退货")
		}
		if po.ReversedAt != nil {
			return errors.New("原采购单已整单红冲，不能退货")
		}

		var poItems []jxc.PurchaseItem
		if err := tx.Where("purchase_id = ?", po.ID).Find(&poItems).Error; err != nil {
			return err
		}
		if len(poItems) == 0 {
			return errors.New("原采购单没有明细")
		}
		purchased, costBySku := aggregatePurchaseItems(poItems)
		returned, err := purchaseReturnedQty(tx, po.ID)
		if err != nil {
			return err
		}

		seen := map[uint]bool{}
		var total money.Cents
		for i := range ret.Items {
			it := &ret.Items[i]
			if it.SkuID == 0 {
				return errors.New("明细缺少 SKU")
			}
			if it.Qty <= 0 {
				return errors.New("退货数量必须大于 0")
			}
			if seen[it.SkuID] {
				return fmt.Errorf("SKU %d 在本次退货中重复出现", it.SkuID)
			}
			seen[it.SkuID] = true

			bought, ok := purchased[it.SkuID]
			if !ok {
				return fmt.Errorf("SKU %d 不在原采购单中", it.SkuID)
			}
			canReturn := bought - returned[it.SkuID]
			if it.Qty > canReturn {
				return fmt.Errorf("SKU %d 最多可退 %d 件（原采购 %d，已退 %d）",
					it.SkuID, canReturn, bought, returned[it.SkuID])
			}
			// 同一 SKU 多行采购时按总金额/总数量取加权原价；正常单据通常只有一行。
			var unit money.Cents
			if bought > 0 {
				unit = money.FromYuan(costBySku[it.SkuID].Yuan() / float64(bought))
			}
			it.UnitCost = unit.Yuan()
			it.Amount = unit.MulQty(it.Qty).Yuan()
			total = total.Add(money.FromYuan(it.Amount))
		}

		no, err := nextOrderNo(tx, "PR")
		if err != nil {
			return err
		}
		ret.ReturnNo = no
		ret.PurchaseNo = po.OrderNo
		ret.WarehouseID = po.WarehouseID
		ret.SupplierID = po.SupplierID
		ret.Amount = total.Yuan()
		return tx.Create(ret).Error
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// Post 退货出库（草稿 → 已退货）：按原采购单价精确扣减库存。
func (s *PurchaseReturnService) Post(ctx context.Context, id uint, operator string, occurredAt time.Time) error {
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ret jxc.PurchaseReturn
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&ret, id).Error; err != nil {
			return errors.New("采购退货单不存在")
		}
		if ret.Status != jxc.PurchaseReturnStatusDraft {
			return errors.New("只有草稿状态的采购退货单可以出库")
		}

		// 建单后可能又有别的退货单入库（或原采购单被红冲），出库前必须在同一把锁下重算。
		var po jxc.PurchaseOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, ret.PurchaseID).Error; err != nil {
			return errors.New("原采购单不存在")
		}
		if po.Status != jxc.PurchaseStatusReceived {
			return errors.New("原采购单不是已入库状态，不能退货")
		}
		if po.ReversedAt != nil {
			return errors.New("原采购单已整单红冲，不能退货")
		}

		var poItems []jxc.PurchaseItem
		if err := tx.Where("purchase_id = ?", po.ID).Find(&poItems).Error; err != nil {
			return err
		}
		purchased, _ := aggregatePurchaseItems(poItems)
		returned, err := purchaseReturnedQty(tx, po.ID)
		if err != nil {
			return err
		}

		var items []jxc.PurchaseReturnItem
		if err := tx.Where("return_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("采购退货单没有明细")
		}

		// 同一 SKU 多行时先合并校验，再逐行过账，防止两张草稿叠加超退。
		thisQty := map[uint]int{}
		for _, it := range items {
			if it.Qty <= 0 {
				return errors.New("退货数量必须大于 0")
			}
			if _, ok := purchased[it.SkuID]; !ok {
				return fmt.Errorf("SKU %d 不在原采购单中", it.SkuID)
			}
			thisQty[it.SkuID] += it.Qty
		}
		for skuID, qty := range thisQty {
			canReturn := purchased[skuID] - returned[skuID]
			if qty > canReturn {
				return fmt.Errorf("SKU %d 最多可退 %d 件（原采购 %d，已退 %d），本次退货 %d 件",
					skuID, canReturn, purchased[skuID], returned[skuID], qty)
			}
		}

		moves := make([]MoveInput, 0, len(items))
		for _, it := range items {
			exact := money.FromYuan(it.Amount).Neg()
			moves = append(moves, MoveInput{
				// 仓库以原采购单为准：即使草稿里的 warehouse_id 被改过，
				// 也不能把货从别的仓库退出去。
				WarehouseID: po.WarehouseID, SkuID: it.SkuID,
				Qty: -it.Qty, UnitCost: it.UnitCost, AmountOverride: &exact,
				SourceType: jxc.MoveSourcePurchaseReturn, SourceID: ret.ID, SourceLineID: it.ID,
				OccurredAt: occurredAt, Operator: operator,
			})
		}
		if err := s.ledger.Post(ctx, tx, moves); err != nil {
			return err
		}

		return tx.Model(&jxc.PurchaseReturn{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status": jxc.PurchaseReturnStatusPosted, "posted_at": occurredAt,
			}).Error
	})
}

// Cancel 取消采购退货单（仅草稿）
func (s *PurchaseReturnService) Cancel(ctx context.Context, id uint) error {
	res := global.GVA_DB.WithContext(ctx).Model(&jxc.PurchaseReturn{}).
		Where("id = ? AND status = ?", id, jxc.PurchaseReturnStatusDraft).
		Update("status", jxc.PurchaseReturnStatusCanceled)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("只有草稿状态的采购退货单可以取消")
	}
	return nil
}

// Get 详情
func (s *PurchaseReturnService) Get(ctx context.Context, id uint) (*jxc.PurchaseReturn, error) {
	var ret jxc.PurchaseReturn
	if err := global.GVA_DB.WithContext(ctx).Preload("Items").First(&ret, id).Error; err != nil {
		return nil, errors.New("采购退货单不存在")
	}
	return &ret, nil
}

// Page 列表（兼容旧调用；新代码用 Search）
func (s *PurchaseReturnService) Page(ctx context.Context, page, size int) ([]jxc.PurchaseReturn, int64, error) {
	return s.Search(ctx, jxcReq.DocumentSearch{
		PageInfo: common.PageInfo{Page: page, PageSize: size},
	})
}

// Search 服务端筛选 + 分页，并带出供应商/仓库名称与明细行数。
func (s *PurchaseReturnService) Search(ctx context.Context, info jxcReq.DocumentSearch) ([]jxc.PurchaseReturn, int64, error) {
	info.Normalize()
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.PurchaseReturn{}).
		Joins("LEFT JOIN jxc_partner p ON p.id = purchase_return.supplier_id").
		Joins("LEFT JOIN warehouse w ON w.id = purchase_return.warehouse_id")

	if info.Status != nil {
		db = db.Where("purchase_return.status = ?", *info.Status)
	}
	if info.WarehouseID > 0 {
		db = db.Where("purchase_return.warehouse_id = ?", info.WarehouseID)
	}
	if info.PartnerID > 0 {
		db = db.Where("purchase_return.supplier_id = ?", info.PartnerID)
	}
	if kw := keywordPattern(info.Keyword); kw != "" {
		db = db.Where("(purchase_return.return_no LIKE ? OR purchase_return.purchase_no LIKE ? OR purchase_return.reason LIKE ? OR p.name LIKE ? OR p.code LIKE ?)",
			kw, kw, kw, kw, kw)
	}
	db, err := applyDocumentDateRange(db, info, "purchase_return.created_at", "purchase_return.posted_at")
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset := info.LimitOffset()
	var list []jxc.PurchaseReturn
	err = db.Select("purchase_return.*, p.name AS supplier_name, w.name AS warehouse_name").
		Preload("Items").Order("purchase_return.id desc").Offset(offset).Limit(limit).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		list[i].ItemCount = len(list[i].Items)
	}
	return list, total, nil
}

// Returnable 某张采购单每个 SKU 还能退多少（供前端展示与后端校验）
func (s *PurchaseReturnService) Returnable(ctx context.Context, purchaseID uint) ([]PurchaseReturnableLine, error) {
	db := global.GVA_DB.WithContext(ctx)
	var po jxc.PurchaseOrder
	if err := db.First(&po, purchaseID).Error; err != nil {
		return nil, errors.New("采购单不存在")
	}
	if po.Status != jxc.PurchaseStatusReceived {
		return nil, errors.New("只有已入库的采购单可以退货")
	}
	if po.ReversedAt != nil {
		return nil, errors.New("原采购单已整单红冲，不能退货")
	}
	var items []jxc.PurchaseItem
	if err := db.Where("purchase_id = ?", po.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	purchased, costBySku := aggregatePurchaseItems(items)
	returned, err := purchaseReturnedQty(db, po.ID)
	if err != nil {
		return nil, err
	}
	out := make([]PurchaseReturnableLine, 0, len(purchased))
	for skuID, bought := range purchased {
		var unit float64
		if bought > 0 {
			unit = money.FromYuan(costBySku[skuID].Yuan() / float64(bought)).Yuan()
		}
		out = append(out, PurchaseReturnableLine{
			SkuID: skuID, PurchasedQty: bought, ReturnedQty: returned[skuID],
			CanReturn: bought - returned[skuID], UnitCost: unit,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SkuID < out[j].SkuID })
	return out, nil
}

// PurchaseReturnableLine 可退明细
type PurchaseReturnableLine struct {
	SkuID        uint    `json:"skuId"`
	PurchasedQty int     `json:"purchasedQty"`
	ReturnedQty  int     `json:"returnedQty"`
	CanReturn    int     `json:"canReturn"`
	UnitCost     float64 `json:"unitCost"`
}

// aggregatePurchaseItems 把同一 SKU 的多行采购合并成总量与总成本。
func aggregatePurchaseItems(items []jxc.PurchaseItem) (map[uint]int, map[uint]money.Cents) {
	qty := map[uint]int{}
	cost := map[uint]money.Cents{}
	for _, it := range items {
		qty[it.SkuID] += it.Qty
		cost[it.SkuID] = cost[it.SkuID].Add(money.FromYuan(it.Amount))
	}
	return qty, cost
}

// purchaseReturnedQty 已退数量（只统计已出库的采购退货单）
func purchaseReturnedQty(db *gorm.DB, purchaseID uint) (map[uint]int, error) {
	type row struct {
		SkuID uint
		Qty   int
	}
	var rows []row
	err := db.Table("purchase_return_item i").
		Select("i.sku_id AS sku_id, SUM(i.qty) AS qty").
		Joins("JOIN purchase_return r ON r.id = i.return_id").
		Where("r.purchase_id = ? AND r.status = ?", purchaseID, jxc.PurchaseReturnStatusPosted).
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

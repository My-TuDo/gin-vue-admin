package jxc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// SaleService 销售中心服务
type SaleService struct{}

// ========== 辅助 ==========

// genSaleOrderNo 生成销售单号 SO-YYYYMMDD-XXX
func (s *SaleService) genSaleOrderNo(db *gorm.DB) (string, error) {
	date := time.Now().Format("20060102")
	like := "SO-" + date + "%"
	var maxNo string
	if err := db.Model(&jxc.SaleOrder{}).Where("order_no LIKE ?", like).
		Order("order_no DESC").Limit(1).Pluck("order_no", &maxNo).Error; err != nil {
		return "", err
	}
	if maxNo == "" {
		return "SO-" + date + "-001", nil
	}
	seq, err := strconv.Atoi(maxNo[len(maxNo)-3:])
	if err != nil {
		return "", fmt.Errorf("解析单号序号失败: %s", maxNo)
	}
	return fmt.Sprintf("SO-%s-%03d", date, seq+1), nil
}

// lockStock 锁定库存（校验可售库存充足）
func (s *SaleService) lockStock(tx *gorm.DB, warehouseID, skuID uint, qty int) error {
	var stock jxc.Stock
	if err := tx.Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).First(&stock).Error; err != nil {
		return errors.New("该 SKU 在该仓库无库存记录，请先采购入库")
	}
	available := stock.Quantity - stock.LockQuantity
	if available < qty {
		return fmt.Errorf("库存不足：可售库存 %d，需 %d", available, qty)
	}
	return tx.Model(&stock).Update("lock_quantity", stock.LockQuantity+qty).Error
}

// unlockStock 释放锁定库存
func (s *SaleService) unlockStock(tx *gorm.DB, warehouseID, skuID uint, qty int) error {
	var stock jxc.Stock
	if err := tx.Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).First(&stock).Error; err != nil {
		return err
	}
	return tx.Model(&stock).Update("lock_quantity", stock.LockQuantity-qty).Error
}

// fillItemSnapshot 填充明细快照并汇总金额
// 单价以 SKU 销售价为准；金额符号：正常销售正 / 退货负 / 换货按 direction（1换出正 2换入负）
func (s *SaleService) fillItemSnapshot(db *gorm.DB, orderType int8, items []jxc.SaleItem) (float64, error) {
	// 换货单必须同时包含换出（direction=1）与换入（direction=2）明细
	if orderType == jxc.SaleTypeExchange {
		hasOut, hasIn := false, false
		for _, it := range items {
			if it.Direction == 1 {
				hasOut = true
			}
			if it.Direction == 2 {
				hasIn = true
			}
		}
		if !hasOut || !hasIn {
			return 0, errors.New("换货单必须同时包含换出和换入明细")
		}
	}
	var total float64
	for i := range items {
		it := &items[i]
		if it.Qty <= 0 {
			return 0, errors.New("销售数量必须大于 0")
		}
		var sku jxc.GoodsSku
		if err := db.First(&sku, it.SkuID).Error; err != nil {
			return 0, errors.New("SKU 不存在")
		}
		// 单价固定取 SKU 销售价，前端传入的 price 一律忽略（角色改价权限后续版本开放）
		it.Price = sku.SalePrice
		it.Amount = float64(it.Qty) * it.Price
		switch orderType {
		case jxc.SaleTypeReturn:
			it.Amount = -it.Amount // 退货：收入减少
		case jxc.SaleTypeExchange:
			switch it.Direction {
			case 1: // 换出：正向流水（销售延续）
			case 2: // 换入：负向流水（相当于退回）
				it.Amount = -it.Amount
			default:
				return 0, errors.New("换货明细必须指定方向（1换出/2换入）")
			}
		}
		total += it.Amount
		it.SkuCode = sku.SkuCode
		it.Color = sku.Color
		it.Size = sku.Size
		var goods jxc.Goods
		db.First(&goods, sku.GoodsID)
		it.GoodsName = goods.Name
	}
	return total, nil
}

// checkOriginalOrder 校验关联原单（退货/换货可关联正常销售已出库或换货已完成；换出的商品可继续退/换）
func (s *SaleService) checkOriginalOrder(db *gorm.DB, orderType int8, originalID *uint) error {
	if originalID == nil || *originalID == 0 {
		return errors.New("请选择关联原单")
	}
	var original jxc.SaleOrder
	if err := db.First(&original, *originalID).Error; err != nil {
		return errors.New("关联原单不存在")
	}
	if original.Status != jxc.SaleStatusShipped {
		return errors.New("仅已出库的销售单可作为原单")
	}
	switch orderType {
	case jxc.SaleTypeReturn, jxc.SaleTypeExchange:
		if original.OrderType != jxc.SaleTypeNormal && original.OrderType != jxc.SaleTypeExchange {
			return errors.New("退货/换货单只能关联正常销售或换货单")
		}
	}
	return nil
}

// checkInboundLimit 校验入库数量上限（按商品 Goods 维度，支持同款换码/换色）：
// 退货/换货单的入库明细不得超过原单该商品的出库数量减去已被其他已确认单据占用的数量
func (s *SaleService) checkInboundLimit(db *gorm.DB, orderType int8, originalID uint, items []jxc.SaleItem, excludeID uint) error {
	if orderType != jxc.SaleTypeReturn && orderType != jxc.SaleTypeExchange {
		return nil
	}
	// 收集涉及 SKU 的 Goods 映射
	skuSet := map[uint]struct{}{}
	for _, it := range items {
		skuSet[it.SkuID] = struct{}{}
	}
	var skus []jxc.GoodsSku
	if len(skuSet) > 0 {
		ids := make([]uint, 0, len(skuSet))
		for id := range skuSet {
			ids = append(ids, id)
		}
		db.Where("id IN ?", ids).Find(&skus)
	}
	skuGoods := map[uint]uint{}
	for _, s := range skus {
		skuGoods[s.ID] = s.GoodsID
	}
	goodsName := map[uint]string{}
	for _, s := range skus {
		if s.GoodsID > 0 && goodsName[s.GoodsID] == "" {
			var g jxc.Goods
			db.First(&g, s.GoodsID)
			goodsName[s.GoodsID] = g.Name
		}
	}
	// 原单出库明细（按商品）：正常销售单全部明细；换货单取换出明细（direction=1）
	type qtyRow struct {
		GoodsID uint
		Qty     int
	}
	var outQty []qtyRow
	db.Table("sale_item si").
		Select("gs.goods_id AS goods_id, SUM(si.qty) AS qty").
		Joins("JOIN sale_order o ON o.id = si.sale_id").
		Joins("JOIN goods_sku gs ON gs.id = si.sku_id").
		Where("si.sale_id = ? AND (o.order_type = ? OR si.direction = ?)", originalID, jxc.SaleTypeNormal, 1).
		Group("gs.goods_id").Scan(&outQty)
	// 已被其他已确认单据占用的入库数量（按商品）：退货单明细 direction=0、换货单换入 direction=2
	var used []qtyRow
	db.Table("sale_item si").
		Select("gs.goods_id AS goods_id, SUM(si.qty) AS qty").
		Joins("JOIN sale_order o ON o.id = si.sale_id").
		Joins("JOIN goods_sku gs ON gs.id = si.sku_id").
		Where("o.original_order_id = ? AND o.status = ? AND o.id <> ? AND si.direction IN (0, 2)", originalID, jxc.SaleStatusShipped, excludeID).
		Group("gs.goods_id").Scan(&used)
	// 当前单据入库明细（按商品）
	inQty := map[uint]int{}
	for _, it := range items {
		isIn := orderType == jxc.SaleTypeReturn || (orderType == jxc.SaleTypeExchange && it.Direction == 2)
		if isIn {
			inQty[skuGoods[it.SkuID]] += it.Qty
		}
	}
	// 校验：入库数量 ≤ 原单出库 − 已占用
	limitByGoods := map[uint]int{}
	for _, r := range outQty {
		limitByGoods[r.GoodsID] = r.Qty
	}
	usedByGoods := map[uint]int{}
	for _, r := range used {
		usedByGoods[r.GoodsID] += r.Qty
	}
	for goodsID, qty := range inQty {
		limit := limitByGoods[goodsID] - usedByGoods[goodsID]
		if qty > limit {
			return fmt.Errorf("入库数量超出原单剩余可退换数量：商品「%s」原单出库 %d，已占用 %d，最多还可入库 %d", goodsName[goodsID], limitByGoods[goodsID], usedByGoods[goodsID], limit)
		}
	}
	return nil
}

// ========== 销售单 CRUD ==========

// CreateSaleOrder 创建销售单（待出库, 正常销售锁定库存）
func (s *SaleService) CreateSaleOrder(ctx context.Context, order *jxc.SaleOrder) error {
	if len(order.Items) == 0 {
		return errors.New("销售单至少需要一条明细")
	}
	db := global.GVA_DB.WithContext(ctx)
	orderNo, err := s.genSaleOrderNo(db)
	if err != nil {
		return err
	}
	order.OrderNo = orderNo
	order.Status = jxc.SaleStatusPending
	if order.OrderType == 0 {
		order.OrderType = jxc.SaleTypeNormal
	}
	// 退货/换货必须关联合法原单
	if order.OrderType != jxc.SaleTypeNormal {
		if err := s.checkOriginalOrder(db, order.OrderType, order.OriginalOrderID); err != nil {
			return err
		}
		if err := s.checkInboundLimit(db, order.OrderType, *order.OriginalOrderID, order.Items, 0); err != nil {
			return err
		}
	}
	total, err := s.fillItemSnapshot(db, order.OrderType, order.Items)
	if err != nil {
		return err
	}
	order.TotalAmount = total

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		// 正常销售：建单即锁定库存
		if order.OrderType == jxc.SaleTypeNormal {
			for i := range order.Items {
				if err := s.lockStock(tx, order.WarehouseID, order.Items[i].SkuID, order.Items[i].Qty); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// GetSalePage 分页查询销售单
func (s *SaleService) GetSalePage(ctx context.Context, info request.PageInfo) (list []jxc.SaleOrder, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.SaleOrder{})
	if info.Keyword != "" {
		db = db.Where("order_no LIKE ?", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	limit, offset := info.LimitOffset()
	err = db.Preload("Customer").Preload("Warehouse").
		Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return
}

// GetSaleDetail 查询销售单详情（含明细）
func (s *SaleService) GetSaleDetail(ctx context.Context, id uint) (order jxc.SaleOrder, err error) {
	err = global.GVA_DB.WithContext(ctx).
		Preload("Customer").Preload("Warehouse").Preload("Items").
		First(&order, id).Error
	return
}

// UpdateSaleOrder 更新销售单（仅待出库, 明细重建并重新锁定库存）
func (s *SaleService) UpdateSaleOrder(ctx context.Context, order *jxc.SaleOrder) error {
	if len(order.Items) == 0 {
		return errors.New("销售单至少需要一条明细")
	}
	db := global.GVA_DB.WithContext(ctx)
	var old jxc.SaleOrder
	if err := db.First(&old, order.ID).Error; err != nil {
		return err
	}
	if old.Status != jxc.SaleStatusPending {
		return errors.New("仅待出库状态的销售单可以修改")
	}
	// 退货/换货必须关联合法原单
	if order.OrderType != jxc.SaleTypeNormal {
		if err := s.checkOriginalOrder(db, order.OrderType, order.OriginalOrderID); err != nil {
			return err
		}
		if err := s.checkInboundLimit(db, order.OrderType, *order.OriginalOrderID, order.Items, order.ID); err != nil {
			return err
		}
	}
	total, err := s.fillItemSnapshot(db, order.OrderType, order.Items)
	if err != nil {
		return err
	}
	order.TotalAmount = total
	order.OrderNo = old.OrderNo
	order.Status = old.Status

	return db.Transaction(func(tx *gorm.DB) error {
		// 解锁旧明细
		if old.OrderType == jxc.SaleTypeNormal {
			var oldItems []jxc.SaleItem
			if err := tx.Where("sale_id = ?", order.ID).Find(&oldItems).Error; err != nil {
				return err
			}
			for i := range oldItems {
				if err := s.unlockStock(tx, old.WarehouseID, oldItems[i].SkuID, oldItems[i].Qty); err != nil {
					return err
				}
			}
		}
		if err := tx.Omit("created_at").Save(order).Error; err != nil {
			return err
		}
		if err := tx.Where("sale_id = ?", order.ID).Delete(&jxc.SaleItem{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&order.Items).Error; err != nil {
			return err
		}
		// 锁定新明细
		if order.OrderType == jxc.SaleTypeNormal {
			for i := range order.Items {
				if err := s.lockStock(tx, order.WarehouseID, order.Items[i].SkuID, order.Items[i].Qty); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DeleteSaleOrder 删除销售单（仅待出库, 释放锁定）
func (s *SaleService) DeleteSaleOrder(ctx context.Context, id uint) error {
	db := global.GVA_DB.WithContext(ctx)
	var old jxc.SaleOrder
	if err := db.First(&old, id).Error; err != nil {
		return err
	}
	if old.Status != jxc.SaleStatusPending {
		return errors.New("仅待出库状态的销售单可以删除")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if old.OrderType == jxc.SaleTypeNormal {
			var items []jxc.SaleItem
			if err := tx.Where("sale_id = ?", id).Find(&items).Error; err != nil {
				return err
			}
			for i := range items {
				if err := s.unlockStock(tx, old.WarehouseID, items[i].SkuID, items[i].Qty); err != nil {
					return err
				}
			}
		}
		if err := tx.Delete(&jxc.SaleOrder{}, id).Error; err != nil {
			return err
		}
		return tx.Where("sale_id = ?", id).Delete(&jxc.SaleItem{}).Error
	})
}

// ========== 状态流转 ==========

// CancelSaleOrder 取消销售单（待出库 → 已取消, 释放锁定）
func (s *SaleService) CancelSaleOrder(ctx context.Context, id uint) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.SaleOrder
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.Status != jxc.SaleStatusPending {
			return errors.New("仅待出库状态的销售单可以取消")
		}
		if order.OrderType == jxc.SaleTypeNormal {
			var items []jxc.SaleItem
			if err := tx.Where("sale_id = ?", id).Find(&items).Error; err != nil {
				return err
			}
			for i := range items {
				if err := s.unlockStock(tx, order.WarehouseID, items[i].SkuID, items[i].Qty); err != nil {
					return err
				}
			}
		}
		return tx.Model(&jxc.SaleOrder{}).Where("id = ?", id).
			Update("status", jxc.SaleStatusCanceled).Error
	})
}

// ConfirmOut 确认出库（待出库 → 已出库, 扣减库存+流水）
func (s *SaleService) ConfirmOut(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.SaleOrder
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.Status != jxc.SaleStatusPending {
			return errors.New("仅待出库状态的销售单可以出库")
		}
		var items []jxc.SaleItem
		if err := tx.Where("sale_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		for i := range items {
			it := &items[i]
			var stock jxc.Stock
			if err := tx.Where("warehouse_id = ? AND sku_id = ?", order.WarehouseID, it.SkuID).First(&stock).Error; err != nil {
				return errors.New("库存记录不存在")
			}
			if stock.Quantity < it.Qty {
				return fmt.Errorf("库存不足：SKU %s 现有 %d，需 %d", it.SkuCode, stock.Quantity, it.Qty)
			}
			// 更新前捕获数量（GORM Updates 会回填 struct）
			beforeQty := stock.Quantity
			// 扣减数量与锁定
			if err := tx.Model(&stock).
				Updates(map[string]interface{}{"quantity": stock.Quantity - it.Qty, "lock_quantity": stock.LockQuantity - it.Qty}).Error; err != nil {
				return err
			}
			// 流水
			log := jxc.StockLog{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID,
				BusinessType: "sale_out", BusinessNo: order.OrderNo,
				BeforeQty: beforeQty, ChangeQty: -it.Qty, AfterQty: beforeQty - it.Qty,
				Operator: operator,
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}
		return tx.Model(&jxc.SaleOrder{}).Where("id = ?", id).
			Update("status", jxc.SaleStatusShipped).Error
	})
}

// ConfirmReturn 确认退货入库（退货单 → 已出库, 恢复库存+流水）
func (s *SaleService) ConfirmReturn(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.SaleOrder
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.OrderType != jxc.SaleTypeReturn {
			return errors.New("仅退货单可以执行退货入库")
		}
		if order.Status != jxc.SaleStatusPending {
			return errors.New("仅待出库状态的退货单可以入库")
		}
		var items []jxc.SaleItem
		if err := tx.Where("sale_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		for i := range items {
			it := &items[i]
			var stock jxc.Stock
			err := tx.Where("warehouse_id = ? AND sku_id = ?", order.WarehouseID, it.SkuID).First(&stock).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			beforeQty := stock.Quantity
			if err == gorm.ErrRecordNotFound {
				stock = jxc.Stock{WarehouseID: order.WarehouseID, SkuID: it.SkuID, Quantity: it.Qty}
				if err := tx.Create(&stock).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Model(&stock).Update("quantity", stock.Quantity+it.Qty).Error; err != nil {
					return err
				}
			}
			log := jxc.StockLog{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID,
				BusinessType: "sale_return", BusinessNo: order.OrderNo,
				BeforeQty: beforeQty, ChangeQty: it.Qty, AfterQty: beforeQty + it.Qty,
				Operator: operator,
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}
		return tx.Model(&jxc.SaleOrder{}).Where("id = ?", id).
			Update("status", jxc.SaleStatusShipped).Error
	})
}

// ConfirmExchange 确认换货（换货单 → 已完成: 负明细换出扣库存, 正明细换入加库存, 联合确认）
func (s *SaleService) ConfirmExchange(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.SaleOrder
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.OrderType != jxc.SaleTypeExchange {
			return errors.New("仅换货单可以执行换货确认")
		}
		if order.Status != jxc.SaleStatusPending {
			return errors.New("仅待确认状态的换货单可以执行换货确认")
		}
		var items []jxc.SaleItem
		if err := tx.Where("sale_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		for i := range items {
			it := &items[i]
			var stock jxc.Stock
			err := tx.Where("warehouse_id = ? AND sku_id = ?", order.WarehouseID, it.SkuID).First(&stock).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// 更新前捕获数量（GORM Update 会回填 struct）
			beforeQty := stock.Quantity
			switch it.Direction {
			case 1:
				// 换出明细：扣减库存（出为正）
				if err == gorm.ErrRecordNotFound || stock.Quantity < it.Qty {
					return fmt.Errorf("库存不足：SKU %s 需换出 %d", it.SkuCode, it.Qty)
				}
				if err := tx.Model(&stock).Update("quantity", stock.Quantity-it.Qty).Error; err != nil {
					return err
				}
			case 2:
				// 换入明细：增加库存（入为负）
				if err == gorm.ErrRecordNotFound {
					beforeQty = 0
					stock = jxc.Stock{WarehouseID: order.WarehouseID, SkuID: it.SkuID, Quantity: it.Qty}
					if err := tx.Create(&stock).Error; err != nil {
						return err
					}
				} else {
					if err := tx.Model(&stock).Update("quantity", stock.Quantity+it.Qty).Error; err != nil {
						return err
					}
				}
			default:
				return errors.New("换货明细方向缺失")
			}
			// 流水（换出 ChangeQty 为负, 换入为正）
			businessType := "sale_exchange_out"
			changeQty := -it.Qty
			if it.Direction == 2 {
				businessType = "sale_exchange_in"
				changeQty = it.Qty
			}
			log := jxc.StockLog{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID,
				BusinessType: businessType, BusinessNo: order.OrderNo,
				BeforeQty: beforeQty, ChangeQty: changeQty, AfterQty: beforeQty + changeQty,
				Operator: operator,
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}
		return tx.Model(&jxc.SaleOrder{}).Where("id = ?", id).
			Update("status", jxc.SaleStatusShipped).Error
	})
}

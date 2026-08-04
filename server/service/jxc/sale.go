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

// fillItemSnapshot 填充明细快照并汇总金额（单价以 SKU 销售价为准，不允许单据自定义）
func (s *SaleService) fillItemSnapshot(db *gorm.DB, items []jxc.SaleItem) (float64, error) {
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
	total, err := s.fillItemSnapshot(db, order.Items)
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
	total, err := s.fillItemSnapshot(db, order.Items)
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

// ConfirmExchange 确认换货（换货单 → 已出库: 类型3扣库存, 类型4加库存）
func (s *SaleService) ConfirmExchange(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.SaleOrder
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.OrderType != jxc.SaleTypeExOut && order.OrderType != jxc.SaleTypeExIn {
			return errors.New("仅换货单可以执行换货确认")
		}
		if order.Status != jxc.SaleStatusPending {
			return errors.New("仅待出库状态的换货单可以确认")
		}
		var items []jxc.SaleItem
		if err := tx.Where("sale_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		change := 1
		businessType := "sale_exchange_in"
		if order.OrderType == jxc.SaleTypeExOut {
			change = -1
			businessType = "sale_exchange_out"
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
			if change < 0 {
				// 换出：扣减库存
				if err == gorm.ErrRecordNotFound || stock.Quantity < it.Qty {
					return fmt.Errorf("库存不足：SKU %s 需 %d", it.SkuCode, it.Qty)
				}
				if err := tx.Model(&stock).Update("quantity", stock.Quantity-it.Qty).Error; err != nil {
					return err
				}
			} else {
				// 换入：增加库存
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
			}
			// 流水
			log := jxc.StockLog{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID,
				BusinessType: businessType, BusinessNo: order.OrderNo,
				BeforeQty: beforeQty, ChangeQty: change * it.Qty, AfterQty: beforeQty + change*it.Qty,
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

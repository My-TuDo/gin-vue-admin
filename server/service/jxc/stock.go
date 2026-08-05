package jxc

import (
	"context"
	"errors"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// StockService 库存中心服务
type StockService struct{}

// GetStockPage 库存分页查询（按仓库/关键词/预警过滤）
func (s *StockService) GetStockPage(ctx context.Context, q jxc.StockQuery) (list []jxc.Stock, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.Stock{})
	if q.WarehouseID > 0 {
		db = db.Where("stock.warehouse_id = ?", q.WarehouseID)
	}
	if q.LowStock {
		db = db.Where("stock.quantity <= stock.warning_quantity")
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		db = db.Joins("JOIN goods_sku ON goods_sku.id = stock.sku_id").
			Joins("JOIN goods ON goods.id = goods_sku.goods_id").
			Where("goods_sku.sku_code LIKE ? OR goods.name LIKE ?", kw, kw)
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	limit, offset := q.LimitOffset()
	err = db.Preload("Sku.Goods").
		Limit(limit).Offset(offset).Order("stock.warehouse_id asc, stock.sku_id asc").
		Find(&list).Error
	return
}

// StockLogQuery 流水查询参数
// GetStockLogPage 库存流水分页查询
func (s *StockService) GetStockLogPage(ctx context.Context, q jxc.StockLogQuery) (list []jxc.StockLog, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.StockLog{})
	if q.WarehouseID > 0 {
		db = db.Where("warehouse_id = ?", q.WarehouseID)
	}
	if q.SkuID > 0 {
		db = db.Where("sku_id = ?", q.SkuID)
	}
	if q.BusinessType != "" {
		db = db.Where("business_type = ?", q.BusinessType)
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	limit, offset := q.LimitOffset()
	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return
}

// DirectIn 直接入库（不关联采购单, 如赠品入库）
func (s *StockService) DirectIn(ctx context.Context, warehouseID, skuID uint, qty int, remark, operator string) error {
	if qty <= 0 {
		return errors.New("入库数量必须大于 0")
	}
	var sku jxc.GoodsSku
	if err := global.GVA_DB.WithContext(ctx).First(&sku, skuID).Error; err != nil {
		return errors.New("SKU 不存在")
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stock jxc.Stock
		err := tx.Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).First(&stock).Error
		beforeQty := stock.Quantity
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == gorm.ErrRecordNotFound {
			beforeQty = 0
			stock = jxc.Stock{WarehouseID: warehouseID, SkuID: skuID, Quantity: qty}
			if err := tx.Create(&stock).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&stock).Update("quantity", stock.Quantity+qty).Error; err != nil {
				return err
			}
		}
		log := jxc.StockLog{
			WarehouseID: warehouseID, SkuID: skuID,
			BusinessType: "direct_in", BusinessNo: "DIRECT-IN",
			BeforeQty: beforeQty, ChangeQty: qty, AfterQty: beforeQty + qty,
			Operator: operator, Remark: remark,
		}
		return tx.Create(&log).Error
	})
}

// DirectOut 直接出库（不关联销售单, 如损耗出库）
func (s *StockService) DirectOut(ctx context.Context, warehouseID, skuID uint, qty int, remark, operator string) error {
	if qty <= 0 {
		return errors.New("出库数量必须大于 0")
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stock jxc.Stock
		if err := tx.Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).First(&stock).Error; err != nil {
			return errors.New("该 SKU 在该仓库无库存记录")
		}
		if stock.Quantity < qty {
			return fmt.Errorf("库存不足：现有 %d，需出 %d", stock.Quantity, qty)
		}
		beforeQty := stock.Quantity
		if err := tx.Model(&stock).Update("quantity", stock.Quantity-qty).Error; err != nil {
			return err
		}
		log := jxc.StockLog{
			WarehouseID: warehouseID, SkuID: skuID,
			BusinessType: "direct_out", BusinessNo: "DIRECT-OUT",
			BeforeQty: beforeQty, ChangeQty: -qty, AfterQty: beforeQty - qty,
			Operator: operator, Remark: remark,
		}
		return tx.Create(&log).Error
	})
}

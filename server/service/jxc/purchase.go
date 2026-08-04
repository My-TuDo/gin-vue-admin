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

// PurchaseService 采购中心服务
type PurchaseService struct{}

// ========== 采购单 ==========

// genOrderNo 生成采购单号 PO-YYYYMMDD-XXX（当日序号自增）
func (s *PurchaseService) genOrderNo(db *gorm.DB) (string, error) {
	date := time.Now().Format("20060102")
	like := "PO-" + date + "%"
	var maxNo string
	if err := db.Model(&jxc.PurchaseOrder{}).Where("order_no LIKE ?", like).
		Order("order_no DESC").Limit(1).Pluck("order_no", &maxNo).Error; err != nil {
		return "", err
	}
	if maxNo == "" {
		return "PO-" + date + "-001", nil
	}
	seq, err := strconv.Atoi(maxNo[len(maxNo)-3:])
	if err != nil {
		return "", fmt.Errorf("解析单号序号失败: %s", maxNo)
	}
	return fmt.Sprintf("PO-%s-%03d", date, seq+1), nil
}

// CreatePurchase 创建采购单（状态=待审核, 明细写入, 金额自动汇总）
func (s *PurchaseService) CreatePurchase(ctx context.Context, order *jxc.PurchaseOrder) error {
	if len(order.Items) == 0 {
		return errors.New("采购单至少需要一条明细")
	}
	db := global.GVA_DB.WithContext(ctx)
	orderNo, err := s.genOrderNo(db)
	if err != nil {
		return err
	}
	order.OrderNo = orderNo
	order.Status = jxc.PurchaseStatusPending

	var total float64
	for i := range order.Items {
		it := &order.Items[i]
		if it.Qty <= 0 {
			return errors.New("采购数量必须大于 0")
		}
		if it.Price < 0 {
			return errors.New("采购单价不能为负")
		}
		it.Amount = float64(it.Qty) * it.Price
		total += it.Amount
		// 快照 SKU 信息
		var sku jxc.GoodsSku
		if err := db.First(&sku, it.SkuID).Error; err != nil {
			return errors.New("SKU 不存在")
		}
		it.SkuCode = sku.SkuCode
		it.Color = sku.Color
		it.Size = sku.Size
		var goods jxc.Goods
		db.First(&goods, sku.GoodsID)
		it.GoodsName = goods.Name
	}
	order.TotalAmount = total
	return db.Create(order).Error // 级联写入明细
}

// GetPurchasePage 分页查询采购单列表
func (s *PurchaseService) GetPurchasePage(ctx context.Context, info request.PageInfo) (list []jxc.PurchaseOrder, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.PurchaseOrder{})
	if info.Keyword != "" {
		db = db.Where("order_no LIKE ?", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	limit, offset := info.LimitOffset()
	err = db.Preload("Supplier").Preload("Warehouse").
		Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return
}

// GetPurchaseDetail 查询采购单详情（含明细）
func (s *PurchaseService) GetPurchaseDetail(ctx context.Context, id uint) (order jxc.PurchaseOrder, err error) {
	err = global.GVA_DB.WithContext(ctx).
		Preload("Supplier").Preload("Warehouse").Preload("Items").
		First(&order, id).Error
	return
}

// UpdatePurchase 更新采购单（仅待审核可改, 明细全量重建）
func (s *PurchaseService) UpdatePurchase(ctx context.Context, order *jxc.PurchaseOrder) error {
	if len(order.Items) == 0 {
		return errors.New("采购单至少需要一条明细")
	}
	db := global.GVA_DB.WithContext(ctx)
	var old jxc.PurchaseOrder
	if err := db.First(&old, order.ID).Error; err != nil {
		return err
	}
	if old.Status != jxc.PurchaseStatusPending {
		return errors.New("仅待审核状态的采购单可以修改")
	}
	var total float64
	for i := range order.Items {
		it := &order.Items[i]
		if it.Qty <= 0 {
			return errors.New("采购数量必须大于 0")
		}
		it.PurchaseID = order.ID
		it.Amount = float64(it.Qty) * it.Price
		total += it.Amount
	}
	order.TotalAmount = total
	order.OrderNo = old.OrderNo
	order.Status = old.Status

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("created_at").Save(order).Error; err != nil {
			return err
		}
		if err := tx.Where("purchase_id = ?", order.ID).Delete(&jxc.PurchaseItem{}).Error; err != nil {
			return err
		}
		return tx.Create(&order.Items).Error
	})
}

// DeletePurchase 删除采购单（仅待审核可删, 软删除 + 级联删明细）
func (s *PurchaseService) DeletePurchase(ctx context.Context, id uint) error {
	db := global.GVA_DB.WithContext(ctx)
	var old jxc.PurchaseOrder
	if err := db.First(&old, id).Error; err != nil {
		return err
	}
	if old.Status != jxc.PurchaseStatusPending {
		return errors.New("仅待审核状态的采购单可以删除")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&jxc.PurchaseOrder{}, id).Error; err != nil {
			return err
		}
		return tx.Where("purchase_id = ?", id).Delete(&jxc.PurchaseItem{}).Error
	})
}

// AuditPurchase 审核采购单（待审核 → 已审核）
func (s *PurchaseService) AuditPurchase(ctx context.Context, id uint) error {
	res := global.GVA_DB.WithContext(ctx).Model(&jxc.PurchaseOrder{}).
		Where("id = ? AND status = ?", id, jxc.PurchaseStatusPending).
		Update("status", jxc.PurchaseStatusAudited)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("仅待审核状态的采购单可以审核")
	}
	return nil
}

// CancelPurchase 取消采购单（待审核/已审核 → 已取消）
func (s *PurchaseService) CancelPurchase(ctx context.Context, id uint) error {
	res := global.GVA_DB.WithContext(ctx).Model(&jxc.PurchaseOrder{}).
		Where("id = ? AND status IN ?", id, []int8{jxc.PurchaseStatusPending, jxc.PurchaseStatusAudited}).
		Update("status", jxc.PurchaseStatusCanceled)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("当前状态不可取消（仅待审核/已审核可取消）")
	}
	return nil
}

// StockIn 采购入库（已审核 → 已入库, 事务内增加库存 + 写流水）
func (s *PurchaseService) StockIn(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.PurchaseOrder
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.Status != jxc.PurchaseStatusAudited {
			return errors.New("仅已审核状态的采购单可以入库")
		}
		var items []jxc.PurchaseItem
		if err := tx.Where("purchase_id = ?", id).Find(&items).Error; err != nil {
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
				// 首次入库：创建库存记录
				stock = jxc.Stock{WarehouseID: order.WarehouseID, SkuID: it.SkuID, Quantity: it.Qty}
				if err := tx.Create(&stock).Error; err != nil {
					return err
				}
			} else {
				// 已有库存：增加数量
				if err := tx.Model(&stock).Update("quantity", stock.Quantity+it.Qty).Error; err != nil {
					return err
				}
			}
			// 写库存流水（只追加）
			log := jxc.StockLog{
				WarehouseID:  order.WarehouseID,
				SkuID:        it.SkuID,
				BusinessType: "purchase_in",
				BusinessNo:   order.OrderNo,
				BeforeQty:    beforeQty,
				ChangeQty:    it.Qty,
				AfterQty:     beforeQty + it.Qty,
				Operator:     operator,
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
			// 回写 SKU 成本价（最新进价法）
			if err := tx.Model(&jxc.GoodsSku{}).Where("id = ?", it.SkuID).
				Update("cost_price", it.Price).Error; err != nil {
				return err
			}
		}
		return tx.Model(&jxc.PurchaseOrder{}).Where("id = ?", id).
			Update("status", jxc.PurchaseStatusStockIn).Error
	})
}

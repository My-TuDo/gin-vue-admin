package jxc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// PosScanService 扫码枪队列服务
type PosScanService struct{}

// CreateScan 小程序扫码上架：按条码/SKU编码 查 SKU 并入待处理队列。
// 同一 SKU 已有待处理记录时数量累加（连续扫同款合并）。
func (s *PosScanService) CreateScan(ctx context.Context, barcode string, skuID uint, qty int) (*jxc.PosScan, error) {
	if qty <= 0 {
		qty = 1
	}
	if qty > 99 {
		return nil, errors.New("单次扫码数量不能超过 99")
	}
	db := global.GVA_DB.WithContext(ctx)

	var sku jxc.GoodsSku
	if skuID > 0 {
		if err := db.First(&sku, skuID).Error; err != nil {
			return nil, errors.New("SKU 不存在")
		}
	} else {
		keyword := strings.TrimSpace(barcode)
		if keyword == "" {
			return nil, errors.New("请提供条码或 SKU 编码")
		}
		if err := db.Where("barcode = ? OR sku_code = ?", keyword, keyword).First(&sku).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("未找到该条码对应的商品")
			}
			return nil, err
		}
	}

	// 同 SKU 待处理记录数量累加
	var scan jxc.PosScan
	err := db.Where("sku_id = ? AND status = 0", sku.ID).First(&scan).Error
	if err == nil {
		scan.Qty += qty
		if err := db.Save(&scan).Error; err != nil {
			return nil, err
		}
		return &scan, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	scan = jxc.PosScan{SkuID: sku.ID, Qty: qty, Status: 0}
	if err := db.Create(&scan).Error; err != nil {
		return nil, err
	}
	return &scan, nil
}

// ListPending 待处理扫码条目（PC 收银台轮询），附带 SKU 摘要
func (s *PosScanService) ListPending(ctx context.Context) ([]jxc.PosScanDetail, error) {
	db := global.GVA_DB.WithContext(ctx)
	var list []jxc.PosScan
	if err := db.Where("status = 0").Order("created_at ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	details := make([]jxc.PosScanDetail, 0, len(list))
	for i := range list {
		d := jxc.PosScanDetail{PosScan: list[i]}
		var sku jxc.GoodsSku
		if err := db.First(&sku, list[i].SkuID).Error; err == nil {
			var goods jxc.Goods
			goodsName := ""
			if err := db.First(&goods, sku.GoodsID).Error; err == nil {
				goodsName = goods.Name
			}
			d.Sku = &jxc.SkuBrief{
				ID:        sku.ID,
				SkuCode:   sku.SkuCode,
				GoodsName: goodsName,
				Color:     sku.Color,
				Size:      sku.Size,
				SalePrice: sku.SalePrice,
				Barcode:   sku.Barcode,
			}
		}
		details = append(details, d)
	}
	return details, nil
}

// ConfirmScan PC 收银台确认：批量标记已处理（加入购物车后）
func (s *PosScanService) ConfirmScan(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return errors.New("请选择要确认的扫码条目")
	}
	now := time.Now()
	return global.GVA_DB.WithContext(ctx).Model(&jxc.PosScan{}).
		Where("id IN ? AND status = 0", ids).
		Updates(map[string]interface{}{"status": 1, "handled_at": &now}).Error
}

package jxc

import (
	"context"
	"errors"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

type GoodsService struct{}

// ========== 商品 SPU ==========

// GetGoodsPage 分页查询商品列表（含分类/品牌）
func (s *GoodsService) GetGoodsPage(ctx context.Context, info request.PageInfo) (list []jxc.Goods, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.Goods{})
	if info.Keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+info.Keyword+"%", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	limit, offset := info.LimitOffset()
	err = db.Preload("Category").Preload("Brand").Limit(limit).Offset(offset).Order("id DESC").Find(&list).Error
	return
}

// GetGoodsDetail 查询商品详情（含 SKU 列表）
func (s *GoodsService) GetGoodsDetail(ctx context.Context, id uint) (goods jxc.Goods, err error) {
	err = global.GVA_DB.WithContext(ctx).Preload("Category").Preload("Brand").Preload("Skus").Find(&goods, id).Error
	return
}

// GetAllGoods 获取所有上架商品
func (s *GoodsService) GetAllGoods(ctx context.Context) (list []jxc.Goods, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("status = ?", 1).Find(&list).Error
	return
}

// CreateGoods 创建商品（编码自动生成）
func (s *GoodsService) CreateGoods(ctx context.Context, g *jxc.Goods) error {
	db := global.GVA_DB.WithContext(ctx)
	if g.Code == "" {
		code, err := genCode(db, &jxc.Goods{}, prefixGoods)
		if err != nil {
			return err
		}
		g.Code = code
	}
	return db.Create(g).Error
}

// UpdateGoods 更新商品
func (s *GoodsService) UpdateGoods(ctx context.Context, g *jxc.Goods) error {
	return global.GVA_DB.WithContext(ctx).Omit("code", "created_at", "status").Save(g).Error
}

// DeleteGoods 删除商品
func (s *GoodsService) DeleteGoods(ctx context.Context, id uint) error {
	db := global.GVA_DB.WithContext(ctx)
	var skuCount int64
	db.Model(&jxc.GoodsSku{}).Where("goods_id = ?", id).Count(&skuCount)
	if skuCount > 0 {
		return errors.New("该商品存在 SKU, 请先删除 SKU")
	}
	return db.Delete(&jxc.Goods{}, id).Error
}

// DeleteGoodsForever 彻底删除商品（物理删除，不可恢复；有 SKU 仍拒绝）
func (s *GoodsService) DeleteGoodsForever(ctx context.Context, id uint) error {
	db := global.GVA_DB.WithContext(ctx)
	var skuCount int64
	db.Model(&jxc.GoodsSku{}).Where("goods_id = ?", id).Count(&skuCount)
	if skuCount > 0 {
		return errors.New("该商品存在 SKU, 请先删除 SKU")
	}
	return db.Unscoped().Delete(&jxc.Goods{}, id).Error
}

// SetGoodsStatus 设置商品状态（上架/下架）
func (s *GoodsService) SetGoodsStatus(ctx context.Context, id uint, status int8) error {
	return global.GVA_DB.WithContext(ctx).Model(&jxc.Goods{}).Where("id = ?", id).Update("status", status).Error
}

// ========== 商品 SKU ==========

// GetSkuList 查询SKU列表（goodsId 为 0 时返回全部, 供采购/销售选品）
func (s *GoodsService) GetSkuList(ctx context.Context, goodsId uint) (list []jxc.GoodsSku, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.GoodsSku{}).Where("goods_id > 0")
	if goodsId > 0 {
		db = db.Where("goods_id = ?", goodsId)
	}
	err = db.Preload("Goods").Order("id asc").Find(&list).Error
	return
}

// CreateSku 创建 SKU（编码自动生成: 商品款号-颜色-尺码, 校验同商品组合唯一）
func (s *GoodsService) CreateSku(ctx context.Context, sku *jxc.GoodsSku) error {
	db := global.GVA_DB.WithContext(ctx)
	if sku.GoodsID == 0 {
		return errors.New("请选择所属商品")
	}
	if sku.SkuCode == "" {
		var goods jxc.Goods
		if err := db.First(&goods, sku.GoodsID).Error; err != nil {
			return err
		}
		sku.SkuCode = strings.ToUpper(goods.Code + "-" + sku.Color + "-" + sku.Size)
	}
	var cnt int64
	db.Model(&jxc.GoodsSku{}).Where("goods_id = ? AND color = ? AND size = ?", sku.GoodsID, sku.Color, sku.Size).Count(&cnt)
	if cnt > 0 {
		return errors.New("该商品下已存在相同颜色加尺码的 SKU")
	}
	return db.Create(sku).Error
}

// UpdateSku 更新 SKU （归属与编码不可修改）
func (s *GoodsService) UpdateSku(ctx context.Context, sku *jxc.GoodsSku) error {
	return global.GVA_DB.WithContext(ctx).Omit("goods_id", "sku_code", "created_at", "status").Save(sku).Error
}

// GetSkuByBarcode 按条码/SKU编码查 SKU（含各仓库库存，供小程序扫码查库存）
func (s *GoodsService) GetSkuByBarcode(ctx context.Context, keyword string) (*jxc.GoodsSku, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, errors.New("请输入条码或SKU编码")
	}
	db := global.GVA_DB.WithContext(ctx)
	var sku jxc.GoodsSku
	err := db.Preload("Goods").Where("barcode = ? OR sku_code = ?", keyword, keyword).First(&sku).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("未找到该条码/编码对应的SKU")
		}
		return nil, err
	}
	var rows []jxc.SkuStockRow
	if err := db.Table("stock s").
		Select("s.warehouse_id AS warehouse_id, w.name AS warehouse_name, s.quantity AS quantity, s.lock_quantity AS lock_quantity, (s.quantity - s.lock_quantity) AS available").
		Joins("LEFT JOIN warehouse w ON w.id = s.warehouse_id").
		Where("s.sku_id = ?", sku.ID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	sku.Stocks = rows
	return &sku, nil
}

// DeleteSku 删除 SKU
func (s *GoodsService) DeleteSku(ctx context.Context, id uint) error {
	return global.GVA_DB.WithContext(ctx).Delete(&jxc.GoodsSku{}, id).Error
}

// DeleteSkuForever 彻底删除 SKU（物理删除，不可恢复）
func (s *GoodsService) DeleteSkuForever(ctx context.Context, id uint) error {
	return global.GVA_DB.WithContext(ctx).Unscoped().Delete(&jxc.GoodsSku{}, id).Error
}

// SetSkuStatus 设置 SKU 状态 （启用/停用）
func (s *GoodsService) SetSkuStatus(ctx context.Context, id uint, status int8) error {
	return global.GVA_DB.WithContext(ctx).Model(&jxc.GoodsSku{}).Where("id = ?", id).Update("status", status).Error
}

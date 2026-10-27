package jxc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
	jxcResp "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/response"
	"gorm.io/gorm"
)

// MasterService 基础资料（仓库 / 商品规格 / 往来单位）。
//
// 为什么单独抽出来：原实现把 DB 查询和绑定直接写在 API handler 里，
// 既违反 Router -> API -> Service -> Model 的分层，也让客户端可以把
// ID / status / 时间字段一起 POST 进来（mass assignment）。
// 这里统一收口：入参只认 DTO，状态与主键由服务端决定。
type MasterService struct{}

// ---------- 仓库 ----------

func (s *MasterService) ListWarehouses(ctx context.Context, activeOnly bool) ([]jxc.Warehouse, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.Warehouse{})
	if activeOnly {
		db = db.Where("status = ?", 1)
	}
	var list []jxc.Warehouse
	err := db.Order("id").Find(&list).Error
	return list, err
}

func (s *MasterService) CreateWarehouse(ctx context.Context, in jxcReq.WarehouseInput) (*jxc.Warehouse, error) {
	code, name, err := normalizeCodeName(in.Code, in.Name)
	if err != nil {
		return nil, err
	}
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureCodeFree(db, &jxc.Warehouse{}, code, 0); err != nil {
		return nil, err
	}
	w := jxc.Warehouse{Code: code, Name: name, Status: 1}
	if err := db.Create(&w).Error; err != nil {
		if isDuplicateKeyErr(err) {
			return nil, fmt.Errorf("仓库编码已存在：%s", code)
		}
		return nil, err
	}
	return &w, nil
}

func (s *MasterService) UpdateWarehouse(ctx context.Context, id uint, in jxcReq.WarehouseInput) (*jxc.Warehouse, error) {
	if id == 0 {
		return nil, errors.New("仓库 ID 不合法")
	}
	code, name, err := normalizeCodeName(in.Code, in.Name)
	if err != nil {
		return nil, err
	}
	db := global.GVA_DB.WithContext(ctx)
	var w jxc.Warehouse
	if err := db.First(&w, id).Error; err != nil {
		return nil, errors.New("仓库不存在")
	}
	if err := ensureCodeFree(db, &jxc.Warehouse{}, code, id); err != nil {
		return nil, err
	}
	if err := db.Model(&w).Updates(map[string]interface{}{"code": code, "name": name}).Error; err != nil {
		if isDuplicateKeyErr(err) {
			return nil, fmt.Errorf("仓库编码已存在：%s", code)
		}
		return nil, err
	}
	return s.getWarehouse(ctx, id)
}

func (s *MasterService) SetWarehouseStatus(ctx context.Context, id uint, status int8) error {
	status, err := normalizeStatus(status)
	if err != nil {
		return err
	}
	return s.updateStatus(ctx, &jxc.Warehouse{}, id, status, "仓库")
}

func (s *MasterService) getWarehouse(ctx context.Context, id uint) (*jxc.Warehouse, error) {
	var w jxc.Warehouse
	if err := global.GVA_DB.WithContext(ctx).First(&w, id).Error; err != nil {
		return nil, errors.New("仓库不存在")
	}
	return &w, nil
}

// ---------- 商品与规格 ----------

func (s *MasterService) ListGoods(ctx context.Context, activeOnly bool) ([]jxcResp.GoodsWithSkus, error) {
	db := global.GVA_DB.WithContext(ctx)
	gq := db.Model(&jxc.Goods{}).Order("id desc").Limit(500)
	sq := db.Model(&jxc.GoodsSku{}).Order("id")
	if activeOnly {
		gq = gq.Where("status = ?", 1)
		sq = sq.Where("status = ?", 1)
	}
	var goods []jxc.Goods
	if err := gq.Find(&goods).Error; err != nil {
		return nil, err
	}
	var skus []jxc.GoodsSku
	if err := sq.Find(&skus).Error; err != nil {
		return nil, err
	}
	byGoods := map[uint][]jxc.GoodsSku{}
	for _, sku := range skus {
		byGoods[sku.GoodsID] = append(byGoods[sku.GoodsID], sku)
	}
	out := make([]jxcResp.GoodsWithSkus, 0, len(goods))
	for _, g := range goods {
		out = append(out, jxcResp.GoodsWithSkus{Goods: g, Skus: byGoods[g.ID]})
	}
	return out, nil
}

func (s *MasterService) ListSkus(ctx context.Context, goodsID uint, activeOnly bool) ([]jxc.GoodsSku, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.GoodsSku{})
	if goodsID > 0 {
		db = db.Where("goods_id = ?", goodsID)
	}
	if activeOnly {
		db = db.Where("status = ?", 1)
	}
	var list []jxc.GoodsSku
	err := db.Order("id").Limit(1000).Find(&list).Error
	return list, err
}

func (s *MasterService) CreateGoods(ctx context.Context, in jxcReq.GoodsInput) (*jxcResp.GoodsWithSkus, error) {
	code, name, err := normalizeCodeName(in.Code, in.Name)
	if err != nil {
		return nil, err
	}
	if len(in.Skus) == 0 {
		return nil, errors.New("至少需要一个规格（颜色×尺码）")
	}
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureCodeFree(db, &jxc.Goods{}, code, 0); err != nil {
		return nil, err
	}
	status, err := normalizeStatus(statusOrDefault(in.Status))
	if err != nil {
		return nil, err
	}

	seenSpec := map[string]bool{}
	skus := make([]jxc.GoodsSku, 0, len(in.Skus))
	for i := range in.Skus {
		sku, err := buildNewSku(db, in.Skus[i])
		if err != nil {
			return nil, err
		}
		key := sku.Color + "/" + sku.Size
		if seenSpec[key] {
			return nil, fmt.Errorf("同一商品下颜色+尺码不能重复：%s", key)
		}
		seenSpec[key] = true
		skus = append(skus, sku)
	}

	goods := jxc.Goods{Code: code, Name: name, Status: status}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&goods).Error; err != nil {
			return err
		}
		for i := range skus {
			skus[i].GoodsID = goods.ID
			if err := tx.Create(&skus[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if isDuplicateKeyErr(err) {
			return nil, errors.New("商品编码或规格条码已存在")
		}
		return nil, err
	}
	return &jxcResp.GoodsWithSkus{Goods: goods, Skus: skus}, nil
}

func (s *MasterService) UpdateGoods(ctx context.Context, id uint, in jxcReq.GoodsInput) (*jxcResp.GoodsWithSkus, error) {
	if id == 0 {
		return nil, errors.New("商品 ID 不合法")
	}
	code, name, err := normalizeCodeName(in.Code, in.Name)
	if err != nil {
		return nil, err
	}
	db := global.GVA_DB.WithContext(ctx)
	var goods jxc.Goods
	if err := db.First(&goods, id).Error; err != nil {
		return nil, errors.New("商品不存在")
	}
	if err := ensureCodeFree(db, &jxc.Goods{}, code, id); err != nil {
		return nil, err
	}

	seenSpec := map[string]bool{}
	err = db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"code": code, "name": name}
		if in.Status != nil {
			status, err := normalizeStatus(*in.Status)
			if err != nil {
				return err
			}
			updates["status"] = status
		}
		if err := tx.Model(&jxc.Goods{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}

		for i := range in.Skus {
			si := &in.Skus[i]
			color := strings.TrimSpace(si.Color)
			size := strings.TrimSpace(si.Size)
			skuCode := strings.TrimSpace(si.SkuCode)
			if color == "" || size == "" {
				return errors.New("规格的颜色与尺码不能为空")
			}
			if skuCode == "" {
				return errors.New("规格条码不能为空")
			}
			if si.SalePrice < 0 || si.RefCost < 0 {
				return errors.New("售价与参考进价不能为负")
			}
			key := color + "/" + size
			if seenSpec[key] {
				return fmt.Errorf("同一商品下颜色+尺码不能重复：%s", key)
			}
			seenSpec[key] = true

			if si.ID > 0 {
				var sku jxc.GoodsSku
				if err := tx.Where("id = ? AND goods_id = ?", si.ID, id).First(&sku).Error; err != nil {
					return fmt.Errorf("规格 %d 不属于该商品", si.ID)
				}
				status := sku.Status
				if si.Status != nil {
					status, err = normalizeStatus(*si.Status)
					if err != nil {
						return err
					}
				}
				if err := ensureSkuCodeFree(tx, skuCode, si.ID); err != nil {
					return err
				}
				if err := tx.Model(&jxc.GoodsSku{}).Where("id = ?", si.ID).Updates(map[string]interface{}{
					"color": color, "size": size, "sku_code": skuCode,
					"sale_price": money.RoundYuan(si.SalePrice),
					"ref_cost":   money.RoundYuan(si.RefCost),
					"status":     status,
				}).Error; err != nil {
					return err
				}
				continue
			}

			// 新增规格：走与 CreateGoods 相同的校验与默认值
			sku, err := buildNewSku(tx, *si)
			if err != nil {
				return err
			}
			sku.GoodsID = id
			if err := tx.Create(&sku).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if isDuplicateKeyErr(err) {
			return nil, errors.New("商品编码或规格条码已存在")
		}
		return nil, err
	}
	return s.getGoods(ctx, id)
}

func (s *MasterService) SetGoodsStatus(ctx context.Context, id uint, status int8) error {
	status, err := normalizeStatus(status)
	if err != nil {
		return err
	}
	return s.updateStatus(ctx, &jxc.Goods{}, id, status, "商品")
}

func (s *MasterService) SetSkuStatus(ctx context.Context, id uint, status int8) error {
	status, err := normalizeStatus(status)
	if err != nil {
		return err
	}
	return s.updateStatus(ctx, &jxc.GoodsSku{}, id, status, "规格")
}

func (s *MasterService) getGoods(ctx context.Context, id uint) (*jxcResp.GoodsWithSkus, error) {
	db := global.GVA_DB.WithContext(ctx)
	var goods jxc.Goods
	if err := db.First(&goods, id).Error; err != nil {
		return nil, errors.New("商品不存在")
	}
	var skus []jxc.GoodsSku
	if err := db.Where("goods_id = ?", id).Order("id").Find(&skus).Error; err != nil {
		return nil, err
	}
	return &jxcResp.GoodsWithSkus{Goods: goods, Skus: skus}, nil
}

// ---------- 往来单位 ----------

func (s *MasterService) ListPartners(ctx context.Context, kind string, activeOnly bool) ([]jxc.Partner, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.Partner{})
	if kind != "" {
		if kind != jxc.PartnerSupplier && kind != jxc.PartnerCustomer {
			return nil, errors.New("类型只能是供应商或客户")
		}
		db = db.Where("kind = ?", kind)
	}
	if activeOnly {
		db = db.Where("status = ?", 1)
	}
	var list []jxc.Partner
	err := db.Order("id desc").Limit(1000).Find(&list).Error
	return list, err
}

func (s *MasterService) CreatePartner(ctx context.Context, in jxcReq.PartnerInput) (*jxc.Partner, error) {
	kind := strings.TrimSpace(in.Kind)
	if kind != jxc.PartnerSupplier && kind != jxc.PartnerCustomer {
		return nil, errors.New("类型只能是供应商或客户")
	}
	code, name, err := normalizeCodeName(in.Code, in.Name)
	if err != nil {
		return nil, err
	}
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureCodeFree(db, &jxc.Partner{}, code, 0); err != nil {
		return nil, err
	}
	p := jxc.Partner{
		Kind: kind, Code: code, Name: name,
		Contact: strings.TrimSpace(in.Contact), Phone: strings.TrimSpace(in.Phone),
		Address: strings.TrimSpace(in.Address), Remark: strings.TrimSpace(in.Remark),
		Status: 1,
	}
	if err := db.Create(&p).Error; err != nil {
		if isDuplicateKeyErr(err) {
			return nil, fmt.Errorf("往来单位编码已存在：%s", code)
		}
		return nil, err
	}
	return &p, nil
}

func (s *MasterService) UpdatePartner(ctx context.Context, id uint, in jxcReq.PartnerInput) (*jxc.Partner, error) {
	if id == 0 {
		return nil, errors.New("往来单位 ID 不合法")
	}
	code, name, err := normalizeCodeName(in.Code, in.Name)
	if err != nil {
		return nil, err
	}
	db := global.GVA_DB.WithContext(ctx)
	var p jxc.Partner
	if err := db.First(&p, id).Error; err != nil {
		return nil, errors.New("往来单位不存在")
	}
	kind := strings.TrimSpace(in.Kind)
	if kind != "" && kind != p.Kind {
		return nil, errors.New("往来单位类型不能修改")
	}
	if err := ensureCodeFree(db, &jxc.Partner{}, code, id); err != nil {
		return nil, err
	}
	if err := db.Model(&p).Updates(map[string]interface{}{
		"code": code, "name": name, "contact": strings.TrimSpace(in.Contact),
		"phone": strings.TrimSpace(in.Phone), "address": strings.TrimSpace(in.Address),
		"remark": strings.TrimSpace(in.Remark),
	}).Error; err != nil {
		if isDuplicateKeyErr(err) {
			return nil, fmt.Errorf("往来单位编码已存在：%s", code)
		}
		return nil, err
	}
	var got jxc.Partner
	if err := db.First(&got, id).Error; err != nil {
		return nil, errors.New("往来单位不存在")
	}
	return &got, nil
}

func (s *MasterService) SetPartnerStatus(ctx context.Context, id uint, status int8) error {
	status, err := normalizeStatus(status)
	if err != nil {
		return err
	}
	return s.updateStatus(ctx, &jxc.Partner{}, id, status, "往来单位")
}

// ---------- 内部工具 ----------

func normalizeCodeName(code, name string) (string, string, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return "", "", errors.New("编码与名称不能为空")
	}
	return code, name, nil
}

func normalizeStatus(status int8) (int8, error) {
	if status != 0 && status != 1 {
		return 0, errors.New("状态只能是 0（停用）或 1（启用）")
	}
	return status, nil
}

func statusOrDefault(status *int8) int8 {
	if status == nil {
		return 1
	}
	return *status
}

func ensureCodeFree(db *gorm.DB, model any, code string, excludeID uint) error {
	q := db.Model(model).Where("code = ?", code)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return fmt.Errorf("编码已存在：%s", code)
	}
	return nil
}

func ensureSkuCodeFree(db *gorm.DB, skuCode string, excludeID uint) error {
	q := db.Model(&jxc.GoodsSku{}).Where("sku_code = ?", skuCode)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return fmt.Errorf("规格条码已存在：%s", skuCode)
	}
	return nil
}

func buildNewSku(db *gorm.DB, in jxcReq.GoodsSkuInput) (jxc.GoodsSku, error) {
	color := strings.TrimSpace(in.Color)
	size := strings.TrimSpace(in.Size)
	skuCode := strings.TrimSpace(in.SkuCode)
	if color == "" || size == "" {
		return jxc.GoodsSku{}, errors.New("规格的颜色与尺码不能为空")
	}
	if skuCode == "" {
		return jxc.GoodsSku{}, errors.New("规格条码不能为空")
	}
	if in.SalePrice < 0 || in.RefCost < 0 {
		return jxc.GoodsSku{}, errors.New("售价与参考进价不能为负")
	}
	status, err := normalizeStatus(statusOrDefault(in.Status))
	if err != nil {
		return jxc.GoodsSku{}, err
	}
	if err := ensureSkuCodeFree(db, skuCode, 0); err != nil {
		return jxc.GoodsSku{}, err
	}
	return jxc.GoodsSku{
		Color: color, Size: size, SkuCode: skuCode,
		SalePrice: money.RoundYuan(in.SalePrice),
		RefCost:   money.RoundYuan(in.RefCost),
		Status:    status,
	}, nil
}

func (s *MasterService) updateStatus(ctx context.Context, model any, id uint, status int8, label string) error {
	if id == 0 {
		return fmt.Errorf("%s ID 不合法", label)
	}
	res := global.GVA_DB.WithContext(ctx).Model(model).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%s不存在", label)
	}
	return nil
}

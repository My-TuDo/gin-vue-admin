package jxc

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	jxcmodel "github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MasterApi 基础资料：仓库 / 商品 / 规格。
//
// 刻意保持最小：这是"能开单"所必需的三张表，其余（供应商、客户、品牌、分类）
// 在真正需要时再加，不提前堆模型。
type MasterApi struct{}

// ---------- 仓库 ----------

func (a *MasterApi) ListWarehouses(c *gin.Context) {
	var list []jxcmodel.Warehouse
	if err := global.GVA_DB.WithContext(c.Request.Context()).Order("id").Find(&list).Error; err != nil {
		global.GVA_LOG.Error("获取仓库失败", zap.Error(err))
		response.FailWithMessage("获取仓库失败", c)
		return
	}
	response.OkWithData(list, c)
}

func (a *MasterApi) CreateWarehouse(c *gin.Context) {
	var req jxcmodel.Warehouse
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	if req.Code == "" || req.Name == "" {
		response.FailWithMessage("仓库编码与名称不能为空", c)
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	if err := global.GVA_DB.WithContext(c.Request.Context()).Create(&req).Error; err != nil {
		response.FailWithMessage("创建仓库失败："+err.Error(), c)
		return
	}
	response.OkWithData(req, c)
}

// ---------- 商品（含规格） ----------

// GoodsWithSkus 商品 + 规格，一次返回，前端商品卡片直接可用。
type GoodsWithSkus struct {
	jxcmodel.Goods
	Skus []jxcmodel.GoodsSku `json:"skus"`
}

func (a *MasterApi) ListGoods(c *gin.Context) {
	db := global.GVA_DB.WithContext(c.Request.Context())
	var goods []jxcmodel.Goods
	if err := db.Order("id desc").Limit(500).Find(&goods).Error; err != nil {
		global.GVA_LOG.Error("获取商品失败", zap.Error(err))
		response.FailWithMessage("获取商品失败", c)
		return
	}
	var skus []jxcmodel.GoodsSku
	if err := db.Order("id").Find(&skus).Error; err != nil {
		response.FailWithMessage("获取规格失败", c)
		return
	}
	byGoods := map[uint][]jxcmodel.GoodsSku{}
	for _, s := range skus {
		byGoods[s.GoodsID] = append(byGoods[s.GoodsID], s)
	}
	out := make([]GoodsWithSkus, 0, len(goods))
	for _, g := range goods {
		out = append(out, GoodsWithSkus{Goods: g, Skus: byGoods[g.ID]})
	}
	response.OkWithData(out, c)
}

// CreateGoods 创建商品 + 规格（一次提交，事务内完成）
func (a *MasterApi) CreateGoods(c *gin.Context) {
	var req GoodsWithSkus
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	if req.Code == "" || req.Name == "" {
		response.FailWithMessage("商品编码与名称不能为空", c)
		return
	}
	if len(req.Skus) == 0 {
		response.FailWithMessage("至少需要一个规格（颜色×尺码）", c)
		return
	}
	seen := map[string]bool{}
	for _, s := range req.Skus {
		if s.SkuCode == "" {
			response.FailWithMessage("规格条码不能为空", c)
			return
		}
		key := s.Color + "/" + s.Size
		if seen[key] {
			response.FailWithMessage("同一商品下颜色+尺码不能重复："+key, c)
			return
		}
		seen[key] = true
	}
	if req.Status == 0 {
		req.Status = 1
	}

	err := global.GVA_DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&req.Goods).Error; err != nil {
			return err
		}
		for i := range req.Skus {
			req.Skus[i].GoodsID = req.Goods.ID
			if req.Skus[i].Status == 0 {
				req.Skus[i].Status = 1
			}
			if err := tx.Create(&req.Skus[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		response.FailWithMessage("创建商品失败："+err.Error(), c)
		return
	}
	response.OkWithData(req, c)
}

// ListSkus 规格列表（可按商品过滤）
func (a *MasterApi) ListSkus(c *gin.Context) {
	goodsID, _ := strconv.ParseUint(c.DefaultQuery("goodsId", "0"), 10, 64)
	db := global.GVA_DB.WithContext(c.Request.Context()).Model(&jxcmodel.GoodsSku{})
	if goodsID > 0 {
		db = db.Where("goods_id = ?", goodsID)
	}
	var list []jxcmodel.GoodsSku
	if err := db.Order("id").Limit(1000).Find(&list).Error; err != nil {
		global.GVA_LOG.Error("获取规格失败", zap.Error(err))
		response.FailWithMessage("获取规格失败", c)
		return
	}
	response.OkWithData(list, c)
}

// ---------- 往来单位（供应商 / 客户） ----------

// ListPartners 列表，kind 可过滤
func (a *MasterApi) ListPartners(c *gin.Context) {
	kind := c.DefaultQuery("kind", "")
	db := global.GVA_DB.WithContext(c.Request.Context()).Model(&jxcmodel.Partner{})
	if kind != "" {
		db = db.Where("kind = ?", kind)
	}
	var list []jxcmodel.Partner
	if err := db.Order("id desc").Limit(1000).Find(&list).Error; err != nil {
		global.GVA_LOG.Error("获取往来单位失败", zap.Error(err))
		response.FailWithMessage("获取往来单位失败", c)
		return
	}
	response.OkWithData(list, c)
}

// CreatePartner 新建往来单位
func (a *MasterApi) CreatePartner(c *gin.Context) {
	var req jxcmodel.Partner
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	if req.Kind != jxcmodel.PartnerSupplier && req.Kind != jxcmodel.PartnerCustomer {
		response.FailWithMessage("类型只能是供应商或客户", c)
		return
	}
	if req.Code == "" || req.Name == "" {
		response.FailWithMessage("编码与名称不能为空", c)
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	if err := global.GVA_DB.WithContext(c.Request.Context()).Create(&req).Error; err != nil {
		response.FailWithMessage("创建失败："+err.Error(), c)
		return
	}
	response.OkWithData(req, c)
}

package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GoodsApi struct{}

// ========== 商品 SPU ==========

// GetGoodsPage 分页查询商品列表
// @Tags     JxcGoods
// @Summary  分页查询商品列表
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    page     query     int     true  "页码"
// @Param    pageSize query     int     true  "每页条数"
// @Param    keyword  query     string  false "搜索关键字(编码/名称)"
// @Success  200      {object}  response.PageResult{list=[]jxc.Goods,total=int64}  "商品列表"
// @Router   /jxc/goods/page [get]
func (a *GoodsApi) GetGoodsPage(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)
	list, total, err := goodsService.GetGoodsPage(c.Request.Context(), pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取商品列表失败!", zap.Error(err))
		response.FailWithMessage("获取商品列表失败", c)
		return
	}
	response.OkWithData(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, c)
}

// GetGoodsDetail 查询商品详情
// @Tags     JxcGoods
// @Summary  查询商品详情（含SKU列表）
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    id   query  uint  true  "商品ID"
// @Success  200  {object}  response.Response{data=jxc.Goods}  "商品详情"
// @Router   /jxc/goods/detail [get]
func (a *GoodsApi) GetGoodsDetail(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	goods, err := goodsService.GetGoodsDetail(c.Request.Context(), req.Uint())
	if err != nil {
		global.GVA_LOG.Error("获取商品详情失败!", zap.Error(err))
		response.FailWithMessage("获取商品详情失败", c)
		return
	}
	response.OkWithData(goods, c)
}

// GetAllGoods 获取所有上架商品
// @Tags     JxcGoods
// @Summary  获取所有上架商品（供采购/销售单据选择）
// @Security ApiKeyAuth
// @Produce  application/json
// @Success  200  {object}  response.Response{data=[]jxc.Goods}  "上架商品列表"
// @Router   /jxc/goods/all [get]
func (a *GoodsApi) GetAllGoods(c *gin.Context) {
	list, err := goodsService.GetAllGoods(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("获取商品失败！", zap.Error(err))
		response.FailWithMessage("获取商品失败！", c)
		return
	}
	response.OkWithData(list, c)
}

// CreateGoods 创建商品
// @Tags     JxcGoods
// @Summary  创建商品
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      model/jxc.Goods  true "商品信息"
// @Success  200  {object}  response.Response{msg=string}  "创建商品"
// @Router   /jxc/goods [post]
func (a *GoodsApi) CreateGoods(c *gin.Context) {
	var goods jxc.Goods
	if err := c.ShouldBindJSON(&goods); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.CreateGoods(c.Request.Context(), &goods); err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
		return
	}
	response.OkWithDetailed(&goods, "创建成功", c)
}

// UpdateGoods 更新商品
// @Tags     JxcGoods
// @Summary  更新商品（编码不可修改）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      model/jxc.Goods  true "商品信息（需含ID）"
// @Success  200  {object}  response.Response{msg=string}  "更新商品"
// @Router   /jxc/goods [put]
func (a *GoodsApi) UpdateGoods(c *gin.Context) {
	var goods jxc.Goods
	if err := c.ShouldBindJSON(&goods); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.UpdateGoods(c.Request.Context(), &goods); err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeleteGoods 删除商品
// @Tags     JxcGoods
// @Summary  删除商品（存在SKU时禁止）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "商品ID"
// @Success  200  {object}  response.Response{msg=string}  "删除商品"
// @Router   /jxc/goods [delete]
func (a *GoodsApi) DeleteGoods(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.DeleteGoods(c.Request.Context(), req.Uint()); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteGoodsForever 彻底删除商品
// @Tags     JxcGoods
// @Summary  彻底删除商品（物理删除，不可恢复）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "商品ID"
// @Success  200  {object}  response.Response{msg=string}  "彻底删除成功"
// @Router   /jxc/goods/forever [delete]
func (a *GoodsApi) DeleteGoodsForever(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.DeleteGoodsForever(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("彻底删除成功", c)
}

// SetGoodsStatus 设置商品状态（上架/下架）
// @Tags     JxcGoods
// @Summary  上架/下架商品
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      object{id=uint,status=int8}  true "商品ID与目标状态(1上架/0下架)"
// @Success  200  {object}  response.Response{msg=string}  "操作成功"
// @Router   /jxc/goods/status [put]
func (a *GoodsApi) SetGoodsStatus(c *gin.Context) {
	var req struct {
		ID     uint `json:"id" binding:"required"`
		Status int8 `json:"status" binding:"oneof=0 1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.SetGoodsStatus(c.Request.Context(), req.ID, req.Status); err != nil {
		global.GVA_LOG.Error("操作失败!", zap.Error(err))
		response.FailWithMessage("操作失败", c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

// ========== 商品 SKU ==========

// GetSkuList 查询某商品下的SKU列表
// @Tags     JxcSku
// @Summary  查询某商品下的SKU列表
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    goodsId  query  uint  true  "商品ID"
// @Success  200      {object}  response.Response{data=[]jxc.GoodsSku}  "SKU列表"
// @Router   /jxc/goods/sku/list [get]
func (a *GoodsApi) GetSkuList(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := goodsService.GetSkuList(c.Request.Context(), req.Uint())
	if err != nil {
		global.GVA_LOG.Error("获取商品SKU列表失败!", zap.Error(err))
		response.FailWithMessage("获取商品SKU列表失败", c)
		return
	}
	response.OkWithData(list, c)
}

// CreateSku 创建 SKU
// @Tags     JxcSku
// @Summary  创建SKU（同商品颜色+尺码组合唯一）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      model/jxc.GoodsSku  true "SKU信息"
// @Success  200  {object}  response.Response{msg=string}  "创建SKU"
// @Router   /jxc/goods/sku [post]
func (a *GoodsApi) CreateSku(c *gin.Context) {
	var sku jxc.GoodsSku
	if err := c.ShouldBindJSON(&sku); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.CreateSku(c.Request.Context(), &sku); err != nil {
		global.GVA_LOG.Error("创建 SKU 失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
		return
	}
	response.OkWithDetailed(&sku, "创建成功", c)
}

// GetSkuByBarcode 按条码/SKU编码查 SKU（含各仓库库存）
// @Tags     JxcSku
// @Summary  按条码查SKU（小程序扫码）
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    barcode query string true "条码或SKU编码"
// @Success  200 {object} response.Response{data=jxc.GoodsSku} "SKU及库存"
// @Router   /jxc/goods/sku/by-barcode [get]
func (a *GoodsApi) GetSkuByBarcode(c *gin.Context) {
	keyword := c.Query("barcode")
	sku, err := goodsService.GetSkuByBarcode(c.Request.Context(), keyword)
	if err != nil {
		global.GVA_LOG.Error("按条码查询SKU失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(sku, "查询成功", c)
}

// UpdateSku 更新 SKU
// @Tags     JxcSku
// @Summary  更新SKU（归属与编码不可修改）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      model/jxc.GoodsSku  true "SKU信息（需含ID）"
// @Success  200  {object}  response.Response{msg=string}  "更新SKU"
// @Router   /jxc/goods/sku [put]
func (a *GoodsApi) UpdateSku(c *gin.Context) {
	var sku jxc.GoodsSku
	if err := c.ShouldBindJSON(&sku); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.UpdateSku(c.Request.Context(), &sku); err != nil {
		global.GVA_LOG.Error("更新 SKU 失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeleteSku 删除 SKU
// @Tags     JxcSku
// @Summary  删除SKU
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "SKU ID"
// @Success  200  {object}  response.Response{msg=string}  "删除SKU"
// @Router   /jxc/goods/sku [delete]
func (a *GoodsApi) DeleteSku(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.DeleteSku(c.Request.Context(), req.Uint()); err != nil {
		global.GVA_LOG.Error("删除 SKU 失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteSkuForever 彻底删除 SKU
// @Tags     JxcSku
// @Summary  彻底删除 SKU（物理删除，不可恢复）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "SKU ID"
// @Success  200  {object}  response.Response{msg=string}  "彻底删除成功"
// @Router   /jxc/goods/sku/forever [delete]
func (a *GoodsApi) DeleteSkuForever(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.DeleteSkuForever(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("彻底删除成功", c)
}

// SetSkuStatus 设置 SKU 状态（启用/停用）
// @Tags     JxcSku
// @Summary  启用/停用SKU
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      object{id=uint,status=int8}  true "SKU ID与目标状态(1启用/0停用)"
// @Success  200  {object}  response.Response{msg=string}  "操作成功"
// @Router   /jxc/goods/sku/status [put]
func (a *GoodsApi) SetSkuStatus(c *gin.Context) {
	var req struct {
		ID     uint `json:"id" binding:"required"`
		Status int8 `json:"status" binding:"oneof=0 1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := goodsService.SetSkuStatus(c.Request.Context(), req.ID, req.Status); err != nil {
		global.GVA_LOG.Error("操作失败!", zap.Error(err))
		response.FailWithMessage("操作失败", c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

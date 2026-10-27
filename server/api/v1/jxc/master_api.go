package jxc

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
	jxcsvc "github.com/flipped-aurora/gin-vue-admin/server/service/jxc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MasterApi 基础资料：仓库 / 商品 / 规格 / 往来单位。
//
// handler 只做解析 → 调 Service → 统一响应；DB 访问与业务校验在
// service/jxc/master_service.go。入参用 request 包的 DTO，不直接绑定模型，
// 避免客户端通过多传 ID / status / 时间字段做 mass assignment。
type MasterApi struct{}

// ---------- 仓库 ----------

// ListWarehouses
// @Tags      JxcMaster
// @Summary   仓库列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     active  query     int  false  "1=只返回启用"
// @Success   200     {object}  response.Response{data=[]jxcmodel.Warehouse,msg=string}
// @Router    /jxc/master/warehouses [get]
func (a *MasterApi) ListWarehouses(c *gin.Context) {
	list, err := jxcsvc.ServiceGroupApp.Master.ListWarehouses(c.Request.Context(), c.Query("active") == "1")
	if err != nil {
		global.GVA_LOG.Error("获取仓库失败", zap.Error(err))
		response.FailWithMessage("获取仓库失败", c)
		return
	}
	response.OkWithData(list, c)
}

// CreateWarehouse
// @Tags      JxcMaster
// @Summary   新建仓库
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      jxcReq.WarehouseInput  true  "仓库编码与名称"
// @Success   200   {object}  response.Response{data=jxcmodel.Warehouse,msg=string}
// @Router    /jxc/master/warehouses [post]
func (a *MasterApi) CreateWarehouse(c *gin.Context) {
	var req jxcReq.WarehouseInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	w, err := jxcsvc.ServiceGroupApp.Master.CreateWarehouse(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(w, c)
}

// UpdateWarehouse
// @Tags      JxcMaster
// @Summary   编辑仓库
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id    path      int                    true  "仓库ID"
// @Param     data  body      jxcReq.WarehouseInput  true  "仓库编码与名称"
// @Success   200   {object}  response.Response{data=jxcmodel.Warehouse,msg=string}
// @Router    /jxc/master/warehouses/{id} [put]
func (a *MasterApi) UpdateWarehouse(c *gin.Context) {
	id, ok := parseMasterID(c)
	if !ok {
		return
	}
	var req jxcReq.WarehouseInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	w, err := jxcsvc.ServiceGroupApp.Master.UpdateWarehouse(c.Request.Context(), id, req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(w, c)
}

// SetWarehouseStatus
// @Tags      JxcMaster
// @Summary   启用/停用仓库
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id    path      int                true  "仓库ID"
// @Param     data  body      jxcReq.StatusInput true  "0停用 1启用"
// @Success   200   {object}  response.Response{msg=string}
// @Router    /jxc/master/warehouses/{id}/status [post]
func (a *MasterApi) SetWarehouseStatus(c *gin.Context) {
	id, ok := parseMasterID(c)
	if !ok {
		return
	}
	var req jxcReq.StatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	if err := jxcsvc.ServiceGroupApp.Master.SetWarehouseStatus(c.Request.Context(), id, req.Status); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("状态已更新", c)
}

// ---------- 商品与规格 ----------

// ListGoods
// @Tags      JxcMaster
// @Summary   商品列表（含规格）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     active  query     int  false  "1=只返回启用"
// @Success   200     {object}  response.Response{data=[]jxcResp.GoodsWithSkus,msg=string}
// @Router    /jxc/master/goods [get]
func (a *MasterApi) ListGoods(c *gin.Context) {
	list, err := jxcsvc.ServiceGroupApp.Master.ListGoods(c.Request.Context(), c.Query("active") == "1")
	if err != nil {
		global.GVA_LOG.Error("获取商品失败", zap.Error(err))
		response.FailWithMessage("获取商品失败", c)
		return
	}
	response.OkWithData(list, c)
}

// CreateGoods
// @Tags      JxcMaster
// @Summary   新建商品（规格一次提交）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      jxcReq.GoodsInput  true  "商品与规格"
// @Success   200   {object}  response.Response{data=jxcResp.GoodsWithSkus,msg=string}
// @Router    /jxc/master/goods [post]
func (a *MasterApi) CreateGoods(c *gin.Context) {
	var req jxcReq.GoodsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	goods, err := jxcsvc.ServiceGroupApp.Master.CreateGoods(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(goods, c)
}

// UpdateGoods
// @Tags      JxcMaster
// @Summary   编辑商品与规格
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id    path      int                true  "商品ID"
// @Param     data  body      jxcReq.GoodsInput  true  "商品与规格（已有规格带 ID，新增规格 ID=0）"
// @Success   200   {object}  response.Response{data=jxcResp.GoodsWithSkus,msg=string}
// @Router    /jxc/master/goods/{id} [put]
func (a *MasterApi) UpdateGoods(c *gin.Context) {
	id, ok := parseMasterID(c)
	if !ok {
		return
	}
	var req jxcReq.GoodsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	goods, err := jxcsvc.ServiceGroupApp.Master.UpdateGoods(c.Request.Context(), id, req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(goods, c)
}

// SetGoodsStatus
// @Tags      JxcMaster
// @Summary   启用/停用商品
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id    path      int                true  "商品ID"
// @Param     data  body      jxcReq.StatusInput true  "0停用 1启用"
// @Success   200   {object}  response.Response{msg=string}
// @Router    /jxc/master/goods/{id}/status [post]
func (a *MasterApi) SetGoodsStatus(c *gin.Context) {
	id, ok := parseMasterID(c)
	if !ok {
		return
	}
	var req jxcReq.StatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	if err := jxcsvc.ServiceGroupApp.Master.SetGoodsStatus(c.Request.Context(), id, req.Status); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("状态已更新", c)
}

// ListSkus
// @Tags      JxcMaster
// @Summary   规格列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     goodsId  query     int  false  "商品ID"
// @Param     active   query     int  false  "1=只返回启用"
// @Success   200      {object}  response.Response{data=[]jxcmodel.GoodsSku,msg=string}
// @Router    /jxc/master/skus [get]
func (a *MasterApi) ListSkus(c *gin.Context) {
	goodsID, _ := strconv.ParseUint(c.DefaultQuery("goodsId", "0"), 10, 64)
	list, err := jxcsvc.ServiceGroupApp.Master.ListSkus(c.Request.Context(), uint(goodsID), c.Query("active") == "1")
	if err != nil {
		global.GVA_LOG.Error("获取规格失败", zap.Error(err))
		response.FailWithMessage("获取规格失败", c)
		return
	}
	response.OkWithData(list, c)
}

// SetSkuStatus
// @Tags      JxcMaster
// @Summary   启用/停用规格
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id    path      int                true  "规格ID"
// @Param     data  body      jxcReq.StatusInput true  "0停用 1启用"
// @Success   200   {object}  response.Response{msg=string}
// @Router    /jxc/master/skus/{id}/status [post]
func (a *MasterApi) SetSkuStatus(c *gin.Context) {
	id, ok := parseMasterID(c)
	if !ok {
		return
	}
	var req jxcReq.StatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	if err := jxcsvc.ServiceGroupApp.Master.SetSkuStatus(c.Request.Context(), id, req.Status); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("状态已更新", c)
}

// ---------- 往来单位 ----------

// ListPartners
// @Tags      JxcMaster
// @Summary   往来单位列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     kind    query     string  false  "supplier/customer"
// @Param     active  query     int     false  "1=只返回启用"
// @Success   200     {object}  response.Response{data=[]jxcmodel.Partner,msg=string}
// @Router    /jxc/master/partners [get]
func (a *MasterApi) ListPartners(c *gin.Context) {
	list, err := jxcsvc.ServiceGroupApp.Master.ListPartners(c.Request.Context(), c.Query("kind"), c.Query("active") == "1")
	if err != nil {
		global.GVA_LOG.Error("获取往来单位失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}

// CreatePartner
// @Tags      JxcMaster
// @Summary   新建往来单位
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      jxcReq.PartnerInput  true  "往来单位"
// @Success   200   {object}  response.Response{data=jxcmodel.Partner,msg=string}
// @Router    /jxc/master/partners [post]
func (a *MasterApi) CreatePartner(c *gin.Context) {
	var req jxcReq.PartnerInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	p, err := jxcsvc.ServiceGroupApp.Master.CreatePartner(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(p, c)
}

// UpdatePartner
// @Tags      JxcMaster
// @Summary   编辑往来单位
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id    path      int                  true  "往来单位ID"
// @Param     data  body      jxcReq.PartnerInput  true  "往来单位（类型不可改）"
// @Success   200   {object}  response.Response{data=jxcmodel.Partner,msg=string}
// @Router    /jxc/master/partners/{id} [put]
func (a *MasterApi) UpdatePartner(c *gin.Context) {
	id, ok := parseMasterID(c)
	if !ok {
		return
	}
	var req jxcReq.PartnerInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	p, err := jxcsvc.ServiceGroupApp.Master.UpdatePartner(c.Request.Context(), id, req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(p, c)
}

// SetPartnerStatus
// @Tags      JxcMaster
// @Summary   启用/停用往来单位
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id    path      int                true  "往来单位ID"
// @Param     data  body      jxcReq.StatusInput true  "0停用 1启用"
// @Success   200   {object}  response.Response{msg=string}
// @Router    /jxc/master/partners/{id}/status [post]
func (a *MasterApi) SetPartnerStatus(c *gin.Context) {
	id, ok := parseMasterID(c)
	if !ok {
		return
	}
	var req jxcReq.StatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	if err := jxcsvc.ServiceGroupApp.Master.SetPartnerStatus(c.Request.Context(), id, req.Status); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("状态已更新", c)
}

func parseMasterID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailWithMessage("ID 不合法", c)
		return 0, false
	}
	return uint(id), true
}

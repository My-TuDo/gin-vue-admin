package jxc

import (
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PurchaseApi 采购中心接口
type PurchaseApi struct{}

// operator 当前操作人（用户名）
func operator(c *gin.Context) string {
	return fmt.Sprintf("%d", utils.GetUserID(c))
}

// GetPurchasePage 分页查询采购单列表
// @Tags     JxcPurchase
// @Summary  分页查询采购单列表
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    page     query     int     true  "页码"
// @Param    pageSize query     int     true  "每页条数"
// @Param    keyword  query     string  false "搜索单号"
// @Success  200      {object}  response.PageResult{list=[]jxc.PurchaseOrder,total=int64}  "采购单列表"
// @Router   /jxc/purchase/page [get]
func (a *PurchaseApi) GetPurchasePage(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)
	list, total, err := purchaseService.GetPurchasePage(c.Request.Context(), pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取采购单列表失败!", zap.Error(err))
		response.FailWithMessage("获取采购单列表失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: pageInfo.Page, PageSize: pageInfo.PageSize}, c)
}

// GetPurchaseDetail 查询采购单详情（含明细）
// @Tags     JxcPurchase
// @Summary  查询采购单详情（含明细）
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    id   query  uint  true  "采购单ID"
// @Success  200  {object}  response.Response{data=jxc.PurchaseOrder}  "采购单详情"
// @Router   /jxc/purchase/detail [get]
func (a *PurchaseApi) GetPurchaseDetail(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	order, err := purchaseService.GetPurchaseDetail(c.Request.Context(), req.Uint())
	if err != nil {
		global.GVA_LOG.Error("获取采购单详情失败!", zap.Error(err))
		response.FailWithMessage("获取采购单详情失败", c)
		return
	}
	response.OkWithData(order, c)
}

// CreatePurchase 创建采购单
// @Tags     JxcPurchase
// @Summary  创建采购单（状态=待审核, 含明细）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      model/jxc.PurchaseOrder  true "采购单信息（含 items 明细）"
// @Success  200  {object}  response.Response{msg=string}  "创建采购单"
// @Router   /jxc/purchase [post]
func (a *PurchaseApi) CreatePurchase(c *gin.Context) {
	var order jxc.PurchaseOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	order.Creator = operator(c)
	if err := purchaseService.CreatePurchase(c.Request.Context(), &order); err != nil {
		global.GVA_LOG.Error("创建采购单失败!", zap.Error(err))
		response.FailWithMessage("创建失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(&order, "创建成功", c)
}

// UpdatePurchase 更新采购单（仅待审核）
// @Tags     JxcPurchase
// @Summary  更新采购单（仅待审核, 明细全量重建）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      model/jxc.PurchaseOrder  true "采购单信息（需含ID与items）"
// @Success  200  {object}  response.Response{msg=string}  "更新采购单"
// @Router   /jxc/purchase [put]
func (a *PurchaseApi) UpdatePurchase(c *gin.Context) {
	var order jxc.PurchaseOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := purchaseService.UpdatePurchase(c.Request.Context(), &order); err != nil {
		global.GVA_LOG.Error("更新采购单失败!", zap.Error(err))
		response.FailWithMessage("更新失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeletePurchase 删除采购单（仅待审核）
// @Tags     JxcPurchase
// @Summary  删除采购单（仅待审核）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "采购单ID"
// @Success  200  {object}  response.Response{msg=string}  "删除采购单"
// @Router   /jxc/purchase [delete]
func (a *PurchaseApi) DeletePurchase(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := purchaseService.DeletePurchase(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// AuditPurchase 审核采购单
// @Tags     JxcPurchase
// @Summary  审核采购单（待审核 → 已审核）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "采购单ID"
// @Success  200  {object}  response.Response{msg=string}  "审核成功"
// @Router   /jxc/purchase/audit [put]
func (a *PurchaseApi) AuditPurchase(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := purchaseService.AuditPurchase(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("审核成功", c)
}

// CancelPurchase 取消采购单
// @Tags     JxcPurchase
// @Summary  取消采购单（待审核/已审核 → 已取消）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "采购单ID"
// @Success  200  {object}  response.Response{msg=string}  "已取消"
// @Router   /jxc/purchase/cancel [put]
func (a *PurchaseApi) CancelPurchase(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := purchaseService.CancelPurchase(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已取消", c)
}

// StockIn 采购入库
// @Tags     JxcPurchase
// @Summary  采购入库（已审核 → 已入库, 增加库存并写流水）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "采购单ID"
// @Success  200  {object}  response.Response{msg=string}  "入库成功"
// @Router   /jxc/purchase/stock-in [put]
func (a *PurchaseApi) StockIn(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := purchaseService.StockIn(c.Request.Context(), req.Uint(), operator(c)); err != nil {
		global.GVA_LOG.Error("采购入库失败!", zap.Error(err))
		response.FailWithMessage("入库失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("入库成功", c)
}

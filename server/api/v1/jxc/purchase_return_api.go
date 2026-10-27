package jxc

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	jxcmodel "github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	jxcsvc "github.com/flipped-aurora/gin-vue-admin/server/service/jxc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PurchaseReturnApi 采购退货：部分退货给供应商，按原采购单价出库。
type PurchaseReturnApi struct{}

// CreatePurchaseReturn
// @Tags      JxcPurchaseReturn
// @Summary   建采购退货单（草稿）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      jxcmodel.PurchaseReturn  true  "purchaseId + items[{skuId,qty}]"
// @Success   200   {object}  response.Response{data=jxcmodel.PurchaseReturn,msg=string}
// @Router    /jxc/purchase-return [post]
func (a *PurchaseReturnApi) CreatePurchaseReturn(c *gin.Context) {
	var req jxcmodel.PurchaseReturn
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	ret, err := jxcsvc.ServiceGroupApp.PurchaseReturn.Create(c.Request.Context(), &req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(ret, c)
}

// GetPurchaseReturnPage
// @Tags      JxcPurchaseReturn
// @Summary   采购退货单列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     page      query     int  false  "页码"
// @Param     pageSize  query     int  false  "每页条数"
// @Success   200       {object}  response.Response{data=response.PageResult{list=[]jxcmodel.PurchaseReturn},msg=string}
// @Router    /jxc/purchase-return/page [get]
func (a *PurchaseReturnApi) GetPurchaseReturnPage(c *gin.Context) {
	info, ok := bindDocumentSearch(c)
	if !ok {
		return
	}
	list, total, err := jxcsvc.ServiceGroupApp.PurchaseReturn.Search(c.Request.Context(), info)
	if err != nil {
		global.GVA_LOG.Error("获取采购退货单列表失败", zap.Error(err))
		response.FailWithMessage("获取采购退货单列表失败："+err.Error(), c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: info.Page, PageSize: info.PageSize}, c)
}

// GetPurchaseReturn
// @Tags      JxcPurchaseReturn
// @Summary   采购退货单详情
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id   path      int  true  "退货单ID"
// @Success   200  {object}  response.Response{data=jxcmodel.PurchaseReturn,msg=string}
// @Router    /jxc/purchase-return/{id} [get]
func (a *PurchaseReturnApi) GetPurchaseReturn(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ret, err := jxcsvc.ServiceGroupApp.PurchaseReturn.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(ret, c)
}

// GetPurchaseReturnable
// @Tags      JxcPurchaseReturn
// @Summary   原采购单每个 SKU 还能退多少
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     purchaseId  query     int  true  "原采购单ID"
// @Success   200         {object}  response.Response{data=[]jxcsvc.PurchaseReturnableLine,msg=string}
// @Router    /jxc/purchase-return/returnable [get]
func (a *PurchaseReturnApi) GetPurchaseReturnable(c *gin.Context) {
	purchaseID, _ := strconv.ParseUint(c.DefaultQuery("purchaseId", "0"), 10, 64)
	if purchaseID == 0 {
		response.FailWithMessage("请指定原采购单", c)
		return
	}
	list, err := jxcsvc.ServiceGroupApp.PurchaseReturn.Returnable(c.Request.Context(), uint(purchaseID))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}

// PostPurchaseReturn
// @Tags      JxcPurchaseReturn
// @Summary   采购退货出库（按原采购单价扣减库存）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id   path      int  true  "退货单ID"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /jxc/purchase-return/{id}/post [post]
func (a *PurchaseReturnApi) PostPurchaseReturn(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := jxcsvc.ServiceGroupApp.PurchaseReturn.Post(c.Request.Context(), id, operator(c), parseOccurredAt(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("退货出库成功", c)
}

// CancelPurchaseReturn
// @Tags      JxcPurchaseReturn
// @Summary   取消采购退货单（仅草稿）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id   path      int  true  "退货单ID"
// @Success   200  {object}  response.Response{msg=string}
// @Router    /jxc/purchase-return/{id}/cancel [post]
func (a *PurchaseReturnApi) CancelPurchaseReturn(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := jxcsvc.ServiceGroupApp.PurchaseReturn.Cancel(c.Request.Context(), id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已取消", c)
}

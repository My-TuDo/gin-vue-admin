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

// ReversalApi 红冲：整单撤销已过账单据。
// 设计上不接受修改历史，只追加反向事实；原因必填，作为日后解释账目的线索。
type ReversalApi struct{}

// CreateReversal 红冲一张采购单/销售单
func (a *ReversalApi) CreateReversal(c *gin.Context) {
	var req struct {
		SourceType string `json:"sourceType"` // purchase / sale
		SourceID   uint   `json:"sourceId"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	rev, err := jxcsvc.ServiceGroupApp.Reversal.Reverse(c.Request.Context(),
		req.SourceType, req.SourceID, req.Reason, operator(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(rev, c)
}

// GetReversalPage 红冲记录（支持 status/keyword/sourceType/warehouseId/partnerId/日期）
func (a *ReversalApi) GetReversalPage(c *gin.Context) {
	info, ok := bindDocumentSearch(c)
	if !ok {
		return
	}
	list, total, err := jxcsvc.ServiceGroupApp.Reversal.Search(c.Request.Context(), info)
	if err != nil {
		global.GVA_LOG.Error("获取红冲记录失败", zap.Error(err))
		response.FailWithMessage("获取红冲记录失败："+err.Error(), c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: info.Page, PageSize: info.PageSize}, c)
}

// ReturnApi 销售退货：部分退货，按原单成本回补库存。
type ReturnApi struct{}

// CreateReturn 建退货单（草稿）
func (a *ReturnApi) CreateReturn(c *gin.Context) {
	var req jxcmodel.SaleReturn
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	ret, err := jxcsvc.ServiceGroupApp.Return.Create(c.Request.Context(), &req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(ret, c)
}

// PostReturn 退货入库
func (a *ReturnApi) PostReturn(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := jxcsvc.ServiceGroupApp.Return.Post(c.Request.Context(), id, operator(c), parseOccurredAt(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("退货入库成功", c)
}

// CancelReturn 取消退货单
func (a *ReturnApi) CancelReturn(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := jxcsvc.ServiceGroupApp.Return.Cancel(c.Request.Context(), id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已取消", c)
}

// GetReturn 退货单详情
func (a *ReturnApi) GetReturn(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ret, err := jxcsvc.ServiceGroupApp.Return.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(ret, c)
}

// GetReturnPage 退货单列表（支持 status/keyword/warehouseId/partnerId/日期）
func (a *ReturnApi) GetReturnPage(c *gin.Context) {
	info, ok := bindDocumentSearch(c)
	if !ok {
		return
	}
	list, total, err := jxcsvc.ServiceGroupApp.Return.Search(c.Request.Context(), info)
	if err != nil {
		global.GVA_LOG.Error("获取退货单列表失败", zap.Error(err))
		response.FailWithMessage("获取退货单列表失败："+err.Error(), c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: info.Page, PageSize: info.PageSize}, c)
}

// GetReturnable 某张销售单每个 SKU 还能退多少
func (a *ReturnApi) GetReturnable(c *gin.Context) {
	saleID, _ := strconv.ParseUint(c.DefaultQuery("saleId", "0"), 10, 64)
	if saleID == 0 {
		response.FailWithMessage("请指定原销售单", c)
		return
	}
	lines, err := jxcsvc.ServiceGroupApp.Return.Returnable(c.Request.Context(), uint(saleID))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(lines, c)
}

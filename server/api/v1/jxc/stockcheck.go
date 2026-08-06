package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StockCheckApi 盘点中心接口
type StockCheckApi struct{}

// CreateStockCheck 创建盘点单
// @Tags     JxcStockCheck
// @Summary  创建盘点单（快照仓库库存）
// @Security ApiKeyAuth
// @Accept   application/json
// @Produce  application/json
// @Param    data body jxc.StockCheck true "盘点单（仓库ID）"
// @Success  200 {object} response.Response{} "创建成功"
// @Router   /jxc/stockcheck [post]
func (a *StockCheckApi) CreateStockCheck(c *gin.Context) {
	var check jxc.StockCheck
	if err := c.ShouldBindJSON(&check); err != nil {
		global.GVA_LOG.Error("参数错误!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := stockCheckService.CreateStockCheck(c.Request.Context(), &check); err != nil {
		global.GVA_LOG.Error("创建盘点单失败!", zap.Error(err))
		response.FailWithMessage("创建失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(&check, "创建成功", c)
}

// UpdateStockCheckItems 录入盘点数
// @Tags     JxcStockCheck
// @Summary  录入盘点数（仅盘点中）
// @Security ApiKeyAuth
// @Accept   application/json
// @Produce  application/json
// @Param    data body jxc.UpdateStockCheckItemsReq true "明细列表"
// @Success  200 {object} response.Response{} "录入成功"
// @Router   /jxc/stockcheck/items [put]
func (a *StockCheckApi) UpdateStockCheckItems(c *gin.Context) {
	var req jxc.UpdateStockCheckItemsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := stockCheckService.UpdateStockCheckItems(c.Request.Context(), req.CheckID, req.Items); err != nil {
		global.GVA_LOG.Error("录入盘点数失败!", zap.Error(err))
		response.FailWithMessage("录入失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("录入成功", c)
}

// CompleteStockCheck 完成盘点
// @Tags     JxcStockCheck
// @Summary  完成盘点（差异调整库存并生成流水）
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    id query uint true "盘点单ID"
// @Success  200 {object} response.Response{} "完成成功"
// @Router   /jxc/stockcheck/complete [put]
func (a *StockCheckApi) CompleteStockCheck(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := stockCheckService.CompleteStockCheck(c.Request.Context(), req.Uint(), saleOperator(c)); err != nil {
		global.GVA_LOG.Error("完成盘点失败!", zap.Error(err))
		response.FailWithMessage("完成失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("完成成功", c)
}

// CancelStockCheck 取消盘点单
// @Tags     JxcStockCheck
// @Summary  取消盘点单（仅盘点中）
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    id query uint true "盘点单ID"
// @Success  200 {object} response.Response{} "取消成功"
// @Router   /jxc/stockcheck/cancel [put]
func (a *StockCheckApi) CancelStockCheck(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := stockCheckService.CancelStockCheck(c.Request.Context(), req.Uint()); err != nil {
		global.GVA_LOG.Error("取消盘点单失败!", zap.Error(err))
		response.FailWithMessage("取消失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("取消成功", c)
}

// GetStockCheckPage 分页查询盘点单
// @Tags     JxcStockCheck
// @Summary  分页查询盘点单
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    data query request.PageInfo true "分页参数"
// @Success  200 {object} response.PageResult{list=[]jxc.StockCheck,total=int64}  "盘点单列表"
// @Router   /jxc/stockcheck/page [get]
func (a *StockCheckApi) GetStockCheckPage(c *gin.Context) {
	var info request.PageInfo
	if err := c.ShouldBindQuery(&info); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := stockCheckService.GetStockCheckPage(c.Request.Context(), info)
	if err != nil {
		global.GVA_LOG.Error("获取盘点单列表失败!", zap.Error(err))
		response.FailWithMessage("获取列表失败："+err.Error(), c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: info.Page, PageSize: info.PageSize}, c)
}

// GetStockCheckDetail 查询盘点单详情
// @Tags     JxcStockCheck
// @Summary  查询盘点单详情（含明细）
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    id query uint true "盘点单ID"
// @Success  200 {object} response.Response{data=jxc.StockCheck} "盘点单详情"
// @Router   /jxc/stockcheck/detail [get]
func (a *StockCheckApi) GetStockCheckDetail(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	check, err := stockCheckService.GetStockCheckDetail(c.Request.Context(), req.Uint())
	if err != nil {
		global.GVA_LOG.Error("获取盘点单详情失败!", zap.Error(err))
		response.FailWithMessage("获取详情失败："+err.Error(), c)
		return
	}
	response.OkWithData(check, c)
}

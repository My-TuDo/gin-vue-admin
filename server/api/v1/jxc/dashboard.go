package jxc

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DashboardApi 仪表盘接口
type DashboardApi struct{}

// Overview 概览（今日/本周/本月）
// @Tags     JxcDashboard
// @Summary  概览（销售额/订单数/毛利/退货额）
// @Security ApiKeyAuth
// @Produce  application/json
// @Success  200 {object} response.Response{data=[]jxc.DashboardService.OverviewItem} "概览"
// @Router   /jxc/dashboard/overview [get]
func (a *DashboardApi) Overview(c *gin.Context) {
	data, err := dashboardService.Overview(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("概览聚合失败!", zap.Error(err))
		response.FailWithMessage("获取概览失败", c)
		return
	}
	response.OkWithData(data, c)
}

// Trend 销售趋势（近 N 天）
// @Tags     JxcDashboard
// @Summary  销售趋势
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    days query int false "天数（默认7）"
// @Success  200 {object} response.Response{data=[]jxc.DashboardService.TrendPoint} "趋势"
// @Router   /jxc/dashboard/trend [get]
func (a *DashboardApi) Trend(c *gin.Context) {
	days, _ := strconv.Atoi(c.Query("days"))
	data, err := dashboardService.Trend(c.Request.Context(), days)
	if err != nil {
		global.GVA_LOG.Error("趋势聚合失败!", zap.Error(err))
		response.FailWithMessage("获取趋势失败", c)
		return
	}
	response.OkWithData(data, c)
}

// Top 热销榜
// @Tags     JxcDashboard
// @Summary  热销商品 TOP N
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    days query int false "天数（默认30）"
// @Param    limit query int false "条数（默认10）"
// @Success  200 {object} response.Response{data=[]jxc.DashboardService.TopItem} "热销"
// @Router   /jxc/dashboard/top [get]
func (a *DashboardApi) Top(c *gin.Context) {
	days, _ := strconv.Atoi(c.Query("days"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	data, err := dashboardService.Top(c.Request.Context(), days, limit)
	if err != nil {
		global.GVA_LOG.Error("热销聚合失败!", zap.Error(err))
		response.FailWithMessage("获取热销失败", c)
		return
	}
	response.OkWithData(data, c)
}

// StockAlert 库存预警
// @Tags     JxcDashboard
// @Summary  库存预警列表
// @Security ApiKeyAuth
// @Produce  application/json
// @Success  200 {object} response.Response{data=[]jxc.DashboardService.AlertItem} "预警"
// @Router   /jxc/dashboard/stock-alert [get]
func (a *DashboardApi) StockAlert(c *gin.Context) {
	data, err := dashboardService.StockAlert(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("预警聚合失败!", zap.Error(err))
		response.FailWithMessage("获取预警失败", c)
		return
	}
	response.OkWithData(data, c)
}

// Category 分类销售占比
// @Tags     JxcDashboard
// @Summary  分类销售占比
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    days query int false "天数（默认30）"
// @Success  200 {object} response.Response{data=[]jxc.DashboardService.CatItem} "分类"
// @Router   /jxc/dashboard/category [get]
func (a *DashboardApi) Category(c *gin.Context) {
	days, _ := strconv.Atoi(c.Query("days"))
	data, err := dashboardService.Category(c.Request.Context(), days)
	if err != nil {
		global.GVA_LOG.Error("分类聚合失败!", zap.Error(err))
		response.FailWithMessage("获取分类占比失败", c)
		return
	}
	response.OkWithData(data, c)
}

// SalesHistory 历史销售统计（日期范围，按日/按月）
// @Tags     JxcDashboard
// @Summary  历史销售统计
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    from query string true "起始日期 yyyy-mm-dd"
// @Param    to query string true "结束日期 yyyy-mm-dd"
// @Param    granularity query string false "day/month（默认day）"
// @Success  200 {object} response.Response{data=[]jxc.DashboardService.SalesHistoryItem} "统计"
// @Router   /jxc/dashboard/sales-history [get]
func (a *DashboardApi) SalesHistory(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	data, err := dashboardService.SalesHistory(c.Request.Context(), from, to, c.Query("granularity"))
	if err != nil {
		global.GVA_LOG.Error("历史统计失败!", zap.Error(err))
		response.FailWithMessage("获取历史统计失败："+err.Error(), c)
		return
	}
	response.OkWithData(data, c)
}

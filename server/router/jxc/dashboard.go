package jxc

import "github.com/gin-gonic/gin"

// DashboardRouter 仪表盘路由
type DashboardRouter struct{}

func (r *DashboardRouter) InitDashboardRouter(Router *gin.RouterGroup) {
	readOnlyGroup := Router.Group("jxc")
	{
		readOnlyGroup.GET("dashboard/overview", dashboardApi.Overview)
		readOnlyGroup.GET("dashboard/trend", dashboardApi.Trend)
		readOnlyGroup.GET("dashboard/top", dashboardApi.Top)
		readOnlyGroup.GET("dashboard/stock-alert", dashboardApi.StockAlert)
		readOnlyGroup.GET("dashboard/category", dashboardApi.Category)
		readOnlyGroup.GET("dashboard/sales-history", dashboardApi.SalesHistory)
	}
}

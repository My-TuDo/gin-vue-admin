package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type DashboardRouter struct{}

// InitDashboardRouter 工作台聚合路由（PrivateGroup）
func (r *DashboardRouter) InitDashboardRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.DashboardApi
	g := router.Group("jxc/dashboard")
	{
		g.GET("summary", api.GetDashboardSummary)
	}
}

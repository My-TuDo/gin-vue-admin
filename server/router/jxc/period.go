package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type PeriodRouter struct{}

// InitPeriodRouter 月结与报表路由（PrivateGroup）
func (r *PeriodRouter) InitPeriodRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.PeriodApi
	g := router.Group("jxc/period")
	{
		g.GET("", api.GetPeriods)
		g.POST("close", api.ClosePeriod)
		g.GET("snapshot", api.GetSnapshot)
		g.GET("summary", api.GetMonthlySummary)
	}
}

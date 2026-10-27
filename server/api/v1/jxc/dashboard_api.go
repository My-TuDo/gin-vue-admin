package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	jxcsvc "github.com/flipped-aurora/gin-vue-admin/server/service/jxc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DashboardApi 工作台聚合接口。
type DashboardApi struct{}

// GetDashboardSummary
// @Tags      JxcDashboard
// @Summary   工作台今日经营概览（KPI/待办/低库存）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     date  query     string  false  "YYYY-MM-DD，默认今天；Asia/Shanghai"
// @Success   200   {object}  response.Response{data=jxcsvc.DashboardSummary,msg=string}
// @Router    /jxc/dashboard/summary [get]
func (a *DashboardApi) GetDashboardSummary(c *gin.Context) {
	summary, err := jxcsvc.ServiceGroupApp.Dashboard.Summary(c.Request.Context(), c.Query("date"))
	if err != nil {
		global.GVA_LOG.Error("获取工作台概览失败", zap.Error(err))
		response.FailWithMessage("获取工作台概览失败："+err.Error(), c)
		return
	}
	response.OkWithData(summary, c)
}

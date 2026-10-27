package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type PartnerRouter struct{}

// InitPartnerRouter 往来单位路由（挂在 jxc/master 下）
func (r *PartnerRouter) InitPartnerRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.MasterApi
	g := router.Group("jxc/master")
	{
		g.GET("partners", api.ListPartners)
		g.POST("partners", api.CreatePartner)
		g.PUT("partners/:id", api.UpdatePartner)
		g.POST("partners/:id/status", api.SetPartnerStatus)
	}
}

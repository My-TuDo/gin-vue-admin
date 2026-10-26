package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type MasterRouter struct{}

// InitMasterRouter 基础资料路由（同样在 PrivateGroup 下）
func (r *MasterRouter) InitMasterRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.MasterApi

	g := router.Group("jxc/master")
	{
		g.GET("warehouses", api.ListWarehouses)
		g.POST("warehouses", api.CreateWarehouse)
		g.GET("goods", api.ListGoods)
		g.POST("goods", api.CreateGoods)
		g.GET("skus", api.ListSkus)
	}
}

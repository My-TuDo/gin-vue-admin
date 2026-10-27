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
		g.PUT("warehouses/:id", api.UpdateWarehouse)
		g.POST("warehouses/:id/status", api.SetWarehouseStatus)

		g.GET("goods", api.ListGoods)
		g.POST("goods", api.CreateGoods)
		g.PUT("goods/:id", api.UpdateGoods)
		g.POST("goods/:id/status", api.SetGoodsStatus)

		g.GET("skus", api.ListSkus)
		g.POST("skus/:id/status", api.SetSkuStatus)
	}
}

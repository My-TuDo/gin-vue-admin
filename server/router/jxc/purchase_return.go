package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type PurchaseReturnRouter struct{}

// InitPurchaseReturnRouter 采购退货路由（PrivateGroup）
func (r *PurchaseReturnRouter) InitPurchaseReturnRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.PurchaseReturnApi
	g := router.Group("jxc/purchase-return")
	{
		g.POST("", api.CreatePurchaseReturn)
		g.GET("page", api.GetPurchaseReturnPage)
		g.GET("returnable", api.GetPurchaseReturnable)
		g.GET(":id", api.GetPurchaseReturn)
		g.POST(":id/post", api.PostPurchaseReturn)
		g.POST(":id/cancel", api.CancelPurchaseReturn)
	}
}

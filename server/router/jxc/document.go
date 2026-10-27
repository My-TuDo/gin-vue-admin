package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type DocumentRouter struct{}

// InitDocumentRouter 注册采购/销售路由。
//
// 注意：这些路由必须在 PrivateGroup 下注册，否则等于裸奔（详见 docs 里鉴权那一节）。
func (r *DocumentRouter) InitDocumentRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.DocumentApi

	purchase := router.Group("jxc/purchase")
	{
		purchase.POST("", api.CreatePurchase)
		purchase.GET("page", api.GetPurchasePage)
		purchase.GET(":id", api.GetPurchase)
		purchase.POST(":id/approve", api.ApprovePurchase)
		purchase.POST(":id/receive", api.ReceivePurchase)
		purchase.POST(":id/cancel", api.CancelPurchase)
	}

	sale := router.Group("jxc/sale")
	{
		sale.POST("", api.CreateSale)
		sale.POST("checkout", api.CheckoutSale)
		sale.GET("page", api.GetSalePage)
		sale.GET(":id", api.GetSale)
		sale.POST(":id/ship", api.ShipSale)
		sale.POST(":id/cancel", api.CancelSale)
	}
}

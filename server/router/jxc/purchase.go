package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type PurchaseRouter struct{}

func (r *PurchaseRouter) InitPurchaseRouter(Router *gin.RouterGroup) {
	recordGroup := Router.Group("jxc").Use(middleware.OperationRecord())
	readOnlyGroup := Router.Group("jxc")

	// ===== 采购单 =====
	readOnlyGroup.GET("purchase/page", purchaseApi.GetPurchasePage)
	readOnlyGroup.GET("purchase/detail", purchaseApi.GetPurchaseDetail)
	recordGroup.POST("purchase", purchaseApi.CreatePurchase)
	recordGroup.PUT("purchase", purchaseApi.UpdatePurchase)
	recordGroup.DELETE("purchase", purchaseApi.DeletePurchase)
	recordGroup.PUT("purchase/audit", purchaseApi.AuditPurchase)
	recordGroup.PUT("purchase/cancel", purchaseApi.CancelPurchase)
	recordGroup.PUT("purchase/stock-in", purchaseApi.StockIn)
}

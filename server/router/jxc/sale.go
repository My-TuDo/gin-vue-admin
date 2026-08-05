package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type SaleRouter struct{}

func (r *SaleRouter) InitSaleRouter(Router *gin.RouterGroup) {
	recordGroup := Router.Group("jxc").Use(middleware.OperationRecord())
	readOnlyGroup := Router.Group("jxc")

	// ===== 销售单 =====
	readOnlyGroup.GET("sale/page", saleApi.GetSalePage)
	readOnlyGroup.GET("sale/detail", saleApi.GetSaleDetail)
	readOnlyGroup.GET("sale/remaining", saleApi.GetSaleRemaining)
	recordGroup.POST("sale", saleApi.CreateSaleOrder)
	recordGroup.PUT("sale", saleApi.UpdateSaleOrder)
	recordGroup.DELETE("sale", saleApi.DeleteSaleOrder)
	recordGroup.PUT("sale/cancel", saleApi.CancelSaleOrder)
	recordGroup.PUT("sale/out", saleApi.ConfirmOut)
	recordGroup.PUT("sale/return", saleApi.ConfirmReturn)
	recordGroup.PUT("sale/exchange", saleApi.ConfirmExchange)
}

package jxc

import "github.com/gin-gonic/gin"

// CashierRouter 收银台路由
type CashierRouter struct{}

func (r *CashierRouter) InitCashierRouter(Router *gin.RouterGroup) {
	recordGroup := Router.Group("jxc")
	{
		recordGroup.POST("cashier/checkout", cashierApi.Checkout)
	recordGroup.POST("cashier/refund", cashierApi.Refund)
	recordGroup.POST("cashier/exchange", cashierApi.Exchange)
	readOnlyGroup := Router.Group("jxc")
	{
		readOnlyGroup.GET("cashier/refundable-orders", cashierApi.RefundableOrders)
	}
	}
}

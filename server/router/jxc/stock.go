package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type StockRouter struct{}

func (r *StockRouter) InitStockRouter(Router *gin.RouterGroup) {
	recordGroup := Router.Group("jxc").Use(middleware.OperationRecord())
	readOnlyGroup := Router.Group("jxc")

	// ===== 库存中心 =====
	readOnlyGroup.GET("stock/page", stockApi.GetStockPage)
	readOnlyGroup.GET("stock/log/page", stockApi.GetStockLogPage)
	recordGroup.POST("stock/direct-in", stockApi.DirectIn)
	recordGroup.POST("stock/direct-out", stockApi.DirectOut)
}

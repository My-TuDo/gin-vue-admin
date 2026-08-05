package jxc

import (
	"github.com/gin-gonic/gin"
)

// StockCheckRouter 盘点中心路由
type StockCheckRouter struct{}

func (r *StockCheckRouter) InitStockCheckRouter(Router *gin.RouterGroup) {
	readOnlyGroup := Router.Group("jxc")
	recordGroup := Router.Group("jxc")
	{
		readOnlyGroup.GET("stockcheck/page", stockCheckApi.GetStockCheckPage)
		readOnlyGroup.GET("stockcheck/detail", stockCheckApi.GetStockCheckDetail)
	}
	{
		recordGroup.POST("stockcheck", stockCheckApi.CreateStockCheck)
		recordGroup.PUT("stockcheck/items", stockCheckApi.UpdateStockCheckItems)
		recordGroup.PUT("stockcheck/complete", stockCheckApi.CompleteStockCheck)
		recordGroup.PUT("stockcheck/cancel", stockCheckApi.CancelStockCheck)
	}
}

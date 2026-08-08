package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type GoodsRouter struct{}

func (r *GoodsRouter) InitGoodsRouter(Router *gin.RouterGroup) {
	recordGroup := Router.Group("jxc").Use(middleware.OperationRecord())
	readOnlyGroup := Router.Group("jxc")

	// ===== 商品 SPU =====
	readOnlyGroup.GET("goods/page", goodsApi.GetGoodsPage)
	readOnlyGroup.GET("goods/detail", goodsApi.GetGoodsDetail)
	readOnlyGroup.GET("goods/all", goodsApi.GetAllGoods)
	recordGroup.POST("goods", goodsApi.CreateGoods)
	recordGroup.PUT("goods", goodsApi.UpdateGoods)
	recordGroup.DELETE("goods", goodsApi.DeleteGoods)
	recordGroup.DELETE("goods/forever", goodsApi.DeleteGoodsForever)
	recordGroup.PUT("goods/status", goodsApi.SetGoodsStatus)

	// ===== 商品 SKU =====
	readOnlyGroup.GET("goods/sku/list", goodsApi.GetSkuList)
	readOnlyGroup.GET("goods/sku/by-barcode", goodsApi.GetSkuByBarcode)
	recordGroup.POST("goods/sku", goodsApi.CreateSku)
	recordGroup.PUT("goods/sku", goodsApi.UpdateSku)
	recordGroup.DELETE("goods/sku", goodsApi.DeleteSku)
	recordGroup.DELETE("goods/sku/forever", goodsApi.DeleteSkuForever)
	recordGroup.PUT("goods/sku/status", goodsApi.SetSkuStatus)
}

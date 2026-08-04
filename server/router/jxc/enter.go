package jxc

import (
	api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
)

type RouterGroup struct {
	BasicRouter
	GoodsRouter
	PurchaseRouter
	SaleRouter
}

var (
	basicApi    = api.ApiGroupApp.JxcApiGroup.BasicApi
	goodsApi    = api.ApiGroupApp.JxcApiGroup.GoodsApi
	purchaseApi = api.ApiGroupApp.JxcApiGroup.PurchaseApi
	saleApi     = api.ApiGroupApp.JxcApiGroup.SaleApi
)

package jxc

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	BasicApi
	GoodsApi
	PurchaseApi
}

var (
	basicService    = service.ServiceGroupApp.JxcServiceGroup.BasicService
	goodsService    = service.ServiceGroupApp.JxcServiceGroup.GoodsService
	purchaseService = service.ServiceGroupApp.JxcServiceGroup.PurchaseService
)

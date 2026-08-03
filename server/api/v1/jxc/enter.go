package jxc

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	BasicApi
	GoodsApi
}

var (
	basicService = service.ServiceGroupApp.JxcServiceGroup.BasicService
	goodsService = service.ServiceGroupApp.JxcServiceGroup.GoodsService
)

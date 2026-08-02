package jxc

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	BasicApi
}

var (
	basicService = service.ServiceGroupApp.JxcServiceGroup.BasicService
)

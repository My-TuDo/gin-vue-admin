package jxc

import (
	api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
)

type RouterGroup struct {
	BasicRouter
}

var basicApi = api.ApiGroupApp.JxcApiGroup.BasicApi

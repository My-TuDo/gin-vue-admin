package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type ReturnRouter struct{}

// InitReturnRouter 退货路由（PrivateGroup）
func (r *ReturnRouter) InitReturnRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.ReturnApi
	g := router.Group("jxc/return")
	{
		g.GET("returnable", api.GetReturnable)
		g.POST("", api.CreateReturn)
		g.GET("page", api.GetReturnPage)
		g.GET(":id", api.GetReturn)
		g.POST(":id/post", api.PostReturn)
		g.POST(":id/cancel", api.CancelReturn)
	}
}

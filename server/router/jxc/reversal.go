package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type ReversalRouter struct{}

// InitReversalRouter 红冲路由（PrivateGroup）
func (r *ReversalRouter) InitReversalRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.ReversalApi
	g := router.Group("jxc/reversal")
	{
		g.POST("", api.CreateReversal)
		g.GET("page", api.GetReversalPage)
	}
}

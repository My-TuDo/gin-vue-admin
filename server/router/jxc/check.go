package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type CheckRouter struct{}

// InitCheckRouter 盘点路由（PrivateGroup）
func (r *CheckRouter) InitCheckRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.CheckApi
	g := router.Group("jxc/check")
	{
		g.POST("", api.CreateCheck)
		g.GET("page", api.GetCheckPage)
		g.GET(":id", api.GetCheck)
		g.POST(":id/items", api.UpdateCheckItems)
		g.POST(":id/post", api.PostCheck)
		g.POST(":id/cancel", api.CancelCheck)
	}
}

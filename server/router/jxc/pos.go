package jxc

import "github.com/gin-gonic/gin"

// PosRouter 扫码枪队列路由
type PosRouter struct{}

func (r *PosRouter) InitPosRouter(Router *gin.RouterGroup) {
	writeGroup := Router.Group("jxc")
	{
		writeGroup.POST("pos/scan", posApi.Scan)
		writeGroup.PUT("pos/scan/confirm", posApi.Confirm)
	}
	readGroup := Router.Group("jxc")
	{
		readGroup.GET("pos/scan/pending", posApi.ListPending)
	}
}

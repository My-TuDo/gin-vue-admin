package jxc

import "github.com/gin-gonic/gin"

// PosRouter 扫码枪队列路由
type PosRouter struct{}

func (r *PosRouter) InitPosRouter(Router *gin.RouterGroup) {
	writeGroup := Router.Group("jxc")
	{
		writeGroup.POST("pos/scan", posApi.Scan)                   // 小程序扫码上架（带 session）
		writeGroup.PUT("pos/scan/confirm", posApi.Confirm)         // PC 消费确认
		writeGroup.POST("pos/session", posApi.CreateSession)       // PC 生成收银台码
		writeGroup.PUT("pos/session/disable", posApi.DisableSession) // PC 作废收银台码（重置）
	}
	readGroup := Router.Group("jxc")
	{
		readGroup.GET("pos/scan/pending", posApi.ListPending) // PC 轮询本会话未消费
		readGroup.GET("pos/session/check", posApi.CheckSession) // 小程序绑定校验
	}
}

package jxc

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/gin-gonic/gin"
)

type LedgerRouter struct{}

// InitLedgerRouter 注册账本路由：结存 / 流水 / 对账
func (r *LedgerRouter) InitLedgerRouter(router *gin.RouterGroup) {
	api := v1.ApiGroupApp.JxcApiGroup.LedgerApi

	g := router.Group("jxc/ledger")
	{
		g.GET("balance", api.GetBalances)
		g.GET("moves", api.GetMoves)
		g.GET("reconcile", api.Reconcile)
	}
}

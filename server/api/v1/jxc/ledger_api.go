package jxc

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/ledger"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LedgerApi 账本接口：结存、流水、对账。
//
// 「对账」是本系统面向经营者的核心能力：按下按钮就知道账实是否相符，
// 而不是等月底发现对不上、再靠人肉翻 Excel。
type LedgerApi struct{}

// GetBalances 结存列表（含预留/可售/均价）
func (a *LedgerApi) GetBalances(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.DefaultQuery("warehouseId", "0"), 10, 64)
	list, err := service.ServiceGroupApp.JxcServiceGroup.Ledger.Balances(c.Request.Context(), uint(warehouseID))
	if err != nil {
		global.GVA_LOG.Error("获取结存失败", zap.Error(err))
		response.FailWithMessage("获取结存失败", c)
		return
	}
	response.OkWithData(list, c)
}

// GetMoves 账本流水（分页）
func (a *LedgerApi) GetMoves(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.DefaultQuery("warehouseId", "0"), 10, 64)
	skuID, _ := strconv.ParseUint(c.DefaultQuery("skuId", "0"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	list, total, err := service.ServiceGroupApp.JxcServiceGroup.Ledger.Moves(
		c.Request.Context(), uint(warehouseID), uint(skuID), page, size)
	if err != nil {
		global.GVA_LOG.Error("获取账本流水失败", zap.Error(err))
		response.FailWithMessage("获取账本流水失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: page, PageSize: size}, c)
}

// Reconcile 一键对账：返回所有不变量违规（空表示账是平的）
func (a *LedgerApi) Reconcile(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.DefaultQuery("warehouseId", "0"), 10, 64)
	vs, err := service.ServiceGroupApp.JxcServiceGroup.Ledger.Reconcile(c.Request.Context(), uint(warehouseID))
	if err != nil {
		global.GVA_LOG.Error("对账失败", zap.Error(err))
		response.FailWithMessage("对账失败", c)
		return
	}
	if vs == nil {
		vs = []ledger.Violation{}
	}
	response.OkWithData(gin.H{
		"ok":         len(vs) == 0,
		"violations": vs,
		"count":      len(vs),
	}, c)
}

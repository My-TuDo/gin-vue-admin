package jxc

import (
	"fmt"
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	jxcsvc "github.com/flipped-aurora/gin-vue-admin/server/service/jxc"
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
	list, err := jxcsvc.ServiceGroupApp.Ledger.Balances(c.Request.Context(), uint(warehouseID))
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
	list, total, err := jxcsvc.ServiceGroupApp.Ledger.Moves(
		c.Request.Context(), uint(warehouseID), uint(skuID), page, size)
	if err != nil {
		global.GVA_LOG.Error("获取账本流水失败", zap.Error(err))
		response.FailWithMessage("获取账本流水失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: page, PageSize: size}, c)
}

// Reconcile 一键对账：执行一次校验并缓存结果。
func (a *LedgerApi) Reconcile(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.DefaultQuery("warehouseId", "0"), 10, 64)
	res, err := jxcsvc.ServiceGroupApp.Ledger.Check(c.Request.Context(), uint(warehouseID))
	if err != nil {
		global.GVA_LOG.Error("对账失败", zap.Error(err))
		response.FailWithMessage("对账失败", c)
		return
	}
	response.OkWithData(res, c)
}

// GetLastReconcile
// @Tags      JxcLedger
// @Summary   最近一次对账结果（不触发扫描）
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     warehouseId  query     int  false  "仓库ID；0=全部"
// @Success   200          {object}  response.Response{data=jxcsvc.ReconcileResult,msg=string}
// @Router    /jxc/ledger/reconcile/last [get]
func (a *LedgerApi) GetLastReconcile(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.DefaultQuery("warehouseId", "0"), 10, 64)
	id := uint(warehouseID)
	res := jxcsvc.ServiceGroupApp.Ledger.LastReconcile(id)
	if res == nil {
		scope := "all"
		if id > 0 {
			scope = fmt.Sprintf("warehouse:%d", id)
		}
		response.OkWithData(gin.H{
			"ok": false, "checkedAt": nil, "durationMs": 0, "scope": scope,
			"checkedSkus": 0, "count": 0, "violations": []interface{}{}, "stale": true,
		}, c)
		return
	}
	response.OkWithData(res, c)
}

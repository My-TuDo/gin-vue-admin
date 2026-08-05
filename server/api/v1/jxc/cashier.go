package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CashierApi 收银台接口
type CashierApi struct{}

// Checkout 收银结算（收款即出库）
// @Tags     JxcCashier
// @Summary  收银结算（生成已出库销售单 + 扣库存 + 流水）
// @Security ApiKeyAuth
// @Accept   application/json
// @Produce  application/json
// @Param    data body jxc.POSCheckoutReq true "收银请求"
// @Success  200 {object} response.Response{data=jxc.SaleOrder} "销售单"
// @Router   /jxc/cashier/checkout [post]
func (a *CashierApi) Checkout(c *gin.Context) {
	var req jxc.POSCheckoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		global.GVA_LOG.Error("参数错误!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	order, err := cashierService.Checkout(c.Request.Context(), req, saleOperator(c))
	if err != nil {
		global.GVA_LOG.Error("收银结算失败!", zap.Error(err))
		response.FailWithMessage("收款失败："+err.Error(), c)
		return
	}
	response.OkWithData(order, c)
}

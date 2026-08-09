package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PosScanApi 扫码枪队列 API
type PosScanApi struct{}

// ScanReq 扫码上架请求
type ScanReq struct {
	Barcode string `json:"barcode"`          // 小程序扫码传入
	SkuID   uint   `json:"skuId"`            // 或直接传 SKU ID
	Qty     int    `json:"qty"`              // 数量，默认 1
}

// Scan 小程序扫码上架
func (a *PosScanApi) Scan(c *gin.Context) {
	var req ScanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误："+err.Error(), c)
		return
	}
	scan, err := posScanService.CreateScan(c.Request.Context(), req.Barcode, req.SkuID, req.Qty)
	if err != nil {
		global.GVA_LOG.Error("扫码上架失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(scan, "扫码上架成功", c)
}

// ListPending PC 收银台轮询待处理条目
func (a *PosScanApi) ListPending(c *gin.Context) {
	list, err := posScanService.ListPending(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("查询扫码队列失败", zap.Error(err))
		response.FailWithMessage("查询失败："+err.Error(), c)
		return
	}
	response.OkWithDetailed(list, "查询成功", c)
}

// ConfirmReq 确认请求（内嵌 GVA IdsReq，JSON: {"ids":[1,2]}）
type ConfirmReq struct {
	request.IdsReq
}

// Confirm PC 收银台确认扫码条目（加入购物车后标记）
func (a *PosScanApi) Confirm(c *gin.Context) {
	var req ConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误："+err.Error(), c)
		return
	}
	ids := make([]uint, len(req.Ids))
	for i, v := range req.Ids {
		ids[i] = uint(v)
	}
	if err := posScanService.ConfirmScan(c.Request.Context(), ids); err != nil {
		global.GVA_LOG.Error("确认扫码条目失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("确认成功", c)
}

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
	Session string `json:"session"`          // 收银台码（小程序绑定后携带）
	Barcode string `json:"barcode"`          // 小程序扫码传入
	SkuID   uint   `json:"skuId"`            // 或直接传 SKU ID
	Qty     int    `json:"qty"`              // 数量，默认 1
}

// CreateSessionReq 生成收银台码请求
type CreateSessionReq struct {
	Remark string `json:"remark"` // 备注（如：1号收银机）
}

// Scan 小程序扫码上架（投递到指定收银台会话）
func (a *PosScanApi) Scan(c *gin.Context) {
	var req ScanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误："+err.Error(), c)
		return
	}
	scan, err := posScanService.CreateScan(c.Request.Context(), req.Session, req.Barcode, req.SkuID, req.Qty)
	if err != nil {
		global.GVA_LOG.Error("扫码上架失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(scan, "扫码上架成功", c)
}

// ListPending PC 收银台轮询本会话未消费条目
func (a *PosScanApi) ListPending(c *gin.Context) {
	session := c.Query("session")
	list, err := posScanService.ListPending(c.Request.Context(), session)
	if err != nil {
		global.GVA_LOG.Error("查询扫码队列失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(list, "查询成功", c)
}

// ConfirmReq 确认请求（内嵌 GVA IdsReq，JSON: {"ids":[1,2]}）
type ConfirmReq struct {
	request.IdsReq
}

// Confirm PC 收银台消费确认（自动加入购物车后标记）
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

// CreateSession PC 收银台设置页：生成新收银台码
func (a *PosScanApi) CreateSession(c *gin.Context) {
	var req CreateSessionReq
	_ = c.ShouldBindJSON(&req)
	ss, err := posScanService.CreateSession(c.Request.Context(), req.Remark)
	if err != nil {
		global.GVA_LOG.Error("生成收银台码失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(ss, "生成成功", c)
}

// CheckSession 小程序绑定收银台：校验码有效性
func (a *PosScanApi) CheckSession(c *gin.Context) {
	ss, err := posScanService.CheckSession(c.Request.Context(), c.Query("code"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(ss, "收银台码有效", c)
}

// DisableSession PC 收银台设置页：作废当前码（重置后小程序需重新绑定）
func (a *PosScanApi) DisableSession(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误："+err.Error(), c)
		return
	}
	if err := posScanService.DisableSession(c.Request.Context(), req.Code); err != nil {
		global.GVA_LOG.Error("作废收银台码失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已重置，小程序需重新绑定", c)
}

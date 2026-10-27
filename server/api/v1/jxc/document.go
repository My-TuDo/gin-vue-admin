package jxc

import (
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	jxcmodel "github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	jxcsvc "github.com/flipped-aurora/gin-vue-admin/server/service/jxc"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DocumentApi 单据接口（采购 / 销售）。
//
// 设计约定：
//   - handler 只做"解析 → 调 service → 统一响应"，不含业务判断；
//   - 过账时间由服务端决定（可传 occurredAt 覆盖，用于补录），不接受客户端随意指定；
//   - 所有错误都带可读 msg，前端直接展示，不再二次翻译。
type DocumentApi struct{}

// ---------- 采购 ----------

// CreatePurchase 创建采购单（草稿）
func (a *DocumentApi) CreatePurchase(c *gin.Context) {
	var req jxcmodel.PurchaseOrder
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	order, err := jxcsvc.ServiceGroupApp.Purchase.Create(c.Request.Context(), &req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(order, c)
}

// ApprovePurchase 审核采购单
func (a *DocumentApi) ApprovePurchase(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := jxcsvc.ServiceGroupApp.Purchase.Approve(c.Request.Context(), id, operator(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("审核成功", c)
}

// ReceivePurchase 采购入库（产生账本行）
func (a *DocumentApi) ReceivePurchase(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	err := jxcsvc.ServiceGroupApp.Purchase.Receive(c.Request.Context(), id, operator(c), parseOccurredAt(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("入库成功", c)
}

// CancelPurchase 取消采购单
func (a *DocumentApi) CancelPurchase(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := jxcsvc.ServiceGroupApp.Purchase.Cancel(c.Request.Context(), id, operator(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已取消", c)
}

// GetPurchase 采购单详情
func (a *DocumentApi) GetPurchase(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	order, err := jxcsvc.ServiceGroupApp.Purchase.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(order, c)
}

// GetPurchasePage 采购单列表（支持 status/keyword/warehouseId/partnerId/日期/reversed）
func (a *DocumentApi) GetPurchasePage(c *gin.Context) {
	info, ok := bindDocumentSearch(c)
	if !ok {
		return
	}
	list, total, err := jxcsvc.ServiceGroupApp.Purchase.Search(c.Request.Context(), info)
	if err != nil {
		global.GVA_LOG.Error("获取采购单列表失败", zap.Error(err))
		response.FailWithMessage("获取采购单列表失败："+err.Error(), c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: info.Page, PageSize: info.PageSize}, c)
}

// ---------- 销售 ----------

// CreateSale 开单（自动预留库存）
func (a *DocumentApi) CreateSale(c *gin.Context) {
	var req jxcmodel.SaleOrder
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	order, err := jxcsvc.ServiceGroupApp.Sale.Create(c.Request.Context(), &req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(order, c)
}

// ShipSale 销售出库（产生账本行并回填成本）
func (a *DocumentApi) ShipSale(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	err := jxcsvc.ServiceGroupApp.Sale.Ship(c.Request.Context(), id, operator(c), parseOccurredAt(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("出库成功", c)
}

// CancelSale 取消销售单（释放预留）
func (a *DocumentApi) CancelSale(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := jxcsvc.ServiceGroupApp.Sale.Cancel(c.Request.Context(), id, operator(c)); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已取消", c)
}

// GetSale 销售单详情
func (a *DocumentApi) GetSale(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	order, err := jxcsvc.ServiceGroupApp.Sale.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(order, c)
}

// GetSalePage 销售单列表（支持 status/keyword/warehouseId/partnerId/日期/reversed）
func (a *DocumentApi) GetSalePage(c *gin.Context) {
	info, ok := bindDocumentSearch(c)
	if !ok {
		return
	}
	list, total, err := jxcsvc.ServiceGroupApp.Sale.Search(c.Request.Context(), info)
	if err != nil {
		global.GVA_LOG.Error("获取销售单列表失败", zap.Error(err))
		response.FailWithMessage("获取销售单列表失败："+err.Error(), c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: info.Page, PageSize: info.PageSize}, c)
}

// ---------- 工具 ----------

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailWithMessage("单据 ID 不合法", c)
		return 0, false
	}
	return uint(id), true
}

func operator(c *gin.Context) string {
	if name := utils.GetUserName(c); name != "" {
		return name
	}
	return "system"
}

// parseOccurredAt 允许补录（后端仍会校验期间是否已结账）
func parseOccurredAt(c *gin.Context) time.Time {
	s := c.Query("occurredAt")
	if s == "" {
		return time.Time{}
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

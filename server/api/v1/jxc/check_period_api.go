package jxc

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	jxcsvc "github.com/flipped-aurora/gin-vue-admin/server/service/jxc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CheckApi 盘点
type CheckApi struct{}

// CreateCheck 建盘点单（快照账面数）
func (a *CheckApi) CreateCheck(c *gin.Context) {
	var req struct {
		WarehouseID uint   `json:"warehouseId"`
		Remark      string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	check, err := jxcsvc.ServiceGroupApp.Check.Create(c.Request.Context(), req.WarehouseID, req.Remark)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(check, c)
}

// UpdateCheckItems 录入实盘数
func (a *CheckApi) UpdateCheckItems(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Items []jxcsvc.StockCountInput `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	if err := jxcsvc.ServiceGroupApp.Check.UpdateItems(c.Request.Context(), id, req.Items); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已保存", c)
}

// PostCheck 完成盘点（差异过账）
func (a *CheckApi) PostCheck(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	check, err := jxcsvc.ServiceGroupApp.Check.Post(c.Request.Context(), id, operator(c), parseOccurredAt(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(check, c)
}

// CancelCheck 取消盘点
func (a *CheckApi) CancelCheck(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := jxcsvc.ServiceGroupApp.Check.Cancel(c.Request.Context(), id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("已取消", c)
}

// GetCheck 详情
func (a *CheckApi) GetCheck(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	check, err := jxcsvc.ServiceGroupApp.Check.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(check, c)
}

// GetCheckPage 列表
func (a *CheckApi) GetCheckPage(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	list, total, err := jxcsvc.ServiceGroupApp.Check.Page(c.Request.Context(), page, size)
	if err != nil {
		global.GVA_LOG.Error("获取盘点单失败", zap.Error(err))
		response.FailWithMessage("获取盘点单失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: page, PageSize: size}, c)
}

// PeriodApi 月结与报表
type PeriodApi struct{}

// ClosePeriod 结账
func (a *PeriodApi) ClosePeriod(c *gin.Context) {
	var req struct {
		YearMonth string `json:"yearMonth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return
	}
	p, err := jxcsvc.ServiceGroupApp.Period.Close(c.Request.Context(), req.YearMonth, operator(c))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(p, c)
}

// GetPeriods 期间列表
func (a *PeriodApi) GetPeriods(c *gin.Context) {
	list, err := jxcsvc.ServiceGroupApp.Period.List(c.Request.Context())
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}

// GetSnapshot 某期间的余额快照
func (a *PeriodApi) GetSnapshot(c *gin.Context) {
	id, _ := strconv.ParseUint(c.DefaultQuery("periodId", "0"), 10, 64)
	list, err := jxcsvc.ServiceGroupApp.Period.Snapshot(c.Request.Context(), uint(id))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}

// GetMonthlySummary 月度经营汇总
func (a *PeriodApi) GetMonthlySummary(c *gin.Context) {
	ym := c.DefaultQuery("month", "")
	if ym == "" {
		response.FailWithMessage("请指定月份（YYYY-MM）", c)
		return
	}
	s, err := jxcsvc.ServiceGroupApp.Period.Summary(c.Request.Context(), ym)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(s, c)
}

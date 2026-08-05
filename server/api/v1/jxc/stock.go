package jxc

import (
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StockApi 库存中心接口
type StockApi struct{}

// stockOperator 当前操作人
func stockOperator(c *gin.Context) string {
	return fmt.Sprintf("%d", utils.GetUserID(c))
}

// GetStockPage 库存分页查询
// @Tags     JxcStock
// @Summary  库存分页查询（仓库/关键词/预警过滤）
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    page        query     int     true  "页码"
// @Param    pageSize    query     int     true  "每页条数"
// @Param    keyword     query     string  false "SKU编码/商品名"
// @Param    warehouseId query     uint    false "仓库ID"
// @Param    lowStock    query     bool    false "仅看预警"
// @Success  200         {object}  response.PageResult{list=[]jxc.Stock,total=int64}  "库存列表"
// @Router   /jxc/stock/page [get]
func (a *StockApi) GetStockPage(c *gin.Context) {
	var q jxc.StockQuery
	_ = c.ShouldBindQuery(&q)
	list, total, err := stockService.GetStockPage(c.Request.Context(), q)
	if err != nil {
		global.GVA_LOG.Error("获取库存列表失败!", zap.Error(err))
		response.FailWithMessage("获取库存列表失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: q.Page, PageSize: q.PageSize}, c)
}

// GetStockLogPage 库存流水分页查询
// @Tags     JxcStock
// @Summary  库存流水查询（仓库/SKU/业务类型过滤）
// @Security ApiKeyAuth
// @Produce  application/json
// @Param    page         query     int     true  "页码"
// @Param    pageSize     query     int     true  "每页条数"
// @Param    warehouseId  query     uint    false "仓库ID"
// @Param    skuId        query     uint    false "SKU ID"
// @Param    businessType query     string  false "业务类型"
// @Success  200          {object}  response.PageResult{list=[]jxc.StockLog,total=int64}  "流水列表"
// @Router   /jxc/stock/log/page [get]
func (a *StockApi) GetStockLogPage(c *gin.Context) {
	var q jxc.StockLogQuery
	_ = c.ShouldBindQuery(&q)
	list, total, err := stockService.GetStockLogPage(c.Request.Context(), q)
	if err != nil {
		global.GVA_LOG.Error("获取库存流水失败!", zap.Error(err))
		response.FailWithMessage("获取库存流水失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: q.Page, PageSize: q.PageSize}, c)
}

// DirectInRequest 直接入库请求
type DirectInRequest struct {
	WarehouseID uint   `json:"warehouseId" binding:"required"`
	SkuID       uint   `json:"skuId" binding:"required"`
	Qty         int    `json:"qty" binding:"required,gt=0"`
	Remark      string `json:"remark"`
}

// DirectIn 直接入库
// @Tags     JxcStock
// @Summary  直接入库（不关联采购单）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      jxc.DirectInRequest  true "仓库/SKU/数量/备注"
// @Success  200  {object}  response.Response{msg=string}  "入库成功"
// @Router   /jxc/stock/direct-in [post]
func (a *StockApi) DirectIn(c *gin.Context) {
	var req DirectInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := stockService.DirectIn(c.Request.Context(), req.WarehouseID, req.SkuID, req.Qty, req.Remark, stockOperator(c)); err != nil {
		global.GVA_LOG.Error("直接入库失败!", zap.Error(err))
		response.FailWithMessage("入库失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("入库成功", c)
}

// DirectOut 直接出库
// @Tags     JxcStock
// @Summary  直接出库（不关联销售单, 如损耗）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      jxc.DirectInRequest  true "仓库/SKU/数量/备注"
// @Success  200  {object}  response.Response{msg=string}  "出库成功"
// @Router   /jxc/stock/direct-out [post]
func (a *StockApi) DirectOut(c *gin.Context) {
	var req DirectInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := stockService.DirectOut(c.Request.Context(), req.WarehouseID, req.SkuID, req.Qty, req.Remark, stockOperator(c)); err != nil {
		global.GVA_LOG.Error("直接出库失败!", zap.Error(err))
		response.FailWithMessage("出库失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("出库成功", c)
}

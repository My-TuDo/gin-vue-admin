package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// var basicService = service.ServiceGroupApp.JxcServiceGroup.BasicService

type BasicApi struct{}

// ==============================
// ========== 商品分类 ==========
// ==============================

// GetCategoryTree 获取商品分类树
// @Router /jxc/category/tree [get]
func (a *BasicApi) GetCategoryTree(c *gin.Context) {
	list, err := basicService.GetCategoryTree(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("获取商品分类树失败！", zap.Error(err))
		response.FailWithMessage("获取商品分类树失败", c)
		return
	}
	response.OkWithData(list, c)
}

// GetCategoryList 分页获取分类列表
// @Router /jxc/category/list [get]
func (a *BasicApi) GetCategoryList(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)
	list, total, err := basicService.GetCategoryList(c.Request.Context(), pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取商品分类列表失败！", zap.Error(err))
		response.FailWithMessage("获取商品分类列表失败", c)
		return
	}
	response.OkWithData(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, c)
}

// CreateCategory 创建商品分类
// @Router /jxc/category [post]
func (a *BasicApi) CreateCategory(c *gin.Context) {
	var category jxc.GoodsCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		response.FailWithMessage("参数绑定失败："+err.Error(), c)
		return
	}
	if err := basicService.CreateCategory(c.Request.Context(), &category); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("jxc").Err(err).Error("创建分类失败")
		response.FailWithMessage("创建分类失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("创建分类成功", c)
}

// UpdateCategory 更新商品分类
// @Router /jxc/category [put]
func (a *BasicApi) UpdateCategory(c *gin.Context) {
	var category jxc.GoodsCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		response.FailWithMessage("参数绑定失败："+err.Error(), c)
		return
	}
	if err := basicService.UpdateCategory(c.Request.Context(), &category); err != nil {
		global.GVA_LOG.Error("更新分类失败！", zap.Error(err))
		response.FailWithMessage("更新分类失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("更新分类成功", c)
}

// DeleteCategory 删除商品分类
// @Router /jxc/category/{id} [delete]
func (a *BasicApi) DeleteCategory(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteCategory(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage("删除分类失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("删除分类成功", c)
}

// DeleteCategoryForever 彻底删除商品分类
// @Tags     JxcCategory
// @Summary  彻底删除商品分类（物理删除，不可恢复）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "分类ID"
// @Success  200  {object}  response.Response{msg=string}  "彻底删除成功"
// @Router   /jxc/category/forever [delete]
func (a *BasicApi) DeleteCategoryForever(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteCategoryForever(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("彻底删除成功", c)
}

// SetCategoryStatus 启用/停用商品分类
// @Router /jxc/category/status [put]
func (a *BasicApi) SetCategoryStatus(c *gin.Context) {
	var req struct {
		ID     uint `json:"id"`
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.SetCategoryStatus(c.Request.Context(), req.ID, req.Status); err != nil {
		response.FailWithMessage("操作失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

// ==========================
// ========== 品牌 ==========
// ==========================

// GetBrandList 获取品牌列表
func (a *BasicApi) GetBrandList(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)
	list, total, err := basicService.GetBrandList(c.Request.Context(), pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取品牌列表失败！", zap.Error(err))
		response.FailWithMessage("获取品牌列表失败", c)
		return
	}
	response.OkWithData(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, c)
}

// GetAllBrands 获取品牌
func (a *BasicApi) GetAllBrands(c *gin.Context) {
	list, err := basicService.GetAllBrands(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("获取品牌失败！", zap.Error(err))
		response.FailWithMessage("获取品牌失败！", c)
		return
	}
	response.OkWithData(list, c)
}

// CreateBrand 添加品牌
func (a *BasicApi) CreateBrand(c *gin.Context) {
	var brand jxc.Brand
	if err := c.ShouldBindJSON(&brand); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.CreateBrand(c.Request.Context(), &brand); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("jxc").Err(err).Error("创建品牌失败")
		response.FailWithMessage("创建失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

// UpdateBrand 更新品牌
func (a *BasicApi) UpdateBrand(c *gin.Context) {
	var brand jxc.Brand
	if err := c.ShouldBindJSON(&brand); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.UpdateBrand(c.Request.Context(), &brand); err != nil {
		global.GVA_LOG.Error("更新品牌失败！", zap.Error(err))
		response.FailWithMessage("更新失败："+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeleteBrand 删除品牌
func (a *BasicApi) DeleteBrand(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteBrand(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage("操作失败", c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

// DeleteBrandForever 彻底删除品牌
// @Tags     JxcBrand
// @Summary  彻底删除品牌（物理删除，不可恢复）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "品牌ID"
// @Success  200  {object}  response.Response{msg=string}  "彻底删除成功"
// @Router   /jxc/brand/forever [delete]
func (a *BasicApi) DeleteBrandForever(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteBrandForever(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("彻底删除成功", c)
}

// SetBrandStatus 启用/停用品牌
func (a *BasicApi) SetBrandStatus(c *gin.Context) {
	var req struct {
		ID     uint `json:"id"`
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.SetBrandStatus(c.Request.Context(), req.ID, req.Status); err != nil {
		response.FailWithMessage("操作失败", c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

// ============================
// ========== 供应商 ==========
// ============================

// GetSuoolierList 获取供应商列表
func (a *BasicApi) GetSupplierList(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)
	list, total, err := basicService.GetSupplierList(c.Request.Context(), pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取供应商列表失败!", zap.Error(err))
		response.FailWithMessage("获取供应商列表失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: pageInfo.Page, PageSize: pageInfo.PageSize}, c)
}

// GetAllSupplier 获取供应商
func (a *BasicApi) GetAllSuppliers(c *gin.Context) {
	list, err := basicService.GetAllSuppliers(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("获取供应商失败!", zap.Error(err))
		response.FailWithMessage("获取供应商失败", c)
		return
	}
	response.OkWithData(list, c)
}

// CreateSupplier 添加供应商
func (a *BasicApi) CreateSupplier(c *gin.Context) {
	var supplier jxc.Supplier
	if err := c.ShouldBindJSON(&supplier); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.CreateSupplier(c.Request.Context(), &supplier); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("jxc").Err(err).Error("创建供应商失败")
		response.FailWithMessage("创建失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

// UpateSupplier 更新供应商
func (a *BasicApi) UpdateSupplier(c *gin.Context) {
	var supplier jxc.Supplier
	if err := c.ShouldBindJSON(&supplier); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.UpdateSupplier(c.Request.Context(), &supplier); err != nil {
		global.GVA_LOG.Error("更新供应商失败!", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeleteSupplier 删除供应商
func (a *BasicApi) DeleteSupplier(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteSupplier(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteSupplierForever 彻底删除供应商
// @Tags     JxcSupplier
// @Summary  彻底删除供应商（物理删除，不可恢复）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "供应商ID"
// @Success  200  {object}  response.Response{msg=string}  "彻底删除成功"
// @Router   /jxc/supplier/forever [delete]
func (a *BasicApi) DeleteSupplierForever(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteSupplierForever(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("彻底删除成功", c)
}

// SetSupplierStatus 启用/停用供应商
func (a *BasicApi) SetSupplierStatus(c *gin.Context) {
	var req struct {
		ID     uint `json:"id"`
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.SetSupplierStatus(c.Request.Context(), req.ID, req.Status); err != nil {
		response.FailWithMessage("操作失败", c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

// ==========================
// ========== 客户 ==========
// ==========================

// GetCustomerList 获取客户列表
func (a *BasicApi) GetCustomerList(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)
	list, total, err := basicService.GetCustomerList(c.Request.Context(), pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取客户列表失败!", zap.Error(err))
		response.FailWithMessage("获取客户列表失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: pageInfo.Page, PageSize: pageInfo.PageSize}, c)
}

// GetAllCustomers 获取客户
func (a *BasicApi) GetAllCustomers(c *gin.Context) {
	list, err := basicService.GetAllCustomers(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("获取客户失败!", zap.Error(err))
		response.FailWithMessage("获取客户失败", c)
		return
	}
	response.OkWithData(list, c)
}

// CreaeteCustomer 添加客户
func (a *BasicApi) CreateCustomer(c *gin.Context) {
	var customer jxc.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.CreateCustomer(c.Request.Context(), &customer); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("jxc").Err(err).Error("创建客户失败")
		response.FailWithMessage("创建失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

// UpdataCustomer 更新客户
func (a *BasicApi) UpdateCustomer(c *gin.Context) {
	var customer jxc.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.UpdateCustomer(c.Request.Context(), &customer); err != nil {
		global.GVA_LOG.Error("更新客户失败!", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeleteCustomer 删除客户
func (a *BasicApi) DeleteCustomer(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteCustomer(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteCustomerForever 彻底删除客户
// @Tags     JxcCustomer
// @Summary  彻底删除客户（物理删除，不可恢复）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "客户ID"
// @Success  200  {object}  response.Response{msg=string}  "彻底删除成功"
// @Router   /jxc/customer/forever [delete]
func (a *BasicApi) DeleteCustomerForever(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteCustomerForever(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("彻底删除成功", c)
}

// SetCustomerStatus 启用/停用客户
func (a *BasicApi) SetCustomerStatus(c *gin.Context) {
	var req struct {
		ID     uint `json:"id"`
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.SetCustomerStatus(c.Request.Context(), req.ID, req.Status); err != nil {
		response.FailWithMessage("操作失败", c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

// ==========================
// ========== 仓库 ==========
// ==========================

// GetWarehouseList 获取仓库列表
func (a *BasicApi) GetWarehouseList(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)
	list, total, err := basicService.GetWarehouseList(c.Request.Context(), pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取仓库列表失败!", zap.Error(err))
		response.FailWithMessage("获取仓库列表失败", c)
		return
	}
	response.OkWithData(response.PageResult{List: list, Total: total, Page: pageInfo.Page, PageSize: pageInfo.PageSize}, c)
}

// GetAllWarehouse 获取仓库
func (a *BasicApi) GetAllWarehouses(c *gin.Context) {
	list, err := basicService.GetAllWarehouses(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("获取仓库失败!", zap.Error(err))
		response.FailWithMessage("获取仓库失败", c)
		return
	}
	response.OkWithData(list, c)
}

// CreateWarehouse 添加仓库
func (a *BasicApi) CreateWarehouse(c *gin.Context) {
	var warehouse jxc.Warehouse
	if err := c.ShouldBindJSON(&warehouse); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.CreateWarehouse(c.Request.Context(), &warehouse); err != nil {
		logger.WithCtx(c.Request.Context()).Mod("jxc").Err(err).Error("创建仓库失败")
		response.FailWithMessage("创建失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

// UpdateWarehouse 更新仓库
func (a *BasicApi) UpdateWarehouse(c *gin.Context) {
	var warehouse jxc.Warehouse
	if err := c.ShouldBindJSON(&warehouse); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.UpdateWarehouse(c.Request.Context(), &warehouse); err != nil {
		global.GVA_LOG.Error("更新仓库失败!", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeleteWarehouse 删除仓库
func (a *BasicApi) DeleteWarehouse(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteWarehouse(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteWarehouseForever 彻底删除仓库
// @Tags     JxcWarehouse
// @Summary  彻底删除仓库（物理删除，不可恢复）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body      request.GetById  true "仓库ID"
// @Success  200  {object}  response.Response{msg=string}  "彻底删除成功"
// @Router   /jxc/warehouse/forever [delete]
func (a *BasicApi) DeleteWarehouseForever(c *gin.Context) {
	var req request.GetById
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.DeleteWarehouseForever(c.Request.Context(), req.Uint()); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("彻底删除成功", c)
}

// SetWarehouseStatus 启用/停用仓库
func (a *BasicApi) SetWarehouseStatus(c *gin.Context) {
	var req struct {
		ID     uint `json:"id"`
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := basicService.SetWarehouseStatus(c.Request.Context(), req.ID, req.Status); err != nil {
		response.FailWithMessage("操作失败", c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

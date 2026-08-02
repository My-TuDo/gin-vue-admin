package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type BasicRouter struct{}

func (r *BasicRouter) InitBasicRouter(Router *gin.RouterGroup) {
	// 需要操作日志记录的写接口
	recordGroup := Router.Group("jxc").Use(middleware.OperationRecord())
	// 读接口（不记操作日志）
	readOnlyGroup := Router.Group("jxc")

	// ===== 商品分类 =====
	readOnlyGroup.GET("category/tree", basicApi.GetCategoryTree)
	readOnlyGroup.GET("category/list", basicApi.GetCategoryList)
	recordGroup.POST("category", basicApi.CreateCategory)
	recordGroup.PUT("category", basicApi.UpdateCategory)
	recordGroup.DELETE("category", basicApi.DeleteCategory)
	recordGroup.PUT("category/status", basicApi.SetCategoryStatus)

	// ====== 品牌 ======
	readOnlyGroup.GET("brand/list", basicApi.GetBrandList)
	readOnlyGroup.GET("brand/all", basicApi.GetAllBrands)
	recordGroup.POST("brand", basicApi.CreateBrand)
	recordGroup.PUT("brand", basicApi.UpdateBrand)
	recordGroup.DELETE("brand", basicApi.DeleteBrand)
	recordGroup.PUT("brand/status", basicApi.SetBrandStatus)

	// ===== 供应商 ======
	readOnlyGroup.GET("supplier/list", basicApi.GetSupplierList)
	readOnlyGroup.GET("supplier/all", basicApi.GetAllSuppliers)
	recordGroup.POST("supplier", basicApi.CreateSupplier)
	recordGroup.PUT("supplier", basicApi.UpdateSupplier)
	recordGroup.DELETE("supplier", basicApi.DeleteSupplier)
	recordGroup.PUT("supplier/status", basicApi.SetSupplierStatus)

	// ===== 客户 =====
	readOnlyGroup.GET("customer/list", basicApi.GetCustomerList)
	readOnlyGroup.GET("customer/all", basicApi.GetAllCustomers)
	recordGroup.POST("customer", basicApi.CreateCustomer)
	recordGroup.PUT("customer", basicApi.UpdateCustomer)
	recordGroup.DELETE("customer", basicApi.DeleteCustomer)

	// ====== 仓库 =====
	readOnlyGroup.GET("warehouse/list", basicApi.GetWarehouseList)
	readOnlyGroup.GET("warehouse/all", basicApi.GetAllWarehouses)
	recordGroup.POST("warehouse", basicApi.CreateWarehouse)
	recordGroup.PUT("warehouse", basicApi.UpdateWarehouse)
	recordGroup.DELETE("warehouse", basicApi.DeleteWarehouse)
	recordGroup.PUT("warehouse/status", basicApi.SetWarehouseStatus)
}

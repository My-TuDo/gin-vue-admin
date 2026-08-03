package jxc

import (
	"context"
	"errors"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// BasicService 提供基础数据服务
type BasicService struct{}

// ========== 商品分类 ==========

// GetCategoryTree 获取商品分类树形结构
func (s *BasicService) GetCategoryTree(ctx context.Context) (list []jxc.GoodsCategory, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.GoodsCategory{})
	err = db.Where("parent_id IS NULL").Order("sort").Find(&list).Error
	if err != nil {
		return
	}
	for i := range list {
		s.fillChildren(ctx, &list[i])
	}
	return
}

// fillChildren 递归填充子分类
func (s *BasicService) fillChildren(ctx context.Context, parent *jxc.GoodsCategory) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.GoodsCategory{})
	db.Where("parent_id = ?", parent.ID).Order("sort").Find(&parent.Children)
	for i := range parent.Children {
		s.fillChildren(ctx, &parent.Children[i]) // 递归
	}
}

// GetCategoryList 分页查询分类列表
func (s *BasicService) GetCategoryList(ctx context.Context, info request.PageInfo) (list interface{}, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.GoodsCategory{})
	if info.Keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+info.Keyword+"%", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	var records []jxc.GoodsCategory
	limit, offset := info.LimitOffset()
	err = db.Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

// CreateCategory 创建商品分类
func (s *BasicService) CreateCategory(ctx context.Context, c *jxc.GoodsCategory) error {
	return global.GVA_DB.WithContext(ctx).Create(c).Error
}

// UpdateCategory 更新分类
func (s *BasicService) UpdateCategory(ctx context.Context, c *jxc.GoodsCategory) error {
	return global.GVA_DB.WithContext(ctx).Omit("code", "created_at").Save(c).Error // 忽略code字段，避免更新时修改唯一约束
}

// DeleteCategory 删除分类（软删除，需嵌入 global.GVA_MODEL/Deleted_At）
func (s *BasicService) DeleteCategory(ctx context.Context, id uint) error {
	db := global.GVA_DB.WithContext(ctx)
	// 检查子分类
	var childCount int64
	db.Model(&jxc.GoodsCategory{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return errors.New("该分类存在子分类，请先删除子分类")
	}
	// 检查商品引用
	var goodsCount int64
	db.Model(&jxc.Goods{}).Where("category_id = ?", id).Count(&goodsCount)
	if goodsCount > 0 {
		return errors.New("该分类已被商品引用，请先删除相关商品")
	}
	return db.Delete(&jxc.GoodsCategory{}, id).Error
}

// SetCategoryStatus 启用/停用分类
func (s *BasicService) SetCategoryStatus(ctx context.Context, id uint, status int8) error {
	return global.GVA_DB.WithContext(ctx).Model(&jxc.GoodsCategory{}).Where("id = ?", id).Update("status", status).Error
}

// -------------------------------------------------------------------------------------------------------------------------

// ========== 品牌 ==========

// GetBrandList 分页查询品牌列表
// 逻辑于分类列表类似，可优化提取出通用函数
func (s *BasicService) GetBrandList(ctx context.Context, info request.PageInfo) (list interface{}, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.Brand{})
	if info.Keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+info.Keyword+"%", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	var records []jxc.Brand
	limit, offset := info.LimitOffset()
	err = db.Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

// GetAllBrands 获取所有品牌列表（不分页）
func (s *BasicService) GetAllBrands(ctx context.Context) (list []jxc.Brand, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("status = 1").Find(&list).Error
	return
}

// 以下三种也可以优化提取为通用的函数方法:Create、Update、Delete
// CreateBrand 创建品牌
func (s *BasicService) CreateBrand(ctx context.Context, b *jxc.Brand) error {
	return global.GVA_DB.WithContext(ctx).Create(b).Error
}

// UpdateBrand 更新品牌
func (s *BasicService) UpdateBrand(ctx context.Context, b *jxc.Brand) error {
	return global.GVA_DB.WithContext(ctx).Omit("code", "created_at").Save(b).Error // 忽略code字段，避免更新时修改唯一约束
}

// DeleteBrand 删除品牌（软删除，需嵌入 global.GVA_MODEL/Deleted_At）
func (s *BasicService) DeleteBrand(ctx context.Context, id uint) error {
	var goodsCount int64
	global.GVA_DB.WithContext(ctx).Model(&jxc.Goods{}).Where("brand_id = ?", id).Count(&goodsCount)
	if goodsCount > 0 {
		return errors.New("该品牌已被商品引用，请先删除相关商品")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&jxc.Brand{}, id).Error
}

// SetBrandStatus 启用/停用品牌
func (s *BasicService) SetBrandStatus(ctx context.Context, id uint, status int8) error {
	return global.GVA_DB.WithContext(ctx).Model(&jxc.Brand{}).Where("id = ?", id).
		Update("status", status).Error
}

// -------------------------------------------------------------------------------------------------------------

// ========== 供应商 ==========

// GetSupplierList 分页查询供应商列表
func (s *BasicService) GetSupplierList(ctx context.Context, info request.PageInfo) (list interface{}, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.Supplier{})
	if info.Keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+info.Keyword+"%", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	var records []jxc.Supplier
	limit, offset := info.LimitOffset()
	err = db.Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

// GetAllSuppliers 获取所有供应商列表（不分页）
func (s *BasicService) GetAllSuppliers(ctx context.Context) (list []jxc.Supplier, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("status = 1").Find(&list).Error
	return
}

// CreateSupplier 创建供应商
func (s *BasicService) CreateSupplier(ctx context.Context, spli *jxc.Supplier) error {
	return global.GVA_DB.WithContext(ctx).Create(spli).Error
}

// UpdateSupplier 更新供应商
func (s *BasicService) UpdateSupplier(ctx context.Context, spli *jxc.Supplier) error {
	return global.GVA_DB.WithContext(ctx).Omit("code", "created_at").Save(spli).Error // 忽略code字段，避免更新时修改唯一约束
}

type PurchaseOrder struct{} // 占位使用，采购订单表暂未实现

// DeleteSupplier 删除供应商（软删除，需嵌入 global.GVA_MODEL/Deleted_At）
func (s *BasicService) DeleteSupplier(ctx context.Context, id uint) error {
	var orderCount int64
	global.GVA_DB.WithContext(ctx).Model(&PurchaseOrder{}).Where("supplier_id = ?", id).Count(&orderCount) // PurchaseOrder 采购订单表暂未实现
	if orderCount > 0 {
		return errors.New("该供应商已被采购订单引用，无法删除")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&jxc.Supplier{}, id).Error
}

// SetSupplierStatus 启用/停用供应商
func (s *BasicService) SetSupplierStatus(ctx context.Context, id uint, status int8) error {
	return global.GVA_DB.WithContext(ctx).Model(&jxc.Supplier{}).Where("id = ?", id).
		Update("status", status).Error
}

// ------------------------------------------------------------------------------------------------------------------------------------

// ========== 客户 ==========

// GetCustomerList 分页查询客户列表
func (s *BasicService) GetCustomerList(ctx context.Context, info request.PageInfo) (list interface{}, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.Customer{})
	if info.Keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+info.Keyword+"%", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	var records []jxc.Customer
	limit, offset := info.LimitOffset()
	err = db.Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

// GetAllCustomers 获取所有客户列表（不分页）
func (s *BasicService) GetAllCustomers(ctx context.Context) (list []jxc.Customer, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("status = 1").Find(&list).Error
	return
}

// CreateCustomer 创建客户
func (s *BasicService) CreateCustomer(ctx context.Context, c *jxc.Customer) error {
	return global.GVA_DB.WithContext(ctx).Create(c).Error
}

// UpdateCustomer 更新客户
func (s *BasicService) UpdateCustomer(ctx context.Context, c *jxc.Customer) error {
	return global.GVA_DB.WithContext(ctx).Omit("code", "created_at").Save(c).Error // 忽略code字段，避免更新时修改唯一约束
}

type SalesOrder struct{} // 占位使用，销售订单表暂未实现

// DeleteCustomer 删除客户（软删除，需嵌入 global.GVA_MODEL/Deleted_At）
func (s *BasicService) DeleteCustomer(ctx context.Context, id uint) error {
	var orderCount int64
	global.GVA_DB.WithContext(ctx).Model(&SalesOrder{}).Where("customer_id = ?", id).Count(&orderCount) // SalesOrder 销售订单表暂未实现
	if orderCount > 0 {
		return errors.New("该客户已被销售订单引用，无法删除")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&jxc.Customer{}, id).Error
}

// SetCustomerStatus 启用/停用客户
func (s *BasicService) SetCustomerStatus(ctx context.Context, id uint, status int8) error {
	return global.GVA_DB.WithContext(ctx).Model(&jxc.Customer{}).Where("id = ?", id).
		Update("status", status).Error
}

// -------------------------------------------------------------------------------------------------------------------------------------

// ========== 仓库 ==========

// GetWarehouseList 分页查询仓库列表
func (s *BasicService) GetWarehouseList(ctx context.Context, info request.PageInfo) (list interface{}, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.Warehouse{})
	if info.Keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+info.Keyword+"%", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	var records []jxc.Warehouse
	limit, offset := info.LimitOffset()
	err = db.Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

// GetAllWarehouses 获取所有仓库列表（不分页）
func (s *BasicService) GetAllWarehouses(ctx context.Context) (list []jxc.Warehouse, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("status = 1").Find(&list).Error
	return
}

// CreateWarehouse 创建仓库
func (s *BasicService) CreateWarehouse(ctx context.Context, w *jxc.Warehouse) error {
	return global.GVA_DB.WithContext(ctx).Create(w).Error
}

// UpdateWarehouse 更新仓库
func (s *BasicService) UpdateWarehouse(ctx context.Context, w *jxc.Warehouse) error {
	return global.GVA_DB.WithContext(ctx).Omit("code", "created_at").Save(w).Error // 忽略code字段，避免更新时修改唯一约束
}

type Stock struct{} // 占位使用，库存表暂未实现

// DeleteWarehouse 删除仓库（软删除，需嵌入 global.GVA_MODEL/Deleted_At）
func (s *BasicService) DeleteWarehouse(ctx context.Context, id uint) error {
	var stockTotal int64
	global.GVA_DB.WithContext(ctx).Model(&Stock{}).Where("warehouse_id = ? AND quantity > 0", id).Count(&stockTotal)
	if stockTotal > 0 {
		return errors.New("该仓库仍存在库存，无法删除")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&jxc.Warehouse{}, id).Error
}

// SetWarehouseStatus 启用/停用仓库
func (s *BasicService) SetWarehouseStatus(ctx context.Context, id uint, status int8) error {
	return global.GVA_DB.WithContext(ctx).Model(&jxc.Warehouse{}).Where("id = ?", id).
		Update("status", status).Error
}

package jxc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

var svc = &BasicService{}
var ctx = context.Background()

// testDB : sqlite:memory: + AutoMigrate + 赋值GVA_DB + t.Cleanup还原
func testDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t, &jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{})
}

// ensureRefTable 为占位 struct，手动建表，用于测试“被引用拒绝”分支
func ensureRefTable(t *testing.T, sql string) {
	t.Helper()
	if err := global.GVA_DB.Exec(sql).Error; err != nil {
		t.Fatalf("建引用表失败： %v", err)
	}
}

// ========== 商品分类 ==========

// TestGetCategoryTree 测试获取商品分类树
func TestGetCategoryTree(t *testing.T) {
	testDB(t)
	parent := jxc.GoodsCategory{Code: "C001", Name: "男装", Sort: 1, Status: 1}
	if err := global.GVA_DB.Create(&parent).Error; err != nil {
		t.Fatalf("创造数据失败: %v", err)
	}
	child := jxc.GoodsCategory{ParentID: &parent.ID, Code: "C001-1", Name: "T恤", Sort: 1, Status: 1}
	if err := global.GVA_DB.Create(&child).Error; err != nil {
		t.Fatalf("创造数据失败: %v", err)
	}
	list, err := svc.GetCategoryTree(ctx)
	if err != nil {
		t.Fatalf("GetCategoryTree err: %v", err)
	}
	if len(list) != 1 || len(list[0].Children) != 1 {
		t.Fatalf("树结构不存在: %+v", list)
	}
}

// TestGetCategoryTree_Empty 测试获取商品分类树为空的情况
func TestGetCategoryTree_Empty(t *testing.T) {
	testDB(t)
	list, err := svc.GetCategoryTree(ctx)
	if err != nil || len(list) != 0 {
		t.Fatalf("空库应该返回空树: list=%+v, err=%v", list, err)
	}
}

// TestGetCategoryList 测试获取商品分类列表
func TestGetCategoryList(t *testing.T) {
	testDB(t)
	global.GVA_DB.Create(&jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1})
	list, total, err := svc.GetCategoryList(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(list.([]jxc.GoodsCategory)) != 1 {
		t.Fatalf("list=%v total=%d err=%v", list, total, err)
	}
}

// TestGetCategoryList_Keyword 测试获取商品分类列表带关键字搜索
func TestGetCategoryList_Keyword(t *testing.T) {
	testDB(t)
	global.GVA_DB.Create(&jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1})
	global.GVA_DB.Create(&jxc.GoodsCategory{Code: "C002", Name: "女装", Status: 1})
	_, total, err := svc.GetCategoryList(ctx, request.PageInfo{Page: 1, PageSize: 10, Keyword: "男"})
	if err != nil || total != 1 {
		t.Fatalf("keyword 过滤失败: total=%d err=%v", total, err)
	}
}

// TestGetCategoryList_Empty 测试获取商品分类列表为空的情况
func TestGetCategoryList_Empty(t *testing.T) {
	testDB(t)
	_, total, err := svc.GetCategoryList(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 0 {
		t.Fatalf("空库 total 应该为0: %d err=%v", total, err)
	}
}

// TestCreateCategory 测试创建商品分类
func TestCreateCategory(t *testing.T) {
	testDB(t)
	c := &jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	if err := svc.CreateCategory(ctx, c); err != nil {
		t.Fatalf("CreateCategory err: %v", err)
	}
	if c.ID == 0 {
		t.Fatalf("创建后应回填 ID")
	}
	var got jxc.GoodsCategory
	if err := global.GVA_DB.First(&got, c.ID).Error; err != nil {
		t.Fatalf("落库校验失败：%+v err=%v", got, err)
	}
}

// TestUpdateCategory 测试更新商品分类
func TestUpdateCategory(t *testing.T) {
	testDB(t)
	c := jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	global.GVA_DB.Create(&c)
	c.Name = "男装(改)"
	if err := svc.UpdateCategory(ctx, &c); err != nil {
		t.Fatalf("UpdateCategory err: %v", err)
	}
	var got jxc.GoodsCategory
	global.GVA_DB.First(&got, c.ID)
	if got.Name != "男装(改)" || got.Code != "C001" {
		t.Fatalf("更新后数据不正确: %+v", got)
	}
}

// TestDeleteCategory_OK 测试删除商品分类成功
func TestDeleteCategory_OK(t *testing.T) {
	testDB(t)
	c := &jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	global.GVA_DB.Create(&c)
	if err := svc.DeleteCategory(ctx, c.ID); err != nil {
		t.Fatalf("DeleteCategory err: %v", err)
	}
	var cnt int64
	global.GVA_DB.Unscoped().Model(&jxc.GoodsCategory{}).Where("id = ?", c.ID).Count(&cnt)
	if cnt != 1 { // 软删除，记录仍然在
		t.Fatalf("应为软删除，cnt=%d", cnt)
	}
}

// TestDeleteCategory_Ref 测试删除商品分类被引用拒绝
func TestDeleteCategory_Ref(t *testing.T) {
	testDB(t)
	c := &jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	global.GVA_DB.Create(&c)
	ensureRefTable(t, "CREATE TABLE goods (category_id int, deleted_at datetime)")
	global.GVA_DB.Exec("INSERT INTO goods (category_id) VALUES (?)", c.ID)
	err := svc.DeleteCategory(ctx, c.ID)
	if err == nil || err.Error() != "该分类已被商品引用，请先删除相关商品" {
		t.Fatalf("应返回引用拒绝, got %v", err)
	}
}

// TestSetCategoryStatus 测试设置商品分类状态
func TestSetCategoryStatus(t *testing.T) {
	testDB(t)
	c := &jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	global.GVA_DB.Create(&c)
	if err := svc.SetCategoryStatus(ctx, c.ID, 0); err != nil {
		t.Fatalf("SetCategoryStatus err: %v", err)
	}
	var got jxc.GoodsCategory
	global.GVA_DB.First(&got, c.ID)
	if got.Status != 0 {
		t.Fatalf("状态未更新: %+v", got)
	}
}

// ========== 品牌 ==========

// TestBrandCRUD 测试品牌的增删改查
func TestBrandCRUD(t *testing.T) {
	testDB(t)
	b := jxc.Brand{Code: "B001", Name: "耐克", Status: 1}
	if err := svc.CreateBrand(ctx, &b); err != nil || b.ID == 0 {
		t.Fatalf("create err=%v", err)
	}
	b.Name = "耐克(改)"
	if err := svc.UpdateBrand(ctx, &b); err != nil {
		t.Fatalf("update err=%v", err)
	}
	_, total, err := svc.GetBrandList(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("list total=%d err=%v", total, err)
	}
	all, err := svc.GetAllBrands(ctx)
	if err != nil || len(all) != 1 || all[0].Name != "耐克(改)" {
		t.Fatalf("getAll err=%v list=%+v", err, all)
	}
	if err := svc.SetBrandStatus(ctx, b.ID, 0); err != nil {
		t.Fatalf("setStatus err=%v", err)
	}
	if err := svc.DeleteBrand(ctx, b.ID); err != nil {
		t.Fatalf("delete err=%v", err)
	}
}

// TestDeleteBrand_Referenced 测试删除品牌时被商品引用拒绝
func TestDeleteBrand_Referenced(t *testing.T) {
	testDB(t)
	b := jxc.Brand{Code: "B001", Name: "耐克", Status: 1}
	global.GVA_DB.Create(&b)
	ensureRefTable(t, "CREATE TABLE goods (brand_id int, deleted_at datetime)")
	global.GVA_DB.Exec("INSERT INTO goods (brand_id) VALUES (?)", b.ID)
	if err := svc.DeleteBrand(ctx, b.ID); err == nil || err.Error() != "该品牌已被商品引用，请先删除相关商品" {
		t.Fatalf("应返回引用拒绝, got %v", err)
	}
}

// ========== 供应商 ==========

// TestSupplierCRUD 测试供应商的增删改查及状态切换
func TestSupplierCRUD(t *testing.T) {
	testDB(t)
	s := jxc.Supplier{Code: "S001", Name: "广州布行", Contact: "张", Phone: "13800000000", Status: 1}
	if err := svc.CreateSupplier(ctx, &s); err != nil || s.ID == 0 {
		t.Fatalf("create err=%v", err)
	}
	s.Name = "广州布行(改)"
	if err := svc.UpdateSupplier(ctx, &s); err != nil {
		t.Fatalf("update err=%v", err)
	}
	_, total, err := svc.GetSupplierList(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("list total=%d err=%v", total, err)
	}
	all, err := svc.GetAllSuppliers(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("getAll err=%v", err)
	}
	if err := svc.SetSupplierStatus(ctx, s.ID, 0); err != nil {
		t.Fatalf("setStatus err=%v", err)
	}
	if err := svc.DeleteSupplier(ctx, s.ID); err != nil {
		t.Fatalf("delete err=%v", err)
	}
}

// TestDeleteSupplier_Referenced 测试删除供应商时被采购订单引用拒绝
func TestDeleteSupplier_Referenced(t *testing.T) {
	testDB(t)
	s := jxc.Supplier{Code: "S001", Name: "广州布行", Status: 1}
	global.GVA_DB.Create(&s)
	ensureRefTable(t, "CREATE TABLE purchase_order (supplier_id int, deleted_at datetime)")
	global.GVA_DB.Exec("INSERT INTO purchase_order (supplier_id, deleted_at) VALUES (?, NULL)", s.ID)
	if err := svc.DeleteSupplier(ctx, s.ID); err == nil || err.Error() != "该供应商已被采购订单引用，无法删除" {
		t.Fatalf("应返回引用拒绝, got %v", err)
	}
}

// ========== 客户 ==========

// TestCustomerCRUD 测试客户的增删改查及状态切换
func TestCustomerCRUD(t *testing.T) {
	testDB(t)
	c := jxc.Customer{Code: "K001", Name: "张三", Phone: "13900000000", Level: 1, Status: 1}
	if err := svc.CreateCustomer(ctx, &c); err != nil || c.ID == 0 {
		t.Fatalf("create err=%v", err)
	}
	c.Level = 2
	if err := svc.UpdateCustomer(ctx, &c); err != nil {
		t.Fatalf("update err=%v", err)
	}
	_, total, err := svc.GetCustomerList(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("list total=%d err=%v", total, err)
	}
	all, err := svc.GetAllCustomers(ctx)
	if err != nil || len(all) != 1 || all[0].Level != 2 {
		t.Fatalf("getAll err=%v list=%+v", err, all)
	}
	if err := svc.SetCustomerStatus(ctx, c.ID, 0); err != nil {
		t.Fatalf("setStatus err=%v", err)
	}
	if err := svc.DeleteCustomer(ctx, c.ID); err != nil {
		t.Fatalf("delete err=%v", err)
	}
}

// TestDeleteCustomer_Referenced 测试删除客户时被销售订单引用拒绝
func TestDeleteCustomer_Referenced(t *testing.T) {
	testDB(t)
	c := jxc.Customer{Code: "K001", Name: "张三", Status: 1}
	global.GVA_DB.Create(&c)
	ensureRefTable(t, "CREATE TABLE sales_orders (customer_id int)")
	global.GVA_DB.Exec("INSERT INTO sales_orders (customer_id) VALUES (?)", c.ID)
	if err := svc.DeleteCustomer(ctx, c.ID); err == nil || err.Error() != "该客户已被销售订单引用，无法删除" {
		t.Fatalf("应返回引用拒绝, got %v", err)
	}
}

// ========== 仓库 ==========

// TestWarehouseCRUD 测试仓库的增删改查及状态切换
func TestWarehouseCRUD(t *testing.T) {
	testDB(t)
	w := jxc.Warehouse{Code: "W001", Name: "总仓", Manager: "李", Phone: "13700000000", Status: 1}
	if err := svc.CreateWarehouse(ctx, &w); err != nil || w.ID == 0 {
		t.Fatalf("create err=%v", err)
	}
	w.Name = "总仓(改)"
	if err := svc.UpdateWarehouse(ctx, &w); err != nil {
		t.Fatalf("update err=%v", err)
	}
	_, total, err := svc.GetWarehouseList(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("list total=%d err=%v", total, err)
	}
	all, err := svc.GetAllWarehouses(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("getAll err=%v", err)
	}
	if err := svc.SetWarehouseStatus(ctx, w.ID, 0); err != nil {
		t.Fatalf("setStatus err=%v", err)
	}
	if err := svc.DeleteWarehouse(ctx, w.ID); err != nil {
		t.Fatalf("delete err=%v", err)
	}
}

// TestDeleteWarehouse_Referenced 测试删除仓库时仍存在库存被拒绝
func TestDeleteWarehouse_Referenced(t *testing.T) {
	testDB(t)
	w := jxc.Warehouse{Code: "W001", Name: "总仓", Status: 1}
	global.GVA_DB.Create(&w)
	ensureRefTable(t, "CREATE TABLE stock (warehouse_id int, quantity int)")
	global.GVA_DB.Exec("INSERT INTO stock (warehouse_id, quantity) VALUES (?, 3)", w.ID)
	if err := svc.DeleteWarehouse(ctx, w.ID); err == nil || err.Error() != "该仓库仍存在库存，无法删除" {
		t.Fatalf("应返回引用拒绝, got %v", err)
	}
}

// ========== 数据库异常错误分支 ==========

// dropTable 删除指定表，用于触发数据库操作错误分支（no such table）
func dropTable(t *testing.T, table string) {
	t.Helper()
	if err := global.GVA_DB.Exec("DROP TABLE " + table).Error; err != nil {
		t.Fatalf("DROP TABLE %s 失败: %v", table, err)
	}
}

// TestCategoryErrors 测试商品分类各方法在数据库异常时的错误分支
func TestCategoryErrors(t *testing.T) {
	testDB(t)
	dropTable(t, "goods_category")
	if _, err := svc.GetCategoryTree(ctx); err == nil {
		t.Fatalf("GetCategoryTree 应报错")
	}
	if _, _, err := svc.GetCategoryList(ctx, request.PageInfo{Page: 1, PageSize: 10}); err == nil {
		t.Fatalf("GetCategoryList 应报错")
	}
	if err := svc.CreateCategory(ctx, &jxc.GoodsCategory{Code: "C001", Name: "x"}); err == nil {
		t.Fatalf("CreateCategory 应报错")
	}
	c := &jxc.GoodsCategory{}
	c.ID = 1
	if err := svc.UpdateCategory(ctx, c); err == nil {
		t.Fatalf("UpdateCategory 应报错")
	}
	if err := svc.DeleteCategory(ctx, 1); err == nil {
		t.Fatalf("DeleteCategory 应报错")
	}
	if err := svc.SetCategoryStatus(ctx, 1, 0); err == nil {
		t.Fatalf("SetCategoryStatus 应报错")
	}
}

// TestBrandErrors 测试品牌各方法在数据库异常时的错误分支
func TestBrandErrors(t *testing.T) {
	testDB(t)
	dropTable(t, "brand")
	if _, _, err := svc.GetBrandList(ctx, request.PageInfo{Page: 1, PageSize: 10}); err == nil {
		t.Fatalf("GetBrandList 应报错")
	}
	if _, err := svc.GetAllBrands(ctx); err == nil {
		t.Fatalf("GetAllBrands 应报错")
	}
	if err := svc.CreateBrand(ctx, &jxc.Brand{Code: "B001", Name: "x"}); err == nil {
		t.Fatalf("CreateBrand 应报错")
	}
	b := &jxc.Brand{}
	b.ID = 1
	if err := svc.UpdateBrand(ctx, b); err == nil {
		t.Fatalf("UpdateBrand 应报错")
	}
	if err := svc.DeleteBrand(ctx, 1); err == nil {
		t.Fatalf("DeleteBrand 应报错")
	}
	if err := svc.SetBrandStatus(ctx, 1, 0); err == nil {
		t.Fatalf("SetBrandStatus 应报错")
	}
}

// TestSupplierErrors 测试供应商各方法在数据库异常时的错误分支
func TestSupplierErrors(t *testing.T) {
	testDB(t)
	dropTable(t, "supplier")
	if _, _, err := svc.GetSupplierList(ctx, request.PageInfo{Page: 1, PageSize: 10}); err == nil {
		t.Fatalf("GetSupplierList 应报错")
	}
	if _, err := svc.GetAllSuppliers(ctx); err == nil {
		t.Fatalf("GetAllSuppliers 应报错")
	}
	if err := svc.CreateSupplier(ctx, &jxc.Supplier{Code: "S001", Name: "x"}); err == nil {
		t.Fatalf("CreateSupplier 应报错")
	}
	s := &jxc.Supplier{}
	s.ID = 1
	if err := svc.UpdateSupplier(ctx, s); err == nil {
		t.Fatalf("UpdateSupplier 应报错")
	}
	if err := svc.DeleteSupplier(ctx, 1); err == nil {
		t.Fatalf("DeleteSupplier 应报错")
	}
	if err := svc.SetSupplierStatus(ctx, 1, 0); err == nil {
		t.Fatalf("SetSupplierStatus 应报错")
	}
}

// TestCustomerErrors 测试客户各方法在数据库异常时的错误分支
func TestCustomerErrors(t *testing.T) {
	testDB(t)
	dropTable(t, "customer")
	if _, _, err := svc.GetCustomerList(ctx, request.PageInfo{Page: 1, PageSize: 10}); err == nil {
		t.Fatalf("GetCustomerList 应报错")
	}
	if _, err := svc.GetAllCustomers(ctx); err == nil {
		t.Fatalf("GetAllCustomers 应报错")
	}
	if err := svc.CreateCustomer(ctx, &jxc.Customer{Code: "K001", Name: "x"}); err == nil {
		t.Fatalf("CreateCustomer 应报错")
	}
	c := &jxc.Customer{}
	c.ID = 1
	if err := svc.UpdateCustomer(ctx, c); err == nil {
		t.Fatalf("UpdateCustomer 应报错")
	}
	if err := svc.DeleteCustomer(ctx, 1); err == nil {
		t.Fatalf("DeleteCustomer 应报错")
	}
	if err := svc.SetCustomerStatus(ctx, 1, 0); err == nil {
		t.Fatalf("SetCustomerStatus 应报错")
	}
}

// TestWarehouseErrors 测试仓库各方法在数据库异常时的错误分支
func TestWarehouseErrors(t *testing.T) {
	testDB(t)
	dropTable(t, "warehouse")
	if _, _, err := svc.GetWarehouseList(ctx, request.PageInfo{Page: 1, PageSize: 10}); err == nil {
		t.Fatalf("GetWarehouseList 应报错")
	}
	if _, err := svc.GetAllWarehouses(ctx); err == nil {
		t.Fatalf("GetAllWarehouses 应报错")
	}
	if err := svc.CreateWarehouse(ctx, &jxc.Warehouse{Code: "W001", Name: "x"}); err == nil {
		t.Fatalf("CreateWarehouse 应报错")
	}
	w := &jxc.Warehouse{}
	w.ID = 1
	if err := svc.UpdateWarehouse(ctx, w); err == nil {
		t.Fatalf("UpdateWarehouse 应报错")
	}
	if err := svc.DeleteWarehouse(ctx, 1); err == nil {
		t.Fatalf("DeleteWarehouse 应报错")
	}
	if err := svc.SetWarehouseStatus(ctx, 1, 0); err == nil {
		t.Fatalf("SetWarehouseStatus 应报错")
	}
}

// TestBasicAutoCode 测试基础资料各实体创建时编码自动生成（前缀+日期+序号）
func TestBasicAutoCode(t *testing.T) {
	testDB(t)
	prefix := time.Now().Format("20060102")
	// 分类
	cat := &jxc.GoodsCategory{Name: "男装", Status: 1}
	if err := svc.CreateCategory(ctx, cat); err != nil {
		t.Fatalf("create category err=%v", err)
	}
	if !strings.HasPrefix(cat.Code, "C"+prefix) {
		t.Fatalf("分类编码应为 C%s 前缀, got %s", prefix, cat.Code)
	}
	// 品牌
	b := &jxc.Brand{Name: "耐克", Status: 1}
	if err := svc.CreateBrand(ctx, b); err != nil {
		t.Fatalf("create brand err=%v", err)
	}
	if !strings.HasPrefix(b.Code, "B"+prefix) {
		t.Fatalf("品牌编码应为 B%s 前缀, got %s", prefix, b.Code)
	}
	// 供应商
	s := &jxc.Supplier{Name: "广州布行", Status: 1}
	if err := svc.CreateSupplier(ctx, s); err != nil {
		t.Fatalf("create supplier err=%v", err)
	}
	if !strings.HasPrefix(s.Code, "S"+prefix) {
		t.Fatalf("供应商编码应为 S%s 前缀, got %s", prefix, s.Code)
	}
	// 客户
	c := &jxc.Customer{Name: "张三", Status: 1}
	if err := svc.CreateCustomer(ctx, c); err != nil {
		t.Fatalf("create customer err=%v", err)
	}
	if !strings.HasPrefix(c.Code, "K"+prefix) {
		t.Fatalf("客户编码应为 K%s 前缀, got %s", prefix, c.Code)
	}
	// 仓库
	w := &jxc.Warehouse{Name: "总仓", Status: 1}
	if err := svc.CreateWarehouse(ctx, w); err != nil {
		t.Fatalf("create warehouse err=%v", err)
	}
	if !strings.HasPrefix(w.Code, "W"+prefix) {
		t.Fatalf("仓库编码应为 W%s 前缀, got %s", prefix, w.Code)
	}
}

// TestBasicDeleteForever 测试基础资料各实体彻底删除（物理删除后无记录）
func TestBasicDeleteForever(t *testing.T) {
	testDB(t)
	// 分类
	cat := &jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	if err := svc.CreateCategory(ctx, cat); err != nil {
		t.Fatalf("create category err=%v", err)
	}
	if err := svc.DeleteCategoryForever(ctx, cat.ID); err != nil {
		t.Fatalf("delete category forever err=%v", err)
	}
	var cnt int64
	global.GVA_DB.Unscoped().Model(&jxc.GoodsCategory{}).Where("id = ?", cat.ID).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("物理删除后应无记录, cnt=%d", cnt)
	}
	// 品牌
	b := &jxc.Brand{Code: "B001", Name: "耐克", Status: 1}
	svc.CreateBrand(ctx, b)
	if err := svc.DeleteBrandForever(ctx, b.ID); err != nil {
		t.Fatalf("delete brand forever err=%v", err)
	}
	global.GVA_DB.Unscoped().Model(&jxc.Brand{}).Where("id = ?", b.ID).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("品牌物理删除失败, cnt=%d", cnt)
	}
	// 供应商
	s := &jxc.Supplier{Code: "S001", Name: "布行", Status: 1}
	svc.CreateSupplier(ctx, s)
	if err := svc.DeleteSupplierForever(ctx, s.ID); err != nil {
		t.Fatalf("delete supplier forever err=%v", err)
	}
	// 客户
	c := &jxc.Customer{Code: "K001", Name: "张三", Status: 1}
	svc.CreateCustomer(ctx, c)
	if err := svc.DeleteCustomerForever(ctx, c.ID); err != nil {
		t.Fatalf("delete customer forever err=%v", err)
	}
	// 仓库
	w := &jxc.Warehouse{Code: "W001", Name: "总仓", Status: 1}
	svc.CreateWarehouse(ctx, w)
	if err := svc.DeleteWarehouseForever(ctx, w.ID); err != nil {
		t.Fatalf("delete warehouse forever err=%v", err)
	}
}

// TestDeleteCategoryForever_Referenced 测试分类被引用时彻底删除被拒绝
func TestDeleteCategoryForever_Referenced(t *testing.T) {
	testDB(t)
	cat := &jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	svc.CreateCategory(ctx, cat)
	ensureRefTable(t, "CREATE TABLE goods (category_id int, deleted_at datetime)")
	global.GVA_DB.Exec("INSERT INTO goods (category_id) VALUES (?)", cat.ID)
	err := svc.DeleteCategoryForever(ctx, cat.ID)
	if err == nil || err.Error() != "该分类已被商品引用，请先删除相关商品" {
		t.Fatalf("应返回引用拒绝, got %v", err)
	}
}
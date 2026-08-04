package jxc

import (
	"context"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

var purchaseSvc = new(PurchaseService)

// purchaseTestDB 采购模块测试库（基础+商品+SKU+采购单+库存）
func purchaseTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
	)
}

// newPurchaseFixture 构造前置数据（供应商/仓库/商品/SKU），返回 skuID
func newPurchaseFixture(t *testing.T) uint {
	t.Helper()
	db := global.GVA_DB
	supplier := jxc.Supplier{Code: "GYS001", Name: "测试供应商"}
	if err := db.Create(&supplier).Error; err != nil {
		t.Fatalf("创建供应商失败: %v", err)
	}
	warehouse := jxc.Warehouse{Code: "CK001", Name: "总仓"}
	if err := db.Create(&warehouse).Error; err != nil {
		t.Fatalf("创建仓库失败: %v", err)
	}
	goods := jxc.Goods{Code: "SP001", Name: "测试商品"}
	if err := db.Create(&goods).Error; err != nil {
		t.Fatalf("创建商品失败: %v", err)
	}
	sku := jxc.GoodsSku{GoodsID: goods.ID, SkuCode: "SP001-M-红", Color: "红", Size: "M", CostPrice: 10}
	if err := db.Create(&sku).Error; err != nil {
		t.Fatalf("创建SKU失败: %v", err)
	}
	return sku.ID
}

// newPurchaseOrder 快捷创建采购单（1 条明细 qty=10 price=10），返回单ID
func newPurchaseOrder(t *testing.T, skuID uint) uint {
	t.Helper()
	order := &jxc.PurchaseOrder{
		SupplierID: 1, WarehouseID: 1,
		Items: []jxc.PurchaseItem{{SkuID: skuID, Qty: 10, Price: 10}},
	}
	if err := purchaseSvc.CreatePurchase(context.Background(), order); err != nil {
		t.Fatalf("创建采购单失败: %v", err)
	}
	return order.ID
}

// TestCreatePurchase 测试创建采购单（单号/金额汇总/明细快照/级联写入）
func TestCreatePurchase(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	order := &jxc.PurchaseOrder{
		SupplierID: 1, WarehouseID: 1,
		Items: []jxc.PurchaseItem{
			{SkuID: skuID, Qty: 10, Price: 10},
			{SkuID: skuID, Qty: 5, Price: 8},
		},
	}
	if err := purchaseSvc.CreatePurchase(context.Background(), order); err != nil {
		t.Fatalf("创建采购单失败: %v", err)
	}
	if order.Status != jxc.PurchaseStatusPending {
		t.Errorf("初始状态应为待审核, got %d", order.Status)
	}
	if !strings.HasPrefix(order.OrderNo, "PO-") || !strings.HasSuffix(order.OrderNo, "-001") {
		t.Errorf("单号格式错误: %s", order.OrderNo)
	}
	if order.TotalAmount != 140 {
		t.Errorf("总金额应为 140, got %v", order.TotalAmount)
	}
	// 明细级联写入 + 快照
	var items []jxc.PurchaseItem
	global.GVA_DB.Where("purchase_id = ?", order.ID).Find(&items)
	if len(items) != 2 {
		t.Fatalf("明细应写入 2 条, got %d", len(items))
	}
	if items[0].SkuCode != "SP001-M-红" || items[0].GoodsName != "测试商品" || items[0].Color != "红" || items[0].Size != "M" {
		t.Errorf("SKU 快照未填充: %+v", items[0])
	}
}

// TestCreatePurchase_Invalid 测试创建采购单参数校验
func TestCreatePurchase_Invalid(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	ctx := context.Background()
	// 无明细
	if err := purchaseSvc.CreatePurchase(ctx, &jxc.PurchaseOrder{SupplierID: 1, WarehouseID: 1}); err == nil {
		t.Error("无明细应报错")
	}
	// 数量 <= 0
	if err := purchaseSvc.CreatePurchase(ctx, &jxc.PurchaseOrder{SupplierID: 1, WarehouseID: 1, Items: []jxc.PurchaseItem{{SkuID: skuID, Qty: 0}}}); err == nil {
		t.Error("数量为 0 应报错")
	}
	// 单价为负
	if err := purchaseSvc.CreatePurchase(ctx, &jxc.PurchaseOrder{SupplierID: 1, WarehouseID: 1, Items: []jxc.PurchaseItem{{SkuID: skuID, Qty: 1, Price: -1}}}); err == nil {
		t.Error("负单价应报错")
	}
	// SKU 不存在
	if err := purchaseSvc.CreatePurchase(ctx, &jxc.PurchaseOrder{SupplierID: 1, WarehouseID: 1, Items: []jxc.PurchaseItem{{SkuID: 999}}}); err == nil {
		t.Error("SKU 不存在应报错")
	}
}

// TestGetPurchasePage_Detail 测试采购单分页/详情
func TestGetPurchasePage_Detail(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	oid := newPurchaseOrder(t, skuID)

	// 分页
	list, total, err := purchaseSvc.GetPurchasePage(context.Background(), testPageInfo())
	if err != nil {
		t.Fatalf("分页查询失败: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != oid {
		t.Errorf("分页结果不符: total=%d list=%d", total, len(list))
	}
	if list[0].Supplier == nil || list[0].Warehouse == nil {
		t.Error("列表未预加载供应商/仓库")
	}
	// 关键词
	_, total, _ = purchaseSvc.GetPurchasePage(context.Background(), testPageInfoKeyword(oid))
	if total != 1 {
		t.Errorf("按单号搜索应命中 1 条, got %d", total)
	}
	// 详情含明细
	detail, err := purchaseSvc.GetPurchaseDetail(context.Background(), oid)
	if err != nil {
		t.Fatalf("详情查询失败: %v", err)
	}
	if len(detail.Items) != 1 || detail.Items[0].Qty != 10 {
		t.Errorf("详情明细不符: %+v", detail.Items)
	}
}

// TestUpdatePurchase 测试更新采购单（仅待审核, 明细重建）
func TestUpdatePurchase(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	oid := newPurchaseOrder(t, skuID)

	upd := &jxc.PurchaseOrder{SupplierID: 1, WarehouseID: 1,
		Items: []jxc.PurchaseItem{{SkuID: skuID, Qty: 20, Price: 5}}}
	upd.ID = oid
	if err := purchaseSvc.UpdatePurchase(context.Background(), upd); err != nil {
		t.Fatalf("更新采购单失败: %v", err)
	}
	if upd.TotalAmount != 100 {
		t.Errorf("更新后金额应为 100, got %v", upd.TotalAmount)
	}
	var items []jxc.PurchaseItem
	global.GVA_DB.Where("purchase_id = ?", oid).Find(&items)
	if len(items) != 1 || items[0].Qty != 20 {
		t.Errorf("明细未重建: %+v", items)
	}
	// 审核后不可改
	if err := purchaseSvc.AuditPurchase(context.Background(), oid); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := purchaseSvc.UpdatePurchase(context.Background(), upd); err == nil {
		t.Error("已审核采购单不应可修改")
	}
}

// TestDeletePurchase 测试删除采购单（仅待审核）
func TestDeletePurchase(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	oid := newPurchaseOrder(t, skuID)
	if err := purchaseSvc.AuditPurchase(context.Background(), oid); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := purchaseSvc.DeletePurchase(context.Background(), oid); err == nil {
		t.Error("已审核采购单不应可删除")
	}
	// 待审核可删
	oid2 := newPurchaseOrder(t, skuID)
	if err := purchaseSvc.DeletePurchase(context.Background(), oid2); err != nil {
		t.Fatalf("待审核采购单删除失败: %v", err)
	}
	var cnt int64
	global.GVA_DB.Model(&jxc.PurchaseItem{}).Where("purchase_id = ?", oid2).Count(&cnt)
	if cnt != 0 {
		t.Errorf("级联明细未删除: %d", cnt)
	}
}

// TestAuditCancelPurchase 测试审核/取消状态流转
func TestAuditCancelPurchase(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	ctx := context.Background()
	oid := newPurchaseOrder(t, skuID)

	// 审核：1 → 2
	if err := purchaseSvc.AuditPurchase(ctx, oid); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	// 重复审核拒绝
	if err := purchaseSvc.AuditPurchase(ctx, oid); err == nil {
		t.Error("重复审核应拒绝")
	}
	// 取消：2 → 4
	if err := purchaseSvc.CancelPurchase(ctx, oid); err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	// 已取消不可再取消
	if err := purchaseSvc.CancelPurchase(ctx, oid); err == nil {
		t.Error("已取消单不应可再取消")
	}
	// 已取消不可入库
	if err := purchaseSvc.StockIn(ctx, oid, "tester"); err == nil {
		t.Error("已取消单不应可入库")
	}
}

// TestStockIn 测试采购入库（库存增加 + 流水写入 + 状态变更）
func TestStockIn(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	ctx := context.Background()
	oid := newPurchaseOrder(t, skuID)
	if err := purchaseSvc.AuditPurchase(ctx, oid); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := purchaseSvc.StockIn(ctx, oid, "tester"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	// 状态 → 已入库
	var order jxc.PurchaseOrder
	global.GVA_DB.First(&order, oid)
	if order.Status != jxc.PurchaseStatusStockIn {
		t.Errorf("状态应为已入库, got %d", order.Status)
	}
	// 库存记录
	var stock jxc.Stock
	if err := global.GVA_DB.Where("warehouse_id = 1 AND sku_id = ?", skuID).First(&stock).Error; err != nil {
		t.Fatalf("库存记录未创建: %v", err)
	}
	if stock.Quantity != 10 {
		t.Errorf("库存应为 10, got %d", stock.Quantity)
	}
	// 流水
	var logs []jxc.StockLog
	global.GVA_DB.Where("sku_id = ?", skuID).Find(&logs)
	if len(logs) != 1 || logs[0].BusinessType != "purchase_in" || logs[0].BeforeQty != 0 || logs[0].AfterQty != 10 {
		t.Errorf("流水不符: %+v", logs)
	}
	// 成本价回写（最新进价法）
	var sku jxc.GoodsSku
	global.GVA_DB.First(&sku, skuID)
	if sku.CostPrice != 10 {
		t.Errorf("入库后 SKU 成本价应回写为采购单价 10, got %v", sku.CostPrice)
	}
	// 重复入库拒绝（状态已变）
	if err := purchaseSvc.StockIn(ctx, oid, "tester"); err == nil {
		t.Error("重复入库应拒绝")
	}
}

// TestStockIn_Accumulate 测试多次入库库存累加（首次创建 + 再次累加）
func TestStockIn_Accumulate(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	ctx := context.Background()
	// 第一单 10 件
	oid1 := newPurchaseOrder(t, skuID)
	if err := purchaseSvc.AuditPurchase(ctx, oid1); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := purchaseSvc.StockIn(ctx, oid1, "tester"); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	// 第二单 5 件
	oid2 := newPurchaseOrder(t, skuID)
	if err := purchaseSvc.AuditPurchase(ctx, oid2); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := purchaseSvc.StockIn(ctx, oid2, "tester"); err != nil {
		t.Fatalf("二次入库失败: %v", err)
	}
	var stock jxc.Stock
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = ?", skuID).First(&stock)
	if stock.Quantity != 20 {
		t.Errorf("库存应累加为 20, got %d", stock.Quantity)
	}
	var logs []jxc.StockLog
	global.GVA_DB.Where("sku_id = ?", skuID).Order("id").Find(&logs)
	if len(logs) != 2 || logs[1].BeforeQty != 10 || logs[1].AfterQty != 20 {
		t.Errorf("二次流水 before/after 不符: %+v", logs)
	}
}

// testPageInfo 采购单分页参数
func testPageInfo() request.PageInfo {
	return request.PageInfo{Page: 1, PageSize: 10}
}

// testPageInfoKeyword 采购单分页参数（按单号搜索）
func testPageInfoKeyword(oid uint) request.PageInfo {
	info := request.PageInfo{Page: 1, PageSize: 10}
	var order jxc.PurchaseOrder
	global.GVA_DB.First(&order, oid)
	info.Keyword = order.OrderNo
	return info
}

// TestPurchase_ErrorBranches 测试 service 异常/错误分支
func TestPurchase_ErrorBranches(t *testing.T) {
	purchaseTestDB(t)
	skuID := newPurchaseFixture(t)
	ctx := context.Background()

	// 空库分页（total==0 提前返回）
	if _, total, err := purchaseSvc.GetPurchasePage(ctx, request.PageInfo{Page: 1, PageSize: 10}); err != nil || total != 0 {
		t.Errorf("空库分页应返回 0, got total=%d err=%v", total, err)
	}
	// 删除/更新不存在的单
	if err := purchaseSvc.DeletePurchase(ctx, 999); err == nil {
		t.Error("删除不存在单应报错")
	}
	upd := &jxc.PurchaseOrder{SupplierID: 1, Items: []jxc.PurchaseItem{{SkuID: skuID, Qty: 1}}}
	upd.ID = 999
	if err := purchaseSvc.UpdatePurchase(ctx, upd); err == nil {
		t.Error("更新不存在单应报错")
	}
	// 入库时流水表缺失（事务内 Create 失败回滚）
	oid := newPurchaseOrder(t, skuID)
	purchaseSvc.AuditPurchase(ctx, oid)
	global.GVA_DB.Exec("DROP TABLE stock_log")
	if err := purchaseSvc.StockIn(ctx, oid, "t"); err == nil {
		t.Error("流水表缺失入库应报错")
	}
	// 单号序号解析失败（坏单号插入后再创建）
	global.GVA_DB.Create(&jxc.PurchaseOrder{OrderNo: "PO-20260804-XXX", SupplierID: 1, WarehouseID: 1, Status: 1})
	if err := purchaseSvc.CreatePurchase(ctx, &jxc.PurchaseOrder{SupplierID: 1, WarehouseID: 1, Items: []jxc.PurchaseItem{{SkuID: skuID, Qty: 1}}}); err == nil {
		t.Error("单号解析失败应报错")
	}
	// 主表缺失时审核/取消/入库报错
	global.GVA_DB.Exec("DROP TABLE purchase_order")
	if err := purchaseSvc.AuditPurchase(ctx, 1); err == nil {
		t.Error("主表缺失审核应报错")
	}
	if err := purchaseSvc.CancelPurchase(ctx, 1); err == nil {
		t.Error("主表缺失取消应报错")
	}
	if err := purchaseSvc.StockIn(ctx, 1, "t"); err == nil {
		t.Error("主表缺失入库应报错")
	}
}
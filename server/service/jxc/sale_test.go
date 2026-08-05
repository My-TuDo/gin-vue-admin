package jxc

import (
	"context"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

var saleSvc = new(SaleService)

// saleTestDB 销售模块测试库（基础+商品+SKU+采购+库存+销售）
func saleTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
		&jxc.SaleOrder{}, &jxc.SaleItem{},
	)
}

// newSaleFixture 前置：客户/仓库/商品/SKU + 库存 10 件，返回 skuID
func newSaleFixture(t *testing.T) uint {
	t.Helper()
	db := global.GVA_DB
	db.Create(&jxc.Customer{Code: "KH001", Name: "测试客户", Status: 1})
	db.Create(&jxc.Warehouse{Code: "CK001", Name: "总仓", Status: 1})
	g := jxc.Goods{Code: "SP001", Name: "测试商品", Status: 1}
	db.Create(&g)
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-M-红", Color: "红", Size: "M", CostPrice: 8, SalePrice: 15, Status: 1}
	db.Create(&sku)
	db.Create(&jxc.Stock{WarehouseID: 1, SkuID: sku.ID, Quantity: 10})
	return sku.ID
}

// getStockQty 读取某 SKU 库存（quantity/lock_quantity）
func getStockQty(t *testing.T, skuID uint) (qty, lock int) {
	t.Helper()
	var stock jxc.Stock
	if err := global.GVA_DB.Where("warehouse_id = 1 AND sku_id = ?", skuID).First(&stock).Error; err != nil {
		t.Fatalf("读取库存失败: %v", err)
	}
	return stock.Quantity, stock.LockQuantity
}

// TestCreateSaleOrder_Lock 测试建单锁定库存（单价强制取 SKU 销售价）
func TestCreateSaleOrder_Lock(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	order := &jxc.SaleOrder{
		WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 3, Price: 99}}, // 前端传 99，应被强制为 SKU 销售价 15
	}
	if err := saleSvc.CreateSaleOrder(context.Background(), order); err != nil {
		t.Fatalf("创建销售单失败: %v", err)
	}
	if order.Status != jxc.SaleStatusPending {
		t.Errorf("初始状态应为待出库, got %d", order.Status)
	}
	if order.TotalAmount != 45 {
		t.Errorf("总金额应为 45（3*15, 强制销售价）, got %v", order.TotalAmount)
	}
	qty, lock := getStockQty(t, skuID)
	if qty != 10 || lock != 3 {
		t.Errorf("锁定后应为 qty=10 lock=3, got qty=%d lock=%d", qty, lock)
	}
	var items []jxc.SaleItem
	global.GVA_DB.Where("sale_id = ?", order.ID).Find(&items)
	if len(items) != 1 || items[0].SkuCode != "SP001-M-红" {
		t.Errorf("明细快照未写入: %+v", items)
	}
	if items[0].Price != 15 {
		t.Errorf("明细单价应强制为 SKU 销售价 15, got %v", items[0].Price)
	}
}

// TestCreateSaleOrder_Insufficient 测试库存不足拒绝且不锁定
func TestCreateSaleOrder_Insufficient(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	order := &jxc.SaleOrder{
		WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 11, Price: 15}},
	}
	if err := saleSvc.CreateSaleOrder(context.Background(), order); err == nil {
		t.Fatal("库存不足应报错")
	}
	_, lock := getStockQty(t, skuID)
	if lock != 0 {
		t.Errorf("失败创建不应锁定, lock=%d", lock)
	}
	// 无库存记录
	saleTestDB2 := func() {}
	_ = saleTestDB2
	var cnt int64
	global.GVA_DB.Model(&jxc.SaleOrder{}).Count(&cnt)
	if cnt != 0 {
		t.Error("失败创建不应产生销售单")
	}
}

// TestCreateSaleOrder_NoStock 测试无库存记录拒绝
func TestCreateSaleOrder_NoStock(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	global.GVA_DB.Exec("DELETE FROM stock")
	order := &jxc.SaleOrder{
		WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}},
	}
	if err := saleSvc.CreateSaleOrder(context.Background(), order); err == nil {
		t.Error("无库存记录应报错")
	}
}

// TestConfirmOut 测试出库（扣减+流水+状态）
func TestConfirmOut(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	order := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 3, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, order); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if err := saleSvc.ConfirmOut(ctx, order.ID, "tester"); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	qty, lock := getStockQty(t, skuID)
	if qty != 7 || lock != 0 {
		t.Errorf("出库后应为 qty=7 lock=0, got qty=%d lock=%d", qty, lock)
	}
	var logs []jxc.StockLog
	global.GVA_DB.Where("sku_id = ?", skuID).Find(&logs)
	if len(logs) != 1 || logs[0].BusinessType != "sale_out" || logs[0].ChangeQty != -3 || logs[0].AfterQty != 7 {
		t.Errorf("出库流水不符: %+v", logs)
	}
	var order2 jxc.SaleOrder
	global.GVA_DB.First(&order2, order.ID)
	if order2.Status != jxc.SaleStatusShipped {
		t.Errorf("出库后状态应为已出库, got %d", order2.Status)
	}
	// 已出库不可再出库
	if err := saleSvc.ConfirmOut(ctx, order.ID, "tester"); err == nil {
		t.Error("重复出库应拒绝")
	}
}

// TestCancelSaleOrder 测试取消释放锁定
func TestCancelSaleOrder(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	order := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 3, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, order); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if err := saleSvc.CancelSaleOrder(ctx, order.ID); err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	_, lock := getStockQty(t, skuID)
	if lock != 0 {
		t.Errorf("取消后锁定应释放, lock=%d", lock)
	}
	var order2 jxc.SaleOrder
	global.GVA_DB.First(&order2, order.ID)
	if order2.Status != jxc.SaleStatusCanceled {
		t.Errorf("状态应为已取消, got %d", order2.Status)
	}
}

// TestDeleteUpdateSaleOrder 测试删除/更新（锁定调整）
func TestDeleteUpdateSaleOrder(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	order := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 3, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, order); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	// 更新数量 3 → 5（锁 3 → 5）
	upd := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 5, Price: 15}}}
	upd.ID = order.ID
	if err := saleSvc.UpdateSaleOrder(ctx, upd); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	_, lock := getStockQty(t, skuID)
	if lock != 5 {
		t.Errorf("更新后锁定应为 5, got %d", lock)
	}
	// 删除释放锁
	if err := saleSvc.DeleteSaleOrder(ctx, order.ID); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	_, lock = getStockQty(t, skuID)
	if lock != 0 {
		t.Errorf("删除后锁定应释放, got %d", lock)
	}
}

// TestConfirmReturn 测试退货入库
func TestConfirmReturn(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	// 原销售单出库
	sale := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 3, Price: 15}}}
	saleSvc.CreateSaleOrder(ctx, sale)
	saleSvc.ConfirmOut(ctx, sale.ID, "tester")
	// 退货单（关联原单）
	ret := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeReturn, OriginalOrderID: &sale.ID,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, ret); err != nil {
		t.Fatalf("创建退货单失败: %v", err)
	}
	if err := saleSvc.ConfirmReturn(ctx, ret.ID, "tester"); err != nil {
		t.Fatalf("退货入库失败: %v", err)
	}
	qty, _ := getStockQty(t, skuID)
	if qty != 8 {
		t.Errorf("退货后库存应为 8, got %d", qty)
	}
	var logs []jxc.StockLog
	global.GVA_DB.Where("sku_id = ? AND business_type = ?", skuID, "sale_return").Find(&logs)
	if len(logs) != 1 || logs[0].ChangeQty != 1 {
		t.Errorf("退货流水不符: %+v", logs)
	}
	// 非退货单不可退货
	if err := saleSvc.ConfirmReturn(ctx, sale.ID, "tester"); err == nil {
		t.Error("正常销售单执行退货应拒绝")
	}
}

// TestConfirmExchange 测试换货（出库扣减/入库增加）
func TestConfirmExchange(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	sale := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 2, Price: 15}}}
	saleSvc.CreateSaleOrder(ctx, sale)
	saleSvc.ConfirmOut(ctx, sale.ID, "tester")

	// 换货出库（type=3）
	exOut := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeExOut, OriginalOrderID: &sale.ID,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, exOut); err != nil {
		t.Fatalf("创建换货出库单失败: %v", err)
	}
	if err := saleSvc.ConfirmExchange(ctx, exOut.ID, "tester"); err != nil {
		t.Fatalf("换货出库失败: %v", err)
	}
	qty, _ := getStockQty(t, skuID)
	if qty != 7 {
		t.Errorf("换出后库存应为 7, got %d", qty)
	}

	// 换货入库（type=4）
	exIn := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeExIn, OriginalOrderID: &sale.ID,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 2, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, exIn); err != nil {
		t.Fatalf("创建换货入库单失败: %v", err)
	}
	if err := saleSvc.ConfirmExchange(ctx, exIn.ID, "tester"); err != nil {
		t.Fatalf("换货入库失败: %v", err)
	}
	qty, _ = getStockQty(t, skuID)
	if qty != 9 {
		t.Errorf("换入后库存应为 9, got %d", qty)
	}
	// 非换货单不可执行换货
	if err := saleSvc.ConfirmExchange(ctx, sale.ID, "tester"); err == nil {
		t.Error("正常销售单执行换货应拒绝")
	}
}

// TestSale_StockErrors 测试库存异常场景（出库不足/解锁失败/退货表缺失）
func TestSale_StockErrors(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	// 出库时库存不足（建单锁定后被手动改小）
	o1 := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 2, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, o1); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	global.GVA_DB.Exec("UPDATE stock SET quantity = 1 WHERE sku_id = ?", skuID)
	if err := saleSvc.ConfirmOut(ctx, o1.ID, "t"); err == nil {
		t.Error("库存不足出库应报错")
	}
	// 删除时库存记录缺失（解锁失败）
	global.GVA_DB.Exec("DELETE FROM stock")
	global.GVA_DB.Create(&jxc.Stock{WarehouseID: 1, SkuID: skuID, Quantity: 10})
	o2 := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, o2); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	global.GVA_DB.Exec("DELETE FROM stock")
	if err := saleSvc.DeleteSaleOrder(ctx, o2.ID); err == nil {
		t.Error("库存记录缺失删除应报错")
	}
	// 退货时库存表缺失
	global.GVA_DB.Create(&jxc.Stock{WarehouseID: 1, SkuID: skuID, Quantity: 10})
	ret := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeReturn,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, ret); err != nil {
		t.Fatalf("创建退货单失败: %v", err)
	}
	global.GVA_DB.Exec("DROP TABLE stock")
	if err := saleSvc.ConfirmReturn(ctx, ret.ID, "t"); err == nil {
		t.Error("库存表缺失退货应报错")
	}
	// 更新后重新锁定失败（可售库存不足）
	saleTestDB(t)
	skuID = newSaleFixture(t)
	o3 := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, o3); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	global.GVA_DB.Exec("UPDATE stock SET quantity = 0 WHERE sku_id = ?", skuID)
	upd3 := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 5, Price: 15}}}
	upd3.ID = o3.ID
	if err := saleSvc.UpdateSaleOrder(ctx, upd3); err == nil {
		t.Error("重新锁定库存不足应报错")
	}
}

// TestGetSalePage_Empty 测试空库分页提前返回
func TestGetSalePage_Empty(t *testing.T) {
	saleTestDB(t)
	_, total, err := saleSvc.GetSalePage(context.Background(), request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 0 {
		t.Errorf("空库应返回 0, got total=%d err=%v", total, err)
	}
}

// TestGetSalePage_Detail 测试分页/详情
func TestGetSalePage_Detail(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	order := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(context.Background(), order); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	list, total, err := saleSvc.GetSalePage(context.Background(), request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("分页不符: total=%d list=%d err=%v", total, len(list), err)
	}
	// 按单号搜索
	if _, total, err := saleSvc.GetSalePage(context.Background(), request.PageInfo{Page: 1, PageSize: 10, Keyword: order.OrderNo}); err != nil || total != 1 {
		t.Fatalf("按单号搜索不符: total=%d err=%v", total, err)
	}
	detail, err := saleSvc.GetSaleDetail(context.Background(), order.ID)
	if err != nil || len(detail.Items) != 1 {
		t.Fatalf("详情不符: %+v err=%v", detail.Items, err)
	}
}

// TestSale_ErrorBranches 测试销售 service 异常/错误分支
func TestSale_ErrorBranches(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()

	// 无明细 / 数量0 / 负单价 / SKU不存在
	if err := saleSvc.CreateSaleOrder(ctx, &jxc.SaleOrder{WarehouseID: 1}); err == nil {
		t.Error("无明细应报错")
	}
	if err := saleSvc.CreateSaleOrder(ctx, &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 0}}}); err == nil {
		t.Error("数量0应报错")
	}
	if err := saleSvc.CreateSaleOrder(ctx, &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: 999, Qty: 1}}}); err == nil {
		t.Error("SKU不存在应报错")
	}
	// 更新/删除/取消不存在的单
	upd := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1}}}
	upd.ID = 999
	if err := saleSvc.UpdateSaleOrder(ctx, upd); err == nil {
		t.Error("更新不存在单应报错")
	}
	if err := saleSvc.DeleteSaleOrder(ctx, 999); err == nil {
		t.Error("删除不存在单应报错")
	}
	if err := saleSvc.CancelSaleOrder(ctx, 999); err == nil {
		t.Error("取消不存在单应报错")
	}
	// 正常建单 → 出库 → 已出库不可改/删/取消
	order := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, order); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if err := saleSvc.ConfirmOut(ctx, order.ID, "t"); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	if err := saleSvc.UpdateSaleOrder(ctx, upd); err == nil {
		t.Error("已出库单不应可更新")
	}
	if err := saleSvc.DeleteSaleOrder(ctx, order.ID); err == nil {
		t.Error("已出库单不应可删除")
	}
	if err := saleSvc.CancelSaleOrder(ctx, order.ID); err == nil {
		t.Error("已出库单不应可取消")
	}
	// 出库时库存不足（库存已扣减后再建单锁不住）
	order2 := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 10, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, order2); err == nil {
		t.Error("可售库存不足应报错")
	}
	// 已出库单不可更新（正确 ID）
	order3 := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, order3); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if err := saleSvc.ConfirmOut(ctx, order3.ID, "t"); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	upd3 := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	upd3.ID = order3.ID
	if err := saleSvc.UpdateSaleOrder(ctx, upd3); err == nil {
		t.Error("已出库单不应可更新")
	}
	// 库存记录缺失：出库/取消报错
	order4 := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, order4); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	global.GVA_DB.Exec("DELETE FROM stock")
	if err := saleSvc.ConfirmOut(ctx, order4.ID, "t"); err == nil {
		t.Error("库存记录缺失出库应报错")
	}
	if err := saleSvc.CancelSaleOrder(ctx, order4.ID); err == nil {
		t.Error("库存记录缺失取消应报错")
	}
	// 退货：非退货单 / 非待出库退货
	ret := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeReturn, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, ret); err != nil {
		t.Fatalf("创建退货单失败: %v", err)
	}
	if err := saleSvc.ConfirmReturn(ctx, order.ID, "t"); err == nil {
		t.Error("正常销售单执行退货应拒绝")
	}
	// 换货：出库单库存不足（清空库存记录）
	global.GVA_DB.Exec("DELETE FROM stock")
	exOut := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeExOut, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, exOut); err != nil {
		t.Fatalf("创建换货单失败: %v", err)
	}
	if err := saleSvc.ConfirmExchange(ctx, exOut.ID, "t"); err == nil {
		t.Error("换出库存不足应报错")
	}
	// 更新无明细
	upd2 := &jxc.SaleOrder{WarehouseID: 1}
	upd2.ID = 1
	if err := saleSvc.UpdateSaleOrder(ctx, upd2); err == nil {
		t.Error("更新无明细应报错")
	}
	// 退货/换货重复确认（非待出库拒绝）
	global.GVA_DB.Exec("DELETE FROM stock")
	global.GVA_DB.Create(&jxc.Stock{WarehouseID: 1, SkuID: skuID, Quantity: 5})
	okRet := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeReturn, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, okRet); err != nil {
		t.Fatalf("创建退货单失败: %v", err)
	}
	if err := saleSvc.ConfirmReturn(ctx, okRet.ID, "t"); err != nil {
		t.Fatalf("退货入库失败: %v", err)
	}
	if err := saleSvc.ConfirmReturn(ctx, okRet.ID, "t"); err == nil {
		t.Error("重复退货应拒绝")
	}
	okEx := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeExIn, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, okEx); err != nil {
		t.Fatalf("创建换货单失败: %v", err)
	}
	if err := saleSvc.ConfirmExchange(ctx, okEx.ID, "t"); err != nil {
		t.Fatalf("换货确认失败: %v", err)
	}
	if err := saleSvc.ConfirmExchange(ctx, okEx.ID, "t"); err == nil {
		t.Error("重复换货应拒绝")
	}
	// 单号序号解析失败（坏单号插入后再创建，须在 DROP 之前）
	badDate := time.Now().Format("20060102")
	global.GVA_DB.Create(&jxc.SaleOrder{OrderNo: "SO-" + badDate + "-XXX", WarehouseID: 1, Status: 1})
	if err := saleSvc.CreateSaleOrder(ctx, &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 1}}}); err == nil {
		t.Error("单号解析失败应报错")
	}
	// 主表缺失时报错
	global.GVA_DB.Exec("DROP TABLE sale_order")
	if err := saleSvc.CreateSaleOrder(ctx, &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1}}}); err == nil {
		t.Error("主表缺失创建应报错")
	}
	if err := saleSvc.ConfirmOut(ctx, 1, "t"); err == nil {
		t.Error("主表缺失出库应报错")
	}
	if err := saleSvc.ConfirmReturn(ctx, 1, "t"); err == nil {
		t.Error("主表缺失退货应报错")
	}
	if err := saleSvc.ConfirmExchange(ctx, 1, "t"); err == nil {
		t.Error("主表缺失换货应报错")
	}
}
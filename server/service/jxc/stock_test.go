package jxc

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

var stockSvc = new(StockService)

// stockTestDB 库存模块测试库
func stockTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
		&jxc.SaleOrder{}, &jxc.SaleItem{},
	)
}

// newStockFixture 前置：仓库 + 商品/SKU + 库存 10 件（预警 15），返回 skuID
func newStockFixture(t *testing.T) uint {
	t.Helper()
	db := global.GVA_DB
	db.Create(&jxc.Warehouse{Code: "CK001", Name: "总仓", Status: 1})
	g := jxc.Goods{Code: "SP001", Name: "测试商品", Status: 1}
	db.Create(&g)
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-M-红", Color: "红", Size: "M", CostPrice: 8, SalePrice: 15, Status: 1}
	db.Create(&sku)
	db.Create(&jxc.Stock{WarehouseID: 1, SkuID: sku.ID, Quantity: 10, WarningQuantity: 15})
	return sku.ID
}

// TestGetStockPage 测试库存分页（关键词/仓库/预警过滤）
func TestGetStockPage(t *testing.T) {
	stockTestDB(t)
	skuID := newStockFixture(t)
	ctx := context.Background()

	// 全量
	list, total, err := stockSvc.GetStockPage(ctx, jxc.StockQuery{PageInfo: request.PageInfo{Page: 1, PageSize: 10}})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("全量不符: total=%d list=%d err=%v", total, len(list), err)
	}
	if list[0].Sku == nil || list[0].Sku.Goods == nil || list[0].Sku.Goods.Name != "测试商品" {
		t.Error("SKU/商品未预加载")
	}
	// 关键词（skuCode 命中）
	_, total, _ = stockSvc.GetStockPage(ctx, jxc.StockQuery{PageInfo: pageInfoKW("SP001")})
	if total != 1 {
		t.Errorf("关键词 skuCode 应命中 1, got %d", total)
	}
	// 关键词（商品名命中）
	_, total, _ = stockSvc.GetStockPage(ctx, jxc.StockQuery{PageInfo: pageInfoKW("测试商品")})
	if total != 1 {
		t.Errorf("关键词商品名应命中 1, got %d", total)
	}
	// 关键词无命中
	_, total, _ = stockSvc.GetStockPage(ctx, jxc.StockQuery{PageInfo: pageInfoKW("不存在")})
	if total != 0 {
		t.Errorf("无关键词应命中 0, got %d", total)
	}
	// 仓库过滤
	_, total, _ = stockSvc.GetStockPage(ctx, jxc.StockQuery{WarehouseID: 1})
	if total != 1 {
		t.Errorf("仓库过滤应命中 1, got %d", total)
	}
	_, total, _ = stockSvc.GetStockPage(ctx, jxc.StockQuery{WarehouseID: 99})
	if total != 0 {
		t.Errorf("错误仓库应命中 0, got %d", total)
	}
	// 预警过滤（quantity 10 <= warning 15 → 命中）
	_, total, _ = stockSvc.GetStockPage(ctx, jxc.StockQuery{LowStock: true})
	if total != 1 {
		t.Errorf("预警过滤应命中 1, got %d", total)
	}
	// 空库
	global.GVA_DB.Exec("DELETE FROM stock")
	_, total, err = stockSvc.GetStockPage(ctx, jxc.StockQuery{})
	if err != nil || total != 0 {
		t.Errorf("空库应返回 0, got %d err=%v", total, err)
	}
	_ = skuID
}

// pageInfoKW 关键词分页参数
func pageInfoKW(kw string) request.PageInfo {
	return request.PageInfo{Page: 1, PageSize: 10, Keyword: kw}
}

// TestGetStockLogPage 测试流水分页过滤
func TestGetStockLogPage(t *testing.T) {
	stockTestDB(t)
	ctx := context.Background()
	// 两条流水（采购入库 + 直接出库）
	global.GVA_DB.Create(&jxc.StockLog{WarehouseID: 1, SkuID: 1, BusinessType: "purchase_in", BusinessNo: "PO-1", BeforeQty: 0, ChangeQty: 10, AfterQty: 10})
	global.GVA_DB.Create(&jxc.StockLog{WarehouseID: 1, SkuID: 1, BusinessType: "direct_out", BusinessNo: "DIRECT-OUT", BeforeQty: 10, ChangeQty: -2, AfterQty: 8})

	list, total, err := stockSvc.GetStockLogPage(ctx, jxc.StockLogQuery{PageInfo: request.PageInfo{Page: 1, PageSize: 10}})
	if err != nil || total != 2 || len(list) != 2 {
		t.Fatalf("全量流水不符: total=%d list=%d err=%v", total, len(list), err)
	}
	if list[0].BusinessType != "direct_out" {
		t.Error("应按 id 倒序（最新在前）")
	}
	// 按类型过滤
	_, total, _ = stockSvc.GetStockLogPage(ctx, jxc.StockLogQuery{BusinessType: "purchase_in"})
	if total != 1 {
		t.Errorf("类型过滤应命中 1, got %d", total)
	}
	// 空库
	global.GVA_DB.Exec("DELETE FROM stock_log")
	_, total, err = stockSvc.GetStockLogPage(ctx, jxc.StockLogQuery{})
	if err != nil || total != 0 {
		t.Errorf("空流水应返回 0, got %d err=%v", total, err)
	}
}

// TestDirectIn 测试直接入库（新增/累加 + 流水）
func TestDirectIn(t *testing.T) {
	stockTestDB(t)
	skuID := newStockFixture(t)
	ctx := context.Background()

	// 已有库存累加
	if err := stockSvc.DirectIn(ctx, 1, skuID, 5, "赠品入库", "tester"); err != nil {
		t.Fatalf("直接入库失败: %v", err)
	}
	var stock jxc.Stock
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = ?", skuID).First(&stock)
	if stock.Quantity != 15 {
		t.Errorf("累加入库后应为 15, got %d", stock.Quantity)
	}
	var logs []jxc.StockLog
	global.GVA_DB.Where("sku_id = ?", skuID).Find(&logs)
	if len(logs) != 1 || logs[0].BusinessType != "direct_in" || logs[0].BeforeQty != 10 || logs[0].AfterQty != 15 || logs[0].Remark != "赠品入库" {
		t.Errorf("入库流水不符: %+v", logs)
	}
	// 无库存记录时新建
	global.GVA_DB.Exec("DELETE FROM stock")
	if err := stockSvc.DirectIn(ctx, 1, skuID, 3, "", "tester"); err != nil {
		t.Fatalf("新建库存入库失败: %v", err)
	}
	var fresh jxc.Stock
	if err := global.GVA_DB.Where("warehouse_id = 1 AND sku_id = ?", skuID).First(&fresh).Error; err != nil {
		t.Fatalf("新建库存记录应存在: %v", err)
	}
	if fresh.Quantity != 3 {
		t.Errorf("新建库存应为 3, got %d", fresh.Quantity)
	}
	// 参数校验
	if err := stockSvc.DirectIn(ctx, 1, skuID, 0, "", "t"); err == nil {
		t.Error("数量 0 应报错")
	}
	if err := stockSvc.DirectIn(ctx, 1, 999, 1, "", "t"); err == nil {
		t.Error("SKU 不存在应报错")
	}
}

// TestDirectOut 测试直接出库（扣减/不足拒绝）
func TestDirectOut(t *testing.T) {
	stockTestDB(t)
	skuID := newStockFixture(t)
	ctx := context.Background()

	if err := stockSvc.DirectOut(ctx, 1, skuID, 4, "损耗出库", "tester"); err != nil {
		t.Fatalf("直接出库失败: %v", err)
	}
	var stock jxc.Stock
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = ?", skuID).First(&stock)
	if stock.Quantity != 6 {
		t.Errorf("出库后应为 6, got %d", stock.Quantity)
	}
	var logs []jxc.StockLog
	global.GVA_DB.Where("sku_id = ?", skuID).Find(&logs)
	if len(logs) != 1 || logs[0].BusinessType != "direct_out" || logs[0].ChangeQty != -4 || logs[0].AfterQty != 6 || logs[0].Remark != "损耗出库" {
		t.Errorf("出库流水不符: %+v", logs)
	}
	// 库存不足拒绝
	if err := stockSvc.DirectOut(ctx, 1, skuID, 10, "", "t"); err == nil {
		t.Error("库存不足出库应报错")
	}
	// 无库存记录拒绝
	global.GVA_DB.Exec("DELETE FROM stock")
	if err := stockSvc.DirectOut(ctx, 1, skuID, 1, "", "t"); err == nil {
		t.Error("无库存记录出库应报错")
	}
}

// TestStock_ErrorBranches 测试表缺失错误分支
func TestStock_ErrorBranches(t *testing.T) {
	stockTestDB(t)
	skuID := newStockFixture(t)
	ctx := context.Background()
	global.GVA_DB.Exec("DROP TABLE stock_log")
	global.GVA_DB.Exec("DROP TABLE stock")

	if _, _, err := stockSvc.GetStockPage(ctx, jxc.StockQuery{}); err == nil {
		t.Error("表缺失查询库存应报错")
	}
	if _, _, err := stockSvc.GetStockLogPage(ctx, jxc.StockLogQuery{}); err == nil {
		t.Error("表缺失查询流水应报错")
	}
	if err := stockSvc.DirectIn(ctx, 1, skuID, 1, "", "t"); err == nil {
		t.Error("表缺失入库应报错")
	}
	if err := stockSvc.DirectOut(ctx, 1, skuID, 1, "", "t"); err == nil {
		t.Error("表缺失出库应报错")
	}
}

package jxc

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// stockCheckTestDB 盘点测试库（含盘点表）
func stockCheckTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
		&jxc.SaleOrder{}, &jxc.SaleItem{},
		&jxc.StockCheck{}, &jxc.StockCheckItem{},
	)
}

// stockCheckFixture 前置：仓库 + 2 个 SKU（库存 10 / 5）
func stockCheckFixture(t *testing.T) (skuID1, skuID2 uint) {
	t.Helper()
	db := global.GVA_DB
	db.Create(&jxc.Warehouse{Code: "CK001", Name: "总仓", Status: 1})
	g := jxc.Goods{Code: "SP001", Name: "测试商品", Status: 1}
	db.Create(&g)
	sku1 := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-M-红", Color: "红", Size: "M", CostPrice: 8, SalePrice: 15, Status: 1}
	db.Create(&sku1)
	sku2 := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-L-蓝", Color: "蓝", Size: "L", CostPrice: 9, SalePrice: 18, Status: 1}
	db.Create(&sku2)
	db.Create(&jxc.Stock{WarehouseID: 1, SkuID: sku1.ID, Quantity: 10})
	db.Create(&jxc.Stock{WarehouseID: 1, SkuID: sku2.ID, Quantity: 5})
	return sku1.ID, sku2.ID
}

// TestStockCheckFlow 盘点全流程：创建快照 → 录入 → 完成（盘盈/盘亏调整 + 流水）
func TestStockCheckFlow(t *testing.T) {
	stockCheckTestDB(t)
	skuID1, skuID2 := stockCheckFixture(t)
	ctx := context.Background()
	svc := &StockCheckService{}

	check := &jxc.StockCheck{WarehouseID: 1, Checker: "tester", Remark: "月度盘点"}
	if err := svc.CreateStockCheck(ctx, check); err != nil {
		t.Fatalf("创建盘点单失败: %v", err)
	}
	if check.CheckNo == "" || check.Status != jxc.StockCheckStatusChecking {
		t.Fatalf("盘点单初始状态不符: %+v", check)
	}
	if len(check.Items) != 2 {
		t.Fatalf("快照明细应为 2 条, got %d", len(check.Items))
	}
	// 明细 system_qty 快照
	if check.Items[0].SystemQty != 10 || check.Items[1].SystemQty != 5 {
		t.Fatalf("快照数量不符: %+v", check.Items)
	}
	// 录入盘点数：SKU1 实盘 12（盘盈2）、SKU2 实盘 3（盘亏2）
	actual := 12
	actual2 := 3
	items := []jxc.StockCheckItem{{ID: check.Items[0].ID, ActualQty: &actual}, {ID: check.Items[1].ID, ActualQty: &actual2}}
	if err := svc.UpdateStockCheckItems(ctx, check.ID, items); err != nil {
		t.Fatalf("录入失败: %v", err)
	}
	// 完成盘点
	if err := svc.CompleteStockCheck(ctx, check.ID, "tester"); err != nil {
		t.Fatalf("完成盘点失败: %v", err)
	}
	var done jxc.StockCheck
	global.GVA_DB.First(&done, check.ID)
	if done.Status != jxc.StockCheckStatusDone {
		t.Errorf("完成状态不符: %d", done.Status)
	}
	// 库存调整：10 → 12、5 → 3
	var st1, st2 jxc.Stock
	global.GVA_DB.Where("sku_id = ?", skuID1).First(&st1)
	global.GVA_DB.Where("sku_id = ?", skuID2).First(&st2)
	if st1.Quantity != 12 || st2.Quantity != 3 {
		t.Errorf("库存调整不符: sku1=%d sku2=%d", st1.Quantity, st2.Quantity)
	}
	// 流水：check_in +2 / check_out -2
	var logs []jxc.StockLog
	global.GVA_DB.Where("business_no = ?", check.CheckNo).Order("id").Find(&logs)
	if len(logs) != 2 || logs[0].BusinessType != "check_in" || logs[0].ChangeQty != 2 ||
		logs[1].BusinessType != "check_out" || logs[1].ChangeQty != -2 {
		t.Errorf("盘点流水不符: %+v", logs)
	}
	// 明细 diff 回写
	var it1, it2 jxc.StockCheckItem
	global.GVA_DB.First(&it1, check.Items[0].ID)
	global.GVA_DB.First(&it2, check.Items[1].ID)
	if it1.DiffQty != 2 || it2.DiffQty != -2 {
		t.Errorf("diff 回写不符: it1=%d it2=%d", it1.DiffQty, it2.DiffQty)
	}
	// 完成/取消/录入已被拒绝
	if err := svc.CompleteStockCheck(ctx, check.ID, "t"); err == nil {
		t.Error("已完成盘点单重复完成应拒绝")
	}
	if err := svc.CancelStockCheck(ctx, check.ID); err == nil {
		t.Error("已完成盘点单取消应拒绝")
	}
	if err := svc.UpdateStockCheckItems(ctx, check.ID, items); err == nil {
		t.Error("已完成盘点单录入应拒绝")
	}
	// 分页 + 详情
	list, total, err := svc.GetStockCheckPage(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("分页不符: total=%d list=%d err=%v", total, len(list), err)
	}
	if _, total, err := svc.GetStockCheckPage(ctx, request.PageInfo{Page: 1, PageSize: 10, Keyword: check.CheckNo}); err != nil || total != 1 {
		t.Fatalf("按单号搜索不符: %v", err)
	}
	det, err := svc.GetStockCheckDetail(ctx, check.ID)
	if err != nil || len(det.Items) != 2 || det.Items[0].Sku == nil {
		t.Fatalf("详情不符: %+v err=%v", det.Items, err)
	}
}

// TestStockCheckPage_Empty 空库分页
func TestStockCheckPage_Empty(t *testing.T) {
	stockCheckTestDB(t)
	ctx := context.Background()
	svc := &StockCheckService{}
	list, total, err := svc.GetStockCheckPage(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 0 || len(list) != 0 {
		t.Fatalf("空库分页不符: total=%d list=%d err=%v", total, len(list), err)
	}
}

// TestStockCheck_LogMissing 流水表缺失：完成（写流水）应报错
func TestStockCheck_LogMissing(t *testing.T) {
	stockCheckTestDB(t)
	stockCheckFixture(t)
	ctx := context.Background()
	svc := &StockCheckService{}

	check := &jxc.StockCheck{WarehouseID: 1}
	if err := svc.CreateStockCheck(ctx, check); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	actual := 8 // 盘亏 2
	if err := svc.UpdateStockCheckItems(ctx, check.ID, []jxc.StockCheckItem{{ID: check.Items[0].ID, ActualQty: &actual}}); err != nil {
		t.Fatalf("录入失败: %v", err)
	}
	global.GVA_DB.Exec("DROP TABLE stock_log")
	if err := svc.CompleteStockCheck(ctx, check.ID, "t"); err == nil {
		t.Error("流水表缺失完成应报错")
	}
	// 事务回滚：状态仍为盘点中
	var c jxc.StockCheck
	global.GVA_DB.First(&c, check.ID)
	if c.Status != jxc.StockCheckStatusChecking {
		t.Errorf("事务回滚后状态应为盘点中, got %d", c.Status)
	}
}

// TestStockCheck_Errors 盘点异常分支
func TestStockCheck_Errors(t *testing.T) {
	stockCheckTestDB(t)
	skuID1, _ := stockCheckFixture(t)
	ctx := context.Background()
	svc := &StockCheckService{}

	// 仓库缺失 / 仓库不存在
	if err := svc.CreateStockCheck(ctx, &jxc.StockCheck{}); err == nil {
		t.Error("未选仓库应报错")
	}
	if err := svc.CreateStockCheck(ctx, &jxc.StockCheck{WarehouseID: 99}); err == nil {
		t.Error("仓库不存在应报错")
	}
	// 无库存仓库：创建空盘点单成功
	db := global.GVA_DB
	db.Create(&jxc.Warehouse{Code: "CK002", Name: "空仓", Status: 1})
	empty := &jxc.StockCheck{WarehouseID: 2}
	if err := svc.CreateStockCheck(ctx, empty); err != nil {
		t.Fatalf("空仓创建失败: %v", err)
	}
	if len(empty.Items) != 0 {
		t.Errorf("空仓应无明细, got %d", len(empty.Items))
	}
	// 录入：空列表 / 明细ID缺失 / 负数
	if err := svc.UpdateStockCheckItems(ctx, empty.ID, nil); err == nil {
		t.Error("空明细录入应报错")
	}
	neg := -1
	if err := svc.UpdateStockCheckItems(ctx, empty.ID, []jxc.StockCheckItem{{ID: 1, ActualQty: &neg}}); err == nil {
		t.Error("负数实盘应报错")
	}
	if err := svc.UpdateStockCheckItems(ctx, empty.ID, []jxc.StockCheckItem{{ID: 999, ActualQty: &neg}}); err == nil {
		t.Error("明细不存在应报错")
	}
	// 盘点单不存在
	if err := svc.CompleteStockCheck(ctx, 999, "t"); err == nil {
		t.Error("盘点单不存在完成应报错")
	}
	if err := svc.CancelStockCheck(ctx, 999); err == nil {
		t.Error("盘点单不存在取消应报错")
	}
	if err := svc.UpdateStockCheckItems(ctx, 999, []jxc.StockCheckItem{{ID: 1, ActualQty: &neg}}); err == nil {
		t.Error("盘点单不存在录入应报错")
	}
	// 未录入明细完成：按账存视为相符（无差异）
	check := &jxc.StockCheck{WarehouseID: 1}
	if err := svc.CreateStockCheck(ctx, check); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if err := svc.CompleteStockCheck(ctx, check.ID, "t"); err != nil {
		t.Fatalf("未录入完成失败: %v", err)
	}
	var st jxc.Stock
	global.GVA_DB.Where("sku_id = ?", skuID1).First(&st)
	if st.Quantity != 10 {
		t.Errorf("未录入完成不应调整库存, got %d", st.Quantity)
	}
	// 取消
	c2 := &jxc.StockCheck{WarehouseID: 1}
	if err := svc.CreateStockCheck(ctx, c2); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if err := svc.CancelStockCheck(ctx, c2.ID); err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	// 取消后完成应拒绝
	if err := svc.CompleteStockCheck(ctx, c2.ID, "t"); err == nil {
		t.Error("已取消盘点单完成应拒绝")
	}
	// 盘点新增 SKU（库存表无记录但实盘有）→ 创建库存行
	db.Exec("DELETE FROM stock")
	check2 := &jxc.StockCheck{WarehouseID: 1}
	if err := svc.CreateStockCheck(ctx, check2); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if len(check2.Items) != 0 {
		t.Fatalf("无库存应无明细, got %d", len(check2.Items))
	}
	// 手工插入一条明细（模拟实盘发现新 SKU）
	actual := 7
	newItem := jxc.StockCheckItem{CheckID: check2.ID, SkuID: skuID1, SystemQty: 0, ActualQty: &actual, DiffQty: 7}
	if err := db.Create(&newItem).Error; err != nil {
		t.Fatalf("插入明细失败: %v", err)
	}
	if err := svc.CompleteStockCheck(ctx, check2.ID, "t"); err != nil {
		t.Fatalf("完成失败: %v", err)
	}
	var st2 jxc.Stock
	global.GVA_DB.Where("sku_id = ?", skuID1).First(&st2)
	if st2.Quantity != 7 {
		t.Errorf("盘点新增库存应为 7, got %d", st2.Quantity)
	}
	var log jxc.StockLog
	global.GVA_DB.Where("business_no = ? AND business_type = ?", check2.CheckNo, "check_in").First(&log)
	if log.ChangeQty != 7 {
		t.Errorf("盘点新增流水应为 +7, got %d", log.ChangeQty)
	}
	// 库存表缺失：完成（差异调整）应报错
	c3 := &jxc.StockCheck{WarehouseID: 1}
	if err := svc.CreateStockCheck(ctx, c3); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	actual3 := 8
	if err := svc.UpdateStockCheckItems(ctx, c3.ID, []jxc.StockCheckItem{{ID: c3.Items[0].ID, ActualQty: &actual3}}); err != nil {
		t.Fatalf("录入失败: %v", err)
	}
	db.Exec("DROP TABLE stock")
	if err := svc.CompleteStockCheck(ctx, c3.ID, "t"); err == nil {
		t.Error("库存表缺失完成应报错")
	}
	// 单号解析失败（坏单号插入后再创建，须在 DROP stock_check 之前）
	bad := jxc.StockCheck{CheckNo: "CK-20260805-XXX", WarehouseID: 1, Status: 1}
	db.Create(&bad)
	if err := svc.CreateStockCheck(ctx, &jxc.StockCheck{WarehouseID: 1}); err == nil {
		t.Error("单号解析失败应报错")
	}
	// 明细表缺失
	db.Exec("DROP TABLE stock_check_item")
	if err := svc.CreateStockCheck(ctx, &jxc.StockCheck{WarehouseID: 1}); err == nil {
		t.Error("明细表缺失创建应报错")
	}
	if err := svc.CompleteStockCheck(ctx, c2.ID, "t"); err == nil {
		t.Error("明细表缺失完成应报错")
	}
	// 主表缺失：单号生成查询报错
	db.Exec("DROP TABLE stock_check")
	if err := svc.CreateStockCheck(ctx, &jxc.StockCheck{WarehouseID: 1}); err == nil {
		t.Error("主表缺失创建应报错")
	}
	if err := svc.CancelStockCheck(ctx, 1); err == nil {
		t.Error("主表缺失取消应报错")
	}
	if err := svc.CompleteStockCheck(ctx, c2.ID, "t"); err == nil {
		t.Error("主表缺失完成应报错")
	}
}

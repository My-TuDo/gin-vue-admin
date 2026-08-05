package jxc

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// TestDashboard 仪表盘聚合（造销售/退货/换货数据验证口径）
func TestDashboard(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	saleSvc := &SaleService{}
	dashSvc := &DashboardService{}

	// SKU 设安全库存 5（当前库存 10，不预警）
	global.GVA_DB.Model(&jxc.GoodsSku{}).Where("id = ?", skuID).Update("safe_stock", 5)

	// 正常销售 2 件出库（15*2=30，成本 8*2=16）
	sale := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeNormal,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 2, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, sale); err != nil {
		t.Fatalf("建单失败: %v", err)
	}
	if err := saleSvc.ConfirmOut(ctx, sale.ID, "t"); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	// 退货 1 件（-15，成本 -8）
	ret := &jxc.SaleOrder{WarehouseID: 1, OrderType: jxc.SaleTypeReturn, OriginalOrderID: &sale.ID,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, ret); err != nil {
		t.Fatalf("退货单失败: %v", err)
	}
	if err := saleSvc.ConfirmReturn(ctx, ret.ID, "t"); err != nil {
		t.Fatalf("退货失败: %v", err)
	}

	// 概览：净销售额 30-15=15；毛利 (30-16)+(-15+8)=7；退货额 -15
	ov, err := dashSvc.Overview(ctx)
	if err != nil || len(ov) != 3 {
		t.Fatalf("概览失败: %+v err=%v", ov, err)
	}
	if ov[0].Sales != 15 || ov[0].Orders != 2 {
		t.Errorf("今日销售额/订单数不符: %+v", ov[0])
	}
	if ov[0].Profit != 7 {
		t.Errorf("今日毛利应为 7, got %v", ov[0].Profit)
	}
	if ov[0].ReturnAmt != -15 {
		t.Errorf("今日退货额应为 -15, got %v", ov[0].ReturnAmt)
	}

	// 趋势（近 7 天）：至少 2 单
	trend, err := dashSvc.Trend(ctx, 7)
	if err != nil || len(trend) == 0 || trend[0].Orders < 2 {
		t.Fatalf("趋势不符: %+v err=%v", trend, err)
	}

	// 热销（正常销售+换出）：销量 2
	top, err := dashSvc.Top(ctx, 30, 5)
	if err != nil || len(top) == 0 || top[0].Qty != 2 {
		t.Fatalf("热销不符: %+v err=%v", top, err)
	}

	// 分类占比：无分类 → 未分类
	cat, err := dashSvc.Category(ctx, 30)
	if err != nil || len(cat) == 0 {
		t.Fatalf("分类不符: %+v err=%v", cat, err)
	}

	// 预警：库存 9（10-2+1），安全线 5 → 不预警
	alerts, err := dashSvc.StockAlert(ctx)
	if err != nil || len(alerts) != 0 {
		t.Fatalf("不应预警: %+v err=%v", alerts, err)
	}
	// 调高安全线到 50 → 预警
	global.GVA_DB.Model(&jxc.GoodsSku{}).Where("id = ?", skuID).Update("safe_stock", 50)
	alerts, err = dashSvc.StockAlert(ctx)
	if err != nil || len(alerts) != 1 || alerts[0].Available != 9 {
		t.Fatalf("预警不符: %+v err=%v", alerts, err)
	}
	// 空库聚合（无数据）
	global.GVA_DB.Exec("DELETE FROM sale_order")
	ov, err = dashSvc.Overview(ctx)
	if err != nil || ov[0].Sales != 0 || ov[0].Orders != 0 {
		t.Fatalf("空库概览不符: %+v err=%v", ov, err)
	}
	// 默认参数（days/limit <= 0 走默认值）
	if _, err := dashSvc.Trend(ctx, 0); err != nil {
		t.Errorf("Trend 默认参数失败: %v", err)
	}
	if _, err := dashSvc.Top(ctx, 0, 0); err != nil {
		t.Errorf("Top 默认参数失败: %v", err)
	}
	if _, err := dashSvc.Category(ctx, 0); err != nil {
		t.Errorf("Category 默认参数失败: %v", err)
	}
	// 主表缺失：聚合报错
	global.GVA_DB.Exec("DROP TABLE sale_order")
	if _, err := dashSvc.Overview(ctx); err == nil {
		t.Error("主表缺失概览应报错")
	}
	if _, err := dashSvc.Trend(ctx, 7); err == nil {
		t.Error("主表缺失趋势应报错")
	}
	if _, err := dashSvc.Top(ctx, 30, 5); err == nil {
		t.Error("主表缺失热销应报错")
	}
	if _, err := dashSvc.Category(ctx, 30); err == nil {
		t.Error("主表缺失分类应报错")
	}
}

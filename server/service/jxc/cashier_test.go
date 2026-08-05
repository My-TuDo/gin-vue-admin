package jxc

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// TestCashierCheckout 收银结算：收款即出库
func TestCashierCheckout(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	svc := &CashierService{}

	order, err := svc.Checkout(ctx, jxc.POSCheckoutReq{
		WarehouseID: 1, PayMethod: "cash", PaidAmount: 100,
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 3}},
	}, "tester")
	if err != nil {
		t.Fatalf("收银失败: %v", err)
	}
	// 收款即出库：状态已出库、金额 3*15=45
	if order.Status != jxc.SaleStatusShipped {
		t.Errorf("收银单应直接已出库, got %d", order.Status)
	}
	if order.TotalAmount != 45 {
		t.Errorf("金额应为 45, got %v", order.TotalAmount)
	}
	// 库存 10-3=7
	qty, _ := getStockQty(t, skuID)
	if qty != 7 {
		t.Errorf("收银后库存应为 7, got %d", qty)
	}
	// 流水 sale_out -3
	var log jxc.StockLog
	global.GVA_DB.Where("business_no = ?", order.OrderNo).First(&log)
	if log.BusinessType != "sale_out" || log.ChangeQty != -3 {
		t.Errorf("收银流水不符: %+v", log)
	}
	// 微信支付（实收=应付）
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{
		WarehouseID: 1, PayMethod: "wechat",
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}},
	}, "t"); err != nil {
		t.Fatalf("微信收款失败: %v", err)
	}
	// 换货单确认后金额符号不影响收款（收银只走正常销售）
}

// TestCashierCheckout_Errors 收银异常分支
func TestCashierCheckout_Errors(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	svc := &CashierService{}

	// 空明细
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{WarehouseID: 1, PayMethod: "cash"}, "t"); err == nil {
		t.Error("空明细应报错")
	}
	// 数量 0
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{WarehouseID: 1, PayMethod: "cash",
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 0}}}, "t"); err == nil {
		t.Error("数量0应报错")
	}
	// 仓库缺失 / 不存在
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{PayMethod: "cash",
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("未选仓库应报错")
	}
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{WarehouseID: 99, PayMethod: "cash",
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("仓库不存在应报错")
	}
	// 支付方式非法
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{WarehouseID: 1, PayMethod: "card",
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("非法支付方式应报错")
	}
	// 实收不足
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{WarehouseID: 1, PayMethod: "cash", PaidAmount: 10,
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 3}}}, "t"); err == nil {
		t.Error("实收不足应报错")
	}
	// 可售不足：锁定 10 件后收银
	global.GVA_DB.Model(&jxc.Stock{}).Where("sku_id = ?", skuID).Update("lock_quantity", 10)
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{WarehouseID: 1, PayMethod: "cash", PaidAmount: 100,
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("可售不足应报错")
	}
	global.GVA_DB.Model(&jxc.Stock{}).Where("sku_id = ?", skuID).Update("lock_quantity", 0)
	// 无库存记录
	global.GVA_DB.Exec("DELETE FROM stock")
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{WarehouseID: 1, PayMethod: "cash", PaidAmount: 100,
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("无库存记录应报错")
	}
	// 明细表缺失（单号生成/创建报错）
	global.GVA_DB.Create(&jxc.Stock{WarehouseID: 1, SkuID: skuID, Quantity: 10})
	global.GVA_DB.Exec("DROP TABLE sale_item")
	if _, err := svc.Checkout(ctx, jxc.POSCheckoutReq{WarehouseID: 1, PayMethod: "cash", PaidAmount: 100,
		Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("明细表缺失应报错")
	}
}

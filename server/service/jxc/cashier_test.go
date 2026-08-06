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

// TestCashierRefundableOrders 可退换原单列表（剩余>0 过滤）
func TestCashierRefundableOrders(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	svc := &CashierService{}
	saleSvc := &SaleService{}

	// 空库
	list, err := svc.RefundableOrders(ctx)
	if err != nil || len(list) != 0 {
		t.Fatalf("空库应无原单: %+v err=%v", list, err)
	}
	// 单 A 出库 3 件、单 B 出库 2 件
	a := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 3, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, a); err != nil {
		t.Fatalf("建单A失败: %v", err)
	}
	if err := saleSvc.ConfirmOut(ctx, a.ID, "t"); err != nil {
		t.Fatalf("出库A失败: %v", err)
	}
	b := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 2, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, b); err != nil {
		t.Fatalf("建单B失败: %v", err)
	}
	if err := saleSvc.ConfirmOut(ctx, b.ID, "t"); err != nil {
		t.Fatalf("出库B失败: %v", err)
	}
	// 未出库单不出现
	pending := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, pending); err != nil {
		t.Fatalf("建待出库单失败: %v", err)
	}
	list, err = svc.RefundableOrders(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("应有 2 张可退换原单: %+v err=%v", list, err)
	}
	// 最新创建的单排最前（B 后创建，剩余 2）
	if list[0].OrderNo != b.OrderNo || list[0].Remaining != 2 {
		t.Errorf("最新单 B 应排最前: %+v", list[0])
	}
	// 退掉 A 全部 → A 不再出现
	// 退掉 A 全部 → A 不再出现
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{
		OriginalOrderID: a.ID,
		Items:           []jxc.POSItem{{SkuID: skuID, Qty: 3}},
	}, "t"); err != nil {
		t.Fatalf("退款失败: %v", err)
	}
	list, err = svc.RefundableOrders(ctx)
	if err != nil || len(list) != 1 || list[0].ID != b.ID {
		t.Fatalf("退完后 A 应被过滤: %+v err=%v", list, err)
	}
	if list[0].UsedQty != 0 || list[0].Remaining != 2 {
		t.Errorf("B 剩余应为 2: %+v", list[0])
	}
	// 明细表缺失报错
	global.GVA_DB.Exec("DROP TABLE sale_item")
	if _, err := svc.RefundableOrders(ctx); err == nil {
		t.Error("明细表缺失应报错")
	}
}

// TestCashierRefund 收银退款：退货单直接入库
func TestCashierRefund(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	svc := &CashierService{}
	saleSvc := &SaleService{}

	// 原销售单：出库 3 件（库存 10 → 7）
	sale := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 3, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, sale); err != nil {
		t.Fatalf("建单失败: %v", err)
	}
	if err := saleSvc.ConfirmOut(ctx, sale.ID, "t"); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	// 退款 2 件
	ret, err := svc.Refund(ctx, jxc.POSRefundReq{
		OriginalOrderID: sale.ID,
		Items:           []jxc.POSItem{{SkuID: skuID, Qty: 2}},
	}, "tester")
	if err != nil {
		t.Fatalf("退款失败: %v", err)
	}
	// 退货单已入库：状态 2、金额 -30
	if ret.Status != jxc.SaleStatusShipped {
		t.Errorf("退款单应直接已入库, got %d", ret.Status)
	}
	if ret.TotalAmount != -30 {
		t.Errorf("退款金额应为 -30, got %v", ret.TotalAmount)
	}
	// 库存回补：7+2=9
	qty, _ := getStockQty(t, skuID)
	if qty != 9 {
		t.Errorf("退款后库存应为 9, got %d", qty)
	}
	// 流水 sale_return +2
	var log jxc.StockLog
	global.GVA_DB.Where("business_no = ?", ret.OrderNo).First(&log)
	if log.BusinessType != "sale_return" || log.ChangeQty != 2 {
		t.Errorf("退款流水不符: %+v", log)
	}
	// 超额度退款（剩余 1 件，退 2）拒绝
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{
		OriginalOrderID: sale.ID,
		Items:           []jxc.POSItem{{SkuID: skuID, Qty: 2}},
	}, "t"); err == nil {
		t.Error("超额度退款应拒绝")
	}
	// 原单外 SKU 拒绝
	var g2 jxc.Goods
	g2 = jxc.Goods{Code: "SP002", Name: "另一商品", Status: 1}
	global.GVA_DB.Create(&g2)
	var sku2 jxc.GoodsSku
	sku2 = jxc.GoodsSku{GoodsID: g2.ID, SkuCode: "SP002-M-红", Color: "红", Size: "M", CostPrice: 8, SalePrice: 15, Status: 1}
	global.GVA_DB.Create(&sku2)
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{
		OriginalOrderID: sale.ID,
		Items:           []jxc.POSItem{{SkuID: sku2.ID, Qty: 1}},
	}, "t"); err == nil {
		t.Error("原单外商品退款应拒绝")
	}
}

// TestCashierRefund_Errors 退款异常分支
func TestCashierRefund_Errors(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	svc := &CashierService{}
	saleSvc := &SaleService{}

	// 参数缺失
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("未选原单应报错")
	}
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{OriginalOrderID: 1}, "t"); err == nil {
		t.Error("空明细应报错")
	}
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{OriginalOrderID: 1, Items: []jxc.POSItem{{SkuID: skuID, Qty: 0}}}, "t"); err == nil {
		t.Error("数量0应报错")
	}
	// 原单不存在 / 未出库
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{OriginalOrderID: 999, Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("原单不存在应报错")
	}
	sale := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, sale); err != nil {
		t.Fatalf("建单失败: %v", err)
	}
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{OriginalOrderID: sale.ID, Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("未出库原单退款应报错")
	}
	// 非法原单类型（退货单作原单）
	db := global.GVA_DB
	db.Create(&jxc.SaleOrder{OrderNo: "SO-R1", WarehouseID: 1, OrderType: jxc.SaleTypeReturn, Status: jxc.SaleStatusShipped})
	var retOrder jxc.SaleOrder
	db.Where("order_no = ?", "SO-R1").First(&retOrder)
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{OriginalOrderID: retOrder.ID, Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("退货单作原单退款应报错")
	}
	// 明细表缺失
	db.Exec("DROP TABLE sale_item")
	if _, err := svc.Refund(ctx, jxc.POSRefundReq{OriginalOrderID: sale.ID, Items: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("明细表缺失退款应报错")
	}
}

// TestCashierExchange 收银换货：退回入库 + 换出出库，差额多退少补
func TestCashierExchange(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	svc := &CashierService{}
	saleSvc := &SaleService{}

	// 原单：出库 2 件（库存 10 → 8）
	sale := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 2, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, sale); err != nil {
		t.Fatalf("建单失败: %v", err)
	}
	if err := saleSvc.ConfirmOut(ctx, sale.ID, "t"); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	// 换货：退回 1 件（-15）+ 换出 2 件新商品（30）→ 差额 15 应收
	var g2 jxc.Goods
	g2 = jxc.Goods{Code: "SP002", Name: "另一商品", Status: 1}
	global.GVA_DB.Create(&g2)
	var sku2 jxc.GoodsSku
	sku2 = jxc.GoodsSku{GoodsID: g2.ID, SkuCode: "SP002-M-红", Color: "红", Size: "M", CostPrice: 8, SalePrice: 15, Status: 1}
	global.GVA_DB.Create(&sku2)
	global.GVA_DB.Create(&jxc.Stock{WarehouseID: 1, SkuID: sku2.ID, Quantity: 10})

	res, err := svc.Exchange(ctx, jxc.POSExchangeReq{
		OriginalOrderID: sale.ID, PayMethod: "cash", PaidAmount: 15,
		ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}},
		OutItems:    []jxc.POSItem{{SkuID: sku2.ID, Qty: 2}},
	}, "tester")
	if err != nil {
		t.Fatalf("换货失败: %v", err)
	}
	// 差额 30-15=15
	if res.DiffAmount != 15 {
		t.Errorf("差额应为 15, got %v", res.DiffAmount)
	}
	// 退货单已入库、销售单已出库
	if res.ReturnOrder.Status != jxc.SaleStatusShipped || res.ReturnOrder.TotalAmount != -15 {
		t.Errorf("退货单状态不符: %+v", res.ReturnOrder)
	}
	if res.OutOrder.Status != jxc.SaleStatusShipped || res.OutOrder.TotalAmount != 30 {
		t.Errorf("销售单状态不符: %+v", res.OutOrder)
	}
	// 库存：原 SKU 8+1=9；新 SKU 10-2=8
	q1, _ := getStockQty(t, skuID)
	q2, _ := getStockQty(t, sku2.ID)
	if q1 != 9 || q2 != 8 {
		t.Errorf("换货库存不符: sku1=%d sku2=%d", q1, q2)
	}
	// 现金实收不足拒绝（差额 15，实收 10）
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{
		OriginalOrderID: sale.ID, PayMethod: "cash", PaidAmount: 10,
		ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}},
		OutItems:    []jxc.POSItem{{SkuID: sku2.ID, Qty: 2}},
	}, "t"); err == nil {
		t.Error("实收不足应拒绝")
	}
	// 退回超额度拒绝（原单剩 1 件，退 2）
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{
		OriginalOrderID: sale.ID, PayMethod: "wechat",
		ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 2}},
		OutItems:    []jxc.POSItem{{SkuID: sku2.ID, Qty: 1}},
	}, "t"); err == nil {
		t.Error("退回超额度应拒绝")
	}
}

// TestCashierExchange_Errors 换货异常分支
func TestCashierExchange_Errors(t *testing.T) {
	saleTestDB(t)
	skuID := newSaleFixture(t)
	ctx := context.Background()
	svc := &CashierService{}
	saleSvc := &SaleService{}

	// 参数缺失
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{PayMethod: "cash"}, "t"); err == nil {
		t.Error("未选原单应报错")
	}
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{OriginalOrderID: 1, PayMethod: "cash", OutItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("缺退回商品应报错")
	}
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{OriginalOrderID: 1, PayMethod: "cash", ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("缺换出商品应报错")
	}
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{OriginalOrderID: 1, PayMethod: "card",
		ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}, OutItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("非法支付方式应报错")
	}
	// 原单不存在
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{OriginalOrderID: 999, PayMethod: "cash",
		ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}, OutItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("原单不存在应报错")
	}
	// 原单未出库
	sale := &jxc.SaleOrder{WarehouseID: 1, Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1, Price: 15}}}
	if err := saleSvc.CreateSaleOrder(ctx, sale); err != nil {
		t.Fatalf("建单失败: %v", err)
	}
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{OriginalOrderID: sale.ID, PayMethod: "cash",
		ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}, OutItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("未出库原单换货应报错")
	}
	// 非法原单类型（退货单作原单）
	db := global.GVA_DB
	db.Create(&jxc.SaleOrder{OrderNo: "SO-R1", WarehouseID: 1, OrderType: jxc.SaleTypeReturn, Status: jxc.SaleStatusShipped})
	var retOrder jxc.SaleOrder
	db.Where("order_no = ?", "SO-R1").First(&retOrder)
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{OriginalOrderID: retOrder.ID, PayMethod: "cash",
		ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}, OutItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("退货单作原单换货应报错")
	}
	// 明细表缺失
	if err := saleSvc.ConfirmOut(ctx, sale.ID, "t"); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	// 不存在的 SKU（fillItemSnapshot 拒绝）
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{OriginalOrderID: sale.ID, PayMethod: "cash",
		ReturnItems: []jxc.POSItem{{SkuID: 9999, Qty: 1}}, OutItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("不存在的 SKU 应报错")
	}
	db.Exec("DROP TABLE sale_item")
	if _, err := svc.Exchange(ctx, jxc.POSExchangeReq{OriginalOrderID: sale.ID, PayMethod: "cash",
		ReturnItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}, OutItems: []jxc.POSItem{{SkuID: skuID, Qty: 1}}}, "t"); err == nil {
		t.Error("明细表缺失换货应报错")
	}
}

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

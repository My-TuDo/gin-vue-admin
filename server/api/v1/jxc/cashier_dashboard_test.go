package jxc

import (
	"net/http"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
)

var cashierApi2 = &CashierApi{}
var dashboardApi2 = &DashboardApi{}

func cashierApiTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
		&jxc.SaleOrder{}, &jxc.SaleItem{},
		&jxc.StockCheck{}, &jxc.StockCheckItem{},
	)
}

// TestApiCashierCheckout 收银 api
func TestApiCashierCheckout(t *testing.T) {
	cashierApiTestDB(t)
	seedSale(t)

	// 成功（现金 50 元买 2 件 15 元）
	_, rb := doReq(t, cashierApi2.Checkout, http.MethodPost,
		`{"warehouseId":1,"payMethod":"cash","paidAmount":50,"items":[{"skuId":1,"qty":2}]}`)
	if rb.Code != 0 {
		t.Fatalf("checkout code=%d msg=%s", rb.Code, rb.Msg)
	}
	var order jxc.SaleOrder
	global.GVA_DB.Where("order_type = 1 AND status = 2").First(&order)
	if order.TotalAmount != 30 {
		t.Errorf("金额应为 30, got %v", order.TotalAmount)
	}
	// 微信支付
	_, rb = doReq(t, cashierApi2.Checkout, http.MethodPost,
		`{"warehouseId":1,"payMethod":"wechat","items":[{"skuId":1,"qty":1}]}`)
	if rb.Code != 0 {
		t.Fatalf("wechat code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 库存不足
	_, rb = doReq(t, cashierApi2.Checkout, http.MethodPost,
		`{"warehouseId":1,"payMethod":"cash","paidAmount":1000,"items":[{"skuId":1,"qty":999}]}`)
	if rb.Code == 0 {
		t.Error("库存不足应失败")
	}
	// 参数绑定错误
	_, rb = doReq(t, cashierApi2.Checkout, http.MethodPost, `not-json`)
	if rb.Code == 0 {
		t.Error("坏参数应失败")
	}
	// 收银退款（原单=第一笔 30 元单，退 1 件 15 元）
	_, rb = doReq(t, cashierApi2.Refund, http.MethodPost, `{"originalOrderId":1,"items":[{"skuId":1,"qty":1}]}`)
	if rb.Code != 0 {
		t.Fatalf("refund code=%d msg=%s", rb.Code, rb.Msg)
	}
	var ret jxc.SaleOrder
	global.GVA_DB.Where("order_type = 2").First(&ret)
	if ret.TotalAmount != -15 || ret.Status != jxc.SaleStatusShipped {
		t.Errorf("退款单不符: %+v", ret)
	}
	// 退款超额度拒绝
	_, rb = doReq(t, cashierApi2.Refund, http.MethodPost, `{"originalOrderId":1,"items":[{"skuId":1,"qty":99}]}`)
	if rb.Code == 0 {
		t.Error("退款超额度应失败")
	}
	// 退款参数绑定错误
	_, rb = doReq(t, cashierApi2.Refund, http.MethodPost, `not-json`)
	if rb.Code == 0 {
		t.Error("坏参数退款应失败")
	}
	// 收银换货（退 1 件 + 换出 1 件同 SKU，差额 0）
	_, rb = doReq(t, cashierApi2.Exchange, http.MethodPost,
		`{"originalOrderId":1,"payMethod":"wechat","returnItems":[{"skuId":1,"qty":1}],"outItems":[{"skuId":1,"qty":1}]}`)
	if rb.Code != 0 {
		t.Fatalf("exchange code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 换货实收不足拒绝
	_, rb = doReq(t, cashierApi2.Exchange, http.MethodPost,
		`{"originalOrderId":1,"payMethod":"cash","paidAmount":1,"returnItems":[{"skuId":1,"qty":1}],"outItems":[{"skuId":1,"qty":1}]}`)
	if rb.Code == 0 {
		t.Error("换货实收不足应失败")
	}
	// 换货参数绑定错误
	_, rb = doReq(t, cashierApi2.Exchange, http.MethodPost, `not-json`)
	if rb.Code == 0 {
		t.Error("坏参数换货应失败")
	}
}

// TestApiDashboard 仪表盘 api
func TestApiDashboard(t *testing.T) {
	cashierApiTestDB(t)
	seedSale(t)
	// 建销售单并出库（状态 2）
	_, rb := doReq(t, sapi.CreateSaleOrder, http.MethodPost, saleBody)
	if rb.Code != 0 {
		t.Fatalf("create code=%d msg=%s", rb.Code, rb.Msg)
	}
	_, rb = doReq(t, sapi.ConfirmOut, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("out code=%d msg=%s", rb.Code, rb.Msg)
	}

	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"overview", dashboardApi2.Overview},
		{"trend", dashboardApi2.Trend},
		{"top", dashboardApi2.Top},
		{"stock-alert", dashboardApi2.StockAlert},
		{"category", dashboardApi2.Category},
	} {
		c, w := newCtxQuery(t, "")
		tc.call(c)
		if w.Code != http.StatusOK {
			t.Fatalf("%s status=%d", tc.name, w.Code)
		}
	}
	// 带参查询
	c, w := newCtxQuery(t, "days=7&limit=5")
	dashboardApi2.Top(c)
	if w.Code != http.StatusOK {
		t.Fatalf("top params status=%d", w.Code)
	}
	// 聚合失败分支（主表缺失）
	global.GVA_DB.Exec("DROP TABLE sale_order")
	c, w = newCtxQuery(t, "")
	dashboardApi2.Overview(c)
	if w.Code != http.StatusOK {
		t.Fatalf("overview err status=%d", w.Code)
	}
	c, w = newCtxQuery(t, "")
	dashboardApi2.Trend(c)
	if w.Code != http.StatusOK {
		t.Fatalf("trend err status=%d", w.Code)
	}
	c, w = newCtxQuery(t, "")
	dashboardApi2.Top(c)
	if w.Code != http.StatusOK {
		t.Fatalf("top err status=%d", w.Code)
	}
	c, w = newCtxQuery(t, "")
	dashboardApi2.Category(c)
	if w.Code != http.StatusOK {
		t.Fatalf("category err status=%d", w.Code)
	}
}

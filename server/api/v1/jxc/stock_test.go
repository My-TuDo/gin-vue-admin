package jxc

import (
	"net/http"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
)

// stockapi 被测的 StockApi 实例
var stockapi = &StockApi{}

// stockApiTestDB 库存模块测试库
func stockApiTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
		&jxc.SaleOrder{}, &jxc.SaleItem{},
	)
}

// seedStockApi 前置：仓库 + 商品/SKU + 库存 10 件
func seedStockApi(t *testing.T) {
	t.Helper()
	db := global.GVA_DB
	db.Create(&jxc.Warehouse{Code: "CK001", Name: "总仓", Status: 1})
	g := jxc.Goods{Code: "SP001", Name: "测试商品", Status: 1}
	db.Create(&g)
	db.Create(&jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-M-红", Color: "红", Size: "M", CostPrice: 8, SalePrice: 15, Status: 1})
	db.Create(&jxc.Stock{WarehouseID: 1, SkuID: 1, Quantity: 10, WarningQuantity: 15})
}

// TestApiStockFlow 测试库存查询 + 直接入库 + 直接出库
func TestApiStockFlow(t *testing.T) {
	stockApiTestDB(t)
	seedStockApi(t)

	// 库存列表
	c, w := newCtxQuery(t, "page=1&pageSize=10")
	stockapi.GetStockPage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("stock page status=%d", w.Code)
	}
	// 预警过滤
	c, w = newCtxQuery(t, "page=1&pageSize=10&lowStock=true")
	stockapi.GetStockPage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("lowStock status=%d", w.Code)
	}
	// 流水（空）
	c, w = newCtxQuery(t, "page=1&pageSize=10")
	stockapi.GetStockLogPage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("log status=%d", w.Code)
	}
	// 直接入库
	_, rb := doReq(t, stockapi.DirectIn, http.MethodPost, `{"warehouseId":1,"skuId":1,"qty":5,"remark":"赠品"}`)
	if rb.Code != 0 {
		t.Fatalf("directIn code=%d msg=%s", rb.Code, rb.Msg)
	}
	var stock jxc.Stock
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = 1").First(&stock)
	if stock.Quantity != 15 {
		t.Errorf("入库后应为 15, got %d", stock.Quantity)
	}
	// 流水有记录
	c, w = newCtxQuery(t, "page=1&pageSize=10&businessType=direct_in")
	stockapi.GetStockLogPage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("log2 status=%d", w.Code)
	}
	// 直接出库
	_, rb = doReq(t, stockapi.DirectOut, http.MethodPost, `{"warehouseId":1,"skuId":1,"qty":4,"remark":"损耗"}`)
	if rb.Code != 0 {
		t.Fatalf("directOut code=%d msg=%s", rb.Code, rb.Msg)
	}
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = 1").First(&stock)
	if stock.Quantity != 11 {
		t.Errorf("出库后应为 11, got %d", stock.Quantity)
	}
}

// TestApiStockBadBody 测试绑定失败分支
func TestApiStockBadBody(t *testing.T) {
	stockApiTestDB(t)
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
		body string
	}{
		{"DirectIn", stockapi.DirectIn, `{"warehouseId":1,"skuId":1}`},       // 缺 qty
		{"DirectIn_ZeroQty", stockapi.DirectIn, `{"warehouseId":1,"skuId":1,"qty":0}`},
		{"DirectIn_NoWh", stockapi.DirectIn, `{"skuId":1,"qty":1}`},
		{"DirectOut", stockapi.DirectOut, `{invalid`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, tc.body)
			if rb.Code == 0 {
				t.Error("绑定失败应返回非 0 code")
			}
		})
	}
}

// TestApiStockErrors 测试表缺失错误分支
func TestApiStockErrors(t *testing.T) {
	stockApiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE stock_log")
	global.GVA_DB.Exec("DROP TABLE stock")

	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetStockPage", stockapi.GetStockPage},
		{"GetStockLogPage", stockapi.GetStockLogPage},
		{"DirectIn", stockapi.DirectIn},
		{"DirectOut", stockapi.DirectOut},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "GetStockPage" || tc.name == "GetStockLogPage" {
				c, w := newCtxQuery(t, "page=1&pageSize=10")
				tc.call(c)
				if w.Code != http.StatusOK {
					t.Errorf("%s 表缺失时返回非200", tc.name)
				}
				return
			}
			_, rb := doReq(t, tc.call, http.MethodPost, `{"warehouseId":1,"skuId":1,"qty":1}`)
			if rb.Code == 0 {
				t.Errorf("%s 表缺失时应报错", tc.name)
			}
		})
	}
}

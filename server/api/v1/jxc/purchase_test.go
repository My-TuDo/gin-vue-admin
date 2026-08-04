package jxc

import (
	"net/http"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
)

// papi 被测的 PurchaseApi 实例
var papi = &PurchaseApi{}

// purchaseApiTestDB 内存库 + 基础/商品/采购/库存全表
func purchaseApiTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
	)
}

// seedPurchase 构造采购所需前置数据（供应商/仓库/商品/SKU）
func seedPurchase(t *testing.T) {
	t.Helper()
	db := global.GVA_DB
	db.Create(&jxc.Supplier{Code: "GYS001", Name: "测试供应商"})
	db.Create(&jxc.Warehouse{Code: "CK001", Name: "总仓"})
	g := jxc.Goods{Code: "SP001", Name: "测试商品"}
	db.Create(&g)
	db.Create(&jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-M-红", Color: "红", Size: "M"})
}

const purchaseBody = `{"supplierId":1,"warehouseId":1,"remark":"测试","items":[{"skuId":1,"qty":10,"price":10}]}`

// TestApiPurchaseFlow 测试采购单创建→查询→审核→入库全流程
func TestApiPurchaseFlow(t *testing.T) {
	purchaseApiTestDB(t)
	seedPurchase(t)

	_, rb := doReq(t, papi.CreatePurchase, http.MethodPost, purchaseBody)
	if rb.Code != 0 {
		t.Fatalf("create code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 分页列表
	c, w := newCtxQuery(t, "page=1&pageSize=10")
	papi.GetPurchasePage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("page status=%d", w.Code)
	}
	// 详情
	c, w = newCtxQuery(t, "id=1")
	papi.GetPurchaseDetail(c)
	if w.Code != http.StatusOK {
		t.Fatalf("detail status=%d", w.Code)
	}
	// 审核
	_, rb = doReq(t, papi.AuditPurchase, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("audit code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 入库
	_, rb = doReq(t, papi.StockIn, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("stockIn code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 入库后状态验证：库存已建 + 流水已写
	var stock jxc.Stock
	if err := global.GVA_DB.Where("warehouse_id = 1 AND sku_id = 1").First(&stock).Error; err != nil {
		t.Fatalf("入库后库存未创建: %v", err)
	}
	if stock.Quantity != 10 {
		t.Errorf("库存应为 10, got %d", stock.Quantity)
	}
}

// TestApiPurchase_AuditCancel 测试审核后取消
func TestApiPurchase_AuditCancel(t *testing.T) {
	purchaseApiTestDB(t)
	seedPurchase(t)
	_, rb := doReq(t, papi.CreatePurchase, http.MethodPost, purchaseBody)
	if rb.Code != 0 {
		t.Fatalf("create code=%d", rb.Code)
	}
	_, rb = doReq(t, papi.AuditPurchase, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("audit code=%d", rb.Code)
	}
	_, rb = doReq(t, papi.CancelPurchase, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("cancel code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// TestApiPurchase_UpdateDelete 测试更新/删除
func TestApiPurchase_UpdateDelete(t *testing.T) {
	purchaseApiTestDB(t)
	seedPurchase(t)
	_, rb := doReq(t, papi.CreatePurchase, http.MethodPost, purchaseBody)
	if rb.Code != 0 {
		t.Fatalf("create code=%d", rb.Code)
	}
	_, rb = doReq(t, papi.UpdatePurchase, http.MethodPut,
		`{"ID":1,"supplierId":1,"warehouseId":1,"items":[{"skuId":1,"qty":5,"price":20}]}`)
	if rb.Code != 0 {
		t.Fatalf("update code=%d msg=%s", rb.Code, rb.Msg)
	}
	_, rb = doReq(t, papi.DeletePurchase, http.MethodDelete, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("delete code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// TestApiPurchaseBadBody 测试绑定失败分支
func TestApiPurchaseBadBody(t *testing.T) {
	purchaseApiTestDB(t)
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
		body string
	}{
		{"CreatePurchase", papi.CreatePurchase, `{invalid`},
		{"UpdatePurchase", papi.UpdatePurchase, `{invalid`},
		{"DeletePurchase", papi.DeletePurchase, `{invalid`},
		{"AuditPurchase", papi.AuditPurchase, `{invalid`},
		{"CancelPurchase", papi.CancelPurchase, `{invalid`},
		{"StockIn", papi.StockIn, `{invalid`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, tc.body)
			if rb.Code == 0 {
				t.Error("绑定失败应返回非 0 code")
			}
		})
	}
}

// TestApiPurchaseErrors 测试服务层错误分支（DROP 表后各操作报错）
func TestApiPurchaseErrors(t *testing.T) {
	purchaseApiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE purchase_item")
	global.GVA_DB.Exec("DROP TABLE purchase_order")
	global.GVA_DB.Exec("DROP TABLE stock_log")
	global.GVA_DB.Exec("DROP TABLE stock")

	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetPurchasePage", papi.GetPurchasePage},
		{"GetPurchaseDetail", papi.GetPurchaseDetail},
		{"CreatePurchase", papi.CreatePurchase},
		{"UpdatePurchase", papi.UpdatePurchase},
		{"DeletePurchase", papi.DeletePurchase},
		{"AuditPurchase", papi.AuditPurchase},
		{"CancelPurchase", papi.CancelPurchase},
		{"StockIn", papi.StockIn},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, purchaseBody)
			if rb.Code == 0 {
				t.Errorf("%s 在表缺失时应报错, got code=0", tc.name)
			}
		})
	}
}

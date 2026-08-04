package jxc

import (
	"net/http"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
)

// sapi 被测的 SaleApi 实例
var sapi = &SaleApi{}

// saleApiTestDB 内存库 + 全表
func saleApiTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
		&jxc.SaleOrder{}, &jxc.SaleItem{},
	)
}

// seedSale 前置：客户/仓库/商品/SKU/库存 10 件
func seedSale(t *testing.T) {
	t.Helper()
	db := global.GVA_DB
	db.Create(&jxc.Customer{Code: "KH001", Name: "测试客户", Status: 1})
	db.Create(&jxc.Warehouse{Code: "CK001", Name: "总仓", Status: 1})
	g := jxc.Goods{Code: "SP001", Name: "测试商品", Status: 1}
	db.Create(&g)
	db.Create(&jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-M-红", Color: "红", Size: "M", CostPrice: 8, SalePrice: 15, Status: 1})
	db.Create(&jxc.Stock{WarehouseID: 1, SkuID: 1, Quantity: 10})
}

const saleBody = `{"orderType":1,"customerId":1,"warehouseId":1,"remark":"测试","items":[{"skuId":1,"qty":2,"price":15}]}`

// TestApiSaleFlow 测试销售单创建→出库全流程
func TestApiSaleFlow(t *testing.T) {
	saleApiTestDB(t)
	seedSale(t)

	_, rb := doReq(t, sapi.CreateSaleOrder, http.MethodPost, saleBody)
	if rb.Code != 0 {
		t.Fatalf("create code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 分页
	c, w := newCtxQuery(t, "page=1&pageSize=10")
	sapi.GetSalePage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("page status=%d", w.Code)
	}
	// 详情
	c, w = newCtxQuery(t, "id=1")
	sapi.GetSaleDetail(c)
	if w.Code != http.StatusOK {
		t.Fatalf("detail status=%d", w.Code)
	}
	// 出库
	_, rb = doReq(t, sapi.ConfirmOut, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("out code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 验证库存 10-2=8、锁定 0
	var stock jxc.Stock
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = 1").First(&stock)
	if stock.Quantity != 8 || stock.LockQuantity != 0 {
		t.Errorf("出库后库存应为 qty=8 lock=0, got qty=%d lock=%d", stock.Quantity, stock.LockQuantity)
	}
}

// TestApiSale_Return 测试退货流程
func TestApiSale_Return(t *testing.T) {
	saleApiTestDB(t)
	seedSale(t)
	_, rb := doReq(t, sapi.CreateSaleOrder, http.MethodPost, saleBody)
	if rb.Code != 0 {
		t.Fatalf("create code=%d", rb.Code)
	}
	_, rb = doReq(t, sapi.ConfirmOut, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("out code=%d", rb.Code)
	}
	// 退货单（关联原单 1）
	_, rb = doReq(t, sapi.CreateSaleOrder, http.MethodPost, `{"orderType":2,"originalOrderId":1,"warehouseId":1,"items":[{"skuId":1,"qty":1,"price":15}]}`)
	if rb.Code != 0 {
		t.Fatalf("return create code=%d msg=%s", rb.Code, rb.Msg)
	}
	_, rb = doReq(t, sapi.ConfirmReturn, http.MethodPut, `{"id":2}`)
	if rb.Code != 0 {
		t.Fatalf("return confirm code=%d msg=%s", rb.Code, rb.Msg)
	}
	var stock jxc.Stock
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = 1").First(&stock)
	if stock.Quantity != 9 {
		t.Errorf("退货后库存应为 9, got %d", stock.Quantity)
	}
}

// TestApiSale_CancelUpdateDelete 测试取消/更新/删除
func TestApiSale_CancelUpdateDelete(t *testing.T) {
	saleApiTestDB(t)
	seedSale(t)
	_, rb := doReq(t, sapi.CreateSaleOrder, http.MethodPost, saleBody)
	if rb.Code != 0 {
		t.Fatalf("create code=%d", rb.Code)
	}
	// 取消（锁释放）
	_, rb = doReq(t, sapi.CancelSaleOrder, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("cancel code=%d", rb.Code)
	}
	var stock jxc.Stock
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = 1").First(&stock)
	if stock.LockQuantity != 0 {
		t.Errorf("取消后锁定应释放, got %d", stock.LockQuantity)
	}
	// 已取消不可出库
	_, rb = doReq(t, sapi.ConfirmOut, http.MethodPut, `{"id":1}`)
	if rb.Code == 0 {
		t.Error("已取消单出库应失败")
	}
	// 更新（重建明细并重新锁定）
	_, rb = doReq(t, sapi.CreateSaleOrder, http.MethodPost, saleBody)
	if rb.Code != 0 {
		t.Fatalf("create2 code=%d", rb.Code)
	}
	_, rb = doReq(t, sapi.UpdateSaleOrder, http.MethodPut, `{"ID":2,"orderType":1,"warehouseId":1,"items":[{"skuId":1,"qty":3,"price":15}]}`)
	if rb.Code != 0 {
		t.Fatalf("update code=%d msg=%s", rb.Code, rb.Msg)
	}
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = 1").First(&stock)
	if stock.LockQuantity != 3 {
		t.Errorf("更新后锁定应为 3, got %d", stock.LockQuantity)
	}
	// 删除（释放锁）
	_, rb = doReq(t, sapi.DeleteSaleOrder, http.MethodDelete, `{"id":2}`)
	if rb.Code != 0 {
		t.Fatalf("delete code=%d msg=%s", rb.Code, rb.Msg)
	}
	global.GVA_DB.Where("warehouse_id = 1 AND sku_id = 1").First(&stock)
	if stock.LockQuantity != 0 {
		t.Errorf("删除后锁定应释放, got %d", stock.LockQuantity)
	}
}

// TestApiSaleBadBody 测试绑定失败分支
func TestApiSaleBadBody(t *testing.T) {
	saleApiTestDB(t)
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"CreateSaleOrder", sapi.CreateSaleOrder},
		{"UpdateSaleOrder", sapi.UpdateSaleOrder},
		{"DeleteSaleOrder", sapi.DeleteSaleOrder},
		{"CancelSaleOrder", sapi.CancelSaleOrder},
		{"ConfirmOut", sapi.ConfirmOut},
		{"ConfirmReturn", sapi.ConfirmReturn},
		{"ConfirmExchange", sapi.ConfirmExchange},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{invalid`)
			if rb.Code == 0 {
				t.Error("绑定失败应返回非 0 code")
			}
		})
	}
}

// TestApiSaleErrors 测试服务层错误分支（DROP 表后）
func TestApiSaleErrors(t *testing.T) {
	saleApiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE sale_item")
	global.GVA_DB.Exec("DROP TABLE sale_order")
	global.GVA_DB.Exec("DROP TABLE stock_log")
	global.GVA_DB.Exec("DROP TABLE stock")

	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetSalePage", sapi.GetSalePage},
		{"GetSaleDetail", sapi.GetSaleDetail},
		{"CreateSaleOrder", sapi.CreateSaleOrder},
		{"UpdateSaleOrder", sapi.UpdateSaleOrder},
		{"DeleteSaleOrder", sapi.DeleteSaleOrder},
		{"CancelSaleOrder", sapi.CancelSaleOrder},
		{"ConfirmOut", sapi.ConfirmOut},
		{"ConfirmReturn", sapi.ConfirmReturn},
		{"ConfirmExchange", sapi.ConfirmExchange},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, saleBody)
			if rb.Code == 0 {
				t.Errorf("%s 在表缺失时应报错, got code=0", tc.name)
			}
		})
	}
}

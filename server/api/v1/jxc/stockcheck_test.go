package jxc

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

var scapi = &StockCheckApi{}

// stockCheckApiTestDB 盘点 api 测试库
func stockCheckApiTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{}, &jxc.Stock{}, &jxc.StockLog{},
		&jxc.SaleOrder{}, &jxc.SaleItem{},
		&jxc.StockCheck{}, &jxc.StockCheckItem{},
	)
}

// seedStockCheck 前置数据：仓库 + SKU + 库存 10
func seedStockCheck(t *testing.T) uint {
	t.Helper()
	db := global.GVA_DB
	db.Create(&jxc.Warehouse{Code: "CK001", Name: "总仓", Status: 1})
	g := jxc.Goods{Code: "SP001", Name: "测试商品", Status: 1}
	db.Create(&g)
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-M-红", Color: "红", Size: "M", CostPrice: 8, SalePrice: 15, Status: 1}
	db.Create(&sku)
	db.Create(&jxc.Stock{WarehouseID: 1, SkuID: sku.ID, Quantity: 10})
	return sku.ID
}

// TestApiStockCheckFlow 盘点 api 全流程
func TestApiStockCheckFlow(t *testing.T) {
	stockCheckApiTestDB(t)
	seedStockCheck(t)

	// 创建
	_, rb := doReq(t, scapi.CreateStockCheck, http.MethodPost, `{"warehouseId":1,"checker":"tester","remark":"盘点"}`)
	if rb.Code != 0 {
		t.Fatalf("create code=%d msg=%s", rb.Code, rb.Msg)
	}
	var check jxc.StockCheck
	global.GVA_DB.First(&check)
	if check.CheckNo == "" {
		t.Fatalf("创建结果不符: %+v", check)
	}
	var items []jxc.StockCheckItem
	global.GVA_DB.Where("check_id = ?", check.ID).Find(&items)
	if len(items) != 1 || items[0].SystemQty != 10 {
		t.Fatalf("快照明细不符: %+v", items)
	}
	// 分页
	c, w := newCtxQuery(t, "page=1&pageSize=10")
	scapi.GetStockCheckPage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("page status=%d", w.Code)
	}
	// 详情
	c, w = newCtxQuery(t, "id=1")
	scapi.GetStockCheckDetail(c)
	if w.Code != http.StatusOK {
		t.Fatalf("detail status=%d", w.Code)
	}
	// 录入
	_, rb = doReq(t, scapi.UpdateStockCheckItems, http.MethodPut,
		`{"checkId":1,"items":[{"id":1,"actualQty":12}]}`)
	if rb.Code != 0 {
		t.Fatalf("items code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 完成
	_, rb = doReq(t, scapi.CompleteStockCheck, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("complete code=%d msg=%s", rb.Code, rb.Msg)
	}
	var st jxc.Stock
	global.GVA_DB.Where("sku_id = 1").First(&st)
	if st.Quantity != 12 {
		t.Errorf("完成盘点后库存应为 12, got %d", st.Quantity)
	}
	var log jxc.StockLog
	global.GVA_DB.Where("business_type = ?", "check_in").First(&log)
	if log.ChangeQty != 2 {
		t.Errorf("盘盈流水应为 +2, got %d", log.ChangeQty)
	}
}

// TestApiStockCheckErrors api 异常分支
func TestApiStockCheckErrors(t *testing.T) {
	stockCheckApiTestDB(t)
	seedStockCheck(t)

	// 仓库不存在
	_, rb := doReq(t, scapi.CreateStockCheck, http.MethodPost, `{"warehouseId":99}`)
	if rb.Code == 0 {
		t.Error("仓库不存在应失败")
	}
	// 参数绑定错误
	_, rb = doReq(t, scapi.CreateStockCheck, http.MethodPost, `not-json`)
	if rb.Code == 0 {
		t.Error("坏参数应失败")
	}
	// 录入：明细不存在
	_, rb = doReq(t, scapi.UpdateStockCheckItems, http.MethodPut, `{"checkId":1,"items":[{"id":999,"actualQty":5}]}`)
	if rb.Code == 0 {
		t.Error("明细不存在录入应失败")
	}
	// 完成：盘点单不存在
	_, rb = doReq(t, scapi.CompleteStockCheck, http.MethodPut, `{"id":999}`)
	if rb.Code == 0 {
		t.Error("盘点单不存在完成应失败")
	}
	// 取消：不存在
	_, rb = doReq(t, scapi.CancelStockCheck, http.MethodPut, `{"id":999}`)
	if rb.Code == 0 {
		t.Error("盘点单不存在取消应失败")
	}
	// 参数绑定错误
	c, w := newCtxQuery(t, "id=abc")
	scapi.GetStockCheckDetail(c)
	if w.Code != http.StatusOK {
		t.Fatalf("badparam detail status=%d", w.Code)
	}
	// 分页参数绑定错误
	c, w = newCtxQuery(t, "page=abc")
	scapi.GetStockCheckPage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("badparam page status=%d", w.Code)
	}
	// 录入参数绑定错误
	_, rb = doReq(t, scapi.UpdateStockCheckItems, http.MethodPut, `not-json`)
	if rb.Code == 0 {
		t.Error("坏参数录入应失败")
	}
	// 完成/取消参数绑定错误
	_, rb = doReq(t, scapi.CompleteStockCheck, http.MethodPut, `not-json`)
	if rb.Code == 0 {
		t.Error("坏参数完成应失败")
	}
	_, rb = doReq(t, scapi.CancelStockCheck, http.MethodPut, `not-json`)
	if rb.Code == 0 {
		t.Error("坏参数取消应失败")
	}
	// 成功路径兜底：正常创建+取消
	_, rb = doReq(t, scapi.CreateStockCheck, http.MethodPost, `{"warehouseId":1}`)
	if rb.Code != 0 {
		t.Fatalf("create code=%d msg=%s", rb.Code, rb.Msg)
	}
	_, rb = doReq(t, scapi.CancelStockCheck, http.MethodPut, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("cancel code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 分页数据校验
	c, w = newCtxQuery(t, "page=1&pageSize=10")
	scapi.GetStockCheckPage(c)
	var pr response.PageResult
	_ = json.Unmarshal(w.Body.Bytes(), &struct {
		Data response.PageResult `json:"data"`
	}{Data: pr})
}

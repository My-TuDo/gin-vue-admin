package jxc

import (
	"net/http"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

var posApi2 = &PosScanApi{}

// TestApiPosScan 扫码枪 API：上架 / 待处理列表 / 确认
func TestApiPosScan(t *testing.T) {
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Goods{}, &jxc.GoodsSku{}, &jxc.PosScan{},
	)
	db := global.GVA_DB

	cat := jxc.GoodsCategory{Name: "扫码API分类", Sort: 1, Status: 1}
	db.Create(&cat)
	goods := jxc.Goods{Name: "扫码API商品", CategoryID: &cat.ID, Unit: "件", Status: 1}
	db.Create(&goods)
	sku := jxc.GoodsSku{GoodsID: goods.ID, SkuCode: "API-POS", Barcode: "6901112223334", Color: "红", Size: "S", SalePrice: 15, Status: 1}
	db.Create(&sku)

	// 1. 上架成功（按条码）
	_, rb := doReq(t, posApi2.Scan, http.MethodPost, `{"barcode":"6901112223334","qty":2}`)
	if rb.Code != 0 {
		t.Fatalf("上架 code=%d msg=%s", rb.Code, rb.Msg)
	}
	var scan jxc.PosScan
	global.GVA_DB.Where("sku_id = ? AND status = 0", sku.ID).First(&scan)
	if scan.Qty != 2 {
		t.Fatalf("上架数量异常: %d", scan.Qty)
	}

	// 2. 上架失败（条码不存在）
	_, rb = doReq(t, posApi2.Scan, http.MethodPost, `{"barcode":"9999999999999","qty":1}`)
	if rb.Code == 0 {
		t.Fatal("条码不存在应失败")
	}

	// 3. 上架失败（空参数）
	_, rb = doReq(t, posApi2.Scan, http.MethodPost, `{}`)
	if rb.Code == 0 {
		t.Fatal("空参数应失败")
	}

	// 4. 待处理列表
	_, rb = doReq(t, posApi2.ListPending, http.MethodGet, "")
	if rb.Code != 0 {
		t.Fatalf("列表 code=%d msg=%s", rb.Code, rb.Msg)
	}

	// 5. 确认成功
	_, rb = doReq(t, posApi2.Confirm, http.MethodPut, `{"ids":[1]}`)
	if rb.Code != 0 {
		t.Fatalf("确认 code=%d msg=%s", rb.Code, rb.Msg)
	}

	// 6. 确认空 ids 失败
	_, rb = doReq(t, posApi2.Confirm, http.MethodPut, `{"ids":[]}`)
	if rb.Code == 0 {
		t.Fatal("空 ids 应失败")
	}

	// 7. 确认参数格式错误
	_, rb = doReq(t, posApi2.Confirm, http.MethodPut, `{"bad`)
	if rb.Code == 0 {
		t.Fatal("坏 JSON 应失败")
	}
}

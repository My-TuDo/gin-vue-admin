package jxc

import (
	"encoding/json"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// TestApiGetSkuByBarcode api 层：按条码查 SKU 成功/未命中/空参
func TestApiGetSkuByBarcode(t *testing.T) {
	testutil.NewMemoryDB(t,
		&jxc.Warehouse{}, &jxc.Goods{}, &jxc.GoodsSku{}, &jxc.Stock{},
	)
	global.GVA_DB.Create(&jxc.Warehouse{Code: "W001", Name: "主仓"})
	g := jxc.Goods{Code: "SP001", Name: "联测上衣", Unit: "件", Status: 1}
	global.GVA_DB.Create(&g)
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-白-M", Barcode: "6901234567890", Color: "白", Size: "M", Status: 1}
	global.GVA_DB.Create(&sku)
	global.GVA_DB.Create(&jxc.Stock{WarehouseID: 1, SkuID: sku.ID, Quantity: 10, LockQuantity: 2})

	// 命中
	c, w := newCtxQuery(t, "barcode=6901234567890")
	gapi.GetSkuByBarcode(c)
	rb := parseBody(t, w)
	if rb.Code != 0 {
		t.Fatalf("命中失败 code=%d msg=%s", rb.Code, rb.Msg)
	}
	raw, _ := json.Marshal(rb.Data)
	var got jxc.GoodsSku
	if err := json.Unmarshal(raw, &got); err != nil || got.ID != sku.ID {
		t.Fatalf("data 解析失败: %v %s", err, string(raw))
	}
	if len(got.Stocks) != 1 || got.Stocks[0].Available != 8 {
		t.Fatalf("stocks 不符: %+v", got.Stocks)
	}

	// 未命中
	c2, w2 := newCtxQuery(t, "barcode=999999999")
	gapi.GetSkuByBarcode(c2)
	rb2 := parseBody(t, w2)
	if rb2.Code == 0 {
		t.Fatal("未命中应失败")
	}

	// 空参
	c3, w3 := newCtxQuery(t, "")
	gapi.GetSkuByBarcode(c3)
	rb3 := parseBody(t, w3)
	if rb3.Code == 0 {
		t.Fatal("空参应失败")
	}
}

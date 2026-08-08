package jxc

import (
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// ========== 按条码查 SKU（小程序扫码查库存） ==========

// newBarcodeFixture 建 2 仓 + 商品 + SKU（带条码）+ 两仓库存
func newBarcodeFixture(t *testing.T) (warehouse1, warehouse2 jxc.Warehouse, sku jxc.GoodsSku) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.Warehouse{}, &jxc.Goods{}, &jxc.GoodsSku{}, &jxc.Stock{},
	)
	global.GVA_DB.Create(&jxc.Warehouse{Code: "W001", Name: "主仓"})
	global.GVA_DB.Create(&jxc.Warehouse{Code: "W002", Name: "分仓"})
	global.GVA_DB.First(&warehouse1, "name = ?", "主仓")
	global.GVA_DB.First(&warehouse2, "name = ?", "分仓")

	g := jxc.Goods{Code: "SP001", Name: "联测上衣", Unit: "件", Status: 1}
	global.GVA_DB.Create(&g)
	sku = jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-白-M", Barcode: "6901234567890", Color: "白", Size: "M",
		CostPrice: 10, SalePrice: 20, SafeStock: 3, Status: 1}
	global.GVA_DB.Create(&sku)

	global.GVA_DB.Create(&jxc.Stock{WarehouseID: warehouse1.ID, SkuID: sku.ID, Quantity: 10, LockQuantity: 2})
	global.GVA_DB.Create(&jxc.Stock{WarehouseID: warehouse2.ID, SkuID: sku.ID, Quantity: 5, LockQuantity: 0})
	return
}

// TestGetSkuByBarcode_ByBarcode 按条码命中，含多仓库存与可售数
func TestGetSkuByBarcode_ByBarcode(t *testing.T) {
	w1, w2, sku := newBarcodeFixture(t)
	got, err := gsvc.GetSkuByBarcode(ctx, "6901234567890")
	if err != nil {
		t.Fatalf("按条码查询失败: %v", err)
	}
	if got.ID != sku.ID || got.SkuCode != "SP001-白-M" || got.Goods == nil || got.Goods.Name != "联测上衣" {
		t.Fatalf("SKU 信息不符: %+v", got)
	}
	if len(got.Stocks) != 2 {
		t.Fatalf("应返回 2 仓库存, got %d", len(got.Stocks))
	}
	m := map[uint]int{}
	for _, s := range got.Stocks {
		m[s.WarehouseID] = s.Available
		if s.WarehouseName == "" {
			t.Fatalf("仓库名称缺失: %+v", s)
		}
	}
	if m[w1.ID] != 8 || m[w2.ID] != 5 {
		t.Fatalf("可售数量不符: %+v (期望 主仓8 分仓5)", got.Stocks)
	}
}

// TestGetSkuByBarcode_BySkuCode 按 SKU 编码命中
func TestGetSkuByBarcode_BySkuCode(t *testing.T) {
	_, _, sku := newBarcodeFixture(t)
	got, err := gsvc.GetSkuByBarcode(ctx, sku.SkuCode)
	if err != nil || got.ID != sku.ID {
		t.Fatalf("按编码查询失败: got=%+v err=%v", got, err)
	}
}

// TestGetSkuByBarcode_NotFound 未命中报错
func TestGetSkuByBarcode_NotFound(t *testing.T) {
	newBarcodeFixture(t)
	_, err := gsvc.GetSkuByBarcode(ctx, "9999999999999")
	if err == nil || !strings.Contains(err.Error(), "未找到") {
		t.Fatalf("未命中应报错, got %v", err)
	}
}

// TestGetSkuByBarcode_EmptyKeyword 空参数报错
func TestGetSkuByBarcode_EmptyKeyword(t *testing.T) {
	newBarcodeFixture(t)
	if _, err := gsvc.GetSkuByBarcode(ctx, "  "); err == nil {
		t.Fatal("空参数应报错")
	}
}

// TestGetSkuByBarcode_NoStock 无库存记录返回空 stocks
func TestGetSkuByBarcode_NoStock(t *testing.T) {
	w1, _, _ := newBarcodeFixture(t)
	// 只留主仓无库存：删掉库存记录后新建一个无库存 SKU
	g := jxc.Goods{Code: "SP002", Name: "无库存商品", Unit: "件", Status: 1}
	global.GVA_DB.Create(&g)
	sku2 := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP002-黑-L", Barcode: "6902222222222", Color: "黑", Size: "L", Status: 1}
	global.GVA_DB.Create(&sku2)
	global.GVA_DB.Exec("DELETE FROM stock WHERE warehouse_id = ?", w1.ID)
	got, err := gsvc.GetSkuByBarcode(ctx, "6902222222222")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(got.Stocks) != 0 {
		t.Fatalf("无库存应返回空列表, got %+v", got.Stocks)
	}
}

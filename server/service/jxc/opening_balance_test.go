package jxc

import (
	"context"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 期初建账：新店开张时账上没有结存，但仓里实际有货。
//
// 这条路径必须成立，否则新客户无法开始用——旧实现（盘点只取有结存的行）
// 会让"期初"变成不可能操作。现在盘点覆盖全部在售 SKU，账面全为 0，
// 录实盘 → 盘盈 → 库存就有了。
func TestStockCheck_OpeningBalance(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db) // 不放任何库存
	cs := NewStockCheckService()
	ls := new(LedgerService)
	ctx := context.Background()

	if q, _ := balanceOf(t, db, wh, sku); q != 0 {
		t.Fatalf("初始结存应为 0，实际 %d", q)
	}

	check, err := cs.Create(ctx, wh, "期初建账")
	if err != nil {
		t.Fatalf("建盘点单失败: %v", err)
	}
	if len(check.Items) != 1 {
		t.Fatalf("盘点单应包含全部在售 SKU（1 个），实际 %d 行", len(check.Items))
	}
	if check.Items[0].BookQty != 0 {
		t.Errorf("期初账面应为 0，实际 %d", check.Items[0].BookQty)
	}

	// 实盘 25 件，成本需人工给（账上没有成本可推）——这里用 0 成本验证数量正确
	if err := cs.UpdateItems(ctx, check.ID, []StockCountInput{{ID: check.Items[0].ID, ActualQty: 25}}); err != nil {
		t.Fatalf("录入实盘失败: %v", err)
	}
	if _, err := cs.Post(ctx, check.ID, "tester", time.Now()); err != nil {
		t.Fatalf("完成盘点失败: %v", err)
	}

	if q, _ := balanceOf(t, db, wh, sku); q != 25 {
		t.Errorf("期初录入后应有 25 件，实际 %d", q)
	}
	var cnt int64
	db.Model(&jxc.StockMove{}).Where("source_type = ?", jxc.MoveSourceCheckGain).Count(&cnt)
	if cnt != 1 {
		t.Errorf("应有一条盘盈账本行，实际 %d", cnt)
	}
	mustInvariant(t, ls)
}

// 账面为 0 但实际有货（漏记入库）也必须能被盘出来
func TestStockCheck_ZeroBookStillCountable(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 5, 10)
	// 另一个 SKU 没有任何库存
	g := jxc.Goods{Code: "G2", Name: "无库存商品", Status: 1}
	db.Create(&g)
	sku2 := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SKU2", Color: "白", Size: "L", Status: 1}
	db.Create(&sku2)

	cs := NewStockCheckService()
	ctx := context.Background()
	check, err := cs.Create(ctx, wh, "")
	if err != nil {
		t.Fatalf("建单失败: %v", err)
	}
	if len(check.Items) != 2 {
		t.Fatalf("应覆盖 2 个 SKU，实际 %d", len(check.Items))
	}

	var target *jxc.StockCheckItem
	for i := range check.Items {
		if check.Items[i].SkuID == sku2.ID {
			target = &check.Items[i]
		}
	}
	if target == nil {
		t.Fatal("无库存 SKU 未被纳入盘点")
	}
	// 盘出 3 件（漏记入库）
	if err := cs.UpdateItems(ctx, check.ID, []StockCountInput{{ID: target.ID, ActualQty: 3}}); err != nil {
		t.Fatalf("录入失败: %v", err)
	}
	if _, err := cs.Post(ctx, check.ID, "t", time.Now()); err != nil {
		t.Fatalf("完成失败: %v", err)
	}
	if q, _ := balanceOf(t, db, wh, sku2.ID); q != 3 {
		t.Errorf("盘盈后应为 3 件，实际 %d", q)
	}
	_ = money.FromYuan(0)
}

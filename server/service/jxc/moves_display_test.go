package jxc

import (
	"context"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 流水展示语义的回归测试。
//
// 背景：账本行的 amount 是有符号的（入正出负），这是不变量成立的前提。
// 但把它直接当"金额"展示，出库行会变成 -264.00，被读成"卖亏了 264 元"——
// 那不是亏损，是成本从存货变成了销货成本。
//
// 所以展示层（MoveRow）额外给 Cost（恒正）与 SourceNo（人能读的单号）。
// 这组测试钉住三件事：Cost 非负、SourceNo 能回溯到单据、商品名不为空。
func TestMoves_DisplaySemantics(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps, ss, ls := newTestServices()
	ctx := context.Background()

	// 进 10 件 @88
	po, err := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
		Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 10, UnitCost: 88}}})
	if err != nil {
		t.Fatalf("建采购单失败: %v", err)
	}
	if err := ps.Approve(ctx, po.ID, "t"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if err := ps.Receive(ctx, po.ID, "t", time.Now()); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	// 卖 3 件
	so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 3, Price: 199}}})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}
	if err := ss.Ship(ctx, so.ID, "t", time.Now()); err != nil {
		t.Fatalf("出库失败: %v", err)
	}

	rows, total, err := ls.Moves(ctx, wh, sku, 1, 50)
	if err != nil {
		t.Fatalf("取流水失败: %v", err)
	}
	if total != 2 || len(rows) != 2 {
		t.Fatalf("应有 2 条流水，实际 total=%d len=%d", total, len(rows))
	}

	for _, r := range rows {
		if r.Cost < 0 {
			t.Errorf("成本金额不允许为负（展示语义）：id=%d cost=%.2f", r.ID, r.Cost)
		}
		if r.Cost != absF(r.Amount) {
			t.Errorf("Cost 应等于 |Amount|：id=%d amount=%.2f cost=%.2f", r.ID, r.Amount, r.Cost)
		}
		if r.GoodsName == "" || r.SkuCode == "" {
			t.Errorf("流水应带商品名与条码：%+v", r)
		}
		if r.SourceNo == "" {
			t.Errorf("流水应能回溯到单据号（sourceType=%s sourceId=%d）", r.SourceType, r.SourceID)
		}
	}

	// 出库行的 amount 仍是负的——账本语义不能被展示需求改写
	var outRow MoveRow
	for _, r := range rows {
		if r.SourceType == jxc.MoveSourceSaleOut {
			outRow = r
		}
	}
	if outRow.Amount >= 0 {
		t.Errorf("出库行的账本金额必须为负，实际 %.2f", outRow.Amount)
	}
	if outRow.Cost != 264 {
		t.Errorf("出库 3×88 的成本应为 264.00，实际 %.2f", outRow.Cost)
	}
	if outRow.SourceNo != so.OrderNo {
		t.Errorf("出库行单据号应为 %s，实际 %s", so.OrderNo, outRow.SourceNo)
	}
}

// TestMoves_BalanceAfterAcrossPages 结存必须按全量过账顺序算，不能只按当前页从 0 累加。
func TestMoves_BalanceAfterAcrossPages(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 10, 10)
	_, ss, ls := newTestServices()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
			Items: []jxc.SaleItem{{SkuID: sku, Qty: 1}}})
		if err != nil {
			t.Fatalf("开单失败: %v", err)
		}
		if err := ss.Ship(ctx, so.ID, "t", time.Now()); err != nil {
			t.Fatalf("出库失败: %v", err)
		}
	}

	all, total, err := ls.Moves(ctx, wh, sku, 1, 10)
	if err != nil {
		t.Fatalf("取流水失败: %v", err)
	}
	if total != 4 || len(all) != 4 {
		t.Fatalf("应有 4 条流水（1 入 + 3 出），实际 total=%d len=%d", total, len(all))
	}
	// 最新在前：出库后结存 7/8/9/10（最后一笔是最早的入库）
	want := []int{7, 8, 9, 10}
	for i, r := range all {
		if r.BalanceAfter == nil {
			t.Fatalf("流水 %d 缺少 balanceAfter", r.ID)
		}
		if *r.BalanceAfter != want[i] {
			t.Errorf("第 %d 条流水结存应为 %d，实际 %d", i, want[i], *r.BalanceAfter)
		}
	}
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

package jxc

import (
	"context"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 本文件是第二轮开发的分位残差回归测试。
//
// 背景：金额字段是 decimal(14,2)，单价字段是 decimal(12,2)。出库成本按移动加权
// 平均算出来再四舍五入到分，单价乘回去不一定等于原金额：
// 3 件 / 100.00 的出库行单价只能是 33.33，33.33 × 3 = 99.99。
// 红冲和"最后一次退货"如果按单价反乘，就会永久留下 1 分钱残差。

// seedThreeForHundred 铺 3 件、总成本恰好 100.00 的库存。
// 用三笔 33.33 / 33.33 / 33.34 入库，避免直接构造余额（账本必须能解释余额）。
func seedThreeForHundred(t *testing.T, wh, sku uint) {
	t.Helper()
	ls := new(LedgerService)
	now := time.Now()
	moves := []MoveInput{
		{WarehouseID: wh, SkuID: sku, Qty: 1, UnitCost: 33.33,
			SourceType: jxc.MoveSourcePurchaseIn, SourceID: 9001, SourceLineID: 1, OccurredAt: now},
		{WarehouseID: wh, SkuID: sku, Qty: 1, UnitCost: 33.33,
			SourceType: jxc.MoveSourcePurchaseIn, SourceID: 9001, SourceLineID: 2, OccurredAt: now},
		{WarehouseID: wh, SkuID: sku, Qty: 1, UnitCost: 33.34,
			SourceType: jxc.MoveSourcePurchaseIn, SourceID: 9001, SourceLineID: 3, OccurredAt: now},
	}
	if err := ls.Post(context.Background(), nil, moves); err != nil {
		t.Fatalf("铺底库存失败: %v", err)
	}
}

// TestReversal_ExactAmountNoResidual 销售红冲必须按原账本金额反向，不能按
// "单价 × 数量"反乘。否则 3 件/100.00 的单红冲后只剩 99.99。
func TestReversal_ExactAmountNoResidual(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedThreeForHundred(t, wh, sku)
	_, ss, rs, _, ls := servicesAll()
	ctx := context.Background()

	so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 3}}})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}
	if err := ss.Ship(ctx, so.ID, "t", time.Now()); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 0 || v != 0 {
		t.Fatalf("全部卖出后应精确归零，实际 %d/%s", q, v)
	}

	if _, err := rs.Reverse(ctx, jxc.ReversalSourceSale, so.ID, "录错了", "t"); err != nil {
		t.Fatalf("红冲失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 3 || v != money.FromYuan(100) {
		t.Fatalf("红冲后应精确回到 3/100.00，实际 %d/%s", q, v)
	}
	var revMove jxc.StockMove
	if err := db.Where("source_type = ?", jxc.MoveSourceReversal).Order("id").First(&revMove).Error; err != nil {
		t.Fatalf("查红冲账本行失败: %v", err)
	}
	if money.FromYuan(revMove.Amount) != money.FromYuan(100) {
		t.Errorf("红冲金额应精确抵消 100.00，实际 %.2f", revMove.Amount)
	}
	mustInvariant(t, ls)
}

// TestReturn_FinalReturnAbsorbsResidual 销售 3 件 / 成本 100.00，分三次各退 1 件：
// 前两次按 33.33 回补，最后一次必须回补 33.34，合计恰好 100.00。
func TestReturn_FinalReturnAbsorbsResidual(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedThreeForHundred(t, wh, sku)
	_, ss, _, rts, ls := servicesAll()
	ctx := context.Background()

	so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 3}}})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}
	if err := ss.Ship(ctx, so.ID, "t", time.Now()); err != nil {
		t.Fatalf("出库失败: %v", err)
	}

	wantCosts := []float64{33.33, 33.33, 33.34}
	for i, want := range wantCosts {
		ret, err := rts.Create(ctx, &jxc.SaleReturn{SaleID: so.ID,
			Items: []jxc.SaleReturnItem{{SkuID: sku, Qty: 1}}})
		if err != nil {
			t.Fatalf("第 %d 次建退货单失败: %v", i+1, err)
		}
		if err := rts.Post(ctx, ret.ID, "t", time.Now()); err != nil {
			t.Fatalf("第 %d 次退货入库失败: %v", i+1, err)
		}
		var got jxc.SaleReturn
		if err := db.First(&got, ret.ID).Error; err != nil {
			t.Fatalf("查退货单失败: %v", err)
		}
		if money.FromYuan(got.CostAmount) != money.FromYuan(want) {
			t.Errorf("第 %d 次退货成本应为 %.2f，实际 %.2f", i+1, want, got.CostAmount)
		}
	}

	if q, v := balanceOf(t, db, wh, sku); q != 3 || v != money.FromYuan(100) {
		t.Fatalf("退完后应精确回到 3/100.00，实际 %d/%s", q, v)
	}
	var itemSum float64
	if err := db.Model(&jxc.SaleReturnItem{}).
		Select("COALESCE(SUM(cost_amount),0)").Scan(&itemSum).Error; err != nil {
		t.Fatalf("汇总退货明细成本失败: %v", err)
	}
	if money.FromYuan(itemSum) != money.FromYuan(100) {
		t.Errorf("三次退货明细成本合计应为 100.00，实际 %.2f", itemSum)
	}
	mustInvariant(t, ls)
}

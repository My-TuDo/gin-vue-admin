package jxc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 红冲与退货的测试。
//
// 这两条路径是整套账本设计的"出口"：没有它们，"已过账不可变"就等于"错了没法救"。
// 所以测试重点不是接口返回码，而是**余额最终是否精确回到该有的值**。

func servicesAll() (*PurchaseService, *SaleService, *ReversalService, *ReturnService, *LedgerService) {
	return NewPurchaseService(), NewSaleService(), NewReversalService(), NewReturnService(), new(LedgerService)
}

// TestReversal_PurchaseRestoresBalance 采购整单红冲后，库存应精确归零
func TestReversal_PurchaseRestoresBalance(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps, _, rs, _, ls := servicesAll()
	ctx := context.Background()

	po, err := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
		Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 10, UnitCost: 88}}})
	if err != nil {
		t.Fatalf("建采购单失败: %v", err)
	}
	_ = ps.Approve(ctx, po.ID, "t")
	if err := ps.Receive(ctx, po.ID, "t", time.Now()); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 10 || v != money.FromYuan(880) {
		t.Fatalf("入库后应为 10/880，实际 %d/%s", q, v)
	}

	rev, err := rs.Reverse(ctx, jxc.ReversalSourcePurchase, po.ID, "供应商发错货", "tester")
	if err != nil {
		t.Fatalf("红冲失败: %v", err)
	}
	if rev.ReversalNo == "" || rev.Amount != 880 || rev.Qty != 10 {
		t.Errorf("红冲单信息不对：%+v", rev)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 0 || v != 0 {
		t.Errorf("红冲后库存应归零，实际 %d/%s", q, v)
	}

	// 原单据被标记
	var got jxc.PurchaseOrder
	db.First(&got, po.ID)
	if got.ReversedAt == nil {
		t.Error("原采购单应被标记为已红冲")
	}
	mustInvariant(t, ls)
}

// TestReversal_IdempotentAndGuards 重复红冲、非过账单据红冲都要被挡
func TestReversal_Guards(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps, _, rs, _, _ := servicesAll()
	ctx := context.Background()

	po, _ := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
		Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 1, UnitCost: 10}}})

	// 草稿状态不能红冲
	if _, err := rs.Reverse(ctx, jxc.ReversalSourcePurchase, po.ID, "试一下", "t"); err == nil ||
		!strings.Contains(err.Error(), "只有已入库") {
		t.Errorf("草稿单不应能红冲，got %v", err)
	}

	_ = ps.Approve(ctx, po.ID, "t")
	_ = ps.Receive(ctx, po.ID, "t", time.Now())
	if _, err := rs.Reverse(ctx, jxc.ReversalSourcePurchase, po.ID, "第一次", "t"); err != nil {
		t.Fatalf("首次红冲失败: %v", err)
	}
	if _, err := rs.Reverse(ctx, jxc.ReversalSourcePurchase, po.ID, "第二次", "t"); err == nil ||
		!strings.Contains(err.Error(), "已经红冲") {
		t.Errorf("重复红冲应被拒绝，got %v", err)
	}
}

// TestReversal_PurchaseAfterSold 货卖了还整单红冲，会把库存冲成负数，必须拒绝
func TestReversal_PurchaseAfterSold(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps, ss, rs, _, ls := servicesAll()
	ctx := context.Background()

	po, _ := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
		Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 4, UnitCost: 50}}})
	_ = ps.Approve(ctx, po.ID, "t")
	_ = ps.Receive(ctx, po.ID, "t", time.Now())

	// 卖掉 3 件，只剩 1 件
	so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 3, Price: 99}}})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}
	_ = ss.Ship(ctx, so.ID, "t", time.Now())

	_, err = rs.Reverse(ctx, jxc.ReversalSourcePurchase, po.ID, "想退货给供应商", "t")
	if err == nil || !strings.Contains(err.Error(), "已卖出") {
		t.Fatalf("已卖出的采购单不应能整单红冲，got %v", err)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 1 {
		t.Errorf("被拒绝后库存应仍为 1，实际 %d", q)
	}
	mustInvariant(t, ls)
}

// TestReversal_SaleRestoresStock 销售红冲：货回来，成本精确抵消
func TestReversal_SaleRestoresStock(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 10, 88)
	_, ss, rs, _, ls := servicesAll()
	ctx := context.Background()

	so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 3, Price: 199}}})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}
	_ = ss.Ship(ctx, so.ID, "t", time.Now())
	if q, _ := balanceOf(t, db, wh, sku); q != 7 {
		t.Fatalf("出库后应剩 7，实际 %d", q)
	}

	if _, err := rs.Reverse(ctx, jxc.ReversalSourceSale, so.ID, "客户取消订单", "tester"); err != nil {
		t.Fatalf("销售红冲失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 10 || v != money.FromYuan(880) {
		t.Errorf("红冲后应回到 10/880，实际 %d/%s", q, v)
	}
	mustInvariant(t, ls)
}

// TestReturn_PartialWithOriginalCost 部分退货按原单成本回补，而不是当前均价
func TestReturn_PartialWithOriginalCost(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 10, 88) // 成本 88
	_, ss, _, rts, ls := servicesAll()
	ctx := context.Background()

	so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 4, Price: 199}}})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}
	_ = ss.Ship(ctx, so.ID, "t", time.Now())

	// 关键：把库存均价改掉（模拟后来涨价进货），退货成本仍必须是 88
	seedStock2(t, db, wh, sku, 10, 120)

	ret, err := rts.Create(ctx, &jxc.SaleReturn{SaleID: so.ID,
		Items: []jxc.SaleReturnItem{{SkuID: sku, Qty: 2}}, Reason: "尺码不合适"})
	if err != nil {
		t.Fatalf("建退货单失败: %v", err)
	}
	if ret.CostAmount != 176 {
		t.Errorf("退货成本应为 2×88=176（原单成本），实际 %.2f", ret.CostAmount)
	}
	if ret.Amount != 198 {
		t.Errorf("退款金额应为 2×99=198（单价取 SKU 售价，前端传 199 不生效），实际 %.2f", ret.Amount)
	}

	if err := rts.Post(ctx, ret.ID, "t", time.Now()); err != nil {
		t.Fatalf("退货入库失败: %v", err)
	}
	// 库存回到 7 件：原 6 + 退 2（前面又进了 10 件，均价被拉高）
	var mv jxc.StockMove
	if err := db.Where("source_type = ? AND source_id = ?", jxc.MoveSourceSaleReturn, ret.ID).First(&mv).Error; err != nil {
		t.Fatalf("查退货账本行失败: %v", err)
	}
	if money.FromYuan(mv.UnitCost) != money.FromYuan(88) {
		t.Errorf("退货入库成本必须是原单 88，实际 %.2f（用当前均价就是改历史毛利）", mv.UnitCost)
	}
	mustInvariant(t, ls)
}

// TestReturn_OverReturnRejected 超退、重复入库都要被挡
func TestReturn_Guards(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 10, 50)
	_, ss, _, rts, _ := servicesAll()
	ctx := context.Background()

	so, _ := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 2, Price: 99}}})
	_ = ss.Ship(ctx, so.ID, "t", time.Now())

	// 退 3 件超过原单 2 件
	if _, err := rts.Create(ctx, &jxc.SaleReturn{SaleID: so.ID,
		Items: []jxc.SaleReturnItem{{SkuID: sku, Qty: 3}}}); err == nil ||
		!strings.Contains(err.Error(), "最多可退") {
		t.Errorf("超退应被拒绝，got %v", err)
	}

	ret, err := rts.Create(ctx, &jxc.SaleReturn{SaleID: so.ID,
		Items: []jxc.SaleReturnItem{{SkuID: sku, Qty: 2}}})
	if err != nil {
		t.Fatalf("建退货单失败: %v", err)
	}
	if err := rts.Post(ctx, ret.ID, "t", time.Now()); err != nil {
		t.Fatalf("退货入库失败: %v", err)
	}
	if err := rts.Post(ctx, ret.ID, "t", time.Now()); err == nil {
		t.Error("重复入库应被拒绝")
	}

	// 已全部退完，再退应被拒
	if _, err := rts.Create(ctx, &jxc.SaleReturn{SaleID: so.ID,
		Items: []jxc.SaleReturnItem{{SkuID: sku, Qty: 1}}}); err == nil ||
		!strings.Contains(err.Error(), "最多可退 0") {
		t.Errorf("已退完还需退应被拒绝，got %v", err)
	}
}

// seedStock2 再进一批货（用于改变移动均价）
func seedStock2(t *testing.T, db any, wh, sku uint, qty int, cost float64) {
	t.Helper()
	ls := new(LedgerService)
	if err := ls.Post(context.Background(), nil, []MoveInput{{
		WarehouseID: wh, SkuID: sku, Qty: qty, UnitCost: cost,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 91, SourceLineID: uint(qty), OccurredAt: time.Now(),
	}}); err != nil {
		t.Fatalf("二次进货失败: %v", err)
	}
}

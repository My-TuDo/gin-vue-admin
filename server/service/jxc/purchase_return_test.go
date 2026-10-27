package jxc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 采购退货测试：真实业务路径是“买多了/质量问题，退一部分给供应商”，
// 不是整单红冲。重点验证：
//   - 金额按原采购单价，而不是当前移动均价；
//   - 超退（包括两张草稿叠加）被挡住；
//   - 库存不足时不能硬退；
//   - 已红冲的采购单不能再退。

func receivePurchase(t *testing.T, ps *PurchaseService, wh, sku uint, qty int, cost float64) *jxc.PurchaseOrder {
	t.Helper()
	po, err := ps.Create(context.Background(), &jxc.PurchaseOrder{
		WarehouseID: wh,
		Items:       []jxc.PurchaseItem{{SkuID: sku, Qty: qty, UnitCost: cost}},
	})
	if err != nil {
		t.Fatalf("建采购单失败: %v", err)
	}
	if err := ps.Approve(context.Background(), po.ID, "t"); err != nil {
		t.Fatalf("审核采购单失败: %v", err)
	}
	if err := ps.Receive(context.Background(), po.ID, "t", time.Now()); err != nil {
		t.Fatalf("采购入库失败: %v", err)
	}
	return po
}

// TestPurchaseReturn_PostAtOriginalCost 买 10@5 + 10@7，退第一张的 5 件：
// 库存应变成 15 件/95 元（扣 5×5），而不是按均价 6 扣 30。
func TestPurchaseReturn_PostAtOriginalCost(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps := NewPurchaseService()
	prs := NewPurchaseReturnService()
	ls := new(LedgerService)
	ctx := context.Background()

	po1 := receivePurchase(t, ps, wh, sku, 10, 5)
	_ = receivePurchase(t, ps, wh, sku, 10, 7)

	ret, err := prs.Create(ctx, &jxc.PurchaseReturn{
		PurchaseID: po1.ID,
		Items:      []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 5}},
	})
	if err != nil {
		t.Fatalf("建采购退货单失败: %v", err)
	}
	if ret.ReturnNo == "" || ret.Amount != 25 || ret.Items[0].UnitCost != 5 {
		t.Fatalf("退货单应按原价 5 计 25 元，实际 %+v / item=%+v", ret, ret.Items)
	}
	if err := prs.Post(ctx, ret.ID, "t", time.Now()); err != nil {
		t.Fatalf("采购退货出库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 15 || v != money.FromYuan(95) {
		t.Fatalf("退货后应为 15/95.00，实际 %d/%s", q, v)
	}

	var mv jxc.StockMove
	if err := db.Where("source_type = ? AND source_id = ?", jxc.MoveSourcePurchaseReturn, ret.ID).
		First(&mv).Error; err != nil {
		t.Fatalf("查采购退货账本行失败: %v", err)
	}
	if mv.Qty != -5 || mv.UnitCost != 5 || money.FromYuan(mv.Amount) != money.FromYuan(-25) {
		t.Errorf("账本行应为 -5 件 @5 / -25.00，实际 %+v", mv)
	}
	if mv.Amount > 0 {
		t.Errorf("采购退货是出库，金额必须为负，实际 %.2f", mv.Amount)
	}

	// 流水应能回溯到采购退货单号
	rows, _, err := ls.Moves(ctx, wh, sku, 1, 50)
	if err != nil {
		t.Fatalf("查流水失败: %v", err)
	}
	found := false
	for _, row := range rows {
		if row.SourceType == jxc.MoveSourcePurchaseReturn && row.SourceNo == ret.ReturnNo {
			found = true
		}
	}
	if !found {
		t.Errorf("流水里应能按采购退货单号 %s 找到来源", ret.ReturnNo)
	}
	mustInvariant(t, ls)
}

// TestPurchaseReturn_DraftsCannotOverReturn 两张各 6 件的草稿，先后出库时第二张必须被拒。
func TestPurchaseReturn_DraftsCannotOverReturn(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps := NewPurchaseService()
	prs := NewPurchaseReturnService()
	ls := new(LedgerService)
	ctx := context.Background()

	po := receivePurchase(t, ps, wh, sku, 10, 5)
	mk := func() *jxc.PurchaseReturn {
		ret, err := prs.Create(ctx, &jxc.PurchaseReturn{
			PurchaseID: po.ID,
			Items:      []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 6}},
		})
		if err != nil {
			t.Fatalf("建退货草稿失败: %v", err)
		}
		return ret
	}
	r1, r2 := mk(), mk()
	if err := prs.Post(ctx, r1.ID, "t", time.Now()); err != nil {
		t.Fatalf("第一张退货出库失败: %v", err)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 4 {
		t.Fatalf("第一张退 6 件后应剩 4，实际 %d", q)
	}
	if err := prs.Post(ctx, r2.ID, "t", time.Now()); err == nil ||
		!strings.Contains(err.Error(), "最多可退") {
		t.Fatalf("第二张超退应被拒绝，got %v", err)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 4 {
		t.Errorf("拒绝后库存应仍为 4，实际 %d", q)
	}
	if _, err := prs.Create(ctx, &jxc.PurchaseReturn{
		PurchaseID: po.ID,
		Items:      []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 5}},
	}); err == nil || !strings.Contains(err.Error(), "最多可退 4") {
		t.Errorf("已退 6 后只能再退 4，创建 5 应被拒，got %v", err)
	}
	mustInvariant(t, ls)
}

// TestPurchaseReturn_InsufficientStockRejected 买 10 卖 8 后只剩 2，
// 退 5 件给供应商必须在过账时被库存校验挡住，不能把库存退成负数。
func TestPurchaseReturn_InsufficientStockRejected(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps := NewPurchaseService()
	ss := NewSaleService()
	prs := NewPurchaseReturnService()
	ls := new(LedgerService)
	ctx := context.Background()

	po := receivePurchase(t, ps, wh, sku, 10, 5)
	so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 8}}})
	if err != nil {
		t.Fatalf("开销售单失败: %v", err)
	}
	if err := ss.Ship(ctx, so.ID, "t", time.Now()); err != nil {
		t.Fatalf("销售出库失败: %v", err)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 2 {
		t.Fatalf("卖 8 后应剩 2，实际 %d", q)
	}

	ret, err := prs.Create(ctx, &jxc.PurchaseReturn{
		PurchaseID: po.ID,
		Items:      []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 5}},
	})
	if err != nil {
		t.Fatalf("建退货单失败: %v", err)
	}
	if err := prs.Post(ctx, ret.ID, "t", time.Now()); err == nil ||
		!strings.Contains(err.Error(), "结存不足") {
		t.Fatalf("库存不足时退货应被拒绝，got %v", err)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 2 {
		t.Errorf("拒绝后库存应仍为 2，实际 %d", q)
	}
	var got jxc.PurchaseReturn
	db.First(&got, ret.ID)
	if got.Status != jxc.PurchaseReturnStatusDraft {
		t.Errorf("失败后退货单应保持草稿，实际状态 %d", got.Status)
	}
	mustInvariant(t, ls)
}

// TestPurchaseReturn_ReversedPurchaseRejected 已整单红冲的采购单不能再退货。
func TestPurchaseReturn_ReversedPurchaseRejected(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps := NewPurchaseService()
	rs := NewReversalService()
	prs := NewPurchaseReturnService()
	ctx := context.Background()

	po := receivePurchase(t, ps, wh, sku, 10, 5)
	if _, err := rs.Reverse(ctx, jxc.ReversalSourcePurchase, po.ID, "供应商发错货", "t"); err != nil {
		t.Fatalf("采购红冲失败: %v", err)
	}
	if _, err := prs.Create(ctx, &jxc.PurchaseReturn{
		PurchaseID: po.ID,
		Items:      []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 1}},
	}); err == nil || !strings.Contains(err.Error(), "红冲") {
		t.Fatalf("已红冲的采购单不应能退货，got %v", err)
	}
}

// TestPurchaseReturn_Cancel 取消草稿后不能再出库。
func TestPurchaseReturn_Cancel(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps := NewPurchaseService()
	prs := NewPurchaseReturnService()
	ctx := context.Background()

	po := receivePurchase(t, ps, wh, sku, 3, 8)
	ret, err := prs.Create(ctx, &jxc.PurchaseReturn{
		PurchaseID: po.ID,
		Items:      []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 1}},
	})
	if err != nil {
		t.Fatalf("建退货单失败: %v", err)
	}
	if err := prs.Cancel(ctx, ret.ID); err != nil {
		t.Fatalf("取消退货单失败: %v", err)
	}
	if err := prs.Post(ctx, ret.ID, "t", time.Now()); err == nil ||
		!strings.Contains(err.Error(), "草稿") {
		t.Errorf("取消后的退货单不应能出库，got %v", err)
	}
}

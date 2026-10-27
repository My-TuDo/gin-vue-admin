package jxc

import (
	"context"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 月报采购额必须是净额：已入库采购 − 已过账采购退货；采购退货计入退货发生月。
func TestPeriod_PurchaseReturnNetsPurchaseAmount(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps := NewPurchaseService()
	prs := NewPurchaseReturnService()
	periods := NewPeriodService()
	ctx := context.Background()

	sep := time.Date(2026, 9, 15, 10, 0, 0, 0, time.Local)
	oct := time.Date(2026, 10, 15, 10, 0, 0, 0, time.Local)

	po, err := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
		Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 10, UnitCost: 5}}})
	if err != nil {
		t.Fatalf("建采购单失败: %v", err)
	}
	_ = ps.Approve(ctx, po.ID, "t")
	if err := ps.Receive(ctx, po.ID, "t", sep); err != nil {
		t.Fatalf("9 月采购入库失败: %v", err)
	}

	ret, err := prs.Create(ctx, &jxc.PurchaseReturn{PurchaseID: po.ID,
		Items: []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 2}}})
	if err != nil {
		t.Fatalf("建采购退货单失败: %v", err)
	}
	if err := prs.Post(ctx, ret.ID, "t", oct); err != nil {
		t.Fatalf("10 月采购退货出库失败: %v", err)
	}

	sepSum, err := periods.Summary(ctx, "2026-09")
	if err != nil {
		t.Fatalf("9 月报失败: %v", err)
	}
	if sepSum.PurchaseAmount != 50 || sepSum.PurchaseReturnAmount != 0 || sepSum.StockValue != 50 {
		t.Errorf("9 月应为采购 50/退货 0/期末 50，实际 %.2f/%.2f/%.2f",
			sepSum.PurchaseAmount, sepSum.PurchaseReturnAmount, sepSum.StockValue)
	}

	octSum, err := periods.Summary(ctx, "2026-10")
	if err != nil {
		t.Fatalf("10 月报失败: %v", err)
	}
	if octSum.PurchaseAmount != -10 || octSum.PurchaseReturnAmount != 10 || octSum.StockValue != 40 {
		t.Errorf("10 月应为净采购 -10/采购退货 10/期末 40，实际 %.2f/%.2f/%.2f",
			octSum.PurchaseAmount, octSum.PurchaseReturnAmount, octSum.StockValue)
	}
}

package jxc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 采购退货红冲：退给供应商的货还没发生，整单撤销，库存按原退货金额加回来。
func TestReversal_PurchaseReturnRestoresStock(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps := NewPurchaseService()
	prs := NewPurchaseReturnService()
	rvs := NewReversalService()
	ls := new(LedgerService)
	ctx := context.Background()

	po, err := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
		Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 10, UnitCost: 5}}})
	if err != nil {
		t.Fatalf("建采购单失败: %v", err)
	}
	_ = ps.Approve(ctx, po.ID, "t")
	if err := ps.Receive(ctx, po.ID, "t", time.Now()); err != nil {
		t.Fatalf("采购入库失败: %v", err)
	}
	pr, err := prs.Create(ctx, &jxc.PurchaseReturn{PurchaseID: po.ID,
		Items: []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 2}}})
	if err != nil {
		t.Fatalf("建采购退货单失败: %v", err)
	}
	if err := prs.Post(ctx, pr.ID, "t", time.Now()); err != nil {
		t.Fatalf("采购退货出库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 8 || v != money.FromYuan(40) {
		t.Fatalf("退货后应为 8/40，实际 %d/%s", q, v)
	}

	rev, err := rvs.Reverse(ctx, jxc.ReversalSourcePurchaseReturn, pr.ID, "供应商取消退货", "t")
	if err != nil {
		t.Fatalf("采购退货红冲失败: %v", err)
	}
	if rev.SourceType != jxc.ReversalSourcePurchaseReturn || rev.Qty != 2 ||
		money.FromYuan(rev.Amount) != money.FromYuan(10) {
		t.Errorf("红冲单信息不对：%+v", rev)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 10 || v != money.FromYuan(50) {
		t.Errorf("红冲后应回到 10/50，实际 %d/%s", q, v)
	}
	var got jxc.PurchaseReturn
	db.First(&got, pr.ID)
	if got.ReversedAt == nil {
		t.Error("采购退货单应被标记为已红冲")
	}

	// 已红冲的退货不再占用可退数量，供应商那批货可以重新退。
	lines, err := prs.Returnable(ctx, po.ID)
	if err != nil {
		t.Fatalf("查可退数量失败: %v", err)
	}
	if len(lines) != 1 || lines[0].CanReturn != 10 {
		t.Errorf("红冲后可退数量应回到 10，实际 %+v", lines)
	}

	if _, err := rvs.Reverse(ctx, jxc.ReversalSourcePurchaseReturn, pr.ID, "再冲一次", "t"); err == nil ||
		!strings.Contains(err.Error(), "已经红冲") {
		t.Errorf("重复红冲应被拒绝，got %v", err)
	}
	mustInvariant(t, ls)
}

// 草稿采购退货不能红冲。
func TestReversal_PurchaseReturnDraftRejected(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps := NewPurchaseService()
	prs := NewPurchaseReturnService()
	rvs := NewReversalService()
	ctx := context.Background()

	po, _ := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
		Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 3, UnitCost: 5}}})
	_ = ps.Approve(ctx, po.ID, "t")
	_ = ps.Receive(ctx, po.ID, "t", time.Now())
	pr, err := prs.Create(ctx, &jxc.PurchaseReturn{PurchaseID: po.ID,
		Items: []jxc.PurchaseReturnItem{{SkuID: sku, Qty: 1}}})
	if err != nil {
		t.Fatalf("建退货草稿失败: %v", err)
	}
	if _, err := rvs.Reverse(ctx, jxc.ReversalSourcePurchaseReturn, pr.ID, "试一下", "t"); err == nil ||
		!strings.Contains(err.Error(), "只有已退货") {
		t.Errorf("草稿退货不应能红冲，got %v", err)
	}
}

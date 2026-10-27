package jxc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 原子开单+出库：一次请求完成“开单 → 预留 → 出库”，失败时单据/预留/账本一起回滚。
func TestSaleCheckout_AtomicSuccess(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 5, 10)
	ss := NewSaleService()
	ls := new(LedgerService)
	ctx := context.Background()

	order, err := ss.CreateAndShip(ctx, &jxc.SaleOrder{
		WarehouseID: wh,
		Items:       []jxc.SaleItem{{SkuID: sku, Qty: 2}},
	}, "t", time.Now())
	if err != nil {
		t.Fatalf("原子开单出库失败: %v", err)
	}
	if order.Status != jxc.SaleStatusShipped || order.TotalAmount != 198 || order.CostAmount != 20 {
		t.Fatalf("单据应为已出库/198/成本20，实际 %+v", order)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 3 || v != money.FromYuan(30) {
		t.Fatalf("库存应为 3/30，实际 %d/%s", q, v)
	}
	var consumed int64
	db.Model(&jxc.StockReservation{}).
		Where("source_type = ? AND source_id = ? AND status = ?", jxc.ReserveSourceSaleOrder, order.ID, jxc.ReservationConsumed).
		Count(&consumed)
	if consumed != 1 {
		t.Errorf("预留应转为已出库，实际 %d 条", consumed)
	}
	var moves int64
	db.Model(&jxc.StockMove{}).
		Where("source_type = ? AND source_id = ?", jxc.MoveSourceSaleOut, order.ID).Count(&moves)
	if moves != 1 {
		t.Errorf("应只有 1 条出库账本行，实际 %d", moves)
	}
	mustInvariant(t, ls)
}

// TestSaleCheckout_RollsBackOnReserveFailure 出库失败/可售不足时不能留下草稿或预留。
func TestSaleCheckout_RollsBackOnReserveFailure(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 1, 10)
	ss := NewSaleService()
	ls := new(LedgerService)
	ctx := context.Background()

	if _, err := ss.CreateAndShip(ctx, &jxc.SaleOrder{
		WarehouseID: wh,
		Items:       []jxc.SaleItem{{SkuID: sku, Qty: 2}},
	}, "t", time.Now()); err == nil {
		t.Fatal("库存不足时应失败")
	}
	var orders, reservations int64
	db.Model(&jxc.SaleOrder{}).Count(&orders)
	db.Model(&jxc.StockReservation{}).Count(&reservations)
	if orders != 0 || reservations != 0 {
		t.Fatalf("失败后不应留下销售单/预留，实际订单=%d 预留=%d", orders, reservations)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 1 || v != money.FromYuan(10) {
		t.Errorf("库存应保持 1/10，实际 %d/%s", q, v)
	}
	mustInvariant(t, ls)
}

// TestSaleCheckout_ConcurrentLastItem 最后一件只能被一个 checkout 抢到。
func TestSaleCheckout_ConcurrentLastItem(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 1, 10)
	ss := NewSaleService()
	ls := new(LedgerService)
	ctx := context.Background()

	const n = 4
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ss.CreateAndShip(ctx, &jxc.SaleOrder{
				WarehouseID: wh,
				Items:       []jxc.SaleItem{{SkuID: sku, Qty: 1}},
			}, "t", time.Now()); err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Fatalf("最后一件应只成交 1 次，实际 %d 次", success)
	}
	var orders int64
	db.Model(&jxc.SaleOrder{}).Count(&orders)
	if orders != 1 {
		t.Errorf("应只落 1 张销售单，实际 %d", orders)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 0 {
		t.Errorf("库存应归零，实际 %d", q)
	}
	mustInvariant(t, ls)
}

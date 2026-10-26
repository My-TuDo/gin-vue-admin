package jxc

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// 单据层测试：验证「单据只是壳，账本才是真相」这条设计真的成立。
//
// 重点不是"接口能返回 200"，而是：
//   - 状态机能挡住非法流转；
//   - 单据落账后，账本与余额保持不变量；
//   - 并发出库只有一个能成功；
//   - 取消单据能正确释放预留，不会留下僵尸占用。

func newTestServices() (*PurchaseService, *SaleService, *LedgerService) {
	return NewPurchaseService(), NewSaleService(), new(LedgerService)
}

func seedStock(t *testing.T, db *gorm.DB, wh, sku uint, qty int, unitCost float64) {
	t.Helper()
	svc := new(LedgerService)
	if err := svc.Post(context.Background(), nil, []MoveInput{{
		WarehouseID: wh, SkuID: sku, Qty: qty, UnitCost: unitCost,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 90, SourceLineID: 1, OccurredAt: time.Now(),
	}}); err != nil {
		t.Fatalf("铺底库存失败: %v", err)
	}
}

// TestPurchaseFlow 采购：草稿 → 审核 → 入库，只有入库动库存
func TestPurchaseFlow(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps, _, ls := newTestServices()
	ctx := context.Background()

	order, err := ps.Create(ctx, &jxc.PurchaseOrder{
		WarehouseID: wh,
		Items:       []jxc.PurchaseItem{{SkuID: sku, Qty: 10, UnitCost: 5}},
	})
	if err != nil {
		t.Fatalf("建采购单失败: %v", err)
	}
	if order.Status != jxc.PurchaseStatusDraft || order.TotalAmount != 50 {
		t.Fatalf("应为草稿且总额 50，实际状态 %d 总额 %.2f", order.Status, order.TotalAmount)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 0 {
		t.Errorf("草稿状态不应动库存，实际 %d 件", q)
	}

	if err := ps.Approve(ctx, order.ID, "tester"); err != nil {
		t.Fatalf("审核失败: %v", err)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 0 {
		t.Errorf("审核也不应动库存，实际 %d 件", q)
	}

	if err := ps.Receive(ctx, order.ID, "tester", time.Now()); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 10 || v != money.FromYuan(50) {
		t.Fatalf("入库后应为 10/50.00，实际 %d/%s", q, v)
	}

	// 重复入库必须被拒（状态已变）
	if err := ps.Receive(ctx, order.ID, "tester", time.Now()); err == nil {
		t.Error("重复入库应被拒绝")
	}
	// 已入库不可取消
	if err := ps.Cancel(ctx, order.ID, "tester"); err == nil ||
		!strings.Contains(err.Error(), "红冲") {
		t.Errorf("已入库应不可取消，并提示红冲，got %v", err)
	}
	mustInvariant(t, ls)
}

// TestSaleFlow_ReserveThenShip 销售：开单预留 → 出库结转成本
func TestSaleFlow_ReserveThenShip(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 10, 5) // 10 件 @5.00

	_, ss, ls := newTestServices()
	ctx := context.Background()

	order, err := ss.Create(ctx, &jxc.SaleOrder{
		WarehouseID: wh,
		Items:       []jxc.SaleItem{{SkuID: sku, Qty: 3, Price: 99}},
	})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}
	// 开单只预留，不动库存
	if q, _ := balanceOf(t, db, wh, sku); q != 10 {
		t.Errorf("开单不应扣库存，实际 %d", q)
	}
	var res jxc.StockReservation
	if err := db.Where("source_type = ? AND source_id = ?", jxc.ReserveSourceSaleOrder, order.ID).First(&res).Error; err != nil {
		t.Fatalf("应生成预留记录: %v", err)
	}
	if res.Qty != 3 || res.Status != jxc.ReservationActive {
		t.Errorf("预留应为 3 件有效，实际 %d 件状态 %d", res.Qty, res.Status)
	}

	// 再开一张 8 件的单：可售 10-3=7 < 8，必须被拒
	if _, err := ss.Create(ctx, &jxc.SaleOrder{
		WarehouseID: wh,
		Items:       []jxc.SaleItem{{SkuID: sku, Qty: 8, Price: 99}},
	}); err == nil {
		t.Fatal("可售不足时应拒绝开单")
	}

	if err := ss.Ship(ctx, order.ID, "tester", time.Now()); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 7 || v != money.FromYuan(35) {
		t.Fatalf("出库后应为 7/35.00，实际 %d/%s", q, v)
	}

	// 明细成本应被回填为 3×5=15
	var item jxc.SaleItem
	if err := db.Where("sale_id = ?", order.ID).First(&item).Error; err != nil {
		t.Fatalf("查明细失败: %v", err)
	}
	if item.CostAmt != 15 {
		t.Errorf("明细成本应为 15.00，实际 %.2f", item.CostAmt)
	}
	var got jxc.SaleOrder
	db.First(&got, order.ID)
	if got.CostAmount != 15 || got.Status != jxc.SaleStatusShipped {
		t.Errorf("单据成本应为 15、状态已出库，实际 %.2f / %d", got.CostAmount, got.Status)
	}
	// 预留转为已出库
	db.Where("source_type = ? AND source_id = ?", jxc.ReserveSourceSaleOrder, order.ID).First(&res)
	if res.Status != jxc.ReservationConsumed {
		t.Errorf("出库后预留应转为已出库(3)，实际 %d", res.Status)
	}
	mustInvariant(t, ls)
}

// TestSaleCancel_ReleasesReservation 取消要释放预留，否则可售量会永久被占
func TestSaleCancel_ReleasesReservation(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 5, 10)
	_, ss, ls := newTestServices()
	ctx := context.Background()

	order, err := ss.Create(ctx, &jxc.SaleOrder{
		WarehouseID: wh,
		Items:       []jxc.SaleItem{{SkuID: sku, Qty: 5, Price: 20}},
	})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}
	// 此时可售为 0，再开 1 件应失败
	if _, err := ss.Create(ctx, &jxc.SaleOrder{
		WarehouseID: wh, Items: []jxc.SaleItem{{SkuID: sku, Qty: 1, Price: 20}},
	}); err == nil {
		t.Fatal("可售为 0 时开单应失败")
	}
	if err := ss.Cancel(ctx, order.ID, "tester"); err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	// 释放后应能再开 5 件
	if _, err := ss.Create(ctx, &jxc.SaleOrder{
		WarehouseID: wh, Items: []jxc.SaleItem{{SkuID: sku, Qty: 5, Price: 20}},
	}); err != nil {
		t.Fatalf("取消后预留应已释放，实际仍开不了单: %v", err)
	}
	mustInvariant(t, ls)
}

// TestSaleShip_Concurrent 并发出库：只能成功一次
func TestSaleShip_Concurrent(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 10, 5)
	_, ss, ls := newTestServices()
	ctx := context.Background()

	order, err := ss.Create(ctx, &jxc.SaleOrder{
		WarehouseID: wh,
		Items:       []jxc.SaleItem{{SkuID: sku, Qty: 4, Price: 30}},
	})
	if err != nil {
		t.Fatalf("开单失败: %v", err)
	}

	const n = 6
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := ss.Ship(ctx, order.ID, "tester", time.Now()); err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Fatalf("并发出库应只成功 1 次，实际 %d 次", success)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 6 {
		t.Errorf("应只扣一次 4 件，剩 6，实际 %d", q)
	}
	var cnt int64
	db.Model(&jxc.StockMove{}).Where("source_type = ? AND source_id = ?", jxc.MoveSourceSaleOut, order.ID).Count(&cnt)
	if cnt != 1 {
		t.Errorf("应只有 1 条出库账本行，实际 %d", cnt)
	}
	mustInvariant(t, ls)
}

// TestDocument_Validation 参数校验（挡住脏数据进入账本）
func TestDocument_Validation(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps, ss, _ := newTestServices()
	ctx := context.Background()

	cases := []struct {
		name string
		run  func() error
		want string
	}{
		{"采购空明细", func() error {
			_, err := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh})
			return err
		}, "至少需要一条明细"},
		{"采购数量为0", func() error {
			_, err := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
				Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 0, UnitCost: 1}}})
			return err
		}, "数量必须大于 0"},
		{"采购单价为负", func() error {
			_, err := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
				Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 1, UnitCost: -1}}})
			return err
		}, "单价不能为负"},
		{"SKU 不存在", func() error {
			_, err := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
				Items: []jxc.PurchaseItem{{SkuID: 9999, Qty: 1, UnitCost: 1}}})
			return err
		}, "SKU 不存在"},
		{"仓库不存在", func() error {
			_, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: 9999,
				Items: []jxc.SaleItem{{SkuID: sku, Qty: 1, Price: 1}}})
			return err
		}, "仓库不存在"},
		{"未选仓库", func() error {
			_, err := ss.Create(ctx, &jxc.SaleOrder{Items: []jxc.SaleItem{{SkuID: sku, Qty: 1, Price: 1}}})
			return err
		}, "请选择出库仓库"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.run()
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("应报 %q，got %v", c.want, err)
			}
		})
	}
}

// TestDocument_StatusGuards 非法状态流转必须被挡住
func TestDocument_StatusGuards(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	ps, ss, _ := newTestServices()
	ctx := context.Background()

	po, _ := ps.Create(ctx, &jxc.PurchaseOrder{WarehouseID: wh,
		Items: []jxc.PurchaseItem{{SkuID: sku, Qty: 1, UnitCost: 1}}})
	if err := ps.Approve(ctx, po.ID, "t"); err != nil {
		t.Fatalf("首次审核失败: %v", err)
	}
	if err := ps.Approve(ctx, po.ID, "t"); err == nil {
		t.Error("重复审核应被拒绝")
	}

	seedStock(t, db, wh, sku, 5, 2)
	so, _ := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: sku, Qty: 1, Price: 3}}})
	if err := ss.Ship(ctx, so.ID, "t", time.Now()); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	if err := ss.Ship(ctx, so.ID, "t", time.Now()); err == nil {
		t.Error("重复出库应被拒绝")
	}
	if err := ss.Cancel(ctx, so.ID, "t"); err == nil ||
		!strings.Contains(err.Error(), "退货单") {
		t.Errorf("已出库应不可取消，并提示开退货单，got %v", err)
	}
}

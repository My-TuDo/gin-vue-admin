package jxc

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 账本集成测试：跑在**真实 MySQL** 上。
//
// 为什么不用 sqlite：本设计依赖行锁（SELECT ... FOR UPDATE）、
// ON DUPLICATE KEY、唯一索引冲突语义，sqlite 的行为不同，测出来的结论不可信。
//
// 无 MySQL 环境时自动 Skip。

const defaultBaseDSN = "root:123456@tcp(127.0.0.1:3306)"

func baseDSN() string {
	if v := os.Getenv("JXC_IT_MYSQL_BASE"); v != "" {
		return v
	}
	return defaultBaseDSN
}

// newLedgerTestDB 建一次性库、迁移表、赋全局 DB，测完删库
func newLedgerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	admin, err := gorm.Open(mysql.Open(baseDSN()+"/?charset=utf8mb4&parseTime=True&loc=Local"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("跳过：无法连接 MySQL（%s）：%v", baseDSN(), err)
	}
	sqlDB, err := admin.DB()
	if err != nil {
		t.Skipf("跳过：%v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		t.Skipf("跳过：MySQL 不可达：%v", err)
	}

	name := fmt.Sprintf("jxc_ledger_%d_%d", time.Now().UnixNano()%1e6, rand.Intn(10000))
	if err := admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4").Error; err != nil {
		_ = sqlDB.Close()
		t.Fatalf("建库失败: %v", err)
	}
	_ = sqlDB.Close()

	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s/%s?charset=utf8mb4&parseTime=True&loc=Local", baseDSN(), name)),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("连库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&jxc.Warehouse{}, &jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.StockMove{}, &jxc.StockBalance{}, &jxc.StockReservation{},
		&jxc.AccountingPeriod{}, &jxc.StockBalanceSnapshot{},
		&jxc.PurchaseOrder{}, &jxc.PurchaseItem{},
		&jxc.SaleOrder{}, &jxc.SaleItem{},
		&jxc.StockReversal{}, &jxc.SaleReturn{}, &jxc.SaleReturnItem{},
		&jxc.StockCheck{}, &jxc.StockCheckItem{},
		&jxc.DocSequence{},
	); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	old := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() {
		global.GVA_DB = old
		cleaner, e := gorm.Open(mysql.Open(baseDSN()+"/?charset=utf8mb4&parseTime=True&loc=Local"),
			&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if e == nil {
			_ = cleaner.Exec("DROP DATABASE IF EXISTS `" + name + "`").Error
		}
	})
	return db
}

// seedSku 建仓库 + 商品 + SKU，返回 warehouseID, skuID
func seedSku(t *testing.T, db *gorm.DB) (uint, uint) {
	t.Helper()
	wh := jxc.Warehouse{Code: "WH1", Name: "主仓", Status: 1}
	if err := db.Create(&wh).Error; err != nil {
		t.Fatalf("建仓库失败: %v", err)
	}
	g := jxc.Goods{Code: "G1", Name: "测试商品", Status: 1}
	if err := db.Create(&g).Error; err != nil {
		t.Fatalf("建商品失败: %v", err)
	}
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SKU1", Color: "黑", Size: "M", SalePrice: 99, Status: 1}
	if err := db.Create(&sku).Error; err != nil {
		t.Fatalf("建SKU失败: %v", err)
	}
	return wh.ID, sku.ID
}

func balanceOf(t *testing.T, db *gorm.DB, wh, sku uint) (int, money.Cents) {
	t.Helper()
	var b jxc.StockBalance
	if err := db.Where("warehouse_id = ? AND sku_id = ?", wh, sku).First(&b).Error; err != nil {
		return 0, 0
	}
	return b.Qty, money.FromYuan(b.Value)
}

func mustInvariant(t *testing.T, svc *LedgerService) {
	t.Helper()
	vs, err := svc.Reconcile(context.Background(), 0)
	if err != nil {
		t.Fatalf("对账失败: %v", err)
	}
	if len(vs) != 0 {
		t.Fatalf("账本与余额不一致（%d 项）：%v", len(vs), vs)
	}
}

// TestLedger_FullLifecycle 进 → 再进（不同价）→ 出库，验证移动加权平均与不变量
func TestLedger_FullLifecycle(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	svc := new(LedgerService)
	ctx := context.Background()
	now := time.Now()

	// 入库 10 件 @5.00
	if err := svc.Post(ctx, nil, []MoveInput{{
		WarehouseID: wh, SkuID: sku, Qty: 10, UnitCost: 5,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 1, SourceLineID: 1, OccurredAt: now,
	}}); err != nil {
		t.Fatalf("首次入库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 10 || v != money.FromYuan(50) {
		t.Fatalf("入库后应为 10/50.00，实际 %d/%s", q, v)
	}

	// 再入 10 件 @7.00 → 20 件 / 120.00，均价 6.00
	if err := svc.Post(ctx, nil, []MoveInput{{
		WarehouseID: wh, SkuID: sku, Qty: 10, UnitCost: 7,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 2, SourceLineID: 1, OccurredAt: now,
	}}); err != nil {
		t.Fatalf("二次入库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 20 || v != money.FromYuan(120) {
		t.Fatalf("二次入库后应为 20/120.00，实际 %d/%s", q, v)
	}

	// 出库 5 件 → 成本 30.00，剩 15 件 / 90.00
	if err := svc.Post(ctx, nil, []MoveInput{{
		WarehouseID: wh, SkuID: sku, Qty: -5,
		SourceType: jxc.MoveSourceSaleOut, SourceID: 100, SourceLineID: 1, OccurredAt: now,
	}}); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 15 || v != money.FromYuan(90) {
		t.Fatalf("出库后应为 15/90.00，实际 %d/%s", q, v)
	}

	// 出库行的成本必须被冻结成 6.00（移动加权平均）
	var out jxc.StockMove
	if err := db.Where("source_type = ? AND source_id = ?", jxc.MoveSourceSaleOut, 100).First(&out).Error; err != nil {
		t.Fatalf("查账本行失败: %v", err)
	}
	if money.FromYuan(out.UnitCost) != money.FromYuan(6) {
		t.Errorf("出库结转成本应为 6.00，实际 %.2f", out.UnitCost)
	}
	if money.FromYuan(out.Amount) != money.FromYuan(-30) {
		t.Errorf("出库金额应为 -30.00，实际 %.2f", out.Amount)
	}

	mustInvariant(t, svc)
}

// TestLedger_Idempotent 同一来源重复过账只产生一行
func TestLedger_Idempotent(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	svc := new(LedgerService)
	ctx := context.Background()

	m := MoveInput{WarehouseID: wh, SkuID: sku, Qty: 5, UnitCost: 10,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 7, SourceLineID: 1, OccurredAt: time.Now()}

	for i := 0; i < 3; i++ {
		if err := svc.Post(ctx, nil, []MoveInput{m}); err != nil {
			t.Fatalf("第 %d 次过账失败: %v", i+1, err)
		}
	}
	var cnt int64
	db.Model(&jxc.StockMove{}).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("重复过账只应有 1 行账本，实际 %d 行", cnt)
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 5 {
		t.Errorf("余额应为 5，实际 %d", q)
	}
	mustInvariant(t, svc)
}

// TestLedger_OversellRejected 超出结存的出库必须被拒绝，且不留下半条账
func TestLedger_OversellRejected(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	svc := new(LedgerService)
	ctx := context.Background()
	now := time.Now()

	_ = svc.Post(ctx, nil, []MoveInput{{WarehouseID: wh, SkuID: sku, Qty: 2, UnitCost: 5,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 1, SourceLineID: 1, OccurredAt: now}})

	err := svc.Post(ctx, nil, []MoveInput{{WarehouseID: wh, SkuID: sku, Qty: -3,
		SourceType: jxc.MoveSourceSaleOut, SourceID: 2, SourceLineID: 1, OccurredAt: now}})
	if err == nil {
		t.Fatal("卖出 3 件而只有 2 件，应被拒绝")
	}
	if q, _ := balanceOf(t, db, wh, sku); q != 2 {
		t.Errorf("被拒绝后余额应保持 2，实际 %d", q)
	}
	var cnt int64
	db.Model(&jxc.StockMove{}).Count(&cnt)
	if cnt != 1 {
		t.Errorf("失败不应留下账本行，实际 %d 行", cnt)
	}
	mustInvariant(t, svc)
}

// TestLedger_Reservation 预留占用可售量，重复预留幂等
func TestLedger_Reservation(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	svc := new(LedgerService)
	ctx := context.Background()
	now := time.Now()

	_ = svc.Post(ctx, nil, []MoveInput{{WarehouseID: wh, SkuID: sku, Qty: 5, UnitCost: 5,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 1, SourceLineID: 1, OccurredAt: now}})

	r := MoveInput{WarehouseID: wh, SkuID: sku, Qty: 4,
		SourceType: jxc.ReserveSourceSaleOrder, SourceID: 10, SourceLineID: 1}
	if err := svc.Reserve(ctx, nil, r); err != nil {
		t.Fatalf("预留 4 件应成功: %v", err)
	}
	if err := svc.Reserve(ctx, nil, r); err != nil {
		t.Fatalf("重复预留应幂等: %v", err)
	}
	var cnt int64
	db.Model(&jxc.StockReservation{}).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("重复预留只应有 1 行，实际 %d", cnt)
	}

	// 再预留 2 件：结存 5 - 已预留 4 < 2，应拒绝
	err := svc.Reserve(ctx, nil, MoveInput{WarehouseID: wh, SkuID: sku, Qty: 2,
		SourceType: jxc.ReserveSourceSaleOrder, SourceID: 11, SourceLineID: 1})
	if err == nil {
		t.Fatal("可售不足时预留应被拒绝")
	}
	mustInvariant(t, svc)
}

// TestLedger_Reversal 红冲：反向记账，余额回到之前
func TestLedger_Reversal(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	svc := new(LedgerService)
	ctx := context.Background()
	now := time.Now()

	_ = svc.Post(ctx, nil, []MoveInput{{WarehouseID: wh, SkuID: sku, Qty: 10, UnitCost: 5,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 1, SourceLineID: 1, OccurredAt: now}})

	var orig jxc.StockMove
	if err := db.Where("source_id = ? AND source_type = ?", 1, jxc.MoveSourcePurchaseIn).First(&orig).Error; err != nil {
		t.Fatalf("查原账本行失败: %v", err)
	}

	origID := orig.ID
	if err := svc.Post(ctx, nil, []MoveInput{{
		WarehouseID: wh, SkuID: sku, Qty: -10, SourceType: jxc.MoveSourceReversal,
		SourceID: 900, SourceLineID: 1, OccurredAt: now, ReversalOf: &origID,
	}}); err != nil {
		t.Fatalf("红冲失败: %v", err)
	}
	if q, v := balanceOf(t, db, wh, sku); q != 0 || v != 0 {
		t.Errorf("红冲后余额应归零，实际 %d/%s", q, v)
	}
	mustInvariant(t, svc)
}

// TestLedger_ClosedPeriodRejected 已结账期间拒绝写入
func TestLedger_ClosedPeriodRejected(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	svc := new(LedgerService)
	ctx := context.Background()

	old := time.Date(2026, 9, 15, 10, 0, 0, 0, time.Local)
	if err := db.Create(&jxc.AccountingPeriod{PeriodMonth: "2026-09", Status: jxc.PeriodClosed}).Error; err != nil {
		t.Fatalf("建期间失败: %v", err)
	}
	err := svc.Post(ctx, nil, []MoveInput{{WarehouseID: wh, SkuID: sku, Qty: 1, UnitCost: 1,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 1, SourceLineID: 1, OccurredAt: old}})
	if err == nil {
		t.Fatal("已结账期间应拒绝写入")
	}
	if !strings.Contains(err.Error(), "已结账") {
		t.Errorf("错误信息应说明期间已结账，got %v", err)
	}
}

// TestLedger_ReconcileDetectsDrift 人为篡改余额后，对账必须报出来。
// 这条是整套设计的价值证明：余额漂移不可避免，但必须可被发现。
func TestLedger_ReconcileDetectsDrift(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	svc := new(LedgerService)
	ctx := context.Background()

	_ = svc.Post(ctx, nil, []MoveInput{{WarehouseID: wh, SkuID: sku, Qty: 10, UnitCost: 5,
		SourceType: jxc.MoveSourcePurchaseIn, SourceID: 1, SourceLineID: 1, OccurredAt: time.Now()}})
	mustInvariant(t, svc)

	// 模拟"有人手工改了余额"或"某次崩溃半途而废"
	if err := db.Model(&jxc.StockBalance{}).
		Where("warehouse_id = ? AND sku_id = ?", wh, sku).
		Update("qty", 7).Error; err != nil {
		t.Fatalf("篡改余额失败: %v", err)
	}

	vs, err := svc.Reconcile(ctx, 0)
	if err != nil {
		t.Fatalf("对账失败: %v", err)
	}
	if len(vs) == 0 {
		t.Fatal("余额被改成 7，对账必须报出不一致")
	}
	var found bool
	for _, v := range vs {
		if v.Kind == "qty_mismatch" && v.Expected == "10" && v.Actual == "7" {
			found = true
		}
	}
	if !found {
		t.Fatalf("应报 qty_mismatch(10 vs 7)，实际：%v", vs)
	}
}

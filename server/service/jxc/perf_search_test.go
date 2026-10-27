package jxc

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	common "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
)

// 5 万条单据的列表性能验收。
//
// 默认跳过，避免 CI 每次都造 25 万行；需要时：
//
//	JXC_PERF=1 go test ./service/jxc -run TestDocumentSearch_Performance50k -v -timeout 30m
//
// 测试会在临时库里造数据、补复合索引、ANALYZE，然后量 5 个列表查询的 P50/P95。
// 索引定义与 initialize/gorm_biz_indexes.go 保持一致。
func TestDocumentSearch_Performance50k(t *testing.T) {
	if os.Getenv("JXC_PERF") != "1" {
		t.Skip("设置 JXC_PERF=1 才运行 5 万条性能验收")
	}

	db := newLedgerTestDB(t)
	wh, _ := seedSku(t, db)
	ctx := context.Background()
	now := time.Now()

	supplier := jxc.Partner{Kind: jxc.PartnerSupplier, Code: "PERF-SUP", Name: "性能供应商", Status: 1}
	customer := jxc.Partner{Kind: jxc.PartnerCustomer, Code: "PERF-CUS", Name: "性能客户", Status: 1}
	if err := db.Create(&supplier).Error; err != nil {
		t.Fatalf("建供应商失败: %v", err)
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("建客户失败: %v", err)
	}

	const n = 50000
	t.Logf("开始造数：每类 %d 条", n)

	purchases := make([]jxc.PurchaseOrder, 0, n)
	for i := 0; i < n; i++ {
		at := now.Add(-time.Duration(i) * time.Minute)
		purchases = append(purchases, jxc.PurchaseOrder{
			GVA_MODEL:   global.GVA_MODEL{CreatedAt: at, UpdatedAt: at},
			OrderNo:     fmt.Sprintf("PO-PERF-%06d", i),
			SupplierID:  &supplier.ID,
			WarehouseID: wh,
			Status:      int8(1 + i%4),
			TotalAmount: 100,
			PostedAt:    &at,
			Remark:      "性能测试采购单",
		})
	}
	if err := db.CreateInBatches(&purchases, 1000).Error; err != nil {
		t.Fatalf("造采购单失败: %v", err)
	}

	sales := make([]jxc.SaleOrder, 0, n)
	for i := 0; i < n; i++ {
		at := now.Add(-time.Duration(i) * time.Minute)
		sales = append(sales, jxc.SaleOrder{
			GVA_MODEL:   global.GVA_MODEL{CreatedAt: at, UpdatedAt: at},
			OrderNo:     fmt.Sprintf("SO-PERF-%06d", i),
			CustomerID:  &customer.ID,
			WarehouseID: wh,
			Status:      int8(1 + i%3),
			TotalAmount: 100,
			CostAmount:  50,
			PostedAt:    &at,
			Remark:      "性能测试销售单",
		})
	}
	if err := db.CreateInBatches(&sales, 1000).Error; err != nil {
		t.Fatalf("造销售单失败: %v", err)
	}

	saleReturns := make([]jxc.SaleReturn, 0, n)
	for i := 0; i < n; i++ {
		at := now.Add(-time.Duration(i) * time.Minute)
		saleReturns = append(saleReturns, jxc.SaleReturn{
			GVA_MODEL:   global.GVA_MODEL{CreatedAt: at, UpdatedAt: at},
			ReturnNo:    fmt.Sprintf("RT-PERF-%06d", i),
			SaleID:      sales[i].ID,
			SaleNo:      sales[i].OrderNo,
			WarehouseID: wh,
			Status:      int8(1 + i%3),
			Amount:      100,
			CostAmount:  50,
			PostedAt:    &at,
			Reason:      "性能测试退货",
		})
	}
	if err := db.CreateInBatches(&saleReturns, 1000).Error; err != nil {
		t.Fatalf("造销售退货失败: %v", err)
	}

	purchaseReturns := make([]jxc.PurchaseReturn, 0, n)
	for i := 0; i < n; i++ {
		at := now.Add(-time.Duration(i) * time.Minute)
		purchaseReturns = append(purchaseReturns, jxc.PurchaseReturn{
			GVA_MODEL:   global.GVA_MODEL{CreatedAt: at, UpdatedAt: at},
			ReturnNo:    fmt.Sprintf("PR-PERF-%06d", i),
			PurchaseID:  purchases[i].ID,
			PurchaseNo:  purchases[i].OrderNo,
			SupplierID:  &supplier.ID,
			WarehouseID: wh,
			Status:      int8(1 + i%3),
			Amount:      100,
			PostedAt:    &at,
			Reason:      "性能测试采购退货",
		})
	}
	if err := db.CreateInBatches(&purchaseReturns, 1000).Error; err != nil {
		t.Fatalf("造采购退货失败: %v", err)
	}

	reversals := make([]jxc.StockReversal, 0, n)
	for i := 0; i < n; i++ {
		at := now.Add(-time.Duration(i) * time.Minute)
		sourceType := jxc.ReversalSourcePurchase
		sourceID := purchases[i].ID
		sourceNo := purchases[i].OrderNo
		if i%2 == 0 {
			sourceType = jxc.ReversalSourceSale
			sourceID = sales[i].ID
			sourceNo = sales[i].OrderNo
		}
		reversals = append(reversals, jxc.StockReversal{
			GVA_MODEL:  global.GVA_MODEL{CreatedAt: at, UpdatedAt: at},
			ReversalNo: fmt.Sprintf("RC-PERF-%06d", i),
			SourceType: sourceType,
			SourceID:   sourceID,
			SourceNo:   sourceNo,
			Reason:     "性能测试红冲",
			Amount:     100,
			Qty:        1,
			Status:     jxc.ReversalPosted,
			PostedAt:   at,
		})
	}
	if err := db.CreateInBatches(&reversals, 1000).Error; err != nil {
		t.Fatalf("造红冲失败: %v", err)
	}

	// 复合索引与 initialize/gorm_biz_indexes.go 保持一致；先造数再建索引更快。
	type idx struct{ name, table, cols string }
	indexes := []idx{
		{"idx_po_status_created", "purchase_order", "status, created_at"},
		{"idx_po_wh_created", "purchase_order", "warehouse_id, created_at"},
		{"idx_po_supplier_created", "purchase_order", "supplier_id, created_at"},
		{"idx_so_status_created", "sale_order", "status, created_at"},
		{"idx_so_wh_created", "sale_order", "warehouse_id, created_at"},
		{"idx_so_customer_created", "sale_order", "customer_id, created_at"},
		{"idx_sar_status_created", "sale_return", "status, created_at"},
		{"idx_sar_wh_created", "sale_return", "warehouse_id, created_at"},
		{"idx_sar_sale_created", "sale_return", "sale_id, created_at"},
		{"idx_pr_status_created", "purchase_return", "status, created_at"},
		{"idx_pr_wh_created", "purchase_return", "warehouse_id, created_at"},
		{"idx_pr_supplier_created", "purchase_return", "supplier_id, created_at"},
		{"idx_rev_type_posted", "stock_reversal", "source_type, posted_at"},
	}
	for _, index := range indexes {
		if err := db.Exec("CREATE INDEX " + index.name + " ON " + index.table + " (" + index.cols + ")").Error; err != nil {
			t.Fatalf("建索引 %s 失败: %v", index.name, err)
		}
	}
	if err := db.Exec("ANALYZE TABLE purchase_order, sale_order, sale_return, purchase_return, stock_reversal").Error; err != nil {
		t.Logf("ANALYZE 失败（不影响正确性）: %v", err)
	}

	ps, ss, rs, sar, prs := NewPurchaseService(), NewSaleService(), NewReversalService(), NewReturnService(), NewPurchaseReturnService()
	status2 := int8(2)
	startAt := now.AddDate(0, 0, -7).Format("2006-01-02")
	endAt := now.Format("2006-01-02")

	measure := func(name string, fn func() error) {
		for i := 0; i < 3; i++ {
			if err := fn(); err != nil {
				t.Fatalf("%s 预热失败: %v", name, err)
			}
		}
		const rounds = 20
		durations := make([]time.Duration, 0, rounds)
		for i := 0; i < rounds; i++ {
			started := time.Now()
			if err := fn(); err != nil {
				t.Fatalf("%s 查询失败: %v", name, err)
			}
			durations = append(durations, time.Since(started))
		}
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		p50 := durations[len(durations)/2]
		p95 := durations[int(float64(len(durations)-1)*0.95)]
		t.Logf("%-18s p50=%s p95=%s", name, p50.Round(time.Millisecond), p95.Round(time.Millisecond))
	}

	search := func(info jxcReq.DocumentSearch) error {
		info.PageInfo = common.PageInfo{Page: 1, PageSize: 20}
		info.Status = &status2
		info.WarehouseID = wh
		info.StartAt = startAt
		info.EndAt = endAt
		_, _, err := ps.Search(ctx, info)
		return err
	}
	measure("purchase_page", func() error { return search(jxcReq.DocumentSearch{}) })

	measure("sale_page", func() error {
		_, _, err := ss.Search(ctx, jxcReq.DocumentSearch{
			PageInfo: common.PageInfo{Page: 1, PageSize: 20}, Status: &status2,
			WarehouseID: wh, StartAt: startAt, EndAt: endAt,
		})
		return err
	})
	measure("sale_return_page", func() error {
		_, _, err := sar.Search(ctx, jxcReq.DocumentSearch{
			PageInfo: common.PageInfo{Page: 1, PageSize: 20}, Status: &status2,
			WarehouseID: wh, StartAt: startAt, EndAt: endAt,
		})
		return err
	})
	measure("purchase_return_page", func() error {
		_, _, err := prs.Search(ctx, jxcReq.DocumentSearch{
			PageInfo: common.PageInfo{Page: 1, PageSize: 20}, Status: &status2,
			WarehouseID: wh, StartAt: startAt, EndAt: endAt,
		})
		return err
	})
	measure("reversal_page", func() error {
		_, _, err := rs.Search(ctx, jxcReq.DocumentSearch{
			PageInfo:    common.PageInfo{Page: 1, PageSize: 20},
			WarehouseID: wh, DateField: "posted", StartAt: startAt, EndAt: endAt,
		})
		return err
	})

	// 关键词单独量：LIKE %...% 用不上普通 B-Tree 索引，最容易成为慢查询。
	measure("purchase_keyword", func() error {
		_, _, err := ps.Search(ctx, jxcReq.DocumentSearch{
			PageInfo:    common.PageInfo{Page: 1, PageSize: 20, Keyword: "PO-PERF-000001"},
			WarehouseID: wh, StartAt: startAt, EndAt: endAt,
		})
		return err
	})
	measure("sale_keyword", func() error {
		_, _, err := ss.Search(ctx, jxcReq.DocumentSearch{
			PageInfo:    common.PageInfo{Page: 1, PageSize: 20, Keyword: "SO-PERF-000001"},
			WarehouseID: wh, StartAt: startAt, EndAt: endAt,
		})
		return err
	})
	measure("sale_return_keyword", func() error {
		_, _, err := sar.Search(ctx, jxcReq.DocumentSearch{
			PageInfo:    common.PageInfo{Page: 1, PageSize: 20, Keyword: "RT-PERF-000001"},
			WarehouseID: wh, StartAt: startAt, EndAt: endAt,
		})
		return err
	})
	measure("purchase_return_keyword", func() error {
		_, _, err := prs.Search(ctx, jxcReq.DocumentSearch{
			PageInfo:    common.PageInfo{Page: 1, PageSize: 20, Keyword: "PR-PERF-000001"},
			WarehouseID: wh, StartAt: startAt, EndAt: endAt,
		})
		return err
	})
	measure("reversal_keyword", func() error {
		_, _, err := rs.Search(ctx, jxcReq.DocumentSearch{
			PageInfo:    common.PageInfo{Page: 1, PageSize: 20, Keyword: "RC-PERF-000001"},
			WarehouseID: wh, DateField: "posted", StartAt: startAt, EndAt: endAt,
		})
		return err
	})
}

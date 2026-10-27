package jxc

import (
	"context"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// DashboardService 工作台聚合：一次请求返回 KPI、待办与低库存。
//
// 为什么不让前端继续拿 5 个 page?pageSize=100 的列表自己算：
// 超过 100 条后 KPI 会少算，而且首屏要串行拉 5 个列表。这里用 SQL 聚合，
// 口径与 docs/23 的月报一致（红冲计入发生日、退货冲减）。
type DashboardService struct{}

func NewDashboardService() *DashboardService { return new(DashboardService) }

// DashboardSummary 工作台响应。
type DashboardSummary struct {
	Date                string            `json:"date"`
	StockValue          float64           `json:"stockValue"`
	SkuCount            int64             `json:"skuCount"`
	TodaySalesAmount    float64           `json:"todaySalesAmount"`
	TodaySalesOrders    int64             `json:"todaySalesOrders"`
	TodayReturnAmount   float64           `json:"todayReturnAmount"`
	TodayReturnOrders   int64             `json:"todayReturnOrders"`
	TodayGross          float64           `json:"todayGross"`
	TodayPurchaseAmount float64           `json:"todayPurchaseAmount"`
	Pending             DashboardPending  `json:"pending"`
	LowStock            DashboardLowStock `json:"lowStock"`
}

// DashboardPending 待办计数（不受分页影响）。
type DashboardPending struct {
	Approve             int64 `json:"approve"`             // 草稿采购单待审核
	Receive             int64 `json:"receive"`             // 已审核采购单待入库
	Ship                int64 `json:"ship"`                // 草稿销售单待出库
	PurchaseReturnDraft int64 `json:"purchaseReturnDraft"` // 采购退货草稿待出库
	Total               int64 `json:"total"`
}

// DashboardLowStock 低库存。
type DashboardLowStock struct {
	Threshold int                     `json:"threshold"`
	Count     int64                   `json:"count"`
	Items     []DashboardLowStockItem `json:"items"`
}

// DashboardLowStockItem 低库存明细行。
type DashboardLowStockItem struct {
	WarehouseID   uint   `json:"warehouseId"`
	WarehouseName string `json:"warehouseName"`
	SkuID         uint   `json:"skuId"`
	GoodsName     string `json:"goodsName"`
	Color         string `json:"color"`
	Size          string `json:"size"`
	SkuCode       string `json:"skuCode"`
	Available     int    `json:"available"`
}

const dashboardLowStockThreshold = 10

type dashboardAgg struct {
	Cnt    int64
	Amount float64
	Cost   float64
}

// Summary 计算某一天的经营概览。date 为空时取 Asia/Shanghai 的今天。
func (s *DashboardService) Summary(ctx context.Context, date string) (*DashboardSummary, error) {
	if date == "" {
		date = time.Now().In(jxcLocation).Format("2006-01-02")
	}
	start, err := time.ParseInLocation("2006-01-02", date, jxcLocation)
	if err != nil {
		return nil, fmt.Errorf("date 格式应为 YYYY-MM-DD：%s", date)
	}
	end := start.AddDate(0, 0, 1)
	db := global.GVA_DB.WithContext(ctx)

	var saleAgg, retAgg, revSaleAgg, purAgg, revPurAgg, prAgg dashboardAgg

	if err := db.Model(&jxc.SaleOrder{}).
		Select("COUNT(*) AS cnt, COALESCE(SUM(total_amount),0) AS amount, COALESCE(SUM(cost_amount),0) AS cost").
		Where("status = ? AND posted_at >= ? AND posted_at < ?", jxc.SaleStatusShipped, start, end).
		Scan(&saleAgg).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.SaleReturn{}).
		Select("COUNT(*) AS cnt, COALESCE(SUM(amount),0) AS amount, COALESCE(SUM(cost_amount),0) AS cost").
		Where("status = ? AND posted_at >= ? AND posted_at < ?", jxc.ReturnStatusPosted, start, end).
		Scan(&retAgg).Error; err != nil {
		return nil, err
	}
	if err := db.Table("stock_reversal r").
		Select("COUNT(*) AS cnt, COALESCE(SUM(so.total_amount),0) AS amount, COALESCE(SUM(so.cost_amount),0) AS cost").
		Joins("JOIN sale_order so ON so.id = r.source_id").
		Where("r.source_type = ? AND r.posted_at >= ? AND r.posted_at < ?", jxc.ReversalSourceSale, start, end).
		Scan(&revSaleAgg).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.PurchaseOrder{}).
		Select("COUNT(*) AS cnt, COALESCE(SUM(total_amount),0) AS amount").
		Where("status = ? AND posted_at >= ? AND posted_at < ?", jxc.PurchaseStatusReceived, start, end).
		Scan(&purAgg).Error; err != nil {
		return nil, err
	}
	if err := db.Table("stock_reversal r").
		Select("COUNT(*) AS cnt, COALESCE(SUM(po.total_amount),0) AS amount").
		Joins("JOIN purchase_order po ON po.id = r.source_id").
		Where("r.source_type = ? AND r.posted_at >= ? AND r.posted_at < ?", jxc.ReversalSourcePurchase, start, end).
		Scan(&revPurAgg).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.PurchaseReturn{}).
		Select("COUNT(*) AS cnt, COALESCE(SUM(amount),0) AS amount").
		Where("status = ? AND reversed_at IS NULL AND posted_at >= ? AND posted_at < ?",
			jxc.PurchaseReturnStatusPosted, start, end).
		Scan(&prAgg).Error; err != nil {
		return nil, err
	}

	var stockValue float64
	if err := db.Model(&jxc.StockBalance{}).
		Select("COALESCE(SUM(value),0)").Scan(&stockValue).Error; err != nil {
		return nil, err
	}
	var skuCount int64
	if err := db.Model(&jxc.StockBalance{}).Count(&skuCount).Error; err != nil {
		return nil, err
	}

	var pending DashboardPending
	if err := db.Model(&jxc.PurchaseOrder{}).Where("status = ?", jxc.PurchaseStatusDraft).Count(&pending.Approve).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.PurchaseOrder{}).Where("status = ?", jxc.PurchaseStatusApproved).Count(&pending.Receive).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.SaleOrder{}).Where("status = ?", jxc.SaleStatusDraft).Count(&pending.Ship).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.PurchaseReturn{}).Where("status = ?", jxc.PurchaseReturnStatusDraft).Count(&pending.PurchaseReturnDraft).Error; err != nil {
		return nil, err
	}
	pending.Total = pending.Approve + pending.Receive + pending.Ship + pending.PurchaseReturnDraft

	lowStock, err := s.lowStock(db)
	if err != nil {
		return nil, err
	}

	salesAmount := money.FromYuan(saleAgg.Amount - revSaleAgg.Amount - retAgg.Amount).Yuan()
	salesCost := money.FromYuan(saleAgg.Cost - revSaleAgg.Cost - retAgg.Cost).Yuan()
	return &DashboardSummary{
		Date:                date,
		StockValue:          money.FromYuan(stockValue).Yuan(),
		SkuCount:            skuCount,
		TodaySalesAmount:    salesAmount,
		TodaySalesOrders:    saleAgg.Cnt,
		TodayReturnAmount:   money.FromYuan(retAgg.Amount).Yuan(),
		TodayReturnOrders:   retAgg.Cnt,
		TodayGross:          money.FromYuan(salesAmount).Sub(money.FromYuan(salesCost)).Yuan(),
		TodayPurchaseAmount: money.FromYuan(purAgg.Amount - revPurAgg.Amount - prAgg.Amount).Yuan(),
		Pending:             pending,
		LowStock:            lowStock,
	}, nil
}

// lowStock 计算总低库存数与前 5 条明细。
// available = 结存 - 有效预留；要求 qty >= 0，available <= threshold。
func (s *DashboardService) lowStock(db *gorm.DB) (DashboardLowStock, error) {
	const threshold = dashboardLowStockThreshold
	out := DashboardLowStock{Threshold: threshold, Items: []DashboardLowStockItem{}}

	reservedExpr := "COALESCE((SELECT SUM(r.qty) FROM stock_reservation r WHERE r.warehouse_id = b.warehouse_id AND r.sku_id = b.sku_id AND r.status = ?),0)"
	joins := []string{
		"LEFT JOIN warehouse w ON w.id = b.warehouse_id",
		"LEFT JOIN goods_sku s ON s.id = b.sku_id",
		"LEFT JOIN goods g ON g.id = s.goods_id",
	}

	countQ := db.Table("stock_balance b").Joins(joins[0]).Joins(joins[1]).Joins(joins[2]).
		Where("b.qty >= 0").
		Where("(b.qty - "+reservedExpr+") <= ?", jxc.ReservationActive, threshold)
	if err := countQ.Count(&out.Count).Error; err != nil {
		return out, err
	}

	itemsQ := db.Table("stock_balance b").
		Select("b.warehouse_id, w.name AS warehouse_name, b.sku_id, g.name AS goods_name, s.color, s.size, s.sku_code, b.qty - "+reservedExpr+" AS available", jxc.ReservationActive).
		Joins(joins[0]).Joins(joins[1]).Joins(joins[2]).
		Where("b.qty >= 0").
		Having("available <= ?", threshold).
		Order("available ASC, b.warehouse_id, b.sku_id").
		Limit(5)
	if err := itemsQ.Scan(&out.Items).Error; err != nil {
		return out, err
	}
	return out, nil
}

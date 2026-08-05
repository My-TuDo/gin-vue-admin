package jxc

import (
	"context"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// DashboardService 仪表盘聚合
type DashboardService struct{}

// OverviewItem 概览（今日/本周/本月）
type OverviewItem struct {
	Label    string  `json:"label"`
	Sales    float64 `json:"sales"`    // 净销售额（含退换符号）
	Orders   int64   `json:"orders"`   // 订单数（已确认单）
	Profit   float64 `json:"profit"`   // 毛利 = Σ(amount) - Σ(qty*成本)
	ReturnAmt float64 `json:"returnAmt"` // 退货额（退货+换入的负数部分）
}

// TrendPoint 趋势点
type TrendPoint struct {
	Date   string  `json:"date"`
	Sales  float64 `json:"sales"`
	Orders int64   `json:"orders"`
}

// TopItem 热销
type TopItem struct {
	SkuCode  string  `json:"skuCode"`
	GoodsName string `json:"goodsName"`
	Qty      int     `json:"qty"`
	Amount   float64 `json:"amount"`
}

// AlertItem 库存预警
type AlertItem struct {
	SkuID     uint   `json:"skuId"`
	SkuCode   string `json:"skuCode"`
	GoodsName string `json:"goodsName"`
	SafeStock int    `json:"safeStock"`
	Available int    `json:"available"`
}

// CatItem 分类占比
type CatItem struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// Overview 今日/本周/本月概览
func (s *DashboardService) Overview(ctx context.Context) ([]OverviewItem, error) {
	db := global.GVA_DB.WithContext(ctx)
	now := time.Now()
	periods := []struct {
		label string
		from  string
		to    string
	}{
		{"今日", now.Format("2006-01-02"), now.Format("2006-01-02")},
		{"本周", mondayOf(now).Format("2006-01-02"), now.Format("2006-01-02")},
		{"本月", time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"), now.Format("2006-01-02")},
	}
	result := make([]OverviewItem, 0, 3)
	for _, p := range periods {
		var sales struct {
			Total  float64
			Orders int64
		}
		if err := db.Table("sale_order").
			Select("COALESCE(SUM(total_amount),0) AS total, COUNT(*) AS orders").
			Where("status = ? AND created_at >= ? AND created_at < ?", 2, p.from, nextDay(p.from)).
			Scan(&sales).Error; err != nil {
			return nil, err
		}
		var profit struct {
			Total     float64
			ReturnAmt float64
		}
		if err := db.Table("sale_item i").
			Select("COALESCE(SUM(i.amount - i.qty * COALESCE(sku.cost_price,0) * CASE WHEN i.amount >= 0 THEN 1 ELSE -1 END),0) AS total, "+
				"COALESCE(SUM(CASE WHEN i.amount < 0 THEN i.amount ELSE 0 END),0) AS return_amt").
			Joins("JOIN sale_order o ON o.id = i.sale_id").
			Joins("JOIN goods_sku sku ON sku.id = i.sku_id").
			Where("o.status = ? AND o.created_at >= ? AND o.created_at < ?", 2, p.from, nextDay(p.from)).
			Scan(&profit).Error; err != nil {
			return nil, err
		}
		result = append(result, OverviewItem{
			Label: p.label, Sales: sales.Total, Orders: sales.Orders,
			Profit: profit.Total, ReturnAmt: profit.ReturnAmt,
		})
	}
	return result, nil
}

// Trend 近 N 天销售趋势
func (s *DashboardService) Trend(ctx context.Context, days int) ([]TrendPoint, error) {
	if days <= 0 {
		days = 7
	}
	db := global.GVA_DB.WithContext(ctx)
	from := time.Now().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	var list []TrendPoint
	err := db.Table("sale_order").
		Select("DATE(created_at) AS date, COALESCE(SUM(total_amount),0) AS sales, COUNT(*) AS orders").
		Where("status = ? AND created_at >= ?", 2, from).
		Group("DATE(created_at)").Order("date").Scan(&list).Error
	return list, err
}

// Top 热销商品（正常销售 + 换出，按销量）
func (s *DashboardService) Top(ctx context.Context, days, limit int) ([]TopItem, error) {
	if days <= 0 {
		days = 30
	}
	if limit <= 0 {
		limit = 10
	}
	db := global.GVA_DB.WithContext(ctx)
	from := time.Now().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	var list []TopItem
	err := db.Table("sale_item i").
		Select("sku.sku_code AS sku_code, COALESCE(g.name,'') AS goods_name, "+
			"SUM(i.qty) AS qty, COALESCE(SUM(i.amount),0) AS amount").
		Joins("JOIN sale_order o ON o.id = i.sale_id").
		Joins("JOIN goods_sku sku ON sku.id = i.sku_id").
		Joins("JOIN goods g ON g.id = sku.goods_id").
		Where("o.status = ? AND (o.order_type = ? OR i.direction = ?) AND o.created_at >= ?", 2, 1, 1, from).
		Group("sku.id").Order("qty DESC").Limit(limit).Scan(&list).Error
	return list, err
}

// StockAlert 库存预警（可售 ≤ 安全库存，safe_stock > 0）
func (s *DashboardService) StockAlert(ctx context.Context) ([]AlertItem, error) {
	db := global.GVA_DB.WithContext(ctx)
	var list []AlertItem
	err := db.Table("goods_sku sku").
		Select("sku.id AS sku_id, sku.sku_code, COALESCE(g.name,'') AS goods_name, "+
			"sku.safe_stock, COALESCE(SUM(st.quantity),0) - COALESCE(SUM(st.lock_quantity),0) AS available").
		Joins("JOIN goods g ON g.id = sku.goods_id").
		Joins("LEFT JOIN stock st ON st.sku_id = sku.id").
		Where("sku.safe_stock > 0 AND sku.deleted_at IS NULL").
		Group("sku.id").
		Having("available <= sku.safe_stock").
		Order("available").Scan(&list).Error
	return list, err
}

// Category 分类销售占比（近 N 天）
func (s *DashboardService) Category(ctx context.Context, days int) ([]CatItem, error) {
	if days <= 0 {
		days = 30
	}
	db := global.GVA_DB.WithContext(ctx)
	from := time.Now().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	var list []CatItem
	err := db.Table("sale_item i").
		Select("COALESCE(c.name,'未分类') AS name, COALESCE(SUM(i.amount),0) AS amount").
		Joins("JOIN sale_order o ON o.id = i.sale_id").
		Joins("JOIN goods_sku sku ON sku.id = i.sku_id").
		Joins("JOIN goods g ON g.id = sku.goods_id").
		Joins("LEFT JOIN goods_category c ON c.id = g.category_id").
		Where("o.status = ? AND o.created_at >= ?", 2, from).
		Group("c.id").Order("amount DESC").Scan(&list).Error
	return list, err
}

// mondayOf 本周一
func mondayOf(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-wd+1, 0, 0, 0, 0, t.Location())
}

// nextDay 次日（yyyy-mm-dd）
func nextDay(d string) string {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return d
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

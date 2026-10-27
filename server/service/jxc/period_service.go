package jxc

import (
	"context"
	"errors"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PeriodService 会计期间（月结）与月报。
//
// 月结做两件事：
//  1. 冻结当月余额快照（以后所有人看到的"上月底库存"都是这个数，不再实时重算）；
//  2. 关闭期间（此后不能再往这个月过账，更正只能在本月红冲）。
//
// 没有月结，"上月底库存是多少"这个问题永远算不准——今天补一张上月的单，
// 昨天的报表就变了。这不是技术洁癖，是商家对账的基础。
type PeriodService struct{}

func NewPeriodService() *PeriodService { return new(PeriodService) }

// Close 结账。幂等：已结账会报错而不是覆盖快照。
func (s *PeriodService) Close(ctx context.Context, yearMonth, operator string) (*jxc.AccountingPeriod, error) {
	if _, err := time.Parse("2006-01", yearMonth); err != nil {
		return nil, errors.New("期间格式应为 YYYY-MM")
	}

	var period jxc.AccountingPeriod
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("period_month = ?", yearMonth).First(&period).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			period = jxc.AccountingPeriod{PeriodMonth: yearMonth, Status: jxc.PeriodOpen}
			if e := tx.Create(&period).Error; e != nil {
				return e
			}
		} else if err != nil {
			return err
		}
		if period.Status == jxc.PeriodClosed {
			return errors.New("该期间已经结账")
		}

		// 结账前先对账：账本都不平就结账，等于把错误固化下来
		ledgerSvc := new(LedgerService)
		violations, err := ledgerSvc.Reconcile(ctx, 0)
		if err != nil {
			return err
		}
		if len(violations) > 0 {
			return errors.New("账实不符，禁止结账：请先在「库存 → 对账」中处理差异")
		}

		var balances []jxc.StockBalance
		if err := tx.Find(&balances).Error; err != nil {
			return err
		}
		for _, b := range balances {
			if b.Qty == 0 && b.Value == 0 {
				continue
			}
			snap := jxc.StockBalanceSnapshot{
				PeriodID: period.ID, WarehouseID: b.WarehouseID, SkuID: b.SkuID,
				Qty: b.Qty, Value: b.Value,
			}
			if err := tx.Create(&snap).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		return tx.Model(&jxc.AccountingPeriod{}).Where("id = ?", period.ID).
			Updates(map[string]interface{}{
				"status": jxc.PeriodClosed, "closed_at": now, "closed_by": operator,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	period.Status = jxc.PeriodClosed
	now := time.Now()
	period.ClosedAt = &now
	period.ClosedBy = operator
	return &period, nil
}

// List 期间列表（倒序）
func (s *PeriodService) List(ctx context.Context) ([]jxc.AccountingPeriod, error) {
	var list []jxc.AccountingPeriod
	err := global.GVA_DB.WithContext(ctx).Order("period_month desc").Find(&list).Error
	return list, err
}

// Snapshot 某期间冻结的余额
func (s *PeriodService) Snapshot(ctx context.Context, periodID uint) ([]jxc.StockBalanceSnapshot, error) {
	var list []jxc.StockBalanceSnapshot
	err := global.GVA_DB.WithContext(ctx).Where("period_id = ?", periodID).Find(&list).Error
	return list, err
}

// MonthlySummary 月度经营汇总。
//
// 口径说明（写在这里，避免以后每个人算出来的毛利都不一样）：
//   - 销售额 = 已出库且**未红冲**的销售单金额合计 − 已入库的退货金额
//   - 销售成本 = 对应单据上冻结的成本合计 − 退回货物按原单成本的金额
//   - 毛利 = 销售额 − 销售成本
//   - 采购额 = 已入库且未红冲的采购单金额合计
//   - 期末库存 = 当前结存金额（若已结账，应等于该期间快照）
type MonthlySummary struct {
	Month          string  `json:"month"`
	SalesAmount    float64 `json:"salesAmount"`
	SalesCost      float64 `json:"salesCost"`
	ReturnAmount   float64 `json:"returnAmount"`
	ReturnCost     float64 `json:"returnCost"`
	Gross          float64 `json:"gross"`
	SalesOrders    int64   `json:"salesOrders"`
	ReturnOrders   int64   `json:"returnOrders"`
	PurchaseAmount float64 `json:"purchaseAmount"`
	StockValue     float64 `json:"stockValue"`
}

func (s *PeriodService) Summary(ctx context.Context, yearMonth string) (*MonthlySummary, error) {
	start, err := time.ParseInLocation("2006-01", yearMonth, time.Local)
	if err != nil {
		return nil, errors.New("期间格式应为 YYYY-MM")
	}
	end := start.AddDate(0, 1, 0)
	db := global.GVA_DB.WithContext(ctx)
	out := &MonthlySummary{Month: yearMonth}

	type agg struct {
		Cnt    int64
		Amount float64
		Cost   float64
	}
	var saleAgg, retAgg, purAgg agg

	if err := db.Model(&jxc.SaleOrder{}).
		Select("COUNT(*) AS cnt, COALESCE(SUM(total_amount),0) AS amount, COALESCE(SUM(cost_amount),0) AS cost").
		Where("status = ? AND reversed_at IS NULL AND posted_at >= ? AND posted_at < ?",
			jxc.SaleStatusShipped, start, end).
		Scan(&saleAgg).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.SaleReturn{}).
		Select("COUNT(*) AS cnt, COALESCE(SUM(amount),0) AS amount, COALESCE(SUM(cost_amount),0) AS cost").
		Where("status = ? AND posted_at >= ? AND posted_at < ?", jxc.ReturnStatusPosted, start, end).
		Scan(&retAgg).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.PurchaseOrder{}).
		Select("COUNT(*) AS cnt, COALESCE(SUM(total_amount),0) AS amount").
		Where("status = ? AND reversed_at IS NULL AND posted_at >= ? AND posted_at < ?",
			jxc.PurchaseStatusReceived, start, end).
		Scan(&purAgg).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&jxc.StockBalance{}).
		Select("COALESCE(SUM(value),0)").Scan(&out.StockValue).Error; err != nil {
		return nil, err
	}

	out.SalesOrders = saleAgg.Cnt
	out.SalesAmount = money.FromYuan(saleAgg.Amount - retAgg.Amount).Yuan()
	out.SalesCost = money.FromYuan(saleAgg.Cost - retAgg.Cost).Yuan()
	out.ReturnOrders = retAgg.Cnt
	out.ReturnAmount = money.FromYuan(retAgg.Amount).Yuan()
	out.ReturnCost = money.FromYuan(retAgg.Cost).Yuan()
	out.Gross = money.FromYuan(out.SalesAmount).Sub(money.FromYuan(out.SalesCost)).Yuan()
	out.PurchaseAmount = money.FromYuan(purAgg.Amount).Yuan()
	return out, nil
}

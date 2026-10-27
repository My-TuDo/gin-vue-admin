package jxc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/ledger"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LedgerService 库存账本的**唯一写入口**。
//
// 约定（见 docs/16-库存账本核心设计.md）：
//   - 任何对 stock_move / stock_balance 的写入都必须经过这里，其它地方不允许直接写；
//   - 一张单据的所有明细在**一个事务**里过账，要么全成要么全不成；
//   - 幂等由 (source_type, source_id, source_line_id) 唯一索引保证，
//     重复过账不会产生第二行（POS 双击 / 网络重试 / 消息重投都安全）；
//   - 已结账期间拒绝写入；
//   - 出库成本按移动加权平均，**冻结在账本行上**。
type LedgerService struct{}

// MoveInput 一条待过账的账本行。
//
// Qty 有符号：入库为正、出库为负。用符号而不是方向字段，
// 是因为金额汇总、余额推导都能直接 SUM，不需要一堆 CASE WHEN。
type MoveInput struct {
	WarehouseID  uint
	SkuID        uint
	Qty          int
	UnitCost     float64 // 仅入库使用；出库由系统按移动加权平均计算
	SourceType   string
	SourceID     uint
	SourceLineID uint
	OccurredAt   time.Time
	Operator     string
	ReversalOf   *uint // 红冲时指向被冲销的账本行

	// AmountOverride 精确金额（销售退货入库 / 采购退货出库使用）。
	// 销售退货要按原单成本回补，而原单成本除以数量后往往不是整分：
	// 3 件 100.00 的单，前两次各退 33.33，最后一次必须退 33.34。
	// 采购退货则要按原采购价扣减，不能走当前移动加权平均。
	// 只传 UnitCost 无法表达这两类语义，所以用金额把结果钉死；
	// 为 nil 时仍按 UnitCost × Qty / 当前均价计算。
	AmountOverride *money.Cents
}

// 允许的出入方向（防止把退货单当成出库这类错用）
var inboundTypes = map[string]bool{
	jxc.MoveSourcePurchaseIn: true,
	jxc.MoveSourceSaleReturn: true,
	jxc.MoveSourceDirectIn:   true,
	jxc.MoveSourceCheckGain:  true,
}
var outboundTypes = map[string]bool{
	jxc.MoveSourcePurchaseReturn: true,
	jxc.MoveSourceSaleOut:        true,
	jxc.MoveSourceDirectOut:      true,
	jxc.MoveSourceCheckLoss:      true,
}

// validate 校验单条移动的语义
func (m MoveInput) validate() error {
	if m.WarehouseID == 0 || m.SkuID == 0 {
		return errors.New("仓库与 SKU 均不能为空")
	}
	if m.Qty == 0 {
		return errors.New("数量不能为 0")
	}
	if m.OccurredAt.IsZero() {
		return errors.New("业务发生时间不能为空")
	}
	if m.AmountOverride != nil {
		// 精确金额只用于退货残差吸收；其它来源必须由成本方法推导，避免调用方
		// 用"随便传一个金额"绕过移动加权平均。
		if m.SourceType != jxc.MoveSourceSaleReturn && m.SourceType != jxc.MoveSourcePurchaseReturn {
			return errors.New("只有退货（销售退货入库/采购退货出库）可以指定精确金额")
		}
		if *m.AmountOverride != 0 && (*m.AmountOverride > 0) != (m.Qty > 0) {
			return errors.New("精确金额的方向必须与数量方向一致")
		}
	}
	switch {
	case inboundTypes[m.SourceType]:
		if m.Qty < 0 {
			return fmt.Errorf("%s 是入库类型，数量必须为正", m.SourceType)
		}
		if m.UnitCost < 0 {
			return errors.New("入库成本不能为负")
		}
	case outboundTypes[m.SourceType]:
		if m.Qty > 0 {
			return fmt.Errorf("%s 是出库类型，数量必须为负", m.SourceType)
		}
	case m.SourceType == jxc.MoveSourceReversal:
		if m.ReversalOf == nil {
			return errors.New("红冲必须指定被冲销的账本行")
		}
		if m.UnitCost < 0 {
			return errors.New("红冲单价不能为负")
		}
	default:
		return fmt.Errorf("未知的账本来源类型: %s", m.SourceType)
	}
	return nil
}

// Post 过账。可在已有事务内调用（db 传 tx），也可独立调用（db 传 nil 时自建事务）。
func (s *LedgerService) Post(ctx context.Context, db *gorm.DB, moves []MoveInput) error {
	if len(moves) == 0 {
		return nil
	}
	for i, m := range moves {
		if err := m.validate(); err != nil {
			return fmt.Errorf("第 %d 条账本行不合法：%w", i+1, err)
		}
	}
	if db != nil {
		return s.postTx(ctx, db, moves)
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.postTx(ctx, tx, moves)
	})
}

func (s *LedgerService) postTx(ctx context.Context, tx *gorm.DB, moves []MoveInput) error {
	// 1) 期间必须处于打开状态
	checkedPeriod := map[string]bool{}
	for _, m := range moves {
		ym := m.OccurredAt.Format("2006-01")
		if checkedPeriod[ym] {
			continue
		}
		if err := s.ensurePeriodOpen(tx, ym); err != nil {
			return err
		}
		checkedPeriod[ym] = true
	}

	// 固定加锁顺序：不同单据的同一批 SKU 如果按不同顺序加锁，会互相等成死锁。
	// 账本行的处理顺序不影响账务语义，按 (仓库, SKU) 排序是零成本的防护。
	ordered := append([]MoveInput(nil), moves...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].WarehouseID != ordered[j].WarehouseID {
			return ordered[i].WarehouseID < ordered[j].WarehouseID
		}
		if ordered[i].SkuID != ordered[j].SkuID {
			return ordered[i].SkuID < ordered[j].SkuID
		}
		return ordered[i].SourceLineID < ordered[j].SourceLineID
	})

	for _, m := range ordered {
		// 2) 幂等：同一来源行已过账则跳过
		posted, err := s.alreadyPosted(tx, m)
		if err != nil {
			return err
		}
		if posted {
			continue
		}

		// 3) 锁住余额行（没有就建一行 0）
		pos, err := s.lockBalance(tx, m.WarehouseID, m.SkuID)
		if err != nil {
			return err
		}

		// 4) 按成本方法计算本行
		var unitCost money.Cents
		var amount money.Cents
		var after ledger.Position

		switch {
		case m.SourceType == jxc.MoveSourceReversal:
			// 红冲必须按原账本行的单价精确反向，不能走当前移动均价。
			// 单价直接从被冲销的账本行读取，不信任调用方传值——
			// 否则传 0 就能把库存数量冲掉、金额留在账上。
			var orig jxc.StockMove
			if err := tx.First(&orig, *m.ReversalOf).Error; err != nil {
				return fmt.Errorf("红冲失败：找不到被冲销的账本行 %d", *m.ReversalOf)
			}
			if orig.WarehouseID != m.WarehouseID || orig.SkuID != m.SkuID {
				return errors.New("红冲失败：被冲销账本行的仓库/SKU 与本次不一致")
			}
			if orig.Qty != -m.Qty {
				return errors.New("红冲失败：冲销数量必须与原账本行数量相反")
			}
			unitCost = money.FromYuan(orig.UnitCost)
			// 按被冲销账本行的**精确金额**反向，而不是"单价 × 数量"。
			// 出库单价是移动加权平均后四舍五入到分的，乘回去会差 1 分：
			// 3 件/100.00 的出库行单价 33.33，按单价冲只回来 99.99。
			amount = money.FromYuan(orig.Amount).Neg()
			after, err = pos.ApplyAmount(m.Qty, amount)
			if err != nil {
				return fmt.Errorf("红冲失败：SKU %d %w", m.SkuID, err)
			}
		case m.Qty > 0:
			unitCost = money.FromYuan(m.UnitCost)
			if m.AmountOverride != nil {
				// 退货残差：数量按明细，金额按原单剩余成本精确回补。
				amount = *m.AmountOverride
				after, err = pos.ApplyAmount(m.Qty, amount)
				if err != nil {
					return fmt.Errorf("入库失败：SKU %d %w", m.SkuID, err)
				}
			} else {
				amount = unitCost.MulQty(m.Qty)
				after = pos.ApplyInbound(m.Qty, unitCost)
			}
		default:
			if m.AmountOverride != nil {
				// 采购退货必须按原采购单价出库，不能走当前移动加权平均：
				// 供应商按原价退款，账上却按今天的均价减成本，差额会凭空出现。
				// 与销售退货一样，金额由 Service 精确给出，这里只做方向/结存校验。
				unitCost = money.FromYuan(m.UnitCost)
				amount = *m.AmountOverride
				after, err = pos.ApplyAmount(m.Qty, amount)
				if err != nil {
					return fmt.Errorf("出库失败：SKU %d %w", m.SkuID, err)
				}
			} else {
				cost, next, err := pos.ApplyOutbound(-m.Qty)
				if err != nil {
					return fmt.Errorf("出库失败：SKU %d %w", m.SkuID, err)
				}
				unitCost = money.FromYuan(cost.Yuan() / float64(-m.Qty))
				amount = cost.Neg()
				after = next
			}
		}

		// 5) 可售量不得为负（卖超必须挡住）
		reserved, err := s.reservedQty(tx, m.WarehouseID, m.SkuID)
		if err != nil {
			return err
		}
		if after.Qty-reserved < 0 {
			return fmt.Errorf("可售库存不足：SKU %d 结存 %d，已预留 %d，本次需要 %d",
				m.SkuID, pos.Qty, reserved, -m.Qty)
		}

		// 6) 写账本行（唯一索引兜底幂等）
		row := jxc.StockMove{
			WarehouseID: m.WarehouseID, SkuID: m.SkuID,
			Qty: m.Qty, UnitCost: unitCost.Yuan(), Amount: amount.Yuan(),
			SourceType: m.SourceType, SourceID: m.SourceID, SourceLineID: m.SourceLineID,
			OccurredAt: m.OccurredAt, Operator: m.Operator, ReversalOf: m.ReversalOf,
		}
		if err := tx.Create(&row).Error; err != nil {
			if isDuplicateKeyErr(err) {
				continue // 并发下的另一笔已经写进去了
			}
			return err
		}

		// 7) 更新物化余额
		if err := tx.Model(&jxc.StockBalance{}).
			Where("warehouse_id = ? AND sku_id = ?", m.WarehouseID, m.SkuID).
			Updates(map[string]interface{}{"qty": after.Qty, "value": after.Value.Yuan()}).Error; err != nil {
			return err
		}
	}
	return nil
}

// alreadyPosted 该来源行是否已过账
func (s *LedgerService) alreadyPosted(tx *gorm.DB, m MoveInput) (bool, error) {
	var cnt int64
	err := tx.Model(&jxc.StockMove{}).
		Where("source_type = ? AND source_id = ? AND source_line_id = ?", m.SourceType, m.SourceID, m.SourceLineID).
		Count(&cnt).Error
	return cnt > 0, err
}

// lockBalance 取余额行并加行锁；不存在则先建 0 行再取。
func (s *LedgerService) lockBalance(tx *gorm.DB, warehouseID, skuID uint) (ledger.Position, error) {
	var b jxc.StockBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).
		First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 先插入 0 行（并发下可能被别的 tx 抢先，用 INSERT IGNORE 语义）
		row := jxc.StockBalance{WarehouseID: warehouseID, SkuID: skuID}
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; e != nil {
			return ledger.Position{}, e
		}
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).
			First(&b).Error
	}
	if err != nil {
		return ledger.Position{}, err
	}
	return ledger.Position{Qty: b.Qty, Value: money.FromYuan(b.Value)}, nil
}

// reservedQty 当前有效预留总量
func (s *LedgerService) reservedQty(tx *gorm.DB, warehouseID, skuID uint) (int, error) {
	var sum int64
	err := tx.Model(&jxc.StockReservation{}).
		Where("warehouse_id = ? AND sku_id = ? AND status = ?", warehouseID, skuID, jxc.ReservationActive).
		Select("COALESCE(SUM(qty),0)").Scan(&sum).Error
	return int(sum), err
}

// ensurePeriodOpen 期间必须存在且未结账
func (s *LedgerService) ensurePeriodOpen(tx *gorm.DB, yearMonth string) error {
	var p jxc.AccountingPeriod
	err := tx.Where("period_month = ?", yearMonth).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row := jxc.AccountingPeriod{PeriodMonth: yearMonth, Status: jxc.PeriodOpen}
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; e != nil {
			return e
		}
		return nil
	}
	if err != nil {
		return err
	}
	if p.Status == jxc.PeriodClosed {
		return fmt.Errorf("%s 已结账，不能再写入账本（更正请在当期开红冲单）", yearMonth)
	}
	return nil
}

// ============ 预留 ============

// Reserve 预留库存（开单时占用），同一来源行重复调用幂等。
func (s *LedgerService) Reserve(ctx context.Context, db *gorm.DB, m MoveInput) error {
	if m.Qty <= 0 {
		return errors.New("预留数量必须大于 0")
	}
	fn := func(tx *gorm.DB) error {
		// 锁余额行，避免与并发的过账/预留互相穿插
		pos, err := s.lockBalance(tx, m.WarehouseID, m.SkuID)
		if err != nil {
			return err
		}
		var existing jxc.StockReservation
		err = tx.Where("source_type = ? AND source_id = ? AND source_line_id = ?",
			m.SourceType, m.SourceID, m.SourceLineID).First(&existing).Error
		if err == nil {
			return nil // 幂等
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		reserved, err := s.reservedQty(tx, m.WarehouseID, m.SkuID)
		if err != nil {
			return err
		}
		if pos.Qty-reserved < m.Qty {
			return fmt.Errorf("可售库存不足：结存 %d，已预留 %d，本次需要 %d", pos.Qty, reserved, m.Qty)
		}
		return tx.Create(&jxc.StockReservation{
			WarehouseID: m.WarehouseID, SkuID: m.SkuID, Qty: m.Qty,
			SourceType: m.SourceType, SourceID: m.SourceID, SourceLineID: m.SourceLineID,
			Status: jxc.ReservationActive,
		}).Error
	}
	if db != nil {
		return fn(db)
	}
	return global.GVA_DB.WithContext(ctx).Transaction(fn)
}

// ReleaseReservation 释放预留（取消单据时 status=Released，出库时 status=Consumed）。
//
// 必须支持传入事务 db：出库时"占用→转出"要和过账在同一个事务里，
// 走全局连接会出现"事务回滚了但预留已经被改掉"的脏状态。
func (s *LedgerService) ReleaseReservation(ctx context.Context, db *gorm.DB, sourceType string, sourceID, sourceLineID uint, status int8) error {
	if db == nil {
		db = global.GVA_DB.WithContext(ctx)
	}
	return db.Model(&jxc.StockReservation{}).
		Where("source_type = ? AND source_id = ? AND source_line_id = ? AND status = ?",
			sourceType, sourceID, sourceLineID, jxc.ReservationActive).
		Update("status", status).Error
}

// ============ 对账 ============

// Reconcile 校验某仓库（warehouseID=0 表示全部）的账本与余额是否一致。
// 这是"一键对账"的后端实现：返回空表示账是平的。
func (s *LedgerService) Reconcile(ctx context.Context, warehouseID uint) ([]ledger.Violation, error) {
	db := global.GVA_DB.WithContext(ctx)

	// 只把聚合结果读进内存：流水表可能有百万行，直接 Find 全部会 OOM，
	// 而对账只需要每个 (仓库, SKU) 的 SUM。索引 idx_move_wh_sku_time 支撑这个聚合。
	type balanceAgg struct {
		WarehouseID uint
		SkuID       uint
		Qty         int
		Value       float64
	}
	var balances []balanceAgg
	bq := db.Model(&jxc.StockBalance{}).
		Select("warehouse_id, sku_id, qty, value")
	if warehouseID > 0 {
		bq = bq.Where("warehouse_id = ?", warehouseID)
	}
	if err := bq.Scan(&balances).Error; err != nil {
		return nil, err
	}

	type moveAgg struct {
		WarehouseID uint
		SkuID       uint
		Qty         int
		Amount      float64
	}
	var moves []moveAgg
	mq := db.Model(&jxc.StockMove{}).
		Select("warehouse_id, sku_id, SUM(qty) AS qty, SUM(amount) AS amount").
		Group("warehouse_id, sku_id")
	if warehouseID > 0 {
		mq = mq.Where("warehouse_id = ?", warehouseID)
	}
	if err := mq.Scan(&moves).Error; err != nil {
		return nil, err
	}

	type reservationAgg struct {
		WarehouseID uint
		SkuID       uint
		Qty         int
	}
	var reservations []reservationAgg
	rq := db.Model(&jxc.StockReservation{}).
		Select("warehouse_id, sku_id, SUM(qty) AS qty").
		Where("status = ?", jxc.ReservationActive).
		Group("warehouse_id, sku_id")
	if warehouseID > 0 {
		rq = rq.Where("warehouse_id = ?", warehouseID)
	}
	if err := rq.Scan(&reservations).Error; err != nil {
		return nil, err
	}

	bs := make([]ledger.PositionRow, 0, len(balances))
	for _, b := range balances {
		bs = append(bs, ledger.PositionRow{WarehouseID: b.WarehouseID, SkuID: b.SkuID, Qty: b.Qty, Value: money.FromYuan(b.Value)})
	}
	es := make([]ledger.Entry, 0, len(moves))
	for _, m := range moves {
		es = append(es, ledger.Entry{WarehouseID: m.WarehouseID, SkuID: m.SkuID, Qty: m.Qty, Amount: money.FromYuan(m.Amount)})
	}
	rs := make([]ledger.ReservationRow, 0, len(reservations))
	for _, r := range reservations {
		rs = append(rs, ledger.ReservationRow{WarehouseID: r.WarehouseID, SkuID: r.SkuID, Qty: r.Qty, Status: jxc.ReservationActive})
	}
	return ledger.Check(bs, es, rs), nil
}

// BalanceRow 余额行（带 SKU 展示信息，供前端表格用）
type BalanceRow struct {
	WarehouseID   uint    `json:"warehouseId"`
	WarehouseName string  `json:"warehouseName"`
	SkuID         uint    `json:"skuId"`
	SkuCode       string  `json:"skuCode"`
	GoodsName     string  `json:"goodsName"`
	Color         string  `json:"color"`
	Size          string  `json:"size"`
	Qty           int     `json:"qty"`
	Reserved      int     `json:"reserved"`
	Available     int     `json:"available"`
	Value         float64 `json:"value"`
	AvgCost       float64 `json:"avgCost"`
}

// Balances 结存列表（含预留与可售），供"库存台账"页与对账页使用。
func (s *LedgerService) Balances(ctx context.Context, warehouseID uint) ([]BalanceRow, error) {
	db := global.GVA_DB.WithContext(ctx)
	// 预留量用相关子查询一次带出，避免每个结存行再查一次库（N+1）。
	// 台账页在 1 万个 SKU 时，旧写法会发 1 万次 SQL。
	q := db.Table("stock_balance b").
		Select(`b.warehouse_id, w.name AS warehouse_name, b.sku_id, s.sku_code,
		        g.name AS goods_name, s.color, s.size, b.qty, b.value,
		        COALESCE((SELECT SUM(r.qty) FROM stock_reservation r
		                  WHERE r.warehouse_id = b.warehouse_id AND r.sku_id = b.sku_id
		                    AND r.status = ?), 0) AS reserved`, jxc.ReservationActive).
		Joins("LEFT JOIN warehouse w ON w.id = b.warehouse_id").
		Joins("LEFT JOIN goods_sku s ON s.id = b.sku_id").
		Joins("LEFT JOIN goods g ON g.id = s.goods_id").
		Order("b.warehouse_id, b.sku_id")
	if warehouseID > 0 {
		q = q.Where("b.warehouse_id = ?", warehouseID)
	}
	var rows []BalanceRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Available = rows[i].Qty - rows[i].Reserved
		if rows[i].Qty > 0 {
			avg := money.FromYuan(rows[i].Value).Yuan() / float64(rows[i].Qty)
			rows[i].AvgCost = money.FromYuan(avg).Yuan()
		}
	}
	return rows, nil
}

// MoveRow 一条账本流水。**面向展示**，不是在 StockMove 上直接返回：
//   - 账本的 Qty/Amount 是有符号的（入正出负），这是不变量成立的前提；
//   - 但给人看的表格里，"金额"是成本金额，"数量"要带方向，
//     直接把负数金额摆出来会被读成"卖了 -264 元"，属于语义误导。
//   - 裸的 sku_id / sale_out#2 也没法读，这里补齐商品名、规格、单据号。
type MoveRow struct {
	ID          uint    `json:"ID"`
	WarehouseID uint    `json:"warehouseId"`
	Warehouse   string  `json:"warehouseName"`
	SkuID       uint    `json:"skuId"`
	GoodsName   string  `json:"goodsName"`
	SkuCode     string  `json:"skuCode"`
	Color       string  `json:"color"`
	Size        string  `json:"size"`
	Qty         int     `json:"qty"` // 有符号：入库正、出库负
	UnitCost    float64 `json:"unitCost"`
	Amount      float64 `json:"amount"` // 有符号金额（账本原始语义，累加等于余额金额）
	// Cost 成本金额，恒为正。
	// 为什么单独给一个字段：展示层需要的是"这笔移动涉及多少成本"，
	// 直接渲染 Amount 会把出库显示成负数金额，被读成"卖亏了"。
	// 与其让每个调用方自己 Math.abs()（迟早有人忘），不如在这里给出正确语义。
	Cost       float64   `json:"cost"`
	SourceType string    `json:"sourceType"`
	SourceID   uint      `json:"sourceId"`
	SourceNo   string    `json:"sourceNo"`
	OccurredAt time.Time `json:"occurredAt"`
	Operator   string    `json:"operator"`
	ReversalOf *uint     `json:"reversalOf"`
	// BalanceAfter 这笔移动之后的结存数量。
	// 仅当查询指定了 warehouseId + skuId 时返回；跨 SKU 混排没有"结存"语义。
	// 由后端计算而不是前端累加：前端只看得到当前这一页，按页累加会算错。
	BalanceAfter *int `json:"balanceAfter,omitempty"`
}

// 单据号来源表：把账本行回溯到"人认识的那个单号"
var sourceNoTable = map[string]struct {
	table  string
	column string
}{
	jxc.MoveSourcePurchaseIn:     {"purchase_order", "order_no"},
	jxc.MoveSourcePurchaseReturn: {"purchase_return", "return_no"},
	jxc.MoveSourceSaleOut:        {"sale_order", "order_no"},
	jxc.MoveSourceSaleReturn:     {"sale_return", "return_no"},
	jxc.MoveSourceCheckGain:      {"stock_check", "check_no"},
	jxc.MoveSourceCheckLoss:      {"stock_check", "check_no"},
	jxc.MoveSourceReversal:       {"stock_reversal", "reversal_no"},
}

// Moves 账本流水（分页，按发生时间倒序）
func (s *LedgerService) Moves(ctx context.Context, warehouseID, skuID uint, page, size int) ([]MoveRow, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 200 {
		size = 50
	}
	db := global.GVA_DB.WithContext(ctx)
	q := db.Table("stock_move m").
		Select(`m.id, m.warehouse_id, w.name AS warehouse, m.sku_id,
		        g.name AS goods_name, sk.sku_code, sk.color, sk.size,
		        m.qty, m.unit_cost, m.amount,
		        m.source_type, m.source_id, m.occurred_at, m.operator, m.reversal_of`).
		Joins("LEFT JOIN warehouse w ON w.id = m.warehouse_id").
		Joins("LEFT JOIN goods_sku sk ON sk.id = m.sku_id").
		Joins("LEFT JOIN goods g ON g.id = sk.goods_id")
	if warehouseID > 0 {
		q = q.Where("m.warehouse_id = ?", warehouseID)
	}
	if skuID > 0 {
		q = q.Where("m.sku_id = ?", skuID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []MoveRow
	if err := q.Order("m.id desc").Offset((page - 1) * size).Limit(size).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	// 补"这笔之后还剩多少"。余额是按过账顺序（id）累加的，所以：
	// 本页第一行之后的结存 = 当前结存 - 比本页第一行更新的所有移动。
	// 这样翻到第 2 页也不会算错（旧前端只拿当前页从 0 累加，超过一页就是错的）。
	if warehouseID > 0 && skuID > 0 && len(rows) > 0 {
		var balance jxc.StockBalance
		if err := db.Where("warehouse_id = ? AND sku_id = ?", warehouseID, skuID).
			First(&balance).Error; err == nil {
			var newerQty int64
			if err := db.Model(&jxc.StockMove{}).
				Where("warehouse_id = ? AND sku_id = ? AND id > ?", warehouseID, skuID, rows[0].ID).
				Select("COALESCE(SUM(qty),0)").Scan(&newerQty).Error; err != nil {
				return nil, 0, err
			}
			running := balance.Qty - int(newerQty)
			for i := range rows {
				v := running
				rows[i].BalanceAfter = &v
				running -= rows[i].Qty
			}
		}
	}

	// 批量补单据号：按来源类型分组查，避免 N+1
	pairs := map[string]map[uint]string{}
	columns := map[string]string{}
	for _, r := range rows {
		meta, ok := sourceNoTable[r.SourceType]
		if !ok || r.SourceID == 0 {
			continue
		}
		if pairs[meta.table] == nil {
			pairs[meta.table] = map[uint]string{}
			columns[meta.table] = meta.column
		}
		pairs[meta.table][r.SourceID] = ""
	}
	for table, ids := range pairs {
		idList := make([]uint, 0, len(ids))
		for id := range ids {
			idList = append(idList, id)
		}
		type noRow struct {
			ID      uint
			OrderNo string
		}
		var nos []noRow
		col := columns[table]
		if err := db.Table(table).Select("id, "+col+" AS order_no").Where("id IN ?", idList).Scan(&nos).Error; err != nil {
			return nil, 0, err
		}
		for _, n := range nos {
			pairs[table][n.ID] = n.OrderNo
		}
	}
	for i := range rows {
		if meta, ok := sourceNoTable[rows[i].SourceType]; ok {
			rows[i].SourceNo = pairs[meta.table][rows[i].SourceID]
		}
		if rows[i].Amount < 0 {
			rows[i].Cost = -rows[i].Amount
		} else {
			rows[i].Cost = rows[i].Amount
		}
	}
	return rows, total, nil
}

// isDuplicateKeyErr 判定唯一键冲突（MySQL 1062 / sqlite UNIQUE）
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key")
}

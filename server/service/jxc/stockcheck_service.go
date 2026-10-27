package jxc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StockCheckService 盘点。
//
// 流程：建单（快照账面数）→ 录实盘数 → 完成（差异过账）。
// 只有"完成"这一步会动账本，且差异是**追加移动**而不是覆盖余额。
type StockCheckService struct {
	ledger *LedgerService
}

func NewStockCheckService() *StockCheckService {
	return &StockCheckService{ledger: new(LedgerService)}
}

// Create 建盘点单：把当前结存快照成明细，实盘数留空等人工录入。
func (s *StockCheckService) Create(ctx context.Context, warehouseID uint, remark string) (*jxc.StockCheck, error) {
	if warehouseID == 0 {
		return nil, errors.New("请选择盘点仓库")
	}
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureWarehouse(db, warehouseID); err != nil {
		return nil, err
	}

	var check *jxc.StockCheck
	err := db.Transaction(func(tx *gorm.DB) error {
		// 同一仓库不允许有未完成的盘点单，否则两次盘点会互相覆盖。
		//
		// "先 count 再 insert" 在并发下不是原子的：两个请求都读到 0，就会各建一张。
		// MySQL 没有条件唯一索引（status=1 时才唯一），所以用仓库行锁做串行化：
		// 同一仓库的建单请求在这里排队，后到的一定能看到先到者已提交的盘点单。
		// 锁顺序与 Ledger.Post（按仓库/SKU）一致，不会引入新的死锁路径。
		var wh jxc.Warehouse
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&wh, warehouseID).Error; err != nil {
			return errors.New("仓库不存在")
		}
		var open int64
		if err := tx.Model(&jxc.StockCheck{}).
			Where("warehouse_id = ? AND status = ?", warehouseID, jxc.StockCheckDraft).
			Count(&open).Error; err != nil {
			return err
		}
		if open > 0 {
			return errors.New("该仓库已有进行中的盘点单，请先完成或取消")
		}

		no, err := nextOrderNo(tx, "PD")
		if err != nil {
			return err
		}
		check = &jxc.StockCheck{CheckNo: no, WarehouseID: warehouseID, Status: jxc.StockCheckDraft, Remark: remark}
		if err := tx.Create(check).Error; err != nil {
			return err
		}

		// 盘点要覆盖**全部在售 SKU**，而不只是有结存的行。
		// 两个理由：① 期初建账时所有 SKU 结存都是 0，若只取有结存的行，
		// 根本没有行可录，"盘盈入期初"这条路就断了；
		// ② 账面为 0 但实际有货（漏记入库）也必须能被盘出来。
		type row struct {
			SkuID uint
			Qty   int
			Value float64
		}
		var rows []row
		err = tx.Table("goods_sku s").
			Select("s.id AS sku_id, COALESCE(b.qty,0) AS qty, COALESCE(b.value,0) AS value").
			Joins("LEFT JOIN stock_balance b ON b.sku_id = s.id AND b.warehouse_id = ?", warehouseID).
			Where("s.status = 1 AND s.deleted_at IS NULL").
			Order("s.id").Scan(&rows).Error
		if err != nil {
			return err
		}
		for _, r := range rows {
			item := jxc.StockCheckItem{CheckID: check.ID, SkuID: r.SkuID, BookQty: r.Qty}
			if r.Qty > 0 {
				item.UnitCost = money.FromYuan(r.Value).Yuan() / float64(r.Qty)
				item.UnitCost = money.FromYuan(item.UnitCost).Yuan()
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, check.ID)
}

// UpdateItems 录入实盘数。只允许盘点中的单。
func (s *StockCheckService) UpdateItems(ctx context.Context, checkID uint, inputs []StockCountInput) error {
	if len(inputs) == 0 {
		return errors.New("请至少录入一条实盘数")
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var check jxc.StockCheck
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&check, checkID).Error; err != nil {
			return errors.New("盘点单不存在")
		}
		if check.Status != jxc.StockCheckDraft {
			return errors.New("只有盘点中的单据可以录入")
		}
		for _, in := range inputs {
			if in.ActualQty < 0 {
				return errors.New("实盘数不能为负")
			}
			if in.UnitCost != nil && *in.UnitCost < 0 {
				return errors.New("成本单价不能为负")
			}
			var item jxc.StockCheckItem
			if err := tx.Where("id = ? AND check_id = ?", in.ID, checkID).First(&item).Error; err != nil {
				return errors.New("盘点明细不存在")
			}
			actual := in.ActualQty
			updates := map[string]interface{}{
				"actual_qty": actual,
				"diff":       actual - item.BookQty,
			}
			// 期初建账/盘盈时账面成本为 0，必须允许人工指定成本，
			// 否则库存数量进来了、金额永远是 0，卖出去就是 100% 毛利。
			if in.UnitCost != nil {
				updates["unit_cost"] = money.FromYuan(*in.UnitCost).Yuan()
			}
			if err := tx.Model(&jxc.StockCheckItem{}).Where("id = ?", item.ID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// StockCountInput 一条实盘录入。
// UnitCost 可选：盘盈/期初建账时指定入库成本；不传则沿用建单时的账面均价。
type StockCountInput struct {
	ID        uint     `json:"id"`
	ActualQty int      `json:"actualQty"`
	UnitCost  *float64 `json:"unitCost,omitempty"`
}

// Post 完成盘点：把差异写进账本（盘盈正数、盘亏负数），然后锁死单据。
func (s *StockCheckService) Post(ctx context.Context, checkID uint, operator string, occurredAt time.Time) (*jxc.StockCheck, error) {
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	var out *jxc.StockCheck
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var check jxc.StockCheck
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&check, checkID).Error; err != nil {
			return errors.New("盘点单不存在")
		}
		if check.Status != jxc.StockCheckDraft {
			return errors.New("只有盘点中的单据可以完成")
		}
		var items []jxc.StockCheckItem
		if err := tx.Where("check_id = ?", checkID).Find(&items).Error; err != nil {
			return err
		}

		// 盘点建单时快照的账面数，可能在录入期间又被销售/入库改掉了。
		// 此时按旧 diff 过账会把这段时间的真实变化抹掉，所以先锁住余额并核对。
		// 加锁顺序按 SKU，与 Ledger.Post 一致。
		var bals []jxc.StockBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("warehouse_id = ?", check.WarehouseID).Order("sku_id").Find(&bals).Error; err != nil {
			return err
		}
		book := make(map[uint]int, len(bals))
		for _, b := range bals {
			book[b.SkuID] = b.Qty
		}
		for _, it := range items {
			if it.ActualQty == nil {
				continue
			}
			if cur := book[it.SkuID]; cur != it.BookQty {
				return fmt.Errorf("SKU %d 的账面数在建单后已变化（快照 %d，当前 %d），请取消并重新盘点",
					it.SkuID, it.BookQty, cur)
			}
		}

		var moves []MoveInput
		counted := 0
		for _, it := range items {
			if it.ActualQty == nil {
				continue // 没录的不动
			}
			counted++
			if it.Diff == 0 {
				continue
			}
			src := jxc.MoveSourceCheckGain
			if it.Diff < 0 {
				src = jxc.MoveSourceCheckLoss
			}
			moves = append(moves, MoveInput{
				WarehouseID: check.WarehouseID, SkuID: it.SkuID,
				Qty: it.Diff, UnitCost: it.UnitCost,
				SourceType: src, SourceID: check.ID, SourceLineID: it.ID,
				OccurredAt: occurredAt, Operator: operator,
			})
		}
		if counted == 0 {
			return errors.New("还没有录入任何实盘数")
		}
		if len(moves) > 0 {
			if err := s.ledger.Post(ctx, tx, moves); err != nil {
				return err
			}
		}

		postedAt := occurredAt
		if err := tx.Model(&jxc.StockCheck{}).Where("id = ?", checkID).
			Updates(map[string]interface{}{"status": jxc.StockCheckPosted, "posted_at": postedAt}).Error; err != nil {
			return err
		}
		out = &check
		out.Status = jxc.StockCheckPosted
		out.PostedAt = &postedAt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, out.ID)
}

// Cancel 取消盘点（仅盘点中）
func (s *StockCheckService) Cancel(ctx context.Context, checkID uint) error {
	res := global.GVA_DB.WithContext(ctx).Model(&jxc.StockCheck{}).
		Where("id = ? AND status = ?", checkID, jxc.StockCheckDraft).
		Update("status", jxc.StockCheckCanceled)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("只有盘点中的单据可以取消")
	}
	return nil
}

// Get 详情（带明细与 SKU 展示信息）
func (s *StockCheckService) Get(ctx context.Context, id uint) (*jxc.StockCheck, error) {
	var check jxc.StockCheck
	if err := global.GVA_DB.WithContext(ctx).Preload("Items").First(&check, id).Error; err != nil {
		return nil, errors.New("盘点单不存在")
	}
	return &check, nil
}

// Page 列表
func (s *StockCheckService) Page(ctx context.Context, page, size int) ([]jxc.StockCheck, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.StockCheck{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []jxc.StockCheck
	err := db.Preload("Items").Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

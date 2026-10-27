package jxc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReversalService 红冲：整单撤销一张已过账的单据。
//
// 为什么必须有它：账本行的不可变性是这套设计的地基，但"不可变"如果没有出口，
// 就等于"录错了没法救"。红冲就是这个出口——不改历史，只追加一条反向事实。
//
// 两条硬约束：
//  1. 已红冲的单不能再红冲（唯一性靠 stock_reversal 的业务校验 + 账本行幂等键）；
//  2. 采购红冲后不能把库存冲成负数（货都卖了还整单红冲，账就假了）。
type ReversalService struct {
	ledger *LedgerService
}

func NewReversalService() *ReversalService {
	return &ReversalService{ledger: new(LedgerService)}
}

// Reverse 对一张采购单/销售单做整单红冲。
func (s *ReversalService) Reverse(ctx context.Context, sourceType string, sourceID uint, reason, operator string) (*jxc.StockReversal, error) {
	if sourceID == 0 {
		return nil, errors.New("请选择要红冲的单据")
	}
	if sourceType != jxc.ReversalSourcePurchase && sourceType != jxc.ReversalSourceSale {
		return nil, errors.New("红冲来源只能是采购单或销售单")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errors.New("请填写红冲原因（这是日后解释账目变化的唯一线索）")
	}

	var rev *jxc.StockReversal
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1) 校验原单据状态，并锁住它
		var (
			docNo   string
			moveSrc string
			docID   uint
		)
		switch sourceType {
		case jxc.ReversalSourcePurchase:
			var po jxc.PurchaseOrder
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&po, sourceID).Error; err != nil {
				return errors.New("采购单不存在")
			}
			if po.Status != jxc.PurchaseStatusReceived {
				return errors.New("只有已入库的采购单可以红冲")
			}
			docNo, moveSrc, docID = po.OrderNo, jxc.MoveSourcePurchaseIn, po.ID
		case jxc.ReversalSourceSale:
			var so jxc.SaleOrder
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&so, sourceID).Error; err != nil {
				return errors.New("销售单不存在")
			}
			if so.Status != jxc.SaleStatusShipped {
				return errors.New("只有已出库的销售单可以红冲")
			}
			// 已有退货的销售单不能整单红冲：退货已经回补过库存，再红冲会重复回补。
			// 草稿退货也一并挡住——否则红冲后这张草稿永远无法入库，只能取消，属于埋雷。
			var retCnt int64
			if err := tx.Model(&jxc.SaleReturn{}).
				Where("sale_id = ? AND status IN ?", so.ID,
					[]int8{jxc.ReturnStatusDraft, jxc.ReturnStatusPosted}).
				Count(&retCnt).Error; err != nil {
				return err
			}
			if retCnt > 0 {
				return errors.New("该销售单已存在退货单（草稿或已入库），不能整单红冲；请先取消退货单，或改用退货处理")
			}
			docNo, moveSrc, docID = so.OrderNo, jxc.MoveSourceSaleOut, so.ID
		}

		// 2) 不能重复红冲
		var exist int64
		if err := tx.Model(&jxc.StockReversal{}).
			Where("source_type = ? AND source_id = ?", sourceType, docID).Count(&exist).Error; err != nil {
			return err
		}
		if exist > 0 {
			return errors.New("该单据已经红冲过了")
		}

		// 3) 找出原始账本行
		var orig []jxc.StockMove
		if err := tx.Where("source_type = ? AND source_id = ?", moveSrc, docID).
			Order("id").Find(&orig).Error; err != nil {
			return err
		}
		if len(orig) == 0 {
			return errors.New("未找到该单据的账本记录，无法红冲")
		}

		// 4) 采购红冲会让库存减少：不能冲成负数
		if sourceType == jxc.ReversalSourcePurchase {
			need := map[uint]int{}
			whID := uint(0)
			for _, m := range orig {
				// 采购入库的账本行 qty 为正，红冲要扣掉的就是这个正数
				need[m.SkuID] += m.Qty
				whID = m.WarehouseID
			}
			for skuID, q := range need {
				var b jxc.StockBalance
				if err := tx.Where("warehouse_id = ? AND sku_id = ?", whID, skuID).First(&b).Error; err != nil {
					return fmt.Errorf("查不到结存，无法红冲：SKU %d", skuID)
				}
				reserved, err := s.ledger.reservedQty(tx, whID, skuID)
				if err != nil {
					return err
				}
				if b.Qty-reserved < q {
					return fmt.Errorf("该批货已卖出（SKU %d 可售 %d，需冲回 %d），不能整单红冲；请改用盘亏或退货处理",
						skuID, b.Qty-reserved, q)
				}
			}
		}

		// 5) 建红冲单（先拿到 ID 作为账本行的来源）
		var qtySum int
		var amtSum money.Cents
		for _, m := range orig {
			qtySum += absInt(m.Qty)
			amtSum = amtSum.Add(money.FromYuan(m.Amount).Abs())
		}
		no, err := nextOrderNo(tx, "RC")
		if err != nil {
			return err
		}
		rev = &jxc.StockReversal{
			ReversalNo: no, SourceType: sourceType, SourceID: docID, SourceNo: docNo,
			Reason: reason, Amount: amtSum.Yuan(), Qty: qtySum,
			Status: jxc.ReversalPosted, PostedAt: time.Now(), Operator: operator,
		}
		if err := tx.Create(rev).Error; err != nil {
			return err
		}

		// 6) 反向记账：数量取反，成本沿用原值（这样金额精确抵消，结存回到红冲前）
		moves := make([]MoveInput, 0, len(orig))
		for _, m := range orig {
			origID := m.ID
			moves = append(moves, MoveInput{
				WarehouseID: m.WarehouseID, SkuID: m.SkuID,
				Qty: -m.Qty, UnitCost: m.UnitCost,
				SourceType: jxc.MoveSourceReversal, SourceID: rev.ID, SourceLineID: origID,
				OccurredAt: rev.PostedAt, Operator: operator, ReversalOf: &origID,
			})
		}
		if err := s.ledger.Post(ctx, tx, moves); err != nil {
			return err
		}

		// 7) 标记原单据（保留状态，只加时间戳，避免状态码语义膨胀）
		now := time.Now()
		table := "purchase_order"
		if sourceType == jxc.ReversalSourceSale {
			table = "sale_order"
		}
		return tx.Table(table).Where("id = ?", docID).Update("reversed_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return rev, nil
}

// Page 红冲记录
func (s *ReversalService) Page(ctx context.Context, page, size int) ([]jxc.StockReversal, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.StockReversal{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []jxc.StockReversal
	err := db.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

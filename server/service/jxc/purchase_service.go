package jxc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	common "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PurchaseService 采购单：负责业务规则与状态机，**不直接碰库存**。
// 落账一律交给 LedgerService，单据层只负责"组装 MoveInput"。
type PurchaseService struct {
	ledger *LedgerService
}

func NewPurchaseService() *PurchaseService {
	return &PurchaseService{ledger: new(LedgerService)}
}

// Create 创建采购单（草稿）。审核之前不动库存。
func (s *PurchaseService) Create(ctx context.Context, order *jxc.PurchaseOrder) (*jxc.PurchaseOrder, error) {
	if len(order.Items) == 0 {
		return nil, errors.New("采购单至少需要一条明细")
	}
	if order.WarehouseID == 0 {
		return nil, errors.New("请选择入库仓库")
	}
	db := global.GVA_DB.WithContext(ctx)
	if err := ensureWarehouse(db, order.WarehouseID); err != nil {
		return nil, err
	}
	if order.SupplierID != nil && *order.SupplierID > 0 {
		var cnt int64
		if err := db.Model(&jxc.Partner{}).
			Where("id = ? AND kind = ?", *order.SupplierID, jxc.PartnerSupplier).Count(&cnt).Error; err != nil {
			return nil, err
		}
		if cnt == 0 {
			return nil, errors.New("供应商不存在")
		}
	}
	total, err := normalizePurchaseItems(db, order.Items)
	if err != nil {
		return nil, err
	}
	// 忽略客户端传入的主键/单号/过账字段：这些必须由服务端决定
	order.ID = 0
	order.OrderNo = ""
	order.Status = jxc.PurchaseStatusDraft
	order.PostedAt = nil
	order.ReversedAt = nil
	order.TotalAmount = total
	for i := range order.Items {
		order.Items[i].ID = 0
		order.Items[i].PurchaseID = 0
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		no, err := nextOrderNo(tx, "PO")
		if err != nil {
			return err
		}
		order.OrderNo = no
		return tx.Create(order).Error
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

// Approve 审核（草稿 → 已审核）。仍不动库存。
func (s *PurchaseService) Approve(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ok, err := advanceDocStatus(tx, &jxc.PurchaseOrder{}, id,
			jxc.PurchaseStatusDraft, jxc.PurchaseStatusApproved)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("仅草稿状态的采购单可以审核")
		}
		return nil
	})
}

// Receive 入库（已审核 → 已入库）：**这里才产生账本行**。
//
// 事务内顺序：锁单据 → 校验状态 → 组装账本行 → 过账 → 推进状态。
// 任何一步失败整体回滚，不会出现"库存加了但单据没变"。
func (s *PurchaseService) Receive(ctx context.Context, id uint, operator string, occurredAt time.Time) error {
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.PurchaseOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return errors.New("采购单不存在")
		}
		if order.Status != jxc.PurchaseStatusApproved {
			return fmt.Errorf("仅已审核的采购单可以入库（当前状态 %d）", order.Status)
		}
		var items []jxc.PurchaseItem
		if err := tx.Where("purchase_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("采购单没有明细")
		}

		moves := make([]MoveInput, 0, len(items))
		for _, it := range items {
			moves = append(moves, MoveInput{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID,
				Qty: it.Qty, UnitCost: it.UnitCost,
				SourceType: jxc.MoveSourcePurchaseIn, SourceID: order.ID, SourceLineID: it.ID,
				OccurredAt: occurredAt, Operator: operator,
			})
		}
		if err := s.ledger.Post(ctx, tx, moves); err != nil {
			return err
		}

		ok, err := advanceDocStatus(tx, &jxc.PurchaseOrder{}, id,
			jxc.PurchaseStatusApproved, jxc.PurchaseStatusReceived)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("单据状态已变更，请刷新后重试")
		}
		return tx.Model(&jxc.PurchaseOrder{}).Where("id = ?", id).Update("posted_at", occurredAt).Error
	})
}

// Cancel 取消：草稿/已审核可取消；**已入库不可取消**（要用红冲，见账本设计）。
func (s *PurchaseService) Cancel(ctx context.Context, id uint, operator string) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order jxc.PurchaseOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return errors.New("采购单不存在")
		}
		switch order.Status {
		case jxc.PurchaseStatusDraft, jxc.PurchaseStatusApproved:
		case jxc.PurchaseStatusReceived:
			return errors.New("已入库的采购单不能取消；如需更正，请在当期开红冲单")
		default:
			return errors.New("当前状态不可取消")
		}
		return tx.Model(&jxc.PurchaseOrder{}).Where("id = ?", id).Update("status", jxc.PurchaseStatusCanceled).Error
	})
}

// Get 详情
func (s *PurchaseService) Get(ctx context.Context, id uint) (*jxc.PurchaseOrder, error) {
	var order jxc.PurchaseOrder
	err := global.GVA_DB.WithContext(ctx).Preload("Items").First(&order, id).Error
	if err != nil {
		return nil, errors.New("采购单不存在")
	}
	return &order, nil
}

// Page 列表（兼容旧调用；新代码用 Search）
func (s *PurchaseService) Page(ctx context.Context, page, size int) ([]jxc.PurchaseOrder, int64, error) {
	return s.Search(ctx, jxcReq.DocumentSearch{
		PageInfo: common.PageInfo{Page: page, PageSize: size},
	})
}

// Search 服务端筛选 + 分页。
//
// 关键词覆盖单号、备注、供应商名称/编码；状态、仓库、供应商、日期范围、是否红冲
// 都可组合。total 是数据库的真实总数，不再是前端取回 100 条后的长度。
func (s *PurchaseService) Search(ctx context.Context, info jxcReq.DocumentSearch) ([]jxc.PurchaseOrder, int64, error) {
	info.Normalize()
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.PurchaseOrder{}).
		Joins("LEFT JOIN jxc_partner p ON p.id = purchase_order.supplier_id")

	if info.Status != nil {
		db = db.Where("purchase_order.status = ?", *info.Status)
	}
	if info.WarehouseID > 0 {
		db = db.Where("purchase_order.warehouse_id = ?", info.WarehouseID)
	}
	if info.PartnerID > 0 {
		db = db.Where("purchase_order.supplier_id = ?", info.PartnerID)
	}
	if info.Reversed != nil {
		if *info.Reversed {
			db = db.Where("purchase_order.reversed_at IS NOT NULL")
		} else {
			db = db.Where("purchase_order.reversed_at IS NULL")
		}
	}
	if kw := keywordPattern(info.Keyword); kw != "" {
		db = db.Where("(purchase_order.order_no LIKE ? OR purchase_order.remark LIKE ? OR p.name LIKE ? OR p.code LIKE ?)",
			kw, kw, kw, kw)
	}
	db, err := applyDocumentDateRange(db, info, "purchase_order.created_at", "purchase_order.posted_at")
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit, offset := info.LimitOffset()
	var list []jxc.PurchaseOrder
	err = db.Preload("Items").Order("purchase_order.id desc").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

// ============ 内部工具 ============

// normalizePurchaseItems 校验并归一化明细，返回总额（分制累加，落库 2 位小数）
func normalizePurchaseItems(db *gorm.DB, items []jxc.PurchaseItem) (float64, error) {
	var total money.Cents
	for i := range items {
		it := &items[i]
		if it.SkuID == 0 {
			return 0, errors.New("明细缺少 SKU")
		}
		if it.Qty <= 0 {
			return 0, errors.New("采购数量必须大于 0")
		}
		if it.UnitCost < 0 {
			return 0, errors.New("采购单价不能为负")
		}
		if err := ensureSku(db, it.SkuID); err != nil {
			return 0, err
		}
		unit := money.FromYuan(it.UnitCost)
		amount := unit.MulQty(it.Qty)
		it.UnitCost = unit.Yuan()
		it.Amount = amount.Yuan()
		total = total.Add(amount)
	}
	return total.Yuan(), nil
}

// nextOrderNo 生成 PREFIX-YYYYMMDD-### 形式的单号。
//
// 必须传入事务：发号器行锁要一直持有到单据落库提交，否则并发下仍可能拿到同号。
// 旧实现扫描业务表求 MAX，在并发插入时会互相加间隙锁并死锁（实测 8 个并发开单
// 有 7 个失败），而且 REPEATABLE READ 下重试读不到对方未提交的行。
func nextOrderNo(tx *gorm.DB, prefix string) (string, error) {
	date := time.Now().Format("20060102")
	scope := prefix + "-" + date

	created := tx.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&jxc.DocSequence{Scope: scope, NextValue: 1})
	if created.Error != nil {
		return "", created.Error
	}
	// 发号器是本次升级新增的表。老库升级后第一次发号时，表里还没有
	// (prefix, 日期) 这一行，必须先从历史单号里找到当日最大序号，
	// 否则会从 001 重新开始，直接撞上历史单号（线上已真实发生）。
	if created.RowsAffected > 0 {
		start, err := existingMaxSeq(tx, prefix, date)
		if err != nil {
			return "", err
		}
		if start > 0 {
			if err := tx.Model(&jxc.DocSequence{}).Where("scope = ?", scope).
				Update("next_value", start).Error; err != nil {
				return "", err
			}
		}
	}

	var seq jxc.DocSequence
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("scope = ?", scope).First(&seq).Error; err != nil {
		return "", err
	}
	next := seq.NextValue
	if next < 1 {
		next = 1
	}
	if err := tx.Model(&jxc.DocSequence{}).Where("scope = ?", scope).
		Update("next_value", next+1).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%03d", prefix, date, next), nil
}

// existingMaxSeq 查历史单号里同前缀、同一天的最大序号 + 1。
// 用 Unscoped：软删除的单据仍然占着唯一索引，不能把它的号发出去。
func existingMaxSeq(tx *gorm.DB, prefix, date string) (int, error) {
	meta, ok := orderNoTables[prefix]
	if !ok {
		return 1, nil
	}
	like := prefix + "-" + date + "-%"
	var maxNo string
	// 先按长度再按字典序：001..999 补齐三位，到 1000 以后长度会多一位，
	// 单纯按字符串排序会把 999 当成最大。
	if err := tx.Unscoped().Table(meta.table).
		Where(meta.column+" LIKE ?", like).
		Order("LENGTH("+meta.column+") DESC, "+meta.column+" DESC").Limit(1).
		Pluck(meta.column, &maxNo).Error; err != nil {
		return 0, err
	}
	if maxNo == "" {
		return 1, nil
	}
	idx := strings.LastIndex(maxNo, "-")
	if idx < 0 || idx == len(maxNo)-1 {
		return 1, nil
	}
	n, err := strconv.Atoi(maxNo[idx+1:])
	if err != nil || n < 0 {
		return 1, nil
	}
	return n + 1, nil
}

var orderNoTables = map[string]struct {
	table  string
	column string
}{
	"PO": {"purchase_order", "order_no"},
	"PR": {"purchase_return", "return_no"},
	"SO": {"sale_order", "order_no"},
	"RT": {"sale_return", "return_no"},
	"PD": {"stock_check", "check_no"},
	"RC": {"stock_reversal", "reversal_no"},
}

// advanceDocStatus 条件推进单据状态：并发下只有一个请求能成功。
func advanceDocStatus(tx *gorm.DB, model any, id uint, from, to int8) (bool, error) {
	res := tx.Model(model).Where("id = ? AND status = ?", id, from).Update("status", to)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func ensureWarehouse(db *gorm.DB, id uint) error {
	var cnt int64
	if err := db.Model(&jxc.Warehouse{}).Where("id = ?", id).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("仓库不存在：%d", id)
	}
	return nil
}

func ensureSku(db *gorm.DB, id uint) error {
	var cnt int64
	if err := db.Model(&jxc.GoodsSku{}).Where("id = ?", id).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("SKU 不存在：%d", id)
	}
	return nil
}

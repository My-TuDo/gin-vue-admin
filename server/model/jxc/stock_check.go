package jxc

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 盘点：把"账上有多少"和"实际有多少"对一次，差异进账本。
//
// 关键取舍：盘点**不是**直接改余额，而是产生一条盘盈/盘亏的账本移动。
// 旧实现是 `UPDATE stock SET quantity = actual`——这一改，账本与余额的因果关系就断了，
// 事后没人能解释"这 3 件是怎么没的"。
const (
	StockCheckDraft    = 1 // 盘点中
	StockCheckPosted   = 2 // 已完成（差异已过账）
	StockCheckCanceled = 3 // 已取消
)

// StockCheck 盘点单
type StockCheck struct {
	global.GVA_MODEL
	CheckNo     string     `json:"checkNo" gorm:"column:check_no;size:50;not null;uniqueIndex:uk_check_no"`
	WarehouseID uint       `json:"warehouseId" gorm:"column:warehouse_id;not null"`
	Status      int8       `json:"status" gorm:"column:status;not null;default:1;comment:1盘点中 2已完成 3已取消"`
	PostedAt    *time.Time `json:"postedAt" gorm:"column:posted_at"`
	Remark      string     `json:"remark" gorm:"column:remark;size:500"`

	Items []StockCheckItem `json:"items" gorm:"foreignKey:CheckID;references:ID"`
}

func (StockCheck) TableName() string { return "stock_check" }

// StockCheckItem 盘点明细。
// BookQty 是建单那一刻的账面快照，Diff 在过账时按"实盘 - 账面"生成账本移动。
type StockCheckItem struct {
	ID        uint    `json:"ID" gorm:"primarykey"`
	CheckID   uint    `json:"checkId" gorm:"column:check_id;not null;index"`
	SkuID     uint    `json:"skuId" gorm:"column:sku_id;not null"`
	BookQty   int     `json:"bookQty" gorm:"column:book_qty;not null;default:0;comment:账面数（建单快照）"`
	ActualQty *int    `json:"actualQty" gorm:"column:actual_qty;comment:实盘数（未录为空）"`
	Diff      int     `json:"diff" gorm:"column:diff;not null;default:0;comment:差异=实盘-账面"`
	UnitCost  float64 `json:"unitCost" gorm:"column:unit_cost;type:decimal(12,2);not null;default:0;comment:过账时使用的成本"`
}

func (StockCheckItem) TableName() string { return "stock_check_item" }

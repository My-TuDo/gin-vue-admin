package jxc

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 红冲（reversal）与退货（return）的模型。
//
// 这两个概念不能混：
//   - 红冲：单据**录错了**，整单撤销，账本上留一条反向移动做证据，原单据作废；
//   - 退货：客户**真的退了货**，是一笔新业务，数量可以少于原单，公司真的退了钱。
//
// 把它们合成一个"负销售单"是很多进销存的通病：报表里分不清"卖得少"和"录错了"，
// 毛利被两种完全不同的原因同时扭曲。这里分成两张表、两种来源类型。

// 红冲来源（被冲销的单据类型）
const (
	ReversalSourcePurchase       = "purchase"
	ReversalSourceSale           = "sale"
	ReversalSourcePurchaseReturn = "purchase_return"
)

// 红冲单状态：红冲是即时生效的，没有草稿态（避免"半吊子红冲"挂在半空）
const (
	ReversalPosted = 1
)

// StockReversal 红冲单：对一张已过账单据的整单撤销。
type StockReversal struct {
	global.GVA_MODEL
	ReversalNo string `json:"reversalNo" gorm:"column:reversal_no;size:50;not null;uniqueIndex:uk_rev_no"`
	// (source_type, source_id) 唯一：一张单据理论上只能有一张红冲单。
	// 应用层已经在锁住原单后做了检查，这条唯一索引是最后一道保险。
	SourceType string    `json:"sourceType" gorm:"column:source_type;size:20;not null;uniqueIndex:uk_rev_source,priority:1;comment:purchase/sale"`
	SourceID   uint      `json:"sourceId" gorm:"column:source_id;not null;uniqueIndex:uk_rev_source,priority:2;comment:被冲销的单据ID"`
	SourceNo   string    `json:"sourceNo" gorm:"column:source_no;size:50;not null;comment:被冲销的单号（快照，便于查阅）"`
	Reason     string    `json:"reason" gorm:"column:reason;size:500;not null;comment:红冲原因（必填）"`
	Amount     float64   `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:冲销金额（正数表示冲了多少）"`
	Qty        int       `json:"qty" gorm:"column:qty;not null;default:0;comment:冲销件数合计"`
	Status     int8      `json:"status" gorm:"column:status;not null;default:1"`
	PostedAt   time.Time `json:"postedAt" gorm:"column:posted_at;index:idx_rev_posted"`
	Operator   string    `json:"operator" gorm:"column:operator;size:50"`
}

func (StockReversal) TableName() string { return "stock_reversal" }

// 退货单状态：草稿 → 已入库（退款线下完成，系统只记库存与成本）
const (
	ReturnStatusDraft    = 1
	ReturnStatusPosted   = 2
	ReturnStatusCanceled = 3
)

// SaleReturn 销售退货单（部分或全部退货），必须关联原销售单。
type SaleReturn struct {
	global.GVA_MODEL
	ReturnNo    string     `json:"returnNo" gorm:"column:return_no;size:50;not null;uniqueIndex:uk_ret_no"`
	SaleID      uint       `json:"saleId" gorm:"column:sale_id;not null;index:idx_sar_sale;comment:原销售单ID"`
	SaleNo      string     `json:"saleNo" gorm:"column:sale_no;size:50;not null;comment:原单号（快照）"`
	WarehouseID uint       `json:"warehouseId" gorm:"column:warehouse_id;not null;index:idx_sar_warehouse"`
	Status      int8       `json:"status" gorm:"column:status;not null;default:1;index:idx_sar_status"`
	Amount      float64    `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:退款金额（按原售价）"`
	CostAmount  float64    `json:"costAmount" gorm:"column:cost_amount;type:decimal(14,2);not null;default:0;comment:退回成本（按原单冻结成本）"`
	PostedAt    *time.Time `json:"postedAt" gorm:"column:posted_at;index:idx_sar_posted"`
	Reason      string     `json:"reason" gorm:"column:reason;size:500"`

	Items []SaleReturnItem `json:"items" gorm:"foreignKey:ReturnID;references:ID"`
}

func (SaleReturn) TableName() string { return "sale_return" }

// SaleReturnItem 退货明细。
// UnitCost 来自**原销售单出库时冻结的成本**，不是当前均价——
// 用当前成本退，等于用今天的成本改写上个月的毛利。
//
// CostAmount 是这一行实际回补的**精确成本金额**。为什么不能只存 UnitCost：
// 原单 3 件 100.00 时单价只能是 33.33，分三次退各乘 3 会少 1 分；
// 最后一次退货把这 1 分吸收进 CostAmount，账本金额才能精确回到 100.00。
type SaleReturnItem struct {
	ID         uint    `json:"ID" gorm:"primarykey"`
	ReturnID   uint    `json:"returnId" gorm:"column:return_id;not null;index"`
	SkuID      uint    `json:"skuId" gorm:"column:sku_id;not null"`
	Qty        int     `json:"qty" gorm:"column:qty;not null"`
	Price      float64 `json:"price" gorm:"column:price;type:decimal(12,2);not null;default:0;comment:原售价"`
	Amount     float64 `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0"`
	UnitCost   float64 `json:"unitCost" gorm:"column:unit_cost;type:decimal(12,2);not null;default:0;comment:原出库成本"`
	CostAmount float64 `json:"costAmount" gorm:"column:cost_amount;type:decimal(14,2);not null;default:0;comment:本行实际回补成本金额（吸收分位残差）"`
}

func (SaleReturnItem) TableName() string { return "sale_return_item" }

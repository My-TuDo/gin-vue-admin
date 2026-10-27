package jxc

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 采购退货：把已经入库的货退给供应商。
//
// 与“采购红冲”的区别：
//   - 红冲是整单录错，整单反向，要求库存足够；
//   - 采购退货是真实业务，可以只退一部分，退款金额按**原采购单价**计算。
//
// 采购退货是出库，账本行 qty 为负、amount 为负；成本取原采购行单价，
// 不使用当前移动加权平均——否则退给供应商的金额和账上减掉的成本会对不上。
const (
	PurchaseReturnStatusDraft    = 1 // 草稿
	PurchaseReturnStatusPosted   = 2 // 已退货出库
	PurchaseReturnStatusCanceled = 3 // 已取消
)

// PurchaseReturn 采购退货单（聚合根）。
type PurchaseReturn struct {
	global.GVA_MODEL
	ReturnNo    string     `json:"returnNo" gorm:"column:return_no;size:50;not null;uniqueIndex:uk_pr_no;comment:采购退货单号"`
	PurchaseID  uint       `json:"purchaseId" gorm:"column:purchase_id;not null;index;comment:原采购单ID"`
	PurchaseNo  string     `json:"purchaseNo" gorm:"column:purchase_no;size:50;not null;comment:原采购单号（快照）"`
	SupplierID  *uint      `json:"supplierId" gorm:"column:supplier_id;comment:供应商ID（从原采购单带出）"`
	WarehouseID uint       `json:"warehouseId" gorm:"column:warehouse_id;not null;comment:出库仓库（从原采购单带出）"`
	Status      int8       `json:"status" gorm:"column:status;not null;default:1;comment:1草稿 2已退货 3已取消"`
	Amount      float64    `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:退货金额（按原采购单价）"`
	PostedAt    *time.Time `json:"postedAt" gorm:"column:posted_at;comment:过账时间"`
	Reason      string     `json:"reason" gorm:"column:reason;size:500"`

	Items []PurchaseReturnItem `json:"items" gorm:"foreignKey:ReturnID;references:ID"`
}

func (PurchaseReturn) TableName() string { return "purchase_return" }

// PurchaseReturnItem 采购退货明细。Qty 恒为正，账本行才是负的。
type PurchaseReturnItem struct {
	ID       uint    `json:"ID" gorm:"primarykey"`
	ReturnID uint    `json:"returnId" gorm:"column:return_id;not null;index;comment:退货单ID"`
	SkuID    uint    `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`
	Qty      int     `json:"qty" gorm:"column:qty;not null;comment:退货数量（正）"`
	UnitCost float64 `json:"unitCost" gorm:"column:unit_cost;type:decimal(12,2);not null;default:0;comment:原采购单价"`
	Amount   float64 `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:退货金额=qty*unitCost"`
}

func (PurchaseReturnItem) TableName() string { return "purchase_return_item" }

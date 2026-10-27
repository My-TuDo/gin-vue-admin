package jxc

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 采购单状态机：
//
//	1 草稿 ──审核──> 2 已审核 ──入库──> 3 已入库
//	   │                 │
//	   └──── 取消 ───────┴──> 4 已取消
//
// 关键设计：**审核不动库存**，只有"入库"才产生账本行。
// 旧实现里"审核"和"入库"边界模糊，导致有人以为审核就算入库、
// 也有人重复点入库（那时还没有条件更新保护）。
const (
	PurchaseStatusDraft    = 1 // 草稿
	PurchaseStatusApproved = 2 // 已审核（待入库）
	PurchaseStatusReceived = 3 // 已入库
	PurchaseStatusCanceled = 4 // 已取消
)

// PurchaseOrder 采购单（聚合根）
type PurchaseOrder struct {
	global.GVA_MODEL
	OrderNo     string     `json:"orderNo" gorm:"column:order_no;size:50;not null;uniqueIndex:uk_po_no;comment:采购单号"`
	SupplierID  *uint      `json:"supplierId" gorm:"column:supplier_id;comment:供应商ID（可空）"`
	WarehouseID uint       `json:"warehouseId" gorm:"column:warehouse_id;not null;comment:入库仓库"`
	Status      int8       `json:"status" gorm:"column:status;not null;default:1;comment:1草稿 2已审核 3已入库 4已取消"`
	TotalAmount float64    `json:"totalAmount" gorm:"column:total_amount;type:decimal(14,2);not null;default:0;comment:采购总额"`
	PostedAt    *time.Time `json:"postedAt" gorm:"column:posted_at;comment:过账时间（入库时写入，之后单据不可改）"`
	ReversedAt  *time.Time `json:"reversedAt" gorm:"column:reversed_at;comment:红冲时间（非空表示已整单冲销）"`
	Remark      string     `json:"remark" gorm:"column:remark;size:500"`

	Items []PurchaseItem `json:"items" gorm:"foreignKey:PurchaseID;references:ID"`
}

func (PurchaseOrder) TableName() string { return "purchase_order" }

// PurchaseItem 采购明细。Qty 恒为正，方向由所在单据决定（账本行才是有符号的）。
type PurchaseItem struct {
	ID         uint    `json:"ID" gorm:"primarykey"`
	PurchaseID uint    `json:"purchaseId" gorm:"column:purchase_id;not null;index;comment:采购单ID"`
	SkuID      uint    `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`
	Qty        int     `json:"qty" gorm:"column:qty;not null;comment:采购数量（正）"`
	UnitCost   float64 `json:"unitCost" gorm:"column:unit_cost;type:decimal(12,2);not null;default:0;comment:采购单价"`
	Amount     float64 `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:金额=qty*unitCost"`
}

func (PurchaseItem) TableName() string { return "purchase_item" }

// 销售单状态机：
//
//	1 草稿（已预留库存）──出库──> 2 已出库
//	   │
//	   └── 取消（释放预留）──> 3 已取消
//
// 关键设计：**开单即预留**（Reserve），出库时把预留转为账本出库（Consume）。
// 旧实现用一个 lock_quantity 计数器表达预留，没有归属、没有释放路径，盘点时还被无视。
const (
	SaleStatusDraft    = 1 // 草稿（库存已预留）
	SaleStatusShipped  = 2 // 已出库
	SaleStatusCanceled = 3 // 已取消
)

// SaleOrder 销售单（聚合根）
type SaleOrder struct {
	global.GVA_MODEL
	OrderNo     string     `json:"orderNo" gorm:"column:order_no;size:50;not null;uniqueIndex:uk_so_no;comment:销售单号"`
	CustomerID  *uint      `json:"customerId" gorm:"column:customer_id;comment:客户ID（散客可空）"`
	WarehouseID uint       `json:"warehouseId" gorm:"column:warehouse_id;not null;comment:出库仓库"`
	Status      int8       `json:"status" gorm:"column:status;not null;default:1;comment:1草稿 2已出库 3已取消"`
	TotalAmount float64    `json:"totalAmount" gorm:"column:total_amount;type:decimal(14,2);not null;default:0;comment:销售总额"`
	CostAmount  float64    `json:"costAmount" gorm:"column:cost_amount;type:decimal(14,2);not null;default:0;comment:出库成本（过账时冻结，用于毛利）"`
	PostedAt    *time.Time `json:"postedAt" gorm:"column:posted_at;comment:过账时间（出库时写入）"`
	ReversedAt  *time.Time `json:"reversedAt" gorm:"column:reversed_at;comment:红冲时间（非空表示已整单冲销）"`
	Remark      string     `json:"remark" gorm:"column:remark;size:500"`

	Items []SaleItem `json:"items" gorm:"foreignKey:SaleID;references:ID"`
}

func (SaleOrder) TableName() string { return "sale_order" }

// SaleItem 销售明细。Qty 恒为正，账本行才是负的。
type SaleItem struct {
	ID      uint    `json:"ID" gorm:"primarykey"`
	SaleID  uint    `json:"saleId" gorm:"column:sale_id;not null;index;comment:销售单ID"`
	SkuID   uint    `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`
	Qty     int     `json:"qty" gorm:"column:qty;not null;comment:销售数量（正）"`
	Price   float64 `json:"price" gorm:"column:price;type:decimal(12,2);not null;default:0;comment:成交单价"`
	Amount  float64 `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:金额=qty*price"`
	CostAmt float64 `json:"costAmt" gorm:"column:cost_amt;type:decimal(14,2);not null;default:0;comment:过账时回填的该行成本"`
}

func (SaleItem) TableName() string { return "sale_item" }

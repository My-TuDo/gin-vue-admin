package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 采购单状态
const (
	PurchaseStatusPending  = 1 // 待审核
	PurchaseStatusAudited  = 2 // 已审核（待入库）
	PurchaseStatusStockIn  = 3 // 已入库
	PurchaseStatusCanceled = 4 // 已取消
)

// PurchaseOrder 采购单
type PurchaseOrder struct {
	global.GVA_MODEL
	OrderNo     string  `json:"orderNo" gorm:"column:order_no;size:50;not null;uniqueIndex:uk_purchase_order_no;comment:采购单号"`
	SupplierID  uint    `json:"supplierId" gorm:"column:supplier_id;not null;index:idx_purchase_supplier_id;comment:供应商ID"`
	WarehouseID uint    `json:"warehouseId" gorm:"column:warehouse_id;not null;comment:入库仓库ID"`
	Status      int8    `json:"status" gorm:"column:status;not null;default:1;comment:状态 1待审核 2已审核 3已入库 4已取消"`
	TotalAmount float64 `json:"totalAmount" gorm:"column:total_amount;type:decimal(12,2);not null;default:0;comment:采购总金额"`
	Remark      string  `json:"remark" gorm:"column:remark;size:500;comment:备注"`
	Creator     string  `json:"creator" gorm:"column:creator;size:50;comment:创建人"`

	// 关联
	Supplier *Supplier       `json:"supplier" gorm:"foreignKey:SupplierID;references:ID"`
	Warehouse *Warehouse     `json:"warehouse" gorm:"foreignKey:WarehouseID;references:ID"`
	Items    []PurchaseItem  `json:"items" gorm:"foreignKey:PurchaseID;references:ID"`
}

func (PurchaseOrder) TableName() string { return "purchase_order" }

// PurchaseItem 采购单明细（生命周期跟随采购单，不软删除）
type PurchaseItem struct {
	ID         uint    `json:"ID" gorm:"primarykey;comment:主键ID"`
	PurchaseID uint    `json:"purchaseId" gorm:"column:purchase_id;not null;index:idx_purchase_item_order;comment:采购单ID"`
	SkuID      uint    `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`
	SkuCode    string  `json:"skuCode" gorm:"column:sku_code;size:50;comment:SKU编码快照"`
	GoodsName  string  `json:"goodsName" gorm:"column:goods_name;size:200;comment:商品名称快照"`
	Color      string  `json:"color" gorm:"column:color;size:50;comment:颜色快照"`
	Size       string  `json:"size" gorm:"column:size;size:50;comment:尺码快照"`
	Qty        int     `json:"qty" gorm:"column:qty;not null;default:0;comment:采购数量"`
	Price      float64 `json:"price" gorm:"column:price;type:decimal(12,2);not null;default:0;comment:采购单价"`
	Amount     float64 `json:"amount" gorm:"column:amount;type:decimal(12,2);not null;default:0;comment:采购金额 qty*price"`

	Sku *GoodsSku `json:"sku" gorm:"foreignKey:SkuID;references:ID"`
}

func (PurchaseItem) TableName() string { return "purchase_item" }

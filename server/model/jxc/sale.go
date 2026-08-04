package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 销售单状态
const (
	SaleStatusPending  = 1 // 待出库
	SaleStatusShipped  = 2 // 已出库
	SaleStatusCanceled = 3 // 已取消
)

// 销售单类型
const (
	SaleTypeNormal    = 1 // 正常销售
	SaleTypeReturn    = 2 // 退货退款
	SaleTypeExOut     = 3 // 换货出库
	SaleTypeExIn      = 4 // 换货入库
)

// SaleOrder 销售单
type SaleOrder struct {
	global.GVA_MODEL
	OrderNo         string  `json:"orderNo" gorm:"column:order_no;size:50;not null;uniqueIndex:uk_sale_order_no;comment:销售单号"`
	CustomerID      *uint   `json:"customerId" gorm:"column:customer_id;index:idx_sale_customer_id;comment:客户ID（零售可空）"`
	WarehouseID     uint    `json:"warehouseId" gorm:"column:warehouse_id;not null;comment:出库仓库ID"`
	OrderType       int8    `json:"orderType" gorm:"column:order_type;not null;default:1;comment:单据类型 1正常销售 2退货退款 3换货出库 4换货入库"`
	OriginalOrderID *uint   `json:"originalOrderId" gorm:"column:original_order_id;index:idx_sale_original_order_id;comment:关联原单ID（退货/换货）"`
	Status          int8    `json:"status" gorm:"column:status;not null;default:1;comment:状态 1待出库 2已出库 3已取消"`
	TotalAmount     float64 `json:"totalAmount" gorm:"column:total_amount;type:decimal(12,2);not null;default:0;comment:销售总金额"`
	Remark          string  `json:"remark" gorm:"column:remark;size:500;comment:备注"`
	Creator         string  `json:"creator" gorm:"column:creator;size:50;comment:创建人"`

	// 关联
	Customer  *Customer   `json:"customer" gorm:"foreignKey:CustomerID;references:ID"`
	Warehouse *Warehouse  `json:"warehouse" gorm:"foreignKey:WarehouseID;references:ID"`
	Original  *SaleOrder  `json:"original" gorm:"foreignKey:OriginalOrderID;references:ID"`
	Items     []SaleItem  `json:"items" gorm:"foreignKey:SaleID;references:ID"`
}

func (SaleOrder) TableName() string { return "sale_order" }

// SaleItem 销售明细（生命周期跟随销售单，不软删除）
type SaleItem struct {
	ID        uint     `json:"ID" gorm:"primarykey;comment:主键ID"`
	SaleID    uint     `json:"saleId" gorm:"column:sale_id;not null;index:idx_sale_item_order;comment:销售单ID"`
	SkuID     uint     `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`
	SkuCode   string   `json:"skuCode" gorm:"column:sku_code;size:50;comment:SKU编码快照"`
	GoodsName string   `json:"goodsName" gorm:"column:goods_name;size:200;comment:商品名称快照"`
	Color     string   `json:"color" gorm:"column:color;size:50;comment:颜色快照"`
	Size      string   `json:"size" gorm:"column:size;size:50;comment:尺码快照"`
	Qty       int      `json:"qty" gorm:"column:qty;not null;default:0;comment:销售数量"`
	Price     float64  `json:"price" gorm:"column:price;type:decimal(12,2);not null;default:0;comment:销售单价"`
	Amount    float64  `json:"amount" gorm:"column:amount;type:decimal(12,2);not null;default:0;comment:销售金额 qty*price"`

	Sku *GoodsSku `json:"sku" gorm:"foreignKey:SkuID;references:ID"`
}

func (SaleItem) TableName() string { return "sale_item" }

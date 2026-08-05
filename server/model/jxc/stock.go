package jxc

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

// StockQuery 库存查询参数
type StockQuery struct {
	request.PageInfo
	WarehouseID uint `json:"warehouseId" form:"warehouseId"` // 仓库ID（0=全部）
	LowStock    bool `json:"lowStock" form:"lowStock"`       // 仅看预警（quantity <= warning_quantity）
}

// StockLogQuery 流水查询参数
type StockLogQuery struct {
	request.PageInfo
	WarehouseID  uint   `json:"warehouseId" form:"warehouseId"`
	SkuID        uint   `json:"skuId" form:"skuId"`
	BusinessType string `json:"businessType" form:"businessType"`
}

// Stock 库存（仓库 x SKU 快照，不软删除）
type Stock struct {
	ID              uint    `json:"ID" gorm:"primarykey;comment:主键ID"`
	WarehouseID     uint    `json:"warehouseId" gorm:"column:warehouse_id;not null;uniqueIndex:uk_stock_warehouse_sku,priority:1;comment:仓库ID"`
	SkuID           uint    `json:"skuId" gorm:"column:sku_id;not null;uniqueIndex:uk_stock_warehouse_sku,priority:2;index:idx_stock_sku_id;comment:SKU ID"`
	Quantity        int     `json:"quantity" gorm:"column:quantity;not null;default:0;comment:当前库存数量"`
	LockQuantity    int     `json:"lockQuantity" gorm:"column:lock_quantity;not null;default:0;comment:锁定数量"`
	WarningQuantity int     `json:"warningQuantity" gorm:"column:warning_quantity;not null;default:0;comment:预警数量"`

	// 关联（查询时加载）
	Sku *GoodsSku `json:"sku" gorm:"foreignKey:SkuID;references:ID"`
}

func (Stock) TableName() string { return "stock" }

// StockLog 库存流水（只追加，不可变）
type StockLog struct {
	ID           uint    `json:"ID" gorm:"primarykey;comment:主键ID"`
	WarehouseID  uint    `json:"warehouseId" gorm:"column:warehouse_id;not null;comment:仓库ID"`
	SkuID        uint    `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`
	BusinessType string  `json:"businessType" gorm:"column:business_type;size:30;not null;comment:业务类型 purchase_in/sale_out/..."`
	BusinessNo   string  `json:"businessNo" gorm:"column:business_no;size:50;not null;comment:业务单号"`
	BeforeQty    int     `json:"beforeQty" gorm:"column:before_qty;not null;default:0;comment:变更前数量"`
	ChangeQty    int     `json:"changeQty" gorm:"column:change_qty;not null;default:0;comment:变更数量 正入负出"`
	AfterQty     int     `json:"afterQty" gorm:"column:after_qty;not null;default:0;comment:变更后数量"`
	Operator     string  `json:"operator" gorm:"column:operator;size:50;comment:操作人"`
	Remark       string  `json:"remark" gorm:"column:remark;size:200;comment:备注/原因（直接出入库等）"`
}

func (StockLog) TableName() string { return "stock_log" }

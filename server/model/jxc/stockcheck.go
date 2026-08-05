package jxc

import "github.com/flipped-aurora/gin-vue-admin/server/global"

// 盘点单状态
const (
	StockCheckStatusChecking = 1 // 盘点中
	StockCheckStatusDone     = 2 // 已完成
	StockCheckStatusCanceled = 3 // 已取消
)

// StockCheck 盘点单（生命周期跟随盘点任务，不软删除）
type StockCheck struct {
	global.GVA_MODEL
	CheckNo     string `json:"checkNo" gorm:"column:check_no;size:50;not null;uniqueIndex:uk_check_no;comment:盘点单号 CK-YYYYMMDD-XXX"`
	WarehouseID uint   `json:"warehouseId" gorm:"column:warehouse_id;not null;index:idx_check_warehouse;comment:盘点仓库ID"`
	Status      int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态 1盘点中 2已完成 3已取消"`
	Checker     string `json:"checker" gorm:"column:checker;size:50;comment:盘点人"`
	Remark      string `json:"remark" gorm:"column:remark;size:500;comment:备注"`

	// 关联（仅用于查询预加载）
	Warehouse *Warehouse       `json:"warehouse" gorm:"foreignKey:WarehouseID;references:ID"`
	Items     []StockCheckItem `json:"items" gorm:"foreignKey:CheckID;references:ID"`
}

func (StockCheck) TableName() string { return "stock_check" }

// UpdateStockCheckItemsReq 录入盘点数请求
type UpdateStockCheckItemsReq struct {
	CheckID uint             `json:"checkId" binding:"required"`
	Items   []StockCheckItem `json:"items"`
}

// StockCheckItem 盘点明细（账存快照 + 实盘录入 + 差异）
type StockCheckItem struct {
	ID        uint `json:"ID" gorm:"primarykey;comment:主键ID"`
	CheckID   uint `json:"checkId" gorm:"column:check_id;not null;index:idx_check_item_check;comment:盘点单ID"`
	SkuID     uint `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`
	SystemQty int  `json:"systemQty" gorm:"column:system_qty;not null;default:0;comment:系统库存（创建时快照）"`
	ActualQty *int `json:"actualQty" gorm:"column:actual_qty;comment:实盘数量（录入后非空）"`
	DiffQty   int  `json:"diffQty" gorm:"column:diff_qty;not null;default:0;comment:差异数量 actual-system"`

	// 关联（仅用于查询预加载）
	Sku *GoodsSku `json:"sku" gorm:"foreignKey:SkuID;references:ID"`
}

func (StockCheckItem) TableName() string { return "stock_check_item" }

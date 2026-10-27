package jxc

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 本文件是「进销存」的领域模型：主数据 + 账本。
//
// 设计原则（见 docs/16-库存账本核心设计.md）：
//   - 库存的真相是账本（StockMove，只追加），余额（StockBalance）是可由账本重算的物化视图；
//   - 单据可以草稿、可以作废，但已过账的账本行永不可变，更正只能靠红冲（ReversalOf）；
//   - 成本冻结在发生那一刻（UnitCost 落在账本行上），历史毛利不会被后来的进价改写；
//   - 预留（Reservation）是独立实体，有归属、有到期，不用余额上的 lock 列。

// ============ 主数据（最小集，后续按需扩展） ============

// Warehouse 仓库
type Warehouse struct {
	global.GVA_MODEL
	Code   string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_warehouse_code;comment:仓库编码"`
	Name   string `json:"name" gorm:"column:name;size:100;not null;comment:仓库名称"`
	Status int8   `json:"status" gorm:"column:status;not null;default:1;comment:1启用 0停用"`
}

func (Warehouse) TableName() string { return "warehouse" }

// Goods 商品（SPU）
type Goods struct {
	global.GVA_MODEL
	Code   string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_goods_code;comment:商品编码"`
	Name   string `json:"name" gorm:"column:name;size:200;not null;comment:商品名称"`
	Status int8   `json:"status" gorm:"column:status;not null;default:1;comment:1上架 0下架"`
}

func (Goods) TableName() string { return "goods" }

// GoodsSku 商品规格（颜色 × 尺码）。库存与成本都挂在 SKU 维度。
type GoodsSku struct {
	global.GVA_MODEL
	GoodsID   uint    `json:"goodsId" gorm:"column:goods_id;not null;index;comment:商品ID"`
	SkuCode   string  `json:"skuCode" gorm:"column:sku_code;size:50;not null;uniqueIndex:uk_goods_sku_code;comment:SKU编码/条码"`
	Color     string  `json:"color" gorm:"column:color;size:50;not null;comment:颜色"`
	Size      string  `json:"size" gorm:"column:size;size:50;not null;comment:尺码"`
	SalePrice float64 `json:"salePrice" gorm:"column:sale_price;type:decimal(12,2);not null;default:0;comment:销售价"`
	RefCost   float64 `json:"refCost" gorm:"column:ref_cost;type:decimal(12,2);not null;default:0;comment:参考进价（仅供开单提示，不参与毛利计算）"`
	Status    int8    `json:"status" gorm:"column:status;not null;default:1;comment:1启用 0停用"`
}

func (GoodsSku) TableName() string { return "goods_sku" }

// ============ 账本 ============

// 账本来源类型。写入后不再变更，报表与追溯都靠它。
const (
	MoveSourcePurchaseIn = "purchase_in" // 采购入库
	MoveSourceSaleOut    = "sale_out"    // 销售出库
	MoveSourceSaleReturn = "sale_return" // 退货入库（成本取原单）
	MoveSourceDirectIn   = "direct_in"   // 直接入库
	MoveSourceDirectOut  = "direct_out"  // 直接出库
	MoveSourceCheckGain  = "check_gain"  // 盘盈
	MoveSourceCheckLoss  = "check_loss"  // 盘亏
	MoveSourceReversal   = "reversal"    // 红冲（配合 ReversalOf 指向被冲销行）
)

// StockMove 库存账本行。只允许 INSERT。
//
// 为什么 Qty 有符号：金额、余额、汇总都能直接 SUM，不需要 CASE WHEN 方向列；
// 旧实现用 direction 列带来的"必须双向/方向不能为 0"等特判在账本模型里不存在。
type StockMove struct {
	ID          uint `json:"ID" gorm:"primarykey"`
	WarehouseID uint `json:"warehouseId" gorm:"column:warehouse_id;not null;comment:仓库ID"`
	SkuID       uint `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`

	Qty      int     `json:"qty" gorm:"column:qty;not null;comment:有符号数量，入库正/出库负"`
	UnitCost float64 `json:"unitCost" gorm:"column:unit_cost;type:decimal(12,2);not null;default:0;comment:入库单价或出库结转成本(COGS)"`
	Amount   float64 `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:有符号金额 = qty * unit_cost"`

	// (SourceType, SourceID, SourceLineID) 唯一：同一张单据的同一行只能产生一条账本行。
	// 这是幂等的物理保障——POS 双击、网络重试、消息重投都不会造成重复过账。
	SourceType   string `json:"sourceType" gorm:"column:source_type;size:32;not null;uniqueIndex:uk_move_idem,priority:1;comment:来源类型"`
	SourceID     uint   `json:"sourceId" gorm:"column:source_id;not null;uniqueIndex:uk_move_idem,priority:2;comment:来源单据ID"`
	SourceLineID uint   `json:"sourceLineId" gorm:"column:source_line_id;not null;default:0;uniqueIndex:uk_move_idem,priority:3;comment:来源明细ID"`

	OccurredAt time.Time `json:"occurredAt" gorm:"column:occurred_at;not null;comment:业务发生时间"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime;comment:写入时间"`

	ReversalOf *uint  `json:"reversalOf" gorm:"column:reversal_of;comment:被冲销的账本行ID"`
	Operator   string `json:"operator" gorm:"column:operator;size:50;comment:操作人"`
}

func (StockMove) TableName() string { return "stock_move" }

// StockBalance 物化余额。只由过账逻辑写入，禁止手工 UPDATE。
// 不变量：Qty == SUM(StockMove.Qty) 且 Value == SUM(StockMove.Amount)。
type StockBalance struct {
	WarehouseID uint      `json:"warehouseId" gorm:"column:warehouse_id;primaryKey;comment:仓库ID"`
	SkuID       uint      `json:"skuId" gorm:"column:sku_id;primaryKey;comment:SKU ID"`
	Qty         int       `json:"qty" gorm:"column:qty;not null;default:0;comment:结存数量"`
	Value       float64   `json:"value" gorm:"column:value;type:decimal(14,2);not null;default:0;comment:结存金额"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (StockBalance) TableName() string { return "stock_balance" }

// 预留状态
const (
	ReservationActive   = 1 // 有效
	ReservationReleased = 2 // 已释放
	ReservationConsumed = 3 // 已转出库
)

// 预留来源类型（预留不是账本行，但同样需要来源标识以便幂等与释放）
const (
	ReserveSourceSaleOrder = "sale_order" // 销售开单占用
	ReserveSourceExchange  = "exchange"   // 换货换出占用
)

// StockReservation 预留：替代旧实现的 stock.lock_quantity（半成品，无归属无到期）。
type StockReservation struct {
	ID          uint `json:"ID" gorm:"primarykey"`
	WarehouseID uint `json:"warehouseId" gorm:"column:warehouse_id;not null;comment:仓库ID"`
	SkuID       uint `json:"skuId" gorm:"column:sku_id;not null;comment:SKU ID"`
	Qty         int  `json:"qty" gorm:"column:qty;not null;comment:预留数量"`

	SourceType   string `json:"sourceType" gorm:"column:source_type;size:32;not null;uniqueIndex:uk_res,priority:1"`
	SourceID     uint   `json:"sourceId" gorm:"column:source_id;not null;uniqueIndex:uk_res,priority:2"`
	SourceLineID uint   `json:"sourceLineId" gorm:"column:source_line_id;not null;default:0;uniqueIndex:uk_res,priority:3"`

	Status    int8       `json:"status" gorm:"column:status;not null;default:1;comment:1有效 2已释放 3已转出库"`
	ExpiresAt *time.Time `json:"expiresAt" gorm:"column:expires_at;comment:到期自动失效"`
	CreatedAt time.Time  `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
}

func (StockReservation) TableName() string { return "stock_reservation" }

// 期间状态
const (
	PeriodOpen   = 1
	PeriodClosed = 2
)

// AccountingPeriod 会计期间（月）。已结账期间禁止再写入账本行。
type AccountingPeriod struct {
	ID uint `json:"ID" gorm:"primarykey"`
	// 列名用 period_month 而不是 year_month：YEAR_MONTH 是 MySQL 的保留关键字（interval 单位），
	// 裸写进 SQL 会直接语法错误。踩过一次，这里留个记号。
	PeriodMonth string     `json:"periodMonth" gorm:"column:period_month;size:7;not null;uniqueIndex:uk_period;comment:2026-10"`
	Status      int8       `json:"status" gorm:"column:status;not null;default:1;comment:1打开 2已结账"`
	ClosedAt    *time.Time `json:"closedAt" gorm:"column:closed_at"`
	ClosedBy    string     `json:"closedBy" gorm:"column:closed_by;size:50"`
}

func (AccountingPeriod) TableName() string { return "accounting_period" }

// StockBalanceSnapshot 结账时冻结的余额快照。之后所有人看到的"上月底库存"都是它。
type StockBalanceSnapshot struct {
	PeriodID    uint    `json:"periodId" gorm:"column:period_id;primaryKey"`
	WarehouseID uint    `json:"warehouseId" gorm:"column:warehouse_id;primaryKey"`
	SkuID       uint    `json:"skuId" gorm:"column:sku_id;primaryKey"`
	Qty         int     `json:"qty" gorm:"column:qty;not null"`
	Value       float64 `json:"value" gorm:"column:value;type:decimal(14,2);not null"`
}

func (StockBalanceSnapshot) TableName() string { return "stock_balance_snapshot" }

// 往来单位类型
const (
	PartnerSupplier = "supplier" // 供应商
	PartnerCustomer = "customer" // 客户
)

// Partner 往来单位（供应商 / 客户）。
// 用一张表 + kind 字段而不是两张表：字段完全一致，报表也常常"按往来单位汇总"，
// 分成两张表只会让查询与前端多一倍分支。
type Partner struct {
	global.GVA_MODEL
	Kind    string `json:"kind" gorm:"column:kind;size:16;not null;index;comment:supplier/customer"`
	Code    string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_partner_code;comment:编码"`
	Name    string `json:"name" gorm:"column:name;size:100;not null;comment:名称"`
	Contact string `json:"contact" gorm:"column:contact;size:50;comment:联系人"`
	Phone   string `json:"phone" gorm:"column:phone;size:30;comment:电话"`
	Address string `json:"address" gorm:"column:address;size:200;comment:地址"`
	Remark  string `json:"remark" gorm:"column:remark;size:500"`
	Status  int8   `json:"status" gorm:"column:status;not null;default:1;comment:1启用 0停用"`
}

func (Partner) TableName() string { return "jxc_partner" }

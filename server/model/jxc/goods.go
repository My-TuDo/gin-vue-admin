package jxc

import "github.com/flipped-aurora/gin-vue-admin/server/global"

// Goods 商品 SPU
type Goods struct {
	global.GVA_MODEL
	Code       string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_goods_code;comment:商品编码（款号）"`
	Name       string `json:"name" gorm:"column:name;size:200;not null;comment:商品名称"`
	CategoryID *uint  `json:"categoryId" gorm:"column:category_id;index:idx_goods_category_id;comment:分类ID"`
	BrandID    *uint  `json:"brandId" gorm:"column:brand_id;index:idx_goods_brand_id;comment:品牌ID"`
	Unit       string `json:"unit" gorm:"column:unit;size:20;not null;default:件;comment:计量单位"`
	Image      string `json:"image" gorm:"column:image;size:500;comment:主图URL"`
	Status     int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态 1上架 0下架"`
	Remark     string `json:"remark" gorm:"column:remark;size:500;comment:备注"`

	// 关联（查询时加载）
	Category *GoodsCategory `json:"category" gorm:"foreignKey:CategoryID;references:ID"`
	Brand    *Brand         `json:"brand" gorm:"foreignKey:BrandID;references:ID"`
	Skus     []GoodsSku     `json:"skus" gorm:"foreignKey:GoodsID;references:ID"`
}

func (Goods) TableName() string { return "goods" }

// SkuStockRow 单个仓库库存行（扫码查库存用，非表字段）
type SkuStockRow struct {
	WarehouseID   uint   `json:"warehouseId" comment:"仓库ID"`
	WarehouseName string `json:"warehouseName" comment:"仓库名称"`
	Quantity      int    `json:"quantity" comment:"当前库存"`
	LockQuantity  int    `json:"lockQuantity" comment:"锁定数量"`
	Available     int    `json:"available" comment:"可售数量 quantity-lockQuantity"`
}

// GoodsSku 商品 SKU
type GoodsSku struct {
	global.GVA_MODEL
	GoodsID   uint    `json:"goodsId" gorm:"column:goods_id;not null;index:idx_sku_goods_id;uniqueIndex:uk_sku_goods_color_size,priority:1;comment:所属商品ID"`
	SkuCode   string  `json:"skuCode" gorm:"column:sku_code;size:50;not null;uniqueIndex:uk_goods_sku_code;comment:SKU编码, 如 SP001-RED-M"`
	Barcode   string  `json:"barcode" gorm:"column:barcode;size:50;index:idx_sku_barcode;comment:条码"`
	Color     string  `json:"color" gorm:"column:color;size:50;not null;uniqueIndex:uk_sku_goods_color_size,priority:2;comment:颜色"`
	Size      string  `json:"size" gorm:"column:size;size:50;not null;uniqueIndex:uk_sku_goods_color_size,priority:3;comment:尺码"`
	CostPrice float64 `json:"costPrice" gorm:"column:cost_price;type:decimal(12,2);not null;default:0;comment:成本价"`
	SalePrice float64 `json:"salePrice" gorm:"column:sale_price;type:decimal(12,2);not null;default:0;comment:销售价"`
	SafeStock int     `json:"safeStock" gorm:"column:safe_stock;not null;default:0;comment:安全库存（0=不预警）"`
	Status    int8    `json:"status" gorm:"column:status;not null;default:1;comment:状态 1启用 0停用"`

	Goods *Goods         `json:"goods" gorm:"foreignKey:GoodsID;references:ID"`
	Stocks []SkuStockRow `json:"stocks" gorm:"-" comment:"各仓库库存（扫码查库存时加载）"`
}

func (GoodsSku) TableName() string { return "goods_sku" }

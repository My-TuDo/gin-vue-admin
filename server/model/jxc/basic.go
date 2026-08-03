package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// ========== GoodsCategory 商品分类 ==========
type GoodsCategory struct {
	global.GVA_MODEL
	ParentID *uint  `json:"parentId" gorm:"column:parent_id;comment:父分类ID, NULL表示根节点"`
	Code     string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_goods_category_code;comment:分类编码，全局唯一"`
	Name     string `json:"name" gorm:"column:name;size:100;not null;comment:分类名称"`
	Sort     int    `json:"sort" gorm:"column:sort;not null;default:0;comment:排序，数值越小越靠前"`
	Status   int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态 1启用 0停用"`
	Remark   string `json:"remark" gorm:"column:remark;size:500;comment:备注"`

	// 关联关系
	Parent   *GoodsCategory  `json:"parent" gorm:"foreignKey:ParentID;references:ID"`
	Children []GoodsCategory `json:"children" gorm:"foreignKey:ParentID;references:ID"`
}

// 生成表名 goods_category
func (GoodsCategory) TableName() string {
	return "goods_category"
}

// ========== Brand 品牌 ==========
type Brand struct {
	global.GVA_MODEL
	Code   string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_brand_code;comment:品牌编码，全局唯一"`
	Name   string `json:"name" gorm:"column:name;size:100;not null;index:idx_brand_name;comment:品牌名称"`
	Logo   string `json:"logo" gorm:"column:logo;size:500;comment:品牌logo图片URL"`
	Status int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态 1启用 0停用"`
	Remark string `json:"remark" gorm:"column:remark;size:500;comment:备注"`
}

func (Brand) TableName() string {
	return "brand"
}

// ========== Supplier 供应商 ==========
type Supplier struct {
	global.GVA_MODEL
	Code    string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_supplier_code;comment:供应商编码，全局唯一"`
	Name    string `json:"name" gorm:"column:name;size:100;not null;index:idx_supplier_name;comment:供应商名称"`
	Contact string `json:"contact" gorm:"column:contact;size:50;comment:联系人"`
	Phone   string `json:"phone" gorm:"column:phone;size:20;comment:联系电话"`
	Address string `json:"address" gorm:"column:address;size:200;comment:联系地址"`
	Status  int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态 1启用 0停用"`
	Remark  string `json:"remark" gorm:"column:remark;size:500;comment:备注"`
}

func (Supplier) TableName() string {
	return "supplier"
}

// ========== Customer 客户 ==========
type Customer struct {
	global.GVA_MODEL
	Code    string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_customer_code;comment:客户编码，全局唯一"`
	Name    string `json:"name" gorm:"column:name;size:100;not null;index:idx_customer_name;comment:客户名称"`
	Phone   string `json:"phone" gorm:"column:phone;size:20;index:idx_phone;comment:联系电话"`
	Address string `json:"address" gorm:"column:address;size:200;comment:联系地址"`
	Level   int8   `json:"level" gorm:"column:level;not null;default:1;comment:客户等级 1普通客户 2VIP客户 3批发客户"`
	Status  int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态 1正常 0停用"`
	Remark  string `json:"remark" gorm:"column:remark;size:500;comment:备注"`
}

func (Customer) TableName() string {
	return "customer"
}

// ========== Warehouse 仓库 ==========
type Warehouse struct {
	global.GVA_MODEL
	Code    string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex:uk_warehouse_code;comment:仓库编码，全局唯一"`
	Name    string `json:"name" gorm:"column:name;size:100;not null;comment:仓库名称"`
	Manager string `json:"manager" gorm:"column:manager;size:50;comment:仓库负责人姓名"`
	Phone   string `json:"phone" gorm:"column:phone;size:20;comment:负责人电话"`
	Address string `json:"address" gorm:"column:address;size:200;comment:仓库地址"`
	Status  int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态 1启用 0停用"`
}

func (Warehouse) TableName() string {
	return "warehouse"
}

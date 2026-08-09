package jxc

import "time"

// PosScan 扫码枪队列：小程序扫码上架，PC 收银台轮询确认
// Status: 0=待处理（PC 尚未加入购物车） 1=已处理
type PosScan struct {
	ID        uint       `json:"id" gorm:"primarykey"`
	SkuID     uint       `json:"skuId" gorm:"column:sku_id;not null;index"`
	Qty       int        `json:"qty" gorm:"not null;default:1"`
	Status    int        `json:"status" gorm:"not null;default:0;index"`
	CreatedAt time.Time  `json:"createdAt"`
	HandledAt *time.Time `json:"handledAt"`
}

func (PosScan) TableName() string { return "pos_scan" }

// PosScanDetail 列表返回项（附带 SKU 信息）
type PosScanDetail struct {
	PosScan
	Sku *SkuBrief `json:"sku,omitempty"`
}

// SkuBrief SKU 摘要（扫码枪展示用）
type SkuBrief struct {
	ID        uint    `json:"id"`
	SkuCode   string  `json:"skuCode"`
	GoodsName string  `json:"goodsName"`
	Color     string  `json:"color"`
	Size      string  `json:"size"`
	SalePrice float64 `json:"salePrice"`
	Barcode   string  `json:"barcode"`
}

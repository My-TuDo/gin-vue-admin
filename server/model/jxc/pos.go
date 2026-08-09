package jxc

import "time"

// PosScan 扫码枪队列：小程序扫码上架，PC 收银台轮询自动加入购物车
// Status: 0=未消费（PC 尚未加入购物车） 1=已消费
// Session: 收银台码（PosSession.Code），小程序扫码时携带，PC 只拉取自己的会话
type PosScan struct {
	ID        uint       `json:"id" gorm:"primarykey"`
	Session   string     `json:"session" gorm:"column:session;size:32;not null;index"`
	SkuID     uint       `json:"skuId" gorm:"column:sku_id;not null;index"`
	Qty       int        `json:"qty" gorm:"not null;default:1"`
	Status    int        `json:"status" gorm:"not null;default:0;index"`
	CreatedAt time.Time  `json:"createdAt"`
	HandledAt *time.Time `json:"handledAt"`
}

func (PosScan) TableName() string { return "pos_scan" }

// PosSession 收银台会话：PC 端在收银台设置中生成/重置，小程序绑定后扫码投递到对应 PC
type PosSession struct {
	ID         uint       `json:"id" gorm:"primarykey"`
	Code       string     `json:"code" gorm:"column:code;size:16;not null;uniqueIndex"`
	Remark     string     `json:"remark" gorm:"column:remark;size:64;comment:备注（如：1号收银机）"`
	Status     int        `json:"status" gorm:"not null;default:1;comment:1=有效 0=已作废"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
}

func (PosSession) TableName() string { return "pos_session" }

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

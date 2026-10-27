package request

// 基础资料入参。
//
// 为什么不用 jxc.Warehouse / jxc.Goods 直接绑定 JSON：
// 模型里带着 ID / Status / CreatedAt 等字段，客户端只要多传一个字段就能
// 覆盖主键、状态或时间（mass assignment）。入参只保留业务允许修改的字段，
// 其余一律由服务端决定。

// WarehouseInput 仓库创建/编辑入参
type WarehouseInput struct {
	Code string `json:"code" form:"code"`
	Name string `json:"name" form:"name"`
}

// GoodsSkuInput 商品规格创建/编辑入参。
// ID > 0 表示更新已有规格；ID == 0 表示在同一商品下新增规格。
type GoodsSkuInput struct {
	ID        uint    `json:"ID" form:"ID"`
	Color     string  `json:"color" form:"color"`
	Size      string  `json:"size" form:"size"`
	SkuCode   string  `json:"skuCode" form:"skuCode"`
	SalePrice float64 `json:"salePrice" form:"salePrice"`
	RefCost   float64 `json:"refCost" form:"refCost"`
	Status    *int8   `json:"status" form:"status"`
}

// GoodsInput 商品创建/编辑入参
type GoodsInput struct {
	Code   string          `json:"code" form:"code"`
	Name   string          `json:"name" form:"name"`
	Status *int8           `json:"status" form:"status"`
	Skus   []GoodsSkuInput `json:"skus" form:"skus"`
}

// PartnerInput 往来单位创建/编辑入参。
// Kind 只在创建时生效；编辑时不允许改类型（供应商/客户被单据引用后改类型会失真）。
type PartnerInput struct {
	Kind    string `json:"kind" form:"kind"`
	Code    string `json:"code" form:"code"`
	Name    string `json:"name" form:"name"`
	Contact string `json:"contact" form:"contact"`
	Phone   string `json:"phone" form:"phone"`
	Address string `json:"address" form:"address"`
	Remark  string `json:"remark" form:"remark"`
}

// StatusInput 启用/停用入参。只允许 0/1。
type StatusInput struct {
	Status int8 `json:"status" form:"status"`
}

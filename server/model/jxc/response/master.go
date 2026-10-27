package response

import "github.com/flipped-aurora/gin-vue-admin/server/model/jxc"

// GoodsWithSkus 商品 + 规格，一次返回，前端商品卡片直接可用。
// 放在 response 包而不是 API 层，是为了让 Service / API 共用同一个返回类型。
type GoodsWithSkus struct {
	jxc.Goods
	Skus []jxc.GoodsSku `json:"skus"`
}

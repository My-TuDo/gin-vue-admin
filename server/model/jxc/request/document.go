package request

import (
	common "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// DocumentSearch 单据列表的统一筛选参数。
//
// 五个单据端点（销售/采购/销售退货/采购退货/红冲）共用这一套查询参数，
// 由各 Service 的 Search 方法翻译成对应表的 WHERE / JOIN。
type DocumentSearch struct {
	common.PageInfo

	Status      *int8  `json:"status" form:"status"`           // 单据状态；不传=全部
	WarehouseID uint   `json:"warehouseId" form:"warehouseId"` // 仓库
	PartnerID   uint   `json:"partnerId" form:"partnerId"`     // 客户/供应商
	StartAt     string `json:"startAt" form:"startAt"`         // YYYY-MM-DD，含当天
	EndAt       string `json:"endAt" form:"endAt"`             // YYYY-MM-DD，含当天
	DateField   string `json:"dateField" form:"dateField"`     // created（默认）/ posted
	Reversed    *bool  `json:"reversed" form:"reversed"`       // 仅采购/销售：是否已红冲
	SourceType  string `json:"sourceType" form:"sourceType"`   // 仅红冲：purchase / sale
}

// Normalize 把分页参数拉回安全范围。
// 与 common.PageInfo 的 MaxPageSize 保持一致；service 层再调用一次也无副作用。
func (s *DocumentSearch) Normalize() {
	if s.Page <= 0 {
		s.Page = 1
	}
	if s.PageSize <= 0 {
		s.PageSize = 20
	}
	if s.PageSize > common.MaxPageSize {
		s.PageSize = common.MaxPageSize
	}
	if s.DateField == "" {
		s.DateField = "created"
	}
}

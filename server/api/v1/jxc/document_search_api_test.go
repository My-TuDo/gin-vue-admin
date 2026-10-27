package jxc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindDocumentSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet,
		"/?page=2&pageSize=500&status=3&keyword=abc&warehouseId=7&partnerId=8&startAt=2026-10-01&endAt=2026-10-02&dateField=posted&reversed=true&sourceType=sale", nil)

	info, ok := bindDocumentSearch(c)
	if !ok {
		t.Fatal("合法查询参数不应绑定失败")
	}
	if info.Page != 2 || info.PageSize != 100 {
		t.Errorf("分页应为 2/100（pageSize 超上限被截断），实际 %d/%d", info.Page, info.PageSize)
	}
	if info.Status == nil || *info.Status != 3 {
		t.Errorf("status 应解析为 3，实际 %+v", info.Status)
	}
	if info.Keyword != "abc" || info.WarehouseID != 7 || info.PartnerID != 8 {
		t.Errorf("关键词/仓库/往来单位解析不对：%+v", info)
	}
	if info.StartAt != "2026-10-01" || info.EndAt != "2026-10-02" || info.DateField != "posted" {
		t.Errorf("日期参数解析不对：%+v", info)
	}
	if info.Reversed == nil || !*info.Reversed || info.SourceType != "sale" {
		t.Errorf("reversed/sourceType 解析不对：%+v", info)
	}
}

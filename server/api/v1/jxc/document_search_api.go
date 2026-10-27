package jxc

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
	"github.com/gin-gonic/gin"
)

// bindDocumentSearch 统一解析单据列表的筛选参数。
// 五个 page 端点共用，避免每个 handler 各写一套 strconv.Atoi。
func bindDocumentSearch(c *gin.Context) (jxcReq.DocumentSearch, bool) {
	var info jxcReq.DocumentSearch
	if err := c.ShouldBindQuery(&info); err != nil {
		response.FailWithMessage("请求参数错误："+err.Error(), c)
		return info, false
	}
	info.Normalize()
	return info, true
}

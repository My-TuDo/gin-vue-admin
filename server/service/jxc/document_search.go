package jxc

import (
	"fmt"
	"strings"
	"time"

	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
	"gorm.io/gorm"
)

const documentDateLayout = "2006-01-02"

// jxcLocation 业务时区固定为 Asia/Shanghai。
// 不依赖服务器本地时区：部署到 UTC 机器时，日期筛选和月报仍然按中国时间切日。
// 用 FixedZone 而不是 LoadLocation，避免 Windows 上缺少 tzdata 时启动失败。
var jxcLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// applyDocumentDateRange 给单据查询追加日期范围。
//
// start/end 都是“含当天”的语义：end 会转成次日 00:00 的 < 条件，
// 这样 2026-10-27 能包含当天 23:59:59 的单据。
func applyDocumentDateRange(db *gorm.DB, info jxcReq.DocumentSearch, createdCol, postedCol string) (*gorm.DB, error) {
	var col string
	switch info.DateField {
	case "", "created":
		col = createdCol
	case "posted":
		col = postedCol
	default:
		return nil, fmt.Errorf("dateField 只能是 created 或 posted")
	}

	start := strings.TrimSpace(info.StartAt)
	end := strings.TrimSpace(info.EndAt)
	var startTime, endTime time.Time
	if start != "" {
		t, err := time.ParseInLocation(documentDateLayout, start, jxcLocation)
		if err != nil {
			return nil, fmt.Errorf("开始日期格式应为 YYYY-MM-DD：%s", start)
		}
		startTime = t
		db = db.Where(col+" >= ?", t)
	}
	if end != "" {
		t, err := time.ParseInLocation(documentDateLayout, end, jxcLocation)
		if err != nil {
			return nil, fmt.Errorf("结束日期格式应为 YYYY-MM-DD：%s", end)
		}
		endTime = t
		db = db.Where(col+" < ?", t.AddDate(0, 0, 1))
	}
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		return nil, fmt.Errorf("开始日期不能晚于结束日期")
	}
	return db, nil
}

// keywordPattern 统一关键词 LIKE 模式；空关键词返回空串，由调用方决定是否加条件。
func keywordPattern(keyword string) string {
	k := strings.TrimSpace(keyword)
	if k == "" {
		return ""
	}
	return "%" + k + "%"
}

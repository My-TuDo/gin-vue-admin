package jxc

import (
	"context"
	"strconv"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// 对账最近结果：/last 读取缓存，不应触发新的全仓扫描；
// 手工 Check 后应带出 checkedAt/durationMs/scope/checkedSkus。
func TestLedger_LastReconcile(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, sku := seedSku(t, db)
	seedStock(t, db, wh, sku, 5, 10)
	ls := new(LedgerService)
	ctx := context.Background()

	const probeID = uint(987654)
	if got := ls.LastReconcile(probeID); got != nil {
		t.Fatalf("未对账过的仓库应返回 nil，实际 %+v", got)
	}

	res, err := ls.Check(ctx, wh)
	if err != nil {
		t.Fatalf("对账失败: %v", err)
	}
	if !res.OK || res.Count != 0 || res.CheckedSkus != 1 {
		t.Fatalf("账本应一致/1 个 SKU，实际 %+v", res)
	}
	if res.Scope != "warehouse:"+strconv.Itoa(int(wh)) || res.CheckedAt.IsZero() || res.DurationMs < 0 {
		t.Errorf("对账元信息不完整：%+v", res)
	}
	last := ls.LastReconcile(wh)
	if last == nil || last.CheckedAt.IsZero() || last.Stale {
		t.Fatalf("刚对账完的最近结果应为新鲜，实际 %+v", last)
	}
	if last.CheckedSkus != 1 || !last.OK {
		t.Errorf("最近结果应与本次一致：%+v", last)
	}

	// 人为制造漂移：余额数量被直接改掉，下次对账必须报 qty_mismatch
	if err := db.Model(&jxc.StockBalance{}).
		Where("warehouse_id = ? AND sku_id = ?", wh, sku).
		Update("qty", 0).Error; err != nil {
		t.Fatalf("制造漂移失败: %v", err)
	}
	bad, err := ls.Check(ctx, wh)
	if err != nil {
		t.Fatalf("二次对账失败: %v", err)
	}
	if bad.OK || bad.Count == 0 {
		t.Fatalf("余额被改后应报不一致，实际 %+v", bad)
	}
	if last := ls.LastReconcile(wh); last == nil || last.OK || last.Count == 0 {
		t.Errorf("最近结果应更新为不一致，实际 %+v", last)
	}
}

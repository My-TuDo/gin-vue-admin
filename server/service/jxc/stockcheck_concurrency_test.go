package jxc

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// TestStockCheck_ConcurrentCreateOnlyOne 同一仓库并发发起盘点，只能成功一张。
// 旧实现是 check-then-insert，并发下可以建出两张进行中的盘点单。
func TestStockCheck_ConcurrentCreateOnlyOne(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, _ := seedSku(t, db)
	cs := NewStockCheckService()
	ctx := context.Background()

	const n = 4
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := cs.Create(ctx, wh, "并发盘点"); err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if success != 1 {
		t.Fatalf("同一仓库并发建盘点单应只成功 1 次，实际 %d 次", success)
	}
	var open int64
	db.Model(&jxc.StockCheck{}).
		Where("warehouse_id = ? AND status = ?", wh, jxc.StockCheckDraft).Count(&open)
	if open != 1 {
		t.Fatalf("进行中的盘点单应只有 1 张，实际 %d", open)
	}
	// 已有一张时再建必须给出可读错误
	_, err := cs.Create(ctx, wh, "再来一张")
	if err == nil || !strings.Contains(err.Error(), "已有进行中的盘点单") {
		t.Fatalf("已有盘点单时应拒绝并提示，got %v", err)
	}
}

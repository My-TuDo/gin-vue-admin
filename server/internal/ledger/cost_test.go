package ledger

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
)

// 移动加权平均的成本结转测试。
// 这些用例对应真实场景，不是为覆盖率编的：进两次不同价格、卖一部分、清空、成本变化后再卖。

func c(yuan float64) money.Cents { return money.FromYuan(yuan) }

func TestApplyInbound_Accumulates(t *testing.T) {
	var p Position
	p = p.ApplyInbound(10, c(5)) // 10 件 @5.00
	if p.Qty != 10 || p.Value != c(50) {
		t.Fatalf("入库后应为 10 件 / 50.00 元，实际 %d / %s", p.Qty, p.Value)
	}
	p = p.ApplyInbound(10, c(7)) // 再进 10 件 @7.00
	if p.Qty != 20 || p.Value != c(120) {
		t.Fatalf("二次入库后应为 20 件 / 120.00 元，实际 %d / %s", p.Qty, p.Value)
	}
	if got := p.AvgCost(); got != c(6) {
		t.Errorf("移动加权平均应为 6.00，实际 %s", got)
	}
}

func TestApplyOutbound_UsesMovingAverage(t *testing.T) {
	p := Position{Qty: 20, Value: c(120)} // 均价 6.00
	cost, after, err := p.ApplyOutbound(5)
	if err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	if cost != c(30) {
		t.Errorf("5 件应有 30.00 成本，实际 %s", cost)
	}
	if after.Qty != 15 || after.Value != c(90) {
		t.Errorf("出库后应为 15 件 / 90.00 元，实际 %d / %s", after.Qty, after.Value)
	}
}

// TestApplyOutbound_CostIsFrozenForHistory 关键回归：
// 卖完之后再以更高价进货，已发生的出库成本不得变化。
// 旧实现用 goods_sku.cost_price（被最新进价覆盖），历史毛利会被事后改写。
func TestApplyOutbound_CostIsFrozenForHistory(t *testing.T) {
	p := Position{Qty: 10, Value: c(50)} // @5.00
	cost1, after, _ := p.ApplyOutbound(10)
	if cost1 != c(50) {
		t.Fatalf("第一次出库成本应为 50.00，实际 %s", cost1)
	}
	// 事后以 9.00 进货
	after = after.ApplyInbound(10, c(9))
	// 早已发生的 cost1 仍然是 50.00（这是账本行里冻结的值，不是重新算的）
	if cost1 != c(50) {
		t.Errorf("历史成本被改写了：%s", cost1)
	}
	if after.Qty != 10 || after.Value != c(90) {
		t.Errorf("新库存应为 10 件 / 90.00 元，实际 %d / %s", after.Qty, after.Value)
	}
}

// TestApplyOutbound_SellAllAbsorbsResidual 清空库存时吸收四舍五入残差。
// 否则会出现"库存 0 件、账上还剩 0.01 元"的脏数据。
func TestApplyOutbound_SellAllAbsorbsResidual(t *testing.T) {
	// 100 元 / 3 件 = 33.333...，均价只能取 33.33
	p := Position{Qty: 3, Value: c(100)}
	cost, after, err := p.ApplyOutbound(3)
	if err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	if cost != c(100) {
		t.Errorf("清空时应结转全部剩余金额 100.00，实际 %s", cost)
	}
	if after.Qty != 0 || after.Value != 0 {
		t.Errorf("清空后必须精确归零，实际 %d 件 / %s 元", after.Qty, after.Value)
	}
}

func TestApplyOutbound_PartialKeepsResidual(t *testing.T) {
	// 100 元 / 3 件，卖 1 件按 33.33 结转，剩 66.67
	p := Position{Qty: 3, Value: c(100)}
	cost, after, _ := p.ApplyOutbound(1)
	if cost != c(33.33) {
		t.Errorf("1 件成本应为 33.33，实际 %s", cost)
	}
	if after.Qty != 2 || after.Value != c(66.67) {
		t.Errorf("应为 2 件 / 66.67，实际 %d / %s", after.Qty, after.Value)
	}
	// 再清空：剩余 2 件应结转全部 66.67，残差被吸收，余额精确归零
	cost2, after2, err := after.ApplyOutbound(2)
	if err != nil {
		t.Fatalf("清空失败: %v", err)
	}
	if cost2 != c(66.67) {
		t.Errorf("清空应结转剩余 66.67，实际 %s", cost2)
	}
	if after2.Qty != 0 || after2.Value != 0 {
		t.Errorf("最终应精确归零，实际 %d / %s", after2.Qty, after2.Value)
	}
}

// TestApplyAtCost_Reversal 红冲按原单价反向，而不是当前均价。
func TestApplyAtCost_Reversal(t *testing.T) {
	// 10@5 + 10@7 = 20 件 / 120，均价 6
	p := Position{Qty: 20, Value: c(120)}
	after, err := p.ApplyAtCost(-10, c(5))
	if err != nil {
		t.Fatalf("按原价冲回失败: %v", err)
	}
	// 冲掉第一张 10@5 后，剩余的是第二批 10@7
	if after.Qty != 10 || after.Value != c(70) {
		t.Fatalf("应为 10 件 / 70.00，实际 %d / %s", after.Qty, after.Value)
	}
}

func TestApplyAtCost_Guards(t *testing.T) {
	p := Position{Qty: 3, Value: c(100)}
	if _, err := p.ApplyAtCost(-4, c(10)); err == nil {
		t.Error("冲回数量超过结存应报错")
	}
	if _, err := p.ApplyAtCost(0, c(10)); err == nil {
		t.Error("数量为 0 应报错")
	}
	if _, err := p.ApplyAtCost(1, c(-1)); err == nil {
		t.Error("负单价应报错")
	}
	// 按 1 元冲掉 3 件会让金额变成 97，数量归零但金额不为零，必须拒绝
	if _, err := p.ApplyAtCost(-3, c(1)); err == nil {
		t.Error("数量归零但金额不为零应报错")
	}
}

func TestApplyOutbound_Guards(t *testing.T) {
	if _, _, err := (Position{}).ApplyOutbound(1); err == nil {
		t.Error("结存 0 时出库应报错")
	}
	if _, _, err := (Position{Qty: 2, Value: c(10)}).ApplyOutbound(3); err == nil {
		t.Error("超出结存应报错")
	}
	if _, _, err := (Position{Qty: 2, Value: c(10)}).ApplyOutbound(0); err == nil {
		t.Error("出库 0 件应报错")
	}
	if (Position{}).AvgCost() != 0 {
		t.Error("结存 0 时均价应为 0")
	}
}

// Package ledger 是库存账本的**纯领域逻辑**：成本结转与不变量校验。
//
// 刻意不依赖 GORM / gin / 数据库：这部分是系统里最需要被反复验证的数学，
// 混进持久化细节之后就很难单独测、也很难在别人 review 时一眼看懂。
// 持久化与事务在 service/jxc 里，本包只做"纯函数式"的账。
package ledger

import (
	"errors"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
)

// ErrNoStock 结存为 0 时要求出库
var ErrNoStock = errors.New("结存为 0，无法出库")

// Position 某个 (仓库, SKU) 的结存：数量 + 金额。
// 用金额而不是平均单价做主状态，是因为平均单价是派生值，四舍五入后不可逆；
// 金额是账本累加出来的，可以精确校验。
type Position struct {
	Qty   int
	Value money.Cents
}

// AvgCost 当前移动加权平均单价。Qty<=0 时返回 0。
func (p Position) AvgCost() money.Cents {
	if p.Qty <= 0 {
		return 0
	}
	return money.Cents(roundDiv(int64(p.Value), int64(p.Qty)))
}

// ApplyInbound 入库：数量加、金额加（数量×入库单价）。
// unitCost 由调用方给定（采购单价格 / 手工指定成本 / 盘盈时的当前成本）。
func (p Position) ApplyInbound(qty int, unitCost money.Cents) Position {
	if qty <= 0 {
		return p
	}
	return Position{
		Qty:   p.Qty + qty,
		Value: p.Value + unitCost.MulQty(qty),
	}
}

// ApplyOutbound 出库：按**当前移动加权平均**结转成本，返回出库成本与新的结存。
//
// 两个关键细节：
//  1. 出库单价 = 当前平均值（四舍五入到分），不是"当前成本价"，
//     更不是 goods_sku 上那个会被最新进价覆盖的参考价。
//  2. 当一次把库存清空时，结转金额直接取剩余 Value（吸收累计四舍五入残差），
//     保证结存精确归零——否则会出现"库存 0 件但账上还有 0.01 元"这种脏数据。
func (p Position) ApplyOutbound(qty int) (cost money.Cents, after Position, err error) {
	if qty <= 0 {
		return 0, p, errors.New("出库数量必须大于 0")
	}
	if p.Qty <= 0 {
		return 0, p, ErrNoStock
	}
	if qty > p.Qty {
		return 0, p, errors.New("结存不足")
	}
	if qty == p.Qty {
		// 清空：把剩余金额全部结转，余额精确归零
		return p.Value, Position{Qty: 0, Value: 0}, nil
	}
	unit := p.AvgCost()
	cost = unit.MulQty(qty)
	return cost, Position{Qty: p.Qty - qty, Value: p.Value - cost}, nil
}

// ApplyAtCost 按指定单价增减库存，不经过移动加权平均。
//
// 只用于红冲：红冲必须按**原账本行的单价**精确反向，才能让金额抵消。
// 如果走 ApplyOutbound（当前移动均价），"10@5 + 10@7 后冲掉 10@5" 会按 6 元冲，
// 剩余批次的金额被改写成 60，而正确结果是 70。
//
// qty 有符号：正数增加、负数减少。减少时不允许把数量或金额冲成负数。
//
// 注意：它按"单价 × 数量"算金额。出库成本是移动加权平均后四舍五入到分的，
// 一条 3 件/100.00 的出库行单价只能是 33.33，再乘 3 就是 99.99。
// 因此红冲与"最后一次退货"必须走 ApplyAmount，按原账本行/原单的**精确金额**反向。
func (p Position) ApplyAtCost(qty int, unitCost money.Cents) (Position, error) {
	if qty == 0 {
		return p, errors.New("数量不能为 0")
	}
	if unitCost < 0 {
		return p, errors.New("单价不能为负")
	}
	if qty < 0 && -qty > p.Qty {
		return p, errors.New("结存不足")
	}
	after := Position{Qty: p.Qty + qty, Value: p.Value + unitCost.MulQty(qty)}
	if after.Value < 0 {
		return p, errors.New("按原价冲回后结存金额为负")
	}
	if after.Qty == 0 && after.Value != 0 {
		return p, errors.New("按原价冲回后数量归零但结存金额不为零；该批货很可能已被部分销售")
	}
	return after, nil
}

// ApplyAmount 按**精确金额**增减库存，不经过"单价 × 数量"。
//
// 为什么需要它：金额是账本的真相，单价只是展示/参考值。移动加权平均出库时
// 一条 3 件/100.00 的账本行单价只能是 33.33，若红冲按 33.33×3=99.99 反向，
// 就会永久留下 0.01 的残差。退货也有同样的问题：原单 3 件 100.00，分三次退，
// 前两次各 33.33，最后一次必须退 33.34 才能把原成本恰好退完。
//
// qty 有符号，amount 有符号且方向必须与 qty 一致；减少时不允许把数量或金额冲成负数。
func (p Position) ApplyAmount(qty int, amount money.Cents) (Position, error) {
	if qty == 0 {
		return p, errors.New("数量不能为 0")
	}
	if amount != 0 && (amount > 0) != (qty > 0) {
		return p, errors.New("金额方向与数量方向不一致")
	}
	if qty < 0 && -qty > p.Qty {
		return p, errors.New("结存不足")
	}
	after := Position{Qty: p.Qty + qty, Value: p.Value + amount}
	if after.Qty < 0 {
		return p, errors.New("结存数量不能为负")
	}
	if after.Value < 0 {
		return p, errors.New("结存金额不能为负")
	}
	if after.Qty == 0 && after.Value != 0 {
		return p, errors.New("数量归零但结存金额不为零；该批货很可能已被部分销售，或存在分位残差")
	}
	return after, nil
}

// roundDiv 整数除法四舍五入（对齐到"分"）
func roundDiv(a, b int64) int64 {
	if b == 0 {
		return 0
	}
	// Go 的整数除法向零截断；这里对正数做四舍五入
	if a >= 0 {
		return (a + b/2) / b
	}
	return -((-a + b/2) / b)
}

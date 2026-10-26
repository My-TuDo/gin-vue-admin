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

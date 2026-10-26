package ledger

import (
	"fmt"
	"sort"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
)

// 账本不变量。
//
// 这套东西的意义：余额（stock_balance）是物化视图，随时可能因为 bug、手工改库、
// 中途崩溃而漂移。成熟实现（如 ERPNext 的 Stock Ledger Invariant Check）都承认这一点，
// 区别在于它们**有检测手段**。这里把检测做成纯函数，便于单测、便于接成"一键对账"接口。
//
// 断言只有三条：
//  1. 余额数量 == 账本累加数量
//  2. 余额金额 == 账本累加金额
//  3. 可用量（余额 - 有效预留）不得为负
//
// 注意第 3 条不是"余额不得为负"：允许账面为负（比如先卖后录采购的补录场景）是业务选择，
// 但"可用量为负"意味着卖超了，必须报出来。

// Entry 参与校验的一条账本行（只取必要字段，避免依赖 model 包）
type Entry struct {
	WarehouseID uint
	SkuID       uint
	Qty         int
	Amount      money.Cents
}

// Violation 一条不变量违规
type Violation struct {
	Kind        string // qty_mismatch / value_mismatch / negative_available / orphan_balance
	WarehouseID uint
	SkuID       uint
	Expected    string
	Actual      string
	Detail      string
}

func (v Violation) String() string {
	return fmt.Sprintf("[%s] 仓库=%d SKU=%d 期望=%s 实际=%s %s",
		v.Kind, v.WarehouseID, v.SkuID, v.Expected, v.Actual, v.Detail)
}

// key 用于把账本行按 (仓库, SKU) 聚合
type key struct {
	WarehouseID uint
	SkuID       uint
}

// Check 校验账本与余额是否一致，返回所有违规项（空表示账是平的）。
//
// balances 传入当前物化余额；entries 传入账本行；reservations 传入有效预留数量（可为 nil）。
func Check(balances []PositionRow, entries []Entry, reservations []ReservationRow) []Violation {
	var out []Violation

	sumQty := map[key]int{}
	sumAmt := map[key]money.Cents{}
	for _, e := range entries {
		k := key{e.WarehouseID, e.SkuID}
		sumQty[k] += e.Qty
		sumAmt[k] += e.Amount
	}

	balanceSet := map[key]bool{}
	for _, b := range balances {
		k := key{b.WarehouseID, b.SkuID}
		balanceSet[k] = true

		if got := sumQty[k]; got != b.Qty {
			out = append(out, Violation{
				Kind: "qty_mismatch", WarehouseID: b.WarehouseID, SkuID: b.SkuID,
				Expected: fmt.Sprint(got), Actual: fmt.Sprint(b.Qty),
				Detail: "余额数量应等于账本累加",
			})
		}
		if got := sumAmt[k]; got != b.Value {
			out = append(out, Violation{
				Kind: "value_mismatch", WarehouseID: b.WarehouseID, SkuID: b.SkuID,
				Expected: got.String(), Actual: b.Value.String(),
				Detail: "余额金额应等于账本累加",
			})
		}
	}

	// 有账本行但余额表里没有对应行 —— 同样是不一致
	for k, q := range sumQty {
		if !balanceSet[k] && q != 0 {
			out = append(out, Violation{
				Kind: "orphan_balance", WarehouseID: k.WarehouseID, SkuID: k.SkuID,
				Expected: fmt.Sprint(q), Actual: "0",
				Detail: "存在账本行但缺少余额行",
			})
		}
	}

	// 可用量不得为负
	reserved := map[key]int{}
	for _, r := range reservations {
		if r.Status != 1 {
			continue
		}
		reserved[key{r.WarehouseID, r.SkuID}] += r.Qty
	}
	for _, b := range balances {
		k := key{b.WarehouseID, b.SkuID}
		if avail := b.Qty - reserved[k]; avail < 0 {
			out = append(out, Violation{
				Kind: "negative_available", WarehouseID: b.WarehouseID, SkuID: b.SkuID,
				Expected: ">= 0", Actual: fmt.Sprint(avail),
				Detail: "可售量（余额-预留）为负，说明卖超了",
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].WarehouseID != out[j].WarehouseID {
			return out[i].WarehouseID < out[j].WarehouseID
		}
		if out[i].SkuID != out[j].SkuID {
			return out[i].SkuID < out[j].SkuID
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

// PositionRow 余额行（只取校验需要的字段）
type PositionRow struct {
	WarehouseID uint
	SkuID       uint
	Qty         int
	Value       money.Cents
}

// ReservationRow 预留行（只取校验需要的字段）
type ReservationRow struct {
	WarehouseID uint
	SkuID       uint
	Qty         int
	Status      int8
}

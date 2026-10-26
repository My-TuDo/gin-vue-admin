package ledger

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
)

func TestCheck_Balanced(t *testing.T) {
	balances := []PositionRow{{WarehouseID: 1, SkuID: 1, Qty: 8, Value: money.FromYuan(40)}}
	entries := []Entry{
		{WarehouseID: 1, SkuID: 1, Qty: 10, Amount: money.FromYuan(50)},
		{WarehouseID: 1, SkuID: 1, Qty: -2, Amount: money.FromYuan(-10)},
	}
	if v := Check(balances, entries, nil); len(v) != 0 {
		t.Fatalf("账是平的，不应有违规，实际：%v", v)
	}
}

func TestCheck_DetectsQtyMismatch(t *testing.T) {
	balances := []PositionRow{{WarehouseID: 1, SkuID: 1, Qty: 9, Value: money.FromYuan(50)}} // 应为 8
	entries := []Entry{
		{WarehouseID: 1, SkuID: 1, Qty: 10, Amount: money.FromYuan(50)},
		{WarehouseID: 1, SkuID: 1, Qty: -2, Amount: money.FromYuan(0)},
	}
	vs := Check(balances, entries, nil)
	if len(vs) != 1 || vs[0].Kind != "qty_mismatch" {
		t.Fatalf("应只报数量不一致，实际：%v", vs)
	}
	if vs[0].Expected != "8" || vs[0].Actual != "9" {
		t.Errorf("期望/实际字段不对：%+v", vs[0])
	}
}

func TestCheck_DetectsValueMismatch(t *testing.T) {
	balances := []PositionRow{{WarehouseID: 1, SkuID: 1, Qty: 8, Value: money.FromYuan(41)}}
	entries := []Entry{
		{WarehouseID: 1, SkuID: 1, Qty: 10, Amount: money.FromYuan(50)},
		{WarehouseID: 1, SkuID: 1, Qty: -2, Amount: money.FromYuan(-10)},
	}
	vs := Check(balances, entries, nil)
	if len(vs) != 1 || vs[0].Kind != "value_mismatch" {
		t.Fatalf("应报金额不一致，实际：%v", vs)
	}
}

func TestCheck_DetectsOrphanEntries(t *testing.T) {
	// 有账本行但余额表没有对应行
	entries := []Entry{{WarehouseID: 2, SkuID: 9, Qty: 5, Amount: money.FromYuan(25)}}
	vs := Check(nil, entries, nil)
	if len(vs) != 1 || vs[0].Kind != "orphan_balance" {
		t.Fatalf("应报缺少余额行，实际：%v", vs)
	}
}

func TestCheck_NegativeAvailable(t *testing.T) {
	balances := []PositionRow{{WarehouseID: 1, SkuID: 1, Qty: 2, Value: money.FromYuan(10)}}
	// 账面 2 件，却预留了 5 件 → 卖超
	reservations := []ReservationRow{{WarehouseID: 1, SkuID: 1, Qty: 5, Status: 1}}
	// 没给 entries，所以数量/金额也会被报成不一致；这里重点是 negative_available 必须出现
	vs := Check(balances, nil, reservations)
	var hasAvail bool
	for _, v := range vs {
		if v.Kind == "negative_available" {
			hasAvail = true
			if v.Actual != "-3" {
				t.Errorf("可用量应为 -3，实际 %s", v.Actual)
			}
		}
	}
	if !hasAvail {
		t.Fatalf("未报出 negative_available，实际：%v", vs)
	}
}

func TestCheck_IgnoresReleasedReservations(t *testing.T) {
	balances := []PositionRow{{WarehouseID: 1, SkuID: 1, Qty: 2, Value: money.FromYuan(10)}}
	entries := []Entry{{WarehouseID: 1, SkuID: 1, Qty: 2, Amount: money.FromYuan(10)}}
	reservations := []ReservationRow{{WarehouseID: 1, SkuID: 1, Qty: 99, Status: 2}} // 已释放
	if vs := Check(balances, entries, reservations); len(vs) != 0 {
		t.Fatalf("已释放的预留不应影响可用量，实际：%v", vs)
	}
}

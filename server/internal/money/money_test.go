package money

import "testing"

func TestFromYuan_DecimalExact(t *testing.T) {
	cases := []struct {
		in   float64
		want Cents
	}{
		{0, 0},
		{0.1, 10},
		{0.2, 20},
		{0.3, 30},
		{0.1 + 0.2, 30}, // 0.30000000000000004 应规整为 30 分
		{1, 100},
		{19.99, 1999},
		// 注意：0.615 在 float64 里的实际值是 0.61499999...，
		// 因此按"真实值"舍入得到 0.61。本包的语义是「舍入这个数实际是什么」，
		// 而不是「舍入用户在纸上写的那串数字」——后者需要 decimal 类型，超出当前范围。
		{0.615, 61},
		{1.005, 100},
		{-3.45, -345},
		{1234567.89, 123456789},
	}
	for _, c := range cases {
		if got := FromYuan(c.in); got != c.want {
			t.Errorf("FromYuan(%v) = %d 分，期望 %d 分", c.in, got, c.want)
		}
	}
}

func TestCents_MulQtyAndYuan(t *testing.T) {
	// 0.1 元 × 3 件 = 0.3 元，不能出现 0.30000000000000004
	total := FromYuan(0.1).MulQty(3)
	if total != 30 {
		t.Errorf("0.1×3 = %d 分，期望 30 分", total)
	}
	if got := total.Yuan(); got != 0.3 {
		t.Errorf("0.1×3 转元 = %v，期望 0.3", got)
	}
}

func TestCents_Compare(t *testing.T) {
	// 累加之后仍然精确相等：这正是「实收 = 应收」判定的关键
	var total Cents
	for i := 0; i < 10; i++ {
		total += FromYuan(0.1)
	}
	if total != FromYuan(1) {
		t.Errorf("累加 10 次 0.1 应等于 1 元，got %d 分", total)
	}
	if total.LessThan(FromYuan(1)) {
		t.Error("1 元不应小于 1 元（float64 累加会出现 0.9999999999999999 < 1）")
	}
}

func TestSum(t *testing.T) {
	if got := Sum(0.1, 0.2, 0.3); got != 60 {
		t.Errorf("Sum(0.1,0.2,0.3) = %d 分，期望 60 分", got)
	}
	if got := Sum().Yuan(); got != 0 {
		t.Errorf("空求和应为 0 元，got %v", got)
	}
}

func TestString(t *testing.T) {
	if got := FromYuan(19.9).String(); got != "19.90" {
		t.Errorf("String() = %q，期望 \"19.90\"", got)
	}
	if got := FromYuan(-1.5).String(); got != "-1.50" {
		t.Errorf("负金额 String() = %q，期望 \"-1.50\"", got)
	}
}

func TestRoundYuan(t *testing.T) {
	cases := map[float64]float64{
		0.1 + 0.2: 0.3,
		19.999:    20,
		9.994:     9.99,
		9.995:     9.99, // 9.995 的实际值是 9.99499...，舍入为 9.99（见 TestFromYuan_DecimalExact 注释）
	}
	for in, want := range cases {
		if got := RoundYuan(in); got != want {
			t.Errorf("RoundYuan(%v) = %v，期望 %v", in, got, want)
		}
	}
}

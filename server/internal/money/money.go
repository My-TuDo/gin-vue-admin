// Package money 提供以「分」为单位的金额运算。
//
// 为什么需要它：金额字段在数据库里是 decimal(12,2)、在 Go 里是 float64。
// float64 无法精确表示 0.1、0.2 这类十进制小数，累加几次就会出现
// 30.000000000000004 这样的值。历史上因此出现过真实故障：
// 收银台现金校验是严格比较 `实收 < 应收`，用户输入精确的 0.30 却被判「实收金额不足」。
//
// 本包的约定（见 docs/adr/0004）：
//   - 对外（JSON、数据库）仍然用「元 + 2 位小数」表达，不动既有契约；
//   - 对内（累加、乘积、比较）一律换算成「分」的整数运算；
//   - 任何写回金额字段的地方都先经过本包转成元，保证落库是 2 位小数。
package money

import (
	"strconv"
	"strings"
)

// Cents 以分为单位的金额。
type Cents int64

// FromYuan 把「元」转成「分」，按十进制四舍五入到分。
//
// 实现刻意走 strconv 的十进制格式化而不是 `y*100`：
// y*100 会继承 float64 的误差（0.615*100 = 61.49999999999999），
// 而 FormatFloat 会先得到该浮点数最短的正确十进制表示再补齐 2 位，结果符合直觉。
func FromYuan(y float64) Cents {
	s := strconv.FormatFloat(y, 'f', 2, 64)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, fracPart, _ := strings.Cut(s, ".")
	yuan, _ := strconv.ParseInt(intPart, 10, 64)
	var frac int64
	if fracPart != "" {
		fracPart += "00"
		v, err := strconv.ParseInt(fracPart[:2], 10, 64)
		if err == nil {
			frac = v
		}
	}
	c := Cents(yuan*100 + frac)
	if neg {
		return -c
	}
	return c
}

// Yuan 转回「元」，供写库/返回给前端使用。
func (c Cents) Yuan() float64 { return float64(c) / 100 }

// RoundYuan 把元的价格/金额规整到 2 位小数。
func RoundYuan(y float64) float64 { return FromYuan(y).Yuan() }

// MulQty 单价 × 数量（数量为整数）。
func (c Cents) MulQty(qty int) Cents { return Cents(int64(c) * int64(qty)) }

// Add 相加。
func (c Cents) Add(o Cents) Cents { return c + o }

// Sub 相减。
func (c Cents) Sub(o Cents) Cents { return c - o }

// Abs 取绝对值
func (c Cents) Abs() Cents {
	if c < 0 {
		return -c
	}
	return c
}

// Neg 取负。
func (c Cents) Neg() Cents { return -c }

// LessThan 严格小于。
func (c Cents) LessThan(o Cents) bool { return c < o }

// IsZero 是否为零。
func (c Cents) IsZero() bool { return c == 0 }

// String 便于拼错误提示。
func (c Cents) String() string { return strconv.FormatFloat(c.Yuan(), 'f', 2, 64) }

// Sum 对一组「元」金额求和，返回「分」。
func Sum(values ...float64) Cents {
	var total Cents
	for _, v := range values {
		total += FromYuan(v)
	}
	return total
}

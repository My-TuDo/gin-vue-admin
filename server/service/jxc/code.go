package jxc

import (
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// 各实体编码前缀（统一大写，日期紧跟前缀避免跨实体误匹配）
const (
	prefixCategory  = "C"  // 商品分类
	prefixBrand     = "B"  // 品牌
	prefixSupplier  = "S"  // 供应商
	prefixCustomer  = "K"  // 客户
	prefixWarehouse = "W"  // 仓库
	prefixGoods     = "SP" // 商品
)

// genCode 生成业务编码: 前缀 + 日期(yyyymmdd) + 3位当日序号
// 示例: SP20260803001；序号按当日同前缀最大编码自增，跨天自动重置。
func genCode(db *gorm.DB, model interface{}, prefix string) (string, error) {
	date := time.Now().Format("20060102")
	like := prefix + date + "%"
	var maxCode string
	if err := db.Model(model).Where("code LIKE ?", like).
		Order("code DESC").Limit(1).Pluck("code", &maxCode).Error; err != nil {
		return "", err
	}
	if maxCode == "" {
		return prefix + date + "001", nil
	}
	seq, err := strconv.Atoi(maxCode[len(maxCode)-3:])
	if err != nil {
		return "", fmt.Errorf("解析已有编码序号失败: %s", maxCode)
	}
	return fmt.Sprintf("%s%s%03d", prefix, date, seq+1), nil
}

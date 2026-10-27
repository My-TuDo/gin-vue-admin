package initialize

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// perfIndex 一个复合索引定义。
// table/columns 都是代码常量，不接受外部输入，因此可以安全拼进 DDL。
type perfIndex struct {
	table   string
	name    string
	columns string
}

// perfIndexes 单据列表默认按 created_at 倒序，并常用 status/仓库/往来单位筛选。
// 单列索引只能命中其中一个条件；这里补 (status, created_at) 这类复合索引，
// 让“状态 + 时间范围 + 倒序”走同一条索引。
//
// 注意：这些是**加索引**，不改列语义；老库首次启动会逐个检查后创建。
var perfIndexes = []perfIndex{
	{"purchase_order", "idx_po_status_created", "status, created_at"},
	{"purchase_order", "idx_po_wh_created", "warehouse_id, created_at"},
	{"purchase_order", "idx_po_supplier_created", "supplier_id, created_at"},

	{"sale_order", "idx_so_status_created", "status, created_at"},
	{"sale_order", "idx_so_wh_created", "warehouse_id, created_at"},
	{"sale_order", "idx_so_customer_created", "customer_id, created_at"},

	{"sale_return", "idx_sar_status_created", "status, created_at"},
	{"sale_return", "idx_sar_wh_created", "warehouse_id, created_at"},
	{"sale_return", "idx_sar_sale_created", "sale_id, created_at"},

	{"purchase_return", "idx_pr_status_created", "status, created_at"},
	{"purchase_return", "idx_pr_wh_created", "warehouse_id, created_at"},
	{"purchase_return", "idx_pr_supplier_created", "supplier_id, created_at"},

	{"stock_reversal", "idx_rev_type_posted", "source_type, posted_at"},
}

// ensurePerfIndexes 在 MySQL 上幂等地补列表性能索引。
// 其它方言（测试里的 sqlite）直接跳过，避免 information_schema 不存在。
func ensurePerfIndexes(db *gorm.DB) error {
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	for _, idx := range perfIndexes {
		var count int64
		if err := db.Raw(`SELECT COUNT(*) FROM information_schema.statistics
			WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`,
			idx.table, idx.name).Scan(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		ddl := fmt.Sprintf("CREATE INDEX `%s` ON `%s` (%s)", idx.name, idx.table, idx.columns)
		if err := db.Exec(ddl).Error; err != nil {
			// 并发启动时另一个实例可能刚好创建成功；重复索引错误可以忽略。
			if strings.Contains(err.Error(), "Duplicate key name") ||
				strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("创建索引 %s 失败: %w", idx.name, err)
		}
	}
	return nil
}

package jxc

import (
	"context"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"gorm.io/gorm"
)

type GenericService[T any] struct{}

// db 获取数据库连接
func (s *GenericService[T]) db(ctx context.Context) *gorm.DB {
	return global.GVA_DB.WithContext(ctx)
}

// Create 创建
func (s *GenericService[T]) Create(ctx context.Context, m *T) error {
	return s.db(ctx).Create(m).Error
}

// Update 更新
func (s *GenericService[T]) Update(ctx context.Context, m *T) error {
	return s.db(ctx).Omit("code").Save(m).Error // 忽略code字段，避免更新时修改唯一约束
}

// Delete 软删除（需要 T 嵌入 GVA_MODEL/Deleted_At）
// new(T) 根据泛型T 创建一个指向 T 的零值指针 *T
// s.db(ctx) 拿到带有上下文的gorm.db实例
func (s *GenericService[T]) Delete(ctx context.Context, id uint) error {
	return s.db(ctx).Delete(new(T), id).Error
}

// GetPage 分页 + keyword 查询
func (s *GenericService[T]) GetPage(ctx context.Context, info request.PageInfo, searchFields []string, extraScopes ...func(*gorm.DB) *gorm.DB) (list []T, total int64, err error) {
	db := s.db(ctx).Model(new(T))
	clause, values := keywordClause(searchFields, info.Keyword)
	if clause != "" {
		db = db.Where(clause, values...)
	}
	if len(extraScopes) > 0 {
		db = db.Scopes(extraScopes...)
	}
	err = db.Count(&total).Error
	if err != nil || total == 0 {
		return
	}
	limit, offset := info.LimitOffset()
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return
}

// keywordClause 构建关键字查询条件
func keywordClause(fields []string, keyword string) (clause string, values []interface{}) {
	if len(fields) == 0 || keyword == "" {
		return "", nil
	}
	patten := "%" + keyword + "%"
	parts := make([]string, len(fields))
	values = make([]interface{}, len(fields))
	for i, f := range fields {
		parts[i] = f + " LIKE ?"
		values[i] = patten
	}
	// 防止 OR 条件破坏其他条件优先级
	clause = "(" + strings.Join(parts, " OR ") + ")"
	return clause, values
}

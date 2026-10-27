package system

import (
	"context"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
)

// 原测试既没有初始化 global.GVA_DB（Preview 第一行就解引用 nil），
// 又把 want 写成 nil 却对返回的 map 做 DeepEqual——无论环境如何都不可能通过。
// 这里补上内存库与模板包记录，断言"能预览出带代码块的模板"。

func Test_autoCodeTemplate_Preview(t *testing.T) {
	useRepoRoot(t)
	testutil.NewMemoryDB(t, &model.SysAutoCodePackage{}, &model.SysAutoCodeHistory{})

	if err := global.GVA_DB.Create(&model.SysAutoCodePackage{
		Label:       "gva",
		Template:    "package",
		PackageName: "gva",
	}).Error; err != nil {
		t.Fatalf("创建模板包记录失败: %v", err)
	}

	info := request.AutoCode{
		Package:             "gva",
		StructName:          "SysUser",
		TableName:           "sys_users",
		PackageName:         "sysUsers",
		Description:         "用户",
		Abbreviation:        "sysUsers",
		HumpPackageName:     "sys_users",
		GvaModel:            true,
		AutoMigrate:         true,
		AutoCreateApiToSql:  true,
		AutoCreateMenuToSql: true,
		GenerateWeb:         true,
		GenerateServer:      true,
		Fields: []*request.AutoCodeField{
			{
				FieldName: "Username", FieldDesc: "用户名", FieldType: "string",
				DataTypeLong: "191", FieldJson: "username",
				ColumnName: "username", Comment: "用户名",
			},
			{
				FieldName: "Age", FieldDesc: "年龄", FieldType: "int",
				DataTypeLong: "20", FieldJson: "age",
				ColumnName: "age", Comment: "年龄",
			},
		},
	}
	if err := info.Pretreatment(); err != nil {
		t.Fatalf("Pretreatment 失败: %v", err)
	}

	got, err := AutoCodeTemplate.Preview(context.Background(), info)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if len(got) == 0 {
		t.Fatal("Preview() 没有返回任何模板")
	}
	for name, content := range got {
		if !strings.HasPrefix(content, "```") {
			t.Errorf("%s 的预览内容不是代码块：%.40s", name, content)
		}
	}
}

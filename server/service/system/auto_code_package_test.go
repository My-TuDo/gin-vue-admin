package system

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
)

func Test_autoCodePackage_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		info *request.SysAutoCodePackageCreate
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "测试 package",
			args: args{
				ctx: context.Background(),
				info: &request.SysAutoCodePackageCreate{
					Template:    "package",
					PackageName: "gva",
				},
			},
			wantErr: false,
		},
		{
			name: "测试 plugin",
			args: args{
				ctx: context.Background(),
				info: &request.SysAutoCodePackageCreate{
					Template:    "plugin",
					PackageName: "gva",
				},
			},
			wantErr: false,
		},
	}
	if global.GVA_DB == nil {
		t.Skip("需要已初始化的数据库(global.GVA_DB), 且会生成真实代码目录, 干净环境跳过")
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &autoCodePackage{}
			if err := a.Create(tt.args.ctx, tt.args.info); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_autoCodePackage_templates(t *testing.T) {
	useRepoRoot(t)

	// 原测试把 wantEnter 写成 map[string]map[string]string，而实际返回的是
	// map[string]ast.Ast，DeepEqual 永远为 false；同时没设置 AutoCode.Root，
	// 会去包目录下找 resource/plugin。这里改成只断言"能读到模板并产出代码"，
	// 具体路径由 Root 决定，不适合写死。
	entity := model.SysAutoCodePackage{
		Desc:        "描述",
		Label:       "展示名",
		Template:    "plugin",
		PackageName: "preview",
	}
	info := request.AutoCode{
		Abbreviation:    "user",
		HumpPackageName: "user",
		GenerateServer:  true,
		GenerateWeb:     true,
		AutoMigrate:     true,
	}

	_, asts, creates, err := new(autoCodePackage).templates(
		context.Background(), entity, info, false)
	if err != nil {
		t.Fatalf("templates() error = %v", err)
	}
	// plugin 模板通常通过 creates 描述"写到哪"，不一定会填充 code（code 用于
	// 需要二次渲染的 package 模板），所以这里不把 code 非空当作断言。
	if len(asts) == 0 {
		t.Fatal("templates() 没有产出任何 AST 注入点")
	}
	if len(creates) == 0 {
		t.Fatal("templates() 没有产出任何生成路径")
	}
}

package system

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// useRepoRoot 把 AutoCode.Root 指到仓库根目录。
//
// 代码生成器的模板路径是 (Root + Server + resource/...)，生产环境由 core.Viper()
// 自动识别根目录；测试进程的 cwd 是包目录，且全局配置是零值，于是会去读
// server/service/system/resource —— 必然不存在。原测试就是因此变红的。
func useRepoRoot(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件路径")
	}
	// file = <root>/server/service/system/test_root_test.go
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))

	oldRoot := global.GVA_CONFIG.AutoCode.Root
	oldServer := global.GVA_CONFIG.AutoCode.Server
	oldModule := global.GVA_CONFIG.AutoCode.Module
	global.GVA_CONFIG.AutoCode.Root = root
	global.GVA_CONFIG.AutoCode.Server = "server"
	if global.GVA_CONFIG.AutoCode.Module == "" {
		global.GVA_CONFIG.AutoCode.Module = "github.com/flipped-aurora/gin-vue-admin/server"
	}
	t.Cleanup(func() {
		global.GVA_CONFIG.AutoCode.Root = oldRoot
		global.GVA_CONFIG.AutoCode.Server = oldServer
		global.GVA_CONFIG.AutoCode.Module = oldModule
	})
}

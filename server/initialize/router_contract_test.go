package initialize

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 路由鉴权契约测试。
//
// 存在理由：GVA 的鉴权是**按路由组显式挂载**的（只有 PrivateGroup 带 JWT/Casbin），
// 一旦业务路由被注册到 PublicGroup，它就直接裸奔——旧项目就是这么丢掉 96 条路由的。
// 所以这里不测"能不能登录"，只测"有没有挂错组"。
func TestJxcRoutesMustBePrivate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	privateEngine := gin.New()
	publicEngine := gin.New()
	initBizRouter(privateEngine.Group("/"), publicEngine.Group("/"))

	// 1) 公开组不允许出现任何 jxc 业务路由
	for _, r := range publicEngine.Routes() {
		if strings.HasPrefix(r.Path, "/jxc/") {
			t.Errorf("业务路由不允许挂在公开组：%s %s", r.Method, r.Path)
		}
	}

	// 2) 私有组必须真的注册了路由，且覆盖四块能力
	private := privateEngine.Routes()
	if len(private) == 0 {
		t.Fatal("私有组没有任何进销存路由，注册逻辑可能没生效")
	}
	want := map[string]bool{
		"/jxc/purchase":          false,
		"/jxc/sale":              false,
		"/jxc/ledger/reconcile":  false,
		"/jxc/master/warehouses": false,
	}
	for _, r := range private {
		if _, ok := want[r.Path]; ok {
			want[r.Path] = true
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("缺少关键路由：%s", path)
		}
	}
}

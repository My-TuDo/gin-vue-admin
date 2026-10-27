package initialize

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInitializeRouterRegistersAutoCodeRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	InitializeRouter(engine)

	foundAutoCode := false
	for _, route := range engine.Routes() {
		if route.Path == "/autoCode/getDB" {
			foundAutoCode = true
		}
	}

	if !foundAutoCode {
		t.Fatalf("expected /autoCode/getDB to be registered by plugin router")
	}
	// 注意：/skills/* 属于主系统 / plugin-ai 的路由，不在 auto 插件里。
	// 原测试要求 auto 插件注册 /skills/getTools，与实现不符；照抄会重复注册并 panic。
}

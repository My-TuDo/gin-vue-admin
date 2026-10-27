package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router"
	"github.com/gin-gonic/gin"
)

// 占位方法，保证文件可以正确加载，避免go空变量检测报错，请勿删除。
func holder(routers ...*gin.RouterGroup) {
	_ = routers
	_ = router.RouterGroupApp
}

func initBizRouter(routers ...*gin.RouterGroup) {
	privateGroup := routers[0]
	publicGroup := routers[1]

	holder(publicGroup, privateGroup)

	// 进销存全部路由挂在 privateGroup：
	// JWT 鉴权 → 强制改密 → Casbin 权限 → 数据权限，缺一不可。
	router.RouterGroupApp.Jxc.InitMasterRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitPartnerRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitDocumentRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitPurchaseReturnRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitDashboardRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitLedgerRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitReversalRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitReturnRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitCheckRouter(privateGroup)
	router.RouterGroupApp.Jxc.InitPeriodRouter(privateGroup)
}

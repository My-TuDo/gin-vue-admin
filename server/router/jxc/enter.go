package jxc

// RouterGroup 进销存路由组。
// 与 GVA 生态保持一致：业务路由一律挂在 PrivateGroup 下（JWT + Casbin + DataScope）。
type RouterGroup struct {
	DocumentRouter
	PurchaseReturnRouter
	DashboardRouter
	LedgerRouter
	MasterRouter
	ReversalRouter
	ReturnRouter
	CheckRouter
	PeriodRouter
	PartnerRouter
}

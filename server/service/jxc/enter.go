package jxc

// ServiceGroup 进销存服务组。
//
// 分层约定：
//   - LedgerService 是**唯一**写库存的入口（账本行 + 余额 + 预留）；
//   - Purchase/Sale 只负责业务规则与状态机，落账一律委托给 LedgerService；
//   - 任何绕过 LedgerService 直接改 stock_balance 的代码都是 bug。
type ServiceGroup struct {
	Ledger   *LedgerService
	Purchase *PurchaseService
	Sale     *SaleService
}

// ServiceGroupApp 进程级单例。
// 必须显式构造：如果留成零值，字段就是 nil，handler 一解引用就 panic（返回 500 空响应，
// 排查起来极其费劲——这条注释是用一次 500 换来的）。
var ServiceGroupApp = &ServiceGroup{
	Ledger:   new(LedgerService),
	Purchase: NewPurchaseService(),
	Sale:     NewSaleService(),
}

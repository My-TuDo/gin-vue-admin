package jxc

// ServiceGroup 进销存服务组。
//
// 分层约定：
//   - LedgerService 是**唯一**写库存的入口（账本行 + 余额 + 预留）；
//   - Purchase/Sale 等单据服务只负责业务规则与状态机，落账一律委托给 LedgerService；
//   - 任何绕过 LedgerService 直接改 stock_balance 的代码都是 bug。
type ServiceGroup struct {
	Ledger   *LedgerService
	Purchase *PurchaseService
	Sale     *SaleService
}

var ServiceGroupApp = new(ServiceGroup)

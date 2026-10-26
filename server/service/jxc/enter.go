package jxc

// ServiceGroup 进销存服务组。
// 账本是唯一写库存的入口（LedgerService），后续的单据服务（采购/销售/盘点）
// 都只负责"业务规则 + 组装 MoveInput"，把落账交给 LedgerService。
type ServiceGroup struct {
	LedgerService
}

var ServiceGroupApp = new(ServiceGroup)

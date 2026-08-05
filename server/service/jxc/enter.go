package jxc

type ServiceGroup struct {
	BasicService
	GoodsService
	PurchaseService
	SaleService
	StockService
	StockCheckService
	CashierService
	DashboardService
}

var JxcServiceGroup = new(ServiceGroup)

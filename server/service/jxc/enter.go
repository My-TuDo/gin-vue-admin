package jxc

type ServiceGroup struct {
	BasicService
	GoodsService
	PurchaseService
	SaleService
	StockService
	StockCheckService
}

var JxcServiceGroup = new(ServiceGroup)

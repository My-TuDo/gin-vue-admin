package jxc

type ServiceGroup struct {
	BasicService
	GoodsService
	PurchaseService
	SaleService
	StockService
}

var JxcServiceGroup = new(ServiceGroup)

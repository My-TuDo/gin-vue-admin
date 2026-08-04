package jxc

type ServiceGroup struct {
	BasicService
	GoodsService
	PurchaseService
	SaleService
}

var JxcServiceGroup = new(ServiceGroup)

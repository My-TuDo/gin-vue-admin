package jxc

type ServiceGroup struct {
	BasicService
	GoodsService
	PurchaseService
}

var JxcServiceGroup = new(ServiceGroup)

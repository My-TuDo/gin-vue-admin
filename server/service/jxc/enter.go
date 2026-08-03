package jxc

type ServiceGroup struct {
	BasicService
	GoodsService
}

var JxcServiceGroup = new(ServiceGroup)

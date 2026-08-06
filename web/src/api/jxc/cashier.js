import service from '@/utils/request'

// 收银结算（收款即出库）
export const checkout = (data) => service.post('/jxc/cashier/checkout', data)
// 收银退款（退货单直接入库）
export const refund = (data) => service.post('/jxc/cashier/refund', data)
// 收银换货（退回入库 + 换出出库，差额多退少补）
export const exchange = (data) => service.post('/jxc/cashier/exchange', data)

import service from '@/utils/request'

// 收银结算（收款即出库）
export const checkout = (data) => service.post('/jxc/cashier/checkout', data)

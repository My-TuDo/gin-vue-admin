import service from '@/utils/request'

// 销售单
export const getSalePage = (params) => service.get('/jxc/sale/page', { params })
export const getSaleDetail = (id) => service.get('/jxc/sale/detail', { params: { id } })
export const createSaleOrder = (data) => service.post('/jxc/sale', data)
export const updateSaleOrder = (data) => service.put('/jxc/sale', data)
export const deleteSaleOrder = (id) => service.delete('/jxc/sale', { data: { id } })
export const cancelSaleOrder = (id) => service.put('/jxc/sale/cancel', { id })
export const confirmOut = (id) => service.put('/jxc/sale/out', { id })
export const confirmReturn = (id) => service.put('/jxc/sale/return', { id })
export const confirmExchange = (id) => service.put('/jxc/sale/exchange', { id })

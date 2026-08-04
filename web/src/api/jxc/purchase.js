import service from '@/utils/request'

// 采购单
export const getPurchasePage = (params) => service.get('/jxc/purchase/page', { params })
export const getPurchaseDetail = (id) => service.get('/jxc/purchase/detail', { params: { id } })
export const createPurchase = (data) => service.post('/jxc/purchase', data)
export const updatePurchase = (data) => service.put('/jxc/purchase', data)
export const deletePurchase = (id) => service.delete('/jxc/purchase', { data: { id } })
export const auditPurchase = (id) => service.put('/jxc/purchase/audit', { id })
export const cancelPurchase = (id) => service.put('/jxc/purchase/cancel', { id })
export const stockInPurchase = (id) => service.put('/jxc/purchase/stock-in', { id })

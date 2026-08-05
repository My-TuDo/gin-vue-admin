import service from '@/utils/request'

// 盘点单
export const getStockCheckPage = (params) => service.get('/jxc/stockcheck/page', { params })
export const getStockCheckDetail = (id) => service.get('/jxc/stockcheck/detail', { params: { id } })
export const createStockCheck = (data) => service.post('/jxc/stockcheck', data)
export const updateStockCheckItems = (data) => service.put('/jxc/stockcheck/items', data)
export const completeStockCheck = (id) => service.put('/jxc/stockcheck/complete', { id })
export const cancelStockCheck = (id) => service.put('/jxc/stockcheck/cancel', { id })

import service from '@/utils/request'

// 库存中心
export const getStockPage = (params) => service.get('/jxc/stock/page', { params })
export const getStockLogPage = (params) => service.get('/jxc/stock/log/page', { params })
export const directIn = (data) => service.post('/jxc/stock/direct-in', data)
export const directOut = (data) => service.post('/jxc/stock/direct-out', data)

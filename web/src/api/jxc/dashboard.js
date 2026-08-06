import service from '@/utils/request'

// 仪表盘
export const getDashboardOverview = () => service.get('/jxc/dashboard/overview')
export const getDashboardTrend = (days) => service.get('/jxc/dashboard/trend', { params: { days } })
export const getDashboardTop = (days, limit) => service.get('/jxc/dashboard/top', { params: { days, limit } })
export const getDashboardStockAlert = () => service.get('/jxc/dashboard/stock-alert')
export const getDashboardCategory = (days) => service.get('/jxc/dashboard/category', { params: { days } })
export const getSalesHistory = (from, to, granularity) => service.get('/jxc/dashboard/sales-history', { params: { from, to, granularity } })

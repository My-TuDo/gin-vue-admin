import service from '@/utils/request'

// 进销存接口封装。约定：所有方法返回 { code, data, msg }，code===0 为成功。

// ---------- 基础资料 ----------
export const listWarehouses = () => service({ url: '/jxc/master/warehouses', method: 'get' })
export const createWarehouse = (data) => service({ url: '/jxc/master/warehouses', method: 'post', data })
export const listGoods = () => service({ url: '/jxc/master/goods', method: 'get' })
export const createGoods = (data) => service({ url: '/jxc/master/goods', method: 'post', data })
export const listSkus = (params) => service({ url: '/jxc/master/skus', method: 'get', params })

// ---------- 采购 ----------
export const createPurchase = (data) => service({ url: '/jxc/purchase', method: 'post', data })
export const pagePurchase = (params) => service({ url: '/jxc/purchase/page', method: 'get', params })
export const getPurchase = (id) => service({ url: `/jxc/purchase/${id}`, method: 'get' })
export const approvePurchase = (id) => service({ url: `/jxc/purchase/${id}/approve`, method: 'post' })
export const receivePurchase = (id) => service({ url: `/jxc/purchase/${id}/receive`, method: 'post' })
export const cancelPurchase = (id) => service({ url: `/jxc/purchase/${id}/cancel`, method: 'post' })

// ---------- 销售 ----------
export const createSale = (data) => service({ url: '/jxc/sale', method: 'post', data })
export const pageSale = (params) => service({ url: '/jxc/sale/page', method: 'get', params })
export const getSale = (id) => service({ url: `/jxc/sale/${id}`, method: 'get' })
export const shipSale = (id) => service({ url: `/jxc/sale/${id}/ship`, method: 'post' })
export const cancelSale = (id) => service({ url: `/jxc/sale/${id}/cancel`, method: 'post' })

// ---------- 账本 ----------
export const ledgerBalances = (params) => service({ url: '/jxc/ledger/balance', method: 'get', params })
export const ledgerMoves = (params) => service({ url: '/jxc/ledger/moves', method: 'get', params })
export const ledgerReconcile = (params) => service({ url: '/jxc/ledger/reconcile', method: 'get', params })

// ---------- 红冲 ----------
export const createReversal = (data) => service({ url: '/jxc/reversal', method: 'post', data })
export const pageReversal = (params) => service({ url: '/jxc/reversal/page', method: 'get', params })

// ---------- 退货 ----------
export const createReturn = (data) => service({ url: '/jxc/return', method: 'post', data })
export const pageReturn = (params) => service({ url: '/jxc/return/page', method: 'get', params })
export const getReturn = (id) => service({ url: `/jxc/return/${id}`, method: 'get' })
export const postReturn = (id) => service({ url: `/jxc/return/${id}/post`, method: 'post' })
export const cancelReturn = (id) => service({ url: `/jxc/return/${id}/cancel`, method: 'post' })
export const getReturnable = (saleId) => service({ url: '/jxc/return/returnable', method: 'get', params: { saleId } })

// ---------- 盘点 ----------
export const createCheck = (data) => service({ url: '/jxc/check', method: 'post', data })
export const pageCheck = (params) => service({ url: '/jxc/check/page', method: 'get', params })
export const getCheck = (id) => service({ url: `/jxc/check/${id}`, method: 'get' })
export const updateCheckItems = (id, items) => service({ url: `/jxc/check/${id}/items`, method: 'post', data: { items } })
export const postCheck = (id) => service({ url: `/jxc/check/${id}/post`, method: 'post' })
export const cancelCheck = (id) => service({ url: `/jxc/check/${id}/cancel`, method: 'post' })

// ---------- 月结与报表 ----------
export const listPeriods = () => service({ url: '/jxc/period', method: 'get' })
export const closePeriod = (yearMonth) => service({ url: '/jxc/period/close', method: 'post', data: { yearMonth } })
export const getSnapshot = (periodId) => service({ url: '/jxc/period/snapshot', method: 'get', params: { periodId } })
export const getMonthlySummary = (month) => service({ url: '/jxc/period/summary', method: 'get', params: { month } })

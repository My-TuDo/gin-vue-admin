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

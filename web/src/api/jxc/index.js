import service from '@/utils/request'

// 进销存接口封装。约定：所有方法返回 { code, data, msg }，code===0 为成功。

// ---------- 基础资料 ----------
// params.active = 1 时只返回启用项：开单/补货只应看到能用的主数据；
// 基础资料页不传，才能看到并重新启用已停用的项。
export const listWarehouses = (params = {}) => service({ url: '/jxc/master/warehouses', method: 'get', params })
export const createWarehouse = (data) => service({ url: '/jxc/master/warehouses', method: 'post', data })
export const updateWarehouse = (id, data) => service({ url: `/jxc/master/warehouses/${id}`, method: 'put', data })
export const setWarehouseStatus = (id, status) => service({ url: `/jxc/master/warehouses/${id}/status`, method: 'post', data: { status } })

export const listGoods = (params = {}) => service({ url: '/jxc/master/goods', method: 'get', params })
export const createGoods = (data) => service({ url: '/jxc/master/goods', method: 'post', data })
export const updateGoods = (id, data) => service({ url: `/jxc/master/goods/${id}`, method: 'put', data })
export const setGoodsStatus = (id, status) => service({ url: `/jxc/master/goods/${id}/status`, method: 'post', data: { status } })

export const listSkus = (params = {}) => service({ url: '/jxc/master/skus', method: 'get', params })
export const setSkuStatus = (id, status) => service({ url: `/jxc/master/skus/${id}/status`, method: 'post', data: { status } })

export const listPartners = (kind, params = {}) => service({
  url: '/jxc/master/partners',
  method: 'get',
  params: { kind, ...params }
})
export const createPartner = (data) => service({ url: '/jxc/master/partners', method: 'post', data })
export const updatePartner = (id, data) => service({ url: `/jxc/master/partners/${id}`, method: 'put', data })
export const setPartnerStatus = (id, status) => service({ url: `/jxc/master/partners/${id}/status`, method: 'post', data: { status } })

// ---------- 采购 ----------
export const createPurchase = (data) => service({ url: '/jxc/purchase', method: 'post', data })
export const pagePurchase = (params) => service({ url: '/jxc/purchase/page', method: 'get', params })
export const getPurchase = (id) => service({ url: `/jxc/purchase/${id}`, method: 'get' })
export const approvePurchase = (id) => service({ url: `/jxc/purchase/${id}/approve`, method: 'post' })
export const receivePurchase = (id) => service({ url: `/jxc/purchase/${id}/receive`, method: 'post' })
export const cancelPurchase = (id) => service({ url: `/jxc/purchase/${id}/cancel`, method: 'post' })

// ---------- 销售 ----------
export const createSale = (data) => service({ url: '/jxc/sale', method: 'post', data })
// 原子开单：建单 + 预留 + 出库在一个事务里完成；失败不留草稿/预留。
export const checkoutSale = (data) => service({ url: '/jxc/sale/checkout', method: 'post', data })
export const pageSale = (params) => service({ url: '/jxc/sale/page', method: 'get', params })
export const getSale = (id) => service({ url: `/jxc/sale/${id}`, method: 'get' })
export const shipSale = (id) => service({ url: `/jxc/sale/${id}/ship`, method: 'post' })
export const cancelSale = (id) => service({ url: `/jxc/sale/${id}/cancel`, method: 'post' })

// ---------- 工作台 ----------
// 一次聚合返回 KPI/待办/低库存；替换工作台原先的 5 个 pageSize=100 列表。
export const getDashboardSummary = (params = {}) => service({ url: '/jxc/dashboard/summary', method: 'get', params })

// ---------- 账本 ----------
export const ledgerBalances = (params) => service({ url: '/jxc/ledger/balance', method: 'get', params })
export const ledgerMoves = (params) => service({ url: '/jxc/ledger/moves', method: 'get', params })
export const ledgerReconcile = (params) => service({ url: '/jxc/ledger/reconcile', method: 'get', params })
// 只读最近一次对账结果，不触发全仓扫描。
export const ledgerReconcileLast = (params = {}) => service({ url: '/jxc/ledger/reconcile/last', method: 'get', params })

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

// ---------- 采购退货 ----------
// 与销售退货不同：原单是采购单，退货金额按原采购单价精确出库；
// 建单为草稿，过账（/:id/post）才产生负向账本行。
export const createPurchaseReturn = (data) => service({ url: '/jxc/purchase-return', method: 'post', data })
export const pagePurchaseReturn = (params) => service({ url: '/jxc/purchase-return/page', method: 'get', params })
export const getPurchaseReturn = (id) => service({ url: `/jxc/purchase-return/${id}`, method: 'get' })
export const getPurchaseReturnable = (purchaseId) => service({ url: '/jxc/purchase-return/returnable', method: 'get', params: { purchaseId } })
export const postPurchaseReturn = (id) => service({ url: `/jxc/purchase-return/${id}/post`, method: 'post' })
export const cancelPurchaseReturn = (id) => service({ url: `/jxc/purchase-return/${id}/cancel`, method: 'post' })

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

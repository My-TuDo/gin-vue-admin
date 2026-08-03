import request from '@/utils/request'

// ========== 商品 SPU ==========
export const getGoodsPage = (params) =>
  request({ url: '/jxc/goods/page', method: 'get', params })

export const getGoodsDetail = (params) =>
  request({ url: '/jxc/goods/detail', method: 'get', params })

export const getAllGoods = () =>
  request({ url: '/jxc/goods/all', method: 'get' })

export const createGoods = (data) =>
  request({ url: '/jxc/goods', method: 'post', data })

export const updateGoods = (data) =>
  request({ url: '/jxc/goods', method: 'put', data })

export const deleteGoods = (data) =>
  request({ url: '/jxc/goods', method: 'delete', data })

export const setGoodsStatus = (data) =>
  request({ url: '/jxc/goods/status', method: 'put', data })

// ========== 商品 SKU ==========
export const getSkuList = (params) =>
  request({ url: '/jxc/goods/sku/list', method: 'get', params })

export const createSku = (data) =>
  request({ url: '/jxc/goods/sku', method: 'post', data })

export const updateSku = (data) =>
  request({ url: '/jxc/goods/sku', method: 'put', data })

export const deleteSku = (data) =>
  request({ url: '/jxc/goods/sku', method: 'delete', data })

export const setSkuStatus = (data) =>
  request({ url: '/jxc/goods/sku/status', method: 'put', data })
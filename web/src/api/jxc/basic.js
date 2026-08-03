import request from '@/utils/request'

// ========== 商品分类 ==========
export const getCategoryTree = () =>
    request({ url: '/jxc/category/tree', method: 'get' })

export const getCategoryList = (params) =>
  request({ url: '/jxc/category/list', method: 'get', params })

export const createCategory = (data) =>
  request({ url: '/jxc/category', method: 'post', data })

export const updateCategory = (data) =>
  request({ url: '/jxc/category', method: 'put', data })

export const deleteCategory = (data) =>
  request({ url: '/jxc/category', method: 'delete', data })

export const deleteCategoryForever = (data) =>
  request({ url: '/jxc/category/forever', method: 'delete', data })

export const setCategoryStatus = (data) =>
  request({ url: '/jxc/category/status', method: 'put', data })

// ========== 品牌 ==========
export const getBrandList = (params) =>
  request({ url: '/jxc/brand/list', method: 'get', params })

export const getAllBrands = () =>
  request({ url: '/jxc/brand/all', method: 'get' })

export const createBrand = (data) =>
  request({ url: '/jxc/brand', method: 'post', data })

export const updateBrand = (data) =>
  request({ url: '/jxc/brand', method: 'put', data })

export const deleteBrand = (data) =>
  request({ url: '/jxc/brand', method: 'delete', data })

export const deleteBrandForever = (data) =>
  request({ url: '/jxc/brand/forever', method: 'delete', data })

export const setBrandStatus = (data) =>
  request({ url: '/jxc/brand/status', method: 'put', data })

// ========== 供应商 ==========
export const getSupplierList = (params) =>
  request({ url: '/jxc/supplier/list', method: 'get', params })

export const getAllSuppliers = () =>
  request({ url: '/jxc/supplier/all', method: 'get' })

export const createSupplier = (data) =>
  request({ url: '/jxc/supplier', method: 'post', data })

export const updateSupplier = (data) =>
  request({ url: '/jxc/supplier', method: 'put', data })

export const deleteSupplier = (data) =>
  request({ url: '/jxc/supplier', method: 'delete', data })

export const deleteSupplierForever = (data) =>
  request({ url: '/jxc/supplier/forever', method: 'delete', data })

export const setSupplierStatus = (data) =>
  request({ url: '/jxc/supplier/status', method: 'put', data })

// ========== 客户 ==========
export const getCustomerList = (params) =>
  request({ url: '/jxc/customer/list', method: 'get', params })

export const getAllCustomers = () =>
  request({ url: '/jxc/customer/all', method: 'get' })

export const createCustomer = (data) =>
  request({ url: '/jxc/customer', method: 'post', data })

export const updateCustomer = (data) =>
  request({ url: '/jxc/customer', method: 'put', data })

export const deleteCustomer = (data) =>
  request({ url: '/jxc/customer', method: 'delete', data })

export const deleteCustomerForever = (data) =>
  request({ url: '/jxc/customer/forever', method: 'delete', data })

export const setCustomerStatus = (data) =>
  request({ url: '/jxc/customer/status', method: 'put', data })

// ========== 仓库 ==========
export const getWarehouseList = (params) =>
  request({ url: '/jxc/warehouse/list', method: 'get', params })

export const getAllWarehouses = () =>
  request({ url: '/jxc/warehouse/all', method: 'get' })

export const createWarehouse = (data) =>
  request({ url: '/jxc/warehouse', method: 'post', data })

export const updateWarehouse = (data) =>
  request({ url: '/jxc/warehouse', method: 'put', data })

export const deleteWarehouse = (data) =>
  request({ url: '/jxc/warehouse', method: 'delete', data })

export const deleteWarehouseForever = (data) =>
  request({ url: '/jxc/warehouse/forever', method: 'delete', data })

export const setWarehouseStatus = (data) =>
  request({ url: '/jxc/warehouse/status', method: 'put', data })
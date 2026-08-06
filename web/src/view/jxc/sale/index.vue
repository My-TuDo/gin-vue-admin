<template>
  <div class="jxc-page">
    <!-- 搜索栏 -->
    <div class="crud-search">
      <el-input
        v-model="search.keyword"
        placeholder="销售单号搜索"
        clearable
        style="width: 230px"
        @keyup.enter="fetchData"
        @clear="fetchData"
      />
      <div class="search-actions">
        <el-button type="primary" :icon="Search" @click="fetchData">查询</el-button>
        <el-button type="success" class="btn-create" :icon="Plus" @click="openCreate">新建销售单</el-button>
      </div>
    </div>

    <!-- 列表 -->
    <div class="crud-table">
      <el-table :data="list" v-loading="loading" border stripe class="pos-table">
        <el-table-column label="销售单号" prop="orderNo" width="180" fixed="left" />
        <el-table-column label="类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="typeTag(row.orderType).type" size="small" effect="light">{{ typeTag(row.orderType).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="客户" min-width="120">
          <template #default="{ row }">{{ row.customer?.name || '散客' }}</template>
        </el-table-column>
        <el-table-column label="出库仓库" min-width="110">
          <template #default="{ row }">{{ row.warehouse?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="总金额" width="120" align="right">
          <template #default="{ row }"><span class="col-amount">¥ {{ row.totalAmount?.toFixed(2) }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status, row.orderType).type">{{ statusTag(row.status, row.orderType).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建人" prop="creator" width="90" />
        <el-table-column label="创建时间" prop="createdAt" width="170" />
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <div class="op-group">
              <el-button link type="primary" @click="openDetail(row.ID)">查看</el-button>
              <template v-if="row.status === 1">
                <span class="op-sep"></span>
                <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
                <el-popconfirm :title="`确认${actionText(row)}?`" @confirm="handleConfirm(row)">
                  <template #reference><el-button link type="success" class="op-action">{{ actionText(row) }}</el-button></template>
                </el-popconfirm>
                <el-popconfirm title="确认取消该销售单? 已锁定库存将释放" @confirm="handleCancel(row.ID)">
                  <template #reference><el-button link type="info">取消</el-button></template>
                </el-popconfirm>
                <el-popconfirm title="确认删除该销售单?" @confirm="handleDelete(row.ID)">
                  <template #reference><el-button link type="danger">删除</el-button></template>
                </el-popconfirm>
              </template>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="page-bar"
        background
        layout="total, sizes, prev, pager, next"
        :total="total"
        v-model:current-page="search.page"
        v-model:page-size="search.pageSize"
        :page-sizes="[10, 20, 50]"
        @size-change="fetchData"
        @current-change="fetchData"
      />
    </div>

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="editVisible" :title="form.ID ? '编辑销售单' : '新建销售单'" width="880px" destroy-on-close class="crud-dialog">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="84px">
        <div class="form-grid">
          <el-form-item label="单据类型" prop="orderType">
            <el-select v-model="form.orderType" style="width: 100%" @change="onTypeChange">
              <el-option label="正常销售" :value="1" />
              <el-option label="退货退款" :value="2" />
              <el-option label="换货" :value="3" />
            </el-select>
          </el-form-item>
          <el-form-item label="客户">
            <el-select v-model="form.customerId" placeholder="散客可不选" style="width: 100%" filterable clearable>
              <el-option v-for="cu in customers" :key="cu.ID" :label="cu.name" :value="cu.ID" />
            </el-select>
          </el-form-item>
          <el-form-item label="出库仓库" prop="warehouseId">
            <el-select v-model="form.warehouseId" placeholder="选择仓库" style="width: 100%">
              <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.orderType !== 1" label="关联原单" prop="originalOrderId">
            <el-select v-model="form.originalOrderId" placeholder="选择已出库的原销售单" style="width: 100%" filterable @change="onOriginalChange">
              <el-option v-for="o in shippedOrders" :key="o.ID" :label="`${o.orderNo}（${o.orderType === 3 ? '换货' : '销售'}·${o.customer?.name || '散客'}）`" :value="o.ID" />
            </el-select>
          </el-form-item>
          <el-form-item label="备注" class="span-2">
            <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="备注（可选）" />
          </el-form-item>
        </div>

        <!-- 明细区 -->
        <div class="items-block">
          <!-- 换货：出库/入库双子列表 -->
          <template v-if="form.orderType === 3">
            <div class="sub-block out">
              <div class="sub-title">换出明细<i>（卖出，金额为正）</i></div>
              <el-table :data="outItems" border size="small" max-height="200">
                <el-table-column label="SKU" min-width="200">
                  <template #default="{ row }">
                    <el-select v-model="row.skuId" placeholder="选择SKU" filterable style="width: 100%" @change="onSkuChange(row)">
                      <el-option v-for="s in rowSkuOptions(row)" :key="s.ID" :label="s.skuCode" :value="s.ID" />
                    </el-select>
                  </template>
                </el-table-column>
                <el-table-column label="数量" width="110">
                  <template #default="{ row }"><el-input-number v-model="row.qty" :min="1" :max="rowMax(row)" :disabled="rowDisabled(row)" style="width: 100%" /></template>
                </el-table-column>
                <el-table-column label="金额" width="110" align="right">
                  <template #default="{ row }"><span class="col-amount">¥ {{ ((row.qty || 0) * (row.price || 0)).toFixed(2) }}</span></template>
                </el-table-column>
                <el-table-column label="操作" width="60" align="center">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="form.items.splice(outIndex($index), 1)">删</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <el-button type="primary" link :icon="Plus" class="add-btn" @click="addOutItem">添加换出明细</el-button>
            </div>
            <div class="sub-block in">
              <div class="sub-title">换入明细<i>（收回，金额为负）</i></div>
              <el-table :data="inItems" border size="small" max-height="200">
                <el-table-column label="SKU" min-width="200">
                  <template #default="{ row }">
                    <el-select v-model="row.skuId" placeholder="选择SKU" filterable style="width: 100%" @change="onSkuChange(row)">
                      <el-option v-for="s in rowSkuOptions(row)" :key="s.ID" :label="s.skuCode" :value="s.ID" />
                    </el-select>
                  </template>
                </el-table-column>
                <el-table-column label="数量" width="110">
                  <template #default="{ row }"><el-input-number v-model="row.qty" :min="1" :max="rowMax(row)" :disabled="rowDisabled(row)" style="width: 100%" /></template>
                </el-table-column>
                <el-table-column label="金额" width="110" align="right">
                  <template #default="{ row }"><span class="col-amount neg">-¥ {{ ((row.qty || 0) * (row.price || 0)).toFixed(2) }}</span></template>
                </el-table-column>
                <el-table-column label="操作" width="60" align="center">
                  <template #default="{ $index }">
                    <el-button link type="danger" @click="form.items.splice(inIndex($index), 1)">删</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <el-button type="primary" link :icon="Plus" class="add-btn" @click="addInItem">添加换入明细</el-button>
            </div>
            <div class="diff-row">
              <span>差额（出 − 入，多退少补）</span>
              <b :style="{ color: totalAmount >= 0 ? '#67c23a' : '#f56c6c' }">¥ {{ totalAmount.toFixed(2) }}</b>
            </div>
          </template>
          <!-- 正常销售/退货：单一列表 -->
          <template v-else>
            <div class="items-head">
              <span class="items-title">销售明细</span>
              <span class="items-total">合计：<b class="col-amount">¥ {{ totalAmount.toFixed(2) }}</b></span>
            </div>
            <el-table :data="form.items" border size="small" max-height="260">
              <el-table-column label="SKU" min-width="200">
                <template #default="{ row }">
                  <el-select v-model="row.skuId" placeholder="选择SKU" filterable style="width: 100%" @change="onSkuChange(row)">
                    <el-option v-for="s in rowSkuOptions(row)" :key="s.ID" :label="s.skuCode" :value="s.ID" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="数量" width="110">
                <template #default="{ row }"><el-input-number v-model="row.qty" :min="1" :max="rowMax(row)" :disabled="rowDisabled(row)" style="width: 100%" /></template>
              </el-table-column>
              <el-table-column label="单价" width="110" align="right">
                <template #default="{ row }">
                  <span v-if="row.price" class="price">¥ {{ row.price.toFixed(2) }}</span>
                  <span v-else class="text-muted">选SKU后带出</span>
                </template>
              </el-table-column>
              <el-table-column label="金额" width="110" align="right">
                <template #default="{ row }"><span class="col-amount">¥ {{ ((row.qty || 0) * (row.price || 0)).toFixed(2) }}</span></template>
              </el-table-column>
              <el-table-column label="操作" width="60" align="center">
                <template #default="{ $index }">
                  <el-button link type="danger" @click="form.items.splice($index, 1)">删</el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-button type="primary" link :icon="Plus" class="add-btn" @click="addItem">添加明细</el-button>
          </template>
        </div>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="large" @click="editVisible = false">取消</el-button>
          <el-button type="primary" size="large" class="btn-save" :loading="saving" @click="handleSave">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 查看弹窗 -->
    <el-dialog v-model="detailVisible" title="销售单详情" width="840px" class="crud-dialog">
      <template v-if="detail.ID">
        <el-descriptions :column="2" border size="small" class="detail-desc">
          <el-descriptions-item label="销售单号">{{ detail.orderNo }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusTag(detail.status, detail.orderType).type">{{ statusTag(detail.status, detail.orderType).text }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="类型">
            <el-tag :type="typeTag(detail.orderType).type" size="small">{{ typeTag(detail.orderType).text }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="客户">{{ detail.customer?.name || '散客' }}</el-descriptions-item>
          <el-descriptions-item label="出库仓库">{{ detail.warehouse?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="总金额"><span class="col-amount">¥ {{ detail.totalAmount?.toFixed(2) }}</span></el-descriptions-item>
          <el-descriptions-item label="创建人">{{ detail.creator }}</el-descriptions-item>
          <el-descriptions-item label="备注" :span="2">{{ detail.remark || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="detail.items || []" border size="small" class="detail-table">
          <el-table-column label="SKU编码" prop="skuCode" min-width="140" />
          <el-table-column label="商品名称" prop="goodsName" min-width="130" />
          <el-table-column label="颜色/尺码" min-width="100">
            <template #default="{ row }">{{ row.color }}{{ row.size ? '/' + row.size : '' }}</template>
          </el-table-column>
          <el-table-column label="数量" prop="qty" width="80" align="center" />
          <el-table-column label="单价" width="100" align="right">
            <template #default="{ row }">¥ {{ row.price?.toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="金额" width="110" align="right">
            <template #default="{ row }"><span class="col-amount">¥ {{ row.amount?.toFixed(2) }}</span></template>
          </el-table-column>
        </el-table>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'
import { getSalePage, getSaleDetail, getSaleRemaining, createSaleOrder, updateSaleOrder, deleteSaleOrder, cancelSaleOrder, confirmOut, confirmReturn, confirmExchange } from '@/api/jxc/sale'
import { getCustomerList, getWarehouseList } from '@/api/jxc/basic'
import { getSkuList } from '@/api/jxc/goods'
import { getStockPage } from '@/api/jxc/stock'

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const total = ref(0)
const search = reactive({ page: 1, pageSize: 10, keyword: '' })

const customers = ref([])
const warehouses = ref([])
const skuOptions = ref([])

const editVisible = ref(false)
const detailVisible = ref(false)
const detail = ref({})
const formRef = ref()
const form = reactive({ ID: 0, orderType: 1, customerId: undefined, warehouseId: undefined, originalOrderId: undefined, remark: '', items: [] })

const rules = {
  orderType: [{ required: true, message: '请选择单据类型', trigger: 'change' }],
  warehouseId: [{ required: true, message: '请选择出库仓库', trigger: 'change' }],
  originalOrderId: [{ required: true, message: '请选择关联原单', trigger: 'change' }],
}

const typeTag = (t) => ({ 1: { type: 'primary', text: '正常销售' }, 2: { type: 'warning', text: '退货退款' }, 3: { type: 'danger', text: '换货' } }[t] || { type: 'info', text: t })

// 状态文案按单据类型区分（退货/换货不叫"出库"）
const statusTag = (s, t = 1) => {
  const map = {
    1: { 1: { type: 'info', text: '待出库' }, 2: { type: 'success', text: '已出库' }, 3: { type: 'danger', text: '已取消' } },
    2: { 1: { type: 'info', text: '待退货' }, 2: { type: 'success', text: '已退货' }, 3: { type: 'danger', text: '已取消' } },
    3: { 1: { type: 'info', text: '待确认' }, 2: { type: 'success', text: '已完成' }, 3: { type: 'danger', text: '已取消' } },
  }
  return (map[t] || map[1])[s] || { type: 'info', text: s }
}

// 待出库单据的确认动作文案
const actionText = (row) => ({ 1: '出库', 2: '退货入库', 3: '换货确认' }[row.orderType] || '确认')

// 合计金额：换入（direction=2）与退货单为负，换出/正常销售为正
const totalAmount = computed(() =>
  form.items.reduce((sum, it) => {
    let amt = (it.qty || 0) * (it.price || 0)
    if (it.direction === 2 || form.orderType === 2) amt = -amt
    return sum + amt
  }, 0)
)

const fetchData = async () => {
  loading.value = true
  try {
    const { data } = await getSalePage(search)
    list.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

const loadOptions = async () => {
  const [cu, w, sku] = await Promise.all([
    getCustomerList({ page: 1, pageSize: 999 }),
    getWarehouseList({ page: 1, pageSize: 999 }),
    getSkuList({ page: 1, pageSize: 999 }),
  ])
  customers.value = cu.data.list || []
  warehouses.value = w.data.list || []
  skuOptions.value = (sku.data || []).filter((s) => s.status === 1)
  await loadStock()
}

// SKU 可售库存映射（数量上限：销售/换出）
const skuStockMap = ref({})
const loadStock = async () => {
  const { data } = await getStockPage({ page: 1, pageSize: 999 })
  const map = {}
  ;(data.list || []).forEach((s) => {
    map[s.skuId] = (s.quantity || 0) - (s.lockQuantity || 0)
  })
  skuStockMap.value = map
}

// 原单剩余可退换件数（数量上限：退货/换入，按单据总量）
const remainingTotal = ref(undefined)
// 原单出库过的 SKU（退货单只能退原单商品，业界退货限定原单商品）
const originalSkus = ref([])
const onOriginalChange = async (id) => {
  remainingTotal.value = undefined
  originalSkus.value = []
  if (!id) return
  const [rem, det] = await Promise.all([getSaleRemaining(id), getSaleDetail(id)])
  remainingTotal.value = rem.data?.remaining
  const detData = det.data || {}
  originalSkus.value = (detData.items || [])
    .filter((it) => detData.orderType === 1 || it.direction === 1)
    .map((it) => it.skuId)
}

// SKU 下拉：退货单只显示原单出过的商品；换货/正常销售显示全部
const rowSkuOptions = (row) => {
  if (form.orderType === 2) {
    return skuOptions.value.filter((s) => originalSkus.value.includes(s.ID))
  }
  return skuOptions.value
}

// 行数量上限：退货/换入=原单剩余件数；销售/换出=可售库存（键盘超限自动钳制、+ 按钮达上限禁用）
const rowMax = (row) => {
  if (form.orderType === 2 || (form.orderType === 3 && row.direction === 2)) {
    const rem = remainingTotal.value
    return rem !== undefined && rem > 0 ? rem : undefined
  }
  if (row.skuId) {
    const avail = skuStockMap.value[row.skuId]
    return avail !== undefined && avail > 0 ? avail : undefined
  }
  return undefined
}
// 剩余件数/可售库存为 0 时禁用数量输入
const rowDisabled = (row) => {
  if (form.orderType === 2 || (form.orderType === 3 && row.direction === 2)) {
    const rem = remainingTotal.value
    return rem !== undefined && rem <= 0
  }
  if (row.skuId) {
    const avail = skuStockMap.value[row.skuId]
    return avail !== undefined && avail <= 0
  }
  return false
}

// 已出库的正常销售/换货单（退货/换货的原单候选；换出的商品可继续退/换）
const normalShipped = ref([])
const exchangeShipped = ref([])
const shippedOrders = computed(() => (form.orderType === 1 ? [] : [...normalShipped.value, ...exchangeShipped.value]))

const loadShippedOrders = async () => {
  const { data } = await getSalePage({ page: 1, pageSize: 999 })
  const all = data.list || []
  normalShipped.value = all.filter((o) => o.orderType === 1 && o.status === 2)
  exchangeShipped.value = all.filter((o) => o.orderType === 3 && o.status === 2)
}

const addItem = () => form.items.push({ skuId: undefined, qty: 1, price: 0, direction: 0 })

// 换货：出库/入库两个子列表
const outItems = computed(() => form.items.filter((it) => it.direction === 1))
const inItems = computed(() => form.items.filter((it) => it.direction === 2))
const addOutItem = () => form.items.push({ skuId: undefined, qty: 1, price: 0, direction: 1 })
const addInItem = () => form.items.push({ skuId: undefined, qty: 1, price: 0, direction: 2 })

// 子列表索引 → 原数组索引（用于删除）
const outIndex = (i) => {
  let count = 0
  for (let idx = 0; idx < form.items.length; idx++) {
    if (form.items[idx].direction === 1) {
      if (count === i) return idx
      count++
    }
  }
  return -1
}
const inIndex = (i) => {
  let count = 0
  for (let idx = 0; idx < form.items.length; idx++) {
    if (form.items[idx].direction === 2) {
      if (count === i) return idx
      count++
    }
  }
  return -1
}

// 选中 SKU 后带出其销售价作为单价（单价以 SKU 销售价为准，不可修改）
const onSkuChange = (row) => {
  const sku = skuOptions.value.find((s) => s.ID === row.skuId)
  if (sku) row.price = sku.salePrice || 0
}

const openCreate = () => {
  Object.assign(form, { ID: 0, orderType: 1, customerId: undefined, warehouseId: undefined, originalOrderId: undefined, remark: '', items: [] })
  addItem()
  // 每次打开都刷新：原单候选（刚完成的销售单立即可关联）+ 可售库存（数量上限）
  loadShippedOrders()
  loadStock()
  editVisible.value = true
}

// 单据类型切换时规整明细：换货=保留方向行（默认给一行换出+一行换入）；其他=仅保留无方向行
const onTypeChange = (t) => {
  if (t === 3) {
    form.items = form.items.filter((it) => it.direction !== 0)
    if (!form.items.length) {
      addOutItem()
      addInItem()
    }
  } else {
    form.items = form.items.filter((it) => it.direction === 0)
    if (!form.items.length) addItem()
  }
}

const openEdit = (row) => {
  Object.assign(form, { ID: row.ID, orderType: row.orderType, customerId: row.customerId, warehouseId: row.warehouseId, originalOrderId: row.originalOrderId, remark: row.remark, items: [] })
  if (row.originalOrderId) onOriginalChange(row.originalOrderId)
  getSaleDetail(row.ID).then(({ data }) => {
    form.items = (data.items || []).map((it) => ({ skuId: it.skuId, qty: it.qty, price: it.price, direction: it.direction || 0 }))
  })
  editVisible.value = true
}

const openDetail = async (id) => {
  const { data } = await getSaleDetail(id)
  detail.value = data
  detailVisible.value = true
}

const handleSave = async () => {
  await formRef.value.validate()
  if (!form.items.length) {
    ElMessage.warning('请至少添加一条明细')
    return
  }
  if (form.items.some((it) => !it.skuId)) {
    ElMessage.warning('请完整选择每行的 SKU')
    return
  }
  // 换货单必须同时包含换出与换入明细（防御：剔除可能残留的无方向行）
  if (form.orderType === 3) {
    form.items = form.items.filter((it) => it.direction !== 0)
    if (!outItems.value.length || !inItems.value.length) {
      ElMessage.warning('换货单必须同时包含换出和换入明细')
      return
    }
  }
  saving.value = true
  try {
    const res = form.ID ? await updateSaleOrder(form) : await createSaleOrder(form)
    if (res && res.code !== 0) return
    ElMessage.success(form.ID ? '更新成功' : '创建成功')
    editVisible.value = false
    await fetchData()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (id) => {
  const res = await deleteSaleOrder(id)
  if (res && res.code !== 0) return
  ElMessage.success('删除成功')
  await fetchData()
}

const handleCancel = async (id) => {
  const res = await cancelSaleOrder(id)
  if (res && res.code !== 0) return
  ElMessage.success('已取消，锁定库存已释放')
  await fetchData()
}

// 出库 / 退货入库 / 换货确认（按单据类型分发）
const handleConfirm = async (row) => {
  const res = { 1: confirmOut, 2: confirmReturn, 3: confirmExchange }[row.orderType](row.ID)
  const r = await res
  if (r && r.code !== 0) return
  ElMessage.success(actionText(row) + '成功')
  await fetchData()
}

onMounted(() => {
  fetchData()
  loadOptions()
  loadShippedOrders()
})
</script>

<style scoped>
/* ===== POS 设计语言 · 公共 ===== */
.jxc-page { padding: 4px 0; }
.crud-search {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  background: #fff;
  border-radius: 12px;
  padding: 12px 16px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
  margin-bottom: 12px;
}
.search-actions { display: flex; gap: 10px; }
.btn-create { height: 38px; font-weight: 600; }
.crud-table {
  background: #fff;
  border-radius: 12px;
  padding: 6px 14px 14px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
}
.pos-table { width: 100%; }
.pos-table :deep(th.el-table__cell) { background: #f7f9fc; font-weight: 700; color: #303133; }
.pos-table :deep(.el-table__cell) { padding: 9px 0; }
.col-amount { font-weight: 700; color: #e6a23c; font-variant-numeric: tabular-nums; }
.col-amount.neg { color: #f56c6c; }
.price { font-weight: 600; color: #606266; }
.text-muted { color: #c0c4cc; font-size: 12px; }
.op-group { display: flex; align-items: center; flex-wrap: nowrap; }
.op-sep { width: 1px; height: 14px; background: #e4e7ed; margin: 0 6px; flex-shrink: 0; }
.op-action { font-weight: 600; }
.page-bar { margin-top: 14px; justify-content: flex-end; }

/* ===== 弹窗 ===== */
.crud-dialog :deep(.el-dialog__header) { padding-bottom: 8px; }
.crud-dialog :deep(.el-dialog__title) { font-weight: 700; }
.crud-dialog :deep(.el-dialog__body) { padding-top: 12px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; column-gap: 18px; row-gap: 2px; }
.form-grid .el-form-item { margin-bottom: 14px; }
.form-grid .span-2 { grid-column: span 2; }
.items-block {
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 12px 12px 6px;
  margin-top: 2px;
}
.items-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.items-title { font-weight: 700; font-size: 14px; color: #303133; }
.items-total { font-size: 13px; color: #606266; }
.sub-block { border-radius: 8px; padding: 8px 10px 4px; margin-bottom: 10px; }
.sub-block.out { background: #f0f9eb; border: 1px solid #e1f3d8; }
.sub-block.in { background: #fef0f0; border: 1px solid #fde2e2; }
.sub-title { font-weight: 600; font-size: 13px; margin-bottom: 6px; color: #303133; }
.sub-title i { font-style: normal; color: #909399; font-size: 12px; font-weight: 400; margin-left: 4px; }
.add-btn { margin-top: 6px; }
.diff-row {
  display: flex;
  justify-content: flex-end;
  align-items: baseline;
  gap: 8px;
  font-size: 13px;
  color: #606266;
  padding: 10px 4px 6px;
  border-top: 1px dashed #ebeef5;
}
.diff-row b { font-size: 18px; font-weight: 700; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 10px; }
.btn-save { min-width: 120px; font-weight: 600; }
.detail-desc { margin-bottom: 14px; }
.detail-table :deep(.el-table__cell) { padding: 8px 0; }
</style>

<template>
  <div>
    <!-- 搜索栏 -->
    <el-card shadow="never" class="mb-4">
      <el-form :inline="true">
        <el-form-item label="销售单号">
          <el-input v-model="search.keyword" placeholder="输入单号搜索" clearable style="width: 200px" @keyup.enter="fetchData" @clear="fetchData" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="Search" @click="fetchData">查询</el-button>
          <el-button type="success" icon="Plus" @click="openCreate">新建销售单</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 列表 -->
    <el-card shadow="never">
      <el-table :data="list" v-loading="loading" border stripe>
        <el-table-column label="销售单号" prop="orderNo" width="180" fixed="left" />
        <el-table-column label="类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="typeTag(row.orderType).type" size="small">{{ typeTag(row.orderType).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="客户" min-width="120">
          <template #default="{ row }">{{ row.customer?.name || '散客' }}</template>
        </el-table-column>
        <el-table-column label="出库仓库" min-width="110">
          <template #default="{ row }">{{ row.warehouse?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="总金额" width="110" align="right">
          <template #default="{ row }">¥ {{ row.totalAmount?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status).type">{{ statusTag(row.status).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建人" prop="creator" width="90" />
        <el-table-column label="创建时间" prop="createdAt" width="170" />
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row.ID)">查看</el-button>
            <template v-if="row.status === 1">
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-popconfirm :title="`确认${actionText(row)}?`" @confirm="handleConfirm(row)">
                <template #reference><el-button link type="success">{{ actionText(row) }}</el-button></template>
              </el-popconfirm>
              <el-popconfirm title="确认取消该销售单? 已锁定库存将释放" @confirm="handleCancel(row.ID)">
                <template #reference><el-button link type="info">取消</el-button></template>
              </el-popconfirm>
              <el-popconfirm title="确认删除该销售单?" @confirm="handleDelete(row.ID)">
                <template #reference><el-button link type="danger">删除</el-button></template>
              </el-popconfirm>
            </template>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="mt-3 justify-end"
        background
        layout="total, sizes, prev, pager, next"
        :total="total"
        v-model:current-page="search.page"
        v-model:page-size="search.pageSize"
        :page-sizes="[10, 20, 50]"
        @size-change="fetchData"
        @current-change="fetchData"
      />
    </el-card>

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="editVisible" :title="form.ID ? '编辑销售单' : '新建销售单'" width="880px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="单据类型" prop="orderType">
          <el-select v-model="form.orderType" style="width: 200px">
            <el-option label="正常销售" :value="1" />
            <el-option label="退货退款" :value="2" />
            <el-option label="换货出库" :value="3" />
            <el-option label="换货入库" :value="4" />
          </el-select>
        </el-form-item>
        <el-form-item label="客户">
          <el-select v-model="form.customerId" placeholder="散客可不选" style="width: 240px" filterable clearable>
            <el-option v-for="cu in customers" :key="cu.ID" :label="cu.name" :value="cu.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="出库仓库" prop="warehouseId">
          <el-select v-model="form.warehouseId" placeholder="选择仓库" style="width: 240px">
            <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.orderType !== 1" label="关联原单" prop="originalOrderId">
          <el-select v-model="form.originalOrderId" placeholder="选择已出库的原销售单" style="width: 300px" filterable>
            <el-option v-for="o in shippedOrders" :key="o.ID" :label="`${o.orderNo}（${o.customer?.name || '散客'}）`" :value="o.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="备注（可选）" />
        </el-form-item>

        <!-- 明细 -->
        <el-form-item label="销售明细">
          <div style="width: 100%">
            <el-table :data="form.items" border size="small" max-height="320">
              <el-table-column label="SKU" min-width="200">
                <template #default="{ row }">
                  <el-select v-model="row.skuId" placeholder="选择SKU" filterable style="width: 100%" @change="onSkuChange(row)">
                    <el-option v-for="s in skuOptions" :key="s.ID" :label="s.skuCode" :value="s.ID" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="数量" width="120">
                <template #default="{ row }">
                  <el-input-number v-model="row.qty" :min="1" style="width: 100%" />
                </template>
              </el-table-column>
              <el-table-column label="单价" width="130">
                <template #default="{ row }">
                  <el-input-number v-model="row.price" :min="0" :precision="2" style="width: 100%" />
                </template>
              </el-table-column>
              <el-table-column label="金额" width="110" align="right">
                <template #default="{ row }">¥ {{ ((row.qty || 0) * (row.price || 0)).toFixed(2) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="60" align="center">
                <template #default="{ $index }">
                  <el-button link type="danger" @click="form.items.splice($index, 1)">删</el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-button type="primary" link icon="Plus" class="mt-2" @click="addItem">添加明细</el-button>
            <div class="mt-2 text-right">合计金额：<b style="color: #e6a23c">¥ {{ totalAmount.toFixed(2) }}</b></div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 查看弹窗 -->
    <el-dialog v-model="detailVisible" title="销售单详情" width="820px">
      <template v-if="detail.ID">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="销售单号">{{ detail.orderNo }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusTag(detail.status).type">{{ statusTag(detail.status).text }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="类型">
            <el-tag :type="typeTag(detail.orderType).type" size="small">{{ typeTag(detail.orderType).text }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="客户">{{ detail.customer?.name || '散客' }}</el-descriptions-item>
          <el-descriptions-item label="出库仓库">{{ detail.warehouse?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="总金额">¥ {{ detail.totalAmount?.toFixed(2) }}</el-descriptions-item>
          <el-descriptions-item label="创建人">{{ detail.creator }}</el-descriptions-item>
          <el-descriptions-item label="备注" :span="2">{{ detail.remark || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="detail.items || []" border size="small" class="mt-3">
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
            <template #default="{ row }">¥ {{ row.amount?.toFixed(2) }}</template>
          </el-table-column>
        </el-table>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSalePage, getSaleDetail, createSaleOrder, updateSaleOrder, deleteSaleOrder, cancelSaleOrder, confirmOut, confirmReturn, confirmExchange } from '@/api/jxc/sale'
import { getCustomerList, getWarehouseList } from '@/api/jxc/basic'
import { getSkuList } from '@/api/jxc/goods'

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const total = ref(0)
const search = reactive({ page: 1, pageSize: 10, keyword: '' })

const customers = ref([])
const warehouses = ref([])
const skuOptions = ref([])
const shippedOrders = ref([])

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

const typeTag = (t) => ({ 1: { type: 'primary', text: '正常销售' }, 2: { type: 'warning', text: '退货退款' }, 3: { type: 'danger', text: '换货出库' }, 4: { type: 'success', text: '换货入库' } }[t] || { type: 'info', text: t })
const statusTag = (s) => ({ 1: { type: 'info', text: '待出库' }, 2: { type: 'success', text: '已出库' }, 3: { type: 'danger', text: '已取消' } }[s] || { type: 'info', text: s })

// 待出库单据的确认动作文案
const actionText = (row) => ({ 1: '出库', 2: '退货入库', 3: '换货确认', 4: '换货确认' }[row.orderType] || '确认')

const totalAmount = computed(() => form.items.reduce((sum, it) => sum + (it.qty || 0) * (it.price || 0), 0))

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
}

// 已出库的销售单（退货/换货的原单候选）
const loadShippedOrders = async () => {
  const { data } = await getSalePage({ page: 1, pageSize: 999 })
  shippedOrders.value = (data.list || []).filter((o) => o.status === 2)
}

const addItem = () => form.items.push({ skuId: undefined, qty: 1, price: 0 })

// 选中 SKU 后带出其销售价作为单价（可手动修改）
const onSkuChange = (row) => {
  const sku = skuOptions.value.find((s) => s.ID === row.skuId)
  if (sku) row.price = sku.salePrice || 0
}

const openCreate = () => {
  Object.assign(form, { ID: 0, orderType: 1, customerId: undefined, warehouseId: undefined, originalOrderId: undefined, remark: '', items: [] })
  addItem()
  editVisible.value = true
}

const openEdit = (row) => {
  Object.assign(form, { ID: row.ID, orderType: row.orderType, customerId: row.customerId, warehouseId: row.warehouseId, originalOrderId: row.originalOrderId, remark: row.remark, items: [] })
  getSaleDetail(row.ID).then(({ data }) => {
    form.items = (data.items || []).map((it) => ({ skuId: it.skuId, qty: it.qty, price: it.price }))
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
  const res = { 1: confirmOut, 2: confirmReturn, 3: confirmExchange, 4: confirmExchange }[row.orderType](row.ID)
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
.mb-4 { margin-bottom: 16px; }
.mt-2 { margin-top: 8px; }
.mt-3 { margin-top: 12px; }
.justify-end { justify-content: flex-end; }
.text-right { text-align: right; }
</style>

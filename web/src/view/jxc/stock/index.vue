<template>
  <div class="jxc-page">
    <div class="crud-card">
      <el-tabs v-model="activeTab" class="pos-tabs">
        <!-- 库存列表 -->
        <el-tab-pane label="库存列表" name="stock">
          <div class="tab-toolbar">
            <div class="toolbar-fields">
              <el-select v-model="stockQuery.warehouseId" placeholder="全部仓库" clearable style="width: 160px" @change="fetchStock">
                <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
              </el-select>
              <el-input v-model="stockQuery.keyword" placeholder="SKU编码/商品名" clearable style="width: 190px" @keyup.enter="fetchStock" @clear="fetchStock" />
              <el-checkbox v-model="stockQuery.lowStock" @change="fetchStock">仅看预警</el-checkbox>
            </div>
            <div class="toolbar-actions">
              <el-button type="primary" :icon="Search" @click="fetchStock">查询</el-button>
              <el-button type="success" class="btn-create" :icon="Plus" @click="openDirect('in')">直接入库</el-button>
              <el-button type="warning" :icon="Minus" @click="openDirect('out')">直接出库</el-button>
            </div>
          </div>
          <el-table :data="stockList" v-loading="loading" border stripe class="pos-table">
            <el-table-column label="仓库" min-width="110">
              <template #default="{ row }">{{ warehouseName(row.warehouseId) }}</template>
            </el-table-column>
            <el-table-column label="SKU编码" min-width="150">
              <template #default="{ row }">{{ row.sku?.skuCode || '-' }}</template>
            </el-table-column>
            <el-table-column label="商品" min-width="130">
              <template #default="{ row }">{{ row.sku?.goods?.name || '-' }}</template>
            </el-table-column>
            <el-table-column label="颜色/尺码" min-width="100">
              <template #default="{ row }">{{ row.sku?.color }}{{ row.sku?.size ? '/' + row.sku.size : '' }}</template>
            </el-table-column>
            <el-table-column label="现有库存" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="row.quantity <= row.warningQuantity ? 'danger' : 'success'" effect="plain">{{ row.quantity }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="锁定" prop="lockQuantity" width="80" align="center" />
            <el-table-column label="可售" width="90" align="center">
              <template #default="{ row }">
                <span class="avail" :class="{ zero: row.quantity - row.lockQuantity <= 0 }">{{ row.quantity - row.lockQuantity }}</span>
              </template>
            </el-table-column>
            <el-table-column label="预警值" prop="warningQuantity" width="80" align="center" />
          </el-table>
          <el-pagination
            class="page-bar"
            background
            layout="total, sizes, prev, pager, next"
            :total="stockTotal"
            v-model:current-page="stockQuery.page"
            v-model:page-size="stockQuery.pageSize"
            :page-sizes="[10, 20, 50]"
            @size-change="fetchStock"
            @current-change="fetchStock"
          />
        </el-tab-pane>

        <!-- 库存流水 -->
        <el-tab-pane label="库存流水" name="log">
          <div class="tab-toolbar">
            <div class="toolbar-fields">
              <el-select v-model="logQuery.warehouseId" placeholder="全部仓库" clearable style="width: 160px" @change="fetchLog">
                <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
              </el-select>
              <el-select v-model="logQuery.businessType" placeholder="全部类型" clearable style="width: 150px" @change="fetchLog">
                <el-option v-for="t in bizTypes" :key="t.value" :label="t.label" :value="t.value" />
              </el-select>
            </div>
            <div class="toolbar-actions">
              <el-button type="primary" :icon="Search" @click="fetchLog">查询</el-button>
            </div>
          </div>
          <el-table :data="logList" v-loading="logLoading" border stripe class="pos-table">
            <el-table-column label="时间" prop="createdAt" width="170" />
            <el-table-column label="仓库" width="100">
              <template #default="{ row }">{{ warehouseName(row.warehouseId) }}</template>
            </el-table-column>
            <el-table-column label="SKU" prop="skuId" width="80" align="center" />
            <el-table-column label="业务类型" width="110">
              <template #default="{ row }">
                <el-tag size="small" :type="bizTag(row.businessType).type" effect="light">{{ bizTag(row.businessType).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="业务单号" prop="businessNo" min-width="150" />
            <el-table-column label="变动前" prop="beforeQty" width="80" align="center" />
            <el-table-column label="变动" width="90" align="center">
              <template #default="{ row }">
                <b :style="{ color: row.changeQty >= 0 ? '#67c23a' : '#f56c6c' }">{{ row.changeQty >= 0 ? '+' : '' }}{{ row.changeQty }}</b>
              </template>
            </el-table-column>
            <el-table-column label="变动后" prop="afterQty" width="80" align="center" />
            <el-table-column label="操作人" prop="operator" width="90" />
            <el-table-column label="备注" prop="remark" min-width="120" :show-overflow-tooltip="true" />
          </el-table>
          <el-pagination
            class="page-bar"
            background
            layout="total, sizes, prev, pager, next"
            :total="logTotal"
            v-model:current-page="logQuery.page"
            v-model:page-size="logQuery.pageSize"
            :page-sizes="[10, 20, 50]"
            @size-change="fetchLog"
            @current-change="fetchLog"
          />
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 直接出入库弹窗 -->
    <el-dialog v-model="directVisible" :title="directMode === 'in' ? '直接入库' : '直接出库'" width="480px" destroy-on-close class="crud-dialog">
      <el-form ref="directFormRef" :model="directForm" :rules="directRules" label-width="92px">
        <el-form-item label="仓库" prop="warehouseId">
          <el-select v-model="directForm.warehouseId" placeholder="选择仓库" style="width: 100%">
            <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="SKU" prop="skuId">
          <el-select v-model="directForm.skuId" placeholder="选择SKU" filterable style="width: 100%">
            <el-option v-for="s in skuOptions" :key="s.ID" :label="s.skuCode" :value="s.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量" prop="qty">
          <el-input-number v-model="directForm.qty" :min="1" style="width: 100%" size="large" />
        </el-form-item>
        <el-form-item label="原因/备注">
          <el-input v-model="directForm.remark" type="textarea" :rows="2" :placeholder="directMode === 'in' ? '如：赠品入库、盘盈调整' : '如：损耗出库、盘亏调整'" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="large" @click="directVisible = false">取消</el-button>
          <el-button :type="directMode === 'in' ? 'success' : 'warning'" size="large" class="btn-save" :loading="directSaving" @click="handleDirect">
            确认{{ directMode === 'in' ? '入库' : '出库' }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Plus, Minus } from '@element-plus/icons-vue'
import { getStockPage, getStockLogPage, directIn, directOut } from '@/api/jxc/stock'
import { getWarehouseList } from '@/api/jxc/basic'
import { getSkuList } from '@/api/jxc/goods'

const activeTab = ref('stock')
const loading = ref(false)
const logLoading = ref(false)

const warehouses = ref([])
const skuOptions = ref([])
const warehouseMap = reactive({})

const stockList = ref([])
const stockTotal = ref(0)
const stockQuery = reactive({ page: 1, pageSize: 10, keyword: '', warehouseId: undefined, lowStock: false })

const logList = ref([])
const logTotal = ref(0)
const logQuery = reactive({ page: 1, pageSize: 10, warehouseId: undefined, businessType: '' })

const bizTypes = [
  { value: 'purchase_in', label: '采购入库' },
  { value: 'sale_out', label: '销售出库' },
  { value: 'sale_return', label: '退货入库' },
  { value: 'sale_exchange_out', label: '换货出库' },
  { value: 'sale_exchange_in', label: '换货入库' },
  { value: 'direct_in', label: '直接入库' },
  { value: 'direct_out', label: '直接出库' },
]

const bizTag = (t) => {
  const map = {
    purchase_in: { type: 'success', text: '采购入库' },
    sale_out: { type: 'warning', text: '销售出库' },
    sale_return: { type: 'success', text: '退货入库' },
    sale_exchange_out: { type: 'warning', text: '换货出库' },
    sale_exchange_in: { type: 'success', text: '换货入库' },
    direct_in: { type: 'success', text: '直接入库' },
    direct_out: { type: 'danger', text: '直接出库' },
  }
  return map[t] || { type: 'info', text: t }
}

const warehouseName = (id) => warehouseMap[id] || `仓库${id}`

const directVisible = ref(false)
const directMode = ref('in')
const directSaving = ref(false)
const directFormRef = ref()
const directForm = reactive({ warehouseId: undefined, skuId: undefined, qty: 1, remark: '' })
const directRules = {
  warehouseId: [{ required: true, message: '请选择仓库', trigger: 'change' }],
  skuId: [{ required: true, message: '请选择SKU', trigger: 'change' }],
  qty: [{ required: true, message: '请输入数量', trigger: 'change' }],
}

const fetchStock = async () => {
  loading.value = true
  try {
    const { data } = await getStockPage(stockQuery)
    stockList.value = data.list
    stockTotal.value = data.total
  } finally {
    loading.value = false
  }
}

const fetchLog = async () => {
  logLoading.value = true
  try {
    const { data } = await getStockLogPage(logQuery)
    logList.value = data.list
    logTotal.value = data.total
  } finally {
    logLoading.value = false
  }
}

const loadOptions = async () => {
  const [w, sku] = await Promise.all([
    getWarehouseList({ page: 1, pageSize: 999 }),
    getSkuList({ page: 1, pageSize: 999 }),
  ])
  warehouses.value = w.data.list || []
  skuOptions.value = (sku.data || []).filter((s) => s.status === 1)
  warehouses.value.forEach((x) => (warehouseMap[x.ID] = x.name))
}

const openDirect = (mode) => {
  directMode.value = mode
  Object.assign(directForm, { warehouseId: undefined, skuId: undefined, qty: 1, remark: '' })
  directVisible.value = true
}

const handleDirect = async () => {
  await directFormRef.value.validate()
  directSaving.value = true
  try {
    const res = directMode.value === 'in' ? await directIn(directForm) : await directOut(directForm)
    if (res && res.code !== 0) return
    ElMessage.success(directMode.value === 'in' ? '入库成功' : '出库成功')
    directVisible.value = false
    await fetchStock()
  } finally {
    directSaving.value = false
  }
}

onMounted(() => {
  fetchStock()
  fetchLog()
  loadOptions()
})
</script>

<style scoped>
/* ===== POS 设计语言 · 公共 ===== */
.jxc-page { padding: 4px 0; }
.crud-card {
  background: #fff;
  border-radius: 12px;
  padding: 6px 16px 14px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
}
.pos-tabs :deep(.el-tabs__item) { font-size: 15px; font-weight: 600; }
.pos-tabs :deep(.el-tabs__content) { overflow: visible; }
.tab-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 10px 0 14px;
}
.toolbar-fields { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.toolbar-actions { display: flex; gap: 10px; }
.btn-create { height: 38px; font-weight: 600; }
.pos-table { width: 100%; }
.pos-table :deep(th.el-table__cell) { background: #f7f9fc; font-weight: 700; color: #303133; }
.pos-table :deep(.el-table__cell) { padding: 9px 0; }
.avail { font-weight: 700; color: #67c23a; font-variant-numeric: tabular-nums; }
.avail.zero { color: #f56c6c; }
.page-bar { margin-top: 14px; justify-content: flex-end; }

/* ===== 弹窗 ===== */
.crud-dialog :deep(.el-dialog__header) { padding-bottom: 8px; }
.crud-dialog :deep(.el-dialog__title) { font-weight: 700; }
.crud-dialog :deep(.el-dialog__body) { padding-top: 12px; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 10px; }
.btn-save { min-width: 130px; font-weight: 600; }
</style>

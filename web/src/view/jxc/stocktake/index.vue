<template>
  <div class="jxc-page">
    <!-- 搜索栏 -->
    <div class="crud-search">
      <el-input
        v-model="search.keyword"
        placeholder="盘点单号搜索"
        clearable
        style="width: 230px"
        @keyup.enter="fetchData"
        @clear="fetchData"
      />
      <div class="search-actions">
        <el-button type="primary" :icon="Search" @click="fetchData">查询</el-button>
        <el-button type="success" class="btn-create" :icon="Plus" @click="openCreate">新建盘点单</el-button>
      </div>
    </div>

    <!-- 列表 -->
    <div class="crud-table">
      <el-table :data="list" v-loading="loading" border stripe class="pos-table">
        <el-table-column label="盘点单号" prop="checkNo" width="180" fixed="left" />
        <el-table-column label="仓库" min-width="110">
          <template #default="{ row }">{{ row.warehouse?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status).type" effect="light">{{ statusTag(row.status).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="盘点人" prop="checker" width="100" />
        <el-table-column label="备注" prop="remark" min-width="140" show-overflow-tooltip />
        <el-table-column label="创建时间" prop="createdAt" width="170" />
        <el-table-column label="操作" width="270" fixed="right">
          <template #default="{ row }">
            <div class="op-group">
              <el-button link type="primary" @click="openDetail(row)">详情</el-button>
              <template v-if="row.status === 1">
                <span class="op-sep"></span>
                <el-button link type="warning" class="op-action" @click="openRecord(row)">录入盘点</el-button>
                <el-button link type="success" @click="handleComplete(row)">完成</el-button>
                <el-button link type="info" @click="handleCancel(row)">取消</el-button>
              </template>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="page-bar"
        v-model:current-page="search.page"
        v-model:page-size="search.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="fetchData"
      />
    </div>

    <!-- 新建盘点单 -->
    <el-dialog v-model="createVisible" title="新建盘点单" width="420px" destroy-on-close class="crud-dialog">
      <el-form ref="createRef" :model="createForm" label-width="92px">
        <el-form-item label="盘点仓库" prop="warehouseId" :rules="[{ required: true, message: '请选择盘点仓库', trigger: 'change' }]">
          <el-select v-model="createForm.warehouseId" placeholder="选择仓库" style="width: 100%">
            <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="盘点人">
          <el-input v-model="createForm.checker" placeholder="默认当前用户" style="width: 100%" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="createForm.remark" type="textarea" :rows="2" placeholder="备注（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="large" @click="createVisible = false">取消</el-button>
          <el-button type="primary" size="large" class="btn-save" :loading="saving" @click="handleCreate">创建</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 录入盘点数 -->
    <el-dialog v-model="recordVisible" title="录入盘点数" width="760px" destroy-on-close class="crud-dialog">
      <div class="tip-bar">录入各 SKU 的实盘数量，未录入的明细完成时按账存视为相符</div>
      <el-table :data="recordItems" border size="small" max-height="420" class="pos-table">
        <el-table-column label="SKU 编码" width="150">
          <template #default="{ row }">{{ row.sku?.skuCode || row.skuId }}</template>
        </el-table-column>
        <el-table-column label="商品" min-width="130">
          <template #default="{ row }">{{ row.sku?.goods?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="规格" width="120">
          <template #default="{ row }">{{ (row.sku?.color || '') + ' / ' + (row.sku?.size || '') }}</template>
        </el-table-column>
        <el-table-column label="账存数量" width="90" align="right">
          <template #default="{ row }"><span class="sys-qty">{{ row.systemQty }}</span></template>
        </el-table-column>
        <el-table-column label="实盘数量" width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.actualQty" :min="0" style="width: 100%" size="large" />
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="large" @click="recordVisible = false">取消</el-button>
          <el-button type="primary" size="large" class="btn-save" :loading="saving" @click="handleSaveRecord">保存录入</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 详情 -->
    <el-dialog v-model="detailVisible" title="盘点单详情" width="780px" destroy-on-close class="crud-dialog">
      <el-descriptions :column="2" border size="small" class="detail-desc">
        <el-descriptions-item label="盘点单号">{{ detail.checkNo }}</el-descriptions-item>
        <el-descriptions-item label="仓库">{{ detail.warehouse?.name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="statusTag(detail.status).type" effect="light">{{ statusTag(detail.status).text }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="盘点人">{{ detail.checker }}</el-descriptions-item>
        <el-descriptions-item label="备注" :span="2">{{ detail.remark || '-' }}</el-descriptions-item>
      </el-descriptions>
      <el-table :data="detail.items || []" border size="small" max-height="360" class="pos-table">
        <el-table-column label="SKU 编码" width="150">
          <template #default="{ row }">{{ row.sku?.skuCode || row.skuId }}</template>
        </el-table-column>
        <el-table-column label="商品" min-width="120">
          <template #default="{ row }">{{ row.sku?.goods?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="账存" width="80" align="right">
          <template #default="{ row }">{{ row.systemQty }}</template>
        </el-table-column>
        <el-table-column label="实盘" width="80" align="right">
          <template #default="{ row }">{{ row.actualQty ?? row.systemQty }}</template>
        </el-table-column>
        <el-table-column label="差异" width="90" align="right">
          <template #default="{ row }">
            <b :style="{ color: row.diffQty > 0 ? '#67c23a' : row.diffQty < 0 ? '#f56c6c' : '#909399' }">
              {{ row.diffQty > 0 ? '+' : '' }}{{ row.diffQty }}
            </b>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'
import {
  getStockCheckPage, getStockCheckDetail, createStockCheck,
  updateStockCheckItems, completeStockCheck, cancelStockCheck,
} from '@/api/jxc/stockcheck'
import { getWarehouseList } from '@/api/jxc/basic'

const list = ref([])
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const search = reactive({ keyword: '', page: 1, pageSize: 10 })

const warehouses = ref([])
const createVisible = ref(false)
const createRef = ref()
const createForm = reactive({ warehouseId: undefined, checker: '', remark: '' })

const recordVisible = ref(false)
const recordItems = ref([])
const recordCheckID = ref(0)

const detailVisible = ref(false)
const detail = ref({})

const statusTag = (s) => {
  if (s === 1) return { text: '盘点中', type: 'warning' }
  if (s === 2) return { text: '已完成', type: 'success' }
  return { text: '已取消', type: 'info' }
}

const fetchData = async () => {
  loading.value = true
  try {
    const { data } = await getStockCheckPage(search)
    list.value = data.list || []
    total.value = data.total || 0
  } finally {
    loading.value = false
  }
}

const loadWarehouses = async () => {
  const { data } = await getWarehouseList({ page: 1, pageSize: 999 })
  warehouses.value = data.list || []
}

const openCreate = () => {
  Object.assign(createForm, { warehouseId: undefined, checker: '', remark: '' })
  createVisible.value = true
}

const handleCreate = async () => {
  await createRef.value.validate()
  saving.value = true
  try {
    const res = await createStockCheck(createForm)
    if (res && res.code !== 0) return
    ElMessage.success('创建成功，请在「录入盘点」中填写实盘数量')
    createVisible.value = false
    await fetchData()
  } finally {
    saving.value = false
  }
}

const openRecord = async (row) => {
  const { data } = await getStockCheckDetail(row.ID)
  recordCheckID.value = row.ID
  recordItems.value = (data.items || []).map((it) => ({ ...it, actualQty: it.actualQty ?? it.systemQty }))
  recordVisible.value = true
}

const handleSaveRecord = async () => {
  saving.value = true
  try {
    const items = recordItems.value.map((it) => ({ id: it.ID, actualQty: it.actualQty }))
    const res = await updateStockCheckItems({ checkId: recordCheckID.value, items })
    if (res && res.code !== 0) return
    ElMessage.success('录入成功')
    recordVisible.value = false
    await fetchData()
  } finally {
    saving.value = false
  }
}

const handleComplete = async (row) => {
  const { data } = await getStockCheckDetail(row.ID)
  const diffs = (data.items || []).filter((it) => {
    const actual = it.actualQty ?? it.systemQty
    return actual - it.systemQty !== 0
  })
  const summary = diffs.length
    ? `共 ${diffs.length} 项差异（盘盈 ${diffs.filter((d) => (d.actualQty ?? d.systemQty) - d.systemQty > 0).length} 项 / 盘亏 ${diffs.filter((d) => (d.actualQty ?? d.systemQty) - d.systemQty < 0).length} 项），完成后将自动调整库存并生成流水。`
    : '该盘点单无差异，完成后不调整库存。'
  await ElMessageBox.confirm(`确认完成盘点？${summary}`, '完成盘点', { type: 'warning' })
  const res = await completeStockCheck(row.ID)
  if (res && res.code !== 0) return
  ElMessage.success('盘点完成，库存已调整')
  await fetchData()
}

const handleCancel = async (row) => {
  await ElMessageBox.confirm('确认取消该盘点单？', '取消盘点', { type: 'warning' })
  const res = await cancelStockCheck(row.ID)
  if (res && res.code !== 0) return
  ElMessage.success('已取消')
  await fetchData()
}

const openDetail = async (row) => {
  const { data } = await getStockCheckDetail(row.ID)
  detail.value = data
  detailVisible.value = true
}

onMounted(() => {
  fetchData()
  loadWarehouses()
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
.op-group { display: flex; align-items: center; flex-wrap: nowrap; }
.op-sep { width: 1px; height: 14px; background: #e4e7ed; margin: 0 6px; flex-shrink: 0; }
.op-action { font-weight: 600; }
.page-bar { margin-top: 14px; justify-content: flex-end; }

/* ===== 弹窗 ===== */
.crud-dialog :deep(.el-dialog__header) { padding-bottom: 8px; }
.crud-dialog :deep(.el-dialog__title) { font-weight: 700; }
.crud-dialog :deep(.el-dialog__body) { padding-top: 12px; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 10px; }
.btn-save { min-width: 120px; font-weight: 600; }
.tip-bar {
  background: #f0f9eb;
  border: 1px solid #e1f3d8;
  color: #67c23a;
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 13px;
  margin-bottom: 12px;
}
.sys-qty { font-weight: 700; font-variant-numeric: tabular-nums; }
.detail-desc { margin-bottom: 14px; }
</style>

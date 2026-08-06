<template>
  <div class="jxc-page">
    <!-- 搜索栏 -->
    <div class="crud-search">
      <el-input
        v-model="search.keyword"
        placeholder="采购单号搜索"
        clearable
        style="width: 230px"
        @keyup.enter="fetchData"
        @clear="fetchData"
      />
      <div class="search-actions">
        <el-button type="primary" :icon="Search" @click="fetchData">查询</el-button>
        <el-button type="success" class="btn-create" :icon="Plus" @click="openCreate">新建采购单</el-button>
      </div>
    </div>

    <!-- 列表 -->
    <div class="crud-table">
      <el-table :data="list" v-loading="loading" border stripe class="pos-table">
        <el-table-column label="采购单号" prop="orderNo" width="180" fixed="left" />
        <el-table-column label="供应商" min-width="140">
          <template #default="{ row }">{{ row.supplier?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="入库仓库" min-width="120">
          <template #default="{ row }">{{ row.warehouse?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="总金额" width="120" align="right">
          <template #default="{ row }"><span class="col-amount">¥ {{ row.totalAmount?.toFixed(2) }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status).type" effect="light">{{ statusTag(row.status).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建人" prop="creator" width="90" />
        <el-table-column label="创建时间" prop="createdAt" width="170" />
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <div class="op-group">
              <el-button link type="primary" @click="openDetail(row.ID)">查看</el-button>
              <template v-if="row.status === 1">
                <span class="op-sep"></span>
                <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
                <el-popconfirm title="确认删除该采购单?" @confirm="handleDelete(row.ID)">
                  <template #reference><el-button link type="danger">删除</el-button></template>
                </el-popconfirm>
                <el-popconfirm title="确认审核通过?" @confirm="handleAudit(row.ID)">
                  <template #reference><el-button link type="warning" class="op-action">审核</el-button></template>
                </el-popconfirm>
                <el-popconfirm title="确认取消该采购单?" @confirm="handleCancel(row.ID)">
                  <template #reference><el-button link type="info">取消</el-button></template>
                </el-popconfirm>
              </template>
              <template v-else-if="row.status === 2">
                <span class="op-sep"></span>
                <el-popconfirm title="确认入库? 入库后库存将增加" @confirm="handleStockIn(row.ID)">
                  <template #reference><el-button link type="success" class="op-action">入库</el-button></template>
                </el-popconfirm>
                <el-popconfirm title="确认取消该采购单?" @confirm="handleCancel(row.ID)">
                  <template #reference><el-button link type="info">取消</el-button></template>
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
    <el-dialog v-model="editVisible" :title="form.ID ? '编辑采购单' : '新建采购单'" width="880px" destroy-on-close class="crud-dialog">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="84px">
        <div class="form-grid">
          <el-form-item label="供应商" prop="supplierId">
            <el-select v-model="form.supplierId" placeholder="选择供应商" style="width: 100%" filterable>
              <el-option v-for="s in suppliers" :key="s.ID" :label="s.name" :value="s.ID" />
            </el-select>
          </el-form-item>
          <el-form-item label="入库仓库" prop="warehouseId">
            <el-select v-model="form.warehouseId" placeholder="选择仓库" style="width: 100%">
              <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
            </el-select>
          </el-form-item>
          <el-form-item label="备注" class="span-2">
            <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="备注（可选）" />
          </el-form-item>
        </div>

        <!-- 明细 -->
        <div class="items-block">
          <div class="items-head">
            <span class="items-title">采购明细</span>
            <span class="items-total">合计：<b class="col-amount">¥ {{ totalAmount.toFixed(2) }}</b></span>
          </div>
          <el-table :data="form.items" border size="small" max-height="280" class="pos-table">
            <el-table-column label="SKU" min-width="220">
              <template #default="{ row }">
                <el-select v-model="row.skuId" placeholder="选择SKU" filterable style="width: 100%" @change="onSkuChange(row)">
                  <el-option v-for="s in skuOptions" :key="s.ID" :label="s.skuCode" :value="s.ID" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="数量" width="130">
              <template #default="{ row }">
                <el-input-number v-model="row.qty" :min="1" style="width: 100%" />
              </template>
            </el-table-column>
            <el-table-column label="单价" width="140">
              <template #default="{ row }">
                <el-input-number v-model="row.price" :min="0" :precision="2" style="width: 100%" />
              </template>
            </el-table-column>
            <el-table-column label="金额" width="120" align="right">
              <template #default="{ row }"><span class="col-amount">¥ {{ ((row.qty || 0) * (row.price || 0)).toFixed(2) }}</span></template>
            </el-table-column>
            <el-table-column label="操作" width="60" align="center">
              <template #default="{ $index }">
                <el-button link type="danger" @click="form.items.splice($index, 1)">删</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button type="primary" link :icon="Plus" class="add-btn" @click="addItem">添加明细</el-button>
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
    <el-dialog v-model="detailVisible" title="采购单详情" width="840px" class="crud-dialog">
      <template v-if="detail.ID">
        <el-descriptions :column="2" border size="small" class="detail-desc">
          <el-descriptions-item label="采购单号">{{ detail.orderNo }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusTag(detail.status).type" effect="light">{{ statusTag(detail.status).text }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="供应商">{{ detail.supplier?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="入库仓库">{{ detail.warehouse?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="总金额"><span class="col-amount">¥ {{ detail.totalAmount?.toFixed(2) }}</span></el-descriptions-item>
          <el-descriptions-item label="创建人">{{ detail.creator }}</el-descriptions-item>
          <el-descriptions-item label="备注" :span="2">{{ detail.remark || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="detail.items || []" border size="small" class="pos-table">
          <el-table-column label="SKU编码" prop="skuCode" min-width="140" />
          <el-table-column label="商品名称" prop="goodsName" min-width="140" />
          <el-table-column label="颜色/尺码" min-width="110">
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
import { getPurchasePage, getPurchaseDetail, createPurchase, updatePurchase, deletePurchase, auditPurchase, cancelPurchase, stockInPurchase } from '@/api/jxc/purchase'
import { getSupplierList } from '@/api/jxc/basic'
import { getWarehouseList } from '@/api/jxc/basic'
import { getSkuList } from '@/api/jxc/goods'

const loading = ref(false)
const saving = ref(false)
const list = ref([])
const total = ref(0)
const search = reactive({ page: 1, pageSize: 10, keyword: '' })

const suppliers = ref([])
const warehouses = ref([])
const skuOptions = ref([])

const editVisible = ref(false)
const detailVisible = ref(false)
const detail = ref({})
const formRef = ref()
const form = reactive({ ID: 0, supplierId: undefined, warehouseId: undefined, remark: '', items: [] })

const rules = {
  supplierId: [{ required: true, message: '请选择供应商', trigger: 'change' }],
  warehouseId: [{ required: true, message: '请选择入库仓库', trigger: 'change' }],
}

const statusTag = (s) => ({ 1: { type: 'info', text: '待审核' }, 2: { type: 'warning', text: '已审核' }, 3: { type: 'success', text: '已入库' }, 4: { type: 'danger', text: '已取消' } }[s] || { type: 'info', text: s })

const totalAmount = computed(() => form.items.reduce((sum, it) => sum + (it.qty || 0) * (it.price || 0), 0))

const fetchData = async () => {
  loading.value = true
  try {
    const { data } = await getPurchasePage(search)
    list.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

const loadOptions = async () => {
  const [s, w, sku] = await Promise.all([
    getSupplierList({ page: 1, pageSize: 999 }),
    getWarehouseList({ page: 1, pageSize: 999 }),
    getSkuList({ page: 1, pageSize: 999 }),
  ])
  suppliers.value = s.data.list || []
  warehouses.value = w.data.list || []
  skuOptions.value = (sku.data || []).filter(s => s.status === 1)
}

const addItem = () => form.items.push({ skuId: undefined, qty: 1, price: 0 })

// 选中 SKU 后自动带出其成本价作为采购单价（可手动修改）
const onSkuChange = (row) => {
  const sku = skuOptions.value.find((s) => s.ID === row.skuId)
  if (sku) row.price = sku.costPrice || 0
}

const openCreate = () => {
  Object.assign(form, { ID: 0, supplierId: undefined, warehouseId: undefined, remark: '', items: [] })
  addItem()
  editVisible.value = true
}

const openEdit = (row) => {
  Object.assign(form, { ID: row.ID, supplierId: row.supplierId, warehouseId: row.warehouseId, remark: row.remark, items: [] })
  getPurchaseDetail(row.ID).then(({ data }) => {
    form.items = (data.items || []).map((it) => ({ skuId: it.skuId, qty: it.qty, price: it.price }))
  })
  editVisible.value = true
}

const openDetail = async (id) => {
  const { data } = await getPurchaseDetail(id)
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
    const res = form.ID ? await updatePurchase(form) : await createPurchase(form)
    if (res && res.code !== 0) return
    ElMessage.success(form.ID ? '更新成功' : '创建成功')
    editVisible.value = false
    await fetchData()
  } finally {
    saving.value = false
  }
}

const handleDelete = async (id) => {
  const res = await deletePurchase(id)
  if (res && res.code !== 0) return
  ElMessage.success('删除成功')
  await fetchData()
}

const handleAudit = async (id) => {
  const res = await auditPurchase(id)
  if (res && res.code !== 0) return
  ElMessage.success('审核成功')
  await fetchData()
}

const handleCancel = async (id) => {
  const res = await cancelPurchase(id)
  if (res && res.code !== 0) return
  ElMessage.success('已取消')
  await fetchData()
}

const handleStockIn = async (id) => {
  const res = await stockInPurchase(id)
  if (res && res.code !== 0) return
  ElMessage.success('入库成功，库存已更新')
  await fetchData()
}

onMounted(() => {
  fetchData()
  loadOptions()
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
.add-btn { margin-top: 6px; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 10px; }
.btn-save { min-width: 120px; font-weight: 600; }
.detail-desc { margin-bottom: 14px; }
</style>

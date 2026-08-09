<template>
  <div class="jxc-page">
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <div class="btn-list-left">
          <el-button icon="arrow-left" @click="goBack">返回</el-button>
          <el-button type="primary" icon="plus" @click="openDialog()">新增规格</el-button>
        </div>
        <div class="btn-list-right">
          <el-select
            v-model="barcodeFormat"
            title="条码格式（列表/打印贴标生效）"
            style="width: 130px"
            @change="saveBarcodeFormat"
          >
            <el-option label="CODE128" value="CODE128" />
            <el-option label="EAN13" value="EAN13" />
            <el-option label="CODE39" value="CODE39" />
            <el-option label="QR 码" value="QR" />
          </el-select>
          <el-button
            type="primary"
            plain
            :icon="Printer"
            :disabled="!selectedRows.length"
            @click="openPrint(selectedRows)"
          >
            打印贴标{{ selectedRows.length ? `（${selectedRows.length}）` : '' }}
          </el-button>
        </div>
      </div>
      <div class="gva-search">
        <el-select v-if="!goodsId" v-model="form.goodsId" placeholder="选择所属商品" filterable style="width: 260px" @change="fetchData">
          <el-option v-for="g in goodsOptions" :key="g.ID" :label="`${g.code} ${g.name}`" :value="g.ID" />
        </el-select>
        <span v-else class="goods-title">商品：{{ goodsName }}</span>
      </div>
      <el-table ref="tableRef" :data="tableData" border v-loading="loading" class="pos-table" @selection-change="onSelectionChange">
        <el-table-column type="selection" width="46" align="center" />
        <el-table-column label="SKU编码" prop="skuCode" min-width="150">
          <template #default="{ row }"><span class="cell-name">{{ row.skuCode }}</span></template>
        </el-table-column>
        <el-table-column label="条码" prop="barcode" width="190">
          <template #default="{ row }">
            <!-- 条码可视化：无值用 SKU 编码兜底；点击打开放大预览（打印弹窗单行） -->
            <div class="barcode-cell" title="点击放大预览" @click="openPrint([row])">
              <Barcode :value="row.barcode || row.skuCode" :format="barcodeFormat" :width="150" :height="50" @fallback="onBarcodeFallback" />
            </div>
          </template>
        </el-table-column>
        <el-table-column label="颜色" prop="color" width="100" />
        <el-table-column label="尺码" prop="size" width="90" align="center" />
        <el-table-column label="成本价" width="120" align="right">
          <template #default="{ row }"><span class="col-cost">¥ {{ row.costPrice?.toFixed(2) }}</span></template>
        </el-table-column>
        <el-table-column label="销售价" width="120" align="right">
          <template #default="{ row }"><span class="col-amount">¥ {{ row.salePrice?.toFixed(2) }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 1" @change="toggleStatus(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <div class="op-group">
              <el-button type="primary" link @click="openDialog(row.ID)">编辑</el-button>
              <el-popconfirm title="确认删除该规格?" @confirm="handleDelete(row.ID)">
                <template #reference>
                  <el-button type="danger" link>删除</el-button>
                </template>
              </el-popconfirm>
              <el-popconfirm title="彻底删除不可恢复，确认?" @confirm="handleDeleteForever(row.ID)">
                <template #reference>
                  <el-button type="danger" link>彻底删除</el-button>
                </template>
              </el-popconfirm>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑规格' : '新增规格'" width="500px" class="crud-dialog" @closed="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item v-if="!goodsId" label="所属商品" prop="goodsId">
          <el-select v-model="form.goodsId" placeholder="选择商品" filterable style="width: 100%">
            <el-option v-for="g in goodsOptions" :key="g.ID" :label="`${g.code} ${g.name}`" :value="g.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="SKU编码" prop="skuCode">
          <el-input v-model="form.skuCode" :disabled="!!form.ID" placeholder="保存后自动生成" />
        </el-form-item>
        <div class="form-row">
          <el-form-item label="颜色" prop="color">
            <el-input v-model="form.color" />
          </el-form-item>
          <el-form-item label="尺码" prop="size">
            <el-input v-model="form.size" />
          </el-form-item>
        </div>
        <el-form-item label="条码" prop="barcode">
          <el-input v-model="form.barcode" placeholder="可选" clearable>
            <template #append><el-button @click="generateBarcodeFromCode">按编码生成</el-button></template>
          </el-input>
        </el-form-item>
        <div class="form-row">
          <el-form-item label="成本价" prop="costPrice">
            <el-input-number v-model="form.costPrice" :min="0" :precision="2" :step="1" style="width: 100%" />
          </el-form-item>
          <el-form-item label="销售价" prop="salePrice">
            <el-input-number v-model="form.salePrice" :min="0" :precision="2" :step="1" style="width: 100%" />
          </el-form-item>
        </div>
        <el-form-item label="安全库存">
          <el-input-number v-model="form.safeStock" :min="0" :step="1" style="width: 100%" />
          <div class="tip-text">低于该库存时仪表盘预警（0=不预警）</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="large" @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" size="large" class="btn-save" @click="submitForm" :loading="submitLoading">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 打印贴标弹窗：选中行 / 点击行条码进入；打印时仅打印该区域 -->
    <el-dialog v-model="printVisible" title="打印贴标" width="780px" class="print-dialog" append-to-body @closed="onPrintClosed">
      <div id="print-area" class="label-grid">
        <div v-for="(row, i) in printRows" :key="row.ID ?? i" class="label-item">
          <div class="label-name">{{ (row.goods && row.goods.name) || row.skuCode }}</div>
          <div class="label-spec">
            {{ row.skuCode }}<template v-if="row.color || row.size"> · {{ row.color }} / {{ row.size }}</template>
          </div>
          <Barcode :value="row.barcode || row.skuCode" :format="barcodeFormat" :width="200" :height="60" @fallback="onBarcodeFallback" />
          <div class="label-price">¥ {{ (row.salePrice || 0).toFixed(2) }}</div>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="large" @click="printVisible = false">关闭</el-button>
          <el-button type="primary" size="large" :icon="Printer" @click="doPrint">打印</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { Printer } from '@element-plus/icons-vue'
import Barcode from '@/components/jxc/Barcode.vue'
import { getSkuList, createSku, updateSku, deleteSku, deleteSkuForever, setSkuStatus } from '@/api/jxc/goods'
import { getAllGoods } from '@/api/jxc/goods'

const route = useRoute()
const router = useRouter()
const goodsId = Number(route.query.goodsId || 0)
const goodsName = route.query.goodsName || ''
const loading = ref(false)
const tableData = ref([])
const goodsOptions = ref([])
const dialogVisible = ref(false), submitLoading = ref(false), isEdit = ref(false)
const formRef = ref(null)
const form = reactive({ ID: 0, goodsId, skuCode: '', barcode: '', color: '', size: '', costPrice: 0, salePrice: 0, safeStock: 0 })
const rules = {
  // skuCode: 编码由后端自动生成（商品编码-颜色-尺码）
  goodsId: [{ required: true, message: '请选择所属商品', trigger: 'change' }],
  color: [{ required: true, message: '请输入颜色', trigger: 'blur' }],
  size: [{ required: true, message: '请输入尺码', trigger: 'blur' }],
}

async function loadGoods() {
  if (goodsId) return
  const { data } = await getAllGoods()
  goodsOptions.value = data || []
}

function resetForm() { Object.assign(form, { ID: 0, goodsId, skuCode: '', barcode: '', color: '', size: '', costPrice: 0, salePrice: 0, safeStock: 0 }) }

async function fetchData() {
  loading.value = true
  try {
    const { data } = await getSkuList({ id: goodsId })
    tableData.value = data || []
  } finally { loading.value = false }
}

onMounted(() => { loadGoods(); fetchData() })

function openDialog(id) {
  isEdit.value = !!id
  resetForm()
  if (id) {
    const row = tableData.value.find(r => r.ID === id)
    if (row) Object.assign(form, {
      ID: row.ID, goodsId: row.goodsId, skuCode: row.skuCode, barcode: row.barcode || '',
      color: row.color, size: row.size, costPrice: row.costPrice || 0, salePrice: row.salePrice || 0,
    })
  }
  dialogVisible.value = true
}

function goBack() { router.back() }

async function submitForm() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  if (!form.goodsId) {
    ElMessage.warning('请选择所属商品')
    return
  }
  submitLoading.value = true
  try {
    const res = form.ID ? await updateSku(form) : await createSku(form)
    if (res && res.code !== 0) return
    ElMessage.success(form.ID ? '更新成功' : '创建成功')
    dialogVisible.value = false
    await fetchData()
  } finally { submitLoading.value = false }
}

async function handleDelete(id) {
  const res = await deleteSku({ id })
  if (res && res.code !== 0) return
  ElMessage.success('删除成功')
  await fetchData()
}

async function handleDeleteForever(id) {
  const res = await deleteSkuForever({ id })
  if (res && res.code !== 0) return
  ElMessage.success('已彻底删除')
  await fetchData()
}

async function toggleStatus(row) {
  const s = row.status === 1 ? 0 : 1
  await setSkuStatus({ id: row.ID, status: s })
  row.status = s
  ElMessage.success(s === 1 ? '已启用' : '已停用')
}

// ===== 条码可视化 / 打印贴标（纯前端渲染，不影响业务逻辑） =====
const BARCODE_FORMAT_KEY = 'jxc-barcode-format'
// 条码格式选择：localStorage 持久化，进入页面时读取
const barcodeFormat = ref(localStorage.getItem(BARCODE_FORMAT_KEY) || 'CODE128')
const saveBarcodeFormat = () => localStorage.setItem(BARCODE_FORMAT_KEY, barcodeFormat.value)

// 表格勾选行：打印贴标按钮按选中数启用
const tableRef = ref(null)
const selectedRows = ref([])
const onSelectionChange = (rows) => { selectedRows.value = rows }

// 打印弹窗：选中行批量 / 点击行条码单行预览
const printVisible = ref(false)
const printRows = ref([])
const openPrint = (rows) => {
  printRows.value = rows || []
  printVisible.value = true
}
// 弹窗关闭后清空勾选，避免残留选中态
const onPrintClosed = () => {
  printRows.value = []
  tableRef.value?.clearSelection()
}
const doPrint = () => window.print()

// 表单条码输入辅助：把 SKU 编码填入条码字段（无编码时提示）
const generateBarcodeFromCode = () => {
  if (!form.skuCode) {
    ElMessage.warning('SKU 编码为空，暂无法生成')
    return
  }
  form.barcode = form.skuCode
  ElMessage.success('已按 SKU 编码填入条码')
}

// fallback 节流提示：同一批（300ms 内）渲染失败只提示一次；展示组件给出的具体原因（如「内容含中文」「EAN13 需 12/13 位数字」）
let fallbackAt = 0
const onBarcodeFallback = (info) => {
  const now = Date.now()
  if (now - fallbackAt < 300) return
  fallbackAt = now
  ElMessage.warning((info && info.reason) || '条码格式与内容不匹配')
}

</script>

<style scoped>
/* ===== POS 设计语言 · 覆盖 GVA 容器 ===== */
.jxc-page { padding: 4px 0; }
.gva-table-box {
  background: #fff;
  border-radius: 12px;
  padding: 14px 14px 16px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
}
.gva-btn-list .el-button { height: 36px; font-weight: 600; }
.gva-search {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 2px 14px;
}
.goods-title { font-size: 15px; font-weight: 700; color: #303133; }
.pos-table { width: 100%; }
.pos-table :deep(th.el-table__cell) { background: #f7f9fc; font-weight: 700; color: #303133; }
.pos-table :deep(.el-table__cell) { padding: 9px 0; }
.cell-name { font-weight: 600; color: #303133; }
.col-cost { font-weight: 600; color: #909399; font-variant-numeric: tabular-nums; }
.col-amount { font-weight: 700; color: #e6a23c; font-variant-numeric: tabular-nums; }
.op-group { display: flex; align-items: center; flex-wrap: nowrap; }
.op-group .el-button { margin-left: 0 !important; }
.op-group .el-button + .el-button { margin-left: 2px; }

/* ===== 弹窗 ===== */
.crud-dialog :deep(.el-dialog__header) { padding-bottom: 8px; }
.crud-dialog :deep(.el-dialog__title) { font-weight: 700; }
.crud-dialog :deep(.el-dialog__body) { padding-top: 12px; }
.form-row { display: flex; gap: 14px; }
.form-row .el-form-item { flex: 1; }
.tip-text { color: #909399; font-size: 12px; margin-top: 6px; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 10px; }
.btn-save { min-width: 110px; font-weight: 600; }

/* ===== 工具区：左右分组 ===== */
.gva-btn-list { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 10px; }
.btn-list-left,
.btn-list-right { display: flex; align-items: center; gap: 10px; }

/* ===== 条码可视化列 ===== */
.barcode-cell { cursor: pointer; border-radius: 6px; transition: background 0.15s; }
.barcode-cell:hover { background: #f5f8ff; }

/* ===== 打印贴标弹窗 ===== */
.print-dialog :deep(.el-dialog__body) { padding-top: 12px; }
.label-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}
.label-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  border: 1px dashed #999;
  border-radius: 8px;
  padding: 14px 10px;
  page-break-inside: avoid;
  break-inside: avoid;
}
.label-name {
  font-size: 13px;
  font-weight: 700;
  color: #303133;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.label-spec {
  font-size: 11px;
  color: #909399;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.label-price { font-size: 16px; font-weight: 700; color: #ff5a1f; }
</style>

<!-- 打印样式：非 scoped 保证全局命中（scoped 会给选择器加 data-v 后缀导致 body * 失效） -->
<style>
@media print {
  body * { visibility: hidden; }
  #print-area,
  #print-area * { visibility: visible; }
  #print-area {
    position: absolute;
    left: 0;
    top: 0;
    width: 100%;
    margin: 0;
    padding: 12px;
  }
  #print-area .label-item {
    page-break-inside: avoid;
    break-inside: avoid;
  }
}
</style>

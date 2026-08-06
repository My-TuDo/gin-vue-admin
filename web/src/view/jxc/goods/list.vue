<template>
  <div class="jxc-page">
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" icon="plus" @click="openDialog()">新增商品</el-button>
      </div>
      <div class="gva-search">
        <el-input v-model="keyword" placeholder="搜索编码/名称" clearable @clear="fetchData" @keyup.enter="fetchData" style="width: 260px" />
        <el-button type="primary" @click="fetchData">搜索</el-button>
      </div>
      <el-table :data="tableData" border v-loading="loading" class="pos-table">
        <el-table-column label="编码" prop="code" width="120" />
        <el-table-column label="名称" prop="name" min-width="150">
          <template #default="{ row }"><span class="cell-name">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="主图" width="70" align="center">
          <template #default="{ row }">
            <el-image v-if="row.image" :src="getUrl(row.image)" :preview-src-list="[getUrl(row.image)]" preview-teleported fit="cover" style="width: 40px; height: 40px; border-radius: 6px" />
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="分类" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.category?.name" type="info" effect="plain" size="small">{{ row.category.name }}</el-tag>
            <span v-else class="text-muted">未分类</span>
          </template>
        </el-table-column>
        <el-table-column label="品牌" width="110">
          <template #default="{ row }">{{ row.brand?.name || '无品牌' }}</template>
        </el-table-column>
        <el-table-column label="单位" prop="unit" width="70" align="center" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 1" @change="toggleStatus(row)" />
          </template>
        </el-table-column>
        <el-table-column label="备注" prop="remark" min-width="120" :show-overflow-tooltip="true" />
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }"> {{ dayjs(row.CreatedAt).format('YYYY-MM-DD HH:mm') }} </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <div class="op-group">
              <el-button type="primary" link @click="openSku(row)">规格</el-button>
              <el-button type="primary" link @click="openDialog(row.ID)">编辑</el-button>
              <el-popconfirm title="确认删除该商品?" @confirm="handleDelete(row.ID)">
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
      <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total"
        layout="total, sizes, prev, pager, next" class="page-bar" @current-change="fetchData" @size-change="fetchData" />
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑商品' : '新增商品'" width="520px" class="crud-dialog" @closed="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="编码" prop="code">
          <el-input v-model="form.code" :disabled="!!form.ID" placeholder="保存后自动生成" />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <div class="form-row">
          <el-form-item label="分类" prop="categoryId">
            <el-select v-model="form.categoryId" clearable placeholder="请选择分类">
              <el-option v-for="c in categoryList" :key="c.ID" :label="c.name" :value="c.ID" />
            </el-select>
          </el-form-item>
          <el-form-item label="品牌" prop="brandId">
            <el-select v-model="form.brandId" clearable placeholder="请选择品牌">
              <el-option v-for="b in brandList" :key="b.ID" :label="b.name" :value="b.ID" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="单位" prop="unit">
          <el-input v-model="form.unit" placeholder="件/条/双/套" />
        </el-form-item>
        <el-form-item label="商品主图">
          <upload-image :image-url="form.image" @on-success="(url) => (form.image = url)" />
          <el-image v-if="form.image" :src="getUrl(form.image)" :preview-src-list="[getUrl(form.image)]" preview-teleported fit="cover" style="width: 60px; height: 60px; border-radius: 6px; margin-left: 8px" />
        </el-form-item>
        <el-form-item label="备注" prop="remark">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="large" @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" size="large" class="btn-save" @click="submitForm" :loading="submitLoading">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import { useRouter } from 'vue-router'
import { getGoodsPage, createGoods, updateGoods, deleteGoods, deleteGoodsForever, setGoodsStatus } from '@/api/jxc/goods'
import { getCategoryTree, getAllBrands } from '@/api/jxc/basic'
import UploadImage from '@/components/upload/image.vue'
import { getUrl } from '@/utils/image'

const router = useRouter()
const loading = ref(false), page = ref(1), pageSize = ref(10), total = ref(0), keyword = ref('')
const tableData = ref([]), categoryList = ref([]), brandList = ref([])
const dialogVisible = ref(false), submitLoading = ref(false), isEdit = ref(false)
const formRef = ref(null)
const form = reactive({ ID: 0, code: '', name: '', categoryId: null, brandId: null, unit: '件', image: '', remark: '' })
const rules = {
  // code: 编码由后端自动生成（前缀+日期+序号）
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

function resetForm() { Object.assign(form, { ID: 0, code: '', name: '', categoryId: null, brandId: null, unit: '件', image: '', remark: '' }) }

async function fetchData() {
  loading.value = true
  try {
    const { data } = await getGoodsPage({ page: page.value, pageSize: pageSize.value, keyword: keyword.value })
    tableData.value = data.list || []
    total.value = data.total || 0
  } finally { loading.value = false }
}

async function loadSelects() {
  const [cat, brand] = await Promise.all([getCategoryTree(), getAllBrands()])
  categoryList.value = cat.data || []
  brandList.value = brand.data || []
}

function openDialog(id) {
  isEdit.value = !!id
  resetForm()
  if (id) {
    const row = tableData.value.find(r => r.ID === id)
    if (row) Object.assign(form, {
      ID: row.ID, code: row.code, name: row.name,
      categoryId: row.categoryId || null, brandId: row.brandId || null,
      unit: row.unit || '件', image: row.image || '', remark: row.remark || '',
    })
  }
  dialogVisible.value = true
}

function openSku(row) { router.push({ path: '/jxc/goods/sku', query: { goodsId: row.ID, goodsName: row.name } }) }

async function submitForm() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitLoading.value = true
  try {
    const res = form.ID ? await updateGoods(form) : await createGoods(form)
    if (res && res.code !== 0) return
    ElMessage.success(form.ID ? '更新成功' : '创建成功')
    dialogVisible.value = false
    await fetchData()
  } finally { submitLoading.value = false }
}

async function handleDelete(id) {
  const res = await deleteGoods({ id })
  if (res && res.code !== 0) return
  ElMessage.success('删除成功')
  await fetchData()
}

async function handleDeleteForever(id) {
  const res = await deleteGoodsForever({ id })
  if (res && res.code !== 0) return
  ElMessage.success('已彻底删除')
  await fetchData()
}

async function toggleStatus(row) {
  const s = row.status === 1 ? 0 : 1
  const res = await setGoodsStatus({ id: row.ID, status: s })
  if (res && res.code !== 0) return
  row.status = s;
  ElMessage.success(s === 1 ? '已上架' : '已下架')
}

onMounted(() => { fetchData(); loadSelects() })
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
.gva-search .el-input { width: 260px !important; }
.gva-search .el-button { height: 36px; }
.pos-table { width: 100%; }
.pos-table :deep(th.el-table__cell) { background: #f7f9fc; font-weight: 700; color: #303133; }
.pos-table :deep(.el-table__cell) { padding: 9px 0; }
.cell-name { font-weight: 600; color: #303133; }
.text-muted { color: #c0c4cc; }
.op-group { display: flex; align-items: center; flex-wrap: nowrap; }
.op-group .el-button { margin-left: 0 !important; }
.op-group .el-button + .el-button { margin-left: 2px; }
.page-bar { margin-top: 14px; justify-content: flex-end; }

/* ===== 弹窗 ===== */
.crud-dialog :deep(.el-dialog__header) { padding-bottom: 8px; }
.crud-dialog :deep(.el-dialog__title) { font-weight: 700; }
.crud-dialog :deep(.el-dialog__body) { padding-top: 12px; }
.form-row { display: flex; }
.form-row .el-form-item { flex: 1; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 10px; }
.btn-save { min-width: 110px; font-weight: 600; }
</style>

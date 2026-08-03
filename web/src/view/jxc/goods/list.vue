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
      <el-table :data="tableData" border v-loading="loading">
        <el-table-column label="编码" prop="code" width="120" />
        <el-table-column label="名称" prop="name" min-width="150" />
        <el-table-column label="分类" width="120">
          <template #default="{ row }">{{ row.category?.name || '未分类' }}</template>
        </el-table-column>
        <el-table-column label="品牌" width="110">
          <template #default="{ row }">{{ row.brand?.name || '无品牌' }}</template>
        </el-table-column>
        <el-table-column label="单位" prop="unit" width="70" align="center" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 1" active-text="上架" inactive-text="下架" @change="toggleStatus(row)" />
          </template>
        </el-table-column>
        <el-table-column label="备注" prop="remark" min-width="120" :show-overflow-tooltip="true" />
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }"> {{ dayjs(row.CreatedAt).format('YYYY-MM-DD HH:mm') }} </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openSku(row)">规格</el-button>
            <el-button type="primary" link @click="openDialog(row.ID)">编辑</el-button>
            <el-popconfirm title="确认删除该商品?" @confirm="handleDelete(row.ID)">
              <template #reference>
                <el-button type="danger" link>删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total"
        layout="total, sizes, prev, pager, next" @change="fetchData" />
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑商品' : '新增商品'" width="520px" @closed="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="编码" prop="code">
          <el-input v-model="form.code" :disabled="!!form.ID" placeholder="保存后自动生成" />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
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
        <el-form-item label="单位" prop="unit">
          <el-input v-model="form.unit" placeholder="件/条/双/套" />
        </el-form-item>
        <el-form-item label="主图URL" prop="image">
          <el-input v-model="form.image" placeholder="可选" />
        </el-form-item>
        <el-form-item label="备注" prop="remark">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitLoading">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import { useRouter } from 'vue-router'
import { getGoodsPage, createGoods, updateGoods, deleteGoods, setGoodsStatus } from '@/api/jxc/goods'
import { getCategoryTree, getAllBrands } from '@/api/jxc/basic'

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
    form.ID ? await updateGoods(form) : await createGoods(form)
    ElMessage.success(form.ID ? '更新成功' : '创建成功')
    dialogVisible.value = false
    await fetchData()
  } finally { submitLoading.value = false }
}

async function handleDelete(id) { await deleteGoods({ id }); ElMessage.success('删除成功'); await fetchData() }

async function toggleStatus(row) {
  const s = row.status === 1 ? 0 : 1
  await setGoodsStatus({ id: row.ID, status: s })
  row.status = s
  ElMessage.success(s === 1 ? '已上架' : '已下架')
}

onMounted(() => { fetchData(); loadSelects() })
</script>
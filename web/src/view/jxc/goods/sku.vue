<template>
  <div class="jxc-page">
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button icon="arrow-left" @click="goBack">返回</el-button>
        <el-button type="primary" icon="plus" @click="openDialog()">新增规格</el-button>
      </div>
      <div class="gva-search">
        <el-select v-if="!goodsId" v-model="form.goodsId" placeholder="选择所属商品" filterable style="width: 260px" @change="fetchData">
          <el-option v-for="g in goodsOptions" :key="g.ID" :label="`${g.code} ${g.name}`" :value="g.ID" />
        </el-select>
        <span v-else class="goods-title">商品：{{ goodsName }}</span>
      </div>
      <el-table :data="tableData" border v-loading="loading">
        <el-table-column label="SKU编码" prop="skuCode" min-width="160" />
        <el-table-column label="条码" prop="barcode" width="140" />
        <el-table-column label="颜色" prop="color" width="100" />
        <el-table-column label="尺码" prop="size" width="90" align="center" />
        <el-table-column label="成本价" width="110" align="right">
          <template #default="{ row }">{{ row.costPrice?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="销售价" width="110" align="right">
          <template #default="{ row }">{{ row.salePrice?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 1" @change="toggleStatus(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
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
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑规格' : '新增规格'" width="480px" @closed="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item v-if="!goodsId" label="所属商品" prop="goodsId">
          <el-select v-model="form.goodsId" placeholder="选择商品" filterable style="width: 100%">
            <el-option v-for="g in goodsOptions" :key="g.ID" :label="`${g.code} ${g.name}`" :value="g.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="SKU编码" prop="skuCode">
          <el-input v-model="form.skuCode" :disabled="!!form.ID" placeholder="保存后自动生成" />
        </el-form-item>
        <el-form-item label="条码" prop="barcode">
          <el-input v-model="form.barcode" placeholder="可选" />
        </el-form-item>
        <el-form-item label="颜色" prop="color">
          <el-input v-model="form.color" />
        </el-form-item>
        <el-form-item label="尺码" prop="size">
          <el-input v-model="form.size" />
        </el-form-item>
        <el-form-item label="成本价" prop="costPrice">
          <el-input-number v-model="form.costPrice" :min="0" :precision="2" :step="1" />
        </el-form-item>
        <el-form-item label="销售价" prop="salePrice">
          <el-input-number v-model="form.salePrice" :min="0" :precision="2" :step="1" />
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
import { useRoute, useRouter } from 'vue-router'
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
const form = reactive({ ID: 0, goodsId, skuCode: '', barcode: '', color: '', size: '', costPrice: 0, salePrice: 0 })
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

function resetForm() { Object.assign(form, { ID: 0, goodsId, skuCode: '', barcode: '', color: '', size: '', costPrice: 0, salePrice: 0 }) }

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
  submitLoading.value = true
  try {
    form.ID ? await updateSku(form) : await createSku(form)
    ElMessage.success(form.ID ? '更新成功' : '创建成功')
    dialogVisible.value = false
    await fetchData()
  } finally { submitLoading.value = false }
}

async function handleDelete(id) { await deleteSku({ id }); ElMessage.success('删除成功'); await fetchData() }

async function handleDeleteForever(id) { await deleteSkuForever({ id }); ElMessage.success('已彻底删除'); await fetchData() }

async function toggleStatus(row) {
  const s = row.status === 1 ? 0 : 1
  await setSkuStatus({ id: row.ID, status: s })
  row.status = s
  ElMessage.success(s === 1 ? '已启用' : '已停用')
}

</script>
<template>
  <div class="jxc-page">
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" icon="plus" @click="openDialog()">新增仓库</el-button>
      </div>
      <div class="gva-search">
        <el-input v-model="keyword" placeholder="搜索编码/名称" clearable @clear="fetchData" @keyup.enter="fetchData" style="width: 240px" />
        <el-button type="primary" @click="fetchData">搜索</el-button>
      </div>
      <el-table :data="tableData" border v-loading="loading">
        <el-table-column label="编码" prop="code" width="120" />
        <el-table-column label="仓库名称" prop="name" min-width="140" />
        <el-table-column label="负责人" prop="manager" width="100" />
        <el-table-column label="电话" prop="phone" width="130" />
        <el-table-column label="地址" prop="address" min-width="160" :show-overflow-tooltip="true" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 1" @change="toggleStatus(row)" />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }"> {{ dayjs(row.CreatedAt).format('YYYY-MM-DD HH:mm') }} </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDialog(row.ID)">编辑</el-button>
            <el-popconfirm title="确认删除?" @confirm="handleDelete(row.ID)">
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
      <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total"
        layout="total, sizes, prev, pager, next" @change="fetchData" />
    </div>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑仓库' : '新增仓库'" width="520px" @closed="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="编码" prop="code">
          <el-input v-model="form.code" :disabled="!!form.ID" placeholder="保存后自动生成" />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="负责人" prop="manager">
          <el-input v-model="form.manager" />
        </el-form-item>
        <el-form-item label="电话" prop="phone">
          <el-input v-model="form.phone" />
        </el-form-item>
        <el-form-item label="地址" prop="address">
          <el-input v-model="form.address" />
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
import { getWarehouseList, createWarehouse, updateWarehouse, deleteWarehouse, deleteWarehouseForever, setWarehouseStatus } from '@/api/jxc/basic'

const loading = ref(false), page = ref(1), pageSize = ref(10), total = ref(0), keyword = ref('')
const tableData = ref([]), dialogVisible = ref(false), submitLoading = ref(false), isEdit = ref(false)
const formRef = ref(null)
const form = reactive({ ID: 0, code: '', name: '', manager: '', phone: '', address: '' })
const rules = {
  // code: 编码由后端自动生成（前缀+日期+序号）
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

function resetForm() { Object.assign(form, { ID: 0, code: '', name: '', manager: '', phone: '', address: '' }) }

async function fetchData() {
  loading.value = true
  try {
    const { data } = await getWarehouseList({ page: page.value, pageSize: pageSize.value, keyword: keyword.value })
    tableData.value = data.list || []; total.value = data.total || 0
  } finally { loading.value = false }
}

function openDialog(id) {
  isEdit.value = !!id; resetForm()
  if (id) {
    const row = tableData.value.find(r => r.ID === id)
    if (row) Object.assign(form, { ID: row.ID, code: row.code, name: row.name, manager: row.manager || '', phone: row.phone || '', address: row.address || '' })
  }
  dialogVisible.value = true
}

async function submitForm() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitLoading.value = true
  try {
    const res = form.ID ? await updateWarehouse(form) : await createWarehouse(form)
    if (res && res.code !== 0) return
    ElMessage.success(form.ID ? '更新成功' : '创建成功')
    dialogVisible.value = false; await fetchData()
  } finally { submitLoading.value = false }
}

async function handleDelete(id) {
  const res = await deleteWarehouse({ id })
  if (res && res.code !== 0) return
  ElMessage.success('删除成功')
  await fetchData()
}

async function handleDeleteForever(id) {
  const res = await deleteWarehouseForever({ id })
  if (res && res.code !== 0) return
  ElMessage.success('已彻底删除')
  await fetchData()
}

async function toggleStatus(row) {
  const s = row.status === 1 ? 0 : 1
  const res = await setWarehouseStatus({ id: row.ID, status: s })
  if (res && res.code !== 0) return
  row.status = s; ElMessage.success(s ? '已启用' : '已停用')
}

onMounted(fetchData)
</script>
<template>
  <div class="jxc-page">
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" icon="plus" @click="openDialog()">新增分类</el-button>
      </div>
      <el-table
        row-key="ID"
        :data="tableData"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        border
        default-expand-all
        v-loading="loading"
      >
        <el-table-column label="编码" prop="code" width="120" />
        <el-table-column label="分类名称" prop="name" min-width="150" />
        <el-table-column label="排序" prop="sort" width="80" align="center" />
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-switch :model-value="row.status === 1" @change="toggleStatus(row)" />
          </template>
        </el-table-column>
        <el-table-column label="备注" prop="remark" min-width="140" :show-overflow-tooltip="true" />
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }"> {{ dayjs(row.CreatedAt).format('YYYY-MM-DD HH:mm') }} </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDialog(row.ID)">编辑</el-button>
            <el-button type="primary" link @click="openDialog(row.ID, 'child')">添加子分类</el-button>
            <el-popconfirm title="确认删除该分类?" @confirm="handleDelete(row.ID)">
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

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px" @closed="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item v-if="parentName" label="上级分类">
          <el-input :model-value="parentName" disabled />
        </el-form-item>
        <el-form-item label="编码" prop="code">
          <el-input v-model="form.code" :disabled="!!form.ID" placeholder="保存后自动生成" />
        </el-form-item>
        <el-form-item label="分类名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="form.sort" :min="0" :max="9999" />
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
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import { getCategoryTree, createCategory, updateCategory, deleteCategory, deleteCategoryForever, setCategoryStatus } from '@/api/jxc/basic'

const loading = ref(false)
const tableData = ref([])
const dialogVisible = ref(false), submitLoading = ref(false), isEdit = ref(false)
const formRef = ref(null)
const parentName = ref('')

const form = reactive({ ID: 0, parentId: null, code: '', name: '', sort: 0, status: 1, remark: '' })
const rules = {
  // code: 编码由后端自动生成（前缀+日期+序号）
  name: [{ required: true, message: '请输入分类名称', trigger: 'blur' }],
}

const dialogTitle = computed(() => {
  if (isEdit.value) return '编辑分类'
  return parentName.value ? `添加子分类（${parentName.value}）` : '新增分类'
})

function resetForm() {
  Object.assign(form, { ID: 0, parentId: null, code: '', name: '', sort: 0, status: 1, remark: '' })
  parentName.value = ''
}

// 在树形数据中递归查找节点
function findNode(list, id) {
  for (const item of list || []) {
    if (item.ID === id) return item
    const hit = findNode(item.children, id)
    if (hit) return hit
  }
  return null
}

async function fetchData() {
  loading.value = true
  try {
    const { data } = await getCategoryTree()
    tableData.value = data || []
  } finally { loading.value = false }
}

// 模式：不传参=新增顶级分类；传 id=编辑；传 id + 'child'=添加子分类
function openDialog(id, mode) {
  isEdit.value = !!id && mode !== 'child'
  resetForm()
  if (id && mode === 'child') {
    const parent = findNode(tableData.value, id)
    if (parent) {
      form.parentId = parent.ID
      parentName.value = parent.name
    }
  } else if (id) {
    const row = findNode(tableData.value, id)
    if (row) Object.assign(form, {
      ID: row.ID,
      parentId: row.parentId || null,
      code: row.code,
      name: row.name,
      sort: row.sort || 0,
      status: row.status === 1 ? 1 : 0,
      remark: row.remark || '',
    })
  }
  dialogVisible.value = true
}

async function submitForm() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  submitLoading.value = true
  try {
    const res = form.ID ? await updateCategory(form) : await createCategory(form)
    if (res && res.code !== 0) return
    ElMessage.success(form.ID ? '更新成功' : '创建成功')
    dialogVisible.value = false
    await fetchData()
  } finally { submitLoading.value = false }
}

async function handleDelete(id) {
  const res = await deleteCategory({ id })
  if (res && res.code !== 0) return
  ElMessage.success('删除成功')
  await fetchData()
}

async function handleDeleteForever(id) {
  const res = await deleteCategoryForever({ id })
  if (res && res.code !== 0) return
  ElMessage.success('已彻底删除')
  await fetchData()
}

async function toggleStatus(row) {
  const s = row.status === 1 ? 0 : 1
  const res = await setCategoryStatus({ id: row.ID, status: s })
  if (res && res.code !== 0) return
  row.status = s;
  ElMessage.success(s === 1 ? '已启用' : '已停用')
}

onMounted(fetchData)
</script>

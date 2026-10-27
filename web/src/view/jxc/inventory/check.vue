<template>
  <div class="check-panel">
    <div class="toolbar">
      <el-select v-model="warehouseId" placeholder="选择仓库" style="width: 160px">
        <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
      </el-select>
      <el-button type="primary" :disabled="!warehouseId" @click="createNew">发起盘点</el-button>
      <div class="spacer"></div>
      <el-button link @click="load">刷新</el-button>
    </div>

    <el-table v-loading="loading" :data="list" size="small" @row-click="open">
      <el-table-column prop="checkNo" label="盘点单号" width="170" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <span class="tag" :class="tone(row.status)">{{ label(row.status) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="明细数" width="90" align="right">
        <template #default="{ row }">{{ row.items?.length || 0 }}</template>
      </el-table-column>
      <el-table-column label="已录" width="90" align="right">
        <template #default="{ row }">{{ counted(row) }}</template>
      </el-table-column>
      <el-table-column label="差异项" width="90" align="right">
        <template #default="{ row }">{{ diffCount(row) }}</template>
      </el-table-column>
      <el-table-column label="创建时间" width="150">
        <template #default="{ row }">{{ fmt(row.CreatedAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" min-width="200">
        <template #default="{ row }">
          <el-button link type="primary" @click.stop="open(row)">录入</el-button>
          <el-button v-if="row.status === 1" link type="success" @click.stop="finish(row)">完成</el-button>
          <el-button v-if="row.status === 1" link type="danger" @click.stop="cancel(row)">取消</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="drawer" :title="current?.checkNo || '盘点'" size="720px">
      <div class="hint">
        账面上的数在建单时已快照；录入实盘数后点「完成」，差异会生成盘盈/盘亏账本行，
        <b>不会直接改库存</b>。盘盈行请填写成本单价（期初建账尤其重要），否则库存金额会一直是 0。
      </div>
      <el-table :data="current?.items || []" size="small" max-height="520">
        <el-table-column prop="skuId" label="SKU" width="70" />
        <el-table-column prop="bookQty" label="账面" width="80" align="right" />
        <el-table-column label="实盘" width="150">
          <template #default="{ row }">
            <el-input-number v-model="row.actualQty" :min="0" size="small" controls-position="right"
              style="width: 120px" :disabled="current.status !== 1" />
          </template>
        </el-table-column>
        <el-table-column label="差异" width="90" align="right">
          <template #default="{ row }">
            <span v-if="row.actualQty === null || row.actualQty === undefined" class="muted">—</span>
            <span v-else :class="diffTone(row)">{{ row.actualQty - row.bookQty > 0 ? '+' : '' }}{{ row.actualQty - row.bookQty }}</span>
          </template>
        </el-table-column>
        <el-table-column label="成本单价" width="140" align="right">
          <template #default="{ row }">
            <!-- 盘盈/期初建账必须能填成本，否则库存有数量、金额为 0，卖出去就是 100% 毛利 -->
            <el-input-number v-if="current?.status === 1 && row.actualQty > row.bookQty"
              v-model="row.unitCost" :min="0" :precision="2" size="small" controls-position="right"
              style="width: 118px" />
            <MoneyText v-else :value="row.unitCost" />
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button v-if="current?.status === 1" @click="saveItems">保存实盘</el-button>
        <el-button v-if="current?.status === 1" type="primary" @click="finish(current)">完成盘点</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import MoneyText from '@/components/jxc/MoneyText.vue'
import {
  cancelCheck, createCheck, getCheck, pageCheck, postCheck, updateCheckItems, listWarehouses
} from '@/api/jxc'

const warehouses = ref([])
const warehouseId = ref(null)
const list = ref([])
const loading = ref(false)
const drawer = ref(false)
const current = ref(null)

const STATUS = { 1: ['盘点中', 'warn'], 2: ['已完成', 'ok'], 3: ['已取消', 'muted'] }
const label = (s) => STATUS[s]?.[0] ?? '未知'
const tone = (s) => STATUS[s]?.[1] ?? 'muted'

const fmt = (s) => {
  if (!s) return ''
  const d = new Date(s)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}
const counted = (row) => (row.items || []).filter((i) => i.actualQty !== null && i.actualQty !== undefined).length
const diffCount = (row) =>
  (row.items || []).filter((i) => i.actualQty !== null && i.actualQty !== undefined && i.actualQty !== i.bookQty).length
const diffTone = (row) => {
  const d = row.actualQty - row.bookQty
  if (d > 0) return 'gain'
  if (d < 0) return 'loss'
  return 'muted'
}

const load = async () => {
  loading.value = true
  try {
    const res = await pageCheck({ page: 1, pageSize: 30 })
    list.value = res.code === 0 ? res.data.list || [] : []
  } finally {
    loading.value = false
  }
}

const createNew = async () => {
  const res = await createCheck({ warehouseId: warehouseId.value })
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success(`已发起盘点 ${res.data.checkNo}`)
  await load()
  open(res.data)
}

const open = async (row) => {
  const res = await getCheck(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  // 未录入的实盘数显示为账面数（省得每条都要手打），但保持 null 语义需要用户显式保存
  current.value = res.data
  current.value.items = (res.data.items || []).map((i) => ({
    ...i,
    unitCost: Number(i.unitCost || 0),
    actualQty: i.actualQty === null || i.actualQty === undefined ? i.bookQty : i.actualQty
  }))
  drawer.value = true
}

const saveItems = async () => {
  const items = current.value.items
    .filter((i) => i.actualQty !== null && i.actualQty !== undefined)
    .map((i) => ({
      id: i.ID,
      actualQty: Number(i.actualQty),
      ...(i.actualQty > i.bookQty ? { unitCost: Number(i.unitCost || 0) } : {})
    }))
  if (items.length === 0) return ElMessage.warning('请先录入实盘数')
  const res = await updateCheckItems(current.value.ID, items)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('已保存')
  await load()
}

const finish = async (row) => {
  await ElMessageBox.confirm('完成后差异会写入账本，盘点单不可再改。确认完成？', '完成盘点', { type: 'warning' })
  const items = (current.value?.ID === row.ID ? current.value.items : row.items || [])
    .filter((i) => i.actualQty !== null && i.actualQty !== undefined)
    .map((i) => ({
      id: i.ID,
      actualQty: Number(i.actualQty),
      ...(i.actualQty > i.bookQty ? { unitCost: Number(i.unitCost || 0) } : {})
    }))
  if (items.length) {
    const saved = await updateCheckItems(row.ID, items)
    if (saved.code !== 0) return ElMessage.error(saved.msg)
  }
  const res = await postCheck(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('盘点已完成，差异已记账')
  drawer.value = false
  await load()
}

const cancel = async (row) => {
  await ElMessageBox.confirm('确认取消该盘点单？', '提示', { type: 'warning' })
  const res = await cancelCheck(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  ElMessage.success('已取消')
  await load()
}

onMounted(async () => {
  const w = await listWarehouses()
  if (w.code === 0) {
    warehouses.value = w.data || []
    if (warehouses.value.length) warehouseId.value = warehouses.value[0].ID
  }
  await load()
})
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 10px;
}
.spacer {
  flex: 1;
}
.hint {
  background: var(--el-fill-color-light);
  border-radius: 6px;
  padding: 8px 10px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  margin-bottom: 10px;
}
.muted {
  color: var(--el-text-color-placeholder);
}
.gain {
  color: var(--el-color-success);
}
.loss {
  color: var(--el-color-danger);
}
.tag {
  font-size: 12px;
  padding: 1px 8px;
  border-radius: 10px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.tag.ok {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}
.tag.warn {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}
</style>

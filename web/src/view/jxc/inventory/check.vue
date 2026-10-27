<template>
  <div class="flex h-full min-h-0 flex-col">
    <div class="flex flex-wrap items-center gap-2 border-b border-border px-4 py-2.5">
      <div class="w-[168px] shrink-0">
        <el-select v-model="warehouseId" class="w-full" placeholder="选择仓库">
          <el-option v-for="warehouse in warehouses" :key="warehouse.ID" :label="warehouse.name" :value="warehouse.ID" />
        </el-select>
      </div>
      <el-button type="primary" :disabled="!warehouseId" @click="createNew">
        <svg-icon icon="lucide:clipboard-plus" class="mr-1.5 h-3.5 w-3.5" />
        发起盘点
      </el-button>
      <span class="text-[11px] text-muted-foreground">同一仓库同时只能有一张进行中的盘点</span>
      <el-button link class="ml-auto" @click="load">刷新</el-button>
    </div>

    <div v-loading="loading" class="min-h-0 flex-1 overflow-hidden">
      <el-table :data="list" height="100%" class="el-table--clickable" @row-click="open">
        <el-table-column label="盘点单号" width="160">
          <template #default="{ row }">
            <el-button link type="primary" class="font-mono text-[12px]" @click.stop="open(row)">{{ row.checkNo }}</el-button>
          </template>
        </el-table-column>
        <el-table-column label="仓库" width="110">
          <template #default="{ row }">{{ warehouseName(row.warehouseId) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }"><StatusTag kind="check" :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="录入进度" width="150">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-progress :percentage="progressOf(row)" :stroke-width="4" :show-text="false" class="w-[86px]" />
              <span class="text-[11px] tabular-nums text-muted-foreground">{{ counted(row) }}/{{ row.items?.length || 0 }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="差异项" width="72" align="right">
          <template #default="{ row }">
            <span class="tabular-nums" :class="diffCount(row) > 0 ? 'font-medium text-warning-700' : 'text-muted-foreground'">{{ diffCount(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="136">
          <template #default="{ row }"><span class="text-[12px] tabular-nums text-muted-foreground">{{ formatDateTime(row.CreatedAt) }}</span></template>
        </el-table-column>
        <el-table-column label="完成时间" width="136">
          <template #default="{ row }"><span class="text-[12px] tabular-nums text-muted-foreground">{{ formatDateTime(row.postedAt) || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="操作" min-width="150" fixed="right">
          <template #default="{ row }">
            <div class="flex items-center gap-2">
              <el-button link type="primary" @click.stop="open(row)">{{ row.status === 1 ? '录入' : '查看' }}</el-button>
              <el-button v-if="row.status === 1" link type="success" @click.stop="finish(row)">完成</el-button>
              <el-button v-if="row.status === 1" link type="danger" @click.stop="cancel(row)">取消</el-button>
            </div>
          </template>
        </el-table-column>
        <template #empty>
          <EmptyState
            icon="lucide:clipboard-check"
            title="还没有盘点单"
            description="选一个仓库发起盘点。建单时会快照账面数，录入实盘后差异自动记账。"
          />
        </template>
      </el-table>
    </div>

    <el-drawer v-model="drawer" :title="current?.checkNo || '盘点单'" size="900px" destroy-on-close>
      <template v-if="current">
        <div class="rounded-lg border border-border bg-muted px-4 py-3 text-[12px] leading-5 text-muted-foreground">
          账面数在建单时已快照。录入实盘数后点「完成盘点」，差异会生成盘盈/盘亏账本行，
          <b class="font-medium text-base-text">不会直接改库存</b>。盘盈行必须填写成本单价，否则库存金额会一直为 0。
        </div>

        <div class="mt-4 grid grid-cols-2 gap-x-8 gap-y-2 text-[12px]">
          <div class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">仓库</span><span>{{ warehouseName(current.warehouseId) }}</span></div>
          <div class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">状态</span><StatusTag kind="check" :status="current.status" size="sm" /></div>
          <div class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">创建时间</span><span class="tabular-nums">{{ formatDateTime(current.CreatedAt) }}</span></div>
          <div class="flex gap-3"><span class="w-16 shrink-0 text-muted-foreground">完成时间</span><span class="tabular-nums">{{ formatDateTime(current.postedAt) || '—' }}</span></div>
        </div>

        <div class="mt-4 flex flex-wrap items-center gap-3 rounded-lg border border-border px-4 py-3">
          <div class="text-[12px] text-muted-foreground">录入 <b class="text-base-text tabular-nums">{{ counted(current) }}/{{ current.items?.length || 0 }}</b></div>
          <el-progress :percentage="progressOf(current)" :stroke-width="6" :show-text="false" class="w-[180px]" />
          <div class="text-[12px] text-muted-foreground">差异 <b class="tabular-nums" :class="diffCount(current) > 0 ? 'text-warning-700' : 'text-base-text'">{{ diffCount(current) }}</b> 项</div>
          <div v-if="current.status === 1" class="ml-auto flex items-center gap-2">
            <el-button size="small" @click="fillBook">全部按账面填充</el-button>
          </div>
        </div>
        <div class="mt-1.5 text-[10.5px] text-muted-foreground">未录入的行先显示账面数作为参考，编辑后才计入「录入」进度</div>

        <el-table :data="current.items || []" size="small" max-height="440" class="mt-3">
          <el-table-column label="商品 / 规格" min-width="190">
            <template #default="{ row }">
              <div class="text-[12.5px]">{{ skuInfo(row.skuId).goodsName }}</div>
              <div class="mt-0.5 text-[11px] text-muted-foreground">{{ skuInfo(row.skuId).color }}/{{ skuInfo(row.skuId).size }} · {{ skuInfo(row.skuId).skuCode }}</div>
            </template>
          </el-table-column>
          <el-table-column label="账面" width="76" align="right">
            <template #default="{ row }"><span class="tabular-nums">{{ row.bookQty }}</span></template>
          </el-table-column>
          <el-table-column label="实盘" width="128">
            <template #default="{ row }">
              <el-input-number
                :model-value="row.actualQty ?? row.bookQty"
                :min="0"
                :precision="0"
                :controls="false"
                :disabled="current.status !== 1"
                class="w-full"
                @update:model-value="(value) => (row.actualQty = value)"
              />
            </template>
          </el-table-column>
          <el-table-column label="差异" width="82" align="right">
            <template #default="{ row }">
              <span v-if="row.actualQty === null || row.actualQty === undefined" class="text-muted-foreground">—</span>
              <span v-else class="font-medium tabular-nums" :class="diffTone(row)">{{ diffText(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="成本单价" width="128" align="right">
            <template #default="{ row }">
              <el-input-number
                v-if="current.status === 1 && row.actualQty !== null && row.actualQty !== undefined && Number(row.actualQty) > Number(row.bookQty)"
                v-model="row.unitCost"
                :min="0"
                :precision="2"
                :controls="false"
                class="w-full"
              />
              <MoneyText v-else :value="row.unitCost" size="sm" />
            </template>
          </el-table-column>
        </el-table>
      </template>

      <template #footer>
        <div class="flex items-center justify-end gap-2">
          <el-button @click="drawer = false">关闭</el-button>
          <el-button v-if="current?.status === 1" @click="saveItems">保存实盘</el-button>
          <el-button v-if="current?.status === 1" type="primary" @click="finish(current)">完成盘点</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import EmptyState from '@/components/jxc/EmptyState.vue'
import MoneyText from '@/components/jxc/MoneyText.vue'
import StatusTag from '@/components/jxc/StatusTag.vue'
import {
  cancelCheck, createCheck, getCheck, listGoods, listWarehouses, pageCheck, postCheck, updateCheckItems
} from '@/api/jxc'
import { formatDateTime } from '@/view/jxc/shared/format'

const warehouses = ref([])
const warehouseId = ref(null)
const list = ref([])
const loading = ref(false)
const drawer = ref(false)
const current = ref(null)
const skuMap = ref({})

const warehouseName = (id) => warehouses.value.find((item) => item.ID === Number(id))?.name || '—'
const skuInfo = (id) => skuMap.value[String(id)] || { goodsName: '未知规格', color: '', size: '', skuCode: '' }

const counted = (row) => (row?.items || []).filter((item) => item.actualQty !== null && item.actualQty !== undefined).length
const diffCount = (row) => (row?.items || []).filter((item) =>
  item.actualQty !== null && item.actualQty !== undefined && Number(item.actualQty) !== Number(item.bookQty)
).length
const progressOf = (row) => {
  const total = row?.items?.length || 0
  return total ? Math.round((counted(row) / total) * 100) : 0
}
const diffText = (row) => {
  const diff = Number(row.actualQty) - Number(row.bookQty)
  return diff > 0 ? `+${diff}` : String(diff)
}
const diffTone = (row) => {
  const diff = Number(row.actualQty) - Number(row.bookQty)
  if (diff > 0) return 'text-success'
  if (diff < 0) return 'text-error'
  return 'text-muted-foreground'
}

const load = async () => {
  loading.value = true
  try {
    const res = await pageCheck({ page: 1, pageSize: 50 })
    const rows = res.code === 0 ? res.data.list || [] : []
    // 进行中的盘点排最前：这是唯一还需要人动手的状态，不能埋在历史记录里。
    list.value = [...rows].sort((a, b) => {
      const weight = (row) => (row.status === 1 ? 0 : 1)
      return weight(a) - weight(b) || new Date(b.CreatedAt || 0) - new Date(a.CreatedAt || 0)
    })
  } finally {
    loading.value = false
  }
}

// 不把未录入的行预填成账面数：输入框可以显示账面数作为占位，但 actualQty 保持 null，
// 这样「已录/差异」统计与保存语义都不会被 UI 便利性污染。
const normalizeItems = (items) => (items || []).map((item) => ({
  ...item,
  unitCost: Number(item.unitCost || 0),
  actualQty: item.actualQty === null || item.actualQty === undefined ? null : item.actualQty
}))

const open = async (row) => {
  const res = await getCheck(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '加载盘点单失败')
  current.value = { ...res.data, items: normalizeItems(res.data.items) }
  drawer.value = true
}

const createNew = async () => {
  const res = await createCheck({ warehouseId: warehouseId.value })
  if (res.code !== 0) return ElMessage.error(res.msg || '发起盘点失败')
  ElMessage.success(`已发起盘点 ${res.data.checkNo}`)
  await load()
  await open(res.data)
}

const fillBook = () => {
  for (const item of current.value.items || []) item.actualQty = item.bookQty
  ElMessage.success('已按账面数填充，确认后请点「保存实盘」')
}

const itemsPayload = (items) => (items || [])
  .filter((item) => item.actualQty !== null && item.actualQty !== undefined)
  .map((item) => ({
    id: item.ID,
    actualQty: Number(item.actualQty),
    ...(Number(item.actualQty) > Number(item.bookQty) ? { unitCost: Number(item.unitCost || 0) } : {})
  }))

const saveItems = async () => {
  const items = itemsPayload(current.value?.items)
  if (!items.length) return ElMessage.warning('请先录入实盘数')
  const res = await updateCheckItems(current.value.ID, items)
  if (res.code !== 0) return ElMessage.error(res.msg || '保存失败')
  ElMessage.success('已保存实盘数')
  await load()
  await open({ ID: current.value.ID })
}

const finish = async (row) => {
  try {
    await ElMessageBox.confirm('完成后差异会写入账本，盘点单不可再改。确认完成？', '完成盘点', { type: 'warning' })
  } catch { return }

  const sourceItems = current.value?.ID === row.ID ? current.value.items : normalizeItems(row.items)
  const gainWithoutCost = sourceItems.filter((item) =>
    item.actualQty !== null && item.actualQty !== undefined &&
    Number(item.actualQty) > Number(item.bookQty) && Number(item.unitCost || 0) <= 0
  )
  if (gainWithoutCost.length) {
    try {
      await ElMessageBox.confirm(
        `有 ${gainWithoutCost.length} 行盘盈成本为 0，过账后库存金额会记为 0。仍要完成吗？`,
        '盘盈成本为 0',
        { type: 'warning', confirmButtonText: '仍然完成' }
      )
    } catch { return }
  }

  const items = itemsPayload(sourceItems)
  if (items.length) {
    const saved = await updateCheckItems(row.ID, items)
    if (saved.code !== 0) return ElMessage.error(saved.msg || '保存实盘失败')
  }
  const res = await postCheck(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '完成盘点失败')
  ElMessage.success('盘点已完成，差异已记账')
  drawer.value = false
  await load()
}

const cancel = async (row) => {
  try {
    await ElMessageBox.confirm('确认取消该盘点单？', '取消盘点', { type: 'warning' })
  } catch { return }
  const res = await cancelCheck(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '取消失败')
  ElMessage.success('已取消')
  await load()
}

onMounted(async () => {
  const [warehouseRes, goodsRes] = await Promise.all([listWarehouses(), listGoods()])
  if (warehouseRes.code === 0) {
    warehouses.value = warehouseRes.data || []
    if (warehouses.value.length) warehouseId.value = warehouses.value[0].ID
  }
  if (goodsRes.code === 0) {
    const map = {}
    for (const product of goodsRes.data || []) {
      for (const sku of product.skus || []) {
        map[String(sku.ID)] = { ...sku, goodsName: product.name }
      }
    }
    skuMap.value = map
  }
  await load()
})
</script>

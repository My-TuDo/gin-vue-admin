<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <PageHeader
      eyebrow="INVENTORY"
      title="库存"
      description="库存的真相是账本：结存、预留、可售与均价都是账本算出来的视图。"
    >
      <template #actions>
        <el-button :loading="reconing" @click="doReconcile">
          <svg-icon icon="lucide:shield-check" class="mr-1.5 h-3.5 w-3.5" />
          一键对账
        </el-button>
        <el-button @click="load">
          <svg-icon icon="lucide:refresh-cw" class="mr-1.5 h-3.5 w-3.5" />
          刷新
        </el-button>
      </template>
    </PageHeader>

    <div class="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border md:grid-cols-4">
      <div class="bg-container px-4 py-3">
        <div class="text-[11px] text-muted-foreground">有结存 SKU</div>
        <div class="mt-1 text-[20px] font-semibold leading-none tracking-tight tabular-nums">{{ summary.skuCount }}</div>
        <div class="mt-1.5 text-[11px] text-muted-foreground">共 {{ balances.length }} 条余额记录</div>
      </div>
      <div class="bg-container px-4 py-3">
        <div class="text-[11px] text-muted-foreground">结存件数</div>
        <div class="mt-1 text-[20px] font-semibold leading-none tracking-tight tabular-nums">{{ formatQty(summary.qty) }}</div>
        <div class="mt-1.5 text-[11px] text-muted-foreground">所有仓库合计</div>
      </div>
      <div class="bg-container px-4 py-3">
        <div class="text-[11px] text-muted-foreground">库存金额</div>
        <MoneyText :value="summary.value" size="lg" strong class="mt-1" />
        <div class="mt-1.5 text-[11px] text-muted-foreground">按移动加权平均成本</div>
      </div>
      <div class="bg-container px-4 py-3">
        <div class="text-[11px] text-muted-foreground">预留 / 可售</div>
        <div class="mt-1 flex items-baseline gap-1.5">
          <span class="text-[20px] font-semibold leading-none tracking-tight tabular-nums">{{ formatQty(summary.reserved) }}</span>
          <span class="text-[11px] text-muted-foreground">/</span>
          <span class="text-[20px] font-semibold leading-none tracking-tight tabular-nums" :class="summary.available < 0 ? 'text-error' : ''">{{ formatQty(summary.available) }}</span>
        </div>
        <div class="mt-1.5 text-[11px] text-muted-foreground">可售为负说明卖超了</div>
      </div>
    </div>

    <SectionCard :padded="false" class="min-h-0 flex-1">
      <template #header>
        <div class="flex items-center gap-1">
          <button
            v-for="item in TABS"
            :key="item.key"
            type="button"
            class="flex h-8 items-center gap-1.5 rounded-md px-3 text-[12.5px] transition-colors"
            :class="tab === item.key ? 'bg-primary-50 font-medium text-primary-700' : 'text-muted-foreground hover:bg-muted hover:text-base-text'"
            @click="tab = item.key"
          >
            <svg-icon :icon="item.icon" class="h-3.5 w-3.5" />
            {{ item.label }}
          </button>
        </div>
        <div class="ml-auto text-[11px] text-muted-foreground">
          {{ tab === 'ledger' ? '点行查看该 SKU 的完整流水' : '差异会生成盘盈/盘亏账本行' }}
        </div>
      </template>

      <div v-if="tab === 'ledger'" class="flex h-full min-h-0 flex-col">
        <div class="flex flex-wrap items-center gap-2 border-b border-border px-4 py-2.5">
          <div class="w-[168px] shrink-0">
            <el-select v-model="warehouseId" class="w-full" placeholder="全部仓库" clearable @change="load">
              <el-option v-for="warehouse in warehouses" :key="warehouse.ID" :label="warehouse.name" :value="warehouse.ID" />
            </el-select>
          </div>
          <div class="w-[240px] shrink-0">
            <el-input v-model="keyword" class="w-full" placeholder="搜索商品 / 条码 / 规格" clearable>
              <template #prefix><svg-icon icon="lucide:search" class="h-3.5 w-3.5" /></template>
            </el-input>
          </div>
          <button
            type="button"
            class="flex h-8 items-center gap-1.5 rounded-md border px-2.5 text-[12px] transition-colors"
            :class="showZero ? 'border-primary-200 bg-primary-50 text-primary-700' : 'border-border text-muted-foreground hover:bg-muted'"
            :aria-pressed="showZero"
            @click="showZero = !showZero"
          >
            <svg-icon icon="lucide:layers" class="h-3.5 w-3.5" />
            含零结存
          </button>
          <div class="ml-auto flex items-center gap-3 text-[11px] text-muted-foreground">
            <span>共 {{ filteredRows.length }} 个 SKU</span>
            <span v-if="negativeCount" class="flex items-center gap-1 text-error">
              <svg-icon icon="lucide:triangle-alert" class="h-3.5 w-3.5" />
              {{ negativeCount }} 个可售为负
            </span>
          </div>
        </div>

        <div v-loading="loading" class="min-h-0 flex-1 overflow-hidden">
          <el-table :data="filteredRows" height="100%" class="el-table--clickable" @row-click="openMoves">
            <el-table-column label="商品" min-width="180">
              <template #default="{ row }">
                <div class="flex min-w-0 items-baseline gap-2">
                  <span class="truncate text-[12.5px] font-medium text-base-text">{{ row.goodsName }}</span>
                  <span class="shrink-0 font-mono text-[10.5px] text-muted-foreground">{{ row.skuCode }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="规格" width="92">
              <template #default="{ row }">{{ row.color }}/{{ row.size }}</template>
            </el-table-column>
            <el-table-column label="仓库" width="108">
              <template #default="{ row }"><span class="text-[12px] text-muted-foreground">{{ row.warehouseName }}</span></template>
            </el-table-column>
            <el-table-column label="结存" width="80" align="right">
              <template #default="{ row }">
                <span class="tabular-nums" :class="row.qty === 0 ? 'text-muted-foreground' : 'font-medium'">{{ row.qty }}</span>
              </template>
            </el-table-column>
            <el-table-column label="预留" width="72" align="right">
              <template #default="{ row }">
                <span class="tabular-nums" :class="row.reserved === 0 ? 'text-muted-foreground' : ''">{{ row.reserved }}</span>
              </template>
            </el-table-column>
            <el-table-column label="可售" width="80" align="right">
              <template #default="{ row }">
                <span class="tabular-nums font-medium" :class="row.available < 0 ? 'text-error' : row.available === 0 ? 'text-muted-foreground' : 'text-base-text'">{{ row.available }}</span>
              </template>
            </el-table-column>
            <el-table-column label="均价" width="94" align="right">
              <template #default="{ row }">
                <MoneyText v-if="row.qty > 0" :value="row.avgCost" size="sm" />
                <el-tooltip v-else content="没有结存，暂无成本" placement="top">
                  <span class="text-muted-foreground">—</span>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column label="结存金额" width="112" align="right">
              <template #default="{ row }"><MoneyText :value="row.value" size="sm" /></template>
            </el-table-column>
            <el-table-column label="" width="72" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click.stop="openMoves(row)">流水</el-button>
              </template>
            </el-table-column>

            <template #empty>
              <EmptyState
                icon="lucide:warehouse"
                title="没有可显示的库存"
                :description="keyword ? '换个商品名或条码试试。' : '先补货入库，或在筛选里打开「含零结存」。'"
              />
            </template>
          </el-table>
        </div>
      </div>

      <StockCheck v-else />
    </SectionCard>

    <!-- 流水抽屉 -->
    <el-drawer v-model="drawerOpen" :title="drawerTitle" size="860px" destroy-on-close>
      <template v-if="currentRow">
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-[12px] text-muted-foreground">
          <span class="font-mono">{{ currentRow.skuCode }}</span>
          <span>{{ currentRow.warehouseName }}</span>
          <span>账本只追加，历史行不会改</span>
        </div>

        <div class="mt-4 grid grid-cols-5 gap-px overflow-hidden rounded-lg border border-border bg-border">
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">结存</div>
            <div class="mt-1 text-[17px] font-semibold tabular-nums">{{ currentRow.qty }}</div>
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">预留</div>
            <div class="mt-1 text-[17px] font-semibold tabular-nums">{{ currentRow.reserved }}</div>
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">可售</div>
            <div class="mt-1 text-[17px] font-semibold tabular-nums" :class="currentRow.available < 0 ? 'text-error' : ''">{{ currentRow.available }}</div>
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">均价</div>
            <MoneyText :value="currentRow.qty > 0 ? currentRow.avgCost : null" size="lg" class="mt-0.5" />
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">结存金额</div>
            <MoneyText :value="currentRow.value" size="lg" class="mt-0.5" />
          </div>
        </div>

        <div class="mt-5 flex items-center justify-between">
          <h3 class="text-[13px] font-semibold">库存流水</h3>
          <span class="text-[11px] text-muted-foreground">共 {{ moveTotal }} 条</span>
        </div>
        <el-table v-loading="loadingMoves" :data="moves" size="small" max-height="470" class="mt-2">
          <el-table-column label="时间" width="138">
            <template #default="{ row }">
              <span class="text-[11.5px] tabular-nums text-muted-foreground">{{ formatDateTime(row.occurredAt) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="单据" width="152">
            <template #default="{ row }">
              <span class="font-mono text-[11.5px]">{{ row.sourceNo || sourceLabel(row.sourceType) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="类型" width="92">
            <template #default="{ row }"><span class="text-[11.5px] text-muted-foreground">{{ sourceLabel(row.sourceType) }}</span></template>
          </el-table-column>
          <el-table-column label="方向" width="58">
            <template #default="{ row }">
              <span
                class="inline-flex rounded border px-1.5 py-0.5 text-[10px] font-medium"
                :class="row.qty > 0 ? 'border-success-200 bg-success-50 text-success-700' : 'border-warning-200 bg-warning-50 text-warning-700'"
              >{{ row.qty > 0 ? '入' : '出' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="数量" width="64" align="right">
            <template #default="{ row }"><span class="tabular-nums">{{ Math.abs(row.qty) }}</span></template>
          </el-table-column>
          <el-table-column label="成本单价" width="92" align="right">
            <template #default="{ row }"><MoneyText :value="row.unitCost" size="xs" /></template>
          </el-table-column>
          <el-table-column label="成本金额" width="100" align="right">
            <template #default="{ row }">
              <el-tooltip placement="top" :content="row.qty > 0 ? '入库成本（采购价）' : '出库结转成本（出库时冻结）'">
                <span><MoneyText :value="row.cost" size="xs" /></span>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="结存" width="64" align="right">
            <template #default="{ row }"><span class="tabular-nums">{{ row.balanceAfter ?? '—' }}</span></template>
          </el-table-column>
          <el-table-column label="操作人" min-width="72">
            <template #default="{ row }"><span class="text-[11px] text-muted-foreground">{{ row.operator || '—' }}</span></template>
          </el-table-column>
          <template #empty>
            <EmptyState icon="lucide:history" title="还没有流水" description="该规格的第一笔出入库会立即出现在这里。" />
          </template>
        </el-table>
        <div v-if="moves.length < moveTotal" class="mt-3 flex justify-center">
          <el-button :loading="loadingMore" size="small" @click="loadMoreMoves">
            加载更多（{{ moves.length }} / {{ moveTotal }}）
          </el-button>
        </div>
      </template>
    </el-drawer>

    <!-- 对账结果 -->
    <el-dialog v-model="reconDialog" title="对账结果" width="760px">
      <div v-if="reconResult.ok" class="flex flex-col items-center py-6 text-center">
        <div class="grid h-12 w-12 place-items-center rounded-full bg-success-50 text-success">
          <svg-icon icon="lucide:circle-check" class="h-6 w-6" />
        </div>
        <div class="mt-3 text-[18px] font-semibold">账实相符</div>
        <p class="mt-1 max-w-[420px] text-[12px] leading-5 text-muted-foreground">
          共核对 {{ balances.length }} 条余额记录，结存数量与金额都等于账本累加。
        </p>
        <p class="mt-1 text-[11px] text-muted-foreground">
          检查时间 <span class="tabular-nums">{{ formatDateTime(reconCheckedAt) }}</span>
        </p>
      </div>
      <div v-else>
        <el-alert
          type="error"
          :closable="false"
          show-icon
          title="余额与账本不一致"
          description="常见原因：有人直接改过库存表，或某次写入中途失败。历史数据不会被自动修正，请按差额补一条调整单并联系维护者。"
        />
        <el-table :data="reconResult.violations || []" size="small" max-height="360" class="mt-3">
          <el-table-column label="类型" width="110">
            <template #default="{ row }">{{ kindLabel(row.kind) }}</template>
          </el-table-column>
          <el-table-column label="仓库" width="90">
            <template #default="{ row }">{{ warehouseName(row.warehouseId) }}</template>
          </el-table-column>
          <el-table-column label="SKU" min-width="150">
            <template #default="{ row }">{{ skuDisplay(row.skuId) }}</template>
          </el-table-column>
          <el-table-column prop="expected" label="账本" width="90" align="right" />
          <el-table-column prop="actual" label="余额" width="90" align="right" />
          <el-table-column prop="detail" label="说明" min-width="180" show-overflow-tooltip />
        </el-table>
      </div>
      <template #footer>
        <el-button @click="reconDialog = false">关闭</el-button>
        <el-button type="primary" :loading="reconing" @click="doReconcile">重新检查</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import PageHeader from '@/components/jxc/PageHeader.vue'
import SectionCard from '@/components/jxc/SectionCard.vue'
import EmptyState from '@/components/jxc/EmptyState.vue'
import MoneyText from '@/components/jxc/MoneyText.vue'
import StockCheck from './check.vue'
import { ledgerBalances, ledgerMoves, ledgerReconcile, listWarehouses } from '@/api/jxc'
import { formatDateTime, formatQty, sourceLabel } from '@/view/jxc/shared/format'

const route = useRoute()
const TABS = [
  { key: 'ledger', label: '库存台账', icon: 'lucide:table-2' },
  { key: 'check', label: '盘点', icon: 'lucide:clipboard-check' }
]
const tab = ref(route.query.tab === 'check' ? 'check' : 'ledger')

const warehouses = ref([])
const warehouseId = ref(null)
const balances = ref([])
const keyword = ref('')
const showZero = ref(false)
const loading = ref(false)

const summary = reactive({ skuCount: 0, qty: 0, value: 0, reserved: 0, available: 0 })
const negativeCount = computed(() => balances.value.filter((row) => Number(row.available) < 0).length)

const filteredRows = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  return balances.value.filter((row) => {
    if (!showZero.value && Number(row.qty) === 0) return false
    if (!k) return true
    return [row.goodsName, row.skuCode, row.color, row.size, row.warehouseName]
      .some((value) => String(value ?? '').toLowerCase().includes(k))
  })
})

const recomputeSummary = () => {
  summary.skuCount = balances.value.filter((row) => Number(row.qty) > 0).length
  summary.qty = balances.value.reduce((sum, row) => sum + Number(row.qty || 0), 0)
  summary.value = balances.value.reduce((sum, row) => sum + Number(row.value || 0), 0)
  summary.reserved = balances.value.reduce((sum, row) => sum + Number(row.reserved || 0), 0)
  summary.available = balances.value.reduce((sum, row) => sum + Number(row.available || 0), 0)
}

const load = async () => {
  loading.value = true
  try {
    const res = await ledgerBalances({ warehouseId: warehouseId.value || 0 })
    if (res.code !== 0) return ElMessage.error(res.msg || '加载台账失败')
    balances.value = res.data || []
    recomputeSummary()
  } finally {
    loading.value = false
  }
}

const warehouseName = (id) => warehouses.value.find((item) => item.ID === Number(id))?.name || '—'
const skuDisplay = (id) => {
  const row = balances.value.find((item) => Number(item.skuId) === Number(id))
  if (!row) return `SKU #${id}`
  return `${row.goodsName} ${row.color}/${row.size}`
}
const KIND_LABEL = {
  qty_mismatch: '数量不一致',
  value_mismatch: '金额不一致',
  negative_available: '可售为负',
  orphan_balance: '余额无账本'
}
const kindLabel = (kind) => KIND_LABEL[kind] || kind

// ---------- 流水 ----------
const drawerOpen = ref(false)
const currentRow = ref(null)
const moves = ref([])
const loadingMoves = ref(false)
const loadingMore = ref(false)
const movePage = ref(1)
const moveTotal = ref(0)
const drawerTitle = computed(() => {
  if (!currentRow.value) return '库存流水'
  return `${currentRow.value.goodsName} · ${currentRow.value.color}/${currentRow.value.size}`
})

const fetchMoves = async (page) => {
  const row = currentRow.value
  const res = await ledgerMoves({ warehouseId: row.warehouseId, skuId: row.skuId, page, pageSize: 200 })
  if (res.code !== 0) {
    ElMessage.error(res.msg || '加载流水失败')
    return null
  }
  moveTotal.value = Number(res.data.total || 0)
  return res.data.list || []
}

const openMoves = async (row) => {
  currentRow.value = row
  drawerOpen.value = true
  loadingMoves.value = true
  movePage.value = 1
  moves.value = []
  try {
    moves.value = (await fetchMoves(1)) || []
  } finally {
    loadingMoves.value = false
  }
}

const loadMoreMoves = async () => {
  if (loadingMore.value) return
  loadingMore.value = true
  try {
    const next = await fetchMoves(movePage.value + 1)
    if (next) {
      moves.value = moves.value.concat(next)
      movePage.value += 1
    }
  } finally {
    loadingMore.value = false
  }
}

// ---------- 对账 ----------
const reconDialog = ref(false)
const reconing = ref(false)
const reconResult = ref({ ok: true, violations: [] })
const reconCheckedAt = ref('')

const doReconcile = async () => {
  reconing.value = true
  try {
    const res = await ledgerReconcile({ warehouseId: warehouseId.value || 0 })
    if (res.code !== 0) return ElMessage.error(res.msg || '对账失败')
    const data = res.data || { ok: true, violations: [] }
    reconResult.value = {
      ...data,
      violations: (data.violations || []).map((item) => ({
        ...item,
        kind: item.kind ?? item.Kind,
        warehouseId: item.warehouseId ?? item.WarehouseID,
        skuId: item.skuId ?? item.SkuID,
        expected: item.expected ?? item.Expected,
        actual: item.actual ?? item.Actual,
        detail: item.detail ?? item.Detail
      }))
    }
    reconCheckedAt.value = new Date().toISOString()
    reconDialog.value = true
    window.dispatchEvent(new CustomEvent('jxc:reconcile', {
      detail: {
        ok: reconResult.value.ok,
        count: reconResult.value.count || reconResult.value.violations?.length || 0,
        message: reconResult.value.ok ? '余额与账本累加一致' : '余额与账本累加不一致'
      }
    }))
  } finally {
    reconing.value = false
  }
}

// 从命令面板/顶栏对账状态跳进来时，可能已经停在库存页：query 变化也要生效。
watch(() => route.query.tab, (value) => {
  tab.value = value === 'check' ? 'check' : 'ledger'
})

watch(() => route.query.reconcile, (value) => {
  if (value === '1') doReconcile()
})

onMounted(async () => {
  const res = await listWarehouses()
  if (res.code === 0) warehouses.value = res.data || []
  await load()
  if (route.query.tab === 'check') tab.value = 'check'
  if (route.query.reconcile === '1') await doReconcile()
})
</script>

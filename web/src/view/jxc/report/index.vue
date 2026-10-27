<template>
  <div class="flex h-full min-h-0 flex-col gap-4 overflow-y-auto pr-1">
    <PageHeader
      eyebrow="REPORTS"
      title="报表"
      description="月度经营汇总与会计期间。口径写在页面上，避免每个人算出来的毛利不一样。"
    >
      <template #actions>
        <el-date-picker
          v-model="month"
          type="month"
          value-format="YYYY-MM"
          :clearable="false"
          class="!w-[148px]"
          @change="loadSummary"
        />
        <el-button :loading="closing" :disabled="currentClosed" @click="doClose">
          <svg-icon icon="lucide:lock" class="mr-1.5 h-3.5 w-3.5" />
          {{ currentClosed ? '本月已结账' : '月结' }}
        </el-button>
      </template>
    </PageHeader>

    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <MetricTile label="销售额（净）" icon="lucide:receipt">
        <MoneyText :value="summary.salesAmount" size="xl" strong />
        <template #hint>
          {{ summary.salesOrders }} 单 · 退货 {{ summary.returnOrders }} 单冲减
          <MoneyText :value="summary.returnAmount" size="xs" muted />
        </template>
      </MetricTile>
      <MetricTile label="销售成本（净）" icon="lucide:layers">
        <MoneyText :value="summary.salesCost" size="xl" strong />
        <template #hint>
          退货成本 <MoneyText :value="summary.returnCost" size="xs" muted /> · 出库时冻结
        </template>
      </MetricTile>
      <MetricTile label="毛利" icon="lucide:trending-up">
        <MoneyText :value="summary.gross" size="xl" strong />
        <template #hint>毛利率 {{ marginText }} · 销售额 − 销售成本</template>
      </MetricTile>
      <MetricTile label="期末库存" icon="lucide:warehouse">
        <MoneyText :value="summary.stockValue" size="xl" strong />
        <template #hint>当前结存；已结账月份以快照为准</template>
      </MetricTile>
    </div>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.55fr)_minmax(320px,1fr)]">
      <SectionCard title="经营构成" :description="`${month} · 销售额拆分`">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div>
            <div class="text-[11px] text-muted-foreground">销售额</div>
            <MoneyText :value="summary.salesAmount" size="lg" strong class="mt-0.5" />
          </div>
          <div class="flex items-center gap-4 text-[11px]">
            <span class="flex items-center gap-1.5"><i class="h-2 w-2 rounded-full bg-primary"></i>销售成本 <MoneyText :value="summary.salesCost" size="xs" /></span>
            <span class="flex items-center gap-1.5"><i class="h-2 w-2 rounded-full bg-success"></i>毛利 <MoneyText :value="summary.gross" size="xs" /></span>
          </div>
        </div>

        <div v-if="summary.salesAmount > 0" class="mt-3">
          <div class="flex h-2.5 w-full overflow-hidden rounded-full bg-muted">
            <div class="h-full bg-primary transition-[width]" :style="{ '--w': costPercent + '%' }" :class="'w-[var(--w)]'"></div>
            <div class="h-full bg-success transition-[width]" :style="{ '--w': grossPercent + '%' }" :class="'w-[var(--w)]'"></div>
          </div>
          <div class="mt-2 flex justify-between text-[11px] tabular-nums text-muted-foreground">
            <span>成本 {{ costPercent }}%</span>
            <span>毛利 {{ grossPercent }}%</span>
          </div>
        </div>
        <div v-else class="mt-3 flex h-2.5 w-full items-center rounded-full bg-muted px-3 text-[10px] text-muted-foreground">
          本月还没有已出库销售
        </div>

        <div class="mt-5 grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border sm:grid-cols-4">
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">采购金额（净）</div>
            <MoneyText :value="summary.purchaseAmount" size="lg" class="mt-0.5" />
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">采购退货</div>
            <MoneyText :value="summary.purchaseReturnAmount" size="lg" class="mt-0.5" />
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">销售单</div>
            <div class="mt-0.5 text-[16px] font-semibold tabular-nums">{{ summary.salesOrders }}</div>
          </div>
          <div class="bg-container px-3 py-2.5">
            <div class="text-[10.5px] text-muted-foreground">退货单</div>
            <div class="mt-0.5 text-[16px] font-semibold tabular-nums">{{ summary.returnOrders }}</div>
          </div>
        </div>
      </SectionCard>

      <SectionCard title="口径" description="这些数字是怎么算的">
        <ul class="space-y-2.5 text-[11.5px] leading-5 text-muted-foreground">
          <li class="flex gap-2"><span class="mt-2 h-1 w-1 shrink-0 rounded-full bg-primary"></span><span><b class="font-medium text-base-text">销售额</b> = 已出库且未红冲的销售 − 已入库退货</span></li>
          <li class="flex gap-2"><span class="mt-2 h-1 w-1 shrink-0 rounded-full bg-primary"></span><span><b class="font-medium text-base-text">销售成本</b> = 出库时冻结的成本 − 退回货物成本</span></li>
          <li class="flex gap-2"><span class="mt-2 h-1 w-1 shrink-0 rounded-full bg-primary"></span><span><b class="font-medium text-base-text">毛利</b> = 销售额 − 销售成本</span></li>
          <li class="flex gap-2"><span class="mt-2 h-1 w-1 shrink-0 rounded-full bg-primary"></span><span><b class="font-medium text-base-text">采购金额</b> = 已入库采购 − 采购红冲 − 采购退货；采购退货计入退货发生月</span></li>
          <li class="flex gap-2"><span class="mt-2 h-1 w-1 shrink-0 rounded-full bg-primary"></span><span><b class="font-medium text-base-text">红冲</b>计入红冲发生月，不追溯修改已结账月份</span></li>
        </ul>
      </SectionCard>
    </div>

    <SectionCard title="会计期间" description="结账后该月不能再过账；更正只能在当前月开红冲" :padded="false">
      <template #actions>
        <el-button link @click="load">刷新</el-button>
      </template>
      <el-table :data="periods" max-height="360">
        <el-table-column prop="periodMonth" label="月份" width="120">
          <template #default="{ row }"><span class="font-mono text-[12px]">{{ row.periodMonth }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><StatusTag kind="period" :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="结账时间" width="160">
          <template #default="{ row }"><span class="text-[12px] tabular-nums text-muted-foreground">{{ formatDateTime(row.closedAt) || '—' }}</span></template>
        </el-table-column>
        <el-table-column prop="closedBy" label="结账人" width="120">
          <template #default="{ row }">{{ row.closedBy || '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" min-width="120">
          <template #default="{ row }">
            <el-button v-if="row.status === 2" link type="primary" @click="viewSnapshot(row)">查看期末快照</el-button>
            <span v-else class="text-[11px] text-muted-foreground">未结账</span>
          </template>
        </el-table-column>
        <template #empty>
          <EmptyState icon="lucide:calendar-clock" title="还没有会计期间" description="第一次月结后，这里会出现该月的冻结快照。" />
        </template>
      </el-table>
    </SectionCard>

    <el-dialog v-model="snapDialog" :title="`${snapPeriod} 期末库存快照`" width="760px">
      <div class="mb-3 flex items-center justify-between text-[12px] text-muted-foreground">
        <span>快照在结账时冻结，之后不会随当前库存变化</span>
        <span>{{ snapshots.length }} 条</span>
      </div>
      <el-table :data="snapshots" size="small" max-height="420">
        <el-table-column label="仓库" width="110">
          <template #default="{ row }">{{ warehouseName(row.warehouseId) }}</template>
        </el-table-column>
        <el-table-column label="商品 / 规格" min-width="200">
          <template #default="{ row }">
            <span>{{ skuInfo(row.skuId).goodsName }}</span>
            <span class="ml-1 text-[11px] text-muted-foreground">{{ skuInfo(row.skuId).color }}/{{ skuInfo(row.skuId).size }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="qty" label="数量" width="90" align="right" />
        <el-table-column label="金额" width="120" align="right">
          <template #default="{ row }"><MoneyText :value="row.value" size="sm" /></template>
        </el-table-column>
      </el-table>
      <template #footer>
        <div class="flex items-center justify-between">
          <span class="text-[12px] text-muted-foreground">合计金额 <MoneyText :value="snapTotalValue" size="sm" strong /></span>
          <el-button @click="snapDialog = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/jxc/PageHeader.vue'
import SectionCard from '@/components/jxc/SectionCard.vue'
import MetricTile from '@/components/jxc/MetricTile.vue'
import EmptyState from '@/components/jxc/EmptyState.vue'
import MoneyText from '@/components/jxc/MoneyText.vue'
import StatusTag from '@/components/jxc/StatusTag.vue'
import { closePeriod, getMonthlySummary, getSnapshot, listGoods, listPeriods, listWarehouses } from '@/api/jxc'
import { formatDateTime, formatPercent, marginOf, round2 } from '@/view/jxc/shared/format'

const now = new Date()
const month = ref(`${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`)
const summary = reactive({
  salesAmount: 0, salesCost: 0, gross: 0, stockValue: 0,
  salesOrders: 0, returnOrders: 0, returnAmount: 0, returnCost: 0,
  purchaseAmount: 0, purchaseReturnAmount: 0
})
const periods = ref([])
const closing = ref(false)
const snapDialog = ref(false)
const snapshots = ref([])
const snapPeriod = ref('')
const skuMap = ref({})
const warehouseMap = ref({})

const marginText = computed(() => {
  const ratio = marginOf(summary.gross, summary.salesAmount)
  return ratio === null ? '—' : formatPercent(ratio)
})

const costPercent = computed(() => {
  if (summary.salesAmount <= 0) return 0
  return Math.max(0, Math.min(100, Math.round((summary.salesCost / summary.salesAmount) * 100)))
})
const grossPercent = computed(() => Math.max(0, 100 - costPercent.value))
const currentClosed = computed(() => periods.value.some((item) => item.periodMonth === month.value && item.status === 2))
const snapTotalValue = computed(() => round2(snapshots.value.reduce((sum, row) => sum + Number(row.value || 0), 0)))

const warehouseName = (id) => warehouseMap.value[String(id)]?.name || '—'
const skuInfo = (id) => skuMap.value[String(id)] || { goodsName: '未知规格', color: '', size: '' }

const loadSummary = async () => {
  const res = await getMonthlySummary(month.value)
  if (res.code !== 0) return ElMessage.error(res.msg || '加载月报失败')
  Object.assign(summary, {
    salesAmount: 0, salesCost: 0, gross: 0, stockValue: 0,
    salesOrders: 0, returnOrders: 0, returnAmount: 0, returnCost: 0,
    purchaseAmount: 0, purchaseReturnAmount: 0,
    ...(res.data || {})
  })
}

const load = async () => {
  const res = await listPeriods()
  if (res.code === 0) periods.value = res.data || []
  await loadSummary()
}

const doClose = async () => {
  if (currentClosed.value) return ElMessage.info('该月已经结账')
  try {
    await ElMessageBox.confirm(
      `结账后 ${month.value} 不能再过账，并会冻结当月库存快照。确认结账？`,
      '月结',
      { type: 'warning', confirmButtonText: '确认结账' }
    )
  } catch { return }
  closing.value = true
  try {
    const res = await closePeriod(month.value)
    if (res.code !== 0) return ElMessage.error(res.msg || '结账失败')
    ElMessage.success(`${month.value} 已结账`)
    await load()
  } finally {
    closing.value = false
  }
}

const viewSnapshot = async (row) => {
  snapPeriod.value = row.periodMonth
  const res = await getSnapshot(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg || '加载快照失败')
  snapshots.value = res.data || []
  snapDialog.value = true
}

onMounted(async () => {
  const [warehouseRes, goodsRes] = await Promise.all([listWarehouses(), listGoods()])
  if (warehouseRes.code === 0) {
    warehouseMap.value = Object.fromEntries((warehouseRes.data || []).map((item) => [String(item.ID), item]))
  }
  if (goodsRes.code === 0) {
    const map = {}
    for (const product of goodsRes.data || []) {
      for (const sku of product.skus || []) map[String(sku.ID)] = { ...sku, goodsName: product.name }
    }
    skuMap.value = map
  }
  await load()
})
</script>

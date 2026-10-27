<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <PageHeader
      eyebrow="TODAY"
      title="工作台"
      description="今天的经营数字、需要处理的单据，以及账本里最近发生的事。"
    >
      <template #actions>
        <el-button @click="router.push('/jxc/purchase')">
          <svg-icon icon="lucide:package-plus" class="mr-1.5 h-3.5 w-3.5" />
          补货
        </el-button>
        <el-button type="primary" @click="router.push('/jxc/sale')">
          <svg-icon icon="lucide:scan-line" class="mr-1.5 h-3.5 w-3.5" />
          开单
        </el-button>
      </template>
    </PageHeader>

    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <MetricTile label="库存金额" icon="lucide:wallet">
        <MoneyText :value="stats.stockValue" size="xl" strong />
        <template #hint>
          <span class="tabular-nums">{{ stats.skuCount }}</span> 个 SKU 有结存
        </template>
      </MetricTile>

      <MetricTile label="今日销售额（净）" icon="lucide:receipt">
        <MoneyText :value="stats.todayAmount" size="xl" strong />
        <template #hint>
          {{ ordersHint }} · 退货 {{ stats.todayReturns }} 单冲减
          <MoneyText :value="stats.returnAmount" size="xs" muted />
        </template>
      </MetricTile>

      <MetricTile label="今日毛利" icon="lucide:trending-up">
        <MoneyText :value="stats.todayGross" size="xl" strong />
        <template #hint>
          毛利率 {{ grossMarginText }} · 成本已冻结在账本行
        </template>
      </MetricTile>

      <MetricTile label="待处理" icon="lucide:list-todo" :tone="stats.pending > 0 ? 'warning' : 'default'">
        <span class="text-[26px] font-semibold leading-none tracking-tight tabular-nums" :class="stats.pending > 0 ? 'text-warning-700' : ''">
          {{ stats.pending }}
        </span>
        <template #hint>
          审核 {{ stats.toApprove }} · 入库 {{ stats.toReceive }} · 出库 {{ stats.toShip }} · 退货 {{ stats.toPostPurchaseReturn }}
        </template>
      </MetricTile>
    </div>

    <div class="grid min-h-0 flex-1 grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.7fr)_minmax(320px,1fr)]">
      <SectionCard title="最近库存流水" description="账本视角 · 只追加" :padded="false" class="min-h-0">
        <template #actions>
          <el-button link type="primary" @click="router.push('/jxc/inventory')">
            查看台账
            <svg-icon icon="lucide:arrow-up-right" class="ml-1 h-3.5 w-3.5" />
          </el-button>
        </template>
        <el-table
          v-loading="loading"
          :data="moves"
          height="100%"
          class="el-table--clickable"
          @row-click="goInventory"
        >
          <el-table-column label="时间" width="92">
            <template #default="{ row }">
              <span class="text-[12px] tabular-nums text-muted-foreground">{{ formatShortTime(row.occurredAt) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="单据" width="140">
            <template #default="{ row }">
              <span class="font-mono text-[12px]">{{ row.sourceNo || sourceLabel(row.sourceType) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="商品" min-width="160">
            <template #default="{ row }">
              <span class="text-[13px]">{{ row.goodsName }}</span>
              <span class="ml-1 text-[11px] text-muted-foreground">{{ row.color }}/{{ row.size }}</span>
            </template>
          </el-table-column>
          <el-table-column label="方向" width="52">
            <template #default="{ row }">
              <span
                class="inline-flex rounded border px-1.5 py-0.5 text-[10px] font-medium"
                :class="row.qty > 0 ? 'border-success-200 bg-success-50 text-success-700' : 'border-warning-200 bg-warning-50 text-warning-700'"
              >{{ row.qty > 0 ? '入' : '出' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="数量" width="60" align="right">
            <template #default="{ row }">
              <span class="tabular-nums">{{ Math.abs(row.qty) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="成本金额" width="100" align="right">
            <template #default="{ row }"><MoneyText :value="row.cost" size="sm" /></template>
          </el-table-column>
          <template #empty>
            <EmptyState icon="lucide:history" title="还没有库存流水" description="采购入库或销售出库后，这里会出现第一条记录。" />
          </template>
        </el-table>
      </SectionCard>

      <div class="flex min-h-0 flex-col gap-4">
        <SectionCard title="待办" description="按下一步动作分组" :padded="false">
          <div class="divide-y divide-border">
            <button
              v-for="task in tasks"
              :key="task.key"
              type="button"
              class="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-muted"
              @click="router.push(task.to)"
            >
              <span class="grid h-8 w-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
                <svg-icon :icon="task.icon" class="h-4 w-4" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-[12.5px] font-medium text-base-text">{{ task.label }}</span>
                <span class="mt-0.5 block truncate text-[11px] text-muted-foreground">{{ task.hint }}</span>
              </span>
              <span
                class="min-w-6 rounded px-1.5 py-0.5 text-center text-[12px] font-semibold tabular-nums"
                :class="task.count > 0 ? 'bg-warning-50 text-warning-700' : 'bg-muted text-muted-foreground'"
              >{{ task.count }}</span>
              <svg-icon icon="lucide:chevron-right" class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            </button>
          </div>
          <div v-if="stats.pending === 0" class="flex items-center gap-2 border-t border-border bg-muted px-4 py-2.5 text-[11.5px] text-muted-foreground">
            <svg-icon icon="lucide:circle-check" class="h-3.5 w-3.5 text-success" />
            今天没有卡在半路的单据
          </div>
        </SectionCard>

        <SectionCard title="可售偏低" :description="lowStockDescription" :padded="false" class="min-h-0 flex-1">
          <template #actions>
            <el-button link @click="router.push('/jxc/inventory')">库存</el-button>
          </template>
          <div class="h-full overflow-y-auto">
            <div v-if="stats.lowStockItems.length" class="divide-y divide-border">
              <button
                v-for="row in stats.lowStockItems"
                :key="row.warehouseId + '-' + row.skuId"
                type="button"
                class="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-muted"
                @click="router.push('/jxc/inventory')"
              >
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-[12.5px] text-base-text">{{ row.goodsName }}</span>
                  <span class="mt-0.5 block truncate text-[11px] text-muted-foreground">{{ row.color }}/{{ row.size }} · {{ row.warehouseName }}</span>
                </span>
                <span
                  class="shrink-0 text-[12px] font-medium tabular-nums"
                  :class="row.available <= 0 ? 'text-error' : 'text-warning-700'"
                >可售 {{ row.available }}</span>
              </button>
            </div>
            <div v-else class="flex h-full min-h-[96px] items-center justify-center gap-2 px-4 py-6 text-[12px] text-muted-foreground">
              <svg-icon icon="lucide:badge-check" class="h-4 w-4 text-success" />
              所有有结存的规格都高于预警线
            </div>
          </div>
        </SectionCard>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import MoneyText from '@/components/jxc/MoneyText.vue'
import MetricTile from '@/components/jxc/MetricTile.vue'
import PageHeader from '@/components/jxc/PageHeader.vue'
import SectionCard from '@/components/jxc/SectionCard.vue'
import EmptyState from '@/components/jxc/EmptyState.vue'
import { getDashboardSummary, ledgerMoves } from '@/api/jxc'
import { formatPercent, formatShortTime, marginOf, sourceLabel } from '@/view/jxc/shared/format'

const router = useRouter()
const loading = ref(false)
const moves = ref([])

const stats = reactive({
  stockValue: 0,
  skuCount: 0,
  todayAmount: 0,
  todayOrders: 0,
  reversedOrders: 0,
  todayGross: 0,
  returnAmount: 0,
  todayReturns: 0,
  pending: 0,
  toApprove: 0,
  toReceive: 0,
  toShip: 0,
  toPostPurchaseReturn: 0,
  lowStockThreshold: 10,
  lowStockCount: 0,
  lowStockItems: []
})

const grossMargin = computed(() => marginOf(stats.todayGross, stats.todayAmount))
const grossMarginText = computed(() => (grossMargin.value === null ? '—' : formatPercent(grossMargin.value)))
const ordersHint = computed(() => {
  const base = `${stats.todayOrders} 单`
  return stats.reversedOrders > 0 ? `${base}（含 ${stats.reversedOrders} 张已红冲）` : base
})
const lowStockDescription = computed(() => `可售 ≤ ${stats.lowStockThreshold} · 共 ${stats.lowStockCount} 个`)

const tasks = computed(() => [
  { key: 'approve', label: '待审核采购', hint: '审核后才会进入待入库', count: stats.toApprove, icon: 'lucide:clipboard-check', to: '/jxc/documents?tab=purchase&status=1' },
  { key: 'receive', label: '待入库采购', hint: '入库才产生账本行与成本', count: stats.toReceive, icon: 'lucide:package-check', to: '/jxc/documents?tab=purchase&status=2' },
  { key: 'ship', label: '待出库销售', hint: '草稿单已占用预留库存', count: stats.toShip, icon: 'lucide:truck', to: '/jxc/documents?tab=sale&status=1' },
  { key: 'purchase-return', label: '待出库采购退货', hint: '草稿退货单，出库才扣库存', count: stats.toPostPurchaseReturn, icon: 'lucide:package-minus', to: '/jxc/documents?tab=purchaseReturn&status=1' }
])

const goInventory = () => router.push('/jxc/inventory')

const load = async () => {
  loading.value = true
  try {
    const [dashboardRes, moveRes] = await Promise.all([
      getDashboardSummary(),
      ledgerMoves({ page: 1, pageSize: 10 })
    ])

    if (dashboardRes.code === 0) {
      const data = dashboardRes.data || {}
      const pending = data.pending || {}
      const lowStock = data.lowStock || {}

      Object.assign(stats, {
        stockValue: Number(data.stockValue) || 0,
        skuCount: Number(data.skuCount) || 0,
        todayAmount: Number(data.todaySalesAmount) || 0,
        todayOrders: Number(data.todaySalesOrders) || 0,
        reversedOrders: Number(data.todaySalesOrdersReversed) || 0,
        todayGross: Number(data.todayGross) || 0,
        returnAmount: Number(data.todayReturnAmount) || 0,
        todayReturns: Number(data.todayReturnOrders) || 0,
        pending: Number(pending.total) || 0,
        toApprove: Number(pending.approve) || 0,
        toReceive: Number(pending.receive) || 0,
        toShip: Number(pending.ship) || 0,
        toPostPurchaseReturn: Number(pending.purchaseReturnDraft) || 0,
        lowStockThreshold: Number(lowStock.threshold) || 10,
        lowStockCount: Number(lowStock.count) || 0,
        lowStockItems: Array.isArray(lowStock.items) ? lowStock.items : []
      })
    } else {
      ElMessage.error(dashboardRes.msg || '加载工作台概览失败')
    }

    if (moveRes.code === 0) moves.value = moveRes.data.list || []
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

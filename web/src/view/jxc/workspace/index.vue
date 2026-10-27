<template>
  <div class="workspace">
    <div class="stats">
      <div class="stat">
        <div class="k">库存金额</div>
        <div class="v"><MoneyText :value="stats.stockValue" /></div>
        <div class="s">{{ stats.skuCount }} 个 SKU 有结存</div>
      </div>
      <div class="stat">
        <div class="k">今日销售额</div>
        <div class="v"><MoneyText :value="stats.todayAmount" /></div>
        <div class="s">{{ stats.todayOrders }} 单</div>
      </div>
      <div class="stat">
        <div class="k">今日毛利</div>
        <div class="v"><MoneyText :value="stats.todayGross" :class="{ neg: stats.todayGross < 0 }" /></div>
        <div class="s">销售额 − 出库成本（成本已冻结在账本行）</div>
      </div>
      <div class="stat" :class="{ warn: stats.pending > 0 }">
        <div class="k">待处理</div>
        <div class="v">{{ stats.pending }}</div>
        <div class="s">待审核 {{ stats.toApprove }} · 待入库 {{ stats.toReceive }} · 待出库 {{ stats.toShip }}</div>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <span class="title">最近库存流水</span>
        <span class="muted">账本视角：每条都是不可变的事实</span>
        <div class="spacer"></div>
        <el-button link @click="$router.push('/jxc/inventory')">查看台账 →</el-button>
      </div>
      <el-table v-loading="loading" :data="moves" size="small" max-height="420">
        <el-table-column label="时间" width="140">
          <template #default="{ row }">{{ fmt(row.occurredAt) }}</template>
        </el-table-column>
        <el-table-column label="单据" width="170">
          <template #default="{ row }">{{ row.sourceNo || sourceLabel(row.sourceType) }}</template>
        </el-table-column>
        <el-table-column label="商品" min-width="160">
          <template #default="{ row }">{{ row.goodsName }} {{ row.color }}/{{ row.size }}</template>
        </el-table-column>
        <el-table-column label="方向" width="64">
          <template #default="{ row }">
            <span :class="row.qty > 0 ? 'in' : 'out'">{{ row.qty > 0 ? '入' : '出' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="数量" width="72" align="right">
          <template #default="{ row }">{{ Math.abs(row.qty) }}</template>
        </el-table-column>
        <el-table-column label="成本金额" width="110" align="right">
          <template #default="{ row }"><MoneyText :value="row.cost" /></template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import MoneyText from '@/components/jxc/MoneyText.vue'
import { ledgerBalances, ledgerMoves, pagePurchase, pageSale } from '@/api/jxc'

const loading = ref(false)
const moves = ref([])
const stats = reactive({
  stockValue: 0, skuCount: 0, todayAmount: 0, todayOrders: 0, todayGross: 0,
  pending: 0, toApprove: 0, toReceive: 0, toShip: 0
})

const SOURCE = {
  purchase_in: '采购入库', sale_out: '销售出库', sale_return: '退货入库',
  direct_in: '直接入库', direct_out: '直接出库', check_gain: '盘盈', check_loss: '盘亏', reversal: '红冲'
}
const sourceLabel = (t) => SOURCE[t] ?? t
const fmt = (s) => {
  const d = new Date(s)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}
const isToday = (s) => {
  const d = new Date(s)
  const n = new Date()
  return d.getFullYear() === n.getFullYear() && d.getMonth() === n.getMonth() && d.getDate() === n.getDate()
}

onMounted(async () => {
  loading.value = true
  try {
    const [bal, mv, sales, purchases] = await Promise.all([
      ledgerBalances({}),
      ledgerMoves({ page: 1, pageSize: 15 }),
      pageSale({ page: 1, pageSize: 100 }),
      pagePurchase({ page: 1, pageSize: 100 })
    ])

    if (bal.code === 0) {
      const rows = bal.data || []
      stats.stockValue = rows.reduce((s, r) => s + Number(r.value || 0), 0)
      stats.skuCount = rows.filter((r) => r.qty > 0).length
    }
    if (mv.code === 0) moves.value = mv.data.list || []

    if (sales.code === 0) {
      // 注意：GVA 的 GVA_MODEL 没给 CreatedAt 加 json tag，所以字段名是大写 CreatedAt。
      // 这里同时兼容两种写法，避免以后加了 tag 又静默失效。
      const created = (o) => o.postedAt || o.CreatedAt || o.createdAt
      const list = (sales.data.list || []).filter((o) => o.status === 2 && isToday(created(o)))
      stats.todayOrders = list.length
      stats.todayAmount = list.reduce((s, o) => s + Number(o.totalAmount || 0), 0)
      const cost = list.reduce((s, o) => s + Number(o.costAmount || 0), 0)
      stats.todayGross = Math.round((stats.todayAmount - cost) * 100) / 100
      stats.toShip = (sales.data.list || []).filter((o) => o.status === 1).length
    }
    if (purchases.code === 0) {
      const list = purchases.data.list || []
      stats.toApprove = list.filter((o) => o.status === 1).length
      stats.toReceive = list.filter((o) => o.status === 2).length
    }
    stats.pending = stats.toApprove + stats.toReceive + stats.toShip
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.workspace {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
.stat {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 14px 16px;
}
.stat.warn {
  border-color: var(--el-color-warning-light-7);
  background: var(--el-color-warning-light-9);
}
.k {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-bottom: 6px;
}
.v {
  font-size: 24px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.s {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  margin-top: 6px;
}
.panel {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
}
.panel-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.title {
  font-weight: 600;
}
.muted {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
.spacer {
  flex: 1;
}
.neg {
  color: var(--el-color-danger);
}
.in {
  color: var(--el-color-success);
  font-weight: 500;
}
.out {
  color: var(--el-color-warning);
  font-weight: 500;
}
</style>

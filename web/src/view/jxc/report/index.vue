<template>
  <div class="report">
    <div class="toolbar">
      <el-date-picker v-model="month" type="month" value-format="YYYY-MM" placeholder="选择月份"
        :clearable="false" style="width: 150px" @change="loadSummary" />
      <div class="spacer"></div>
      <el-button :loading="closing" @click="doClose">结账（{{ month }}）</el-button>
    </div>

    <!-- 月报：口径写在卡片下方，避免每个人算的毛利不一样 -->
    <div class="cards">
      <div class="card">
        <div class="k">销售额（净）</div>
        <div class="v"><MoneyText :value="sum.salesAmount" /></div>
        <div class="s">{{ sum.salesOrders }} 单，已扣除退货 {{ sum.returnOrders }} 单</div>
      </div>
      <div class="card">
        <div class="k">销售成本（净）</div>
        <div class="v"><MoneyText :value="sum.salesCost" /></div>
        <div class="s">出库时冻结的成本，扣掉退回部分</div>
      </div>
      <div class="card" :class="{ neg: sum.gross < 0 }">
        <div class="k">毛利</div>
        <div class="v"><MoneyText :value="sum.gross" /></div>
        <div class="s">销售额 − 销售成本</div>
      </div>
      <div class="card">
        <div class="k">期末库存</div>
        <div class="v"><MoneyText :value="sum.stockValue" /></div>
        <div class="s">当前结存金额（已结账的月份看快照）</div>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <span class="title">会计期间</span>
        <span class="muted">结账后该月不能再过账；更正只能在当前月开红冲</span>
        <div class="spacer"></div>
        <el-button link @click="load">刷新</el-button>
      </div>
      <el-table :data="periods" size="small">
        <el-table-column prop="periodMonth" label="月份" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="tag" :class="row.status === 2 ? 'ok' : 'warn'">
              {{ row.status === 2 ? '已结账' : '开放' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="结账时间" width="160">
          <template #default="{ row }">{{ fmt(row.closedAt) || '—' }}</template>
        </el-table-column>
        <el-table-column prop="closedBy" label="结账人" width="120" />
        <el-table-column label="操作" min-width="140">
          <template #default="{ row }">
            <el-button v-if="row.status === 2" link type="primary" @click="viewSnapshot(row)">查看快照</el-button>
            <span v-else class="muted">未结账</span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="snapDialog" :title="`${snapPeriod} 期末库存快照`" width="640px">
      <el-table :data="snapshots" size="small" max-height="420">
        <el-table-column prop="warehouseId" label="仓库" width="90" />
        <el-table-column prop="skuId" label="SKU" width="90" />
        <el-table-column prop="qty" label="数量" width="100" align="right" />
        <el-table-column label="金额" width="120" align="right">
          <template #default="{ row }"><MoneyText :value="row.value" /></template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import MoneyText from '@/components/jxc/MoneyText.vue'
import { closePeriod, getMonthlySummary, getSnapshot, listPeriods } from '@/api/jxc'

const now = new Date()
const month = ref(`${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`)
const sum = reactive({
  salesAmount: 0, salesCost: 0, gross: 0, stockValue: 0,
  salesOrders: 0, returnOrders: 0, returnAmount: 0, purchaseAmount: 0
})
const periods = ref([])
const closing = ref(false)
const snapDialog = ref(false)
const snapshots = ref([])
const snapPeriod = ref('')

const fmt = (s) => {
  if (!s) return ''
  const d = new Date(s)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const loadSummary = async () => {
  const res = await getMonthlySummary(month.value)
  if (res.code !== 0) return ElMessage.error(res.msg)
  Object.assign(sum, res.data)
}

const load = async () => {
  const res = await listPeriods()
  if (res.code === 0) periods.value = res.data || []
  await loadSummary()
}

const doClose = async () => {
  await ElMessageBox.confirm(
    `结账后 ${month.value} 不能再过账，且会冻结当月库存快照。确认结账？`, '月结', { type: 'warning' }
  )
  closing.value = true
  try {
    const res = await closePeriod(month.value)
    if (res.code !== 0) return ElMessage.error(res.msg)
    ElMessage.success(`${month.value} 已结账`)
    await load()
  } finally {
    closing.value = false
  }
}

const viewSnapshot = async (row) => {
  snapPeriod.value = row.periodMonth
  const res = await getSnapshot(row.ID)
  if (res.code !== 0) return ElMessage.error(res.msg)
  snapshots.value = res.data || []
  snapDialog.value = true
}

onMounted(load)
</script>

<style scoped>
.report {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
}
.spacer {
  flex: 1;
}
.cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}
.card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 14px 16px;
}
.card.neg {
  border-color: var(--el-color-danger-light-7);
}
.k {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-bottom: 6px;
}
.v {
  font-size: 22px;
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

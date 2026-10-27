<template>
  <div class="inventory">
    <el-tabs v-model="tab" class="inv-tabs">
      <el-tab-pane label="库存台账" name="ledger">
        <div class="inv-body">
    <div class="toolbar">
      <el-select v-model="warehouseId" placeholder="全部仓库" clearable style="width: 160px" @change="load">
        <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索商品 / 条码" clearable style="width: 220px" />
      <div class="spacer"></div>
      <el-button :loading="reconing" @click="doReconcile">一键对账</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <!-- 结存不是"一个数"，是结存/预留/可售/均价四个数 -->
    <el-table v-loading="loading" :data="filtered" height="calc(100vh - 48px - 32px - 52px - 40px)"
      size="small" @row-click="openMoves" class="ledger-table">
      <el-table-column prop="goodsName" label="商品" min-width="160" />
      <el-table-column label="规格" width="110">
        <template #default="{ row }">{{ row.color }}/{{ row.size }}</template>
      </el-table-column>
      <el-table-column prop="skuCode" label="条码" width="130" />
      <el-table-column label="结存" width="80" align="right">
        <template #default="{ row }">
          <span :class="{ zero: row.qty === 0 }">{{ row.qty }}</span>
        </template>
      </el-table-column>
      <el-table-column label="预留" width="80" align="right">
        <template #default="{ row }">
          <span :class="{ muted: row.reserved === 0 }">{{ row.reserved }}</span>
        </template>
      </el-table-column>
      <el-table-column label="可售" width="80" align="right">
        <template #default="{ row }">
          <!-- 可售为负才是问题（卖超），0 不是错误 -->
          <span :class="{ bad: row.available < 0, zero: row.available === 0 }">{{ row.available }}</span>
        </template>
      </el-table-column>
      <el-table-column label="均价" width="110" align="right">
        <template #default="{ row }">
          <MoneyText v-if="row.qty > 0" :value="row.avgCost" />
          <el-tooltip v-else content="没有结存，暂无成本" placement="top">
            <span class="muted">—</span>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column label="结存金额" width="120" align="right">
        <template #default="{ row }"><MoneyText :value="row.value" /></template>
      </el-table-column>
      <el-table-column label="" width="72">
        <template #default="{ row }">
          <el-button link type="primary" @click.stop="openMoves(row)">流水</el-button>
        </template>
      </el-table-column>
    </el-table>
        </div>
      </el-tab-pane>
      <el-tab-pane label="盘点" name="check" lazy>
        <StockCheck />
      </el-tab-pane>
    </el-tabs>

    <!-- 流水：账本的原始事实，只读 -->
    <el-drawer v-model="drawer" :title="drawerTitle" size="720px">
      <el-table v-loading="loadingMoves" :data="moves" size="small">
        <el-table-column label="时间" width="140">
          <template #default="{ row }">{{ fmt(row.occurredAt) }}</template>
        </el-table-column>
        <el-table-column label="单据" width="170">
          <template #default="{ row }">
            <span class="doc" :title="row.sourceType">{{ row.sourceNo || sourceLabel(row.sourceType) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="方向" width="64">
          <template #default="{ row }">
            <span :class="row.qty > 0 ? 'in' : 'out'">{{ row.qty > 0 ? '入' : '出' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="数量" width="72" align="right">
          <template #default="{ row }">{{ Math.abs(row.qty) }}</template>
        </el-table-column>
        <el-table-column label="成本单价" width="100" align="right">
          <template #default="{ row }"><MoneyText :value="row.unitCost" /></template>
        </el-table-column>
        <!-- 成本金额取绝对值：出库是"货变成了成本"，不是亏损。
             账本里 amount 有符号是为不变量服务，不该直接摆给用户看。 -->
        <el-table-column label="成本金额" width="110" align="right">
          <template #default="{ row }">
            <el-tooltip placement="top" :content="row.qty > 0 ? '入库成本（采购价）' : '出库结转成本（出库时冻结）'">
              <span><MoneyText :value="row.cost" /></span>
            </el-tooltip>
          </template>
        </el-table-column>
        <!-- 结存：这一笔之后还剩多少。只在单 SKU 视图里算，跨 SKU 混排会看不懂 -->
        <el-table-column label="结存" width="80" align="right">
          <template #default="{ row }">{{ row.balance ?? '—' }}</template>
        </el-table-column>
        <el-table-column label="操作人" min-width="80">
          <template #default="{ row }">{{ row.operator || '—' }}</template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <!-- 对账结果：要么通过，要么说清差额与可能原因 -->
    <el-dialog v-model="reconDialog" title="对账结果" width="720px">
      <div v-if="recon.ok" class="recon-ok">
        <div class="big">✅ 账实相符</div>
        <div class="muted">共核对 {{ rows.length }} 个 SKU，余额与账本累加完全一致。</div>
      </div>
      <div v-else>
        <el-alert type="error" :closable="false" show-icon
          title="余额与账本不一致"
          description="常见原因：有人直接改过库存表，或某次写入中途失败。历史数据不会被自动修正，请联系维护者按差额补一条调整单。" />
        <el-table :data="recon.violations" size="small" style="margin-top: 12px">
          <el-table-column prop="kind" label="类型" width="150" />
          <el-table-column prop="warehouseId" label="仓库" width="70" />
          <el-table-column prop="skuId" label="SKU" width="70" />
          <el-table-column prop="expected" label="期望" width="90" />
          <el-table-column prop="actual" label="实际" width="90" />
          <el-table-column prop="detail" label="说明" min-width="200" />
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import MoneyText from '@/components/jxc/MoneyText.vue'
import StockCheck from './check.vue'
import { ledgerBalances, ledgerMoves, ledgerReconcile, listWarehouses } from '@/api/jxc'

const tab = ref('ledger')
const warehouses = ref([])
const warehouseId = ref(null)
const rows = ref([])
const keyword = ref('')
const loading = ref(false)

const drawer = ref(false)
const moves = ref([])
const loadingMoves = ref(false)
const currentSku = ref(null)
const drawerTitle = ref('')

const reconDialog = ref(false)
const reconing = ref(false)
const recon = ref({ ok: true, violations: [] })

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) return rows.value
  return rows.value.filter((r) =>
    [r.goodsName, r.skuCode, r.color, r.size].some((v) => String(v ?? '').toLowerCase().includes(k))
  )
})

const SOURCE = {
  purchase_in: '采购入库',
  sale_out: '销售出库',
  sale_return: '退货入库',
  direct_in: '直接入库',
  direct_out: '直接出库',
  check_gain: '盘盈',
  check_loss: '盘亏',
  reversal: '红冲'
}
const sourceLabel = (t) => SOURCE[t] ?? t

const fmt = (s) => {
  if (!s) return ''
  const d = new Date(s)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const load = async () => {
  loading.value = true
  try {
    const res = await ledgerBalances({ warehouseId: warehouseId.value || 0 })
    if (res.code === 0) rows.value = res.data || []
    else ElMessage.error(res.msg || '加载台账失败')
  } finally {
    loading.value = false
  }
}

const openMoves = async (row) => {
  currentSku.value = row
  drawerTitle.value = `${row.goodsName} ${row.color}/${row.size}（${row.skuCode}）的库存流水`
  drawer.value = true
  loadingMoves.value = true
  try {
    const res = await ledgerMoves({ warehouseId: row.warehouseId, skuId: row.skuId, page: 1, pageSize: 200 })
    const list = res.code === 0 ? res.data.list || [] : []
    // 接口按 id 倒序返回；结存要从最早一笔往后累加，所以先反转再算，最后翻回来展示
    let running = 0
    const asc = [...list].reverse()
    for (const mv of asc) {
      running += mv.qty
      mv.balance = running
    }
    moves.value = asc.reverse()
  } finally {
    loadingMoves.value = false
  }
}

const doReconcile = async () => {
  reconing.value = true
  try {
    const res = await ledgerReconcile({ warehouseId: warehouseId.value || 0 })
    if (res.code !== 0) return ElMessage.error(res.msg || '对账失败')
    recon.value = res.data
    reconDialog.value = true
  } finally {
    reconing.value = false
  }
}

onMounted(async () => {
  const w = await listWarehouses()
  if (w.code === 0) warehouses.value = w.data || []
  await load()
})
</script>

<style scoped>
.inventory {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
}
.spacer {
  flex: 1;
}
.ledger-table :deep(.el-table__row) {
  cursor: pointer;
}
.zero {
  color: var(--el-text-color-placeholder);
}
.muted {
  color: var(--el-text-color-placeholder);
}
.bad,
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
.doc {
  font-family: ui-monospace, monospace;
}
.inv-tabs :deep(.el-tabs__header) {
  margin-bottom: 8px;
}
.inv-body {
  min-height: 0;
}
.recon-ok {
  text-align: center;
  padding: 24px 0;
}
.recon-ok .big {
  font-size: 18px;
  margin-bottom: 6px;
}
</style>

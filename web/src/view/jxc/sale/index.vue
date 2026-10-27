<template>
  <div class="sale-page">
    <!-- 左：选货。点卡片直接进右侧明细，不弹窗（弹窗会遮挡上下文） -->
    <section class="picker">
      <div class="picker-head">
        <el-input ref="searchRef" v-model="keyword" placeholder="扫码 / 搜索商品名、条码（按 / 聚焦）"
          clearable size="default" @keydown.enter.prevent="onSearchEnter">
          <template #prefix><span class="kbd">/</span></template>
        </el-input>
        <el-select v-model="warehouseId" placeholder="选择仓库" style="width: 150px" @change="loadBalances">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
        </el-select>
      </div>

      <div v-loading="loadingGoods" class="cards">
        <div v-for="g in filteredGoods" :key="g.ID" class="goods-card">
          <div class="g-name">{{ g.name }}</div>
          <div class="g-code">{{ g.code }}</div>
          <div class="skus">
            <button v-for="s in g.skus" :key="s.ID" class="sku-chip"
              :class="{ out: availableOf(s.ID) <= 0 }" :disabled="availableOf(s.ID) <= 0"
              @click="addLine(s)">
              <span class="spec">{{ s.color }}/{{ s.size }}</span>
              <span class="avail">{{ availableOf(s.ID) > 0 ? `可售 ${availableOf(s.ID)}` : '缺货' }}</span>
            </button>
          </div>
        </div>
        <el-empty v-if="!loadingGoods && filteredGoods.length === 0" description="没有匹配的商品" />
      </div>
    </section>

    <!-- 右：当前单据 -->
    <section class="cart">
      <div class="cart-head">
        <span class="doc-no">{{ orderNo || '新销售单' }}</span>
        <span class="hint">Ctrl+Enter 出库 · Esc 清空</span>
      </div>

      <div class="lines">
        <div v-for="(l, i) in lines" :key="l.skuId" class="line">
          <div class="l-name">
            <div>{{ l.goodsName }}</div>
            <div class="l-spec">{{ l.color }}/{{ l.size }}</div>
          </div>
          <el-input-number v-model="l.qty" :min="1" :max="availableOf(l.skuId) + l.qty" size="small"
            controls-position="right" style="width: 92px" />
          <MoneyText :value="l.price" class="l-price" />
          <MoneyText :value="l.qty * l.price" class="l-amount" />
          <button class="rm" @click="lines.splice(i, 1)">×</button>
        </div>
        <div v-if="lines.length === 0" class="cart-empty">
          点左侧规格加入明细<br />
          <span class="sub">可售为 0 的规格会显示「缺货」并置灰</span>
        </div>
      </div>

      <footer class="cart-foot">
        <div class="total">
          <span>合计</span>
          <MoneyText :value="total" class="total-num" />
        </div>
        <div class="paid">
          <span>实收</span>
          <el-input-number v-model="paid" :min="0" :precision="2" size="small" style="width: 130px" />
          <span v-if="change !== 0" class="change" :class="{ short: change < 0 }">
            {{ change < 0 ? `还差 ${Math.abs(change).toFixed(2)}` : `找零 ${change.toFixed(2)}` }}
          </span>
        </div>
        <div class="actions">
          <el-button @click="reset">清空</el-button>
          <el-button type="primary" :loading="saving" :disabled="lines.length === 0" @click="submit">
            出库（Ctrl+Enter）
          </el-button>
        </div>
        <div v-if="lastResult" class="result">
          已出库 {{ lastResult.orderNo }}：销售额
          <MoneyText :value="lastResult.amount" />，成本
          <MoneyText :value="lastResult.cost" />，毛利
          <MoneyText :value="lastResult.gross" :class="{ neg: lastResult.gross < 0 }" />
        </div>
      </footer>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import MoneyText from '@/components/jxc/MoneyText.vue'
import { cancelSale, createSale, ledgerBalances, listGoods, listWarehouses, shipSale } from '@/api/jxc'

const warehouses = ref([])
const warehouseId = ref(null)
const goods = ref([])
const balances = ref({}) // skuId -> { available }
const keyword = ref('')
const lines = ref([])
const paid = ref(0)
const saving = ref(false)
const loadingGoods = ref(false)
const orderNo = ref('')
const lastResult = ref(null)
const searchRef = ref(null)

const availableOf = (skuId) => balances.value[skuId]?.available ?? 0

const filteredGoods = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) return goods.value
  return goods.value.filter((g) => {
    if (g.name?.toLowerCase().includes(k) || g.code?.toLowerCase().includes(k)) return true
    return g.skus?.some((s) => s.skuCode?.toLowerCase().includes(k))
  })
})

const total = computed(() => lines.value.reduce((s, l) => s + l.qty * l.price, 0))
const change = computed(() => Math.round((paid.value - total.value) * 100) / 100)

const loadGoods = async () => {
  loadingGoods.value = true
  try {
    const res = await listGoods()
    if (res.code === 0) goods.value = res.data || []
    else ElMessage.error(res.msg || '加载商品失败')
  } finally {
    loadingGoods.value = false
  }
}

const loadBalances = async () => {
  balances.value = {}
  if (!warehouseId.value) return
  const res = await ledgerBalances({ warehouseId: warehouseId.value })
  if (res.code === 0) {
    const map = {}
    for (const row of res.data || []) map[row.skuId] = row
    balances.value = map
  }
}

const addLine = (sku) => {
  const exist = lines.value.find((l) => l.skuId === sku.ID)
  const avail = availableOf(sku.ID)
  if (exist) {
    if (exist.qty + 1 > avail) {
      ElMessage.warning(`该规格可售 ${avail} 件，不能再加`)
      return
    }
    exist.qty += 1
    return
  }
  if (avail <= 0) return
  const g = goods.value.find((x) => x.ID === sku.goodsId)
  lines.value.push({
    skuId: sku.ID,
    goodsName: g?.name ?? '',
    color: sku.color,
    size: sku.size,
    qty: 1,
    price: Number(sku.salePrice) || 0
  })
  paid.value = total.value
}

const addFirstMatch = () => {
  const g = filteredGoods.value[0]
  if (!g) return
  const sku = (g.skus || []).find((s) => availableOf(s.ID) > 0)
  if (sku) addLine(sku)
  else ElMessage.warning('该商品没有可售规格')
}

// 搜索框里的 Enter 是"加入当前匹配项"；Ctrl/Cmd+Enter 是"整单出库"，
// 不能顺手再加一件（否则提交前数量多 1，实测踩到）。
const onSearchEnter = (e) => {
  if (e.ctrlKey || e.metaKey) return
  addFirstMatch()
}

const reset = () => {
  lines.value = []
  paid.value = 0
  orderNo.value = ''
}

const submit = async () => {
  if (saving.value) return
  if (!warehouseId.value) return ElMessage.warning('请先选择仓库')
  if (lines.value.length === 0) return ElMessage.warning('请先添加商品')
  saving.value = true
  try {
    const payload = {
      warehouseId: warehouseId.value,
      items: lines.value.map((l) => ({ skuId: l.skuId, qty: l.qty, price: l.price }))
    }
    const created = await createSale(payload)
    if (created.code !== 0) return ElMessage.error(created.msg || '开单失败')
    const id = created.data.ID
    orderNo.value = created.data.orderNo

    const shipped = await shipSale(id)
    if (shipped.code !== 0) {
      // 开单成功但出库失败（例如可售库存在此期间被别人买走）时，
      // 草稿单会继续占着预留。这里主动取消，避免"失败一次、库存被锁一次"。
      const canceled = await cancelSale(id)
      const hint = canceled.code === 0
        ? '草稿单已自动取消并释放预留'
        : `草稿单 ${orderNo.value} 未能自动取消，请到「单据」里手动取消`
      return ElMessage.error(`${shipped.msg || '出库失败'}（${hint}）`)
    }

    const detail = await import('@/api/jxc').then((m) => m.getSale(id))
    const order = detail.code === 0 ? detail.data : null
    lastResult.value = {
      orderNo: order?.orderNo ?? orderNo.value,
      amount: Number(order?.totalAmount ?? total.value),
      cost: Number(order?.costAmount ?? 0),
      gross: Math.round((Number(order?.totalAmount ?? 0) - Number(order?.costAmount ?? 0)) * 100) / 100
    }
    ElMessage.success(`出库成功 ${lastResult.value.orderNo}`)
    reset()
    await loadBalances()
  } finally {
    saving.value = false
  }
}

const isTyping = (el) => {
  if (!el) return false
  return ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName) || el.isContentEditable
}

// 键盘快捷键挂在 window 上：挂在页面 div 上时，只有焦点正好在 div 内才生效，
// 从导航点进来后焦点在 body，Ctrl+Enter 会静默失灵。
const onKeydown = (e) => {
  if (e.key === '/' && !isTyping(e.target)) {
    e.preventDefault()
    searchRef.value?.focus()
    return
  }
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    submit()
    return
  }
  if (e.key === 'Escape') {
    // 正在输入时 Esc 只退出输入，不能把整车货清空（误触代价太大）
    if (isTyping(e.target)) {
      e.target.blur()
      return
    }
    reset()
  }
}

onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  await loadGoods()
  const res = await listWarehouses()
  if (res.code === 0) {
    warehouses.value = res.data || []
    if (warehouses.value.length && !warehouseId.value) {
      warehouseId.value = warehouses.value[0].ID
      await loadBalances()
    }
  }
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  lastResult.value = null
})
</script>

<style scoped>
.sale-page {
  display: grid;
  grid-template-columns: 1fr 460px;
  gap: 16px;
  height: calc(100vh - 48px - 32px);
  min-height: 0;
}
.picker,
.cart {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.picker-head {
  display: flex;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.kbd {
  font-family: ui-monospace, monospace;
  color: var(--el-text-color-placeholder);
}
.cards {
  flex: 1;
  overflow: auto;
  padding: 12px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
  align-content: start;
}
.goods-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px;
}
.g-name {
  font-weight: 500;
  margin-bottom: 2px;
}
.g-code {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-family: ui-monospace, monospace;
  margin-bottom: 8px;
}
.skus {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.sku-chip {
  border: 1px solid var(--el-border-color);
  background: var(--el-bg-color);
  border-radius: 4px;
  padding: 4px 8px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 1px;
  font-size: 12px;
}
.sku-chip:hover:not(:disabled) {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.sku-chip.out {
  opacity: 0.55;
  cursor: not-allowed;
}
.spec {
  font-weight: 500;
}
.avail {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}
.cart-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.doc-no {
  font-family: ui-monospace, monospace;
  font-weight: 500;
}
.hint {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
.lines {
  flex: 1;
  overflow: auto;
  padding: 8px 12px;
}
.line {
  display: grid;
  grid-template-columns: 1fr 100px 84px 84px 24px;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px dashed var(--el-border-color-lighter);
}
.l-name {
  font-size: 13px;
}
.l-spec {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.rm {
  border: 0;
  background: none;
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  font-size: 16px;
}
.rm:hover {
  color: var(--el-color-danger);
}
.cart-empty {
  text-align: center;
  color: var(--el-text-color-secondary);
  padding: 40px 0;
  font-size: 13px;
}
.sub {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
.cart-foot {
  border-top: 1px solid var(--el-border-color-lighter);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.total {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.total-num {
  font-size: 22px;
  font-weight: 600;
}
.paid {
  display: flex;
  align-items: center;
  gap: 8px;
}
.change {
  color: var(--el-color-success);
  font-size: 12px;
}
.change.short {
  color: var(--el-color-danger);
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.result {
  font-size: 12px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  border-radius: 6px;
  padding: 8px 10px;
}
.neg {
  color: var(--el-color-danger);
}
</style>

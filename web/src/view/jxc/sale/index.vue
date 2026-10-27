<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <PageHeader
      eyebrow="SALES"
      title="开单"
      description="扫码或搜索商品，点规格直接加入右侧；一次提交完成开单、预留与出库，失败全部回滚。"
    >
      <template #actions>
        <div class="flex items-center gap-2 rounded-md border border-border bg-container px-2.5 py-1.5 text-[11px] text-muted-foreground">
          <kbd class="jxc-kbd">/</kbd><span>搜索</span>
          <span class="h-3 w-px bg-border"></span>
          <kbd class="jxc-kbd">Ctrl ↵</kbd><span>出库</span>
          <span class="h-3 w-px bg-border"></span>
          <kbd class="jxc-kbd">Esc</kbd><span>清空</span>
        </div>
      </template>
    </PageHeader>

    <div class="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_380px] xl:grid-cols-[minmax(0,1fr)_440px]">
      <!-- 左：选货。点规格直接进右侧明细，不弹窗 -->
      <SectionCard title="选择商品" description="点规格加入明细" :padded="false" class="min-h-0">
        <template #actions>
          <span class="text-[11px] text-muted-foreground">{{ filteredGoods.length }} 个商品</span>
        </template>

        <div class="flex h-full min-h-0 flex-col">
          <div class="flex flex-wrap items-center gap-2 border-b border-border px-4 py-3">
            <div class="min-w-[220px] flex-1">
              <el-input
                ref="searchRef"
                v-model="keyword"
                class="w-full"
                placeholder="扫码 / 搜索商品名、编码、条码"
                clearable
                @keydown.enter="onSearchEnter"
              >
                <template #prefix><svg-icon icon="lucide:search" class="h-3.5 w-3.5" /></template>
                <template #suffix>
                  <kbd v-if="!keyword" class="jxc-kbd">/</kbd>
                </template>
              </el-input>
            </div>
            <div class="w-[148px] shrink-0">
              <el-select v-model="warehouseId" class="w-full" placeholder="选择仓库" @change="onWarehouseChange">
                <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
              </el-select>
            </div>
            <button
              type="button"
              class="flex h-8 items-center gap-1.5 rounded-md border px-2.5 text-[12px] transition-colors"
              :class="hideOutOfStock ? 'border-primary-200 bg-primary-50 text-primary-700' : 'border-border text-muted-foreground hover:bg-muted'"
              :aria-pressed="hideOutOfStock"
              @click="hideOutOfStock = !hideOutOfStock"
            >
              <svg-icon icon="lucide:package-check" class="h-3.5 w-3.5" />
              仅看有货
            </button>
          </div>

          <div v-if="!warehouses.length" class="border-b border-warning-200 bg-warning-50 px-4 py-2.5 text-[12px] text-warning-800">
            还没有启用仓库。先到「基础资料 → 仓库」建一个，否则无法开单。
          </div>

          <div v-loading="loading || loadingBalances" class="min-h-0 flex-1 overflow-y-auto">
            <div
              v-for="product in filteredGoods"
              :key="product.ID"
              class="border-b border-border px-4 py-3 transition-colors last:border-b-0 hover:bg-muted"
            >
              <div class="flex items-baseline gap-2">
                <span class="truncate text-[13px] font-medium text-base-text">{{ product.name }}</span>
                <span class="shrink-0 font-mono text-[11px] text-muted-foreground">{{ product.code }}</span>
              </div>
              <div class="mt-2 flex flex-wrap gap-1.5">
                <button
                  v-for="sku in product.skus"
                  :key="sku.ID"
                  type="button"
                  class="group inline-flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-[12px] transition-colors"
                  :class="sku.avail > 0
                    ? 'border-border bg-container hover:border-primary-300 hover:bg-primary-50 hover:text-primary-700'
                    : 'cursor-not-allowed border-dashed border-border bg-muted text-muted-foreground opacity-60'"
                  :disabled="sku.avail <= 0"
                  :title="`售价 ¥${Number(sku.salePrice || 0).toFixed(2)} · 条码 ${sku.skuCode}`"
                  @click="addLine(sku, product)"
                >
                  <span class="font-medium">{{ sku.color }}/{{ sku.size }}</span>
                  <span class="tabular-nums text-[11px]" :class="sku.avail > 0 ? 'text-muted-foreground group-hover:text-primary-600' : ''">
                    {{ sku.avail > 0 ? `可售 ${sku.avail}` : '缺货' }}
                  </span>
                  <span class="text-[11px] tabular-nums text-muted-foreground group-hover:text-primary-600">¥{{ Number(sku.salePrice || 0).toFixed(2) }}</span>
                </button>
                <span v-if="!product.skus.length" class="text-[11px] text-muted-foreground">没有启用的规格</span>
              </div>
            </div>

            <EmptyState
              v-if="!loading && filteredGoods.length === 0"
              icon="lucide:package-search"
              :title="keyword ? '没有匹配的商品' : '当前仓库没有可售商品'"
              :description="keyword ? '换个商品名、编码或条码后四位试试，或关闭「仅看有货」。' : '可以先补货入库，或切换到其他仓库。'"
            >
              <el-button v-if="!keyword" @click="hideOutOfStock = false">显示缺货商品</el-button>
            </EmptyState>
          </div>

          <div class="flex shrink-0 items-center gap-2 border-t border-border bg-muted px-4 py-2 text-[11px] text-muted-foreground">
            <svg-icon icon="lucide:keyboard" class="h-3.5 w-3.5" />
            <span>扫码枪直接扫条码</span>
            <span class="ml-auto">缺货规格仍可见但不可点</span>
          </div>
        </div>
      </SectionCard>

      <!-- 右：当前单据 -->
      <SectionCard :title="orderNo || '新销售单'" description="原子提交 · 失败全回滚" :padded="false" class="min-h-0">
        <template #actions>
          <el-button link :disabled="!lines.length && !lastResult" @click="resetCart">清空</el-button>
        </template>

        <div class="flex h-full min-h-0 flex-col">
          <div class="grid shrink-0 grid-cols-2 gap-2 border-b border-border px-4 py-3">
            <label class="block min-w-0">
              <span class="mb-1.5 block text-[11px] text-muted-foreground">出库仓库</span>
              <el-select v-model="warehouseId" class="w-full" placeholder="选择仓库" @change="onWarehouseChange">
                <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
              </el-select>
            </label>
            <label class="block min-w-0">
              <span class="mb-1.5 block text-[11px] text-muted-foreground">客户</span>
              <el-select v-model="customerId" class="w-full" placeholder="散客（不记名）" clearable>
                <el-option v-for="c in customers" :key="c.ID" :label="c.name" :value="c.ID" />
              </el-select>
            </label>
          </div>

          <div v-if="lastResult" class="shrink-0 border-b border-success-200 bg-success-50 px-4 py-2.5">
            <div class="flex items-start gap-2">
              <svg-icon icon="lucide:circle-check" class="mt-0.5 h-4 w-4 shrink-0 text-success" />
              <div class="min-w-0 flex-1 text-[11.5px] leading-5 text-success-800">
                <div class="flex items-center gap-2 font-medium">
                  <span>已出库 {{ lastResult.orderNo }}</span>
                  <button type="button" class="text-success-700/70 hover:text-success-800" aria-label="关闭结果" @click="lastResult = null">
                    <svg-icon icon="lucide:x" class="h-3.5 w-3.5" />
                  </button>
                </div>
                <div class="flex flex-wrap items-center gap-x-2">
                  <span>销售额 <MoneyText :value="lastResult.amount" size="xs" /></span>
                  <span>成本 <MoneyText :value="lastResult.cost" size="xs" /></span>
                  <span>毛利 <MoneyText :value="lastResult.gross" size="xs" /></span>
                </div>
              </div>
            </div>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto">
            <template v-if="lines.length">
              <div class="sticky top-0 z-10 grid grid-cols-[minmax(0,1fr)_76px_62px_70px_20px] items-center gap-2 border-b border-border bg-container px-4 py-2 text-[11px] text-muted-foreground">
                <span>商品</span>
                <span class="text-center">数量</span>
                <span class="text-right">单价</span>
                <span class="text-right">小计</span>
                <span></span>
              </div>
              <div
                v-for="(line, index) in lines"
                :key="line.skuId"
                class="grid grid-cols-[minmax(0,1fr)_76px_62px_70px_20px] items-center gap-2 border-b border-border px-4 py-2.5 last:border-b-0"
              >
                <div class="min-w-0">
                  <div class="truncate text-[12.5px] font-medium text-base-text">{{ line.goodsName }}</div>
                  <div class="mt-0.5 truncate text-[11px] text-muted-foreground">{{ line.color }}/{{ line.size }} · {{ line.skuCode }}</div>
                </div>
                <div class="inline-flex h-7 items-center justify-self-center overflow-hidden rounded-md border border-border bg-container">
                  <button
                    type="button"
                    class="grid h-full w-6 place-items-center text-muted-foreground transition-colors hover:bg-muted hover:text-base-text"
                    aria-label="减少数量"
                    @click="changeQty(line, -1)"
                  >−</button>
                  <input
                    v-model.number="line.qty"
                    type="number"
                    min="1"
                    class="h-full w-8 min-w-0 border-0 bg-transparent text-center text-[12px] tabular-nums text-base-text outline-none"
                    :aria-label="`${line.goodsName} 数量`"
                    @change="clampLine(line)"
                  />
                  <button
                    type="button"
                    class="grid h-full w-6 place-items-center text-muted-foreground transition-colors hover:bg-muted hover:text-base-text"
                    aria-label="增加数量"
                    @click="changeQty(line, 1)"
                  >+</button>
                </div>
                <div class="text-right"><MoneyText :value="line.price" size="sm" /></div>
                <div class="text-right"><MoneyText :value="round2(line.qty * line.price)" size="sm" strong /></div>
                <button
                  type="button"
                  class="grid h-6 w-6 place-items-center justify-self-end rounded text-muted-foreground transition-colors hover:bg-error-50 hover:text-error"
                  aria-label="移除该行"
                  @click="removeLine(index)"
                >
                  <svg-icon icon="lucide:x" class="h-3.5 w-3.5" />
                </button>
              </div>
            </template>

            <EmptyState
              v-else-if="!lastResult"
              icon="lucide:shopping-cart"
              title="单据还是空的"
              description="在左侧点任意规格加入明细。可售为 0 的规格会置灰，但仍可见——你知道有这款，只是没货。"
            />
            <div v-else class="px-6 py-10 text-center text-[12px] text-muted-foreground">
              继续扫码或搜索下一件商品
            </div>
          </div>

          <div class="shrink-0 border-t border-border px-4 py-3">
            <div class="flex items-baseline justify-between">
              <div class="flex items-baseline gap-2">
                <span class="text-[12px] text-muted-foreground">合计</span>
                <span class="text-[11px] text-muted-foreground">{{ totalQty }} 件</span>
              </div>
              <MoneyText :value="total" size="xl" strong />
            </div>

            <div class="mt-3 flex items-center gap-2">
              <span class="shrink-0 text-[12px] text-muted-foreground">实收</span>
              <el-input-number
                v-model="paid"
                class="!w-[132px] shrink-0"
                :min="0"
                :precision="2"
                :step="1"
                :controls="false"
                @change="paidTouched = true"
              />
              <span class="ml-auto flex items-center gap-1 text-[11px]" :class="change < 0 ? 'text-error' : 'text-muted-foreground'">
                {{ change < 0 ? '还差' : '找零' }}
                <MoneyText :value="Math.abs(change)" size="xs" :muted="change >= 0" />
              </span>
            </div>

            <div class="mt-3 flex items-center gap-2">
              <el-button size="large" class="flex-1" @click="resetCart">清空</el-button>
              <el-button
                type="primary"
                size="large"
                class="flex-[2]"
                :loading="saving"
                :disabled="lines.length === 0 || !warehouseId"
                @click="submit"
              >
                出库
                <span class="ml-2 rounded border border-white/25 px-1 py-0.5 font-mono text-[10px] text-white/80">Ctrl ↵</span>
              </el-button>
            </div>
            <div class="mt-2 flex items-center gap-2 text-[11px] text-muted-foreground">
              <svg-icon icon="lucide:lock" class="h-3 w-3" />
              <span>单价固定取规格售价，开单后改价由服务端拒绝</span>
            </div>
          </div>
        </div>
      </SectionCard>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import PageHeader from '@/components/jxc/PageHeader.vue'
import SectionCard from '@/components/jxc/SectionCard.vue'
import EmptyState from '@/components/jxc/EmptyState.vue'
import MoneyText from '@/components/jxc/MoneyText.vue'
import { checkoutSale, ledgerBalances, listGoods, listPartners, listWarehouses } from '@/api/jxc'
import { isTypingTarget } from '@/view/jxc/shared/keyboard'
import { round2 } from '@/view/jxc/shared/format'

const warehouses = ref([])
const customers = ref([])
const goods = ref([])
const balances = ref({})
const warehouseId = ref(null)
const customerId = ref(null)
const keyword = ref('')
const hideOutOfStock = ref(true)
const lines = ref([])
const paid = ref(0)
const paidTouched = ref(false)
const saving = ref(false)
const loading = ref(false)
const loadingBalances = ref(false)
const orderNo = ref('')
const lastResult = ref(null)
const searchRef = ref(null)

const availableOf = (skuId) => {
  const row = balances.value[skuId]
  return row ? Number(row.available) || 0 : 0
}

const filteredGoods = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  const rows = goods.value.map((product) => {
    const skus = (product.skus || []).map((sku) => ({ ...sku, avail: availableOf(sku.ID) }))
    return { ...product, skus, maxAvail: skus.reduce((max, sku) => Math.max(max, sku.avail), 0) }
  })

  let list = rows
  if (hideOutOfStock.value) list = list.filter((product) => product.skus.some((sku) => sku.avail > 0))
  if (k) {
    list = list.filter((product) => {
      if ((product.name || '').toLowerCase().includes(k) || (product.code || '').toLowerCase().includes(k)) return true
      return product.skus.some((sku) =>
        `${sku.skuCode} ${sku.color} ${sku.size}`.toLowerCase().includes(k)
      )
    })
  }
  return list.sort((a, b) => b.maxAvail - a.maxAvail || String(a.name).localeCompare(String(b.name), 'zh-CN'))
})

const total = computed(() => round2(lines.value.reduce((sum, line) => sum + Number(line.qty || 0) * Number(line.price || 0), 0)))
const totalQty = computed(() => lines.value.reduce((sum, line) => sum + Number(line.qty || 0), 0))
const change = computed(() => round2(Number(paid.value || 0) - total.value))

watch(total, (value) => {
  if (!paidTouched.value) paid.value = value
})

const loadGoods = async () => {
  loading.value = true
  try {
    const res = await listGoods({ active: 1 })
    if (res.code !== 0) return ElMessage.error(res.msg || '加载商品失败')
    goods.value = (res.data || [])
      .map((product) => ({ ...product, skus: (product.skus || []).filter((sku) => sku.status === 1) }))
      .filter((product) => product.skus.length > 0)
  } finally {
    loading.value = false
  }
}

const loadBalances = async () => {
  balances.value = {}
  if (!warehouseId.value) return
  loadingBalances.value = true
  try {
    const res = await ledgerBalances({ warehouseId: warehouseId.value })
    if (res.code !== 0) return ElMessage.error(res.msg || '加载可售库存失败')
    const map = {}
    for (const row of res.data || []) map[row.skuId] = row
    balances.value = map
  } finally {
    loadingBalances.value = false
  }
}

const addLine = (sku, product) => {
  const avail = availableOf(sku.ID)
  if (avail <= 0) return
  const exist = lines.value.find((line) => line.skuId === sku.ID)
  if (exist) {
    if (exist.qty + 1 > avail) {
      ElMessage.warning(`该规格可售 ${avail} 件，不能再加`)
      return
    }
    exist.qty += 1
  } else {
    lines.value.push({
      skuId: sku.ID,
      skuCode: sku.skuCode,
      goodsName: product.name,
      color: sku.color,
      size: sku.size,
      qty: 1,
      price: Number(sku.salePrice) || 0
    })
  }
  if (!paidTouched.value) paid.value = total.value
}

const addFirstMatch = () => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) return

  // 先找条码完全匹配（扫码枪场景），再找颜色/尺码/条码包含关键字的规格，
  // 最后才退回「第一个有货规格」。否则搜「白/L」会加进第一行的黑/M。
  let match = null
  for (const product of filteredGoods.value) {
    const exact = (product.skus || []).find((sku) => String(sku.skuCode || '').toLowerCase() === k && sku.avail > 0)
    if (exact) {
      match = { sku: exact, product }
      break
    }
  }
  if (!match) {
    for (const product of filteredGoods.value) {
      const hit = (product.skus || []).find((sku) =>
        sku.avail > 0 && `${sku.skuCode} ${sku.color} ${sku.size}`.toLowerCase().includes(k)
      )
      if (hit) {
        match = { sku: hit, product }
        break
      }
    }
  }
  if (!match) {
    const product = filteredGoods.value[0]
    const sku = product?.skus?.find((item) => item.avail > 0)
    if (sku) match = { sku, product }
  }
  if (!match) return ElMessage.warning('没有匹配的可售规格')

  addLine(match.sku, match.product)
  keyword.value = ''
  nextTick(() => searchRef.value?.focus())
}

const onSearchEnter = (event) => {
  if (event.ctrlKey || event.metaKey) return
  event.preventDefault()
  addFirstMatch()
}

const changeQty = (line, delta) => {
  const next = Number(line.qty || 0) + delta
  const max = availableOf(line.skuId)
  if (next > max) {
    ElMessage.warning(`该规格可售 ${max} 件`)
    return
  }
  line.qty = Math.max(1, next)
  if (!paidTouched.value) paid.value = total.value
}

const clampLine = (line) => {
  const max = Math.max(1, availableOf(line.skuId))
  line.qty = Math.min(max, Math.max(1, Math.floor(Number(line.qty) || 1)))
  if (!paidTouched.value) paid.value = total.value
}

const removeLine = (index) => {
  lines.value.splice(index, 1)
  if (!paidTouched.value) paid.value = total.value
}

const resetCart = () => {
  lines.value = []
  paid.value = 0
  paidTouched.value = false
  orderNo.value = ''
  lastResult.value = null
}

const onWarehouseChange = async () => {
  if (lines.value.length) {
    resetCart()
    ElMessage.info('已切换仓库，当前明细已清空')
  }
  await loadBalances()
}

const submit = async () => {
  if (saving.value) return
  if (!warehouseId.value) return ElMessage.warning('请先选择仓库')
  if (!lines.value.length) return ElMessage.warning('请先添加商品')

  saving.value = true
  try {
    // 原子接口：建单、预留、出库同一事务；失败不会留下草稿或占用。
    const res = await checkoutSale({
      warehouseId: warehouseId.value,
      customerId: customerId.value || null,
      items: lines.value.map((line) => ({ skuId: line.skuId, qty: line.qty }))
    })
    if (res.code !== 0) return ElMessage.error(res.msg || '开单出库失败')

    const order = res.data || {}
    const amount = Number(order.totalAmount) || 0
    const cost = Number(order.costAmount) || 0
    orderNo.value = order.orderNo || ''
    lastResult.value = {
      orderNo: order.orderNo || orderNo.value,
      amount,
      cost,
      gross: round2(amount - cost)
    }
    ElMessage.success(`出库成功 ${lastResult.value.orderNo}`)
    orderNo.value = ''
    lines.value = []
    paid.value = 0
    paidTouched.value = false
    await loadBalances()
  } finally {
    saving.value = false
  }
}

const onKeydown = (event) => {
  if (event.key === '/' && !isTypingTarget(event.target)) {
    event.preventDefault()
    searchRef.value?.focus()
    return
  }
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
    event.preventDefault()
    submit()
    return
  }
  if (event.key === 'Escape') {
    if (isTypingTarget(event.target)) {
      event.target.blur()
      return
    }
    if (lines.value.length) resetCart()
  }
}

onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  const [warehouseRes, customerRes] = await Promise.all([
    listWarehouses({ active: 1 }),
    listPartners('customer', { active: 1 })
  ])
  if (warehouseRes.code === 0) warehouses.value = warehouseRes.data || []
  if (customerRes.code === 0) customers.value = customerRes.data || []
  if (warehouses.value.length) {
    warehouseId.value = warehouses.value[0].ID
    await loadBalances()
  }
  await loadGoods()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})

</script>

<template>
  <div class="cashier">
    <el-tabs v-model="activeTab" class="cashier-tabs">
      <!-- ==================== 收款 ==================== -->
      <el-tab-pane label="收款" name="cash">
        <div class="cashier">
          <div class="left">
            <el-card shadow="never" class="mb-4">
              <el-input
                v-model="keyword"
                placeholder="搜索商品名称 / SKU编码 / 条码，回车搜索"
                clearable
                size="large"
                @keyup.enter="searchGoods"
                @clear="searchGoods"
              >
                <template #append><el-button icon="Search" @click="searchGoods" /></template>
              </el-input>
            </el-card>
            <el-card shadow="never" class="goods-card" v-loading="loading">
              <el-empty v-if="!goodsList.length" description="输入关键词搜索商品" />
              <div v-else class="goods-grid">
                <div v-for="s in goodsList" :key="s.ID" class="goods-item" :class="{ disabled: (s.available || 0) <= 0 }" @click="addToCart(s)">
                  <div class="goods-name">{{ s.goodsName || s.skuCode }}</div>
                  <div class="goods-sku">{{ s.skuCode }}</div>
                  <div class="goods-spec">{{ s.color }} / {{ s.size }}</div>
                  <div class="goods-price">¥ {{ (s.salePrice || 0).toFixed(2) }}</div>
                  <div class="goods-stock">可售 {{ s.available ?? '-' }}</div>
                </div>
              </div>
            </el-card>
          </div>
          <div class="right">
            <el-card shadow="never" class="cart-card">
              <template #header>
                <div class="flex-between">
                  <span>购物车（{{ cart.length }} 项）</span>
                  <el-button link type="danger" @click="clearCart">清空</el-button>
                </div>
              </template>
              <el-table :data="cart" border size="small" max-height="260">
                <el-table-column label="商品" min-width="140">
                  <template #default="{ row }">
                    <div>{{ row.skuCode }}</div>
                    <div class="tip-text">{{ row.color }} / {{ row.size }}</div>
                  </template>
                </el-table-column>
                <el-table-column label="单价" width="80" align="right">
                  <template #default="{ row }">¥ {{ row.salePrice.toFixed(2) }}</template>
                </el-table-column>
                <el-table-column label="数量" width="110" align="center">
                  <template #default="{ row }"><el-input-number v-model="row.qty" :min="1" :max="row.available" size="small" /></template>
                </el-table-column>
                <el-table-column label="小计" width="90" align="right">
                  <template #default="{ row }">¥ {{ (row.salePrice * row.qty).toFixed(2) }}</template>
                </el-table-column>
                <el-table-column label="" width="46" align="center">
                  <template #default="{ $index }"><el-button link type="danger" @click="cart.splice($index, 1)">删</el-button></template>
                </el-table-column>
              </el-table>
              <div class="total-row">合计：<b class="total-amount">¥ {{ totalAmount.toFixed(2) }}</b></div>
              <el-form label-width="90px" class="pay-form">
                <el-form-item label="收银仓库">
                  <el-select v-model="warehouseId" placeholder="选择仓库" style="width: 200px">
                    <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
                  </el-select>
                </el-form-item>
                <el-form-item label="客户">
                  <el-select v-model="customerId" placeholder="散客" clearable filterable style="width: 200px">
                    <el-option v-for="cu in customers" :key="cu.ID" :label="cu.name" :value="cu.ID" />
                  </el-select>
                </el-form-item>
                <el-form-item label="支付方式">
                  <el-radio-group v-model="payMethod">
                    <el-radio-button value="cash">现金</el-radio-button>
                    <el-radio-button value="wechat">微信</el-radio-button>
                    <el-radio-button value="alipay">支付宝</el-radio-button>
                  </el-radio-group>
                </el-form-item>
                <el-form-item v-if="payMethod === 'cash'" label="实收金额">
                  <el-input-number v-model="paidAmount" :min="totalAmount" :precision="2" style="width: 200px" />
                </el-form-item>
                <el-form-item v-if="payMethod === 'cash' && change >= 0" label="找零">
                  <b style="color: #e6a23c">¥ {{ change.toFixed(2) }}</b>
                </el-form-item>
              </el-form>
              <el-button type="success" size="large" class="pay-btn" :disabled="!cart.length || !warehouseId" :loading="saving" @click="handleCheckout">
                收款 ¥ {{ totalAmount.toFixed(2) }}
              </el-button>
            </el-card>
          </div>
        </div>
      </el-tab-pane>

      <!-- ==================== 退款 ==================== -->
      <el-tab-pane label="退款" name="refund">
        <el-card shadow="never">
          <el-form :inline="true">
            <el-form-item label="原销售单">
              <el-select v-model="refundOrderId" placeholder="选择已出库的销售单" filterable style="width: 260px" @change="loadRefundDetail">
                <el-option v-for="o in refundOrders" :key="o.ID" :label="`${o.orderNo}（¥ ${o.totalAmount?.toFixed(2)}）`" :value="o.ID" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button @click="loadRefundOrders">刷新</el-button>
            </el-form-item>
          </el-form>
          <div v-if="refundDetail.orderNo" class="mb-2 tip-text">
            原单 {{ refundDetail.orderNo }} · 剩余可退 <b>{{ remainingTotal }}</b> 件（退回商品仅限原单商品）
          </div>
          <el-table :data="refundItems" border size="small" max-height="360">
            <el-table-column label="SKU" width="160" prop="skuCode" />
            <el-table-column label="商品" min-width="120">
              <template #default="{ row }">{{ row.goodsName }}</template>
            </el-table-column>
            <el-table-column label="规格" width="110">
              <template #default="{ row }">{{ (row.color || '') + ' / ' + (row.size || '') }}</template>
            </el-table-column>
            <el-table-column label="原单数量" width="90" align="right" prop="outQty" />
            <el-table-column label="退款数量" width="130">
              <template #default="{ row }">
                <el-input-number v-model="row.qty" :min="0" :max="row.outQty" style="width: 100%" />
              </template>
            </el-table-column>
            <el-table-column label="退款金额" width="110" align="right">
              <template #default="{ row }">-¥ {{ ((row.qty || 0) * row.price).toFixed(2) }}</template>
            </el-table-column>
          </el-table>
          <div class="total-row mt-2">退款合计：<b class="refund-amount">-¥ {{ refundTotal.toFixed(2) }}</b></div>
          <el-button type="danger" size="large" class="pay-btn" :disabled="!refundOrderId || refundTotal <= 0" :loading="saving" @click="handleRefund">
            确认退款 ¥ {{ refundTotal.toFixed(2) }}
          </el-button>
        </el-card>
      </el-tab-pane>

      <!-- ==================== 换货 ==================== -->
      <el-tab-pane label="换货" name="exchange">
        <div class="cashier">
          <div class="left">
            <el-card shadow="never" class="mb-4">
              <el-form :inline="true">
                <el-form-item label="原销售单">
                  <el-select v-model="exchangeOrderId" placeholder="选择已出库的销售单" filterable style="width: 240px" @change="loadExchangeDetail">
                    <el-option v-for="o in refundOrders" :key="o.ID" :label="`${o.orderNo}（¥ ${o.totalAmount?.toFixed(2)}）`" :value="o.ID" />
                  </el-select>
                </el-form-item>
                <el-form-item label="换出仓库">
                  <el-select v-model="exchangeWarehouseId" placeholder="默认原单仓库" clearable style="width: 150px">
                    <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
                  </el-select>
                </el-form-item>
              </el-form>
            </el-card>
            <!-- 退回明细 -->
            <el-card shadow="never" class="mb-4">
              <template #header>退回商品（原单商品，入库）</template>
              <el-table :data="exchangeReturnItems" border size="small" max-height="200">
                <el-table-column label="SKU" width="150" prop="skuCode" />
                <el-table-column label="规格" width="100">
                  <template #default="{ row }">{{ (row.color || '') + ' / ' + (row.size || '') }}</template>
                </el-table-column>
                <el-table-column label="原单数量" width="80" align="right" prop="outQty" />
                <el-table-column label="退回数量" width="110">
                  <template #default="{ row }"><el-input-number v-model="row.qty" :min="0" :max="row.outQty" size="small" style="width: 100%" /></template>
                </el-table-column>
              </el-table>
              <div class="tip-text mt-2">退回合计：-¥ {{ exchangeReturnTotal.toFixed(2) }}</div>
            </el-card>
            <!-- 换出商品 -->
            <el-card shadow="never" class="goods-card">
              <template #header>换出商品（新商品，出库）</template>
              <el-input v-model="exchangeKeyword" placeholder="搜索商品名称 / SKU编码，回车" clearable size="small" class="mb-2" @keyup.enter="searchExchangeGoods" @clear="searchExchangeGoods" />
              <el-empty v-if="!exchangeGoods.length" description="输入关键词搜索商品" />
              <div v-else class="goods-grid">
                <div v-for="s in exchangeGoods" :key="s.ID" class="goods-item small" :class="{ disabled: (s.available || 0) <= 0 }" @click="addExchange(s)">
                  <div class="goods-sku">{{ s.skuCode }}</div>
                  <div class="goods-spec">{{ s.color }} / {{ s.size }}</div>
                  <div class="goods-price">¥ {{ (s.salePrice || 0).toFixed(2) }}</div>
                  <div class="goods-stock">可售 {{ s.available ?? '-' }}</div>
                </div>
              </div>
            </el-card>
          </div>
          <div class="right">
            <el-card shadow="never">
              <template #header>换出购物车</template>
              <el-table :data="exchangeCart" border size="small" max-height="220">
                <el-table-column label="商品" min-width="130">
                  <template #default="{ row }">
                    <div>{{ row.skuCode }}</div>
                    <div class="tip-text">{{ row.color }} / {{ row.size }}</div>
                  </template>
                </el-table-column>
                <el-table-column label="数量" width="100" align="center">
                  <template #default="{ row }"><el-input-number v-model="row.qty" :min="1" :max="row.available" size="small" /></template>
                </el-table-column>
                <el-table-column label="小计" width="90" align="right">
                  <template #default="{ row }">¥ {{ (row.salePrice * row.qty).toFixed(2) }}</template>
                </el-table-column>
                <el-table-column label="" width="46" align="center">
                  <template #default="{ $index }"><el-button link type="danger" @click="exchangeCart.splice($index, 1)">删</el-button></template>
                </el-table-column>
              </el-table>
              <div class="total-row">
                换出合计：<b class="total-amount">¥ {{ exchangeOutTotal.toFixed(2) }}</b>
                <div class="tip-text">差额（换出−退回）：<b :style="{ color: exchangeDiff >= 0 ? '#67c23a' : '#f56c6c' }">¥ {{ exchangeDiff.toFixed(2) }}</b>{{ exchangeDiff >= 0 ? '（应收）' : '（应退）' }}</div>
              </div>
              <el-form label-width="90px" class="pay-form">
                <el-form-item label="支付方式">
                  <el-radio-group v-model="exchangePayMethod">
                    <el-radio-button value="cash">现金</el-radio-button>
                    <el-radio-button value="wechat">微信</el-radio-button>
                    <el-radio-button value="alipay">支付宝</el-radio-button>
                  </el-radio-group>
                </el-form-item>
                <el-form-item v-if="exchangePayMethod === 'cash' && exchangeDiff > 0" label="实收金额">
                  <el-input-number v-model="exchangePaid" :min="exchangeDiff" :precision="2" style="width: 200px" />
                </el-form-item>
              </el-form>
              <el-button type="success" size="large" class="pay-btn" :disabled="!exchangeOrderId || !exchangeReturnTotal || !exchangeCart.length" :loading="saving" @click="handleExchange">
                确认换货
              </el-button>
            </el-card>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 结果弹窗 -->
    <el-dialog v-model="doneVisible" :title="doneTitle" width="440px" destroy-on-close>
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item v-for="line in doneLines" :key="line[0]" :label="line[0]">{{ line[1] }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button type="primary" @click="doneVisible = false">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { checkout, refund, exchange } from '@/api/jxc/cashier'
import { getSalePage, getSaleDetail, getSaleRemaining } from '@/api/jxc/sale'
import { getSkuList } from '@/api/jxc/goods'
import { getWarehouseList, getCustomerList } from '@/api/jxc/basic'
import { getStockPage } from '@/api/jxc/stock'

const activeTab = ref('cash')

// ===== 收款 =====
const keyword = ref('')
const goodsList = ref([])
const cart = ref([])
const loading = ref(false)
const saving = ref(false)
const warehouses = ref([])
const customers = ref([])
const warehouseId = ref(undefined)
const customerId = ref(undefined)
const payMethod = ref('cash')
const paidAmount = ref(0)

const skuStock = ref({})
const loadStock = async () => {
  const { data } = await getStockPage({ page: 1, pageSize: 999 })
  const map = {}
  ;(data.list || []).forEach((s) => { map[s.skuId] = (s.quantity || 0) - (s.lockQuantity || 0) })
  skuStock.value = map
}

const enrichGoods = (list) =>
  list.filter((s) => s.status === 1).slice(0, 50).map((s) => ({
    ...s, goodsName: s.goods?.name || '', available: skuStock.value[s.ID] ?? 0,
  }))

const searchGoods = async () => {
  loading.value = true
  try {
    const kw = keyword.value.trim().toLowerCase()
    const { data } = await getSkuList({ page: 1, pageSize: 999 })
    let list = (data || []).filter((s) => s.status === 1)
    if (kw) {
      list = list.filter((s) =>
        (s.skuCode || '').toLowerCase().includes(kw) ||
        (s.barcode || '').toLowerCase().includes(kw) ||
        (s.goods?.name || '').toLowerCase().includes(kw))
    }
    goodsList.value = enrichGoods(list)
  } finally {
    loading.value = false
  }
}

const addToCart = (sku) => {
  if ((sku.available || 0) <= 0) { ElMessage.warning('该 SKU 无可售库存'); return }
  const exist = cart.value.find((it) => it.skuId === sku.ID)
  if (exist) { exist.qty = Math.min(exist.qty + 1, exist.available); return }
  cart.value.push({ skuId: sku.ID, skuCode: sku.skuCode, color: sku.color, size: sku.size, salePrice: sku.salePrice || 0, qty: 1, available: sku.available })
}

const clearCart = () => { cart.value = []; paidAmount.value = 0 }
const totalAmount = computed(() => cart.value.reduce((s, it) => s + it.salePrice * it.qty, 0))
const change = computed(() => paidAmount.value - totalAmount.value)

const handleCheckout = async () => {
  if (!cart.value.length) return
  if (!warehouseId.value) { ElMessage.warning('请选择收银仓库'); return }
  saving.value = true
  try {
    const res = await checkout({
      warehouseId: warehouseId.value, customerId: customerId.value,
      payMethod: payMethod.value,
      paidAmount: payMethod.value === 'cash' ? paidAmount.value : totalAmount.value,
      items: cart.value.map((it) => ({ skuId: it.skuId, qty: it.qty })),
    })
    if (res && res.code !== 0) return
    const data = res.data || {}
    const changeAmt = payMethod.value === 'cash' ? (paidAmount.value - totalAmount.value) : 0
    doneTitle.value = '收款成功'
    doneLines.value = [
      ['销售单号', data.orderNo], ['应收金额', `¥ ${(data.totalAmount || 0).toFixed(2)}`],
      ['支付方式', payMethodText(payMethod.value)],
      ...(changeAmt > 0 ? [['找零', `¥ ${changeAmt.toFixed(2)}`]] : []),
    ]
    doneVisible.value = true
    clearCart()
    loadStock(); searchGoods()
  } finally { saving.value = false }
}

// ===== 退款 =====
const refundOrders = ref([])
const refundOrderId = ref(undefined)
const refundDetail = ref({})
const refundItems = ref([])
const remainingTotal = ref(0)

const loadRefundOrders = async () => {
  const { data } = await getSalePage({ page: 1, pageSize: 999 })
  refundOrders.value = (data.list || []).filter((o) => o.status === 2 && (o.orderType === 1 || o.orderType === 3))
}

const loadRefundDetail = async (id) => {
  refundItems.value = []
  remainingTotal.value = 0
  if (!id) return
  const [det, rem] = await Promise.all([getSaleDetail(id), getSaleRemaining(id)])
  const d = det.data || {}
  refundDetail.value = d
  remainingTotal.value = rem.data?.remaining ?? 0
  refundItems.value = (d.items || [])
    .filter((it) => d.orderType === 1 || it.direction === 1)
    .map((it) => ({ ...it, outQty: it.qty, qty: 0 }))
}

const refundTotal = computed(() => refundItems.value.reduce((s, it) => s + (it.qty || 0) * it.price, 0))

const handleRefund = async () => {
  if (!refundTotal.value) { ElMessage.warning('请填写退款数量'); return }
  saving.value = true
  try {
    const res = await refund({
      originalOrderId: refundOrderId.value,
      items: refundItems.value.filter((it) => it.qty > 0).map((it) => ({ skuId: it.skuId, qty: it.qty })),
    })
    if (res && res.code !== 0) return
    const data = res.data || {}
    doneTitle.value = '退款成功'
    doneLines.value = [
      ['退货单号', data.orderNo], ['退款金额', `¥ ${(data.totalAmount || 0).toFixed(2)}`], ['状态', '已入库（库存已回补）'],
    ]
    doneVisible.value = true
    refundOrderId.value = undefined
    refundDetail.value = {}
    refundItems.value = []
    loadRefundOrders(); loadStock()
  } finally { saving.value = false }
}

// ===== 换货 =====
const exchangeOrderId = ref(undefined)
const exchangeWarehouseId = ref(undefined)
const exchangeDetail = ref({})
const exchangeReturnItems = ref([])
const exchangeKeyword = ref('')
const exchangeGoods = ref([])
const exchangeCart = ref([])
const exchangePayMethod = ref('cash')
const exchangePaid = ref(0)

const loadExchangeDetail = async (id) => {
  exchangeReturnItems.value = []
  if (!id) return
  const det = await getSaleDetail(id)
  const d = det.data || {}
  exchangeDetail.value = d
  exchangeReturnItems.value = (d.items || [])
    .filter((it) => d.orderType === 1 || it.direction === 1)
    .map((it) => ({ ...it, outQty: it.qty, qty: 0 }))
}

const searchExchangeGoods = async () => {
  const kw = exchangeKeyword.value.trim().toLowerCase()
  const { data } = await getSkuList({ page: 1, pageSize: 999 })
  let list = (data || []).filter((s) => s.status === 1)
  if (kw) {
    list = list.filter((s) =>
      (s.skuCode || '').toLowerCase().includes(kw) ||
      (s.goods?.name || '').toLowerCase().includes(kw))
  }
  exchangeGoods.value = enrichGoods(list)
}

const addExchange = (sku) => {
  if ((sku.available || 0) <= 0) { ElMessage.warning('该 SKU 无可售库存'); return }
  const exist = exchangeCart.value.find((it) => it.skuId === sku.ID)
  if (exist) { exist.qty = Math.min(exist.qty + 1, exist.available); return }
  exchangeCart.value.push({ skuId: sku.ID, skuCode: sku.skuCode, color: sku.color, size: sku.size, salePrice: sku.salePrice || 0, qty: 1, available: sku.available })
}

const exchangeReturnTotal = computed(() => exchangeReturnItems.value.reduce((s, it) => s + (it.qty || 0) * it.price, 0))
const exchangeOutTotal = computed(() => exchangeCart.value.reduce((s, it) => s + it.salePrice * it.qty, 0))
const exchangeDiff = computed(() => exchangeOutTotal.value - exchangeReturnTotal.value)

const handleExchange = async () => {
  const returnItems = exchangeReturnItems.value.filter((it) => it.qty > 0)
  if (!returnItems.length) { ElMessage.warning('请填写退回数量'); return }
  if (!exchangeCart.value.length) { ElMessage.warning('请添加换出商品'); return }
  saving.value = true
  try {
    const res = await exchange({
      originalOrderId: exchangeOrderId.value,
      warehouseId: exchangeWarehouseId.value,
      payMethod: exchangePayMethod.value,
      paidAmount: exchangePayMethod.value === 'cash' ? exchangePaid.value : 0,
      returnItems: returnItems.map((it) => ({ skuId: it.skuId, qty: it.qty })),
      outItems: exchangeCart.value.map((it) => ({ skuId: it.skuId, qty: it.qty })),
    })
    if (res && res.code !== 0) return
    const d = res.data || {}
    const diff = d.diffAmount || 0
    doneTitle.value = '换货成功'
    doneLines.value = [
      ['退货单号', d.returnOrder?.orderNo], ['退回金额', `¥ ${(d.returnOrder?.totalAmount || 0).toFixed(2)}`],
      ['销售单号', d.outOrder?.orderNo], ['换出金额', `¥ ${(d.outOrder?.totalAmount || 0).toFixed(2)}`],
      ['差额', diff >= 0 ? `应收 ¥ ${diff.toFixed(2)}` : `应退 ¥ ${(-diff).toFixed(2)}`],
    ]
    doneVisible.value = true
    exchangeOrderId.value = undefined
    exchangeDetail.value = {}
    exchangeReturnItems.value = []
    exchangeCart.value = []
    exchangePaid.value = 0
    loadRefundOrders(); loadStock()
  } finally { saving.value = false }
}

// ===== 公共 =====
const doneVisible = ref(false)
const doneTitle = ref('')
const doneLines = ref([])

const payMethodText = (m) => (m === 'cash' ? '现金' : m === 'wechat' ? '微信' : '支付宝')

const loadOptions = async () => {
  const [w, cu] = await Promise.all([
    getWarehouseList({ page: 1, pageSize: 999 }),
    getCustomerList({ page: 1, pageSize: 999 }),
  ])
  warehouses.value = w.data.list || []
  customers.value = cu.data.list || []
  await loadStock()
  searchGoods()
  loadRefundOrders()
}

onMounted(loadOptions)
</script>

<style scoped>
.cashier { display: flex; gap: 16px; align-items: flex-start; }
.cashier-tabs { min-height: calc(100vh - 120px); }
.left { flex: 1; min-width: 0; }
.right { width: 440px; }
.mb-2 { margin-bottom: 8px; }
.mb-4 { margin-bottom: 16px; }
.mt-2 { margin-top: 8px; }
.goods-card { max-height: calc(100vh - 320px); overflow: auto; }
.goods-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 10px; }
.goods-item { border: 1px solid #e4e7ed; border-radius: 6px; padding: 10px; cursor: pointer; transition: all .2s; }
.goods-item:hover { border-color: #409eff; box-shadow: 0 2px 8px rgba(64,158,255,.2); }
.goods-item.disabled { opacity: .5; cursor: not-allowed; }
.goods-item.small { padding: 8px; }
.goods-name { font-weight: 600; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.goods-sku { color: #909399; font-size: 12px; margin-top: 2px; }
.goods-spec { color: #606266; font-size: 12px; }
.goods-price { color: #f56c6c; font-weight: 600; margin-top: 4px; }
.goods-stock { color: #909399; font-size: 12px; }
.flex-between { display: flex; justify-content: space-between; align-items: center; }
.tip-text { color: #909399; font-size: 12px; }
.total-row { text-align: right; margin: 12px 0 4px; font-size: 14px; }
.total-amount { color: #f56c6c; font-size: 20px; }
.refund-amount { color: #f56c6c; font-size: 20px; }
.pay-form { margin-top: 12px; }
.pay-btn { width: 100%; margin-top: 8px; font-size: 16px; height: 44px; }
</style>

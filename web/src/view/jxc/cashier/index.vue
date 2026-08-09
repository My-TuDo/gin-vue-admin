<template>
  <div class="cashier">
    <el-tabs v-model="activeTab" class="cashier-tabs" @tab-change="onTabChange">
      <!-- 顶部工具区：扫码枪入口（badge 显示待处理总数，全确认后自动隐藏） -->
      <template #extra>
        <el-badge :value="posPendingCount" :hidden="posPendingCount <= 0" :max="99" class="pos-gun-badge">
          <el-button :icon="Monitor" @click="posDrawerVisible = true">扫码枪</el-button>
        </el-badge>
      </template>
      <!-- ==================== 收款 ==================== -->
      <el-tab-pane label="收款" name="cash">
        <div class="pos-cash">
          <!-- 左侧：搜索 + 商品网格 -->
          <div class="pos-left">
            <div class="search-bar">
              <el-input
                v-model="keyword"
                ref="searchRef"
                placeholder="搜索商品名称 / SKU / 条码，回车确认（扫码枪可直接扫描）"
                clearable
                size="large"
                @keyup.enter="searchGoods"
                @clear="searchGoods"
              >
                <template #prefix><el-icon class="search-icon"><Search /></el-icon></template>
                <template #append><el-button :icon="Search" @click="searchGoods" /></template>
              </el-input>
            </div>
            <div class="goods-panel" v-loading="loading">
              <el-empty v-if="!goodsList.length" description="输入关键词或扫描条码搜索商品" :image-size="110" />
              <div v-else class="goods-scroll">
                <div class="goods-grid">
                  <div
                    v-for="s in goodsList"
                    :key="s.ID"
                    class="goods-card"
                    :class="{ disabled: (s.available || 0) <= 0 }"
                    @click="addToCart(s)"
                  >
                    <span v-if="cartQtyMap[s.ID]" class="g-badge">{{ cartQtyMap[s.ID] }}</span>
                    <div class="g-name">{{ s.goodsName || s.skuCode }}</div>
                    <div class="g-meta">
                      {{ s.skuCode }}<template v-if="s.color || s.size"> · {{ s.color }} / {{ s.size }}</template>
                    </div>
                    <div class="g-foot">
                      <div class="g-price">¥<span class="int">{{ moneyParts(s.salePrice)[0] }}</span><span class="dec">{{ moneyParts(s.salePrice)[1] }}</span></div>
                      <span class="g-stock" :class="stockInfo(s).cls">{{ stockInfo(s).text }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <!-- 右侧：购物车 + 结算 -->
          <div class="pos-right">
            <div class="cart-panel">
              <div class="cart-head">
                <span class="cart-title">购物车<i>{{ cart.length }} 项</i></span>
                <el-button link type="danger" @click="clearCart">清空</el-button>
              </div>
              <div class="cart-scroll">
                <el-empty v-if="!cart.length" description="点击左侧商品加入购物车" :image-size="70" />
                <div v-for="(row, idx) in cart" :key="row.skuId" class="cart-item">
                  <div class="ci-main">
                    <div class="ci-name">{{ row.skuCode }}</div>
                    <div class="ci-spec">{{ row.color }} / {{ row.size }}</div>
                  </div>
                  <div class="ci-right">
                    <div class="ci-price">¥ {{ row.salePrice.toFixed(2) }}</div>
                    <QtyStepper v-model="row.qty" :min="1" :max="row.available" />
                    <div class="ci-sub">¥ {{ (row.salePrice * row.qty).toFixed(2) }}</div>
                    <el-icon class="ci-del" @click="cart.splice(idx, 1)"><Delete /></el-icon>
                  </div>
                </div>
              </div>
            </div>
            <div class="settle">
              <div class="total-line">
                <span>应收合计</span>
                <div class="total-amount">¥<span class="int">{{ moneyParts(totalAmount)[0] }}</span><span class="dec">{{ moneyParts(totalAmount)[1] }}</span></div>
              </div>
              <div class="settle-fields">
                <div class="field">
                  <span class="f-label">收银仓库 <b class="req">*</b></span>
                  <el-select v-model="warehouseId" placeholder="选择仓库" size="large">
                    <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
                  </el-select>
                </div>
                <div class="field">
                  <span class="f-label">客户</span>
                  <el-select v-model="customerId" placeholder="散客" clearable filterable size="large">
                    <el-option v-for="cu in customers" :key="cu.ID" :label="cu.name" :value="cu.ID" />
                  </el-select>
                </div>
              </div>
              <div class="pay-methods">
                <span class="f-label">支付方式</span>
                <el-radio-group v-model="payMethod" class="pay-radio">
                  <el-radio-button value="cash"><b>现金</b><em>F1</em></el-radio-button>
                  <el-radio-button value="wechat"><b>微信</b><em>F2</em></el-radio-button>
                  <el-radio-button value="alipay"><b>支付宝</b><em>F3</em></el-radio-button>
                </el-radio-group>
              </div>
              <div v-if="payMethod === 'cash'" class="cash-row">
                <div class="field">
                  <span class="f-label">实收金额</span>
                  <el-input-number
                    v-model="paidAmount"
                    :min="totalAmount"
                    :precision="2"
                    size="large"
                    controls-position="right"
                    class="paid-input"
                  />
                </div>
                <div class="change-box">
                  <span>找零</span>
                  <b>¥ {{ Math.max(change, 0).toFixed(2) }}</b>
                </div>
              </div>
              <el-button
                type="success"
                size="large"
                class="pay-btn"
                :disabled="!cart.length || !warehouseId"
                :loading="saving"
                @click="handleCheckout"
              >
                收款<b>¥ {{ totalAmount.toFixed(2) }}</b>
              </el-button>
              <div class="shortcut-hint">F1 现金 · F2 微信 · F3 支付宝 · Ctrl + Enter 快捷收款</div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- ==================== 退款 ==================== -->
      <el-tab-pane label="退款" name="refund">
        <div class="refund-page">
          <div class="refund-top">
            <div class="order-select">
              <div class="field">
                <span class="f-label">原销售单</span>
                <el-select v-model="refundOrderId" placeholder="选择已出库的销售单" filterable size="large" @change="loadRefundDetail" @focus="loadRefundOrders">
                  <el-option v-for="o in refundOrders" :key="o.ID" :label="`${o.orderNo}（剩 ${o.remaining} 件·¥ ${o.totalAmount?.toFixed(2)}）`" :value="o.ID" />
                </el-select>
              </div>
              <el-button size="large" @click="loadRefundOrders"><el-icon class="btn-icon"><Refresh /></el-icon>刷新</el-button>
            </div>
            <div v-if="refundDetail.orderNo" class="refund-info">
              <span>原单 <b>{{ refundDetail.orderNo }}</b></span>
              <span>剩余可退 <b class="remain">{{ remainingTotal }}</b> 件</span>
              <span class="tip">退回商品仅限原单商品</span>
            </div>
          </div>
          <div class="refund-body">
            <el-empty v-if="!refundDetail.orderNo" description="请先选择原销售单" :image-size="110" />
            <template v-else>
              <div v-for="(row, idx) in refundItems" :key="idx" class="refund-item">
                <div class="ri-main">
                  <div class="ri-name">{{ row.goodsName }}</div>
                  <div class="ri-spec">{{ row.skuCode }}<template v-if="row.color || row.size"> · {{ row.color }} / {{ row.size }}</template></div>
                </div>
                <div class="ri-right">
                  <span class="ri-orig">原单 <b>{{ row.outQty }}</b> 件</span>
                  <QtyStepper v-model="row.qty" :min="0" :max="row.outQty" />
                  <div class="ri-amount">-¥ {{ ((row.qty || 0) * row.price).toFixed(2) }}</div>
                </div>
              </div>
              <div class="refund-foot">
                <div class="refund-total">退款合计<b>-¥ {{ refundTotal.toFixed(2) }}</b></div>
                <el-button
                  type="danger"
                  size="large"
                  class="pay-btn"
                  :disabled="!refundOrderId || refundTotal <= 0"
                  :loading="saving"
                  @click="handleRefund"
                >
                  确认退款<b>¥ {{ refundTotal.toFixed(2) }}</b>
                </el-button>
              </div>
            </template>
          </div>
        </div>
      </el-tab-pane>

      <!-- ==================== 换货 ==================== -->
      <el-tab-pane label="换货" name="exchange">
        <div class="pos-cash">
          <div class="pos-left">
            <div class="exchange-top">
              <div class="field">
                <span class="f-label">原销售单</span>
                <el-select v-model="exchangeOrderId" placeholder="选择已出库的销售单" filterable size="large" @change="loadExchangeDetail" @focus="loadRefundOrders">
                  <el-option v-for="o in refundOrders" :key="o.ID" :label="`${o.orderNo}（剩 ${o.remaining} 件）`" :value="o.ID" />
                </el-select>
              </div>
              <div class="field">
                <span class="f-label">换出仓库</span>
                <el-select v-model="exchangeWarehouseId" placeholder="默认原单仓库" clearable size="large">
                  <el-option v-for="w in warehouses" :key="w.ID" :label="w.name" :value="w.ID" />
                </el-select>
              </div>
            </div>
            <!-- 退回明细 -->
            <div class="panel return-panel">
              <div class="panel-title">退回商品<i>（原单商品 · 入库）</i></div>
              <div v-if="!exchangeReturnItems.length" class="panel-empty">请先选择原销售单</div>
              <div v-else class="return-list">
                <div v-for="(row, idx) in exchangeReturnItems" :key="idx" class="return-item">
                  <div class="ri-main">
                    <div class="ri-name">{{ row.goodsName }}</div>
                    <div class="ri-spec">{{ row.skuCode }}<template v-if="row.color || row.size"> · {{ row.color }} / {{ row.size }}</template></div>
                  </div>
                  <div class="ri-right">
                    <span class="ri-orig">原单 <b>{{ row.outQty }}</b> 件</span>
                    <QtyStepper v-model="row.qty" :min="0" :max="row.outQty" />
                    <div class="ri-amount">-¥ {{ ((row.qty || 0) * row.price).toFixed(2) }}</div>
                  </div>
                </div>
              </div>
              <div class="return-total">退回合计<b>-¥ {{ exchangeReturnTotal.toFixed(2) }}</b></div>
            </div>
            <!-- 换出商品 -->
            <div class="panel exchange-goods">
              <div class="panel-title">换出商品<i>（新商品 · 出库）</i></div>
              <el-input
                v-model="exchangeKeyword"
                ref="exchangeSearchRef"
                placeholder="搜索商品名称 / SKU编码，回车"
                clearable
                size="large"
                @keyup.enter="searchExchangeGoods"
                @clear="searchExchangeGoods"
              >
                <template #prefix><el-icon class="search-icon"><Search /></el-icon></template>
                <template #append><el-button :icon="Search" @click="searchExchangeGoods" /></template>
              </el-input>
              <div class="goods-scroll">
                <el-empty v-if="!exchangeGoods.length" description="输入关键词搜索换出商品" :image-size="80" />
                <div v-else class="goods-grid">
                  <div
                    v-for="s in exchangeGoods"
                    :key="s.ID"
                    class="goods-card"
                    :class="{ disabled: (s.available || 0) <= 0 }"
                    @click="addExchange(s)"
                  >
                    <span v-if="exchangeCartQtyMap[s.ID]" class="g-badge">{{ exchangeCartQtyMap[s.ID] }}</span>
                    <div class="g-name">{{ s.goodsName || s.skuCode }}</div>
                    <div class="g-meta">
                      {{ s.skuCode }}<template v-if="s.color || s.size"> · {{ s.color }} / {{ s.size }}</template>
                    </div>
                    <div class="g-foot">
                      <div class="g-price">¥<span class="int">{{ moneyParts(s.salePrice)[0] }}</span><span class="dec">{{ moneyParts(s.salePrice)[1] }}</span></div>
                      <span class="g-stock" :class="stockInfo(s).cls">{{ stockInfo(s).text }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="pos-right">
            <div class="cart-panel">
              <div class="cart-head">
                <span class="cart-title">换出购物车<i>{{ exchangeCart.length }} 项</i></span>
              </div>
              <div class="cart-scroll">
                <el-empty v-if="!exchangeCart.length" description="点击左侧商品加入换出清单" :image-size="70" />
                <div v-for="(row, idx) in exchangeCart" :key="row.skuId" class="cart-item">
                  <div class="ci-main">
                    <div class="ci-name">{{ row.skuCode }}</div>
                    <div class="ci-spec">{{ row.color }} / {{ row.size }}</div>
                  </div>
                  <div class="ci-right">
                    <div class="ci-price">¥ {{ row.salePrice.toFixed(2) }}</div>
                    <QtyStepper v-model="row.qty" :min="1" :max="row.available" />
                    <div class="ci-sub">¥ {{ (row.salePrice * row.qty).toFixed(2) }}</div>
                    <el-icon class="ci-del" @click="exchangeCart.splice(idx, 1)"><Delete /></el-icon>
                  </div>
                </div>
              </div>
            </div>
            <div class="settle">
              <div class="total-line">
                <span>换出合计</span>
                <div class="total-amount">¥<span class="int">{{ moneyParts(exchangeOutTotal)[0] }}</span><span class="dec">{{ moneyParts(exchangeOutTotal)[1] }}</span></div>
              </div>
              <div class="diff-line">
                <span>差额（换出 − 退回）</span>
                <b :style="{ color: exchangeDiff >= 0 ? '#67c23a' : '#f56c6c' }">
                  ¥ {{ exchangeDiff.toFixed(2) }}<i>{{ exchangeDiff >= 0 ? '（应收）' : '（应退）' }}</i>
                </b>
              </div>
              <div class="pay-methods">
                <span class="f-label">支付方式</span>
                <el-radio-group v-model="exchangePayMethod" class="pay-radio">
                  <el-radio-button value="cash"><b>现金</b></el-radio-button>
                  <el-radio-button value="wechat"><b>微信</b></el-radio-button>
                  <el-radio-button value="alipay"><b>支付宝</b></el-radio-button>
                </el-radio-group>
              </div>
              <div v-if="exchangePayMethod === 'cash' && exchangeDiff > 0" class="cash-row">
                <div class="field">
                  <span class="f-label">实收金额</span>
                  <el-input-number
                    v-model="exchangePaid"
                    :min="exchangeDiff"
                    :precision="2"
                    size="large"
                    controls-position="right"
                    class="paid-input"
                  />
                </div>
                <div class="change-box">
                  <span>找零</span>
                  <b>¥ {{ Math.max(exchangePaid - exchangeDiff, 0).toFixed(2) }}</b>
                </div>
              </div>
              <el-button
                type="success"
                size="large"
                class="pay-btn"
                :disabled="!exchangeOrderId || !exchangeReturnTotal || !exchangeCart.length"
                :loading="saving"
                @click="handleExchange"
              >
                确认换货
              </el-button>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 扫码枪抽屉：打开时启动 3s 轮询，关闭时停止；新条目出现时播放提示音 -->
    <el-drawer
      v-model="posDrawerVisible"
      title="扫码枪 · 待加入商品"
      size="420px"
      @open="startPosPolling"
      @close="stopPosPolling"
    >
      <div class="pos-drawer-body">
        <div class="pos-drawer-tip">店员在小程序扫码后自动出现在此，点击「加入」进入购物车并确认</div>
        <div v-if="posPending.length" class="pos-list">
          <div v-for="it in posPending" :key="it.id" class="pos-item">
            <div class="pi-main">
              <template v-if="it.sku">
                <div class="pi-name">{{ it.sku.goodsName || it.sku.skuCode }}</div>
                <div class="pi-spec">
                  {{ it.sku.skuCode }}<template v-if="it.sku.color || it.sku.size"> · {{ it.sku.color }} / {{ it.sku.size }}</template>
                </div>
                <div class="pi-price">¥ {{ (it.sku.salePrice || 0).toFixed(2) }} × {{ it.qty }}</div>
              </template>
              <template v-else>
                <div class="pi-name deleted">商品已删除</div>
                <div class="pi-spec">SKU 已被删除，点击「移除」确认清理</div>
              </template>
            </div>
            <el-button
              v-if="it.sku"
              size="small"
              type="primary"
              :disabled="(skuStock[it.sku.id] ?? 0) <= 0"
              @click="handlePosAdd(it)"
            >加入</el-button>
            <el-button v-else size="small" type="danger" plain @click="handlePosRemove(it)">移除</el-button>
          </div>
        </div>
        <el-empty v-else description="暂无待加入的扫码商品" :image-size="80" />
        <div class="pos-drawer-foot">
          <el-button type="success" size="large" :disabled="!posPending.length" @click="handlePosAddAll">全部加入</el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 成功结果弹窗 -->
    <ResultDialog v-model="doneVisible" :title="doneTitle" :lines="doneLines" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Delete, Refresh, Monitor } from '@element-plus/icons-vue'
import service from '@/utils/request'
import { checkout, refund, exchange, getRefundableOrders } from '@/api/jxc/cashier'
import { getSalePage, getSaleDetail, getSaleRemaining } from '@/api/jxc/sale'
import { getSkuList } from '@/api/jxc/goods'
import { getWarehouseList, getCustomerList } from '@/api/jxc/basic'
import { getStockPage } from '@/api/jxc/stock'
import QtyStepper from './components/QtyStepper.vue'
import ResultDialog from './components/ResultDialog.vue'

const activeTab = ref('cash')

// 切 tab 时加载对应数据：换货默认显示全部商品；退款刷新原单候选
const onTabChange = (name) => {
  if (name === 'exchange') searchExchangeGoods()
  if (name === 'refund') loadRefundOrders()
}

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

// 可退换原单（后端已过滤剩余 0 的单，下拉展开时自动刷新保证最新）
const loadRefundOrders = async () => {
  const { data } = await getRefundableOrders()
  refundOrders.value = data || []
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

// ===== 扫码枪（M3）：小程序扫码队列轮询 + 加入购物车 =====
const posDrawerVisible = ref(false)
const posPending = ref([])      // 待处理扫码条目 [{id, skuId, qty, sku:{id,skuCode,goodsName,color,size,salePrice}}]
const posPendingCount = ref(0)  // 待处理总数（抽屉按钮角标，全确认后隐藏）
let posPollTimer = null         // 3s 轮询定时器句柄
let posKnownIds = new Set()     // 上次已见条目 id 集合（用于检测"新出现"）
let posFirstLoad = false        // 首次打开标记：首拉不提示音，避免打开瞬间轰炸

// Web Audio API 生成短促 beep 提示音（无需音频文件）
let posAudioCtx = null
const playPosBeep = () => {
  try {
    const Ctx = window.AudioContext || window.webkitAudioContext
    if (!Ctx) return
    posAudioCtx = posAudioCtx || new Ctx()
    if (posAudioCtx.state === 'suspended') posAudioCtx.resume()
    const osc = posAudioCtx.createOscillator()
    const gain = posAudioCtx.createGain()
    osc.type = 'square'
    osc.frequency.value = 880
    gain.gain.setValueAtTime(0.001, posAudioCtx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.25, posAudioCtx.currentTime + 0.01)
    gain.gain.exponentialRampToValueAtTime(0.001, posAudioCtx.currentTime + 0.2)
    osc.connect(gain)
    gain.connect(posAudioCtx.destination)
    osc.start()
    osc.stop(posAudioCtx.currentTime + 0.22)
  } catch (e) {
    console.warn('[pos] 提示音播放失败', e)
  }
}

// 轮询加载待处理列表；isFirst=true 为打开抽屉时的首次拉取（不提示音）
const loadPosPending = async (isFirst = false) => {
  try {
    const res = await service.get('/jxc/pos/scan/pending', { donNotShowLoading: true })
    const list = Array.isArray(res.data) ? res.data : []
    posPending.value = list
    posPendingCount.value = list.length
    // 提示音触发条件：非首次加载，且出现了上次未见过的条目 id
    if (!isFirst && posFirstLoad) {
      const fresh = list.filter((it) => !posKnownIds.has(it.id))
      if (fresh.length) playPosBeep()
    }
    posKnownIds = new Set(list.map((it) => it.id))
    posFirstLoad = true
  } catch (e) {
    // 轮询失败静默，不影响页面其它功能
    console.warn('[pos] 扫码队列轮询失败', e)
  }
}

// 面板打开：立即拉取一次并启动 3s 轮询；重复调用不叠加定时器
const startPosPolling = () => {
  if (posPollTimer) return
  posFirstLoad = false
  loadPosPending(true)
  posPollTimer = setInterval(() => loadPosPending(false), 3000)
}

// 面板关闭 / 页面卸载：清理定时器，避免泄漏
const stopPosPolling = () => {
  if (posPollTimer) {
    clearInterval(posPollTimer)
    posPollTimer = null
  }
}

// 扫码条目加入购物车：复用现有 cart 数据结构，同 SKU 合并数量（totalAmount 由 computed 自动更新）
const posAddToCart = (it) => {
  const sku = it.sku
  if (!sku) return 0
  const available = skuStock.value[sku.id] ?? 0
  if (available <= 0) { ElMessage.warning('该 SKU 无可售库存'); return 0 }
  const added = Math.min(it.qty, available)
  const exist = cart.value.find((r) => r.skuId === sku.id)
  if (exist) exist.qty = Math.min(exist.qty + added, exist.available)
  else cart.value.push({ skuId: sku.id, skuCode: sku.skuCode, color: sku.color, size: sku.size, salePrice: sku.salePrice || 0, qty: added, available })
  if (added < it.qty) ElMessage.warning(`库存不足，已加入 ${added} 件`)
  return added
}

// 加入单条：加入购物车 → 调 confirm 标记已处理 → 刷新列表
const handlePosAdd = async (it) => {
  if (!it.sku) { await handlePosRemove(it); return }
  if (posAddToCart(it) <= 0) return
  try {
    await service.put('/jxc/pos/scan/confirm', { ids: [it.id] })
  } catch (e) {
    console.warn('[pos] 确认扫码条目失败', e)
  } finally {
    loadPosPending(false)
  }
}

// 移除（sku 为 null 的条目）：直接 confirm 掉，从队列清理
const handlePosRemove = async (it) => {
  try {
    await service.put('/jxc/pos/scan/confirm', { ids: [it.id] })
  } catch (e) {
    console.warn('[pos] 移除扫码条目失败', e)
  } finally {
    loadPosPending(false)
  }
}

// 全部加入：逐条加入购物车（含删除条目一并 confirm 清理）→ 批量确认 → 刷新
const handlePosAddAll = async () => {
  const ids = posPending.value.map((it) => it.id)
  if (!ids.length) return
  posPending.value.forEach((it) => { posAddToCart(it) })
  try {
    await service.put('/jxc/pos/scan/confirm', { ids })
  } catch (e) {
    console.warn('[pos] 批量确认扫码条目失败', e)
  } finally {
    loadPosPending(false)
  }
}

// ===== POS 辅助（纯展示，不改业务逻辑） =====
const searchRef = ref(null)
const exchangeSearchRef = ref(null)

// 商品卡片角标：购物车内已选数量
const cartQtyMap = computed(() => {
  const m = {}
  cart.value.forEach((it) => { m[it.skuId] = (m[it.skuId] || 0) + it.qty })
  return m
})
const exchangeCartQtyMap = computed(() => {
  const m = {}
  exchangeCart.value.forEach((it) => { m[it.skuId] = (m[it.skuId] || 0) + it.qty })
  return m
})

// 价格大数字：整数/小数分段显示
const moneyParts = (n) => {
  const s = (Number(n) || 0).toFixed(2)
  return [s.split('.')[0], s.split('.')[1]]
}

// 库存状态徽标
const stockInfo = (s) => {
  const a = s.available ?? 0
  if (a <= 0) return { text: '缺货', cls: 'out' }
  if (a <= 10) return { text: `仅 ${a} 件`, cls: 'low' }
  return { text: `可售 ${a}`, cls: 'ok' }
}

// 快捷键：F1/F2/F3 切换支付方式；Ctrl+Enter 快捷收款
const onGlobalKeydown = (e) => {
  if (doneVisible.value) return
  if (e.key === 'F1' || e.key === 'F2' || e.key === 'F3') {
    if (activeTab.value === 'cash') {
      e.preventDefault()
      payMethod.value = e.key === 'F1' ? 'cash' : e.key === 'F2' ? 'wechat' : 'alipay'
    }
    return
  }
  if (e.ctrlKey && (e.key === 'Enter' || e.code === 'Enter')) {
    const t = e.target
    const typing = t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable)
    if (!typing && activeTab.value === 'cash' && cart.value.length && warehouseId.value) {
      e.preventDefault()
      handleCheckout()
    }
  }
}

watch(activeTab, (v) => {
  nextTick(() => {
    if (v === 'cash') searchRef.value?.focus()
    else if (v === 'exchange') exchangeSearchRef.value?.focus()
  })
})

onMounted(() => {
  loadOptions()
  window.addEventListener('keydown', onGlobalKeydown)
  searchRef.value?.focus()
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKeydown)
  stopPosPolling()
})
</script>

<style scoped>
.cashier { padding: 2px 0; }
.cashier-tabs {
  height: calc(100vh - 150px);
  min-height: 540px;
  display: flex;
  flex-direction: column;
}
.cashier-tabs :deep(.el-tabs__header) { margin-bottom: 10px; }
.cashier-tabs :deep(.el-tabs__content) { flex: 1; min-height: 0; overflow: hidden; }
.cashier-tabs :deep(.el-tab-pane) { height: 100%; overflow: hidden; }
.cashier-tabs :deep(.el-tabs__item) { font-size: 15px; font-weight: 600; }

/* ===== 布局骨架 ===== */
.pos-cash { height: 100%; display: flex; gap: 12px; }
.pos-left { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 10px; }
.pos-right { width: 436px; min-width: 0; display: flex; flex-direction: column; gap: 10px; }

.search-bar { background: #fff; border-radius: 12px; padding: 12px; box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05); }
.search-bar :deep(.el-input__wrapper) { border-radius: 8px; }
.search-icon { color: #909399; }

.goods-panel {
  flex: 1;
  min-height: 0;
  background: #fff;
  border-radius: 12px;
  padding: 12px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.goods-scroll { flex: 1; min-height: 0; overflow: auto; }

/* ===== 商品卡片 ===== */
.goods-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(148px, 1fr)); gap: 10px; padding: 2px 4px 6px; }
.goods-card {
  position: relative;
  background: #fff;
  border: 1px solid #e8ecf1;
  border-radius: 10px;
  padding: 10px 12px;
  cursor: pointer;
  transition: all 0.18s;
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-height: 92px;
}
.goods-card:hover { border-color: #409eff; box-shadow: 0 4px 14px rgba(64, 158, 255, 0.18); transform: translateY(-2px); }
.goods-card:active { transform: scale(0.98); }
.goods-card.disabled { opacity: 0.45; cursor: not-allowed; }
.goods-card.disabled:hover { transform: none; box-shadow: none; border-color: #e8ecf1; }
.g-name {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.g-meta { font-size: 11px; color: #909399; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.g-foot { margin-top: auto; display: flex; justify-content: space-between; align-items: flex-end; gap: 6px; }
.g-price { color: #ff5a1f; font-weight: 800; line-height: 1; }
.g-price .int { font-size: 20px; }
.g-price .dec { font-size: 12px; }
.g-price .dec::before { content: '.'; }
.g-stock { font-size: 11px; padding: 2px 8px; border-radius: 20px; flex-shrink: 0; }
.g-stock.ok { color: #67c23a; background: #f0f9eb; }
.g-stock.low { color: #e6a23c; background: #fdf6ec; }
.g-stock.out { color: #909399; background: #f4f4f5; }
.g-badge {
  position: absolute;
  top: -6px;
  right: -6px;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  background: #409eff;
  color: #fff;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 700;
  line-height: 20px;
  text-align: center;
  box-shadow: 0 2px 6px rgba(64, 158, 255, 0.4);
  animation: badge-in 0.2s ease;
}
@keyframes badge-in { from { transform: scale(0.4); } to { transform: scale(1); } }

/* ===== 购物车面板 ===== */
.cart-panel {
  flex: 1;
  min-height: 0;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.cart-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 14px;
  border-bottom: 1px solid #f0f2f5;
}
.cart-title { font-size: 15px; font-weight: 700; color: #303133; }
.cart-title i { font-style: normal; color: #909399; font-size: 13px; font-weight: 500; margin-left: 4px; }
.cart-scroll { flex: 1; min-height: 0; overflow: auto; padding: 4px 8px; }
.cart-scroll :deep(.el-empty) { padding: 26px 0; }
.cart-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  padding: 9px 8px;
  border-bottom: 1px dashed #f0f2f5;
}
.cart-item:last-child { border-bottom: none; }
.cart-item:hover { background: #f7faff; border-radius: 8px; }
.ci-main { min-width: 0; flex: 1; }
.ci-name { font-size: 13px; font-weight: 600; color: #303133; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ci-spec { font-size: 11px; color: #909399; margin-top: 1px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ci-right { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.ci-price { font-size: 13px; color: #606266; width: 66px; text-align: right; }
.ci-sub { font-size: 14px; font-weight: 700; color: #ff5a1f; width: 72px; text-align: right; }
.ci-del { color: #c0c4cc; cursor: pointer; transition: color 0.15s; }
.ci-del:hover { color: #f56c6c; }

/* ===== 结算区 ===== */
.settle {
  background: #fff;
  border-radius: 12px;
  padding: 12px 14px 14px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
}
.total-line { display: flex; justify-content: space-between; align-items: flex-end; padding: 2px 0 4px; }
.total-line > span { color: #606266; font-weight: 600; font-size: 14px; padding-bottom: 3px; }
.total-amount { color: #ff5a1f; font-size: 27px; font-weight: 800; line-height: 1.15; }
.total-amount .int { font-size: 27px; }
.total-amount .dec { font-size: 15px; }
.total-amount .dec::before { content: '.'; }

.settle-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-top: 10px; }
.field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.field :deep(.el-select) { width: 100%; }
.field :deep(.el-input-number) { width: 100%; }
.f-label { font-size: 12px; color: #606266; font-weight: 500; }
.f-label .req { color: #f56c6c; font-size: 12px; }

.pay-methods { margin-top: 12px; }
.pay-methods .f-label { display: block; margin-bottom: 6px; }
.pay-radio { display: flex; width: 100%; gap: 8px; }
.pay-radio :deep(.el-radio-button) { flex: 1; }
.pay-radio :deep(.el-radio-button__inner) {
  width: 100%;
  height: 46px;
  line-height: 46px;
  padding: 0;
  font-size: 15px;
  border: 1px solid #dcdfe6;
  border-radius: 10px !important;
  box-shadow: none !important;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: all 0.15s;
}
.pay-radio :deep(.el-radio-button__inner b) { font-size: 16px; font-weight: 700; }
.pay-radio :deep(.el-radio-button__inner em) { font-style: normal; font-size: 11px; opacity: 0.6; }
.pay-radio :deep(.el-radio-button.is-active .el-radio-button__inner) {
  background: linear-gradient(135deg, #409eff, #337ecc);
  border-color: #409eff;
  color: #fff;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.35) !important;
}
.pay-radio :deep(.el-radio-button.is-active .el-radio-button__inner em) { opacity: 0.85; }

.cash-row { display: flex; gap: 10px; align-items: center; margin-top: 12px; }
.cash-row .field { flex: 1; }
.cash-row :deep(.el-input-number) { width: 100%; }
.change-box { width: 116px; text-align: center; background: #fdf6ec; border-radius: 10px; padding: 6px 0; flex-shrink: 0; }
.change-box span { font-size: 11px; color: #909399; display: block; }
.change-box b { color: #e6a23c; font-size: 17px; }

.pay-btn {
  width: 100%;
  height: 52px;
  font-size: 17px;
  font-weight: 600;
  margin-top: 12px;
  border-radius: 10px;
  letter-spacing: 1px;
}
.pay-btn b { font-size: 19px; margin-left: 4px; font-weight: 700; }
.pay-btn.el-button--success:not(.is-plain):not(.is-text) {
  background: linear-gradient(135deg, #67c23a, #4da32b);
  border: none;
  box-shadow: 0 4px 14px rgba(103, 194, 58, 0.35);
}
.pay-btn.el-button--success:not(.is-plain):not(.is-text):not(.is-disabled):hover {
  box-shadow: 0 6px 18px rgba(103, 194, 58, 0.45);
  transform: translateY(-1px);
}
.pay-btn.el-button--danger:not(.is-plain):not(.is-text) {
  background: linear-gradient(135deg, #f56c6c, #e04848);
  border: none;
  box-shadow: 0 4px 14px rgba(245, 108, 108, 0.35);
}
.shortcut-hint { margin-top: 8px; text-align: center; font-size: 11px; color: #c0c4cc; user-select: none; }

/* ===== 退款 ===== */
.refund-page { height: 100%; display: flex; flex-direction: column; gap: 12px; }
.refund-top {
  background: #fff;
  border-radius: 12px;
  padding: 14px 16px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
}
.order-select { display: flex; align-items: flex-end; gap: 12px; }
.order-select .field { width: 340px; }
.btn-icon { margin-right: 3px; }
.refund-info {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 20px;
  font-size: 13px;
  color: #606266;
  background: #f8fafc;
  padding: 9px 14px;
  border-radius: 8px;
}
.refund-info b { color: #303133; }
.refund-info .remain { color: #e6a23c; font-size: 16px; }
.refund-info .tip { color: #909399; font-size: 12px; }
.refund-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  background: #fff;
  border-radius: 12px;
  padding: 6px 18px 12px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
}
.refund-body :deep(.el-empty) { padding: 60px 0; }
.refund-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 12px 4px;
  border-bottom: 1px dashed #f0f2f5;
}
.ri-main { min-width: 0; flex: 1; }
.ri-name { font-size: 14px; font-weight: 600; color: #303133; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ri-spec { font-size: 11px; color: #909399; margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ri-right { display: flex; align-items: center; gap: 16px; flex-shrink: 0; }
.ri-orig { font-size: 12px; color: #909399; }
.ri-orig b { color: #606266; }
.ri-amount { font-size: 15px; font-weight: 700; color: #f56c6c; width: 92px; text-align: right; }
.refund-foot {
  position: sticky;
  bottom: 0;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 16px;
  padding: 14px 4px 6px;
  background: #fff;
}
.refund-foot .pay-btn { width: 260px; margin-top: 0; }
.refund-total { font-size: 14px; color: #606266; }
.refund-total b { color: #f56c6c; font-size: 22px; margin-left: 8px; }

/* ===== 换货 ===== */
.exchange-top {
  background: #fff;
  border-radius: 12px;
  padding: 12px 14px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
  display: flex;
  gap: 12px;
}
.exchange-top .field { flex: 1; }
.panel {
  background: #fff;
  border-radius: 12px;
  padding: 12px 14px;
  box-shadow: 0 2px 10px rgba(31, 45, 61, 0.05);
}
.return-panel { flex: 0 1 auto; max-height: 210px; display: flex; flex-direction: column; overflow: hidden; }
.return-list { flex: 1; min-height: 0; overflow: auto; }
.panel-title { font-size: 15px; font-weight: 700; color: #303133; margin-bottom: 8px; display: flex; align-items: baseline; gap: 6px; flex-shrink: 0; }
.panel-title i { font-style: normal; font-size: 12px; color: #909399; font-weight: 400; }
.panel-empty { text-align: center; color: #c0c4cc; font-size: 13px; padding: 22px 0; }
.return-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 2px;
  border-bottom: 1px dashed #f0f2f5;
}
.return-item .ri-right { gap: 12px; }
.return-item .ri-amount { width: 84px; }
.return-total { text-align: right; font-size: 12px; color: #909399; margin-top: 6px; flex-shrink: 0; }
.return-total b { color: #f56c6c; font-size: 15px; margin-left: 6px; }
.exchange-goods { flex: 1; min-height: 0; display: flex; flex-direction: column; gap: 10px; overflow: hidden; }
.exchange-goods .goods-scroll { flex: 1; min-height: 0; overflow: auto; }

.diff-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
  font-size: 13px;
  color: #606266;
  background: #f8fafc;
  border-radius: 8px;
  padding: 8px 10px;
}
.diff-line b { font-size: 15px; }
.diff-line b i { font-style: normal; font-size: 12px; margin-left: 4px; }

/* ===== 扫码枪抽屉 ===== */
.pos-gun-badge { margin-left: 10px; }
.pos-drawer-body { display: flex; flex-direction: column; height: 100%; }
.pos-drawer-tip {
  font-size: 12px;
  color: #909399;
  background: #f8fafc;
  border-radius: 8px;
  padding: 8px 10px;
  margin-bottom: 10px;
}
.pos-list { flex: 1; min-height: 0; overflow: auto; }
.pos-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 10px 4px;
  border-bottom: 1px dashed #f0f2f5;
}
.pos-item:last-child { border-bottom: none; }
.pos-item .pi-main { min-width: 0; flex: 1; }
.pos-item .pi-name { font-size: 14px; font-weight: 600; color: #303133; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pos-item .pi-name.deleted { color: #f56c6c; }
.pos-item .pi-spec { font-size: 11px; color: #909399; margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pos-item .pi-price { font-size: 13px; color: #ff5a1f; font-weight: 700; margin-top: 4px; }
.pos-drawer-foot { margin-top: 12px; padding-top: 12px; border-top: 1px solid #f0f2f5; flex-shrink: 0; }
.pos-drawer-foot .el-button { width: 100%; }
</style>

<template>
  <div class="cashier">
    <!-- 左侧：商品选择 -->
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
          <template #append>
            <el-button icon="Search" @click="searchGoods" />
          </template>
        </el-input>
      </el-card>
      <el-card shadow="never" class="goods-card" v-loading="loading">
        <el-empty v-if="!goodsList.length" description="输入关键词搜索商品" />
        <div v-else class="goods-grid">
          <div
            v-for="s in goodsList"
            :key="s.ID"
            class="goods-item"
            :class="{ disabled: (s.available || 0) <= 0 }"
            @click="addToCart(s)"
          >
            <div class="goods-name">{{ s.goodsName || s.skuCode }}</div>
            <div class="goods-sku">{{ s.skuCode }}</div>
            <div class="goods-spec">{{ s.color }} / {{ s.size }}</div>
            <div class="goods-price">¥ {{ (s.salePrice || 0).toFixed(2) }}</div>
            <div class="goods-stock">可售 {{ s.available ?? '-' }}</div>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 右侧：购物车与结算 -->
    <div class="right">
      <el-card shadow="never" class="cart-card">
        <template #header>
          <div class="flex-between">
            <span>购物车（{{ cart.length }} 项）</span>
            <el-button link type="danger" @click="clearCart">清空</el-button>
          </div>
        </template>
        <el-table :data="cart" border size="small" max-height="320">
          <el-table-column label="商品" min-width="140">
            <template #default="{ row }">
              <div>{{ row.skuCode }}</div>
              <div class="tip-text">{{ row.color }} / {{ row.size }}</div>
            </template>
          </el-table-column>
          <el-table-column label="单价" width="80" align="right">
            <template #default="{ row }">¥ {{ row.salePrice.toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="数量" width="120" align="center">
            <template #default="{ row }">
              <el-input-number v-model="row.qty" :min="1" :max="row.available" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="小计" width="90" align="right">
            <template #default="{ row }">¥ {{ (row.salePrice * row.qty).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="" width="50" align="center">
            <template #default="{ $index }">
              <el-button link type="danger" @click="cart.splice($index, 1)">删</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="total-row">
          合计：<b class="total-amount">¥ {{ totalAmount.toFixed(2) }}</b>
        </div>

        <!-- 结算区 -->
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

    <!-- 收款成功 -->
    <el-dialog v-model="doneVisible" title="收款成功" width="420px" destroy-on-close>
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item label="销售单号">{{ lastOrder.orderNo }}</el-descriptions-item>
        <el-descriptions-item label="应收金额">¥ {{ lastOrder.totalAmount?.toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item v-if="lastOrder.change > 0" label="找零">¥ {{ lastOrder.change.toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item label="支付方式">{{ payMethodText(lastOrder.payMethod) }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="doneVisible = false">继续收银</el-button>
        <el-button type="primary" @click="printReceipt">打印小票</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { checkout } from '@/api/jxc/cashier'
import { getSkuList } from '@/api/jxc/goods'
import { getWarehouseList, getCustomerList } from '@/api/jxc/basic'
import { getStockPage } from '@/api/jxc/stock'

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
const doneVisible = ref(false)
const lastOrder = ref({})

const skuStock = ref({})
const loadStock = async () => {
  const { data } = await getStockPage({ page: 1, pageSize: 999 })
  const map = {}
  ;(data.list || []).forEach((s) => {
    map[s.skuId] = (s.quantity || 0) - (s.lockQuantity || 0)
  })
  skuStock.value = map
}

const searchGoods = async () => {
  loading.value = true
  try {
    const kw = keyword.value.trim().toLowerCase()
    const { data } = await getSkuList({ page: 1, pageSize: 999 })
    let list = (data || []).filter((s) => s.status === 1)
    if (kw) {
      list = list.filter(
        (s) =>
          (s.skuCode || '').toLowerCase().includes(kw) ||
          (s.barcode || '').toLowerCase().includes(kw) ||
          (s.goods?.name || '').toLowerCase().includes(kw)
      )
    }
    goodsList.value = list.slice(0, 50).map((s) => ({
      ...s,
      goodsName: s.goods?.name || '',
      available: skuStock.value[s.ID] ?? 0,
    }))
  } finally {
    loading.value = false
  }
}

const addToCart = (sku) => {
  if ((sku.available || 0) <= 0) {
    ElMessage.warning('该 SKU 无可售库存')
    return
  }
  const exist = cart.value.find((it) => it.skuId === sku.ID)
  if (exist) {
    exist.qty = Math.min(exist.qty + 1, exist.available)
    return
  }
  cart.value.push({ skuId: sku.ID, skuCode: sku.skuCode, color: sku.color, size: sku.size, salePrice: sku.salePrice || 0, qty: 1, available: sku.available })
}

const clearCart = () => {
  cart.value = []
  paidAmount.value = 0
}

const totalAmount = computed(() => cart.value.reduce((s, it) => s + it.salePrice * it.qty, 0))
const change = computed(() => paidAmount.value - totalAmount.value)

const payMethodText = (m) => (m === 'cash' ? '现金' : m === 'wechat' ? '微信' : '支付宝')

const handleCheckout = async () => {
  if (!cart.value.length) return
  if (!warehouseId.value) {
    ElMessage.warning('请选择收银仓库')
    return
  }
  saving.value = true
  try {
    const res = await checkout({
      warehouseId: warehouseId.value,
      customerId: customerId.value,
      payMethod: payMethod.value,
      paidAmount: payMethod.value === 'cash' ? paidAmount.value : totalAmount.value,
      items: cart.value.map((it) => ({ skuId: it.skuId, qty: it.qty })),
    })
    if (res && res.code !== 0) return
    const data = res.data || {}
    lastOrder.value = { ...data, change: payMethod.value === 'cash' ? (paidAmount.value - totalAmount.value) : 0, payMethod: payMethod.value }
    doneVisible.value = true
    clearCart()
    loadStock()
    loadGoods()
  } finally {
    saving.value = false
  }
}

// 小票打印（浏览器打印区域）
const printReceipt = () => {
  const o = lastOrder.value
  const lines = [
    '      收 银 小 票',
    '------------------------------',
    `单号：${o.orderNo || ''}`,
    `支付：${payMethodText(o.payMethod)}`,
    `应收：¥ ${(o.totalAmount || 0).toFixed(2)}`,
    `找零：¥ ${(o.change || 0).toFixed(2)}`,
    '------------------------------',
    `时间：${new Date().toLocaleString()}`,
  ]
  const win = window.open('', '_blank', 'width=320,height=400')
  win.document.write(`<pre style="font-family:monospace;font-size:14px">${lines.join('\n')}</pre>`)
  win.document.close()
  win.print()
}

const loadGoods = () => {
  if (keyword.value.trim()) searchGoods()
}

const loadOptions = async () => {
  const [w, cu] = await Promise.all([
    getWarehouseList({ page: 1, pageSize: 999 }),
    getCustomerList({ page: 1, pageSize: 999 }),
  ])
  warehouses.value = w.data.list || []
  customers.value = cu.data.list || []
  await loadStock()
  searchGoods()
}

onMounted(loadOptions)
</script>

<style scoped>
.cashier { display: flex; gap: 16px; align-items: flex-start; }
.left { flex: 1; min-width: 0; }
.right { width: 480px; }
.mb-4 { margin-bottom: 16px; }
.goods-card { max-height: calc(100vh - 220px); overflow: auto; }
.goods-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 10px; }
.goods-item { border: 1px solid #e4e7ed; border-radius: 6px; padding: 10px; cursor: pointer; transition: all .2s; }
.goods-item:hover { border-color: #409eff; box-shadow: 0 2px 8px rgba(64,158,255,.2); }
.goods-item.disabled { opacity: .5; cursor: not-allowed; }
.goods-name { font-weight: 600; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.goods-sku { color: #909399; font-size: 12px; margin-top: 2px; }
.goods-spec { color: #606266; font-size: 12px; }
.goods-price { color: #f56c6c; font-weight: 600; margin-top: 4px; }
.goods-stock { color: #909399; font-size: 12px; }
.flex-between { display: flex; justify-content: space-between; align-items: center; }
.tip-text { color: #909399; font-size: 12px; }
.total-row { text-align: right; margin: 12px 0 4px; font-size: 14px; }
.total-amount { color: #f56c6c; font-size: 22px; }
.pay-form { margin-top: 12px; }
.pay-btn { width: 100%; margin-top: 8px; font-size: 18px; height: 48px; }
</style>

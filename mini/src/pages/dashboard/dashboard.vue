<template>
  <view class="ui-page">
    <!-- 销售趋势（折线） -->
    <view class="card ui-card">
      <view class="card-title ui-section-title">销售趋势（近 7 天）</view>
      <view v-if="trendLoading" class="ui-empty">加载中…</view>
      <view v-else-if="trendError" class="ui-err">
        <text class="ui-err-text">{{ trendError }}</text>
        <button class="ui-retry" size="mini" @click="loadTrend">重试</button>
      </view>
      <qiun-chart
        v-else
        type="line"
        :chart-data="trendData"
        :height="300"
        empty-text="近 7 天暂无销售数据"
      />
    </view>

    <!-- 热销 TOP5（柱状） -->
    <view class="card ui-card">
      <view class="card-title ui-section-title">热销 TOP5（近 30 天）</view>
      <view v-if="topLoading" class="ui-empty">加载中…</view>
      <view v-else-if="topError" class="ui-err">
        <text class="ui-err-text">{{ topError }}</text>
        <button class="ui-retry" size="mini" @click="loadTop">重试</button>
      </view>
      <qiun-chart
        v-else
        type="column"
        :chart-data="topData"
        :height="300"
        empty-text="近 30 天暂无销售记录"
      />
    </view>

    <!-- 分类销售占比（环形） -->
    <view class="card ui-card">
      <view class="card-title ui-section-title">分类销售占比（近 30 天）</view>
      <view v-if="catLoading" class="ui-empty">加载中…</view>
      <view v-else-if="catError" class="ui-err">
        <text class="ui-err-text">{{ catError }}</text>
        <button class="ui-retry" size="mini" @click="loadCat">重试</button>
      </view>
      <qiun-chart
        v-else
        type="ring"
        :chart-data="catData"
        :height="300"
        empty-text="近 30 天暂无销售记录"
      />
    </view>

    <!-- 库存预警列表 -->
    <view class="card ui-card">
      <view class="card-title ui-section-title">库存预警</view>
      <view v-if="alertLoading" class="ui-empty">加载中…</view>
      <view v-else-if="alertError" class="ui-err">
        <text class="ui-err-text">{{ alertError }}</text>
        <button class="ui-retry" size="mini" @click="loadAlerts">重试</button>
      </view>
      <view v-else-if="alerts.length" class="alert-list">
        <view v-for="(item, idx) in alerts" :key="idx" class="alert-item ui-cell">
          <view class="alert-main">
            <text class="alert-name">{{ item.goodsName || '未知商品' }}</text>
            <text class="alert-sku">{{ item.skuCode || '' }}</text>
          </view>
          <view class="alert-qty">
            <text class="qty-label">可售</text>
            <text class="qty-num" :class="{ danger: item.available <= 0 }">{{ fmtInt(item.available) }}</text>
            <text class="qty-safe">安全线 {{ fmtInt(item.safeStock) }}</text>
          </view>
        </view>
      </view>
      <view v-else class="ui-empty">暂无库存预警</view>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onPullDownRefresh } from '@dcloudio/uni-app'
import { get } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

const trendLoading = ref(false)
const trendError = ref('')
const trendData = ref({ categories: [], series: [] })

const topLoading = ref(false)
const topError = ref('')
const topData = ref({ categories: [], series: [] })

const catLoading = ref(false)
const catError = ref('')
const catData = ref({ categories: [], series: [] })

const alertLoading = ref(false)
const alertError = ref('')
const alerts = ref([])

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  loadAll()
})

onPullDownRefresh(async () => {
  await loadAll()
  uni.stopPullDownRefresh()
})

// 模块级防重锁：onLoad 与 onPullDownRefresh 可能并发触发（如数据加载中下拉刷新），
// 去重避免并发重复请求（社区方法论第 3 条：onShow/下拉刷新与加载中标志去重）。
let loadingAll = false

async function loadAll() {
  if (loadingAll) return
  loadingAll = true
  try {
    // 图表数据（trendData/topData/catData）为响应式绑定，更新后由 qiun-chart 组件内部
    // 的 watch 自动走 uCharts updateData 增量路径重绘（配合 qiun-chart 优化，见其源码注释）。
    await Promise.all([loadTrend(), loadTop(), loadCat(), loadAlerts()])
  } finally {
    loadingAll = false
  }
}

// ---------- 销售趋势 ----------
async function loadTrend() {
  trendLoading.value = true
  trendError.value = ''
  try {
    const data = await get('/jxc/dashboard/trend', { days: 7 })
    const list = normalize(data)
    trendData.value = {
      categories: list.map((it) => shortDate(it.date)),
      series: [
        { name: '销售额(元)', data: list.map((it) => round2(it.sales)) },
        { name: '订单数(单)', data: list.map((it) => toInt(it.orders)) }
      ]
    }
  } catch (e) {
    console.error('[dashboard] trend 请求失败', e)
    trendError.value = '趋势数据加载失败'
  } finally {
    trendLoading.value = false
  }
}

// ---------- 热销 TOP5 ----------
async function loadTop() {
  topLoading.value = true
  topError.value = ''
  try {
    const data = await get('/jxc/dashboard/top', { days: 30, limit: 5 })
    const list = normalize(data).slice(0, 5)
    topData.value = {
      categories: list.map((it) => shortName(it.goodsName || it.skuCode || '未知商品', 8)),
      series: [{ name: '销量(件)', data: list.map((it) => toInt(it.qty)) }]
    }
  } catch (e) {
    console.error('[dashboard] top 请求失败', e)
    topError.value = '热销数据加载失败'
  } finally {
    topLoading.value = false
  }
}

// ---------- 分类占比 ----------
async function loadCat() {
  catLoading.value = true
  catError.value = ''
  try {
    const data = await get('/jxc/dashboard/category', { days: 30 })
    const list = normalize(data)
    catData.value = {
      categories: list.map((it) => it.name || '未分类'),
      series: [{ name: '销售额(元)', data: list.map((it) => round2(it.amount)) }]
    }
  } catch (e) {
    console.error('[dashboard] category 请求失败', e)
    catError.value = '分类数据加载失败'
  } finally {
    catLoading.value = false
  }
}

// ---------- 库存预警 ----------
async function loadAlerts() {
  alertLoading.value = true
  alertError.value = ''
  try {
    const data = await get('/jxc/dashboard/stock-alert')
    const list = normalize(data)
    alerts.value = list.map((it) => ({
      skuCode: it.skuCode || it.sku || '',
      goodsName: it.goodsName || it.goods_name || it.name || '',
      available: firstNumber(it, ['available', 'quantity', 'stock']),
      safeStock: firstNumber(it, ['safeStock', 'safe_stock'])
    }))
  } catch (e) {
    console.error('[dashboard] stock-alert 请求失败', e)
    alertError.value = '库存预警加载失败'
  } finally {
    alertLoading.value = false
  }
}

// ---------- 工具 ----------
function normalize(data) {
  if (Array.isArray(data)) return data
  if (data && Array.isArray(data.list)) return data.list
  return []
}

function shortDate(d) {
  if (!d) return ''
  // 兼容 ISO 时间戳（如 2026-08-04T00:00:00+08:00）：先截断 T 之前
  const s = String(d).split('T')[0]
  const parts = s.split('-')
  if (parts.length === 3) return `${parts[1]}/${parts[2]}`
  return s.length > 5 ? s.slice(5) : s
}

function shortName(s, len) {
  const str = String(s || '')
  return str.length > len ? str.slice(0, len) + '…' : str
}

function round2(n) {
  const num = Number(n)
  return isNaN(num) ? 0 : Math.round(num * 100) / 100
}

function toInt(n) {
  const num = Number(n)
  return isNaN(num) ? 0 : Math.round(num)
}

function firstNumber(obj, keys) {
  if (!obj) return 0
  for (let i = 0; i < keys.length; i++) {
    const v = obj[keys[i]]
    if (v !== undefined && v !== null && v !== '') {
      const n = Number(v)
      return isNaN(n) ? 0 : n
    }
  }
  return 0
}

function fmtInt(n) {
  const num = Number(n) || 0
  return String(num).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}
</script>

<style lang="scss" scoped>
.card {
  background: $ui-bg-card;
  border-radius: $ui-radius-md;
  padding: $ui-space-md;
  margin-bottom: $ui-space-md;
  box-shadow: $ui-shadow-card;
}

.card-title {
  font-size: $ui-font-md;
  font-weight: 600;
  color: $ui-text-1;
  margin-bottom: 20rpx;
}

/* 库存预警 */
.alert-list {
  overflow: hidden;
}

.alert-item {
  padding: $ui-space-md 0;
  border-bottom: 1rpx solid $ui-border;
}

.alert-main {
  display: flex;
  flex-direction: column;
  flex: 1;
  margin-right: $ui-space-sm;
}

.alert-name {
  font-size: $ui-font-base;
  color: $ui-text-1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alert-sku {
  margin-top: 8rpx;
  font-size: $ui-font-xs;
  color: $ui-text-3;
}

.alert-qty {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.qty-label {
  font-size: 20rpx;
  color: $ui-text-3;
}

.qty-num {
  font-size: 34rpx;
  font-weight: 600;
  color: $ui-danger;
}

.qty-safe {
  font-size: 20rpx;
  color: $ui-text-3;
}
</style>

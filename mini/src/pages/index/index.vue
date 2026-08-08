<template>
  <view class="page">
    <!-- 概览卡片（今日/本周/本月） -->
    <view class="section">
      <view v-if="overviewLoading" class="block-tip">加载中…</view>
      <view v-else-if="overviewError" class="error-box">
        <text class="error-text">{{ overviewError }}</text>
        <button class="retry-btn" size="mini" @click="loadOverview">重试</button>
      </view>
      <view v-else class="cards">
        <view v-for="(item, idx) in overview" :key="idx" class="card">
          <view class="card-title">{{ item.label }}</view>
          <view class="card-row">
            <text class="card-k">销售额</text>
            <text class="card-v">¥{{ fmtMoney(item.sales) }}</text>
          </view>
          <view class="card-row">
            <text class="card-k">订单数</text>
            <text class="card-v">{{ fmtInt(item.orders) }} 单</text>
          </view>
          <view class="card-row">
            <text class="card-k">毛利</text>
            <text class="card-v" :class="{ warn: item.profit < 0 }">¥{{ fmtMoney(item.profit) }}</text>
          </view>
        </view>
      </view>
    </view>

    <!-- 库存预警列表 -->
    <view class="section">
      <view class="section-title">库存预警</view>
      <view v-if="alertLoading" class="block-tip">加载中…</view>
      <view v-else-if="alertError" class="error-box">
        <text class="error-text">{{ alertError }}</text>
        <button class="retry-btn" size="mini" @click="loadAlerts">重试</button>
      </view>
      <view v-else-if="alerts.length" class="alert-list">
        <view v-for="(item, idx) in alerts" :key="idx" class="alert-item">
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
      <view v-else class="empty">暂无库存预警</view>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onPullDownRefresh, onShow } from '@dcloudio/uni-app'
import { get } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

const overview = ref([])
const overviewLoading = ref(false)
const overviewError = ref('')

const alerts = ref([])
const alertLoading = ref(false)
const alertError = ref('')

// 登录态守卫 + 首次加载
onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  loadAll()
})

// 从登录页回跳后刷新数据（避免 onLoad 尚未完成时重复请求）
onShow(() => {
  if (
    isLoggedIn() &&
    !overviewLoading.value &&
    !alertLoading.value &&
    overview.value.length === 0 &&
    !overviewError.value
  ) {
    loadAll()
  }
})

onPullDownRefresh(async () => {
  await loadAll()
  uni.stopPullDownRefresh()
})

async function loadAll() {
  await Promise.all([loadOverview(), loadAlerts()])
}

async function loadOverview() {
  overviewLoading.value = true
  overviewError.value = ''
  try {
    const data = await get('/jxc/dashboard/overview')
    // 容错：兼容 {label,sales,orders,profit} 数组；若返回空则给占位
    const list = Array.isArray(data) ? data : data && Array.isArray(data.list) ? data.list : []
    const labels = ['今日', '本周', '本月']
    overview.value = list.map((it, i) => ({
      label: it.label || labels[i] || `周期${i + 1}`,
      sales: Number(it.sales) || 0,
      orders: Number(it.orders) || 0,
      profit: Number(it.profit) || 0
    }))
    if (!overview.value.length) {
      overviewError.value = '暂无经营数据'
    }
  } catch (e) {
    console.error('[index] overview 请求失败', e)
    overviewError.value = '概览数据加载失败'
  } finally {
    overviewLoading.value = false
  }
}

async function loadAlerts() {
  alertLoading.value = true
  alertError.value = ''
  try {
    const data = await get('/jxc/dashboard/stock-alert')
    // 容错：接口可能返回数组或 { list: [...] }
    const list = Array.isArray(data)
      ? data
      : data && Array.isArray(data.list)
        ? data.list
        : []
    alerts.value = list.map((it) => ({
      skuCode: it.skuCode || it.sku || '',
      goodsName: it.goodsName || it.goods_name || it.name || '',
      // 字段容错：available / quantity / stock；0 是合法值，不能用 || 串联
      available: firstNumber(it, ['available', 'quantity', 'stock']),
      safeStock: firstNumber(it, ['safeStock', 'safe_stock'])
    }))
  } catch (e) {
    console.error('[index] stock-alert 请求失败', e)
    alertError.value = '库存预警加载失败'
  } finally {
    alertLoading.value = false
  }
}

/** 按 keys 顺序取第一个非空数值（0 也会被保留） */
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

function fmtMoney(n) {
  const num = Number(n) || 0
  const fixed = num.toFixed(2)
  return fixed.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

function fmtInt(n) {
  const num = Number(n) || 0
  return String(num).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  padding: 24rpx;
  box-sizing: border-box;
}

.section {
  margin-bottom: 32rpx;
}

.section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
  margin-bottom: 20rpx;
}

.block-tip {
  padding: 60rpx 0;
  text-align: center;
  color: #999;
  font-size: 26rpx;
}

/* 概览卡片 */
.cards {
  display: flex;
  flex-direction: column;
}

.card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 24rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.card-title {
  font-size: 26rpx;
  color: #888;
  margin-bottom: 16rpx;
}

.card-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8rpx 0;
}

.card-k {
  font-size: 26rpx;
  color: #999;
}

.card-v {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;

  &.warn {
    color: #e64340;
  }
}

/* 库存预警 */
.alert-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
}

.alert-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}

.alert-main {
  display: flex;
  flex-direction: column;
  flex: 1;
  margin-right: 16rpx;
}

.alert-name {
  font-size: 28rpx;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alert-sku {
  margin-top: 8rpx;
  font-size: 22rpx;
  color: #999;
}

.alert-qty {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.qty-label {
  font-size: 20rpx;
  color: #999;
}

.qty-num {
  font-size: 34rpx;
  font-weight: 600;
  color: #e64340;

  &.danger {
    color: #e64340;
  }
}

.qty-safe {
  font-size: 20rpx;
  color: #b2b2b2;
}

.empty {
  padding: 60rpx 0;
  text-align: center;
  color: #999;
  font-size: 26rpx;
  background: #ffffff;
  border-radius: 16rpx;
}

/* 错误占位 */
.error-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60rpx 0;
  background: #ffffff;
  border-radius: 16rpx;
}

.error-text {
  font-size: 26rpx;
  color: #999;
  margin-bottom: 20rpx;
}

.retry-btn {
  background: #2b8a3e;
  color: #fff;
  border-radius: 8rpx;
  font-size: 24rpx;

  &::after {
    border: none;
  }
}
</style>

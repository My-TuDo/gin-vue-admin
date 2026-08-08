<template>
  <view class="page">
    <!-- 经营看板缩略卡片：点击进入看板详情页 -->
    <view class="dash-card" @click="goDashboard">
      <view class="dash-head">
        <text class="dash-title">📊 经营看板</text>
        <view v-if="alertCount > 0" class="badge">预警 {{ alertCount }}</view>
        <view v-else-if="alertError" class="badge badge-muted">预警 -</view>
      </view>

      <view v-if="overviewLoading" class="dash-tip">经营数据加载中…</view>
      <view v-else-if="overviewError" class="dash-tip">
        <text class="dash-error">{{ overviewError }}</text>
      </view>
      <view v-else-if="today" class="dash-grid">
        <view class="dash-item">
          <text class="dash-label">今日销售额</text>
          <text class="dash-value">¥{{ fmtMoney(today.sales) }}</text>
        </view>
        <view class="dash-item">
          <text class="dash-label">今日订单</text>
          <text class="dash-value">{{ fmtInt(today.orders) }} 单</text>
        </view>
        <view class="dash-item">
          <text class="dash-label">今日毛利</text>
          <text class="dash-value" :class="{ warn: today.profit < 0 }">¥{{ fmtMoney(today.profit) }}</text>
        </view>
      </view>
      <view v-else class="dash-tip">暂无经营数据</view>

      <view class="dash-foot">
        <text class="dash-more">点击查看完整看板 ›</text>
      </view>
    </view>

    <!-- 功能九宫格（3 列） -->
    <view class="section-title">功能</view>
    <view class="grid">
      <view
        v-for="item in menus"
        :key="item.key"
        class="grid-item"
        :class="{ disabled: item.disabled }"
        @click="onMenu(item)"
      >
        <view class="grid-icon">{{ item.icon }}</view>
        <text class="grid-name">{{ item.name }}</text>
        <text v-if="item.tag" class="grid-tag">{{ item.tag }}</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onPullDownRefresh, onShow } from '@dcloudio/uni-app'
import { get } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

const overviewLoading = ref(false)
const overviewError = ref('')
const today = ref(null)

const alertCount = ref(0)
const alertError = ref('')

let loadingAll = false
let loadedOnce = false

// 功能入口：type=tab 走 switchTab，type=page 走 navigateTo，disabled 为占位
const menus = [
  { key: 'scan', name: '扫码查库存', icon: '📷', type: 'tab', url: '/pages/stock/stock' },
  { key: 'dashboard', name: '经营看板', icon: '📊', type: 'page', url: '/pages/dashboard/dashboard' },
  { key: 'cashier', name: '收银', icon: '🧾', disabled: true, tag: '敬请期待' },
  { key: 'check', name: '盘点', icon: '📋', disabled: true },
  { key: 'io', name: '出入库', icon: '📦', disabled: true },
  { key: 'sales', name: '销售记录', icon: '🛒', disabled: true }
]

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  loadedOnce = true
  loadAll(true)
})

// 每次回到首页静默刷新，保证今日数据/预警徽标最新
onShow(() => {
  if (!isLoggedIn() || !loadedOnce) return
  loadAll(false)
})

onPullDownRefresh(async () => {
  await loadAll(false)
  uni.stopPullDownRefresh()
})

async function loadAll(showLoading) {
  if (loadingAll) return
  loadingAll = true
  try {
    if (showLoading) {
      overviewLoading.value = true
    }
    await Promise.all([loadOverview(), loadAlerts()])
  } catch (e) {
    console.error('[index] loadAll 失败', e)
  } finally {
    loadingAll = false
    overviewLoading.value = false
  }
}

async function loadOverview() {
  overviewError.value = ''
  try {
    const data = await get('/jxc/dashboard/overview')
    // 容错：数组或 { list: [...] }
    const list = Array.isArray(data) ? data : data && Array.isArray(data.list) ? data.list : []
    const item = list.find((it) => it && it.label === '今日')
    today.value = item
      ? {
          sales: Number(item.sales) || 0,
          orders: Number(item.orders) || 0,
          profit: Number(item.profit) || 0
        }
      : null
  } catch (e) {
    console.error('[index] overview 请求失败', e)
    overviewError.value = '经营数据加载失败'
  }
}

async function loadAlerts() {
  alertError.value = ''
  try {
    const data = await get('/jxc/dashboard/stock-alert')
    const list = Array.isArray(data) ? data : data && Array.isArray(data.list) ? data.list : []
    alertCount.value = list.length
  } catch (e) {
    console.error('[index] stock-alert 请求失败', e)
    alertError.value = '库存预警加载失败'
  }
}

function goDashboard() {
  uni.navigateTo({ url: '/pages/dashboard/dashboard' })
}

function onMenu(item) {
  if (item.disabled) {
    uni.showToast({ title: item.tag || 'M3 开发中，敬请期待', icon: 'none' })
    return
  }
  if (item.type === 'tab') {
    uni.switchTab({ url: item.url })
  } else {
    uni.navigateTo({ url: item.url })
  }
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

.section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
  margin: 8rpx 0 20rpx;
}

/* ---- 看板缩略卡片 ---- */
.dash-card {
  background: linear-gradient(135deg, #2b8a3e 0%, #3fb45c 100%);
  border-radius: 20rpx;
  padding: 28rpx 28rpx 20rpx;
  margin-bottom: 32rpx;
  box-shadow: 0 8rpx 24rpx rgba(43, 138, 62, 0.25);
}

.dash-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20rpx;
}

.dash-title {
  font-size: 32rpx;
  font-weight: 600;
  color: #fff;
}

.badge {
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  font-size: 22rpx;
  padding: 6rpx 18rpx;
  border-radius: 999rpx;
  font-weight: 600;
}

.badge-muted {
  opacity: 0.7;
}

.dash-tip {
  color: rgba(255, 255, 255, 0.85);
  font-size: 26rpx;
  padding: 30rpx 0;
  text-align: center;
}

.dash-error {
  font-size: 24rpx;
}

.dash-grid {
  display: flex;
}

.dash-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.dash-item + .dash-item {
  border-left: 1rpx solid rgba(255, 255, 255, 0.25);
}

.dash-label {
  font-size: 22rpx;
  color: rgba(255, 255, 255, 0.8);
  margin-bottom: 10rpx;
}

.dash-value {
  font-size: 30rpx;
  font-weight: 600;
  color: #fff;

  &.warn {
    color: #ffd9d9;
  }
}

.dash-foot {
  margin-top: 20rpx;
  padding-top: 16rpx;
  border-top: 1rpx solid rgba(255, 255, 255, 0.25);
  text-align: right;
}

.dash-more {
  font-size: 24rpx;
  color: rgba(255, 255, 255, 0.9);
}

/* ---- 功能九宫格 ---- */
.grid {
  display: flex;
  flex-wrap: wrap;
  background: #ffffff;
  border-radius: 20rpx;
  padding: 12rpx 0;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.grid-item {
  width: 33.333%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 30rpx 0 26rpx;
  position: relative;

  &.disabled {
    opacity: 0.55;
  }
}

.grid-icon {
  width: 88rpx;
  height: 88rpx;
  border-radius: 20rpx;
  background: #f1f8f2;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 44rpx;
  margin-bottom: 14rpx;
}

.grid-name {
  font-size: 24rpx;
  color: #333;
}

.grid-tag {
  position: absolute;
  top: 22rpx;
  right: calc(33.333% / 2 - 60rpx);
  font-size: 18rpx;
  color: #e64340;
  background: #fff1f0;
  border-radius: 999rpx;
  padding: 2rpx 10rpx;
}
</style>

<template>
  <view class="page">
    <!-- 搜索 + 扫码 -->
    <view class="search-bar">
      <input
        v-model="keyword"
        class="search-input"
        type="text"
        placeholder="输入条码或SKU编码"
        confirm-type="search"
        @confirm="doSearch"
      />
      <view class="scan-btn" @click="doScan">扫码</view>
    </view>
    <view class="query-row">
      <button class="query-btn" @click="doSearch">查 询</button>
    </view>

    <!-- 查询中 -->
    <view v-if="searching" class="block-tip">查询中…</view>

    <!-- 未找到 -->
    <view v-else-if="notFound" class="block-tip">未找到该条码对应的商品</view>

    <!-- 查询失败 -->
    <view v-else-if="queryError" class="err-box">
      <text class="err-text">{{ queryError }}</text>
      <button class="retry-btn" size="mini" @click="doSearch">重试</button>
    </view>

    <!-- 查询结果 -->
    <template v-else-if="sku">
      <!-- SKU 信息卡 -->
      <view class="sku-card">
        <view class="sku-name-row">
          <text class="sku-name">{{ skuGoodsName }}</text>
          <text v-if="sku.status === 0" class="sku-off">已停用</text>
          <text v-else class="sku-on">正常</text>
        </view>
        <view class="sku-row">
          <text class="sku-k">SKU 编码</text>
          <text class="sku-v">{{ sku.skuCode || '-' }}</text>
        </view>
        <view class="sku-row">
          <text class="sku-k">条码</text>
          <text class="sku-v">{{ sku.barcode || '-' }}</text>
        </view>
        <view class="sku-row">
          <text class="sku-k">商品编码</text>
          <text class="sku-v">{{ skuGoodsCode }}</text>
        </view>
        <view class="sku-row">
          <text class="sku-k">颜色 / 尺码</text>
          <text class="sku-v">{{ sku.color || '-' }} / {{ sku.size || '-' }}</text>
        </view>
        <view class="sku-row">
          <text class="sku-k">销售价</text>
          <text class="sku-v">¥{{ fmtMoney(sku.salePrice) }}</text>
        </view>
        <view class="sku-row">
          <text class="sku-k">成本价</text>
          <text class="sku-v">¥{{ fmtMoney(sku.costPrice) }}</text>
        </view>
      </view>

      <!-- 各仓库库存 -->
      <view class="section-title">仓库库存</view>
      <view v-if="stockList.length" class="stock-list">
        <view v-for="(it, idx) in stockList" :key="idx" class="stock-item">
          <view class="stock-main">
            <text class="stock-wh">{{ it.warehouseName || '未命名仓库' }}</text>
            <view class="stock-sub">
              <text>库存 {{ fmtInt(it.quantity) }}</text>
              <text>锁定 {{ fmtInt(it.lockQuantity) }}</text>
            </view>
          </view>
          <view class="stock-avail">
            <text class="avail-label">可售</text>
            <text class="avail-num" :class="{ danger: it.available <= 0 }">{{ fmtInt(it.available) }}</text>
          </view>
        </view>
      </view>
      <view v-else class="empty-tip">暂无库存记录</view>
    </template>
  </view>
</template>

<script setup>
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { get } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

const keyword = ref('')
const searching = ref(false)
const notFound = ref(false)
const queryError = ref('')
const sku = ref(null)

const skuGoodsName = computed(() => {
  const g = sku.value && sku.value.goods
  return (g && g.name) || '未知商品'
})

const skuGoodsCode = computed(() => {
  const g = sku.value && sku.value.goods
  return (g && g.code) || '-'
})

const stockList = computed(() => {
  const s = sku.value
  const list = s && Array.isArray(s.stocks) ? s.stocks : []
  return list.map((it) => {
    const quantity = toInt(it.quantity)
    const lockQuantity = toInt(it.lockQuantity)
    let available = toInt(it.available)
    if (isNaN(Number(it.available)) || Number(it.available) === null || it.available === undefined) {
      available = quantity - lockQuantity
    }
    return {
      warehouseName: it.warehouseName || '',
      quantity,
      lockQuantity,
      available
    }
  })
})

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
  }
})

onShow(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
  }
})

// 调用微信扫码
function doScan() {
  // #ifdef MP-WEIXIN
  uni.scanCode({
    onlyFromCamera: false,
    scanType: ['barCode', 'qrCode'],
    success: (res) => {
      const result = res && (res.result || '')
      if (!result) {
        uni.showToast({ title: '未识别到条码内容', icon: 'none' })
        return
      }
      keyword.value = String(result).trim()
      doSearch()
    },
    fail: (err) => {
      console.warn('[stock] 扫码失败/取消', err)
      uni.showToast({ title: '扫码已取消', icon: 'none' })
    }
  })
  // #endif
  // #ifndef MP-WEIXIN
  uni.showToast({ title: '扫码功能请在微信小程序中使用', icon: 'none' })
  // #endif
}

async function doSearch() {
  const kw = (keyword.value || '').trim()
  if (!kw) {
    uni.showToast({ title: '请输入条码或SKU编码', icon: 'none' })
    return
  }
  searching.value = true
  notFound.value = false
  queryError.value = ''
  try {
    // 契约：GET /jxc/goods/sku/by-barcode?barcode=xxx
    // data: { ID, skuCode, barcode, color, size, costPrice, salePrice, status,
    //         goods: { ID, name, code, unit },
    //         stocks: [{ warehouseId, warehouseName, quantity, lockQuantity, available }] }
    const data = await get('/jxc/goods/sku/by-barcode', { barcode: kw }, { silent: true })
    if (!data || typeof data !== 'object') {
      notFound.value = true
      sku.value = null
      return
    }
    sku.value = data
  } catch (e) {
    const code = e && e.code
    const msg = (e && e.msg) || ''
    // GVA 记录不存在错误码为 7；404 或 "未找到" 提示也视为未找到
    if (code === 7 || /404|not ?found|未找到/i.test(msg)) {
      notFound.value = true
      sku.value = null
    } else {
      queryError.value = (e && e.msg) || '查询失败，请稍后重试'
    }
  } finally {
    searching.value = false
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

function toInt(n) {
  const num = Number(n)
  return isNaN(num) ? 0 : Math.round(num)
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  padding: 24rpx;
  box-sizing: border-box;
}

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 12rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.search-input {
  flex: 1;
  height: 72rpx;
  line-height: 72rpx;
  background: #f5f6f7;
  border-radius: 12rpx;
  padding: 0 24rpx;
  font-size: 26rpx;
}

.scan-btn {
  margin-left: 16rpx;
  height: 72rpx;
  line-height: 72rpx;
  padding: 0 32rpx;
  background: #2b8a3e;
  color: #fff;
  font-size: 28rpx;
  border-radius: 12rpx;
  font-weight: 600;
}

.query-row {
  margin-top: 20rpx;
}

.query-btn {
  height: 80rpx;
  line-height: 80rpx;
  background: #ffffff;
  color: #2b8a3e;
  font-size: 28rpx;
  border-radius: 16rpx;
  border: 1rpx solid #2b8a3e;

  &::after {
    border: none;
  }
}

.section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
  margin: 32rpx 0 20rpx;
}

.block-tip {
  padding: 100rpx 0;
  text-align: center;
  color: #999;
  font-size: 26rpx;
}

.empty-tip {
  padding: 60rpx 0;
  text-align: center;
  color: #999;
  font-size: 26rpx;
  background: #ffffff;
  border-radius: 16rpx;
}

.err-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 80rpx 0;
}

.err-text {
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

/* SKU 信息卡 */
.sku-card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 24rpx 28rpx;
  margin-top: 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.sku-name-row {
  display: flex;
  align-items: center;
  padding: 8rpx 0 20rpx;
  border-bottom: 1rpx solid #f0f0f0;
  margin-bottom: 8rpx;
}

.sku-name {
  flex: 1;
  font-size: 32rpx;
  font-weight: 600;
  color: #222;
}

.sku-on {
  font-size: 20rpx;
  color: #2b8a3e;
  background: #eaf7ee;
  border-radius: 999rpx;
  padding: 4rpx 16rpx;
}

.sku-off {
  font-size: 20rpx;
  color: #999;
  background: #f0f0f0;
  border-radius: 999rpx;
  padding: 4rpx 16rpx;
}

.sku-row {
  display: flex;
  justify-content: space-between;
  padding: 18rpx 0;
  border-bottom: 1rpx solid #f7f7f7;

  &:last-child {
    border-bottom: none;
  }
}

.sku-k {
  font-size: 26rpx;
  color: #999;
}

.sku-v {
  font-size: 26rpx;
  color: #222;
  max-width: 440rpx;
  text-align: right;
  word-break: break-all;
}

/* 仓库库存 */
.stock-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.stock-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24rpx 28rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}

.stock-main {
  display: flex;
  flex-direction: column;
  flex: 1;
  margin-right: 16rpx;
}

.stock-wh {
  font-size: 28rpx;
  color: #222;
  font-weight: 500;
}

.stock-sub {
  margin-top: 8rpx;
  display: flex;
  font-size: 22rpx;
  color: #999;

  text + text {
    margin-left: 24rpx;
  }
}

.stock-avail {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.avail-label {
  font-size: 20rpx;
  color: #999;
}

.avail-num {
  font-size: 34rpx;
  font-weight: 600;
  color: #2b8a3e;

  &.danger {
    color: #e64340;
  }
}
</style>

<template>
  <view class="page">
    <!-- 顶部大按钮：扫码上架（微信扫条码 → 入 PC 待处理队列） -->
    <view class="scan-hero" @click="doScan">
      <text class="hero-glyph">扫</text>
      <text class="hero-text">扫码上架</text>
      <text class="hero-sub">扫描商品条码，自动加入 PC 收银台待确认</text>
    </view>

    <!-- 条码手动输入（无扫码枪 / 真机调试用） -->
    <view class="manual-bar">
      <input
        v-model="barcode"
        class="manual-input"
        type="text"
        placeholder="手动输入条码或 SKU 编码"
        confirm-type="send"
        @confirm="doManualScan"
      />
      <view class="manual-btn" @click="doManualScan">上 架</view>
    </view>

    <!-- 已扫列表：PC 已确认的条目会自动消失（后端 status=0 过滤） -->
    <view class="section-title">
      待收银台确认
      <text class="section-count">{{ pendingList.length }} 条</text>
    </view>

    <view v-if="listLoading" class="block-tip">加载中…</view>
    <view v-else-if="!pendingList.length" class="empty-tip">暂无扫码上架的商品</view>
    <view v-else class="pending-list">
      <view v-for="it in pendingList" :key="it.id" class="pending-item">
        <view class="pi-main">
          <view v-if="it.sku">
            <text class="pi-name">{{ it.sku.goodsName || it.sku.skuCode }}</text>
          </view>
          <view v-else>
            <text class="pi-name deleted">商品已删除</text>
          </view>
          <view class="pi-spec">
            <template v-if="it.sku">
              {{ it.sku.skuCode }}<template v-if="it.sku.color || it.sku.size"> · {{ it.sku.color }} / {{ it.sku.size }}</template>
            </template>
            <template v-else>SKU 已被删除，无法加入购物车</template>
          </view>
        </view>
        <view class="pi-right">
          <text class="pi-price">¥{{ fmtMoney(it.sku ? it.sku.salePrice : 0) }}</text>
          <text class="pi-qty">×{{ it.qty }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, onUnmounted } from 'vue'
import { onLoad, onShow, onHide } from '@dcloudio/uni-app'
import { get, post } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

const barcode = ref('')
const pendingList = ref([])
const listLoading = ref(false)

// 3s 轮询定时器句柄；onShow 启动、onHide/onUnmounted 停止，避免后台空转
let pollTimer = null
const POLL_INTERVAL = 3000

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
})

// 页面显示：立即刷新一次并启动轮询；隐藏：停止轮询
onShow(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  startPolling()
})

onHide(() => {
  stopPolling()
})

onUnmounted(() => {
  stopPolling()
})

function startPolling() {
  if (pollTimer) return
  loadPending()
  pollTimer = setInterval(loadPending, POLL_INTERVAL)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 拉取待处理列表；PC 收银台确认后条目自动消失
async function loadPending() {
  listLoading.value = true
  try {
    const data = await get('/jxc/pos/scan/pending', {}, { silent: true })
    pendingList.value = Array.isArray(data) ? data : []
  } catch (e) {
    // 轮询失败静默，不打断页面
    console.warn('[pos] 拉取扫码队列失败', e)
  } finally {
    listLoading.value = false
  }
}

// 微信扫码：onlyFromCamera:false 允许相册识别；成功后直接上架
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
      submitScan(String(result).trim())
    },
    fail: (err) => {
      console.warn('[pos] 扫码失败/取消', err)
      uni.showToast({ title: '扫码已取消', icon: 'none' })
    }
  })
  // #endif
  // #ifndef MP-WEIXIN
  uni.showToast({ title: '扫码功能请在微信小程序中使用', icon: 'none' })
  // #endif
}

// 手动输入上架（无扫码枪调试）
function doManualScan() {
  const kw = (barcode.value || '').trim()
  if (!kw) {
    uni.showToast({ title: '请输入条码或 SKU 编码', icon: 'none' })
    return
  }
  submitScan(kw)
}

// 提交扫码：小程序扫码固定 qty=1（后端同 SKU 待处理数量自动累加）
async function submitScan(barcodeStr) {
  try {
    await post('/jxc/pos/scan', { barcode: barcodeStr, qty: 1 })
    barcode.value = ''
    uni.showToast({ title: '已加入收银台待确认', icon: 'success' })
    loadPending()
  } catch (e) {
    // 失败提示由 request 封装统一弹 toast（icon: none），此处兜底记录
    console.warn('[pos] 扫码上架失败', e)
  }
}

function fmtMoney(n) {
  const num = Number(n) || 0
  const fixed = num.toFixed(2)
  return fixed.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  padding: 24rpx;
  box-sizing: border-box;
}

/* 顶部大按钮 */
.scan-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  background: linear-gradient(135deg, #2b8a3e 0%, #3fb45c 100%);
  border-radius: 20rpx;
  padding: 44rpx 24rpx 36rpx;
  box-shadow: 0 8rpx 24rpx rgba(43, 138, 62, 0.25);
}

.hero-glyph {
  width: 96rpx;
  height: 96rpx;
  line-height: 96rpx;
  text-align: center;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  font-size: 44rpx;
  font-weight: 600;
  border-radius: 24rpx;
  margin-bottom: 18rpx;
}

.hero-text {
  color: #fff;
  font-size: 38rpx;
  font-weight: 700;
}

.hero-sub {
  margin-top: 10rpx;
  color: rgba(255, 255, 255, 0.85);
  font-size: 24rpx;
}

/* 手动输入栏 */
.manual-bar {
  display: flex;
  align-items: center;
  margin-top: 24rpx;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 12rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.manual-input {
  flex: 1;
  height: 72rpx;
  line-height: 72rpx;
  background: #f5f6f7;
  border-radius: 12rpx;
  padding: 0 24rpx;
  font-size: 26rpx;
}

.manual-btn {
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

/* 列表标题 */
.section-title {
  display: flex;
  align-items: baseline;
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
  margin: 32rpx 0 20rpx;
}

.section-count {
  font-size: 22rpx;
  color: #999;
  font-weight: 400;
  margin-left: 12rpx;
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

/* 已扫列表 */
.pending-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}

.pending-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24rpx 28rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}

.pi-main {
  flex: 1;
  min-width: 0;
  margin-right: 16rpx;
}

.pi-name {
  display: block;
  font-size: 28rpx;
  font-weight: 600;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  &.deleted {
    color: #e64340;
  }
}

.pi-spec {
  margin-top: 6rpx;
  font-size: 22rpx;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pi-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  flex-shrink: 0;
}

.pi-price {
  font-size: 28rpx;
  font-weight: 600;
  color: #ff5a1f;
}

.pi-qty {
  margin-top: 6rpx;
  font-size: 24rpx;
  color: #666;
}
</style>

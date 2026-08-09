<template>
  <view class="page">
    <!-- 收银台绑定状态条：未绑定时不可扫码，点击跳「我的」绑定 -->
    <view class="session-bar" :class="{ 'no-session': !posSession }" @click="goMine">
      <text class="session-label">{{ posSession ? '收银台' : '未绑定收银台' }}</text>
      <text class="session-code">{{ posSession || '请先在「我的」中绑定' }}</text>
      <text class="session-arrow">›</text>
    </view>

    <!-- 顶部大按钮：扫码加购（微信扫条码 → 自动加入 PC 收银台购物车） -->
    <view class="scan-hero" @click="doScan">
      <text class="hero-glyph">扫</text>
      <text class="hero-text">扫码加购</text>
      <text class="hero-sub">扫描商品条码，自动加入 PC 收银台购物车</text>
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
      <view class="manual-btn" @click="doManualScan">加 购</view>
    </view>

    <!-- 轮询降频提示：连续空结果达上限自动切低频保活，拉到条目自动恢复高频 -->
    <view v-if="pollSlow" class="poll-paused-bar">
      <text class="pp-text">自动刷新已降频（长时间无扫码）</text>
      <text class="pp-btn" @click="resumePolling">恢复高频</text>
    </view>

    <!-- 已扫列表：PC 已确认的条目会自动消失（后端 status=0 过滤） -->
    <view class="section-title">
      已加入收银台
      <text class="section-count">{{ pendingList.length }} 条</text>
    </view>

    <view v-if="listLoading" class="block-tip">加载中…</view>
    <view v-else-if="!pendingList.length" class="empty-tip">暂无扫码加购的商品</view>
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

const POS_SESSION_KEY = 'posSession'

const barcode = ref('')
const pendingList = ref([])
const listLoading = ref(false)
// 当前绑定的收银台码（本地 storage）；未绑定则扫码按钮不可用
const posSession = ref(uni.getStorageSync(POS_SESSION_KEY) || '')

// 轮询定时器句柄；onShow 启动、onHide/onUnmounted 停止，避免后台空转
let pollTimer = null
const POS_POLL_FAST_MS = 3000  // 高频轮询间隔：正常收银 3s 一次
const POS_POLL_SLOW_MS = 30000 // 低频保活间隔：长时间无扫码 30s 一次（保活且不刷屏）
const POS_POLL_MAX_EMPTY = 60 // 连续空结果次数上限（60 次 × 3s ≈ 3 分钟），超限自动降频
let emptyCount = 0 // 连续空结果计数：有内容清零，达上限触发降频
const pollSlow = ref(false) // 降频态：连续空结果后切 30s 低频保活，拉到任何条目立即恢复 3s 高频

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
})

// 页面显示：刷新本地绑定码；重置空计数与降频态（高频）；已绑定才启动轮询，未绑定停止（无码时后端返回会话无效）
onShow(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  posSession.value = uni.getStorageSync(POS_SESSION_KEY) || ''
  emptyCount = 0
  pollSlow.value = false
  if (posSession.value) startPolling()
  else stopPolling()
})

onHide(() => {
  stopPolling()
})

onUnmounted(() => {
  stopPolling()
})

// 轮询间隔切换：先 clear 再 set，复用 timer 变量，绝不叠加
function applyPollInterval() {
  const interval = pollSlow.value ? POS_POLL_SLOW_MS : POS_POLL_FAST_MS
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  pollTimer = setInterval(loadPending, interval)
}

function startPolling() {
  if (pollTimer) return
  loadPending()
  applyPollInterval()
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 拉取待处理列表；PC 收银台确认后条目自动消失；请求携带当前收银台码
async function loadPending() {
  if (!posSession.value) return
  listLoading.value = true
  try {
    const data = await get('/jxc/pos/scan/pending', { session: posSession.value }, { silent: true })
    const list = Array.isArray(data) ? data : []
    pendingList.value = list
    // 空结果计数：有内容清零并恢复高频；连续空结果达上限自动降频（不清定时器，30s 保活不刷屏）
    if (list.length) {
      emptyCount = 0
      if (pollSlow.value) {
        pollSlow.value = false
        applyPollInterval()
      }
    } else if (!pollSlow.value) {
      emptyCount += 1
      if (emptyCount >= POS_POLL_MAX_EMPTY) {
        pollSlow.value = true
        emptyCount = 0 // 重新计数：恢复高频后从零开始计时
        applyPollInterval()
      }
    }
  } catch (e) {
    // 轮询失败静默，不打断页面（失败不计入空结果计数）
    console.warn('[pos] 拉取扫码队列失败', e)
  } finally {
    listLoading.value = false
  }
}

// 手动恢复高频：清空计数 + 立即拉一次 + 切回 3s（applyPollInterval 幂等，不叠加）
function resumePolling() {
  emptyCount = 0
  if (pollSlow.value) {
    pollSlow.value = false
    applyPollInterval()
  }
  loadPending()
}

// 微信扫码：onlyFromCamera:false 允许相册识别；成功后直接加购到收银台
function doScan() {
  // 未绑定收银台：提示并引导去「我的」绑定
  if (!requireSession()) return
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

// 手动输入加购（无扫码枪调试）
function doManualScan() {
  if (!requireSession()) return
  const kw = (barcode.value || '').trim()
  if (!kw) {
    uni.showToast({ title: '请输入条码或 SKU 编码', icon: 'none' })
    return
  }
  submitScan(kw)
}

// 绑定检查：已绑定返回 true；未绑定提示并跳转「我的」页绑定
function requireSession() {
  if (posSession.value) return true
  uni.showModal({
    title: '提示',
    content: '请先在「我的」中绑定收银台',
    success: (res) => {
      if (res.confirm) goMine()
    }
  })
  return false
}

// 跳转「我的」tab 页
function goMine() {
  uni.switchTab({ url: '/pages/mine/mine' })
}

// 提交扫码：固定 qty=1（后端同 SKU 待处理数量自动累加）；body 携带当前收银台码
async function submitScan(barcodeStr) {
  try {
    await post('/jxc/pos/scan', { barcode: barcodeStr, qty: 1, session: posSession.value }, { silent: true })
    barcode.value = ''
    uni.showToast({ title: '已加入收银台购物车', icon: 'success' })
    // 扫码成功说明有人在用：若处于降频态则恢复高频，否则按原逻辑刷新
    if (pollSlow.value) resumePolling()
    else loadPending()
  } catch (e) {
    // 收银台码被 PC 端作废/重置：清除本地绑定，引导重新绑定
    if ((e && e.code) === 7 || ((e && e.msg) || '').indexOf('无效') >= 0) {
      uni.removeStorageSync(POS_SESSION_KEY)
      posSession.value = ''
      stopPolling()
      uni.showToast({ title: '收银台码已失效，请重新绑定', icon: 'none' })
      return
    }
    // 其余失败展示后端 msg
    uni.showToast({ title: (e && e.msg) || '扫码加购失败', icon: 'none' })
    console.warn('[pos] 扫码加购失败', e)
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

/* 收银台绑定状态条 */
.session-bar {
  display: flex;
  align-items: center;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 18rpx 28rpx;
  margin-bottom: 20rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.session-label {
  font-size: 24rpx;
  color: #2b8a3e;
  background: #e8f5e9;
  border-radius: 20rpx;
  padding: 4rpx 18rpx;
  flex-shrink: 0;
}
.session-code {
  flex: 1;
  margin-left: 20rpx;
  font-size: 28rpx;
  font-weight: 700;
  letter-spacing: 4rpx;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.session-arrow {
  color: #b2b2b2;
  font-size: 30rpx;
  flex-shrink: 0;
}
.session-bar.no-session .session-label {
  color: #e64340;
  background: #fdf0ef;
}
.session-bar.no-session .session-code {
  color: #e64340;
  font-weight: 400;
  letter-spacing: 0;
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

/* 轮询暂停提示条 */
.poll-paused-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fdf6ec;
  border-radius: 12rpx;
  padding: 16rpx 24rpx;
  margin-top: 24rpx;
}
.pp-text {
  font-size: 24rpx;
  color: #e6a23c;
}
.pp-btn {
  font-size: 24rpx;
  font-weight: 600;
  color: #2b8a3e;
  padding: 4rpx 20rpx;
  background: #ffffff;
  border-radius: 24rpx;
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

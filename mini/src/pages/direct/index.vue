<template>
  <view class="page">
    <!-- 仓库选择 + 方向切换 -->
    <view class="top-card">
      <picker :range="warehouseNames" :value="warehouseIndex" @change="onWarehouseChange">
        <view class="wh-pick">
          <text class="wh-label">仓库</text>
          <text class="wh-value" :class="{ placeholder: !warehouseId }">{{ warehouseId ? warehouseName : '请选择仓库' }}</text>
        </view>
      </picker>
      <view class="dir-tabs">
        <view class="dir-tab" :class="{ active: direction === 'in' }" @click="direction = 'in'">入 库</view>
        <view class="dir-tab" :class="{ active: direction === 'out' }" @click="direction = 'out'">出 库</view>
      </view>
    </view>

    <!-- 添加商品：扫码 / 手动输入 -->
    <view class="add-bar">
      <input
        v-model="keyword"
        class="add-input"
        type="text"
        placeholder="输入条码或 SKU 编码，回车添加"
        confirm-type="search"
        @confirm="addByKeyword"
      />
      <view class="scan-btn" @click="doScan">扫码添加</view>
    </view>

    <!-- 待提交列表 -->
    <view class="section-title">待{{ direction === 'in' ? '入库' : '出库' }}商品<text class="section-count">{{ rows.length }} 项</text></view>
    <view v-if="!rows.length" class="empty-tip">请先扫码或输入商品编码添加</view>
    <view v-else class="rows-list">
      <view v-for="(r, idx) in rows" :key="r.skuId" class="row-item">
        <view class="ri-main">
          <text class="ri-name">{{ r.name }}</text>
          <text class="ri-spec">
            {{ r.skuCode }}<template v-if="r.color || r.size"> · {{ r.color }} / {{ r.size }}</template>
          </text>
        </view>
        <view class="ri-right">
          <view class="ri-stepper">
            <view class="step-btn" @click="stepRow(r, -1)">−</view>
            <view class="step-num">{{ r.qty }}</view>
            <view class="step-btn plus" @click="stepRow(r, 1)">+</view>
          </view>
          <text class="ri-del" @click="rows.splice(idx, 1)">删</text>
        </view>
      </view>
    </view>

    <!-- 提交 -->
    <view class="submit-bar">
      <view class="submit-btn" :class="{ disabled: submitting }" @click="submit">提交{{ direction === 'in' ? '入库' : '出库' }}（{{ rows.length }} 项）</view>
    </view>
  </view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { get, post } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

const warehouses = ref([])
const warehouseIndex = ref(0)
const warehouseId = ref(0)
const warehouseName = ref('')
const direction = ref('in') // 'in'=入库 | 'out'=出库
const keyword = ref('')
const rows = ref([]) // [{skuId, skuCode, name, color, size, qty}]
const submitting = ref(false)

const warehouseNames = computed(() => warehouses.value.map((w) => w.name || '未命名仓库'))

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  loadWarehouses()
})

onShow(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
  }
})

// 仓库列表
async function loadWarehouses() {
  try {
    const data = await get('/jxc/warehouse/list', { page: 1, pageSize: 999 }, { silent: true })
    const list = Array.isArray(data && data.list) ? data.list : []
    warehouses.value = list
  } catch (e) {
    uni.showToast({ title: (e && e.msg) || '加载仓库失败', icon: 'none' })
  }
}

function onWarehouseChange(e) {
  const i = Number(e.detail.value)
  warehouseIndex.value = i
  const w = warehouses.value[i]
  warehouseId.value = w ? w.ID : 0
  warehouseName.value = w ? w.name : ''
}

// 扫码添加
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
      addSku(String(result).trim())
    },
    fail: () => {
      // 取消静默，不打扰
    }
  })
  // #endif
  // #ifndef MP-WEIXIN
  uni.showToast({ title: '扫码功能请在微信小程序中使用', icon: 'none' })
  // #endif
}

// 手动输入添加
async function addByKeyword() {
  const kw = (keyword.value || '').trim()
  if (!kw) {
    uni.showToast({ title: '请输入条码或 SKU 编码', icon: 'none' })
    return
  }
  addSku(kw)
}

// 按条码/编码查 SKU 并加入待提交列表（同 SKU 数量累加）
async function addSku(barcode) {
  uni.showLoading({ title: '查询中…', mask: true })
  try {
    const data = await get('/jxc/goods/sku/by-barcode', { barcode }, { silent: true })
    if (!data || !data.ID) {
      uni.hideLoading()
      uni.showToast({ title: '未找到该条码对应的商品', icon: 'none' })
      return
    }
    const exist = rows.value.find((r) => r.skuId === data.ID)
    if (exist) exist.qty += 1
    else {
      rows.value.push({
        skuId: data.ID,
        skuCode: data.skuCode || '',
        name: (data.goods && data.goods.name) || data.skuCode || '未知商品',
        color: data.color || '',
        size: data.size || '',
        qty: 1,
      })
    }
    keyword.value = ''
    uni.showToast({ title: '已添加', icon: 'success' })
  } catch (e) {
    uni.showToast({ title: (e && e.msg) || '未找到该条码对应的商品', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
}

function stepRow(r, d) {
  const next = r.qty + d
  if (next <= 0) return
  r.qty = next
}

// 出库前可售校验：查 stock/page 该 SKU 在该仓库的可售数
async function checkAvailable(skuId) {
  try {
    const data = await get('/jxc/stock/page', { page: 1, pageSize: 999, skuId, warehouseId: warehouseId.value }, { silent: true })
    const list = Array.isArray(data && data.list) ? data.list : []
    return list.length ? Number(list[0].available) || 0 : 0
  } catch (e) {
    return -1 // 校验失败：不拦截，交由后端判定
  }
}

// 逐条提交：成功行移除，失败行保留（后端也会校验可售，前端提示更友好）
async function submit() {
  if (!warehouseId.value) {
    uni.showToast({ title: '请先选择仓库', icon: 'none' })
    return
  }
  if (!rows.value.length) {
    uni.showToast({ title: '请先添加商品', icon: 'none' })
    return
  }
  if (submitting.value) return
  submitting.value = true
  uni.showLoading({ title: '提交中…', mask: true })
  const path = direction.value === 'in' ? '/jxc/stock/direct-in' : '/jxc/stock/direct-out'
  const dirText = direction.value === 'in' ? '入库' : '出库'
  let ok = 0
  const failMsgs = []
  const snapshot = [...rows.value]
  for (const r of snapshot) {
    // 出库：前端可售预校验（可选，后端仍会兜底）
    if (direction.value === 'out') {
      const avail = await checkAvailable(r.skuId)
      if (avail >= 0 && r.qty > avail) {
        failMsgs.push(`${r.skuCode}：可售库存不足（剩 ${avail} 件）`)
        continue
      }
    }
    try {
      await post(path, { warehouseId: warehouseId.value, skuId: r.skuId, qty: r.qty, remark: '小程序出入库' }, { silent: true })
      ok += 1
      rows.value = rows.value.filter((x) => x.skuId !== r.skuId)
    } catch (e) {
      failMsgs.push(`${r.skuCode}：${(e && e.msg) || '提交失败'}`)
    }
  }
  uni.hideLoading()
  submitting.value = false
  if (ok > 0) uni.showToast({ title: `${dirText}成功 ${ok} 条${failMsgs.length ? `，失败 ${failMsgs.length} 条` : ''}`, icon: 'success' })
  if (failMsgs.length) {
    setTimeout(() => {
      uni.showModal({
        title: `部分失败（成功 ${ok} / 失败 ${failMsgs.length}）`,
        content: failMsgs.slice(0, 5).join('\n'),
        showCancel: false,
        confirmText: '知道了',
      })
    }, ok > 0 ? 1200 : 300)
  }
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  padding: 24rpx;
  padding-bottom: 160rpx;
  box-sizing: border-box;
}

.section-title {
  display: flex;
  align-items: baseline;
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
  margin: 28rpx 0 20rpx;
}
.section-count {
  font-size: 22rpx;
  color: #999;
  font-weight: 400;
  margin-left: 12rpx;
}

/* ===== 顶部卡片 ===== */
.top-card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 20rpx 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.wh-pick {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10rpx 0 18rpx;
}
.wh-label {
  font-size: 26rpx;
  color: #999;
}
.wh-value {
  font-size: 28rpx;
  font-weight: 600;
  color: #222;

  &.placeholder {
    color: #b2b2b2;
    font-weight: 400;
  }
}
.dir-tabs {
  display: flex;
  gap: 12rpx;
}
.dir-tab {
  flex: 1;
  height: 76rpx;
  line-height: 76rpx;
  text-align: center;
  background: #f5f6f7;
  color: #666;
  font-size: 28rpx;
  font-weight: 600;
  border-radius: 12rpx;

  &.active {
    background: linear-gradient(135deg, #2b8a3e 0%, #3fb45c 100%);
    color: #fff;
    box-shadow: 0 6rpx 16rpx rgba(43, 138, 62, 0.25);
  }
}

/* ===== 添加栏 ===== */
.add-bar {
  display: flex;
  align-items: center;
  margin-top: 20rpx;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 12rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.add-input {
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

/* ===== 待提交列表 ===== */
.empty-tip {
  padding: 60rpx 0;
  text-align: center;
  color: #999;
  font-size: 26rpx;
  background: #ffffff;
  border-radius: 16rpx;
}
.rows-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.row-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20rpx 24rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.ri-main {
  flex: 1;
  min-width: 0;
  margin-right: 16rpx;
}
.ri-name {
  display: block;
  font-size: 28rpx;
  font-weight: 600;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ri-spec {
  display: block;
  margin-top: 4rpx;
  font-size: 22rpx;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ri-right {
  display: flex;
  align-items: center;
  gap: 16rpx;
  flex-shrink: 0;
}
.ri-stepper {
  display: flex;
  align-items: center;
  gap: 8rpx;
  background: #f5f6f7;
  border-radius: 12rpx;
  padding: 6rpx;
}
.step-btn {
  width: 52rpx;
  height: 52rpx;
  line-height: 52rpx;
  text-align: center;
  background: #ffffff;
  border-radius: 10rpx;
  font-size: 30rpx;
  font-weight: 700;
  color: #303133;
  box-shadow: 0 2rpx 6rpx rgba(0, 0, 0, 0.06);

  &.plus {
    color: #2b8a3e;
  }
}
.step-num {
  min-width: 52rpx;
  text-align: center;
  font-size: 28rpx;
  font-weight: 700;
  color: #222;
}
.ri-del {
  font-size: 24rpx;
  color: #e64340;
  padding: 8rpx 12rpx;
}

/* ===== 提交栏 ===== */
.submit-bar {
  position: fixed;
  left: 24rpx;
  right: 24rpx;
  bottom: 20rpx;
}
.submit-btn {
  height: 96rpx;
  line-height: 96rpx;
  text-align: center;
  background: linear-gradient(135deg, #2b8a3e 0%, #3fb45c 100%);
  color: #fff;
  font-size: 30rpx;
  font-weight: 700;
  border-radius: 16rpx;
  box-shadow: 0 8rpx 24rpx rgba(43, 138, 62, 0.25);

  &.disabled {
    opacity: 0.5;
  }
}
</style>

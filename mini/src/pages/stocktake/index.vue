<template>
  <view class="page">
    <!-- ========== 盘点单选择列表 ========== -->
    <template v-if="mode === 'list'">
      <view class="section-title">选择盘点单</view>
      <view v-if="listLoading" class="block-tip">加载中…</view>
      <view v-else-if="!checkList.length" class="empty-tip">暂无可录入的盘点单，请先在 PC 创建</view>
      <view v-else class="check-list">
        <view
          v-for="c in checkList"
          :key="c.ID"
          class="check-item"
          @click="openCheck(c)"
        >
          <view class="ci-main">
            <text class="ci-no">{{ c.checkNo }}</text>
            <text class="ci-wh">{{ c.warehouseName || '未命名仓库' }}</text>
          </view>
          <view class="ci-right">
            <text class="ci-remark">{{ c.remark || '' }}</text>
            <text class="ci-time">{{ fmtTime(c.createdAt) }}</text>
          </view>
        </view>
      </view>
    </template>

    <!-- ========== 盘点录入 ========== -->
    <template v-else>
      <!-- 顶部：单号 + 仓库 + 统计 -->
      <view class="check-head">
        <view class="ch-top">
          <text class="ch-no">{{ current.checkNo }}</text>
          <view class="ch-actions">
            <view class="ch-switch" @click="backToList">切换盘点单</view>
          </view>
        </view>
        <view class="ch-sub">
          <text class="ch-wh">{{ current.warehouseName || '未命名仓库' }}</text>
          <text class="ch-stat">已录 {{ touchedCount }}/{{ items.length }} 项</text>
        </view>
      </view>

      <!-- 明细列表 -->
      <view class="section-title">盘点明细</view>
      <view v-if="!items.length" class="empty-tip">该盘点单暂无明细</view>
      <view v-else class="item-list">
        <view v-for="row in items" :key="row.ID" class="item-row">
          <view class="ir-main">
            <text class="ir-name">{{ row.skuName }}</text>
            <text class="ir-spec">
              {{ row.skuCode }}<template v-if="row.color || row.size"> · {{ row.color }} / {{ row.size }}</template>
            </text>
            <view class="ir-qty">
              <text class="ir-k">账存</text>
              <text class="ir-v">{{ row.systemQty }}</text>
            </view>
          </view>
          <view class="ir-right">
            <view class="ir-actual" :class="{ touched: row._touched }">
              <text class="ir-k">实盘</text>
              <text class="ir-v">{{ row._touched ? row._actual : '-' }}</text>
            </view>
            <view class="ir-stepper">
              <view class="step-btn" @click="step(row, -1)">−</view>
              <view class="step-num">{{ row._touched ? row._actual : 0 }}</view>
              <view class="step-btn plus" @click="step(row, 1)">+</view>
            </view>
          </view>
        </view>
      </view>

      <!-- 底部：扫码录入 + 提交 -->
      <view class="foot-actions">
        <view class="scan-big" @click="doScan">扫码录入</view>
        <view
          class="submit-btn"
          :class="{ disabled: !touchedCount }"
          @click="submit"
        >提交盘点录入</view>
      </view>
    </template>
  </view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { get, request } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

// request.js 未导出 put 便捷封装（且不在本次可写范围）：页内用底层 request 直调 PUT
const put = (url, data, options = {}) => request({ url, method: 'PUT', data, ...options })

const mode = ref('list') // list=选单 | edit=录入
const listLoading = ref(false)
const checkList = ref([])
const current = ref({})
const items = ref([]) // 明细 [{ID, skuId, systemQty, _actual, _touched, skuCode, skuName, color, size}]
const detailLoading = ref(false)
const submitting = ref(false)

const touchedCount = computed(() => items.value.filter((r) => r._touched).length)

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  loadChecks()
})

onShow(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
  }
})

// 拉取盘点中（status=1）的盘点单
async function loadChecks() {
  listLoading.value = true
  try {
    const data = await get('/jxc/stockcheck/page', { page: 1, pageSize: 999 }, { silent: true })
    const list = Array.isArray(data && data.list) ? data.list : []
    checkList.value = list.filter((it) => Number(it.status) === 1)
  } catch (e) {
    uni.showToast({ title: (e && e.msg) || '加载盘点单失败', icon: 'none' })
  } finally {
    listLoading.value = false
  }
}

// 打开盘点单：拉取明细并进入录入模式
async function openCheck(c) {
  detailLoading.value = true
  uni.showLoading({ title: '加载明细…', mask: true })
  try {
    const data = await get('/jxc/stockcheck/detail', { id: c.ID }, { silent: true })
    const list = Array.isArray(data && data.items) ? data.items : []
    current.value = {
      ID: data.ID || c.ID,
      checkNo: data.checkNo || c.checkNo,
      warehouseName: (data.warehouse && data.warehouse.name) || c.warehouseName || '',
    }
    items.value = list.map((it) => {
      const sku = it.sku || {}
      const goods = sku.goods || {}
      const touched = Number(it.actualQty) > 0 || it.actualQty !== undefined && it.actualQty !== null && it.actualQty !== 0
      return {
        ID: it.ID,
        skuId: it.skuId,
        systemQty: Number(it.systemQty) || 0,
        skuCode: sku.skuCode || '',
        skuName: goods.name || sku.skuCode || '未知商品',
        color: sku.color || '',
        size: sku.size || '',
        // 本地实盘编辑值：后端已录入值回填；未录视为未触及
        _actual: Number(it.actualQty) || 0,
        _touched: touched,
      }
    })
    mode.value = 'edit'
  } catch (e) {
    uni.showToast({ title: (e && e.msg) || '加载明细失败', icon: 'none' })
  } finally {
    uni.hideLoading()
    detailLoading.value = false
  }
}

function backToList() {
  mode.value = 'list'
  current.value = {}
  items.value = []
  loadChecks()
}

// 手动调整实盘数（最小 0），触发即视为已录入
function step(row, d) {
  const next = (row._touched ? row._actual : 0) + d
  if (next < 0) return
  row._touched = true
  row._actual = next
}

// 扫码录入：条码 → SKU → 匹配本盘点单明细行 → 实盘数 +1（连扫累加）
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
      recordScan(String(result).trim())
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

async function recordScan(barcode) {
  let skuId = 0
  uni.showLoading({ title: '匹配中…', mask: true })
  try {
    const data = await get('/jxc/goods/sku/by-barcode', { barcode }, { silent: true })
    skuId = data && data.ID ? data.ID : 0
  } catch (e) {
    // 未找到条码：code=7 或 msg 含未找到
    uni.hideLoading()
    uni.showToast({ title: '条码不存在或未找到商品', icon: 'none' })
    return
  } finally {
    uni.hideLoading()
  }
  if (!skuId) {
    uni.showToast({ title: '条码不存在或未找到商品', icon: 'none' })
    return
  }
  const row = items.value.find((r) => r.skuId === skuId)
  if (!row) {
    uni.showToast({ title: '该商品不在本盘点单中', icon: 'none' })
    return
  }
  row._touched = true
  row._actual += 1
  uni.showToast({ title: `已录入 ${row.skuName} 实盘 ${row._actual}`, icon: 'none' })
}

// 提交实盘录入：只传已录入的行（ID 为盘点明细 ID）
async function submit() {
  if (!touchedCount.value) {
    uni.showToast({ title: '请先录入实盘数量', icon: 'none' })
    return
  }
  if (submitting.value) return
  submitting.value = true
  uni.showLoading({ title: '提交中…', mask: true })
  try {
    const payload = {
      checkId: current.value.ID,
      items: items.value.filter((r) => r._touched).map((r) => ({ ID: r.ID, actualQty: r._actual })),
    }
    await put('/jxc/stockcheck/items', payload, { silent: true })
    uni.hideLoading()
    uni.showToast({ title: '提交成功，请到 PC 端完成盘点', icon: 'success' })
    setTimeout(() => backToList(), 1200)
  } catch (e) {
    uni.hideLoading()
    uni.showToast({ title: (e && e.msg) || '提交失败', icon: 'none' })
  } finally {
    submitting.value = false
  }
}

function fmtTime(t) {
  if (!t) return ''
  const s = String(t).replace('T', ' ').slice(0, 16)
  return s
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  padding: 24rpx;
  padding-bottom: 180rpx;
  box-sizing: border-box;
}

.section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
  margin: 8rpx 0 20rpx;
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

/* ===== 盘点单列表 ===== */
.check-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.check-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24rpx 28rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.ci-main {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
.ci-no {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
}
.ci-wh {
  margin-top: 6rpx;
  font-size: 24rpx;
  color: #2b8a3e;
}
.ci-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  flex-shrink: 0;
}
.ci-remark {
  font-size: 22rpx;
  color: #999;
}
.ci-time {
  margin-top: 6rpx;
  font-size: 20rpx;
  color: #b2b2b2;
}

/* ===== 盘点头 ===== */
.check-head {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 24rpx 28rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.ch-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.ch-no {
  font-size: 32rpx;
  font-weight: 700;
  color: #222;
}
.ch-switch {
  font-size: 24rpx;
  color: #2b8a3e;
  padding: 8rpx 20rpx;
  border: 1rpx solid #2b8a3e;
  border-radius: 999rpx;
}
.ch-sub {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 14rpx;
}
.ch-wh {
  font-size: 24rpx;
  color: #2b8a3e;
}
.ch-stat {
  font-size: 24rpx;
  color: #999;
}

/* ===== 明细列表 ===== */
.item-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.item-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20rpx 24rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.ir-main {
  flex: 1;
  min-width: 0;
  margin-right: 16rpx;
}
.ir-name {
  display: block;
  font-size: 28rpx;
  font-weight: 600;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ir-spec {
  display: block;
  margin-top: 4rpx;
  font-size: 22rpx;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ir-qty {
  display: flex;
  align-items: baseline;
  margin-top: 8rpx;
}
.ir-k {
  font-size: 22rpx;
  color: #999;
  margin-right: 12rpx;
}
.ir-v {
  font-size: 26rpx;
  font-weight: 600;
  color: #303133;
}
.ir-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10rpx;
  flex-shrink: 0;
}
.ir-actual {
  display: flex;
  align-items: baseline;

  .ir-v {
    color: #b2b2b2;
  }

  &.touched .ir-v {
    color: #2b8a3e;
  }
}
.ir-stepper {
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

/* ===== 底部操作 ===== */
.foot-actions {
  position: fixed;
  left: 24rpx;
  right: 24rpx;
  bottom: 20rpx;
  display: flex;
  gap: 16rpx;
}
.scan-big {
  flex: 1;
  height: 92rpx;
  line-height: 92rpx;
  text-align: center;
  background: linear-gradient(135deg, #2b8a3e 0%, #3fb45c 100%);
  color: #fff;
  font-size: 30rpx;
  font-weight: 700;
  border-radius: 16rpx;
  box-shadow: 0 8rpx 24rpx rgba(43, 138, 62, 0.25);
}
.submit-btn {
  flex: 1;
  height: 92rpx;
  line-height: 92rpx;
  text-align: center;
  background: #ffffff;
  color: #2b8a3e;
  font-size: 30rpx;
  font-weight: 700;
  border-radius: 16rpx;
  border: 2rpx solid #2b8a3e;

  &.disabled {
    opacity: 0.4;
  }
}
</style>

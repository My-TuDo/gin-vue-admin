<template>
  <view class="page">
    <!-- 表头区：仓库选择 + 方向切换 + 扫码添加 -->
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

    <!-- 搜索 + 扫码 -->
    <view class="add-bar">
      <input
        v-model="listKeyword"
        class="add-input"
        type="text"
        placeholder="搜索名称 / SKU编码 / 条码"
        confirm-type="search"
      />
      <view class="scan-btn" @click="doScan">扫码添加</view>
    </view>

    <!-- SKU 列表区（可滚动） -->
    <view class="list-head">
      <text class="section-title">商品列表</text>
      <text class="list-count">{{ skuList.length }} 条</text>
    </view>
    <scroll-view class="sku-scroll" scroll-y>
      <view v-if="listLoading" class="block-tip">加载中…</view>
      <view v-else-if="listError" class="err-box">
        <text class="err-text">{{ listError }}</text>
        <view class="retry-btn" @click="loadSkus">重试</view>
      </view>
      <view v-else-if="!skuList.length" class="block-tip">{{ skus.length ? '无匹配商品' : '暂无商品数据' }}</view>
      <view v-else class="sku-list">
        <view v-for="s in skuList" :key="s.ID" class="sku-item">
          <view class="si-main">
            <text class="si-code">{{ s.skuCode }}</text>
            <text class="si-name">{{ s.name }}</text>
            <text v-if="s.color || s.size" class="si-spec">{{ s.color }} / {{ s.size }}</text>
          </view>
          <view class="si-right">
            <text class="si-stock" :class="{ none: !s.available }">{{ s.available > 0 ? `可售 ${s.available}` : '无库存' }}</text>
            <view class="si-add" @click="addSkuById(s)">+</view>
          </view>
        </view>
      </view>
    </scroll-view>

    <!-- 明细区（底部固定）+ 提交 -->
    <view class="detail-panel">
      <view class="detail-head">
        <text class="detail-title">待{{ direction === 'in' ? '入库' : '出库' }}（{{ rows.length }} 项）</text>
        <text class="detail-total">共 {{ rowsTotal }} 件</text>
      </view>
      <scroll-view v-if="rows.length" class="detail-scroll" scroll-y>
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
      </scroll-view>
      <view v-else class="detail-empty">请扫码或点选商品添加</view>
      <view
        class="submit-btn"
        :class="{ disabled: submitting || !rows.length }"
        @click="submit"
      >提交{{ direction === 'in' ? '入库' : '出库' }}（{{ rows.length }} 项）</view>
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
const rows = ref([]) // [{skuId, skuCode, name, color, size, qty}]
const submitting = ref(false)

// ===== SKU 列表区 =====
const skus = ref([]) // 原始 SKU 列表 [{ID, skuCode, color, size, barcode, goods:{name}}]
const stockMap = ref({}) // skuId -> available（当前仓库可售）
const listKeyword = ref('') // 本地过滤词
const listLoading = ref(false)
const listError = ref('')

const warehouseNames = computed(() => warehouses.value.map((w) => w.name || '未命名仓库'))

// 合并 SKU 数据 + 库存并本地过滤：匹配 名称/SKU编码/条码
const skuList = computed(() => {
  const kw = listKeyword.value.trim().toLowerCase()
  return skus.value
    .filter((s) => {
      if (!kw) return true
      const name = ((s.goods && s.goods.name) || '').toLowerCase()
      const code = (s.skuCode || '').toLowerCase()
      const barcode = (s.barcode || '').toLowerCase()
      return name.includes(kw) || code.includes(kw) || barcode.includes(kw)
    })
    .map((s) => ({
      ID: s.ID,
      skuCode: s.skuCode || '',
      name: (s.goods && s.goods.name) || s.skuCode || '未知商品',
      color: s.color || '',
      size: s.size || '',
      barcode: s.barcode || '',
      available: stockMap.value[s.ID] ?? 0,
    }))
})

const rowsTotal = computed(() => rows.value.reduce((s, r) => s + r.qty, 0))

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
  // 仓库切换：重置过滤词并重新拉取该仓库可售库存
  listKeyword.value = ''
  loadStock()
}

// SKU 列表（不依赖仓库，一次拉取）
async function loadSkus() {
  listLoading.value = true
  listError.value = ''
  try {
    const data = await get('/jxc/goods/sku/list', { page: 1, pageSize: 999 }, { silent: true })
    // normalize：兼容数组与 { list: [...] }
    const list = Array.isArray(data) ? data : data && Array.isArray(data.list) ? data.list : []
    skus.value = list
  } catch (e) {
    listError.value = (e && e.msg) || '商品列表加载失败'
  } finally {
    listLoading.value = false
  }
}

// 当前仓库可售库存 → skuId -> available
async function loadStock() {
  if (!warehouseId.value) {
    stockMap.value = {}
    return
  }
  try {
    const data = await get('/jxc/stock/page', { page: 1, pageSize: 999, warehouseId: warehouseId.value }, { silent: true })
    const list = Array.isArray(data && data.list) ? data.list : []
    const m = {}
    list.forEach((it) => {
      let avail = Number(it.available)
      if (isNaN(avail)) avail = Number(it.quantity || 0) - Number(it.lockQuantity || 0)
      m[it.skuId] = avail
    })
    stockMap.value = m
  } catch (e) {
    // 库存加载失败静默：可售列显示 0（提交时后端会兜底校验）
    console.warn('[direct] 库存加载失败', e)
    stockMap.value = {}
  }
}

// 点选 SKU 加入明细（同 SKU 数量累加）
function addSkuById(s) {
  addRow({ skuId: s.ID, skuCode: s.skuCode, name: s.name, color: s.color, size: s.size })
  uni.showToast({ title: `已添加 ${s.name}`, icon: 'none' })
}

// 公共加入明细逻辑
function addRow({ skuId, skuCode, name, color, size }) {
  const exist = rows.value.find((r) => r.skuId === skuId)
  if (exist) exist.qty += 1
  else rows.value.push({ skuId, skuCode, name, color, size, qty: 1 })
}

// 扫码添加（保留）：条码 → 查 SKU → 加入明细
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

async function addSku(barcode) {
  uni.showLoading({ title: '查询中…', mask: true })
  try {
    const data = await get('/jxc/goods/sku/by-barcode', { barcode }, { silent: true })
    if (!data || !data.ID) {
      uni.hideLoading()
      uni.showToast({ title: '未找到该条码对应的商品', icon: 'none' })
      return
    }
    addRow({
      skuId: data.ID,
      skuCode: data.skuCode || '',
      name: (data.goods && data.goods.name) || data.skuCode || '未知商品',
      color: data.color || '',
      size: data.size || '',
    })
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
  // 提交后刷新可售库存（列表可售列即时更新）
  if (ok > 0) loadStock()
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
  height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 24rpx;
  padding-bottom: 12rpx;
  box-sizing: border-box;
}

.section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
}

/* ===== 顶部卡片 ===== */
.top-card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 20rpx 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  flex-shrink: 0;
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
  height: 72rpx;
  line-height: 72rpx;
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

/* ===== 搜索栏 ===== */
.add-bar {
  display: flex;
  align-items: center;
  margin-top: 16rpx;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 12rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  flex-shrink: 0;
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
  flex-shrink: 0;
}

/* ===== SKU 列表区（可滚动） ===== */
.list-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin: 20rpx 4rpx 12rpx;
  flex-shrink: 0;
}
.list-count {
  font-size: 22rpx;
  color: #999;
}
.sku-scroll {
  flex: 1;
  min-height: 0;
}
.sku-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  padding-bottom: 2rpx;
}
.sku-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20rpx 24rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.si-main {
  flex: 1;
  min-width: 0;
  margin-right: 16rpx;
}
.si-code {
  display: block;
  font-size: 28rpx;
  font-weight: 700;
  color: #2b8a3e;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.si-name {
  display: block;
  margin-top: 4rpx;
  font-size: 26rpx;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.si-spec {
  display: block;
  margin-top: 2rpx;
  font-size: 22rpx;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.si-right {
  display: flex;
  align-items: center;
  gap: 16rpx;
  flex-shrink: 0;
}
.si-stock {
  font-size: 22rpx;
  color: #2b8a3e;

  &.none {
    color: #b2b2b2;
  }
}
.si-add {
  width: 56rpx;
  height: 56rpx;
  line-height: 52rpx;
  text-align: center;
  background: linear-gradient(135deg, #2b8a3e 0%, #3fb45c 100%);
  color: #fff;
  font-size: 36rpx;
  font-weight: 700;
  border-radius: 14rpx;
  box-shadow: 0 4rpx 12rpx rgba(43, 138, 62, 0.25);
}

/* ===== 明细区（底部固定） ===== */
.detail-panel {
  flex-shrink: 0;
  margin-top: 16rpx;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 16rpx 24rpx 20rpx;
  box-shadow: 0 -2rpx 16rpx rgba(0, 0, 0, 0.05);
}
.detail-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 8rpx;
}
.detail-title {
  font-size: 28rpx;
  font-weight: 700;
  color: #222;
}
.detail-total {
  font-size: 22rpx;
  color: #999;
}
.detail-scroll {
  max-height: 260rpx;
}
.detail-empty {
  padding: 24rpx 0;
  text-align: center;
  color: #b2b2b2;
  font-size: 24rpx;
}
.row-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14rpx 0;
  border-bottom: 1rpx dashed #f0f0f0;

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
  font-size: 26rpx;
  font-weight: 600;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ri-spec {
  display: block;
  margin-top: 2rpx;
  font-size: 20rpx;
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
  padding: 4rpx;
}
.step-btn {
  width: 48rpx;
  height: 48rpx;
  line-height: 48rpx;
  text-align: center;
  background: #ffffff;
  border-radius: 10rpx;
  font-size: 28rpx;
  font-weight: 700;
  color: #303133;
  box-shadow: 0 2rpx 6rpx rgba(0, 0, 0, 0.06);

  &.plus {
    color: #2b8a3e;
  }
}
.step-num {
  min-width: 48rpx;
  text-align: center;
  font-size: 26rpx;
  font-weight: 700;
  color: #222;
}
.ri-del {
  font-size: 22rpx;
  color: #e64340;
  padding: 8rpx 8rpx;
}

/* ===== 提交 ===== */
.submit-btn {
  margin-top: 14rpx;
  height: 88rpx;
  line-height: 88rpx;
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

/* ===== 状态提示 ===== */
.block-tip {
  padding: 80rpx 0;
  text-align: center;
  color: #999;
  font-size: 26rpx;
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
  padding: 10rpx 40rpx;
  background: #2b8a3e;
  color: #fff;
  font-size: 24rpx;
  border-radius: 8rpx;
}
</style>

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
        placeholder="搜索商品名称 / 款号"
        confirm-type="search"
      />
      <view class="scan-btn" @click="doScan">扫码添加</view>
    </view>

    <!-- 商品列表区（可滚动）：浏览商品 → 点选进入 SKU 选择 -->
    <view class="list-head">
      <text class="section-title">商品列表</text>
      <text class="list-count">{{ goodsListFiltered.length }} 条</text>
    </view>
    <scroll-view class="sku-scroll" scroll-y>
      <view v-if="listLoading" class="block-tip">加载中…</view>
      <view v-else-if="listError" class="err-box">
        <text class="err-text">{{ listError }}</text>
        <view class="retry-btn" @click="retryLoad">重试</view>
      </view>
      <view v-else-if="!goodsListFiltered.length" class="block-tip">{{ goodsList.length ? '无匹配商品' : '暂无商品数据' }}</view>
      <view v-else class="goods-list">
        <view v-for="g in goodsListFiltered" :key="g.ID" class="goods-item" @click="openSkuPop(g)">
          <view class="gi-main">
            <text class="gi-name">{{ g.name }}</text>
            <text v-if="g.code" class="gi-code">{{ g.code }}</text>
          </view>
          <view class="gi-right">
            <text class="gi-count">{{ g.skuCount }} 个规格</text>
            <text class="gi-arrow">›</text>
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

    <!-- SKU 选择弹层（自绘半屏，无 uni-popup 依赖） -->
    <view v-if="skuPopVisible" class="pop-mask" @click="skuPopVisible = false">
      <view class="pop-panel" @click.stop>
        <view class="pop-head">
          <text class="pop-title">{{ currentGoods.name }}</text>
          <text v-if="currentGoods.code" class="pop-code">{{ currentGoods.code }}</text>
          <view class="pop-close" @click="skuPopVisible = false">×</view>
        </view>
        <view v-if="!popSkus.length" class="pop-empty">该商品暂无 SKU，请先在电脑端商品中心添加</view>
        <scroll-view v-else class="pop-scroll" scroll-y>
          <view v-for="(x, idx) in popSkus" :key="x.skuId" class="pop-sku-row">
            <view class="ps-main">
              <text class="ps-spec">{{ specText(x) }}</text>
              <text class="ps-stock" :class="{ none: !x.stock }">{{ x.stock > 0 ? `库存 ${x.stock} · 可售 ${x.avail}` : '无库存' }}</text>
            </view>
            <view class="ps-stepper">
              <view class="step-btn" @click="popStep(x, -1)">−</view>
              <view class="step-num">{{ x.qty }}</view>
              <view class="step-btn plus" @click="popStep(x, 1)">+</view>
            </view>
          </view>
        </scroll-view>
        <view class="pop-foot">
          <view class="pop-add-btn" :class="{ disabled: !popTotal }" @click="addFromPop">加入明细（合计 {{ popTotal }} 件）</view>
        </view>
      </view>
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

// ===== 商品列表区 =====
const goodsList = ref([]) // 商品列表 [{ID, name, code?}]（/jxc/goods/page）
const skus = ref([]) // SKU 列表 [{ID, skuCode, color, size, barcode, goodsId?, goods:{ID,name}}]（/jxc/goods/sku/list）
const stockMap = ref({}) // skuId -> { qty: 总库存, avail: 可售 }（当前仓库）
const listKeyword = ref('') // 本地过滤词（匹配商品名称/款号）
const listLoading = ref(false)
const listError = ref('')

// ===== SKU 选择弹层状态 =====
const skuPopVisible = ref(false)
const currentGoods = ref({})
const popSkus = ref([]) // [{skuId, skuCode, color, size, qty, stock, avail}]

const warehouseNames = computed(() => warehouses.value.map((w) => w.name || '未命名仓库'))

// SKU 所属商品 id 兼容：goodsId 直取 或 关联 goods 对象
const skuGoodsId = (s) => Number(s.goodsId || (s.goods && (s.goods.ID || s.goods.id)))

// 商品列表 + 规格数统计 + 本地过滤（名称/款号）
const goodsListFiltered = computed(() => {
  const kw = listKeyword.value.trim().toLowerCase()
  return goodsList.value
    .filter((g) => {
      if (!kw) return true
      const name = (g.name || '').toLowerCase()
      const code = (g.code || g.skuCode || g.goodsCode || '').toLowerCase()
      return name.includes(kw) || code.includes(kw)
    })
    .map((g) => ({
      ID: g.ID,
      name: g.name || '未命名商品',
      code: g.code || g.skuCode || g.goodsCode || '',
      skuCount: skus.value.filter((s) => skuGoodsId(s) === Number(g.ID)).length,
    }))
})

const rowsTotal = computed(() => rows.value.reduce((s, r) => s + r.qty, 0))
const popTotal = computed(() => popSkus.value.reduce((s, x) => s + x.qty, 0))

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  loadWarehouses()
  loadGoods()
  loadSkus() // SKU 用于规格数统计与弹层选择，与商品/仓库无依赖，并行拉取
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

// 商品列表（不依赖仓库，一次拉取）
async function loadGoods() {
  listLoading.value = true
  listError.value = ''
  try {
    const data = await get('/jxc/goods/page', { page: 1, pageSize: 999 }, { silent: true })
    const list = Array.isArray(data && data.list) ? data.list : []
    goodsList.value = list
  } catch (e) {
    listError.value = (e && e.msg) || '商品列表加载失败'
  } finally {
    listLoading.value = false
  }
}

// SKU 列表（不依赖仓库，一次拉取；用于规格数统计与弹层选择）
async function loadSkus() {
  try {
    const data = await get('/jxc/goods/sku/list', { page: 1, pageSize: 999 }, { silent: true })
    // normalize：兼容数组与 { list: [...] }
    const list = Array.isArray(data) ? data : data && Array.isArray(data.list) ? data.list : []
    skus.value = list
  } catch (e) {
    // SKU 加载失败静默：商品行规格数显示 0，弹层无 SKU（可重试进入弹层）
    console.warn('[direct] SKU 列表加载失败', e)
  }
}

// 重试：商品与 SKU 一并重拉
function retryLoad() {
  loadGoods()
  loadSkus()
}

// 当前仓库库存 → skuId -> { qty, avail }（qty=总库存, avail=可售）
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
      const qty = Number(it.quantity) || 0 // 总库存（NaN/空兜底 0）
      let avail = Number(it.available)
      if (isNaN(avail)) avail = qty - Number(it.lockQuantity || 0) // 可售缺省时按 总库存-锁定 退化计算
      m[it.skuId] = { qty, avail }
    })
    stockMap.value = m
  } catch (e) {
    // 库存加载失败静默：库存列显示无库存（提交时后端会兜底校验）
    console.warn('[direct] 库存加载失败', e)
    stockMap.value = {}
  }
}

// 打开商品 SKU 选择弹层：按当前商品过滤 SKU，初始化数量为 1
function openSkuPop(g) {
  currentGoods.value = g
  popSkus.value = skus.value
    .filter((s) => skuGoodsId(s) === Number(g.ID))
    .map((s) => {
      const st = stockMap.value[s.ID]
      return {
        skuId: s.ID,
        skuCode: s.skuCode || '',
        color: s.color || '',
        size: s.size || '',
        qty: 1,
        stock: st ? st.qty : 0,
        avail: st ? st.avail : 0,
      }
    })
  skuPopVisible.value = true
}

function specText(x) {
  const parts = [x.color, x.size].filter(Boolean)
  return parts.length ? parts.join(' / ') : '默认规格'
}

// 弹层内 stepper：下限 1
function popStep(x, d) {
  const next = x.qty + d
  if (next < 1) return
  x.qty = next
}

// 加入明细：该商品每个 SKU 按所选数量累加（addRow 兼容 qty 参数），关闭弹层
function addFromPop() {
  const g = currentGoods.value
  popSkus.value.forEach((x) => {
    if (!x.qty) return
    addRow({ skuId: x.skuId, skuCode: x.skuCode, name: g.name || x.skuCode, color: x.color, size: x.size }, x.qty)
  })
  skuPopVisible.value = false
  uni.showToast({ title: '已加入明细', icon: 'success' })
}

// 公共加入明细逻辑：同 SKU 数量累加；qty 参数缺省为 1（扫码/点选兼容）
function addRow({ skuId, skuCode, name, color, size }, qty = 1) {
  const exist = rows.value.find((r) => r.skuId === skuId)
  if (exist) exist.qty += qty
  else rows.value.push({ skuId, skuCode, name, color, size, qty })
}

// 扫码添加（保留）：条码 → 查 SKU → 加入明细（与商品点选二选一）
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
  // 提交后刷新可售库存（弹层/明细可售数据即时更新）
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

/* ===== 商品列表区（可滚动） ===== */
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
.goods-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  padding-bottom: 2rpx;
}
.goods-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 22rpx 24rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.gi-main {
  flex: 1;
  min-width: 0;
  margin-right: 16rpx;
}
.gi-name {
  display: block;
  font-size: 29rpx;
  font-weight: 600;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gi-code {
  display: block;
  margin-top: 4rpx;
  font-size: 22rpx;
  color: #2b8a3e;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gi-right {
  display: flex;
  align-items: center;
  gap: 12rpx;
  flex-shrink: 0;
}
.gi-count {
  font-size: 22rpx;
  color: #999;
}
.gi-arrow {
  font-size: 32rpx;
  color: #c0c4cc;
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

/* ===== SKU 选择弹层（自绘半屏） ===== */
.pop-mask {
  position: fixed;
  inset: 0;
  z-index: 999;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
}
.pop-panel {
  background: #ffffff;
  border-radius: 24rpx 24rpx 0 0;
  height: 70vh;
  display: flex;
  flex-direction: column;
  padding-bottom: env(safe-area-inset-bottom);
  animation: popUp 0.25s ease;
}
@keyframes popUp {
  from { transform: translateY(100%); }
  to { transform: translateY(0); }
}
.pop-head {
  display: flex;
  align-items: center;
  padding: 24rpx 28rpx;
  border-bottom: 1rpx solid #f0f0f0;
  flex-shrink: 0;
}
.pop-title {
  font-size: 30rpx;
  font-weight: 700;
  color: #222;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pop-code {
  font-size: 22rpx;
  color: #2b8a3e;
  margin-right: 12rpx;
}
.pop-close {
  font-size: 40rpx;
  line-height: 1;
  color: #999;
  padding: 0 4rpx;
  flex-shrink: 0;
}
.pop-scroll {
  flex: 1;
  min-height: 0;
}
.pop-empty {
  padding: 100rpx 0;
  text-align: center;
  color: #999;
  font-size: 26rpx;
}
.pop-sku-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 22rpx 28rpx;
  border-bottom: 1rpx dashed #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.ps-main {
  flex: 1;
  min-width: 0;
  margin-right: 16rpx;
}
.ps-spec {
  display: block;
  font-size: 28rpx;
  font-weight: 600;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ps-stock {
  display: block;
  margin-top: 4rpx;
  font-size: 22rpx;
  color: #2b8a3e;

  &.none {
    color: #b2b2b2;
  }
}
.ps-stepper {
  display: flex;
  align-items: center;
  gap: 8rpx;
  background: #f5f6f7;
  border-radius: 12rpx;
  padding: 4rpx;
  flex-shrink: 0;
}
.pop-foot {
  padding: 16rpx 24rpx 20rpx;
  border-top: 1rpx solid #f0f0f0;
  flex-shrink: 0;
}
.pop-add-btn {
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
</style>

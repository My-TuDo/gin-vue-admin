<template>
  <view class="page">
    <!-- ================= 列表视图 ================= -->
    <template v-if="!detailVisible">
      <!-- 状态筛选 tab（本地过滤已加载数据） -->
      <view class="tabs-bar">
        <view
          v-for="t in STATUS_TABS"
          :key="t.value"
          class="tab-item"
          :class="{ active: activeStatus === t.value }"
          @click="switchTab(t.value)"
        >{{ t.name }}</view>
      </view>

      <!-- 搜索（单号，confirm 触发） -->
      <view class="search-bar">
        <input
          v-model="keyword"
          class="search-input"
          type="text"
          placeholder="搜索单号"
          confirm-type="search"
          @confirm="onSearch"
        />
        <view v-if="keyword" class="clear-btn" @click="clearSearch">清空</view>
      </view>

      <view class="list-head">
        <text class="section-title">销售单</text>
        <text class="list-count">{{ filteredOrders.length }} / {{ total }} 条</text>
      </view>

      <view v-if="loading && !orders.length" class="block-tip">加载中…</view>
      <view v-else-if="error" class="err-box">
        <text class="err-text">{{ error }}</text>
        <view class="retry-btn" @click="reload">重试</view>
      </view>
      <view v-else-if="!filteredOrders.length" class="block-tip">{{ keyword ? '未找到相关销售单' : '暂无销售记录' }}</view>

      <view v-else class="card-list">
        <view v-for="o in filteredOrders" :key="o.ID" class="order-card" @click="openDetail(o.ID)">
          <view class="oc-head">
            <text class="oc-no">{{ o.orderNo }}</text>
            <text class="type-tag" :class="typeCls(o.orderType)">{{ typeName(o.orderType) }}</text>
          </view>
          <view class="oc-mid">
            <text class="oc-amount" :class="{ neg: (o.totalAmount || 0) < 0 }">{{ fmtMoney(o.totalAmount) }}</text>
            <text class="st-tag" :class="statusCls(o.status)">{{ statusName(o.status) }}</text>
          </view>
          <view class="oc-foot">
            <text class="oc-cust">{{ custName(o) }}</text>
            <text class="oc-time">{{ fmtTime(o.createdAt) }}</text>
          </view>
        </view>
      </view>

      <view v-if="loadingMore" class="loading-more">加载中…</view>
      <view v-else-if="orders.length && orders.length >= total" class="loading-more">没有更多了</view>
    </template>

    <!-- ================= 详情视图（同页切换） ================= -->
    <template v-else>
      <view class="detail-top">
        <view class="back-btn" @click="closeDetail">‹ 返回</view>
        <text class="detail-no">{{ detail.orderNo || '销售单详情' }}</text>
        <text class="type-tag" :class="typeCls(detail.orderType)">{{ typeName(detail.orderType) }}</text>
      </view>

      <view v-if="detailLoading" class="block-tip">加载中…</view>
      <view v-else-if="detailError" class="err-box">
        <text class="err-text">{{ detailError }}</text>
        <view class="retry-btn" @click="reloadDetail">重试</view>
      </view>

      <template v-else-if="detail">
        <!-- 明细表 -->
        <view class="panel">
          <view class="panel-title">商品明细</view>
          <view class="item-row head">
            <text class="ir-goods">商品 / 规格</text>
            <text class="ir-qty">数量</text>
            <text class="ir-price">单价</text>
            <text class="ir-amount">金额</text>
          </view>
          <view v-for="(it, idx) in detail.items || []" :key="it.ID || idx" class="item-row">
            <view class="ir-goods">
              <text class="ir-name">
                {{ it.goodsName }}<text v-if="it.direction" class="dir-tag" :class="dirCls(it.direction)">{{ dirName(it.direction) }}</text>
              </text>
              <text class="ir-spec">{{ specOf(it) }}</text>
            </view>
            <text class="ir-qty">{{ it.qty }}</text>
            <text class="ir-price">{{ fmtMoney(it.price) }}</text>
            <text class="ir-amount" :class="{ neg: (it.amount || 0) < 0 }">{{ fmtMoney(it.amount) }}</text>
          </view>
          <view class="total-row">
            <text class="total-label">合计</text>
            <text class="total-amount" :class="{ neg: (detail.totalAmount || 0) < 0 }">{{ fmtMoney(detail.totalAmount) }}</text>
          </view>
        </view>

        <!-- 单信息卡 -->
        <view class="panel info-panel">
          <view class="info-row"><text class="info-label">单号</text><text class="info-value">{{ detail.orderNo }}</text></view>
          <view class="info-row"><text class="info-label">类型</text><text class="info-value">{{ typeName(detail.orderType) }}</text></view>
          <view class="info-row"><text class="info-label">状态</text><text class="info-value">{{ statusName(detail.status) }}</text></view>
          <view class="info-row"><text class="info-label">客户</text><text class="info-value">{{ custName(detail) }}</text></view>
          <view class="info-row"><text class="info-label">仓库</text><text class="info-value">{{ (detail.warehouse && detail.warehouse.name) || '—' }}</text></view>
          <view class="info-row"><text class="info-label">创建人</text><text class="info-value">{{ detail.creator || '—' }}</text></view>
          <view class="info-row"><text class="info-label">创建时间</text><text class="info-value">{{ fmtTime(detail.createdAt) }}</text></view>
          <view v-if="detail.orderType !== 1" class="info-row">
            <text class="info-label">关联原单</text><text class="info-value">{{ originalNo }}</text>
          </view>
          <view v-if="detail.remark" class="info-row"><text class="info-label">备注</text><text class="info-value">{{ detail.remark }}</text></view>
        </view>
      </template>
    </template>
  </view>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onLoad, onReachBottom } from '@dcloudio/uni-app'
import { get } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

// ===== 状态/类型映射常量 =====
// 筛选 tab：0 全部，其余对应后端 status（1 待出库 / 2 已出库 / 3 已取消）
const STATUS_TABS = [
  { name: '全部', value: 0 },
  { name: '待出库', value: 1 },
  { name: '已出库', value: 2 },
  { name: '已取消', value: 3 }
]
const ORDER_TYPE_MAP = {
  1: { name: '销售', cls: 'type-sale' },
  2: { name: '退货', cls: 'type-return' },
  3: { name: '换货', cls: 'type-exchange' }
}
const STATUS_MAP = {
  1: { name: '待出库', cls: 'st-pending' },
  2: { name: '已出库', cls: 'st-shipped' },
  3: { name: '已取消', cls: 'st-canceled' }
}
const DIR_MAP = {
  1: { name: '换出', cls: 'dir-out' },
  2: { name: '换入', cls: 'dir-in' }
}

// ===== 列表状态 =====
const orders = ref([]) // 已加载列表（含 customer/warehouse 关联）
const total = ref(0)
const page = ref(1)
const pageSize = 20
const keyword = ref('')
const activeStatus = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')

// ===== 详情状态 =====
const detailVisible = ref(false)
const detail = ref(null)
const detailLoading = ref(false)
const detailError = ref('')
const originalNo = ref('')

// 本地过滤：按当前状态 tab 过滤已加载数据
const filteredOrders = computed(() => {
  if (activeStatus.value === 0) return orders.value
  return orders.value.filter((o) => o.status === activeStatus.value)
})

function typeName(t) { return (ORDER_TYPE_MAP[t] || {}).name || '未知' }
function typeCls(t) { return (ORDER_TYPE_MAP[t] || {}).cls || '' }
function statusName(s) { return (STATUS_MAP[s] || {}).name || '未知' }
function statusCls(s) { return (STATUS_MAP[s] || {}).cls || '' }
function dirName(d) { return (DIR_MAP[d] || {}).name || '' }
function dirCls(d) { return (DIR_MAP[d] || {}).cls || '' }

// 客户显示：无 customer 关联为散客
function custName(o) {
  return (o.customer && o.customer.name) || '散客'
}

// 明细规格：颜色 / 尺码（无则「—」）
function specOf(it) {
  const parts = [it.color, it.size].filter(Boolean)
  return parts.length ? parts.join(' / ') : '—'
}

function fmtMoney(n) {
  const num = Number(n) || 0
  const sign = num < 0 ? '-' : ''
  const fixed = Math.abs(num).toFixed(2)
  return `${sign}¥${fixed.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}`
}

// createdAt（ISO 字符串）→ YYYY-MM-DD HH:mm（容错：非法时间显示 —）
function fmtTime(str) {
  if (!str) return '—'
  const d = new Date(str)
  if (Number.isNaN(d.getTime())) return '—'
  const p = (x) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

// ===== 列表加载 =====
async function load(append) {
  const p = append ? page.value + 1 : 1
  if (append) {
    loadingMore.value = true
  } else {
    loading.value = true
    error.value = ''
  }
  try {
    const data = await get('/jxc/sale/page', { page: p, pageSize, keyword: keyword.value.trim() })
    const list = (data && data.list) || []
    total.value = Number((data && data.total) || 0)
    orders.value = append ? orders.value.concat(list) : list
    page.value = p
  } catch (e) {
    if (append) {
      // 上拉加载失败：静默（保留已加载数据），下次触底重试
      uni.showToast({ title: '加载失败', icon: 'none' })
    } else {
      error.value = '销售记录加载失败'
    }
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function reload() { load(false) }

function onSearch() { load(false) }

function clearSearch() {
  keyword.value = ''
  load(false)
}

function switchTab(v) {
  if (activeStatus.value === v) return
  activeStatus.value = v
}

// 触底加载更多（详情视图内滚动不触发）
onReachBottom(() => {
  if (detailVisible.value) return
  if (loading.value || loadingMore.value) return
  if (orders.value.length >= total.value) return
  load(true)
})

// ===== 详情 =====
async function openDetail(id) {
  detailVisible.value = true
  detail.value = null
  detailError.value = ''
  originalNo.value = ''
  detailLoading.value = true
  try {
    const data = await get('/jxc/sale/detail', { id })
    detail.value = data
    // 关联原单号：detail 已 Preload original（后端 M5 已补），无值时才用 remaining 接口兜底（静默）
    if (data && data.original?.orderNo) {
      originalNo.value = data.original.orderNo
    } else if (data && data.orderType !== 1 && data.originalOrderId) {
      fetchOriginalNo(data.originalOrderId)
    }
  } catch (e) {
    detailError.value = '详情加载失败'
  } finally {
    detailLoading.value = false
  }
}

async function fetchOriginalNo(originalId) {
  try {
    const data = await get('/jxc/sale/remaining', { id: originalId }, { silent: true })
    const first = (Array.isArray(data) && data[0]) || null
    originalNo.value = (first && first.orderNo) || '—'
  } catch (e) {
    originalNo.value = '—'
  }
}

function reloadDetail() {
  const id = detail.value && detail.value.ID
  if (id) openDetail(id)
}

function closeDetail() {
  detailVisible.value = false
  detail.value = null
}

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  load(false)
})
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  padding: 24rpx;
  box-sizing: border-box;
  padding-bottom: 40rpx;
}

/* ===== 状态筛选 tab ===== */
.tabs-bar {
  display: flex;
  gap: 12rpx;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 12rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.tab-item {
  flex: 1;
  height: 68rpx;
  line-height: 68rpx;
  text-align: center;
  background: #f5f6f7;
  color: #666;
  font-size: 26rpx;
  font-weight: 600;
  border-radius: 12rpx;

  &.active {
    background: linear-gradient(135deg, #2b8a3e 0%, #3fb45c 100%);
    color: #fff;
    box-shadow: 0 6rpx 16rpx rgba(43, 138, 62, 0.25);
  }
}

/* ===== 搜索栏 ===== */
.search-bar {
  display: flex;
  align-items: center;
  margin-top: 16rpx;
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
.clear-btn {
  margin-left: 16rpx;
  padding: 0 20rpx;
  height: 72rpx;
  line-height: 72rpx;
  font-size: 24rpx;
  color: #999;
  flex-shrink: 0;
}

/* ===== 列表 ===== */
.list-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin: 20rpx 4rpx 12rpx;
}
.section-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
}
.list-count {
  font-size: 22rpx;
  color: #999;
}
.card-list {
  background: #ffffff;
  border-radius: 16rpx;
  overflow: hidden;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
}
.order-card {
  padding: 22rpx 24rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.oc-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12rpx;
}
.oc-no {
  font-size: 28rpx;
  font-weight: 700;
  color: #222;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 16rpx;
}
.oc-mid {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10rpx;
}
.oc-amount {
  font-size: 32rpx;
  font-weight: 700;
  color: #222;

  &.neg {
    color: #e64340;
  }
}
.oc-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.oc-cust {
  font-size: 24rpx;
  color: #666;
}
.oc-time {
  font-size: 22rpx;
  color: #999;
}

/* ===== 类型徽标 ===== */
.type-tag {
  font-size: 20rpx;
  font-weight: 600;
  padding: 4rpx 14rpx;
  border-radius: 8rpx;
  flex-shrink: 0;
}
.type-sale { background: #e8f5e9; color: #2b8a3e; }
.type-return { background: #fdeaea; color: #e64340; }
.type-exchange { background: #e8f0fe; color: #2f6fde; }

/* ===== 状态徽标 ===== */
.st-tag {
  font-size: 22rpx;
  font-weight: 600;
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
  flex-shrink: 0;
}
.st-pending { background: #fff4e5; color: #e6a23c; }
.st-shipped { background: #e8f5e9; color: #2b8a3e; }
.st-canceled { background: #f0f0f0; color: #999; }

/* ===== 状态提示（加载中/空态/错误） ===== */
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
.loading-more {
  padding: 24rpx 0 8rpx;
  text-align: center;
  color: #b2b2b2;
  font-size: 22rpx;
}

/* ================= 详情视图 ================= */
.detail-top {
  display: flex;
  align-items: center;
  background: #ffffff;
  border-radius: 16rpx;
  padding: 20rpx 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 16rpx;
}
.back-btn {
  font-size: 28rpx;
  color: #2b8a3e;
  font-weight: 600;
  padding-right: 20rpx;
  flex-shrink: 0;
}
.detail-no {
  flex: 1;
  min-width: 0;
  font-size: 30rpx;
  font-weight: 700;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 16rpx;
}
.panel {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 20rpx 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
  margin-bottom: 16rpx;
}
.panel-title {
  font-size: 28rpx;
  font-weight: 700;
  color: #222;
  margin-bottom: 16rpx;
}
.item-row {
  display: flex;
  align-items: center;
  padding: 14rpx 0;
  border-bottom: 1rpx dashed #f0f0f0;

  &.head {
    border-bottom: 1rpx solid #eef0ef;
    padding: 8rpx 0;

    .ir-goods, .ir-qty, .ir-price, .ir-amount {
      font-size: 22rpx;
      color: #999;
      font-weight: 400;
    }
  }

  &:last-child {
    border-bottom: none;
  }
}
.ir-goods {
  flex: 1;
  min-width: 0;
  margin-right: 12rpx;
  display: flex;
  flex-direction: column;
}
.ir-name {
  font-size: 26rpx;
  font-weight: 600;
  color: #222;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dir-tag {
  display: inline-block;
  font-size: 18rpx;
  font-weight: 600;
  padding: 2rpx 10rpx;
  border-radius: 6rpx;
  margin-left: 10rpx;
  vertical-align: middle;
}
.dir-out { background: #e8f0fe; color: #2f6fde; }
.dir-in { background: #fdeaea; color: #e64340; }
.ir-spec {
  margin-top: 2rpx;
  font-size: 20rpx;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ir-qty {
  width: 70rpx;
  text-align: center;
  font-size: 26rpx;
  color: #333;
  flex-shrink: 0;
}
.ir-price {
  width: 110rpx;
  text-align: right;
  font-size: 24rpx;
  color: #666;
  flex-shrink: 0;
}
.ir-amount {
  width: 130rpx;
  text-align: right;
  font-size: 26rpx;
  font-weight: 600;
  color: #222;
  flex-shrink: 0;

  &.neg {
    color: #e64340;
  }
}
.total-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 16rpx;
}
.total-label {
  font-size: 26rpx;
  font-weight: 600;
  color: #666;
}
.total-amount {
  font-size: 32rpx;
  font-weight: 700;
  color: #222;

  &.neg {
    color: #e64340;
  }
}

/* 单信息卡 */
.info-panel {
  padding: 8rpx 24rpx;
}
.info-row {
  display: flex;
  align-items: flex-start;
  padding: 14rpx 0;
  border-bottom: 1rpx dashed #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.info-label {
  width: 140rpx;
  font-size: 24rpx;
  color: #999;
  flex-shrink: 0;
}
.info-value {
  flex: 1;
  min-width: 0;
  font-size: 26rpx;
  color: #333;
  word-break: break-all;
}
</style>

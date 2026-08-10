<template>
  <view class="page">
    <!-- ================= 列表视图 ================= -->
    <template v-if="!detailVisible">
      <!-- 搜索（名称/电话，confirm 触发） -->
      <view class="search-bar">
        <input
          v-model="keyword"
          class="search-input"
          type="text"
          placeholder="搜索客户名称 / 电话"
          confirm-type="search"
          @confirm="onSearch"
        />
        <view v-if="keyword" class="clear-btn" @click="clearSearch">清空</view>
      </view>

      <view class="list-head">
        <text class="section-title">客户</text>
        <text class="list-count">{{ customers.length }} / {{ total }} 条</text>
      </view>

      <view v-if="loading && !customers.length" class="block-tip">加载中…</view>
      <view v-else-if="error" class="err-box">
        <text class="err-text">{{ error }}</text>
        <view class="retry-btn" @click="reload">重试</view>
      </view>
      <view v-else-if="!customers.length" class="block-tip">{{ keyword ? '未找到相关客户' : '暂无客户数据' }}</view>

      <view v-else class="card-list">
        <view v-for="c in customers" :key="c.ID" class="cust-card" @click="openDetail(c.ID)">
          <view class="cc-top">
            <text class="cc-name">{{ c.name }}</text>
            <text class="lv-tag" :class="levelCls(c.level)">{{ levelName(c.level) }}</text>
            <text v-if="c.status !== 1" class="st-tag st-off">已停用</text>
          </view>
          <view class="cc-mid">
            <text class="cc-phone">{{ c.phone || '无电话' }}</text>
            <text class="cc-code">{{ c.code }}</text>
          </view>
          <view v-if="c.address" class="cc-addr">{{ c.address }}</view>
        </view>
      </view>

      <view v-if="loadingMore" class="loading-more">加载中…</view>
      <view v-else-if="customers.length && customers.length >= total" class="loading-more">没有更多了</view>
    </template>

    <!-- ================= 详情视图（同页切换） ================= -->
    <template v-else>
      <view class="detail-top">
        <view class="back-btn" @click="closeDetail">‹ 返回</view>
        <text class="detail-no">{{ detail.name || '客户详情' }}</text>
        <text class="lv-tag" :class="levelCls(detail.level)">{{ levelName(detail.level) }}</text>
      </view>

      <view class="panel info-panel">
        <view class="info-row"><text class="info-label">编码</text><text class="info-value">{{ detail.code || '—' }}</text></view>
        <view class="info-row"><text class="info-label">等级</text><text class="info-value">{{ levelName(detail.level) }}</text></view>
        <view class="info-row"><text class="info-label">电话</text><text class="info-value">{{ detail.phone || '—' }}</text></view>
        <view class="info-row"><text class="info-label">地址</text><text class="info-value">{{ detail.address || '—' }}</text></view>
        <view class="info-row"><text class="info-label">状态</text><text class="info-value">{{ detail.status === 1 ? '正常' : '已停用' }}</text></view>
        <view v-if="detail.remark" class="info-row"><text class="info-label">备注</text><text class="info-value">{{ detail.remark }}</text></view>
      </view>
    </template>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad, onReachBottom } from '@dcloudio/uni-app'
import { get } from '@/utils/request'
import { isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

// ===== 客户等级映射（1 普通 / 2 VIP / 3 批发） =====
const LEVEL_MAP = {
  1: { name: '普通', cls: 'lv-normal' },
  2: { name: 'VIP', cls: 'lv-vip' },
  3: { name: '批发', cls: 'lv-wholesale' }
}

// ===== 列表状态 =====
const customers = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const keyword = ref('')
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')

// ===== 详情状态 =====
const detailVisible = ref(false)
const detail = ref(null)

function levelName(l) { return (LEVEL_MAP[l] || {}).name || '未知' }
function levelCls(l) { return (LEVEL_MAP[l] || {}).cls || '' }

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
    const data = await get('/jxc/customer/list', { page: p, pageSize, keyword: keyword.value.trim() })
    const list = (data && data.list) || []
    total.value = Number((data && data.total) || 0)
    customers.value = append ? customers.value.concat(list) : list
    page.value = p
  } catch (e) {
    if (append) {
      uni.showToast({ title: '加载失败', icon: 'none' })
    } else {
      error.value = '客户列表加载失败'
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

onReachBottom(() => {
  if (detailVisible.value) return
  if (loading.value || loadingMore.value) return
  if (customers.value.length >= total.value) return
  load(true)
})

// ===== 详情 =====
function openDetail(id) {
  // 列表数据已含全部展示字段，直接复用（不额外请求，避免详情二次接口）
  const c = customers.value.find((x) => x.ID === id)
  detailVisible.value = true
  detail.value = c || {}
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

/* ===== 搜索栏 ===== */
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
.cust-card {
  padding: 22rpx 24rpx;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}
.cc-top {
  display: flex;
  align-items: center;
}
.cc-name {
  font-size: 30rpx;
  font-weight: 700;
  color: #222;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 16rpx;
}
.cc-mid {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8rpx;
}
.cc-phone {
  font-size: 24rpx;
  color: #666;
}
.cc-code {
  font-size: 22rpx;
  color: #b2b2b2;
}
.cc-addr {
  margin-top: 6rpx;
  font-size: 22rpx;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ===== 等级徽标 ===== */
.lv-tag {
  font-size: 20rpx;
  font-weight: 600;
  padding: 4rpx 14rpx;
  border-radius: 8rpx;
  flex-shrink: 0;
}
.lv-normal { background: #f0f0f0; color: #666; }
.lv-vip { background: #fdf6e3; color: #b8860b; }
.lv-wholesale { background: #e8f0fe; color: #2f6fde; }

/* ===== 状态徽标 ===== */
.st-tag {
  font-size: 20rpx;
  font-weight: 600;
  padding: 4rpx 14rpx;
  border-radius: 8rpx;
  flex-shrink: 0;
  margin-left: 12rpx;
}
.st-off { background: #f0f0f0; color: #999; }

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
.info-panel {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 8rpx 24rpx;
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.04);
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

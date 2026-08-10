<template>
  <view class="ui-page page-customer">
    <!-- ================= 列表视图 ================= -->
    <template v-if="!detailVisible">
      <!-- 搜索（名称/电话，confirm 触发） -->
      <view class="ui-search">
        <input
          v-model="keyword"
          class="ui-search-input"
          type="text"
          placeholder="搜索客户名称 / 电话"
          confirm-type="search"
          @confirm="onSearch"
        />
        <view v-if="keyword" class="ui-search-action" @click="clearSearch">清空</view>
      </view>

      <view class="list-head">
        <view class="ui-section-title">客户</view>
        <text class="list-count">{{ customers.length }} / {{ total }} 条</text>
      </view>

      <view v-if="loading && !customers.length" class="ui-empty">加载中…</view>
      <view v-else-if="error" class="ui-err">
        <text class="ui-err-text">{{ error }}</text>
        <view class="ui-retry" hover-class="ui-hover" hover-stay-time="60" @click="reload">重试</view>
      </view>
      <view v-else-if="!customers.length" class="ui-empty">
        <text class="ui-empty-main">{{ keyword ? '未找到相关客户' : '暂无客户数据' }}</text>
        <text class="ui-empty-sub">{{ keyword ? '请核对名称或电话后重试' : '客户在 PC 端维护，此处实时同步' }}</text>
      </view>

      <view v-else class="card-list">
        <view v-for="c in customers" :key="c.ID" class="cust-card" hover-class="ui-hover" hover-stay-time="60" @click="openDetail(c.ID)">
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
        <view class="back-btn" hover-class="ui-hover" hover-stay-time="60" @click="closeDetail">‹ 返回</view>
        <text class="detail-no">{{ detail.name || '客户详情' }}</text>
        <text class="lv-tag" :class="levelCls(detail.level)">{{ levelName(detail.level) }}</text>
      </view>

      <view class="panel ui-card info-panel">
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
.page-customer {
  padding-bottom: 40rpx; /* 其余 padding 复用 .ui-page */
}

/* ===== 列表 ===== */
.list-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin: 20rpx 4rpx 12rpx;
}
.list-count {
  font-size: $ui-font-xs;
  color: $ui-text-3;
}
.card-list {
  background: $ui-bg-card;
  border-radius: $ui-radius-md;
  overflow: hidden;
  box-shadow: $ui-shadow-card;
}
.cust-card {
  padding: 22rpx $ui-space-md;
  border-bottom: 1rpx solid $ui-border;

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
  color: $ui-text-1;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: $ui-space-sm;
}
.cc-mid {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8rpx;
}
.cc-phone {
  font-size: $ui-font-sm;
  color: $ui-text-2;
}
.cc-code {
  font-size: $ui-font-xs;
  color: $ui-text-3;
}
.cc-addr {
  margin-top: 6rpx;
  font-size: $ui-font-xs;
  color: $ui-text-3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ===== 等级徽标（胶囊） ===== */
.lv-tag {
  font-size: $ui-font-xs;
  font-weight: 600;
  padding: 4rpx 14rpx;
  border-radius: $ui-radius-full;
  flex-shrink: 0;
}
.lv-normal { background: $ui-bg-disabled; color: $ui-text-2; }
.lv-vip { background: $ui-warning-bg; color: $ui-warning-text; }
.lv-wholesale { background: $ui-blue-bg; color: $ui-blue; }

/* ===== 状态徽标 ===== */
.st-tag {
  font-size: $ui-font-xs;
  font-weight: 600;
  padding: 4rpx 14rpx;
  border-radius: $ui-radius-full;
  flex-shrink: 0;
  margin-left: 12rpx;
}
.st-off { background: $ui-bg-disabled; color: $ui-text-3; }

.loading-more {
  padding: $ui-space-md 0 8rpx;
  text-align: center;
  color: $ui-text-3;
  font-size: $ui-font-xs;
}

/* ================= 详情视图 ================= */
.detail-top {
  display: flex;
  align-items: center;
  background: $ui-bg-card;
  border-radius: $ui-radius-md;
  padding: 20rpx $ui-space-md;
  box-shadow: $ui-shadow-card;
  margin-bottom: $ui-space-sm;
}
.back-btn {
  font-size: $ui-font-base;
  color: $ui-primary;
  font-weight: 600;
  padding-right: 20rpx;
  flex-shrink: 0;
}
.detail-no {
  flex: 1;
  min-width: 0;
  font-size: 30rpx;
  font-weight: 700;
  color: $ui-text-1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: $ui-space-sm;
}
.info-panel {
  padding: 8rpx $ui-space-md;
}
.info-row {
  display: flex;
  align-items: flex-start;
  padding: 14rpx 0;
  border-bottom: 1rpx dashed $ui-border;

  &:last-child {
    border-bottom: none;
  }
}
.info-label {
  width: 140rpx;
  font-size: $ui-font-sm;
  color: $ui-text-3;
  flex-shrink: 0;
}
.info-value {
  flex: 1;
  min-width: 0;
  font-size: 26rpx;
  color: $ui-text-1;
  word-break: break-all;
}
</style>

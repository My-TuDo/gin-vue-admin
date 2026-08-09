<template>
  <view class="page">
    <!-- 用户信息 -->
    <view class="profile">
      <image v-if="user.headerImg" class="avatar" :src="user.headerImg" mode="aspectFill" />
      <view v-else class="avatar avatar-placeholder">J</view>
      <view class="profile-main">
        <text class="nickname">{{ user.nickName || '系统用户' }}</text>
        <text class="username">@{{ user.userName }}</text>
      </view>
    </view>

    <view class="info-card">
      <view class="info-row">
        <text class="info-label">角色 ID</text>
        <text class="info-value">{{ user.authorityId != null ? user.authorityId : '-' }}</text>
      </view>
      <view class="info-row">
        <text class="info-label">手机号</text>
        <text class="info-value">{{ user.phone || '-' }}</text>
      </view>
      <view class="info-row">
        <text class="info-label">邮箱</text>
        <text class="info-value">{{ user.email || '-' }}</text>
      </view>
      <view class="info-row">
        <text class="info-label">部门</text>
        <text class="info-value">{{ (user.dept && user.dept.name) || '-' }}</text>
      </view>
    </view>

    <!-- 收银台绑定：绑定后小程序扫码商品自动加入 PC 收银台购物车 -->
    <view class="pos-card">
      <view class="pos-card-head">
        <text class="pos-card-title">收银台</text>
        <text class="pos-card-state" :class="{ bound: !!posSession }">{{ posSession ? '已绑定' : '未绑定' }}</text>
      </view>
      <view v-if="posSession" class="pos-bound">
        <view class="pos-code">
          <text class="pos-code-label">收银台码</text>
          <text class="pos-code-value">{{ posSession }}</text>
        </view>
        <view class="pos-actions">
          <view class="pos-btn" @click="openBind(true)">重新绑定</view>
          <view class="pos-btn pos-btn-danger" @click="unbind">解绑</view>
        </view>
      </view>
      <view v-else class="pos-unbound" @click="openBind(false)">
        <text class="pos-unbound-text">绑定收银台</text>
        <text class="pos-unbound-arrow">›</text>
      </view>
      <view class="pos-tip">绑定后，扫码的商品将自动加入 PC 收银台购物车</view>
      <view class="pos-tip">扫码入口在首页「扫码加购」</view>
    </view>

    <!-- 绑定收银台弹层（自绘，兼容 H5 与小程序） -->
    <view v-if="bindVisible" class="bind-mask" @click="bindVisible = false">
      <view class="bind-pop" @click.stop>
        <view class="bind-title">{{ bindRebind ? '重新绑定收银台' : '绑定收银台' }}</view>
        <view class="bind-desc">请输入 PC 收银台设置的 6 位收银台码</view>
        <input
          class="bind-input"
          v-model="bindCode"
          maxlength="6"
          type="text"
          placeholder="如 A2B3C4"
          @input="onCodeInput"
        />
        <view class="bind-btns">
          <view class="bind-btn" @click="bindVisible = false">取消</view>
          <view class="bind-btn bind-btn-primary" @click="doBind">确认绑定</view>
        </view>
      </view>
    </view>

    <button class="logout-btn" @click="handleLogout">退出登录</button>

    <view class="version">v1.0.0 · 进销存小程序</view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getUserInfo, clearAuth, isLoggedIn } from '@/utils/auth'
import { get } from '@/utils/request'
import { LOGIN_PAGE } from '@/config'

const POS_SESSION_KEY = 'posSession'

const user = ref({})
const posSession = ref(uni.getStorageSync(POS_SESSION_KEY) || '')
const bindVisible = ref(false)
const bindRebind = ref(false)
const bindCode = ref('')

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  user.value = getUserInfo() || {}
})

// 打开绑定弹层；rebind=true 表示已绑定状态下的「重新绑定」
function openBind(rebind) {
  bindRebind.value = rebind
  bindCode.value = ''
  bindVisible.value = true
}

// 输入框自动转大写，仅保留 0-9A-Z（收银台码为 6 位大写字母数字）
function onCodeInput(e) {
  bindCode.value = String(e.detail.value || '').toUpperCase().replace(/[^0-9A-Z]/g, '')
}

// 绑定：先调 check 校验码有效，成功才写入本地 storage
async function doBind() {
  const code = bindCode.value.trim()
  if (code.length !== 6) {
    uni.showToast({ title: '请输入 6 位收银台码', icon: 'none' })
    return
  }
  uni.showLoading({ title: '校验中…' })
  try {
    await get('/jxc/pos/session/check', { code }, { silent: true })
    uni.setStorageSync(POS_SESSION_KEY, code)
    posSession.value = code
    bindVisible.value = false
    uni.showToast({ title: '绑定成功', icon: 'success' })
  } catch (e) {
    // 校验失败：展示后端 msg（如「收银台码无效」）
    uni.showToast({ title: (e && e.msg) || '绑定失败，请确认收银台码', icon: 'none' })
  } finally {
    uni.hideLoading()
  }
}

// 解绑：清除本地绑定，之后扫码需重新绑定
function unbind() {
  uni.showModal({
    title: '提示',
    content: '解绑后扫码的商品将无法自动加入收银台，确定解绑吗？',
    success: (res) => {
      if (!res.confirm) return
      uni.removeStorageSync(POS_SESSION_KEY)
      posSession.value = ''
    }
  })
}

function handleLogout() {
  uni.showModal({
    title: '提示',
    content: '确定退出当前账号吗？',
    success: (res) => {
      if (!res.confirm) return
      clearAuth()
      uni.reLaunch({ url: LOGIN_PAGE })
    }
  })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  padding: 24rpx;
  box-sizing: border-box;
}

.profile {
  display: flex;
  align-items: center;
  background: linear-gradient(135deg, #2b8a3e 0%, #3fb45c 100%);
  border-radius: 16rpx;
  padding: 40rpx 32rpx;
  margin-bottom: 24rpx;
}

.avatar {
  width: 120rpx;
  height: 120rpx;
  border-radius: 50%;
  background: #fff;
  margin-right: 24rpx;
}

.avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 56rpx;
  font-weight: 600;
  color: #2b8a3e;
}

.profile-main {
  display: flex;
  flex-direction: column;
}

.nickname {
  font-size: 34rpx;
  font-weight: 600;
  color: #fff;
}

.username {
  margin-top: 8rpx;
  font-size: 24rpx;
  color: rgba(255, 255, 255, 0.8);
}

.info-card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 8rpx 32rpx;
  margin-bottom: 24rpx;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 26rpx 0;
  border-bottom: 1rpx solid #f0f0f0;

  &:last-child {
    border-bottom: none;
  }
}

.info-label {
  font-size: 26rpx;
  color: #999;
}

.info-value {
  font-size: 26rpx;
  color: #222;
  max-width: 400rpx;
  text-align: right;
  word-break: break-all;
}

.logout-btn {
  margin-top: 32rpx;
  height: 88rpx;
  line-height: 88rpx;
  background: #ffffff;
  color: #e64340;
  font-size: 30rpx;
  border-radius: 16rpx;

  &::after {
    border: none;
  }
}

/* ===== 收银台绑定 ===== */
.pos-card {
  background: #ffffff;
  border-radius: 16rpx;
  padding: 28rpx 32rpx;
  margin-bottom: 24rpx;
}
.pos-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18rpx;
}
.pos-card-title {
  font-size: 30rpx;
  font-weight: 600;
  color: #222;
}
.pos-card-state {
  font-size: 22rpx;
  color: #999;
  background: #f5f6f7;
  border-radius: 20rpx;
  padding: 4rpx 18rpx;

  &.bound {
    color: #2b8a3e;
    background: #e8f5e9;
  }
}
.pos-code {
  display: flex;
  align-items: baseline;
  background: #f8fafc;
  border-radius: 12rpx;
  padding: 20rpx 24rpx;
  margin-bottom: 18rpx;
}
.pos-code-label {
  font-size: 24rpx;
  color: #999;
}
.pos-code-value {
  margin-left: 20rpx;
  font-size: 40rpx;
  font-weight: 700;
  letter-spacing: 6rpx;
  color: #2b8a3e;
}
.pos-actions {
  display: flex;
  gap: 16rpx;
}
.pos-btn {
  flex: 1;
  height: 72rpx;
  line-height: 72rpx;
  text-align: center;
  background: #e8f5e9;
  color: #2b8a3e;
  font-size: 26rpx;
  font-weight: 600;
  border-radius: 12rpx;

  &.pos-btn-danger {
    background: #fdf0ef;
    color: #e64340;
  }
}
.pos-unbound {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #f8fafc;
  border-radius: 12rpx;
  padding: 24rpx 28rpx;
}
.pos-unbound-text {
  font-size: 28rpx;
  color: #222;
  font-weight: 600;
}
.pos-unbound-arrow {
  font-size: 34rpx;
  color: #b2b2b2;
}
.pos-tip {
  margin-top: 16rpx;
  font-size: 22rpx;
  color: #b2b2b2;
}

/* ===== 绑定弹层 ===== */
.bind-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
}
.bind-pop {
  width: 600rpx;
  background: #ffffff;
  border-radius: 20rpx;
  padding: 40rpx 36rpx 32rpx;
}
.bind-title {
  font-size: 32rpx;
  font-weight: 700;
  color: #222;
  text-align: center;
}
.bind-desc {
  margin-top: 10rpx;
  font-size: 24rpx;
  color: #999;
  text-align: center;
}
.bind-input {
  margin-top: 28rpx;
  height: 88rpx;
  line-height: 88rpx;
  background: #f5f6f7;
  border-radius: 14rpx;
  padding: 0 28rpx;
  font-size: 34rpx;
  font-weight: 700;
  letter-spacing: 8rpx;
  text-align: center;
}
.bind-btns {
  display: flex;
  gap: 16rpx;
  margin-top: 32rpx;
}
.bind-btn {
  flex: 1;
  height: 80rpx;
  line-height: 80rpx;
  text-align: center;
  background: #f5f6f7;
  color: #666;
  font-size: 28rpx;
  border-radius: 12rpx;

  &.bind-btn-primary {
    background: #2b8a3e;
    color: #ffffff;
    font-weight: 600;
  }
}

.version {
  margin-top: 48rpx;
  text-align: center;
  font-size: 22rpx;
  color: #b2b2b2;
}
</style>

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

    <button class="logout-btn" @click="handleLogout">退出登录</button>

    <view class="version">v1.0.0 · 进销存小程序</view>
  </view>
</template>

<script setup>
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { getUserInfo, clearAuth, isLoggedIn } from '@/utils/auth'
import { LOGIN_PAGE } from '@/config'

const user = ref({})

onLoad(() => {
  if (!isLoggedIn()) {
    uni.reLaunch({ url: LOGIN_PAGE })
    return
  }
  user.value = getUserInfo() || {}
})

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

.version {
  margin-top: 48rpx;
  text-align: center;
  font-size: 22rpx;
  color: #b2b2b2;
}
</style>

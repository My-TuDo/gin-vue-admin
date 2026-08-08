<template>
  <view class="login-page">
    <view class="login-card">
      <view class="login-title">进销存管理系统</view>
      <view class="login-subtitle">经营 · 库存 · 财务一站式管理</view>

      <view class="form-item">
        <text class="form-label">用户名</text>
        <input
          class="form-input"
          v-model="form.username"
          placeholder="请输入用户名"
          placeholder-class="ph"
          :maxlength="32"
        />
      </view>

      <view class="form-item">
        <text class="form-label">密码</text>
        <input
          class="form-input"
          v-model="form.password"
          password
          placeholder="请输入密码"
          placeholder-class="ph"
          :maxlength="64"
          confirm-type="done"
          @confirm="handleLogin"
        />
      </view>

      <view class="form-item captcha-row">
        <input
          class="form-input captcha-input"
          v-model="form.captcha"
          placeholder="验证码"
          placeholder-class="ph"
          :maxlength="8"
        />
        <image v-if="captchaImg" class="captcha-img" :src="captchaImg" mode="aspectFit" @click="loadCaptcha" />
        <text v-else class="captcha-loading" @click="loadCaptcha">点击获取验证码</text>
      </view>

      <button class="login-btn" :loading="loading" :disabled="loading" @click="handleLogin">
        登 录
      </button>
      <view class="login-tip">账号密码由系统管理员分配（默认 admin / 123456）</view>
    </view>
  </view>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { post } from '@/utils/request'
import md5 from '@/utils/md5'
import { setToken, setUserInfo, setExpiresAt, isLoggedIn } from '@/utils/auth'
import { INDEX_PAGE } from '@/config'

const form = reactive({
  username: '',
  password: '',
  captcha: ''
})
const loading = ref(false)
const captchaId = ref('')
const captchaImg = ref('')

// 加载验证码（GVA: POST /base/captcha → {captchaId, picPath(base64), openCaptcha}）
async function loadCaptcha() {
  try {
    const data = await post('/base/captcha', {}, { silent: true })
    if (data && data.captchaId) {
      captchaId.value = data.captchaId
      captchaImg.value = data.picPath
      form.captcha = ''
    }
  } catch (e) {
    // 验证码接口失败不阻塞登录页展示（openCaptcha=false 时后端不校验验证码）
    console.error('[captcha] 获取验证码失败', e)
    uni.showToast({ title: '验证码加载失败，请检查后端服务', icon: 'none' })
  }
}

// 已登录直接进首页
onLoad(() => {
  if (isLoggedIn()) {
    uni.reLaunch({ url: INDEX_PAGE })
  }
  loadCaptcha()
})

async function handleLogin() {
  const username = form.username.trim()
  if (!username || !form.password) {
    uni.showToast({ title: '请输入用户名和密码', icon: 'none' })
    return
  }
  loading.value = true
  try {
    // 后端校验的是密码的 MD5（小写 hex）；验证码按 GVA 约定提交 captcha/captchaId
    const data = await post('/base/login', {
      username,
      password: md5(form.password),
      captcha: form.captcha,
      captchaId: captchaId.value
    })
    if (!data || !data.token) {
      throw new Error('登录响应缺少 token')
    }
    setToken(data.token)
    setUserInfo(data.user || {})
    setExpiresAt(data.expiresAt || 0)
    uni.showToast({ title: '登录成功', icon: 'success' })
    setTimeout(() => {
      uni.reLaunch({ url: INDEX_PAGE })
    }, 500)
  } catch (e) {
    // 登录失败刷新验证码（失败次数累计会触发验证码校验）
    loadCaptcha()
    console.error('[login] 登录失败', e)
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40rpx;
  box-sizing: border-box;
  background: linear-gradient(160deg, #eef7f0 0%, #f6f7f9 100%);
}

.login-card {
  width: 100%;
  background: #ffffff;
  border-radius: 24rpx;
  padding: 60rpx 48rpx;
  box-shadow: 0 8rpx 40rpx rgba(0, 0, 0, 0.06);
  box-sizing: border-box;
}

.login-title {
  font-size: 44rpx;
  font-weight: 600;
  text-align: center;
  color: #222;
}

.login-subtitle {
  margin-top: 16rpx;
  margin-bottom: 56rpx;
  font-size: 24rpx;
  color: #999;
  text-align: center;
}

.form-item {
  margin-bottom: 32rpx;
}

.form-label {
  display: block;
  font-size: 26rpx;
  color: #666;
  margin-bottom: 12rpx;
}

.form-input {
  height: 88rpx;
  padding: 0 24rpx;
  background: #f5f6f7;
  border-radius: 12rpx;
  font-size: 28rpx;
}

.ph {
  color: #b2b2b2;
}

.login-btn {
  margin-top: 16rpx;
  height: 88rpx;
  line-height: 88rpx;
  background: #2b8a3e;
  color: #fff;
  font-size: 32rpx;
  border-radius: 12rpx;

  &::after {
    border: none;
  }
}

.login-tip {
  margin-top: 32rpx;
  font-size: 22rpx;
  color: #b2b2b2;
  text-align: center;
}

.captcha-row {
  display: flex;
  align-items: center;
}

.captcha-input {
  flex: 1;
}

.captcha-img {
  width: 200rpx;
  height: 88rpx;
  margin-left: 16rpx;
  border-radius: 12rpx;
  background: #f0f1f2;
}

.captcha-loading {
  margin-left: 16rpx;
  width: 200rpx;
  height: 88rpx;
  line-height: 88rpx;
  text-align: center;
  font-size: 22rpx;
  color: #2b8a3e;
  background: #f0f1f2;
  border-radius: 12rpx;
}
</style>

/**
 * token / 用户信息 本地存储封装
 * 存储介质：uni storage（小程序本地缓存，与 uni.request 天然配套）
 */

const TOKEN_KEY = 'token'
const USER_INFO_KEY = 'userInfo'
const EXPIRES_AT_KEY = 'expiresAt'

export function getToken() {
  return uni.getStorageSync(TOKEN_KEY) || ''
}

export function setToken(token) {
  uni.setStorageSync(TOKEN_KEY, token)
}

export function getExpiresAt() {
  return uni.getStorageSync(EXPIRES_AT_KEY) || 0
}

export function setExpiresAt(ts) {
  uni.setStorageSync(EXPIRES_AT_KEY, ts || 0)
}

export function getUserInfo() {
  const info = uni.getStorageSync(USER_INFO_KEY)
  return info && typeof info === 'object' ? info : {}
}

export function setUserInfo(user) {
  uni.setStorageSync(USER_INFO_KEY, user || {})
}

/** 本地是否已有有效登录态 */
export function isLoggedIn() {
  return !!getToken()
}

/** 清除全部登录态（退出登录 / token 失效） */
export function clearAuth() {
  uni.removeStorageSync(TOKEN_KEY)
  uni.removeStorageSync(USER_INFO_KEY)
  uni.removeStorageSync(EXPIRES_AT_KEY)
}

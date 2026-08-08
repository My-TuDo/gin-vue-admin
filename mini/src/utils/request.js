/**
 * 请求封装
 * - 基于 uni.request
 * - 自动携带 header: { 'x-token': <token> }
 * - 响应按 GVA 约定处理：{ code: 0, data, msg }
 *   code === 0  -> resolve(data)
 *   其他        -> reject(原始响应) + uni.showToast(msg)
 * - HTTP 401   -> 清除本地 token / 用户信息，跳转登录页
 * - 网络失败    -> reject + uni.showToast('网络异常...')
 */
import { BASE_URL, LOGIN_PAGE } from '@/config'
import { getToken, clearAuth } from '@/utils/auth'

let redirecting = false

function gotoLogin() {
  if (redirecting) return
  redirecting = true
  clearAuth()
  uni.showToast({ title: '登录已过期，请重新登录', icon: 'none', duration: 1500 })
  setTimeout(() => {
    redirecting = false
    uni.reLaunch({ url: LOGIN_PAGE })
  }, 800)
}

/**
 * @param {Object} options
 * @param {string} options.url      接口路径（不含 baseURL，如 '/base/login'）
 * @param {string} [options.method]  GET / POST / PUT / DELETE
 * @param {Object} [options.data]    请求体
 * @param {Object} [options.header]  额外请求头
 * @param {boolean} [options.silent] 为 true 时不弹 toast（调用方自行处理错误）
 */
export function request(options) {
  const {
    url,
    method = 'GET',
    data,
    header = {},
    silent = false
  } = options

  return new Promise((resolve, reject) => {
    const token = getToken()
    uni.request({
      url: BASE_URL + url,
      method,
      data,
      header: {
        'Content-Type': 'application/json',
        'x-token': token,
        ...header
      },
      success: (res) => {
        const { statusCode, data: body } = res

        // token 失效 / 未授权：清登录态并回登录页
        if (statusCode === 401) {
          gotoLogin()
          reject({ code: 401, msg: '登录已过期' })
          return
        }

        // 成功：GVA 约定 code === 0
        if (body && body.code === 0) {
          resolve(body.data)
          return
        }

        const msg = (body && body.msg) || `请求失败(${statusCode})`
        if (!silent) {
          uni.showToast({ title: msg, icon: 'none', duration: 2000 })
        }
        reject(body || { code: -1, msg })
      },
      fail: (err) => {
        // 网络不通 / 后端未启动 / 域名校验失败等
        const msg = '网络异常，请检查后端服务与网络设置'
        if (!silent) {
          uni.showToast({ title: msg, icon: 'none', duration: 2000 })
        }
        reject({ code: -1, msg, raw: err })
      }
    })
  })
}

export function get(url, data, options = {}) {
  return request({ url, method: 'GET', data, ...options })
}

export function post(url, data, options = {}) {
  return request({ url, method: 'POST', data, ...options })
}

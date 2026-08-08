/**
 * 全局配置
 * - BASE_URL：后端接口地址。
 *   微信开发者工具模拟器可直接使用 http://127.0.0.1:8888；
 *   真机预览/调试需改为电脑的局域网 IP（如 http://192.168.x.x:8888），
 *   并确保后端监听 0.0.0.0 且防火墙放行。
 */
export const BASE_URL = 'http://10.228.82.36:8888'

/** 登录页路径（401 / 未登录跳转用） */
export const LOGIN_PAGE = '/pages/login/login'

/** 首页路径 */
export const INDEX_PAGE = '/pages/index/index'

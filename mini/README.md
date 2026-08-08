# 进销存微信小程序（mini/）

gin-vue-admin 进销存系统配套的微信小程序，基于 **uni-app（Vue3 + Vite）**，仅含 M1 里程碑功能：
登录、经营看板（概览 + 库存预警）、我的页。

## 环境要求

- Node.js ≥ 18（推荐 18/20 LTS，vite 5 要求 Node 18+）
- npm ≥ 9（或 pnpm / yarn）

## 安装与构建

```powershell
cd mini
npm install          # 安装依赖（首次较慢，需外网 npm registry）
npm run build:mp-weixin   # 产出 dist/build/mp-weixin
```

开发调试（监听式）可运行 `npm run dev:mp-weixin`，产物在 `dist/dev/mp-weixin`。

## 微信开发者工具运行

1. 打开「微信开发者工具」→ 导入项目；
2. 目录选择 `mini/dist/build/mp-weixin`（或 `dist/dev/mp-weixin`）；
3. AppID 可先选「测试号」，正式发布前在 `src/manifest.json` 的 `mp-weixin.appid` 填真实 AppID 后重新构建；
4. 本地联调必须勾选「详情 → 本地设置 → 不校验合法域名、web-view（业务域名）、TLS 版本以及 HTTPS 证书」；
5. 后端启动 `server/main.go`（http://127.0.0.1:8888）后即可登录。

## 后端接口约定（本里程碑用到）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /base/login | 入参 `{username, password}`，password 为 MD5 小写 hex；返回 `{user, token, expiresAt}` |
| GET | /jxc/dashboard/overview | 返回 `[{label, sales, orders, profit, returnAmt}]`（今日/本周/本月） |
| GET | /jxc/dashboard/stock-alert | 返回 `[{skuId, skuCode, goodsName, safeStock, available}]` |

统一响应包装：`{code, data, msg}`，`code === 0` 表示成功；token 失效返回 HTTP 401。

## 关键文件

| 文件 | 说明 |
| --- | --- |
| `src/config.js` | 后端 baseURL 等全局配置（改 IP 在这里） |
| `src/utils/request.js` | uni.request 封装：x-token 头、code 处理、401 跳登录、网络错误提示 |
| `src/utils/auth.js` | token / 用户信息的本地存储封装 |
| `src/utils/md5.js` | 内联 MD5 实现（密码加密，无 npm 依赖） |
| `src/pages/login/login.vue` | 登录页 |
| `src/pages/index/index.vue` | 经营看板（概览 + 库存预警） |
| `src/pages/mine/mine.vue` | 我的页（用户信息 + 退出登录） |

## 版本说明（重要）

- 本项目 `package.json` 锁定 `@dcloudio/*` 版本为 `3.0.0-4060620250520001`（uni-app 4.06 系列）。
- 若 `npm install` 报 404 / ETARGET（该版本在 registry 不存在或已被淘汰），执行：

```powershell
npm view @dcloudio/uni-app dist-tags --json
npm view @dcloudio/uni-app versions --json | Select-String "3.0.0-4"
```

取一个 `3.0.0-4xxxxxxx` 版本号，**将 `package.json` 中所有 `@dcloudio/*` 的版本号统一替换**后再 install。
⚠️ 所有 `@dcloudio/*` 包的版本必须完全一致，否则构建会报「uni-app 依赖版本不一致」。

- 若私网无法访问 npm registry，可换镜像：`npm config set registry https://registry.npmmirror.com` 后重装。
- 本项目不使用 `js-md5` 等额外 md5 依赖，密码 MD5 为内联实现（`src/utils/md5.js`），与后端 `utils.MD5V` 结果一致（小写 hex）。

## 已知边界

- 小程序端 `127.0.0.1` 仅在「开发者工具模拟器」可用；真机预览需将 `src/config.js` 的 `BASE_URL` 改为电脑局域网 IP，且后端监听 `0.0.0.0`。
- M1 未实现验证码流程：若后端配置 `security.captcha.open` 开启验证码，登录会提示「验证码错误」，届时需二期补充验证码获取/填写。
- tabBar 暂为纯文字（无图标），微信基础库支持。

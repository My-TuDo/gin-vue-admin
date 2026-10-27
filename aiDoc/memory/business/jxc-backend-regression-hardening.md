# 后端回归与 CI 硬化

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done
- 需求类型：测试/持续集成
- 优先级：P1

## 用户原始意图摘要

每次新功能都要有回归；把能补的后端测试缺口、e2e 覆盖和 CI 硬门禁补上。

## 已落地

- 新增 Dashboard 红冲净额、采购退货净额测试（dashboard_extra_test.go）。
- e2e-smoke.py 扩充：
  - 采购退货红冲；
  - 原子开单成功 + 库存不足回滚；
  - 服务端筛选/分页上限；
  - reconcile/last；
  - dashboard summary；
  - 月报 purchaseReturnAmount。
- CI：全量 Backend go job 接入 MySQL 8.4 service，去掉 continue-on-error；
  删除重复的 backend-jxc job；JXC 集成测试与全量测试都是硬门禁。
- docs/28 增加前端测试与回归工作（工程检查、Playwright 建议、页面用例、角色回归、清单）。
- docs/25/27/20/18/22/23 同步采购退货红冲与 CI 状态。

## 验证

- 全量 go test ./... 通过；e2e 全部通过（含红冲/原子开单/服务端筛选/对账 last/工作台/月报）。
- 本机最后一次 e2e：17 个检查点全部 ✓。

## 后续待办

- ~~前端 9003/8881 逐角色点触~~：已完成，9002/9003 基础策略补后也通过；见 docs/31；
- ~~原子开单 UI 失败路径~~：停用 SKU 已验证；
- 前端 lockfile；
- 演示数据重建/正式角色账号，见 docs/32；
- 反结账产品决策（未做）。

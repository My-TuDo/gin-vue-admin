# 前端接入：服务端筛选、聚合、对账 last、原子开单

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done（前端已接入并通过真实后端联调）
- 需求类型：前后端联动 / 列表与首屏性能 / 对账交互
- 优先级：P1
- 文档：docs/30-前端接入实现.md（验收与限制）

## 用户原始意图摘要

> 后端留下了一个需要你完成的前端任务，你阅读一下文档。

对应 docs/28-前端接入需求：单据页服务端筛选/分页、工作台聚合、对账 last、报表净采购、
采购退货展示字段，以及 docs/29 的原子开单。

## 已落地

- documents：五个 Tab 走对应 page 端点，服务端 total 分页；keyword 300ms 防抖；
  状态/仓库/日期/dateField 筛选；状态计数 pageSize=1；顶部 Tab 计数单独请求；
- workspace：只调 dashboard/summary + ledger/moves；删除 5 个 pageSize=100 列表；
- shell：进入只读 reconcile/last；none/stale/ok/bad 状态；60s 静默轮询；
  顶栏点击才手动 /reconcile（库存页内用 jxc:run-reconcile 事件）；
- report：purchaseReturnAmount + 采购金额净额口径；
- sale：checkoutSale 一次请求，删除 create+ship+cancel 兜底；
- docs：新增 30，更新 24/25/26/27/28/29/README。

## 验证

- 单据：total=25，第 2 页 4 行；待出库 0；关键词 1；日期 2026-10-26 为 0；
  主仓 3；采购退货 2 行且 supplierName/warehouseName/itemCount 正常；
- 工作台：库存 10,461.79；净销售 2,761.00；毛利 1,409.79；25 单/退货 14；低库存 11；
- 对账：进入不扫描；手动 ok/count=0；顶栏 title 含范围/时间/耗时/SKU；库存页重复点击可再次触发；
- 报表：采购金额 12,173.00，采购退货 7.00；
- 原子开单：SO-20261027-025，1 件 50.00 / 成本 15.21 / 毛利 34.79；库存 14→13；
- npm run build、eslint 通过；1280/1440 浏览器检查。

## 已知 / 待确认

- 单据状态/Tab 计数是额外 pageSize=1 请求（进入约 10 个，筛选约 10 个；Tab 计数也受 keyword/仓库/日期过滤）；后端补聚合可降；
- partnerId/reversed/sourceType 未做 UI，keyword 覆盖往来单位；
- 对账是 60s 快照，写入后最多等一个轮询周期才变「已过期」；
- todaySalesOrders/月报 salesOrders 仍是全部已出库（含已红冲），金额/成本已净额；
  旧前端净单数（验收前 16 / 验收后 17），接口 25（多 8 张已红冲），待产品确认；
- 9003/8881 未逐角色登录点触，沿用后端 Casbin 矩阵；
- 演示库新增 SO-20261027-025，冒烟仓 SKU23 结存 13/197.79；未清理。

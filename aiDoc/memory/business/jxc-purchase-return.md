# 采购退货单（后端）

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done（后端）；前端入口 pending
- 需求类型：功能补齐 / 采购流程
- 优先级：P1

## 用户原始意图摘要

采购红冲只适合“整单录错”，现实里的“退一部分给供应商”没有独立单据；补一张可按原采购价、部分退货的采购退货单。

## 影响范围

- 后端：model/jxc/purchase_return.go、service/jxc/purchase_return_service.go、api/v1/jxc/purchase_return_api.go、router/jxc/purchase_return.go、service/jxc/ledger.go
- 数据库：purchase_return、purchase_return_item（AutoMigrate）
- 权限：docs/sql/jxc-casbin.sql（179 → 197 条）
- 文档：docs/18、docs/19、docs/20、docs/25
- 测试：server/service/jxc/purchase_return_test.go
- 前端：未改（等前端视觉改版合并后接入口）

## 已确认约束

- 采购退货是出库：账本 qty 为负、amount 为负，source_type=purchase_return。
- 金额按原采购单价精确出库，不走当前移动加权平均。
- 只接受 purchaseId + items[{skuId,qty}]；单价/金额/仓库/供应商从原采购单带出。
- 建单与出库都校验可退上限；两张草稿不能叠加超退。
- 出库时校验库存与有效预留；按原价冲减后金额不能为负。
- 原采购单已整单红冲时不能退货。

## 当前进展

- 新增两张表、服务、API、路由、AutoMigrate、Casbin 策略。
- 5 条集成测试覆盖原价、超退、库存不足、已红冲、取消；全部通过。
- 账本流水补 purchase_return 来源单号（PR-YYYYMMDD-NNN）。
- docs/18、19、20 已同步；新增 docs/25。

## 后续待办

- 前端入口：api/jxc 封装、采购单行“退货”入口或单据页 tab、returnable 回显、草稿出库/取消。
- 采购退货单本身不能红冲；退错时目前只能反向补单/盘盈，后续可考虑。
- 同 SKU 多行采购按加权原价退货，未做到批次级精确；上 FIFO 后应改批次层。
- aiDoc/memory/business/demand-index.md 当时被前端会话改动占用，本轮未改索引；合并时补登记。

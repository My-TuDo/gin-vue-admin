# 销售开单+出库原子接口

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done（后端）；前端待切换
- 需求类型：并发/一致性 / 接口补齐
- 优先级：P1

## 用户原始意图摘要

销售开单与出库原来是两次 HTTP；前端只能靠“出库失败再取消草稿”兜底，两次请求之间断网仍会留下草稿和有效预留。补一个单事务接口。

## 影响范围

- 后端：service/jxc/sale_service.go、api/v1/jxc/document.go、router/jxc/document.go
- 权限：docs/sql/jxc-casbin.sql（206 → 210 条）
- 文档：docs/20、docs/27、docs/28、docs/29
- 测试：server/service/jxc/sale_checkout_test.go
- 前端：未改；切换见 docs/28 第九节

## 已确认约束

- 请求体与 POST /jxc/sale 相同；单价仍取 SKU 售价。
- 一个事务完成建单、预留、出库、成本回填。
- 任一步失败全部回滚，不留销售单/预留/账本行。
- 原有 POST /jxc/sale 与 /sale/:id/ship 保留兼容。
- 888/9001/9002/9003 可用；8881 只读不可用。

## 当前进展

- 抽出 createTx / shipTx，Create/Ship 保留，CreateAndShip 复用。
- 新增 POST /jxc/sale/checkout。
- 3 条测试：成功、库存不足回滚、最后一件 4 并发只成交 1 次；全部通过。
- Casbin 210 条已同步。

## 后续待办

- 前端开单页切到 checkoutSale，删除两次请求与自动取消兜底。
- 旧接口保留给补录/先建单后出库场景。

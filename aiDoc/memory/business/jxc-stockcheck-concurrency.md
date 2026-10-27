# 盘点进行中唯一（并发安全）

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done
- 需求类型：并发缺陷修复
- 优先级：P1

## 用户原始意图摘要

同一仓库同时只能有一张进行中的盘点单；旧实现是 check-then-insert，并发下会建出两张。

## 影响范围

- 后端：service/jxc/stockcheck_service.go
- 测试：service/jxc/exact_cost_test.go
- 文档：docs/20、docs/23

## 已确认约束

- 不新增只为约束存在的表/列；采用仓库行锁串行化建单。
- 锁顺序与 Ledger.Post 一致（仓库 → SKU），不引入新的死锁路径。
- 已有进行中的盘点单时返回可读错误，前端可直接展示。

## 当前进展

- Create 在事务内先 SELECT ... FOR UPDATE 锁仓库行，再 count + insert。
- 新增 TestStockCheck_ConcurrentCreateOnlyOne：4 并发只成功 1 张。
- 针对性测试与全量 jxc 测试通过。

## 后续待办

- 盘点明细全量快照在 SKU 很多时仍有性能压力，后续可考虑按分类/分页建单。

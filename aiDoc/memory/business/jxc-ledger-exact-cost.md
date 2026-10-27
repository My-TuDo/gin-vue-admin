# 红冲/退货分位残差修复

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done
- 需求类型：账务正确性 / 缺陷修复
- 优先级：P1

## 用户原始意图摘要

接手项目后继续开发，优先修掉交接审计里“上线前必须处理”的账务残差：红冲按单价反乘会差 1 分，退货分三次退完也会少 1 分。

## 影响范围

- 后端：internal/ledger/cost.go、service/jxc/ledger.go、return_service.go、model/jxc/reversal.go
- 数据库：sale_return_item.cost_amount（AutoMigrate 加列）
- 文档：docs/20、docs/23
- 测试：internal/ledger/cost_test.go、service/jxc/exact_cost_test.go

## 已确认约束

- 账本金额是真相，单价只是展示/参考；金额字段仍保持 decimal(14,2)、单价 decimal(12,2)。
- 红冲必须按被冲销账本行的精确金额反向，不能按“单价 × 数量”。
- 退货必须按原单成本；最后一次退货吸收原单成本的分位残差。
- 不允许把数量或金额冲成负数；数量归零时金额必须归零。

## 当前进展

- 新增 Position.ApplyAmount(qty, amount)，校验金额/数量方向与结存非负。
- 红冲改按原账本行 amount 精确反向。
- 退货新增 cost_amount，Post 时按“是否最后一次退货”分配精确成本，最后一次吸收残差。
- 新增 4 条回归测试，全量 ledger/service/jxc 测试通过。

## 后续待办

- 老库已有退货行的 cost_amount 为 0，查询时回退到 qty × unit_cost；如需彻底回填，可写一次性迁移脚本。
- 采购退货单仍未实现，见 docs/23 剩余风险。

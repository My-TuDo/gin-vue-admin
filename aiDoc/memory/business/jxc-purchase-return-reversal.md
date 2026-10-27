# 采购退货红冲

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done（后端 + 前端，2026-10-27）
- 需求类型：业务出口补齐 / 账务正确性
- 优先级：P1

## 用户原始意图摘要

采购退货单退错了不能只靠盘盈或反向单补救；补一个整单红冲，把货和金额按原退货价加回来。

## 影响范围

- 后端：model/jxc/purchase_return.go、model/jxc/reversal.go、service/jxc/reversal_service.go、service/jxc/purchase_return_service.go、period_service.go、dashboard_service.go
- 数据库：purchase_return.reversed_at（AutoMigrate 加列）
- 文档：docs/20、docs/25、docs/27、docs/28
- 测试：server/service/jxc/purchase_return_reversal_test.go、purchase_return_reversal_aggregate_test.go
- 前端：documents/index.vue 已接列表/详情红冲、专用弹窗、红冲记录来源；见 docs/31

## 已确认约束

- 只有已退货（status=2）且未红冲的采购退货单可以红冲。
- 反向账本按原退货单价精确加回库存；采购退货单写 reversed_at，不改变原状态。
- 红冲后不再占用可退数量，同一批货可以重新退货。
- 月报/工作台的 purchaseReturnAmount、purchaseAmount 必须忽略已红冲退货。
- 权限沿用 /jxc/reversal POST（888/9001 可用，9002/9003/8881 不可用）。

## 当前进展

- ReversalService 支持 sourceType=purchase_return，SourceType 常量与字段已加。
- purchaseReturnedQty、月报、工作台均排除 reversed_at 非空。
- 3 条测试：红冲恢复库存/可退数量、草稿拒绝、红冲后聚合不被冲减；全部通过。
- 单据页红冲记录搜索支持 purchase_return 来源与仓库/供应商筛选。

## 后续待办

- ~~前端采购退货列表/详情加「红冲」按钮~~：已完成，PR-004/RC-018 验收。
- 9002/9003/8881 的红冲拒绝由后端 Casbin 保证；UI 失败提示可再做一轮角色点触。
- 演示数据重置见 docs/32。

# 主数据 DTO 与编辑/停用

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done
- 需求类型：分层治理 / 功能补齐
- 优先级：P1

## 用户原始意图摘要

主数据接口直接绑定 model，存在 mass assignment；商品/规格/仓库/往来单位只能新建，不能改价、改名、停用。

## 影响范围

- 后端：api/v1/jxc/master_api.go、service/jxc/master_service.go、model/jxc/request、model/jxc/response、router/jxc
- 前端：view/jxc/master、view/jxc/sale、view/jxc/purchase、api/jxc/index.js
- 权限：docs/sql/jxc-casbin.sql（158 → 179 条）
- 文档：docs/18、docs/19、docs/20、docs/23

## 已确认约束

- 遵循 Router -> API -> Service -> Model；API 不直接写 DB。
- 入参用 DTO，只接收业务允许修改的字段；ID/status/时间由服务端决定。
- 删除不提供：历史单据与账本行引用主数据，只能停用。
- 状态只允许 0/1；停用规格不能开单；列表 active=1 只返回启用项。
- 新增接口必须同步补 Casbin 策略，否则会被 403。

## 当前进展

- 新增 MasterService、request/master.go、response/master.go。
- 新增仓库/商品/规格/往来单位的编辑与启停接口；商品编辑可改规格售价、参考进价与状态。
- 前端基础资料页补编辑/停用，售货/补货页只加载启用项。
- 新增 4 条主数据测试；后端 build、前端 build/lint 通过。
- 真实环境验证：e2e 14 项通过；Playwright 点触完成商品新建/编辑/停用，停用商品不再出现在开单页。

## 后续待办

- 主数据硬删除、批量导入、打印/导出仍未做。
- 商品列表仍是 limit 500 的简单列表，后续可补分页/搜索。

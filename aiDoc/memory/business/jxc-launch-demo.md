# 上线演示环境与演示数据

## 基本信息

- 提出日期：2026-10-27
- 当前状态：done（本机演示库已就绪）
- 需求类型：上线演示 / 数据与验收
- 优先级：P0

## 用户原始意图摘要

把项目做到足够撑得住上线演示；演示数据和账号要可重建，前端配合项写文档，后端缺口继续补。

## 已落地

- 决策：现阶段不引入 Redis；单实例 + MySQL 行锁足够。
- 销售单数口径：salesOrders/todaySalesOrders 含已红冲；新增 salesOrdersReversed/
  todaySalesOrdersReversed；金额/成本/毛利净额；前端显示“N 单（含 M 张已红冲）”。
- scripts/demo-db-reset.py：备份源库 → 克隆 erp_v3_demo → 清空业务表，保留
  系统账号/角色/菜单/casbin；强制 --confirm-target，禁止覆盖 erp_v3/mysql/sys。
- scripts/seed-demo.py 重写：2 仓/8 商品/17 SKU/2 供应商/2 客户/3 采购/7 销售/
  销售退货/采购退货/采购退货红冲/销售红冲/4 个演示账号。
- docs/33 记录演示库、账号、数据、重建步骤与验收清单。
- 本机 erp_v3_demo 已 seed 并切换后端；原库与 config 已备份到 backups/。
- 演示账号 demo_boss/manager/clerk/viewer 已真实登录验证权限。

## 验证

- seed 输出：今日销售 2687.00 / 毛利 1515.00 / 低库存 8；
  2026-10 净销售 3428.00 / 净采购 18645.00 / 采购退货 300.00 / 期末 17151.00；
  对账 ok=true。
- 4 个演示角色登录、dashboard、采购退货权限验证通过。
- JXC 210 + 基础登录 9 = 219 条策略。

## 后续待办

- 反结账产品决策；
- 多实例对账共享缓存（当前单实例）；
- 单据状态/Tab 聚合计数（P2）；
- 生产部署时用独立 config/环境变量，不直接改仓库 config。

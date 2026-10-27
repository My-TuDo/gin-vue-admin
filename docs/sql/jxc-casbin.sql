-- 进销存权限矩阵（幂等：先删后插）
--
-- 为什么需要这个文件：GVA 的鉴权是「角色 + 路径(keyMatch2) + 方法」逐条授权的。
-- 新模块没有策略时，连 admin 都会被 CasbinHandler 拒绝（返回"权限不足"）。
-- keyMatch2 支持 :id 占位，所以 /jxc/sale/:id 能匹配 /jxc/sale/12。
--
-- 角色约定：
--   888  超级管理员   —— 全量（GVA 自带角色，这里是逐条授权，没有 * 通配）
--   9001 老板         —— 全量（与 888 一致，便于把超管账号留给运维）
--   9002 店长         —— 除「红冲」「月结」外的全部（这两件事影响历史账，必须老板拍板）
--   9003 店员         —— 只做开单/退货/盘点录入 + 查询，不能采购入库、不能红冲、不能结账
--   8881 只读         —— GVA 自带普通用户子角色，这里只授查询
--
-- 基础登录/登出：9001/9002/9003 各 3 条（getUserInfo/getMenu/jsonInBlacklist），
-- 共 9 条；JXC 业务 210 条 + 基础 9 条 = 219 条。
--
-- 使用：mysql -u root -p erp_v3 < docs/sql/jxc-casbin.sql
-- 执行后**必须重启后端**（casbin enforcer 有进程内缓存，不重启会一直报权限不足）。

SET NAMES utf8mb4;

DELETE FROM casbin_rule WHERE ptype = 'p' AND v1 LIKE '/jxc/%';

-- 超级管理员（55 条）
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p','888','/jxc/master/warehouses','GET'),
  ('p','888','/jxc/master/warehouses','POST'),
  ('p','888','/jxc/master/warehouses/:id','PUT'),
  ('p','888','/jxc/master/warehouses/:id/status','POST'),
  ('p','888','/jxc/master/goods','GET'),
  ('p','888','/jxc/master/goods','POST'),
  ('p','888','/jxc/master/goods/:id','PUT'),
  ('p','888','/jxc/master/goods/:id/status','POST'),
  ('p','888','/jxc/master/skus','GET'),
  ('p','888','/jxc/master/skus/:id/status','POST'),
  ('p','888','/jxc/master/partners','GET'),
  ('p','888','/jxc/master/partners','POST'),
  ('p','888','/jxc/master/partners/:id','PUT'),
  ('p','888','/jxc/master/partners/:id/status','POST'),
  ('p','888','/jxc/purchase','POST'),
  ('p','888','/jxc/purchase/page','GET'),
  ('p','888','/jxc/purchase/:id','GET'),
  ('p','888','/jxc/purchase/:id/approve','POST'),
  ('p','888','/jxc/purchase/:id/receive','POST'),
  ('p','888','/jxc/purchase/:id/cancel','POST'),
  ('p','888','/jxc/purchase-return','POST'),
  ('p','888','/jxc/purchase-return/page','GET'),
  ('p','888','/jxc/purchase-return/returnable','GET'),
  ('p','888','/jxc/purchase-return/:id','GET'),
  ('p','888','/jxc/purchase-return/:id/post','POST'),
  ('p','888','/jxc/purchase-return/:id/cancel','POST'),
  ('p','888','/jxc/sale','POST'),
  ('p','888','/jxc/sale/checkout','POST'),
  ('p','888','/jxc/sale/page','GET'),
  ('p','888','/jxc/sale/:id','GET'),
  ('p','888','/jxc/sale/:id/ship','POST'),
  ('p','888','/jxc/sale/:id/cancel','POST'),
  ('p','888','/jxc/ledger/balance','GET'),
  ('p','888','/jxc/ledger/moves','GET'),
  ('p','888','/jxc/ledger/reconcile','GET'),
  ('p','888','/jxc/ledger/reconcile/last','GET'),
  ('p','888','/jxc/dashboard/summary','GET'),
  ('p','888','/jxc/reversal','POST'),
  ('p','888','/jxc/reversal/page','GET'),
  ('p','888','/jxc/return','POST'),
  ('p','888','/jxc/return/page','GET'),
  ('p','888','/jxc/return/:id','GET'),
  ('p','888','/jxc/return/:id/post','POST'),
  ('p','888','/jxc/return/:id/cancel','POST'),
  ('p','888','/jxc/return/returnable','GET'),
  ('p','888','/jxc/check','POST'),
  ('p','888','/jxc/check/page','GET'),
  ('p','888','/jxc/check/:id','GET'),
  ('p','888','/jxc/check/:id/items','POST'),
  ('p','888','/jxc/check/:id/post','POST'),
  ('p','888','/jxc/check/:id/cancel','POST'),
  ('p','888','/jxc/period','GET'),
  ('p','888','/jxc/period/close','POST'),
  ('p','888','/jxc/period/snapshot','GET'),
  ('p','888','/jxc/period/summary','GET');

-- 老板（55 条）
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p','9001','/jxc/master/warehouses','GET'),
  ('p','9001','/jxc/master/warehouses','POST'),
  ('p','9001','/jxc/master/warehouses/:id','PUT'),
  ('p','9001','/jxc/master/warehouses/:id/status','POST'),
  ('p','9001','/jxc/master/goods','GET'),
  ('p','9001','/jxc/master/goods','POST'),
  ('p','9001','/jxc/master/goods/:id','PUT'),
  ('p','9001','/jxc/master/goods/:id/status','POST'),
  ('p','9001','/jxc/master/skus','GET'),
  ('p','9001','/jxc/master/skus/:id/status','POST'),
  ('p','9001','/jxc/master/partners','GET'),
  ('p','9001','/jxc/master/partners','POST'),
  ('p','9001','/jxc/master/partners/:id','PUT'),
  ('p','9001','/jxc/master/partners/:id/status','POST'),
  ('p','9001','/jxc/purchase','POST'),
  ('p','9001','/jxc/purchase/page','GET'),
  ('p','9001','/jxc/purchase/:id','GET'),
  ('p','9001','/jxc/purchase/:id/approve','POST'),
  ('p','9001','/jxc/purchase/:id/receive','POST'),
  ('p','9001','/jxc/purchase/:id/cancel','POST'),
  ('p','9001','/jxc/purchase-return','POST'),
  ('p','9001','/jxc/purchase-return/page','GET'),
  ('p','9001','/jxc/purchase-return/returnable','GET'),
  ('p','9001','/jxc/purchase-return/:id','GET'),
  ('p','9001','/jxc/purchase-return/:id/post','POST'),
  ('p','9001','/jxc/purchase-return/:id/cancel','POST'),
  ('p','9001','/jxc/sale','POST'),
  ('p','9001','/jxc/sale/checkout','POST'),
  ('p','9001','/jxc/sale/page','GET'),
  ('p','9001','/jxc/sale/:id','GET'),
  ('p','9001','/jxc/sale/:id/ship','POST'),
  ('p','9001','/jxc/sale/:id/cancel','POST'),
  ('p','9001','/jxc/ledger/balance','GET'),
  ('p','9001','/jxc/ledger/moves','GET'),
  ('p','9001','/jxc/ledger/reconcile','GET'),
  ('p','9001','/jxc/ledger/reconcile/last','GET'),
  ('p','9001','/jxc/dashboard/summary','GET'),
  ('p','9001','/jxc/reversal','POST'),
  ('p','9001','/jxc/reversal/page','GET'),
  ('p','9001','/jxc/return','POST'),
  ('p','9001','/jxc/return/page','GET'),
  ('p','9001','/jxc/return/:id','GET'),
  ('p','9001','/jxc/return/:id/post','POST'),
  ('p','9001','/jxc/return/:id/cancel','POST'),
  ('p','9001','/jxc/return/returnable','GET'),
  ('p','9001','/jxc/check','POST'),
  ('p','9001','/jxc/check/page','GET'),
  ('p','9001','/jxc/check/:id','GET'),
  ('p','9001','/jxc/check/:id/items','POST'),
  ('p','9001','/jxc/check/:id/post','POST'),
  ('p','9001','/jxc/check/:id/cancel','POST'),
  ('p','9001','/jxc/period','GET'),
  ('p','9001','/jxc/period/close','POST'),
  ('p','9001','/jxc/period/snapshot','GET'),
  ('p','9001','/jxc/period/summary','GET');

-- 店长（53 条）
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p','9002','/jxc/master/warehouses','GET'),
  ('p','9002','/jxc/master/warehouses','POST'),
  ('p','9002','/jxc/master/warehouses/:id','PUT'),
  ('p','9002','/jxc/master/warehouses/:id/status','POST'),
  ('p','9002','/jxc/master/goods','GET'),
  ('p','9002','/jxc/master/goods','POST'),
  ('p','9002','/jxc/master/goods/:id','PUT'),
  ('p','9002','/jxc/master/goods/:id/status','POST'),
  ('p','9002','/jxc/master/skus','GET'),
  ('p','9002','/jxc/master/skus/:id/status','POST'),
  ('p','9002','/jxc/master/partners','GET'),
  ('p','9002','/jxc/master/partners','POST'),
  ('p','9002','/jxc/master/partners/:id','PUT'),
  ('p','9002','/jxc/master/partners/:id/status','POST'),
  ('p','9002','/jxc/purchase','POST'),
  ('p','9002','/jxc/purchase/page','GET'),
  ('p','9002','/jxc/purchase/:id','GET'),
  ('p','9002','/jxc/purchase/:id/approve','POST'),
  ('p','9002','/jxc/purchase/:id/receive','POST'),
  ('p','9002','/jxc/purchase/:id/cancel','POST'),
  ('p','9002','/jxc/purchase-return','POST'),
  ('p','9002','/jxc/purchase-return/page','GET'),
  ('p','9002','/jxc/purchase-return/returnable','GET'),
  ('p','9002','/jxc/purchase-return/:id','GET'),
  ('p','9002','/jxc/purchase-return/:id/post','POST'),
  ('p','9002','/jxc/purchase-return/:id/cancel','POST'),
  ('p','9002','/jxc/sale','POST'),
  ('p','9002','/jxc/sale/checkout','POST'),
  ('p','9002','/jxc/sale/page','GET'),
  ('p','9002','/jxc/sale/:id','GET'),
  ('p','9002','/jxc/sale/:id/ship','POST'),
  ('p','9002','/jxc/sale/:id/cancel','POST'),
  ('p','9002','/jxc/ledger/balance','GET'),
  ('p','9002','/jxc/ledger/moves','GET'),
  ('p','9002','/jxc/ledger/reconcile','GET'),
  ('p','9002','/jxc/ledger/reconcile/last','GET'),
  ('p','9002','/jxc/dashboard/summary','GET'),
  ('p','9002','/jxc/reversal/page','GET'),
  ('p','9002','/jxc/return','POST'),
  ('p','9002','/jxc/return/page','GET'),
  ('p','9002','/jxc/return/:id','GET'),
  ('p','9002','/jxc/return/:id/post','POST'),
  ('p','9002','/jxc/return/:id/cancel','POST'),
  ('p','9002','/jxc/return/returnable','GET'),
  ('p','9002','/jxc/check','POST'),
  ('p','9002','/jxc/check/page','GET'),
  ('p','9002','/jxc/check/:id','GET'),
  ('p','9002','/jxc/check/:id/items','POST'),
  ('p','9002','/jxc/check/:id/post','POST'),
  ('p','9002','/jxc/check/:id/cancel','POST'),
  ('p','9002','/jxc/period','GET'),
  ('p','9002','/jxc/period/snapshot','GET'),
  ('p','9002','/jxc/period/summary','GET');

-- 店员（25 条）
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p','9003','/jxc/master/warehouses','GET'),
  ('p','9003','/jxc/master/goods','GET'),
  ('p','9003','/jxc/master/skus','GET'),
  ('p','9003','/jxc/master/partners','GET'),
  ('p','9003','/jxc/purchase/page','GET'),
  ('p','9003','/jxc/purchase/:id','GET'),
  ('p','9003','/jxc/sale','POST'),
  ('p','9003','/jxc/sale/checkout','POST'),
  ('p','9003','/jxc/sale/page','GET'),
  ('p','9003','/jxc/sale/:id','GET'),
  ('p','9003','/jxc/sale/:id/ship','POST'),
  ('p','9003','/jxc/ledger/balance','GET'),
  ('p','9003','/jxc/ledger/moves','GET'),
  ('p','9003','/jxc/return','POST'),
  ('p','9003','/jxc/return/page','GET'),
  ('p','9003','/jxc/return/:id','GET'),
  ('p','9003','/jxc/return/:id/post','POST'),
  ('p','9003','/jxc/return/returnable','GET'),
  ('p','9003','/jxc/check','POST'),
  ('p','9003','/jxc/check/page','GET'),
  ('p','9003','/jxc/check/:id','GET'),
  ('p','9003','/jxc/check/:id/items','POST'),
  ('p','9003','/jxc/period','GET'),
  ('p','9003','/jxc/period/summary','GET'),
  ('p','9003','/jxc/dashboard/summary','GET');

-- 只读（22 条）
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p','8881','/jxc/master/warehouses','GET'),
  ('p','8881','/jxc/master/goods','GET'),
  ('p','8881','/jxc/master/skus','GET'),
  ('p','8881','/jxc/master/partners','GET'),
  ('p','8881','/jxc/purchase/page','GET'),
  ('p','8881','/jxc/purchase/:id','GET'),
  ('p','8881','/jxc/sale/page','GET'),
  ('p','8881','/jxc/sale/:id','GET'),
  ('p','8881','/jxc/ledger/balance','GET'),
  ('p','8881','/jxc/ledger/moves','GET'),
  ('p','8881','/jxc/ledger/reconcile','GET'),
  ('p','8881','/jxc/ledger/reconcile/last','GET'),
  ('p','8881','/jxc/dashboard/summary','GET'),
  ('p','8881','/jxc/reversal/page','GET'),
  ('p','8881','/jxc/return/page','GET'),
  ('p','8881','/jxc/return/:id','GET'),
  ('p','8881','/jxc/return/returnable','GET'),
  ('p','8881','/jxc/check/page','GET'),
  ('p','8881','/jxc/check/:id','GET'),
  ('p','8881','/jxc/period','GET'),
  ('p','8881','/jxc/period/snapshot','GET'),
  ('p','8881','/jxc/period/summary','GET');

-- 9001/9002/9003 基础登录/登出策略（每角色 3 条，共 9 条；不计入 JXC 210）
-- 没有这三条：前端路由守卫拉不到用户/菜单，登出接口也会 403。
DELETE FROM casbin_rule
WHERE ptype = 'p'
  AND v0 IN ('9001','9002','9003')
  AND v1 IN ('/user/getUserInfo','/menu/getMenu','/jwt/jsonInBlacklist');

INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p','9001','/user/getUserInfo','GET'),
  ('p','9001','/menu/getMenu','POST'),
  ('p','9001','/jwt/jsonInBlacklist','POST'),
  ('p','9002','/user/getUserInfo','GET'),
  ('p','9002','/menu/getMenu','POST'),
  ('p','9002','/jwt/jsonInBlacklist','POST'),
  ('p','9003','/user/getUserInfo','GET'),
  ('p','9003','/menu/getMenu','POST'),
  ('p','9003','/jwt/jsonInBlacklist','POST');

-- 业务角色（角色本身由 GVA 的角色管理维护，这里只保证存在，便于直接授人以角色）
INSERT INTO sys_authorities (authority_id, authority_name, parent_id, created_at, updated_at, deleted_at)
SELECT 9001, '老板', 0, NOW(), NOW(), NULL FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id = 9001);
INSERT INTO sys_authorities (authority_id, authority_name, parent_id, created_at, updated_at, deleted_at)
SELECT 9002, '店长', 0, NOW(), NOW(), NULL FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id = 9002);
INSERT INTO sys_authorities (authority_id, authority_name, parent_id, created_at, updated_at, deleted_at)
SELECT 9003, '店员', 0, NOW(), NOW(), NULL FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM sys_authorities WHERE authority_id = 9003);

SELECT CONCAT('进销存策略条数：', COUNT(*)) AS result FROM casbin_rule WHERE ptype='p' AND v1 LIKE '/jxc/%';
SELECT CONCAT('9001/9002/9003 基础策略条数：', COUNT(*)) AS result FROM casbin_rule
WHERE ptype='p' AND v0 IN ('9001','9002','9003')
  AND v1 IN ('/user/getUserInfo','/menu/getMenu','/jwt/jsonInBlacklist');

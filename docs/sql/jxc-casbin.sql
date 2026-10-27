-- 进销存 casbin 策略（幂等：先删后插）
--
-- 背景：GVA 的鉴权是按「角色 + 路径(keyMatch2) + 方法」逐条授权的。
-- 新模块的接口如果没有策略，即使是 admin 也会被 CasbinHandler 拒绝（返回"权限不足"）。
-- 旧项目就是因为漏了这一步，采购/看板等页面一打开就 403。
--
-- 路径写法说明：keyMatch2 支持 `:id` 占位，所以 /jxc/purchase/:id 能匹配 /jxc/purchase/12。
--
-- 使用：mysql -u root -p gva < docs/sql/jxc-casbin.sql
-- 执行后**必须重启后端**（casbin enforcer 有缓存）。

SET NAMES utf8mb4;

-- 只清理进销存自己的策略，不动系统原有规则
DELETE FROM casbin_rule WHERE ptype = 'p' AND v1 LIKE '/jxc/%';

-- 角色 888 = 超管（当前库里 888 是逐条授权，没有 * 通配）
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p','888','/jxc/master/warehouses','GET'),
  ('p','888','/jxc/master/warehouses','POST'),
  ('p','888','/jxc/master/goods','GET'),
  ('p','888','/jxc/master/goods','POST'),
  ('p','888','/jxc/master/skus','GET'),

  ('p','888','/jxc/purchase','POST'),
  ('p','888','/jxc/purchase/page','GET'),
  ('p','888','/jxc/purchase/:id','GET'),
  ('p','888','/jxc/purchase/:id/approve','POST'),
  ('p','888','/jxc/purchase/:id/receive','POST'),
  ('p','888','/jxc/purchase/:id/cancel','POST'),

  ('p','888','/jxc/sale','POST'),
  ('p','888','/jxc/sale/page','GET'),
  ('p','888','/jxc/sale/:id','GET'),
  ('p','888','/jxc/sale/:id/ship','POST'),
  ('p','888','/jxc/sale/:id/cancel','POST'),

  ('p','888','/jxc/ledger/balance','GET'),
  ('p','888','/jxc/ledger/moves','GET'),
  ('p','888','/jxc/ledger/reconcile','GET');

-- 角色 8881 = 普通用户（GVA 自带的测试角色）：只给只读，避免误操作
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p','8881','/jxc/master/warehouses','GET'),
  ('p','8881','/jxc/master/goods','GET'),
  ('p','8881','/jxc/master/skus','GET'),
  ('p','8881','/jxc/purchase/page','GET'),
  ('p','8881','/jxc/purchase/:id','GET'),
  ('p','8881','/jxc/sale/page','GET'),
  ('p','8881','/jxc/sale/:id','GET'),
  ('p','8881','/jxc/ledger/balance','GET'),
  ('p','8881','/jxc/ledger/moves','GET'),
  ('p','8881','/jxc/ledger/reconcile','GET');

SELECT CONCAT('进销存策略条数：', COUNT(*)) AS result FROM casbin_rule WHERE ptype='p' AND v1 LIKE '/jxc/%';

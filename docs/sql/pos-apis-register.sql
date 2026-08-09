-- ============================================================
-- v2-M3 扫码枪接口：sys_apis 字典注册（幂等可重跑）
-- 背景：pos 接口此前只配了 casbin 权限，未注册 sys_apis 字典，
--       GVA「系统管理 → API 管理」界面看不到，也无法可视化配置。
-- 同步：casbin 权限见 pos-permissions.sql；路由挂载见 initialize/router_biz.go
-- ============================================================

INSERT INTO `sys_apis` (`path`, `description`, `api_group`, `method`, `created_at`, `updated_at`)
SELECT '/jxc/pos/scan', '扫码上架（小程序扫码投递收银台）', 'jxc扫码枪', 'POST', NOW(3), NOW(3) FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `sys_apis` WHERE path='/jxc/pos/scan' AND method='POST' AND deleted_at IS NULL);

INSERT INTO `sys_apis` (`path`, `description`, `api_group`, `method`, `created_at`, `updated_at`)
SELECT '/jxc/pos/scan/confirm', '扫码条目消费确认（PC 自动加购后）', 'jxc扫码枪', 'PUT', NOW(3), NOW(3) FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `sys_apis` WHERE path='/jxc/pos/scan/confirm' AND method='PUT' AND deleted_at IS NULL);

INSERT INTO `sys_apis` (`path`, `description`, `api_group`, `method`, `created_at`, `updated_at`)
SELECT '/jxc/pos/scan/pending', '本会话未消费扫码条目（PC 轮询）', 'jxc扫码枪', 'GET', NOW(3), NOW(3) FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `sys_apis` WHERE path='/jxc/pos/scan/pending' AND method='GET' AND deleted_at IS NULL);

INSERT INTO `sys_apis` (`path`, `description`, `api_group`, `method`, `created_at`, `updated_at`)
SELECT '/jxc/pos/session', '生成收银台码（PC 设置）', 'jxc扫码枪', 'POST', NOW(3), NOW(3) FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `sys_apis` WHERE path='/jxc/pos/session' AND method='POST' AND deleted_at IS NULL);

INSERT INTO `sys_apis` (`path`, `description`, `api_group`, `method`, `created_at`, `updated_at`)
SELECT '/jxc/pos/session/disable', '作废收银台码（重置）', 'jxc扫码枪', 'PUT', NOW(3), NOW(3) FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `sys_apis` WHERE path='/jxc/pos/session/disable' AND method='PUT' AND deleted_at IS NULL);

INSERT INTO `sys_apis` (`path`, `description`, `api_group`, `method`, `created_at`, `updated_at`)
SELECT '/jxc/pos/session/check', '校验收银台码（小程序绑定）', 'jxc扫码枪', 'GET', NOW(3), NOW(3) FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `sys_apis` WHERE path='/jxc/pos/session/check' AND method='GET' AND deleted_at IS NULL);

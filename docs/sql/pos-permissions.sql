-- ============================================================
-- v2-M3 扫码枪接口权限补充（幂等可重跑）
-- 背景：M3 新增 pos 接口后未同步 casbin 权限 → 所有角色调用 403
-- 矩阵：
--   老板 888 / 店长 9001：全部 6 个（含收银台码生成/作废——管理操作）
--   销售 9002：业务 4 个（扫码/轮询/确认/绑定校验），无码管理权限
--   采购 9003 / 仓管 9004：不参与收银，不配
-- ============================================================

-- 老板 888：全部 pos 接口
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '888', '/jxc/pos/scan', 'POST' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='888' AND v1='/jxc/pos/scan' AND v2='POST');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '888', '/jxc/pos/scan/confirm', 'PUT' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='888' AND v1='/jxc/pos/scan/confirm' AND v2='PUT');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '888', '/jxc/pos/scan/pending', 'GET' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='888' AND v1='/jxc/pos/scan/pending' AND v2='GET');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '888', '/jxc/pos/session', 'POST' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='888' AND v1='/jxc/pos/session' AND v2='POST');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '888', '/jxc/pos/session/disable', 'PUT' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='888' AND v1='/jxc/pos/session/disable' AND v2='PUT');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '888', '/jxc/pos/session/check', 'GET' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='888' AND v1='/jxc/pos/session/check' AND v2='GET');

-- 店长 9001：全部 pos 接口
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9001', '/jxc/pos/scan', 'POST' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9001' AND v1='/jxc/pos/scan' AND v2='POST');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9001', '/jxc/pos/scan/confirm', 'PUT' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9001' AND v1='/jxc/pos/scan/confirm' AND v2='PUT');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9001', '/jxc/pos/scan/pending', 'GET' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9001' AND v1='/jxc/pos/scan/pending' AND v2='GET');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9001', '/jxc/pos/session', 'POST' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9001' AND v1='/jxc/pos/session' AND v2='POST');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9001', '/jxc/pos/session/disable', 'PUT' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9001' AND v1='/jxc/pos/session/disable' AND v2='PUT');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9001', '/jxc/pos/session/check', 'GET' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9001' AND v1='/jxc/pos/session/check' AND v2='GET');

-- 销售 9002：业务 4 个（无收银台码管理）
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9002', '/jxc/pos/scan', 'POST' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9002' AND v1='/jxc/pos/scan' AND v2='POST');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9002', '/jxc/pos/scan/confirm', 'PUT' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9002' AND v1='/jxc/pos/scan/confirm' AND v2='PUT');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9002', '/jxc/pos/scan/pending', 'GET' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9002' AND v1='/jxc/pos/scan/pending' AND v2='GET');
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', '9002', '/jxc/pos/session/check', 'GET' FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `casbin_rule` WHERE ptype='p' AND v0='9002' AND v1='/jxc/pos/session/check' AND v2='GET');

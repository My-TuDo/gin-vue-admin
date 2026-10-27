#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""从现有库克隆一个干净演示库（默认 erp_v3 -> erp_v3_demo）。

做三件事：
  1. mysqldump 备份源库；
  2. 建目标库并导入源库结构 + 系统数据（用户/角色/菜单/casbin）；
  3. 清空目标库里的进销存业务表，保留系统表。

不会修改 server/config.yaml，也不会启动后端；脚本结束会打印切换步骤。

用法：
  python scripts/demo-db-reset.py --mysql-password 123456 --confirm-target erp_v3_demo
  python scripts/demo-db-reset.py --source erp_v3 --target erp_v3_demo \
      --mysql-user root --mysql-password 123456 --confirm-target erp_v3_demo

注意：目标库名必须在 --confirm-target 里再写一遍；禁止覆盖 erp_v3/mysql/sys。
"""
import argparse
import datetime
import os
import re
import shutil
import subprocess
import sys

try:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
except Exception:
    pass

# 这些是进销存业务表；系统表（sys_*、casbin_rule）保留，账号/角色/权限直接沿用。
BIZ_TABLES = [
    "warehouse", "goods", "goods_sku", "jxc_partner",
    "stock_move", "stock_balance", "stock_reservation",
    "accounting_period", "stock_balance_snapshot",
    "purchase_order", "purchase_item", "purchase_return", "purchase_return_item",
    "sale_order", "sale_item", "stock_reversal", "sale_return", "sale_return_item",
    "stock_check", "stock_check_item", "doc_sequence",
]


def run(cmd, stdin=None, stdout=None, check=True):
    try:
        return subprocess.run(cmd, stdin=stdin, stdout=stdout, check=check)
    except FileNotFoundError as e:
        raise SystemExit(f"找不到命令：{e.filename}，请确认 mysql/mysqldump 在 PATH 中")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--source", default="erp_v3")
    ap.add_argument("--target", default="erp_v3_demo")
    ap.add_argument("--confirm-target", required=True,
                    help="必须与 --target 完全相同，防止误删")
    ap.add_argument("--mysql-user", default="root")
    ap.add_argument("--mysql-password", default=os.environ.get("MYSQL_PASSWORD", ""))
    ap.add_argument("--backup-dir", default="backups")
    args = ap.parse_args()

    name_re = re.compile(r"^[A-Za-z0-9_]+$")
    for name in (args.source, args.target):
        if not name_re.match(name):
            raise SystemExit(f"库名只能包含字母/数字/下划线：{name}")
    if args.target in ("erp_v3", "mysql", "sys", "information_schema", "performance_schema"):
        raise SystemExit(f"拒绝覆盖受保护库：{args.target}")
    if args.source == args.target:
        raise SystemExit("源库和目标库不能相同")
    if args.confirm_target != args.target:
        raise SystemExit("--confirm-target 与 --target 不一致，拒绝执行")
    if not args.mysql_password:
        raise SystemExit("请通过 --mysql-password 或 MYSQL_PASSWORD 提供 MySQL 密码")

    mysql = shutil.which("mysql")
    dump = shutil.which("mysqldump")
    if not mysql or not dump:
        raise SystemExit("mysql / mysqldump 不在 PATH 中")

    os.makedirs(args.backup_dir, exist_ok=True)
    stamp = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
    backup = os.path.join(args.backup_dir, f"{args.source}-{stamp}.sql")
    auth = [f"-u{args.mysql_user}", f"-p{args.mysql_password}"]

    print(f"1/4 备份 {args.source} -> {backup}")
    with open(backup, "wb") as f:
        run([dump, "--single-transaction", "--routines", "--quick",
             "--set-gtid-purged=OFF", "--default-character-set=utf8mb4",
             *auth, args.source], stdout=f)

    print(f"2/4 重建目标库 {args.target}")
    run([mysql, *auth, "-e",
         f"DROP DATABASE IF EXISTS `{args.target}`; "
         f"CREATE DATABASE `{args.target}` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"])

    print("3/4 导入结构与系统数据")
    with open(backup, "rb") as f:
        run([mysql, *auth, args.target], stdin=f)

    print("4/4 清空业务表（保留 sys_* / casbin_rule）")
    sql = ["SET FOREIGN_KEY_CHECKS=0;"]
    for table in BIZ_TABLES:
        sql.append(f"TRUNCATE TABLE `{table}`;")
    sql.append("SET FOREIGN_KEY_CHECKS=1;")
    run([mysql, *auth, args.target, "-e", " ".join(sql)])

    print("")
    print(f"完成 ✅  {args.target} 是干净演示库，系统账号/角色/权限已保留")
    print("下一步：")
    print(f"  1. 修改 server/config.yaml 的 mysql.db-name 为 {args.target}（先备份配置）")
    print("  2. 重启后端")
    print("  3. python scripts/seed-demo.py --password <admin密码> --demo-password <演示密码>")
    print("  4. python scripts/e2e-smoke.py --password <admin密码>  # 可选")
    print(f"  备份文件：{backup}（backups/ 已 gitignore）")
    return 0


if __name__ == "__main__":
    sys.exit(main())

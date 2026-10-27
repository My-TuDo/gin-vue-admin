#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""生成一套可对外演示的进销存数据。

为什么用脚本而不是手工点：演示数据要能反复重建，而且必须保证 UTF-8。
本脚本假设目标库是**干净库**（没有业务单据）；如果已有销售单会直接退出，
避免在旧测试数据上继续叠加。

用法：
    python scripts/seed-demo.py --password Admin@123456
    python scripts/seed-demo.py --password Admin@123456 --demo-password 'Demo@123456'

前置：
    1. 后端已初始化（/init/initdb）；
    2. 已执行 docs/sql/jxc-casbin.sql 并重启；
    3. 目标库没有业务单据（或显式传 --allow-existing 做开发叠加）。

演示内容：
    2 个仓库、8 个商品/18 个 SKU、2 供应商、2 客户；
    3 张采购单（含供应商）、7 张销售单（含今日/历史/原子开单）；
    1 张销售退货、1 张采购退货、1 张采购退货红冲、1 张销售红冲；
    有货/低库存/零库存都有；工作台、月报、对账可直接演示。
"""
import argparse
import datetime
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request

try:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
except Exception:
    pass

TOKEN = None


def call(base, path, method="GET", body=None):
    url = base.rstrip("/") + path
    data = None
    headers = {"Content-Type": "application/json; charset=utf-8"}
    if TOKEN:
        headers["x-token"] = TOKEN
    if body is not None:
        data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            payload = resp.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", "replace")
        try:
            return json.loads(raw)
        except Exception:
            raise SystemExit(f"HTTP {e.code} {path}: {raw[:300]}")
    try:
        return json.loads(payload)
    except json.JSONDecodeError:
        raise SystemExit(f"非 JSON 响应（{path}）：{payload[:300]}")


def must(res, what):
    if res.get("code") != 0:
        raise SystemExit(f"{what} 失败：{res.get('msg')}")
    return res.get("data")


def login(base, user, password):
    global TOKEN
    data = must(call(base, "/base/login", "POST", {"username": user, "password": password}), "登录")
    TOKEN = data["token"]
    print(f"登录成功：{user}")


def at(days_ago=0, hour=10, minute=0):
    now = datetime.datetime.now()
    d = (now - datetime.timedelta(days=days_ago)).replace(
        hour=hour, minute=minute, second=0, microsecond=0)
    return d.strftime("%Y-%m-%dT%H:%M:%S+08:00")


def occurred(path, dt):
    return path + ("?occurredAt=" + urllib.parse.quote(dt) if dt else "")


def ensure_warehouse(base, code, name):
    for w in must(call(base, "/jxc/master/warehouses"), "查仓库") or []:
        if w["code"] == code:
            return w["ID"]
    return must(call(base, "/jxc/master/warehouses", "POST",
                     {"code": code, "name": name}), "建仓库")["ID"]


def ensure_partner(base, kind, code, name, contact="", phone="", address=""):
    for p in must(call(base, f"/jxc/master/partners?kind={kind}"), "查往来单位") or []:
        if p["code"] == code:
            return p["ID"]
    return must(call(base, "/jxc/master/partners", "POST", {
        "kind": kind, "code": code, "name": name,
        "contact": contact, "phone": phone, "address": address,
    }), f"建{kind}")["ID"]


def ensure_goods(base, code, name, skus):
    for g in must(call(base, "/jxc/master/goods"), "查商品") or []:
        if g["code"] == code:
            return {s["skuCode"]: s["ID"] for s in g.get("skus", [])}
    created = must(call(base, "/jxc/master/goods", "POST",
                        {"code": code, "name": name, "skus": skus}), f"建商品 {code}")
    return {s["skuCode"]: s["ID"] for s in created["skus"]}


def receive_purchase(base, wh, supplier_id, items, dt, remark):
    body = {"warehouseId": wh, "items": [{"skuId": sid, "qty": q, "unitCost": c}
                                         for sid, q, c in items], "remark": remark}
    if supplier_id:
        body["supplierId"] = supplier_id
    po = must(call(base, "/jxc/purchase", "POST", body), f"建采购单 {remark}")
    must(call(base, f"/jxc/purchase/{po['ID']}/approve", "POST"), "审核采购单")
    must(call(base, occurred(f"/jxc/purchase/{po['ID']}/receive", dt), "POST"), "采购入库")
    return po


def checkout_sale(base, wh, customer_id, items, dt, remark):
    body = {"warehouseId": wh, "items": [{"skuId": sid, "qty": q} for sid, q in items],
            "remark": remark}
    if customer_id:
        body["customerId"] = customer_id
    return must(call(base, occurred("/jxc/sale/checkout", dt), "POST", body), f"原子开单 {remark}")


def create_demo_user(base, username, nickname, authority, password):
    body = {
        "userName": username, "nickName": nickname, "passWord": password,
        "authorityId": authority, "authorityIds": [authority],
        "enable": 1, "headerImg": "",
    }
    res = call(base, "/user/admin_register", "POST", body)
    if res.get("code") == 0:
        print(f"  演示账号 {username}（角色 {authority}）已创建")
    else:
        msg = res.get("msg") or ""
        if "已存在" in msg or "重复" in msg:
            print(f"  演示账号 {username} 已存在，跳过")
        else:
            print(f"  ⚠ 演示账号 {username} 创建失败：{msg}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1:8888")
    ap.add_argument("--user", default="admin")
    ap.add_argument("--password", default="Admin@123456")
    ap.add_argument("--demo-password", default=os.environ.get("JXC_DEMO_PASSWORD", ""),
                    help="可选：创建 demo_boss/manager/clerk/viewer 四个账号的密码；不传则跳过账号")
    ap.add_argument("--allow-existing", action="store_true",
                    help="允许在已有业务单据的库上追加（会重复演示数据，默认拒绝）")
    args = ap.parse_args()
    base = args.base

    login(base, args.user, args.password)

    existing = must(call(base, "/jxc/sale/page?page=1&pageSize=1"), "检查演示库")
    if existing.get("total", 0) > 0 and not args.allow_existing:
        raise SystemExit("目标库已有销售单，请在干净演示库上运行；"
                         "确需追加请加 --allow-existing（会重复数据）")

    main_wh = ensure_warehouse(base, "WH-MAIN", "主仓")
    satellite_wh = ensure_warehouse(base, "WH-SATELLITE", "卫星仓")
    sup1 = ensure_partner(base, "supplier", "SUP-001", "杭州云裳服饰",
                          "周经理", "13800000001", "杭州市余杭区")
    sup2 = ensure_partner(base, "supplier", "SUP-002", "广州丹宁工坊",
                          "李经理", "13800000002", "广州市海珠区")
    cus1 = ensure_partner(base, "customer", "CUS-001", "林女士",
                          "林女士", "13900000001", "杭州市西湖区")
    cus2 = ensure_partner(base, "customer", "CUS-002", "陈先生",
                          "陈先生", "13900000002", "杭州市滨江区")
    print(f"基础资料：主仓={main_wh} 卫星仓={satellite_wh} 供应商={sup1}/{sup2} 客户={cus1}/{cus2}")

    goods = [
        ("WY-001", "连帽卫衣", [
            {"color": "黑", "size": "M", "skuCode": "WY001-BK-M", "salePrice": 199, "refCost": 88},
            {"color": "白", "size": "L", "skuCode": "WY001-WH-L", "salePrice": 209, "refCost": 92},
        ]),
        ("KZ-002", "直筒牛仔裤", [
            {"color": "蓝", "size": "30", "skuCode": "KZ002-BL-30", "salePrice": 259, "refCost": 120},
            {"color": "蓝", "size": "32", "skuCode": "KZ002-BL-32", "salePrice": 259, "refCost": 120},
        ]),
        ("TX-003", "纯棉T恤", [
            {"color": "白", "size": "M", "skuCode": "TX003-WH-M", "salePrice": 89, "refCost": 35},
            {"color": "黑", "size": "L", "skuCode": "TX003-BK-L", "salePrice": 89, "refCost": 35},
            {"color": "灰", "size": "M", "skuCode": "TX003-GY-M", "salePrice": 89, "refCost": 35},
        ]),
        ("WD-004", "轻薄外套", [
            {"color": "卡其", "size": "M", "skuCode": "WD004-KK-M", "salePrice": 399, "refCost": 160},
            {"color": "黑", "size": "L", "skuCode": "WD004-BK-L", "salePrice": 399, "refCost": 165},
        ]),
        ("LQ-005", "法式连衣裙", [
            {"color": "米白", "size": "S", "skuCode": "LQ005-MB-S", "salePrice": 459, "refCost": 180},
            {"color": "黑", "size": "M", "skuCode": "LQ005-BK-M", "salePrice": 459, "refCost": 185},
            {"color": "黑", "size": "L", "skuCode": "LQ005-BK-L", "salePrice": 459, "refCost": 185},
        ]),
        ("YD-006", "轻便运动鞋", [
            {"color": "白", "size": "42", "skuCode": "YD006-WH-42", "salePrice": 299, "refCost": 150},
            {"color": "黑", "size": "43", "skuCode": "YD006-BK-43", "salePrice": 299, "refCost": 150},
        ]),
        ("MZ-007", "纯棉棒球帽", [
            {"color": "黑", "size": "均码", "skuCode": "MZ007-BK-F", "salePrice": 69, "refCost": 30},
        ]),
        ("WZ-008", "透气棉袜", [
            {"color": "白", "size": "均码", "skuCode": "WZ008-WH-F", "salePrice": 19, "refCost": 8},
        ]),
        ("XB-009", "限量手袋", [
            {"color": "棕", "size": "均码", "skuCode": "XB009-BR-F", "salePrice": 699, "refCost": 320},
        ]),
    ]
    sku = {}
    for code, name, skus in goods:
        sku.update(ensure_goods(base, code, name, skus))
    print(f"商品规格：{len(sku)} 个 SKU")

    # 采购：3 张单，含供应商、跨 10/7/3 天
    receive_purchase(base, main_wh, sup1, [
        (sku["WY001-BK-M"], 20, 88), (sku["WY001-WH-L"], 15, 92),
        (sku["TX003-WH-M"], 30, 35), (sku["TX003-BK-L"], 20, 35),
        (sku["WD004-KK-M"], 5, 160),
    ], at(10, 10), "秋季首批进货")
    receive_purchase(base, main_wh, sup2, [
        (sku["KZ002-BL-30"], 20, 120), (sku["KZ002-BL-32"], 20, 120),
        (sku["LQ005-MB-S"], 12, 180), (sku["LQ005-BK-M"], 3, 185),
        (sku["YD006-WH-42"], 6, 150), (sku["YD006-BK-43"], 6, 150),
        (sku["MZ007-BK-F"], 3, 30), (sku["WZ008-WH-F"], 50, 8),
    ], at(7, 10), "丹宁与配件补货")
    receive_purchase(base, main_wh, sup1, [
        (sku["WD004-BK-L"], 10, 165), (sku["LQ005-BK-L"], 8, 185),
        (sku["XB009-BR-F"], 1, 320),
    ], at(3, 10), "外套与限量款")
    print("采购：3 张已入库")

    # 销售：今日 4 单 + 历史 2 单 + 红冲样例 1 单
    s1 = checkout_sale(base, main_wh, cus1, [
        (sku["WY001-BK-M"], 2), (sku["KZ002-BL-30"], 1), (sku["TX003-WH-M"], 2),
    ], at(0, 10, 30), "林女士到店")
    s2 = checkout_sale(base, main_wh, cus2, [
        (sku["LQ005-MB-S"], 1),
        (sku["YD006-WH-42"], 1),
    ], at(0, 11, 0), "陈先生到店")
    s3 = checkout_sale(base, main_wh, cus1, [
        (sku["WD004-KK-M"], 1), (sku["MZ007-BK-F"], 2), (sku["WZ008-WH-F"], 3),
    ], at(0, 14, 0), "周末搭配")
    s4 = checkout_sale(base, main_wh, None, [
        (sku["WY001-WH-L"], 1), (sku["KZ002-BL-32"], 1),
    ], at(1, 15, 0), "昨日散客")
    s5 = checkout_sale(base, main_wh, cus2, [
        (sku["TX003-BK-L"], 2), (sku["WZ008-WH-F"], 5),
    ], at(3, 16, 0), "老客补货")
    s6 = checkout_sale(base, main_wh, cus1, [(sku["TX003-BK-L"], 1)],
                       at(0, 16, 0), "录错待红冲")
    must(call(base, "/jxc/reversal", "POST", {
        "sourceType": "sale", "sourceId": s6["ID"], "reason": "演示：录错整单红冲"
    }), "销售红冲样例")
    s7 = checkout_sale(base, main_wh, cus2, [(sku["XB009-BR-F"], 1)],
                       at(0, 16, 30), "限量手袋售出")
    print(f"销售：{s1['orderNo']} / {s2['orderNo']} / {s3['orderNo']} / {s4['orderNo']} / {s5['orderNo']} / {s7['orderNo']}")

    # 销售退货：s1 的卫衣退 1 件
    ret = must(call(base, "/jxc/return", "POST", {
        "saleId": s1["ID"], "items": [{"skuId": sku["WY001-BK-M"], "qty": 1}],
        "reason": "演示：客户尺码不合适",
    }), "建销售退货")
    must(call(base, occurred(f"/jxc/return/{ret['ID']}/post", at(0, 15, 0)), "POST"), "销售退货入库")
    print(f"销售退货：{ret['returnNo']}")

    # 采购退货：PO2 的运动鞋退 2 件；PO2 的帽子退 1 件再红冲
    # 找到 PO2/PO3 的 ID：按 remark 查采购单
    def find_purchase(remark):
        rows = must(call(base, "/jxc/purchase/page?page=1&pageSize=50"), "查采购单")["list"]
        row = next((x for x in rows if x.get("remark") == remark), None)
        if not row:
            raise SystemExit(f"找不到采购单：{remark}")
        return row["ID"]

    po2 = find_purchase("丹宁与配件补货")
    po3 = find_purchase("外套与限量款")
    pr1 = must(call(base, "/jxc/purchase-return", "POST", {
        "purchaseId": po2,
        "items": [{"skuId": sku["YD006-WH-42"], "qty": 2}],
        "reason": "演示：供应商尺码发错",
    }), "建采购退货")
    must(call(base, occurred(f"/jxc/purchase-return/{pr1['ID']}/post", at(0, 12, 0)), "POST"), "采购退货出库")
    pr2 = must(call(base, "/jxc/purchase-return", "POST", {
        "purchaseId": po2,
        "items": [{"skuId": sku["MZ007-BK-F"], "qty": 1}],
        "reason": "演示：退货后供应商取消",
    }), "建第二张采购退货")
    must(call(base, occurred(f"/jxc/purchase-return/{pr2['ID']}/post", at(0, 13, 0)), "POST"), "第二张采购退货出库")
    must(call(base, "/jxc/reversal", "POST", {
        "sourceType": "purchase_return", "sourceId": pr2["ID"], "reason": "演示：供应商取消退货"
    }), "采购退货红冲")
    print(f"采购退货：{pr1['returnNo']}（保留）/ {pr2['returnNo']}（已红冲）")

    # 可选演示账号
    if args.demo_password:
        if len(args.demo_password) < 8:
            print("⚠ --demo-password 少于 8 位，跳过演示账号")
        else:
            print("演示账号：")
            create_demo_user(base, "demo_boss", "演示老板", 9001, args.demo_password)
            create_demo_user(base, "demo_manager", "演示店长", 9002, args.demo_password)
            create_demo_user(base, "demo_clerk", "演示店员", 9003, args.demo_password)
            create_demo_user(base, "demo_viewer", "演示只读", 8881, args.demo_password)
    else:
        print("未传 --demo-password，跳过演示账号创建")

    # 对账与摘要
    rec = must(call(base, "/jxc/ledger/reconcile"), "对账")
    if not rec["ok"]:
        raise SystemExit(f"演示数据对账失败：{rec}")
    dash = must(call(base, "/jxc/dashboard/summary"), "工作台")
    ym = datetime.date.today().strftime("%Y-%m")
    s = must(call(base, f"/jxc/period/summary?month={ym}"), "月报")
    print("")
    print("演示数据完成 ✅")
    print(f"  仓库：主仓 {main_wh} / 卫星仓 {satellite_wh}")
    print(f"  SKU：{len(sku)}，今日销售 {dash['todaySalesAmount']:.2f}，"
          f"今日毛利 {dash['todayGross']:.2f}，低库存 {dash['lowStock']['count']}")
    print(f"  {ym}：净销售 {s['salesAmount']:.2f} / 净采购 {s['purchaseAmount']:.2f} / "
          f"采购退货 {s['purchaseReturnAmount']:.2f} / 期末库存 {s['stockValue']:.2f}")
    print(f"  对账：ok={rec['ok']}，检查 {rec['checkedSkus']} 个 SKU")
    return 0


if __name__ == "__main__":
    sys.exit(main())

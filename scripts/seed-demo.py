#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""生成一套可演示的进销存数据。

为什么用脚本而不是手工点：演示数据要能反复重建，而且必须保证 UTF-8——
在 Windows 终端里用 curl 直接传中文会被本地码页破坏（踩过）。

用法：
    python scripts/seed-demo.py --base http://127.0.0.1:8888 --user admin --password Admin@123456

前置：后端已启动且已初始化（/init/initdb），并且已灌入 casbin 策略（docs/sql/jxc-casbin.sql）。
"""

import argparse
import json
import sys
import urllib.request

# Windows 控制台默认 GBK，打印中文/emoji 会直接抛 UnicodeEncodeError。
# 这里强制 UTF-8 并用 replace 兜底，保证脚本在任意终端都能跑完。
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
    with urllib.request.urlopen(req, timeout=20) as resp:
        payload = resp.read().decode("utf-8")
    try:
        return json.loads(payload)
    except json.JSONDecodeError:
        raise SystemExit(f"非 JSON 响应（{path}）：{payload[:200]}")


def must(res, what):
    if res.get("code") != 0:
        raise SystemExit(f"{what} 失败：{res.get('msg')}")
    return res.get("data")


def login(base, user, password):
    global TOKEN
    res = call(base, "/base/login", "POST", {"username": user, "password": password})
    data = must(res, "登录")
    TOKEN = data["token"]


def ensure_warehouse(base, code, name):
    for w in must(call(base, "/jxc/master/warehouses"), "查仓库") or []:
        if w["code"] == code:
            return w["ID"]
    return must(call(base, "/jxc/master/warehouses", "POST", {"code": code, "name": name}), "建仓库")["ID"]


def ensure_goods(base, code, name, skus):
    for g in must(call(base, "/jxc/master/goods"), "查商品") or []:
        if g["code"] == code:
            return g["skus"]
    created = must(call(base, "/jxc/master/goods", "POST", {"code": code, "name": name, "skus": skus}), "建商品")
    return created["skus"]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1:8888")
    ap.add_argument("--user", default="admin")
    ap.add_argument("--password", default="Admin@123456")
    args = ap.parse_args()
    base = args.base

    login(base, args.user, args.password)

    wh = ensure_warehouse(base, "WH-MAIN", "主仓")

    goods = [
        ("WY-001", "连帽卫衣", [
            {"color": "黑", "size": "M", "skuCode": "WY001-BK-M", "salePrice": 199},
            {"color": "白", "size": "L", "skuCode": "WY001-WH-L", "salePrice": 209},
        ]),
        ("KZ-002", "直筒牛仔裤", [
            {"color": "蓝", "size": "30", "skuCode": "KZ002-BL-30", "salePrice": 259},
            {"color": "蓝", "size": "32", "skuCode": "KZ002-BL-32", "salePrice": 259},
        ]),
    ]
    sku_map = {}
    for code, name, skus in goods:
        for s in ensure_goods(base, code, name, skus):
            sku_map[s["skuCode"]] = s["ID"]

    # 进货：卫衣 10 件 @88，牛仔裤 6 件 @120
    for sku_code, qty, cost in [("WY001-BK-M", 10, 88), ("WY001-WH-L", 5, 92),
                                ("KZ002-BL-30", 6, 120), ("KZ002-BL-32", 4, 120)]:
        po = must(call(base, "/jxc/purchase", "POST", {
            "warehouseId": wh,
            "items": [{"skuId": sku_map[sku_code], "qty": qty, "unitCost": cost}],
        }), f"建采购单 {sku_code}")
        must(call(base, f"/jxc/purchase/{po['ID']}/approve", "POST"), "审核")
        must(call(base, f"/jxc/purchase/{po['ID']}/receive", "POST"), "入库")
        print(f"进货 {sku_code} ×{qty} @{cost}")

    # 销售：卖 3 件卫衣 + 1 件牛仔裤
    so = must(call(base, "/jxc/sale", "POST", {
        "warehouseId": wh,
        "items": [
            {"skuId": sku_map["WY001-BK-M"], "qty": 3, "price": 199},
            {"skuId": sku_map["KZ002-BL-30"], "qty": 1, "price": 259},
        ],
    }), "开单")
    must(call(base, f"/jxc/sale/{so['ID']}/ship", "POST"), "出库")
    detail = must(call(base, f"/jxc/sale/{so['ID']}"), "查单")
    print(f"销售 {detail['orderNo']}：销售额 {detail['totalAmount']}，成本 {detail['costAmount']}，"
          f"毛利 {round(detail['totalAmount'] - detail['costAmount'], 2)}")

    bal = must(call(base, f"/jxc/ledger/balance?warehouseId={wh}"), "查台账")
    for row in bal:
        print(f"  台账 {row['goodsName']} {row['color']}/{row['size']}：结存 {row['qty']}，"
              f"可售 {row['available']}，均价 {row['avgCost']}")

    rec = must(call(base, "/jxc/ledger/reconcile"), "对账")
    print("对账：" + ("账实相符 ✅" if rec["ok"] else f"异常 {rec['count']} 项 ⚠️"))
    return 0 if rec["ok"] else 1


if __name__ == "__main__":
    sys.exit(main())

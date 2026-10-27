#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""端到端冒烟：把红冲、退货、盘点、月结跑一遍，并断言余额回到预期。

用法：python scripts/e2e-smoke.py --base http://127.0.0.1:8888 --password Admin@123456
"""
import argparse, json, sys, time, urllib.request

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
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read().decode("utf-8"))

def must(res, what):
    if res.get("code") != 0:
        raise SystemExit(f"✗ {what}: {res.get('msg')}")
    return res.get("data")

def ok(msg):
    print(f"✓ {msg}")

def main():
    global TOKEN
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1:8888")
    ap.add_argument("--user", default="admin")
    ap.add_argument("--password", default="Admin@123456")
    a = ap.parse_args()
    base = a.base

    TOKEN = must(call(base, "/base/login", "POST", {"username": a.user, "password": a.password}), "登录")["token"]
    ok("登录成功")

    # 每次用一套全新的仓库+商品：冒烟要断言绝对数量，复用旧数据会被上一轮污染
    tag = str(int(time.time()))[-6:]
    wh = must(call(base, "/jxc/master/warehouses", "POST",
                   {"code": f"WH-SMOKE-{tag}", "name": f"冒烟仓{tag}"}), "建仓")["ID"]
    g = must(call(base, "/jxc/master/goods", "POST",
                  {"code": f"SMOKE-{tag}", "name": "冒烟商品",
                   "skus": [{"color": "黑", "size": "M", "skuCode": f"SMOKE-{tag}-BK-M", "salePrice": 100}]}), "建商品")
    sku = g["skus"][0]["ID"]
    ok(f"基础资料：仓库 {wh} / SKU {sku}")

    bal = lambda: must(call(base, f"/jxc/ledger/balance?warehouseId={wh}"), "查台账")
    def qty():
        rows = [r for r in bal() if r["skuId"] == sku]
        return rows[0]["qty"] if rows else 0

    # 采购 10 件 @60
    po = must(call(base, "/jxc/purchase", "POST", {"warehouseId": wh,
        "items": [{"skuId": sku, "qty": 10, "unitCost": 60}]}), "建采购单")
    must(call(base, f"/jxc/purchase/{po['ID']}/approve", "POST"), "审核")
    must(call(base, f"/jxc/purchase/{po['ID']}/receive", "POST"), "入库")
    assert qty() == 10, qty()
    ok("采购入库 10 件，台账正确")

    # 销售 4 件
    so = must(call(base, "/jxc/sale", "POST", {"warehouseId": wh,
        "items": [{"skuId": sku, "qty": 4, "price": 100}]}), "开单")
    must(call(base, f"/jxc/sale/{so['ID']}/ship", "POST"), "出库")
    assert qty() == 6, qty()
    ok("销售出库 4 件，台账 6")

    # 退货 1 件（成本应为原单 60）
    ret = must(call(base, "/jxc/return", "POST", {"saleId": so["ID"],
        "items": [{"skuId": sku, "qty": 1}], "reason": "冒烟退货"}), "建退货单")
    must(call(base, f"/jxc/return/{ret['ID']}/post", "POST"), "退货入库")
    assert qty() == 7, qty()
    assert abs(ret["costAmount"] - 60) < 0.001, ret
    ok("退货 1 件，成本按原单 60 回补，台账 7")

    # 盘点：实盘 5 → 盘亏 2
    chk = must(call(base, "/jxc/check", "POST", {"warehouseId": wh, "remark": "冒烟盘点"}), "建盘点单")
    items = chk["items"]
    assert len(items) == 1
    must(call(base, f"/jxc/check/{chk['ID']}/items", "POST",
              {"items": [{"id": items[0]["ID"], "actualQty": 5}]}), "录实盘")
    must(call(base, f"/jxc/check/{chk['ID']}/post", "POST"), "完成盘点")
    assert qty() == 5, qty()
    ok("盘点盘亏 2 件，台账 5（差异进账本，不是覆盖余额）")

    # 红冲销售单：货回来
    rev = must(call(base, "/jxc/reversal", "POST",
        {"sourceType": "sale", "sourceId": so["ID"], "reason": "冒烟红冲"}), "红冲销售单")
    assert qty() == 9, qty()
    ok(f"红冲 {rev['reversalNo']}：销售单整单冲销，台账回到 9")

    # 重复红冲必须被拒
    r2 = call(base, "/jxc/reversal", "POST", {"sourceType": "sale", "sourceId": so["ID"], "reason": "再来一次"})
    assert r2["code"] != 0 and "已经红冲" in r2["msg"], r2
    ok("重复红冲被拒绝")

    # 对账
    rec = must(call(base, "/jxc/ledger/reconcile"), "对账")
    assert rec["ok"], rec
    ok("对账：账实相符")

    # 月报
    import datetime
    ym = datetime.date.today().strftime("%Y-%m")
    s = must(call(base, f"/jxc/period/summary?month={ym}"), "月报")
    ok(f"月报 {ym}：销售额 {s['salesAmount']:.2f}，成本 {s['salesCost']:.2f}，毛利 {s['gross']:.2f}")
    print("\n全部通过 ✅")

if __name__ == "__main__":
    main()

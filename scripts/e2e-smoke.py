#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""端到端冒烟：用真实接口跑一遍进销存的关键账务路径。

覆盖：
  采购入库 / 销售出库 / 退货 / 超额退货拦截 / 已退货单红冲拦截 /
  销售红冲 / 采购按原价红冲 / 盘点账面变化拦截 / 对账 / 月报期末库存

用法：
  python scripts/e2e-smoke.py --password Admin@123456
  python scripts/e2e-smoke.py --token <已登录的 x-token>

说明：
  * 登录模式需要临时关闭验证码（见 docs/18），或直接用 --token。
  * 每次运行使用独立的仓库/商品/SKU，断言只针对本次数据；但系统没有删除接口，
    演示库会持续累积 smoke 数据，这是当前版本的已知限制。
"""
import argparse
import datetime
import json
import os
import sys
import time
import urllib.error
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
        with urllib.request.urlopen(req, timeout=30) as r:
            return json.loads(r.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        # GVA 的鉴权失败是 HTTP 401/403，业务失败才是 200 + code != 0
        raw = e.read().decode("utf-8", "replace")
        try:
            return json.loads(raw)
        except Exception:
            raise SystemExit(f"✗ {path}: HTTP {e.code} {raw[:200]}")


def must(res, what):
    if res.get("code") != 0:
        raise SystemExit(f"✗ {what}: {res.get('msg')}")
    return res.get("data")


def ok(msg):
    print(f"✓ {msg}")


def expect_fail(base, path, body, what, keyword):
    res = call(base, path, "POST", body)
    if res.get("code") == 0:
        raise SystemExit(f"✗ {what}: 预期失败，实际成功")
    if keyword not in (res.get("msg") or ""):
        raise SystemExit(f"✗ {what}: 错误信息应包含「{keyword}」，实际：{res.get('msg')}")
    ok(f"{what}（{res.get('msg')}）")


def main():
    global TOKEN
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1:8888")
    ap.add_argument("--user", default="admin")
    ap.add_argument("--password", default="Admin@123456")
    ap.add_argument("--token", default=os.environ.get("JXC_TOKEN", ""),
                    help="已登录的 x-token；提供后不再走登录/验证码")
    a = ap.parse_args()
    base = a.base

    if a.token:
        TOKEN = a.token
        ok("使用已有 token")
    else:
        login = call(base, "/base/login", "POST",
                     {"username": a.user, "password": a.password})
        if login.get("code") != 0:
            msg = login.get("msg") or ""
            if "验证码" in msg:
                raise SystemExit(
                    "✗ 登录需要验证码。请按 docs/18 临时把 captcha_open 改成 500 并重启，"
                    "或用 --token 传入已登录的 x-token。")
            raise SystemExit(f"✗ 登录失败：{msg}")
        TOKEN = login["data"]["token"]
        ok("登录成功")

    # 每次用一套全新的仓库 + 两个 SKU：断言只针对本次数据，可反复运行
    tag = f"{int(time.time()) % 1000000:06d}"
    wh = must(call(base, "/jxc/master/warehouses", "POST",
                   {"code": f"WH-SMOKE-{tag}", "name": f"冒烟仓{tag}"}), "建仓")["ID"]
    ga = must(call(base, "/jxc/master/goods", "POST",
                   {"code": f"SMOKE-A-{tag}", "name": "冒烟商品A",
                    "skus": [{"color": "黑", "size": "M",
                              "skuCode": f"SMOKE-A-{tag}-M", "salePrice": 100}]}), "建商品A")
    gb = must(call(base, "/jxc/master/goods", "POST",
                   {"code": f"SMOKE-B-{tag}", "name": "冒烟商品B",
                    "skus": [{"color": "白", "size": "L",
                              "skuCode": f"SMOKE-B-{tag}-L", "salePrice": 50}]}), "建商品B")
    a = ga["skus"][0]["ID"]
    b = gb["skus"][0]["ID"]
    ok(f"基础资料：仓库 {wh} / SKU A={a} B={b}")

    def balances():
        rows = must(call(base, f"/jxc/ledger/balance?warehouseId={wh}"), "查台账")
        return {r["skuId"]: r for r in rows}

    def balance_of(sku):
        return balances().get(sku, {"qty": 0, "value": 0, "available": 0})

    # 采购：A 10@60，B 5@30
    po = must(call(base, "/jxc/purchase", "POST", {"warehouseId": wh, "items": [
        {"skuId": a, "qty": 10, "unitCost": 60},
        {"skuId": b, "qty": 5, "unitCost": 30},
    ]}), "建采购单")
    must(call(base, f"/jxc/purchase/{po['ID']}/approve", "POST"), "审核采购单")
    must(call(base, f"/jxc/purchase/{po['ID']}/receive", "POST"), "采购入库")
    assert balance_of(a)["qty"] == 10 and balance_of(a)["value"] == 600, balance_of(a)
    assert balance_of(b)["qty"] == 5 and balance_of(b)["value"] == 150, balance_of(b)
    ok("采购入库：A 10/600，B 5/150")

    # 销售单价必须取 SKU 售价：前端传 1 元不能生效
    so = must(call(base, "/jxc/sale", "POST", {"warehouseId": wh, "items": [
        {"skuId": a, "qty": 2, "price": 1},
    ]}), "开单（传篡改价格）")
    assert so["totalAmount"] == 200, so
    must(call(base, f"/jxc/sale/{so['ID']}/ship", "POST"), "销售出库")
    assert balance_of(a)["qty"] == 8, balance_of(a)
    ok("销售 2 件 A：单价被强制为 100，总额 200，结存 8")

    # 退货 1 件：成本按原单 60
    ret = must(call(base, "/jxc/return", "POST", {
        "saleId": so["ID"], "items": [{"skuId": a, "qty": 1}], "reason": "冒烟退货"
    }), "建退货单")
    assert ret["amount"] == 100 and ret["costAmount"] == 60, ret
    must(call(base, f"/jxc/return/{ret['ID']}/post", "POST"), "退货入库")
    assert balance_of(a)["qty"] == 9, balance_of(a)
    ok("退货 1 件：退款 100 / 成本 60，结存 9")

    # 已退货的销售单不能整单红冲（否则库存会多回补）
    expect_fail(base, "/jxc/reversal", {
        "sourceType": "sale", "sourceId": so["ID"], "reason": "错误地整单红冲"
    }, "已退货销售单红冲被拒绝", "退货")

    # 两张各退 1 件的草稿单：第二张入库必须被可退上限挡住
    r1 = must(call(base, "/jxc/return", "POST", {
        "saleId": so["ID"], "items": [{"skuId": a, "qty": 1}], "reason": "草稿1"
    }), "建退货草稿1")
    r2 = must(call(base, "/jxc/return", "POST", {
        "saleId": so["ID"], "items": [{"skuId": a, "qty": 1}], "reason": "草稿2"
    }), "建退货草稿2")
    must(call(base, f"/jxc/return/{r1['ID']}/post", "POST"), "退货草稿1入库")
    expect_fail(base, f"/jxc/return/{r2['ID']}/post", {}, "第二张退货超退被拒绝", "最多可退")
    assert balance_of(a)["qty"] == 10, balance_of(a)
    ok("退货上限按入库时校验：两张草稿不能叠加超退，结存 10")

    # 销售红冲：货回来、成本精确抵消
    so2 = must(call(base, "/jxc/sale", "POST", {"warehouseId": wh, "items": [
        {"skuId": a, "qty": 2},
    ]}), "再开一单")
    must(call(base, f"/jxc/sale/{so2['ID']}/ship", "POST"), "出库")
    assert balance_of(a)["qty"] == 8
    rev = must(call(base, "/jxc/reversal", "POST", {
        "sourceType": "sale", "sourceId": so2["ID"], "reason": "冒烟红冲"
    }), "红冲销售单")
    assert balance_of(a)["qty"] == 10, balance_of(a)
    ok(f"销售红冲 {rev['reversalNo']}：结存回到 10")

    # 采购按原价红冲：10@5 + 10@7 后冲掉 10@5，剩余金额必须是 220 而不是 230
    po1 = must(call(base, "/jxc/purchase", "POST", {"warehouseId": wh, "items": [
        {"skuId": b, "qty": 10, "unitCost": 5},
    ]}), "采购批次1")
    must(call(base, f"/jxc/purchase/{po1['ID']}/approve", "POST"), "审核批次1")
    must(call(base, f"/jxc/purchase/{po1['ID']}/receive", "POST"), "入库批次1")
    po2 = must(call(base, "/jxc/purchase", "POST", {"warehouseId": wh, "items": [
        {"skuId": b, "qty": 10, "unitCost": 7},
    ]}), "采购批次2")
    must(call(base, f"/jxc/purchase/{po2['ID']}/approve", "POST"), "审核批次2")
    must(call(base, f"/jxc/purchase/{po2['ID']}/receive", "POST"), "入库批次2")
    must(call(base, "/jxc/reversal", "POST", {
        "sourceType": "purchase", "sourceId": po1["ID"], "reason": "供应商开错票"
    }), "红冲采购批次1")
    bb = balance_of(b)
    assert bb["qty"] == 15 and bb["value"] == 220, bb
    ok("采购红冲按原价：B 15 件 / 220 元（若按均价冲会是 230）")

    # 采购退货 + 采购退货红冲：退 1 件再红冲，库存/金额应完整回到 15/220
    pr = must(call(base, "/jxc/purchase-return", "POST", {
        "purchaseId": po2["ID"], "items": [{"skuId": b, "qty": 1}], "reason": "冒烟采购退货"
    }), "建采购退货单")
    assert pr["amount"] == 7 and pr["items"][0]["unitCost"] == 7, pr
    must(call(base, f"/jxc/purchase-return/{pr['ID']}/post", "POST"), "采购退货出库")
    bb = balance_of(b)
    assert bb["qty"] == 14 and bb["value"] == 213, bb
    pr_rev = must(call(base, "/jxc/reversal", "POST", {
        "sourceType": "purchase_return", "sourceId": pr["ID"], "reason": "供应商取消退货"
    }), "红冲采购退货")
    bb = balance_of(b)
    assert bb["qty"] == 15 and bb["value"] == 220, bb
    prd = must(call(base, f"/jxc/purchase-return/{pr['ID']}"), "查采购退货详情")
    assert prd["reversedAt"], prd
    returnable = must(call(base, f"/jxc/purchase-return/returnable?purchaseId={po2['ID']}"), "查可退数量")
    row = next(i for i in returnable if i["skuId"] == b)
    assert row["canReturn"] == 10, row
    ok(f"采购退货红冲 {pr_rev['reversalNo']}：库存回到 15/220，可退数量回到 10")

    # 盘点建单后账面又发生变化：完成盘点必须被拒绝
    chk = must(call(base, "/jxc/check", "POST", {"warehouseId": wh, "remark": "冒烟盘点"}), "建盘点单")
    row = next(i for i in chk["items"] if i["skuId"] == a)
    must(call(base, f"/jxc/check/{chk['ID']}/items", "POST", {
        "items": [{"id": row["ID"], "actualQty": row["bookQty"]}]
    }), "录实盘")
    so3 = must(call(base, "/jxc/sale", "POST", {"warehouseId": wh, "items": [
        {"skuId": a, "qty": 1},
    ]}), "盘点期间又开单")
    must(call(base, f"/jxc/sale/{so3['ID']}/ship", "POST"), "盘点期间又出库")
    expect_fail(base, f"/jxc/check/{chk['ID']}/post", {}, "账面变化后完成盘点被拒绝", "账面")
    must(call(base, f"/jxc/check/{chk['ID']}/cancel", "POST"), "取消盘点")

    # 原子开单：失败不留草稿/预留；成功后只发一次 checkout
    before_drafts = must(call(base, "/jxc/sale/page?page=1&pageSize=1&status=1"), "查草稿单")["total"]
    expect_fail(base, "/jxc/sale/checkout", {
        "warehouseId": wh, "items": [{"skuId": a, "qty": 999}],
    }, "原子开单库存不足回滚", "可售")
    after_drafts = must(call(base, "/jxc/sale/page?page=1&pageSize=1&status=1"), "再查草稿单")["total"]
    assert after_drafts == before_drafts, (before_drafts, after_drafts)

    co = must(call(base, "/jxc/sale/checkout", "POST", {
        "warehouseId": wh, "items": [{"skuId": a, "qty": 1}],
    }), "原子开单+出库")
    assert co["status"] == 2 and co["totalAmount"] == 100 and co["costAmount"] == 60, co
    assert co["items"][0]["costAmt"] == 60, co["items"][0]
    assert balance_of(a)["qty"] == 8, balance_of(a)
    sp = must(call(base, f"/jxc/sale/page?page=1&pageSize=1&status=2&keyword={co['orderNo']}"), "单据服务端筛选")
    assert sp["total"] == 1 and len(sp["list"]) == 1, sp
    cap = must(call(base, "/jxc/sale/page?page=1&pageSize=500"), "分页上限")
    assert cap["pageSize"] == 100, cap
    ok(f"原子开单 {co['orderNo']}：状态2/总额100/成本60，结存8；服务端筛选total=1，pageSize截断100")

    # 流水：结存列由后端计算，第一行（最新）应等于当前结存；退货/盘点的单据号也要能回溯
    mv = must(call(base, f"/jxc/ledger/moves?warehouseId={wh}&skuId={a}&page=1&pageSize=200"), "查流水")
    assert mv["total"] >= 1 and mv["list"][0].get("balanceAfter") == balance_of(a)["qty"], mv["list"][0]
    assert all(x.get("sourceNo") for x in mv["list"]), [x.get("sourceType") for x in mv["list"] if not x.get("sourceNo")]
    ok("账本流水：balanceAfter 与当前结存一致，且每条都有可读单号")

    # 对账
    rec = must(call(base, "/jxc/ledger/reconcile"), "对账")
    assert rec["ok"], rec
    last = must(call(base, "/jxc/ledger/reconcile/last"), "最近对账")
    assert last["ok"] is True and last["checkedAt"] and last["stale"] is False, last
    assert last["checkedSkus"] >= 1, last
    ok(f"对账：账实相符；last checkedAt={last['checkedAt']} / {last['checkedSkus']} SKU")

    # 月报期末库存 = 账本累加，且与当前全部结存金额一致
    ym = datetime.date.today().strftime("%Y-%m")
    s = must(call(base, f"/jxc/period/summary?month={ym}"), "月报")
    all_bal = must(call(base, "/jxc/ledger/balance"), "查全部结存")
    total_value = round(sum(float(r["value"]) for r in all_bal), 2)
    assert abs(float(s["stockValue"]) - total_value) < 0.01, (s["stockValue"], total_value)
    assert "purchaseReturnAmount" in s, s
    dash = must(call(base, "/jxc/dashboard/summary"), "工作台聚合")
    assert abs(float(dash["stockValue"]) - total_value) < 0.01, (dash["stockValue"], total_value)
    assert dash["todaySalesAmount"] >= 100, dash
    assert dash["lowStock"]["threshold"] == 10, dash["lowStock"]
    assert dash["pending"]["total"] >= 0, dash["pending"]
    ok(f"月报 {ym}：净销售 {s['salesAmount']:.2f}，净成本 {s['salesCost']:.2f}，"
       f"毛利 {s['gross']:.2f}，采购退货 {s['purchaseReturnAmount']:.2f}，期末库存 {s['stockValue']:.2f}")
    ok(f"工作台：库存 {dash['stockValue']:.2f}，今日销售 {dash['todaySalesAmount']:.2f}，"
       f"低库存 {dash['lowStock']['count']}，待办 {dash['pending']['total']}")

    print("\n全部通过 ✅  (仓库/SKU 前缀：%s)" % tag)


if __name__ == "__main__":
    main()

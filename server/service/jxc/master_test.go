package jxc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/money"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	jxcReq "github.com/flipped-aurora/gin-vue-admin/server/model/jxc/request"
)

// 基础资料测试：重点不是 CRUD 能跑，而是
//   - 编码唯一、状态只能是 0/1；
//   - 主键/状态由服务端决定（DTO 里根本没有这些字段）；
//   - 停用后的仓库/商品/规格不能再被开单使用。

func TestMaster_WarehouseCRUDAndStatus(t *testing.T) {
	db := newLedgerTestDB(t)
	ms := new(MasterService)
	ctx := context.Background()

	w, err := ms.CreateWarehouse(ctx, jxcReq.WarehouseInput{Code: "WH-A", Name: "主仓"})
	if err != nil {
		t.Fatalf("建仓库失败: %v", err)
	}
	if w.ID == 0 || w.Status != 1 {
		t.Fatalf("应生成 ID 且默认启用，实际 %+v", w)
	}
	if _, err := ms.CreateWarehouse(ctx, jxcReq.WarehouseInput{Code: "WH-A", Name: "重复"}); err == nil {
		t.Fatal("重复编码应被拒绝")
	}
	if _, err := ms.CreateWarehouse(ctx, jxcReq.WarehouseInput{Code: " ", Name: "空编码"}); err == nil {
		t.Fatal("空编码应被拒绝")
	}

	got, err := ms.UpdateWarehouse(ctx, w.ID, jxcReq.WarehouseInput{Code: "WH-B", Name: "主仓（改）"})
	if err != nil {
		t.Fatalf("编辑仓库失败: %v", err)
	}
	if got.Code != "WH-B" || got.Name != "主仓（改）" {
		t.Errorf("编辑后字段不对：%+v", got)
	}
	if err := ms.SetWarehouseStatus(ctx, w.ID, 0); err != nil {
		t.Fatalf("停用仓库失败: %v", err)
	}
	active, _ := ms.ListWarehouses(ctx, true)
	if len(active) != 0 {
		t.Errorf("停用后 active 列表应为空，实际 %d", len(active))
	}
	all, _ := ms.ListWarehouses(ctx, false)
	if len(all) != 1 {
		t.Errorf("全量列表应有 1 条，实际 %d", len(all))
	}
	if err := ms.SetWarehouseStatus(ctx, w.ID, 2); err == nil {
		t.Error("非法状态应被拒绝")
	}
	if err := ms.SetWarehouseStatus(ctx, 99999, 1); err == nil {
		t.Error("不存在的仓库应报错")
	}
	if q, _ := balanceOf(t, db, w.ID, 1); q != 0 {
		t.Errorf("不应产生库存，实际 %d", q)
	}
}

func TestMaster_GoodsCreateUpdateAndSkuStatus(t *testing.T) {
	_ = newLedgerTestDB(t)
	ms := new(MasterService)
	ctx := context.Background()

	goods, err := ms.CreateGoods(ctx, jxcReq.GoodsInput{
		Code: "G-A", Name: "卫衣",
		Skus: []jxcReq.GoodsSkuInput{
			{Color: "黑", Size: "M", SkuCode: "SKU-A1", SalePrice: 99.999, RefCost: 40.556},
			{Color: "白", Size: "L", SkuCode: "SKU-A2", SalePrice: 109},
		},
	})
	if err != nil {
		t.Fatalf("建商品失败: %v", err)
	}
	if len(goods.Skus) != 2 || goods.Skus[0].ID == 0 {
		t.Fatalf("应生成两个规格，实际 %+v", goods)
	}
	if goods.Skus[0].SalePrice != 100 || goods.Skus[0].RefCost != 40.56 {
		t.Errorf("金额应规整到 2 位小数，实际 %.4f/%.4f", goods.Skus[0].SalePrice, goods.Skus[0].RefCost)
	}

	firstID := goods.Skus[0].ID
	off := int8(0)
	updated, err := ms.UpdateGoods(ctx, goods.ID, jxcReq.GoodsInput{
		Code: "G-A", Name: "卫衣（改）",
		Skus: []jxcReq.GoodsSkuInput{
			{ID: firstID, Color: "黑", Size: "M", SkuCode: "SKU-A1", SalePrice: 88.8, RefCost: 30, Status: &off},
			{Color: "灰", Size: "S", SkuCode: "SKU-A3", SalePrice: 77},
		},
	})
	if err != nil {
		t.Fatalf("编辑商品失败: %v", err)
	}
	if updated.Name != "卫衣（改）" || len(updated.Skus) != 3 {
		t.Fatalf("编辑后应有 3 个规格，实际 %+v", updated)
	}
	if updated.Skus[0].SalePrice != 88.8 || updated.Skus[0].Status != 0 {
		t.Errorf("规格改价/停用失败：%+v", updated.Skus[0])
	}

	active, err := ms.ListGoods(ctx, true)
	if err != nil {
		t.Fatalf("active 列表失败: %v", err)
	}
	if len(active) != 1 || len(active[0].Skus) != 2 {
		t.Fatalf("应返回 1 个商品 / 2 个启用规格（第一个已停用），实际 %d/%d",
			len(active), len(active[0].Skus))
	}
	for _, s := range active[0].Skus {
		if s.ID == firstID {
			t.Error("已停用的规格不应出现在 active 列表")
		}
	}

	if err := ms.SetSkuStatus(ctx, firstID, 1); err != nil {
		t.Fatalf("恢复规格失败: %v", err)
	}
	if err := ms.SetGoodsStatus(ctx, goods.ID, 0); err != nil {
		t.Fatalf("停用商品失败: %v", err)
	}
	if active, _ := ms.ListGoods(ctx, true); len(active) != 0 {
		t.Errorf("商品停用后 active 列表应为空，实际 %d", len(active))
	}

	if _, err := ms.CreateGoods(ctx, jxcReq.GoodsInput{
		Code: "G-B", Name: "坏价格", Skus: []jxcReq.GoodsSkuInput{
			{Color: "黑", Size: "M", SkuCode: "SKU-B1", SalePrice: -1},
		},
	}); err == nil {
		t.Error("负售价应被拒绝")
	}
	if _, err := ms.CreateGoods(ctx, jxcReq.GoodsInput{
		Code: "G-C", Name: "重复条码", Skus: []jxcReq.GoodsSkuInput{
			{Color: "黑", Size: "M", SkuCode: "SKU-A1"},
		},
	}); err == nil || !strings.Contains(err.Error(), "条码") {
		t.Errorf("重复规格条码应被拒绝，got %v", err)
	}
}

func TestMaster_PartnerCRUDAndStatus(t *testing.T) {
	_ = newLedgerTestDB(t)
	ms := new(MasterService)
	ctx := context.Background()

	p, err := ms.CreatePartner(ctx, jxcReq.PartnerInput{
		Kind: jxc.PartnerSupplier, Code: "S-A", Name: "供应商 A", Phone: "138",
	})
	if err != nil {
		t.Fatalf("建供应商失败: %v", err)
	}
	if p.Status != 1 || p.Kind != jxc.PartnerSupplier {
		t.Fatalf("默认状态/类型不对：%+v", p)
	}
	if _, err := ms.CreatePartner(ctx, jxcReq.PartnerInput{Kind: "bad", Code: "X", Name: "X"}); err == nil {
		t.Error("非法类型应被拒绝")
	}

	got, err := ms.UpdatePartner(ctx, p.ID, jxcReq.PartnerInput{
		Kind: jxc.PartnerSupplier, Code: "S-A", Name: "供应商 A（改）", Contact: "张三", Phone: "139",
	})
	if err != nil {
		t.Fatalf("编辑供应商失败: %v", err)
	}
	if got.Name != "供应商 A（改）" || got.Contact != "张三" {
		t.Errorf("编辑后字段不对：%+v", got)
	}
	if _, err := ms.UpdatePartner(ctx, p.ID, jxcReq.PartnerInput{
		Kind: jxc.PartnerCustomer, Code: "S-A", Name: "供应商 A（改）",
	}); err == nil {
		t.Error("往来单位类型不应允许修改")
	}

	if err := ms.SetPartnerStatus(ctx, p.ID, 0); err != nil {
		t.Fatalf("停用供应商失败: %v", err)
	}
	active, _ := ms.ListPartners(ctx, jxc.PartnerSupplier, true)
	if len(active) != 0 {
		t.Errorf("停用后 active 列表应为空，实际 %d", len(active))
	}
}

// TestMaster_DisabledSkuCannotSell 停用规格后，开单必须被拒绝；
// 恢复后又能卖。状态不是展示字段，它必须真的影响业务。
func TestMaster_DisabledSkuCannotSell(t *testing.T) {
	db := newLedgerTestDB(t)
	wh, skuID := seedSku(t, db)
	ms := new(MasterService)
	ss := NewSaleService()
	ls := new(LedgerService)
	ctx := context.Background()
	seedStock(t, db, wh, skuID, 3, 10)

	if err := ms.SetSkuStatus(ctx, skuID, 0); err != nil {
		t.Fatalf("停用规格失败: %v", err)
	}
	if _, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1}}}); err == nil {
		t.Fatal("停用规格不应能开单")
	}
	if err := ms.SetSkuStatus(ctx, skuID, 1); err != nil {
		t.Fatalf("恢复规格失败: %v", err)
	}
	so, err := ss.Create(ctx, &jxc.SaleOrder{WarehouseID: wh,
		Items: []jxc.SaleItem{{SkuID: skuID, Qty: 1}}})
	if err != nil {
		t.Fatalf("恢复后应能开单: %v", err)
	}
	if err := ss.Ship(ctx, so.ID, "t", time.Now()); err != nil {
		t.Fatalf("出库失败: %v", err)
	}
	if _, v := balanceOf(t, db, wh, skuID); v != money.FromYuan(20) {
		t.Errorf("卖 1 件后应剩 2 件/20.00，实际 %.2f", v.Yuan())
	}
	mustInvariant(t, ls)
}

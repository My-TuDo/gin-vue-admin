package jxc

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

func TestPosScan(t *testing.T) {
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Goods{}, &jxc.GoodsSku{}, &jxc.PosScan{},
	)
	db := global.GVA_DB
	ctx := context.Background()

	// 造数据：分类 + 商品 + 2 个 SKU（带条码）
	cat := jxc.GoodsCategory{Name: "扫码枪测试分类", Sort: 1, Status: 1}
	db.Create(&cat)
	goods := jxc.Goods{Name: "扫码枪测试商品", CategoryID: &cat.ID, Unit: "件", Status: 1}
	db.Create(&goods)
	skuA := jxc.GoodsSku{GoodsID: goods.ID, SkuCode: "POS-A", Barcode: "6901234567890", Color: "白", Size: "M", SalePrice: 20, Status: 1}
	skuB := jxc.GoodsSku{GoodsID: goods.ID, SkuCode: "POS-B", Barcode: "6901234567891", Color: "黑", Size: "L", SalePrice: 30, Status: 1}
	db.Create(&skuA)
	db.Create(&skuB)

	s := PosScanService{}

	// 1. 按条码上架
	scan, err := s.CreateScan(ctx, "6901234567890", 0, 1)
	if err != nil {
		t.Fatalf("按条码上架失败: %v", err)
	}
	if scan.Qty != 1 || scan.Status != 0 {
		t.Fatalf("上架记录异常: qty=%d status=%d", scan.Qty, scan.Status)
	}

	// 2. 同 SKU 连续扫 → 数量累加
	scan, err = s.CreateScan(ctx, "6901234567890", 0, 2)
	if err != nil {
		t.Fatalf("累加上架失败: %v", err)
	}
	if scan.Qty != 3 {
		t.Fatalf("累加失败: 期望 3 实际 %d", scan.Qty)
	}

	// 2b. qty<=0 时默认 1（合并到同 SKU 待处理，数量 +1）
	scan, err = s.CreateScan(ctx, "6901234567890", 0, 0)
	if err != nil {
		t.Fatalf("默认数量上架失败: %v", err)
	}
	if scan.Qty != 4 {
		t.Fatalf("默认数量失败: 期望 4 实际 %d", scan.Qty)
	}

	// 3. 按 skuCode 上架（无条码路径）
	_, err = s.CreateScan(ctx, "POS-B", 0, 1)
	if err != nil {
		t.Fatalf("按编码上架失败: %v", err)
	}

	// 4. 未找到条码
	_, err = s.CreateScan(ctx, "9999999999999", 0, 1)
	if err == nil {
		t.Fatal("未找到条码应报错")
	}

	// 4b. skuID 指定但不存在
	_, err = s.CreateScan(ctx, "", 99999, 1)
	if err == nil {
		t.Fatal("不存在的 skuID 应报错")
	}

	// 4c. skuID 指定成功路径
	scanByID, err := s.CreateScan(ctx, "", skuA.ID, 1)
	if err != nil {
		t.Fatalf("按 skuID 上架失败: %v", err)
	}
	if scanByID.SkuID != skuA.ID {
		t.Fatalf("按 skuID 上架异常: %+v", scanByID)
	}

	// 5. 数量上限
	_, err = s.CreateScan(ctx, "6901234567890", 0, 100)
	if err == nil {
		t.Fatal("数量超限应报错")
	}

	// 6. 待处理列表（2 条：POS-A qty=3 + POS-B qty=1）
	list, err := s.ListPending(ctx)
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("待处理条数异常: 期望 2 实际 %d", len(list))
	}
	if list[0].Sku == nil || list[0].Sku.SkuCode != "POS-A" || list[0].Sku.GoodsName != "扫码枪测试商品" {
		t.Fatalf("SKU 摘要缺失: %+v", list[0].Sku)
	}

	// 7. 确认（仅确认 POS-A 那条）
	ids := []uint{list[0].ID}
	if err := s.ConfirmScan(ctx, ids); err != nil {
		t.Fatalf("确认失败: %v", err)
	}
	list2, err := s.ListPending(ctx)
	if err != nil {
		t.Fatalf("列表2失败: %v", err)
	}
	if len(list2) != 1 || list2[0].Sku.SkuCode != "POS-B" {
		t.Fatalf("确认后待处理应只剩 POS-B: %+v", list2)
	}

	// 8. 空 ids 确认报错
	if err := s.ConfirmScan(ctx, []uint{}); err == nil {
		t.Fatal("空 ids 应报错")
	}

	// 9. SKU 在但商品被删 → GoodsName 为空（不报错，覆盖 goods 查询失败分支）
	db.Unscoped().Delete(&jxc.Goods{}, goods.ID)
	list3, err := s.ListPending(ctx)
	if err != nil {
		t.Fatalf("列表3失败: %v", err)
	}
	if len(list3) != 1 || list3[0].Sku == nil || list3[0].Sku.GoodsName != "" {
		t.Fatalf("商品删除后 GoodsName 应为空: %+v", list3)
	}

	// 10. SKU 被删后 ListPending → Sku 摘要为 nil（不报错，覆盖 sku 查询失败分支）
	db.Unscoped().Delete(&jxc.GoodsSku{}, skuB.ID)
	list4, err := s.ListPending(ctx)
	if err != nil {
		t.Fatalf("列表4失败: %v", err)
	}
	if len(list4) != 1 || list4[0].Sku != nil {
		t.Fatalf("SKU 删除后摘要应为 nil: %+v", list4)
	}
}

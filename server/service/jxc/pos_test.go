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
		&jxc.GoodsCategory{}, &jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PosScan{}, &jxc.PosSession{},
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

	// 0. 生成收银台码（会话）
	ss1, err := s.CreateSession(ctx, "1号收银机")
	if err != nil {
		t.Fatalf("生成会话失败: %v", err)
	}
	if len(ss1.Code) != 6 {
		t.Fatalf("会话码长度异常: %s", ss1.Code)
	}
	// 0b. 生成第二个会话（模拟另一台 PC）
	ss2, err := s.CreateSession(ctx, "2号收银机")
	if err != nil {
		t.Fatalf("生成会话2失败: %v", err)
	}
	// 0c. 会话校验：有效
	if _, err := s.CheckSession(ctx, ss1.Code); err != nil {
		t.Fatalf("有效会话校验失败: %v", err)
	}
	// 0d. 会话校验：无效码
	if _, err := s.CheckSession(ctx, "XXXXXX"); err == nil {
		t.Fatal("无效码应校验失败")
	}

	// 1. 按条码上架（投递到会话1）
	scan, err := s.CreateScan(ctx, ss1.Code, "6901234567890", 0, 1)
	if err != nil {
		t.Fatalf("按条码上架失败: %v", err)
	}
	if scan.Qty != 1 || scan.Status != 0 || scan.Session != ss1.Code {
		t.Fatalf("上架记录异常: qty=%d status=%d session=%s", scan.Qty, scan.Status, scan.Session)
	}

	// 2. 同 SKU 同会话连续扫 → 数量累加
	scan, err = s.CreateScan(ctx, ss1.Code, "6901234567890", 0, 2)
	if err != nil {
		t.Fatalf("累加上架失败: %v", err)
	}
	if scan.Qty != 3 {
		t.Fatalf("累加失败: 期望 3 实际 %d", scan.Qty)
	}

	// 2b. qty<=0 时默认 1（合并到同会话同 SKU 未消费，数量 +1）
	scan, err = s.CreateScan(ctx, ss1.Code, "6901234567890", 0, 0)
	if err != nil {
		t.Fatalf("默认数量上架失败: %v", err)
	}
	if scan.Qty != 4 {
		t.Fatalf("默认数量失败: 期望 4 实际 %d", scan.Qty)
	}

	// 3. 按 skuCode 上架（无条码路径）+ 另一会话同 SKU（互不累加）
	_, err = s.CreateScan(ctx, ss1.Code, "POS-B", 0, 1)
	if err != nil {
		t.Fatalf("按编码上架失败: %v", err)
	}
	_, err = s.CreateScan(ctx, ss2.Code, "6901234567890", 0, 1)
	if err != nil {
		t.Fatalf("会话2上架失败: %v", err)
	}

	// 4. 未找到条码
	_, err = s.CreateScan(ctx, ss1.Code, "9999999999999", 0, 1)
	if err == nil {
		t.Fatal("未找到条码应报错")
	}

	// 4b. skuID 指定但不存在
	_, err = s.CreateScan(ctx, ss1.Code, "", 99999, 1)
	if err == nil {
		t.Fatal("不存在的 skuID 应报错")
	}

	// 4c. skuID 指定成功路径
	scanByID, err := s.CreateScan(ctx, ss1.Code, "", skuA.ID, 1)
	if err != nil {
		t.Fatalf("按 skuID 上架失败: %v", err)
	}
	if scanByID.SkuID != skuA.ID {
		t.Fatalf("按 skuID 上架异常: %+v", scanByID)
	}

	// 4d. 无效会话扫码应报错
	if _, err := s.CreateScan(ctx, "BADCOD", "6901234567890", 0, 1); err == nil {
		t.Fatal("无效会话应报错")
	}
	// 4e. 空会话扫码应报错
	if _, err := s.CreateScan(ctx, "", "6901234567890", 0, 1); err == nil {
		t.Fatal("空会话应报错")
	}

	// 5. 数量上限
	_, err = s.CreateScan(ctx, ss1.Code, "6901234567890", 0, 100)
	if err == nil {
		t.Fatal("数量超限应报错")
	}

	// 6. 待处理列表（会话1：POS-A qty=5 + POS-B qty=1，共 2 条；会话2 的 1 条不可见）
	list, err := s.ListPending(ctx, ss1.Code)
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("会话1待处理条数异常: 期望 2 实际 %d", len(list))
	}
	if list[0].Sku == nil || list[0].Sku.SkuCode != "POS-A" || list[0].Sku.GoodsName != "扫码枪测试商品" {
		t.Fatalf("SKU 摘要缺失: %+v", list[0].Sku)
	}
	// 6b. 会话2 列表只有自己的 1 条
	list2s, err := s.ListPending(ctx, ss2.Code)
	if err != nil {
		t.Fatalf("会话2列表失败: %v", err)
	}
	if len(list2s) != 1 || list2s[0].Sku.SkuCode != "POS-A" {
		t.Fatalf("会话2隔离异常: %+v", list2s)
	}
	// 6c. 空会话列表报错
	if _, err := s.ListPending(ctx, ""); err == nil {
		t.Fatal("空会话列表应报错")
	}

	// 7. 确认（仅确认 POS-A 那条）
	ids := []uint{list[0].ID}
	if err := s.ConfirmScan(ctx, ids); err != nil {
		t.Fatalf("确认失败: %v", err)
	}
	list2, err := s.ListPending(ctx, ss1.Code)
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
	list3, err := s.ListPending(ctx, ss1.Code)
	if err != nil {
		t.Fatalf("列表3失败: %v", err)
	}
	if len(list3) != 1 || list3[0].Sku == nil || list3[0].Sku.GoodsName != "" {
		t.Fatalf("商品删除后 GoodsName 应为空: %+v", list3)
	}

	// 10. SKU 被删后 ListPending → Sku 摘要为 nil（不报错，覆盖 sku 查询失败分支）
	db.Unscoped().Delete(&jxc.GoodsSku{}, skuB.ID)
	list4, err := s.ListPending(ctx, ss1.Code)
	if err != nil {
		t.Fatalf("列表4失败: %v", err)
	}
	if len(list4) != 1 || list4[0].Sku != nil {
		t.Fatalf("SKU 删除后摘要应为 nil: %+v", list4)
	}

	// 11. 作废会话后：校验失败 / 扫码失败 / 列表失败
	if err := s.DisableSession(ctx, ss1.Code); err != nil {
		t.Fatalf("作废会话失败: %v", err)
	}
	if _, err := s.CheckSession(ctx, ss1.Code); err == nil {
		t.Fatal("作废后校验应失败")
	}
	if _, err := s.CreateScan(ctx, ss1.Code, "6901234567890", 0, 1); err == nil {
		t.Fatal("作废后会话扫码应失败")
	}
	if _, err := s.ListPending(ctx, ss1.Code); err == nil {
		t.Fatal("作废后列表应失败")
	}
	// 11b. 空码作废报错
	if err := s.DisableSession(ctx, ""); err == nil {
		t.Fatal("空码作废应报错")
	}
}

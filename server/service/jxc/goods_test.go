package jxc

import (
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// gsvc 被测的 GoodsService 实例
var gsvc = &GoodsService{}

// goodsTestDB 内存库 + 5 张基础表 + 商品/规格表（复用 testutil 惯例）
func goodsTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
	)
}

// newTestGoods 构造一个测试商品并入库，返回带 ID 的实体
func newTestGoods(t *testing.T, code, name string) jxc.Goods {
	t.Helper()
	g := jxc.Goods{Code: code, Name: name, Unit: "件", Status: 1}
	if err := global.GVA_DB.Create(&g).Error; err != nil {
		t.Fatalf("创建测试商品失败: %v", err)
	}
	return g
}

// ========== 商品 SPU ==========

// TestGetGoodsPage 测试分页查询商品列表
func TestGetGoodsPage(t *testing.T) {
	goodsTestDB(t)
	newTestGoods(t, "SP001", "纯棉T恤")
	newTestGoods(t, "SP002", "牛仔裤")
	list, total, err := gsvc.GetGoodsPage(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 2 || len(list) != 2 {
		t.Fatalf("list=%v total=%d err=%v", list, total, err)
	}
}

// TestGetGoodsPage_Keyword 测试商品列表关键字过滤
func TestGetGoodsPage_Keyword(t *testing.T) {
	goodsTestDB(t)
	newTestGoods(t, "SP001", "纯棉T恤")
	newTestGoods(t, "SP002", "牛仔裤")
	_, total, err := gsvc.GetGoodsPage(ctx, request.PageInfo{Page: 1, PageSize: 10, Keyword: "T恤"})
	if err != nil || total != 1 {
		t.Fatalf("keyword 过滤失败: total=%d err=%v", total, err)
	}
}

// TestGetGoodsPage_Empty 测试空库分页查询
func TestGetGoodsPage_Empty(t *testing.T) {
	goodsTestDB(t)
	_, total, err := gsvc.GetGoodsPage(ctx, request.PageInfo{Page: 1, PageSize: 10})
	if err != nil || total != 0 {
		t.Fatalf("空库 total 应为 0: %d err=%v", total, err)
	}
}

// TestGetGoodsDetail 测试查询商品详情（含 SKU 列表）
func TestGetGoodsDetail(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	global.GVA_DB.Create(&jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M"})
	got, err := gsvc.GetGoodsDetail(ctx, g.ID)
	if err != nil || got.ID != g.ID || len(got.Skus) != 1 {
		t.Fatalf("detail=%+v err=%v", got, err)
	}
}

// TestGetGoodsDetail_NotFound 测试查询不存在的商品（Find 返回零值商品, 无错误）
func TestGetGoodsDetail_NotFound(t *testing.T) {
	goodsTestDB(t)
	got, err := gsvc.GetGoodsDetail(ctx, 999)
	if err != nil {
		t.Fatalf("Find 语义下查询不存在不应报错: %v", err)
	}
	if got.ID != 0 {
		t.Fatalf("应返回零值商品, got ID=%d", got.ID)
	}
}

// TestGetAllGoods 测试获取所有上架商品
func TestGetAllGoods(t *testing.T) {
	goodsTestDB(t)
	newTestGoods(t, "SP001", "纯棉T恤")
	// 构造下架商品：Create 时 status=0 会被 default:1 跳过, 需建后 Update
	g2 := jxc.Goods{Code: "SP002", Name: "下架款", Unit: "件", Status: 1}
	global.GVA_DB.Create(&g2)
	if err := global.GVA_DB.Model(&g2).Update("status", 0).Error; err != nil {
		t.Fatalf("设置下架失败: %v", err)
	}
	list, err := gsvc.GetAllGoods(ctx)
	if err != nil || len(list) != 1 || list[0].Code != "SP001" {
		t.Fatalf("list=%v err=%v", list, err)
	}
}

// TestCreateGoods 测试创建商品
func TestCreateGoods(t *testing.T) {
	goodsTestDB(t)
	g := &jxc.Goods{Code: "SP001", Name: "纯棉T恤", Unit: "件", Status: 1}
	if err := gsvc.CreateGoods(ctx, g); err != nil || g.ID == 0 {
		t.Fatalf("create err=%v", err)
	}
	var got jxc.Goods
	if err := global.GVA_DB.First(&got, g.ID).Error; err != nil || got.Name != "纯棉T恤" {
		t.Fatalf("落库校验失败: %+v err=%v", got, err)
	}
}

// TestUpdateGoods 测试更新商品（编码不可修改）
func TestUpdateGoods(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	g.Name = "重磅纯棉T恤"
	if err := gsvc.UpdateGoods(ctx, &g); err != nil {
		t.Fatalf("update err=%v", err)
	}
	var got jxc.Goods
	global.GVA_DB.First(&got, g.ID)
	if got.Name != "重磅纯棉T恤" || got.Code != "SP001" {
		t.Fatalf("更新结果不符: %+v", got)
	}
}

// TestDeleteGoods_OK 测试删除无 SKU 的商品成功
func TestDeleteGoods_OK(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	if err := gsvc.DeleteGoods(ctx, g.ID); err != nil {
		t.Fatalf("delete err=%v", err)
	}
	var cnt int64
	global.GVA_DB.Unscoped().Model(&jxc.Goods{}).Where("id = ?", g.ID).Count(&cnt)
	if cnt != 1 { // 软删除：记录仍在
		t.Fatalf("应为软删除, cnt=%d", cnt)
	}
}

// TestDeleteGoods_HasSku 测试删除存在 SKU 的商品被拒绝
func TestDeleteGoods_HasSku(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	global.GVA_DB.Create(&jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M"})
	err := gsvc.DeleteGoods(ctx, g.ID)
	if err == nil || err.Error() != "该商品存在 SKU, 请先删除 SKU" {
		t.Fatalf("应返回 SKU 引用拒绝, got %v", err)
	}
}

// TestSetGoodsStatus 测试上架/下架商品
func TestSetGoodsStatus(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	if err := gsvc.SetGoodsStatus(ctx, g.ID, 0); err != nil {
		t.Fatalf("setStatus err=%v", err)
	}
	var got jxc.Goods
	global.GVA_DB.First(&got, g.ID)
	if got.Status != 0 {
		t.Fatalf("状态未更新: %+v", got)
	}
}

// ========== 商品 SKU ==========

// TestGetSkuList 测试查询 SKU 列表（按商品过滤 / 全量）
func TestGetSkuList(t *testing.T) {
	goodsTestDB(t)
	g1 := newTestGoods(t, "SP001", "纯棉T恤")
	g2 := newTestGoods(t, "SP002", "牛仔裤")
	global.GVA_DB.Create(&jxc.GoodsSku{GoodsID: g1.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M"})
	global.GVA_DB.Create(&jxc.GoodsSku{GoodsID: g1.ID, SkuCode: "SP001-RED-L", Color: "红色", Size: "L"})
	global.GVA_DB.Create(&jxc.GoodsSku{GoodsID: g2.ID, SkuCode: "SP002-BLUE-M", Color: "蓝色", Size: "M"})
	// 按商品过滤
	list, err := gsvc.GetSkuList(ctx, g1.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("按商品过滤 list=%v err=%v", list, err)
	}
	// goodsId=0 返回全部（供采购/销售选品）
	all, err := gsvc.GetSkuList(ctx, 0)
	if err != nil || len(all) != 3 {
		t.Fatalf("全量 list=%v err=%v", all, err)
	}
}

// TestCreateSku_OK 测试创建 SKU 成功
func TestCreateSku_OK(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	sku := &jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M", CostPrice: 10, SalePrice: 20}
	if err := gsvc.CreateSku(ctx, sku); err != nil || sku.ID == 0 {
		t.Fatalf("create sku err=%v", err)
	}
}

// TestCreateSku_Duplicate 测试同商品相同颜色+尺码组合被拒绝
func TestCreateSku_Duplicate(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	global.GVA_DB.Create(&jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M"})
	err := gsvc.CreateSku(ctx, &jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M-2", Color: "红色", Size: "M"})
	if err == nil || err.Error() != "该商品下已存在相同颜色加尺码的 SKU" {
		t.Fatalf("应返回组合唯一拒绝, got %v", err)
	}
}

// TestCreateSku_DiffGoods 测试不同商品允许相同颜色+尺码
func TestCreateSku_DiffGoods(t *testing.T) {
	goodsTestDB(t)
	g1 := newTestGoods(t, "SP001", "纯棉T恤")
	g2 := newTestGoods(t, "SP002", "牛仔裤")
	if err := gsvc.CreateSku(ctx, &jxc.GoodsSku{GoodsID: g1.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M"}); err != nil {
		t.Fatalf("create g1 sku err=%v", err)
	}
	if err := gsvc.CreateSku(ctx, &jxc.GoodsSku{GoodsID: g2.ID, SkuCode: "SP002-RED-M", Color: "红色", Size: "M"}); err != nil {
		t.Fatalf("不同商品同色码应允许, err=%v", err)
	}
}

// TestUpdateSku 测试更新 SKU（归属与编码不可修改）
func TestUpdateSku(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M", SalePrice: 20}
	global.GVA_DB.Create(&sku)
	sku.SalePrice = 25
	if err := gsvc.UpdateSku(ctx, &sku); err != nil {
		t.Fatalf("update sku err=%v", err)
	}
	var got jxc.GoodsSku
	global.GVA_DB.First(&got, sku.ID)
	if got.SalePrice != 25 || got.SkuCode != "SP001-RED-M" {
		t.Fatalf("更新结果不符: %+v", got)
	}
}

// TestDeleteSku 测试删除 SKU（软删除）
func TestDeleteSku(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M"}
	global.GVA_DB.Create(&sku)
	if err := gsvc.DeleteSku(ctx, sku.ID); err != nil {
		t.Fatalf("delete sku err=%v", err)
	}
	var cnt int64
	global.GVA_DB.Unscoped().Model(&jxc.GoodsSku{}).Where("id = ?", sku.ID).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("应为软删除, cnt=%d", cnt)
	}
}

// TestSetSkuStatus 测试启用/停用 SKU
func TestSetSkuStatus(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M", Status: 1}
	global.GVA_DB.Create(&sku)
	if err := gsvc.SetSkuStatus(ctx, sku.ID, 0); err != nil {
		t.Fatalf("setStatus err=%v", err)
	}
	var got jxc.GoodsSku
	global.GVA_DB.First(&got, sku.ID)
	if got.Status != 0 {
		t.Fatalf("状态未更新: %+v", got)
	}
}

// ========== 编码自动生成 ==========

// TestCreateGoods_AutoCode 测试创建商品时编码自动生成（SP+日期+序号）
func TestCreateGoods_AutoCode(t *testing.T) {
	goodsTestDB(t)
	g := &jxc.Goods{Name: "纯棉T恤", Unit: "件", Status: 1}
	if err := gsvc.CreateGoods(ctx, g); err != nil {
		t.Fatalf("create err=%v", err)
	}
	prefix := prefixGoods + time.Now().Format("20060102")
	if !strings.HasPrefix(g.Code, prefix) {
		t.Fatalf("编码应为 %s 前缀, got %s", prefix, g.Code)
	}
}

// TestCreateGoods_AutoCodeSeq 测试同日前缀编码序号自增
func TestCreateGoods_AutoCodeSeq(t *testing.T) {
	goodsTestDB(t)
	g1 := &jxc.Goods{Name: "A", Unit: "件", Status: 1}
	g2 := &jxc.Goods{Name: "B", Unit: "件", Status: 1}
	if err := gsvc.CreateGoods(ctx, g1); err != nil {
		t.Fatalf("create g1 err=%v", err)
	}
	if err := gsvc.CreateGoods(ctx, g2); err != nil {
		t.Fatalf("create g2 err=%v", err)
	}
	if g2.Code <= g1.Code {
		t.Fatalf("第二个编码应大于第一个: %s vs %s", g1.Code, g2.Code)
	}
}

// TestCreateSku_AutoCode 测试创建 SKU 时编码自动生成（商品编码-颜色-尺码）
func TestCreateSku_AutoCode(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	sku := &jxc.GoodsSku{GoodsID: g.ID, Color: "红色", Size: "M"}
	if err := gsvc.CreateSku(ctx, sku); err != nil {
		t.Fatalf("create sku err=%v", err)
	}
	want := strings.ToUpper(g.Code + "-红色-M")
	if sku.SkuCode != want {
		t.Fatalf("skuCode 应为 %s, got %s", want, sku.SkuCode)
	}
}

// TestDeleteGoodsForever 测试彻底删除商品（物理删除后无记录）
func TestDeleteGoodsForever(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	if err := gsvc.DeleteGoodsForever(ctx, g.ID); err != nil {
		t.Fatalf("delete forever err=%v", err)
	}
	var cnt int64
	global.GVA_DB.Unscoped().Model(&jxc.Goods{}).Where("id = ?", g.ID).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("物理删除后应无记录, cnt=%d", cnt)
	}
}

// TestDeleteGoodsForever_HasSku 测试有 SKU 时彻底删除商品被拒绝
func TestDeleteGoodsForever_HasSku(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	global.GVA_DB.Create(&jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M"})
	err := gsvc.DeleteGoodsForever(ctx, g.ID)
	if err == nil || err.Error() != "该商品存在 SKU, 请先删除 SKU" {
		t.Fatalf("应返回 SKU 引用拒绝, got %v", err)
	}
}

// TestDeleteSkuForever 测试彻底删除 SKU（物理删除后无记录）
func TestDeleteSkuForever(t *testing.T) {
	goodsTestDB(t)
	g := newTestGoods(t, "SP001", "纯棉T恤")
	sku := jxc.GoodsSku{GoodsID: g.ID, SkuCode: "SP001-RED-M", Color: "红色", Size: "M"}
	global.GVA_DB.Create(&sku)
	if err := gsvc.DeleteSkuForever(ctx, sku.ID); err != nil {
		t.Fatalf("delete sku forever err=%v", err)
	}
	var cnt int64
	global.GVA_DB.Unscoped().Model(&jxc.GoodsSku{}).Where("id = ?", sku.ID).Count(&cnt)
	if cnt != 0 {
		t.Fatalf("物理删除后应无记录, cnt=%d", cnt)
	}
}

// ========== 数据库异常错误分支 ==========

// TestGoodsErrors 测试商品各方法在数据库异常时的错误分支
func TestGoodsErrors(t *testing.T) {
	goodsTestDB(t)
	dropTable(t, "goods")
	if _, _, err := gsvc.GetGoodsPage(ctx, request.PageInfo{Page: 1, PageSize: 10}); err == nil {
		t.Fatalf("GetGoodsPage 应报错")
	}
	if _, err := gsvc.GetGoodsDetail(ctx, 1); err == nil {
		t.Fatalf("GetGoodsDetail 应报错")
	}
	if _, err := gsvc.GetAllGoods(ctx); err == nil {
		t.Fatalf("GetAllGoods 应报错")
	}
	if err := gsvc.CreateGoods(ctx, &jxc.Goods{Code: "SP001", Name: "x"}); err == nil {
		t.Fatalf("CreateGoods 应报错")
	}
	g := &jxc.Goods{}
	g.ID = 1
	if err := gsvc.UpdateGoods(ctx, g); err == nil {
		t.Fatalf("UpdateGoods 应报错")
	}
	if err := gsvc.DeleteGoods(ctx, 1); err == nil {
		t.Fatalf("DeleteGoods 应报错")
	}
	if err := gsvc.SetGoodsStatus(ctx, 1, 0); err == nil {
		t.Fatalf("SetGoodsStatus 应报错")
	}
}

// TestSkuErrors 测试 SKU 各方法在数据库异常时的错误分支
func TestSkuErrors(t *testing.T) {
	goodsTestDB(t)
	dropTable(t, "goods_sku")
	if _, err := gsvc.GetSkuList(ctx, 1); err == nil {
		t.Fatalf("GetSkuList 应报错")
	}
	if err := gsvc.CreateSku(ctx, &jxc.GoodsSku{GoodsID: 1, Color: "红", Size: "M"}); err == nil {
		t.Fatalf("CreateSku 应报错")
	}
	sku := &jxc.GoodsSku{}
	sku.ID = 1
	if err := gsvc.UpdateSku(ctx, sku); err == nil {
		t.Fatalf("UpdateSku 应报错")
	}
	if err := gsvc.DeleteSku(ctx, 1); err == nil {
		t.Fatalf("DeleteSku 应报错")
	}
	if err := gsvc.SetSkuStatus(ctx, 1, 0); err == nil {
		t.Fatalf("SetSkuStatus 应报错")
	}
}

package jxc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
)

// gapi 被测的 GoodsApi 实例
var gapi = &GoodsApi{}

// goodsApiTestDB 内存库 + 5 张基础表 + 商品/规格表
func goodsApiTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{},
		&jxc.Goods{}, &jxc.GoodsSku{},
	)
}

// newCtxQuery 构造带 query 参数的 GET 请求上下文
func newCtxQuery(t *testing.T, rawQuery string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	return c, w
}

// parseBody 解析 handler 已写入的响应体
func parseBody(t *testing.T, w *httptest.ResponseRecorder) respBody {
	t.Helper()
	var rb respBody
	if err := json.Unmarshal(w.Body.Bytes(), &rb); err != nil {
		t.Fatalf("响应非 JSON: %s", w.Body.String())
	}
	return rb
}

// TestApiGoodsFlow 测试商品接口的增删改查及上下架流程
func TestApiGoodsFlow(t *testing.T) {
	goodsApiTestDB(t)
	// 创建
	_, rb := doReq(t, gapi.CreateGoods, http.MethodPost, `{"code":"SP001","name":"纯棉T恤","unit":"件","status":1}`)
	if rb.Code != 0 {
		t.Fatalf("create code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 分页列表
	c, w := newCtxQuery(t, "page=1&pageSize=10")
	gapi.GetGoodsPage(c)
	if w.Code != http.StatusOK {
		t.Fatalf("page status=%d", w.Code)
	}
	// 详情
	c, w = newCtxQuery(t, "id=1")
	gapi.GetGoodsDetail(c)
	if w.Code != http.StatusOK {
		t.Fatalf("detail status=%d", w.Code)
	}
	// 所有上架商品
	_, rb = doReq(t, gapi.GetAllGoods, http.MethodGet, "")
	if rb.Code != 0 {
		t.Fatalf("all code=%d", rb.Code)
	}
	// 更新
	_, rb = doReq(t, gapi.UpdateGoods, http.MethodPut, `{"ID":1,"code":"SP001","name":"重磅T恤","unit":"件","status":1}`)
	if rb.Code != 0 {
		t.Fatalf("update code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 上下架
	_, rb = doReq(t, gapi.SetGoodsStatus, http.MethodPut, `{"id":1,"status":0}`)
	if rb.Code != 0 {
		t.Fatalf("setStatus code=%d", rb.Code)
	}
	// 删除
	_, rb = doReq(t, gapi.DeleteGoods, http.MethodDelete, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("delete code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// TestApiSkuFlow 测试 SKU 接口的增删改查及启停流程
func TestApiSkuFlow(t *testing.T) {
	goodsApiTestDB(t)
	global.GVA_DB.Create(&jxc.Goods{Code: "SP001", Name: "纯棉T恤", Unit: "件", Status: 1})
	// 创建 SKU
	_, rb := doReq(t, gapi.CreateSku, http.MethodPost, `{"goodsId":1,"skuCode":"SP001-RED-M","color":"红色","size":"M","costPrice":10,"salePrice":20}`)
	if rb.Code != 0 {
		t.Fatalf("create sku code=%d msg=%s", rb.Code, rb.Msg)
	}
	// SKU 列表
	c, w := newCtxQuery(t, "id=1")
	gapi.GetSkuList(c)
	if w.Code != http.StatusOK {
		t.Fatalf("sku list status=%d", w.Code)
	}
	// 更新
	_, rb = doReq(t, gapi.UpdateSku, http.MethodPut, `{"ID":1,"goodsId":1,"skuCode":"SP001-RED-M","color":"红色","size":"M","salePrice":25}`)
	if rb.Code != 0 {
		t.Fatalf("update sku code=%d msg=%s", rb.Code, rb.Msg)
	}
	// 启停
	_, rb = doReq(t, gapi.SetSkuStatus, http.MethodPut, `{"id":1,"status":0}`)
	if rb.Code != 0 {
		t.Fatalf("setSkuStatus code=%d", rb.Code)
	}
	// 删除
	_, rb = doReq(t, gapi.DeleteSku, http.MethodDelete, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("delete sku code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// TestApiGoodsBadBody 测试商品写接口非法 JSON 绑定失败
func TestApiGoodsBadBody(t *testing.T) {
	goodsApiTestDB(t)
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"CreateGoods", gapi.CreateGoods}, {"UpdateGoods", gapi.UpdateGoods},
		{"DeleteGoods", gapi.DeleteGoods}, {"SetGoodsStatus", gapi.SetGoodsStatus},
		{"DeleteGoodsForever", gapi.DeleteGoodsForever},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{bad json`)
			if rb.Code == 0 {
				t.Fatalf("坏 JSON 应返回业务错误码")
			}
		})
	}
}

// TestApiDeleteGoodsForever 测试彻底删除商品接口
func TestApiDeleteGoodsForever(t *testing.T) {
	goodsApiTestDB(t)
	global.GVA_DB.Create(&jxc.Goods{Code: "SP001", Name: "纯棉T恤", Unit: "件", Status: 1})
	_, rb := doReq(t, gapi.DeleteGoodsForever, http.MethodDelete, `{"id":1}`)
	if rb.Code != 0 {
		t.Fatalf("delete forever code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// TestApiSkuBadBody 测试 SKU 写接口非法 JSON 绑定失败
func TestApiSkuBadBody(t *testing.T) {
	goodsApiTestDB(t)
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"CreateSku", gapi.CreateSku}, {"UpdateSku", gapi.UpdateSku},
		{"DeleteSku", gapi.DeleteSku}, {"SetSkuStatus", gapi.SetSkuStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{bad json`)
			if rb.Code == 0 {
				t.Fatalf("坏 JSON 应返回业务错误码")
			}
		})
	}
}

// TestApiGoodsErrors 测试商品接口在数据库异常时的错误分支
func TestApiGoodsErrors(t *testing.T) {
	goodsApiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE goods")
	// GET 类接口（query 参数, 直接走 service 错误分支）
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetGoodsPage", gapi.GetGoodsPage}, {"GetGoodsDetail", gapi.GetGoodsDetail},
		{"GetAllGoods", gapi.GetAllGoods},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, w := newCtxQuery(t, "page=1&pageSize=10&id=1")
			tc.call(c)
			if parseBody(t, w).Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
	// 写类接口（body 参数, 走 service 错误分支）
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"CreateGoods", gapi.CreateGoods}, {"UpdateGoods", gapi.UpdateGoods},
		{"DeleteGoods", gapi.DeleteGoods}, {"SetGoodsStatus", gapi.SetGoodsStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{"id":1,"code":"SP001","name":"x","status":1}`)
			if rb.Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
}

// TestApiSkuErrors 测试 SKU 接口在数据库异常时的错误分支
func TestApiSkuErrors(t *testing.T) {
	goodsApiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE goods_sku")
	// GET 类接口
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetSkuList", gapi.GetSkuList},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, w := newCtxQuery(t, "id=1")
			tc.call(c)
			if parseBody(t, w).Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
	// 写类接口
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"CreateSku", gapi.CreateSku}, {"UpdateSku", gapi.UpdateSku},
		{"DeleteSku", gapi.DeleteSku}, {"SetSkuStatus", gapi.SetSkuStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{"id":1,"goodsId":1,"color":"红","size":"M","status":1}`)
			if rb.Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
}

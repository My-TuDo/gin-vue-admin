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

var posApi2 = &PosScanApi{}

// doReqQuery 带 query 的 GET 请求（doReq 的 URL 固定为 /，无法传 query）
func doReqQuery(t *testing.T, h gin.HandlerFunc, path string) respBody {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	h(c)
	var rb respBody
	if err := json.Unmarshal(w.Body.Bytes(), &rb); err != nil {
		t.Fatalf("响应非 JSON: %s", w.Body.String())
	}
	return rb
}

// TestApiPosScan 扫码枪 API：收银台码 / 上架 / 待处理列表 / 消费确认
func TestApiPosScan(t *testing.T) {
	testutil.NewMemoryDB(t,
		&jxc.GoodsCategory{}, &jxc.Goods{}, &jxc.GoodsSku{},
		&jxc.PosScan{}, &jxc.PosSession{},
	)
	db := global.GVA_DB

	cat := jxc.GoodsCategory{Name: "扫码API分类", Sort: 1, Status: 1}
	db.Create(&cat)
	goods := jxc.Goods{Name: "扫码API商品", CategoryID: &cat.ID, Unit: "件", Status: 1}
	db.Create(&goods)
	sku := jxc.GoodsSku{GoodsID: goods.ID, SkuCode: "API-POS", Barcode: "6901112223334", Color: "红", Size: "S", SalePrice: 15, Status: 1}
	db.Create(&sku)

	// 0. 生成收银台码
	_, rb := doReq(t, posApi2.CreateSession, http.MethodPost, `{"remark":"1号收银机"}`)
	if rb.Code != 0 {
		t.Fatalf("生成会话 code=%d msg=%s", rb.Code, rb.Msg)
	}
	var sess struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rb.Data, &sess); err != nil {
		t.Fatalf("解析会话码失败: %v", err)
	}
	code := sess.Code

	// 0b. 校验有效码 / 无效码
	rb = doReqQuery(t, posApi2.CheckSession, "/?code="+code)
	if rb.Code != 0 {
		t.Fatalf("有效码校验 code=%d msg=%s", rb.Code, rb.Msg)
	}
	rb = doReqQuery(t, posApi2.CheckSession, "/?code=BADCOD")
	if rb.Code == 0 {
		t.Fatal("无效码应校验失败")
	}

	// 1. 上架成功（按条码 + 会话）
	_, rb = doReq(t, posApi2.Scan, http.MethodPost, `{"session":"`+code+`","barcode":"6901112223334","qty":2}`)
	if rb.Code != 0 {
		t.Fatalf("上架 code=%d msg=%s", rb.Code, rb.Msg)
	}
	var scan jxc.PosScan
	global.GVA_DB.Where("sku_id = ? AND status = 0", sku.ID).First(&scan)
	if scan.Qty != 2 || scan.Session != code {
		t.Fatalf("上架记录异常: %+v", scan)
	}

	// 2. 上架失败（条码不存在）
	_, rb = doReq(t, posApi2.Scan, http.MethodPost, `{"session":"`+code+`","barcode":"9999999999999","qty":1}`)
	if rb.Code == 0 {
		t.Fatal("条码不存在应失败")
	}

	// 3. 上架失败（空参数）
	_, rb = doReq(t, posApi2.Scan, http.MethodPost, `{}`)
	if rb.Code == 0 {
		t.Fatal("空参数应失败")
	}

	// 4. 待处理列表（带会话）
	rb = doReqQuery(t, posApi2.ListPending, "/?session="+code)
	if rb.Code != 0 {
		t.Fatalf("列表 code=%d msg=%s", rb.Code, rb.Msg)
	}

	// 5. 确认成功
	_, rb = doReq(t, posApi2.Confirm, http.MethodPut, `{"ids":[1]}`)
	if rb.Code != 0 {
		t.Fatalf("确认 code=%d msg=%s", rb.Code, rb.Msg)
	}

	// 6. 确认空 ids 失败
	_, rb = doReq(t, posApi2.Confirm, http.MethodPut, `{"ids":[]}`)
	if rb.Code == 0 {
		t.Fatal("空 ids 应失败")
	}

	// 7. 确认参数格式错误
	_, rb = doReq(t, posApi2.Confirm, http.MethodPut, `{"bad`)
	if rb.Code == 0 {
		t.Fatal("坏 JSON 应失败")
	}

	// 8. 作废收银台码（重置）
	_, rb = doReq(t, posApi2.DisableSession, http.MethodPut, `{"code":"`+code+`"}`)
	if rb.Code != 0 {
		t.Fatalf("作废会话 code=%d msg=%s", rb.Code, rb.Msg)
	}
	rb = doReqQuery(t, posApi2.CheckSession, "/?code="+code)
	if rb.Code == 0 {
		t.Fatal("作废后校验应失败")
	}
}

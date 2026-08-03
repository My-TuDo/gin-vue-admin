package jxc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// api 被测的 handler 实例
var api = &BasicApi{}

// apiTestDB 内存库 + 5 张 jxc 表 + 赋值 GVA_DB（api handler 依赖全局单例）
func apiTestDB(t *testing.T) {
	t.Helper()
	testutil.NewMemoryDB(t, &jxc.GoodsCategory{}, &jxc.Brand{}, &jxc.Supplier{}, &jxc.Customer{}, &jxc.Warehouse{})
}

// respBody 解出响应 JSON，GVA response 格式: {code, data, msg}
type respBody struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func doReq(t *testing.T, h gin.HandlerFunc, method, body string) (*httptest.ResponseRecorder, respBody) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h(c)
	var rb respBody
	if err := json.Unmarshal(w.Body.Bytes(), &rb); err != nil {
		t.Fatalf("响应非 JSON: %s", w.Body.String())
	}
	return w, rb
}

// ========== 商品分类 ==========

// TestApiGetCategoryTree 测试获取商品分类树接口
func TestApiGetCategoryTree(t *testing.T) {
	apiTestDB(t)
	global.GVA_DB.Create(&jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1})
	_, rb := doReq(t, api.GetCategoryTree, http.MethodGet, "")
	if rb.Code != 0 {
		t.Fatalf("code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// TestApiGetCategoryList 测试获取商品分类列表接口
func TestApiGetCategoryList(t *testing.T) {
	apiTestDB(t)
	global.GVA_DB.Create(&jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=1&pageSize=10", nil)
	api.GetCategoryList(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

// TestApiCreateCategory 测试创建商品分类接口
func TestApiCreateCategory(t *testing.T) {
	apiTestDB(t)
	_, rb := doReq(t, api.CreateCategory, http.MethodPost, `{"code":"C001","name":"男装","sort":1,"status":1}`)
	if rb.Code != 0 {
		t.Fatalf("create code=%d msg=%s", rb.Code, rb.Msg)
	}
	var cnt int64
	global.GVA_DB.Model(&jxc.GoodsCategory{}).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("落库数=%d", cnt)
	}
}

// TestApiCreateCategory_BadBody 测试创建商品分类接口非法 JSON 绑定失败
func TestApiCreateCategory_BadBody(t *testing.T) {
	apiTestDB(t)
	_, rb := doReq(t, api.CreateCategory, http.MethodPost, `{bad json`)
	if rb.Code == 0 {
		t.Fatalf("坏 JSON 应返回业务错误码")
	}
}

// TestApiUpdateCategory 测试更新商品分类接口
func TestApiUpdateCategory(t *testing.T) {
	apiTestDB(t)
	c := jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	global.GVA_DB.Create(&c)
	_, rb := doReq(t, api.UpdateCategory, http.MethodPut,
		`{"ID":`+itoa(c.ID)+`,"code":"C001","name":"男装2","sort":1,"status":1}`)
	if rb.Code != 0 {
		t.Fatalf("update code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// TestApiDeleteCategory 测试删除商品分类接口
func TestApiDeleteCategory(t *testing.T) {
	apiTestDB(t)
	c := jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	global.GVA_DB.Create(&c)
	_, rb := doReq(t, api.DeleteCategory, http.MethodDelete, `{"id":`+itoa(c.ID)+`}`)
	if rb.Code != 0 {
		t.Fatalf("delete code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// TestApiSetCategoryStatus 测试设置商品分类状态接口
func TestApiSetCategoryStatus(t *testing.T) {
	apiTestDB(t)
	c := jxc.GoodsCategory{Code: "C001", Name: "男装", Status: 1}
	global.GVA_DB.Create(&c)
	_, rb := doReq(t, api.SetCategoryStatus, http.MethodPut, `{"id":`+itoa(c.ID)+`,"status":0}`)
	if rb.Code != 0 {
		t.Fatalf("setStatus code=%d msg=%s", rb.Code, rb.Msg)
	}
}

// ========== 品牌 / 供应商 / 客户 / 仓库（同构精简） ==========

// TestApiBrandFlow 测试品牌接口的增删改查及状态切换流程
func TestApiBrandFlow(t *testing.T) {
	apiTestDB(t)
	_, rb := doReq(t, api.CreateBrand, http.MethodPost, `{"code":"B001","name":"耐克","status":1}`)
	if rb.Code != 0 {
		t.Fatalf("create code=%d", rb.Code)
	}
	var b jxc.Brand
	global.GVA_DB.First(&b)
	doReq(t, api.UpdateBrand, http.MethodPut, `{"ID":`+itoa(b.ID)+`,"code":"B001","name":"耐克2","status":1}`)
	_, rb = doReq(t, api.GetAllBrands, http.MethodGet, "")
	if rb.Code != 0 {
		t.Fatalf("getAll code=%d", rb.Code)
	}
	doReq(t, api.SetBrandStatus, http.MethodPut, `{"id":`+itoa(b.ID)+`,"status":0}`)
	_, rb = doReq(t, api.DeleteBrand, http.MethodDelete, `{"id":`+itoa(b.ID)+`}`)
	if rb.Code != 0 {
		t.Fatalf("delete code=%d", rb.Code)
	}
}

// TestApiSupplierFlow 测试供应商接口的增删改查及状态切换流程
func TestApiSupplierFlow(t *testing.T) {
	apiTestDB(t)
	_, rb := doReq(t, api.CreateSupplier, http.MethodPost, `{"code":"S001","name":"广州布行","status":1}`)
	if rb.Code != 0 {
		t.Fatalf("create code=%d", rb.Code)
	}
	var s jxc.Supplier
	global.GVA_DB.First(&s)
	doReq(t, api.UpdateSupplier, http.MethodPut, `{"ID":`+itoa(s.ID)+`,"code":"S001","name":"广州布行2","status":1}`)
	doReq(t, api.SetSupplierStatus, http.MethodPut, `{"id":`+itoa(s.ID)+`,"status":0}`)
	_, rb = doReq(t, api.DeleteSupplier, http.MethodDelete, `{"id":`+itoa(s.ID)+`}`)
	if rb.Code != 0 {
		t.Fatalf("delete code=%d", rb.Code)
	}
}

// TestApiCustomerFlow 测试客户接口的增删改查及状态切换流程
func TestApiCustomerFlow(t *testing.T) {
	apiTestDB(t)
	_, rb := doReq(t, api.CreateCustomer, http.MethodPost, `{"code":"K001","name":"张三","level":1,"status":1}`)
	if rb.Code != 0 {
		t.Fatalf("create code=%d", rb.Code)
	}
	var c jxc.Customer
	global.GVA_DB.First(&c)
	doReq(t, api.UpdateCustomer, http.MethodPut, `{"ID":`+itoa(c.ID)+`,"code":"K001","name":"张三2","level":2,"status":1}`)
	doReq(t, api.SetCustomerStatus, http.MethodPut, `{"id":`+itoa(c.ID)+`,"status":0}`)
	_, rb = doReq(t, api.DeleteCustomer, http.MethodDelete, `{"id":`+itoa(c.ID)+`}`)
	if rb.Code != 0 {
		t.Fatalf("delete code=%d", rb.Code)
	}
}

// TestApiWarehouseFlow 测试仓库接口的增删改查及状态切换流程
func TestApiWarehouseFlow(t *testing.T) {
	apiTestDB(t)
	_, rb := doReq(t, api.CreateWarehouse, http.MethodPost, `{"code":"W001","name":"总仓","status":1}`)
	if rb.Code != 0 {
		t.Fatalf("create code=%d", rb.Code)
	}
	var w jxc.Warehouse
	global.GVA_DB.First(&w)
	doReq(t, api.UpdateWarehouse, http.MethodPut, `{"ID":`+itoa(w.ID)+`,"code":"W001","name":"总仓2","status":1}`)
	doReq(t, api.SetWarehouseStatus, http.MethodPut, `{"id":`+itoa(w.ID)+`,"status":0}`)
	_, rb = doReq(t, api.DeleteWarehouse, http.MethodDelete, `{"id":`+itoa(w.ID)+`}`)
	if rb.Code != 0 {
		t.Fatalf("delete code=%d", rb.Code)
	}
}

// ========== 所有写接口的坏 JSON 绑定失败分支 ==========

// TestApiBindFailures 测试所有写接口非法 JSON 绑定失败分支
func TestApiBindFailures(t *testing.T) {
	apiTestDB(t)
	cases := []struct {
		name string
		call func(c *gin.Context)
	}{
		{"CreateCategory", api.CreateCategory}, {"UpdateCategory", api.UpdateCategory},
		{"DeleteCategory", api.DeleteCategory}, {"SetCategoryStatus", api.SetCategoryStatus},
		{"CreateBrand", api.CreateBrand}, {"UpdateBrand", api.UpdateBrand},
		{"DeleteBrand", api.DeleteBrand}, {"SetBrandStatus", api.SetBrandStatus},
		{"CreateSupplier", api.CreateSupplier}, {"UpdateSupplier", api.UpdateSupplier},
		{"DeleteSupplier", api.DeleteSupplier}, {"SetSupplierStatus", api.SetSupplierStatus},
		{"CreateCustomer", api.CreateCustomer}, {"UpdateCustomer", api.UpdateCustomer},
		{"DeleteCustomer", api.DeleteCustomer}, {"SetCustomerStatus", api.SetCustomerStatus},
		{"CreateWarehouse", api.CreateWarehouse}, {"UpdateWarehouse", api.UpdateWarehouse},
		{"DeleteWarehouse", api.DeleteWarehouse}, {"SetWarehouseStatus", api.SetWarehouseStatus},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{bad json`)
			if rb.Code == 0 {
				t.Fatalf("坏 JSON 应返回业务错误码")
			}
		})
	}
}

// ========== 接口数据库异常错误分支 ==========

// TestApiCategoryErrors 测试商品分类接口在数据库异常时的错误分支
func TestApiCategoryErrors(t *testing.T) {
	apiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE goods_category")
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetCategoryTree", api.GetCategoryTree},
		{"GetCategoryList", api.GetCategoryList},
		{"CreateCategory", api.CreateCategory},
		{"UpdateCategory", api.UpdateCategory},
		{"DeleteCategory", api.DeleteCategory},
		{"SetCategoryStatus", api.SetCategoryStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{"id":1,"code":"C001","name":"x","status":1}`)
			if rb.Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
}

// TestApiBrandErrors 测试品牌接口在数据库异常时的错误分支
func TestApiBrandErrors(t *testing.T) {
	apiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE brand")
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetBrandList", api.GetBrandList}, {"GetAllBrands", api.GetAllBrands},
		{"CreateBrand", api.CreateBrand}, {"UpdateBrand", api.UpdateBrand},
		{"DeleteBrand", api.DeleteBrand}, {"SetBrandStatus", api.SetBrandStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{"id":1,"code":"B001","name":"x","status":1}`)
			if rb.Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
}

// TestApiSupplierErrors 测试供应商接口在数据库异常时的错误分支
func TestApiSupplierErrors(t *testing.T) {
	apiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE supplier")
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetSupplierList", api.GetSupplierList}, {"GetAllSuppliers", api.GetAllSuppliers},
		{"CreateSupplier", api.CreateSupplier}, {"UpdateSupplier", api.UpdateSupplier},
		{"DeleteSupplier", api.DeleteSupplier}, {"SetSupplierStatus", api.SetSupplierStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{"id":1,"code":"S001","name":"x","status":1}`)
			if rb.Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
}

// TestApiCustomerErrors 测试客户接口在数据库异常时的错误分支
func TestApiCustomerErrors(t *testing.T) {
	apiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE customer")
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetCustomerList", api.GetCustomerList}, {"GetAllCustomers", api.GetAllCustomers},
		{"CreateCustomer", api.CreateCustomer}, {"UpdateCustomer", api.UpdateCustomer},
		{"DeleteCustomer", api.DeleteCustomer}, {"SetCustomerStatus", api.SetCustomerStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{"id":1,"code":"K001","name":"x","level":1,"status":1}`)
			if rb.Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
}

// TestApiWarehouseErrors 测试仓库接口在数据库异常时的错误分支
func TestApiWarehouseErrors(t *testing.T) {
	apiTestDB(t)
	global.GVA_DB.Exec("DROP TABLE warehouse")
	for _, tc := range []struct {
		name string
		call func(c *gin.Context)
	}{
		{"GetWarehouseList", api.GetWarehouseList}, {"GetAllWarehouses", api.GetAllWarehouses},
		{"CreateWarehouse", api.CreateWarehouse}, {"UpdateWarehouse", api.UpdateWarehouse},
		{"DeleteWarehouse", api.DeleteWarehouse}, {"SetWarehouseStatus", api.SetWarehouseStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, rb := doReq(t, tc.call, http.MethodPost, `{"id":1,"code":"W001","name":"x","status":1}`)
			if rb.Code == 0 {
				t.Fatalf("%s 应返回业务错误码", tc.name)
			}
		})
	}
}

// itoa 将 uint 转为十进制字符串（拼接 JSON 主键用）
func itoa(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}

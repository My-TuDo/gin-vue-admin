package jxc

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// PosScanService 扫码枪队列服务
type PosScanService struct{}

const sessionCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 去掉易混淆的 I/O/0/1

// CreateSession 生成新的收银台码（PC 收银台设置页）
func (s *PosScanService) CreateSession(ctx context.Context, remark string) (*jxc.PosSession, error) {
	db := global.GVA_DB.WithContext(ctx)
	for i := 0; i < 10; i++ {
		code := randomCode(6)
		// 唯一碰撞重试
		var cnt int64
		if err := db.Model(&jxc.PosSession{}).Where("code = ?", code).Count(&cnt).Error; err != nil {
			return nil, err
		}
		if cnt == 0 {
			ss := jxc.PosSession{Code: code, Remark: strings.TrimSpace(remark), Status: 1}
			if err := db.Create(&ss).Error; err != nil {
				return nil, err
			}
			return &ss, nil
		}
	}
	return nil, errors.New("生成收银台码失败，请重试")
}

// CheckSession 校验收银台码是否有效（小程序绑定用）
func (s *PosScanService) CheckSession(ctx context.Context, code string) (*jxc.PosSession, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("请输入收银台码")
	}
	var ss jxc.PosSession
	err := global.GVA_DB.WithContext(ctx).Where("code = ? AND status = 1", code).First(&ss).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("收银台码无效，请核对后在 PC 收银台设置中获取")
		}
		return nil, err
	}
	return &ss, nil
}

// DisableSession 作废收银台码（PC 重置：旧码立即失效，小程序需重新绑定）
func (s *PosScanService) DisableSession(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return errors.New("收银台码不能为空")
	}
	return global.GVA_DB.WithContext(ctx).Model(&jxc.PosSession{}).
		Where("code = ? AND status = 1", code).
		Updates(map[string]interface{}{"status": 0}).Error
}

// CreateScan 小程序扫码上架：按条码/SKU编码 查 SKU，投递到指定收银台会话。
// 同一 SKU 在同一会话已有未消费记录时数量累加（连续扫同款合并）。
func (s *PosScanService) CreateScan(ctx context.Context, session, barcode string, skuID uint, qty int) (*jxc.PosScan, error) {
	if qty <= 0 {
		qty = 1
	}
	if qty > 99 {
		return nil, errors.New("单次扫码数量不能超过 99")
	}
	// 会话必须有效
	if _, err := s.CheckSession(ctx, session); err != nil {
		return nil, err
	}
	db := global.GVA_DB.WithContext(ctx)

	var sku jxc.GoodsSku
	if skuID > 0 {
		if err := db.First(&sku, skuID).Error; err != nil {
			return nil, errors.New("SKU 不存在")
		}
	} else {
		keyword := strings.TrimSpace(barcode)
		if keyword == "" {
			return nil, errors.New("请提供条码或 SKU 编码")
		}
		if err := db.Where("barcode = ? OR sku_code = ?", keyword, keyword).First(&sku).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("未找到该条码对应的商品")
			}
			return nil, err
		}
	}

	// 同会话同 SKU 未消费记录数量累加
	session = strings.TrimSpace(session)
	var scan jxc.PosScan
	err := db.Where("session = ? AND sku_id = ? AND status = 0", session, sku.ID).First(&scan).Error
	if err == nil {
		scan.Qty += qty
		if err := db.Save(&scan).Error; err != nil {
			return nil, err
		}
		return &scan, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	scan = jxc.PosScan{Session: session, SkuID: sku.ID, Qty: qty, Status: 0}
	if err := db.Create(&scan).Error; err != nil {
		return nil, err
	}
	return &scan, nil
}

// ListPending 指定收银台会话的未消费扫码条目（PC 轮询），附带 SKU 摘要
func (s *PosScanService) ListPending(ctx context.Context, session string) ([]jxc.PosScanDetail, error) {
	session = strings.TrimSpace(session)
	if session == "" {
		return nil, errors.New("收银台码不能为空")
	}
	// 会话必须有效（作废/不存在 → 前端提示重新生成绑定）
	if _, err := s.CheckSession(ctx, session); err != nil {
		return nil, err
	}
	db := global.GVA_DB.WithContext(ctx)
	var list []jxc.PosScan
	if err := db.Where("session = ? AND status = 0", session).Order("created_at ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	details := make([]jxc.PosScanDetail, 0, len(list))
	for i := range list {
		d := jxc.PosScanDetail{PosScan: list[i]}
		var sku jxc.GoodsSku
		if err := db.First(&sku, list[i].SkuID).Error; err == nil {
			var goods jxc.Goods
			goodsName := ""
			if err := db.First(&goods, sku.GoodsID).Error; err == nil {
				goodsName = goods.Name
			}
			d.Sku = &jxc.SkuBrief{
				ID:        sku.ID,
				SkuCode:   sku.SkuCode,
				GoodsName: goodsName,
				Color:     sku.Color,
				Size:      sku.Size,
				SalePrice: sku.SalePrice,
				Barcode:   sku.Barcode,
			}
		}
		details = append(details, d)
	}
	return details, nil
}

// ConfirmScan PC 收银台消费确认：批量标记已处理（自动加入购物车后）
func (s *PosScanService) ConfirmScan(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return errors.New("请选择要确认的扫码条目")
	}
	now := time.Now()
	return global.GVA_DB.WithContext(ctx).Model(&jxc.PosScan{}).
		Where("id IN ? AND status = 0", ids).
		Updates(map[string]interface{}{"status": 1, "handled_at": &now}).Error
}

// randomCode 生成 n 位不混淆字符收银台码
func randomCode(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = sessionCodeChars[r.Intn(len(sessionCodeChars))]
	}
	return string(b)
}

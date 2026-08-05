package jxc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// StockCheckService 盘点中心业务
type StockCheckService struct{}

// ========== 辅助 ==========

// genCheckNo 生成盘点单号 CK-YYYYMMDD-XXX（Unscoped 含软删行，避免删除后重建撞唯一索引）
func (s *StockCheckService) genCheckNo(db *gorm.DB) (string, error) {
	date := time.Now().Format("20060102")
	like := "CK-" + date + "%"
	var maxNo string
	if err := db.Unscoped().Model(&jxc.StockCheck{}).Where("check_no LIKE ?", like).
		Order("check_no DESC").Limit(1).Pluck("check_no", &maxNo).Error; err != nil {
		return "", err
	}
	if maxNo == "" {
		return "CK-" + date + "-001", nil
	}
	seq, err := strconv.Atoi(maxNo[len(maxNo)-3:])
	if err != nil {
		return "", fmt.Errorf("解析盘点单号序号失败: %s", maxNo)
	}
	return fmt.Sprintf("CK-%s-%03d", date, seq+1), nil
}

// ========== 盘点单 CRUD ==========

// CreateStockCheck 创建盘点单（盘点中）：快照仓库当前有库存的 SKU 明细
func (s *StockCheckService) CreateStockCheck(ctx context.Context, check *jxc.StockCheck) error {
	if check.WarehouseID == 0 {
		return errors.New("请选择盘点仓库")
	}
	var wh jxc.Warehouse
	if err := global.GVA_DB.WithContext(ctx).First(&wh, check.WarehouseID).Error; err != nil {
		return errors.New("盘点仓库不存在")
	}
	check.Status = jxc.StockCheckStatusChecking
	for attempt := 0; attempt < 3; attempt++ {
		err := s.createStockCheckTx(ctx, check)
		if err == nil {
			return nil
		}
		if !isDuplicateKey(err) {
			return err
		}
		check.CheckNo = ""
		check.ID = 0
		check.Items = nil
	}
	return errors.New("单号生成冲突，请重试")
}

func (s *StockCheckService) createStockCheckTx(ctx context.Context, check *jxc.StockCheck) error {
	db := global.GVA_DB.WithContext(ctx)
	checkNo, err := s.genCheckNo(db)
	if err != nil {
		return err
	}
	check.CheckNo = checkNo
	return db.Transaction(func(tx *gorm.DB) error {
		// 快照该仓库所有有库存的 SKU（system_qty = 当前库存）
		var stocks []jxc.Stock
		if err := tx.Where("warehouse_id = ? AND quantity > 0", check.WarehouseID).Find(&stocks).Error; err != nil {
			return err
		}
		items := make([]jxc.StockCheckItem, 0, len(stocks))
		for _, st := range stocks {
			items = append(items, jxc.StockCheckItem{
				SkuID:     st.SkuID,
				SystemQty: st.Quantity,
				DiffQty:   0,
			})
		}
		check.Items = items
		return tx.Create(check).Error
	})
}

// UpdateStockCheckItems 录入盘点数（仅盘点中）
func (s *StockCheckService) UpdateStockCheckItems(ctx context.Context, checkID uint, items []jxc.StockCheckItem) error {
	if len(items) == 0 {
		return errors.New("请至少录入一条盘点数")
	}
	db := global.GVA_DB.WithContext(ctx)
	var check jxc.StockCheck
	if err := db.First(&check, checkID).Error; err != nil {
		return errors.New("盘点单不存在")
	}
	if check.Status != jxc.StockCheckStatusChecking {
		return errors.New("仅盘点中的盘点单可以录入")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, it := range items {
			if it.ID == 0 {
				return errors.New("盘点明细ID缺失")
			}
			if it.ActualQty == nil || *it.ActualQty < 0 {
				return errors.New("实盘数量不能为负数")
			}
			res := tx.Model(&jxc.StockCheckItem{}).
				Where("id = ? AND check_id = ?", it.ID, checkID).
				Update("actual_qty", *it.ActualQty)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("盘点明细不存在")
			}
		}
		return nil
	})
}

// CompleteStockCheck 完成盘点：差异对比 → 调整库存 → 生成流水 → 已完成
func (s *StockCheckService) CompleteStockCheck(ctx context.Context, id uint, operator string) error {
	db := global.GVA_DB.WithContext(ctx)
	var check jxc.StockCheck
	if err := db.First(&check, id).Error; err != nil {
		return errors.New("盘点单不存在")
	}
	if check.Status != jxc.StockCheckStatusChecking {
		return errors.New("仅盘点中的盘点单可以完成")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var items []jxc.StockCheckItem
		if err := tx.Where("check_id = ?", id).Find(&items).Error; err != nil {
			return err
		}
		for i := range items {
			it := &items[i]
			// 未录入的明细按账存视为相符
			actual := it.SystemQty
			if it.ActualQty != nil {
				actual = *it.ActualQty
			}
			diff := actual - it.SystemQty
			it.DiffQty = diff
			if diff != 0 {
				// 调整库存（盘盈增加/盘亏扣减，直接覆盖为实盘数）
				var stock jxc.Stock
				err := tx.Where("warehouse_id = ? AND sku_id = ?", check.WarehouseID, it.SkuID).First(&stock).Error
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				beforeQty := stock.Quantity
				businessType := "check_in"
				if diff < 0 {
					businessType = "check_out"
				}
				if err == gorm.ErrRecordNotFound {
					stock = jxc.Stock{WarehouseID: check.WarehouseID, SkuID: it.SkuID, Quantity: actual}
					if err := tx.Create(&stock).Error; err != nil {
						return err
					}
					beforeQty = 0
				} else {
					if err := tx.Model(&stock).Update("quantity", actual).Error; err != nil {
						return err
					}
				}
				log := jxc.StockLog{
					WarehouseID: check.WarehouseID, SkuID: it.SkuID,
					BusinessType: businessType, BusinessNo: check.CheckNo,
					BeforeQty: beforeQty, ChangeQty: diff, AfterQty: actual,
					Operator: operator,
				}
				if err := tx.Create(&log).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&jxc.StockCheckItem{}).Where("id = ?", it.ID).
				Updates(map[string]interface{}{"actual_qty": actual, "diff_qty": diff}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&jxc.StockCheck{}).Where("id = ?", id).
			Update("status", jxc.StockCheckStatusDone).Error
	})
}

// CancelStockCheck 取消盘点单（仅盘点中）
func (s *StockCheckService) CancelStockCheck(ctx context.Context, id uint) error {
	db := global.GVA_DB.WithContext(ctx)
	var check jxc.StockCheck
	if err := db.First(&check, id).Error; err != nil {
		return errors.New("盘点单不存在")
	}
	if check.Status != jxc.StockCheckStatusChecking {
		return errors.New("仅盘点中的盘点单可以取消")
	}
	return db.Model(&jxc.StockCheck{}).Where("id = ?", id).
		Update("status", jxc.StockCheckStatusCanceled).Error
}

// GetStockCheckPage 分页查询盘点单
func (s *StockCheckService) GetStockCheckPage(ctx context.Context, info request.PageInfo) (list []jxc.StockCheck, total int64, err error) {
	db := global.GVA_DB.WithContext(ctx).Model(&jxc.StockCheck{})
	if info.Keyword != "" {
		db = db.Where("check_no LIKE ?", "%"+info.Keyword+"%")
	}
	if err = db.Count(&total).Error; err != nil || total == 0 {
		return
	}
	limit, offset := info.LimitOffset()
	err = db.Preload("Warehouse").
		Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return
}

// GetStockCheckDetail 查询盘点单详情（含明细）
func (s *StockCheckService) GetStockCheckDetail(ctx context.Context, id uint) (check jxc.StockCheck, err error) {
	err = global.GVA_DB.WithContext(ctx).
		Preload("Warehouse").Preload("Items.Sku.Goods").
		First(&check, id).Error
	return
}

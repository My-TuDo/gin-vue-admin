package jxc

import (
	"context"
	"errors"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// CashierService 收银台业务（收款即出库）
type CashierService struct{}

// Checkout 收银结算：一个事务内生成销售单（已出库）+ 扣减库存 + 写流水
func (s *CashierService) Checkout(ctx context.Context, req jxc.POSCheckoutReq, operator string) (*jxc.SaleOrder, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("请先添加商品")
	}
	for _, it := range req.Items {
		if it.SkuID == 0 {
			return nil, errors.New("商品缺失")
		}
		if it.Qty <= 0 {
			return nil, errors.New("商品数量必须大于 0")
		}
	}
	if req.WarehouseID == 0 {
		return nil, errors.New("请选择收银仓库")
	}
	switch req.PayMethod {
	case "cash", "wechat", "alipay":
	default:
		return nil, errors.New("支付方式不合法")
	}
	db := global.GVA_DB.WithContext(ctx)
	var wh jxc.Warehouse
	if err := db.First(&wh, req.WarehouseID).Error; err != nil {
		return nil, errors.New("收银仓库不存在")
	}

	saleSvc := &SaleService{}
	var order *jxc.SaleOrder
	for attempt := 0; attempt < 3; attempt++ {
		var err error
		order, err = s.checkoutTx(ctx, req, operator, saleSvc)
		if err == nil {
			return order, nil
		}
		if !isDuplicateKey(err) {
			return nil, err
		}
	}
	return nil, errors.New("单号生成冲突，请重试")
}

func (s *CashierService) checkoutTx(ctx context.Context, req jxc.POSCheckoutReq, operator string, saleSvc *SaleService) (*jxc.SaleOrder, error) {
	db := global.GVA_DB.WithContext(ctx)
	order := &jxc.SaleOrder{
		WarehouseID: req.WarehouseID,
		CustomerID:  req.CustomerID,
		OrderType:   jxc.SaleTypeNormal,
		Status:      jxc.SaleStatusShipped, // 收银即出库
		Creator:     operator,
		Remark:      req.Remark,
	}
	items := make([]jxc.SaleItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, jxc.SaleItem{SkuID: it.SkuID, Qty: it.Qty})
	}
	order.Items = items
	err := db.Transaction(func(tx *gorm.DB) error {
		orderNo, err := saleSvc.genSaleOrderNo(tx)
		if err != nil {
			return err
		}
		order.OrderNo = orderNo
		total, err := saleSvc.fillItemSnapshot(tx, jxc.SaleTypeNormal, order.Items)
		if err != nil {
			return err
		}
		order.TotalAmount = total
		// 实收校验（现金要求实收 ≥ 应付；微信/支付宝实收=应付）
		if req.PayMethod == "cash" && req.PaidAmount < total {
			return fmt.Errorf("实收金额不足：应付 %.2f，实收 %.2f", total, req.PaidAmount)
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		// 收款即出库：校验可售库存并扣减 + 流水
		for i := range order.Items {
			it := &order.Items[i]
			var stock jxc.Stock
			if err := tx.Where("warehouse_id = ? AND sku_id = ?", order.WarehouseID, it.SkuID).First(&stock).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("库存不足：SKU %s 无库存记录", it.SkuCode)
				}
				return err
			}
			available := stock.Quantity - stock.LockQuantity
			if available < it.Qty {
				return fmt.Errorf("库存不足：SKU %s 可售 %d，需要 %d", it.SkuCode, available, it.Qty)
			}
			if err := tx.Model(&stock).Update("quantity", stock.Quantity-it.Qty).Error; err != nil {
				return err
			}
			log := jxc.StockLog{
				WarehouseID: order.WarehouseID, SkuID: it.SkuID,
				BusinessType: "sale_out", BusinessNo: order.OrderNo,
				BeforeQty: stock.Quantity, ChangeQty: -it.Qty, AfterQty: stock.Quantity - it.Qty,
				Operator: operator,
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

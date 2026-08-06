package jxc

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
	"gorm.io/gorm"
)

// CashierService 收银台业务（收款/退款/换货，均即时完成）
type CashierService struct{}

// RefundableOrder 可退换原单（含剩余件数）
type RefundableOrder struct {
	ID          uint    `json:"ID"`
	OrderNo     string  `json:"orderNo"`
	OrderType   int8    `json:"orderType"`
	TotalAmount float64 `json:"totalAmount"`
	OutQty      int     `json:"outQty"`
	UsedQty     int     `json:"usedQty"`
	Remaining   int     `json:"remaining"`
}

// RefundableOrders 可退换原单列表：已出库的正常销售/换货单，剩余件数 > 0
func (s *CashierService) RefundableOrders(ctx context.Context) ([]RefundableOrder, error) {
	db := global.GVA_DB.WithContext(ctx)
	// 每张已出库原单的出库件数（正常销售=全部明细；换货=换出明细）
	type qtyRow struct {
		ID  uint
		Qty int
	}
	var outs []qtyRow
	if err := db.Table("sale_item si").
		Select("o.id AS id, SUM(si.qty) AS qty").
		Joins("JOIN sale_order o ON o.id = si.sale_id").
		Where("o.status = ? AND (o.order_type = ? OR si.direction = ?)", jxc.SaleStatusShipped, jxc.SaleTypeNormal, 1).
		Group("o.id").Scan(&outs).Error; err != nil {
		return nil, err
	}
	// 每张原单已被确认单据占用的件数
	var useds []qtyRow
	if err := db.Table("sale_item i").
		Select("o.original_order_id AS id, SUM(i.qty) AS qty").
		Joins("JOIN sale_order o ON o.id = i.sale_id").
		Where("o.status = ? AND o.original_order_id IS NOT NULL AND i.direction IN (0, 2)", jxc.SaleStatusShipped).
		Group("o.original_order_id").Scan(&useds).Error; err != nil {
		return nil, err
	}
	usedMap := map[uint]int{}
	for _, u := range useds {
		usedMap[u.ID] += u.Qty
	}
	outMap := map[uint]int{}
	for _, o := range outs {
		outMap[o.ID] = o.Qty
	}
	// 原单主信息（已出库正常销售/换货）
	var orders []jxc.SaleOrder
	if err := db.Where("status = ? AND order_type IN (?, ?)", jxc.SaleStatusShipped, jxc.SaleTypeNormal, jxc.SaleTypeExchange).
		Find(&orders).Error; err != nil {
		return nil, err
	}
	result := make([]RefundableOrder, 0, len(orders))
	for _, o := range orders {
		outQty := outMap[o.ID]
		usedQty := usedMap[o.ID]
		remaining := outQty - usedQty
		if remaining <= 0 {
			continue // 已退/换完的单不再出现
		}
		result = append(result, RefundableOrder{
			ID: o.ID, OrderNo: o.OrderNo, OrderType: o.OrderType,
			TotalAmount: o.TotalAmount, OutQty: outQty, UsedQty: usedQty, Remaining: remaining,
		})
	}
	// 按剩余件数降序，单号升序
	sort.Slice(result, func(a, b int) bool {
		if result[a].Remaining != result[b].Remaining {
			return result[a].Remaining > result[b].Remaining
		}
		return result[a].OrderNo < result[b].OrderNo
	})
	return result, nil
}

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
			if err := s.outStockTx(tx, order, it, operator); err != nil {
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

// outStockTx 扣减可售库存并写 sale_out 流水（须在事务内调用）
func (s *CashierService) outStockTx(tx *gorm.DB, order *jxc.SaleOrder, it *jxc.SaleItem, operator string) error {
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
	return tx.Create(&log).Error
}

// ========== 退款 ==========

// Refund 收银退款：生成退货单（关联原单）并直接确认入库，退回金额 = -TotalAmount
func (s *CashierService) Refund(ctx context.Context, req jxc.POSRefundReq, operator string) (*jxc.SaleOrder, error) {
	if req.OriginalOrderID == 0 {
		return nil, errors.New("请选择原销售单")
	}
	if len(req.Items) == 0 {
		return nil, errors.New("请选择退款商品")
	}
	for _, it := range req.Items {
		if it.SkuID == 0 {
			return nil, errors.New("退款商品缺失")
		}
		if it.Qty <= 0 {
			return nil, errors.New("退款数量必须大于 0")
		}
	}
	saleSvc := &SaleService{}
	var ret *jxc.SaleOrder
	for attempt := 0; attempt < 3; attempt++ {
		var err error
		ret, err = s.refundTx(ctx, req, operator, saleSvc)
		if err == nil {
			return ret, nil
		}
		if !isDuplicateKey(err) {
			return nil, err
		}
	}
	return nil, errors.New("单号生成冲突，请重试")
}

func (s *CashierService) refundTx(ctx context.Context, req jxc.POSRefundReq, operator string, saleSvc *SaleService) (*jxc.SaleOrder, error) {
	db := global.GVA_DB.WithContext(ctx)
	var result *jxc.SaleOrder
	err := db.Transaction(func(tx *gorm.DB) error {
		var original jxc.SaleOrder
		if err := tx.First(&original, req.OriginalOrderID).Error; err != nil {
			return errors.New("关联原单不存在")
		}
		if original.Status != jxc.SaleStatusShipped {
			return errors.New("仅已出库的销售单可以退款")
		}
		if original.OrderType != jxc.SaleTypeNormal && original.OrderType != jxc.SaleTypeExchange {
			return errors.New("仅正常销售或换货单可以退款")
		}
		ret := &jxc.SaleOrder{
			WarehouseID: original.WarehouseID, OriginalOrderID: &original.ID,
			OrderType: jxc.SaleTypeReturn, Status: jxc.SaleStatusPending,
			Creator: operator, Remark: req.Remark,
		}
		items := make([]jxc.SaleItem, 0, len(req.Items))
		for _, it := range req.Items {
			items = append(items, jxc.SaleItem{SkuID: it.SkuID, Qty: it.Qty})
		}
		ret.Items = items
		orderNo, err := saleSvc.genSaleOrderNo(tx)
		if err != nil {
			return err
		}
		ret.OrderNo = orderNo
		total, err := saleSvc.fillItemSnapshot(tx, jxc.SaleTypeReturn, ret.Items)
		if err != nil {
			return err
		}
		ret.TotalAmount = total
		if err := saleSvc.checkReturnSkus(tx, jxc.SaleTypeReturn, original.ID, ret.Items); err != nil {
			return err
		}
		if err := saleSvc.checkInboundLimit(tx, jxc.SaleTypeReturn, original.ID, ret.Items, 0); err != nil {
			return err
		}
		if err := tx.Create(ret).Error; err != nil {
			return err
		}
		// 直接确认退货入库（回补库存 + sale_return 流水）
		if err := saleSvc.confirmReturnTx(tx, ret.ID, operator); err != nil {
			return err
		}
		ret.Status = jxc.SaleStatusShipped // 回填（confirmReturnTx 更新的是 DB 行）
		result = ret
		return nil
	})
	return result, err
}

// ========== 换货 ==========

// Exchange 收银换货：退回原单商品（退货单直接入库）+ 换出商品（销售单直接出库），差额多退少补
func (s *CashierService) Exchange(ctx context.Context, req jxc.POSExchangeReq, operator string) (*jxc.POSExchangeResult, error) {
	if req.OriginalOrderID == 0 {
		return nil, errors.New("请选择原销售单")
	}
	if len(req.ReturnItems) == 0 {
		return nil, errors.New("请选择退回商品")
	}
	if len(req.OutItems) == 0 {
		return nil, errors.New("请选择换出商品")
	}
	for _, it := range append(append([]jxc.POSItem{}, req.ReturnItems...), req.OutItems...) {
		if it.SkuID == 0 {
			return nil, errors.New("商品缺失")
		}
		if it.Qty <= 0 {
			return nil, errors.New("商品数量必须大于 0")
		}
	}
	switch req.PayMethod {
	case "cash", "wechat", "alipay":
	default:
		return nil, errors.New("支付方式不合法")
	}
	saleSvc := &SaleService{}
	var result *jxc.POSExchangeResult
	for attempt := 0; attempt < 3; attempt++ {
		var err error
		result, err = s.exchangeTx(ctx, req, operator, saleSvc)
		if err == nil {
			return result, nil
		}
		if !isDuplicateKey(err) {
			return nil, err
		}
	}
	return nil, errors.New("单号生成冲突，请重试")
}

func (s *CashierService) exchangeTx(ctx context.Context, req jxc.POSExchangeReq, operator string, saleSvc *SaleService) (*jxc.POSExchangeResult, error) {
	db := global.GVA_DB.WithContext(ctx)
	var result *jxc.POSExchangeResult
	err := db.Transaction(func(tx *gorm.DB) error {
		var original jxc.SaleOrder
		if err := tx.First(&original, req.OriginalOrderID).Error; err != nil {
			return errors.New("关联原单不存在")
		}
		if original.Status != jxc.SaleStatusShipped {
			return errors.New("仅已出库的销售单可以换货")
		}
		if original.OrderType != jxc.SaleTypeNormal && original.OrderType != jxc.SaleTypeExchange {
			return errors.New("仅正常销售或换货单可以换货")
		}
		if req.WarehouseID == 0 {
			req.WarehouseID = original.WarehouseID
		}
		// 1. 退货单：退回商品直接入库
		ret := &jxc.SaleOrder{
			WarehouseID: original.WarehouseID, OriginalOrderID: &original.ID,
			OrderType: jxc.SaleTypeReturn, Status: jxc.SaleStatusPending,
			Creator: operator, Remark: req.Remark,
		}
		retItems := make([]jxc.SaleItem, 0, len(req.ReturnItems))
		for _, it := range req.ReturnItems {
			retItems = append(retItems, jxc.SaleItem{SkuID: it.SkuID, Qty: it.Qty})
		}
		ret.Items = retItems
		retNo, err := saleSvc.genSaleOrderNo(tx)
		if err != nil {
			return err
		}
		ret.OrderNo = retNo
		retTotal, err := saleSvc.fillItemSnapshot(tx, jxc.SaleTypeReturn, ret.Items)
		if err != nil {
			return err
		}
		ret.TotalAmount = retTotal
		if err := saleSvc.checkReturnSkus(tx, jxc.SaleTypeReturn, original.ID, ret.Items); err != nil {
			return err
		}
		if err := saleSvc.checkInboundLimit(tx, jxc.SaleTypeReturn, original.ID, ret.Items, 0); err != nil {
			return err
		}
		if err := tx.Create(ret).Error; err != nil {
			return err
		}
		if err := saleSvc.confirmReturnTx(tx, ret.ID, operator); err != nil {
			return err
		}
		ret.Status = jxc.SaleStatusShipped // 回填（confirmReturnTx 更新的是 DB 行）
		// 2. 销售单：换出商品直接出库
		out := &jxc.SaleOrder{
			WarehouseID: req.WarehouseID, CustomerID: req.CustomerID,
			OrderType: jxc.SaleTypeNormal, Status: jxc.SaleStatusShipped,
			Creator: operator, Remark: req.Remark,
		}
		outItems := make([]jxc.SaleItem, 0, len(req.OutItems))
		for _, it := range req.OutItems {
			outItems = append(outItems, jxc.SaleItem{SkuID: it.SkuID, Qty: it.Qty})
		}
		out.Items = outItems
		outNo, err := saleSvc.genSaleOrderNo(tx)
		if err != nil {
			return err
		}
		out.OrderNo = outNo
		outTotal, err := saleSvc.fillItemSnapshot(tx, jxc.SaleTypeNormal, out.Items)
		if err != nil {
			return err
		}
		out.TotalAmount = outTotal
		if err := tx.Create(out).Error; err != nil {
			return err
		}
		for i := range out.Items {
			if err := s.outStockTx(tx, out, &out.Items[i], operator); err != nil {
				return err
			}
		}
		// 3. 差额 = 换出 - 退回（退回为负）：正=应收，负=应退
		diff := outTotal + retTotal
		if diff > 0 && req.PayMethod == "cash" && req.PaidAmount < diff {
			return fmt.Errorf("实收金额不足：应付 %.2f，实收 %.2f", diff, req.PaidAmount)
		}
		result = &jxc.POSExchangeResult{ReturnOrder: ret, OutOrder: out, DiffAmount: diff}
		return nil
	})
	return result, err
}

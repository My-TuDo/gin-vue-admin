package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/jxc"
)

// bizModel 业务表迁移。
//
// 注意：账本相关表（stock_move / stock_balance / stock_reservation /
// accounting_period / stock_balance_snapshot）的字段变更要格外小心——
// 这些表承载的是商家已经过账的历史数据，AutoMigrate 只允许"加列"，不允许改语义。
func bizModel() error {
	db := global.GVA_DB
	err := db.AutoMigrate(
		&jxc.Warehouse{},
		&jxc.Goods{},
		&jxc.GoodsSku{},
		&jxc.StockMove{},
		&jxc.StockBalance{},
		&jxc.StockReservation{},
		&jxc.AccountingPeriod{},
		&jxc.StockBalanceSnapshot{},
	)
	if err != nil {
		return err
	}
	return nil
}

/**
 * 单据 / 主数据状态字典。
 *
 * 一条状态永远同时包含「文字 + 色调」：颜色只作为辅助信息，
 * 色弱、打印、截图转发都要能读懂（见 docs/17）。
 */

export const STATUS_DICT = {
  purchase: {
    1: { label: '草稿', tone: 'neutral' },
    2: { label: '已审核', tone: 'info' },
    3: { label: '已入库', tone: 'success' },
    4: { label: '已取消', tone: 'neutral' }
  },
  sale: {
    1: { label: '待出库', tone: 'warning' },
    2: { label: '已出库', tone: 'success' },
    3: { label: '已取消', tone: 'neutral' }
  },
  return: {
    1: { label: '待入库', tone: 'warning' },
    2: { label: '已入库', tone: 'success' },
    3: { label: '已取消', tone: 'neutral' }
  },
  purchaseReturn: {
    1: { label: '草稿', tone: 'warning' },
    2: { label: '已退货', tone: 'success' },
    3: { label: '已取消', tone: 'neutral' }
  },
  check: {
    1: { label: '盘点中', tone: 'info' },
    2: { label: '已完成', tone: 'success' },
    3: { label: '已取消', tone: 'neutral' }
  },
  period: {
    1: { label: '开放', tone: 'info' },
    2: { label: '已结账', tone: 'neutral' }
  },
  reversal: {
    1: { label: '已红冲', tone: 'warning' }
  },
  active: {
    1: { label: '启用', tone: 'success' },
    0: { label: '停用', tone: 'neutral' }
  }
}

export const statusMeta = (kind, status) => {
  const dict = STATUS_DICT[kind] || {}
  return dict[status] || { label: '未知', tone: 'neutral' }
}

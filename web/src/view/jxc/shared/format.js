/**
 * 进销存前端展示格式化。
 *
 * 这里只做「展示层」的翻译：账本/单据的内部字段（有符号数量、裸 ID、ISO 时间）
 * 不直接上屏。页面可以组合这些函数，但不要把格式化逻辑散落回模板。
 */

export const round2 = (value) => Math.round((Number(value) || 0) * 100) / 100

const pad = (n) => String(n).padStart(2, '0')

/** 2026-10-27 17:42 */
export const formatDateTime = (value) => {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** 10-27 17:42（列表里省年份） */
export const formatShortTime = (value) => {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** 2026-10-27 */
export const formatDate = (value) => {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 今日 HH:mm */
export const formatClock = (value) => {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export const isToday = (value) => {
  if (!value) return false
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return false
  const now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate()
}

/** 整数千分位；用于件数、SKU 数等 */
export const formatQty = (value) => {
  const n = Number(value)
  if (!Number.isFinite(n)) return '0'
  return n.toLocaleString('zh-CN')
}

/** 百分比：0.507 -> 50.7%（传 0-1 小数） */
export const formatPercent = (value, digits = 1) => {
  const n = Number(value)
  if (!Number.isFinite(n)) return '—'
  return `${(n * 100).toFixed(digits)}%`
}

/** 毛利/销售额，分母为 0 时给 —，不要显示 Infinity */
export const marginOf = (gross, amount) => {
  const a = Number(amount) || 0
  if (!a) return null
  return (Number(gross) || 0) / a
}

export const MOVEMENT_SOURCE = {
  purchase_in: '采购入库',
  sale_out: '销售出库',
  sale_return: '销售退货',
  purchase_return: '采购退货',
  direct_in: '直接入库',
  direct_out: '直接出库',
  check_gain: '盘盈',
  check_loss: '盘亏',
  reversal: '红冲'
}

export const sourceLabel = (type) => MOVEMENT_SOURCE[type] ?? (type || '—')

/** 来源类型 → 单据页签，用于从流水/Documents 交叉跳转 */
export const sourceTab = (type) => {
  if (type === 'purchase_in') return 'purchase'
  if (type === 'sale_out') return 'sale'
  if (type === 'sale_return') return 'return'
  if (type === 'purchase_return') return 'purchaseReturn'
  if (type === 'reversal') return 'reversal'
  return 'sale'
}

/** 数组按字段求和，统一转 Number，避免 string 拼接 */
export const sumBy = (list, pick) => (list || []).reduce((sum, row) => sum + (Number(pick(row)) || 0), 0)

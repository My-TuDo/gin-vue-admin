/**
 * 公共日期格式化：输出 YYYY-MM-DD HH:mm
 * 容错：空值/非法时间返回 '—'
 * 说明：iOS 对带时区的 ISO（2026-08-04T10:30:00+08:00）可解析；
 *      对无 T 的空格格式（2026-08-04 10:30）需先补 T，见下方 replace。
 */
export function fmtTime(iso) {
  if (!iso) return '—'
  const d = new Date(String(iso).replace(' ', 'T'))
  if (Number.isNaN(d.getTime())) return '—'
  const p = (x) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

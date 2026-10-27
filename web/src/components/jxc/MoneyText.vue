<template>
  <span class="inline-flex items-baseline justify-end tabular-nums" :class="[sizeClass, toneClass, { 'font-semibold': strong }]">
    {{ text }}
  </span>
</template>

<script setup>
import { computed } from 'vue'

/**
 * 金额展示：千分位、两位小数、等宽数字、右对齐。
 * 负数标红、正数可带 +；空值用 —，不要显示「NaN」或「0.00」误导人。
 * 金额散落在表格/合计/明细里，格式必须完全一致，否则一列数字对不齐。
 */
const props = defineProps({
  value: { type: [Number, String], default: null },
  size: { type: String, default: 'md', validator: (v) => ['xs', 'sm', 'md', 'lg', 'xl'].includes(v) },
  strong: { type: Boolean, default: false },
  muted: { type: Boolean, default: false },
  signed: { type: Boolean, default: false },
  empty: { type: String, default: '—' },
  prefix: { type: String, default: '' }
})

const SIZE_CLASS = {
  xs: 'text-[11px]',
  sm: 'text-[12px]',
  md: 'text-[13px]',
  lg: 'text-[16px]',
  xl: 'text-[26px] leading-none'
}

const num = computed(() => {
  if (props.value === null || props.value === undefined || props.value === '') return null
  const n = Number(props.value)
  return Number.isFinite(n) ? n : null
})

const sizeClass = computed(() => SIZE_CLASS[props.size] || SIZE_CLASS.md)
const toneClass = computed(() => {
  if (num.value === null) return 'text-muted-foreground'
  if (num.value < 0) return 'text-error'
  if (props.muted) return 'text-muted-foreground'
  return ''
})

const text = computed(() => {
  if (num.value === null) return props.empty
  const n = num.value
  const abs = Math.abs(n)
  const body = abs.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  const sign = n < 0 ? '-' : props.signed && n > 0 ? '+' : ''
  return `${sign}${props.prefix}${body}`
})
</script>

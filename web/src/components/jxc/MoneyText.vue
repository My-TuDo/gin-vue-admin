<template>
  <span class="money" :class="{ neg: isNeg, muted: muted }">{{ text }}</span>
</template>

<script setup>
import { computed } from 'vue'

// 金额显示：千分位、两位小数、等宽、右对齐；负数标红。
// 为什么要单独一个组件：金额散落在表格/合计/明细里，格式必须完全一致，
// 否则一列数字对不齐，读表的人要逐位比。
const props = defineProps({
  value: { type: [Number, String], default: 0 },
  showSign: { type: Boolean, default: false },
  muted: { type: Boolean, default: false },
  emptyAs: { type: String, default: '' }
})

const num = computed(() => {
  const n = Number(props.value)
  return Number.isFinite(n) ? n : 0
})
const isNeg = computed(() => num.value < 0)
const text = computed(() => {
  if (props.emptyAs && (props.value === null || props.value === undefined || props.value === '')) return props.emptyAs
  const abs = Math.abs(num.value)
  const s = abs.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  const sign = num.value < 0 ? '-' : props.showSign && num.value > 0 ? '+' : ''
  return sign + s
})
</script>

<style scoped>
.money {
  font-family: ui-monospace, 'SF Mono', Consolas, monospace;
  font-variant-numeric: tabular-nums;
  text-align: right;
  display: inline-block;
  min-width: 4em;
}
.neg {
  color: var(--el-color-danger);
}
.muted {
  color: var(--el-text-color-placeholder);
}
</style>

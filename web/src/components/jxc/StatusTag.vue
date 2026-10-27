<template>
  <span class="inline-flex items-center gap-1.5 whitespace-nowrap rounded border font-medium leading-none" :class="[toneClass, sizeClass]" :title="title || displayLabel">
    <i v-if="dot" class="h-1.5 w-1.5 shrink-0 rounded-full bg-current" aria-hidden="true"></i>
    <span>{{ displayLabel }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { statusMeta } from '@/view/jxc/shared/status'

// 类名必须留在 .vue 文件里，UnoCSS 才会扫描生成；
// 放到 shared/status.js 会静默丢色（content.pipeline 默认不扫业务 .js）。
const TONE_CLASS = {
  neutral: 'border-border bg-muted text-muted-foreground',
  info: 'border-primary-200 bg-primary-50 text-primary-700',
  success: 'border-success-200 bg-success-50 text-success-700',
  warning: 'border-warning-200 bg-warning-50 text-warning-700',
  danger: 'border-error-200 bg-error-50 text-error-700'
}

/**
 * 状态标签：颜色 + 文字。颜色只辅助，不单独承载信息。
 * 支持 purchase/sale/return/check/period/reversal/active 七类字典；
 * 也可直接传 label + tone 覆盖（例如「已红冲」叠加在单据状态旁）。
 */
const props = defineProps({
  kind: { type: String, default: 'active' },
  status: { type: [Number, String], default: 1 },
  label: { type: String, default: '' },
  tone: { type: String, default: '' },
  dot: { type: Boolean, default: true },
  size: { type: String, default: 'md', validator: (v) => ['xs', 'sm'].includes(v) },
  title: { type: String, default: '' }
})

const meta = computed(() => {
  const fallback = statusMeta(props.kind, Number(props.status))
  return {
    label: props.label || fallback.label,
    tone: props.tone || fallback.tone
  }
})

const displayLabel = computed(() => meta.value.label)
const toneClass = computed(() => TONE_CLASS[meta.value.tone] || TONE_CLASS.neutral)
const sizeClass = computed(() =>
  props.size === 'xs' ? 'px-1.5 py-1 text-[10px]' : 'px-2 py-1 text-[11px]'
)
</script>

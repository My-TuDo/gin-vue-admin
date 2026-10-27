<template>
  <div class="group relative flex min-w-0 flex-col justify-between rounded-lg border border-border bg-container px-4 py-3.5 transition-colors" :class="tone === 'warning' ? 'border-warning-200' : ''">
    <div class="flex items-center justify-between gap-3">
      <span class="truncate text-[12px] text-muted-foreground">{{ label }}</span>
      <span v-if="icon" class="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground transition-colors group-hover:text-base-text">
        <svg-icon :icon="icon" class="h-3.5 w-3.5" />
      </span>
    </div>
    <div class="mt-2.5 flex min-w-0 items-baseline gap-1.5">
      <slot>
        <span class="truncate text-[26px] font-semibold leading-none tracking-tight tabular-nums">{{ value }}</span>
      </slot>
      <span v-if="unit" class="shrink-0 text-[11px] text-muted-foreground">{{ unit }}</span>
    </div>
    <div class="mt-2 min-h-[16px] text-[11.5px] leading-4 text-muted-foreground">
      <slot name="hint">{{ hint }}</slot>
    </div>
  </div>
</template>

<script setup>
/**
 * KPI 磁贴：标签 / 数值 / 说明三层。数值由调用方决定格式（金额请用 MoneyText），
 * 磁贴只负责排版，不猜测业务口径。
 */
defineProps({
  label: { type: String, required: true },
  value: { type: [String, Number], default: '' },
  unit: { type: String, default: '' },
  hint: { type: String, default: '' },
  icon: { type: String, default: '' },
  tone: { type: String, default: 'default', validator: (v) => ['default', 'warning'].includes(v) }
})
</script>

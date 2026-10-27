<template>
  <div class="flex flex-col items-center justify-center px-6 py-14 text-center">
    <div class="grid h-11 w-11 place-items-center rounded-full border border-border bg-muted text-muted-foreground">
      <svg-icon :icon="icon" class="h-5 w-5" />
    </div>
    <div class="mt-3 text-[13px] font-medium text-base-text">{{ title }}</div>
    <p v-if="description" class="mt-1 max-w-[320px] text-[12px] leading-5 text-muted-foreground">{{ description }}</p>
    <div v-if="$slots.default" class="mt-4">
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
/** 空态：说明「为什么空」和「下一步做什么」，不要只放一个灰图标。 */
defineProps({
  icon: { type: String, default: 'lucide:inbox' },
  title: { type: String, default: '暂无数据' },
  description: { type: String, default: '' }
})
</script>

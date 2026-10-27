<template>
  <header class="flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
    <div class="min-w-0">
      <div v-if="eyebrow" class="mb-1.5 flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-[0.12em] text-muted-foreground">
        <span>{{ eyebrow }}</span>
      </div>
      <div class="flex items-center gap-3">
        <h1 class="truncate text-[22px] font-semibold tracking-tight text-base-text">{{ title }}</h1>
        <slot name="badge"></slot>
      </div>
      <p v-if="description" class="mt-1 text-[12.5px] leading-5 text-muted-foreground">{{ description }}</p>
      <slot name="meta"></slot>
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center justify-end gap-2">
      <slot name="actions"></slot>
    </div>
  </header>
</template>

<script setup>
/** 页面标题区：标题、说明、操作。所有业务页共用，保证首屏层级一致。 */
defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
  eyebrow: { type: String, default: '' }
})
</script>

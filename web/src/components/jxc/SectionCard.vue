<template>
  <section class="flex min-h-0 flex-col overflow-hidden rounded-lg border border-border bg-container" :class="$attrs.class">
    <header v-if="title || $slots.actions || $slots.header" class="flex min-h-[46px] shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b border-border px-4 py-2.5">
      <slot name="header">
        <h2 class="text-[13px] font-semibold text-base-text">{{ title }}</h2>
        <span v-if="description" class="text-[12px] text-muted-foreground">{{ description }}</span>
      </slot>
      <div v-if="$slots.actions" class="ml-auto flex items-center gap-2">
        <slot name="actions"></slot>
      </div>
    </header>
    <div class="min-h-0" :class="padded ? 'flex-1 overflow-auto p-4' : 'flex-1 overflow-hidden'">
      <slot></slot>
    </div>
  </section>
</template>

<script setup>
/** 工作台卡片：一层边框、一条分隔线，不叠加阴影。表格页把 padded 设为 false，让表格占满。 */
defineProps({
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  padded: { type: Boolean, default: true }
})
defineOptions({ inheritAttrs: false })
</script>

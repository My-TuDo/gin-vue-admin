<template>
  <Teleport to="body">
    <Transition name="jxc-palette">
      <div
        v-if="open"
        class="jxc-overlay fixed inset-0 z-[3000] flex items-start justify-center bg-[rgba(9,14,24,0.42)] px-4 pt-[14vh]"
        @mousedown.self="close"
      >
        <div
          class="jxc-palette-panel w-full max-w-[580px] overflow-hidden rounded-xl border border-border bg-container shadow-2xl"
          role="dialog"
          aria-modal="true"
          aria-label="命令面板"
        >
          <div class="flex items-center gap-2.5 border-b border-border px-4">
            <svg-icon icon="lucide:search" class="h-4 w-4 shrink-0 text-muted-foreground" />
            <input
              ref="inputRef"
              v-model="keyword"
              class="h-12 min-w-0 flex-1 bg-transparent text-[14px] text-base-text outline-none placeholder:text-muted-foreground"
              placeholder="搜索页面或操作…"
              autocomplete="off"
              spellcheck="false"
              @keydown="onKeydown"
            />
            <kbd class="shrink-0 rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">ESC</kbd>
          </div>

          <div class="max-h-[352px] overflow-y-auto p-2">
            <div v-if="filtered.length === 0" class="px-4 py-12 text-center">
              <svg-icon icon="lucide:search-x" class="mx-auto h-5 w-5 text-muted-foreground" />
              <p class="mt-2 text-[12.5px] text-muted-foreground">没有匹配的页面或操作</p>
            </div>
            <button
              v-for="(item, index) in filtered"
              :key="item.key"
              :ref="(el) => setItemEl(el, index)"
              type="button"
              class="flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-left outline-none transition-colors"
              :class="index === active ? 'bg-primary-50 text-primary-700' : 'hover:bg-muted'"
              @mousemove="active = index"
              @click="run(item)"
            >
              <span
                class="grid h-8 w-8 shrink-0 place-items-center rounded-md border bg-container"
                :class="index === active ? 'border-primary-200 text-primary' : 'border-border text-muted-foreground'"
              >
                <svg-icon :icon="item.icon" class="h-4 w-4" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-[13px] font-medium" :class="index === active ? 'text-primary-700' : 'text-base-text'">{{ item.label }}</span>
                <span class="mt-0.5 block truncate text-[11px]" :class="index === active ? 'text-primary-600' : 'text-muted-foreground'">{{ item.desc }}</span>
              </span>
              <kbd v-if="item.shortcut" class="shrink-0 rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">{{ item.shortcut }}</kbd>
              <svg-icon v-else icon="lucide:corner-down-left" class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            </button>
          </div>

          <div class="flex items-center gap-4 border-t border-border bg-muted px-4 py-2 text-[11px] text-muted-foreground">
            <span class="flex items-center gap-1"><kbd class="jxc-kbd">↑</kbd><kbd class="jxc-kbd">↓</kbd> 选择</span>
            <span class="flex items-center gap-1"><kbd class="jxc-kbd">↵</kbd> 打开</span>
            <span class="ml-auto">进销存工作台</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

/**
 * 全局命令面板（Ctrl/Cmd+K）：导航与高频操作统一入口。
 * 只做「跳转」，不在这里执行业务写操作，避免把危险动作藏进搜索框。
 */
const props = defineProps({
  items: { type: Array, default: () => [] }
})

const open = defineModel({ type: Boolean, default: false })
const router = useRouter()
const keyword = ref('')
const active = ref(0)
const inputRef = ref(null)
const itemEls = ref([])

const setItemEl = (el, index) => {
  itemEls.value[index] = el
}

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) return props.items
  return props.items.filter((item) =>
    `${item.label} ${item.desc} ${item.keywords || ''}`.toLowerCase().includes(k)
  )
})

watch(filtered, () => {
  active.value = 0
  itemEls.value = []
})

watch(active, async () => {
  await nextTick()
  itemEls.value[active.value]?.scrollIntoView({ block: 'nearest' })
})

watch(open, async (value) => {
  if (!value) return
  keyword.value = ''
  active.value = 0
  await nextTick()
  inputRef.value?.focus()
})

const close = () => {
  open.value = false
}

const run = (item) => {
  if (!item) return
  close()
  router.push(item.to)
}

const onKeydown = (event) => {
  if (event.key === 'Escape') {
    event.preventDefault()
    close()
    return
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    if (filtered.value.length) active.value = (active.value + 1) % filtered.value.length
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    if (filtered.value.length) active.value = (active.value - 1 + filtered.value.length) % filtered.value.length
    return
  }
  if (event.key === 'Enter') {
    event.preventDefault()
    run(filtered.value[active.value])
  }
}
</script>

<style scoped>
.jxc-palette-enter-active,
.jxc-palette-leave-active {
  transition: opacity 0.12s ease;
}
.jxc-palette-enter-active .jxc-palette-panel,
.jxc-palette-leave-active .jxc-palette-panel {
  transition: transform 0.12s ease, opacity 0.12s ease;
}
.jxc-palette-enter-from,
.jxc-palette-leave-to {
  opacity: 0;
}
.jxc-palette-enter-from .jxc-palette-panel,
.jxc-palette-leave-to .jxc-palette-panel {
  transform: translateY(-4px) scale(0.995);
  opacity: 0;
}
.jxc-kbd {
  display: inline-flex;
  min-width: 16px;
  justify-content: center;
  border: 1px solid var(--el-border-color);
  border-radius: 3px;
  background: var(--el-bg-color);
  padding: 0 3px;
  font-family: ui-monospace, monospace;
  font-size: 10px;
  line-height: 14px;
}
</style>

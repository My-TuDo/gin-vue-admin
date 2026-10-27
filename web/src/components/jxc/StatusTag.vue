<template>
  <span class="status" :class="tone">
    <i class="dot"></i>{{ label }}
  </span>
</template>

<script setup>
import { computed } from 'vue'

// 状态标签：**颜色 + 文字**。不靠颜色单独承载信息——
// 色弱用户、打印出来的单据、截图转发都要能读懂。
const props = defineProps({
  kind: { type: String, required: true }, // purchase / sale
  status: { type: Number, required: true }
})

const MAP = {
  purchase: {
    1: { label: '草稿', tone: 'muted' },
    2: { label: '已审核', tone: 'info' },
    3: { label: '已入库', tone: 'success' },
    4: { label: '已取消', tone: 'muted' }
  },
  sale: {
    1: { label: '待出库', tone: 'warning' },
    2: { label: '已出库', tone: 'success' },
    3: { label: '已取消', tone: 'muted' }
  }
}

const label = computed(() => MAP[props.kind]?.[props.status]?.label ?? '未知')
const tone = computed(() => MAP[props.kind]?.[props.status]?.tone ?? 'muted')
</script>

<style scoped>
.status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  padding: 0 8px;
  line-height: 20px;
  border-radius: 10px;
  white-space: nowrap;
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
.success {
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
}
.warning {
  color: var(--el-color-warning);
  background: var(--el-color-warning-light-9);
}
.info {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.muted {
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
}
</style>

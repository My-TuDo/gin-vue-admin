<template>
  <el-dialog
    v-model="modelValue"
    width="400px"
    class="result-dialog"
    append-to-body
    destroy-on-close
    :show-close="false"
    :close-on-click-modal="false"
  >
    <div class="rd-wrap">
      <div class="rd-check">
        <el-icon :size="64" color="#67c23a"><CircleCheckFilled /></el-icon>
      </div>
      <div class="rd-title">{{ title }}</div>
      <div class="rd-lines">
        <div
          v-for="(line, i) in lines"
          :key="i"
          class="rd-line"
          :style="{ animationDelay: `${0.25 + i * 0.06}s` }"
        >
          <span class="rd-k">{{ line[0] }}</span>
          <span class="rd-v">{{ line[1] }}</span>
        </div>
      </div>
    </div>
    <template #footer>
      <el-button type="primary" size="large" class="rd-ok" @click="close">完 成</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { CircleCheckFilled } from '@element-plus/icons-vue'

defineProps({
  title: { type: String, default: '' },
  lines: { type: Array, default: () => [] },
})
const modelValue = defineModel({ type: Boolean, default: false })
const close = () => { modelValue.value = false }
</script>

<style scoped>
.rd-wrap { text-align: center; padding: 8px 6px 2px; }
.rd-check { animation: rd-pop 0.5s cubic-bezier(0.18, 0.89, 0.32, 1.28) both; }
.rd-check .el-icon { filter: drop-shadow(0 4px 10px rgba(103, 194, 58, 0.35)); }
.rd-title {
  font-size: 22px;
  font-weight: 700;
  color: #303133;
  margin-top: 10px;
  animation: rd-fade 0.35s ease 0.15s both;
}
.rd-lines {
  margin-top: 18px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid #ebeef5;
}
.rd-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 9px 14px;
  font-size: 14px;
  opacity: 0;
  animation: rd-fade 0.3s ease both;
}
.rd-line:nth-child(odd) { background: #f8fafc; }
.rd-k { color: #909399; }
.rd-v { color: #303133; font-weight: 600; font-size: 15px; }
.rd-ok { width: 160px; height: 42px; font-size: 16px; font-weight: 600; border-radius: 9px; }

@keyframes rd-pop {
  0% { transform: scale(0); opacity: 0; }
  60% { transform: scale(1.12); }
  100% { transform: scale(1); opacity: 1; }
}
@keyframes rd-fade {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: none; }
}
</style>

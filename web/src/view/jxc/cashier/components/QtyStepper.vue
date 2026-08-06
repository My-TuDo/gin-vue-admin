<template>
  <div class="qty-stepper">
    <button
      type="button"
      class="qs-btn minus"
      :class="{ disabled: disabledMin }"
      :disabled="disabledMin"
      aria-label="减少数量"
      @click="step(-1)"
    >−</button>
    <input
      class="qs-input"
      :value="cur"
      inputmode="numeric"
      @change="onChange"
      @blur="onChange"
      @keydown.enter.prevent="onChange($event, true)"
    />
    <button
      type="button"
      class="qs-btn plus"
      :class="{ disabled: disabledMax }"
      :disabled="disabledMax"
      aria-label="增加数量"
      @click="step(1)"
    >+</button>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  min: { type: Number, default: 0 },
  max: { type: Number, default: Number.MAX_SAFE_INTEGER },
})
const model = defineModel({ type: Number, default: 0 })

const cur = computed(() => {
  const n = Number(model.value)
  return Number.isFinite(n) ? Math.floor(n) : 0
})
const disabledMin = computed(() => cur.value <= props.min)
const disabledMax = computed(() => cur.value >= props.max)
const clamp = (v) => Math.min(Math.max(Math.floor(v), props.min), props.max)

const step = (d) => { model.value = clamp(cur.value + d) }
const onChange = (e, force) => {
  let n = Number(e.target.value)
  if (!Number.isFinite(n)) n = props.min
  model.value = clamp(n)
  if (force) e.target.blur()
}
</script>

<style scoped>
.qty-stepper {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: #f2f6fc;
  border-radius: 9px;
  padding: 3px;
  user-select: none;
}
.qs-btn {
  width: 26px;
  height: 26px;
  border: none;
  background: #fff;
  border-radius: 6px;
  font-size: 16px;
  font-weight: 700;
  line-height: 1;
  color: #303133;
  cursor: pointer;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  transition: all 0.15s;
}
.qs-btn:hover:not(.disabled) { background: #409eff; color: #fff; }
.qs-btn.plus { background: #ecf5ff; color: #409eff; }
.qs-btn.plus:hover:not(.disabled) { background: #409eff; color: #fff; }
.qs-btn.minus:hover:not(.disabled) { background: #f56c6c; color: #fff; }
.qs-btn.disabled { opacity: 0.35; cursor: not-allowed; box-shadow: none; }
.qs-input {
  width: 38px;
  height: 26px;
  border: none;
  background: transparent;
  text-align: center;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  -moz-appearance: textfield;
  padding: 0;
}
.qs-input::-webkit-outer-spin-button,
.qs-input::-webkit-inner-spin-button { -webkit-appearance: none; margin: 0; }
.qs-input:focus { outline: none; color: #409eff; }
</style>

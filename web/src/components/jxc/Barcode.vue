<template>
  <div class="jxc-barcode">
    <!-- 有值：QR 用 img（dataURL 打印兼容性优于 canvas）；条码用 svg 交给 JsBarcode -->
    <template v-if="value">
      <img v-if="format === 'QR' && qrUrl" :src="qrUrl" class="barcode-img" :style="qrStyle" alt="QRCode" />
      <svg v-else ref="svgRef" class="barcode-svg" :style="barStyle" />
    </template>
    <!-- 空值占位：灰底虚线 + 「无条码」 -->
    <div v-else class="barcode-empty">
      <span class="bc-dash" />
      <span class="bc-label">无条码</span>
    </div>
    <!-- 条码下方文字（QR 不显示文字） -->
    <div v-if="value && showText && format !== 'QR'" class="barcode-text">{{ value }}</div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import JsBarcode from 'jsbarcode'
import QRCode from 'qrcode'

const props = defineProps({
  value: { type: String, default: '' },
  // 支持 CODE128 / EAN13 / CODE39 / QR
  format: { type: String, default: 'CODE128' },
  width: { type: Number, default: 180 },
  height: { type: Number, default: 60 },
  showText: { type: Boolean, default: true },
})
const emit = defineEmits(['fallback'])

const svgRef = ref(null)
const qrUrl = ref('')

const barStyle = computed(() => ({ width: `${props.width}px`, height: `${props.height}px` }))
// QR 为正方形，边长取 width，高度随宽度自适应
const qrStyle = computed(() => ({ width: `${props.width}px`, height: `${props.width}px` }))

// 条形码渲染：所选 format 与内容不匹配（如 EAN13 收到非 13 位数字）会抛错，
// 捕获后自动回退 CODE128 重渲染，并通过 fallback 事件通知父组件提示
const renderBar = () => {
  if (!svgRef.value) return
  const opts = { width: 2, height: props.height - 16, displayValue: false, margin: 0 }
  try {
    JsBarcode(svgRef.value, props.value, { ...opts, format: props.format })
  } catch (e) {
    emit('fallback', { format: props.format, reason: (e && e.message) || '条码渲染失败' })
    try {
      JsBarcode(svgRef.value, props.value, { ...opts, format: 'CODE128' })
    } catch (e2) {
      // CODE128 理论上接受任意 ASCII，此处为极端兜底
      console.warn('[Barcode] CODE128 渲染失败', e2)
    }
  }
}

// QR 渲染：黑色前景白底，toDataURL 生成方形图片
const renderQR = async () => {
  try {
    qrUrl.value = await QRCode.toDataURL(props.value, {
      width: props.width,
      margin: 1,
      errorCorrectionLevel: 'M',
      color: { dark: '#000000', light: '#ffffff' },
    })
  } catch (e) {
    // QR 生成失败：回退 CODE128 条码展示
    emit('fallback', { format: props.format, reason: (e && e.message) || 'QR 生成失败' })
    renderBar()
  }
}

const render = () => {
  if (!props.value) {
    qrUrl.value = ''
    return
  }
  if (props.format === 'QR') renderQR()
  else renderBar()
}

// 首次挂载与 value/format 变化时重新渲染（DOM 就绪后再画）
onMounted(() => nextTick(render))
watch(
  () => [props.value, props.format],
  () => nextTick(render)
)
</script>

<style scoped>
.jxc-barcode {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
}
.barcode-svg,
.barcode-img { display: block; }
.barcode-text {
  margin-top: 4px;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12px;
  letter-spacing: 1px;
  color: #303133;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 空值占位 */
.barcode-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  height: 42px;
  background: #f7f9fc;
  border: 1px dashed #dcdfe6;
  border-radius: 6px;
}
.bc-dash {
  width: 70%;
  height: 6px;
  background: repeating-linear-gradient(90deg, #c0c4cc 0 2px, transparent 2px 4px);
}
.bc-label { font-size: 11px; color: #909399; }
</style>

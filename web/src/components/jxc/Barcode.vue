<template>
  <div class="jxc-barcode">
    <!-- 无值：空占位（未填条码） -->
    <div v-if="!value" class="barcode-empty">
      <span class="bc-dash" />
      <span class="bc-label">无条码</span>
    </div>
    <!-- 内容预校验失败：不支持占位（内容无法生成条码，与「无条码」区分） -->
    <div v-else-if="invalidReason" class="barcode-empty invalid">
      <span class="bc-dash" />
      <span class="bc-label">内容不支持</span>
      <span class="bc-detail" :title="invalidReason">{{ invalidReason }}</span>
    </div>
    <!-- 正常渲染：QR 用 img（dataURL 打印兼容性优于 canvas）；条码用 svg 交给 JsBarcode -->
    <template v-else>
      <img v-if="format === 'QR' && qrUrl" :src="qrUrl" class="barcode-img" :style="qrStyle" alt="QRCode" />
      <svg v-else ref="svgRef" class="barcode-svg" :style="barStyle" />
    </template>
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
const invalidReason = ref('') // 非空=内容无法生成条码（显示「内容不支持」占位）

const barStyle = computed(() => ({ width: `${props.width}px`, height: `${props.height}px` }))
// QR 为正方形，边长取 width，高度随宽度自适应
const qrStyle = computed(() => ({ width: `${props.width}px`, height: `${props.width}px` }))

// 条形码内容预校验：返回空串表示可通过；否则返回不可渲染的具体原因
const validateBarContent = () => {
  // 条形码只支持 ASCII（中文等字符无法编码进 CODE128/CODE39/EAN13）
  if (/[^\x00-\x7F]/.test(props.value)) {
    return '条码内容含中文等非 ASCII 字符，无法生成条形码，请填写商品条码'
  }
  // EAN13 仅接受 12 或 13 位数字
  if (props.format === 'EAN13' && !/^\d{12}(\d)?$/.test(props.value)) {
    return 'EAN13 需要 12 或 13 位数字条码'
  }
  return ''
}

// 条形码渲染：预校验通过后交给 JsBarcode；渲染抛错（如 CODE39 遇小写字母）时
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
    invalidReason.value = ''
    return
  }
  // QR 支持中文，不做条码内容预校验
  if (props.format === 'QR') {
    invalidReason.value = ''
    renderQR()
    return
  }
  // 条形码预校验：内容不合法 → 不渲染，明确占位 + fallback 事件（EAN13 不回退 CODE128，让用户明确知道内容不符）
  const reason = validateBarContent()
  if (reason) {
    invalidReason.value = reason
    emit('fallback', { format: props.format, reason })
    return
  }
  invalidReason.value = ''
  renderBar()
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
/* 内容不支持占位：暖色提示，高度自适应以容纳小字原因 */
.barcode-empty.invalid {
  height: auto;
  padding: 8px 6px;
  background: #fdf6ec;
  border-color: #f3d19e;
}
.bc-dash {
  width: 70%;
  height: 6px;
  background: repeating-linear-gradient(90deg, #c0c4cc 0 2px, transparent 2px 4px);
}
.bc-label { font-size: 11px; color: #909399; }
.barcode-empty.invalid .bc-label { color: #b88230; }
.bc-detail {
  max-width: 96%;
  font-size: 10px;
  color: #b88230;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

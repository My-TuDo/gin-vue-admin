<template>
  <view class="qiun-chart">
    <!-- canvas type="2d"：微信小程序端由 uCharts 绘制；数据为空或初始化失败时显示占位 -->
    <canvas
      v-if="showCanvas"
      :id="cid"
      :canvas-id="cid"
      type="2d"
      class="qiun-canvas"
      :style="{ height: height + 'px' }"
      @touchstart="onTouchStart"
      @touchmove="onTouchMove"
      @touchend="onTouchEnd"
    />
    <view v-else class="qiun-empty">{{ emptyText }}</view>
  </view>
</template>

<script setup>
import { computed, getCurrentInstance, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import uCharts from '@qiun/ucharts'

const props = defineProps({
  /** 图表类型：line 折线 | column 柱状 | ring 环形 | pie 饼图 | area 面积 */
  type: { type: String, default: 'line' },
  /** 图表数据：{ categories: [], series: [{ name, data: [] }] } */
  chartData: {
    type: Object,
    default: () => ({ categories: [], series: [] })
  },
  /** 画布高度（px） */
  height: { type: Number, default: 280 },
  /** 系列配色 */
  colors: {
    type: Array,
    default: () => ['#2b8a3e', '#f5a623', '#e64340', '#4a90d9', '#9b59b6', '#1abc9c', '#e67e22']
  },
  /** 空数据占位文案 */
  emptyText: { type: String, default: '暂无数据' }
})

const instance = getCurrentInstance()
// canvas 唯一 id（避免多实例冲突）
const cid = 'qiun_' + Math.random().toString(36).slice(2, 10)

let chart = null
let ctx = null
let canvasNode = null
let pixelRatio = 1
let cssWidth = 0
let cssHeight = 0
let inited = false

const initFailed = ref(false)

const hasData = computed(() => {
  const data = props.chartData || {}
  const cats = Array.isArray(data.categories) ? data.categories : []
  const series = Array.isArray(data.series) ? data.series : []
  return series.some((s) => s && Array.isArray(s.data) && s.data.length > 0) && cats.length > 0
})

const showCanvas = computed(() => hasData.value && !initFailed.value)

onMounted(() => {
  // 组件挂载后初始化 canvas（数据未就绪时由 watch 兜底）
  initCanvas()
})

watch(
  () => props.chartData,
  () => {
    if (!hasData.value) return
    if (canvasNode && ctx) {
      draw()
    } else if (!initFailed.value) {
      initCanvas()
    }
  },
  { deep: true }
)

function queryCanvas() {
  return new Promise((resolve) => {
    let q = uni.createSelectorQuery()
    if (instance && instance.proxy) {
      q = q.in(instance.proxy)
    }
    q.select('#' + cid)
      .fields({ node: true, size: true })
      .exec((res) => {
        resolve(res && res[0])
      })
  })
}

async function initCanvas() {
  if (inited || initFailed.value || !hasData.value) return
  // 等待原生节点就绪（小程序端首帧可能查不到，做几次重试）
  await nextTick()
  let res = null
  for (let i = 0; i < 5; i++) {
    res = await queryCanvas()
    if (res && res.node) break
    await sleep(60)
  }
  if (!res || !res.node) {
    console.error('[qiun-chart] canvas 节点获取失败', cid)
    initFailed.value = true
    return
  }
  canvasNode = res.node
  ctx = canvasNode.getContext('2d')
  const sys = uni.getSystemInfoSync()
  pixelRatio = (sys && sys.pixelRatio) || 1
  cssWidth = res.width
  cssHeight = res.height
  canvasNode.width = cssWidth * pixelRatio
  canvasNode.height = cssHeight * pixelRatio
  ctx.scale(pixelRatio, pixelRatio)
  inited = true
  draw()
}

function buildOptions() {
  const data = props.chartData || {}
  const categories = Array.isArray(data.categories) ? data.categories : []
  const series = Array.isArray(data.series) ? data.series : []
  const isPieLike = props.type === 'pie' || props.type === 'ring'
  return {
    type: props.type,
    context: ctx,
    // uCharts 的 width/height 使用 CSS 逻辑像素，pixelRatio 单独传入
    width: cssWidth,
    height: cssHeight,
    pixelRatio: pixelRatio,
    categories: categories,
    series: series,
    animation: true,
    background: '#FFFFFF',
    color: props.colors,
    padding: [12, 5, 0, 5],
    dataLabel: false,
    enableScroll: false,
    legend: {
      show: series.length > 1 || isPieLike,
      position: 'bottom',
      fontSize: 11,
      lineHeight: 16,
      itemGap: 12
    },
    xAxis: {
      disableGrid: true,
      fontSize: 10,
      fontColor: '#999999',
      labelCount: 4,
      rotateLabel: false,
      disabled: isPieLike
    },
    yAxis: {
      gridType: 'dash',
      dashLength: 2,
      data: [],
      fontSize: 10,
      fontColor: '#999999',
      splitNumber: 4,
      disabled: isPieLike
    },
    extra: {
      line: { type: 'curve', width: 2 },
      area: { type: 'curve', width: 2, opacity: 0.15 },
      column: { width: 16 },
      ring: { ringWidth: 26, centerColor: '#FFFFFF', activeOpacity: 0.6, offsetAngle: 0 },
      pie: { activeOpacity: 0.6 }
    }
  }
}

function draw() {
  if (!ctx || !canvasNode) return
  try {
    chart = new uCharts(buildOptions())
  } catch (e) {
    console.error('[qiun-chart] 绘制失败:', e)
    initFailed.value = true
  }
}

// 图例点击 / 十字准星 tooltip（小程序端事件对象含 touches）
function onTouchStart(e) {
  if (chart && typeof chart.touchLegend === 'function') chart.touchLegend(e)
}
function onTouchMove(e) {
  if (chart && typeof chart.showToolTip === 'function') chart.showToolTip(e)
}
function onTouchEnd(e) {
  if (chart && typeof chart.touchLegend === 'function') chart.touchLegend(e)
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

onBeforeUnmount(() => {
  chart = null
  ctx = null
  canvasNode = null
  inited = false
})
</script>

<style lang="scss" scoped>
.qiun-chart {
  width: 100%;
  position: relative;
}

.qiun-canvas {
  display: block;
  width: 100%;
}

.qiun-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200rpx;
  color: #999;
  font-size: 24rpx;
  background: #fafbfc;
  border-radius: 12rpx;
}
</style>

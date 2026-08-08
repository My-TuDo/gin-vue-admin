<template>
  <view class="qiun-chart">
    <!-- canvas type="2d"：微信小程序端由 uCharts 绘制；数据为空或初始化失败时显示占位 -->
    <canvas
      v-show="showCanvas"
      :id="cid"
      :canvas-id="cid"
      type="2d"
      class="qiun-canvas"
      :style="{ height: height + 'px' }"
      @touchstart="onTouchStart"
      @touchmove="onTouchMove"
      @touchend="onTouchEnd"
    />
    <!-- 占位（空数据/初始化失败）：与 canvas 同用 v-show 切换。
         Vue3 中 v-else 必须紧跟 v-if/v-else-if，无法与 v-show 配对（否则编译报错）；
         双 v-show 只切换 display，canvas 节点常驻不复建 —— 图表初始化昂贵（canvas 2d 节点查询+
         实例构造），避免反复销毁重建造成二次初始化的空白与卡顿（社区方法论第 2 条）。 -->
    <view v-show="!showCanvas" class="qiun-empty">{{ emptyText }}</view>
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
  emptyText: { type: String, default: '暂无数据' },
  /**
   * 是否启用绘制动画，默认 false（社区方法论第 1 条）：
   * uCharts 动画为逐帧重绘，在真机（iPhone）上触发逻辑层/视图层高频跨层通信造成卡顿；
   * 关闭后整图一次性绘制，数据更新走 updateData 也不会逐帧。
   */
  animation: { type: Boolean, default: false }
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

let initTimer = null

onMounted(() => {
  // 首绘延后约 150ms 执行，避开页面转场动画抢帧（社区方法论第 2 条：uCharts 官方 README
  // 建议复杂 canvas 延后 100-300ms 渲染）。仅首绘延迟；watch 触发的数据更新不延迟。
  initTimer = setTimeout(() => {
    initTimer = null
    initCanvas()
  }, 150)
})

watch(
  () => props.chartData,
  () => {
    if (!hasData.value) return
    if (chart && canvasNode && ctx) {
      // 数据更新走 uCharts 增量路径：u-charts.js 第 7149 行 uCharts.prototype.updateData
      // 实现为 this.opts = assign({}, this.opts, data) 后重走 drawCharts，复用同一实例与
      // context，避免 new uCharts 全量重建（再次构造实例/解析默认 opts）。updateData 走的是
      // 同一套 drawCharts 绘制管线，对 line/column/area/ring 均安全。显式传入 animation，
      // 确保不启动逐帧动画（drawCharts 内 duration = opts.animation ? opts.duration : 0，
      // 见 u-charts.js 第 6397 行）。
      const data = props.chartData || {}
      chart.updateData({
        categories: Array.isArray(data.categories) ? data.categories : [],
        series: Array.isArray(data.series) ? data.series : [],
        animation: props.animation
      })
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
  pixelRatio = getPixelRatio()
  cssWidth = res.width
  cssHeight = res.height
  // canvas 2d：drawing buffer 使用物理像素（CSS 尺寸 × dpr）
  // 注意：这里不能 ctx.scale(pixelRatio, pixelRatio)。
  // 已核查 node_modules/@qiun/ucharts/u-charts.js：库内部没有任何 ctx.scale/setTransform 调用，
  // 而是把所有 fontSize/padding/area/线宽等数值 × opts.pixelRatio（opts.pix）手工放大后，
  // 统一在【物理像素坐标系】中绘制（坐标基于物理像素的 width/height，见 buildOptions 注释）。
  // 若此处再手动 scale，会造成双重缩放：字体/柱宽/轴标签被放大 dpr 倍而重叠溢出（本次 bug 根因）。
  canvasNode.width = cssWidth * pixelRatio
  canvasNode.height = cssHeight * pixelRatio
  inited = true
  draw()
}

// uni.getSystemInfoSync 已废弃，改用 uni.getWindowInfo（兼容低版本降级）
function getPixelRatio() {
  if (typeof uni.getWindowInfo === 'function') {
    try {
      const info = uni.getWindowInfo()
      if (info && info.pixelRatio) return info.pixelRatio
    } catch (e) {
      // 低版本基础库可能不支持，继续走降级逻辑
    }
  }
  const sys = uni.getSystemInfoSync()
  return (sys && sys.pixelRatio) || 1
}

function buildOptions() {
  const data = props.chartData || {}
  const categories = Array.isArray(data.categories) ? data.categories : []
  const series = Array.isArray(data.series) ? data.series : []
  const isPieLike = props.type === 'pie' || props.type === 'ring'
  const isColumn = props.type === 'column'
  // 柱状图柱子穿透 X 轴修复（真机现象）。源码依据 @qiun/ucharts/u-charts.js：
  //  1) 柱子基线（底部 y）由 drawColumnDataPoints 按 0 轴位置计算（第 3000-3002 行）：
  //       zeroHeight = (opts.height - area[0] - area[2]) * (0 - minRange) / (maxRange - minRange)
  //       zeroPoints  = opts.height - Math.round(zeroHeight) - area[2]
  //     柱子底边 = zeroPoints；X 轴线画在 opts.height - area[2]（drawXAxis 第 4646 行）。
  //  2) y 轴刻度由 getYAxisTextList 自动计算（第 1953 行）：yAxis.data 为空时 minRange
  //     取自 getDataRange/findRange（第 1952、425 行），对全正数据（如销售额）向下取整
  //     但不为 0（数据 500~800 时 minRange≈500）。
  //  3) 于是 minRange>0 时 zeroHeight<0，zeroPoints 大于 opts.height - area[2]（X 轴线
  //     位置），柱子底边越界画到 X 轴线之下 → 真机柱子穿透 X 轴。
  //  修复组合（仅 column 生效）：
  //   a) yAxis.data = [{ min: 0 }]：calYAxisData（第 2000 行）会把 min 传入
  //      getYAxisTextList，第 1953 行 minRange 取 0，zeroHeight=0，柱子底部恒等于
  //      X 轴线位置，不再穿透；data 项内补 fontSize/fontColor 保持刻度样式不变
  //      （走 data 分支后第 4733 行绘制回退 config.fontSize=13，需显式传 10）。
  //   b) padding 底部给 6px（仅 column）：x 轴刻度区域（area[2] 含 xAxisHeight，
  //      第 6487 行）之外再留白，避免柱底/刻度贴 canvas 底边。
  //   c) xAxis.axisLine = true：显式打开轴线（uCharts 默认即 true，第 7036 行），
  //      作为柱底参照；折线/饼图维持原配置不受影响。
  return {
    type: props.type,
    context: ctx,
    // uCharts 2.5.0（u-charts.js）内部不做 ctx.scale/setTransform（已核查源码：全文件无相关调用），
    // 绘制坐标全部基于 opts.width/opts.height/opts.area，而 area=padding×pixelRatio、
    // 字号/线宽/柱宽等全部 ×pixelRatio（见 u-charts.js 第 6420 行、7095 行、1500 行），
    // 触摸坐标也 ×pixelRatio（第 497-506 行 getTouches），即库采用“物理像素坐标系”。
    // 因此 width/height 必须传入物理像素（CSS 逻辑像素 × dpr），pixelRatio 单独传入 dpr，
    // 外部禁止再手动 ctx.scale，否则与库内部 ×pix 构成双重缩放（真机图表错乱根因）。
    width: cssWidth * pixelRatio,
    height: cssHeight * pixelRatio,
    pixelRatio: pixelRatio,
    categories: categories,
    series: series,
    // animation 默认关闭（由 prop 控制）：开启时为逐帧动画，真机跨层通信开销大（社区方法论第 1 条）
    animation: props.animation,
    background: '#FFFFFF',
    color: props.colors,
    padding: [12, 5, isColumn ? 6 : 0, 5],
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
      // X 轴标签抽稀依据（@qiun/ucharts 2.5.0 u-charts.js 第 4545-4566 行）：
      //   drawXAxis 中 labelCount 语义为「X 轴单屏最多标签数」。实现：未设 xAxis.itemCount 时
      //   maxXAxisListLength = labelCount - 1（第 4550 行），ratio = ceil(categories.length /
      //   maxXAxisListLength)，仅 i % ratio === 0 的索引绘制标签、其余置空（第 4559-4565 行），
      //   且最后一个标签强制显示（第 4566 行）。
      //   → 本页趋势传近 7 天（7 个分类）：labelCount=4 ⇒ max=3 ⇒ ratio=ceil(7/3)=3 ⇒
      //     显示索引 0/3/6（首/中/尾）3 个日期标签，彼此间隔 3 个数据槽位，真机小屏不重叠。
      //   注意：labelCount=7 并非全显，而是 max=6、ratio=ceil(7/6)=2 ⇒ 显示 0/2/4/6 共 4 个；
      //   全显需 labelCount >= categories.length + 1（ratio 退化为 1）。7 天取 4 更直观。
      //   另：labelCount 直接生效，无需配合 itemCount（仅 enableScroll 场景才需要）；rotateLabel
      //   仅控制标签旋转绘制分支（第 4568/4591 行），与抽稀无关，保持 false 不旋转。
      labelCount: 4,
      rotateLabel: false,
      // 显示 X 轴线（柱子底边参照；uCharts 默认即 true，见 u-charts.js 第 7036 行）
      axisLine: true,
      disabled: isPieLike
    },
    yAxis: {
      gridType: 'dash',
      dashLength: 2,
      // column：强制 y 轴从 0 开始（见 buildOptions 注释 a），柱子基线=绘图区底部=X 轴线；
      // 其它类型保持 data:[] 自动刻度（折线图聚焦数据波动、保持现状）。
      data: isColumn ? [{ min: 0, fontSize: 10, fontColor: '#999999' }] : [],
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
  if (initTimer) clearTimeout(initTimer)
  initTimer = null
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

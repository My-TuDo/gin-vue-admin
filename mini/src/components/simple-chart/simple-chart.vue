<template>
  <view class="simple-chart">
    <!-- 柱状/折线：横向条形图（纯 CSS，零依赖降级方案） -->
    <view v-if="type === 'column' || type === 'line'" class="sc-bars">
      <view v-if="hasData">
        <view v-for="(row, i) in barRows" :key="i" class="sc-bar-row">
          <text class="sc-bar-label">{{ row.label }}</text>
          <view class="sc-bar-track">
            <view
              class="sc-bar-fill"
              :style="{ width: row.percent + '%', background: row.color }"
            />
          </view>
          <text class="sc-bar-value">{{ row.value }}</text>
        </view>
      </view>
      <view v-else class="ui-empty">{{ emptyText }}</view>
    </view>

    <!-- 环形/饼图：conic-gradient 简易环形 + 占比列表 -->
    <view v-else-if="type === 'ring' || type === 'pie'" class="sc-ring-wrap">
      <view v-if="hasData" class="sc-ring-body">
        <view class="sc-ring" :style="ringStyle" />
        <view class="sc-ring-center">{{ ringCenter }}</view>
      </view>
      <view v-else class="ui-empty">{{ emptyText }}</view>
      <view v-if="hasData" class="sc-legend">
        <view v-for="(row, i) in ringRows" :key="i" class="sc-legend-row">
          <view class="sc-legend-dot" :style="{ background: row.color }" />
          <text class="sc-legend-name">{{ row.label }}</text>
          <text class="sc-legend-value">{{ row.percentText }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

// 注意：defineProps 的 default 会被提升到模块作用域，不能引用 <script setup> 局部变量，
// 故 colors default 用内联数组字面量；改动时需与 uni.scss $ui-chart-colors 同步。

const props = defineProps({
  type: { type: String, default: 'column' },
  chartData: { type: Object, default: () => ({ categories: [], series: [] }) },
  height: { type: Number, default: 280 },
  colors: {
    type: Array,
    default: () => ['#2264f2', '#f9901f', '#f56c6c', '#2f6fde', '#4b83f5', '#909399']
  },
  emptyText: { type: String, default: '暂无数据' }
})

const palette = computed(() => {
  const c = props.colors || []
  return c.length ? c : ['#2264f2']
})

const hasData = computed(() => {
  const data = props.chartData || {}
  const cats = Array.isArray(data.categories) ? data.categories : []
  const series = Array.isArray(data.series) ? data.series : []
  return series.some((s) => s && Array.isArray(s.data) && s.data.length > 0) && cats.length > 0
})

// ---- 柱状/折线：转横向条形 ----
const barRows = computed(() => {
  const data = props.chartData || {}
  const cats = Array.isArray(data.categories) ? data.categories : []
  const series = Array.isArray(data.series) ? data.series : []
  if (!series.length || !cats.length) return []
  const values = cats.map((label, i) => ({
    label: String(label || ''),
    value: toNumber(series[0].data && series[0].data[i])
  }))
  const max = Math.max.apply(null, values.map((v) => v.value).concat([1]))
  return values.map((row, i) => ({
    label: row.label,
    value: row.value,
    percent: max > 0 ? Math.round((row.value / max) * 100) : 0,
    color: palette.value[i % palette.value.length]
  }))
})

// ---- 环形图 ----
const ringRows = computed(() => {
  const data = props.chartData || {}
  const cats = Array.isArray(data.categories) ? data.categories : []
  const series = Array.isArray(data.series) ? data.series : []
  if (!series.length || !cats.length) return []
  const values = cats.map((label, i) => ({
    label: String(label || ''),
    value: toNumber(series[0].data && series[0].data[i])
  }))
  const total = values.reduce((s, v) => s + v.value, 0)
  if (total <= 0) return values.map((v, i) => ({ ...v, percent: 0, color: palette.value[i % palette.value.length] }))
  return values.map((v, i) => ({
    ...v,
    percent: Math.round((v.value / total) * 1000) / 10,
    color: palette.value[i % palette.value.length]
  }))
})

const ringCenter = computed(() => {
  const total = ringRows.value.reduce((s, v) => s + v.value, 0)
  return total > 0 ? '合计 ' + total : ''
})

const ringStyle = computed(() => {
  const rows = ringRows.value.filter((r) => r.percent > 0)
  if (!rows.length) return {}
  let cursor = 0
  const stops = rows.map((r) => {
    const start = cursor
    cursor += r.percent
    return `${r.color} ${start}% ${cursor}%`
  })
  return { background: 'conic-gradient(' + stops.join(', ') + ')' }
})

function toNumber(n) {
  const num = Number(n)
  return isNaN(num) ? 0 : num
}
</script>

<style lang="scss" scoped>
.simple-chart {
  width: 100%;
  padding: 10rpx 0;
  box-sizing: border-box;
}

/* 横向条形 */
.sc-bar-row {
  display: flex;
  align-items: center;
  padding: 10rpx 0;
}

.sc-bar-label {
  width: 160rpx;
  font-size: $ui-font-xs;
  color: $ui-text-2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sc-bar-track {
  flex: 1;
  height: 28rpx;
  background: $ui-bg-hover;
  border-radius: 999rpx;
  overflow: hidden;
  margin: 0 $ui-space-sm;
}

.sc-bar-fill {
  height: 100%;
  border-radius: 999rpx;
  transition: width 0.3s ease;
}

.sc-bar-value {
  width: 110rpx;
  text-align: right;
  font-size: $ui-font-xs;
  color: $ui-text-1;
}

/* 环形 */
.sc-ring-body {
  display: flex;
  justify-content: center;
  padding: 20rpx 0;
  position: relative;
}

.sc-ring {
  width: 260rpx;
  height: 260rpx;
  border-radius: 50%;
  position: relative;
}

.sc-ring-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: $ui-font-sm;
  color: $ui-text-1;
  font-weight: 600;
}

.sc-legend {
  margin-top: 20rpx;
}

.sc-legend-row {
  display: flex;
  align-items: center;
  padding: 8rpx 0;
}

.sc-legend-dot {
  width: 20rpx;
  height: 20rpx;
  border-radius: 4rpx;
  margin-right: 12rpx;
}

.sc-legend-name {
  flex: 1;
  font-size: $ui-font-sm;
  color: $ui-text-1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sc-legend-value {
  font-size: $ui-font-sm;
  color: $ui-text-3;
}
</style>

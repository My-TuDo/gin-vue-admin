<template>
  <div>
    <!-- 概览卡片 -->
    <el-row :gutter="16" class="mb-4">
      <el-col v-for="ov in overview" :key="ov.label" :span="8">
        <el-card shadow="never">
          <div class="ov-label">{{ ov.label }}</div>
          <div class="ov-row">销售额 <b class="ov-sales">¥ {{ ov.sales.toFixed(2) }}</b></div>
          <div class="ov-row">毛利 <b :class="ov.profit >= 0 ? 'ov-profit' : 'ov-loss'">¥ {{ ov.profit.toFixed(2) }}</b></div>
          <div class="ov-row">订单数 <b>{{ ov.orders }}</b> · 退货额 <b class="ov-loss">¥ {{ ov.returnAmt.toFixed(2) }}</b></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <!-- 趋势 -->
      <el-col :span="14">
        <el-card shadow="never" class="mb-4">
          <template #header>近 {{ trendDays }} 天销售趋势</template>
          <VCharts :option="trendOption" autoresize style="height: 300px; width: 100%" />
        </el-card>
        <!-- 分类占比 -->
        <el-card shadow="never">
          <template #header>近 30 天分类销售占比</template>
          <VCharts :option="categoryOption" autoresize style="height: 280px; width: 100%" />
        </el-card>
      </el-col>

      <!-- 右侧：热销 + 预警 -->
      <el-col :span="10">
        <el-card shadow="never" class="mb-4">
          <template #header>热销商品 TOP{{ topLimit }}</template>
          <VCharts :option="topOption" autoresize style="height: 300px; width: 100%" />
        </el-card>
        <el-card shadow="never">
          <template #header>
            <div class="flex-between">
              <span>库存预警</span>
              <el-tag v-if="stockAlerts.length" type="danger" size="small">{{ stockAlerts.length }} 项</el-tag>
            </div>
          </template>
          <el-table :data="stockAlerts" border size="small" max-height="260">
            <el-table-column label="SKU" prop="skuCode" width="140" />
            <el-table-column label="商品" prop="goodsName" min-width="90" show-overflow-tooltip />
            <el-table-column label="可售" width="70" align="right">
              <template #default="{ row }">
                <span style="color: #f56c6c">{{ row.available }}</span>
              </template>
            </el-table-column>
            <el-table-column label="安全线" width="70" align="right" prop="safeStock" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import VCharts from 'vue-echarts'
import {
  getDashboardOverview, getDashboardTrend, getDashboardTop,
  getDashboardStockAlert, getDashboardCategory,
} from '@/api/jxc/dashboard'

const overview = ref([])
const stockAlerts = ref([])
const trendDays = 14
const topLimit = 10

const trendOption = ref({})
const topOption = ref({})
const categoryOption = ref({})

const loadAll = async () => {
  const [ov, trend, topData, alertData, catData] = await Promise.all([
    getDashboardOverview(),
    getDashboardTrend(trendDays),
    getDashboardTop(30, topLimit),
    getDashboardStockAlert(),
    getDashboardCategory(30),
  ])
  overview.value = ov.data || []
  stockAlerts.value = alertData.data || []

  const t = trend.data || []
  trendOption.value = {
    tooltip: { trigger: 'axis' },
    legend: { data: ['销售额', '订单数'] },
    grid: { left: 60, right: 40, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: t.map((i) => i.date.slice(5)) },
    yAxis: [
      { type: 'value', name: '销售额' },
      { type: 'value', name: '订单数' },
    ],
    series: [
      { name: '销售额', type: 'line', smooth: true, areaStyle: { opacity: 0.15 }, data: t.map((i) => i.sales) },
      { name: '订单数', type: 'bar', yAxisIndex: 1, barWidth: 12, data: t.map((i) => i.orders) },
    ],
  }

  const topList = topData.data || []
  topOption.value = {
    tooltip: { trigger: 'axis' },
    grid: { left: 90, right: 40, top: 30, bottom: 30 },
    xAxis: { type: 'value' },
    yAxis: { type: 'category', data: topList.map((i) => (i.skuCode || '').slice(0, 14)).reverse() },
    series: [{ type: 'bar', barWidth: 14, itemStyle: { color: '#409eff' }, data: topList.map((i) => i.qty).reverse() }],
  }

  const catList = catData.data || []
  categoryOption.value = {
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [{
      type: 'pie', radius: ['35%', '65%'],
      label: { formatter: '{b}: {d}%' },
      data: catList.map((i) => ({ name: i.name || '未分类', value: i.amount })),
    }],
  }
}

onMounted(loadAll)
</script>

<style scoped>
.mb-4 { margin-bottom: 16px; }
.ov-label { color: #909399; font-size: 13px; margin-bottom: 8px; }
.ov-row { margin: 4px 0; font-size: 13px; color: #606266; }
.ov-sales { color: #303133; font-size: 18px; }
.ov-profit { color: #67c23a; }
.ov-loss { color: #f56c6c; }
.flex-between { display: flex; justify-content: space-between; align-items: center; }
</style>

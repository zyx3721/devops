<template>
  <div>
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-select v-model:value="selectedCluster" style="width: 100%" placeholder="全部集群" allowClear @change="fetchData">
          <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
        </a-select>
      </a-col>
      <a-col :span="4">
        <a-select v-model:value="days" style="width: 100%" @change="fetchData">
          <a-select-option :value="7">最近7天</a-select-option>
          <a-select-option :value="30">最近30天</a-select-option>
          <a-select-option :value="90">最近90天</a-select-option>
        </a-select>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="8">
        <a-card size="small">
          <a-statistic title="趋势方向" :value="getTrendText(trend.trend_direction)" :value-style="{ color: getTrendColor(trend.trend_direction) }" />
        </a-card>
      </a-col>
      <a-col :span="8">
        <a-card size="small">
          <a-statistic title="趋势变化" :value="Math.abs(trend.trend_percentage || 0)" :precision="1" suffix="%" />
        </a-card>
      </a-col>
      <a-col :span="8">
        <a-card size="small">
          <a-statistic title="异常点" :value="trend.anomalies?.length || 0" suffix="个" :value-style="{ color: trend.anomalies?.length ? '#ff4d4f' : '#52c41a' }" />
        </a-card>
      </a-col>
    </a-row>

    <a-card title="成本趋势图">
      <div ref="chartRef" style="height: 400px"></div>
    </a-card>

    <a-card title="异常检测" style="margin-top: 16px" v-if="trend.anomalies?.length">
      <a-table :columns="columns" :data-source="trend.anomalies" :pagination="false" size="small">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'deviation'">
            <a-tag :color="record.deviation > 50 ? 'red' : 'orange'">{{ record.deviation?.toFixed(1) }}%</a-tag>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import * as echarts from 'echarts'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'

const selectedCluster = ref<number>()
const days = ref(30)
const clusters = ref<any[]>([])
const trend = ref<any>({ items: [], anomalies: [], prediction: [] })
const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null

const columns = [
  { title: '日期', dataIndex: 'date' },
  { title: '实际成本', dataIndex: 'actual_cost', customRender: ({ text }: any) => `¥${text?.toFixed(2)}` },
  { title: '预期成本', dataIndex: 'expected_cost', customRender: ({ text }: any) => `¥${text?.toFixed(2)}` },
  { title: '偏差', key: 'deviation' },
  { title: '可能原因', dataIndex: 'reason' }
]

const fetchData = async () => {
  const [res, c]: any[] = await Promise.all([
    costApi.getTrend({ cluster_id: selectedCluster.value, days: days.value }),
    k8sClusterApi.list()
  ])
  if (res?.code === 0) { trend.value = res.data || {}; nextTick(renderChart) }
  if (c?.code === 0) clusters.value = c.data?.items || []
}

const renderChart = () => {
  if (!chartRef.value) return
  if (!chart) chart = echarts.init(chartRef.value)
  const items = trend.value.items || []
  const predictions = trend.value.prediction || []
  const allDates = [...items.map((i: any) => i.date), ...predictions.map((p: any) => p.date)]
  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['总成本', '预测', 'CPU', '内存', '存储'] },
    grid: { left: 60, right: 20, bottom: 40 },
    xAxis: { type: 'category', data: allDates },
    yAxis: { type: 'value', name: '成本（元）' },
    series: [
      { name: '总成本', type: 'line', data: items.map((i: any) => i.total_cost), smooth: true, itemStyle: { color: '#1890ff' } },
      { name: '预测', type: 'line', data: [...new Array(items.length).fill(null), ...predictions.map((p: any) => p.predicted_cost)], smooth: true, lineStyle: { type: 'dashed' }, itemStyle: { color: '#52c41a' } },
      { name: 'CPU', type: 'line', data: items.map((i: any) => i.cpu_cost), smooth: true, itemStyle: { color: '#faad14' } },
      { name: '内存', type: 'line', data: items.map((i: any) => i.memory_cost), smooth: true, itemStyle: { color: '#722ed1' } },
      { name: '存储', type: 'line', data: items.map((i: any) => i.storage_cost), smooth: true, itemStyle: { color: '#13c2c2' } }
    ]
  })
}

const getTrendText = (d: string) => ({ up: '上升', down: '下降', stable: '平稳' }[d] || '平稳')
const getTrendColor = (d: string) => ({ up: '#ff4d4f', down: '#52c41a', stable: '#1890ff' }[d] || '#1890ff')

onMounted(fetchData)
</script>

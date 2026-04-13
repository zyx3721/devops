<template>
  <div class="pipeline-stats">
    <a-page-header title="流水线统计" sub-title="执行报表与趋势分析">
      <template #extra>
        <a-space>
          <a-range-picker v-model:value="dateRange" @change="loadStats" />
          <a-button @click="loadStats" :loading="loading">
            <template #icon><ReloadOutlined /></template>
            刷新
          </a-button>
        </a-space>
      </template>
    </a-page-header>

    <!-- 概览卡片 -->
    <a-row :gutter="[16, 16]">
      <a-col :xs="12" :sm="6">
        <a-card :bordered="false">
          <a-statistic title="总执行次数" :value="overview.total" :value-style="{ color: '#1890ff' }">
            <template #prefix><PlayCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="6">
        <a-card :bordered="false">
          <a-statistic title="成功次数" :value="overview.success" :value-style="{ color: '#52c41a' }">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="6">
        <a-card :bordered="false">
          <a-statistic title="失败次数" :value="overview.failed" :value-style="{ color: '#ff4d4f' }">
            <template #prefix><CloseCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="6">
        <a-card :bordered="false">
          <a-statistic title="成功率" :value="overview.successRate" suffix="%" :precision="1" :value-style="{ color: getSuccessRateColor(overview.successRate) }">
            <template #prefix><PercentageOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 图表区域 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :lg="12">
        <a-card title="执行趋势" :bordered="false" :loading="loading">
          <div ref="trendChartRef" style="height: 300px"></div>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="状态分布" :bordered="false" :loading="loading">
          <div ref="statusChartRef" style="height: 300px"></div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :lg="12">
        <a-card title="平均耗时趋势" :bordered="false" :loading="loading">
          <div ref="durationChartRef" style="height: 300px"></div>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="流水线排行" :bordered="false" :loading="loading">
          <a-table :columns="rankColumns" :data-source="pipelineRank" :pagination="false" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <router-link :to="`/pipeline/${record.id}`">{{ record.name }}</router-link>
              </template>
              <template v-if="column.key === 'successRate'">
                <a-progress :percent="record.successRate" :stroke-color="getSuccessRateColor(record.successRate)" size="small" />
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>

    <!-- 最近失败 -->
    <a-card title="最近失败的执行" :bordered="false" style="margin-top: 16px">
      <a-table :columns="failedColumns" :data-source="recentFailed" :pagination="{ pageSize: 5 }" size="small">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'pipeline'">
            <router-link :to="`/pipeline/${record.pipeline_id}`">{{ record.pipeline_name }}</router-link>
          </template>
          <template v-if="column.key === 'status'">
            <a-tag color="red">{{ record.status }}</a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-button type="link" size="small" @click="viewRun(record)">查看详情</a-button>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ReloadOutlined, PlayCircleOutlined, CheckCircleOutlined, CloseCircleOutlined, PercentageOutlined } from '@ant-design/icons-vue'
import * as echarts from 'echarts'
import dayjs, { Dayjs } from 'dayjs'
import request from '@/services/api'

const router = useRouter()
const loading = ref(false)
const dateRange = ref<[Dayjs, Dayjs]>([dayjs().subtract(7, 'day'), dayjs()])

const trendChartRef = ref<HTMLElement | null>(null)
const statusChartRef = ref<HTMLElement | null>(null)
const durationChartRef = ref<HTMLElement | null>(null)
let trendChart: echarts.ECharts | null = null
let statusChart: echarts.ECharts | null = null
let durationChart: echarts.ECharts | null = null

const overview = ref({
  total: 0,
  success: 0,
  failed: 0,
  successRate: 0
})

const pipelineRank = ref<any[]>([])
const recentFailed = ref<any[]>([])

const rankColumns = [
  { title: '流水线', key: 'name', dataIndex: 'name' },
  { title: '执行次数', dataIndex: 'total', width: 100 },
  { title: '成功率', key: 'successRate', width: 200 },
  { title: '平均耗时', dataIndex: 'avgDuration', width: 100 }
]

const failedColumns = [
  { title: '流水线', key: 'pipeline' },
  { title: '执行编号', dataIndex: 'run_number', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '失败原因', dataIndex: 'error_message', ellipsis: true },
  { title: '执行时间', dataIndex: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 100 }
]

const getSuccessRateColor = (rate: number) => {
  if (rate >= 90) return '#52c41a'
  if (rate >= 70) return '#faad14'
  return '#ff4d4f'
}

const loadStats = async () => {
  loading.value = true
  try {
    const [start, end] = dateRange.value
    const params = {
      start_date: start.format('YYYY-MM-DD'),
      end_date: end.format('YYYY-MM-DD')
    }

    // 获取统计数据
    const res = await request.get('/pipelines/stats', { params })
    if (res?.data) {
      const data = res.data
      overview.value = data.overview || overview.value
      pipelineRank.value = data.rank || []
      recentFailed.value = data.recentFailed || []
      
      nextTick(() => {
        if (data.trend && data.trend.length > 0) {
          updateTrendChart(data.trend)
        }
        if (data.statusDistribution) {
          updateStatusChart(data.statusDistribution)
        }
        if (data.durationTrend && data.durationTrend.length > 0) {
          updateDurationChart(data.durationTrend)
        }
      })
    } else {
      // API 返回空数据时使用模拟数据
      loadMockData()
    }
  } catch (error) {
    console.error('加载统计数据失败', error)
    // 使用模拟数据
    loadMockData()
  } finally {
    loading.value = false
  }
}

const loadMockData = () => {
  overview.value = {
    total: 156,
    success: 142,
    failed: 14,
    successRate: 91.0
  }

  pipelineRank.value = [
    { id: 1, name: 'frontend-build', total: 45, successRate: 95, avgDuration: '3m 20s' },
    { id: 2, name: 'backend-deploy', total: 38, successRate: 89, avgDuration: '5m 10s' },
    { id: 3, name: 'api-test', total: 32, successRate: 94, avgDuration: '2m 45s' },
    { id: 4, name: 'docker-build', total: 28, successRate: 86, avgDuration: '8m 30s' },
    { id: 5, name: 'release-prod', total: 13, successRate: 100, avgDuration: '12m 15s' },
  ]

  recentFailed.value = [
    { id: 1, pipeline_id: 2, pipeline_name: 'backend-deploy', run_number: 156, status: 'failed', error_message: 'Build failed: npm install error', created_at: '2026-01-11 10:30:00' },
    { id: 2, pipeline_id: 4, pipeline_name: 'docker-build', run_number: 89, status: 'failed', error_message: 'Docker push timeout', created_at: '2026-01-11 09:15:00' },
    { id: 3, pipeline_id: 2, pipeline_name: 'backend-deploy', run_number: 155, status: 'failed', error_message: 'Test failed: 3 tests failed', created_at: '2026-01-10 16:45:00' },
  ]

  // 模拟趋势数据
  const days = []
  const successData = []
  const failedData = []
  const durationData = []
  for (let i = 6; i >= 0; i--) {
    days.push(dayjs().subtract(i, 'day').format('MM-DD'))
    successData.push(Math.floor(15 + Math.random() * 10))
    failedData.push(Math.floor(Math.random() * 3))
    durationData.push(Math.floor(180 + Math.random() * 120))
  }

  nextTick(() => {
    updateTrendChart(days.map((d, i) => ({ date: d, success: successData[i], failed: failedData[i] })))
    updateStatusChart({ success: 142, failed: 14, running: 0, cancelled: 0 })
    updateDurationChart(days.map((d, i) => ({ date: d, duration: durationData[i] })))
  })
}

const updateTrendChart = (data: any[]) => {
  if (!trendChartRef.value) return
  if (!trendChart) {
    trendChart = echarts.init(trendChartRef.value)
  }
  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['成功', '失败'], bottom: 0 },
    grid: { left: 50, right: 20, top: 20, bottom: 40 },
    xAxis: { type: 'category', data: data.map(d => d.date) },
    yAxis: { type: 'value' },
    series: [
      { name: '成功', type: 'bar', stack: 'total', data: data.map(d => d.success), itemStyle: { color: '#52c41a' } },
      { name: '失败', type: 'bar', stack: 'total', data: data.map(d => d.failed), itemStyle: { color: '#ff4d4f' } }
    ]
  })
}

const updateStatusChart = (data: Record<string, number>) => {
  if (!statusChartRef.value) return
  if (!statusChart) {
    statusChart = echarts.init(statusChartRef.value)
  }
  const pieData = [
    { name: '成功', value: data.success || 0, itemStyle: { color: '#52c41a' } },
    { name: '失败', value: data.failed || 0, itemStyle: { color: '#ff4d4f' } },
    { name: '运行中', value: data.running || 0, itemStyle: { color: '#1890ff' } },
    { name: '已取消', value: data.cancelled || 0, itemStyle: { color: '#d9d9d9' } }
  ].filter(d => d.value > 0)

  statusChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { bottom: 0 },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      center: ['50%', '45%'],
      data: pieData,
      label: { show: true, formatter: '{b}: {c}' }
    }]
  })
}

const updateDurationChart = (data: any[]) => {
  if (!durationChartRef.value) return
  if (!durationChart) {
    durationChart = echarts.init(durationChartRef.value)
  }
  durationChart.setOption({
    tooltip: { trigger: 'axis', formatter: (params: any) => `${params[0].name}<br/>平均耗时: ${Math.floor(params[0].value / 60)}m ${params[0].value % 60}s` },
    grid: { left: 50, right: 20, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: data.map(d => d.date) },
    yAxis: { type: 'value', axisLabel: { formatter: (v: number) => `${Math.floor(v / 60)}m` } },
    series: [{
      type: 'line',
      data: data.map(d => d.duration),
      smooth: true,
      areaStyle: { opacity: 0.3 },
      lineStyle: { color: '#1890ff' },
      itemStyle: { color: '#1890ff' }
    }]
  })
}

const viewRun = (record: any) => {
  router.push(`/pipeline/${record.pipeline_id}?run=${record.id}`)
}

const handleResize = () => {
  trendChart?.resize()
  statusChart?.resize()
  durationChart?.resize()
}

onMounted(() => {
  loadStats()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  trendChart?.dispose()
  statusChart?.dispose()
  durationChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.pipeline-stats {
  padding: 0;
}
</style>

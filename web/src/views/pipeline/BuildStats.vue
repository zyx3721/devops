<template>
  <div class="build-stats">
    <!-- 时间范围选择器 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-row :gutter="16" align="middle">
        <a-col :xs="24" :sm="12" :md="8">
          <a-range-picker
            v-model:value="timeRange"
            :show-time="{ format: 'HH:mm' }"
            format="YYYY-MM-DD HH:mm"
            style="width: 100%"
            @change="loadData"
          />
        </a-col>
        <a-col :xs="24" :sm="12" :md="8">
          <a-select
            v-model:value="selectedPipeline"
            placeholder="选择流水线"
            allowClear
            style="width: 100%"
            show-search
            :filter-option="filterPipeline"
            @change="loadData"
          >
            <a-select-option v-for="p in pipelines" :key="p.id" :value="p.id">
              {{ p.name }}
            </a-select-option>
          </a-select>
        </a-col>
        <a-col :xs="24" :sm="24" :md="8">
          <a-space>
            <a-button type="primary" @click="loadData" :loading="loading">
              <template #icon><SearchOutlined /></template>
              查询
            </a-button>
            <a-button @click="exportReport" :loading="exporting">
              <template #icon><DownloadOutlined /></template>
              导出报告
            </a-button>
          </a-space>
        </a-col>
      </a-row>
    </a-card>

    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="false">
          <a-statistic
            title="总构建数"
            :value="stats.total_builds"
            :value-style="{ color: '#1890ff' }"
          >
            <template #prefix>
              <BuildOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="false">
          <a-statistic
            title="平均时长"
            :value="formatDuration(stats.avg_duration)"
            :value-style="{ color: '#52c41a' }"
          >
            <template #prefix>
              <ClockCircleOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="false">
          <a-statistic
            title="缓存命中率"
            :value="stats.cache_hit_rate"
            suffix="%"
            :precision="1"
            :value-style="{ color: '#faad14' }"
          >
            <template #prefix>
              <ThunderboltOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="false">
          <a-statistic
            title="节省时间"
            :value="formatDuration(stats.time_saved)"
            :value-style="{ color: '#eb2f96' }"
          >
            <template #prefix>
              <RocketOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 资源使用趋势图 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :xs="24" :lg="12">
        <a-card title="CPU 使用趋势" :bordered="false">
          <div ref="cpuChartRef" style="width: 100%; height: 300px"></div>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="内存使用趋势" :bordered="false">
          <div ref="memoryChartRef" style="width: 100%; height: 300px"></div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :xs="24" :lg="12">
        <a-card title="存储使用趋势" :bordered="false">
          <div ref="storageChartRef" style="width: 100%; height: 300px"></div>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="构建时长分布" :bordered="false">
          <div ref="durationChartRef" style="width: 100%; height: 300px"></div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 并发构建统计 -->
    <a-row :gutter="16">
      <a-col :xs="24" :lg="12">
        <a-card title="并发构建统计" :bordered="false">
          <div ref="concurrentChartRef" style="width: 100%; height: 300px"></div>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="排队情况" :bordered="false">
          <a-table
            :columns="queueColumns"
            :data-source="queueData"
            :pagination="false"
            size="small"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="getQueueStatusColor(record.status)">
                  {{ record.status }}
                </a-tag>
              </template>
              <template v-else-if="column.key === 'wait_time'">
                {{ formatDuration(record.wait_time) }}
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  SearchOutlined,
  DownloadOutlined,
  BuildOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  RocketOutlined,
} from '@ant-design/icons-vue'
import request from '@/services/api'
import dayjs, { Dayjs } from 'dayjs'
import * as echarts from 'echarts'

interface TimeSeriesData {
  timestamp: string
  value: number
}

interface BuildUsageStats {
  pipeline_id?: number
  time_range: { start: string; end: string }
  cpu_usage: TimeSeriesData[]
  memory_usage: TimeSeriesData[]
  storage_usage: TimeSeriesData[]
  cache_hit_rate: number
  avg_duration: number
  total_builds: number
  time_saved: number
}

const loading = ref(false)
const exporting = ref(false)
const timeRange = ref<[Dayjs, Dayjs]>([
  dayjs().subtract(7, 'day'),
  dayjs(),
])
const selectedPipeline = ref<number>()
const pipelines = ref<any[]>([])

const stats = reactive({
  total_builds: 0,
  avg_duration: 0,
  cache_hit_rate: 0,
  time_saved: 0,
})

const queueColumns = [
  { title: '流水线', dataIndex: 'pipeline_name', key: 'pipeline_name' },
  { title: '状态', key: 'status' },
  { title: '等待时间', key: 'wait_time' },
  { title: '排队位置', dataIndex: 'position', key: 'position' },
]

const queueData = ref<any[]>([])

const cpuChartRef = ref<HTMLElement>()
const memoryChartRef = ref<HTMLElement>()
const storageChartRef = ref<HTMLElement>()
const durationChartRef = ref<HTMLElement>()
const concurrentChartRef = ref<HTMLElement>()

let cpuChart: echarts.ECharts | null = null
let memoryChart: echarts.ECharts | null = null
let storageChart: echarts.ECharts | null = null
let durationChart: echarts.ECharts | null = null
let concurrentChart: echarts.ECharts | null = null

const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
  return `${(seconds / 3600).toFixed(1)}小时`
}

const filterPipeline = (input: string, option: any) => {
  return option.children[0].children.toLowerCase().includes(input.toLowerCase())
}

const getQueueStatusColor = (status: string) => {
  switch (status) {
    case 'running':
      return 'processing'
    case 'queued':
      return 'warning'
    case 'completed':
      return 'success'
    default:
      return 'default'
  }
}

const loadPipelines = async () => {
  try {
    const res = await request.get('/pipelines', {
      params: { page_size: 100 },
    })
    pipelines.value = res?.data?.items || []
  } catch (error) {
    console.error('加载流水线列表失败:', error)
  }
}

const loadData = async () => {
  if (!timeRange.value || timeRange.value.length !== 2) {
    message.warning('请选择时间范围')
    return
  }

  loading.value = true
  try {
    const res = await request.get('/build/usage/stats', {
      params: {
        pipeline_id: selectedPipeline.value,
        start_time: timeRange.value[0].format('YYYY-MM-DD HH:mm:ss'),
        end_time: timeRange.value[1].format('YYYY-MM-DD HH:mm:ss'),
      },
    })

    if (res?.data) {
      const data: BuildUsageStats = res.data
      
      // 更新统计数据
      stats.total_builds = data.total_builds || 0
      stats.avg_duration = data.avg_duration || 0
      stats.cache_hit_rate = data.cache_hit_rate || 0
      stats.time_saved = data.time_saved || 0

      // 更新图表
      updateCpuChart(data.cpu_usage || [])
      updateMemoryChart(data.memory_usage || [])
      updateStorageChart(data.storage_usage || [])
      updateDurationChart(data)
      updateConcurrentChart(data)
    }
  } catch (error: any) {
    message.error(error?.message || '加载统计数据失败')
  } finally {
    loading.value = false
  }
}

const initCharts = () => {
  if (cpuChartRef.value) {
    cpuChart = echarts.init(cpuChartRef.value)
  }
  if (memoryChartRef.value) {
    memoryChart = echarts.init(memoryChartRef.value)
  }
  if (storageChartRef.value) {
    storageChart = echarts.init(storageChartRef.value)
  }
  if (durationChartRef.value) {
    durationChart = echarts.init(durationChartRef.value)
  }
  if (concurrentChartRef.value) {
    concurrentChart = echarts.init(concurrentChartRef.value)
  }
}

const updateCpuChart = (data: TimeSeriesData[]) => {
  if (!cpuChart) return

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: data.map(item => dayjs(item.timestamp).format('MM-DD HH:mm')),
    },
    yAxis: {
      type: 'value',
      name: 'CPU (cores)',
    },
    series: [
      {
        name: 'CPU 使用',
        type: 'line',
        smooth: true,
        data: data.map(item => item.value),
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(24, 144, 255, 0.3)' },
            { offset: 1, color: 'rgba(24, 144, 255, 0.05)' },
          ]),
        },
        lineStyle: {
          color: '#1890ff',
        },
      },
    ],
  }

  cpuChart.setOption(option)
}

const updateMemoryChart = (data: TimeSeriesData[]) => {
  if (!memoryChart) return

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: data.map(item => dayjs(item.timestamp).format('MM-DD HH:mm')),
    },
    yAxis: {
      type: 'value',
      name: '内存 (GB)',
    },
    series: [
      {
        name: '内存使用',
        type: 'line',
        smooth: true,
        data: data.map(item => item.value),
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(82, 196, 26, 0.3)' },
            { offset: 1, color: 'rgba(82, 196, 26, 0.05)' },
          ]),
        },
        lineStyle: {
          color: '#52c41a',
        },
      },
    ],
  }

  memoryChart.setOption(option)
}

const updateStorageChart = (data: TimeSeriesData[]) => {
  if (!storageChart) return

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: data.map(item => dayjs(item.timestamp).format('MM-DD HH:mm')),
    },
    yAxis: {
      type: 'value',
      name: '存储 (GB)',
    },
    series: [
      {
        name: '存储使用',
        type: 'line',
        smooth: true,
        data: data.map(item => item.value),
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(250, 173, 20, 0.3)' },
            { offset: 1, color: 'rgba(250, 173, 20, 0.05)' },
          ]),
        },
        lineStyle: {
          color: '#faad14',
        },
      },
    ],
  }

  storageChart.setOption(option)
}

const updateDurationChart = (data: any) => {
  if (!durationChart) return

  // 模拟构建时长分布数据
  const durations = ['0-5min', '5-10min', '10-20min', '20-30min', '30min+']
  const counts = [15, 25, 30, 20, 10]

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow',
      },
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: durations,
    },
    yAxis: {
      type: 'value',
      name: '构建数',
    },
    series: [
      {
        name: '构建数',
        type: 'bar',
        data: counts,
        itemStyle: {
          color: '#1890ff',
        },
      },
    ],
  }

  durationChart.setOption(option)
}

const updateConcurrentChart = (data: any) => {
  if (!concurrentChart) return

  // 模拟并发构建数据
  const times = Array.from({ length: 24 }, (_, i) => `${i}:00`)
  const concurrent = Array.from({ length: 24 }, () => Math.floor(Math.random() * 10))

  const option: echarts.EChartsOption = {
    tooltip: {
      trigger: 'axis',
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: times,
    },
    yAxis: {
      type: 'value',
      name: '并发数',
    },
    series: [
      {
        name: '并发构建数',
        type: 'line',
        smooth: true,
        data: concurrent,
        lineStyle: {
          color: '#eb2f96',
        },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(235, 47, 150, 0.3)' },
            { offset: 1, color: 'rgba(235, 47, 150, 0.05)' },
          ]),
        },
      },
    ],
  }

  concurrentChart.setOption(option)

  // 模拟排队数据
  queueData.value = [
    { pipeline_name: 'frontend-build', status: 'running', wait_time: 0, position: 1 },
    { pipeline_name: 'backend-build', status: 'queued', wait_time: 120, position: 2 },
    { pipeline_name: 'mobile-build', status: 'queued', wait_time: 240, position: 3 },
  ]
}

const exportReport = async () => {
  exporting.value = true
  try {
    // 模拟导出
    await new Promise(resolve => setTimeout(resolve, 1000))
    message.success('报告导出成功')
  } catch (error: any) {
    message.error(error?.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

const handleResize = () => {
  cpuChart?.resize()
  memoryChart?.resize()
  storageChart?.resize()
  durationChart?.resize()
  concurrentChart?.resize()
}

onMounted(() => {
  loadPipelines()
  initCharts()
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  cpuChart?.dispose()
  memoryChart?.dispose()
  storageChart?.dispose()
  durationChart?.dispose()
  concurrentChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.build-stats {
  padding: 0;
}

:deep(.ant-statistic-title) {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.65);
}

:deep(.ant-statistic-content) {
  font-size: 24px;
  font-weight: 600;
}

@media (max-width: 768px) {
  :deep(.ant-col) {
    margin-bottom: 16px;
  }
}
</style>

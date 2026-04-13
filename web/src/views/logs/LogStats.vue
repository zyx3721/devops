<template>
  <div class="log-stats">
    <!-- 筛选条件 -->
    <a-card class="filter-card">
      <a-form layout="inline" :model="filterForm">
        <a-form-item label="集群">
          <a-select v-model:value="filterForm.cluster_id" placeholder="选择集群" @change="onClusterChange" style="width: 180px">
            <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">{{ cluster.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="命名空间">
          <a-select v-model:value="filterForm.namespace" placeholder="选择命名空间" style="width: 180px">
            <a-select-option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="时间范围">
          <a-range-picker
            v-model:value="timeRange"
            show-time
            format="YYYY-MM-DD HH:mm:ss"
            style="width: 350px"
          />
        </a-form-item>
        <a-form-item label="时间粒度">
          <a-radio-group v-model:value="filterForm.interval">
            <a-radio-button value="hour">小时</a-radio-button>
            <a-radio-button value="day">天</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="loadStats" :loading="loading">查询</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 统计概览 -->
    <a-row :gutter="20" style="margin-top: 20px">
      <a-col :span="6">
        <a-card class="stat-card">
          <div class="stat-value">{{ formatNumber(stats?.total_count || 0) }}</div>
          <div class="stat-label">总日志数</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card error">
          <div class="stat-value">{{ formatNumber(stats?.level_counts?.ERROR || 0) }}</div>
          <div class="stat-label">错误日志</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card warning">
          <div class="stat-value">{{ formatNumber(stats?.level_counts?.WARN || 0) }}</div>
          <div class="stat-label">警告日志</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card info">
          <div class="stat-value">{{ formatNumber(stats?.level_counts?.INFO || 0) }}</div>
          <div class="stat-label">信息日志</div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="20" style="margin-top: 20px">
      <!-- 日志趋势图 -->
      <a-col :span="16">
        <a-card title="日志量趋势">
          <div ref="trendChartRef" style="height: 350px"></div>
        </a-card>
      </a-col>

      <!-- 级别分布饼图 -->
      <a-col :span="8">
        <a-card title="级别分布">
          <div ref="levelChartRef" style="height: 350px"></div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 错误排行榜 -->
    <a-card style="margin-top: 20px">
      <template #title>
        <div class="card-header">
          <span>错误 Top 10</span>
          <a-button type="link" @click="goToSearch">查看详情</a-button>
        </div>
      </template>
      <a-table :dataSource="stats?.top_errors || []" :columns="columns" :loading="loading" :pagination="false" rowKey="pattern">
        <template #bodyCell="{ column, record, index }">
          <template v-if="column.key === 'index'">
            {{ index + 1 }}
          </template>
          <template v-else-if="column.key === 'count'">
            <a-tag color="error">{{ formatNumber(record.count) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-button type="link" size="small" @click="searchError(record.pattern)">搜索</a-button>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import * as echarts from 'echarts'
import dayjs, { Dayjs } from 'dayjs'
import { k8sApi } from '@/services/k8s'
import { logApi } from '@/services/logs'

interface Cluster {
  id: number
  name: string
}

interface LogStats {
  total_count: number
  level_counts: Record<string, number>
  trend: Array<{ time: string; count: number; level: string }>
  top_errors: Array<{ pattern: string; count: number; sample: string }>
}

const router = useRouter()
const clusters = ref<Cluster[]>([])
const namespaces = ref<string[]>([])
const stats = ref<LogStats | null>(null)
const loading = ref(false)

const filterForm = reactive({
  cluster_id: null as number | null,
  namespace: '',
  interval: 'hour'
})

const timeRange = ref<[Dayjs, Dayjs] | null>(null)

const trendChartRef = ref<HTMLElement | null>(null)
const levelChartRef = ref<HTMLElement | null>(null)
let trendChart: echarts.ECharts | null = null
let levelChart: echarts.ECharts | null = null

const columns = [
  {
    title: '#',
    key: 'index',
    width: 50,
  },
  {
    title: '错误模式',
    dataIndex: 'pattern',
    minWidth: 300,
    ellipsis: true,
  },
  {
    title: '出现次数',
    key: 'count',
    width: 120,
  },
  {
    title: '示例',
    dataIndex: 'sample',
    minWidth: 400,
    ellipsis: true,
  },
  {
    title: '操作',
    key: 'action',
    width: 100,
  },
]

const loadClusters = async () => {
  try {
    const res = await k8sApi.getClusters()
    clusters.value = res.data || []
  } catch (error) {
    message.error('加载集群列表失败')
  }
}

const onClusterChange = async () => {
  filterForm.namespace = ''
  namespaces.value = []
  
  if (!filterForm.cluster_id) return
  
  try {
    const res = await k8sApi.getNamespaces(filterForm.cluster_id)
    namespaces.value = res.data || []
  } catch (error) {
    message.error('加载命名空间失败')
  }
}

const loadStats = async () => {
  if (!filterForm.cluster_id || !filterForm.namespace) {
    message.warning('请选择集群和命名空间')
    return
  }

  loading.value = true
  try {
    const params: any = {
      cluster_id: filterForm.cluster_id,
      namespace: filterForm.namespace,
      interval: filterForm.interval
    }

    if (timeRange.value) {
      params.start_time = timeRange.value[0].toISOString()
      params.end_time = timeRange.value[1].toISOString()
    }

    const res = await logApi.getLogStats(params)
    stats.value = res.data

    nextTick(() => {
      renderTrendChart()
      renderLevelChart()
    })
  } catch (error) {
    message.error('加载统计数据失败')
  } finally {
    loading.value = false
  }
}

const renderTrendChart = () => {
  if (!trendChartRef.value || !stats.value) return

  if (!trendChart) {
    trendChart = echarts.init(trendChartRef.value)
  }

  // 处理趋势数据
  const timeSet = new Set<string>()
  const levelData: Record<string, Record<string, number>> = {}

  stats.value.trend.forEach(item => {
    timeSet.add(item.time)
    if (!levelData[item.level]) {
      levelData[item.level] = {}
    }
    levelData[item.level][item.time] = item.count
  })

  const times = Array.from(timeSet).sort()
  const series = Object.keys(levelData).map(level => ({
    name: level,
    type: 'line',
    stack: 'total',
    areaStyle: {},
    data: times.map(t => levelData[level][t] || 0)
  }))

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' }
    },
    legend: {
      data: Object.keys(levelData)
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: times
    },
    yAxis: {
      type: 'value'
    },
    series
  }

  trendChart.setOption(option)
}

const renderLevelChart = () => {
  if (!levelChartRef.value || !stats.value) return

  if (!levelChart) {
    levelChart = echarts.init(levelChartRef.value)
  }

  const levelColors: Record<string, string> = {
    ERROR: '#F56C6C',
    FATAL: '#F56C6C',
    WARN: '#E6A23C',
    WARNING: '#E6A23C',
    INFO: '#409EFF',
    DEBUG: '#909399',
    UNKNOWN: '#C0C4CC'
  }

  const data = Object.entries(stats.value.level_counts).map(([name, value]) => ({
    name,
    value,
    itemStyle: { color: levelColors[name] || '#909399' }
  }))

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 20,
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data
      }
    ]
  }

  levelChart.setOption(option)
}

const formatNumber = (num: number) => {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

const goToSearch = () => {
  router.push({
    path: '/logs/search',
    query: {
      cluster_id: filterForm.cluster_id?.toString(),
      namespace: filterForm.namespace,
      level: 'ERROR'
    }
  })
}

const searchError = (pattern: string) => {
  router.push({
    path: '/logs/search',
    query: {
      cluster_id: filterForm.cluster_id?.toString(),
      namespace: filterForm.namespace,
      keyword: pattern.replace(/<[^>]+>/g, '').substring(0, 50)
    }
  })
}

// 监听窗口大小变化
const handleResize = () => {
  trendChart?.resize()
  levelChart?.resize()
}

onMounted(() => {
  loadClusters()
  window.addEventListener('resize', handleResize)
})

// 清理
watch(() => router.currentRoute.value.path, () => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.log-stats {
  padding: 20px;
}

.filter-card {
  margin-bottom: 20px;
}

.stat-card {
  text-align: center;
  padding: 20px;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #1890ff;
}

.stat-card.error .stat-value {
  color: #F56C6C;
}

.stat-card.warning .stat-value {
  color: #E6A23C;
}

.stat-card.info .stat-value {
  color: #409EFF;
}

.stat-label {
  margin-top: 10px;
  color: rgba(0, 0, 0, 0.45);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

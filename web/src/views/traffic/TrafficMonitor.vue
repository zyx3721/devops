<template>
  <div class="traffic-monitor">
    <a-page-header title="流量监控" sub-title="实时监控流量治理效果" />

    <!-- 统计概览 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="总请求数" :value="stats.total_requests" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="成功率" :value="successRate" suffix="%" :value-style="{ color: successRate >= 95 ? '#52c41a' : '#ff4d4f' }" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="限流次数" :value="stats.rate_limited_count" :value-style="{ color: stats.rate_limited_count > 0 ? '#faad14' : '#52c41a' }" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="熔断状态" :value="stats.circuit_breaker_open ? '已熔断' : '正常'" :value-style="{ color: stats.circuit_breaker_open ? '#ff4d4f' : '#52c41a' }" />
        </a-card>
      </a-col>
    </a-row>

    <!-- 熔断状态 -->
    <a-card title="熔断器状态" style="margin-bottom: 16px">
      <a-table :columns="cbColumns" :data-source="circuitBreakers" :loading="loading" :pagination="false" size="small">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-badge :status="record.status === 'closed' ? 'success' : record.status === 'open' ? 'error' : 'warning'" :text="getStatusText(record.status)" />
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-tag :color="record.enabled ? 'green' : 'default'">{{ record.enabled ? '启用' : '禁用' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'last_open_time'">
            {{ record.last_open_time ? formatTime(record.last_open_time) : '-' }}
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 流量分布 -->
    <a-card title="流量分布" style="margin-bottom: 16px">
      <a-row :gutter="16">
        <a-col :span="12">
          <div ref="trafficChartRef" style="height: 300px"></div>
        </a-col>
        <a-col :span="12">
          <a-table :columns="distributionColumns" :data-source="trafficDistribution" :pagination="false" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'percentage'">
                <a-progress :percent="record.percentage" size="small" style="width: 150px" />
              </template>
            </template>
          </a-table>
        </a-col>
      </a-row>
    </a-card>

    <!-- Istio 资源 -->
    <a-card title="Istio 资源">
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="vs" tab="VirtualService">
          <a-table :columns="istioColumns" :data-source="virtualServices" :loading="loadingIstio" :pagination="false" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a-typography-text code>{{ record.metadata?.name }}</a-typography-text>
              </template>
              <template v-else-if="column.key === 'hosts'">
                <a-tag v-for="host in (record.spec?.hosts || [])" :key="host">{{ host }}</a-tag>
              </template>
              <template v-else-if="column.key === 'created'">
                {{ formatTime(record.metadata?.creationTimestamp) }}
              </template>
            </template>
          </a-table>
        </a-tab-pane>
        <a-tab-pane key="dr" tab="DestinationRule">
          <a-table :columns="istioColumns" :data-source="destinationRules" :loading="loadingIstio" :pagination="false" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a-typography-text code>{{ record.metadata?.name }}</a-typography-text>
              </template>
              <template v-else-if="column.key === 'hosts'">
                <a-tag>{{ record.spec?.host }}</a-tag>
              </template>
              <template v-else-if="column.key === 'created'">
                {{ formatTime(record.metadata?.creationTimestamp) }}
              </template>
            </template>
          </a-table>
        </a-tab-pane>
        <a-tab-pane key="gw" tab="Gateway">
          <a-table :columns="istioColumns" :data-source="gateways" :loading="loadingIstio" :pagination="false" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <a-typography-text code>{{ record.metadata?.name }}</a-typography-text>
              </template>
              <template v-else-if="column.key === 'hosts'">
                <template v-for="server in (record.spec?.servers || [])" :key="server">
                  <a-tag v-for="host in (server.hosts || [])" :key="host">{{ host }}</a-tag>
                </template>
              </template>
              <template v-else-if="column.key === 'created'">
                {{ formatTime(record.metadata?.creationTimestamp) }}
              </template>
            </template>
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import * as echarts from 'echarts'
import { trafficApi, type TrafficStatistics, type CircuitBreakerStatus } from '@/services/traffic'

const route = useRoute()
const appId = computed(() => Number(route.params.id || route.query.appId) || 0)

const loading = ref(false)
const loadingIstio = ref(false)
const activeTab = ref('vs')

const stats = ref<TrafficStatistics>({
  total_requests: 0,
  success_requests: 0,
  failed_requests: 0,
  rate_limited_count: 0,
  circuit_breaker_open: false,
  avg_latency_ms: 0,
  p99_latency_ms: 0,
  traffic_distribution: {}
})

const circuitBreakers = ref<CircuitBreakerStatus[]>([])
const virtualServices = ref<any[]>([])
const destinationRules = ref<any[]>([])
const gateways = ref<any[]>([])

const trafficChartRef = ref<HTMLElement | null>(null)
let trafficChart: echarts.ECharts | null = null

const successRate = computed(() => {
  if (stats.value.total_requests === 0) return 100
  return Math.round((stats.value.success_requests / stats.value.total_requests) * 10000) / 100
})

const trafficDistribution = computed(() => {
  const dist = stats.value.traffic_distribution || {}
  return Object.entries(dist).map(([subset, percentage]) => ({
    subset,
    percentage: Math.round(percentage as number)
  }))
})

const cbColumns = [
  { title: '规则名称', dataIndex: 'name', key: 'name' },
  { title: '资源', dataIndex: 'resource', key: 'resource' },
  { title: '状态', key: 'status' },
  { title: '启用', key: 'enabled' },
  { title: '上次熔断', key: 'last_open_time' }
]

const distributionColumns = [
  { title: '子集', dataIndex: 'subset', key: 'subset' },
  { title: '流量占比', key: 'percentage' }
]

const istioColumns = [
  { title: '名称', key: 'name' },
  { title: 'Hosts', key: 'hosts' },
  { title: '创建时间', key: 'created' }
]

const fetchStats = async () => {
  if (!appId.value) return
  loading.value = true
  try {
    const res = await trafficApi.getStats(appId.value)
    if (res?.data) {
      stats.value = res.data
      updateChart()
    }
  } catch (error) {
    console.error('获取统计失败', error)
  } finally {
    loading.value = false
  }
}

const fetchCircuitBreakers = async () => {
  if (!appId.value) return
  try {
    const res = await trafficApi.getCircuitBreakerStatus(appId.value)
    if (res?.data) {
      circuitBreakers.value = res.data.items || []
    }
  } catch (error) {
    console.error('获取熔断状态失败', error)
  }
}

const fetchIstioResources = async () => {
  if (!appId.value) return
  loadingIstio.value = true
  try {
    const [vsRes, drRes, gwRes] = await Promise.all([
      trafficApi.getVirtualServices(appId.value),
      trafficApi.getDestinationRules(appId.value),
      trafficApi.getGateways(appId.value)
    ])
    virtualServices.value = vsRes?.data?.items || []
    destinationRules.value = drRes?.data?.items || []
    gateways.value = gwRes?.data?.items || []
  } catch (error) {
    console.error('获取Istio资源失败', error)
  } finally {
    loadingIstio.value = false
  }
}

const initChart = () => {
  if (!trafficChartRef.value) return
  trafficChart = echarts.init(trafficChartRef.value)
  updateChart()
}

const updateChart = () => {
  if (!trafficChart) return
  const dist = stats.value.traffic_distribution || {}
  const data = Object.entries(dist).map(([name, value]) => ({ name, value }))
  
  if (data.length === 0) {
    data.push({ name: '默认', value: 100 })
  }

  trafficChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c}%' },
    legend: { orient: 'vertical', left: 'left' },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      avoidLabelOverlap: false,
      label: { show: true, formatter: '{b}\n{c}%' },
      data
    }]
  })
}

const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    closed: '关闭',
    open: '打开',
    half_open: '半开'
  }
  return map[status] || status
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return time.replace('T', ' ').substring(0, 19)
}

let refreshTimer: number | null = null

onMounted(() => {
  fetchStats()
  fetchCircuitBreakers()
  fetchIstioResources()
  initChart()
  refreshTimer = window.setInterval(() => {
    fetchStats()
    fetchCircuitBreakers()
  }, 30000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (trafficChart) trafficChart.dispose()
})
</script>

<style scoped>
.traffic-monitor {
  padding: 16px;
}
</style>

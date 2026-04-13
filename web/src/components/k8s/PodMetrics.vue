<template>
  <div class="pod-metrics">
    <a-spin :spinning="loading">
      <template v-if="!metricsAvailable">
        <a-alert
          type="warning"
          :message="metricsMessage || 'metrics-server 不可用'"
          show-icon
        />
      </template>
      <template v-else-if="metrics">
        <div class="metrics-summary">
          <a-statistic title="总 CPU 使用" :value="formatCPU(metrics.total_cpu)" />
          <a-statistic title="总内存使用" :value="formatMemory(metrics.total_mem)" />
        </div>
        
        <a-divider orientation="left">容器资源使用</a-divider>
        
        <div class="container-metrics" v-for="container in metrics.containers" :key="container.name">
          <div class="container-name">{{ container.name }}</div>
          <div class="metrics-row">
            <div class="metric-item">
              <span class="metric-label">CPU</span>
              <a-progress
                :percent="container.cpu_percent || 0"
                :status="getProgressStatus(container.cpu_percent)"
                size="small"
              />
              <span class="metric-value">{{ formatCPU(container.cpu_usage) }} / {{ formatCPU(container.cpu_limit) || '无限制' }}</span>
            </div>
            <div class="metric-item">
              <span class="metric-label">内存</span>
              <a-progress
                :percent="container.mem_percent || 0"
                :status="getProgressStatus(container.mem_percent)"
                size="small"
              />
              <span class="metric-value">{{ formatMemory(container.mem_usage) }} / {{ formatMemory(container.mem_limit) || '无限制' }}</span>
            </div>
          </div>
        </div>
      </template>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { k8sMetricsApi, formatCPU, formatMemory, type PodMetricsResponse } from '@/services/k8s'

const props = defineProps<{
  clusterId: number
  namespace: string
  podName: string
  autoRefresh?: boolean
  refreshInterval?: number
}>()

const loading = ref(false)
const metrics = ref<PodMetricsResponse | null>(null)
const metricsAvailable = ref(true)
const metricsMessage = ref('')
let refreshTimer: number | null = null

const fetchMetrics = async () => {
  if (!props.clusterId || !props.namespace || !props.podName) return
  
  loading.value = true
  try {
    const res = await k8sMetricsApi.getPodMetrics(props.clusterId, props.namespace, props.podName)
    if (res?.data) {
      metrics.value = res.data
      metricsAvailable.value = res.data.available
      metricsMessage.value = res.data.message || ''
    }
  } catch (error) {
    console.error('获取 Pod 指标失败:', error)
    metricsAvailable.value = false
    metricsMessage.value = '获取指标失败'
  } finally {
    loading.value = false
  }
}

const getProgressStatus = (percent: number | undefined) => {
  if (!percent) return 'normal'
  if (percent >= 90) return 'exception'
  if (percent >= 70) return 'active'
  return 'normal'
}

const startAutoRefresh = () => {
  if (props.autoRefresh && props.refreshInterval) {
    refreshTimer = window.setInterval(fetchMetrics, props.refreshInterval * 1000)
  }
}

const stopAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

watch(() => [props.clusterId, props.namespace, props.podName], () => {
  fetchMetrics()
})

onMounted(() => {
  fetchMetrics()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.pod-metrics {
  padding: 16px;
}

.metrics-summary {
  display: flex;
  gap: 48px;
  margin-bottom: 16px;
}

.container-metrics {
  margin-bottom: 16px;
  padding: 12px;
  background: #fafafa;
  border-radius: 6px;
}

.container-name {
  font-weight: 500;
  margin-bottom: 8px;
  color: #333;
}

.metrics-row {
  display: flex;
  gap: 24px;
}

.metric-item {
  flex: 1;
}

.metric-label {
  display: block;
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
}

.metric-value {
  display: block;
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}
</style>

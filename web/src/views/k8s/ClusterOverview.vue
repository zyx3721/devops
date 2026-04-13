<template>
  <div class="cluster-overview">
    <a-page-header title="集群概览" sub-title="多集群资源统计与健康状态">
      <template #extra>
        <a-button type="primary" @click="fetchData" :loading="loading">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </template>
    </a-page-header>

    <!-- 汇总统计 -->
    <a-row :gutter="16" class="summary-cards">
      <a-col :span="4">
        <a-statistic title="集群总数" :value="overview?.summary.total_clusters || 0">
          <template #prefix><CloudServerOutlined /></template>
        </a-statistic>
      </a-col>
      <a-col :span="4">
        <a-statistic title="健康集群" :value="overview?.summary.healthy_clusters || 0" :value-style="{ color: '#52c41a' }">
          <template #prefix><CheckCircleOutlined /></template>
        </a-statistic>
      </a-col>
      <a-col :span="4">
        <a-statistic title="节点总数" :value="overview?.summary.total_nodes || 0">
          <template #prefix><ClusterOutlined /></template>
        </a-statistic>
      </a-col>
      <a-col :span="4">
        <a-statistic title="Pod 总数" :value="overview?.summary.total_pods || 0">
          <template #prefix><AppstoreOutlined /></template>
        </a-statistic>
      </a-col>
      <a-col :span="4">
        <a-statistic title="Deployment 总数" :value="overview?.summary.total_deployments || 0">
          <template #prefix><DeploymentUnitOutlined /></template>
        </a-statistic>
      </a-col>
    </a-row>

    <!-- 资源使用对比图表 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :lg="12">
        <a-card title="集群资源对比" :bordered="false">
          <div ref="resourceChartRef" style="height: 280px"></div>
        </a-card>
      </a-col>
      <a-col :xs="24" :lg="12">
        <a-card title="集群健康状态" :bordered="false">
          <div ref="healthChartRef" style="height: 280px"></div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 集群列表 -->
    <a-card title="集群详情" class="cluster-list">
      <a-row :gutter="[16, 16]">
        <a-col :span="8" v-for="cluster in overview?.clusters" :key="cluster.cluster_id">
          <a-card :bordered="true" size="small" :class="['cluster-card', cluster.status]">
            <template #title>
              <div class="cluster-title">
                <a-badge :status="getStatusBadge(cluster.status)" />
                <span>{{ cluster.cluster_name }}</span>
              </div>
            </template>
            <template #extra>
              <a-tag :color="getStatusColor(cluster.status)">{{ cluster.status }}</a-tag>
            </template>

            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="节点">
                {{ cluster.node_ready }}/{{ cluster.node_total }}
              </a-descriptions-item>
              <a-descriptions-item label="Pod">
                {{ cluster.pod_used }}/{{ cluster.pod_capacity }}
              </a-descriptions-item>
              <a-descriptions-item label="CPU">
                {{ cluster.cpu_capacity }}
              </a-descriptions-item>
              <a-descriptions-item label="内存">
                {{ cluster.memory_capacity }}
              </a-descriptions-item>
              <a-descriptions-item label="Deployment">
                {{ cluster.deployment_ready }}/{{ cluster.deployment_total }}
              </a-descriptions-item>
              <a-descriptions-item label="StatefulSet">
                {{ cluster.statefulset_ready }}/{{ cluster.statefulset_total }}
              </a-descriptions-item>
            </a-descriptions>

            <div class="cluster-actions">
              <a-space>
                <a-button type="link" size="small" @click="viewCluster(cluster.cluster_id)">
                  资源管理
                </a-button>
                <a-button type="link" size="small" @click="viewPods(cluster.cluster_id)">
                  Pod 管理
                </a-button>
                <a-button type="link" size="small" @click="viewDeployments(cluster.cluster_id)">
                  Deployment
                </a-button>
              </a-space>
            </div>
          </a-card>
        </a-col>
      </a-row>

      <a-empty v-if="!loading && (!overview?.clusters || overview.clusters.length === 0)" description="暂无集群数据" />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  ReloadOutlined,
  CloudServerOutlined,
  CheckCircleOutlined,
  ClusterOutlined,
  AppstoreOutlined,
  DeploymentUnitOutlined
} from '@ant-design/icons-vue'
import * as echarts from 'echarts'
import { k8sOverviewApi, type MultiClusterOverview } from '@/services/k8s'

const router = useRouter()
const loading = ref(false)
const overview = ref<MultiClusterOverview | null>(null)

const resourceChartRef = ref<HTMLElement | null>(null)
const healthChartRef = ref<HTMLElement | null>(null)
let resourceChart: echarts.ECharts | null = null
let healthChart: echarts.ECharts | null = null

const fetchData = async () => {
  loading.value = true
  try {
    const res = await k8sOverviewApi.getMultiClusterOverview()
    if (res.code === 0) {
      overview.value = res.data
      nextTick(() => {
        updateCharts()
      })
    } else {
      message.error(res.message || '获取集群概览失败')
    }
  } catch (error) {
    message.error('获取集群概览失败')
  } finally {
    loading.value = false
  }
}

const updateCharts = () => {
  if (!overview.value?.clusters) return

  // 资源对比图
  if (resourceChartRef.value) {
    if (!resourceChart) {
      resourceChart = echarts.init(resourceChartRef.value)
    }
    const clusters = overview.value.clusters
    resourceChart.setOption({
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      legend: { data: ['节点数', 'Pod 数', 'Deployment'], bottom: 0 },
      grid: { left: 50, right: 20, top: 20, bottom: 40 },
      xAxis: { type: 'category', data: clusters.map(c => c.cluster_name) },
      yAxis: { type: 'value' },
      series: [
        { name: '节点数', type: 'bar', data: clusters.map(c => c.node_total), itemStyle: { color: '#1890ff' } },
        { name: 'Pod 数', type: 'bar', data: clusters.map(c => c.pod_used), itemStyle: { color: '#52c41a' } },
        { name: 'Deployment', type: 'bar', data: clusters.map(c => c.deployment_total), itemStyle: { color: '#faad14' } }
      ]
    })
  }

  // 健康状态图
  if (healthChartRef.value) {
    if (!healthChart) {
      healthChart = echarts.init(healthChartRef.value)
    }
    const clusters = overview.value.clusters
    const connected = clusters.filter(c => c.status === 'connected').length
    const disconnected = clusters.filter(c => c.status === 'disconnected').length
    const unknown = clusters.filter(c => c.status === 'unknown').length

    healthChart.setOption({
      tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
      legend: { bottom: 0 },
      series: [{
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['50%', '45%'],
        data: [
          { name: '健康', value: connected, itemStyle: { color: '#52c41a' } },
          { name: '断开', value: disconnected, itemStyle: { color: '#ff4d4f' } },
          { name: '未知', value: unknown, itemStyle: { color: '#faad14' } }
        ].filter(d => d.value > 0),
        label: { show: true, formatter: '{b}: {c}' }
      }]
    })
  }
}

const getStatusBadge = (status: string) => {
  const map: Record<string, 'success' | 'error' | 'warning' | 'default'> = {
    connected: 'success',
    disconnected: 'error',
    unknown: 'warning'
  }
  return map[status] || 'default'
}

const getStatusColor = (status: string) => {
  const map: Record<string, string> = {
    connected: 'green',
    disconnected: 'red',
    unknown: 'orange'
  }
  return map[status] || 'default'
}

const viewCluster = (clusterId: number) => {
  router.push(`/k8s/clusters/${clusterId}/resources`)
}

const viewPods = (clusterId: number) => {
  router.push(`/k8s/clusters/${clusterId}/pods`)
}

const viewDeployments = (clusterId: number) => {
  router.push(`/k8s/clusters/${clusterId}/deployments`)
}

const handleResize = () => {
  resourceChart?.resize()
  healthChart?.resize()
}

onMounted(() => {
  fetchData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  resourceChart?.dispose()
  healthChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.cluster-overview {
  padding: 16px;
}

.summary-cards {
  margin-bottom: 24px;
  padding: 16px;
  background: #fff;
  border-radius: 8px;
}

.cluster-list {
  margin-top: 16px;
}

.cluster-card {
  transition: all 0.3s;
}

.cluster-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.cluster-card.disconnected {
  border-color: #ff4d4f;
}

.cluster-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cluster-actions {
  margin-top: 12px;
  text-align: right;
}
</style>

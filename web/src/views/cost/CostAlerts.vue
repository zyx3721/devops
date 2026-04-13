<template>
  <div class="cost-alerts">
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-select v-model:value="selectedCluster" style="width: 100%" placeholder="全部集群" allowClear @change="fetchData">
          <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
        </a-select>
      </a-col>
      <a-col :span="4">
        <a-select v-model:value="selectedStatus" style="width: 100%" placeholder="全部状态" allowClear @change="fetchData">
          <a-select-option value="active">活跃</a-select-option>
          <a-select-option value="acknowledged">已确认</a-select-option>
          <a-select-option value="resolved">已解决</a-select-option>
        </a-select>
      </a-col>
      <a-col :span="14" style="text-align: right">
        <a-button @click="fetchData">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </a-col>
    </a-row>

    <!-- 告警统计 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="活跃告警" :value="stats.active" :value-style="{ color: '#ff4d4f' }">
            <template #prefix><AlertOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="严重告警" :value="stats.critical" :value-style="{ color: '#ff4d4f' }">
            <template #prefix><WarningOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="警告" :value="stats.warning" :value-style="{ color: '#faad14' }">
            <template #prefix><ExclamationCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="已确认" :value="stats.acknowledged" :value-style="{ color: '#52c41a' }">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 告警列表 -->
    <a-card title="告警列表">
      <a-table :columns="columns" :data-source="alerts" :loading="loading" row-key="id" :pagination="{ pageSize: 20 }">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'severity'">
            <a-tag :color="getSeverityColor(record.severity)">{{ getSeverityText(record.severity) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'alert_type'">
            <span>{{ getAlertTypeText(record.alert_type) }}</span>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">{{ getStatusText(record.status) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'threshold'">
            <span v-if="record.threshold">{{ record.threshold.toFixed(1) }}%</span>
            <span v-else>-</span>
          </template>
          <template v-else-if="column.key === 'actual_value'">
            <span :style="{ color: record.actual_value > record.threshold ? '#ff4d4f' : '#52c41a' }">
              {{ record.actual_value?.toFixed(1) || '-' }}%
            </span>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-button v-if="record.status === 'active'" type="link" size="small" @click="acknowledgeAlert(record.id)">
              确认
            </a-button>
            <a-button type="link" size="small" @click="showDetail(record)">详情</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 详情弹窗 -->
    <a-modal v-model:open="detailVisible" title="告警详情" :footer="null" width="600px">
      <a-descriptions :column="1" bordered v-if="currentAlert">
        <a-descriptions-item label="告警类型">{{ getAlertTypeText(currentAlert.alert_type) }}</a-descriptions-item>
        <a-descriptions-item label="严重程度">
          <a-tag :color="getSeverityColor(currentAlert.severity)">{{ getSeverityText(currentAlert.severity) }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="标题">{{ currentAlert.title }}</a-descriptions-item>
        <a-descriptions-item label="详细信息">
          <pre style="white-space: pre-wrap; margin: 0">{{ currentAlert.message }}</pre>
        </a-descriptions-item>
        <a-descriptions-item label="阈值">{{ currentAlert.threshold?.toFixed(1) || '-' }}%</a-descriptions-item>
        <a-descriptions-item label="实际值">{{ currentAlert.actual_value?.toFixed(1) || '-' }}%</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="getStatusColor(currentAlert.status)">{{ getStatusText(currentAlert.status) }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="创建时间">{{ currentAlert.created_at }}</a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ReloadOutlined, AlertOutlined, WarningOutlined, ExclamationCircleOutlined, CheckCircleOutlined } from '@ant-design/icons-vue'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'

interface Alert {
  id: number
  cluster_id: number
  alert_type: string
  severity: string
  title: string
  message: string
  threshold: number
  actual_value: number
  status: string
  created_at: string
}

const selectedCluster = ref<number>()
const selectedStatus = ref<string>()
const clusters = ref<any[]>([])
const alerts = ref<Alert[]>([])
const loading = ref(false)
const detailVisible = ref(false)
const currentAlert = ref<Alert | null>(null)

const columns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 160 },
  { title: '类型', dataIndex: 'alert_type', key: 'alert_type', width: 100 },
  { title: '严重程度', dataIndex: 'severity', key: 'severity', width: 90 },
  { title: '标题', dataIndex: 'title', key: 'title' },
  { title: '阈值', dataIndex: 'threshold', key: 'threshold', width: 80 },
  { title: '实际值', dataIndex: 'actual_value', key: 'actual_value', width: 80 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const stats = computed(() => {
  const active = alerts.value.filter(a => a.status === 'active').length
  const critical = alerts.value.filter(a => a.severity === 'critical' && a.status === 'active').length
  const warning = alerts.value.filter(a => a.severity === 'warning' && a.status === 'active').length
  const acknowledged = alerts.value.filter(a => a.status === 'acknowledged').length
  return { active, critical, warning, acknowledged }
})

const fetchData = async () => {
  loading.value = true
  try {
    const res: any = await costApi.getAlerts(selectedCluster.value, selectedStatus.value)
    if (res?.code === 0) {
      alerts.value = res.data || []
    }
  } finally {
    loading.value = false
  }
}

const fetchClusters = async () => {
  const res: any = await k8sClusterApi.list()
  if (res?.code === 0) {
    clusters.value = res.data?.items || []
  }
}

const acknowledgeAlert = async (id: number) => {
  try {
    const res: any = await costApi.acknowledgeAlert(id)
    if (res?.code === 0) {
      message.success('确认成功')
      fetchData()
    }
  } catch (e) {
    message.error('操作失败')
  }
}

const showDetail = (alert: Alert) => {
  currentAlert.value = alert
  detailVisible.value = true
}

const getSeverityColor = (s: string) => ({ critical: 'red', warning: 'orange', info: 'blue' }[s] || 'default')
const getSeverityText = (s: string) => ({ critical: '严重', warning: '警告', info: '信息' }[s] || s)
const getAlertTypeText = (t: string) => ({
  budget_warning: '预算预警',
  budget_exceeded: '预算超支',
  anomaly: '成本异常',
  waste: '资源浪费'
}[t] || t)
const getStatusColor = (s: string) => ({ active: 'red', acknowledged: 'green', resolved: 'default' }[s] || 'default')
const getStatusText = (s: string) => ({ active: '活跃', acknowledged: '已确认', resolved: '已解决' }[s] || s)

onMounted(async () => {
  await fetchClusters()
  fetchData()
})
</script>

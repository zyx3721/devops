<template>
  <div>
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-select v-model:value="selectedCluster" style="width: 100%" placeholder="全部集群" allowClear @change="fetchData">
          <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
        </a-select>
      </a-col>
      <a-col :span="4">
        <a-select v-model:value="status" style="width: 100%" @change="fetchData">
          <a-select-option value="">全部状态</a-select-option>
          <a-select-option value="pending">待处理</a-select-option>
          <a-select-option value="applied">已应用</a-select-option>
          <a-select-option value="ignored">已忽略</a-select-option>
        </a-select>
      </a-col>
    </a-row>

    <a-alert v-if="suggestions.total_savings > 0" type="success" show-icon style="margin-bottom: 16px">
      <template #message>共 {{ suggestions.items?.length || 0 }} 条优化建议，预计可节省 <b>¥{{ suggestions.total_savings?.toFixed(2) }}</b></template>
    </a-alert>

    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="8"><a-tag color="red">高优先级: {{ suggestions.high_count || 0 }}</a-tag></a-col>
      <a-col :span="8"><a-tag color="orange">中优先级: {{ suggestions.medium_count || 0 }}</a-tag></a-col>
      <a-col :span="8"><a-tag color="blue">低优先级: {{ suggestions.low_count || 0 }}</a-tag></a-col>
    </a-row>

    <a-table :columns="columns" :data-source="suggestions.items" :loading="loading" size="small">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'resource'">
          <div>{{ record.resource_name }}</div>
          <div style="color: #999; font-size: 12px">{{ record.namespace }} / {{ record.resource_type }}</div>
        </template>
        <template v-if="column.key === 'severity'">
          <a-tag :color="getSeverityColor(record.severity)">{{ getSeverityText(record.severity) }}</a-tag>
        </template>
        <template v-if="column.key === 'savings'">
          <div style="color: #52c41a; font-weight: bold">¥{{ record.savings?.toFixed(2) }}</div>
          <div style="color: #999; font-size: 12px">节省 {{ record.savings_percent?.toFixed(0) }}%</div>
        </template>
        <template v-if="column.key === 'status'">
          <a-tag :color="getStatusColor(record.status)">{{ getStatusText(record.status) }}</a-tag>
        </template>
        <template v-if="column.key === 'action'">
          <a-space v-if="record.status === 'pending'">
            <a-button type="link" size="small" @click="apply(record)">应用</a-button>
            <a-button type="link" size="small" @click="ignore(record)">忽略</a-button>
          </a-space>
          <span v-else style="color: #999">-</span>
        </template>
      </template>
    </a-table>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'

const selectedCluster = ref<number>()
const status = ref('pending')
const clusters = ref<any[]>([])
const suggestions = ref<any>({ items: [] })
const loading = ref(false)

const columns = [
  { title: '集群', dataIndex: 'cluster_name', width: 120 },
  { title: '资源', key: 'resource', width: 200 },
  { title: '类型', dataIndex: 'title', width: 100 },
  { title: '级别', key: 'severity', width: 80 },
  { title: '描述', dataIndex: 'description' },
  { title: '可节省', key: 'savings', width: 100 },
  { title: '状态', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const fetchData = async () => {
  loading.value = true
  const [s, c]: any[] = await Promise.all([costApi.getSuggestions(selectedCluster.value, status.value), k8sClusterApi.list()])
  if (s?.code === 0) suggestions.value = s.data || { items: [] }
  if (c?.code === 0) clusters.value = c.data?.items || []
  loading.value = false
}

const apply = async (record: any) => {
  const res: any = await costApi.applySuggestion(record.id)
  if (res?.code === 0) { message.success('应用成功'); fetchData() }
}

const ignore = async (record: any) => {
  const res: any = await costApi.ignoreSuggestion(record.id)
  if (res?.code === 0) { message.success('已忽略'); fetchData() }
}

const getSeverityColor = (s: string) => ({ high: 'red', medium: 'orange', low: 'blue' }[s] || 'default')
const getSeverityText = (s: string) => ({ high: '高', medium: '中', low: '低' }[s] || s)
const getStatusColor = (s: string) => ({ pending: 'orange', applied: 'green', ignored: 'default' }[s] || 'default')
const getStatusText = (s: string) => ({ pending: '待处理', applied: '已应用', ignored: '已忽略' }[s] || s)

onMounted(fetchData)
</script>

<template>
  <div class="audit-log">
    <a-card title="安全审计日志">
      <template #extra>
        <a-space>
          <a-button @click="handleExport('csv')"><DownloadOutlined /> 导出CSV</a-button>
          <a-button @click="handleExport('json')"><DownloadOutlined /> 导出JSON</a-button>
        </a-space>
      </template>

      <!-- 筛选条件 -->
      <a-form layout="inline" style="margin-bottom: 16px">
        <a-form-item label="操作类型">
          <a-select v-model:value="filters.action" placeholder="全部" allowClear style="width: 150px" @change="loadLogs">
            <a-select-option value="create">创建</a-select-option>
            <a-select-option value="update">更新</a-select-option>
            <a-select-option value="delete">删除</a-select-option>
            <a-select-option value="deploy">部署</a-select-option>
            <a-select-option value="scale">扩缩容</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="资源类型">
          <a-select v-model:value="filters.resource_type" placeholder="全部" allowClear style="width: 150px" @change="loadLogs">
            <a-select-option value="deployment">Deployment</a-select-option>
            <a-select-option value="service">Service</a-select-option>
            <a-select-option value="configmap">ConfigMap</a-select-option>
            <a-select-option value="secret">Secret</a-select-option>
            <a-select-option value="pod">Pod</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="集群">
          <a-select v-model:value="filters.cluster_id" placeholder="全部" allowClear style="width: 150px" @change="loadLogs">
            <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="时间范围">
          <a-range-picker v-model:value="dateRange" @change="handleDateChange" />
        </a-form-item>
      </a-form>

      <a-table :dataSource="logs" :loading="loading" :pagination="pagination" @change="handleTableChange" rowKey="id">
        <a-table-column title="时间" dataIndex="created_at" :width="180">
          <template #default="{ record }">{{ formatTime(record.created_at) }}</template>
        </a-table-column>
        <a-table-column title="用户" dataIndex="username" :width="120" />
        <a-table-column title="操作" dataIndex="action" :width="100">
          <template #default="{ record }">
            <a-tag :color="getActionColor(record.action)">{{ getActionLabel(record.action) }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="资源" :width="250">
          <template #default="{ record }">
            <div>{{ record.resource_type }}/{{ record.resource_name }}</div>
            <div style="color: #999; font-size: 12px">{{ record.namespace }} @ {{ record.cluster_name }}</div>
          </template>
        </a-table-column>
        <a-table-column title="结果" dataIndex="result" :width="80">
          <template #default="{ record }">
            <a-tag :color="record.result === 'success' ? 'green' : 'red'">{{ record.result === 'success' ? '成功' : '失败' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="详情" dataIndex="detail" :ellipsis="true" />
        <a-table-column title="IP" dataIndex="client_ip" :width="130" />
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { DownloadOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { getAuditLogs, exportAuditLogs } from '@/services/security'
import { k8sClusterApi } from '@/services/k8s'
import dayjs, { Dayjs } from 'dayjs'

const loading = ref(false)
const logs = ref<any[]>([])
const clusters = ref<any[]>([])
const pagination = ref({ current: 1, pageSize: 20, total: 0 })
const dateRange = ref<[Dayjs, Dayjs] | null>(null)
const filters = ref({
  action: undefined as string | undefined,
  resource_type: undefined as string | undefined,
  cluster_id: undefined as number | undefined,
  start_time: '',
  end_time: ''
})

const loadClusters = async () => {
  try {
    const res: any = await k8sClusterApi.list()
    clusters.value = res?.data?.items || []
  } catch (error) {
    console.error('加载集群列表失败', error)
  }
}

const loadLogs = async () => {
  loading.value = true
  try {
    const res = await getAuditLogs({
      ...filters.value,
      page: pagination.value.current,
      page_size: pagination.value.pageSize
    })
    logs.value = res?.data?.items || []
    pagination.value.total = res?.data?.total || 0
  } catch (error) {
    console.error('加载审计日志失败', error)
  } finally {
    loading.value = false
  }
}

const handleTableChange = (pag: any) => {
  pagination.value.current = pag.current
  pagination.value.pageSize = pag.pageSize
  loadLogs()
}

const handleDateChange = (dates: [Dayjs, Dayjs] | null) => {
  if (dates) {
    filters.value.start_time = dates[0].format('YYYY-MM-DD')
    filters.value.end_time = dates[1].format('YYYY-MM-DD')
  } else {
    filters.value.start_time = ''
    filters.value.end_time = ''
  }
  loadLogs()
}

const handleExport = async (format: string) => {
  try {
    const res = await exportAuditLogs({ ...filters.value, format })
    const blob = new Blob([res as any], { type: format === 'csv' ? 'text/csv' : 'application/json' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `audit_logs_${dayjs().format('YYYYMMDD')}.${format}`
    link.click()
    window.URL.revokeObjectURL(url)
    message.success('导出成功')
  } catch (error) {
    message.error('导出失败')
  }
}

const getActionColor = (action: string) => ({ create: 'green', update: 'blue', delete: 'red', deploy: 'purple', scale: 'orange' }[action] || 'default')
const getActionLabel = (action: string) => ({ create: '创建', update: '更新', delete: '删除', deploy: '部署', scale: '扩缩容' }[action] || action)
const formatTime = (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'

onMounted(() => { loadClusters(); loadLogs() })
</script>

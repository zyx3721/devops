<template>
  <div class="log-export-page">
    <a-card>
      <template #title>
        <div class="card-header">
          <span>日志导出</span>
        </div>
      </template>

      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <a-row :gutter="20">
          <a-col :span="8">
            <a-form-item label="集群">
              <a-select v-model:value="form.cluster_id" placeholder="选择集群" @change="onClusterChange" style="width: 100%">
                <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">{{ cluster.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="命名空间">
              <a-select v-model:value="form.namespace" placeholder="选择命名空间" @change="onNamespaceChange" style="width: 100%">
                <a-select-option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="Pod">
              <a-select v-model:value="form.pod_names" mode="multiple" placeholder="选择 Pod（可多选）" :max-tag-count="1" style="width: 100%">
                <a-select-option v-for="pod in pods" :key="pod.name" :value="pod.name">{{ pod.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="20">
          <a-col :span="8">
            <a-form-item label="导出格式">
              <a-radio-group v-model:value="form.format">
                <a-radio-button value="txt">TXT</a-radio-button>
                <a-radio-button value="json">JSON</a-radio-button>
                <a-radio-button value="csv">CSV</a-radio-button>
              </a-radio-group>
            </a-form-item>
          </a-col>
          <a-col :span="16">
            <a-form-item label="时间范围">
              <a-range-picker
                v-model:value="timeRange"
                show-time
                format="YYYY-MM-DD HH:mm:ss"
                style="width: 100%"
              />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="20">
          <a-col :span="8">
            <a-form-item label="关键词过滤">
              <a-input v-model:value="form.keyword" placeholder="可选，过滤包含关键词的日志" allow-clear />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="日志级别">
              <a-select v-model:value="form.level" placeholder="可选，过滤指定级别" allow-clear style="width: 100%">
                <a-select-option value="ERROR">ERROR</a-select-option>
                <a-select-option value="WARN">WARN</a-select-option>
                <a-select-option value="INFO">INFO</a-select-option>
                <a-select-option value="DEBUG">DEBUG</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <!-- 导出进度 -->
        <div v-if="exportTask" class="export-progress">
          <a-progress :percent="exportTask.progress" :status="progressStatus" />
          <div class="progress-info">
            <span>{{ statusText }}</span>
            <a-button v-if="exportTask.status === 'processing'" type="primary" danger size="small" @click="cancelExport">
              取消
            </a-button>
          </div>
        </div>

        <a-form-item :wrapper-col="{ span: 24, offset: 0 }">
          <a-button 
            type="primary" 
            :loading="loading" 
            :disabled="!canExport || exportTask?.status === 'processing'"
            @click="startExport"
          >
            {{ exportTask?.status === 'completed' ? '重新导出' : '开始导出' }}
          </a-button>
          <a-button 
            v-if="exportTask?.status === 'completed'" 
            type="default" 
            @click="downloadFile"
            style="margin-left: 10px; border-color: #52c41a; color: #52c41a"
          >
            <template #icon><DownloadOutlined /></template>
            下载文件
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 导出历史 -->
    <a-card style="margin-top: 20px">
      <template #title>
        <span>导出历史</span>
      </template>
      <a-empty description="暂无导出记录" />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { DownloadOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import dayjs, { Dayjs } from 'dayjs'
import { k8sApi } from '@/services/k8s'
import { logApi, type LogExportResponse } from '@/services/logs'

interface Cluster {
  id: number
  name: string
}

interface Pod {
  name: string
  status: string
}

const clusters = ref<Cluster[]>([])
const namespaces = ref<string[]>([])
const pods = ref<Pod[]>([])

const form = reactive({
  cluster_id: null as number | null,
  namespace: '',
  pod_names: [] as string[],
  format: 'txt' as 'txt' | 'json' | 'csv',
  keyword: '',
  level: undefined as string | undefined
})

const timeRange = ref<[Dayjs, Dayjs] | null>(null)
const loading = ref(false)
const exportTask = ref<LogExportResponse | null>(null)
let pollTimer: number | null = null

const canExport = computed(() => {
  return form.cluster_id && form.namespace
})

const progressStatus = computed(() => {
  if (!exportTask.value) return 'normal'
  switch (exportTask.value.status) {
    case 'completed': return 'success'
    case 'failed': return 'exception'
    case 'processing': return 'active'
    default: return 'normal'
  }
})

const statusText = computed(() => {
  if (!exportTask.value) return ''
  switch (exportTask.value.status) {
    case 'pending': return '准备中...'
    case 'processing': return '导出中...'
    case 'completed': return '导出完成'
    case 'failed': return `导出失败: ${exportTask.value.error}`
    default: return ''
  }
})

const loadClusters = async () => {
  try {
    const res = await k8sApi.getClusters()
    clusters.value = res.data || []
  } catch (error) {
    message.error('加载集群列表失败')
  }
}

const onClusterChange = async () => {
  form.namespace = ''
  form.pod_names = []
  namespaces.value = []
  pods.value = []
  
  if (!form.cluster_id) return
  
  try {
    const res = await k8sApi.getNamespaces(form.cluster_id)
    namespaces.value = res.data || []
  } catch (error) {
    message.error('加载命名空间失败')
  }
}

const onNamespaceChange = async () => {
  form.pod_names = []
  pods.value = []
  
  if (!form.cluster_id || !form.namespace) return
  
  try {
    const res = await k8sApi.getPods(form.cluster_id, form.namespace)
    pods.value = (res.data || []).map((pod: any) => ({
      name: pod.name,
      status: pod.status
    }))
  } catch (error) {
    message.error('加载 Pod 列表失败')
  }
}

const startExport = async () => {
  if (!form.cluster_id || !form.namespace) {
    message.warning('请选择集群和命名空间')
    return
  }

  loading.value = true
  exportTask.value = null

  try {
    const res = await logApi.exportLogs({
      cluster_id: form.cluster_id,
      namespace: form.namespace,
      pod_names: form.pod_names.length > 0 ? form.pod_names : undefined,
      start_time: timeRange.value?.[0]?.toISOString(),
      end_time: timeRange.value?.[1]?.toISOString(),
      format: form.format,
      keyword: form.keyword || undefined,
      level: form.level || undefined
    })

    exportTask.value = res.data
    startPolling()
  } catch (error: any) {
    message.error(error.message || '导出失败')
  } finally {
    loading.value = false
  }
}

const startPolling = () => {
  if (pollTimer) clearInterval(pollTimer)
  
  pollTimer = window.setInterval(async () => {
    if (!exportTask.value) return
    
    try {
      const res = await logApi.getExportStatus(exportTask.value.task_id)
      exportTask.value = res.data
      
      if (res.data.status === 'completed' || res.data.status === 'failed') {
        stopPolling()
      }
    } catch (error) {
      stopPolling()
    }
  }, 1000)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const cancelExport = async () => {
  if (!exportTask.value) return
  
  try {
    await logApi.cancelExport(exportTask.value.task_id)
    stopPolling()
    exportTask.value = null
    message.info('已取消导出')
  } catch (error: any) {
    message.error(error.message || '取消失败')
  }
}

const downloadFile = () => {
  if (!exportTask.value?.task_id) return
  
  const url = logApi.downloadExport(exportTask.value.task_id)
  window.open(url, '_blank')
}

onMounted(() => {
  loadClusters()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.log-export-page {
  padding: 20px;
}

.card-header {
  font-weight: 500;
}

.export-progress {
  margin: 20px 0;
  padding: 15px;
  background: #fafafa;
  border-radius: 4px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.45);
}
</style>

<template>
  <div class="log-export">
    <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
      <a-form-item label="导出格式">
        <a-radio-group v-model:value="form.format">
          <a-radio-button value="txt">TXT</a-radio-button>
          <a-radio-button value="json">JSON</a-radio-button>
          <a-radio-button value="csv">CSV</a-radio-button>
        </a-radio-group>
      </a-form-item>

      <a-form-item label="时间范围">
        <a-range-picker
          v-model:value="timeRange"
          show-time
          :placeholder="['开始时间', '结束时间']"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DDTHH:mm:ssZ"
          style="width: 100%"
        />
      </a-form-item>

      <a-form-item label="关键词过滤">
        <a-input v-model:value="form.keyword" placeholder="可选，过滤包含关键词的日志" allow-clear />
      </a-form-item>

      <a-form-item label="日志级别">
        <a-select v-model:value="form.level" placeholder="可选，过滤指定级别" allow-clear style="width: 100%">
          <a-select-option value="">全部</a-select-option>
          <a-select-option value="ERROR">ERROR</a-select-option>
          <a-select-option value="WARN">WARN</a-select-option>
          <a-select-option value="INFO">INFO</a-select-option>
          <a-select-option value="DEBUG">DEBUG</a-select-option>
        </a-select>
      </a-form-item>
    </a-form>

    <!-- 导出进度 -->
    <div v-if="exportTask" class="export-progress">
      <a-progress :percent="exportTask.progress" :status="progressStatus" />
      <div class="progress-info">
        <span>{{ statusText }}</span>
        <a-button v-if="exportTask.status === 'processing'" danger size="small" @click="cancelExport">
          取消
        </a-button>
      </div>
    </div>

    <div class="actions">
      <a-button @click="emit('close')">取消</a-button>
      <a-button 
        type="primary" 
        :loading="loading" 
        :disabled="exportTask?.status === 'processing'"
        @click="startExport"
      >
        {{ exportTask?.status === 'completed' ? '重新导出' : '开始导出' }}
      </a-button>
      <a-button 
        v-if="exportTask?.status === 'completed'" 
        type="primary"
        @click="downloadFile"
      >
        <DownloadOutlined />
        下载文件
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { DownloadOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { logApi, type LogExportResponse } from '@/services/logs'

const props = defineProps<{
  clusterId: number
  namespace: string
  podName?: string
  podNames?: string[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const form = ref({
  format: 'txt' as 'txt' | 'json' | 'csv',
  keyword: '',
  level: ''
})

const timeRange = ref<[string, string] | null>(null)
const loading = ref(false)
const exportTask = ref<LogExportResponse | null>(null)
let pollTimer: number | null = null

const progressStatus = computed(() => {
  if (!exportTask.value) return ''
  switch (exportTask.value.status) {
    case 'completed': return 'success'
    case 'failed': return 'exception'
    default: return ''
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

const startExport = async () => {
  loading.value = true
  exportTask.value = null

  try {
    const res = await logApi.exportLogs({
      cluster_id: props.clusterId,
      namespace: props.namespace,
      pod_names: props.podName ? [props.podName] : undefined,
      start_time: timeRange.value?.[0],
      end_time: timeRange.value?.[1],
      format: form.value.format,
      keyword: form.value.keyword || undefined,
      level: form.value.level || undefined
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
  // 使用 a 标签下载，避免打开新窗口
  const link = document.createElement('a')
  link.href = url
  link.download = `logs_${new Date().toISOString().slice(0, 10)}.${form.value.format}`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.log-export {
  padding: 16px 0;
}

.export-progress {
  margin: 24px 0;
  padding: 16px;
  background: #fafafa;
  border-radius: 4px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.45);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid #f0f0f0;
}
</style>

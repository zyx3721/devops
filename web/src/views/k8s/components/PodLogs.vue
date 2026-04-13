<template>
  <div class="pod-logs">
    <div class="logs-toolbar">
      <a-space>
        <a-select v-model:value="selectedContainer" placeholder="选择容器" style="width: 200px" @change="loadLogs">
          <a-select-option v-for="c in containers" :key="c.name" :value="c.name">
            {{ c.name }}
          </a-select-option>
        </a-select>
        <a-input-number v-model:value="tailLines" :min="10" :max="10000" addon-before="行数" style="width: 150px" />
        <a-checkbox v-model:checked="showTimestamps">显示时间戳</a-checkbox>
        <a-checkbox v-model:checked="autoScroll">自动滚动</a-checkbox>
        <a-button @click="loadLogs" :loading="loading">刷新</a-button>
        <a-button type="primary" @click="toggleStream" :danger="streaming">
          {{ streaming ? '停止' : '实时' }}
        </a-button>
        <a-button @click="downloadLogs">下载</a-button>
      </a-space>
    </div>

    <div class="logs-content" ref="logsContainer">
      <pre>{{ logs }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import { k8sPodApi, type K8sContainer } from '@/services/k8s'

const props = defineProps<{
  clusterId: number
  namespace: string
  podName: string
}>()

const containers = ref<K8sContainer[]>([])
const selectedContainer = ref('')
const tailLines = ref(100)
const showTimestamps = ref(false)
const autoScroll = ref(true)
const loading = ref(false)
const streaming = ref(false)
const logs = ref('')
const logsContainer = ref<HTMLElement | null>(null)
let ws: WebSocket | null = null

const loadContainers = async () => {
  try {
    const res = await k8sPodApi.getContainers(props.clusterId, props.namespace, props.podName)
    containers.value = res.data || []
    if (containers.value.length > 0 && !selectedContainer.value) {
      selectedContainer.value = containers.value[0].name
    }
  } catch {}
}

const loadLogs = async () => {
  if (!selectedContainer.value) return
  loading.value = true
  try {
    const res = await k8sPodApi.getLogs(
      props.clusterId, props.namespace, props.podName,
      selectedContainer.value, tailLines.value, showTimestamps.value
    )
    logs.value = res.data?.logs || ''
    scrollToBottom()
  } finally {
    loading.value = false
  }
}

const toggleStream = () => {
  if (streaming.value) {
    stopStream()
  } else {
    startStream()
  }
}

const startStream = () => {
  if (!selectedContainer.value) {
    message.warning('请先选择容器')
    return
  }
  
  const url = k8sPodApi.getLogsStreamUrl(
    props.clusterId, props.namespace, props.podName,
    selectedContainer.value, tailLines.value, showTimestamps.value
  )
  
  ws = new WebSocket(url)
  
  ws.onopen = () => {
    streaming.value = true
    message.success('已连接日志流')
  }
  
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      if (data.type === 'log' && data.data) {
        logs.value += data.data
        if (autoScroll.value) {
          scrollToBottom()
        }
      }
    } catch {
      logs.value += event.data
    }
  }
  
  ws.onerror = () => {
    message.error('日志流连接错误')
    streaming.value = false
  }
  
  ws.onclose = () => {
    streaming.value = false
  }
}

const stopStream = () => {
  if (ws) {
    ws.close()
    ws = null
  }
  streaming.value = false
}

const scrollToBottom = () => {
  nextTick(() => {
    if (logsContainer.value) {
      logsContainer.value.scrollTop = logsContainer.value.scrollHeight
    }
  })
}

const downloadLogs = () => {
  const url = k8sPodApi.downloadLogs(props.clusterId, props.namespace, props.podName, selectedContainer.value)
  window.open(url, '_blank')
}

watch(() => props.podName, () => {
  stopStream()
  logs.value = ''
  loadContainers()
})

onMounted(() => {
  loadContainers()
})

onUnmounted(() => {
  stopStream()
})
</script>

<style scoped>
.pod-logs {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.logs-toolbar {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

.logs-content {
  flex: 1;
  overflow: auto;
  background: #1e1e1e;
  padding: 12px;
}

.logs-content pre {
  margin: 0;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>

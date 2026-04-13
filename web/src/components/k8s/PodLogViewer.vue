<template>
  <a-drawer
    :open="visible"
    :title="null"
    placement="right"
    :width="950"
    :closable="false"
    :bodyStyle="{ padding: 0, display: 'flex', flexDirection: 'column', height: '100%' }"
    @close="handleClose"
  >
    <!-- 头部 -->
    <div class="log-header">
      <div class="header-left">
        <a-button type="text" @click="handleClose"><CloseOutlined /></a-button>
        <a-select
          v-model:value="selectedPodName"
          style="width: 280px"
          size="small"
          show-search
          :filter-option="filterPodOption"
          @change="onPodChange"
        >
          <a-select-option v-for="p in pods" :key="p.name" :value="p.name">
            <div class="pod-option">
              <span>{{ p.name }}</span>
              <a-tag :color="getPodStatusColor(p.status)" size="small">{{ p.status }}</a-tag>
            </div>
          </a-select-option>
        </a-select>
      </div>
      <div class="header-right">
        <a-space>
          <a-select v-model:value="selectedContainer" style="width: 140px" size="small" v-if="containers.length > 1">
            <a-select-option v-for="c in containers" :key="c" :value="c">{{ c }}</a-select-option>
          </a-select>
          <a-select v-model:value="tailLines" style="width: 90px" size="small">
            <a-select-option :value="100">100行</a-select-option>
            <a-select-option :value="500">500行</a-select-option>
            <a-select-option :value="1000">1000行</a-select-option>
            <a-select-option :value="5000">5000行</a-select-option>
          </a-select>
          <a-input-search
            v-model:value="searchKeyword"
            placeholder="搜索..."
            style="width: 160px"
            size="small"
            allow-clear
            @search="highlightSearch"
          />
          <a-button size="small" :type="streaming ? 'primary' : 'default'" :danger="streaming" @click="toggleStream">
            <template v-if="streaming"><PauseOutlined /> 停止</template>
            <template v-else><CaretRightOutlined /> 实时</template>
          </a-button>
          <a-button size="small" @click="fetchLogs" :loading="loading"><ReloadOutlined /></a-button>
          <a-button size="small" @click="downloadLogs"><DownloadOutlined /></a-button>
        </a-space>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="log-toolbar">
      <a-space>
        <a-checkbox v-model:checked="showTimestamp">时间戳</a-checkbox>
        <a-checkbox v-model:checked="autoScroll">自动滚动</a-checkbox>
        <a-checkbox v-model:checked="wrapLine">自动换行</a-checkbox>
      </a-space>
      <div class="log-stats">
        <span v-if="streaming" class="streaming-badge"><span class="dot"></span> 实时更新中</span>
        <span>共 {{ logLines.length }} 行</span>
        <span v-if="searchKeyword">匹配 {{ matchCount }} 处</span>
      </div>
    </div>

    <!-- 日志内容 -->
    <div ref="logContainer" class="log-container" :class="{ 'wrap-line': wrapLine }">
      <div v-if="loading && logLines.length === 0" class="log-loading">
        <a-spin tip="加载中..." />
      </div>
      <div v-else-if="logLines.length === 0" class="log-empty">
        <a-empty description="暂无日志" />
      </div>
      <template v-else>
        <div
          v-for="(line, index) in logLines"
          :key="index"
          class="log-line"
          :class="getLineClass(line)"
        >
          <span class="line-number">{{ index + 1 }}</span>
          <span v-if="showTimestamp && line.timestamp" class="log-time">{{ formatTime(line.timestamp) }}</span>
          <span class="log-content" v-html="highlightContent(line.content)"></span>
        </div>
      </template>
    </div>
  </a-drawer>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onUnmounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import {
  CloseOutlined,
  ReloadOutlined,
  DownloadOutlined,
  CaretRightOutlined,
  PauseOutlined
} from '@ant-design/icons-vue'
import { k8sResourceApi } from '@/services/k8s'

interface LogLine {
  timestamp?: string
  content: string
  level?: string
}

interface PodInfo {
  name: string
  namespace: string
  status: string
  containers: { name: string }[]
}

const props = defineProps<{
  visible: boolean
  clusterId: number
  pod: PodInfo | null
  pods: PodInfo[]
}>()

const emit = defineEmits(['update:visible', 'update:pod'])

const loading = ref(false)
const streaming = ref(false)
const showTimestamp = ref(true)
const autoScroll = ref(true)
const wrapLine = ref(false)
const searchKeyword = ref('')
const tailLines = ref(500)
const selectedContainer = ref('')
const selectedPodName = ref('')
const containers = ref<string[]>([])
const logLines = ref<LogLine[]>([])
const logContainer = ref<HTMLElement | null>(null)

let ws: WebSocket | null = null

const currentPod = computed(() => {
  return props.pods.find(p => p.name === selectedPodName.value) || props.pod
})

const matchCount = computed(() => {
  if (!searchKeyword.value) return 0
  const keyword = searchKeyword.value.toLowerCase()
  return logLines.value.filter(l => l.content.toLowerCase().includes(keyword)).length
})

watch(() => props.visible, async (val) => {
  if (val && props.pod) {
    selectedPodName.value = props.pod.name
    updateContainers(props.pod)
    await fetchLogs()
  } else {
    stopStream()
    logLines.value = []
  }
})

watch(() => props.pod, (newPod) => {
  if (props.visible && newPod) {
    selectedPodName.value = newPod.name
    updateContainers(newPod)
  }
})

const updateContainers = (pod: PodInfo | null) => {
  if (!pod) return
  containers.value = pod.containers?.map((c: any) => c.name) || []
  selectedContainer.value = containers.value[0] || ''
}

const onPodChange = (podName: string) => {
  const pod = props.pods.find(p => p.name === podName)
  if (pod) {
    stopStream()
    updateContainers(pod)
    emit('update:pod', pod)
    fetchLogs()
  }
}

const filterPodOption = (input: string, option: any) => {
  return option.value.toLowerCase().includes(input.toLowerCase())
}

watch(() => selectedContainer.value, (newVal, oldVal) => {
  if (props.visible && oldVal && newVal !== oldVal) {
    stopStream()
    fetchLogs()
  }
})

watch(() => tailLines.value, () => {
  if (props.visible && !streaming.value) {
    fetchLogs()
  }
})

const fetchLogs = async () => {
  const pod = currentPod.value
  if (!pod) return
  loading.value = true
  try {
    const res = await k8sResourceApi.getPodLogs(
      props.clusterId,
      pod.namespace,
      pod.name,
      selectedContainer.value,
      tailLines.value
    )
    if (res?.code === 0) {
      const rawLogs = res.data || ''
      logLines.value = parseLogLines(rawLogs)
      scrollToBottom()
    }
  } catch (error) {
    message.error('获取日志失败')
  } finally {
    loading.value = false
  }
}

const parseLogLines = (raw: string): LogLine[] => {
  if (!raw) return []
  return raw.split('\n').filter(l => l.trim()).map(line => {
    const parsed: LogLine = { content: line }
    const match = line.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[.\d]*Z?)\s+(.*)/)
    if (match) {
      parsed.timestamp = match[1]
      parsed.content = match[2]
    }
    parsed.level = detectLevel(parsed.content)
    return parsed
  })
}

const detectLevel = (content: string): string => {
  const upper = content.toUpperCase()
  if (/\b(ERROR|ERR|FATAL|PANIC|EXCEPTION)\b/.test(upper)) return 'error'
  if (/\b(WARN|WARNING)\b/.test(upper)) return 'warn'
  if (/\b(DEBUG|TRACE)\b/.test(upper)) return 'debug'
  return 'info'
}

const toggleStream = () => {
  if (streaming.value) {
    stopStream()
  } else {
    startStream()
  }
}

const startStream = () => {
  const pod = currentPod.value
  if (!pod) return
  
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const token = localStorage.getItem('token') || ''
  
  const params = new URLSearchParams({
    container: selectedContainer.value,
    tail_lines: String(tailLines.value),
    timestamps: 'true',
    token
  })
  
  const url = `${protocol}//${host}/app/api/v1/k8s/clusters/${props.clusterId}/namespaces/${pod.namespace}/pods/${pod.name}/logs/stream?${params}`
  
  ws = new WebSocket(url)
  
  ws.onopen = () => {
    streaming.value = true
    message.success('已连接实时日志')
  }
  
  ws.onmessage = (event) => {
    const lines = parseLogLines(event.data)
    logLines.value.push(...lines)
    if (logLines.value.length > 10000) {
      logLines.value = logLines.value.slice(-5000)
    }
    if (autoScroll.value) {
      scrollToBottom()
    }
  }
  
  ws.onerror = () => {
    message.error('WebSocket连接失败')
    stopStream()
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

const downloadLogs = () => {
  const pod = currentPod.value
  const content = logLines.value.map(l => {
    if (l.timestamp) {
      return `${l.timestamp} ${l.content}`
    }
    return l.content
  }).join('\n')
  
  const blob = new Blob([content], { type: 'text/plain' })
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${pod?.name || 'pod'}.log`
  a.click()
  window.URL.revokeObjectURL(url)
  message.success('下载成功')
}

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

const formatTime = (timestamp: string) => {
  if (!timestamp) return ''
  return timestamp.replace('T', ' ').replace('Z', '').substring(0, 19)
}

const getPodStatusColor = (status?: string) => {
  if (!status) return 'default'
  const map: Record<string, string> = {
    Running: 'green',
    Succeeded: 'blue',
    Pending: 'orange',
    Failed: 'red',
    Unknown: 'default'
  }
  return map[status] || 'default'
}

const getLineClass = (line: LogLine) => {
  return {
    'log-error': line.level === 'error',
    'log-warn': line.level === 'warn',
    'log-debug': line.level === 'debug',
    'log-highlight': searchKeyword.value && line.content.toLowerCase().includes(searchKeyword.value.toLowerCase())
  }
}

const highlightContent = (content: string) => {
  if (!searchKeyword.value || !content) return content
  const escaped = searchKeyword.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const regex = new RegExp(`(${escaped})`, 'gi')
  return content.replace(regex, '<mark>$1</mark>')
}

const highlightSearch = () => {
  if (searchKeyword.value && logContainer.value) {
    const firstMatch = logContainer.value.querySelector('.log-highlight')
    if (firstMatch) {
      firstMatch.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }
}

const handleClose = () => {
  stopStream()
  emit('update:visible', false)
}

onUnmounted(() => {
  stopStream()
})
</script>

<style scoped>
.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.pod-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.log-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  border-bottom: 1px solid #f0f0f0;
  background: #fff;
}

.log-stats {
  display: flex;
  gap: 16px;
  color: #666;
  font-size: 12px;
}

.streaming-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #52c41a;
}

.streaming-badge .dot {
  width: 6px;
  height: 6px;
  background: #52c41a;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.log-container {
  flex: 1;
  overflow-y: auto;
  background: #1e1e1e;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
}

.log-container.wrap-line .log-line {
  white-space: pre-wrap;
  word-break: break-all;
}

.log-loading,
.log-empty {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: #999;
}

.log-line {
  display: flex;
  white-space: pre;
  color: #d4d4d4;
  padding: 1px 12px;
  border-left: 3px solid transparent;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.05);
}

.log-line.log-error {
  background: rgba(255, 77, 79, 0.15);
  border-left-color: #ff4d4f;
}

.log-line.log-warn {
  background: rgba(250, 173, 20, 0.1);
  border-left-color: #faad14;
}

.log-line.log-debug {
  color: #888;
}

.log-line.log-highlight {
  background: rgba(255, 255, 0, 0.1);
}

.line-number {
  color: #666;
  min-width: 40px;
  text-align: right;
  margin-right: 12px;
  user-select: none;
}

.log-time {
  color: #6a9955;
  margin-right: 12px;
  flex-shrink: 0;
}

.log-content {
  flex: 1;
}

.log-content :deep(mark) {
  background: #ffff00;
  color: #000;
  padding: 0 2px;
  border-radius: 2px;
}
</style>

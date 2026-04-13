<template>
  <div class="log-viewer">
    <!-- 工具栏 -->
    <div class="viewer-toolbar">
      <a-select v-if="!multiPod" v-model:value="currentContainer" placeholder="选择容器" size="small" style="width: 150px" @change="onContainerChange">
        <a-select-option v-for="c in containers" :key="c" :value="c">{{ c }}</a-select-option>
      </a-select>
      
      <LogFilter v-model:keyword="filterKeyword" v-model:level="filterLevel" v-model:use-regex="useRegex" />
      
      <div class="toolbar-actions">
        <a-button-group>
          <a-button :type="isPaused ? 'default' : 'default'" size="small" @click="togglePause" :danger="isPaused">
            <template #icon>
              <PauseCircleOutlined v-if="!isPaused" />
              <PlayCircleOutlined v-else />
            </template>
            {{ isPaused ? '继续' : '暂停' }}
          </a-button>
          <a-button size="small" @click="scrollToBottom">
            <template #icon>
              <VerticalAlignBottomOutlined />
            </template>
            底部
          </a-button>
          <a-button size="small" @click="clearLogs">
            <template #icon>
              <DeleteOutlined />
            </template>
            清空
          </a-button>
        </a-button-group>
        
        <a-tag v-if="!isConnected" color="error">
          <template #icon>
            <WarningOutlined />
          </template>
          已断开
        </a-tag>
        <a-tag v-else color="success">
          <template #icon>
            <ApiOutlined />
          </template>
          已连接
        </a-tag>
      </div>
    </div>

    <!-- 多 Pod 图例 -->
    <div v-if="multiPod && podNames && podNames.length > 1" class="pod-legend">
      <span class="legend-title">Pod 图例:</span>
      <span v-for="pod in podNames" :key="pod" class="legend-item">
        <span class="legend-color" :style="{ backgroundColor: podColors?.[pod] || '#409EFF' }"></span>
        {{ pod.split('-').slice(-2).join('-') }}
      </span>
    </div>

    <!-- 断开重连提示 -->
    <a-alert v-if="!isConnected && connectionError" type="error" :closable="false" class="connection-alert">
      <template #message>
        连接已断开: {{ connectionError }}
        <a-button type="primary" size="small" @click="reconnect" style="margin-left: 10px">重新连接</a-button>
      </template>
    </a-alert>

    <!-- 日志内容 -->
    <div ref="logContainer" class="log-content" @scroll="onScroll">
      <div class="log-list">
        <LogLine 
          v-for="log in displayLogs"
          :key="log.id"
          :log="log" 
          :keyword="filterKeyword"
          :highlight-rules="highlightRules"
          :pod-color="multiPod ? podColors?.[log.pod_name] : undefined"
          :show-pod-name="multiPod"
          @context="showContext"
        />
      </div>
    </div>

    <!-- 状态栏 -->
    <div class="status-bar">
      <span>共 {{ logs.length }} 条日志</span>
      <span v-if="filterKeyword || filterLevel">| 过滤后 {{ filteredLogs.length }} 条</span>
      <span v-if="isPaused">| 已暂停</span>
      <span v-if="multiPod">| {{ podNames?.length || 0 }} 个 Pod</span>
    </div>

    <!-- 日志上下文对话框 -->
    <LogContext
      v-model="showContextDialog"
      :cluster-id="clusterId"
      :namespace="namespace"
      :pod-name="contextLog?.pod_name || podName || ''"
      :container="currentContainer"
      :log="contextLog"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import { PauseCircleOutlined, PlayCircleOutlined, VerticalAlignBottomOutlined, DeleteOutlined, WarningOutlined, ApiOutlined } from '@ant-design/icons-vue'
import { RecycleScroller } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import { logApi, type LogEntry, type HighlightRule } from '@/services/logs'
import LogLine from './components/LogLine.vue'
import LogFilter from './components/LogFilter.vue'
import LogContext from './components/LogContext.vue'

const props = defineProps<{
  clusterId: number
  namespace: string
  podName?: string
  podNames?: string[]
  container?: string
  highlightRules?: HighlightRule[]
  podColors?: Record<string, string>
  multiPod?: boolean
}>()

const emit = defineEmits<{
  (e: 'container-change', container: string): void
}>()

const logs = ref<LogEntry[]>([])
const containers = ref<string[]>([])
const currentContainer = ref('')
const filterKeyword = ref('')
const filterLevel = ref('')
const useRegex = ref(false)
const isPaused = ref(false)
const isConnected = ref(false)
const connectionError = ref('')
const autoScroll = ref(true)
const logContainer = ref<HTMLElement | null>(null)
const scroller = ref<any>(null)
const showContextDialog = ref(false)
const contextLog = ref<LogEntry | null>(null)

let ws: WebSocket | null = null
let reconnectTimer: number | null = null
let logIdCounter = 0

const filteredLogs = computed(() => {
  let result = logs.value
  
  if (filterKeyword.value) {
    if (useRegex.value) {
      try {
        const regex = new RegExp(filterKeyword.value, 'i')
        result = result.filter(log => regex.test(log.content))
      } catch {
        // 无效正则，忽略
      }
    } else {
      const keyword = filterKeyword.value.toLowerCase()
      result = result.filter(log => log.content.toLowerCase().includes(keyword))
    }
  }
  
  if (filterLevel.value) {
    result = result.filter(log => log.level === filterLevel.value)
  }
  
  return result
})

const displayLogs = computed(() => {
  const limit = 2000
  const logsToDisplay = filteredLogs.value
  if (logsToDisplay.length <= limit) return logsToDisplay
  return logsToDisplay.slice(-limit)
})

const loadContainers = async () => {
  if (props.multiPod || !props.podName) return
  try {
    const res = await logApi.getContainers(props.clusterId, props.namespace, props.podName)
    containers.value = res.data || []
    if (containers.value.length > 0) {
      // 确保 container 是字符串
      const containerProp = props.container
      const containerName = typeof containerProp === 'string' ? containerProp : (containerProp as any)?.name
      currentContainer.value = containerName || containers.value[0]
      // 加载完容器后立即连接
      connect()
    }
  } catch (error) {
    message.error('加载容器列表失败')
    console.error('加载容器列表失败', error)
  }
}

const connect = () => {
  if (ws) {
    ws.close()
    ws = null
  }

  const token = localStorage.getItem('token') || ''
  
  if (props.multiPod && props.podNames && props.podNames.length > 0) {
    // 多 Pod 模式
    const pods = props.podNames.map(name => ({ pod_name: name, container: '' }))
    const params = new URLSearchParams({
      cluster_id: String(props.clusterId),
      namespace: props.namespace,
      pods: JSON.stringify(pods),
      tail_lines: '200',
      token
    })

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/app/api/v1/logs/stream/multi?${params}`
    ws = new WebSocket(wsUrl)
  } else if (!props.multiPod && props.podName && currentContainer.value) {
    // 单 Pod 模式
    const params = new URLSearchParams({
      cluster_id: String(props.clusterId),
      namespace: props.namespace,
      pod_name: props.podName,
      container: currentContainer.value,
      tail_lines: '500',
      follow: 'true',
      token
    })

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/app/api/v1/logs/stream?${params}`
    ws = new WebSocket(wsUrl)
  } else {
    return
  }
  
  ws.onopen = () => {
    isConnected.value = true
    connectionError.value = ''
  }
  
  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'log') {
        addLog({
          id: `log-${++logIdCounter}`,
          timestamp: msg.timestamp,
          content: msg.content,
          pod_name: msg.pod_name,
          container: msg.container,
          level: msg.level
        })
      } else if (msg.type === 'error') {
        connectionError.value = msg.content
      } else if (msg.type === 'connected') {
        isConnected.value = true
        connectionError.value = ''
      }
    } catch (e) {
      // 忽略解析错误
    }
  }
  
  ws.onclose = () => {
    isConnected.value = false
    if (!connectionError.value) {
      connectionError.value = '连接已关闭'
    }
  }
  
  ws.onerror = () => {
    isConnected.value = false
    connectionError.value = '连接错误'
  }
}

const addLog = (log: LogEntry) => {
  if (isPaused.value) return
  
  logs.value.push(log)
  
  // 限制日志数量
  if (logs.value.length > 100000) {
    logs.value = logs.value.slice(-50000)
  }
  
  if (autoScroll.value) {
    nextTick(() => scrollToBottom())
  }
}

const togglePause = () => {
  isPaused.value = !isPaused.value
  if (ws) {
    ws.send(JSON.stringify({ action: isPaused.value ? 'pause' : 'resume' }))
  }
}

const scrollToBottom = () => {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
    autoScroll.value = true
  }
}

const clearLogs = () => {
  logs.value = []
  logIdCounter = 0
}

const reconnect = () => {
  connectionError.value = ''
  connect()
}

const onScroll = (e: Event) => {
  const target = e.target as HTMLElement
  const isAtBottom = target.scrollHeight - target.scrollTop - target.clientHeight < 50
  autoScroll.value = isAtBottom
}

const onContainerChange = () => {
  emit('container-change', currentContainer.value)
  // watch 会检测到变化并调用 connect
}

const showContext = (log: LogEntry) => {
  contextLog.value = log
  showContextDialog.value = true
}

// 单 Pod 模式监听
watch(() => props.podName, () => {
  if (!props.multiPod) {
    clearLogs()
    loadContainers()
  }
})

// 多 Pod 模式监听
watch(() => props.podNames, (newPods, oldPods) => {
  if (props.multiPod) {
    // 检查是否有变化
    const newSet = new Set(newPods || [])
    const oldSet = new Set(oldPods || [])
    const hasChange = newSet.size !== oldSet.size || 
      [...newSet].some(p => !oldSet.has(p))
    
    if (hasChange) {
      clearLogs()
      connect()
    }
  }
}, { deep: true })

watch(() => currentContainer.value, (newVal, oldVal) => {
  // 只有在用户手动切换容器时才重新连接（oldVal 存在说明不是初始化）
  if (!props.multiPod && newVal && oldVal) {
    clearLogs()
    connect()
  }
})

onMounted(() => {
  if (props.multiPod) {
    connect()
  } else {
    loadContainers()
  }
})

onUnmounted(() => {
  if (ws) {
    ws.close()
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
  }
})
</script>

<style scoped>
.log-viewer {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
}

.viewer-toolbar {
  padding: 12px 16px;
  background: #ffffff;
  border-bottom: 1px solid #d9d9d9;
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.pod-legend {
  padding: 8px 16px;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.65);
}

.legend-title {
  color: rgba(0, 0, 0, 0.45);
  font-weight: 500;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.legend-color {
  width: 12px;
  height: 12px;
  border-radius: 2px;
}

.connection-alert {
  margin: 0;
  border-radius: 0;
}

.log-content {
  flex: 1;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 24px;
}

.log-list {
  display: flex;
  flex-direction: column;
}

.status-bar {
  padding: 6px 16px;
  background: #ffffff;
  border-top: 1px solid #d9d9d9;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  display: flex;
  gap: 12px;
}

/* Responsive layout adjustments */
@media (max-width: 1200px) {
  .viewer-toolbar {
    flex-wrap: wrap;
  }
  
  .pod-legend {
    font-size: 11px;
    gap: 8px;
  }
}
</style>

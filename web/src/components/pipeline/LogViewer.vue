<template>
  <div class="log-viewer">
    <div class="log-toolbar">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索日志..."
        prefix-icon="Search"
        clearable
        style="width: 300px"
        @input="handleSearch"
      />
      <div class="toolbar-actions">
        <el-button :type="autoScroll ? 'primary' : ''" @click="toggleAutoScroll">
          <el-icon><Bottom /></el-icon>
          {{ autoScroll ? '自动滚动' : '手动滚动' }}
        </el-button>
        <el-button @click="downloadLogs">
          <el-icon><Download /></el-icon>
          下载
        </el-button>
        <el-button @click="clearLogs">
          <el-icon><Delete /></el-icon>
          清空
        </el-button>
      </div>
    </div>

    <div class="log-status" v-if="wsStatus !== 'connected'">
      <el-tag :type="wsStatus === 'connecting' ? 'warning' : 'danger'">
        {{ wsStatus === 'connecting' ? '连接中...' : '连接断开' }}
      </el-tag>
      <el-button v-if="wsStatus === 'disconnected'" size="small" @click="reconnect">
        重新连接
      </el-button>
    </div>

    <div ref="logContainer" class="log-container" @scroll="handleScroll">
      <div
        v-for="(line, index) in filteredLogs"
        :key="index"
        class="log-line"
        :class="{ 'log-error': isErrorLine(line), 'log-highlight': isHighlighted(line) }"
      >
        <span class="line-number">{{ index + 1 }}</span>
        <span class="line-content" v-html="highlightKeyword(line)"></span>
      </div>
      <div v-if="loading" class="log-loading">
        <el-icon class="is-loading"><Loading /></el-icon>
        加载中...
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Bottom, Download, Delete, Loading, Search } from '@element-plus/icons-vue'

const props = defineProps({
  runId: {
    type: Number,
    required: true
  },
  stepRunId: {
    type: Number,
    default: null
  }
})

const logContainer = ref(null)
const logs = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const autoScroll = ref(true)
const wsStatus = ref('disconnected')
let ws = null

const filteredLogs = computed(() => {
  if (!searchKeyword.value) return logs.value
  const keyword = searchKeyword.value.toLowerCase()
  return logs.value.filter(line => line.toLowerCase().includes(keyword))
})

const connectWebSocket = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  let url = `${protocol}//${host}/api/v1/pipelines/runs/${props.runId}/logs/stream`
  if (props.stepRunId) {
    url = `${protocol}//${host}/api/v1/pipelines/runs/${props.runId}/steps/${props.stepRunId}/logs/stream`
  }

  wsStatus.value = 'connecting'
  ws = new WebSocket(url)

  ws.onopen = () => {
    wsStatus.value = 'connected'
  }

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      if (data.type === 'log' && data.content) {
        const lines = data.content.split('\n').filter(l => l)
        logs.value.push(...lines)
        if (autoScroll.value) {
          scrollToBottom()
        }
      } else if (data.type === 'status') {
        ElMessage.info(`状态更新: ${data.content}`)
      }
    } catch (e) {
      // 非 JSON 格式，直接作为日志处理
      if (event.data) {
        logs.value.push(event.data)
        if (autoScroll.value) {
          scrollToBottom()
        }
      }
    }
  }

  ws.onclose = () => {
    wsStatus.value = 'disconnected'
  }

  ws.onerror = () => {
    wsStatus.value = 'disconnected'
  }
}

const reconnect = () => {
  if (ws) {
    ws.close()
  }
  connectWebSocket()
}

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

const handleScroll = () => {
  if (!logContainer.value) return
  const { scrollTop, scrollHeight, clientHeight } = logContainer.value
  // 如果用户手动滚动到非底部，禁用自动滚动
  if (scrollHeight - scrollTop - clientHeight > 50) {
    autoScroll.value = false
  }
}

const toggleAutoScroll = () => {
  autoScroll.value = !autoScroll.value
  if (autoScroll.value) {
    scrollToBottom()
  }
}

const handleSearch = () => {
  // 搜索时禁用自动滚动
  if (searchKeyword.value) {
    autoScroll.value = false
  }
}

const isErrorLine = (line) => {
  const errorPatterns = /\b(error|failed|failure|exception|panic|fatal)\b/i
  return errorPatterns.test(line)
}

const isHighlighted = (line) => {
  if (!searchKeyword.value) return false
  return line.toLowerCase().includes(searchKeyword.value.toLowerCase())
}

const highlightKeyword = (line) => {
  if (!searchKeyword.value) return escapeHtml(line)
  const escaped = escapeHtml(line)
  const regex = new RegExp(`(${escapeRegex(searchKeyword.value)})`, 'gi')
  return escaped.replace(regex, '<mark>$1</mark>')
}

const escapeHtml = (text) => {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

const escapeRegex = (string) => {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

const downloadLogs = () => {
  const content = logs.value.join('\n')
  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `pipeline-${props.runId}-logs.txt`
  a.click()
  URL.revokeObjectURL(url)
}

const clearLogs = () => {
  logs.value = []
}

onMounted(() => {
  connectWebSocket()
})

onUnmounted(() => {
  if (ws) {
    ws.close()
  }
})

watch(() => props.runId, () => {
  logs.value = []
  reconnect()
})

watch(() => props.stepRunId, () => {
  logs.value = []
  reconnect()
})
</script>

<style scoped>
.log-viewer {
  display: flex;
  flex-direction: column;
  height: 70vh;
}

.log-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
}

.toolbar-actions {
  display: flex;
  gap: 8px;
}

.log-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #fef0f0;
}

.log-container {
  flex: 1;
  overflow-y: auto;
  background: #1e1e1e;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.log-line {
  display: flex;
  padding: 2px 12px;
  color: #d4d4d4;
}

.log-line:hover {
  background: #2d2d2d;
}

.log-error {
  background: rgba(244, 67, 54, 0.1);
  color: #f44336;
}

.log-highlight {
  background: rgba(255, 235, 59, 0.2);
}

.line-number {
  min-width: 50px;
  color: #858585;
  text-align: right;
  padding-right: 12px;
  user-select: none;
}

.line-content {
  flex: 1;
  white-space: pre-wrap;
  word-break: break-all;
}

.line-content :deep(mark) {
  background: #ffeb3b;
  color: #000;
  padding: 0 2px;
}

.log-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px;
  color: #909399;
}
</style>

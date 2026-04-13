<template>
  <a-modal
    :open="open"
    title="日志上下文"
    width="80%"
    :maskClosable="false"
    destroyOnClose
    @cancel="close"
    :footer="null"
  >
    <div class="context-toolbar">
      <span>上下文行数:</span>
      <a-input-number v-model:value="linesBefore" :min="10" :max="500" size="small" style="width: 100px" />
      <span>前</span>
      <a-input-number v-model:value="linesAfter" :min="10" :max="500" size="small" style="width: 100px" />
      <span>后</span>
      <a-button type="primary" size="small" @click="loadContext" :loading="loading">刷新</a-button>
    </div>

    <div class="context-content" ref="contentRef">
      <div v-if="loading" class="loading-container">
        <LoadingOutlined class="is-loading" />
        <span>加载中...</span>
      </div>
      
      <template v-else-if="contextData">
        <!-- 前面的日志 -->
        <div 
          v-for="(log, index) in contextData.before" 
          :key="`before-${index}`"
          class="log-line"
        >
          <span class="line-number">{{ index + 1 }}</span>
          <span class="timestamp">{{ formatTimestamp(log.timestamp) }}</span>
          <span :class="['level', `level-${log.level?.toLowerCase()}`]">[{{ log.level }}]</span>
          <span class="content">{{ log.content }}</span>
        </div>

        <!-- 当前行（高亮） -->
        <div class="log-line current-line" ref="currentLineRef">
          <span class="line-number">{{ contextData.before.length + 1 }}</span>
          <span class="timestamp">{{ formatTimestamp(contextData.current.timestamp) }}</span>
          <span :class="['level', `level-${contextData.current.level?.toLowerCase()}`]">[{{ contextData.current.level }}]</span>
          <span class="content">{{ contextData.current.content }}</span>
          <a-tag color="warning" class="current-tag">当前行</a-tag>
        </div>

        <!-- 后面的日志 -->
        <div 
          v-for="(log, index) in contextData.after" 
          :key="`after-${index}`"
          class="log-line"
        >
          <span class="line-number">{{ contextData.before.length + 2 + index }}</span>
          <span class="timestamp">{{ formatTimestamp(log.timestamp) }}</span>
          <span :class="['level', `level-${log.level?.toLowerCase()}`]">[{{ log.level }}]</span>
          <span class="content">{{ log.content }}</span>
        </div>
      </template>

      <a-empty v-else description="暂无数据" />
    </div>

    <div class="dialog-footer">
      <span class="stats">
        共 {{ totalLines }} 行 (前 {{ contextData?.total_before || 0 }} 行, 后 {{ contextData?.total_after || 0 }} 行)
      </span>
      <div class="footer-buttons">
        <a-button @click="close">关闭</a-button>
        <a-button type="primary" @click="scrollToCurrent">定位当前行</a-button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { LoadingOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { logApi, type LogEntry } from '@/services/logs'

interface ContextData {
  before: LogEntry[]
  current: LogEntry
  after: LogEntry[]
  total_before: number
  total_after: number
}

const props = defineProps<{
  open: boolean
  clusterId: number
  namespace: string
  podName: string
  container?: string
  log: LogEntry | null
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
}>()

const loading = ref(false)
const contextData = ref<ContextData | null>(null)
const linesBefore = ref(100)
const linesAfter = ref(100)
const contentRef = ref<HTMLElement | null>(null)
const currentLineRef = ref<HTMLElement | null>(null)

const totalLines = computed(() => {
  if (!contextData.value) return 0
  return contextData.value.before.length + 1 + contextData.value.after.length
})

const close = () => {
  emit('update:open', false)
}

const formatTimestamp = (ts: string) => {
  if (!ts) return ''
  try {
    const date = new Date(ts)
    const time = date.toLocaleTimeString('zh-CN', { 
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
    const ms = date.getMilliseconds().toString().padStart(3, '0')
    return `${time}.${ms}`
  } catch {
    return ts.substring(11, 23)
  }
}

const loadContext = async () => {
  if (!props.log) return
  
  loading.value = true
  try {
    const res = await logApi.getLogContext({
      cluster_id: props.clusterId,
      namespace: props.namespace,
      pod_name: props.podName,
      container: props.container || '',
      timestamp: props.log.timestamp,
      lines_before: linesBefore.value,
      lines_after: linesAfter.value
    })
    contextData.value = res.data
    
    // 滚动到当前行
    nextTick(() => scrollToCurrent())
  } catch (error) {
    message.error('加载日志上下文失败')
    console.error(error)
  } finally {
    loading.value = false
  }
}

const scrollToCurrent = () => {
  if (currentLineRef.value) {
    currentLineRef.value.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

watch(() => props.open, (val) => {
  if (val && props.log) {
    loadContext()
  }
})
</script>

<style scoped>
.context-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 15px;
  padding: 10px;
  background: #fafafa;
  border-radius: 4px;
}

.context-content {
  height: 500px;
  overflow: auto;
  background: #1e1e1e;
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  margin-bottom: 15px;
}

.loading-container {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #d4d4d4;
  gap: 10px;
}

.log-line {
  display: flex;
  align-items: flex-start;
  padding: 4px 15px;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
  border-left: 3px solid transparent;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.05);
}

.log-line.current-line {
  background: rgba(255, 193, 7, 0.15);
  border-left-color: #ffc107;
}

.line-number {
  color: #858585;
  margin-right: 15px;
  min-width: 40px;
  text-align: right;
  flex-shrink: 0;
  user-select: none;
}

.timestamp {
  color: #6a9955;
  margin-right: 10px;
  flex-shrink: 0;
}

.level {
  margin-right: 10px;
  flex-shrink: 0;
  font-weight: 500;
}

.level-error, .level-fatal {
  color: #f14c4c;
}

.level-warn, .level-warning {
  color: #cca700;
}

.level-info {
  color: #3794ff;
}

.level-debug {
  color: #808080;
}

.content {
  flex: 1;
}

.current-tag {
  margin-left: 10px;
  flex-shrink: 0;
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
}

.stats {
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
}

.footer-buttons {
  display: flex;
  gap: 8px;
}
</style>

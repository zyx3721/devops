<template>
  <div :class="['log-line', `level-${log.level?.toLowerCase()}`]" :style="lineStyle">
    <span v-if="podColor" class="pod-indicator" :style="{ backgroundColor: podColor }"></span>
    <span class="timestamp">{{ formatTimestamp(log.timestamp) }}</span>
    <span v-if="showPodName && log.pod_name" class="pod-name" :style="{ color: podColor }">[{{ shortPodName }}]</span>
    <span :class="['level', `level-${log.level?.toLowerCase()}`]">[{{ log.level }}]</span>
    <span class="content" v-html="highlightedContent"></span>
    <div class="actions">
      <a-button type="link" size="small" @click="emit('context', log)">
        <template #icon><MoreOutlined /></template>
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { MoreOutlined } from '@ant-design/icons-vue'
import type { LogEntry, HighlightRule } from '@/services/logs'

const props = defineProps<{
  log: LogEntry
  keyword?: string
  highlightRules?: HighlightRule[]
  podColor?: string
  showPodName?: boolean
}>()

const emit = defineEmits<{
  (e: 'context', log: LogEntry): void
}>()

const shortPodName = computed(() => {
  if (!props.log.pod_name) return ''
  // 截取最后两段
  const parts = props.log.pod_name.split('-')
  if (parts.length > 2) {
    return parts.slice(-2).join('-')
  }
  return props.log.pod_name
})

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

const lineStyle = computed(() => {
  if (!props.highlightRules) return {}
  
  // 按优先级排序
  const sortedRules = [...props.highlightRules]
    .filter(r => r.enabled)
    .sort((a, b) => a.priority - b.priority)
  
  for (const rule of sortedRules) {
    if (matchRule(rule)) {
      return {
        color: rule.fg_color || undefined,
        backgroundColor: rule.bg_color || undefined
      }
    }
  }
  
  return {}
})

const matchRule = (rule: HighlightRule): boolean => {
  const content = props.log.content || ''
  const level = props.log.level || ''
  
  switch (rule.match_type) {
    case 'keyword':
      return content.toLowerCase().includes(rule.match_value.toLowerCase())
    case 'regex':
      try {
        return new RegExp(rule.match_value, 'i').test(content)
      } catch {
        return false
      }
    case 'level':
      return level.toUpperCase() === rule.match_value.toUpperCase()
    default:
      return false
  }
}

const highlightedContent = computed(() => {
  let content = escapeHtml(props.log.content || '')
  
  // 高亮关键词
  if (props.keyword) {
    try {
      const regex = new RegExp(`(${escapeRegex(props.keyword)})`, 'gi')
      content = content.replace(regex, '<mark>$1</mark>')
    } catch {
      // 忽略无效正则
    }
  }
  
  // 尝试格式化 JSON
  if (content.includes('{') && content.includes('}')) {
    content = formatJson(content)
  }
  
  return content
})

const escapeHtml = (str: string) => {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

const escapeRegex = (str: string) => {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

const formatJson = (content: string) => {
  // 简单的 JSON 高亮
  return content
    .replace(/"([^"]+)":/g, '<span class="json-key">"$1"</span>:')
    .replace(/: "([^"]+)"/g, ': <span class="json-string">"$1"</span>')
    .replace(/: (\d+)/g, ': <span class="json-number">$1</span>')
    .replace(/: (true|false)/g, ': <span class="json-boolean">$1</span>')
    .replace(/: (null)/g, ': <span class="json-null">$1</span>')
}
</script>

<style scoped>
.log-line {
  display: flex;
  align-items: flex-start;
  padding: 2px 15px;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
  position: relative;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.05);
}

.log-line:hover .actions {
  opacity: 1;
}

.pod-indicator {
  width: 3px;
  height: 100%;
  position: absolute;
  left: 0;
  top: 0;
  min-height: 20px;
}

.timestamp {
  color: #6a9955;
  margin-right: 10px;
  flex-shrink: 0;
}

.pod-name {
  margin-right: 5px;
  flex-shrink: 0;
  font-weight: 500;
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

.actions {
  opacity: 0;
  transition: opacity 0.2s;
  flex-shrink: 0;
}

:deep(mark) {
  background: #613214;
  color: #fff;
  padding: 0 2px;
  border-radius: 2px;
}

:deep(.json-key) {
  color: #9cdcfe;
}

:deep(.json-string) {
  color: #ce9178;
}

:deep(.json-number) {
  color: #b5cea8;
}

:deep(.json-boolean) {
  color: #569cd6;
}

:deep(.json-null) {
  color: #569cd6;
}
</style>

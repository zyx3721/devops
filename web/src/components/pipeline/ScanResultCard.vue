<template>
  <a-card :bordered="false" class="scan-result-card">
    <template #title>
      <div class="card-title">
        <component :is="getIcon()" />
        <span>{{ title }}</span>
      </div>
    </template>
    
    <div class="scan-status">
      <a-tag :color="getStatusColor()">
        {{ getStatusText() }}
      </a-tag>
      <span v-if="result.scanned_at" class="scan-time">
        {{ formatTime(result.scanned_at) }}
      </span>
    </div>

    <div v-if="result.status === 'passed' || result.status === 'failed' || result.status === 'warning'" class="scan-summary">
      <div class="summary-item critical">
        <div class="count">{{ result.critical_count || 0 }}</div>
        <div class="label">严重</div>
      </div>
      <div class="summary-item high">
        <div class="count">{{ result.high_count || 0 }}</div>
        <div class="label">高危</div>
      </div>
      <div class="summary-item medium">
        <div class="count">{{ result.medium_count || 0 }}</div>
        <div class="label">中危</div>
      </div>
      <div class="summary-item low">
        <div class="count">{{ result.low_count || 0 }}</div>
        <div class="label">低危</div>
      </div>
    </div>

    <div v-if="result.status === 'scanning'" class="scanning-progress">
      <a-progress :percent="50" status="active" :show-info="false" />
      <span class="scanning-text">扫描中...</span>
    </div>

    <div v-if="showDetails && result.details" class="scan-details">
      <a-divider style="margin: 12px 0" />
      <slot name="details" :details="parseDetails()">
        <pre class="details-json">{{ parseDetails() }}</pre>
      </slot>
    </div>
  </a-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  BugOutlined,
  SafetyOutlined,
  FileProtectOutlined,
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'

interface Props {
  title: string
  scanType: 'vulnerability' | 'license' | 'quality'
  result: {
    id?: number
    version_id?: number
    scan_type: string
    scanner: string
    status: 'pending' | 'scanning' | 'passed' | 'failed' | 'warning'
    critical_count?: number
    high_count?: number
    medium_count?: number
    low_count?: number
    details?: string
    scanned_at?: string
  }
  showDetails?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showDetails: false,
})

const getIcon = () => {
  switch (props.scanType) {
    case 'vulnerability':
      return BugOutlined
    case 'license':
      return SafetyOutlined
    case 'quality':
      return FileProtectOutlined
    default:
      return BugOutlined
  }
}

const getStatusColor = () => {
  switch (props.result.status) {
    case 'passed':
      return 'success'
    case 'failed':
      return 'error'
    case 'warning':
      return 'warning'
    case 'scanning':
      return 'processing'
    case 'pending':
      return 'default'
    default:
      return 'default'
  }
}

const getStatusText = () => {
  switch (props.result.status) {
    case 'passed':
      return '通过'
    case 'failed':
      return '失败'
    case 'warning':
      return '警告'
    case 'scanning':
      return '扫描中'
    case 'pending':
      return '待扫描'
    default:
      return '未知'
  }
}

const formatTime = (time: string) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

const parseDetails = () => {
  if (!props.result.details) return null
  try {
    return JSON.parse(props.result.details)
  } catch {
    return props.result.details
  }
}
</script>

<style scoped>
.scan-result-card {
  height: 100%;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
}

.scan-status {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.scan-time {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.scan-summary {
  display: flex;
  justify-content: space-around;
  gap: 12px;
}

.summary-item {
  flex: 1;
  text-align: center;
  padding: 12px;
  border-radius: 4px;
  background: #fafafa;
}

.summary-item.critical {
  background: #fff1f0;
}

.summary-item.high {
  background: #fff7e6;
}

.summary-item.medium {
  background: #fffbe6;
}

.summary-item.low {
  background: #f6ffed;
}

.summary-item .count {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 4px;
}

.summary-item.critical .count {
  color: #cf1322;
}

.summary-item.high .count {
  color: #d46b08;
}

.summary-item.medium .count {
  color: #d4b106;
}

.summary-item.low .count {
  color: #389e0d;
}

.summary-item .label {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.scanning-progress {
  text-align: center;
}

.scanning-text {
  display: block;
  margin-top: 8px;
  color: rgba(0, 0, 0, 0.45);
}

.scan-details {
  margin-top: 12px;
}

.details-json {
  background: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  font-size: 12px;
  max-height: 200px;
  overflow: auto;
}
</style>

<template>
  <div class="quota-usage-chart">
    <div class="usage-item">
      <div class="usage-label">
        <DesktopOutlined />
        <span>CPU</span>
      </div>
      <a-progress
        :percent="usage.cpu_percent"
        :status="getStatus(usage.cpu_percent)"
        :stroke-color="getColor(usage.cpu_percent)"
      />
      <div class="usage-text">
        {{ usage.cpu_used }} / {{ quota.max_cpu }}
      </div>
    </div>

    <div class="usage-item">
      <div class="usage-label">
        <DatabaseOutlined />
        <span>内存</span>
      </div>
      <a-progress
        :percent="usage.memory_percent"
        :status="getStatus(usage.memory_percent)"
        :stroke-color="getColor(usage.memory_percent)"
      />
      <div class="usage-text">
        {{ usage.memory_used }} / {{ quota.max_memory }}
      </div>
    </div>

    <div class="usage-item">
      <div class="usage-label">
        <HddOutlined />
        <span>存储</span>
      </div>
      <a-progress
        :percent="usage.storage_percent"
        :status="getStatus(usage.storage_percent)"
        :stroke-color="getColor(usage.storage_percent)"
      />
      <div class="usage-text">
        {{ usage.storage_used }} / {{ quota.max_storage }}
      </div>
    </div>

    <div class="usage-item">
      <div class="usage-label">
        <ThunderboltOutlined />
        <span>并发</span>
      </div>
      <a-progress
        :percent="concurrentPercent"
        :status="getStatus(concurrentPercent)"
        :stroke-color="getColor(concurrentPercent)"
      />
      <div class="usage-text">
        {{ usage.concurrent_used }} / {{ quota.max_concurrent }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  DesktopOutlined,
  DatabaseOutlined,
  HddOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons-vue'

interface Props {
  usage: {
    quota_id: number
    cpu_used: string
    memory_used: string
    storage_used: string
    concurrent_used: number
    cpu_percent: number
    memory_percent: number
    storage_percent: number
  }
  quota: {
    max_cpu: string
    max_memory: string
    max_storage: string
    max_concurrent: number
  }
}

const props = defineProps<Props>()

const concurrentPercent = computed(() => {
  if (props.quota.max_concurrent === 0) return 0
  return Math.round((props.usage.concurrent_used / props.quota.max_concurrent) * 100)
})

const getStatus = (percent: number) => {
  if (percent >= 90) return 'exception'
  if (percent >= 70) return 'normal'
  return 'success'
}

const getColor = (percent: number) => {
  if (percent >= 90) return '#ff4d4f'
  if (percent >= 70) return '#faad14'
  return '#52c41a'
}
</script>

<style scoped>
.quota-usage-chart {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.usage-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.usage-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 4px;
}

.usage-text {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  text-align: right;
}

:deep(.ant-progress) {
  margin-bottom: 0;
}
</style>

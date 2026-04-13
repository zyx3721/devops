<template>
  <div class="log-filter">
    <a-input
      v-model:value="keyword"
      placeholder="搜索关键词..."
      allow-clear
      style="width: 200px"
      @change="onKeywordChange"
    >
      <template #prefix>
        <SearchOutlined />
      </template>
      <template #addonAfter>
        <a-tooltip title="使用正则表达式">
          <a-button :type="useRegex ? 'primary' : 'default'" size="small" @click="toggleRegex" style="border: none; background: transparent; padding: 0; height: auto;">
            .*
          </a-button>
        </a-tooltip>
      </template>
    </a-input>

    <a-select
      v-model:value="level"
      placeholder="日志级别"
      allow-clear
      style="width: 120px"
      @change="onLevelChange"
    >
      <a-select-option value="">全部级别</a-select-option>
      <a-select-option value="FATAL">
        <span style="color: #721C24">● FATAL</span>
      </a-select-option>
      <a-select-option value="ERROR">
        <span style="color: #DC3545">● ERROR</span>
      </a-select-option>
      <a-select-option value="WARN">
        <span style="color: #FFC107">● WARN</span>
      </a-select-option>
      <a-select-option value="INFO">
        <span style="color: #17A2B8">● INFO</span>
      </a-select-option>
      <a-select-option value="DEBUG">
        <span style="color: #6C757D">● DEBUG</span>
      </a-select-option>
    </a-select>

    <a-range-picker
      v-model:value="timeRange"
      show-time
      format="YYYY-MM-DD HH:mm:ss"
      value-format="YYYY-MM-DDTHH:mm:ssZ"
      style="width: 340px"
      @change="onTimeRangeChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { SearchOutlined } from '@ant-design/icons-vue'

const props = defineProps<{
  keyword?: string
  level?: string
  useRegex?: boolean
  startTime?: string
  endTime?: string
}>()

const emit = defineEmits<{
  (e: 'update:keyword', value: string): void
  (e: 'update:level', value: string): void
  (e: 'update:useRegex', value: boolean): void
  (e: 'update:startTime', value: string): void
  (e: 'update:endTime', value: string): void
}>()

const keyword = ref(props.keyword || '')
const level = ref(props.level || undefined)
const useRegex = ref(props.useRegex || false)
const timeRange = ref<[string, string] | null>(null)

watch(() => props.keyword, (val) => {
  keyword.value = val || ''
})

watch(() => props.level, (val) => {
  level.value = val || undefined
})

watch(() => props.useRegex, (val) => {
  useRegex.value = val || false
})

const onKeywordChange = () => {
  emit('update:keyword', keyword.value)
}

const onLevelChange = () => {
  emit('update:level', level.value || '')
}

const toggleRegex = () => {
  useRegex.value = !useRegex.value
  emit('update:useRegex', useRegex.value)
}

const onTimeRangeChange = () => {
  if (timeRange.value) {
    emit('update:startTime', timeRange.value[0])
    emit('update:endTime', timeRange.value[1])
  } else {
    emit('update:startTime', '')
    emit('update:endTime', '')
  }
}
</script>

<style scoped>
.log-filter {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>

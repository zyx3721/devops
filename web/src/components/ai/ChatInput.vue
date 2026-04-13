<template>
  <div class="chat-input">
    <a-textarea
      v-model:value="inputValue"
      :placeholder="placeholder"
      :auto-size="{ minRows: 1, maxRows: 4 }"
      :disabled="disabled"
      @keydown="handleKeydown"
    />
    <div class="chat-input-actions">
      <a-button
        v-if="isStreaming"
        type="text"
        danger
        size="small"
        @click="$emit('stop')"
      >
        <template #icon><StopOutlined /></template>
        停止
      </a-button>
      <a-button
        type="primary"
        :loading="loading"
        :disabled="!canSend"
        @click="handleSend"
      >
        <template #icon><SendOutlined /></template>
        发送
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { SendOutlined, StopOutlined } from '@ant-design/icons-vue'

const props = defineProps<{
  loading?: boolean
  disabled?: boolean
  isStreaming?: boolean
  placeholder?: string
}>()

const emit = defineEmits<{
  (e: 'send', message: string): void
  (e: 'stop'): void
}>()

const inputValue = ref('')

const canSend = computed(() => {
  return inputValue.value.trim() && !props.loading && !props.disabled && !props.isStreaming
})

const handleSend = () => {
  if (!canSend.value) return
  emit('send', inputValue.value.trim())
  inputValue.value = ''
}

const handleKeydown = (e: KeyboardEvent) => {
  // Enter 发送，Shift+Enter 换行
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}
</script>

<style scoped>
.chat-input {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid #f0f0f0;
  background: #fff;
}

.chat-input :deep(.ant-input) {
  resize: none;
  border-radius: 12px;
  border: 1px solid #e8e8e8;
  padding: 12px 16px;
  font-size: 14px;
  line-height: 1.5;
  transition: all 0.2s;
}

.chat-input :deep(.ant-input:hover) {
  border-color: #667eea;
}

.chat-input :deep(.ant-input:focus) {
  border-color: #667eea;
  box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
}

.chat-input :deep(.ant-input::placeholder) {
  color: #bfbfbf;
}

.chat-input-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
}

.chat-input-actions :deep(.ant-btn-primary) {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  border-radius: 8px;
  height: 36px;
  padding: 0 20px;
  font-weight: 500;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
  transition: all 0.2s;
}

.chat-input-actions :deep(.ant-btn-primary:hover) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.chat-input-actions :deep(.ant-btn-primary:active) {
  transform: translateY(0);
}

.chat-input-actions :deep(.ant-btn-primary:disabled) {
  background: #e8e8e8;
  box-shadow: none;
}

.chat-input-actions :deep(.ant-btn-text.ant-btn-dangerous) {
  color: #ff4d4f;
  border-radius: 8px;
  height: 36px;
  padding: 0 12px;
}

.chat-input-actions :deep(.ant-btn-text.ant-btn-dangerous:hover) {
  background: #fff1f0;
}
</style>

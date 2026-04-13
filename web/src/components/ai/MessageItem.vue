<template>
  <div :class="['message-item', `message-${message.role}`]">
    <div class="message-avatar">
      <div v-if="message.role === 'user'" class="avatar user-avatar">
        <UserOutlined />
      </div>
      <div v-else class="avatar ai-avatar">
        <RobotOutlined />
      </div>
    </div>
    <div class="message-content">
      <div class="message-header">
        <span class="message-role">{{ roleLabel }}</span>
        <span class="message-time">{{ formatTime(message.created_at) }}</span>
      </div>
      <div class="message-bubble">
        <div v-if="message.status === 'streaming'" class="message-streaming">
          <span v-html="renderedContent"></span>
          <span class="typing-cursor"></span>
        </div>
        <div v-else-if="message.status === 'error'" class="message-error">
          <ExclamationCircleOutlined />
          <span>{{ message.content || '发生错误，请重试' }}</span>
        </div>
        <div v-else class="message-text" v-html="renderedContent"></div>
      </div>
      <div v-if="message.tool_calls?.length" class="message-tools">
        <div v-for="tc in message.tool_calls" :key="tc.id" class="tool-tag">
          <ToolOutlined />
          <span>{{ tc.function.name }}</span>
        </div>
      </div>
      <!-- 反馈按钮 -->
      <div v-if="message.role === 'assistant' && message.status === 'completed'" class="message-feedback">
        <button 
          :class="['feedback-btn', { active: feedbackRating === 'like' }]"
          @click="handleFeedback('like')"
          :disabled="feedbackLoading"
          title="有帮助"
        >
          <LikeOutlined />
        </button>
        <button 
          :class="['feedback-btn', { active: feedbackRating === 'dislike' }]"
          @click="handleFeedback('dislike')"
          :disabled="feedbackLoading"
          title="没帮助"
        >
          <DislikeOutlined />
        </button>
        <span v-if="feedbackRating" class="feedback-text">
          {{ feedbackRating === 'like' ? '感谢反馈！' : '我会改进的' }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { UserOutlined, RobotOutlined, ExclamationCircleOutlined, ToolOutlined, LikeOutlined, DislikeOutlined } from '@ant-design/icons-vue'
import { marked } from 'marked'
import { message } from 'ant-design-vue'
import { aiApi } from '../../services/ai'
import type { AIMessage } from '../../types/ai'

const props = defineProps<{
  message: AIMessage
}>()

const feedbackRating = ref<'like' | 'dislike' | null>(props.message.feedback_rating as any || null)
const feedbackLoading = ref(false)

watch(() => props.message.feedback_rating, (newVal) => {
  feedbackRating.value = newVal as any || null
})

const roleLabel = computed(() => {
  return props.message.role === 'user' ? '你' : '小运'
})

const renderedContent = computed(() => {
  if (!props.message.content) return ''
  try {
    return marked.parse(props.message.content, { breaks: true })
  } catch {
    return props.message.content
  }
})

const formatTime = (time: string) => {
  if (!time) return ''
  const date = new Date(time)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

const handleFeedback = async (rating: 'like' | 'dislike') => {
  if (feedbackLoading.value || feedbackRating.value === rating) return
  
  feedbackLoading.value = true
  try {
    await aiApi.feedback(props.message.id, { rating })
    feedbackRating.value = rating
    message.success(rating === 'like' ? '感谢您的反馈！' : '感谢反馈，我们会继续改进')
  } catch (e: any) {
    message.error('提交反馈失败')
  } finally {
    feedbackLoading.value = false
  }
}
</script>

<style scoped>
.message-item {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  transition: background 0.2s;
}

.message-item:hover {
  background: rgba(0, 0, 0, 0.02);
}

.message-user {
  flex-direction: row-reverse;
}

.message-user .message-content {
  align-items: flex-end;
}

.message-avatar {
  flex-shrink: 0;
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
}

.user-avatar {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
}

.ai-avatar {
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e8eb 100%);
  color: #667eea;
  border: 1px solid #e8e8e8;
}

.message-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 85%;
  min-width: 0;
}

.message-header {
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: #8c8c8c;
}

.message-user .message-header {
  flex-direction: row-reverse;
}

.message-role {
  font-weight: 500;
  color: #595959;
}

.message-bubble {
  padding: 12px 16px;
  border-radius: 16px;
  word-break: break-word;
  line-height: 1.6;
}

.message-user .message-bubble {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  border-bottom-right-radius: 4px;
}

.message-assistant .message-bubble {
  background: #f7f7f8;
  color: #1f1f1f;
  border-bottom-left-radius: 4px;
}

.message-text :deep(p) {
  margin: 0 0 8px;
}

.message-text :deep(p:last-child) {
  margin-bottom: 0;
}

.message-text :deep(pre) {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 12px 0;
  font-size: 13px;
}

.message-text :deep(code) {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
}

.message-text :deep(p code) {
  background: rgba(0, 0, 0, 0.06);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.message-user .message-text :deep(p code) {
  background: rgba(255, 255, 255, 0.2);
}

.message-text :deep(ul), .message-text :deep(ol) {
  padding-left: 20px;
  margin: 8px 0;
}

.message-text :deep(li) {
  margin: 4px 0;
}

.message-streaming {
  display: inline;
}

.typing-cursor {
  display: inline-block;
  width: 2px;
  height: 16px;
  background: currentColor;
  margin-left: 2px;
  animation: blink 1s infinite;
  vertical-align: text-bottom;
}

@keyframes blink {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0; }
}

.message-error {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #ff4d4f;
}

.message-tools {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 4px;
}

.tool-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%);
  color: #667eea;
  border-radius: 12px;
  font-size: 12px;
}

.message-feedback {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.message-item:hover .message-feedback {
  opacity: 1;
}

.feedback-btn {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #bfbfbf;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.feedback-btn:hover {
  background: #f5f5f5;
  color: #667eea;
}

.feedback-btn.active {
  color: #667eea;
}

.feedback-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.feedback-text {
  font-size: 12px;
  color: #52c41a;
  margin-left: 8px;
}
</style>

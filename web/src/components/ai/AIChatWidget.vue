<template>
  <div class="ai-chat-widget">
    <!-- 悬浮按钮 -->
    <div
      v-if="!isOpen"
      class="ai-float-btn"
      @click="openChat"
    >
      <!-- 气泡提示 -->
      <Transition name="bubble">
        <div v-if="showBubble" class="ai-bubble" @click.stop="openChat">
          <span class="bubble-text">{{ currentTip }}</span>
          <button class="bubble-close" @click.stop="closeBubble">
            <CloseOutlined />
          </button>
          <div class="bubble-arrow"></div>
        </div>
      </Transition>
      <div class="ai-float-btn-inner">
        <RobotOutlined class="ai-icon" />
        <span class="ai-label">AI 助手</span>
      </div>
      <div class="ai-float-btn-pulse"></div>
    </div>

    <!-- 聊天窗口 -->
    <Transition name="chat-popup">
      <div
        v-if="isOpen"
        ref="chatWindowRef"
        class="chat-window"
        :style="windowStyle"
      >
        <!-- 标题栏 -->
        <div class="chat-header" @mousedown="startDrag">
          <div class="chat-title">
            <div class="chat-avatar">
              <RobotOutlined />
            </div>
            <div class="chat-title-text">
              <span class="chat-name">小运</span>
              <span class="chat-status">
                <span class="status-dot"></span>
                在线
              </span>
            </div>
          </div>
          <div class="chat-actions">
            <a-tooltip title="新对话" placement="bottom">
              <button class="action-btn" @click.stop="handleNewConversation">
                <PlusOutlined />
              </button>
            </a-tooltip>
            <a-tooltip title="历史记录" placement="bottom">
              <button class="action-btn" :class="{ active: showHistory }" @click.stop="showHistory = !showHistory">
                <HistoryOutlined />
              </button>
            </a-tooltip>
            <a-tooltip title="关闭" placement="bottom">
              <button class="action-btn close-btn" @click.stop="closeChat">
                <CloseOutlined />
              </button>
            </a-tooltip>
          </div>
        </div>

        <!-- 历史记录面板 -->
        <Transition name="slide-left">
          <div v-if="showHistory" class="chat-history">
            <div class="history-header">
              <span class="history-title">历史对话</span>
              <button class="history-close" @click="showHistory = false">
                <CloseOutlined />
              </button>
            </div>
            <div class="history-list">
              <div
                v-for="conv in conversations"
                :key="conv.id"
                :class="['history-item', { active: currentConversation?.id === conv.id }]"
                @click="handleLoadConversation(conv.id)"
              >
                <div class="history-icon">
                  <MessageOutlined />
                </div>
                <div class="history-content">
                  <div class="history-item-title">{{ conv.title || '新对话' }}</div>
                  <div class="history-meta">
                    <span>{{ conv.message_count }} 条消息</span>
                    <span class="meta-dot">·</span>
                    <span>{{ formatDate(conv.updated_at) }}</span>
                  </div>
                </div>
                <button class="history-delete" @click.stop="handleDeleteConversation(conv.id)">
                  <DeleteOutlined />
                </button>
              </div>
              <div v-if="conversations.length === 0" class="history-empty">
                <InboxOutlined class="empty-icon" />
                <span>暂无历史对话</span>
              </div>
            </div>
          </div>
        </Transition>

        <!-- 消息列表 -->
        <MessageList
          v-show="!showHistory"
          :messages="messages"
          :loading="isLoading"
        />

        <!-- 输入框 -->
        <ChatInput
          v-show="!showHistory"
          :loading="isLoading"
          :is-streaming="isStreaming"
          placeholder="有什么可以帮你的？按 Enter 发送，Shift+Enter 换行"
          @send="sendMessage"
          @stop="stopStreaming"
        />

        <!-- 底部品牌 -->
        <div v-show="!showHistory" class="chat-footer">
          <span>Powered by AI Copilot</span>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  RobotOutlined,
  PlusOutlined,
  HistoryOutlined,
  CloseOutlined,
  MessageOutlined,
  DeleteOutlined,
  InboxOutlined,
} from '@ant-design/icons-vue'
import { storeToRefs } from 'pinia'
import { Modal } from 'ant-design-vue'
import { useAIStore } from '../../stores/ai'
import { collectContext } from '../../services/contextCollector'
import MessageList from './MessageList.vue'
import ChatInput from './ChatInput.vue'

const route = useRoute()
const aiStore = useAIStore()

const {
  isOpen,
  conversations,
  currentConversation,
  messages,
  isLoading,
  isStreaming,
} = storeToRefs(aiStore)

const {
  openChat,
  closeChat,
  newConversation,
  sendMessage,
  stopStreaming,
  loadHistory,
  loadConversation,
  deleteConversation,
  setContext,
} = aiStore

const chatWindowRef = ref<HTMLElement | null>(null)
const showHistory = ref(false)

// 气泡提示
const showBubble = ref(false)
const currentTipIndex = ref(0)
let bubbleTimer: ReturnType<typeof setInterval> | null = null
let bubbleHideTimer: ReturnType<typeof setTimeout> | null = null

const tips = [
  '👋 有什么可以帮你的吗？',
  '🚀 我可以帮你查询部署状态',
  '📊 需要查看监控数据吗？',
  '🔧 遇到问题？让我来帮你排查',
  '💡 试试问我关于 K8s 的问题',
  '📝 我可以帮你分析日志',
  '🎯 需要创建流水线吗？',
]

const currentTip = computed(() => tips[currentTipIndex.value])

const closeBubble = () => {
  showBubble.value = false
  // 关闭后 2 分钟内不再显示
  if (bubbleTimer) {
    clearInterval(bubbleTimer)
    bubbleTimer = null
  }
  setTimeout(() => {
    startBubbleTimer()
  }, 120000)
}

const startBubbleTimer = () => {
  if (bubbleTimer) return
  
  // 首次延迟 10 秒显示
  setTimeout(() => {
    if (!isOpen.value) {
      showBubble.value = true
      // 8 秒后自动隐藏
      bubbleHideTimer = setTimeout(() => {
        showBubble.value = false
      }, 8000)
    }
  }, 10000)
  
  // 之后每 60 秒显示一次
  bubbleTimer = setInterval(() => {
    if (!isOpen.value) {
      currentTipIndex.value = (currentTipIndex.value + 1) % tips.length
      showBubble.value = true
      // 8 秒后自动隐藏
      if (bubbleHideTimer) clearTimeout(bubbleHideTimer)
      bubbleHideTimer = setTimeout(() => {
        showBubble.value = false
      }, 8000)
    }
  }, 60000)
}

// 窗口位置
const position = ref({ x: window.innerWidth - 440, y: window.innerHeight - 680 })
const isDragging = ref(false)
const dragOffset = ref({ x: 0, y: 0 })

const windowStyle = computed(() => ({
  left: `${position.value.x}px`,
  top: `${position.value.y}px`,
}))

// 处理新对话
const handleNewConversation = () => {
  newConversation()
  showHistory.value = false
}

// 处理加载对话
const handleLoadConversation = (id: string) => {
  loadConversation(id)
  showHistory.value = false
}

// 处理删除对话
const handleDeleteConversation = (id: string) => {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除这个对话吗？删除后无法恢复。',
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: () => deleteConversation(id),
  })
}

// 拖拽功能
const startDrag = (e: MouseEvent) => {
  if ((e.target as HTMLElement).closest('button')) return
  isDragging.value = true
  dragOffset.value = {
    x: e.clientX - position.value.x,
    y: e.clientY - position.value.y,
  }
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

const onDrag = (e: MouseEvent) => {
  if (!isDragging.value) return
  const newX = Math.max(0, Math.min(window.innerWidth - 420, e.clientX - dragOffset.value.x))
  const newY = Math.max(0, Math.min(window.innerHeight - 100, e.clientY - dragOffset.value.y))
  position.value = { x: newX, y: newY }
}

const stopDrag = () => {
  isDragging.value = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

// 格式化日期
const formatDate = (date: string) => {
  if (!date) return ''
  const d = new Date(date)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  return d.toLocaleDateString('zh-CN')
}

// 监听路由变化，更新上下文
watch(
  () => route.fullPath,
  () => {
    const context = collectContext(route)
    setContext(context)
  },
  { immediate: true }
)

// 打开聊天时隐藏气泡
watch(isOpen, (open) => {
  if (open) {
    showBubble.value = false
  }
})

// 加载历史记录
onMounted(() => {
  loadHistory()
  startBubbleTimer()
})

onUnmounted(() => {
  stopDrag()
  if (bubbleTimer) {
    clearInterval(bubbleTimer)
    bubbleTimer = null
  }
  if (bubbleHideTimer) {
    clearTimeout(bubbleHideTimer)
    bubbleHideTimer = null
  }
})
</script>

<style scoped>
.ai-chat-widget {
  position: fixed;
  z-index: 1000;
}

/* 悬浮按钮 */
.ai-float-btn {
  position: fixed;
  right: 24px;
  bottom: 24px;
  cursor: pointer;
  z-index: 1001;
}

.ai-float-btn-inner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50px;
  color: #fff;
  font-weight: 500;
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
  transition: all 0.3s ease;
}

.ai-float-btn:hover .ai-float-btn-inner {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.5);
}

.ai-float-btn .ai-icon {
  font-size: 20px;
}

.ai-float-btn .ai-label {
  font-size: 14px;
}

.ai-float-btn-pulse {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 100%;
  height: 100%;
  border-radius: 50px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  animation: pulse 2s infinite;
  z-index: -1;
}

@keyframes pulse {
  0% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 0.5;
  }
  100% {
    transform: translate(-50%, -50%) scale(1.3);
    opacity: 0;
  }
}

/* 气泡提示 */
.ai-bubble {
  position: absolute;
  bottom: 100%;
  right: 0;
  margin-bottom: 12px;
  background: #fff;
  border-radius: 12px;
  padding: 12px 36px 12px 16px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  white-space: nowrap;
  cursor: pointer;
  border: 1px solid rgba(102, 126, 234, 0.2);
}

.bubble-text {
  font-size: 14px;
  color: #333;
  font-weight: 500;
}

.bubble-close {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 50%;
  background: #f5f5f5;
  color: #999;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  transition: all 0.2s;
}

.bubble-close:hover {
  background: #e8e8e8;
  color: #666;
}

.bubble-arrow {
  position: absolute;
  bottom: -8px;
  right: 24px;
  width: 0;
  height: 0;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-top: 8px solid #fff;
  filter: drop-shadow(0 2px 2px rgba(0, 0, 0, 0.06));
}

/* 气泡动画 */
.bubble-enter-active,
.bubble-leave-active {
  transition: all 0.3s ease;
}

.bubble-enter-from,
.bubble-leave-to {
  opacity: 0;
  transform: translateY(10px);
}

/* 聊天窗口 */
.chat-window {
  position: fixed;
  width: 420px;
  height: 640px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.06);
}

/* 标题栏 */
.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  cursor: move;
  user-select: none;
}

.chat-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chat-avatar {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.chat-title-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.chat-name {
  font-size: 16px;
  font-weight: 600;
}

.chat-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  opacity: 0.9;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #52c41a;
  box-shadow: 0 0 8px rgba(82, 196, 26, 0.6);
}

.chat-actions {
  display: flex;
  gap: 4px;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.9);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  font-size: 14px;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.25);
  color: #fff;
}

.action-btn.active {
  background: rgba(255, 255, 255, 0.3);
}

.action-btn.close-btn:hover {
  background: rgba(255, 77, 79, 0.8);
}

/* 历史记录面板 */
.chat-history {
  position: absolute;
  top: 72px;
  left: 0;
  right: 0;
  bottom: 0;
  background: #fff;
  z-index: 10;
  display: flex;
  flex-direction: column;
}

.history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
}

.history-title {
  font-size: 15px;
  font-weight: 600;
  color: #1f1f1f;
}

.history-close {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #999;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.history-close:hover {
  background: #f5f5f5;
  color: #666;
}

.history-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 4px;
}

.history-item:hover {
  background: #f7f7f8;
}

.history-item.active {
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%);
}

.history-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
  flex-shrink: 0;
}

.history-item.active .history-icon {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
}

.history-content {
  flex: 1;
  min-width: 0;
}

.history-item-title {
  font-size: 14px;
  font-weight: 500;
  color: #1f1f1f;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 4px;
}

.history-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #999;
}

.meta-dot {
  color: #d9d9d9;
}

.history-delete {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #999;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: all 0.2s;
}

.history-item:hover .history-delete {
  opacity: 1;
}

.history-delete:hover {
  background: #fff1f0;
  color: #ff4d4f;
}

.history-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  color: #999;
  gap: 12px;
}

.empty-icon {
  font-size: 48px;
  color: #d9d9d9;
}

/* 底部品牌 */
.chat-footer {
  padding: 8px 16px;
  text-align: center;
  font-size: 11px;
  color: #bfbfbf;
  border-top: 1px solid #f5f5f5;
  background: #fafafa;
}

/* 动画 */
.chat-popup-enter-active,
.chat-popup-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.chat-popup-enter-from,
.chat-popup-leave-to {
  opacity: 0;
  transform: scale(0.9) translateY(20px);
}

.slide-left-enter-active,
.slide-left-leave-active {
  transition: all 0.25s ease;
}

.slide-left-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}

.slide-left-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}
</style>

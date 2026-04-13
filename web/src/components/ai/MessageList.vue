<template>
  <div ref="listRef" class="message-list" @scroll="handleScroll">
    <div v-if="messages.length === 0" class="message-empty">
      <div class="empty-icon">
        <RobotOutlined />
      </div>
      <div class="empty-title">你好，我是小运</div>
      <div class="empty-subtitle">DevOps 平台智能助手，随时为你服务</div>
      <div class="feature-list">
        <div class="feature-item">
          <div class="feature-icon"><SearchOutlined /></div>
          <div class="feature-text">查询应用日志和告警信息</div>
        </div>
        <div class="feature-item">
          <div class="feature-icon"><BulbOutlined /></div>
          <div class="feature-text">分析问题并提供解决方案</div>
        </div>
        <div class="feature-item">
          <div class="feature-icon"><ThunderboltOutlined /></div>
          <div class="feature-text">执行运维操作（重启、扩缩容）</div>
        </div>
        <div class="feature-item">
          <div class="feature-icon"><QuestionCircleOutlined /></div>
          <div class="feature-text">解答平台使用相关问题</div>
        </div>
      </div>
    </div>
    <!-- 虚拟滚动容器 -->
    <div v-else class="virtual-list-container" :style="{ height: totalHeight + 'px' }">
      <div class="virtual-list-content" :style="{ transform: `translateY(${offsetY}px)` }">
        <MessageItem
          v-for="msg in visibleMessages"
          :key="msg.id"
          :message="msg"
          :ref="(el: any) => setItemRef(msg.id, el)"
        />
      </div>
    </div>
    <div v-if="loading" class="message-loading">
      <a-spin size="small" />
      <span>正在思考...</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, onMounted, onUnmounted, toRefs } from 'vue'
import { RobotOutlined, SearchOutlined, BulbOutlined, ThunderboltOutlined, QuestionCircleOutlined } from '@ant-design/icons-vue'
import MessageItem from './MessageItem.vue'
import type { AIMessage } from '../../types/ai'

const props = defineProps<{
  messages: AIMessage[]
  loading?: boolean
}>()

const { messages, loading } = toRefs(props)

const listRef = ref<HTMLElement | null>(null)
const itemRefs = ref<Map<string, any>>(new Map())

// 虚拟滚动配置
const ITEM_HEIGHT_ESTIMATE = 120 // 预估每条消息高度
const BUFFER_SIZE = 3 // 缓冲区大小
const itemHeights = ref<Map<string, number>>(new Map())

// 滚动状态
const scrollTop = ref(0)
const containerHeight = ref(400)

// 设置消息项引用
const setItemRef = (id: string, el: any) => {
  if (el) {
    itemRefs.value.set(id, el)
    // 更新实际高度
    nextTick(() => {
      const height = el.$el?.offsetHeight || ITEM_HEIGHT_ESTIMATE
      if (height !== itemHeights.value.get(id)) {
        itemHeights.value.set(id, height)
      }
    })
  }
}

// 获取消息高度
const getItemHeight = (id: string): number => {
  return itemHeights.value.get(id) || ITEM_HEIGHT_ESTIMATE
}

// 计算总高度
const totalHeight = computed(() => {
  return props.messages.reduce((sum, msg) => sum + getItemHeight(msg.id), 0)
})

// 计算可见消息
const visibleMessages = computed(() => {
  if (props.messages.length <= 20) {
    // 消息少时不使用虚拟滚动
    return props.messages
  }

  let accHeight = 0
  let startIndex = 0
  let endIndex = props.messages.length

  // 找到起始索引
  for (let i = 0; i < props.messages.length; i++) {
    const height = getItemHeight(props.messages[i].id)
    if (accHeight + height >= scrollTop.value) {
      startIndex = Math.max(0, i - BUFFER_SIZE)
      break
    }
    accHeight += height
  }

  // 找到结束索引
  accHeight = 0
  for (let i = 0; i < props.messages.length; i++) {
    accHeight += getItemHeight(props.messages[i].id)
    if (accHeight >= scrollTop.value + containerHeight.value) {
      endIndex = Math.min(props.messages.length, i + BUFFER_SIZE + 1)
      break
    }
  }

  return props.messages.slice(startIndex, endIndex)
})

// 计算偏移量
const offsetY = computed(() => {
  if (props.messages.length <= 20) return 0

  let offset = 0
  const firstVisible = visibleMessages.value[0]
  if (firstVisible) {
    const index = props.messages.findIndex(m => m.id === firstVisible.id)
    for (let i = 0; i < index; i++) {
      offset += getItemHeight(props.messages[i].id)
    }
  }
  return offset
})

// 处理滚动
const handleScroll = () => {
  if (listRef.value) {
    scrollTop.value = listRef.value.scrollTop
  }
}

// 自动滚动到底部
const scrollToBottom = () => {
  nextTick(() => {
    if (listRef.value) {
      listRef.value.scrollTop = listRef.value.scrollHeight
    }
  })
}

// 更新容器高度
const updateContainerHeight = () => {
  if (listRef.value) {
    containerHeight.value = listRef.value.clientHeight
  }
}

// 监听消息变化
watch(() => props.messages.length, scrollToBottom)
watch(() => props.messages[props.messages.length - 1]?.content, scrollToBottom)

onMounted(() => {
  updateContainerHeight()
  window.addEventListener('resize', updateContainerHeight)
})

onUnmounted(() => {
  window.removeEventListener('resize', updateContainerHeight)
})

defineExpose({ scrollToBottom })
</script>

<style scoped>
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
  position: relative;
  background: linear-gradient(180deg, #fafbfc 0%, #fff 100%);
}

.message-list::-webkit-scrollbar {
  width: 6px;
}

.message-list::-webkit-scrollbar-track {
  background: transparent;
}

.message-list::-webkit-scrollbar-thumb {
  background: #e0e0e0;
  border-radius: 3px;
}

.message-list::-webkit-scrollbar-thumb:hover {
  background: #d0d0d0;
}

.message-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 32px 24px;
  text-align: center;
  color: #666;
}

.message-empty .empty-icon {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}

.message-empty .empty-icon :deep(.anticon) {
  font-size: 36px;
  color: #667eea;
}

.message-empty .empty-title {
  font-size: 18px;
  font-weight: 600;
  color: #1f1f1f;
  margin-bottom: 8px;
}

.message-empty .empty-subtitle {
  font-size: 14px;
  color: #8c8c8c;
  margin-bottom: 24px;
}

.message-empty .feature-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  max-width: 280px;
}

.message-empty .feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  text-align: left;
  transition: all 0.2s;
}

.message-empty .feature-item:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transform: translateY(-1px);
}

.message-empty .feature-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 16px;
  flex-shrink: 0;
}

.message-empty .feature-text {
  font-size: 13px;
  color: #595959;
  line-height: 1.4;
}

.message-loading {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  margin: 8px 16px;
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.08) 0%, rgba(118, 75, 162, 0.08) 100%);
  border-radius: 12px;
  color: #667eea;
  font-size: 14px;
}

.message-loading :deep(.ant-spin-dot-item) {
  background-color: #667eea;
}

.virtual-list-container {
  position: relative;
}

.virtual-list-content {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
}
</style>

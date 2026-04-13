import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { aiApi, AIStreamHandler } from '../services/ai'
import type {
  AIConversation,
  AIMessage,
  PageContext,
  MessageRole,
  MessageStatus,
} from '../types/ai'

export const useAIStore = defineStore('ai', () => {
  // 状态
  const conversations = ref<AIConversation[]>([])
  const currentConversation = ref<AIConversation | null>(null)
  const messages = ref<AIMessage[]>([])
  const isLoading = ref(false)
  const isStreaming = ref(false)
  const streamingContent = ref('')
  const error = ref<string | null>(null)
  const isOpen = ref(false)
  const currentContext = ref<PageContext | null>(null)

  // 分页状态
  const historyPage = ref(1)
  const historyPageSize = ref(20)
  const historyTotal = ref(0)
  const historyLoading = ref(false)
  const hasMoreHistory = computed(() => conversations.value.length < historyTotal.value)

  // 消息分页状态
  const messagesLoading = ref(false)
  const messagesHasMore = ref(true)
  const messagesLimit = ref(50)

  // 流处理器
  let streamHandler: AIStreamHandler | null = null

  // 计算属性
  const hasConversation = computed(() => currentConversation.value !== null)
  const messageCount = computed(() => messages.value.length)

  // 打开/关闭聊天窗口
  const toggleChat = () => {
    isOpen.value = !isOpen.value
  }

  const openChat = () => {
    isOpen.value = true
  }

  const closeChat = () => {
    isOpen.value = false
  }

  // 设置当前上下文
  const setContext = (context: PageContext) => {
    currentContext.value = context
  }

  // 发送消息
  const sendMessage = async (content: string) => {
    if (!content.trim() || isLoading.value || isStreaming.value) return

    error.value = null
    isLoading.value = true

    // 添加用户消息到列表
    const userMessage: AIMessage = {
      id: `temp-${Date.now()}`,
      conversation_id: currentConversation.value?.id || '',
      role: 'user' as MessageRole,
      content: content,
      status: 'completed' as MessageStatus,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    messages.value.push(userMessage)

    try {
      // 发送请求
      const response = await aiApi.chat({
        conversation_id: currentConversation.value?.id,
        message: content,
        context: currentContext.value || undefined,
      })

      if (response.data) {
        const { conversation_id, message_id } = response.data

        // 更新会话ID
        if (!currentConversation.value) {
          currentConversation.value = {
            id: conversation_id,
            user_id: 0,
            title: content.slice(0, 50),
            status: 'active',
            message_count: 1,
            total_tokens: 0,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          }
        }

        // 添加助手消息占位
        const assistantMessage: AIMessage = {
          id: message_id,
          conversation_id: conversation_id,
          role: 'assistant' as MessageRole,
          content: '',
          status: 'streaming' as MessageStatus,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }
        messages.value.push(assistantMessage)

        // 开始流式接收
        isLoading.value = false
        isStreaming.value = true
        streamingContent.value = ''

        streamHandler = new AIStreamHandler(
          // onContent
          (chunk) => {
            streamingContent.value += chunk
            // 更新最后一条消息
            const lastMsg = messages.value[messages.value.length - 1]
            if (lastMsg && lastMsg.role === 'assistant') {
              lastMsg.content = streamingContent.value
            }
          },
          // onToolCall
          (toolCall) => {
            console.log('Tool call:', toolCall)
          },
          // onDone
          (usage) => {
            isStreaming.value = false
            const lastMsg = messages.value[messages.value.length - 1]
            if (lastMsg && lastMsg.role === 'assistant') {
              lastMsg.status = 'completed'
              lastMsg.token_count = usage?.completion_tokens
            }
            streamingContent.value = ''
          },
          // onError
          (err) => {
            isStreaming.value = false
            error.value = err
            const lastMsg = messages.value[messages.value.length - 1]
            if (lastMsg && lastMsg.role === 'assistant') {
              lastMsg.status = 'error'
              lastMsg.content = lastMsg.content || '抱歉，发生了错误'
            }
          }
        )

        streamHandler.start(message_id)
      }
    } catch (e: any) {
      isLoading.value = false
      error.value = e.message || '发送失败'
      // 移除用户消息
      messages.value.pop()
    }
  }

  // 停止流式响应
  const stopStreaming = () => {
    if (streamHandler) {
      streamHandler.close()
      streamHandler = null
    }
    isStreaming.value = false
  }

  // 加载会话历史（支持懒加载）
  const loadHistory = async (page = 1, pageSize = 20, append = false) => {
    if (historyLoading.value) return
    
    historyLoading.value = true
    try {
      const response = await aiApi.getHistory({ page, page_size: pageSize })
      if (response.data) {
        if (append) {
          // 追加模式：用于懒加载
          const newConvs = response.data.list || []
          const existingIds = new Set(conversations.value.map(c => c.id))
          const uniqueNewConvs = newConvs.filter(c => !existingIds.has(c.id))
          conversations.value = [...conversations.value, ...uniqueNewConvs]
        } else {
          // 替换模式：用于刷新
          conversations.value = response.data.list || []
        }
        historyTotal.value = response.data.total || 0
        historyPage.value = page
      }
    } catch (e: any) {
      console.error('Load history error:', e)
    } finally {
      historyLoading.value = false
    }
  }

  // 加载更多会话历史
  const loadMoreHistory = async () => {
    if (!hasMoreHistory.value || historyLoading.value) return
    await loadHistory(historyPage.value + 1, historyPageSize.value, true)
  }

  // 加载会话详情（支持懒加载消息）
  const loadConversation = async (id: string, limit = 50) => {
    isLoading.value = true
    messagesHasMore.value = true
    messagesLimit.value = limit
    
    try {
      const response = await aiApi.getConversation(id, limit)
      if (response.data) {
        currentConversation.value = response.data
        messages.value = response.data.messages || []
        // 如果返回的消息数量小于limit，说明没有更多消息了
        messagesHasMore.value = (response.data.messages?.length || 0) >= limit
      }
    } catch (e: any) {
      error.value = e.message || '加载失败'
    } finally {
      isLoading.value = false
    }
  }

  // 加载更多消息（向上滚动加载历史消息）
  const loadMoreMessages = async () => {
    if (!currentConversation.value || !messagesHasMore.value || messagesLoading.value) return
    
    messagesLoading.value = true
    try {
      const newLimit = messagesLimit.value + 50
      const response = await aiApi.getConversation(currentConversation.value.id, newLimit)
      if (response.data) {
        const newMessages = response.data.messages || []
        messages.value = newMessages
        messagesLimit.value = newLimit
        messagesHasMore.value = newMessages.length >= newLimit
      }
    } catch (e: any) {
      console.error('Load more messages error:', e)
    } finally {
      messagesLoading.value = false
    }
  }

  // 新建会话
  const newConversation = () => {
    currentConversation.value = null
    messages.value = []
    error.value = null
    streamingContent.value = ''
  }

  // 删除会话
  const deleteConversation = async (id: string) => {
    try {
      await aiApi.deleteConversation(id)
      conversations.value = conversations.value.filter(c => c.id !== id)
      if (currentConversation.value?.id === id) {
        newConversation()
      }
    } catch (e: any) {
      error.value = e.message || '删除失败'
    }
  }

  // 执行操作
  const executeAction = async (action: string, params: Record<string, any>) => {
    try {
      const response = await aiApi.execute({
        action,
        params,
        conversation_id: currentConversation.value?.id,
      })
      return response.data
    } catch (e: any) {
      error.value = e.message || '执行失败'
      throw e
    }
  }

  // 清理
  const cleanup = () => {
    stopStreaming()
    currentConversation.value = null
    messages.value = []
    error.value = null
    streamingContent.value = ''
  }

  return {
    // 状态
    conversations,
    currentConversation,
    messages,
    isLoading,
    isStreaming,
    streamingContent,
    error,
    isOpen,
    currentContext,
    // 分页状态
    historyPage,
    historyTotal,
    historyLoading,
    hasMoreHistory,
    messagesLoading,
    messagesHasMore,
    // 计算属性
    hasConversation,
    messageCount,
    // 方法
    toggleChat,
    openChat,
    closeChat,
    setContext,
    sendMessage,
    stopStreaming,
    loadHistory,
    loadMoreHistory,
    loadConversation,
    loadMoreMessages,
    newConversation,
    deleteConversation,
    executeAction,
    cleanup,
  }
})

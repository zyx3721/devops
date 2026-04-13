import request from './api'
import type { ApiResponse } from '../types'
import type {
  AIConversation,
  AIMessage,
  AIKnowledge,
  AILLMConfig,
  ChatRequest,
  ChatResponse,
  ExecuteRequest,
  ExecuteResult,
  PageContext,
  SSEEvent,
} from '../types/ai'

const BASE_URL = import.meta.env.VITE_API_BASE_URL || '/app/api/v1'

// 重试配置
interface RetryConfig {
  maxRetries: number
  baseDelay: number
  maxDelay: number
}

const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 10000,
}

// 带重试的请求包装器
async function withRetry<T>(
  fn: () => Promise<T>,
  config: RetryConfig = DEFAULT_RETRY_CONFIG
): Promise<T> {
  let lastError: Error | null = null
  
  for (let attempt = 0; attempt <= config.maxRetries; attempt++) {
    try {
      return await fn()
    } catch (error: any) {
      lastError = error
      
      // 不重试的错误类型
      if (error.response?.status === 400 || error.response?.status === 401 || error.response?.status === 403) {
        throw error
      }
      
      if (attempt < config.maxRetries) {
        const delay = Math.min(config.baseDelay * Math.pow(2, attempt), config.maxDelay)
        await new Promise(resolve => setTimeout(resolve, delay))
      }
    }
  }
  
  throw lastError
}

// AI 聊天 API
export const aiApi = {
  // 发送消息（带重试）
  chat: (data: ChatRequest): Promise<ApiResponse<ChatResponse>> => {
    return withRetry(() => request.post('/ai/chat', data))
  },

  // 获取会话历史列表（带重试）
  getHistory: (params?: { page?: number; page_size?: number }): Promise<ApiResponse<{ list: AIConversation[]; total: number }>> => {
    return withRetry(() => request.get('/ai/history', { params }))
  },

  // 获取会话详情（带重试）
  getConversation: (id: string, limit?: number): Promise<ApiResponse<AIConversation>> => {
    return withRetry(() => request.get(`/ai/conversation/${id}`, { params: { limit } }))
  },

  // 删除会话
  deleteConversation: (id: string): Promise<ApiResponse> => {
    return request.delete(`/ai/conversation/${id}`)
  },

  // 执行操作（带重试）
  execute: (data: ExecuteRequest): Promise<ApiResponse<ExecuteResult>> => {
    return withRetry(() => request.post('/ai/execute', data))
  },

  // 提交消息反馈
  feedback: (messageId: string, data: { rating: 'like' | 'dislike'; comment?: string }): Promise<ApiResponse> => {
    return request.post(`/ai/message/${messageId}/feedback`, data)
  },
}

// 知识库 API
export const knowledgeApi = {
  // 获取知识列表
  list: (params?: { category?: string; keyword?: string; page?: number; page_size?: number }): Promise<ApiResponse<{ list: AIKnowledge[]; total: number }>> => {
    return request.get('/ai/knowledge', { params })
  },

  // 获取知识详情
  get: (id: number): Promise<ApiResponse<AIKnowledge>> => {
    return request.get(`/ai/knowledge/${id}`)
  },

  // 创建知识
  create: (data: { title: string; content: string; category: string; tags?: string[] }): Promise<ApiResponse<AIKnowledge>> => {
    return request.post('/ai/knowledge', data)
  },

  // 更新知识
  update: (id: number, data: { title?: string; content?: string; category?: string; tags?: string[] }): Promise<ApiResponse<AIKnowledge>> => {
    return request.put(`/ai/knowledge/${id}`, data)
  },

  // 删除知识
  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/ai/knowledge/${id}`)
  },

  // 搜索知识
  search: (q: string, limit?: number): Promise<ApiResponse<AIKnowledge[]>> => {
    return request.get('/ai/knowledge/search', { params: { q, limit } })
  },

  // 获取分类列表
  getCategories: (): Promise<ApiResponse<{ value: string; label: string }[]>> => {
    return request.get('/ai/knowledge/categories')
  },
}

// LLM 配置 API
export const llmConfigApi = {
  // 获取配置列表
  list: (): Promise<ApiResponse<AILLMConfig[]>> => {
    return request.get('/ai/config')
  },

  // 获取配置详情
  get: (id: number): Promise<ApiResponse<AILLMConfig>> => {
    return request.get(`/ai/config/${id}`)
  },

  // 创建配置
  create: (data: Partial<AILLMConfig>): Promise<ApiResponse<AILLMConfig>> => {
    return request.post('/ai/config', data)
  },

  // 更新配置
  update: (id: number, data: Partial<AILLMConfig>): Promise<ApiResponse<AILLMConfig>> => {
    return request.put(`/ai/config/${id}`, data)
  },

  // 删除配置
  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/ai/config/${id}`)
  },

  // 设置默认配置
  setDefault: (id: number): Promise<ApiResponse> => {
    return request.post(`/ai/config/${id}/default`)
  },

  // 获取提供商列表
  getProviders: (): Promise<ApiResponse<{ value: string; label: string }[]>> => {
    return request.get('/ai/config/providers')
  },
}

// SSE 流式响应处理类（带自动重连）
export class AIStreamHandler {
  private eventSource: EventSource | null = null
  private messageId: string = ''
  private reconnectAttempts: number = 0
  private maxReconnectAttempts: number = 3
  private reconnectDelay: number = 1000
  private isManualClose: boolean = false
  private accumulatedContent: string = ''

  constructor(
    private onContent: (content: string) => void,
    private onToolCall?: (toolCall: any) => void,
    private onDone?: (usage: any) => void,
    private onError?: (error: string) => void,
    private onReconnecting?: (attempt: number) => void
  ) {}

  // 开始监听流式响应
  start(messageId: string): void {
    this.messageId = messageId
    this.isManualClose = false
    this.reconnectAttempts = 0
    this.accumulatedContent = ''
    this.connect()
  }

  // 建立连接
  private connect(): void {
    const token = localStorage.getItem('token')
    const url = `${BASE_URL}/ai/stream/${this.messageId}?token=${token}`

    this.eventSource = new EventSource(url)

    this.eventSource.addEventListener('content', (event) => {
      this.reconnectAttempts = 0 // 重置重连计数
      try {
        const data = JSON.parse(event.data)
        const content = data.content || ''
        this.accumulatedContent += content
        this.onContent(content)
      } catch (e) {
        console.error('Parse content error:', e)
      }
    })

    this.eventSource.addEventListener('tool_call', (event) => {
      try {
        const data = JSON.parse(event.data)
        this.onToolCall?.(data.tool_call)
      } catch (e) {
        console.error('Parse tool_call error:', e)
      }
    })

    this.eventSource.addEventListener('done', (event) => {
      try {
        const data = JSON.parse(event.data)
        this.onDone?.(data.usage)
      } catch (e) {
        console.error('Parse done error:', e)
      }
      this.close()
    })

    this.eventSource.addEventListener('error', (event) => {
      if (event instanceof MessageEvent) {
        try {
          const data = JSON.parse(event.data)
          this.onError?.(data.error || '未知错误')
        } catch (e) {
          this.handleConnectionError()
          return
        }
      } else {
        this.handleConnectionError()
        return
      }
      this.close()
    })

    this.eventSource.onerror = () => {
      this.handleConnectionError()
    }
  }

  // 处理连接错误，尝试重连
  private handleConnectionError(): void {
    if (this.isManualClose) return

    this.closeEventSource()

    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      this.onReconnecting?.(this.reconnectAttempts)
      
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
      setTimeout(() => {
        if (!this.isManualClose) {
          this.connect()
        }
      }, delay)
    } else {
      this.onError?.('连接断开，重试次数已用尽')
    }
  }

  // 关闭 EventSource
  private closeEventSource(): void {
    if (this.eventSource) {
      this.eventSource.close()
      this.eventSource = null
    }
  }

  // 关闭连接
  close(): void {
    this.isManualClose = true
    this.closeEventSource()
  }

  // 获取已累积的内容
  getAccumulatedContent(): string {
    return this.accumulatedContent
  }
}

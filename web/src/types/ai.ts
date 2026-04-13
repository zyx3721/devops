// AI Copilot 相关类型定义

// 消息角色
export type MessageRole = 'user' | 'assistant' | 'system' | 'tool'

// 消息状态
export type MessageStatus = 'pending' | 'streaming' | 'completed' | 'error'

// 会话状态
export type ConversationStatus = 'active' | 'archived' | 'deleted'

// 页面上下文
export interface PageContext {
  page_type: string
  page_path: string
  app_id?: number
  app_name?: string
  cluster_id?: number
  cluster_name?: string
  namespace?: string
  deployment_name?: string
  alert_id?: number
  pipeline_id?: number
  extra_data?: Record<string, any>
}

// 聊天消息
export interface AIMessage {
  id: string
  conversation_id: string
  role: MessageRole
  content: string
  status: MessageStatus
  token_count?: number
  tool_calls?: ToolCall[]
  feedback_rating?: 'like' | 'dislike'
  feedback_comment?: string
  created_at: string
  updated_at: string
}

// 工具调用
export interface ToolCall {
  id: string
  type: string
  function: {
    name: string
    arguments: string
  }
}

// 会话
export interface AIConversation {
  id: string
  user_id: number
  title: string
  status: ConversationStatus
  context?: PageContext
  message_count: number
  total_tokens: number
  messages?: AIMessage[]
  created_at: string
  updated_at: string
}

// 聊天请求
export interface ChatRequest {
  conversation_id?: string
  message: string
  context?: PageContext
}

// 聊天响应
export interface ChatResponse {
  conversation_id: string
  message_id: string
  stream_url: string
}

// 执行操作请求
export interface ExecuteRequest {
  action: string
  params: Record<string, any>
  conversation_id?: string
  message_id?: string
}

// 执行结果
export interface ExecuteResult {
  success: boolean
  message: string
  data?: any
  need_confirm: boolean
  confirm_msg?: string
}

// 知识库分类
export type KnowledgeCategory = 
  | 'application'
  | 'traffic'
  | 'approval'
  | 'k8s'
  | 'monitoring'
  | 'pipeline'
  | 'troubleshooting'
  | 'best_practice'
  | 'faq'
  | 'custom'

// 知识条目
export interface AIKnowledge {
  id: number
  title: string
  content: string
  category: KnowledgeCategory
  tags: string[]
  is_system: boolean
  view_count: number
  created_by?: number
  updated_by?: number
  created_at: string
  updated_at: string
}

// LLM 提供商
export type LLMProvider = 'openai' | 'azure' | 'qwen' | 'zhipu' | 'ollama'

// LLM 配置
export interface AILLMConfig {
  id: number
  name: string
  provider: LLMProvider
  api_url: string
  api_key_encrypted: string
  model_name: string
  max_tokens: number
  temperature: number
  timeout_seconds: number
  is_default: boolean
  is_active: boolean
  description?: string
  created_at: string
  updated_at: string
}

// SSE 事件类型
export type SSEEventType = 'content' | 'tool_call' | 'done' | 'error'

// SSE 事件数据
export interface SSEEvent {
  type: SSEEventType
  content?: string
  tool_call?: ToolCall
  usage?: {
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
  error?: string
}

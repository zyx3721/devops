import request from './api'
import type { ApiResponse } from '../types'

// 钉钉应用
export interface DingtalkApp {
  id?: number
  name: string
  app_key: string
  app_secret: string
  agent_id: number
  project: string
  description?: string
  status: string
  is_default: boolean
  created_at?: string
  updated_at?: string
}

// 钉钉机器人
export interface DingtalkBot {
  id?: number
  name: string
  webhook_url: string
  secret?: string
  description?: string
  status: string
  created_at?: string
  updated_at?: string
}

// 钉钉消息日志
export interface DingtalkMessageLog {
  id: number
  created_at: string
  msg_type: string
  target: string
  content: string
  title?: string
  source: string
  status: string
  error_msg?: string
  app_id?: number
}

// 钉钉用户
export interface DingtalkUser {
  userid: string
  name: string
  mobile?: string
  email?: string
  avatar?: string
}

// 发送工作通知请求
export interface SendMessageRequest {
  app_id?: number
  userid_list: string
  msg_type: 'text' | 'markdown'
  content: string
  title?: string
}

// 发送Webhook请求
export interface SendWebhookRequest {
  bot_id: number
  msg_type: 'text' | 'markdown'
  content: { content?: string; title?: string; text?: string }
  at_all?: boolean
  at_users?: string[]
}


// 应用绑定信息
export interface AppBindings {
  jenkins_instances: Array<{ id: number; name: string; url: string }>
  k8s_clusters: Array<{ id: number; name: string }>
}

// 钉钉应用管理 API
export const dingtalkAppApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ list: DingtalkApp[]; total: number }>> => {
    return request.get('/dingtalk/app', { params: { page, page_size: pageSize } })
  },

  get: (id: number): Promise<ApiResponse<DingtalkApp>> => {
    return request.get(`/dingtalk/app/${id}`)
  },

  create: (data: Partial<DingtalkApp>): Promise<ApiResponse<DingtalkApp>> => {
    return request.post('/dingtalk/app', data)
  },

  update: (id: number, data: Partial<DingtalkApp>): Promise<ApiResponse<DingtalkApp>> => {
    return request.put(`/dingtalk/app/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/dingtalk/app/${id}`)
  },

  setDefault: (id: number): Promise<ApiResponse> => {
    return request.post(`/dingtalk/app/${id}/default`)
  },

  getBindings: (id: number): Promise<ApiResponse<AppBindings>> => {
    return request.get(`/dingtalk/app/${id}/bindings`)
  }
}

// 钉钉机器人管理 API
export const dingtalkBotApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ list: DingtalkBot[]; total: number }>> => {
    return request.get('/dingtalk/bot', { params: { page, page_size: pageSize } })
  },

  get: (id: number): Promise<ApiResponse<DingtalkBot>> => {
    return request.get(`/dingtalk/bot/${id}`)
  },

  create: (data: Partial<DingtalkBot>): Promise<ApiResponse<DingtalkBot>> => {
    return request.post('/dingtalk/bot', data)
  },

  update: (id: number, data: Partial<DingtalkBot>): Promise<ApiResponse<DingtalkBot>> => {
    return request.put(`/dingtalk/bot/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/dingtalk/bot/${id}`)
  }
}

// 钉钉消息 API
export const dingtalkApi = {
  sendMessage: (data: SendMessageRequest): Promise<ApiResponse> => {
    return request.post('/dingtalk/send-message', data)
  },

  sendWebhook: (data: SendWebhookRequest): Promise<ApiResponse> => {
    return request.post('/dingtalk/send-webhook', data)
  },

  searchUser: (query: string, appId?: number): Promise<ApiResponse<DingtalkUser[]>> => {
    return request.post('/dingtalk/user/search', { query, app_id: appId })
  }
}

// 钉钉消息日志 API
export const dingtalkLogApi = {
  list: (page = 1, pageSize = 20, msgType = '', source = ''): Promise<ApiResponse<{ list: DingtalkMessageLog[]; total: number }>> => {
    return request.get('/dingtalk/logs', { params: { page, page_size: pageSize, msg_type: msgType, source } })
  }
}

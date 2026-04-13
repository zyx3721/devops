import request from './api'
import type { ApiResponse } from '../types'

// 企业微信应用
export interface WechatWorkApp {
  id?: number
  name: string
  corp_id: string
  agent_id: number
  secret: string
  project: string
  description?: string
  status: string
  is_default: boolean
  created_at?: string
  updated_at?: string
}

// 企业微信机器人
export interface WechatWorkBot {
  id?: number
  name: string
  webhook_url: string
  description?: string
  status: string
  created_at?: string
  updated_at?: string
}

// 企业微信消息日志
export interface WechatWorkMessageLog {
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

// 企业微信用户
export interface WechatWorkUser {
  userid: string
  name: string
  mobile?: string
  email?: string
  avatar?: string
  department?: number[]
}

// 发送应用消息请求
export interface SendMessageRequest {
  app_id?: number
  to_user: string
  to_party?: string
  to_tag?: string
  msg_type: 'text' | 'markdown' | 'textcard' | 'news'
  content: string
  title?: string
  url?: string
}

// 发送Webhook请求
export interface SendWebhookRequest {
  bot_id: number
  msg_type: 'text' | 'markdown' | 'news'
  content: { content?: string }
  mentioned_list?: string[]
  mentioned_mobile_list?: string[]
}


// 应用绑定信息
export interface AppBindings {
  jenkins_instances: Array<{ id: number; name: string; url: string }>
  k8s_clusters: Array<{ id: number; name: string }>
}

// 企业微信应用管理 API
export const wechatworkAppApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ list: WechatWorkApp[]; total: number }>> => {
    return request.get('/wechatwork/app', { params: { page, page_size: pageSize } })
  },

  get: (id: number): Promise<ApiResponse<WechatWorkApp>> => {
    return request.get(`/wechatwork/app/${id}`)
  },

  create: (data: Partial<WechatWorkApp>): Promise<ApiResponse<WechatWorkApp>> => {
    return request.post('/wechatwork/app', data)
  },

  update: (id: number, data: Partial<WechatWorkApp>): Promise<ApiResponse<WechatWorkApp>> => {
    return request.put(`/wechatwork/app/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/wechatwork/app/${id}`)
  },

  setDefault: (id: number): Promise<ApiResponse> => {
    return request.post(`/wechatwork/app/${id}/default`)
  },

  getBindings: (id: number): Promise<ApiResponse<AppBindings>> => {
    return request.get(`/wechatwork/app/${id}/bindings`)
  }
}

// 企业微信机器人管理 API
export const wechatworkBotApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ list: WechatWorkBot[]; total: number }>> => {
    return request.get('/wechatwork/bot', { params: { page, page_size: pageSize } })
  },

  get: (id: number): Promise<ApiResponse<WechatWorkBot>> => {
    return request.get(`/wechatwork/bot/${id}`)
  },

  create: (data: Partial<WechatWorkBot>): Promise<ApiResponse<WechatWorkBot>> => {
    return request.post('/wechatwork/bot', data)
  },

  update: (id: number, data: Partial<WechatWorkBot>): Promise<ApiResponse<WechatWorkBot>> => {
    return request.put(`/wechatwork/bot/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/wechatwork/bot/${id}`)
  }
}

// 企业微信消息 API
export const wechatworkApi = {
  sendMessage: (data: SendMessageRequest): Promise<ApiResponse> => {
    return request.post('/wechatwork/send-message', data)
  },

  sendWebhook: (data: SendWebhookRequest): Promise<ApiResponse> => {
    return request.post('/wechatwork/send-webhook', data)
  },

  searchUser: (query: string, appId?: number): Promise<ApiResponse<WechatWorkUser[]>> => {
    return request.post('/wechatwork/user/search', { query, app_id: appId })
  }
}

// 企业微信消息日志 API
export const wechatworkLogApi = {
  list: (page = 1, pageSize = 20, msgType = '', source = ''): Promise<ApiResponse<{ list: WechatWorkMessageLog[]; total: number }>> => {
    return request.get('/wechatwork/logs', { params: { page, page_size: pageSize, msg_type: msgType, source } })
  }
}

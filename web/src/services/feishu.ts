import request from './api'
import type { ApiResponse } from '../types'

export interface SendCardRequest {
  receive_id: string
  receive_id_type: 'open_id' | 'user_id' | 'union_id' | 'email' | 'chat_id'
  card_data: {
    title?: string
    services: Array<{
      name: string
      object_id: string
      actions: string[]
      branches?: string[]
    }>
  }
}

export interface SendMessageRequest {
  receive_id: string
  receive_id_type: 'open_id' | 'user_id' | 'union_id' | 'email' | 'chat_id'
  msg_type: 'text' | 'post' | 'image' | 'interactive'
  content: string
}

export interface FeishuApp {
  id?: number
  name: string
  app_id: string
  app_secret: string
  webhook?: string
  project: string
  description?: string
  status: string
  is_default: boolean
  created_at?: string
  updated_at?: string
}

export interface FeishuBot {
  id?: number
  name: string
  webhook_url: string
  project?: string
  secret?: string
  description?: string
  status: string
  message_template_id?: number
  created_at?: string
  updated_at?: string
}

export const feishuApi = {
  sendCard: (data: SendCardRequest): Promise<ApiResponse> => {
    return request.post('/feishu/api/send-card', data)
  },

  sendMessage: (data: SendMessageRequest): Promise<ApiResponse> => {
    return request.post('/feishu/send-message', data)
  },

  getVersion: (): Promise<ApiResponse<{ version: string }>> => {
    return request.get('/feishu/version')
  }
}

// 飞书应用管理
export interface AppBindings {
  jenkins_instances: Array<{ id: number; name: string; url: string }>
  k8s_clusters: Array<{ id: number; name: string }>
}

export const feishuAppApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ list: FeishuApp[]; total: number }>> => {
    return request.get('/feishu/app', { params: { page, page_size: pageSize } })
  },

  get: (id: number): Promise<ApiResponse<FeishuApp>> => {
    return request.get(`/feishu/app/${id}`)
  },

  create: (data: Partial<FeishuApp>): Promise<ApiResponse<FeishuApp>> => {
    return request.post('/feishu/app', data)
  },

  update: (id: number, data: Partial<FeishuApp>): Promise<ApiResponse<FeishuApp>> => {
    return request.put(`/feishu/app/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/feishu/app/${id}`)
  },

  setDefault: (id: number): Promise<ApiResponse> => {
    return request.post(`/feishu/app/${id}/default`)
  },

  getBindings: (id: number): Promise<ApiResponse<AppBindings>> => {
    return request.get(`/feishu/app/${id}/bindings`)
  }
}

// 飞书机器人管理
export const feishuBotApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ list: FeishuBot[]; total: number }>> => {
    return request.get('/feishu/bot', { params: { page, page_size: pageSize } })
  },

  get: (id: number): Promise<ApiResponse<FeishuBot>> => {
    return request.get(`/feishu/bot/${id}`)
  },

  create: (data: Partial<FeishuBot>): Promise<ApiResponse<FeishuBot>> => {
    return request.post('/feishu/bot', data)
  },

  update: (id: number, data: Partial<FeishuBot>): Promise<ApiResponse<FeishuBot>> => {
    return request.put(`/feishu/bot/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/feishu/bot/${id}`)
  }
}

// 飞书回调管理
export const feishuCallbackApi = {
  getStatus: (): Promise<ApiResponse<{ running_apps: string[] }>> => {
    return request.get('/feishu/callback/status')
  },

  refresh: (): Promise<ApiResponse<{ running_apps: string[] }>> => {
    return request.post('/feishu/callback/refresh')
  }
}

// 飞书消息日志
export interface FeishuMessageLog {
  id: number
  created_at: string
  msg_type: string
  receive_id: string
  receive_id_type: string
  content: string
  title?: string
  source: string
  status: string
  error_msg?: string
  app_id?: number
}

export const feishuLogApi = {
  list: (page = 1, pageSize = 20, msgType = '', source = ''): Promise<ApiResponse<{ list: FeishuMessageLog[]; total: number }>> => {
    return request.get('/feishu/logs', { params: { page, page_size: pageSize, msg_type: msgType, source } })
  },

  get: (id: number): Promise<ApiResponse<FeishuMessageLog>> => {
    return request.get(`/feishu/logs/${id}`)
  }
}

// 用户搜索
export interface FeishuUser {
  user_id?: string
  open_id?: string
  union_id?: string
  name?: string
  en_name?: string
  email?: string
  mobile?: string
  avatar?: {
    avatar_72?: string
    avatar_240?: string
    avatar_640?: string
    avatar_origin?: string
  }
  department_ids?: string[]
}

export const feishuUserApi = {
  search: (query: string): Promise<ApiResponse<FeishuUser[]>> => {
    return request.post('/feishu/user/search', { query })
  },

  get: (id: string, userIdType = 'user_id'): Promise<ApiResponse<FeishuUser>> => {
    return request.get(`/feishu/user/${id}`, { params: { user_id_type: userIdType } })
  },

  setToken: (appId: string, userToken: string, refreshToken: string): Promise<ApiResponse> => {
    return request.post('/feishu/user/token', { app_id: appId, user_token: userToken, refresh_token: refreshToken })
  },

  getTokenStatus: (appId?: string): Promise<ApiResponse<{ has_token: boolean; refresh_token: string }>> => {
    return request.get('/feishu/user/token/status', { params: { app_id: appId } })
  },

  getAuthorizeUrl: (): string => {
    return '/app/api/v1/feishu/oauth/authorize'
  }
}

// 群聊管理
export interface FeishuChat {
  chat_id: string
  name: string
  description?: string
  avatar?: string
  owner_id?: string
  owner_id_type?: string
  chat_mode?: string
  chat_type?: string
  external?: boolean
  tenant_key?: string
}

export const feishuChatApi = {
  list: (pageToken = '', pageSize = 20): Promise<ApiResponse<{ list: FeishuChat[]; page_token: string }>> => {
    return request.get('/feishu/chat', { params: { page_token: pageToken, page_size: pageSize } })
  },

  create: (name: string, description: string, userIds: string[], userIdType = 'user_id'): Promise<ApiResponse<{ chat_id: string }>> => {
    return request.post('/feishu/chat', { name, description, user_ids: userIds, user_id_type: userIdType })
  },

  addMembers: (chatId: string, userIds: string[], userIdType = 'user_id'): Promise<ApiResponse> => {
    return request.post(`/feishu/chat/${chatId}/members`, { user_ids: userIds, user_id_type: userIdType })
  }
}

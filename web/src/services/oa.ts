import request from './api'
import type { ApiResponse } from '../types'

export interface StoredJSON {
  id: string
  received_at: string
  ip_address: string
  user_agent: string
  original_data: Record<string, any>
}

export interface OASyncData {
  id: number
  unique_id: string
  source: string
  ip_address: string
  user_agent: string
  original_data: string
  created_at: string
  updated_at: string
}

export interface OAAddress {
  id?: number
  name: string
  url: string
  type: string
  description?: string
  status: string
  is_default: boolean
  created_at?: string
  updated_at?: string
}

export const oaApi = {
  storeJson: (data: Record<string, any>): Promise<ApiResponse<{ id: string }>> => {
    return request.post('/oa/api/store-json', data)
  },

  getJson: (id: string): Promise<ApiResponse<StoredJSON>> => {
    return request.get(`/oa/api/get-json/${id}`)
  },

  getAllJson: (): Promise<ApiResponse<StoredJSON[]>> => {
    return request.get('/oa/api/get-json-all')
  },

  getLatestJson: (): Promise<ApiResponse<{ latest_file: StoredJSON }>> => {
    return request.get('/oa/api/get-latest-json')
  }
}

// OA 地址管理
export const oaAddressApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ list: OAAddress[]; total: number }>> => {
    return request.get('/oa/address', { params: { page, page_size: pageSize } })
  },

  get: (id: number): Promise<ApiResponse<OAAddress>> => {
    return request.get(`/oa/address/${id}`)
  },

  create: (data: Partial<OAAddress>): Promise<ApiResponse<OAAddress>> => {
    return request.post('/oa/address', data)
  },

  update: (id: number, data: Partial<OAAddress>): Promise<ApiResponse<OAAddress>> => {
    return request.put(`/oa/address/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/oa/address/${id}`)
  },

  testConnection: (id: number): Promise<ApiResponse<{ connected: boolean; version?: string; server_version?: string; response_time_ms: number; error?: string }>> => {
    return request.post(`/oa/address/${id}/test-connection`)
  }
}

// OA-Jenkins 集成
export const oaJenkinsApi = {
  testFlow: (data: { receive_id: string; receive_id_type: string }): Promise<ApiResponse> => {
    return request.post('/jk/test-flow', data)
  }
}

// OA 同步管理
export const oaSyncApi = {
  getStatus: (): Promise<ApiResponse<{ running: boolean; synced_count: number }>> => {
    return request.get('/oa/sync/status')
  },

  syncNow: (): Promise<ApiResponse> => {
    return request.post('/oa/sync/now')
  },

  syncForce: (): Promise<ApiResponse> => {
    return request.post('/oa/sync/force')
  },

  testSendCard: (uniqueId: string): Promise<ApiResponse> => {
    return request.post(`/oa/sync/test-card/${uniqueId}`)
  }
}

// OA 通知配置
export interface OANotifyConfig {
  id?: number
  name: string
  app_id: number
  receive_id: string
  receive_id_type: string
  description?: string
  status: string
  is_default: boolean
  created_at?: string
  updated_at?: string
}

export const oaNotifyApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ list: OANotifyConfig[]; total: number }>> => {
    return request.get('/oa/notify', { params: { page, page_size: pageSize } })
  },

  get: (id: number): Promise<ApiResponse<OANotifyConfig>> => {
    return request.get(`/oa/notify/${id}`)
  },

  create: (data: Partial<OANotifyConfig>): Promise<ApiResponse<OANotifyConfig>> => {
    return request.post('/oa/notify', data)
  },

  update: (id: number, data: Partial<OANotifyConfig>): Promise<ApiResponse<OANotifyConfig>> => {
    return request.put(`/oa/notify/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/oa/notify/${id}`)
  },

  setDefault: (id: number): Promise<ApiResponse> => {
    return request.post(`/oa/notify/${id}/default`)
  }
}

// OA 同步数据管理
export const oaDataApi = {
  list: (page = 1, pageSize = 20, source = ''): Promise<ApiResponse<{ list: OASyncData[]; total: number }>> => {
    return request.get('/oa/data', { params: { page, page_size: pageSize, source } })
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/oa/data/${id}`)
  }
}

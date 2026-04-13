import request from './api'
import type { ApiResponse } from '../types'

// 灰度发布记录
export interface CanaryRecord {
  id: number
  created_at: string
  updated_at: string
  application_id: number
  app_name: string
  env_name: string
  image_tag: string
  canary_percent: number
  status: string
  description: string
  operator: string
  finished_at?: string
}

// 灰度状态
export interface CanaryStatus {
  deploy_record_id: number
  status: string
  canary_replicas: number
  stable_replicas: number
  canary_ready: number
  stable_ready: number
  canary_image: string
  stable_image: string
  started_at: string
  canary_healthy: boolean
  error_rate?: string
  restart_count?: number
  traffic_percent: number
  canary_header?: string
  canary_cookie?: string
}

// 创建灰度请求
export interface CanaryStartRequest {
  application_id: number
  env_name: string
  image_tag: string
  canary_percent: number
  canary_replicas?: number
  canary_header?: string
  canary_header_value?: string
  canary_cookie?: string
  description?: string
}

export const canaryApi = {
  // 灰度列表
  list: (params?: {
    page?: number
    page_size?: number
    application_id?: number
    env_name?: string
    status?: string
  }): Promise<ApiResponse<{ list: CanaryRecord[]; total: number }>> => {
    return request.get('/deploy/canary/list', { params })
  },

  // 开始灰度
  start: (data: CanaryStartRequest): Promise<ApiResponse<CanaryStatus>> => {
    return request.post('/deploy/canary/start', data)
  },

  // 获取灰度状态
  getStatus: (recordId: number): Promise<ApiResponse<CanaryStatus>> => {
    return request.get(`/deploy/canary/${recordId}/status`)
  },

  // 调整灰度比例
  adjust: (recordId: number, percent: number): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/canary/${recordId}/adjust`, { percent })
  },

  // 全量发布
  promote: (recordId: number): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/canary/${recordId}/promote`)
  },

  // 回滚
  rollback: (recordId: number, reason?: string): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/canary/${recordId}/rollback`, { reason })
  },

  // 获取灰度历史
  history: (params?: {
    page?: number
    page_size?: number
    application_id?: number
  }): Promise<ApiResponse<{ list: CanaryRecord[]; total: number }>> => {
    return request.get('/deploy/canary/list', { params: { ...params, status: 'success,rolled_back' } })
  }
}

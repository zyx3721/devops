import request from './api'
import type { ApiResponse } from '../types'

// 蓝绿部署记录
export interface BlueGreenRecord {
  id: number
  created_at: string
  updated_at: string
  application_id: number
  app_name: string
  env_name: string
  blue_image_tag: string
  green_image_tag: string
  active_version: 'blue' | 'green'
  status: string
  replicas: number
  description: string
  operator: string
  switched_at?: string
  events?: { type: string; message: string; time: string }[]
}

// 创建蓝绿部署请求
export interface BlueGreenStartRequest {
  application_id: number
  env_name: string
  green_image_tag: string
  replicas?: number
  description?: string
}

export const blueGreenApi = {
  // 蓝绿部署列表
  list: (params?: {
    page?: number
    page_size?: number
    application_id?: number
    env_name?: string
    status?: string
  }): Promise<ApiResponse<{ list: BlueGreenRecord[]; total: number }>> => {
    return request.get('/deploy/bluegreen/list', { params })
  },

  // 开始蓝绿部署
  start: (data: BlueGreenStartRequest): Promise<ApiResponse<BlueGreenRecord>> => {
    return request.post('/deploy/bluegreen/start', data)
  },

  // 获取蓝绿部署状态
  getStatus: (recordId: number): Promise<ApiResponse<BlueGreenRecord>> => {
    return request.get(`/deploy/bluegreen/${recordId}/status`)
  },

  // 切换流量
  switch: (recordId: number): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/bluegreen/${recordId}/switch`)
  },

  // 回滚
  rollback: (recordId: number, reason?: string): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/bluegreen/${recordId}/rollback`, { reason })
  },

  // 清理旧版本
  cleanup: (recordId: number): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/bluegreen/${recordId}/cleanup`)
  }
}

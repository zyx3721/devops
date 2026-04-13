import request from './api'
import type { ApiResponse } from '../types'

export interface AuditLog {
  id: number
  created_at: string
  user_id: number
  username: string
  action: string
  resource: string
  resource_id: number
  resource_name: string
  detail: string
  ip_address: string
  user_agent: string
  status: string
  error_msg: string
}

export interface AuditStats {
  action_stats: { name: string; count: number }[]
  resource_stats: { name: string; count: number }[]
  user_stats: { name: string; count: number }[]
  today_count: number
  week_count: number
}

export interface AuditLogFilter {
  page?: number
  page_size?: number
  username?: string
  action?: string
  resource?: string
  status?: string
  start_time?: string
  end_time?: string
}

export const auditApi = {
  list: (params: AuditLogFilter): Promise<ApiResponse<{ list: AuditLog[]; total: number }>> => {
    return request.get('/audit/logs', { params })
  },

  getStats: (): Promise<ApiResponse<AuditStats>> => {
    return request.get('/audit/stats')
  }
}

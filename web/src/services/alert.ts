import request from './api'
import type { ApiResponse } from '../types'

export interface AlertConfig {
  id?: number
  created_at?: string
  updated_at?: string
  name: string
  type: string
  enabled: boolean
  platform: string // @deprecated: use channels instead
  bot_id?: number  // @deprecated: use channels instead
  template_id?: number
  channels?: string // JSON string
  conditions: string
  description?: string
}

export interface AlertHistory {
  id: number
  created_at: string
  alert_config_id: number
  type: string
  title: string
  content: string
  level: string
  status: string
  ack_status: string
  ack_by?: number
  ack_at?: string
  resolved_by?: number
  resolved_at?: string
  resolve_comment?: string
  silenced: boolean
  silence_id?: number
  escalated: boolean
  escalation_id?: number
  error_msg?: string
  source_id: string
  source_url: string
}

export interface AlertSilence {
  id?: number
  created_at?: string
  updated_at?: string
  name: string
  type: string
  matchers?: string
  start_time: string
  end_time: string
  reason?: string
  status: string
}

export interface AlertEscalation {
  id?: number
  created_at?: string
  updated_at?: string
  name: string
  alert_config_id?: number
  level: string
  delay_minutes: number
  platform: string
  bot_id?: number
  notify_users?: string
  enabled: boolean
  description?: string
}

export interface AlertStats {
  type_stats: { name: string; count: number }[]
  level_stats: { name: string; count: number }[]
  ack_stats: { name: string; count: number }[]
  today_count: number
  pending_count: number
  enabled_count: number
  active_silence_count: number
  enabled_escalation_count: number
}

export const alertApi = {
  // 配置管理
  listConfigs: (params?: { page?: number; page_size?: number; type?: string }): Promise<ApiResponse<{ list: AlertConfig[]; total: number }>> => {
    return request.get('/alert/configs', { params })
  },
  getConfig: (id: number): Promise<ApiResponse<AlertConfig>> => {
    return request.get(`/alert/configs/${id}`)
  },
  createConfig: (data: Partial<AlertConfig>): Promise<ApiResponse<AlertConfig>> => {
    return request.post('/alert/configs', data)
  },
  updateConfig: (id: number, data: Partial<AlertConfig>): Promise<ApiResponse<AlertConfig>> => {
    return request.put(`/alert/configs/${id}`, data)
  },
  deleteConfig: (id: number): Promise<ApiResponse> => {
    return request.delete(`/alert/configs/${id}`)
  },
  toggleConfig: (id: number): Promise<ApiResponse<AlertConfig>> => {
    return request.post(`/alert/configs/${id}/toggle`)
  },

  // 历史记录
  listHistories: (params?: { page?: number; page_size?: number; type?: string; ack_status?: string; level?: string; keyword?: string; start_time?: string; end_time?: string }): Promise<ApiResponse<{ list: AlertHistory[]; total: number }>> => {
    return request.get('/alert/histories', { params })
  },
  ackHistory: (id: number): Promise<ApiResponse<AlertHistory>> => {
    return request.post(`/alert/histories/${id}/ack`)
  },
  resolveHistory: (id: number, comment?: string): Promise<ApiResponse<AlertHistory>> => {
    return request.post(`/alert/histories/${id}/resolve`, { comment })
  },

  // 静默规则
  listSilences: (params?: { page?: number; page_size?: number; status?: string }): Promise<ApiResponse<{ list: AlertSilence[]; total: number }>> => {
    return request.get('/alert/silences', { params })
  },
  getSilence: (id: number): Promise<ApiResponse<AlertSilence>> => {
    return request.get(`/alert/silences/${id}`)
  },
  createSilence: (data: Partial<AlertSilence>): Promise<ApiResponse<AlertSilence>> => {
    return request.post('/alert/silences', data)
  },
  updateSilence: (id: number, data: Partial<AlertSilence>): Promise<ApiResponse<AlertSilence>> => {
    return request.put(`/alert/silences/${id}`, data)
  },
  deleteSilence: (id: number): Promise<ApiResponse> => {
    return request.delete(`/alert/silences/${id}`)
  },
  cancelSilence: (id: number): Promise<ApiResponse<AlertSilence>> => {
    return request.post(`/alert/silences/${id}/cancel`)
  },

  // 升级规则
  listEscalations: (params?: { page?: number; page_size?: number }): Promise<ApiResponse<{ list: AlertEscalation[]; total: number }>> => {
    return request.get('/alert/escalations', { params })
  },
  getEscalation: (id: number): Promise<ApiResponse<AlertEscalation>> => {
    return request.get(`/alert/escalations/${id}`)
  },
  createEscalation: (data: Partial<AlertEscalation>): Promise<ApiResponse<AlertEscalation>> => {
    return request.post('/alert/escalations', data)
  },
  updateEscalation: (id: number, data: Partial<AlertEscalation>): Promise<ApiResponse<AlertEscalation>> => {
    return request.put(`/alert/escalations/${id}`, data)
  },
  deleteEscalation: (id: number): Promise<ApiResponse> => {
    return request.delete(`/alert/escalations/${id}`)
  },
  toggleEscalation: (id: number): Promise<ApiResponse<AlertEscalation>> => {
    return request.post(`/alert/escalations/${id}/toggle`)
  },

  // 统计
  getStats: (): Promise<ApiResponse<AlertStats>> => {
    return request.get('/alert/stats')
  },

  // 趋势数据
  getTrend: (params?: { days?: number }): Promise<ApiResponse<{ items: { date: string; count: number }[]; total: number }>> => {
    return request.get('/alert/trend', { params })
  }
}

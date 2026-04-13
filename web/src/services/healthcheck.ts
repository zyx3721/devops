import request from './api'
import type { ApiResponse } from '../types'

export interface HealthCheckConfig {
  id?: number
  created_at?: string
  updated_at?: string
  name: string
  type: string
  target_id?: number
  target_name?: string
  url?: string
  interval: number
  timeout: number
  retry_count: number
  enabled: boolean
  alert_enabled: boolean
  alert_platform?: string
  alert_bot_id?: number
  last_check_at?: string
  last_status?: string
  last_error?: string
}

export interface HealthCheckHistory {
  id: number
  created_at: string
  config_id: number
  config_name: string
  type: string
  target_name: string
  status: string
  response_time_ms: number
  error_msg?: string
  alert_sent: boolean
}

export interface HealthCheckStats {
  type_stats: { name: string; count: number }[]
  status_stats: { name: string; count: number }[]
  enabled_count: number
  healthy_count: number
  unhealthy_count: number
}

export interface OverallStatus {
  status: string
  healthy: number
  unhealthy: number
  unknown: number
  total: number
}

export interface SSLCertConfig extends HealthCheckConfig {
  cert_expiry_date?: string
  cert_days_remaining?: number
  cert_issuer?: string
  cert_subject?: string
  cert_serial_number?: string
  critical_days?: number
  warning_days?: number
  notice_days?: number
  last_alert_level?: string
  last_alert_at?: string
}

export interface ImportDomainsRequest {
  domains: string[]
  interval?: number
  timeout?: number
  retry_count?: number
  critical_days?: number
  warning_days?: number
  notice_days?: number
  alert_platform?: string
  alert_bot_id?: number
}

export interface ImportDomainsResponse {
  success_count: number
  failed_count: number
  failed_domains?: { domain: string; reason: string }[]
}

export const healthCheckApi = {
  // 配置管理
  listConfigs: (params?: { page?: number; page_size?: number; type?: string }): Promise<ApiResponse<{ list: HealthCheckConfig[]; total: number }>> => {
    return request.get('/healthcheck/configs', { params })
  },

  getConfig: (id: number): Promise<ApiResponse<HealthCheckConfig>> => {
    return request.get(`/healthcheck/configs/${id}`)
  },

  createConfig: (data: Partial<HealthCheckConfig>): Promise<ApiResponse<HealthCheckConfig>> => {
    return request.post('/healthcheck/configs', data)
  },

  updateConfig: (id: number, data: Partial<HealthCheckConfig>): Promise<ApiResponse<HealthCheckConfig>> => {
    return request.put(`/healthcheck/configs/${id}`, data)
  },

  deleteConfig: (id: number): Promise<ApiResponse> => {
    return request.delete(`/healthcheck/configs/${id}`)
  },

  toggleConfig: (id: number): Promise<ApiResponse<HealthCheckConfig>> => {
    return request.post(`/healthcheck/configs/${id}/toggle`)
  },

  checkNow: (id: number): Promise<ApiResponse<HealthCheckHistory>> => {
    return request.post(`/healthcheck/configs/${id}/check`)
  },

  // 历史记录
  listHistories: (params?: { page?: number; page_size?: number; config_id?: number }): Promise<ApiResponse<{ list: HealthCheckHistory[]; total: number }>> => {
    return request.get('/healthcheck/histories', { params })
  },

  // 统计
  getStats: (): Promise<ApiResponse<HealthCheckStats>> => {
    return request.get('/healthcheck/stats')
  },

  getOverallStatus: (): Promise<ApiResponse<OverallStatus>> => {
    return request.get('/healthcheck/status')
  }
}

// SSL 证书检查 API
export const sslCertApi = {
  // 列表查询
  list: (params?: { 
    page?: number
    page_size?: number
    alert_level?: string
    keyword?: string
    sort_by?: string
  }): Promise<ApiResponse<{ list: SSLCertConfig[]; total: number }>> => {
    return request.get('/healthcheck/ssl-domains', { params })
  },

  // 即将过期的证书
  expiring: (params?: { days?: number }): Promise<ApiResponse<{ list: SSLCertConfig[]; total: number }>> => {
    return request.get('/healthcheck/ssl-domains/expiring', { params })
  },

  // 批量导入域名
  importDomains: (data: ImportDomainsRequest): Promise<ApiResponse<ImportDomainsResponse>> => {
    return request.post('/healthcheck/ssl-domains/import', data)
  },

  // 批量配置告警阈值
  batchUpdateAlert: (data: {
    ids: number[]
    critical_days?: number
    warning_days?: number
    notice_days?: number
    alert_platform?: string
    alert_bot_id?: number
  }): Promise<ApiResponse> => {
    return request.put('/healthcheck/ssl-domains/alert-config', data)
  },

  // 导出报告
  exportReport: (params?: { 
    alert_level?: string
    keyword?: string
  }): Promise<ApiResponse<SSLCertConfig[]>> => {
    return request.get('/healthcheck/ssl-domains/export', { params })
  },

  // 创建单个配置
  create: (data: Partial<SSLCertConfig>): Promise<ApiResponse<SSLCertConfig>> => {
    return request.post('/healthcheck/configs', data)
  },

  // 更新配置
  update: (id: number, data: Partial<SSLCertConfig>): Promise<ApiResponse<SSLCertConfig>> => {
    return request.put(`/healthcheck/configs/${id}`, data)
  },

  // 删除配置
  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/healthcheck/configs/${id}`)
  },

  // 切换启用状态
  toggle: (id: number): Promise<ApiResponse<SSLCertConfig>> => {
    return request.post(`/healthcheck/configs/${id}/toggle`)
  },

  // 立即检查
  checkNow: (id: number): Promise<ApiResponse<HealthCheckHistory & { cert_days_remaining?: number }>> => {
    return request.post(`/healthcheck/configs/${id}/check`)
  }
}

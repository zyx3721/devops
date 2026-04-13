import request from './api'
import type { ApiResponse } from '../types'

// 日志条目
export interface LogEntry {
  id?: string
  timestamp: string
  content: string
  level: string
  pod_name?: string
  container?: string
  parsed?: Record<string, any>
}

// 日志查询请求
export interface LogQueryRequest {
  cluster_id: number
  namespace: string
  pod_names?: string[]
  container?: string
  tail_lines?: number
  since_time?: string
  until_time?: string
  keyword?: string
  regex?: string
  level?: string
  page?: number
  page_size?: number
  order?: 'asc' | 'desc'
}

// 日志查询响应
export interface LogQueryResponse {
  total: number
  items: LogEntry[]
  has_more: boolean
}

// 染色规则
export interface HighlightRule {
  id: number
  user_id: number
  name: string
  match_type: 'keyword' | 'regex' | 'level'
  match_value: string
  fg_color: string
  bg_color: string
  priority: number
  enabled: boolean
  is_preset: boolean
}

// 导出请求
export interface LogExportRequest {
  cluster_id: number
  namespace: string
  pod_names?: string[]
  start_time?: string
  end_time?: string
  format: 'txt' | 'json' | 'csv'
  keyword?: string
  level?: string
}

// 导出响应
export interface LogExportResponse {
  task_id: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  progress: number
  url?: string
  error?: string
}

export const logApi = {
  // 查询日志
  query: (data: LogQueryRequest): Promise<ApiResponse<LogQueryResponse>> => {
    return request.get('/logs/query', { params: data })
  },

  // 下载日志
  download: (data: LogQueryRequest): Promise<string> => {
    return request.post('/logs/download', data, {
      responseType: 'text'
    })
  },

  // 获取容器列表
  getContainers: (clusterId: number, namespace: string, pod: string): Promise<ApiResponse<string[]>> => {
    return request.get(`/logs/containers/${clusterId}/${namespace}/${pod}`)
  },

  // 导出日志
  exportLogs: (data: LogExportRequest): Promise<ApiResponse<LogExportResponse>> => {
    return request.post('/logs/export', data)
  },

  // 获取导出状态
  getExportStatus: (taskId: string): Promise<ApiResponse<LogExportResponse>> => {
    return request.get(`/logs/export/${taskId}`)
  },

  // 下载导出文件
  downloadExport: (taskId: string): string => {
    const token = localStorage.getItem('token') || ''
    return `/app/api/v1/logs/export/${taskId}/download?token=${token}`
  },

  // 取消导出
  cancelExport: (taskId: string): Promise<ApiResponse<void>> => {
    return request.post(`/logs/export/${taskId}/cancel`)
  },

  // 获取染色规则
  getHighlightRules: (includePreset = true): Promise<ApiResponse<HighlightRule[]>> => {
    return request.get('/logs/highlight-rules', { params: { include_preset: includePreset } })
  },

  // 获取预设规则
  getPresetRules: (): Promise<ApiResponse<HighlightRule[]>> => {
    return request.get('/logs/highlight-rules/presets')
  },

  // 创建染色规则
  createHighlightRule: (data: Partial<HighlightRule>): Promise<ApiResponse<HighlightRule>> => {
    return request.post('/logs/highlight-rules', data)
  },

  // 更新染色规则
  updateHighlightRule: (id: number, data: Partial<HighlightRule>): Promise<ApiResponse<HighlightRule>> => {
    return request.put(`/logs/highlight-rules/${id}`, data)
  },

  // 删除染色规则
  deleteHighlightRule: (id: number): Promise<ApiResponse<void>> => {
    return request.delete(`/logs/highlight-rules/${id}`)
  },

  // 切换染色规则状态
  toggleHighlightRule: (id: number): Promise<ApiResponse<{ enabled: boolean }>> => {
    return request.post(`/logs/highlight-rules/${id}/toggle`)
  },

  // 解析日志
  parseLogs: (data: { type: string; pattern?: string; log_content: string }): Promise<ApiResponse<{ success: boolean; parsed?: Record<string, any>; error?: string }>> => {
    return request.post('/logs/parse', data)
  },

  // 获取日志上下文
  getLogContext: (data: {
    cluster_id: number
    namespace: string
    pod_name: string
    container?: string
    timestamp: string
    lines_before?: number
    lines_after?: number
  }): Promise<ApiResponse<{
    before: LogEntry[]
    current: LogEntry
    after: LogEntry[]
    total_before: number
    total_after: number
  }>> => {
    return request.get('/logs/context', { params: data })
  },

  // 获取快捷查询列表
  getSavedQueries: (includeShared = true): Promise<ApiResponse<any[]>> => {
    return request.get('/logs/saved-queries', { params: { include_shared: includeShared } })
  },

  // 创建快捷查询
  createSavedQuery: (data: {
    name: string
    description?: string
    query_params: Record<string, any>
    is_shared?: boolean
  }): Promise<ApiResponse<any>> => {
    return request.post('/logs/saved-queries', data)
  },

  // 更新快捷查询
  updateSavedQuery: (id: number, data: {
    name: string
    description?: string
    query_params: Record<string, any>
    is_shared?: boolean
  }): Promise<ApiResponse<any>> => {
    return request.put(`/logs/saved-queries/${id}`, data)
  },

  // 删除快捷查询
  deleteSavedQuery: (id: number): Promise<ApiResponse<void>> => {
    return request.delete(`/logs/saved-queries/${id}`)
  },

  // 使用快捷查询
  useSavedQuery: (id: number): Promise<ApiResponse<void>> => {
    return request.post(`/logs/saved-queries/${id}/use`)
  },

  // 获取告警规则列表
  getAlertRules: (clusterId?: number): Promise<ApiResponse<any[]>> => {
    return request.get('/logs/alert-rules', { params: { cluster_id: clusterId } })
  },

  // 创建告警规则
  createAlertRule: (data: any): Promise<ApiResponse<any>> => {
    return request.post('/logs/alert-rules', data)
  },

  // 更新告警规则
  updateAlertRule: (id: number, data: any): Promise<ApiResponse<any>> => {
    return request.put(`/logs/alert-rules/${id}`, data)
  },

  // 删除告警规则
  deleteAlertRule: (id: number): Promise<ApiResponse<void>> => {
    return request.delete(`/logs/alert-rules/${id}`)
  },

  // 切换告警规则状态
  toggleAlertRule: (id: number): Promise<ApiResponse<{ enabled: boolean }>> => {
    return request.post(`/logs/alert-rules/${id}/toggle`)
  },

  // 获取告警历史
  getAlertHistory: (ruleId?: number, page = 1): Promise<ApiResponse<{ items: any[]; total: number }>> => {
    return request.get('/logs/alert-history', { params: { rule_id: ruleId, page } })
  },

  // 获取解析模板列表
  getParseTemplates: (includePreset = true): Promise<ApiResponse<any[]>> => {
    return request.get('/logs/parse-templates', { params: { include_preset: includePreset } })
  },

  // 获取预设模板
  getPresetTemplates: (): Promise<ApiResponse<any[]>> => {
    return request.get('/logs/parse-templates/presets')
  },

  // 创建解析模板
  createParseTemplate: (data: any): Promise<ApiResponse<any>> => {
    return request.post('/logs/parse-templates', data)
  },

  // 更新解析模板
  updateParseTemplate: (id: number, data: any): Promise<ApiResponse<any>> => {
    return request.put(`/logs/parse-templates/${id}`, data)
  },

  // 删除解析模板
  deleteParseTemplate: (id: number): Promise<ApiResponse<void>> => {
    return request.delete(`/logs/parse-templates/${id}`)
  },

  // 测试解析模板
  testParseTemplate: (data: { type: string; pattern?: string; log_content: string }): Promise<ApiResponse<{ success: boolean; parsed?: any; error?: string }>> => {
    return request.post('/logs/parse-templates/test', data)
  },

  // 获取日志统计
  getLogStats: (data: {
    cluster_id: number
    namespace: string
    pod_name?: string
    start_time?: string
    end_time?: string
    interval?: string
  }): Promise<ApiResponse<{
    total_count: number
    level_counts: Record<string, number>
    trend: Array<{ time: string; count: number; level: string }>
    top_errors: Array<{ pattern: string; count: number; sample: string }>
  }>> => {
    return request.get('/logs/stats', { params: data })
  },

  // 日志对比
  compareLogs: (data: {
    cluster_id: number
    namespace: string
    compare_type: string
    left_pod_name: string
    right_pod_name?: string
    container?: string
    left_start_time?: string
    left_end_time?: string
    right_start_time?: string
    right_end_time?: string
  }): Promise<ApiResponse<{
    left_lines: any[]
    right_lines: any[]
    total_left: number
    total_right: number
    added_count: number
    removed_count: number
    same_count: number
  }>> => {
    return request.post('/logs/compare', data)
  },

  // 获取书签列表
  getBookmarks: (page = 1): Promise<ApiResponse<{ items: any[]; total: number }>> => {
    return request.get('/logs/bookmarks', { params: { page } })
  },

  // 创建书签
  createBookmark: (data: {
    cluster_id: number
    namespace: string
    pod_name: string
    container?: string
    log_timestamp?: string
    content: string
    note?: string
  }): Promise<ApiResponse<any>> => {
    return request.post('/logs/bookmarks', data)
  },

  // 更新书签
  updateBookmark: (id: number, data: { note: string }): Promise<ApiResponse<void>> => {
    return request.put(`/logs/bookmarks/${id}`, data)
  },

  // 删除书签
  deleteBookmark: (id: number): Promise<ApiResponse<void>> => {
    return request.delete(`/logs/bookmarks/${id}`)
  },

  // 分享书签
  shareBookmark: (id: number, expiresInDays: number): Promise<ApiResponse<{ share_url: string }>> => {
    return request.post(`/logs/bookmarks/${id}/share`, { expires_in_days: expiresInDays })
  },

  // 获取分享的书签
  getSharedBookmark: (shareUrl: string): Promise<ApiResponse<any>> => {
    return request.get(`/logs/bookmarks/shared/${shareUrl}`)
  }
}

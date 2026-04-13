import request from '@/services/api'

// 安全概览
export function getSecurityOverview(params?: { cluster_id?: number }) {
  return request.get('/security/overview', { params })
}

// 镜像扫描
export function scanImage(data: { image: string; registry_id?: number }) {
  return request.post('/security/scan', data)
}

export function getScanHistory(params?: { image?: string; status?: string; page?: number; page_size?: number }) {
  return request.get('/security/scans', { params })
}

export function getScanResult(id: number) {
  return request.get(`/security/scans/${id}`)
}

// 镜像仓库
export function getRegistries() {
  return request.get('/security/registries')
}

export function createRegistry(data: {
  name: string
  type: string
  url: string
  username?: string
  password?: string
  is_default?: boolean
}) {
  return request.post('/security/registries', data)
}

export function updateRegistry(id: number, data: {
  name: string
  type: string
  url: string
  username?: string
  password?: string
  is_default?: boolean
}) {
  return request.put(`/security/registries/${id}`, data)
}

export function deleteRegistry(id: number) {
  return request.delete(`/security/registries/${id}`)
}

export function testRegistryConnection(data: {
  type: string
  url: string
  username?: string
  password?: string
}) {
  return request.post('/security/registries/test', data)
}

export function getRegistryImages(id: number) {
  return request.get(`/security/registries/${id}/images`)
}

// 配置检查
export function runConfigCheck(data: { cluster_id: number; namespace?: string; rule_ids?: number[] }) {
  return request.post('/security/config-check', data)
}

export function getCheckHistory(params?: { cluster_id?: number; page?: number; page_size?: number }) {
  return request.get('/security/config-checks', { params })
}

export function getCheckResult(id: number) {
  return request.get(`/security/config-checks/${id}`)
}

// 合规规则
export function getRules(params?: { category?: string; enabled?: boolean }) {
  return request.get('/security/rules', { params })
}

export function createRule(data: {
  name: string
  description?: string
  severity: string
  category: string
  enabled: boolean
  condition_json: string
  remediation?: string
}) {
  return request.post('/security/rules', data)
}

export function updateRule(id: number, data: {
  name: string
  description?: string
  severity: string
  category: string
  enabled: boolean
  condition_json: string
  remediation?: string
}) {
  return request.put(`/security/rules/${id}`, data)
}

export function deleteRule(id: number) {
  return request.delete(`/security/rules/${id}`)
}

export function toggleRule(id: number) {
  return request.post(`/security/rules/${id}/toggle`)
}

// 审计日志
export function getAuditLogs(params?: {
  user_id?: number
  action?: string
  resource_type?: string
  cluster_id?: number
  start_time?: string
  end_time?: string
  page?: number
  page_size?: number
}) {
  return request.get('/security/audit-logs', { params })
}

export function exportAuditLogs(params: {
  format: 'csv' | 'json'
  start_time?: string
  end_time?: string
}) {
  return request.get('/security/audit-logs/export', { params, responseType: 'blob' })
}

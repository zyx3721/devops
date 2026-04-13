import request from './api'
import type { ApiResponse } from '../types'

// 流量统计
export interface TrafficStatistics {
  total_requests: number
  success_requests: number
  failed_requests: number
  rate_limited_count: number
  circuit_breaker_open: boolean
  avg_latency_ms: number
  p99_latency_ms: number
  traffic_distribution: Record<string, number>
}

// 熔断器状态
export interface CircuitBreakerStatus {
  id: number
  name: string
  resource: string
  status: 'closed' | 'open' | 'half_open'
  enabled: boolean
  last_open_time?: string
  failure_count: number
  success_count: number
}

// 规则版本
export interface TrafficRuleVersion {
  id: number
  application_id: number
  rule_type: string
  version: number
  config: string
  created_by: string
  created_at: string
  is_active: boolean
}

export const trafficApi = {
  // 获取流量统计
  getStats: (appId: number): Promise<ApiResponse<TrafficStatistics>> => {
    return request.get(`/applications/${appId}/traffic/stats`)
  },

  // 获取熔断器状态
  getCircuitBreakerStatus: (appId: number): Promise<ApiResponse<{ items: CircuitBreakerStatus[] }>> => {
    return request.get(`/applications/${appId}/traffic/circuitbreaker/status`)
  },

  // 获取 VirtualService 列表
  getVirtualServices: (appId: number): Promise<ApiResponse<{ items: any[] }>> => {
    return request.get(`/applications/${appId}/traffic/istio/virtualservices`)
  },

  // 获取 DestinationRule 列表
  getDestinationRules: (appId: number): Promise<ApiResponse<{ items: any[] }>> => {
    return request.get(`/applications/${appId}/traffic/istio/destinationrules`)
  },

  // 获取 Gateway 列表
  getGateways: (appId: number): Promise<ApiResponse<{ items: any[] }>> => {
    return request.get(`/applications/${appId}/traffic/istio/gateways`)
  },

  // 获取规则版本历史
  getRuleVersions: (appId: number, ruleType?: string): Promise<ApiResponse<{ items: TrafficRuleVersion[] }>> => {
    return request.get(`/applications/${appId}/traffic/versions`, { params: { rule_type: ruleType } })
  },

  // 回滚到指定版本
  rollbackVersion: (appId: number, versionId: number): Promise<ApiResponse<void>> => {
    return request.post(`/applications/${appId}/traffic/versions/${versionId}/rollback`)
  }
}

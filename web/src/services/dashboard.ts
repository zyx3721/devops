import request from './api'
import type { ApiResponse } from '../types'

export interface DashboardStats {
  jenkinsInstances: number
  k8sClusters: number
  users: number
  healthChecks: number
  alertsToday: number
  auditsToday: number
}

export interface HealthOverview {
  status: string
  healthy: number
  unhealthy: number
  unknown: number
  total: number
}

export interface RecentAlert {
  id: number
  type: string
  level: string
  title: string
  status: string
  created_at: string
}

export interface RecentAudit {
  id: number
  username: string
  action: string
  resource: string
  resource_id: string
  created_at: string
}

export const dashboardApi = {
  getStats: (): Promise<ApiResponse<DashboardStats>> => {
    return request.get('/dashboard/stats')
  },

  getHealthOverview: (): Promise<ApiResponse<HealthOverview>> => {
    return request.get('/dashboard/health-overview')
  },

  getRecentAlerts: (): Promise<ApiResponse<RecentAlert[]>> => {
    return request.get('/dashboard/recent-alerts')
  },

  getRecentAudits: (): Promise<ApiResponse<RecentAudit[]>> => {
    return request.get('/dashboard/recent-audits')
  }
}

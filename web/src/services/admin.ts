import request from './api'

// 类型定义
export interface AdminStats {
  total_users: number
  total_k8s_clusters: number
  total_jenkins_instances: number
  total_pipelines: number
  total_applications: number
}

// Admin API - 简化版，只保留统计功能
export const adminApi = {
  // 系统统计
  getStats: () => request.get<AdminStats>('/admin/stats'),
}

export default adminApi

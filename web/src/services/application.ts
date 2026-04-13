import request from './api'
import type { ApiResponse } from '../types'

export interface Application {
  id?: number
  created_at?: string
  updated_at?: string
  name: string
  display_name?: string
  description?: string
  git_repo?: string
  language?: string
  framework?: string
  team?: string
  owner?: string
  status: string
  jenkins_instance_id?: number
  jenkins_job_name?: string
  k8s_cluster_id?: number
  k8s_namespace?: string
  k8s_deployment?: string
  notify_platform?: string
  notify_app_id?: number
  notify_receive_id?: string
  notify_receive_type?: string
}

export interface ApplicationEnv {
  id?: number
  application_id: number
  env_name: string
  branch?: string
  jenkins_job?: string
  k8s_namespace?: string
  k8s_deployment?: string
  replicas?: number
  config?: string
}

export interface DeployRecord {
  id: number
  created_at: string
  application_id: number
  app_name: string
  env_name: string
  version: string
  branch: string
  commit_id: string
  deploy_type: string
  status: string
  jenkins_build: number
  jenkins_url: string
  duration: number
  error_msg?: string
  operator: string
  started_at?: string
  finished_at?: string
}

export interface AppStats {
  app_count: number
  team_stats: { name: string; count: number }[]
  lang_stats: { name: string; count: number }[]
  today_deploys: number
  week_deploys: number
  success_rate: number
}

export const applicationApi = {
  // 应用管理
  list: (params?: { page?: number; page_size?: number; name?: string; team?: string; status?: string; language?: string }): Promise<ApiResponse<{ list: Application[]; total: number }>> => {
    return request.get('/app', { params })
  },

  get: (id: number): Promise<ApiResponse<{ app: Application; envs: ApplicationEnv[] }>> => {
    return request.get(`/app/${id}`)
  },

  create: (data: Partial<Application>): Promise<ApiResponse<Application>> => {
    return request.post('/app', data)
  },

  update: (id: number, data: Partial<Application>): Promise<ApiResponse<Application>> => {
    return request.put(`/app/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse> => {
    return request.delete(`/app/${id}`)
  },

  // 环境管理
  listEnvs: (appId: number): Promise<ApiResponse<ApplicationEnv[]>> => {
    return request.get(`/app/${appId}/envs`)
  },

  createEnv: (appId: number, data: Partial<ApplicationEnv>): Promise<ApiResponse<ApplicationEnv>> => {
    return request.post(`/app/${appId}/envs`, data)
  },

  updateEnv: (appId: number, envId: number, data: Partial<ApplicationEnv>): Promise<ApiResponse<ApplicationEnv>> => {
    return request.put(`/app/${appId}/envs/${envId}`, data)
  },

  deleteEnv: (appId: number, envId: number): Promise<ApiResponse> => {
    return request.delete(`/app/${appId}/envs/${envId}`)
  },

  // 部署记录
  listDeploys: (appId: number, params?: { page?: number; page_size?: number; env_name?: string; status?: string }): Promise<ApiResponse<{ list: DeployRecord[]; total: number }>> => {
    return request.get(`/app/${appId}/deploys`, { params })
  },

  listAllDeploys: (params?: { page?: number; page_size?: number; app_name?: string; env?: string; status?: string }): Promise<ApiResponse<{ list: DeployRecord[]; total: number }>> => {
    return request.get('/app/deploys', { params })
  },

  // 发起部署
  deploy: (appId: number, data: { env_name: string; image_tag?: string; branch?: string; description?: string }): Promise<ApiResponse<DeployRecord>> => {
    return request.post(`/app/${appId}/deploy`, data)
  },

  // 统计
  getStats: (): Promise<ApiResponse<AppStats>> => {
    return request.get('/app/stats')
  },

  getTeams: (): Promise<ApiResponse<string[]>> => {
    return request.get('/app/teams')
  }
}

// 紧急部署
export const emergencyDeploy = (appId: number, data: { 
  env_name: string
  image_tag?: string
  branch?: string
  description?: string
  emergency_reason: string 
}): Promise<ApiResponse<DeployRecord>> => {
  return request.post(`/deploy/records/emergency`, { ...data, application_id: appId })
}

// 获取发布窗口状态
export const getDeployWindowStatus = (appId: number, envName: string): Promise<ApiResponse<{
  in_window: boolean
  message: string
  next_window?: string
}>> => {
  return request.get(`/deploy/window/${appId}/${envName}`)
}

// 获取发布锁状态
export const getDeployLockStatus = (appId: number, envName: string): Promise<ApiResponse<{
  locked: boolean
  locked_by?: string
  locked_at?: string
}>> => {
  return request.get(`/deploy/locks/${appId}/${envName}`)
}

// 检查是否需要审批
export const checkApprovalRequired = (appId: number, envName: string): Promise<ApiResponse<{
  required: boolean
  approvers?: string[]
}>> => {
  return request.get(`/approval/check`, { params: { app_id: appId, env: envName } })
}

// 扩展 applicationApi
Object.assign(applicationApi, {
  emergencyDeploy,
  getDeployWindowStatus,
  getDeployLockStatus,
  checkApprovalRequired
})

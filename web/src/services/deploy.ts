import request from './api'
import type { ApiResponse } from '../types'

// 部署记录（含审批）
export interface DeployRecord {
  id: number
  created_at: string
  updated_at: string
  application_id: number
  app_name: string
  env_name: string
  version: string
  branch: string
  commit_id: string
  image_tag: string
  deploy_type: string
  deploy_method: string
  description: string
  status: string
  need_approval: boolean
  approver_id?: number
  approver_name?: string
  approved_at?: string
  reject_reason?: string
  jenkins_build: number
  jenkins_url: string
  duration: number
  error_msg?: string
  operator: string
  operator_id: number
  started_at?: string
  finished_at?: string
  rollback_from?: number
}

// 发布锁
export interface DeployLock {
  id: number
  created_at: string
  application_id: number
  env_name: string
  record_id: number
  locked_by: number
  locked_by_name: string
  expires_at: string
  status: string
  released_at?: string
  released_by?: number
  release_reason?: string
}

// 审批记录
export interface ApprovalRecord {
  id: number
  created_at: string
  record_id: number
  approver_id: number
  approver_name: string
  action: string
  comment?: string
}

// 部署统计
export interface DeployStats {
  total: number
  success: number
  failed: number
  success_rate: number
  avg_duration: number
}

// 创建部署参数
export interface CreateDeployDTO {
  application_id: number
  env_name: string
  version?: string
  branch?: string
  commit_id?: string
  image_tag?: string
  deploy_method?: string  // jenkins, k8s
  description?: string
}

// 回滚参数
export interface RollbackDTO {
  application_id: number
  env_name: string
}

export const deployApi = {
  // 部署记录
  createDeploy: (data: CreateDeployDTO): Promise<ApiResponse<DeployRecord>> => {
    return request.post('/deploy/records', data)
  },

  listRecords: (params?: {
    page?: number
    page_size?: number
    application_id?: number
    app_name?: string
    env_name?: string
    status?: string
  }): Promise<ApiResponse<{ list: DeployRecord[]; total: number }>> => {
    return request.get('/deploy/records', { params })
  },

  getRecord: (id: number): Promise<ApiResponse<{
    record: DeployRecord
    approvals: ApprovalRecord[]
  }>> => {
    return request.get(`/deploy/records/${id}`)
  },

  cancelDeploy: (id: number): Promise<ApiResponse> => {
    return request.post(`/deploy/records/${id}/cancel`)
  },

  approveDeploy: (id: number, comment?: string): Promise<ApiResponse> => {
    return request.post(`/deploy/records/${id}/approve`, { comment })
  },

  rejectDeploy: (id: number, reason: string): Promise<ApiResponse> => {
    return request.post(`/deploy/records/${id}/reject`, { reason })
  },

  executeDeploy: (id: number): Promise<ApiResponse> => {
    return request.post(`/deploy/records/${id}/execute`)
  },

  // 回滚
  createRollback: (data: RollbackDTO): Promise<ApiResponse<DeployRecord>> => {
    return request.post('/deploy/rollback', data)
  },

  getAvailableRollback: (appId: number, env: string): Promise<ApiResponse<DeployRecord>> => {
    return request.get(`/deploy/rollback/${appId}/${env}/available`)
  },

  // 锁管理
  getLockStatus: (appId: number, env: string): Promise<ApiResponse<{ locked: boolean; lock?: DeployLock }>> => {
    return request.get(`/deploy/locks/${appId}/${env}`)
  },

  releaseLock: (appId: number, env: string, reason?: string): Promise<ApiResponse> => {
    return request.post(`/deploy/locks/${appId}/${env}/release`, { reason })
  },

  // 统计
  getStats: (params?: {
    application_id?: number
    env_name?: string
    start_time?: string
    end_time?: string
  }): Promise<ApiResponse<DeployStats>> => {
    return request.get('/deploy/stats', { params })
  }
}


// ==================== 部署前置检查 ====================

export interface PreCheckItem {
  name: string
  status: 'passed' | 'warning' | 'failed' | 'skipped'
  message: string
  detail?: string
}

export interface DeployPreCheckRequest {
  application_id: number
  env_name: string
  image_tag?: string
}

export interface DeployPreCheckResponse {
  can_deploy: boolean
  checks: PreCheckItem[]
  warnings: string[]
  errors: string[]
}

// ==================== 灰度发布 ====================

export interface CanaryDeployRequest {
  application_id: number
  env_name: string
  image_tag: string
  canary_percent: number
  canary_replicas?: number
  description?: string
}

export interface CanaryStatus {
  deploy_record_id: number
  status: string
  canary_replicas: number
  stable_replicas: number
  canary_ready: number
  stable_ready: number
  canary_image: string
  stable_image: string
  started_at: string
  canary_healthy: boolean
  error_rate?: string
}

export interface CanaryRollbackRequest {
  deploy_record_id: number
  reason?: string
}

// ==================== 自动回滚 ====================

export interface AutoRollbackConfig {
  enabled: boolean
  health_check_period: number
  failure_threshold: number
  success_threshold: number
}

export interface DeployHealthStatus {
  deploy_record_id: number
  status: string
  ready_replicas: number
  desired_replicas: number
  unavailable_count: number
  restart_count: number
  last_check_time: string
  consecutive_fails: number
  should_rollback: boolean
  rollback_reason?: string
}

export const deployCheckApi = {
  // 部署前置检查
  preCheck: (data: DeployPreCheckRequest): Promise<ApiResponse<DeployPreCheckResponse>> => {
    return request.post('/deploy/pre-check', data)
  },

  // 灰度发布
  startCanary: (data: CanaryDeployRequest): Promise<ApiResponse<CanaryStatus>> => {
    return request.post('/deploy/canary/start', data)
  },

  getCanaryStatus: (recordId: number): Promise<ApiResponse<CanaryStatus>> => {
    return request.get(`/deploy/canary/${recordId}/status`)
  },

  promoteCanary: (recordId: number): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/canary/${recordId}/promote`)
  },

  rollbackCanary: (recordId: number, reason?: string): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/canary/${recordId}/rollback`, { reason })
  },

  // 自动回滚
  getDeployHealth: (recordId: number): Promise<ApiResponse<DeployHealthStatus>> => {
    return request.get(`/deploy/auto-rollback/${recordId}/health`)
  },

  updateRollbackConfig: (recordId: number, config: AutoRollbackConfig): Promise<ApiResponse<void>> => {
    return request.post(`/deploy/auto-rollback/${recordId}/config`, config)
  }
}

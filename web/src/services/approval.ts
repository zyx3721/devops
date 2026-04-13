import request from './api'
import type { ApiResponse } from '../types'

// 审批规则
export interface ApprovalRule {
  id: number
  app_id: number
  env: string
  need_approval: boolean
  approvers: string
  timeout_minutes: number
  enabled: boolean
  created_by: number
  created_at: string
  updated_at: string
}

// 发布窗口
export interface DeployWindow {
  id: number
  app_id: number
  env: string
  weekdays: string
  start_time: string
  end_time: string
  allow_emergency: boolean
  enabled: boolean
  created_by: number
  created_at: string
  updated_at: string
}

// 审批记录
export interface ApprovalRecord {
  id: number
  record_id: number
  approver_id: number
  approver_name: string
  action: string
  comment: string
  created_at: string
}

// 发布锁
export interface DeployLock {
  id: number
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
  created_at: string
}

// 审批规则 API
export const approvalRuleApi = {
  list: (appId?: number): Promise<ApiResponse<ApprovalRule[]>> => {
    return request.get('/approval/rules', { params: { app_id: appId } })
  },

  get: (id: number): Promise<ApiResponse<ApprovalRule>> => {
    return request.get(`/approval/rules/${id}`)
  },

  create: (data: Partial<ApprovalRule>): Promise<ApiResponse<ApprovalRule>> => {
    return request.post('/approval/rules', data)
  },

  update: (id: number, data: Partial<ApprovalRule>): Promise<ApiResponse<ApprovalRule>> => {
    return request.put(`/approval/rules/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse<void>> => {
    return request.delete(`/approval/rules/${id}`)
  }
}

// 发布窗口 API
export const deployWindowApi = {
  list: (appId?: number): Promise<ApiResponse<DeployWindow[]>> => {
    return request.get('/approval/windows', { params: { app_id: appId } })
  },

  get: (id: number): Promise<ApiResponse<DeployWindow>> => {
    return request.get(`/approval/windows/${id}`)
  },

  create: (data: Partial<DeployWindow>): Promise<ApiResponse<DeployWindow>> => {
    return request.post('/approval/windows', data)
  },

  update: (id: number, data: Partial<DeployWindow>): Promise<ApiResponse<DeployWindow>> => {
    return request.put(`/approval/windows/${id}`, data)
  },

  delete: (id: number): Promise<ApiResponse<void>> => {
    return request.delete(`/approval/windows/${id}`)
  },

  check: (appId: number, env: string): Promise<ApiResponse<{ in_window: boolean; allow_emergency: boolean }>> => {
    return request.get('/approval/windows/check', { params: { app_id: appId, env } })
  }
}

// 审批操作 API
export const approvalApi = {
  getPendingList: (): Promise<ApiResponse<any[]>> => {
    return request.get('/approval/pending')
  },

  getHistory: (params: { page?: number; page_size?: number; app_id?: number; env?: string; status?: string }): Promise<ApiResponse<{ list: any[]; total: number }>> => {
    return request.get('/approval/history', { params })
  },

  approve: (id: number, comment?: string): Promise<ApiResponse<void>> => {
    return request.post(`/approval/${id}/approve`, { comment })
  },

  reject: (id: number, reason: string): Promise<ApiResponse<void>> => {
    return request.post(`/approval/${id}/reject`, { reason })
  },

  cancel: (id: number): Promise<ApiResponse<void>> => {
    return request.post(`/approval/${id}/cancel`)
  },

  getRecords: (id: number): Promise<ApiResponse<ApprovalRecord[]>> => {
    return request.get(`/approval/${id}/records`)
  }
}

// 发布锁 API
export const deployLockApi = {
  list: (): Promise<ApiResponse<DeployLock[]>> => {
    return request.get('/deploy/locks')
  },

  check: (appId: number, env: string): Promise<ApiResponse<{ locked: boolean; lock?: DeployLock }>> => {
    return request.get('/deploy/locks/check', { params: { app_id: appId, env } })
  },

  forceRelease: (appId: number, env: string, reason: string): Promise<ApiResponse<void>> => {
    return request.post('/deploy/locks/release', { reason }, { params: { app_id: appId, env } })
  }
}

// 扩展 approvalApi
Object.assign(approvalApi, {
  // 导出审批历史
  exportHistory: (params: { 
    format: string
    env?: string
    status?: string
    start_time?: string
    end_time?: string 
  }): Promise<Blob> => {
    return request.get('/approval/history/export', { 
      params,
      responseType: 'blob'
    })
  },

  // 获取发布请求详情
  getDeployRequest: (id: number): Promise<ApiResponse<{
    request: any
    records: ApprovalRecord[]
    approvers: any[]
    can_approve: boolean
  }>> => {
    return request.get(`/approval/request/${id}`)
  }
})

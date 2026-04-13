import request from './api'
import type { ApiResponse, JenkinsInstance, PaginatedResponse } from '../types'

export interface JenkinsInstanceRequest {
  page?: number
  page_size?: number
  keyword?: string
  status?: string
}

export interface CreateJenkinsInstanceRequest {
  name: string
  url: string
  username?: string
  api_token?: string
  description?: string
  status: string
  is_default?: boolean
}

export interface UpdateJenkinsInstanceRequest extends CreateJenkinsInstanceRequest {}

export const jenkinsInstanceApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ items: JenkinsInstance[]; total: number }>> => {
    return request.get('/jenkins-instances', { params: { page, page_size: pageSize } })
  },

  getInstances: (params: JenkinsInstanceRequest = {}): Promise<ApiResponse<PaginatedResponse<JenkinsInstance>>> => {
    return request.get('/jenkins-instances', { params })
  },

  getInstance: (id: number): Promise<ApiResponse<JenkinsInstance>> => {
    return request.get(`/jenkins-instances/${id}`)
  },

  getDefaultInstance: (): Promise<ApiResponse<JenkinsInstance>> => {
    return request.get('/jenkins-instances/default')
  },

  createInstance: (data: CreateJenkinsInstanceRequest): Promise<ApiResponse<JenkinsInstance>> => {
    return request.post('/jenkins-instances', data)
  },

  updateInstance: (id: number, data: UpdateJenkinsInstanceRequest): Promise<ApiResponse<JenkinsInstance>> => {
    return request.put(`/jenkins-instances/${id}`, data)
  },

  setDefaultInstance: (id: number): Promise<ApiResponse> => {
    return request.put(`/jenkins-instances/${id}/default`)
  },

  deleteInstance: (id: number): Promise<ApiResponse> => {
    return request.delete(`/jenkins-instances/${id}`)
  },

  testConnection: (id: number): Promise<ApiResponse<ConnectionTestResult>> => {
    return request.post(`/jenkins-instances/${id}/test-connection`)
  },

  getFeishuApps: (id: number): Promise<ApiResponse<FeishuAppSimple[]>> => {
    return request.get(`/jenkins-instances/${id}/feishu-apps`)
  },

  bindFeishuApps: (id: number, appIds: number[]): Promise<ApiResponse> => {
    return request.put(`/jenkins-instances/${id}/feishu-apps`, { app_ids: appIds })
  }
}

export interface ConnectionTestResult {
  connected: boolean
  version?: string
  server_version?: string
  node_count?: number
  response_time_ms: number
  error?: string
}

export interface FeishuAppSimple {
  id: number
  name: string
  app_id: string
  project: string
}

export interface JenkinsJob {
  name: string
  url: string
  color: string
  class: string
  last_build_number: number
  last_build_result: string
  last_build_time: string
}

export interface JenkinsBuild {
  number: number
  result: string
  building: boolean
  timestamp: string
  duration: number
  url: string
}

export const jenkinsJobApi = {
  getJobs: (instanceId: number): Promise<ApiResponse<JenkinsJob[]>> => {
    return request.get(`/jenkins-instances/${instanceId}/jobs`)
  },

  getJobBuilds: (instanceId: number, jobName: string, limit = 20): Promise<ApiResponse<JenkinsBuild[]>> => {
    return request.get(`/jenkins-instances/${instanceId}/jobs/${encodeURIComponent(jobName)}/builds`, { params: { limit } })
  }
}

// Jenkins 构建相关
export const jenkinsApi = {
  triggerBuild: (data: { job_name: string; gitlab_source_branch?: string; change_type?: string }): Promise<ApiResponse> => {
    return request.post('/jenkins/build', data)
  },

  getBuildStatus: (jobName: string, buildNumber: number): Promise<ApiResponse> => {
    return request.get(`/jenkins/build/${jobName}/${buildNumber}/status`)
  },

  getBuildConsole: (jobName: string, buildNumber: number): Promise<ApiResponse> => {
    return request.get(`/jenkins/build/${jobName}/${buildNumber}/console`)
  },

  getJobInfo: (jobName: string): Promise<ApiResponse> => {
    return request.get(`/jenkins/job/${jobName}`)
  }
}

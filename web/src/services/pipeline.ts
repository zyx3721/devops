import request from '@/services/api'

export const pipelineApi = {
  // 流水线管理
  list(params?: { name?: string; project_id?: number; status?: string; page?: number; page_size?: number }) {
    return request.get('/pipelines', { params })
  },

  get(id: number) {
    return request.get(`/pipelines/${id}`)
  },

  create(data: {
    name: string
    description?: string
    project_id?: number
    stages?: any[]
    variables?: any[]
    trigger_config?: any
  }) {
    return request.post('/pipelines', data)
  },

  update(id: number, data: {
    name: string
    description?: string
    project_id?: number
    stages?: any[]
    variables?: any[]
    trigger_config?: any
  }) {
    return request.put(`/pipelines/${id}`, data)
  },

  delete(id: number) {
    return request.delete(`/pipelines/${id}`)
  },

  toggle(id: number) {
    return request.post(`/pipelines/${id}/toggle`)
  },

  // 流水线执行
  run(id: number, data?: { parameters?: Record<string, string>; branch?: string }) {
    return request.post(`/pipelines/${id}/run`, data)
  },

  cancelRun(id: number) {
    return request.post(`/pipelines/runs/${id}/cancel`)
  },

  retryRun(id: number, fromStage?: string) {
    return request.post(`/pipelines/runs/${id}/retry`, null, { params: { from_stage: fromStage } })
  },

  // 执行历史
  listRuns(params?: { pipeline_id?: number; status?: string; page?: number; page_size?: number }) {
    return request.get('/pipelines/runs', { params })
  },

  getRun(id: number) {
    return request.get(`/pipelines/runs/${id}`)
  },

  getStepLogs(stepRunId: number) {
    return request.get(`/pipelines/steps/${stepRunId}/logs`)
  },

  // 模板
  getTemplates(params?: { category?: string }) {
    return request.get('/pipelines/templates', { params })
  },

  getTemplate(id: number) {
    return request.get(`/pipelines/templates/${id}`)
  },

  createFromTemplate(data: { template_id: number; name: string; description?: string; project_id?: number }) {
    return request.post('/pipelines/from-template', data)
  },

  // 凭证
  getCredentials() {
    return request.get('/pipelines/credentials')
  },

  createCredential(data: { name: string; type: string; description?: string; data: string }) {
    return request.post('/pipelines/credentials', data)
  },

  updateCredential(id: number, data: { name: string; type: string; description?: string; data?: string }) {
    return request.put(`/pipelines/credentials/${id}`, data)
  },

  deleteCredential(id: number) {
    return request.delete(`/pipelines/credentials/${id}`)
  },

  // 变量
  getVariables(params?: { scope?: string; pipeline_id?: number }) {
    return request.get('/pipelines/variables', { params })
  },

  createVariable(data: { name: string; value: string; is_secret?: boolean; scope?: string; pipeline_id?: number }) {
    return request.post('/pipelines/variables', data)
  },

  updateVariable(id: number, data: { name: string; value: string; is_secret?: boolean; scope?: string; pipeline_id?: number }) {
    return request.put(`/pipelines/variables/${id}`, data)
  },

  deleteVariable(id: number) {
    return request.delete(`/pipelines/variables/${id}`)
  }
}

// Git 仓库 API
export const gitRepoApi = {
  // 仓库管理
  list(params?: { name?: string; provider?: string; page?: number; page_size?: number }) {
    return request.get('/git/repos', { params })
  },

  get(id: number) {
    return request.get(`/git/repos/${id}`)
  },

  create(data: {
    name: string
    url: string
    provider?: string
    default_branch?: string
    credential_id?: number
    description?: string
  }) {
    return request.post('/git/repos', data)
  },

  update(id: number, data: {
    name: string
    url: string
    provider?: string
    default_branch?: string
    credential_id?: number
    description?: string
  }) {
    return request.put(`/git/repos/${id}`, data)
  },

  delete(id: number) {
    return request.delete(`/git/repos/${id}`)
  },

  // 仓库操作
  testConnection(data: { url: string; credential_id?: number }) {
    return request.post('/git/repos/test', data)
  },

  getBranches(id: number) {
    return request.get(`/git/repos/${id}/branches`)
  },

  getTags(id: number) {
    return request.get(`/git/repos/${id}/tags`)
  },

  regenerateSecret(id: number) {
    return request.post(`/git/repos/${id}/regenerate-secret`)
  }
}

// 扩展 pipelineApi
Object.assign(pipelineApi, {
  // 获取执行详情
  getRunDetail(id: number) {
    return request.get(`/pipelines/runs/${id}`)
  },

  // 获取模板列表
  listTemplates(params?: { category?: string }) {
    return request.get('/pipelines/templates', { params })
  }
})

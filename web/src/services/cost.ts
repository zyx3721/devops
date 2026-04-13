import request from './api'

// 成本管理 API
export const costApi = {
  // 获取成本概览
  getOverview(clusterId?: number, days?: number) {
    return request.get('/cost/overview', {
      params: { cluster_id: clusterId, days }
    })
  },

  // 获取成本趋势
  getTrend(params: { cluster_id?: number; days?: number; dimension?: string }) {
    return request.get('/cost/trend', { params })
  },

  // 获取成本分布
  getDistribution(params: { cluster_id?: number; dimension?: string; start_time?: string; end_time?: string; top_n?: number }) {
    return request.get('/cost/distribution', { params })
  },

  // 获取资源利用率
  getResourceUsage(params: { cluster_id?: number; namespace?: string; top_n?: number }) {
    return request.get('/cost/usage', { params })
  },

  // 获取成本预测
  getForecast(clusterId?: number, days?: number) {
    return request.get('/cost/forecast', {
      params: { cluster_id: clusterId, days }
    })
  },

  // 获取资源浪费检测
  getWasteDetection(clusterId?: number, days?: number) {
    return request.get('/cost/waste', {
      params: { cluster_id: clusterId, days }
    })
  },

  // 获取成本健康评分
  getHealthScore(clusterId?: number) {
    return request.get('/cost/health-score', {
      params: { cluster_id: clusterId }
    })
  },

  // 获取优化建议
  getSuggestions(clusterId?: number, status?: string) {
    return request.get('/cost/suggestions', {
      params: { cluster_id: clusterId, status }
    })
  },

  // 应用优化建议
  applySuggestion(id: number) {
    return request.post(`/cost/suggestions/${id}/apply`)
  },

  // 忽略优化建议
  ignoreSuggestion(id: number) {
    return request.post(`/cost/suggestions/${id}/ignore`)
  },

  // 获取预算列表
  getBudgets(clusterId?: number) {
    return request.get('/cost/budgets', {
      params: { cluster_id: clusterId }
    })
  },

  // 保存预算
  saveBudget(data: { cluster_id: number; namespace?: string; monthly_budget: number; alert_threshold?: number }) {
    return request.post('/cost/budgets', data)
  },

  // 获取成本配置
  getConfig(clusterId: number) {
    return request.get('/cost/config', {
      params: { cluster_id: clusterId }
    })
  },

  // 保存成本配置
  saveConfig(clusterId: number, data: { cpu_price_per_core: number; memory_price_per_gb: number; storage_price_per_gb: number; currency?: string }) {
    return request.post('/cost/config', data, {
      params: { cluster_id: clusterId }
    })
  },

  // 成本对比分析
  getComparison(params: {
    cluster_id?: number
    period1_start: string
    period1_end: string
    period2_start: string
    period2_end: string
  }) {
    return request.get('/cost/comparison', { params })
  },

  // 获取告警列表
  getAlerts(clusterId?: number, status?: string) {
    return request.get('/cost/alerts', {
      params: { cluster_id: clusterId, status }
    })
  },

  // 确认告警
  acknowledgeAlert(id: number) {
    return request.post(`/cost/alerts/${id}/acknowledge`)
  },

  // 导出报表
  exportReport(params: {
    cluster_id?: number
    start_time: string
    end_time: string
    report_type?: string
  }) {
    return request.get('/cost/export', {
      params,
      responseType: 'blob'
    })
  }
}

export default costApi


// 应用维度成本
export const getAppCost = (params: { cluster_id?: number; start_time?: string; end_time?: string; top_n?: number }) => {
  return request.get('/cost/app', { params })
}

// 团队维度成本
export const getTeamCost = (params: { cluster_id?: number; start_time?: string; end_time?: string }) => {
  return request.get('/cost/team', { params })
}

// 节点成本
export const getNodeCost = (clusterId?: number) => {
  return request.get('/cost/node', { params: { cluster_id: clusterId } })
}

// PVC存储成本
export const getPVCCost = (params: { cluster_id?: number; namespace?: string }) => {
  return request.get('/cost/pvc', { params })
}

// 环境维度成本
export const getEnvCost = (params: { cluster_id?: number; start_time?: string; end_time?: string }) => {
  return request.get('/cost/env', { params })
}

// 成本分摊报表
export const getCostAllocation = (params: {
  cluster_id?: number
  start_time: string
  end_time: string
  group_by?: string
  include_shared?: boolean
}) => {
  return request.get('/cost/allocation', { params })
}

// 扩展 costApi 对象
Object.assign(costApi, {
  getAppCost,
  getTeamCost,
  getNodeCost,
  getPVCCost,
  getEnvCost,
  getCostAllocation
})

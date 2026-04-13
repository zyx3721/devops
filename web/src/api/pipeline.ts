import request from '@/utils/request'
import type {
  BuildCache,
  BuildCacheStats,
  ResourceQuota,
  QuotaCheckResult,
  ArtifactVersion,
  ArtifactScanResult,
  BuildStats,
  ParallelConfig,
  PipelineTemplate,
  PipelineConfig,
  PaginationParams,
  PaginationResult,
} from '@/types/pipeline'

// ==================== 构建缓存 API ====================

/**
 * 获取构建缓存列表
 */
export const getBuildCacheList = (params: PaginationParams & {
  pipeline_id?: string
  key?: string
}) => {
  return request.get<PaginationResult<BuildCache>>('/build/cache', { params })
}

/**
 * 获取缓存统计信息
 */
export const getBuildCacheStats = () => {
  return request.get<BuildCacheStats>('/build/cache/stats')
}

/**
 * 删除单个缓存
 */
export const deleteBuildCache = (id: number) => {
  return request.delete(`/build/cache/${id}`)
}

/**
 * 批量删除缓存
 */
export const batchDeleteBuildCache = (ids: number[]) => {
  return request.post('/build/cache/batch-delete', { ids })
}

/**
 * 清理过期缓存
 */
export const cleanExpiredCache = () => {
  return request.post('/build/cache/clean-expired')
}

/**
 * 清理流水线缓存
 */
export const cleanPipelineCache = (pipelineId: number) => {
  return request.post(`/build/cache/clean-pipeline/${pipelineId}`)
}

// ==================== 资源配额 API ====================

/**
 * 获取资源配额列表
 */
export const getResourceQuotaList = (params?: PaginationParams) => {
  return request.get<PaginationResult<ResourceQuota>>('/build/quota', { params })
}

/**
 * 创建资源配额
 */
export const createResourceQuota = (data: Partial<ResourceQuota>) => {
  return request.post<ResourceQuota>('/build/quota', data)
}

/**
 * 更新资源配额
 */
export const updateResourceQuota = (id: number, data: Partial<ResourceQuota>) => {
  return request.put<ResourceQuota>(`/build/quota/${id}`, data)
}

/**
 * 删除资源配额
 */
export const deleteResourceQuota = (id: number) => {
  return request.delete(`/build/quota/${id}`)
}

/**
 * 设置默认配额
 */
export const setDefaultQuota = (id: number) => {
  return request.post(`/build/quota/${id}/set-default`)
}

/**
 * 检查流水线配额
 */
export const checkPipelineQuota = (pipelineId: number) => {
  return request.get<QuotaCheckResult>(`/build/quota/check/${pipelineId}`)
}

// ==================== 制品版本 API ====================

/**
 * 获取制品版本列表
 */
export const getArtifactVersions = (artifactId: number, params?: PaginationParams) => {
  return request.get<PaginationResult<ArtifactVersion>>(`/artifacts/${artifactId}/versions`, { params })
}

/**
 * 获取版本详情
 */
export const getArtifactVersionDetail = (versionId: number) => {
  return request.get<ArtifactVersion>(`/artifacts/versions/${versionId}`)
}

/**
 * 对比两个版本
 */
export const compareArtifactVersions = (versionId1: number, versionId2: number) => {
  return request.get(`/artifacts/versions/compare`, {
    params: { version1: versionId1, version2: versionId2 }
  })
}

/**
 * 删除制品版本
 */
export const deleteArtifactVersion = (versionId: number) => {
  return request.delete(`/artifacts/versions/${versionId}`)
}

// ==================== 制品扫描 API ====================

/**
 * 获取扫描结果
 */
export const getArtifactScanResult = (versionId: number) => {
  return request.get<ArtifactScanResult>(`/artifacts/versions/${versionId}/scan`)
}

/**
 * 触发扫描
 */
export const triggerArtifactScan = (versionId: number) => {
  return request.post(`/artifacts/versions/${versionId}/scan`)
}

/**
 * 获取漏洞列表
 */
export const getVulnerabilities = (versionId: number, params?: {
  severity?: string
  page?: number
  page_size?: number
}) => {
  return request.get(`/artifacts/versions/${versionId}/vulnerabilities`, { params })
}

// ==================== 并行构建配置 API ====================

/**
 * 获取并行构建配置
 */
export const getParallelConfig = (pipelineId: number) => {
  return request.get<ParallelConfig>(`/pipeline/${pipelineId}/parallel`)
}

/**
 * 保存并行构建配置
 */
export const saveParallelConfig = (pipelineId: number, data: ParallelConfig) => {
  return request.post(`/pipeline/${pipelineId}/parallel`, data)
}

// ==================== 构建统计 API ====================

/**
 * 获取构建统计数据
 */
export const getBuildStats = (params: {
  start_date?: string
  end_date?: string
  pipeline_id?: number
}) => {
  return request.get<BuildStats>('/build/stats', { params })
}

/**
 * 导出统计报告
 */
export const exportBuildStats = (params: {
  start_date?: string
  end_date?: string
  format: 'excel' | 'pdf'
}) => {
  return request.get('/build/stats/export', {
    params,
    responseType: 'blob'
  })
}

// ==================== 流水线模板 API ====================

/**
 * 获取模板列表
 */
export const getTemplateList = (params?: PaginationParams & {
  category?: string
  keyword?: string
  order_by?: string
  tags?: string
  favorites_only?: boolean
}) => {
  return request.get<PaginationResult<PipelineTemplate>>('/pipeline/templates', { params })
}

/**
 * 获取模板详情
 */
export const getTemplateDetail = (id: number) => {
  return request.get<PipelineTemplate>(`/pipeline/templates/${id}`)
}

/**
 * 创建模板
 */
export const createTemplate = (data: Partial<PipelineTemplate>) => {
  return request.post<PipelineTemplate>('/pipeline/templates', data)
}

/**
 * 更新模板
 */
export const updateTemplate = (id: number, data: Partial<PipelineTemplate>) => {
  return request.put<PipelineTemplate>(`/pipeline/templates/${id}`, data)
}

/**
 * 删除模板
 */
export const deleteTemplate = (id: number) => {
  return request.delete(`/pipeline/templates/${id}`)
}

/**
 * 使用模板
 */
export const useTemplate = (id: number) => {
  return request.post(`/pipeline/templates/${id}/use`)
}

/**
 * 评分模板
 */
export const rateTemplate = (id: number, rating: number) => {
  return request.post(`/pipeline/templates/${id}/rate`, { rating })
}

/**
 * 收藏模板
 */
export const favoriteTemplate = (id: number) => {
  return request.post(`/pipeline/templates/${id}/favorite`)
}

/**
 * 取消收藏模板
 */
export const unfavoriteTemplate = (id: number) => {
  return request.delete(`/pipeline/templates/${id}/favorite`)
}

/**
 * 获取收藏列表
 */
export const getFavoriteTemplates = () => {
  return request.get('/pipeline/templates/favorites')
}

/**
 * 获取模板分类
 */
export const getTemplateCategories = () => {
  return request.get<string[]>('/pipeline/templates/categories')
}

/**
 * 获取模板标签
 */
export const getTemplateTags = () => {
  return request.get<string[]>('/pipeline/templates/tags')
}

// ==================== 流水线设计器 API ====================

/**
 * 保存流水线配置
 */
export const savePipelineConfig = (pipelineId: number, config: PipelineConfig) => {
  return request.post(`/pipeline/${pipelineId}/config`, config)
}

/**
 * 获取流水线配置
 */
export const getPipelineConfig = (pipelineId: number) => {
  return request.get<PipelineConfig>(`/pipeline/${pipelineId}/config`)
}

/**
 * 验证流水线配置
 */
export const validatePipelineConfig = (config: PipelineConfig) => {
  return request.post('/pipeline/config/validate', config)
}

// ==================== 流水线列表 API ====================

/**
 * 获取流水线列表
 */
export const getPipelineList = (params?: PaginationParams) => {
  return request.get('/pipeline/list', { params })
}

/**
 * 获取流水线详情
 */
export const getPipelineDetail = (id: number) => {
  return request.get(`/pipeline/${id}`)
}

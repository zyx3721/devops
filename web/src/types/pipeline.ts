// ==================== 通用类型 ====================

/**
 * 分页参数
 */
export interface PaginationParams {
  page?: number
  page_size?: number
}

/**
 * 分页结果
 */
export interface PaginationResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

/**
 * API 响应
 */
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

// ==================== 构建缓存相关类型 ====================

/**
 * 构建缓存
 */
export interface BuildCache {
  id: number
  pipeline_id: number
  pipeline_name: string
  key: string
  size: number
  hit_count: number
  created_at: string
  last_used_at: string
  expires_at: string
}

/**
 * 缓存统计
 */
export interface BuildCacheStats {
  total_count: number
  total_size: number
  hit_rate: number
  cache_usage_trend: CacheUsageTrend[]
  hit_rate_trend: HitRateTrend[]
}

/**
 * 缓存使用趋势
 */
export interface CacheUsageTrend {
  date: string
  size: number
  count: number
}

/**
 * 命中率趋势
 */
export interface HitRateTrend {
  date: string
  hit_rate: number
  saved_time: number
}

// ==================== 资源配额相关类型 ====================

/**
 * 资源配额
 */
export interface ResourceQuota {
  id: number
  name: string
  description: string
  cpu_limit: number
  memory_limit: number
  storage_limit: number
  max_concurrent_builds: number
  is_default: boolean
  created_at: string
  updated_at: string
}

/**
 * 配额检查结果
 */
export interface QuotaCheckResult {
  can_build: boolean
  quota: ResourceQuota
  current_usage: ResourceUsage
  message?: string
}

/**
 * 资源使用情况
 */
export interface ResourceUsage {
  cpu_used: number
  memory_used: number
  storage_used: number
  concurrent_builds: number
}

// ==================== 制品版本相关类型 ====================

/**
 * 制品版本
 */
export interface ArtifactVersion {
  id: number
  artifact_id: number
  artifact_name: string
  version: string
  size: number
  checksum: string
  metadata: Record<string, any>
  scan_status: 'pending' | 'scanning' | 'completed' | 'failed'
  scan_result?: ArtifactScanResult
  created_at: string
  created_by: string
}

/**
 * 制品扫描结果
 */
export interface ArtifactScanResult {
  version_id: number
  scan_time: string
  vulnerabilities: VulnerabilitySummary
  licenses: LicenseSummary
  quality: QualitySummary
  vulnerability_list: Vulnerability[]
  license_list: License[]
  quality_issues: QualityIssue[]
}

/**
 * 漏洞摘要
 */
export interface VulnerabilitySummary {
  total: number
  critical: number
  high: number
  medium: number
  low: number
}

/**
 * 漏洞详情
 */
export interface Vulnerability {
  id: string
  cve_id: string
  severity: 'critical' | 'high' | 'medium' | 'low'
  package_name: string
  installed_version: string
  fixed_version?: string
  description: string
  cvss_score?: number
  references: string[]
  fix_available: boolean
}

/**
 * 许可证摘要
 */
export interface LicenseSummary {
  total: number
  compliant: number
  non_compliant: number
  unknown: number
}

/**
 * 许可证详情
 */
export interface License {
  package_name: string
  license_type: string
  is_compliant: boolean
  risk_level: 'high' | 'medium' | 'low'
}

/**
 * 质量摘要
 */
export interface QualitySummary {
  score: number
  issues: number
  code_smells: number
  bugs: number
  security_hotspots: number
}

/**
 * 质量问题
 */
export interface QualityIssue {
  type: 'code_smell' | 'bug' | 'security_hotspot'
  severity: 'critical' | 'major' | 'minor'
  message: string
  file: string
  line: number
}

// ==================== 并行构建相关类型 ====================

/**
 * 并行构建配置
 */
export interface ParallelConfig {
  enabled: boolean
  max_parallel: number
  fail_fast: boolean
  parallel_stages: string[]
  dependencies: StageDependency[]
}

/**
 * 阶段依赖
 */
export interface StageDependency {
  stage: string
  depends_on: string[]
}

// ==================== 构建统计相关类型 ====================

/**
 * 构建统计
 */
export interface BuildStats {
  total_builds: number
  successful_builds: number
  failed_builds: number
  average_duration: number
  resource_usage: ResourceUsageStats
  cache_stats: CacheStats
  concurrent_stats: ConcurrentStats
}

/**
 * 资源使用统计
 */
export interface ResourceUsageStats {
  cpu_trend: ResourceTrend[]
  memory_trend: ResourceTrend[]
  storage_trend: ResourceTrend[]
}

/**
 * 资源趋势
 */
export interface ResourceTrend {
  date: string
  value: number
  unit: string
}

/**
 * 缓存统计
 */
export interface CacheStats {
  hit_rate: number
  saved_time: number
  cache_size: number
}

/**
 * 并发统计
 */
export interface ConcurrentStats {
  max_concurrent: number
  average_concurrent: number
  queue_time_avg: number
}

// ==================== 流水线模板相关类型 ====================

/**
 * 流水线模板
 */
export interface PipelineTemplate {
  id: number
  name: string
  slug: string
  description: string
  category: string
  tags: string[]
  version: string
  config_json: PipelineConfig
  is_public: boolean
  is_official: boolean
  usage_count: number
  rating: number
  rating_count: number
  created_at: string
  updated_at: string
  created_by: string
}

/**
 * 流水线配置
 */
export interface PipelineConfig {
  stages: PipelineStage[]
  variables?: Record<string, any>
  triggers?: PipelineTrigger[]
}

/**
 * 流水线阶段
 */
export interface PipelineStage {
  name: string
  description?: string
  steps: PipelineStep[]
  condition?: string
  parallel?: boolean
}

/**
 * 流水线步骤
 */
export interface PipelineStep {
  name: string
  type: string
  config: Record<string, any>
  timeout?: number
  continue_on_error?: boolean
}

/**
 * 流水线触发器
 */
export interface PipelineTrigger {
  type: 'webhook' | 'schedule' | 'manual'
  config: Record<string, any>
}

// ==================== 流水线设计器相关类型 ====================

/**
 * 流水线节点
 */
export interface PipelineNode {
  id: string
  type: string
  name: string
  config: Record<string, any>
  position: {
    x: number
    y: number
  }
}

/**
 * 流水线边
 */
export interface PipelineEdge {
  id: string
  source: string
  target: string
  type: 'default' | 'conditional'
  condition?: string
}

/**
 * 流水线图
 */
export interface PipelineGraph {
  nodes: PipelineNode[]
  edges: PipelineEdge[]
}

// ==================== 组件模板相关类型 ====================

/**
 * 组件模板
 */
export interface ComponentTemplate {
  type: string
  label: string
  icon: any
  config: Record<string, any>
}

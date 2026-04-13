// API 响应类型
export interface ApiResponse<T = any> {
  code: number
  message: string
  data?: T
}

// 分页响应
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

// 用户类型
export interface User {
  id: number
  username: string
  email: string
  phone?: string
  role: string
  status: string
  last_login_at?: string
  created_at: string
  updated_at: string
}

// Jenkins 实例
export interface JenkinsInstance {
  id: number
  name: string
  url: string
  username: string
  description: string
  status: string
  is_default: boolean
  created_at: string
  updated_at: string
}

// K8s 集群
export interface K8sCluster {
  id: number
  name: string
  namespace: string
  registry: string
  repository: string
  description: string
  status: string
  is_default: boolean
  check_timeout: number
  created_at: string
  updated_at: string
}

// 任务
export interface Task {
  id: number
  name: string
  description: string
  status: string
  created_by: number
  start_time: string
  end_time: string
  jenkins_job: string
  parameters: string
  created_at: string
  updated_at: string
}

// 导出 Pipeline 相关类型
export * from './pipeline'

// 导出 AI 相关类型
export * from './ai'

// 导出 Menu 相关类型
export * from './menu'

import request from './api'
import type { ApiResponse, K8sCluster, PaginatedResponse } from '../types'

export interface K8sClusterRequest {
  page?: number
  page_size?: number
  keyword?: string
  status?: string
}

export interface CreateK8sClusterRequest {
  name: string
  kubeconfig: string
  description?: string
  status: string
  is_default?: boolean
}

export interface UpdateK8sClusterRequest extends CreateK8sClusterRequest {}

export const k8sClusterApi = {
  list: (page = 1, pageSize = 100): Promise<ApiResponse<{ items: K8sCluster[]; total: number }>> => {
    return request.get('/k8s-clusters', { params: { page, page_size: pageSize } })
  },

  getClusters: (params: K8sClusterRequest = {}): Promise<ApiResponse<PaginatedResponse<K8sCluster>>> => {
    return request.get('/k8s-clusters', { params })
  },

  getCluster: (id: number): Promise<ApiResponse<K8sCluster>> => {
    return request.get(`/k8s-clusters/${id}`)
  },

  getDefaultCluster: (): Promise<ApiResponse<K8sCluster>> => {
    return request.get('/k8s-clusters/default')
  },

  createCluster: (data: CreateK8sClusterRequest): Promise<ApiResponse<K8sCluster>> => {
    return request.post('/k8s-clusters', data)
  },

  updateCluster: (id: number, data: UpdateK8sClusterRequest): Promise<ApiResponse<K8sCluster>> => {
    return request.put(`/k8s-clusters/${id}`, data)
  },

  setDefaultCluster: (id: number): Promise<ApiResponse> => {
    return request.put(`/k8s-clusters/${id}/default`)
  },

  deleteCluster: (id: number): Promise<ApiResponse> => {
    return request.delete(`/k8s-clusters/${id}`)
  },

  testConnection: (id: number): Promise<ApiResponse<ConnectionTestResult>> => {
    return request.post(`/k8s-clusters/${id}/test-connection`)
  },

  getFeishuApps: (id: number): Promise<ApiResponse<FeishuAppSimple[]>> => {
    return request.get(`/k8s-clusters/${id}/feishu-apps`)
  },

  bindFeishuApps: (id: number, appIds: number[]): Promise<ApiResponse> => {
    return request.put(`/k8s-clusters/${id}/feishu-apps`, { app_ids: appIds })
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


// K8s 资源相关类型
export interface K8sNamespace {
  name: string
  status: string
  created_at: string
}

export interface K8sDeployment {
  name: string
  namespace: string
  replicas: number
  ready: number
  available: number
  images: string[]
  created_at: string
}

export interface K8sStatefulSet {
  name: string
  namespace: string
  replicas: number
  ready: number
  created_at: string
}

export interface K8sDaemonSet {
  name: string
  namespace: string
  desired: number
  ready: number
  created_at: string
}

export interface K8sJob {
  name: string
  namespace: string
  completions: number
  succeeded: number
  failed: number
  created_at: string
}

export interface K8sCronJob {
  name: string
  namespace: string
  schedule: string
  suspend: boolean
  last_schedule: string
  created_at: string
}

export interface K8sContainer {
  name: string
  image: string
}

export interface K8sPod {
  name: string
  namespace: string
  status: string
  node: string
  ip: string
  restarts: number
  containers: K8sContainer[]
  created_at: string
}

export interface K8sServicePort {
  name: string
  port: number
  target_port: string
  protocol: string
  node_port?: number
}

export interface K8sService {
  name: string
  namespace: string
  type: string
  cluster_ip: string
  ports: K8sServicePort[]
  created_at: string
}

export interface K8sIngress {
  name: string
  namespace: string
  ingress_class: string
  hosts: string[]
  created_at: string
}

export interface K8sConfigMap {
  name: string
  namespace: string
  keys: string[]
  created_at: string
}

export interface K8sSecret {
  name: string
  namespace: string
  type: string
  keys: string[]
  created_at: string
}

export interface K8sPVC {
  name: string
  namespace: string
  status: string
  capacity: string
  storage_class: string
  access_modes: string
  created_at: string
}

// K8s 资源 API
export const k8sResourceApi = {
  getNamespaces: (clusterId: number): Promise<ApiResponse<K8sNamespace[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/namespaces`)
  },

  createNamespace: (clusterId: number, name: string, labels?: Record<string, string>): Promise<ApiResponse> => {
    return request.post(`/k8s-clusters/${clusterId}/resources/namespaces`, { name, labels })
  },

  deleteNamespace: (clusterId: number, name: string): Promise<ApiResponse> => {
    return request.delete(`/k8s-clusters/${clusterId}/resources/namespaces/${name}`)
  },

  getDeployments: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sDeployment[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/deployments`, { params: { namespace } })
  },

  getStatefulSets: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sStatefulSet[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/statefulsets`, { params: { namespace } })
  },

  getDaemonSets: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sDaemonSet[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/daemonsets`, { params: { namespace } })
  },

  getJobs: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sJob[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/jobs`, { params: { namespace } })
  },

  getCronJobs: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sCronJob[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/cronjobs`, { params: { namespace } })
  },

  getPods: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sPod[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/pods`, { params: { namespace } })
  },

  getServices: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sService[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/services`, { params: { namespace } })
  },

  getIngresses: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sIngress[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/ingresses`, { params: { namespace } })
  },

  getConfigMaps: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sConfigMap[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/configmaps`, { params: { namespace } })
  },

  getSecrets: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sSecret[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/secrets`, { params: { namespace } })
  },

  getPVCs: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sPVC[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/pvcs`, { params: { namespace } })
  },

  getPodLogs: (clusterId: number, namespace: string, podName: string, container?: string, tail?: number): Promise<ApiResponse<string>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/pods/${podName}/logs`, { params: { namespace, container, tail } })
  },

  deletePod: (clusterId: number, namespace: string, podName: string): Promise<ApiResponse> => {
    return request.delete(`/k8s-clusters/${clusterId}/resources/pods/${podName}`, { params: { namespace } })
  },

  restartDeployment: (clusterId: number, namespace: string, deploymentName: string): Promise<ApiResponse> => {
    return request.post(`/k8s-clusters/${clusterId}/resources/deployments/${deploymentName}/restart`, null, { params: { namespace } })
  },

  scaleDeployment: (clusterId: number, namespace: string, deploymentName: string, replicas: number): Promise<ApiResponse> => {
    return request.post(`/k8s-clusters/${clusterId}/resources/deployments/${deploymentName}/scale`, { replicas }, { params: { namespace } })
  },

  // CRUD 操作
  getResourceYAML: (clusterId: number, resourceType: string, namespace: string, name: string): Promise<ApiResponse<string>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/yaml/${resourceType}/${name}`, { params: { namespace } })
  },

  applyResource: (clusterId: number, yaml: string): Promise<ApiResponse> => {
    return request.post(`/k8s-clusters/${clusterId}/resources/apply`, { yaml })
  },

  deleteResource: (clusterId: number, resourceType: string, namespace: string, name: string): Promise<ApiResponse> => {
    return request.delete(`/k8s-clusters/${clusterId}/resources/${resourceType}/${name}`, { params: { namespace } })
  },

  // 资源详情
  getResourceDetail: (clusterId: number, resourceType: string, namespace: string, name: string): Promise<ApiResponse<any>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/detail/${resourceType}/${name}`, { params: { namespace } })
  },

  // 关联资源
  getRelatedPods: (clusterId: number, ownerType: string, namespace: string, ownerName: string): Promise<ApiResponse<any[]>> => {
    // ownerType 需要转换为复数形式
    const typeMap: Record<string, string> = {
      deployment: 'deployments',
      statefulset: 'statefulsets',
      daemonset: 'daemonsets'
    }
    const resourcePath = typeMap[ownerType] || ownerType + 's'
    return request.get(`/k8s-clusters/${clusterId}/resources/${resourcePath}/${ownerName}/pods`, { params: { namespace } })
  },

  getServicePods: (clusterId: number, namespace: string, serviceName: string): Promise<ApiResponse<any[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/services/${serviceName}/pods`, { params: { namespace } })
  }
}

// 新增类型定义

export interface K8sNodeTaint {
  key: string
  value: string
  effect: string
}

export interface K8sNode {
  name: string
  status: string
  roles: string[]
  internal_ip: string
  hostname: string
  cpu_capacity: string
  memory_capacity: string
  cpu_allocatable: string
  memory_allocatable: string
  pod_capacity: string
  schedulable: boolean
  taints: K8sNodeTaint[]
  labels: Record<string, string>
  kubelet_version: string
  container_runtime: string
  os_image: string
  kernel_version: string
  architecture: string
  created_at: string
}

export interface K8sNodeCondition {
  type: string
  status: string
  reason: string
  message: string
}

export interface K8sNodePod {
  name: string
  namespace: string
  status: string
  ip: string
}

export interface K8sNodeDetail {
  name: string
  labels: Record<string, string>
  annotations: Record<string, string>
  taints: K8sNodeTaint[]
  conditions: K8sNodeCondition[]
  pods: K8sNodePod[]
  pod_count: number
  schedulable: boolean
  cpu_capacity: string
  memory_capacity: string
  cpu_allocatable: string
  memory_allocatable: string
  created_at: string
}

export interface K8sPV {
  name: string
  status: string
  capacity: string
  access_modes: string
  reclaim_policy: string
  storage_class: string
  claim_ref: string
  volume_mode: string
  created_at: string
}

export interface K8sStorageClass {
  name: string
  provisioner: string
  reclaim_policy: string
  volume_binding_mode: string
  allow_expansion: boolean
  is_default: boolean
  created_at: string
}

export interface K8sEvent {
  name: string
  namespace: string
  type: string
  reason: string
  message: string
  object: string
  count: number
  last_timestamp: string
}

export interface K8sEndpoint {
  name: string
  namespace: string
  addresses: string[]
  ports: string[]
  created_at: string
}

export interface K8sServiceAccount {
  name: string
  namespace: string
  secrets: string[]
  created_at: string
}

// 扩展 k8sResourceApi
export const k8sNodeApi = {
  getNodes: (clusterId: number): Promise<ApiResponse<K8sNode[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/nodes`)
  },

  getNodeDetail: (clusterId: number, nodeName: string): Promise<ApiResponse<K8sNodeDetail>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/nodes/${nodeName}`)
  },

  cordonNode: (clusterId: number, nodeName: string): Promise<ApiResponse> => {
    return request.post(`/k8s-clusters/${clusterId}/resources/nodes/${nodeName}/cordon`)
  },

  uncordonNode: (clusterId: number, nodeName: string): Promise<ApiResponse> => {
    return request.post(`/k8s-clusters/${clusterId}/resources/nodes/${nodeName}/uncordon`)
  },

  addNodeTaint: (clusterId: number, nodeName: string, taint: { key: string; value: string; effect: string }): Promise<ApiResponse> => {
    return request.post(`/k8s-clusters/${clusterId}/resources/nodes/${nodeName}/taints`, taint)
  },

  removeNodeTaint: (clusterId: number, nodeName: string, key: string, effect: string): Promise<ApiResponse> => {
    return request.delete(`/k8s-clusters/${clusterId}/resources/nodes/${nodeName}/taints`, { params: { key, effect } })
  },

  updateNodeLabels: (clusterId: number, nodeName: string, labels: Record<string, string>): Promise<ApiResponse> => {
    return request.put(`/k8s-clusters/${clusterId}/resources/nodes/${nodeName}/labels`, { labels })
  },

  getJoinCommand: (clusterId: number): Promise<ApiResponse<string>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/nodes/join-command`)
  }
}

export const k8sStorageApi = {
  getPVs: (clusterId: number): Promise<ApiResponse<K8sPV[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/pvs`)
  },

  getStorageClasses: (clusterId: number): Promise<ApiResponse<K8sStorageClass[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/storageclasses`)
  }
}

export const k8sEventApi = {
  getEvents: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sEvent[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/events`, { params: { namespace } })
  },

  getResourceEvents: (clusterId: number, resourceType: string, namespace: string, name: string): Promise<ApiResponse<K8sEvent[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/events/resource`, { params: { resource_type: resourceType, namespace, name } })
  }
}

export const k8sConfigApi = {
  getEndpoints: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sEndpoint[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/endpoints`, { params: { namespace } })
  },

  getServiceAccounts: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sServiceAccount[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/serviceaccounts`, { params: { namespace } })
  }
}


// ==================== K8s 运维增强 API ====================

// Pod 详情
export interface K8sPodDetail {
  name: string
  namespace: string
  status: string
  ready: string
  restarts: number
  age: string
  ip: string
  node: string
  containers: K8sContainerDetail[]
  labels: Record<string, string>
  created_at: string
}

export interface K8sContainerDetail {
  name: string
  image: string
  ready: boolean
  state: string
  restart_count: number
}

// Deployment 详情
export interface K8sDeploymentDetail {
  name: string
  namespace: string
  ready: string
  up_to_date: number
  available: number
  age: string
  images: string[]
  replicas: number
  labels: Record<string, string>
  annotations: Record<string, string>
  strategy: string
  selector: Record<string, string>
  conditions: K8sDeploymentCondition[]
  created_at: string
}

export interface K8sDeploymentCondition {
  type: string
  status: string
  reason: string
  message: string
}

// 版本历史
export interface K8sRevisionInfo {
  revision: number
  image: string
  created_at: string
  change_cause: string
}

// 更新进度
export interface K8sUpdateProgress {
  replicas: number
  updated_replicas: number
  ready_replicas: number
  available_replicas: number
  unavailable: number
  status: string
}

// K8s Pod 增强 API
export const k8sPodApi = {
  // 获取 Pod 列表
  list: (clusterId: number, namespace: string, labelSelector?: string): Promise<ApiResponse<K8sPodDetail[]>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/pods`, { params: { label_selector: labelSelector } })
  },

  // 获取 Pod 详情
  get: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<K8sPodDetail>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/pods/${name}`)
  },

  // 删除 Pod
  delete: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<void>> => {
    return request.delete(`/k8s/clusters/${clusterId}/namespaces/${namespace}/pods/${name}`)
  },

  // 获取容器列表
  getContainers: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<K8sContainer[]>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/pods/${name}/containers`)
  },

  // 获取日志
  getLogs: (clusterId: number, namespace: string, name: string, container?: string, tailLines?: number, timestamps?: boolean): Promise<ApiResponse<{ logs: string }>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/pods/${name}/logs`, {
      params: { container, tail_lines: tailLines, timestamps }
    })
  },

  // 下载日志
  downloadLogs: (clusterId: number, namespace: string, name: string, container?: string): string => {
    const baseUrl = '/app/api/v1'
    let url = `${baseUrl}/k8s/clusters/${clusterId}/namespaces/${namespace}/pods/${name}/logs/download`
    const params = new URLSearchParams()
    if (container) params.append('container', container)
    const token = localStorage.getItem('token')
    if (token) params.append('token', token)
    if (params.toString()) url += '?' + params.toString()
    return url
  },

  // WebSocket 日志流 URL
  getLogsStreamUrl: (clusterId: number, namespace: string, name: string, container?: string, tailLines?: number, timestamps?: boolean): string => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    let url = `${protocol}//${host}/app/api/v1/k8s/clusters/${clusterId}/namespaces/${namespace}/pods/${name}/logs/stream`
    const params = new URLSearchParams()
    if (container) params.append('container', container)
    if (tailLines) params.append('tail_lines', tailLines.toString())
    if (timestamps) params.append('timestamps', 'true')
    const token = localStorage.getItem('token')
    if (token) params.append('token', token)
    if (params.toString()) url += '?' + params.toString()
    return url
  },

  // WebSocket 终端 URL
  getTerminalUrl: (clusterId: number, namespace: string, name: string, container?: string, shell?: string): string => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    let url = `${protocol}//${host}/app/api/v1/k8s/clusters/${clusterId}/namespaces/${namespace}/pods/${name}/terminal`
    const params = new URLSearchParams()
    if (container) params.append('container', container)
    if (shell) params.append('shell', shell)
    const token = localStorage.getItem('token')
    if (token) params.append('token', token)
    if (params.toString()) url += '?' + params.toString()
    return url
  }
}

// K8s Deployment 增强 API
export const k8sDeploymentApi = {
  // 获取 Deployment 列表
  list: (clusterId: number, namespace: string): Promise<ApiResponse<K8sDeploymentDetail[]>> => {
    if (namespace) {
      return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/deployments`)
    } else {
      return request.get(`/k8s/clusters/${clusterId}/deployments`)
    }
  },

  // 获取 Deployment 详情
  get: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<K8sDeploymentDetail>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/deployments/${name}`)
  },

  // 更新镜像
  updateImage: (clusterId: number, namespace: string, name: string, container: string, image: string): Promise<ApiResponse<void>> => {
    return request.put(`/k8s/clusters/${clusterId}/namespaces/${namespace}/deployments/${name}/image`, { container, image })
  },

  // 扩缩容
  scale: (clusterId: number, namespace: string, name: string, replicas: number): Promise<ApiResponse<void>> => {
    return request.put(`/k8s/clusters/${clusterId}/namespaces/${namespace}/deployments/${name}/scale`, { replicas })
  },

  // 重启
  restart: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<void>> => {
    return request.post(`/k8s/clusters/${clusterId}/namespaces/${namespace}/deployments/${name}/restart`)
  },

  // 获取版本历史
  getRevisions: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<K8sRevisionInfo[]>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/deployments/${name}/revisions`)
  },

  // 回滚
  rollback: (clusterId: number, namespace: string, name: string, revision: number): Promise<ApiResponse<void>> => {
    return request.post(`/k8s/clusters/${clusterId}/namespaces/${namespace}/deployments/${name}/rollback`, { revision })
  },

  // 获取更新进度
  getProgress: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<K8sUpdateProgress>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/deployments/${name}/progress`)
  }
}


// ==================== HPA 管理 API ====================

export interface K8sHPA {
  name: string
  namespace: string
  target_kind: string
  target_name: string
  min_replicas: number
  max_replicas: number
  current_replicas: number
  desired_replicas: number
  metrics: string[]
  created_at: string
}

export interface CreateHPARequest {
  name: string
  namespace: string
  target_kind: 'Deployment' | 'StatefulSet'
  target_name: string
  min_replicas: number
  max_replicas: number
  cpu_target_percent?: number
  mem_target_percent?: number
}

export interface UpdateHPARequest {
  min_replicas: number
  max_replicas: number
  cpu_target_percent?: number
  mem_target_percent?: number
}

export const k8sHPAApi = {
  list: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sHPA[]>> => {
    return request.get(`/k8s/clusters/${clusterId}/hpa`, { params: { namespace } })
  },

  get: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<K8sHPA>> => {
    return request.get(`/k8s/clusters/${clusterId}/hpa/${namespace}/${name}`)
  },

  create: (clusterId: number, data: CreateHPARequest): Promise<ApiResponse<void>> => {
    return request.post(`/k8s/clusters/${clusterId}/hpa`, data)
  },

  update: (clusterId: number, namespace: string, name: string, data: UpdateHPARequest): Promise<ApiResponse<void>> => {
    return request.put(`/k8s/clusters/${clusterId}/hpa/${namespace}/${name}`, data)
  },

  delete: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<void>> => {
    return request.delete(`/k8s/clusters/${clusterId}/hpa/${namespace}/${name}`)
  }
}

// ==================== CronHPA 管理 API ====================

export interface CronSchedule {
  name: string
  cron: string
  replicas: number
  min_replicas?: number
  max_replicas?: number
}

export interface K8sCronHPA {
  name: string
  namespace: string
  target_kind: string
  target_name: string
  enabled: boolean
  schedules: CronSchedule[]
  created_at: string
}

export interface CreateCronHPARequest {
  name: string
  namespace: string
  target_kind: 'Deployment' | 'StatefulSet'
  target_name: string
  enabled: boolean
  schedules: CronSchedule[]
}

export interface UpdateCronHPARequest {
  enabled: boolean
  schedules: CronSchedule[]
}

export const k8sCronHPAApi = {
  list: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sCronHPA[]>> => {
    return request.get(`/k8s/clusters/${clusterId}/cron-hpa`, { params: { namespace } })
  },

  get: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<K8sCronHPA>> => {
    return request.get(`/k8s/clusters/${clusterId}/cron-hpa/${namespace}/${name}`)
  },

  create: (clusterId: number, data: CreateCronHPARequest): Promise<ApiResponse<void>> => {
    return request.post(`/k8s/clusters/${clusterId}/cron-hpa`, data)
  },

  update: (clusterId: number, namespace: string, name: string, data: UpdateCronHPARequest): Promise<ApiResponse<void>> => {
    return request.put(`/k8s/clusters/${clusterId}/cron-hpa/${namespace}/${name}`, data)
  },

  delete: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<void>> => {
    return request.delete(`/k8s/clusters/${clusterId}/cron-hpa/${namespace}/${name}`)
  }
}

// ==================== 资源配额 API ====================

export interface K8sResourceQuota {
  name: string
  namespace: string
  hard: Record<string, string>
  used: Record<string, string>
  created_at: string
}

export interface CreateResourceQuotaRequest {
  name: string
  namespace: string
  hard: Record<string, string>
}

export interface K8sLimitRange {
  name: string
  namespace: string
  limits: LimitRangeItem[]
  created_at: string
}

export interface LimitRangeItem {
  type: string
  default?: Record<string, string>
  default_request?: Record<string, string>
  max?: Record<string, string>
  min?: Record<string, string>
}

export const k8sQuotaApi = {
  listQuotas: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sResourceQuota[]>> => {
    return request.get(`/k8s/clusters/${clusterId}/quotas`, { params: { namespace } })
  },

  createQuota: (clusterId: number, data: CreateResourceQuotaRequest): Promise<ApiResponse<void>> => {
    return request.post(`/k8s/clusters/${clusterId}/quotas`, data)
  },

  deleteQuota: (clusterId: number, namespace: string, name: string): Promise<ApiResponse<void>> => {
    return request.delete(`/k8s/clusters/${clusterId}/quotas/${namespace}/${name}`)
  },

  listLimitRanges: (clusterId: number, namespace?: string): Promise<ApiResponse<K8sLimitRange[]>> => {
    return request.get(`/k8s/clusters/${clusterId}/limitranges`, { params: { namespace } })
  }
}

// ==================== 集群概览 API ====================

export interface ClusterOverview {
  cluster_id: number
  cluster_name: string
  status: string
  node_total: number
  node_ready: number
  cpu_capacity: string
  cpu_used: string
  memory_capacity: string
  memory_used: string
  pod_capacity: number
  pod_used: number
  deployment_total: number
  deployment_ready: number
  statefulset_total: number
  statefulset_ready: number
  daemonset_total: number
  daemonset_ready: number
}

export interface ClusterSummary {
  total_clusters: number
  healthy_clusters: number
  total_nodes: number
  total_pods: number
  total_deployments: number
}

export interface MultiClusterOverview {
  clusters: ClusterOverview[]
  summary: ClusterSummary
}

export const k8sOverviewApi = {
  getClusterOverview: (clusterId: number): Promise<ApiResponse<ClusterOverview>> => {
    return request.get(`/k8s/clusters/${clusterId}/overview`)
  },

  getMultiClusterOverview: (): Promise<ApiResponse<MultiClusterOverview>> => {
    return request.get('/k8s/overview')
  }
}


// ==================== 简化的 K8s API ====================

export const k8sApi = {
  // 获取集群列表
  getClusters: (): Promise<ApiResponse<K8sCluster[]>> => {
    return request.get('/k8s-clusters', { params: { page: 1, page_size: 100 } }).then(res => {
      // 兼容分页响应格式
      if (res.data && res.data.items) {
        return { ...res, data: res.data.items } as unknown as ApiResponse<K8sCluster[]>
      }
      if (res.data && res.data.list) {
        return { ...res, data: res.data.list } as unknown as ApiResponse<K8sCluster[]>
      }
      return res as unknown as ApiResponse<K8sCluster[]>
    })
  },

  // 获取命名空间列表
  getNamespaces: (clusterId: number): Promise<ApiResponse<string[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/namespaces`).then(res => {
      // 兼容不同响应格式
      if (res.data && Array.isArray(res.data)) {
        // 如果是对象数组，提取 name 字段
        if (res.data.length > 0 && typeof res.data[0] === 'object') {
          return { ...res, data: res.data.map((ns: any) => ns.name || ns) } as unknown as ApiResponse<string[]>
        }
      }
      return res as unknown as ApiResponse<string[]>
    })
  },

  // 获取Pod列表
  getPods: (clusterId: number, namespace: string): Promise<ApiResponse<any[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/resources/pods`, { params: { namespace } })
  },

  // 兼容旧方法名
  listClusters: (): Promise<ApiResponse<{ list: K8sCluster[] }>> => {
    return request.get('/k8s-clusters', { params: { page: 1, page_size: 100 } })
  },

  listNamespaces: (clusterId: number): Promise<ApiResponse<string[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/namespaces`)
  },

  listPods: (clusterId: number, namespace: string): Promise<ApiResponse<any[]>> => {
    return request.get(`/k8s-clusters/${clusterId}/pods`, { params: { namespace } })
  }
}


// ==================== K8s Metrics API ====================

export interface ContainerMetrics {
  name: string
  cpu_usage: number    // 毫核
  cpu_limit: number    // 毫核
  cpu_percent: number  // 百分比
  mem_usage: number    // 字节
  mem_limit: number    // 字节
  mem_percent: number  // 百分比
}

export interface PodMetricsResponse {
  pod_name: string
  namespace: string
  available: boolean
  message?: string
  total_cpu: number    // 毫核
  total_mem: number    // 字节
  containers: ContainerMetrics[]
}

export interface PodMetricsSummary {
  pod_name: string
  namespace: string
  cpu_usage: number    // 毫核
  mem_usage: number    // 字节
}

export interface PodMetricsListResponse {
  available: boolean
  message?: string
  items: PodMetricsSummary[]
}

export interface NodeMetrics {
  node_name: string
  cpu_usage: number     // 毫核
  cpu_capacity: number  // 毫核
  cpu_percent: number   // 百分比
  mem_usage: number     // 字节
  mem_capacity: number  // 字节
  mem_percent: number   // 百分比
}

export interface NodeMetricsListResponse {
  available: boolean
  message?: string
  items: NodeMetrics[]
}

export const k8sMetricsApi = {
  // 获取单个 Pod 的资源指标
  getPodMetrics: (clusterId: number, namespace: string, podName: string): Promise<ApiResponse<PodMetricsResponse>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/pods/${podName}/metrics`)
  },

  // 获取命名空间下所有 Pod 的资源指标
  getPodListMetrics: (clusterId: number, namespace: string): Promise<ApiResponse<PodMetricsListResponse>> => {
    return request.get(`/k8s/clusters/${clusterId}/namespaces/${namespace}/metrics/pods`)
  },

  // 获取节点资源指标
  getNodeMetrics: (clusterId: number): Promise<ApiResponse<NodeMetricsListResponse>> => {
    return request.get(`/k8s/clusters/${clusterId}/metrics/nodes`)
  },

  // 检查 metrics-server 是否可用
  checkMetricsServer: (clusterId: number): Promise<ApiResponse<{ available: boolean; message: string }>> => {
    return request.get(`/k8s/clusters/${clusterId}/metrics/status`)
  }
}

// 格式化 CPU 使用量 (毫核 -> 人类可读)
export function formatCPU(milliCores: number): string {
  if (milliCores < 1000) {
    return `${milliCores}m`
  }
  return `${(milliCores / 1000).toFixed(2)}`
}

// 格式化内存使用量 (字节 -> 人类可读)
export function formatMemory(bytes: number): string {
  const KB = 1024
  const MB = KB * 1024
  const GB = MB * 1024

  if (bytes >= GB) {
    return `${(bytes / GB).toFixed(2)}Gi`
  }
  if (bytes >= MB) {
    return `${(bytes / MB).toFixed(2)}Mi`
  }
  if (bytes >= KB) {
    return `${(bytes / KB).toFixed(2)}Ki`
  }
  return `${bytes}B`
}

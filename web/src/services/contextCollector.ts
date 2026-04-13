import type { PageContext } from '../types/ai'
import type { RouteLocationNormalized } from 'vue-router'

// 页面类型映射
const PAGE_TYPE_MAP: Record<string, string> = {
  '/application': 'application_list',
  '/application/:id': 'application_detail',
  '/deploy': 'deploy_list',
  '/deploy/:id': 'deploy_detail',
  '/k8s/clusters': 'cluster_list',
  '/k8s/clusters/:id': 'cluster_detail',
  '/k8s/pods': 'pod_list',
  '/alert': 'alert_list',
  '/alert/:id': 'alert_detail',
  '/monitoring': 'monitoring_dashboard',
  '/logs': 'logs_query',
  '/pipeline': 'pipeline_list',
  '/pipeline/:id': 'pipeline_detail',
  '/traffic': 'traffic_management',
  '/approval': 'approval_list',
  '/approval/:id': 'approval_detail',
}

// 从路由路径获取页面类型
const getPageType = (path: string, matchedPath?: string): string => {
  // 优先使用匹配的路由路径
  if (matchedPath && PAGE_TYPE_MAP[matchedPath]) {
    return PAGE_TYPE_MAP[matchedPath]
  }

  // 尝试匹配路径
  for (const [pattern, type] of Object.entries(PAGE_TYPE_MAP)) {
    const regex = new RegExp('^' + pattern.replace(/:[\w]+/g, '[^/]+') + '$')
    if (regex.test(path)) {
      return type
    }
  }

  return 'unknown'
}

// 从路由参数提取上下文数据
const extractContextFromRoute = (route: RouteLocationNormalized): Partial<PageContext> => {
  const context: Partial<PageContext> = {}
  const params = route.params
  const query = route.query

  // 应用ID
  if (params.id && route.path.includes('/application')) {
    context.app_id = Number(params.id)
  }

  // 集群ID
  if (params.id && route.path.includes('/k8s/clusters')) {
    context.cluster_id = Number(params.id)
  }

  // 告警ID
  if (params.id && route.path.includes('/alert')) {
    context.alert_id = Number(params.id)
  }

  // 流水线ID
  if (params.id && route.path.includes('/pipeline')) {
    context.pipeline_id = Number(params.id)
  }

  // 命名空间
  if (query.namespace) {
    context.namespace = String(query.namespace)
  }

  // 部署名称
  if (query.deployment) {
    context.deployment_name = String(query.deployment)
  }

  return context
}

// 上下文收集器
export class ContextCollector {
  private currentContext: PageContext | null = null
  private extraData: Record<string, any> = {}

  // 从路由收集上下文
  collectFromRoute(route: RouteLocationNormalized): PageContext {
    const matchedPath = route.matched[route.matched.length - 1]?.path
    const pageType = getPageType(route.path, matchedPath)
    const routeContext = extractContextFromRoute(route)

    this.currentContext = {
      page_type: pageType,
      page_path: route.path,
      ...routeContext,
      extra_data: { ...this.extraData },
    }

    return this.currentContext
  }

  // 设置额外数据
  setExtraData(key: string, value: any): void {
    this.extraData[key] = value
    if (this.currentContext) {
      this.currentContext.extra_data = { ...this.extraData }
    }
  }

  // 设置应用信息
  setApplicationInfo(appId: number, appName: string): void {
    if (this.currentContext) {
      this.currentContext.app_id = appId
      this.currentContext.app_name = appName
    }
    this.extraData.app_id = appId
    this.extraData.app_name = appName
  }

  // 设置集群信息
  setClusterInfo(clusterId: number, clusterName: string, namespace?: string): void {
    if (this.currentContext) {
      this.currentContext.cluster_id = clusterId
      this.currentContext.cluster_name = clusterName
      if (namespace) {
        this.currentContext.namespace = namespace
      }
    }
    this.extraData.cluster_id = clusterId
    this.extraData.cluster_name = clusterName
    if (namespace) {
      this.extraData.namespace = namespace
    }
  }

  // 设置部署信息
  setDeploymentInfo(deploymentName: string): void {
    if (this.currentContext) {
      this.currentContext.deployment_name = deploymentName
    }
    this.extraData.deployment_name = deploymentName
  }

  // 获取当前上下文
  getContext(): PageContext | null {
    return this.currentContext
  }

  // 清除额外数据
  clearExtraData(): void {
    this.extraData = {}
    if (this.currentContext) {
      this.currentContext.extra_data = {}
    }
  }

  // 重置
  reset(): void {
    this.currentContext = null
    this.extraData = {}
  }
}

// 单例实例
export const contextCollector = new ContextCollector()

// 导出便捷函数
export const collectContext = (route: RouteLocationNormalized): PageContext => {
  return contextCollector.collectFromRoute(route)
}

export const setContextExtraData = (key: string, value: any): void => {
  contextCollector.setExtraData(key, value)
}

export const getCurrentContext = (): PageContext | null => {
  return contextCollector.getContext()
}

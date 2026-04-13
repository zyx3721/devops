import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/Login.vue'),
    meta: { title: '登录' },
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [
      {
        path: '',
        redirect: '/dashboard',
      },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '仪表盘' },
      },
      {
        path: 'jenkins/instances',
        name: 'JenkinsInstances',
        component: () => import('@/views/jenkins/JenkinsInstances.vue'),
        meta: { title: 'Jenkins 实例管理' },
      },
      {
        path: 'jenkins/instances/:id/jobs',
        name: 'JenkinsJobs',
        component: () => import('@/views/jenkins/JenkinsJobs.vue'),
        meta: { title: 'Jenkins Jobs' },
      },
      {
        path: 'k8s/clusters',
        name: 'K8sClusters',
        component: () => import('@/views/k8s/K8sClusters.vue'),
        meta: { title: 'K8s 集群管理' },
      },
      {
        path: 'k8s/clusters/:id/resources',
        name: 'K8sResources',
        component: () => import('@/views/k8s/K8sResources.vue'),
        meta: { title: 'K8s 资源管理' },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/user/UserManagement.vue'),
        meta: { title: '用户管理' },
      },
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/user/Profile.vue'),
        meta: { title: '个人中心' },
      },
      {
        path: 'feishu/message',
        name: 'FeishuMessage',
        component: () => import('@/views/feishu/FeishuMessage.vue'),
        meta: { title: '飞书消息' },
      },
      {
        path: 'dingtalk/message',
        name: 'DingtalkMessage',
        component: () => import('@/views/dingtalk/DingtalkMessage.vue'),
        meta: { title: '钉钉管理' },
      },
      {
        path: 'wechatwork/message',
        name: 'WechatWorkMessage',
        component: () => import('@/views/wechatwork/WechatWorkMessage.vue'),
        meta: { title: '企业微信管理' },
      },
      {
        path: 'message/channel',
        name: 'MessageChannel',
        component: () => import('@/views/message/MessageChannel.vue'),
        meta: { title: '消息通道' },
      },
      {
        path: 'oa/data',
        name: 'OAData',
        component: () => import('@/views/oa/OAData.vue'),
        meta: { title: 'OA 数据' },
      },
      {
        path: 'audit/logs',
        name: 'AuditLogs',
        component: () => import('@/views/audit/AuditLogs.vue'),
        meta: { title: '操作审计' },
      },
      {
        path: 'alert/overview',
        name: 'AlertOverview',
        component: () => import('@/views/alert/AlertOverview.vue'),
        meta: { title: '告警概览' },
      },
      {
        path: 'alert/config',
        name: 'AlertConfig',
        component: () => import('@/views/alert/AlertConfig.vue'),
        meta: { title: '告警配置' },
      },
      {
        path: 'alert/templates',
        name: 'MessageTemplate',
        component: () => import('@/views/alert/MessageTemplate.vue'),
        meta: { title: '消息模板' },
      },
      {
        path: 'alert/gateway',
        name: 'GatewayGuide',
        component: () => import('@/views/alert/GatewayGuide.vue'),
        meta: { title: '接入指南' },
      },
      {
        path: 'alert/history',
        name: 'AlertHistory',
        component: () => import('@/views/alert/AlertHistory.vue'),
        meta: { title: '告警历史' },
      },
      {
        path: 'alert/silence',
        name: 'AlertSilence',
        component: () => import('@/views/alert/AlertSilence.vue'),
        meta: { title: '静默规则' },
      },
      {
        path: 'alert/escalation',
        name: 'AlertEscalation',
        component: () => import('@/views/alert/AlertEscalation.vue'),
        meta: { title: '升级规则' },
      },
      {
        path: 'applications',
        name: 'Applications',
        component: () => import('@/views/application/ApplicationList.vue'),
        meta: { title: '应用管理' },
      },
      {
        path: 'applications/:id',
        name: 'ApplicationDetail',
        component: () => import('@/views/application/ApplicationDetail.vue'),
        meta: { title: '应用详情' },
      },
      {
        path: 'applications/traffic',
        name: 'TrafficManagementEntry',
        component: () => import('@/views/application/TrafficManagementEntry.vue'),
        meta: { title: '流量治理' },
      },
      {
        path: 'applications/:id/traffic',
        name: 'AppTrafficManagement',
        component: () => import('@/views/application/AppTrafficManagement.vue'),
        meta: { title: '流量治理' },
      },
      {
        path: 'traffic/ratelimit',
        name: 'RateLimitConfig',
        component: () => import('@/views/traffic/RateLimitConfig.vue'),
        meta: { title: '限流配置' },
      },
      {
        path: 'traffic/circuitbreaker',
        name: 'CircuitBreakerConfig',
        component: () => import('@/views/traffic/CircuitBreakerConfig.vue'),
        meta: { title: '熔断降级' },
      },
      {
        path: 'traffic/routing',
        name: 'TrafficRouting',
        component: () => import('@/views/traffic/TrafficRouting.vue'),
        meta: { title: '流量路由' },
      },
      {
        path: 'traffic/loadbalance',
        name: 'LoadBalanceConfig',
        component: () => import('@/views/traffic/LoadBalanceConfig.vue'),
        meta: { title: '负载均衡' },
      },
      {
        path: 'traffic/timeout',
        name: 'TimeoutConfig',
        component: () => import('@/views/traffic/TimeoutConfig.vue'),
        meta: { title: '超时重试' },
      },
      {
        path: 'traffic/mirror',
        name: 'MirrorConfig',
        component: () => import('@/views/traffic/MirrorConfig.vue'),
        meta: { title: '流量镜像' },
      },
      {
        path: 'traffic/fault',
        name: 'FaultConfig',
        component: () => import('@/views/traffic/FaultConfig.vue'),
        meta: { title: '故障注入' },
      },
      {
        path: 'deploys',
        name: 'DeployHistory',
        component: () => import('@/views/application/DeployHistory.vue'),
        meta: { title: '部署记录' },
      },
      {
        path: 'deploy/requests',
        name: 'DeployRequest',
        component: () => import('@/views/deploy/DeployRequest.vue'),
        meta: { title: '发布请求' },
      },
      {
        path: 'healthcheck',
        name: 'HealthCheck',
        component: () => import('@/views/healthcheck/HealthCheck.vue'),
        meta: { title: '健康检查' },
      },
      {
        path: 'healthcheck/ssl-cert',
        name: 'SSLCertCheck',
        component: () => import('@/views/healthcheck/SSLCertCheck.vue'),
        meta: { title: 'SSL 证书检查' },
      },

      // 审批流程
      {
        path: 'approval/chains',
        name: 'ApprovalChainList',
        component: () => import('@/views/approval/ApprovalChainList.vue'),
        meta: { title: '审批链管理' },
      },
      {
        path: 'approval/chains/:id/design',
        name: 'ApprovalChainDesigner',
        component: () => import('@/views/approval/ApprovalChainDesigner.vue'),
        meta: { title: '审批链设计' },
      },
      {
        path: 'approval/instances',
        name: 'ApprovalInstanceList',
        component: () => import('@/views/approval/ApprovalInstanceList.vue'),
        meta: { title: '审批实例' },
      },
      {
        path: 'approval/instances/:id',
        name: 'ApprovalInstanceDetail',
        component: () => import('@/views/approval/ApprovalInstancePage.vue'),
        meta: { title: '审批实例详情' },
      },
      {
        path: 'approval/rules',
        name: 'ApprovalRules',
        component: () => import('@/views/approval/ApprovalRules.vue'),
        meta: { title: '审批规则' },
      },
      {
        path: 'approval/windows',
        name: 'DeployWindows',
        component: () => import('@/views/approval/DeployWindows.vue'),
        meta: { title: '发布窗口' },
      },
      {
        path: 'approval/pending',
        name: 'PendingApprovals',
        component: () => import('@/views/approval/PendingApprovals.vue'),
        meta: { title: '待审批' },
      },
      {
        path: 'approval/history',
        name: 'ApprovalHistory',
        component: () => import('@/views/approval/ApprovalHistory.vue'),
        meta: { title: '审批历史' },
      },
      {
        path: 'deploy/locks',
        name: 'DeployLocks',
        component: () => import('@/views/deploy/DeployLocks.vue'),
        meta: { title: '部署锁' },
      },
      // K8s 运维增强
      {
        path: 'k8s/clusters/:id/pods',
        name: 'K8sPodManagement',
        component: () => import('@/views/k8s/PodManagement.vue'),
        meta: { title: 'Pod 管理' },
      },
      {
        path: 'k8s/clusters/:id/deployments',
        name: 'K8sDeploymentManagement',
        component: () => import('@/views/k8s/DeploymentManagement.vue'),
        meta: { title: 'Deployment 管理' },
      },
      // K8s 功能增强
      {
        path: 'k8s/overview',
        name: 'K8sClusterOverview',
        component: () => import('@/views/k8s/ClusterOverview.vue'),
        meta: { title: '集群概览' },
      },
      // 部署流程优化
      {
        path: 'deploy/check',
        name: 'DeployCheck',
        component: () => import('@/views/deploy/DeployCheck.vue'),
        meta: { title: '部署检查' },
      },
      // 灰度发布
      {
        path: 'canary/list',
        name: 'CanaryList',
        component: () => import('@/views/canary/CanaryList.vue'),
        meta: { title: '灰度发布' },
      },
      // 蓝绿部署
      {
        path: 'bluegreen/list',
        name: 'BlueGreenList',
        component: () => import('@/views/canary/BlueGreenList.vue'),
        meta: { title: '蓝绿部署' },
      },
      // 弹性工程
      {
        path: 'resilience',
        name: 'ResilienceManagement',
        component: () => import('@/views/resilience/ResilienceManagement.vue'),
        meta: { title: '弹性工程' },
      },
      // 流量监控
      {
        path: 'traffic/monitor',
        name: 'TrafficMonitor',
        component: () => import('@/views/traffic/TrafficMonitor.vue'),
        meta: { title: '流量监控' },
      },
      {
        path: 'applications/:id/traffic/monitor',
        name: 'AppTrafficMonitor',
        component: () => import('@/views/traffic/TrafficMonitor.vue'),
        meta: { title: '流量监控' },
      },
      // 成本管理
      {
        path: 'cost/overview',
        name: 'CostOverview',
        component: () => import('@/views/cost/CostOverview.vue'),
        meta: { title: '成本概览' },
      },
      {
        path: 'cost/trend',
        name: 'CostTrend',
        component: () => import('@/views/cost/CostTrend.vue'),
        meta: { title: '成本趋势' },
      },
      {
        path: 'cost/waste',
        name: 'CostWaste',
        component: () => import('@/views/cost/CostWaste.vue'),
        meta: { title: '资源浪费' },
      },
      {
        path: 'cost/suggestions',
        name: 'CostSuggestions',
        component: () => import('@/views/cost/CostSuggestions.vue'),
        meta: { title: '优化建议' },
      },
      {
        path: 'cost/budget',
        name: 'CostBudget',
        component: () => import('@/views/cost/CostBudget.vue'),
        meta: { title: '预算管理' },
      },
      {
        path: 'cost/config',
        name: 'CostConfig',
        component: () => import('@/views/cost/CostConfig.vue'),
        meta: { title: '成本配置' },
      },
      {
        path: 'cost/alerts',
        name: 'CostAlerts',
        component: () => import('@/views/cost/CostAlerts.vue'),
        meta: { title: '成本告警' },
      },
      {
        path: 'cost/comparison',
        name: 'CostComparison',
        component: () => import('@/views/cost/CostComparison.vue'),
        meta: { title: '成本对比' },
      },
      {
        path: 'cost/analysis',
        name: 'CostAnalysis',
        component: () => import('@/views/cost/CostAnalysis.vue'),
        meta: { title: '多维分析' },
      },
      // 安全合规中心
      {
        path: 'security/overview',
        name: 'SecurityOverview',
        component: () => import('@/views/security/SecurityOverview.vue'),
        meta: { title: '安全概览' },
      },
      {
        path: 'security/image-scan',
        name: 'ImageScan',
        component: () => import('@/views/security/ImageScan.vue'),
        meta: { title: '镜像扫描' },
      },
      {
        path: 'security/config-check',
        name: 'ConfigCheck',
        component: () => import('@/views/security/ConfigCheck.vue'),
        meta: { title: '配置检查' },
      },
      {
        path: 'security/audit-log',
        name: 'SecurityAuditLog',
        component: () => import('@/views/security/AuditLog.vue'),
        meta: { title: '安全审计' },
      },
      {
        path: 'security/image-registry',
        name: 'ImageRegistry',
        component: () => import('@/views/security/ImageRegistry.vue'),
        meta: { title: '镜像仓库' },
      },
      // CI/CD 流水线
      {
        path: 'pipeline/list',
        name: 'PipelineList',
        component: () => import('@/views/pipeline/PipelineList.vue'),
        meta: { title: '流水线列表' },
      },
      {
        path: 'pipeline/:id',
        name: 'PipelineDetail',
        component: () => import('@/views/pipeline/PipelineDetail.vue'),
        meta: { title: '流水线详情' },
      },

      // Pod 终端
      {
        path: 'k8s/terminal',
        name: 'PodTerminal',
        component: () => import('@/views/k8s/PodTerminal.vue'),
        meta: { title: 'Pod 终端' },
      },
      {
        path: 'pipeline/git-repos',
        name: 'GitRepos',
        component: () => import('@/views/pipeline/GitRepos.vue'),
        meta: { title: 'Git 仓库' },
      },
      {
        path: 'pipeline/artifacts',
        name: 'Artifacts',
        component: () => import('@/views/pipeline/Artifacts.vue'),
        meta: { title: '构建制品' },
      },
      {
        path: 'pipeline/artifacts/:artifactId/versions',
        name: 'ArtifactVersions',
        component: () => import('@/views/pipeline/ArtifactVersions.vue'),
        meta: { title: '制品版本' },
      },
      {
        path: 'pipeline/create',
        name: 'PipelineCreate',
        component: () => import('@/views/pipeline/PipelineEditor.vue'),
        meta: { title: '创建流水线' },
      },
      {
        path: 'pipeline/edit/:id',
        name: 'PipelineEdit',
        component: () => import('@/views/pipeline/PipelineEditor.vue'),
        meta: { title: '编辑流水线' },
      },
      {
        path: 'pipeline/notify',
        name: 'PipelineNotify',
        component: () => import('@/views/pipeline/NotifyConfig.vue'),
        meta: { title: '通知配置' },
      },
      {
        path: 'pipeline/stats',
        name: 'PipelineStats',
        component: () => import('@/views/pipeline/PipelineStats.vue'),
        meta: { title: '执行统计' },
      },
      {
        path: 'pipeline/builders',
        name: 'BuilderPods',
        component: () => import('@/views/pipeline/BuilderPods.vue'),
        meta: { title: '构建 Pod' },
      },
      {
        path: 'pipeline/credentials',
        name: 'PipelineCredentials',
        component: () => import('@/views/pipeline/Credentials.vue'),
        meta: { title: '凭证管理' },
      },
      {
        path: 'pipeline/variables',
        name: 'PipelineVariables',
        component: () => import('@/views/pipeline/Variables.vue'),
        meta: { title: '变量管理' },
      },
      // 日志中心
      {
        path: 'logs/center',
        name: 'LogCenter',
        component: () => import('@/views/logs/LogCenter.vue'),
        meta: { title: '日志中心' },
      },
      {
        path: 'logs/viewer',
        name: 'LogViewer',
        redirect: '/logs/center',
        meta: { title: '日志查看' },
      },
      {
        path: 'logs/search',
        name: 'LogSearch',
        component: () => import('@/views/logs/LogSearch.vue'),
        meta: { title: '日志搜索' },
      },
      {
        path: 'logs/export',
        name: 'LogExport',
        component: () => import('@/views/logs/LogExportPage.vue'),
        meta: { title: '日志导出' },
      },
      {
        path: 'logs/alerts',
        name: 'LogAlertConfig',
        component: () => import('@/views/logs/LogAlertConfig.vue'),
        meta: { title: '日志告警' },
      },
      {
        path: 'logs/stats',
        name: 'LogStats',
        component: () => import('@/views/logs/LogStats.vue'),
        meta: { title: '日志统计' },
      },
      {
        path: 'logs/compare',
        name: 'LogCompare',
        component: () => import('@/views/logs/LogCompare.vue'),
        meta: { title: '日志对比' },
      },
      {
        path: 'logs/bookmarks',
        name: 'LogBookmarks',
        component: () => import('@/views/logs/LogBookmarks.vue'),
        meta: { title: '日志书签' },
      },
      // 管理后台
      {
        path: 'admin/dashboard',
        name: 'AdminDashboard',
        component: () => import('@/views/admin/Dashboard.vue'),
        meta: { title: '管理后台' },
      },
      // RBAC 角色管理
      {
        path: 'rbac/roles',
        name: 'RoleManagement',
        component: () => import('@/views/rbac/RoleManagement.vue'),
        meta: { title: '角色管理' },
      },
      // 模板市场
      {
        path: 'pipeline/templates',
        name: 'TemplateMarket',
        component: () => import('@/views/pipeline/TemplateMarket.vue'),
        meta: { title: '模板市场' },
      },
      // 构建缓存管理
      {
        path: 'pipeline/cache',
        name: 'BuildCache',
        component: () => import('@/views/pipeline/BuildCache.vue'),
        meta: { title: '构建缓存' },
      },
      // 构建使用统计
      {
        path: 'pipeline/stats/usage',
        name: 'BuildStats',
        component: () => import('@/views/pipeline/BuildStats.vue'),
        meta: { title: '构建统计' },
      },
      // 并行构建配置
      {
        path: 'pipeline/:pipelineId/parallel',
        name: 'ParallelConfig',
        component: () => import('@/views/pipeline/ParallelConfig.vue'),
        meta: { title: '并行构建配置' },
      },
      // 资源配额管理
      {
        path: 'pipeline/quota',
        name: 'ResourceQuota',
        component: () => import('@/views/pipeline/ResourceQuota.vue'),
        meta: { title: '资源配额' },
      },
      // 制品扫描结果
      {
        path: 'pipeline/artifacts/:versionId/scan',
        name: 'ArtifactScan',
        component: () => import('@/views/pipeline/ArtifactScan.vue'),
        meta: { title: '扫描结果' },
      },
      // 流水线可视化设计器
      {
        path: 'pipeline/designer',
        name: 'PipelineDesigner',
        component: () => import('@/views/pipeline/PipelineDesigner.vue'),
        meta: { title: '流水线设计器' },
      },
      // AI 助手管理
      {
        path: 'ai/knowledge',
        name: 'AIKnowledge',
        component: () => import('@/views/ai/KnowledgeManage.vue'),
        meta: { title: 'AI 知识库' },
      },
      {
        path: 'ai/config',
        name: 'AIConfig',
        component: () => import('@/views/ai/LLMConfig.vue'),
        meta: { title: 'AI 配置' },
      },
    ],
  },
  // 404 页面 - 必须放在最后
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/error/NotFound.vue'),
    meta: { title: '页面不存在' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || 'DevOps'} - 管理系统`

  const token = localStorage.getItem('token')
  if (to.path !== '/login' && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/')
  } else {
    next()
  }
})

// 处理路由错误（如组件加载失败）
router.onError((error) => {
  console.error('路由错误:', error)
  
  // 如果是组件加载失败，跳转到 404 页面
  if (error.message.includes('Failed to fetch dynamically imported module') ||
      error.message.includes('error loading dynamically imported module')) {
    router.push('/404')
  }
})

export default router

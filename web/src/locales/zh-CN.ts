/**
 * 中文语言包
 */
export default {
  menu: {
    dashboard: '仪表盘',
    
    // 应用管理
    app: '应用管理',
    applications: '应用列表',
    traffic: '流量治理',
    ratelimit: '限流配置',
    circuitbreaker: '熔断降级',
    routing: '流量路由',
    loadbalance: '负载均衡',
    timeout: '超时重试',
    mirror: '流量镜像',
    fault: '故障注入',
    trafficMonitor: '流量监控',
    resilience: '弹性工程',
    deploys: '部署记录',
    deployCheck: '部署检查',
    canary: '灰度发布',
    bluegreen: '蓝绿部署',
    
    // 健康检查
    healthcheck: '健康检查',
    serviceHealth: '服务健康检查',
    sslCert: 'SSL 证书检查',
    
    // CI/CD 流水线
    pipeline: 'CI/CD 流水线',
    pipelineDesigner: '流水线设计器',
    pipelineList: '流水线列表',
    pipelineStats: '执行统计',
    templates: '模板市场',
    gitRepos: 'Git 仓库',
    artifacts: '构建制品',
    builders: '构建 Pod',
    buildCache: '构建缓存',
    buildStats: '构建统计',
    quota: '资源配额',
    credentials: '凭证管理',
    variables: '变量管理',
    notify: '通知配置',
    
    // Jenkins
    jenkins: 'Jenkins',
    jenkinsInstances: '实例管理',
    
    // Kubernetes
    k8s: 'Kubernetes',
    k8sOverview: '集群概览',
    k8sClusters: '集群管理',
    securityOverview: '安全概览',
    imageScan: '镜像扫描',
    configCheck: '配置检查',
    auditLog: '安全审计',
    
    // 发布审批
    approval: '发布审批',
    pending: '待审批',
    approvalHistory: '审批历史',
    chains: '审批链管理',
    instances: '审批实例',
    rules: '审批规则',
    windows: '发布窗口',
    deployLocks: '部署锁',
    
    // 消息通道
    message: '消息通道',
    feishu: '飞书',
    dingtalk: '钉钉',
    wechatwork: '企业微信',
    
    // OA
    oa: 'OA',
    oaData: '数据管理',
    
    // 告警管理
    alert: '告警管理',
    alertOverview: '告警概览',
    alertHistory: '告警历史',
    alertConfig: '告警配置',
    alertTemplates: '消息模板',
    alertGateway: '接入指南',
    silence: '静默规则',
    escalation: '升级规则',
    
    // 日志中心
    logs: '日志中心',
    logsCenter: '日志查看',
    logsSearch: '日志搜索',
    logsStats: '日志统计',
    logsCompare: '日志对比',
    logsAlerts: '日志告警',
    logsBookmarks: '日志书签',
    logsExport: '日志导出',
    
    // 成本管理
    cost: '成本管理',
    costOverview: '成本概览',
    costTrend: '成本趋势',
    costComparison: '成本对比',
    costAnalysis: '多维分析',
    costWaste: '资源浪费',
    costSuggestions: '优化建议',
    costAlerts: '成本告警',
    costBudget: '预算管理',
    costConfig: '成本配置',
    
    // 系统管理
    system: '系统管理',
    users: '用户管理',
    roles: '角色权限',
    featureFlags: '功能开关',
    systemMonitor: '系统监控',
    auditLogs: '操作审计',
    aiKnowledge: 'AI 知识库',
    aiConfig: 'AI 配置',
  },
  
  common: {
    home: '首页',
    profile: '个人中心',
    logout: '退出登录',
    search: '全局搜索',
    searchPlaceholder: '搜索流水线、集群、应用、用户...',
    searching: '搜索中...',
    noResults: '未找到相关结果',
    quickNav: '快捷导航',
    select: '选择',
    open: '打开',
    close: '关闭',
  },
  
  breadcrumb: {
    home: '首页',
    dashboard: '仪表盘',
    
    // 应用管理
    app: '应用管理',
    applications: '应用列表',
    traffic: '流量治理',
    ratelimit: '限流配置',
    circuitbreaker: '熔断降级',
    routing: '流量路由',
    loadbalance: '负载均衡',
    timeout: '超时重试',
    mirror: '流量镜像',
    fault: '故障注入',
    trafficMonitor: '流量监控',
    resilience: '弹性工程',
    deploys: '部署记录',
    deployCheck: '部署检查',
    canary: '灰度发布',
    bluegreen: '蓝绿部署',
    
    // 健康检查
    healthcheck: '健康检查',
    serviceHealth: '服务健康检查',
    sslCert: 'SSL 证书检查',
    
    // CI/CD 流水线
    pipeline: 'CI/CD 流水线',
    pipelineDesigner: '流水线设计器',
    pipelineList: '流水线列表',
    pipelineStats: '执行统计',
    pipelineDetail: '流水线详情',
    pipelineEdit: '编辑流水线',
    templates: '模板市场',
    gitRepos: 'Git 仓库',
    artifacts: '构建制品',
    builders: '构建 Pod',
    buildCache: '构建缓存',
    buildStats: '构建统计',
    quota: '资源配额',
    credentials: '凭证管理',
    variables: '变量管理',
    notify: '通知配置',
    
    // Jenkins
    jenkins: 'Jenkins',
    jenkinsInstances: '实例管理',
    jenkinsJobs: 'Jobs',
    
    // Kubernetes
    k8s: 'Kubernetes',
    k8sOverview: '集群概览',
    k8sClusters: '集群管理',
    k8sResources: '资源管理',
    k8sPods: 'Pod 管理',
    k8sDeployments: 'Deployment 管理',
    securityOverview: '安全概览',
    imageScan: '镜像扫描',
    configCheck: '配置检查',
    auditLog: '安全审计',
    
    // 发布审批
    approval: '发布审批',
    pending: '待审批',
    approvalHistory: '审批历史',
    chains: '审批链管理',
    chainDesign: '审批链设计',
    instances: '审批实例',
    instanceDetail: '实例详情',
    rules: '审批规则',
    windows: '发布窗口',
    deployLocks: '部署锁',
    
    // 消息通道
    message: '消息通道',
    feishu: '飞书',
    dingtalk: '钉钉',
    wechatwork: '企业微信',
    
    // OA
    oa: 'OA',
    oaData: '数据管理',
    
    // 告警管理
    alert: '告警管理',
    alertOverview: '告警概览',
    alertHistory: '告警历史',
    alertConfig: '告警配置',
    silence: '静默规则',
    escalation: '升级规则',
    
    // 日志中心
    logs: '日志中心',
    logsCenter: '日志查看',
    logsSearch: '日志搜索',
    logsStats: '日志统计',
    logsCompare: '日志对比',
    logsAlerts: '日志告警',
    logsBookmarks: '日志书签',
    logsViewer: '日志查看器',
    logsExport: '日志导出',
    
    // 成本管理
    cost: '成本管理',
    costOverview: '成本概览',
    costTrend: '成本趋势',
    costComparison: '成本对比',
    costAnalysis: '多维分析',
    costWaste: '资源浪费',
    costSuggestions: '优化建议',
    costAlerts: '成本告警',
    costBudget: '预算管理',
    costConfig: '成本配置',
    
    // 系统管理
    system: '系统管理',
    users: '用户管理',
    roles: '角色权限',
    featureFlags: '功能开关',
    systemMonitor: '系统监控',
    auditLogs: '操作审计',
    aiKnowledge: 'AI 知识库',
    aiConfig: 'AI 配置',
    profile: '个人中心',
  },
  
  search: {
    title: '全局搜索',
    placeholder: '搜索流水线、集群、应用、用户...',
    searching: '搜索中...',
    noResults: '未找到相关结果',
    quickNav: '快捷导航',
    
    // 分组标签
    menu: '功能菜单',
    pipeline: '流水线',
    cluster: 'K8s 集群',
    application: '应用',
    user: '用户',
    
    // 快捷导航项
    quickPipeline: '流水线',
    quickK8s: 'K8s 集群',
    quickTraffic: '流量治理',
    quickLogs: '日志中心',
    quickCost: '成本管理',
    quickSecurity: '安全中心',
    quickAlert: '告警管理',
    quickUsers: '用户管理',
    quickApproval: '待审批',
    
    // 搜索项分类
    categoryTraffic: '流量治理',
    categoryCICD: 'CI/CD',
    categoryMonitor: '监控',
    categorySystem: '系统管理',
    
    // 搜索项标题
    resilience: '弹性工程',
    pipelineDesigner: '流水线设计器',
    templates: '模板市场',
    buildCache: '构建缓存',
    buildStats: '构建统计',
    quota: '资源配额',
    credentials: '凭证管理',
    variables: '变量管理',
    serviceHealth: '服务健康检查',
    sslCert: 'SSL 证书检查',
    featureFlags: '功能开关',
    systemMonitor: '系统监控',
  },
}

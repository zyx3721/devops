<template>
  <a-layout style="min-height: 100vh">
    <a-layout-sider v-model:collapsed="collapsed" :trigger="null" collapsible theme="dark" width="200">
      <div class="logo">
        <!-- 使用 v-show 替代 v-if 以避免重复渲染 -->
        <span v-show="!collapsed">DevOps 管理系统</span>
        <span v-show="collapsed">DevOps</span>
      </div>
      <a-menu 
        v-model:selectedKeys="selectedKeys" 
        v-model:openKeys="openKeys" 
        theme="dark" 
        mode="inline" 
        @click="handleMenuClick"
        :inline-collapsed="collapsed"
      >
        <template v-for="item in filteredMenuConfig" :key="item.key">
          <!-- 单级菜单 -->
          <a-menu-item v-if="!item.children || item.children.length === 0" :key="item.key">
            <template v-if="item.icon" #icon>
              <component :is="item.icon" />
            </template>
            {{ t(item.titleKey) }}
          </a-menu-item>
          
          <!-- 多级菜单 -->
          <a-sub-menu v-else :key="item.key">
            <template v-if="item.icon" #icon>
              <component :is="item.icon" />
            </template>
            <template #title>{{ t(item.titleKey) }}</template>
            
            <!-- 二级菜单 -->
            <template v-for="child in item.children" :key="child.key">
              <!-- 二级单项 -->
              <a-menu-item v-if="!child.children || child.children.length === 0" :key="child.key">
                <template v-if="child.icon" #icon>
                  <component :is="child.icon" />
                </template>
                {{ t(child.titleKey) }}
              </a-menu-item>
              
              <!-- 二级子菜单 -->
              <a-sub-menu v-else :key="child.key">
                <template #title>{{ t(child.titleKey) }}</template>
                <a-menu-item v-for="grandChild in child.children" :key="grandChild.key">
                  <template v-if="grandChild.icon" #icon>
                    <component :is="grandChild.icon" />
                  </template>
                  {{ t(grandChild.titleKey) }}
                </a-menu-item>
              </a-sub-menu>
            </template>
          </a-sub-menu>
        </template>
      </a-menu>
    </a-layout-sider>

    <a-layout>
      <a-layout-header style="background: #fff; padding: 0 24px; display: flex; align-items: center; justify-content: space-between">
        <div class="trigger" @click="collapsed = !collapsed">
          <MenuUnfoldOutlined v-if="collapsed" />
          <MenuFoldOutlined v-else />
        </div>
        <div class="header-right">
          <a-space>
            <GlobalSearch />
            <FavoriteList />
            <ThemeSwitch />
            <LanguageSwitcher />
            <a-dropdown>
              <a-space style="cursor: pointer">
                <a-avatar>{{ userInfo?.username?.charAt(0)?.toUpperCase() || 'U' }}</a-avatar>
                <span>{{ userInfo?.username || '管理员' }}</span>
              </a-space>
              <template #overlay>
                <a-menu @click="handleDropdownClick">
                  <a-menu-item key="profile">{{ t('common.profile') }}</a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="logout">{{ t('common.logout') }}</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </a-space>
        </div>
      </a-layout-header>

      <a-layout-content style="margin: 16px; background: #f0f2f5; min-height: calc(100vh - 80px)">
        <!-- 面包屑导航 -->
        <a-breadcrumb style="margin-bottom: 16px" v-if="breadcrumbs.length > 1">
          <a-breadcrumb-item v-for="(item, index) in breadcrumbs" :key="index">
            <router-link v-if="item.path && index < breadcrumbs.length - 1" :to="item.path">
              {{ item.title }}
            </router-link>
            <span v-else>{{ item.title }}</span>
          </a-breadcrumb-item>
        </a-breadcrumb>
        <div style="background: #fff; padding: 20px; border-radius: 4px">
          <router-view />
        </div>
      </a-layout-content>
    </a-layout>

    <!-- AI 助手悬浮窗 -->
    <AIChatWidget />
  </a-layout>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  DashboardOutlined,
  ThunderboltOutlined,
  CloudOutlined,
  MessageOutlined,
  FileTextOutlined,
  AlertOutlined,
  AppstoreOutlined,
  HeartOutlined,
  SettingOutlined,
  AuditOutlined,
  DollarOutlined,
  RocketOutlined,
  FileSearchOutlined,
  LayoutOutlined,
  ShopOutlined,
  DatabaseOutlined,
  BarChartOutlined,
  FundOutlined,
  KeyOutlined,
  ControlOutlined,
  SafetyCertificateOutlined
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import GlobalSearch from '@/components/GlobalSearch.vue'
import FavoriteList from '@/components/FavoriteList.vue'
import ThemeSwitch from '@/components/ThemeSwitch.vue'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'
import { AIChatWidget } from '@/components/ai'
import { useI18n } from 'vue-i18n'
const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const collapsed = ref(false)
const selectedKeys = ref<string[]>([route.path])

// 移动端检测 - 使用响应式变量
const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1920)

const isMobile = computed(() => {
  return windowWidth.value < 768
})

// 监听窗口大小变化
const handleResize = () => {
  windowWidth.value = window.innerWidth
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
  // 初始化时检查是否为移动端
  if (isMobile.value) {
    collapsed.value = true
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})

// 面包屑配置 - 使用 i18n key
const breadcrumbMap: Record<string, { titleKey: string; parent?: string }> = {
  '/dashboard': { titleKey: 'breadcrumb.dashboard' },
  '/applications': { titleKey: 'breadcrumb.applications', parent: 'app' },
  '/applications/traffic': { titleKey: 'breadcrumb.traffic', parent: 'app' },
  '/traffic/ratelimit': { titleKey: 'breadcrumb.ratelimit', parent: 'traffic' },
  '/traffic/circuitbreaker': { titleKey: 'breadcrumb.circuitbreaker', parent: 'traffic' },
  '/traffic/routing': { titleKey: 'breadcrumb.routing', parent: 'traffic' },
  '/traffic/loadbalance': { titleKey: 'breadcrumb.loadbalance', parent: 'traffic' },
  '/traffic/timeout': { titleKey: 'breadcrumb.timeout', parent: 'traffic' },
  '/traffic/mirror': { titleKey: 'breadcrumb.mirror', parent: 'traffic' },
  '/traffic/fault': { titleKey: 'breadcrumb.fault', parent: 'traffic' },
  '/deploys': { titleKey: 'breadcrumb.deploys', parent: 'app' },
  '/deploy/check': { titleKey: 'breadcrumb.deployCheck', parent: 'app' },
  '/canary/list': { titleKey: 'breadcrumb.canary', parent: 'app' },
  '/bluegreen/list': { titleKey: 'breadcrumb.bluegreen', parent: 'app' },
  '/traffic/monitor': { titleKey: 'breadcrumb.trafficMonitor', parent: 'traffic' },
  '/resilience': { titleKey: 'breadcrumb.resilience', parent: 'traffic' },
  '/healthcheck': { titleKey: 'breadcrumb.serviceHealth', parent: 'healthcheck' },
  '/healthcheck/ssl-cert': { titleKey: 'breadcrumb.sslCert', parent: 'healthcheck' },
  '/pipeline/designer': { titleKey: 'breadcrumb.pipelineDesigner', parent: 'pipeline' },
  '/pipeline/list': { titleKey: 'breadcrumb.pipelineList', parent: 'pipeline' },
  '/pipeline/git-repos': { titleKey: 'breadcrumb.gitRepos', parent: 'pipeline' },
  '/pipeline/artifacts': { titleKey: 'breadcrumb.artifacts', parent: 'pipeline' },
  '/pipeline/builders': { titleKey: 'breadcrumb.builders', parent: 'pipeline' },
  '/pipeline/cache': { titleKey: 'breadcrumb.buildCache', parent: 'pipeline' },
  '/pipeline/stats/usage': { titleKey: 'breadcrumb.buildStats', parent: 'pipeline' },
  '/pipeline/quota': { titleKey: 'breadcrumb.quota', parent: 'pipeline' },
  '/pipeline/credentials': { titleKey: 'breadcrumb.credentials', parent: 'pipeline' },
  '/pipeline/variables': { titleKey: 'breadcrumb.variables', parent: 'pipeline' },
  '/pipeline/templates': { titleKey: 'breadcrumb.templates', parent: 'pipeline' },
  '/pipeline/notify': { titleKey: 'breadcrumb.notify', parent: 'pipeline' },
  '/pipeline/stats': { titleKey: 'breadcrumb.pipelineStats', parent: 'pipeline' },
  '/pipeline/create': { titleKey: 'breadcrumb.pipelineEdit', parent: 'pipeline' },
  '/jenkins/instances': { titleKey: 'breadcrumb.jenkinsInstances', parent: 'jenkins' },
  '/k8s/overview': { titleKey: 'breadcrumb.k8sOverview', parent: 'k8s' },
  '/k8s/clusters': { titleKey: 'breadcrumb.k8sClusters', parent: 'k8s' },
  '/k8s/terminal': { titleKey: 'breadcrumb.k8sPods', parent: 'k8s' },
  '/security/overview': { titleKey: 'breadcrumb.securityOverview', parent: 'k8s' },
  '/security/image-scan': { titleKey: 'breadcrumb.imageScan', parent: 'k8s' },
  '/security/config-check': { titleKey: 'breadcrumb.configCheck', parent: 'k8s' },
  '/security/audit-log': { titleKey: 'breadcrumb.auditLog', parent: 'k8s' },
  '/approval/pending': { titleKey: 'breadcrumb.pending', parent: 'approval' },
  '/approval/history': { titleKey: 'breadcrumb.approvalHistory', parent: 'approval' },
  '/approval/chains': { titleKey: 'breadcrumb.chains', parent: 'approval' },
  '/approval/instances': { titleKey: 'breadcrumb.instances', parent: 'approval' },
  '/approval/rules': { titleKey: 'breadcrumb.rules', parent: 'approval' },
  '/approval/windows': { titleKey: 'breadcrumb.windows', parent: 'approval' },
  '/deploy/locks': { titleKey: 'breadcrumb.deployLocks', parent: 'approval' },
  '/feishu/message': { titleKey: 'breadcrumb.feishu', parent: 'message' },
  '/dingtalk/message': { titleKey: 'breadcrumb.dingtalk', parent: 'message' },
  '/wechatwork/message': { titleKey: 'breadcrumb.wechatwork', parent: 'message' },
  '/oa/data': { titleKey: 'breadcrumb.oaData', parent: 'oa' },
  '/alert/overview': { titleKey: 'breadcrumb.alertOverview', parent: 'alert' },
  '/alert/history': { titleKey: 'breadcrumb.alertHistory', parent: 'alert' },
  '/alert/config': { titleKey: 'breadcrumb.alertConfig', parent: 'alert' },
  '/alert/silence': { titleKey: 'breadcrumb.silence', parent: 'alert' },
  '/alert/escalation': { titleKey: 'breadcrumb.escalation', parent: 'alert' },
  '/cost/overview': { titleKey: 'breadcrumb.costOverview', parent: 'cost' },
  '/cost/trend': { titleKey: 'breadcrumb.costTrend', parent: 'cost' },
  '/cost/comparison': { titleKey: 'breadcrumb.costComparison', parent: 'cost' },
  '/cost/analysis': { titleKey: 'breadcrumb.costAnalysis', parent: 'cost' },
  '/cost/waste': { titleKey: 'breadcrumb.costWaste', parent: 'cost' },
  '/cost/suggestions': { titleKey: 'breadcrumb.costSuggestions', parent: 'cost' },
  '/cost/alerts': { titleKey: 'breadcrumb.costAlerts', parent: 'cost' },
  '/cost/budget': { titleKey: 'breadcrumb.costBudget', parent: 'cost' },
  '/cost/config': { titleKey: 'breadcrumb.costConfig', parent: 'cost' },
  '/logs/center': { titleKey: 'breadcrumb.logsCenter', parent: 'logs' },
  '/logs/search': { titleKey: 'breadcrumb.logsSearch', parent: 'logs' },
  '/logs/stats': { titleKey: 'breadcrumb.logsStats', parent: 'logs' },
  '/logs/compare': { titleKey: 'breadcrumb.logsCompare', parent: 'logs' },
  '/logs/alerts': { titleKey: 'breadcrumb.logsAlerts', parent: 'logs' },
  '/logs/bookmarks': { titleKey: 'breadcrumb.logsBookmarks', parent: 'logs' },
  '/logs/viewer': { titleKey: 'breadcrumb.logsViewer', parent: 'logs' },
  '/logs/export': { titleKey: 'breadcrumb.logsExport', parent: 'logs' },
  '/users': { titleKey: 'breadcrumb.users', parent: 'system' },
  '/rbac/roles': { titleKey: 'breadcrumb.roles', parent: 'system' },
  '/audit/logs': { titleKey: 'breadcrumb.auditLogs', parent: 'system' },
  '/profile': { titleKey: 'breadcrumb.profile' },
}

const parentTitles: Record<string, { titleKey: string; path?: string }> = {
  app: { titleKey: 'breadcrumb.app' },
  traffic: { titleKey: 'breadcrumb.traffic', path: '/traffic/ratelimit' },
  healthcheck: { titleKey: 'breadcrumb.healthcheck' },
  pipeline: { titleKey: 'breadcrumb.pipeline', path: '/pipeline/list' },
  jenkins: { titleKey: 'breadcrumb.jenkins' },
  k8s: { titleKey: 'breadcrumb.k8s', path: '/k8s/clusters' },
  approval: { titleKey: 'breadcrumb.approval' },
  message: { titleKey: 'breadcrumb.message' },
  oa: { titleKey: 'breadcrumb.oa' },
  alert: { titleKey: 'breadcrumb.alert' },
  cost: { titleKey: 'breadcrumb.cost', path: '/cost/overview' },
  logs: { titleKey: 'breadcrumb.logs', path: '/logs/center' },
  system: { titleKey: 'breadcrumb.system' },
}

const breadcrumbs = computed(() => {
  const path = route.path
  const result: { title: string; path?: string }[] = [{ title: t('breadcrumb.home'), path: '/dashboard' }]
  
  // 处理动态路由，如 /pipeline/:id
  let matchedPath = path
  const config = breadcrumbMap[path]
  
  if (!config) {
    // 尝试匹配动态路由
    if (path.match(/^\/pipeline\/\d+$/)) {
      result.push({ title: t('breadcrumb.pipeline'), path: '/pipeline/list' })
      result.push({ title: t('breadcrumb.pipelineDetail') })
      return result
    }
    if (path.match(/^\/pipeline\/edit\/\d+$/)) {
      result.push({ title: t('breadcrumb.pipeline'), path: '/pipeline/list' })
      result.push({ title: t('breadcrumb.pipelineEdit') })
      return result
    }
    if (path.match(/^\/k8s\/clusters\/\d+\/resources$/)) {
      result.push({ title: t('breadcrumb.k8s'), path: '/k8s/clusters' })
      result.push({ title: t('breadcrumb.k8sResources') })
      return result
    }
    if (path.match(/^\/k8s\/clusters\/\d+\/pods$/)) {
      result.push({ title: t('breadcrumb.k8s'), path: '/k8s/clusters' })
      result.push({ title: t('breadcrumb.k8sPods') })
      return result
    }
    if (path.match(/^\/k8s\/clusters\/\d+\/deployments$/)) {
      result.push({ title: t('breadcrumb.k8s'), path: '/k8s/clusters' })
      result.push({ title: t('breadcrumb.k8sDeployments') })
      return result
    }
    if (path.match(/^\/jenkins\/instances\/\d+\/jobs$/)) {
      result.push({ title: t('breadcrumb.jenkins'), path: '/jenkins/instances' })
      result.push({ title: t('breadcrumb.jenkinsJobs') })
      return result
    }
    if (path.match(/^\/approval\/chains\/\d+\/design$/)) {
      result.push({ title: t('breadcrumb.approval') })
      result.push({ title: t('breadcrumb.chains'), path: '/approval/chains' })
      result.push({ title: t('breadcrumb.chainDesign') })
      return result
    }
    if (path.match(/^\/approval\/instances\/\d+$/)) {
      result.push({ title: t('breadcrumb.approval') })
      result.push({ title: t('breadcrumb.instances'), path: '/approval/instances' })
      result.push({ title: t('breadcrumb.instanceDetail') })
      return result
    }
    return result
  }
  
  if (config.parent) {
    const parent = parentTitles[config.parent]
    if (parent) {
      result.push({ title: t(parent.titleKey), path: parent.path })
    }
  }
  
  result.push({ title: t(config.titleKey) })
  return result
})

// 根据路径获取父菜单 key
const getParentKey = (path: string): string => {
  if (path.startsWith('/traffic/') || path.startsWith('/resilience')) return 'traffic'
  if (path.startsWith('/applications') || path.startsWith('/deploys') || path.startsWith('/deploy/check') || path.startsWith('/canary') || path.startsWith('/bluegreen')) return 'app'
  if (path.startsWith('/healthcheck')) return 'healthcheck'
  if (path.startsWith('/pipeline')) return 'pipeline'
  if (path.startsWith('/jenkins')) return 'jenkins'
  if (path.startsWith('/k8s') || path.startsWith('/security')) return 'k8s'
  if (path.startsWith('/approval') || path.startsWith('/deploy/locks')) return 'approval'
  if (path.startsWith('/feishu') || path.startsWith('/dingtalk') || path.startsWith('/wechatwork')) return 'message'
  if (path.startsWith('/oa')) return 'oa'
  if (path.startsWith('/alert')) return 'alert'
  if (path.startsWith('/cost')) return 'cost'
  if (path.startsWith('/logs')) return 'logs'
  if (path.startsWith('/users') || path.startsWith('/rbac') || path.startsWith('/audit') || path.startsWith('/admin') || path.startsWith('/ai')) return 'system'
  return ''
}

// 从 localStorage 恢复菜单展开状态
const getInitialOpenKeys = (): string[] => {
  const saved = localStorage.getItem('menuOpenKeys')
  if (saved) {
    try {
      return JSON.parse(saved)
    } catch (e) {
      console.error('Failed to parse menuOpenKeys from localStorage:', e)
    }
  }
  // 如果没有保存的状态，返回当前路由的父菜单
  return [getParentKey(route.path)].filter(Boolean)
}

const openKeys = ref<string[]>(getInitialOpenKeys())

const userInfo = computed(() => {
  const info = localStorage.getItem('userInfo')
  return info ? JSON.parse(info) : null
})

// ==================== 权限控制 ====================

/**
 * 获取当前用户的权限列表
 * @returns 用户权限数组
 */
const getUserPermissions = (): string[] => {
  const user = userInfo.value
  if (!user) return []
  
  // 如果用户有 permissions 字段，直接返回
  if (user.permissions && Array.isArray(user.permissions)) {
    return user.permissions
  }
  
  // 如果用户有 roles 字段，根据角色返回权限
  if (user.roles && Array.isArray(user.roles)) {
    const roles = user.roles
    
    // 管理员拥有所有权限
    if (roles.includes('admin') || roles.includes('administrator')) {
      return ['*'] // 通配符表示所有权限
    }
    
    // 根据角色映射权限
    const rolePermissionMap: Record<string, string[]> = {
      'developer': [
        'pipeline:view', 'pipeline:create', 'pipeline:edit',
        'application:view', 'application:deploy',
        'k8s:view', 'logs:view'
      ],
      'operator': [
        'pipeline:view', 'application:view', 'application:deploy',
        'k8s:view', 'k8s:manage', 'healthcheck:view',
        'logs:view', 'alert:view', 'cost:view'
      ],
      'viewer': [
        'pipeline:view', 'application:view',
        'k8s:view', 'logs:view', 'alert:view'
      ]
    }
    
    // 合并所有角色的权限
    const permissions = new Set<string>()
    roles.forEach(role => {
      const rolePerms = rolePermissionMap[role] || []
      rolePerms.forEach(perm => permissions.add(perm))
    })
    
    return Array.from(permissions)
  }
  
  // 默认返回空数组（无权限）
  return []
}

/**
 * 检查用户是否有指定权限
 * @param requiredPermissions 需要的权限列表（可选）
 * @returns 是否有权限
 */
const hasPermission = (requiredPermissions?: string[]): boolean => {
  // 如果没有指定权限要求，默认允许访问（向后兼容）
  if (!requiredPermissions || requiredPermissions.length === 0) {
    return true
  }
  
  const userPermissions = getUserPermissions()
  
  // 如果没有用户信息或没有权限配置，默认显示所有菜单（向后兼容）
  if (userPermissions.length === 0) {
    return true
  }
  
  // 如果用户有通配符权限，允许访问所有功能
  if (userPermissions.includes('*')) {
    return true
  }
  
  // 检查用户是否拥有任一所需权限
  return requiredPermissions.some(required => {
    // 支持通配符匹配，如 'pipeline:*' 匹配 'pipeline:view', 'pipeline:create' 等
    if (required.endsWith(':*')) {
      const prefix = required.slice(0, -2)
      return userPermissions.some(perm => perm.startsWith(prefix + ':'))
    }
    return userPermissions.includes(required)
  })
}

// ==================== 菜单配置（带权限控制） ====================

interface MenuItemConfig {
  key: string
  icon?: any
  titleKey: string  // i18n key instead of hardcoded title
  path?: string
  children?: MenuItemConfig[]
  permission?: string[]
}

// 定义菜单配置（包含权限信息）- 使用 i18n key
const getMenuConfig = (): MenuItemConfig[] => [
  {
    key: '/dashboard',
    icon: DashboardOutlined,
    titleKey: 'menu.dashboard',
    path: '/dashboard'
    // 无 permission 字段，所有用户可见
  },
  {
    key: 'app',
    icon: AppstoreOutlined,
    titleKey: 'menu.app',
    permission: ['application:view'],
    children: [
      { key: '/applications', titleKey: 'menu.applications', path: '/applications' },
      {
        key: 'traffic',
        titleKey: 'menu.traffic',
        children: [
          { key: '/traffic/ratelimit', titleKey: 'menu.ratelimit', path: '/traffic/ratelimit' },
          { key: '/traffic/circuitbreaker', titleKey: 'menu.circuitbreaker', path: '/traffic/circuitbreaker' },
          { key: '/traffic/routing', titleKey: 'menu.routing', path: '/traffic/routing' },
          { key: '/traffic/loadbalance', titleKey: 'menu.loadbalance', path: '/traffic/loadbalance' },
          { key: '/traffic/timeout', titleKey: 'menu.timeout', path: '/traffic/timeout' },
          { key: '/traffic/mirror', titleKey: 'menu.mirror', path: '/traffic/mirror' },
          { key: '/traffic/fault', titleKey: 'menu.fault', path: '/traffic/fault' },
          { key: '/traffic/monitor', titleKey: 'menu.trafficMonitor', path: '/traffic/monitor' },
          { key: '/resilience', titleKey: 'menu.resilience', path: '/resilience', icon: ThunderboltOutlined }
        ]
      },
      { key: '/deploys', titleKey: 'menu.deploys', path: '/deploys' },
      { key: '/deploy/check', titleKey: 'menu.deployCheck', path: '/deploy/check' },
      { key: '/canary/list', titleKey: 'menu.canary', path: '/canary/list' },
      { key: '/bluegreen/list', titleKey: 'menu.bluegreen', path: '/bluegreen/list' }
    ]
  },
  {
    key: 'healthcheck',
    icon: HeartOutlined,
    titleKey: 'menu.healthcheck',
    permission: ['healthcheck:view'],
    children: [
      { key: '/healthcheck', titleKey: 'menu.serviceHealth', path: '/healthcheck' },
      { key: '/healthcheck/ssl-cert', titleKey: 'menu.sslCert', path: '/healthcheck/ssl-cert', icon: SafetyCertificateOutlined }
    ]
  },
  {
    key: 'pipeline',
    icon: RocketOutlined,
    titleKey: 'menu.pipeline',
    permission: ['pipeline:view'],
    children: [
      { key: '/pipeline/designer', titleKey: 'menu.pipelineDesigner', path: '/pipeline/designer', icon: LayoutOutlined, permission: ['pipeline:create'] },
      { key: '/pipeline/list', titleKey: 'menu.pipelineList', path: '/pipeline/list' },
      { key: '/pipeline/stats', titleKey: 'menu.pipelineStats', path: '/pipeline/stats' },
      { key: '/pipeline/templates', titleKey: 'menu.templates', path: '/pipeline/templates', icon: ShopOutlined },
      { key: '/pipeline/git-repos', titleKey: 'menu.gitRepos', path: '/pipeline/git-repos' },
      { key: '/pipeline/artifacts', titleKey: 'menu.artifacts', path: '/pipeline/artifacts' },
      { key: '/pipeline/builders', titleKey: 'menu.builders', path: '/pipeline/builders' },
      { key: '/pipeline/cache', titleKey: 'menu.buildCache', path: '/pipeline/cache', icon: DatabaseOutlined },
      { key: '/pipeline/stats/usage', titleKey: 'menu.buildStats', path: '/pipeline/stats/usage', icon: BarChartOutlined },
      { key: '/pipeline/quota', titleKey: 'menu.quota', path: '/pipeline/quota', icon: FundOutlined, permission: ['pipeline:manage'] },
      { key: '/pipeline/credentials', titleKey: 'menu.credentials', path: '/pipeline/credentials', icon: KeyOutlined, permission: ['pipeline:manage'] },
      { key: '/pipeline/variables', titleKey: 'menu.variables', path: '/pipeline/variables', icon: ControlOutlined, permission: ['pipeline:manage'] },
      { key: '/pipeline/notify', titleKey: 'menu.notify', path: '/pipeline/notify' }
    ]
  },
  {
    key: 'jenkins',
    icon: ThunderboltOutlined,
    titleKey: 'menu.jenkins',
    permission: ['jenkins:view'],
    children: [
      { key: '/jenkins/instances', titleKey: 'menu.jenkinsInstances', path: '/jenkins/instances' }
    ]
  },
  {
    key: 'k8s',
    icon: CloudOutlined,
    titleKey: 'menu.k8s',
    permission: ['k8s:view'],
    children: [
      { key: '/k8s/overview', titleKey: 'menu.k8sOverview', path: '/k8s/overview' },
      { key: '/k8s/clusters', titleKey: 'menu.k8sClusters', path: '/k8s/clusters' },
      { key: '/security/overview', titleKey: 'menu.securityOverview', path: '/security/overview' },
      { key: '/security/image-scan', titleKey: 'menu.imageScan', path: '/security/image-scan' },
      { key: '/security/config-check', titleKey: 'menu.configCheck', path: '/security/config-check' },
      { key: '/security/audit-log', titleKey: 'menu.auditLog', path: '/security/audit-log' }
    ]
  },
  {
    key: 'approval',
    icon: AuditOutlined,
    titleKey: 'menu.approval',
    permission: ['approval:view'],
    children: [
      { key: '/approval/pending', titleKey: 'menu.pending', path: '/approval/pending' },
      { key: '/approval/history', titleKey: 'menu.approvalHistory', path: '/approval/history' },
      { key: '/approval/chains', titleKey: 'menu.chains', path: '/approval/chains', permission: ['approval:manage'] },
      { key: '/approval/instances', titleKey: 'menu.instances', path: '/approval/instances' },
      { key: '/approval/rules', titleKey: 'menu.rules', path: '/approval/rules', permission: ['approval:manage'] },
      { key: '/approval/windows', titleKey: 'menu.windows', path: '/approval/windows', permission: ['approval:manage'] },
      { key: '/deploy/locks', titleKey: 'menu.deployLocks', path: '/deploy/locks' }
    ]
  },
  {
    key: 'message',
    icon: MessageOutlined,
    titleKey: 'menu.message',
    permission: ['message:view'],
    children: [
      { key: '/feishu/message', titleKey: 'menu.feishu', path: '/feishu/message' },
      { key: '/dingtalk/message', titleKey: 'menu.dingtalk', path: '/dingtalk/message' },
      { key: '/wechatwork/message', titleKey: 'menu.wechatwork', path: '/wechatwork/message' }
    ]
  },
  {
    key: 'oa',
    icon: FileTextOutlined,
    titleKey: 'menu.oa',
    permission: ['oa:view'],
    children: [
      { key: '/oa/data', titleKey: 'menu.oaData', path: '/oa/data' }
    ]
  },
  {
    key: 'alert',
    icon: AlertOutlined,
    titleKey: 'menu.alert',
    permission: ['alert:view'],
    children: [
      { key: '/alert/overview', titleKey: 'menu.alertOverview', path: '/alert/overview' },
      { key: '/alert/history', titleKey: 'menu.alertHistory', path: '/alert/history' },
      { key: '/alert/config', titleKey: 'menu.alertConfig', path: '/alert/config', permission: ['alert:manage'] },
      { key: '/alert/templates', titleKey: 'menu.alertTemplates', path: '/alert/templates', permission: ['alert:manage'] },
      { key: '/alert/gateway', titleKey: 'menu.alertGateway', path: '/alert/gateway' },
      { key: '/alert/silence', titleKey: 'menu.silence', path: '/alert/silence', permission: ['alert:manage'] },
      { key: '/alert/escalation', titleKey: 'menu.escalation', path: '/alert/escalation', permission: ['alert:manage'] }
    ]
  },
  {
    key: 'logs',
    icon: FileSearchOutlined,
    titleKey: 'menu.logs',
    permission: ['logs:view'],
    children: [
      { key: '/logs/center', titleKey: 'menu.logsCenter', path: '/logs/center' },
      { key: '/logs/search', titleKey: 'menu.logsSearch', path: '/logs/search' },
      { key: '/logs/stats', titleKey: 'menu.logsStats', path: '/logs/stats' },
      { key: '/logs/compare', titleKey: 'menu.logsCompare', path: '/logs/compare' },
      { key: '/logs/alerts', titleKey: 'menu.logsAlerts', path: '/logs/alerts' },
      { key: '/logs/bookmarks', titleKey: 'menu.logsBookmarks', path: '/logs/bookmarks' },
      { key: '/logs/export', titleKey: 'menu.logsExport', path: '/logs/export' }
    ]
  },
  {
    key: 'cost',
    icon: DollarOutlined,
    titleKey: 'menu.cost',
    permission: ['cost:view'],
    children: [
      { key: '/cost/overview', titleKey: 'menu.costOverview', path: '/cost/overview' },
      { key: '/cost/trend', titleKey: 'menu.costTrend', path: '/cost/trend' },
      { key: '/cost/comparison', titleKey: 'menu.costComparison', path: '/cost/comparison' },
      { key: '/cost/analysis', titleKey: 'menu.costAnalysis', path: '/cost/analysis' },
      { key: '/cost/waste', titleKey: 'menu.costWaste', path: '/cost/waste' },
      { key: '/cost/suggestions', titleKey: 'menu.costSuggestions', path: '/cost/suggestions' },
      { key: '/cost/alerts', titleKey: 'menu.costAlerts', path: '/cost/alerts' },
      { key: '/cost/budget', titleKey: 'menu.costBudget', path: '/cost/budget', permission: ['cost:manage'] },
      { key: '/cost/config', titleKey: 'menu.costConfig', path: '/cost/config', permission: ['cost:manage'] }
    ]
  },
  {
    key: 'system',
    icon: SettingOutlined,
    titleKey: 'menu.system',
    permission: ['system:view'],
    children: [
      { key: '/users', titleKey: 'menu.users', path: '/users', permission: ['system:manage'] },
      { key: '/rbac/roles', titleKey: 'menu.roles', path: '/rbac/roles', permission: ['system:manage'] },
      { key: '/audit/logs', titleKey: 'menu.auditLogs', path: '/audit/logs', permission: ['system:manage'] },
      { key: '/ai/knowledge', titleKey: 'menu.aiKnowledge', path: '/ai/knowledge' },
      { key: '/ai/config', titleKey: 'menu.aiConfig', path: '/ai/config', permission: ['system:manage'] }
    ]
  }
]

/**
 * 递归过滤菜单项，只保留有权限的菜单
 * @param items 菜单配置数组
 * @returns 过滤后的菜单配置数组
 */
const filterMenuByPermission = (items: MenuItemConfig[]): MenuItemConfig[] => {
  return items.filter(item => {
    // 检查当前菜单项权限
    if (!hasPermission(item.permission)) {
      return false
    }
    
    // 如果有子菜单，递归过滤
    if (item.children && item.children.length > 0) {
      item.children = filterMenuByPermission(item.children)
      // 如果过滤后没有子菜单了，也不显示父菜单
      if (item.children.length === 0) {
        return false
      }
    }
    
    return true
  })
}

// 计算过滤后的菜单配置
const filteredMenuConfig = computed(() => {
  return filterMenuByPermission(JSON.parse(JSON.stringify(getMenuConfig())))
})

// 监听路由变化，更新选中状态和自动展开父菜单
watch(() => route.path, (newPath) => {
  selectedKeys.value = [newPath]
  const parentKey = getParentKey(newPath)
  if (parentKey && !openKeys.value.includes(parentKey)) {
    openKeys.value = [...openKeys.value, parentKey]
  }
})

// 监听菜单展开状态变化，保存到 localStorage
watch(openKeys, (newKeys) => {
  localStorage.setItem('menuOpenKeys', JSON.stringify(newKeys))
}, { deep: true })

// 监听窗口大小变化，移动端自动折叠侧边栏
watch(isMobile, (mobile) => {
  if (mobile) {
    collapsed.value = true
  }
}, { immediate: true })

const handleMenuClick = ({ key }: { key: string }) => {
  router.push(key)
  // 移动端点击菜单后自动折叠
  if (isMobile.value) {
    collapsed.value = true
  }
}

const handleDropdownClick = ({ key }: { key: string }) => {
  if (key === 'profile') {
    router.push('/profile')
  } else if (key === 'logout') {
    handleLogout()
  }
}

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('userInfo')
  message.success('已退出登录')
  router.push('/login')
}
</script>

<style scoped>
.logo {
  height: 64px;
  line-height: 64px;
  text-align: center;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  white-space: nowrap;
  overflow: hidden;
}

.trigger {
  font-size: 18px;
  line-height: 64px;
  padding: 0 24px;
  cursor: pointer;
  transition: color 0.3s;
}

.trigger:hover {
  color: #1890ff;
}

.header-right {
  display: flex;
  align-items: center;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .logo {
    font-size: 16px;
  }
  
  .trigger {
    padding: 0 16px;
  }
  
  :deep(.ant-layout-header) {
    padding: 0 16px !important;
  }
  
  :deep(.ant-layout-content) {
    margin: 8px !important;
  }
  
  :deep(.ant-layout-content > div) {
    padding: 16px !important;
  }
  
  :deep(.ant-breadcrumb) {
    margin-bottom: 8px !important;
  }
  
  /* 移动端侧边栏覆盖在内容上方 */
  :deep(.ant-layout-sider) {
    position: fixed !important;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 999;
  }
  
  /* 侧边栏折叠时不占用空间 */
  :deep(.ant-layout-sider-collapsed) {
    transform: translateX(-100%);
  }
  
  /* 侧边栏展开时显示遮罩 */
  :deep(.ant-layout-sider:not(.ant-layout-sider-collapsed))::before {
    content: '';
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.45);
    z-index: -1;
  }
}

/* 平板设备适配 */
@media (min-width: 769px) and (max-width: 1024px) {
  .logo {
    font-size: 16px;
  }
  
  :deep(.ant-layout-sider) {
    width: 180px !important;
    min-width: 180px !important;
    max-width: 180px !important;
  }
}
</style>

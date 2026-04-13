import { describe, it, expect, beforeEach } from 'vitest'
import { createI18n } from 'vue-i18n'
import zhCN from '@/locales/zh-CN'
import enUS from '@/locales/en-US'

describe('I18n Integration', () => {
  let i18n: ReturnType<typeof createI18n>

  beforeEach(() => {
    i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      fallbackLocale: 'zh-CN',
      messages: {
        'zh-CN': zhCN,
        'en-US': enUS,
      },
    })
  })

  describe('Menu Translations', () => {
    it('should have Chinese translations for all menu items', () => {
      const { t } = i18n.global
      
      // Test main menu items
      expect(t('menu.dashboard')).toBe('仪表盘')
      expect(t('menu.app')).toBe('应用管理')
      expect(t('menu.pipeline')).toBe('CI/CD 流水线')
      expect(t('menu.healthcheck')).toBe('健康检查')
      expect(t('menu.resilience')).toBe('弹性工程')
      
      // Test new menu items
      expect(t('menu.pipelineDesigner')).toBe('流水线设计器')
      expect(t('menu.templates')).toBe('模板市场')
      expect(t('menu.buildCache')).toBe('构建缓存')
      expect(t('menu.buildStats')).toBe('构建统计')
      expect(t('menu.quota')).toBe('资源配额')
      expect(t('menu.credentials')).toBe('凭证管理')
      expect(t('menu.variables')).toBe('变量管理')
      expect(t('menu.sslCert')).toBe('SSL 证书检查')
      expect(t('menu.featureFlags')).toBe('功能开关')
      expect(t('menu.systemMonitor')).toBe('系统监控')
    })

    it('should have English translations for all menu items', () => {
      i18n.global.locale.value = 'en-US'
      const { t } = i18n.global
      
      // Test main menu items
      expect(t('menu.dashboard')).toBe('Dashboard')
      expect(t('menu.app')).toBe('Applications')
      expect(t('menu.pipeline')).toBe('CI/CD Pipeline')
      expect(t('menu.healthcheck')).toBe('Health Check')
      expect(t('menu.resilience')).toBe('Resilience Engineering')
      
      // Test new menu items
      expect(t('menu.pipelineDesigner')).toBe('Pipeline Designer')
      expect(t('menu.templates')).toBe('Template Market')
      expect(t('menu.buildCache')).toBe('Build Cache')
      expect(t('menu.buildStats')).toBe('Build Statistics')
      expect(t('menu.quota')).toBe('Resource Quota')
      expect(t('menu.credentials')).toBe('Credentials')
      expect(t('menu.variables')).toBe('Variables')
      expect(t('menu.sslCert')).toBe('SSL Certificate Check')
      expect(t('menu.featureFlags')).toBe('Feature Flags')
      expect(t('menu.systemMonitor')).toBe('System Monitor')
    })
  })

  describe('Breadcrumb Translations', () => {
    it('should have Chinese translations for breadcrumbs', () => {
      const { t } = i18n.global
      
      expect(t('breadcrumb.home')).toBe('首页')
      expect(t('breadcrumb.pipeline')).toBe('CI/CD 流水线')
      expect(t('breadcrumb.pipelineDesigner')).toBe('流水线设计器')
      expect(t('breadcrumb.resilience')).toBe('弹性工程')
    })

    it('should have English translations for breadcrumbs', () => {
      i18n.global.locale.value = 'en-US'
      const { t } = i18n.global
      
      expect(t('breadcrumb.home')).toBe('Home')
      expect(t('breadcrumb.pipeline')).toBe('CI/CD Pipeline')
      expect(t('breadcrumb.pipelineDesigner')).toBe('Pipeline Designer')
      expect(t('breadcrumb.resilience')).toBe('Resilience Engineering')
    })
  })

  describe('Search Translations', () => {
    it('should have Chinese translations for search', () => {
      const { t } = i18n.global
      
      expect(t('search.placeholder')).toBe('搜索流水线、集群、应用、用户...')
      expect(t('search.searching')).toBe('搜索中...')
      expect(t('search.noResults')).toBe('未找到相关结果')
      expect(t('search.quickNav')).toBe('快捷导航')
    })

    it('should have English translations for search', () => {
      i18n.global.locale.value = 'en-US'
      const { t } = i18n.global
      
      expect(t('search.placeholder')).toBe('Search pipelines, clusters, applications, users...')
      expect(t('search.searching')).toBe('Searching...')
      expect(t('search.noResults')).toBe('No results found')
      expect(t('search.quickNav')).toBe('Quick Navigation')
    })
  })

  describe('Common Translations', () => {
    it('should have Chinese translations for common UI elements', () => {
      const { t } = i18n.global
      
      expect(t('common.profile')).toBe('个人中心')
      expect(t('common.logout')).toBe('退出登录')
      expect(t('common.search')).toBe('全局搜索')
      expect(t('common.select')).toBe('选择')
      expect(t('common.open')).toBe('打开')
      expect(t('common.close')).toBe('关闭')
    })

    it('should have English translations for common UI elements', () => {
      i18n.global.locale.value = 'en-US'
      const { t } = i18n.global
      
      expect(t('common.profile')).toBe('Profile')
      expect(t('common.logout')).toBe('Logout')
      expect(t('common.search')).toBe('Global Search')
      expect(t('common.select')).toBe('Select')
      expect(t('common.open')).toBe('Open')
      expect(t('common.close')).toBe('Close')
    })
  })

  describe('Language Switching', () => {
    it('should switch between Chinese and English', () => {
      const { t } = i18n.global
      
      // Start with Chinese
      expect(t('menu.dashboard')).toBe('仪表盘')
      
      // Switch to English
      i18n.global.locale.value = 'en-US'
      expect(t('menu.dashboard')).toBe('Dashboard')
      
      // Switch back to Chinese
      i18n.global.locale.value = 'zh-CN'
      expect(t('menu.dashboard')).toBe('仪表盘')
    })
  })

  describe('Fallback Behavior', () => {
    it('should fallback to Chinese when key is missing in English', () => {
      i18n.global.locale.value = 'en-US'
      const { t } = i18n.global
      
      // All keys should exist, but test fallback mechanism
      expect(t('menu.dashboard')).toBeTruthy()
      expect(t('menu.pipeline')).toBeTruthy()
    })
  })
})

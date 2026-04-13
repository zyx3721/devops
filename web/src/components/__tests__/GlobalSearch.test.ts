import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import GlobalSearch from '../GlobalSearch.vue'
import { createRouter, createMemoryHistory } from 'vue-router'

// Mock the API request module
vi.mock('@/services/api', () => ({
  default: {
    get: vi.fn()
  }
}))

describe('GlobalSearch', () => {
  let router: any

  beforeEach(() => {
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div>Home</div>' } },
        { path: '/resilience', component: { template: '<div>Resilience</div>' } },
        { path: '/pipeline/designer', component: { template: '<div>Designer</div>' } },
        { path: '/pipeline/templates', component: { template: '<div>Templates</div>' } },
        { path: '/pipeline/cache', component: { template: '<div>Cache</div>' } },
        { path: '/pipeline/stats/usage', component: { template: '<div>Stats</div>' } },
        { path: '/pipeline/quota', component: { template: '<div>Quota</div>' } },
        { path: '/pipeline/credentials', component: { template: '<div>Credentials</div>' } },
        { path: '/pipeline/variables', component: { template: '<div>Variables</div>' } },
        { path: '/healthcheck', component: { template: '<div>Health</div>' } },
        { path: '/healthcheck/ssl-cert', component: { template: '<div>SSL</div>' } },
        { path: '/feature-flags', component: { template: '<div>Flags</div>' } },
        { path: '/system/monitor', component: { template: '<div>Monitor</div>' } }
      ]
    })
  })

  it('应该渲染搜索按钮', () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })
    expect(wrapper.find('button').exists()).toBe(true)
  })

  it('应该在快捷导航中包含弹性工程', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    // 打开搜索弹窗
    await wrapper.find('button').trigger('click')
    await nextTick()

    // 检查快捷导航中是否包含弹性工程
    const navItems = wrapper.findAll('.nav-item')
    const labels = navItems.map(item => item.text())
    expect(labels).toContain('弹性工程')
  })

  it('应该能够搜索菜单项 - 弹性工程', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    // 打开搜索弹窗
    await wrapper.find('button').trigger('click')
    await nextTick()

    // 输入搜索关键词
    const input = wrapper.find('input')
    await input.setValue('弹性')
    await nextTick()
    
    // 等待搜索完成（300ms debounce + 一些额外时间）
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    // 应该显示搜索结果
    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 流水线设计器', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('设计器')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 模板市场', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('模板')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 构建缓存', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('缓存')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 构建统计', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('统计')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 资源配额', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('配额')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 凭证管理', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('凭证')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 变量管理', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('变量')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - SSL证书检查', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('ssl')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 功能开关', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('功能开关')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该能够搜索菜单项 - 系统监控', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('监控')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该支持英文关键词搜索', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('resilience')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('应该支持模糊搜索', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.setValue('cache')
    await nextTick()
    
    await new Promise(resolve => setTimeout(resolve, 400))
    await nextTick()

    const searchResults = wrapper.find('.search-results')
    expect(searchResults.exists()).toBe(true)
  })

  it('空搜索应该显示快捷导航', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const quickNav = wrapper.find('.quick-nav')
    expect(quickNav.exists()).toBe(true)
  })

  it('应该在ESC键时关闭弹窗', async () => {
    const wrapper = mount(GlobalSearch, {
      global: {
        plugins: [router]
      }
    })

    await wrapper.find('button').trigger('click')
    await nextTick()

    const input = wrapper.find('input')
    await input.trigger('keydown', { key: 'Escape' })
    await nextTick()

    // 弹窗应该关闭
    expect(wrapper.vm.showModal).toBe(false)
  })
})

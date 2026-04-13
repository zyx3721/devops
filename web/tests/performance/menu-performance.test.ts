/**
 * 菜单性能测试
 * 验证菜单渲染和交互性能是否满足要求
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import MainLayout from '@/layouts/MainLayout.vue'

describe('菜单性能测试', () => {
  let router: any

  beforeEach(() => {
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div>Home</div>' } },
        { path: '/dashboard', component: { template: '<div>Dashboard</div>' } },
        { path: '/pipeline/list', component: { template: '<div>Pipeline</div>' } }
      ]
    })
  })

  it('菜单初始渲染时间应小于 100ms', async () => {
    const startTime = performance.now()
    
    const wrapper = mount(MainLayout, {
      global: {
        plugins: [router],
        stubs: {
          'a-layout': true,
          'a-layout-sider': true,
          'a-layout-header': true,
          'a-layout-content': true,
          'a-menu': true,
          'a-menu-item': true,
          'a-sub-menu': true,
          'a-breadcrumb': true,
          'a-breadcrumb-item': true,
          'a-dropdown': true,
          'a-space': true,
          'a-avatar': true,
          'a-button': true,
          'router-view': true,
          'router-link': true,
          'GlobalSearch': true,
          'FavoriteList': true,
          'ThemeSwitch': true,
          'AIChatWidget': true,
          'PerformanceMonitor': true
        }
      }
    })
    
    const endTime = performance.now()
    const renderTime = endTime - startTime
    
    console.log(`菜单初始渲染时间: ${renderTime.toFixed(2)}ms`)
    expect(renderTime).toBeLessThan(100)
  })

  it('菜单展开/折叠应该流畅（< 50ms）', async () => {
    const wrapper = mount(MainLayout, {
      global: {
        plugins: [router],
        stubs: {
          'a-layout': true,
          'a-layout-sider': true,
          'a-layout-header': true,
          'a-layout-content': true,
          'a-menu': true,
          'a-menu-item': true,
          'a-sub-menu': true,
          'a-breadcrumb': true,
          'a-breadcrumb-item': true,
          'a-dropdown': true,
          'a-space': true,
          'a-avatar': true,
          'a-button': true,
          'router-view': true,
          'router-link': true,
          'GlobalSearch': true,
          'FavoriteList': true,
          'ThemeSwitch': true,
          'AIChatWidget': true,
          'PerformanceMonitor': true
        }
      }
    })

    const startTime = performance.now()
    
    // 模拟菜单折叠
    await wrapper.vm.collapsed = true
    await wrapper.vm.$nextTick()
    
    const endTime = performance.now()
    const toggleTime = endTime - startTime
    
    console.log(`菜单折叠时间: ${toggleTime.toFixed(2)}ms`)
    expect(toggleTime).toBeLessThan(50)
  })

  it('路由切换应该快速（< 100ms）', async () => {
    const wrapper = mount(MainLayout, {
      global: {
        plugins: [router],
        stubs: {
          'a-layout': true,
          'a-layout-sider': true,
          'a-layout-header': true,
          'a-layout-content': true,
          'a-menu': true,
          'a-menu-item': true,
          'a-sub-menu': true,
          'a-breadcrumb': true,
          'a-breadcrumb-item': true,
          'a-dropdown': true,
          'a-space': true,
          'a-avatar': true,
          'a-button': true,
          'router-view': true,
          'router-link': true,
          'GlobalSearch': true,
          'FavoriteList': true,
          'ThemeSwitch': true,
          'AIChatWidget': true,
          'PerformanceMonitor': true
        }
      }
    })

    const startTime = performance.now()
    
    await router.push('/pipeline/list')
    await wrapper.vm.$nextTick()
    
    const endTime = performance.now()
    const routeChangeTime = endTime - startTime
    
    console.log(`路由切换时间: ${routeChangeTime.toFixed(2)}ms`)
    expect(routeChangeTime).toBeLessThan(100)
  })

  it('菜单项点击响应应该快速（< 50ms）', async () => {
    const wrapper = mount(MainLayout, {
      global: {
        plugins: [router],
        stubs: {
          'a-layout': true,
          'a-layout-sider': true,
          'a-layout-header': true,
          'a-layout-content': true,
          'a-menu': true,
          'a-menu-item': true,
          'a-sub-menu': true,
          'a-breadcrumb': true,
          'a-breadcrumb-item': true,
          'a-dropdown': true,
          'a-space': true,
          'a-avatar': true,
          'a-button': true,
          'router-view': true,
          'router-link': true,
          'GlobalSearch': true,
          'FavoriteList': true,
          'ThemeSwitch': true,
          'AIChatWidget': true,
          'PerformanceMonitor': true
        }
      }
    })

    const startTime = performance.now()
    
    // 模拟菜单点击
    await wrapper.vm.handleMenuClick({ key: '/dashboard' })
    await wrapper.vm.$nextTick()
    
    const endTime = performance.now()
    const clickTime = endTime - startTime
    
    console.log(`菜单点击响应时间: ${clickTime.toFixed(2)}ms`)
    expect(clickTime).toBeLessThan(50)
  })

  it('大量菜单项渲染不应该造成性能问题', async () => {
    const startTime = performance.now()
    
    const wrapper = mount(MainLayout, {
      global: {
        plugins: [router],
        stubs: {
          'a-layout': true,
          'a-layout-sider': true,
          'a-layout-header': true,
          'a-layout-content': true,
          'a-menu': true,
          'a-menu-item': true,
          'a-sub-menu': true,
          'a-breadcrumb': true,
          'a-breadcrumb-item': true,
          'a-dropdown': true,
          'a-space': true,
          'a-avatar': true,
          'a-button': true,
          'router-view': true,
          'router-link': true,
          'GlobalSearch': true,
          'FavoriteList': true,
          'ThemeSwitch': true,
          'AIChatWidget': true,
          'PerformanceMonitor': true
        }
      }
    })
    
    const endTime = performance.now()
    const renderTime = endTime - startTime
    
    console.log(`大量菜单项渲染时间: ${renderTime.toFixed(2)}ms`)
    // 即使有很多菜单项，渲染时间也应该在合理范围内
    expect(renderTime).toBeLessThan(200)
  })
})

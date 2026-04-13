import type { Component } from 'vue'

/**
 * 菜单项接口
 * 定义侧边栏菜单项的数据结构
 */
export interface MenuItem {
  /** 唯一标识 */
  key: string
  
  /** 图标组件 */
  icon?: Component
  
  /** 显示文本 */
  title: string
  
  /** 路由路径 */
  path?: string
  
  /** 子菜单 */
  children?: MenuItem[]
  
  /** 权限要求 */
  permission?: string[]
  
  /** 徽章（如未读数量） */
  badge?: number | string
  
  /** 是否禁用 */
  disabled?: boolean
}

/**
 * 面包屑配置接口
 * 定义面包屑导航的配置结构
 */
export interface BreadcrumbConfig {
  /** 显示文本 */
  title: string
  
  /** 父级标识 */
  parent?: string
  
  /** 路由路径（可选，用于可点击的面包屑） */
  path?: string
}

/**
 * 菜单状态接口
 * 定义菜单的状态管理结构
 */
export interface MenuState {
  /** 选中的菜单项 */
  selectedKeys: string[]
  
  /** 展开的子菜单 */
  openKeys: string[]
  
  /** 是否折叠 */
  collapsed: boolean
}

/**
 * 搜索项接口
 * 定义全局搜索中的菜单项结构
 */
export interface SearchItem {
  /** 唯一标识 */
  key: string
  
  /** 显示文本 */
  title: string
  
  /** 路由路径 */
  path: string
  
  /** 分类 */
  category: string
  
  /** 搜索关键词 */
  keywords: string[]
  
  /** 图标组件（可选） */
  icon?: Component
}

import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'

const STORAGE_KEY = 'devops_theme'

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>('light')
  const isDark = ref(false)

  // 从 localStorage 加载
  const loadFromStorage = () => {
    if (typeof window === 'undefined') return
    try {
      const stored = localStorage.getItem(STORAGE_KEY) as ThemeMode
      if (stored && ['light', 'dark', 'auto'].includes(stored)) {
        mode.value = stored
      }
    } catch (e) {
      console.error('Failed to load theme', e)
    }
  }

  // 保存到 localStorage
  const saveToStorage = () => {
    if (typeof window === 'undefined') return
    try {
      localStorage.setItem(STORAGE_KEY, mode.value)
    } catch (e) {
      console.error('Failed to save theme', e)
    }
  }

  // 应用主题
  const applyTheme = () => {
    if (typeof window === 'undefined' || typeof document === 'undefined') return
    
    let dark = false
    if (mode.value === 'dark') {
      dark = true
    } else if (mode.value === 'auto') {
      dark = window.matchMedia('(prefers-color-scheme: dark)').matches
    }
    
    isDark.value = dark
    document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light')
    
    // 更新 body 类名
    if (dark) {
      document.body.classList.add('dark-mode')
    } else {
      document.body.classList.remove('dark-mode')
    }
  }

  // 切换主题
  const setMode = (newMode: ThemeMode) => {
    mode.value = newMode
    saveToStorage()
    applyTheme()
  }

  const toggleDark = () => {
    setMode(isDark.value ? 'light' : 'dark')
  }

  // 初始化
  const init = () => {
    if (typeof window === 'undefined') return
    
    loadFromStorage()
    applyTheme()
    
    // 监听系统主题变化
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    mediaQuery.addEventListener('change', () => {
      if (mode.value === 'auto') {
        applyTheme()
      }
    })
  }

  // 延迟初始化，确保 DOM 已就绪
  if (typeof window !== 'undefined') {
    init()
  }

  return {
    mode,
    isDark,
    setMode,
    toggleDark,
    init
  }
})

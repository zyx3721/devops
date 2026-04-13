import { ref, onMounted, onUnmounted } from 'vue'

/**
 * 断点定义
 */
export const breakpoints = {
  xs: 480,
  sm: 576,
  md: 768,
  lg: 992,
  xl: 1200,
  xxl: 1600,
}

/**
 * 屏幕尺寸类型
 */
export type ScreenSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl'

/**
 * 获取当前屏幕尺寸
 */
export function getScreenSize(): ScreenSize {
  const width = window.innerWidth
  
  if (width < breakpoints.xs) return 'xs'
  if (width < breakpoints.sm) return 'sm'
  if (width < breakpoints.md) return 'md'
  if (width < breakpoints.lg) return 'lg'
  if (width < breakpoints.xl) return 'xl'
  return 'xxl'
}

/**
 * 判断是否为移动端
 */
export function isMobile(): boolean {
  return window.innerWidth < breakpoints.md
}

/**
 * 判断是否为平板
 */
export function isTablet(): boolean {
  const width = window.innerWidth
  return width >= breakpoints.md && width < breakpoints.lg
}

/**
 * 判断是否为桌面端
 */
export function isDesktop(): boolean {
  return window.innerWidth >= breakpoints.lg
}

/**
 * 响应式 Hook - 监听屏幕尺寸变化
 */
export function useResponsive() {
  const screenSize = ref<ScreenSize>(getScreenSize())
  const mobile = ref(isMobile())
  const tablet = ref(isTablet())
  const desktop = ref(isDesktop())

  const updateSize = () => {
    screenSize.value = getScreenSize()
    mobile.value = isMobile()
    tablet.value = isTablet()
    desktop.value = isDesktop()
  }

  onMounted(() => {
    window.addEventListener('resize', updateSize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', updateSize)
  })

  return {
    screenSize,
    mobile,
    tablet,
    desktop,
    isMobile: () => mobile.value,
    isTablet: () => tablet.value,
    isDesktop: () => desktop.value,
  }
}

/**
 * 根据屏幕尺寸返回不同的值
 */
export function responsive<T>(values: {
  xs?: T
  sm?: T
  md?: T
  lg?: T
  xl?: T
  xxl?: T
  default: T
}): T {
  const size = getScreenSize()
  return values[size] ?? values.default
}

/**
 * 获取响应式列数
 */
export function getResponsiveColumns(config?: {
  xs?: number
  sm?: number
  md?: number
  lg?: number
  xl?: number
  xxl?: number
}): number {
  const defaultConfig = {
    xs: 1,
    sm: 2,
    md: 3,
    lg: 4,
    xl: 6,
    xxl: 6,
  }
  
  const finalConfig = { ...defaultConfig, ...config }
  const size = getScreenSize()
  
  return finalConfig[size]
}

/**
 * 获取响应式间距
 */
export function getResponsiveGutter(): number {
  return isMobile() ? 12 : 16
}

/**
 * 获取响应式卡片大小
 */
export function getResponsiveCardSize(): 'small' | 'default' | 'large' {
  if (isMobile()) return 'small'
  if (isTablet()) return 'default'
  return 'default'
}

/**
 * 获取响应式表格大小
 */
export function getResponsiveTableSize(): 'small' | 'middle' | 'large' {
  if (isMobile()) return 'small'
  return 'middle'
}

/**
 * 获取响应式模态框宽度
 */
export function getResponsiveModalWidth(defaultWidth: number | string): number | string {
  if (isMobile()) return '100%'
  return defaultWidth
}

/**
 * 获取响应式抽屉宽度
 */
export function getResponsiveDrawerWidth(defaultWidth: number): number {
  if (isMobile()) return window.innerWidth
  if (isTablet()) return Math.min(defaultWidth, window.innerWidth * 0.8)
  return defaultWidth
}

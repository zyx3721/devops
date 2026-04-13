/**
 * 性能监控工具
 * 用于测量首屏加载时间、路由切换性能等指标
 */

export interface PerformanceMetrics {
  // 首屏加载时间
  firstScreenLoadTime: number
  // 路由切换时间
  routeChangeTime: number
  // DOM 渲染时间
  domRenderTime: number
  // 资源加载时间
  resourceLoadTime: number
}

/**
 * 测量首屏加载时间
 */
export function measureFirstScreenLoad(): number {
  if (typeof window === 'undefined' || !window.performance) {
    return 0
  }

  const perfData = window.performance.timing
  const loadTime = perfData.loadEventEnd - perfData.navigationStart
  
  console.log('首屏加载性能指标:')
  console.log(`- DNS 查询: ${perfData.domainLookupEnd - perfData.domainLookupStart}ms`)
  console.log(`- TCP 连接: ${perfData.connectEnd - perfData.connectStart}ms`)
  console.log(`- 请求响应: ${perfData.responseEnd - perfData.requestStart}ms`)
  console.log(`- DOM 解析: ${perfData.domComplete - perfData.domLoading}ms`)
  console.log(`- 总加载时间: ${loadTime}ms`)
  
  return loadTime
}

/**
 * 测量路由切换性能
 */
export function measureRouteChange(routeName: string): () => void {
  const startTime = performance.now()
  
  return () => {
    const endTime = performance.now()
    const duration = endTime - startTime
    console.log(`路由切换到 ${routeName} 耗时: ${duration.toFixed(2)}ms`)
    
    // 如果超过 100ms，记录警告
    if (duration > 100) {
      console.warn(`⚠️ 路由切换较慢: ${routeName} (${duration.toFixed(2)}ms)`)
    }
    
    return duration
  }
}

/**
 * 测量组件渲染性能
 */
export function measureComponentRender(componentName: string): () => void {
  const startTime = performance.now()
  
  return () => {
    const endTime = performance.now()
    const duration = endTime - startTime
    console.log(`组件 ${componentName} 渲染耗时: ${duration.toFixed(2)}ms`)
    
    return duration
  }
}

/**
 * 获取页面性能指标
 */
export function getPerformanceMetrics(): PerformanceMetrics | null {
  if (typeof window === 'undefined' || !window.performance) {
    return null
  }

  const perfData = window.performance.timing
  
  return {
    firstScreenLoadTime: perfData.loadEventEnd - perfData.navigationStart,
    routeChangeTime: 0, // 需要在路由切换时动态测量
    domRenderTime: perfData.domComplete - perfData.domLoading,
    resourceLoadTime: perfData.loadEventEnd - perfData.fetchStart
  }
}

/**
 * 监控长任务（超过 50ms 的任务）
 */
export function monitorLongTasks(): void {
  if (typeof window === 'undefined' || !('PerformanceObserver' in window)) {
    return
  }

  try {
    const observer = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.duration > 50) {
          console.warn(`⚠️ 检测到长任务: ${entry.duration.toFixed(2)}ms`, entry)
        }
      }
    })
    
    observer.observe({ entryTypes: ['longtask'] })
  } catch (e) {
    console.log('浏览器不支持 longtask 监控')
  }
}

/**
 * 监控 FPS（帧率）
 */
export function monitorFPS(callback: (fps: number) => void): () => void {
  let lastTime = performance.now()
  let frames = 0
  let rafId: number

  function measureFPS() {
    frames++
    const currentTime = performance.now()
    
    if (currentTime >= lastTime + 1000) {
      const fps = Math.round((frames * 1000) / (currentTime - lastTime))
      callback(fps)
      
      // 如果 FPS 低于 30，记录警告
      if (fps < 30) {
        console.warn(`⚠️ FPS 较低: ${fps}`)
      }
      
      frames = 0
      lastTime = currentTime
    }
    
    rafId = requestAnimationFrame(measureFPS)
  }
  
  rafId = requestAnimationFrame(measureFPS)
  
  // 返回停止监控的函数
  return () => {
    cancelAnimationFrame(rafId)
  }
}

/**
 * 检查是否满足性能要求
 */
export function checkPerformanceRequirements(): {
  passed: boolean
  metrics: {
    firstScreenLoad: { value: number; passed: boolean; requirement: string }
    routeChange: { value: number; passed: boolean; requirement: string }
  }
} {
  const metrics = getPerformanceMetrics()
  
  if (!metrics) {
    return {
      passed: false,
      metrics: {
        firstScreenLoad: { value: 0, passed: false, requirement: '< 2000ms' },
        routeChange: { value: 0, passed: false, requirement: '< 100ms' }
      }
    }
  }
  
  const firstScreenLoadPassed = metrics.firstScreenLoadTime < 2000
  const routeChangePassed = metrics.routeChangeTime < 100
  
  return {
    passed: firstScreenLoadPassed && routeChangePassed,
    metrics: {
      firstScreenLoad: {
        value: metrics.firstScreenLoadTime,
        passed: firstScreenLoadPassed,
        requirement: '< 2000ms'
      },
      routeChange: {
        value: metrics.routeChangeTime,
        passed: routeChangePassed,
        requirement: '< 100ms'
      }
    }
  }
}

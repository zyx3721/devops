import { App } from 'vue'
import { message, notification } from 'ant-design-vue'
import router from '@/router'

/**
 * 错误类型
 */
export enum ErrorType {
  NETWORK = 'NETWORK',
  SERVER = 'SERVER',
  PERMISSION = 'PERMISSION',
  VALIDATION = 'VALIDATION',
  BUSINESS = 'BUSINESS',
  UNKNOWN = 'UNKNOWN',
}

/**
 * 错误信息接口
 */
export interface ErrorInfo {
  type: ErrorType
  code?: number | string
  message: string
  detail?: any
  timestamp: number
}

/**
 * 错误处理器类
 */
class ErrorHandler {
  private errorLog: ErrorInfo[] = []
  private maxLogSize = 100

  /**
   * 处理错误
   */
  handle(error: any, context?: string) {
    const errorInfo = this.parseError(error)
    this.logError(errorInfo, context)
    this.showError(errorInfo)
    this.reportError(errorInfo, context)
  }

  /**
   * 解析错误
   */
  private parseError(error: any): ErrorInfo {
    const timestamp = Date.now()

    // 网络错误
    if (error.message === 'Network Error' || !error.response) {
      return {
        type: ErrorType.NETWORK,
        message: '网络连接失败，请检查网络设置',
        detail: error,
        timestamp,
      }
    }

    // HTTP 错误
    if (error.response) {
      const { status, data } = error.response

      // 401 未授权
      if (status === 401) {
        return {
          type: ErrorType.PERMISSION,
          code: 401,
          message: '登录已过期，请重新登录',
          detail: data,
          timestamp,
        }
      }

      // 403 禁止访问
      if (status === 403) {
        return {
          type: ErrorType.PERMISSION,
          code: 403,
          message: '没有权限访问此资源',
          detail: data,
          timestamp,
        }
      }

      // 404 未找到
      if (status === 404) {
        return {
          type: ErrorType.SERVER,
          code: 404,
          message: '请求的资源不存在',
          detail: data,
          timestamp,
        }
      }

      // 422 验证错误
      if (status === 422) {
        return {
          type: ErrorType.VALIDATION,
          code: 422,
          message: data?.message || '数据验证失败',
          detail: data,
          timestamp,
        }
      }

      // 500 服务器错误
      if (status >= 500) {
        return {
          type: ErrorType.SERVER,
          code: status,
          message: '服务器错误，请稍后重试',
          detail: data,
          timestamp,
        }
      }

      // 业务错误
      return {
        type: ErrorType.BUSINESS,
        code: data?.code || status,
        message: data?.message || data?.msg || '操作失败',
        detail: data,
        timestamp,
      }
    }

    // 其他错误
    return {
      type: ErrorType.UNKNOWN,
      message: error.message || '未知错误',
      detail: error,
      timestamp,
    }
  }

  /**
   * 记录错误日志
   */
  private logError(errorInfo: ErrorInfo, context?: string) {
    console.error('[Error Handler]', {
      ...errorInfo,
      context,
    })

    // 添加到错误日志
    this.errorLog.push(errorInfo)

    // 限制日志大小
    if (this.errorLog.length > this.maxLogSize) {
      this.errorLog.shift()
    }
  }

  /**
   * 显示错误提示
   */
  private showError(errorInfo: ErrorInfo) {
    const { type, message: msg } = errorInfo

    switch (type) {
      case ErrorType.NETWORK:
        notification.error({
          message: '网络错误',
          description: msg,
          duration: 5,
        })
        break

      case ErrorType.PERMISSION:
        if (errorInfo.code === 401) {
          message.error(msg)
          // 跳转到登录页
          setTimeout(() => {
            localStorage.removeItem('token')
            router.push('/login')
          }, 1500)
        } else {
          notification.error({
            message: '权限错误',
            description: msg,
            duration: 4,
          })
        }
        break

      case ErrorType.VALIDATION:
        message.warning(msg)
        break

      case ErrorType.SERVER:
        notification.error({
          message: '服务器错误',
          description: msg,
          duration: 5,
        })
        break

      case ErrorType.BUSINESS:
        message.error(msg)
        break

      default:
        message.error(msg || '操作失败')
    }
  }

  /**
   * 上报错误（可选）
   */
  private reportError(errorInfo: ErrorInfo, context?: string) {
    // 这里可以实现错误上报逻辑
    // 例如发送到监控平台
    if (import.meta.env.PROD) {
      // 生产环境才上报
      // sendToMonitoring(errorInfo, context)
    }
  }

  /**
   * 获取错误日志
   */
  getErrorLog(): ErrorInfo[] {
    return [...this.errorLog]
  }

  /**
   * 清空错误日志
   */
  clearErrorLog() {
    this.errorLog = []
  }
}

/**
 * 全局错误处理器实例
 */
export const errorHandler = new ErrorHandler()

/**
 * Vue 错误处理插件
 */
export function setupErrorHandler(app: App) {
  // Vue 错误处理
  app.config.errorHandler = (err, instance, info) => {
    console.error('[Vue Error]', err, info)
    errorHandler.handle(err, `Vue: ${info}`)
  }

  // Vue 警告处理
  app.config.warnHandler = (msg, instance, trace) => {
    console.warn('[Vue Warning]', msg, trace)
  }

  // 全局未捕获错误
  window.addEventListener('error', (event) => {
    console.error('[Global Error]', event.error)
    errorHandler.handle(event.error, 'Global')
  })

  // 全局未捕获 Promise 错误
  window.addEventListener('unhandledrejection', (event) => {
    console.error('[Unhandled Promise]', event.reason)
    errorHandler.handle(event.reason, 'Promise')
    event.preventDefault()
  })
}

/**
 * 错误边界组件辅助函数
 */
export function createErrorBoundary(fallback: () => any) {
  return {
    errorCaptured(err: Error, instance: any, info: string) {
      errorHandler.handle(err, `Component: ${info}`)
      return fallback()
    },
  }
}

/**
 * 异步错误处理装饰器
 */
export function handleAsyncError(
  target: any,
  propertyKey: string,
  descriptor: PropertyDescriptor
) {
  const originalMethod = descriptor.value

  descriptor.value = async function (...args: any[]) {
    try {
      return await originalMethod.apply(this, args)
    } catch (error) {
      errorHandler.handle(error, `Method: ${propertyKey}`)
      throw error
    }
  }

  return descriptor
}

/**
 * Try-Catch 包装器
 */
export async function tryCatch<T>(
  fn: () => Promise<T>,
  errorMessage?: string
): Promise<[T | null, Error | null]> {
  try {
    const result = await fn()
    return [result, null]
  } catch (error: any) {
    if (errorMessage) {
      error.message = errorMessage
    }
    errorHandler.handle(error)
    return [null, error]
  }
}

/**
 * 安全执行函数
 */
export async function safeExecute<T>(
  fn: () => Promise<T>,
  fallback?: T
): Promise<T | undefined> {
  try {
    return await fn()
  } catch (error) {
    errorHandler.handle(error)
    return fallback
  }
}

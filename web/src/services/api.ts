import axios from 'axios'
import { message } from 'ant-design-vue'

declare module 'axios' {
  interface InternalAxiosRequestConfig {
    skipErrorToast?: boolean
  }
}

const BASE_URL = import.meta.env.VITE_API_BASE || '/app/api/v1'

const request = axios.create({
  baseURL: BASE_URL,
  timeout: 30000,
})

// HTTP 状态码错误消息映射
const httpErrorMessages: Record<number, string> = {
  400: '请求参数错误，请检查后重试',
  401: '登录已过期，请重新登录',
  403: '您没有权限执行此操作',
  404: '请求的资源不存在',
  409: '操作冲突，资源可能已被修改',
  422: '数据验证失败，请检查输入',
  500: '服务器内部错误，请稍后重试',
  502: '网关错误，服务暂时不可用',
  503: '服务暂时不可用，请稍后重试',
  504: '请求超时，请稍后重试',
}

// 业务错误码消息映射
const businessErrorMessages: Record<number, string> = {
  // 通用错误 (1000-1999)
  1000: '服务器内部错误，请稍后重试',
  1001: '请求参数不正确，请检查后重试',
  1002: '不支持的请求方法',
  1003: '请求超时，请稍后重试',
  // 认证授权错误 (2000-2999)
  2000: '请先登录',
  2001: '您没有权限执行此操作',
  2002: '登录已过期，请重新登录',
  2003: '登录凭证无效，请重新登录',
  // 业务错误 (3000-3999)
  3000: '请求的资源不存在',
  3001: '操作冲突，资源可能已被修改',
  3002: '资源已存在，请勿重复创建',
  3003: '业务处理失败',
  3004: '当前状态不允许此操作',
  // 飞书相关错误 (4000-4999)
  4000: '飞书认证失败，请检查配置',
  4001: '飞书服务调用失败，请稍后重试',
  4002: '飞书消息发送失败',
  // Jenkins相关错误 (5000-5999)
  5000: 'Jenkins 服务连接失败，请检查配置',
  5001: 'Jenkins 操作失败，请稍后重试',
  5002: '构建任务执行失败',
  // K8s相关错误 (6000-6999)
  6000: 'Kubernetes 集群连接失败，请检查配置',
  6001: 'Kubernetes 配置错误',
  6002: '部署失败，请检查配置和资源状态',
  6003: 'Pod 操作失败',
  // Archery相关错误 (7000-7999)
  7000: 'Archery 服务连接失败',
  7001: 'Archery 操作失败',
  7002: 'SQL 工单处理失败',
  // Redis相关错误 (8000-8999)
  8000: '缓存服务连接失败',
  8001: '操作正在进行中，请稍后重试',
  // 数据库相关错误 (9000-9999)
  9000: '数据库连接失败，请稍后重试',
  9001: '数据查询失败，请稍后重试',
  9002: '数据操作失败，请稍后重试',
}

// 获取友好的错误消息
const getFriendlyMessage = (code: number, defaultMsg?: string): string => {
  if (businessErrorMessages[code]) {
    return businessErrorMessages[code]
  }
  return defaultMsg || '操作失败，请稍后重试'
}

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    message.error('请求发送失败，请检查网络连接')
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    const data = response.data
    // 业务错误码处理
    if (data.code && data.code !== 0) {
      // 优先使用后端返回的消息，如果没有则使用映射的友好消息
      const errMsg = data.message || getFriendlyMessage(data.code)
      message.error(errMsg)
      return Promise.reject(new Error(errMsg))
    }
    // 返回完整响应对象，保持向后兼容
    return data
  },
  (error) => {
    const status = error.response?.status
    const data = error.response?.data

    // 获取错误消息：优先使用后端返回的消息
    let errMsg = ''
    if (data?.message && data.message !== 'error' && data.message !== 'Error') {
      errMsg = data.message
    } else if (data?.code) {
      errMsg = getFriendlyMessage(data.code)
    } else {
      errMsg = httpErrorMessages[status] || '请求失败，请稍后重试'
    }
    
    // 特殊处理 401
    if (status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('userInfo')
      message.error('登录已过期，请重新登录')
      setTimeout(() => {
        window.location.href = '/login'
      }, 1500)
      return Promise.reject(error)
    }

    // 网络错误
    if (!error.response) {
      errMsg = '网络连接失败，请检查网络后重试'
    }

    if (!error.config?.skipErrorToast) {
      message.error(errMsg)
    }
    return Promise.reject(error)
  }
)

export default request

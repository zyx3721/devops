import type { Rule } from 'ant-design-vue/es/form'

/**
 * 必填验证
 */
export const required = (message = '此项为必填项'): Rule => ({
  required: true,
  message,
  trigger: 'blur',
})

/**
 * 邮箱验证
 */
export const email = (message = '请输入有效的邮箱地址'): Rule => ({
  type: 'email',
  message,
  trigger: 'blur',
})

/**
 * URL 验证
 */
export const url = (message = '请输入有效的 URL'): Rule => ({
  type: 'url',
  message,
  trigger: 'blur',
})

/**
 * 数字验证
 */
export const number = (message = '请输入数字'): Rule => ({
  type: 'number',
  message,
  trigger: 'blur',
})

/**
 * 整数验证
 */
export const integer = (message = '请输入整数'): Rule => ({
  type: 'integer',
  message,
  trigger: 'blur',
})

/**
 * 最小长度验证
 */
export const minLength = (min: number, message?: string): Rule => ({
  min,
  message: message || `长度不能少于 ${min} 个字符`,
  trigger: 'blur',
})

/**
 * 最大长度验证
 */
export const maxLength = (max: number, message?: string): Rule => ({
  max,
  message: message || `长度不能超过 ${max} 个字符`,
  trigger: 'blur',
})

/**
 * 长度范围验证
 */
export const lengthRange = (min: number, max: number, message?: string): Rule => ({
  min,
  max,
  message: message || `长度必须在 ${min} 到 ${max} 个字符之间`,
  trigger: 'blur',
})

/**
 * 最小值验证
 */
export const minValue = (min: number, message?: string): Rule => ({
  type: 'number',
  min,
  message: message || `值不能小于 ${min}`,
  trigger: 'blur',
})

/**
 * 最大值验证
 */
export const maxValue = (max: number, message?: string): Rule => ({
  type: 'number',
  max,
  message: message || `值不能大于 ${max}`,
  trigger: 'blur',
})

/**
 * 值范围验证
 */
export const valueRange = (min: number, max: number, message?: string): Rule => ({
  type: 'number',
  min,
  max,
  message: message || `值必须在 ${min} 到 ${max} 之间`,
  trigger: 'blur',
})

/**
 * 正则表达式验证
 */
export const pattern = (regex: RegExp, message: string): Rule => ({
  pattern: regex,
  message,
  trigger: 'blur',
})

/**
 * 手机号验证
 */
export const phone = (message = '请输入有效的手机号'): Rule => ({
  pattern: /^1[3-9]\d{9}$/,
  message,
  trigger: 'blur',
})

/**
 * 身份证号验证
 */
export const idCard = (message = '请输入有效的身份证号'): Rule => ({
  pattern: /(^\d{15}$)|(^\d{18}$)|(^\d{17}(\d|X|x)$)/,
  message,
  trigger: 'blur',
})

/**
 * 用户名验证（字母、数字、下划线）
 */
export const username = (message = '用户名只能包含字母、数字和下划线'): Rule => ({
  pattern: /^[a-zA-Z0-9_]+$/,
  message,
  trigger: 'blur',
})

/**
 * 密码强度验证（至少包含大小写字母、数字）
 */
export const strongPassword = (message = '密码必须包含大小写字母和数字，长度至少8位'): Rule => ({
  pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[a-zA-Z\d@$!%*?&]{8,}$/,
  message,
  trigger: 'blur',
})

/**
 * IP 地址验证
 */
export const ipAddress = (message = '请输入有效的 IP 地址'): Rule => ({
  pattern: /^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$/,
  message,
  trigger: 'blur',
})

/**
 * 端口号验证
 */
export const port = (message = '请输入有效的端口号（1-65535）'): Rule => ({
  validator: (_rule, value) => {
    if (!value) return Promise.resolve()
    const num = Number(value)
    if (isNaN(num) || num < 1 || num > 65535) {
      return Promise.reject(message)
    }
    return Promise.resolve()
  },
  trigger: 'blur',
})

/**
 * JSON 格式验证
 */
export const json = (message = '请输入有效的 JSON 格式'): Rule => ({
  validator: (_rule, value) => {
    if (!value) return Promise.resolve()
    try {
      JSON.parse(value)
      return Promise.resolve()
    } catch {
      return Promise.reject(message)
    }
  },
  trigger: 'blur',
})

/**
 * 自定义验证器
 */
export const custom = (
  validator: (rule: any, value: any) => Promise<void>,
  trigger: 'blur' | 'change' = 'blur'
): Rule => ({
  validator,
  trigger,
})

/**
 * 异步验证器（用于检查唯一性等）
 */
export const asyncValidator = (
  checkFn: (value: any) => Promise<boolean>,
  message: string,
  trigger: 'blur' | 'change' = 'blur'
): Rule => ({
  validator: async (_rule, value) => {
    if (!value) return Promise.resolve()
    const isValid = await checkFn(value)
    if (!isValid) {
      return Promise.reject(message)
    }
    return Promise.resolve()
  },
  trigger,
})

/**
 * 白名单验证
 */
export const whitelist = (list: any[], message?: string): Rule => ({
  validator: (_rule, value) => {
    if (!value) return Promise.resolve()
    if (!list.includes(value)) {
      return Promise.reject(message || `值必须是以下之一: ${list.join(', ')}`)
    }
    return Promise.resolve()
  },
  trigger: 'blur',
})

/**
 * 黑名单验证
 */
export const blacklist = (list: any[], message?: string): Rule => ({
  validator: (_rule, value) => {
    if (!value) return Promise.resolve()
    if (list.includes(value)) {
      return Promise.reject(message || `值不能是以下之一: ${list.join(', ')}`)
    }
    return Promise.resolve()
  },
  trigger: 'blur',
})

/**
 * 组合验证规则
 */
export const combine = (...rules: Rule[]): Rule[] => rules

/**
 * 常用表单验证规则组合
 */
export const commonRules = {
  // 必填的用户名
  requiredUsername: [
    required('请输入用户名'),
    minLength(3, '用户名至少3个字符'),
    maxLength(20, '用户名最多20个字符'),
    username(),
  ],
  
  // 必填的邮箱
  requiredEmail: [
    required('请输入邮箱'),
    email(),
  ],
  
  // 必填的密码
  requiredPassword: [
    required('请输入密码'),
    minLength(6, '密码至少6个字符'),
  ],
  
  // 必填的强密码
  requiredStrongPassword: [
    required('请输入密码'),
    strongPassword(),
  ],
  
  // 必填的手机号
  requiredPhone: [
    required('请输入手机号'),
    phone(),
  ],
  
  // 必填的 URL
  requiredUrl: [
    required('请输入 URL'),
    url(),
  ],
  
  // 必填的数字
  requiredNumber: [
    required('请输入数字'),
    number(),
  ],
  
  // 必填的正整数
  requiredPositiveInteger: [
    required('请输入数字'),
    integer(),
    minValue(1, '值必须大于0'),
  ],
  
  // 必填的非负整数
  requiredNonNegativeInteger: [
    required('请输入数字'),
    integer(),
    minValue(0, '值不能小于0'),
  ],
}

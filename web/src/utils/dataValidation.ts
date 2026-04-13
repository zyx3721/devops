/**
 * 数据验证工具
 * 用于验证 API 响应数据和表单数据的有效性
 */

/**
 * 验证结果接口
 */
export interface ValidationResult {
  valid: boolean
  errors: string[]
}

/**
 * 字段验证规则
 */
export interface FieldRule {
  required?: boolean
  type?: 'string' | 'number' | 'boolean' | 'array' | 'object'
  min?: number
  max?: number
  pattern?: RegExp
  custom?: (value: any) => boolean | string
}

/**
 * 验证对象是否为空
 */
export function isEmpty(value: any): boolean {
  if (value === null || value === undefined) return true
  if (typeof value === 'string') return value.trim() === ''
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'object') return Object.keys(value).length === 0
  return false
}

/**
 * 验证字段
 */
export function validateField(value: any, rule: FieldRule): ValidationResult {
  const errors: string[] = []

  // 必填验证
  if (rule.required && isEmpty(value)) {
    errors.push('此字段为必填项')
    return { valid: false, errors }
  }

  // 如果值为空且非必填，跳过其他验证
  if (isEmpty(value) && !rule.required) {
    return { valid: true, errors: [] }
  }

  // 类型验证
  if (rule.type) {
    const actualType = Array.isArray(value) ? 'array' : typeof value
    if (actualType !== rule.type) {
      errors.push(`字段类型应为 ${rule.type}`)
    }
  }

  // 最小值/长度验证
  if (rule.min !== undefined) {
    if (typeof value === 'number' && value < rule.min) {
      errors.push(`值不能小于 ${rule.min}`)
    } else if (typeof value === 'string' && value.length < rule.min) {
      errors.push(`长度不能小于 ${rule.min}`)
    } else if (Array.isArray(value) && value.length < rule.min) {
      errors.push(`数组长度不能小于 ${rule.min}`)
    }
  }

  // 最大值/长度验证
  if (rule.max !== undefined) {
    if (typeof value === 'number' && value > rule.max) {
      errors.push(`值不能大于 ${rule.max}`)
    } else if (typeof value === 'string' && value.length > rule.max) {
      errors.push(`长度不能大于 ${rule.max}`)
    } else if (Array.isArray(value) && value.length > rule.max) {
      errors.push(`数组长度不能大于 ${rule.max}`)
    }
  }

  // 正则验证
  if (rule.pattern && typeof value === 'string') {
    if (!rule.pattern.test(value)) {
      errors.push('格式不正确')
    }
  }

  // 自定义验证
  if (rule.custom) {
    const result = rule.custom(value)
    if (result === false) {
      errors.push('验证失败')
    } else if (typeof result === 'string') {
      errors.push(result)
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 验证对象
 */
export function validateObject(
  data: Record<string, any>,
  rules: Record<string, FieldRule>
): ValidationResult {
  const errors: string[] = []

  for (const [field, rule] of Object.entries(rules)) {
    const result = validateField(data[field], rule)
    if (!result.valid) {
      errors.push(`${field}: ${result.errors.join(', ')}`)
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 验证 API 响应数据结构
 */
export function validateApiResponse(response: any): ValidationResult {
  const errors: string[] = []

  // 检查响应是否存在
  if (!response) {
    errors.push('响应数据为空')
    return { valid: false, errors }
  }

  // 检查响应格式
  if (typeof response !== 'object') {
    errors.push('响应数据格式错误')
    return { valid: false, errors }
  }

  // 检查必要字段
  if (!('code' in response) && !('status' in response)) {
    errors.push('响应缺少状态码字段')
  }

  if (!('data' in response) && !('result' in response)) {
    errors.push('响应缺少数据字段')
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 验证分页数据
 */
export function validatePaginationData(data: any): ValidationResult {
  const errors: string[] = []

  if (!data) {
    errors.push('分页数据为空')
    return { valid: false, errors }
  }

  // 检查必要字段
  if (!('items' in data) && !('list' in data) && !('data' in data)) {
    errors.push('缺少列表数据字段')
  }

  if (!('total' in data)) {
    errors.push('缺少总数字段')
  }

  // 验证数据类型
  const items = data.items || data.list || data.data
  if (items && !Array.isArray(items)) {
    errors.push('列表数据应为数组')
  }

  if (data.total !== undefined && typeof data.total !== 'number') {
    errors.push('总数应为数字')
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 验证 ID
 */
export function validateId(id: any): ValidationResult {
  const errors: string[] = []

  if (isEmpty(id)) {
    errors.push('ID 不能为空')
  } else if (typeof id !== 'number' && typeof id !== 'string') {
    errors.push('ID 类型错误')
  } else if (typeof id === 'number' && id <= 0) {
    errors.push('ID 必须大于 0')
  } else if (typeof id === 'string' && id.trim() === '') {
    errors.push('ID 不能为空字符串')
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 验证日期范围
 */
export function validateDateRange(startDate: any, endDate: any): ValidationResult {
  const errors: string[] = []

  if (!startDate || !endDate) {
    errors.push('日期范围不完整')
    return { valid: false, errors }
  }

  const start = new Date(startDate)
  const end = new Date(endDate)

  if (isNaN(start.getTime())) {
    errors.push('开始日期格式错误')
  }

  if (isNaN(end.getTime())) {
    errors.push('结束日期格式错误')
  }

  if (start > end) {
    errors.push('开始日期不能晚于结束日期')
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 验证文件
 */
export function validateFile(
  file: File,
  options: {
    maxSize?: number // 字节
    allowedTypes?: string[]
    allowedExtensions?: string[]
  } = {}
): ValidationResult {
  const errors: string[] = []

  if (!file) {
    errors.push('文件不能为空')
    return { valid: false, errors }
  }

  // 验证文件大小
  if (options.maxSize && file.size > options.maxSize) {
    const maxSizeMB = (options.maxSize / 1024 / 1024).toFixed(2)
    errors.push(`文件大小不能超过 ${maxSizeMB}MB`)
  }

  // 验证文件类型
  if (options.allowedTypes && !options.allowedTypes.includes(file.type)) {
    errors.push(`不支持的文件类型: ${file.type}`)
  }

  // 验证文件扩展名
  if (options.allowedExtensions) {
    const ext = file.name.split('.').pop()?.toLowerCase()
    if (!ext || !options.allowedExtensions.includes(ext)) {
      errors.push(`不支持的文件扩展名: ${ext}`)
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 安全解析 JSON
 */
export function safeParseJSON<T = any>(
  json: string,
  defaultValue?: T
): [T | null, Error | null] {
  try {
    const data = JSON.parse(json)
    return [data, null]
  } catch (error) {
    return [defaultValue || null, error as Error]
  }
}

/**
 * 验证 JSON 字符串
 */
export function validateJSON(json: string): ValidationResult {
  const errors: string[] = []

  if (!json || json.trim() === '') {
    errors.push('JSON 字符串不能为空')
    return { valid: false, errors }
  }

  try {
    JSON.parse(json)
  } catch (error: any) {
    errors.push(`JSON 格式错误: ${error.message}`)
  }

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 深度验证对象结构
 */
export function validateStructure(
  data: any,
  schema: Record<string, any>
): ValidationResult {
  const errors: string[] = []

  function validate(obj: any, schemaObj: any, path: string = '') {
    for (const [key, expectedType] of Object.entries(schemaObj)) {
      const currentPath = path ? `${path}.${key}` : key
      const value = obj?.[key]

      if (expectedType === 'required') {
        if (isEmpty(value)) {
          errors.push(`${currentPath} 是必填字段`)
        }
      } else if (typeof expectedType === 'string') {
        const actualType = Array.isArray(value) ? 'array' : typeof value
        if (actualType !== expectedType && value !== undefined) {
          errors.push(`${currentPath} 应为 ${expectedType} 类型`)
        }
      } else if (typeof expectedType === 'object' && !Array.isArray(expectedType)) {
        if (value && typeof value === 'object') {
          validate(value, expectedType, currentPath)
        }
      }
    }
  }

  validate(data, schema)

  return {
    valid: errors.length === 0,
    errors,
  }
}

/**
 * 批量验证
 */
export function validateBatch(
  validators: Array<() => ValidationResult>
): ValidationResult {
  const allErrors: string[] = []

  for (const validator of validators) {
    const result = validator()
    if (!result.valid) {
      allErrors.push(...result.errors)
    }
  }

  return {
    valid: allErrors.length === 0,
    errors: allErrors,
  }
}

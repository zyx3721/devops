import { message, Modal, notification } from 'ant-design-vue'
import type { ModalFuncProps } from 'ant-design-vue'

/**
 * 成功提示
 */
export function showSuccess(content: string, duration = 3) {
  message.success(content, duration)
}

/**
 * 错误提示
 */
export function showError(content: string, duration = 5) {
  message.error(content, duration)
}

/**
 * 警告提示
 */
export function showWarning(content: string, duration = 4) {
  message.warning(content, duration)
}

/**
 * 信息提示
 */
export function showInfo(content: string, duration = 3) {
  message.info(content, duration)
}

/**
 * 加载提示
 */
export function showLoading(content = '加载中...', duration = 0) {
  return message.loading(content, duration)
}

/**
 * 确认对话框
 */
export function showConfirm(options: {
  title: string
  content?: string
  onOk?: () => void | Promise<void>
  onCancel?: () => void
  okText?: string
  cancelText?: string
  okType?: 'primary' | 'danger' | 'dashed' | 'link' | 'text' | 'default'
}) {
  return Modal.confirm({
    title: options.title,
    content: options.content,
    okText: options.okText || '确定',
    cancelText: options.cancelText || '取消',
    okType: options.okType || 'primary',
    onOk: options.onOk,
    onCancel: options.onCancel,
  })
}

/**
 * 删除确认对话框
 */
export function showDeleteConfirm(options: {
  title?: string
  content?: string
  onOk: () => void | Promise<void>
  onCancel?: () => void
}) {
  return Modal.confirm({
    title: options.title || '确认删除',
    content: options.content || '删除后无法恢复，确定要删除吗？',
    okText: '删除',
    cancelText: '取消',
    okType: 'danger',
    onOk: options.onOk,
    onCancel: options.onCancel,
  })
}

/**
 * 信息对话框
 */
export function showInfoModal(options: ModalFuncProps) {
  return Modal.info(options)
}

/**
 * 成功对话框
 */
export function showSuccessModal(options: ModalFuncProps) {
  return Modal.success(options)
}

/**
 * 错误对话框
 */
export function showErrorModal(options: ModalFuncProps) {
  return Modal.error(options)
}

/**
 * 警告对话框
 */
export function showWarningModal(options: ModalFuncProps) {
  return Modal.warning(options)
}

/**
 * 通知 - 成功
 */
export function notifySuccess(options: {
  message: string
  description?: string
  duration?: number
}) {
  notification.success({
    message: options.message,
    description: options.description,
    duration: options.duration || 4.5,
  })
}

/**
 * 通知 - 错误
 */
export function notifyError(options: {
  message: string
  description?: string
  duration?: number
}) {
  notification.error({
    message: options.message,
    description: options.description,
    duration: options.duration || 4.5,
  })
}

/**
 * 通知 - 警告
 */
export function notifyWarning(options: {
  message: string
  description?: string
  duration?: number
}) {
  notification.warning({
    message: options.message,
    description: options.description,
    duration: options.duration || 4.5,
  })
}

/**
 * 通知 - 信息
 */
export function notifyInfo(options: {
  message: string
  description?: string
  duration?: number
}) {
  notification.info({
    message: options.message,
    description: options.description,
    duration: options.duration || 4.5,
  })
}

/**
 * 操作成功反馈
 */
export function operationSuccess(action: string, target?: string) {
  const content = target ? `${action}${target}成功` : `${action}成功`
  showSuccess(content)
}

/**
 * 操作失败反馈
 */
export function operationError(action: string, target?: string, error?: any) {
  const content = target ? `${action}${target}失败` : `${action}失败`
  const description = error?.message || error?.msg || ''
  
  if (description) {
    notifyError({
      message: content,
      description,
    })
  } else {
    showError(content)
  }
}

/**
 * 批量操作确认
 */
export function showBatchConfirm(options: {
  count: number
  action: string
  onOk: () => void | Promise<void>
  onCancel?: () => void
}) {
  return Modal.confirm({
    title: `确认${options.action}`,
    content: `确定要${options.action} ${options.count} 项吗？`,
    okText: '确定',
    cancelText: '取消',
    okType: 'primary',
    onOk: options.onOk,
    onCancel: options.onCancel,
  })
}

/**
 * 批量删除确认
 */
export function showBatchDeleteConfirm(options: {
  count: number
  onOk: () => void | Promise<void>
  onCancel?: () => void
}) {
  return Modal.confirm({
    title: '确认批量删除',
    content: `确定要删除选中的 ${options.count} 项吗？删除后无法恢复。`,
    okText: '删除',
    cancelText: '取消',
    okType: 'danger',
    onOk: options.onOk,
    onCancel: options.onCancel,
  })
}

/**
 * 表单验证失败提示
 */
export function showValidationError(message = '请检查表单填写是否正确') {
  showWarning(message)
}

/**
 * 网络错误提示
 */
export function showNetworkError(error?: any) {
  const message = error?.message || '网络请求失败，请检查网络连接'
  notifyError({
    message: '网络错误',
    description: message,
  })
}

/**
 * 权限错误提示
 */
export function showPermissionError() {
  notifyError({
    message: '权限不足',
    description: '您没有权限执行此操作',
  })
}

/**
 * 复制成功提示
 */
export function showCopySuccess() {
  showSuccess('复制成功')
}

/**
 * 导出成功提示
 */
export function showExportSuccess() {
  showSuccess('导出成功')
}

/**
 * 导入成功提示
 */
export function showImportSuccess() {
  showSuccess('导入成功')
}

/**
 * 保存成功提示
 */
export function showSaveSuccess() {
  showSuccess('保存成功')
}

/**
 * 提交成功提示
 */
export function showSubmitSuccess() {
  showSuccess('提交成功')
}

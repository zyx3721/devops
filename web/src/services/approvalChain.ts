import request from './api'

// 审批链相关接口

// 审批链类型定义
export interface ApprovalChain {
  id: number
  name: string
  app_id: number
  env: string
  enabled: boolean
  timeout_minutes: number
  timeout_action: string
  description: string
  created_at: string
  updated_at: string
  nodes?: ApprovalNode[]
}

export interface ApprovalNode {
  id: number
  chain_id: number
  name: string
  node_order: number
  approve_mode: 'any' | 'all' | 'count'
  approve_count: number
  approver_type: 'user' | 'role' | 'app_owner' | 'team_leader'
  approvers: string
  timeout_minutes: number
  timeout_action: string
  reject_on_any: boolean
}

export interface ApprovalInstance {
  id: number
  record_id: number
  chain_id: number
  chain_name: string
  status: 'pending' | 'approved' | 'rejected' | 'cancelled'
  current_node_order: number
  started_at: string
  finished_at: string
  cancel_reason: string
  node_instances?: ApprovalNodeInstance[]
}

export interface ApprovalNodeInstance {
  id: number
  instance_id: number
  node_id: number
  node_name: string
  node_order: number
  approve_mode: string
  approve_count: number
  approvers: string
  status: 'pending' | 'active' | 'approved' | 'rejected' | 'timeout'
  approved_count: number
  rejected_count: number
  activated_at: string
  finished_at: string
  timeout_at: string
  actions?: ApprovalAction[]
}

export interface ApprovalAction {
  id: number
  node_instance_id: number
  user_id: number
  user_name: string
  action: 'approve' | 'reject' | 'transfer'
  comment: string
  transfer_to: number
  transfer_to_name: string
  created_at: string
}

export interface ApprovalStats {
  total: number
  approved: number
  rejected: number
  cancelled: number
  pending: number
  approval_rate: number
  avg_duration_seconds: number
}

// ============================================================================
// 审批链管理 API
// ============================================================================

// 获取审批链列表
export function getChainList(params: {
  page?: number
  page_size?: number
  app_id?: number
  env?: string
}) {
  return request({
    url: '/approval/chains',
    method: 'get',
    params
  })
}

// 创建审批链
export function createChain(data: Partial<ApprovalChain>) {
  return request({
    url: '/approval/chains',
    method: 'post',
    data
  })
}

// 获取审批链详情
export function getChain(id: number) {
  return request({
    url: `/approval/chains/${id}`,
    method: 'get'
  })
}

// 更新审批链
export function updateChain(id: number, data: Partial<ApprovalChain>) {
  return request({
    url: `/approval/chains/${id}`,
    method: 'put',
    data
  })
}

// 删除审批链
export function deleteChain(id: number) {
  return request({
    url: `/approval/chains/${id}`,
    method: 'delete'
  })
}

// 测试审批链
export function testChain(id: number) {
  return request({
    url: `/approval/chains/${id}/test`,
    method: 'post'
  })
}


// ============================================================================
// 审批节点管理 API
// ============================================================================

// 添加审批节点
export function addNode(chainId: number, data: Partial<ApprovalNode>) {
  return request({
    url: `/approval/chains/${chainId}/nodes`,
    method: 'post',
    data
  })
}

// 更新审批节点
export function updateNode(chainId: number, nodeId: number, data: Partial<ApprovalNode>) {
  return request({
    url: `/approval/chains/${chainId}/nodes/${nodeId}`,
    method: 'put',
    data
  })
}

// 删除审批节点
export function deleteNode(chainId: number, nodeId: number) {
  return request({
    url: `/approval/chains/${chainId}/nodes/${nodeId}`,
    method: 'delete'
  })
}

// 调整节点顺序
export function reorderNodes(chainId: number, nodeIds: number[]) {
  return request({
    url: `/approval/chains/${chainId}/nodes/reorder`,
    method: 'put',
    data: { node_ids: nodeIds }
  })
}

// ============================================================================
// 审批实例 API
// ============================================================================

// 获取审批实例列表
export function getInstanceList(params: {
  page?: number
  page_size?: number
  status?: string
}) {
  return request({
    url: '/approval/instances',
    method: 'get',
    params
  })
}

// 获取审批实例详情
export function getInstance(id: number) {
  return request({
    url: `/approval/instances/${id}`,
    method: 'get'
  })
}

// 取消审批实例
export function cancelInstance(id: number, reason?: string) {
  return request({
    url: `/approval/instances/${id}/cancel`,
    method: 'post',
    data: { reason }
  })
}

// ============================================================================
// 审批操作 API
// ============================================================================

// 获取待审批列表
export function getPendingApprovals() {
  return request({
    url: '/approval/chain/pending',
    method: 'get'
  })
}

// 审批通过
export function approveNode(nodeInstanceId: number, comment?: string) {
  return request({
    url: `/approval/nodes/${nodeInstanceId}/approve`,
    method: 'post',
    data: { comment }
  })
}

// 审批拒绝
export function rejectNode(nodeInstanceId: number, reason: string) {
  return request({
    url: `/approval/nodes/${nodeInstanceId}/reject`,
    method: 'post',
    data: { reason }
  })
}

// 转交审批
export function transferNode(nodeInstanceId: number, data: {
  to_user_id: number
  to_user_name: string
  reason?: string
}) {
  return request({
    url: `/approval/nodes/${nodeInstanceId}/transfer`,
    method: 'post',
    data
  })
}

// ============================================================================
// 统计 API
// ============================================================================

// 获取审批统计
export function getApprovalStats() {
  return request({
    url: '/approval/stats',
    method: 'get'
  })
}

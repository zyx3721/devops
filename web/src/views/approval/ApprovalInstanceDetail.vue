<template>
  <div class="instance-detail">
    <!-- 基本信息 -->
    <a-descriptions :column="2" bordered size="small">
      <a-descriptions-item label="审批链">{{ instance.chain_name }}</a-descriptions-item>
      <a-descriptions-item label="部署记录ID">{{ instance.record_id }}</a-descriptions-item>
      <a-descriptions-item label="状态">
        <a-tag :color="getStatusColor(instance.status)">
          {{ getStatusLabel(instance.status) }}
        </a-tag>
      </a-descriptions-item>
      <a-descriptions-item label="当前节点">
        <span v-if="instance.status === 'pending'">第 {{ instance.current_node_order }} 节点</span>
        <span v-else>-</span>
      </a-descriptions-item>
      <a-descriptions-item label="开始时间">{{ formatDate(instance.started_at) }}</a-descriptions-item>
      <a-descriptions-item label="完成时间">{{ formatDate(instance.finished_at) }}</a-descriptions-item>
      <a-descriptions-item v-if="instance.cancel_reason" label="取消原因" :span="2">
        {{ instance.cancel_reason }}
      </a-descriptions-item>
    </a-descriptions>

    <!-- 审批流程 -->
    <div class="flow-section">
      <h4>审批流程</h4>
      <div class="flow-container">
        <div
          v-for="(node, index) in instance.node_instances"
          :key="node.id"
          class="flow-node"
        >
          <!-- 连接线 -->
          <div v-if="index > 0" class="flow-connector">
            <div class="connector-line"></div>
          </div>

          <!-- 节点 -->
          <div class="node-card" :class="getNodeClass(node.status)">
            <div class="node-header">
              <span class="node-order">{{ node.node_order }}</span>
              <span class="node-name">{{ node.node_name }}</span>
              <a-tag :color="getNodeStatusColor(node.status)" size="small">
                {{ getNodeStatusLabel(node.status) }}
              </a-tag>
            </div>
            <div class="node-body">
              <div class="node-info">
                <span class="label">审批模式：</span>
                <span>{{ getModeLabel(node.approve_mode, node.approve_count) }}</span>
              </div>
              <div class="node-info">
                <span class="label">审批人：</span>
                <span>{{ formatApprovers(node) }}</span>
              </div>
              <div class="node-info">
                <span class="label">审批进度：</span>
                <span>{{ node.approved_count }} 通过 / {{ node.rejected_count }} 拒绝</span>
              </div>
              <div v-if="node.activated_at" class="node-info">
                <span class="label">激活时间：</span>
                <span>{{ formatDate(node.activated_at) }}</span>
              </div>
              <div v-if="node.finished_at" class="node-info">
                <span class="label">完成时间：</span>
                <span>{{ formatDate(node.finished_at) }}</span>
              </div>

              <!-- 审批操作按钮 -->
              <div v-if="node.status === 'active' && canApprove(node)" class="node-actions">
                <a-button type="primary" size="small" @click="handleApprove(node)">通过</a-button>
                <a-button type="primary" size="small" danger @click="handleReject(node)">拒绝</a-button>
                <a-button size="small" @click="handleTransfer(node)">转交</a-button>
              </div>

              <!-- 审批记录 -->
              <div v-if="node.actions && node.actions.length > 0" class="action-list">
                <div class="action-title">审批记录</div>
                <a-timeline>
                  <a-timeline-item
                    v-for="action in node.actions"
                    :key="action.id"
                    :color="getActionTimelineColor(action.action)"
                  >
                    <div class="action-content">
                      <span class="action-user">{{ action.user_name }}</span>
                      <a-tag :color="getActionTagColor(action.action)" size="small">
                        {{ getActionLabel(action.action) }}
                      </a-tag>
                      <span v-if="action.action === 'transfer'" class="transfer-info">
                        → {{ action.transfer_to_name }}
                      </span>
                      <span class="action-time">{{ formatDate(action.created_at) }}</span>
                    </div>
                    <div v-if="action.comment" class="action-comment">{{ action.comment }}</div>
                  </a-timeline-item>
                </a-timeline>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 转交对话框 -->
    <a-modal v-model:open="transferDialogVisible" title="转交审批" @ok="submitTransfer" :confirm-loading="submitting">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="转交给" required>
          <a-select
            v-model:value="transferForm.to_user_id"
            show-search
            placeholder="选择用户"
            :filter-option="filterOption"
            style="width: 100%"
          >
            <a-select-option v-for="user in userList" :key="user.id" :value="user.id">
              {{ user.username }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="原因">
          <a-textarea v-model:value="transferForm.reason" :rows="2" placeholder="转交原因（可选）" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 拒绝对话框 -->
    <a-modal 
      v-model:open="rejectDialogVisible" 
      title="拒绝审批" 
      @ok="confirmReject" 
      :confirm-loading="submitting"
      ok-text="确认拒绝"
      :ok-button-props="{ danger: true }"
    >
      <a-form :label-col="{ span: 4 }">
        <a-form-item label="原因" required>
          <a-textarea 
            v-model:value="rejectReason" 
            :rows="3" 
            placeholder="请输入拒绝原因" 
            show-count 
            :maxlength="200"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import {
  approveNode,
  rejectNode,
  transferNode,
  type ApprovalInstance,
  type ApprovalNodeInstance
} from '@/services/approvalChain'
import { userApi } from '@/services/user'
import dayjs from 'dayjs'

const props = defineProps<{
  instance: ApprovalInstance
}>()

const emit = defineEmits(['refresh'])

// 用户列表
const userList = ref<any[]>([])

// 转交对话框
const transferDialogVisible = ref(false)
const submitting = ref(false)
const currentNodeInstance = ref<ApprovalNodeInstance | null>(null)
const transferForm = reactive({
  to_user_id: undefined as number | undefined,
  reason: ''
})

// 拒绝对话框
const rejectDialogVisible = ref(false)
const rejectReason = ref('')
const rejectingNode = ref<ApprovalNodeInstance | null>(null)

// 加载用户列表
const loadUserList = async () => {
  try {
    const res = await userApi.getUsers({ page: 1, page_size: 1000 })
    // 兼容 items 和 list 两种格式
    userList.value = res.data.list || res.data.items || []
  } catch (error) {
    console.error('加载用户列表失败:', error)
  }
}

// 过滤选项
const filterOption = (input: string, option: any) => {
  return option.children[0].children.toLowerCase().indexOf(input.toLowerCase()) >= 0
}

// 检查当前用户是否可以审批
const canApprove = (node: ApprovalNodeInstance) => {
  return true // 简化处理，实际应检查用户ID
}

// 审批通过
const handleApprove = (node: ApprovalNodeInstance) => {
  Modal.confirm({
    title: '审批通过',
    content: '确定要通过此审批吗？',
    onOk: async () => {
      try {
        await approveNode(node.id)
        message.success('审批成功')
        emit('refresh')
      } catch (error: any) {
        // 错误已在拦截器中处理
      }
    }
  })
}

// 审批拒绝
const handleReject = (node: ApprovalNodeInstance) => {
  rejectingNode.value = node
  rejectReason.value = ''
  rejectDialogVisible.value = true
}

// 确认拒绝
const confirmReject = async () => {
  if (!rejectReason.value.trim()) {
    message.warning('请输入拒绝原因')
    return
  }
  if (!rejectingNode.value) return

  submitting.value = true
  try {
    await rejectNode(rejectingNode.value.id, rejectReason.value)
    message.success('已拒绝')
    rejectDialogVisible.value = false
    emit('refresh')
  } catch (error: any) {
    // 错误已在拦截器中处理
  } finally {
    submitting.value = false
  }
}

// 转交审批
const handleTransfer = (node: ApprovalNodeInstance) => {
  currentNodeInstance.value = node
  transferForm.to_user_id = undefined
  transferForm.reason = ''
  transferDialogVisible.value = true
}

// 提交转交
const submitTransfer = async () => {
  if (!transferForm.to_user_id) {
    message.warning('请选择转交用户')
    return
  }
  if (!currentNodeInstance.value) return

  const toUser = userList.value.find(u => u.id === transferForm.to_user_id)
  if (!toUser) {
    message.warning('用户不存在')
    return
  }

  submitting.value = true
  try {
    await transferNode(currentNodeInstance.value.id, {
      to_user_id: transferForm.to_user_id,
      to_user_name: toUser.username,
      reason: transferForm.reason
    })
    message.success('转交成功')
    transferDialogVisible.value = false
    emit('refresh')
  } catch (error: any) {
    // 错误已在拦截器中处理
  } finally {
    submitting.value = false
  }
}

// 工具函数
const formatDate = (date: string) => {
  if (!date) return '-'
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

const getStatusLabel = (status: string) => {
  const map: Record<string, string> = {
    pending: '待审批',
    approved: '已通过',
    rejected: '已拒绝',
    cancelled: '已取消'
  }
  return map[status] || status
}

const getStatusColor = (status: string) => {
  const map: Record<string, string> = {
    pending: 'processing',
    approved: 'success',
    rejected: 'error',
    cancelled: 'default'
  }
  return map[status] || 'default'
}

const getNodeStatusLabel = (status: string) => {
  const map: Record<string, string> = {
    pending: '等待中',
    active: '审批中',
    approved: '已通过',
    rejected: '已拒绝',
    timeout: '已超时'
  }
  return map[status] || status
}

const getNodeStatusColor = (status: string) => {
  const map: Record<string, string> = {
    pending: 'default',
    active: 'processing',
    approved: 'success',
    rejected: 'error',
    timeout: 'error'
  }
  return map[status] || 'default'
}

const getNodeClass = (status: string) => {
  return `node-${status}`
}

const getModeLabel = (mode: string, count: number) => {
  const map: Record<string, string> = {
    any: '任一人通过',
    all: '所有人通过',
    count: `${count}人通过`
  }
  return map[mode] || mode
}

// 格式化审批人显示
const formatApprovers = (node: ApprovalNodeInstance) => {
  if (!node.approvers) return '-'
  
  const ids = node.approvers.split(',').filter(id => id)
  if (ids.length === 0) return '-'
  
  // 尝试从用户列表中获取用户名
  const names = ids.map(id => {
    const user = userList.value.find(u => u.id === Number(id))
    return user?.username || `用户${id}`
  })
  
  return names.join('、')
}

const getActionLabel = (action: string) => {
  const map: Record<string, string> = {
    approve: '通过',
    reject: '拒绝',
    transfer: '转交'
  }
  return map[action] || action
}

const getActionTagColor = (action: string) => {
  const map: Record<string, string> = {
    approve: 'success',
    reject: 'error',
    transfer: 'warning'
  }
  return map[action] || 'default'
}

const getActionTimelineColor = (action: string) => {
  const map: Record<string, string> = {
    approve: 'green',
    reject: 'red',
    transfer: 'orange'
  }
  return map[action] || 'blue'
}

onMounted(() => {
  loadUserList()
})
</script>

<style scoped>
.instance-detail {
  padding: 16px 0;
}

.flow-section {
  margin-top: 24px;
}

.flow-section h4 {
  margin: 0 0 16px 0;
  color: #333;
}

.flow-container {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.flow-node {
  width: 100%;
  max-width: 600px;
}

.flow-connector {
  display: flex;
  justify-content: center;
  padding: 8px 0;
}

.connector-line {
  width: 2px;
  height: 24px;
  background: #d9d9d9;
}

.node-card {
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  overflow: hidden;
}

.node-card.node-active {
  border-color: #1890ff;
  box-shadow: 0 0 8px rgba(24, 144, 255, 0.3);
}

.node-card.node-approved {
  border-color: #52c41a;
}

.node-card.node-rejected,
.node-card.node-timeout {
  border-color: #ff4d4f;
}

.node-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #fafafa;
  border-bottom: 1px solid #d9d9d9;
  gap: 12px;
}

.node-order {
  width: 24px;
  height: 24px;
  line-height: 24px;
  text-align: center;
  background: #1890ff;
  color: #fff;
  border-radius: 50%;
  font-size: 12px;
}

.node-name {
  flex: 1;
  font-weight: 500;
}

.node-body {
  padding: 16px;
}

.node-info {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
}

.node-info .label {
  color: #999;
  width: 80px;
  flex-shrink: 0;
}

.node-actions {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
  display: flex;
  gap: 8px;
}

.action-list {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.action-title {
  font-size: 14px;
  color: #666;
  margin-bottom: 12px;
}

.action-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-user {
  font-weight: 500;
}

.transfer-info {
  color: #999;
}

.action-time {
  color: #999;
  font-size: 12px;
  margin-left: auto;
}

.action-comment {
  margin-top: 4px;
  color: #666;
  font-size: 13px;
}
</style>

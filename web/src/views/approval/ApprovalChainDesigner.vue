<template>
  <div class="chain-designer">
    <!-- 头部 -->
    <a-card :bordered="false" class="header-card">
      <div class="header">
        <div class="header-left">
          <a-button @click="goBack">
            <template #icon><ArrowLeftOutlined /></template>
            返回
          </a-button>
          <h3 style="margin: 0 16px">{{ chain?.name || '审批链设计' }}</h3>
          <a-tag v-if="chain" :color="chain.enabled ? 'success' : 'default'">
            {{ chain.enabled ? '已启用' : '已禁用' }}
          </a-tag>
        </div>
        <div class="header-right">
          <a-button type="primary" @click="handleAddNode">
            <template #icon><PlusOutlined /></template>
            添加节点
          </a-button>
        </div>
      </div>
    </a-card>

    <!-- 节点列表 -->
    <a-card :bordered="false" style="margin-top: 16px">
      <a-spin :spinning="loading">
        <div v-if="nodes.length === 0" class="empty-tip">
          <a-empty description="暂无审批节点，点击上方按钮添加" />
        </div>

        <div v-else class="node-list">
          <div
            v-for="(node, index) in nodes"
            :key="node.id"
            class="node-item"
          >
            <!-- 连接线 -->
            <div v-if="index > 0" class="connector">
              <div class="connector-line"></div>
              <ArrowDownOutlined class="connector-arrow" />
            </div>

            <!-- 节点卡片 -->
            <div class="node-card">
              <div class="node-header">
                <span class="node-order">{{ node.node_order }}</span>
                <span class="node-name">{{ node.name }}</span>
                <div class="node-actions">
                  <a-button type="link" size="small" @click="handleEditNode(node)">编辑</a-button>
                  <a-button type="link" size="small" danger @click="handleDeleteNode(node)">删除</a-button>
                </div>
              </div>
              <div class="node-body">
                <div class="node-info">
                  <span class="label">审批模式：</span>
                  <a-tag :color="getModeTagColor(node.approve_mode)">
                    {{ getModeLabel(node.approve_mode, node.approve_count) }}
                  </a-tag>
                </div>
                <div class="node-info">
                  <span class="label">审批人类型：</span>
                  <a-tag :color="getApproverTypeColor(node.approver_type)">
                    {{ getApproverTypeLabel(node.approver_type) }}
                  </a-tag>
                </div>
                <div class="node-info">
                  <span class="label">审批人：</span>
                  <span>{{ formatApprovers(node) }}</span>
                </div>
                <div class="node-info">
                  <span class="label">超时设置：</span>
                  <span>{{ node.timeout_minutes || '继承链配置' }}{{ node.timeout_minutes ? '分钟' : '' }}</span>
                </div>
                <div class="node-info">
                  <span class="label">拒绝策略：</span>
                  <span>{{ node.reject_on_any ? '任一人拒绝即拒绝' : '需全部拒绝' }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </a-spin>
    </a-card>

    <!-- 节点编辑对话框 -->
    <a-modal
      v-model:open="nodeDialogVisible"
      :title="nodeDialogTitle"
      width="650px"
      :confirm-loading="submitting"
      @ok="handleSubmitNode"
    >
      <a-form
        ref="nodeFormRef"
        :model="nodeForm"
        :rules="nodeFormRules"
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="节点名称" name="name">
          <a-input v-model:value="nodeForm.name" placeholder="如：技术负责人审批" />
        </a-form-item>
        <a-form-item label="审批模式" name="approve_mode">
          <a-radio-group v-model:value="nodeForm.approve_mode">
            <a-radio value="any">任一人通过</a-radio>
            <a-radio value="all">所有人通过</a-radio>
            <a-radio value="count">指定人数</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="nodeForm.approve_mode === 'count'" label="通过人数" name="approve_count">
          <a-input-number v-model:value="nodeForm.approve_count" :min="1" :max="10" />
          <span style="margin-left: 8px">人</span>
        </a-form-item>
        <a-form-item label="审批人类型" name="approver_type">
          <a-radio-group v-model:value="nodeForm.approver_type" @change="handleApproverTypeChange">
            <a-radio value="user">指定用户</a-radio>
            <a-radio value="role">指定角色</a-radio>
            <a-radio value="app_owner">应用负责人</a-radio>
            <a-radio value="team_leader">团队负责人</a-radio>
          </a-radio-group>
        </a-form-item>
        <!-- 指定用户 -->
        <a-form-item v-if="nodeForm.approver_type === 'user'" label="审批人" name="approverIds">
          <a-select
            v-model:value="nodeForm.approverIds"
            mode="multiple"
            :filter-option="filterOption"
            placeholder="选择审批人"
            style="width: 100%"
          >
            <a-select-option
              v-for="user in userList"
              :key="user.id"
              :value="user.id"
            >
              {{ user.username }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <!-- 指定角色 -->
        <a-form-item v-if="nodeForm.approver_type === 'role'" label="审批角色" name="approverRoles">
          <a-select
            v-model:value="nodeForm.approverRoles"
            mode="multiple"
            placeholder="选择角色"
            style="width: 100%"
          >
            <a-select-option
              v-for="role in roleList"
              :key="role.name"
              :value="role.name"
            >
              {{ role.display_name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <!-- 应用负责人/团队负责人 提示 -->
        <a-form-item v-if="nodeForm.approver_type === 'app_owner'" label="说明">
          <a-alert type="info" message="将自动获取发布应用的负责人作为审批人" show-icon />
        </a-form-item>
        <a-form-item v-if="nodeForm.approver_type === 'team_leader'" label="说明">
          <a-alert type="info" message="将自动获取发布应用所属团队的负责人作为审批人" show-icon />
        </a-form-item>
        <a-form-item label="超时时间" name="timeout_minutes">
          <a-input-number v-model:value="nodeForm.timeout_minutes" :min="0" :max="1440" />
          <span style="margin-left: 8px">分钟（0表示继承链配置）</span>
        </a-form-item>
        <a-form-item label="超时动作" name="timeout_action">
          <a-select v-model:value="nodeForm.timeout_action">
            <a-select-option value="auto_reject">自动拒绝</a-select-option>
            <a-select-option value="auto_approve">自动通过</a-select-option>
            <a-select-option value="auto_cancel">自动取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="拒绝策略" name="reject_on_any">
          <a-switch
            v-model:checked="nodeForm.reject_on_any"
            checked-children="任一人拒绝即拒绝"
            un-checked-children="需全部拒绝"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { ArrowLeftOutlined, ArrowDownOutlined, PlusOutlined } from '@ant-design/icons-vue'
import type { FormInstance, Rule } from 'ant-design-vue/es/form'
import {
  getChain,
  addNode,
  updateNode,
  deleteNode,
  type ApprovalChain,
  type ApprovalNode
} from '@/services/approvalChain'
import { userApi } from '@/services/user'
import { roleApi } from '@/services/rbac'

const route = useRoute()
const router = useRouter()

const chainId = computed(() => Number(route.params.id))

// 数据
const loading = ref(false)
const chain = ref<ApprovalChain | null>(null)
const nodes = ref<ApprovalNode[]>([])
const userList = ref<any[]>([])
const roleList = ref<any[]>([])

// 节点对话框
const nodeDialogVisible = ref(false)
const nodeDialogTitle = ref('添加节点')
const submitting = ref(false)
const nodeFormRef = ref<FormInstance>()
const nodeForm = reactive({
  id: 0,
  name: '',
  approve_mode: 'any' as 'any' | 'all' | 'count',
  approve_count: 1,
  approver_type: 'user' as 'user' | 'role' | 'app_owner' | 'team_leader',
  approverIds: [] as number[],
  approverRoles: [] as string[],
  timeout_minutes: 0,
  timeout_action: 'auto_reject',
  reject_on_any: true
})

const nodeFormRules: Record<string, Rule[]> = {
  name: [{ required: true, message: '请输入节点名称', trigger: 'blur' }],
  approve_mode: [{ required: true, message: '请选择审批模式', trigger: 'change' }]
}

// 加载审批链详情
const loadChain = async () => {
  loading.value = true
  try {
    const res = await getChain(chainId.value)
    chain.value = res.data
    nodes.value = res.data.nodes || []
  } catch (error) {
    console.error('加载审批链失败:', error)
    message.error('加载审批链失败')
  } finally {
    loading.value = false
  }
}

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

// 加载角色列表
const loadRoleList = async () => {
  try {
    const res = await roleApi.getRoles({ page: 1, page_size: 1000 })
    roleList.value = res.data.list || []
  } catch (error) {
    console.error('加载角色列表失败:', error)
  }
}

// 返回
const goBack = () => {
  router.push('/approval/chains')
}

// 审批人类型变更
const handleApproverTypeChange = () => {
  nodeForm.approverIds = []
  nodeForm.approverRoles = []
}

// 添加节点
const handleAddNode = () => {
  nodeDialogTitle.value = '添加节点'
  Object.assign(nodeForm, {
    id: 0,
    name: '',
    approve_mode: 'any',
    approve_count: 1,
    approver_type: 'user',
    approverIds: [],
    approverRoles: [],
    timeout_minutes: 0,
    timeout_action: 'auto_reject',
    reject_on_any: true
  })
  nodeDialogVisible.value = true
}

// 编辑节点
const handleEditNode = (node: ApprovalNode) => {
  nodeDialogTitle.value = '编辑节点'
  const approverType = node.approver_type || 'user'
  let approverIds: number[] = []
  let approverRoles: string[] = []
  
  if (approverType === 'user' && node.approvers) {
    approverIds = node.approvers.split(',').map(id => Number(id)).filter(id => id > 0)
  } else if (approverType === 'role' && node.approvers) {
    approverRoles = node.approvers.split(',').filter(r => r)
  }
  
  Object.assign(nodeForm, {
    id: node.id,
    name: node.name,
    approve_mode: node.approve_mode,
    approve_count: node.approve_count,
    approver_type: approverType,
    approverIds,
    approverRoles,
    timeout_minutes: node.timeout_minutes,
    timeout_action: node.timeout_action,
    reject_on_any: node.reject_on_any
  })
  nodeDialogVisible.value = true
}

// 删除节点
const handleDeleteNode = (node: ApprovalNode) => {
  Modal.confirm({
    title: '提示',
    content: `确定要删除节点"${node.name}"吗？`,
    okType: 'danger',
    onOk: async () => {
      try {
        await deleteNode(chainId.value, node.id)
        message.success('删除成功')
        loadChain()
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    }
  })
}

// 提交节点
const handleSubmitNode = async () => {
  if (!nodeFormRef.value) return
  try {
    await nodeFormRef.value.validate()

    // 验证审批人
    if (nodeForm.approver_type === 'user' && nodeForm.approverIds.length === 0) {
      message.warning('请选择至少一个审批人')
      return
    }
    if (nodeForm.approver_type === 'role' && nodeForm.approverRoles.length === 0) {
      message.warning('请选择至少一个角色')
      return
    }

    submitting.value = true
    
    // 根据类型构建 approvers 字段
    let approvers = ''
    if (nodeForm.approver_type === 'user') {
      approvers = nodeForm.approverIds.join(',')
    } else if (nodeForm.approver_type === 'role') {
      approvers = nodeForm.approverRoles.join(',')
    }
    // app_owner 和 team_leader 不需要 approvers，运行时动态获取
    
    const data = {
      name: nodeForm.name,
      approve_mode: nodeForm.approve_mode,
      approve_count: nodeForm.approve_count,
      approver_type: nodeForm.approver_type,
      approvers,
      timeout_minutes: nodeForm.timeout_minutes,
      timeout_action: nodeForm.timeout_action,
      reject_on_any: nodeForm.reject_on_any
    }

    if (nodeForm.id) {
      await updateNode(chainId.value, nodeForm.id, data)
      message.success('更新成功')
    } else {
      await addNode(chainId.value, data)
      message.success('添加成功')
    }
    nodeDialogVisible.value = false
    loadChain()
  } catch (error: any) {
    if (error.errorFields) return // 表单验证错误
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 过滤选项
const filterOption = (input: string, option: any) => {
  return option.children[0].children.toLowerCase().indexOf(input.toLowerCase()) >= 0
}

// 工具函数
const getModeLabel = (mode: string, count: number) => {
  const map: Record<string, string> = {
    any: '任一人通过',
    all: '所有人通过',
    count: `${count}人通过`
  }
  return map[mode] || mode
}

const getModeTagColor = (mode: string) => {
  const map: Record<string, string> = {
    any: 'success',
    all: 'warning',
    count: 'processing'
  }
  return map[mode] || 'default'
}

const getApproverTypeLabel = (type: string) => {
  const map: Record<string, string> = {
    user: '指定用户',
    role: '指定角色',
    app_owner: '应用负责人',
    team_leader: '团队负责人'
  }
  return map[type] || type
}

const getApproverTypeColor = (type: string) => {
  const map: Record<string, string> = {
    user: 'blue',
    role: 'purple',
    app_owner: 'cyan',
    team_leader: 'orange'
  }
  return map[type] || 'default'
}

const formatApprovers = (node: ApprovalNode) => {
  const type = node.approver_type || 'user'
  if (type === 'app_owner') return '（发布时自动获取）'
  if (type === 'team_leader') return '（发布时自动获取）'
  if (!node.approvers) return '-'
  
  if (type === 'user') {
    const ids = node.approvers.split(',').map(id => Number(id))
    const names = ids.map(id => {
      const user = userList.value.find(u => u.id === id)
      return user?.username || `用户${id}`
    })
    return names.join('、')
  }
  
  if (type === 'role') {
    const roleNames = node.approvers.split(',')
    const names = roleNames.map(name => {
      const role = roleList.value.find(r => r.name === name)
      return role?.display_name || name
    })
    return names.join('、')
  }
  
  return node.approvers
}

onMounted(() => {
  loadChain()
  loadUserList()
  loadRoleList()
})
</script>

<style scoped>
.chain-designer {
  padding: 16px;
}

.header-card {
  margin-bottom: 16px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
}

.empty-tip {
  padding: 40px 0;
}

.node-list {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
}

.node-item {
  width: 100%;
  max-width: 500px;
}

.connector {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 0;
}

.connector-line {
  width: 2px;
  height: 20px;
  background: #d9d9d9;
}

.connector-arrow {
  color: #8c8c8c;
  font-size: 16px;
}

.node-card {
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  overflow: hidden;
}

.node-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #fafafa;
  border-bottom: 1px solid #d9d9d9;
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
  margin-right: 12px;
}

.node-name {
  flex: 1;
  font-weight: 500;
}

.node-actions {
  display: flex;
  gap: 8px;
}

.node-body {
  padding: 16px;
}

.node-info {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.node-info:last-child {
  margin-bottom: 0;
}

.node-info .label {
  color: #8c8c8c;
  width: 80px;
  flex-shrink: 0;
}
</style>

<template>
  <div class="user-management">
    <a-card title="用户管理" :bordered="false">
      <template #extra>
        <a-button v-if="canCreate" type="primary" @click="showCreateModal">
          <template #icon><PlusOutlined /></template>
          新增用户
        </a-button>
      </template>

      <!-- 搜索筛选 -->
      <a-form layout="inline" style="margin-bottom: 16px">
        <a-form-item>
          <a-input v-model:value="filters.keyword" placeholder="搜索用户名/邮箱" allowClear style="width: 200px" @pressEnter="fetchUsers">
            <template #prefix><SearchOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item>
          <a-select v-model:value="filters.role" placeholder="角色" allowClear style="width: 120px" @change="fetchUsers">
            <a-select-option value="super_admin">超级管理员</a-select-option>
            <a-select-option value="admin">管理员</a-select-option>
            <a-select-option value="user">普通用户</a-select-option>
            <a-select-option value="guest">访客</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-select v-model:value="filters.status" placeholder="状态" allowClear style="width: 100px" @change="fetchUsers">
            <a-select-option value="active">启用</a-select-option>
            <a-select-option value="inactive">禁用</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button @click="fetchUsers">查询</a-button>
        </a-form-item>
      </a-form>

      <a-table :columns="columns" :data-source="users" :loading="loading" :pagination="pagination" @change="handleTableChange" row-key="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'username'">
            <a @click="showUserDetail(record)">{{ record.username }}</a>
          </template>
          <template v-if="column.key === 'role'">
            <a-tag :color="getRoleColor(record.role)">{{ getRoleText(record.role) }}</a-tag>
          </template>
          <template v-if="column.key === 'status'">
            <a-badge :status="record.status === 'active' ? 'success' : 'error'" :text="record.status === 'active' ? '启用' : '禁用'" />
          </template>
          <template v-if="column.key === 'action'">
            <a-button type="link" size="small" @click="showUserDetail(record)">查看</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 用户详情抽屉 -->
    <a-drawer
      v-model:open="detailVisible"
      :title="selectedUser?.username"
      width="500"
      :footer-style="{ textAlign: 'right' }"
    >
      <template v-if="selectedUser">
        <!-- 用户头像和基本信息 -->
        <div class="user-header">
          <a-avatar :size="64" style="background-color: #1890ff">
            {{ selectedUser.username?.charAt(0).toUpperCase() }}
          </a-avatar>
          <div class="user-header-info">
            <h3>{{ selectedUser.username }}</h3>
            <a-tag :color="getRoleColor(selectedUser.role)">{{ getRoleText(selectedUser.role) }}</a-tag>
            <a-badge :status="selectedUser.status === 'active' ? 'success' : 'error'" :text="selectedUser.status === 'active' ? '启用' : '禁用'" style="margin-left: 8px" />
          </div>
        </div>

        <a-divider />

        <!-- 详细信息 -->
        <a-descriptions :column="1" size="small">
          <a-descriptions-item label="用户ID">{{ selectedUser.id }}</a-descriptions-item>
          <a-descriptions-item label="邮箱">{{ selectedUser.email || '-' }}</a-descriptions-item>
          <a-descriptions-item label="手机号">{{ selectedUser.phone || '-' }}</a-descriptions-item>
          <a-descriptions-item label="最后登录">{{ selectedUser.last_login_at || '-' }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ selectedUser.created_at }}</a-descriptions-item>
          <a-descriptions-item label="更新时间">{{ selectedUser.updated_at }}</a-descriptions-item>
        </a-descriptions>

        <a-divider />

        <!-- 操作按钮 -->
        <div class="action-section" v-if="canManageUser(selectedUser)">
          <h4>用户操作</h4>
          <a-space direction="vertical" style="width: 100%">
            <a-button block @click="showEditModal(selectedUser)">
              <EditOutlined /> 编辑信息
            </a-button>
            <a-button block @click="showRoleModal(selectedUser)">
              <UserSwitchOutlined /> 修改角色
            </a-button>
            <a-button block @click="toggleUserStatus(selectedUser)">
              <template v-if="selectedUser.status === 'active'">
                <StopOutlined /> 禁用账号
              </template>
              <template v-else>
                <CheckCircleOutlined /> 启用账号
              </template>
            </a-button>
            <a-button block @click="showResetPasswordModal(selectedUser)">
              <KeyOutlined /> 重置密码
            </a-button>
            <a-button block danger @click="confirmDelete(selectedUser)" :disabled="!canDeleteUser(selectedUser)">
              <DeleteOutlined /> 删除用户
            </a-button>
          </a-space>
        </div>
        <div v-else class="action-section">
          <a-alert message="无权限管理此用户" type="warning" show-icon />
        </div>
      </template>
    </a-drawer>

    <!-- 新增/编辑用户弹窗 -->
    <a-modal v-model:open="editModalVisible" :title="editingId ? '编辑用户' : '新增用户'" @ok="handleSubmit" :confirm-loading="submitting">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="用户名" required v-if="!editingId">
          <a-input v-model:value="form.username" placeholder="请输入用户名" />
        </a-form-item>
        <a-form-item label="密码" required v-if="!editingId">
          <a-input-password v-model:value="form.password" placeholder="请输入密码" />
        </a-form-item>
        <a-form-item label="邮箱" required>
          <a-input v-model:value="form.email" placeholder="请输入邮箱" />
        </a-form-item>
        <a-form-item label="手机号">
          <a-input v-model:value="form.phone" placeholder="请输入手机号" />
        </a-form-item>
        <a-form-item label="角色" required v-if="!editingId">
          <a-select v-model:value="form.role">
            <a-select-option v-if="isSuperAdmin" value="admin">管理员</a-select-option>
            <a-select-option value="user">普通用户</a-select-option>
            <a-select-option value="guest">访客</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态" required v-if="!editingId">
          <a-select v-model:value="form.status">
            <a-select-option value="active">启用</a-select-option>
            <a-select-option value="inactive">禁用</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 修改角色弹窗 -->
    <a-modal v-model:open="roleModalVisible" title="修改角色" @ok="handleRoleChange" :confirm-loading="submitting">
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="用户">{{ selectedUser?.username }}</a-form-item>
        <a-form-item label="当前角色">
          <a-tag :color="getRoleColor(selectedUser?.role || '')">{{ getRoleText(selectedUser?.role || '') }}</a-tag>
        </a-form-item>
        <a-form-item label="新角色" required>
          <a-select v-model:value="newRole" style="width: 200px">
            <a-select-option v-if="isSuperAdmin" value="admin">管理员</a-select-option>
            <a-select-option value="user">普通用户</a-select-option>
            <a-select-option value="guest">访客</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 重置密码弹窗 -->
    <a-modal v-model:open="resetPasswordVisible" title="重置密码" @ok="handleResetPassword" :confirm-loading="resettingPassword">
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="用户">{{ resetPasswordUser?.username }}</a-form-item>
        <a-form-item label="新密码" required>
          <a-input-password v-model:value="newPassword" placeholder="请输入新密码（至少6位）" />
        </a-form-item>
        <a-form-item label="确认密码" required>
          <a-input-password v-model:value="confirmPassword" placeholder="请再次输入新密码" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { 
  PlusOutlined, SearchOutlined, EditOutlined, KeyOutlined, DeleteOutlined,
  UserSwitchOutlined, StopOutlined, CheckCircleOutlined
} from '@ant-design/icons-vue'
import { userApi } from '@/services/user'
import type { User } from '@/types'

const loading = ref(false)
const submitting = ref(false)
const editModalVisible = ref(false)
const roleModalVisible = ref(false)
const detailVisible = ref(false)
const editingId = ref<number | null>(null)
const users = ref<User[]>([])
const selectedUser = ref<User | null>(null)
const newRole = ref('')
const resetPasswordVisible = ref(false)
const resetPasswordUser = ref<User | null>(null)
const newPassword = ref('')
const confirmPassword = ref('')
const resettingPassword = ref(false)

const filters = reactive({
  keyword: '',
  role: undefined as string | undefined,
  status: undefined as string | undefined
})

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

const form = reactive({
  username: '',
  password: '',
  email: '',
  phone: '',
  role: 'user',
  status: 'active'
})

const currentUser = computed(() => {
  const info = localStorage.getItem('userInfo')
  return info ? JSON.parse(info) : null
})

const roleLevel: Record<string, number> = {
  super_admin: 0,
  admin: 1,
  user: 2,
  guest: 3
}

const isSuperAdmin = computed(() => currentUser.value?.role === 'super_admin')
const isAdmin = computed(() => ['super_admin', 'admin'].includes(currentUser.value?.role || ''))
const canCreate = computed(() => isAdmin.value)

const canManageUser = (record: User) => {
  if (!isAdmin.value) return false
  if (record.id === currentUser.value?.id) return false
  const myLevel = roleLevel[currentUser.value?.role || 'guest'] ?? 99
  const targetLevel = roleLevel[record.role] ?? 99
  return myLevel < targetLevel
}

const canDeleteUser = (record: User) => {
  if (!canManageUser(record)) return false
  if (record.role === 'super_admin' || record.id === 1) return false
  return true
}

const columns = [
  { title: '用户名', key: 'username' },
  { title: '邮箱', dataIndex: 'email', key: 'email', ellipsis: true },
  { title: '手机号', dataIndex: 'phone', key: 'phone' },
  { title: '角色', key: 'role', width: 120 },
  { title: '状态', key: 'status', width: 80 },
  { title: '最后登录', dataIndex: 'last_login_at', key: 'last_login_at', width: 180 },
  { title: '操作', key: 'action', width: 80 }
]

const getRoleColor = (role: string) => {
  const colors: Record<string, string> = { super_admin: 'purple', admin: 'red', user: 'blue', guest: 'default' }
  return colors[role] || 'default'
}

const getRoleText = (role: string) => {
  const texts: Record<string, string> = { super_admin: '超级管理员', admin: '管理员', user: '普通用户', guest: '访客' }
  return texts[role] || role
}

const fetchUsers = async () => {
  loading.value = true
  try {
    const response = await userApi.getUsers({
      page: pagination.current,
      page_size: pagination.pageSize,
      keyword: filters.keyword || undefined,
      role: filters.role,
      status: filters.status
    })
    if (response.code === 0 && response.data) {
      users.value = response.data.items
      pagination.total = response.data.total
    }
  } catch (error: any) {
    message.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

const showUserDetail = (record: User) => {
  selectedUser.value = record
  detailVisible.value = true
}

const showCreateModal = () => {
  editingId.value = null
  Object.assign(form, { username: '', password: '', email: '', phone: '', role: 'user', status: 'active' })
  editModalVisible.value = true
}

const showEditModal = (record: User) => {
  editingId.value = record.id
  Object.assign(form, { username: record.username, password: '', email: record.email, phone: record.phone || '', role: record.role, status: record.status })
  editModalVisible.value = true
}

const showRoleModal = (record: User) => {
  selectedUser.value = record
  newRole.value = record.role
  roleModalVisible.value = true
}

const handleSubmit = async () => {
  if (!editingId.value && (!form.username || !form.password)) {
    message.error('请填写用户名和密码')
    return
  }
  if (!form.email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    message.error('请输入有效的邮箱地址')
    return
  }

  submitting.value = true
  try {
    if (editingId.value) {
      await userApi.updateUser(editingId.value, { email: form.email, phone: form.phone })
      message.success('更新成功')
      if (selectedUser.value?.id === editingId.value) {
        selectedUser.value = { ...selectedUser.value, email: form.email, phone: form.phone }
      }
    } else {
      await userApi.createUser({ username: form.username, password: form.password, email: form.email, phone: form.phone, role: form.role, status: form.status })
      message.success('创建成功')
    }
    editModalVisible.value = false
    fetchUsers()
  } catch (error: any) {
    message.error(error.response?.data?.message || error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleRoleChange = async () => {
  if (!selectedUser.value || !newRole.value) return
  submitting.value = true
  try {
    await userApi.updateUserRole(selectedUser.value.id, newRole.value)
    message.success('角色修改成功')
    selectedUser.value = { ...selectedUser.value, role: newRole.value }
    roleModalVisible.value = false
    fetchUsers()
  } catch (error: any) {
    message.error(error.response?.data?.message || error.message || '修改失败')
  } finally {
    submitting.value = false
  }
}

const toggleUserStatus = async (record: User) => {
  const newStatus = record.status === 'active' ? 'inactive' : 'active'
  const action = newStatus === 'active' ? '启用' : '禁用'
  Modal.confirm({
    title: `确认${action}`,
    content: `确定要${action}用户 "${record.username}" 吗？`,
    onOk: async () => {
      try {
        await userApi.updateUserStatus(record.id, newStatus)
        message.success(`${action}成功`)
        if (selectedUser.value?.id === record.id) {
          selectedUser.value = { ...selectedUser.value, status: newStatus }
        }
        fetchUsers()
      } catch (error: any) {
        message.error(error.response?.data?.message || error.message || '操作失败')
      }
    }
  })
}

const handleDelete = async (id: number) => {
  try {
    await userApi.deleteUser(id)
    message.success('删除成功')
    detailVisible.value = false
    fetchUsers()
  } catch (error: any) {
    message.error(error.response?.data?.message || error.message || '删除失败')
  }
}

const confirmDelete = (record: User) => {
  if (!canDeleteUser(record)) return
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除用户 "${record.username}" 吗？此操作不可恢复！`,
    okText: '删除',
    okType: 'danger',
    onOk: () => handleDelete(record.id)
  })
}

const showResetPasswordModal = (record: User) => {
  resetPasswordUser.value = record
  newPassword.value = ''
  confirmPassword.value = ''
  resetPasswordVisible.value = true
}

const handleResetPassword = async () => {
  if (!newPassword.value || newPassword.value.length < 6) {
    message.error('密码至少6位')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    message.error('两次输入的密码不一致')
    return
  }
  if (!resetPasswordUser.value) return

  resettingPassword.value = true
  try {
    await userApi.resetPassword(resetPasswordUser.value.id, { new_password: newPassword.value })
    message.success('密码重置成功')
    resetPasswordVisible.value = false
  } catch (error: any) {
    message.error(error.response?.data?.message || error.message || '重置密码失败')
  } finally {
    resettingPassword.value = false
  }
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  fetchUsers()
}

onMounted(() => fetchUsers())
</script>

<style scoped>
.user-header {
  display: flex;
  align-items: center;
  gap: 16px;
}
.user-header-info h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
}
.action-section {
  margin-top: 16px;
}
.action-section h4 {
  margin-bottom: 12px;
  color: #666;
}
</style>

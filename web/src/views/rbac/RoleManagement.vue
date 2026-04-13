<template>
  <div class="role-management">
    <a-tabs v-model:activeKey="activeTab">
      <!-- 角色管理 Tab -->
      <a-tab-pane key="roles" tab="角色管理">
        <a-card :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showCreateModal">
              <PlusOutlined /> 新建角色
            </a-button>
          </template>

          <a-table
            :columns="columns"
            :data-source="roles"
            :loading="loading"
            :pagination="pagination"
            @change="handleTableChange"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'is_system'">
                <a-tag :color="record.is_system ? 'blue' : 'default'">
                  {{ record.is_system ? '系统内置' : '自定义' }}
                </a-tag>
              </template>
              <template v-if="column.key === 'status'">
                <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '启用' : '禁用'" />
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showPermissionModal(record)">
                    <KeyOutlined /> 权限
                  </a-button>
                  <a-button type="link" size="small" @click="showUserAssignModal(record)">
                    <UserOutlined /> 用户
                  </a-button>
                  <a-button type="link" size="small" @click="showEditModal(record)" :disabled="record.is_system">
                    <EditOutlined /> 编辑
                  </a-button>
                  <a-popconfirm
                    title="确定删除该角色？"
                    @confirm="handleDelete(record.id)"
                    :disabled="record.is_system"
                  >
                    <a-button type="link" size="small" danger :disabled="record.is_system">
                      <DeleteOutlined /> 删除
                    </a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 用户授权 Tab -->
      <a-tab-pane key="users" tab="用户授权">
        <a-card :bordered="false">
          <a-form layout="inline" style="margin-bottom: 16px">
            <a-form-item>
              <a-input 
                v-model:value="userSearchKeyword" 
                placeholder="搜索用户名/邮箱" 
                allowClear 
                style="width: 200px"
                @pressEnter="fetchUsers"
              >
                <template #prefix><SearchOutlined /></template>
              </a-input>
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="fetchUsers">查询</a-button>
            </a-form-item>
          </a-form>

          <a-table
            :columns="userColumns"
            :data-source="users"
            :loading="usersLoading"
            :pagination="userPagination"
            @change="handleUserTableChange"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'roles'">
                <a-tag v-for="role in record.roles" :key="role.id" color="blue" style="margin: 2px">
                  {{ role.display_name || role.name }}
                </a-tag>
                <span v-if="!record.roles?.length" style="color: #999">未分配角色</span>
              </template>
              <template v-if="column.key === 'status'">
                <a-badge :status="record.status === 'active' ? 'success' : 'error'" :text="record.status === 'active' ? '启用' : '禁用'" />
              </template>
              <template v-if="column.key === 'action'">
                <a-button type="link" size="small" @click="showUserRoleModal(record)">
                  <SettingOutlined /> 分配角色
                </a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 创建/编辑角色弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="editingRole ? '编辑角色' : '新建角色'"
      @ok="handleSubmit"
      :confirm-loading="submitting"
    >
      <a-form :model="formData" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="角色标识" required>
          <a-input v-model:value="formData.name" placeholder="如：manager" :disabled="!!editingRole" />
        </a-form-item>
        <a-form-item label="显示名称" required>
          <a-input v-model:value="formData.display_name" placeholder="如：项目经理" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="formData.description" :rows="3" placeholder="角色描述" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 权限配置弹窗 -->
    <a-modal
      v-model:open="permissionModalVisible"
      title="权限配置"
      width="700px"
      @ok="handleSavePermissions"
      :confirm-loading="savingPermissions"
    >
      <div v-if="currentRole" style="margin-bottom: 16px">
        <a-descriptions :column="2" size="small">
          <a-descriptions-item label="角色">{{ currentRole.display_name }}</a-descriptions-item>
          <a-descriptions-item label="标识">{{ currentRole.name }}</a-descriptions-item>
        </a-descriptions>
      </div>
      
      <a-table
        :columns="permissionColumns"
        :data-source="groupedPermissions"
        :pagination="false"
        size="small"
        row-key="resource"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'permissions'">
            <a-checkbox-group v-model:value="selectedPermissions[record.resource]">
              <a-checkbox
                v-for="perm in record.permissions"
                :key="perm.id"
                :value="perm.id"
              >
                {{ perm.display_name || perm.action }}
              </a-checkbox>
            </a-checkbox-group>
          </template>
        </template>
      </a-table>
    </a-modal>

    <!-- 用户角色分配弹窗（从角色视角） -->
    <a-modal
      v-model:open="userAssignModalVisible"
      title="分配用户"
      width="600px"
      @ok="handleSaveRoleUsers"
      :confirm-loading="savingRoleUsers"
    >
      <div v-if="currentRole" style="margin-bottom: 16px">
        <a-descriptions :column="2" size="small">
          <a-descriptions-item label="角色">{{ currentRole.display_name }}</a-descriptions-item>
          <a-descriptions-item label="标识">{{ currentRole.name }}</a-descriptions-item>
        </a-descriptions>
      </div>
      
      <a-transfer
        v-model:target-keys="selectedUserIds"
        :data-source="allUsers"
        :titles="['可选用户', '已分配用户']"
        :render="item => item.title"
        :list-style="{ width: '220px', height: '300px' }"
        show-search
        :filter-option="filterUserOption"
      />
    </a-modal>

    <!-- 用户角色分配弹窗（从用户视角） -->
    <a-modal
      v-model:open="userRoleModalVisible"
      title="分配角色"
      width="500px"
      @ok="handleSaveUserRoles"
      :confirm-loading="savingUserRoles"
    >
      <div v-if="currentUser" style="margin-bottom: 16px">
        <a-descriptions :column="2" size="small">
          <a-descriptions-item label="用户">{{ currentUser.username }}</a-descriptions-item>
          <a-descriptions-item label="邮箱">{{ currentUser.email || '-' }}</a-descriptions-item>
        </a-descriptions>
      </div>
      
      <a-checkbox-group v-model:value="selectedRoleIds" style="width: 100%">
        <a-row>
          <a-col :span="12" v-for="role in roles" :key="role.id" style="margin-bottom: 8px">
            <a-checkbox :value="role.id">
              <a-tag :color="role.is_system ? 'blue' : 'default'" style="margin-right: 4px">
                {{ role.is_system ? '系统' : '自定义' }}
              </a-tag>
              {{ role.display_name }}
            </a-checkbox>
          </a-col>
        </a-row>
      </a-checkbox-group>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, EditOutlined, DeleteOutlined, KeyOutlined, UserOutlined, SearchOutlined, SettingOutlined } from '@ant-design/icons-vue'
import request from '@/utils/request'

interface Role {
  id: number
  name: string
  display_name: string
  description: string
  is_system: boolean
  status: string
}

interface Permission {
  id: number
  name: string
  display_name: string
  resource: string
  action: string
}

interface User {
  id: number
  username: string
  email: string
  status: string
  roles?: Role[]
}

interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

const activeTab = ref('roles')
const loading = ref(false)
const submitting = ref(false)
const savingPermissions = ref(false)
const savingRoleUsers = ref(false)
const savingUserRoles = ref(false)
const usersLoading = ref(false)
const modalVisible = ref(false)
const permissionModalVisible = ref(false)
const userAssignModalVisible = ref(false)
const userRoleModalVisible = ref(false)
const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const users = ref<User[]>([])
const allUsers = ref<{ key: string; title: string }[]>([])
const editingRole = ref<Role | null>(null)
const currentRole = ref<Role | null>(null)
const currentUser = ref<User | null>(null)
const selectedPermissions = ref<Record<string, number[]>>({})
const selectedUserIds = ref<string[]>([])
const selectedRoleIds = ref<number[]>([])
const userSearchKeyword = ref('')

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
})

const userPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
})

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '角色标识', dataIndex: 'name', key: 'name' },
  { title: '显示名称', dataIndex: 'display_name', key: 'display_name' },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '类型', dataIndex: 'is_system', key: 'is_system', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 250 },
]

const userColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '角色', key: 'roles', width: 300 },
  { title: '状态', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 120 },
]

const permissionColumns = [
  { title: '资源', dataIndex: 'resource', key: 'resource', width: 120 },
  { title: '权限', key: 'permissions' },
]

const formData = reactive({
  name: '',
  display_name: '',
  description: '',
})

const groupedPermissions = computed(() => {
  const groups: Record<string, { resource: string; permissions: Permission[] }> = {}
  permissions.value.forEach(p => {
    if (!groups[p.resource]) {
      groups[p.resource] = { resource: p.resource, permissions: [] }
    }
    groups[p.resource].permissions.push(p)
  })
  return Object.values(groups)
})

const filterUserOption = (inputValue: string, option: any) => {
  return option.title.toLowerCase().includes(inputValue.toLowerCase())
}

const fetchRoles = async () => {
  loading.value = true
  try {
    const res = await request.get<any, ApiResponse>('/rbac/roles', {
      params: { page: pagination.current, page_size: pagination.pageSize }
    })
    const data = res.data || res
    roles.value = data?.list || (Array.isArray(data) ? data : [])
    pagination.total = data?.total || roles.value.length
  } catch (e) {
    console.error('获取角色列表失败', e)
  } finally {
    loading.value = false
  }
}

const fetchPermissions = async () => {
  try {
    const res = await request.get<any, ApiResponse>('/rbac/permissions')
    const data = res.data || res
    permissions.value = data?.list || (Array.isArray(data) ? data : [])
  } catch (e) {
    console.error('获取权限列表失败', e)
  }
}

const fetchRolePermissions = async (roleId: number) => {
  try {
    const res = await request.get<any, ApiResponse>(`/rbac/roles/${roleId}/permissions`)
    const data = res.data || res
    const perms: Permission[] = data?.permissions || []
    selectedPermissions.value = {}
    perms.forEach((p: Permission) => {
      if (!selectedPermissions.value[p.resource]) {
        selectedPermissions.value[p.resource] = []
      }
      selectedPermissions.value[p.resource].push(p.id)
    })
  } catch (e) {
    console.error('获取角色权限失败', e)
  }
}

const fetchUsers = async () => {
  usersLoading.value = true
  try {
    const res = await request.get<any, ApiResponse>('/users', {
      params: { 
        page: userPagination.current, 
        page_size: userPagination.pageSize,
        keyword: userSearchKeyword.value || undefined
      }
    })
    const data = res.data || res
    const userList = data?.items || data?.list || (Array.isArray(data) ? data : [])
    
    // 获取每个用户的角色
    for (const user of userList) {
      try {
        const roleRes = await request.get<any, ApiResponse>(`/rbac/users/${user.id}/roles`)
        const roleData = roleRes.data || roleRes
        user.roles = roleData?.roles || []
      } catch (e) {
        user.roles = []
      }
    }
    
    users.value = userList
    userPagination.total = data?.total || userList.length
  } catch (e) {
    console.error('获取用户列表失败', e)
  } finally {
    usersLoading.value = false
  }
}

const fetchAllUsers = async () => {
  try {
    const res = await request.get<any, ApiResponse>('/users', {
      params: { page: 1, page_size: 1000 }
    })
    const data = res.data || res
    const userList = data?.items || data?.list || (Array.isArray(data) ? data : [])
    allUsers.value = userList.map((u: User) => ({
      key: String(u.id),
      title: `${u.username} (${u.email || '-'})`
    }))
  } catch (e) {
    console.error('获取所有用户失败', e)
  }
}

const fetchRoleUsers = async (roleId: number) => {
  try {
    // 获取所有用户
    await fetchAllUsers()
    
    // 获取该角色下的用户（通过遍历用户获取）
    const res = await request.get<any, ApiResponse>('/users', {
      params: { page: 1, page_size: 1000 }
    })
    const data = res.data || res
    const userList = data?.items || data?.list || (Array.isArray(data) ? data : [])
    
    const usersWithRole: string[] = []
    for (const user of userList) {
      try {
        const roleRes = await request.get<any, ApiResponse>(`/rbac/users/${user.id}/roles`)
        const roleData = roleRes.data || roleRes
        const userRoles = roleData?.roles || []
        if (userRoles.some((r: Role) => r.id === roleId)) {
          usersWithRole.push(String(user.id))
        }
      } catch (e) {
        // ignore
      }
    }
    selectedUserIds.value = usersWithRole
  } catch (e) {
    console.error('获取角色用户失败', e)
  }
}

const fetchUserRoles = async (userId: number) => {
  try {
    const res = await request.get<any, ApiResponse>(`/rbac/users/${userId}/roles`)
    const data = res.data || res
    const userRoles = data?.roles || []
    selectedRoleIds.value = userRoles.map((r: Role) => r.id)
  } catch (e) {
    console.error('获取用户角色失败', e)
    selectedRoleIds.value = []
  }
}

const showCreateModal = () => {
  editingRole.value = null
  formData.name = ''
  formData.display_name = ''
  formData.description = ''
  modalVisible.value = true
}

const showEditModal = (role: Role) => {
  editingRole.value = role
  formData.name = role.name
  formData.display_name = role.display_name
  formData.description = role.description
  modalVisible.value = true
}

const showPermissionModal = async (role: Role) => {
  currentRole.value = role
  selectedPermissions.value = {}
  await fetchRolePermissions(role.id)
  permissionModalVisible.value = true
}

const showUserAssignModal = async (role: Role) => {
  currentRole.value = role
  selectedUserIds.value = []
  await fetchRoleUsers(role.id)
  userAssignModalVisible.value = true
}

const showUserRoleModal = async (user: User) => {
  currentUser.value = user
  selectedRoleIds.value = []
  await fetchUserRoles(user.id)
  userRoleModalVisible.value = true
}

const handleSubmit = async () => {
  if (!formData.name || !formData.display_name) {
    message.warning('请填写必填项')
    return
  }
  
  submitting.value = true
  try {
    if (editingRole.value) {
      await request.put(`/rbac/roles/${editingRole.value.id}`, formData)
      message.success('更新成功')
    } else {
      await request.post('/rbac/roles', formData)
      message.success('创建成功')
    }
    modalVisible.value = false
    fetchRoles()
  } catch (e) {
    // error handled by interceptor
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await request.delete(`/rbac/roles/${id}`)
    message.success('删除成功')
    fetchRoles()
  } catch (e) {
    // error handled by interceptor
  }
}

const handleSavePermissions = async () => {
  if (!currentRole.value) return
  
  savingPermissions.value = true
  try {
    const permissionIds: number[] = []
    Object.values(selectedPermissions.value).forEach(ids => {
      permissionIds.push(...ids)
    })
    
    await request.put(`/rbac/roles/${currentRole.value.id}/permissions`, {
      permission_ids: permissionIds
    })
    message.success('权限保存成功')
    permissionModalVisible.value = false
  } catch (e) {
    // error handled by interceptor
  } finally {
    savingPermissions.value = false
  }
}

const handleSaveRoleUsers = async () => {
  if (!currentRole.value) return
  
  savingRoleUsers.value = true
  try {
    // 获取当前角色的用户列表
    const currentUserIds = new Set(selectedUserIds.value.map(id => parseInt(id)))
    
    // 获取所有用户，更新他们的角色
    for (const userItem of allUsers.value) {
      const userId = parseInt(userItem.key)
      const shouldHaveRole = currentUserIds.has(userId)
      
      // 获取用户当前角色
      const roleRes = await request.get<any, ApiResponse>(`/rbac/users/${userId}/roles`)
      const roleData = roleRes.data || roleRes
      const currentRoles = roleData?.roles || []
      const currentRoleIds = currentRoles.map((r: Role) => r.id)
      
      const hasRole = currentRoleIds.includes(currentRole.value!.id)
      
      if (shouldHaveRole && !hasRole) {
        // 添加角色
        await request.put(`/rbac/users/${userId}/roles`, {
          role_ids: [...currentRoleIds, currentRole.value!.id]
        })
      } else if (!shouldHaveRole && hasRole) {
        // 移除角色
        await request.put(`/rbac/users/${userId}/roles`, {
          role_ids: currentRoleIds.filter((id: number) => id !== currentRole.value!.id)
        })
      }
    }
    
    message.success('用户分配成功')
    userAssignModalVisible.value = false
  } catch (e) {
    console.error('保存角色用户失败', e)
    message.error('保存失败')
  } finally {
    savingRoleUsers.value = false
  }
}

const handleSaveUserRoles = async () => {
  if (!currentUser.value) return
  
  savingUserRoles.value = true
  try {
    await request.put(`/rbac/users/${currentUser.value.id}/roles`, {
      role_ids: selectedRoleIds.value
    })
    message.success('角色分配成功')
    userRoleModalVisible.value = false
    fetchUsers()
  } catch (e) {
    // error handled by interceptor
  } finally {
    savingUserRoles.value = false
  }
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  fetchRoles()
}

const handleUserTableChange = (pag: any) => {
  userPagination.current = pag.current
  userPagination.pageSize = pag.pageSize
  fetchUsers()
}

onMounted(() => {
  fetchRoles()
  fetchPermissions()
  fetchUsers()
})
</script>

<style scoped>
.role-management {
  padding: 0;
}
.role-management :deep(.ant-tabs-nav) {
  margin-bottom: 16px;
}
</style>

import request from '@/utils/request'

export interface Role {
  id: number
  name: string
  display_name: string
  description: string
  is_system: boolean
  status: string
}

export interface Permission {
  id: number
  name: string
  display_name: string
  resource: string
  action: string
  description: string
}

// 获取角色列表
export const getRoles = (params?: { page?: number; page_size?: number }) => {
  return request.get('/rbac/roles', { params })
}

// 获取单个角色
export const getRole = (id: number) => {
  return request.get(`/rbac/roles/${id}`)
}

// 创建角色
export const createRole = (data: Partial<Role>) => {
  return request.post('/rbac/roles', data)
}

// 更新角色
export const updateRole = (id: number, data: Partial<Role>) => {
  return request.put(`/rbac/roles/${id}`, data)
}

// 删除角色
export const deleteRole = (id: number) => {
  return request.delete(`/rbac/roles/${id}`)
}

// 获取权限列表
export const getPermissions = () => {
  return request.get('/rbac/permissions')
}

// 获取角色的权限
export const getRolePermissions = (roleId: number) => {
  return request.get(`/rbac/roles/${roleId}/permissions`)
}

// 更新角色的权限
export const updateRolePermissions = (roleId: number, permissionIds: number[]) => {
  return request.put(`/rbac/roles/${roleId}/permissions`, { permission_ids: permissionIds })
}

// 获取用户的角色
export const getUserRoles = (userId: number) => {
  return request.get(`/rbac/users/${userId}/roles`)
}

// 更新用户的角色
export const updateUserRoles = (userId: number, roleIds: number[]) => {
  return request.put(`/rbac/users/${userId}/roles`, { role_ids: roleIds })
}

// roleApi 对象导出（兼容旧代码）
export const roleApi = {
  getRoles,
  getRole,
  createRole,
  updateRole,
  deleteRole,
  getPermissions,
  getRolePermissions,
  updateRolePermissions,
  getUserRoles,
  updateUserRoles,
}

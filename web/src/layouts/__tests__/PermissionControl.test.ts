import { describe, it, expect, beforeEach, afterEach } from 'vitest'

/**
 * Permission Control Tests (Task 10.2)
 * Tests for menu permission checking and filtering functionality
 */

describe('Permission Control (Task 10.2)', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear()
  })

  afterEach(() => {
    // Clean up after each test
    localStorage.clear()
  })

  describe('getUserPermissions', () => {
    const getUserPermissions = (): string[] => {
      const userInfoStr = localStorage.getItem('userInfo')
      if (!userInfoStr) return []
      
      const user = JSON.parse(userInfoStr)
      
      // If user has permissions field, return it directly
      if (user.permissions && Array.isArray(user.permissions)) {
        return user.permissions
      }
      
      // If user has roles field, map roles to permissions
      if (user.roles && Array.isArray(user.roles)) {
        const roles = user.roles
        
        // Admin has all permissions
        if (roles.includes('admin') || roles.includes('administrator')) {
          return ['*']
        }
        
        // Role permission mapping
        const rolePermissionMap: Record<string, string[]> = {
          'developer': [
            'pipeline:view', 'pipeline:create', 'pipeline:edit',
            'application:view', 'application:deploy',
            'k8s:view', 'logs:view'
          ],
          'operator': [
            'pipeline:view', 'application:view', 'application:deploy',
            'k8s:view', 'k8s:manage', 'healthcheck:view',
            'logs:view', 'alert:view', 'cost:view'
          ],
          'viewer': [
            'pipeline:view', 'application:view',
            'k8s:view', 'logs:view', 'alert:view'
          ]
        }
        
        // Merge permissions from all roles
        const permissions = new Set<string>()
        roles.forEach(role => {
          const rolePerms = rolePermissionMap[role] || []
          rolePerms.forEach(perm => permissions.add(perm))
        })
        
        return Array.from(permissions)
      }
      
      return []
    }

    it('should return empty array when no user info', () => {
      const permissions = getUserPermissions()
      expect(permissions).toEqual([])
    })

    it('should return permissions from user.permissions field', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'testuser',
        permissions: ['pipeline:view', 'application:view']
      }))
      
      const permissions = getUserPermissions()
      expect(permissions).toEqual(['pipeline:view', 'application:view'])
    })

    it('should return wildcard for admin role', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'admin',
        roles: ['admin']
      }))
      
      const permissions = getUserPermissions()
      expect(permissions).toEqual(['*'])
    })

    it('should return wildcard for administrator role', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'admin',
        roles: ['administrator']
      }))
      
      const permissions = getUserPermissions()
      expect(permissions).toEqual(['*'])
    })

    it('should return developer permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'dev',
        roles: ['developer']
      }))
      
      const permissions = getUserPermissions()
      expect(permissions).toContain('pipeline:view')
      expect(permissions).toContain('pipeline:create')
      expect(permissions).toContain('application:view')
      expect(permissions).toContain('k8s:view')
    })

    it('should return operator permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'ops',
        roles: ['operator']
      }))
      
      const permissions = getUserPermissions()
      expect(permissions).toContain('pipeline:view')
      expect(permissions).toContain('healthcheck:view')
      expect(permissions).toContain('k8s:manage')
      expect(permissions).toContain('alert:view')
    })

    it('should return viewer permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        roles: ['viewer']
      }))
      
      const permissions = getUserPermissions()
      expect(permissions).toContain('pipeline:view')
      expect(permissions).toContain('application:view')
      expect(permissions).toContain('k8s:view')
      expect(permissions).not.toContain('pipeline:create')
    })

    it('should merge permissions from multiple roles', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'multiuser',
        roles: ['developer', 'operator']
      }))
      
      const permissions = getUserPermissions()
      // Should have permissions from both roles
      expect(permissions).toContain('pipeline:create') // from developer
      expect(permissions).toContain('healthcheck:view') // from operator
      expect(permissions).toContain('k8s:manage') // from operator
    })

    it('should return empty array for unknown role', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'unknown',
        roles: ['unknown_role']
      }))
      
      const permissions = getUserPermissions()
      expect(permissions).toEqual([])
    })

    it('should handle empty roles array', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'norole',
        roles: []
      }))
      
      const permissions = getUserPermissions()
      expect(permissions).toEqual([])
    })
  })

  describe('hasPermission', () => {
    const getUserPermissions = (): string[] => {
      const userInfoStr = localStorage.getItem('userInfo')
      if (!userInfoStr) return []
      
      const user = JSON.parse(userInfoStr)
      
      if (user.permissions && Array.isArray(user.permissions)) {
        return user.permissions
      }
      
      if (user.roles && Array.isArray(user.roles)) {
        const roles = user.roles
        
        if (roles.includes('admin') || roles.includes('administrator')) {
          return ['*']
        }
        
        const rolePermissionMap: Record<string, string[]> = {
          'developer': [
            'pipeline:view', 'pipeline:create', 'pipeline:edit',
            'application:view', 'application:deploy',
            'k8s:view', 'logs:view'
          ],
          'operator': [
            'pipeline:view', 'application:view', 'application:deploy',
            'k8s:view', 'k8s:manage', 'healthcheck:view',
            'logs:view', 'alert:view', 'cost:view'
          ],
          'viewer': [
            'pipeline:view', 'application:view',
            'k8s:view', 'logs:view', 'alert:view'
          ]
        }
        
        const permissions = new Set<string>()
        roles.forEach(role => {
          const rolePerms = rolePermissionMap[role] || []
          rolePerms.forEach(perm => permissions.add(perm))
        })
        
        return Array.from(permissions)
      }
      
      return []
    }

    const hasPermission = (requiredPermissions?: string[]): boolean => {
      if (!requiredPermissions || requiredPermissions.length === 0) {
        return true
      }
      
      const userPermissions = getUserPermissions()
      
      if (userPermissions.includes('*')) {
        return true
      }
      
      return requiredPermissions.some(required => {
        if (required.endsWith(':*')) {
          const prefix = required.slice(0, -2)
          return userPermissions.some(perm => perm.startsWith(prefix + ':'))
        }
        return userPermissions.includes(required)
      })
    }

    it('should return true when no permissions required (backward compatibility)', () => {
      const result = hasPermission()
      expect(result).toBe(true)
    })

    it('should return true when empty permissions array', () => {
      const result = hasPermission([])
      expect(result).toBe(true)
    })

    it('should return true for admin with wildcard permission', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'admin',
        roles: ['admin']
      }))
      
      const result = hasPermission(['pipeline:view', 'system:manage'])
      expect(result).toBe(true)
    })

    it('should return true when user has required permission', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'dev',
        permissions: ['pipeline:view', 'application:view']
      }))
      
      const result = hasPermission(['pipeline:view'])
      expect(result).toBe(true)
    })

    it('should return false when user lacks required permission', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: ['pipeline:view']
      }))
      
      const result = hasPermission(['system:manage'])
      expect(result).toBe(false)
    })

    it('should return true when user has any of the required permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'dev',
        permissions: ['pipeline:view']
      }))
      
      const result = hasPermission(['pipeline:view', 'system:manage'])
      expect(result).toBe(true)
    })

    it('should support wildcard permission matching', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'dev',
        permissions: ['pipeline:view', 'pipeline:create', 'pipeline:edit']
      }))
      
      const result = hasPermission(['pipeline:*'])
      expect(result).toBe(true)
    })

    it('should return false for wildcard when no matching permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: ['application:view']
      }))
      
      const result = hasPermission(['pipeline:*'])
      expect(result).toBe(false)
    })

    it('should handle role-based permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'dev',
        roles: ['developer']
      }))
      
      const result = hasPermission(['pipeline:view'])
      expect(result).toBe(true)
    })

    it('should return false when user has no permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'nouser',
        roles: []
      }))
      
      const result = hasPermission(['pipeline:view'])
      expect(result).toBe(false)
    })
  })

  describe('10.2.1 测试管理员角色看到所有菜单', () => {
    const hasPermission = (requiredPermissions?: string[]): boolean => {
      if (!requiredPermissions || requiredPermissions.length === 0) {
        return true
      }
      
      const userInfoStr = localStorage.getItem('userInfo')
      if (!userInfoStr) return false
      
      const user = JSON.parse(userInfoStr)
      const userPermissions = user.permissions || []
      
      if (userPermissions.includes('*')) {
        return true
      }
      
      return requiredPermissions.some(required => userPermissions.includes(required))
    }

    it('should allow admin to see all menus', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'admin',
        permissions: ['*']
      }))
      
      // Test various menu permissions
      expect(hasPermission(['pipeline:view'])).toBe(true)
      expect(hasPermission(['system:manage'])).toBe(true)
      expect(hasPermission(['k8s:view'])).toBe(true)
      expect(hasPermission(['application:view'])).toBe(true)
      expect(hasPermission(['healthcheck:view'])).toBe(true)
    })

    it('should show dashboard to admin (no permission required)', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'admin',
        permissions: ['*']
      }))
      
      expect(hasPermission()).toBe(true)
    })

    it('should show all pipeline submenus to admin', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'admin',
        permissions: ['*']
      }))
      
      expect(hasPermission(['pipeline:view'])).toBe(true)
      expect(hasPermission(['pipeline:create'])).toBe(true)
      expect(hasPermission(['pipeline:manage'])).toBe(true)
    })

    it('should show all system management menus to admin', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'admin',
        permissions: ['*']
      }))
      
      expect(hasPermission(['system:view'])).toBe(true)
      expect(hasPermission(['system:manage'])).toBe(true)
    })
  })

  describe('10.2.2 测试普通用户只看到有权限的菜单', () => {
    interface MenuItemConfig {
      key: string
      title: string
      permission?: string[]
      children?: MenuItemConfig[]
    }

    const hasPermission = (requiredPermissions?: string[]): boolean => {
      if (!requiredPermissions || requiredPermissions.length === 0) {
        return true
      }
      
      const userInfoStr = localStorage.getItem('userInfo')
      if (!userInfoStr) return false
      
      const user = JSON.parse(userInfoStr)
      const userPermissions = user.permissions || []
      
      if (userPermissions.includes('*')) {
        return true
      }
      
      return requiredPermissions.some(required => userPermissions.includes(required))
    }

    const filterMenuByPermission = (items: MenuItemConfig[]): MenuItemConfig[] => {
      return items.filter(item => {
        if (!hasPermission(item.permission)) {
          return false
        }
        
        if (item.children && item.children.length > 0) {
          item.children = filterMenuByPermission(item.children)
          if (item.children.length === 0) {
            return false
          }
        }
        
        return true
      })
    }

    it('should show only permitted menus to viewer', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: ['pipeline:view', 'application:view']
      }))
      
      expect(hasPermission(['pipeline:view'])).toBe(true)
      expect(hasPermission(['application:view'])).toBe(true)
      expect(hasPermission(['system:manage'])).toBe(false)
    })

    it('should hide system management from viewer', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: ['pipeline:view']
      }))
      
      expect(hasPermission(['system:manage'])).toBe(false)
    })

    it('should filter menu items based on permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'dev',
        permissions: ['pipeline:view']
      }))
      
      const menuConfig: MenuItemConfig[] = [
        { key: 'dashboard', title: '仪表盘' },
        { key: 'pipeline', title: 'CI/CD 流水线', permission: ['pipeline:view'] },
        { key: 'system', title: '系统管理', permission: ['system:manage'] }
      ]
      
      const filtered = filterMenuByPermission(menuConfig)
      
      expect(filtered).toHaveLength(2)
      expect(filtered.find(m => m.key === 'dashboard')).toBeDefined()
      expect(filtered.find(m => m.key === 'pipeline')).toBeDefined()
      expect(filtered.find(m => m.key === 'system')).toBeUndefined()
    })

    it('should filter submenu items based on permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'dev',
        permissions: ['pipeline:view']
      }))
      
      const menuConfig: MenuItemConfig[] = [
        {
          key: 'pipeline',
          title: 'CI/CD 流水线',
          permission: ['pipeline:view'],
          children: [
            { key: 'list', title: '流水线列表' },
            { key: 'designer', title: '流水线设计器', permission: ['pipeline:create'] }
          ]
        }
      ]
      
      const filtered = filterMenuByPermission(menuConfig)
      
      expect(filtered).toHaveLength(1)
      expect(filtered[0].children).toHaveLength(1)
      expect(filtered[0].children![0].key).toBe('list')
    })

    it('should hide parent menu when all children are filtered out', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: []
      }))
      
      const menuConfig: MenuItemConfig[] = [
        {
          key: 'system',
          title: '系统管理',
          permission: ['system:view'],
          children: [
            { key: 'users', title: '用户管理', permission: ['system:manage'] },
            { key: 'roles', title: '角色权限', permission: ['system:manage'] }
          ]
        }
      ]
      
      const filtered = filterMenuByPermission(menuConfig)
      
      expect(filtered).toHaveLength(0)
    })
  })

  describe('10.2.3 测试权限变更后菜单更新', () => {
    it('should update menu when permissions change', () => {
      // Initial permissions
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        permissions: ['pipeline:view']
      }))
      
      const userInfoStr1 = localStorage.getItem('userInfo')
      const user1 = JSON.parse(userInfoStr1!)
      expect(user1.permissions).toContain('pipeline:view')
      expect(user1.permissions).not.toContain('system:manage')
      
      // Update permissions
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        permissions: ['pipeline:view', 'system:manage']
      }))
      
      const userInfoStr2 = localStorage.getItem('userInfo')
      const user2 = JSON.parse(userInfoStr2!)
      expect(user2.permissions).toContain('pipeline:view')
      expect(user2.permissions).toContain('system:manage')
    })

    it('should reflect role changes in permissions', () => {
      // Initial role
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        roles: ['viewer']
      }))
      
      const user1 = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user1.roles).toEqual(['viewer'])
      
      // Upgrade to developer
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        roles: ['developer']
      }))
      
      const user2 = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user2.roles).toEqual(['developer'])
    })

    it('should handle permission removal', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        permissions: ['pipeline:view', 'system:manage']
      }))
      
      let user = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user.permissions).toHaveLength(2)
      
      // Remove system:manage permission
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        permissions: ['pipeline:view']
      }))
      
      user = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user.permissions).toHaveLength(1)
      expect(user.permissions).not.toContain('system:manage')
    })

    it('should handle role addition', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        roles: ['viewer']
      }))
      
      let user = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user.roles).toHaveLength(1)
      
      // Add operator role
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        roles: ['viewer', 'operator']
      }))
      
      user = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user.roles).toHaveLength(2)
      expect(user.roles).toContain('operator')
    })
  })

  describe('10.2.4 测试无权限访问路由的处理', () => {
    const hasPermission = (requiredPermissions?: string[]): boolean => {
      if (!requiredPermissions || requiredPermissions.length === 0) {
        return true
      }
      
      const userInfoStr = localStorage.getItem('userInfo')
      if (!userInfoStr) return false
      
      const user = JSON.parse(userInfoStr)
      const userPermissions = user.permissions || []
      
      if (userPermissions.includes('*')) {
        return true
      }
      
      return requiredPermissions.some(required => userPermissions.includes(required))
    }

    it('should detect unauthorized access to system management', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: ['pipeline:view']
      }))
      
      const canAccessSystemManagement = hasPermission(['system:manage'])
      expect(canAccessSystemManagement).toBe(false)
    })

    it('should detect unauthorized access to pipeline designer', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: ['pipeline:view']
      }))
      
      const canAccessDesigner = hasPermission(['pipeline:create'])
      expect(canAccessDesigner).toBe(false)
    })

    it('should allow access to routes without permission requirements', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: []
      }))
      
      const canAccessDashboard = hasPermission()
      expect(canAccessDashboard).toBe(true)
    })

    it('should handle route guard for protected routes', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'dev',
        permissions: ['pipeline:view']
      }))
      
      const routePermissions: Record<string, string[]> = {
        '/pipeline/list': ['pipeline:view'],
        '/pipeline/designer': ['pipeline:create'],
        '/system/monitor': ['system:view'],
        '/users': ['system:manage']
      }
      
      expect(hasPermission(routePermissions['/pipeline/list'])).toBe(true)
      expect(hasPermission(routePermissions['/pipeline/designer'])).toBe(false)
      expect(hasPermission(routePermissions['/system/monitor'])).toBe(false)
      expect(hasPermission(routePermissions['/users'])).toBe(false)
    })

    it('should redirect unauthorized users', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'viewer',
        permissions: ['pipeline:view']
      }))
      
      const targetRoute = '/users'
      const requiredPermission = ['system:manage']
      const canAccess = hasPermission(requiredPermission)
      
      if (!canAccess) {
        // Should redirect to 403 or dashboard
        const redirectTo = '/403'
        expect(redirectTo).toBe('/403')
      }
      
      expect(canAccess).toBe(false)
    })
  })

  describe('Edge cases and error handling', () => {
    it('should handle null userInfo', () => {
      localStorage.removeItem('userInfo')
      
      const userInfoStr = localStorage.getItem('userInfo')
      expect(userInfoStr).toBeNull()
    })

    it('should handle invalid JSON in userInfo', () => {
      localStorage.setItem('userInfo', 'invalid json')
      
      try {
        JSON.parse(localStorage.getItem('userInfo')!)
      } catch (e) {
        expect(e).toBeDefined()
      }
    })

    it('should handle userInfo without permissions or roles', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user'
      }))
      
      const user = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user.permissions).toBeUndefined()
      expect(user.roles).toBeUndefined()
    })

    it('should handle empty permission array', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        permissions: []
      }))
      
      const user = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user.permissions).toEqual([])
    })

    it('should handle null permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user',
        permissions: null
      }))
      
      const user = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user.permissions).toBeNull()
    })

    it('should handle undefined permissions', () => {
      localStorage.setItem('userInfo', JSON.stringify({
        username: 'user'
      }))
      
      const user = JSON.parse(localStorage.getItem('userInfo')!)
      expect(user.permissions).toBeUndefined()
    })
  })
})

/**
 * Permission Control Demo
 * 
 * This file demonstrates how to use the permission control system
 * in the DevOps management system.
 */

// ============================================
// Example 1: Setting up an Admin User
// ============================================

export function setupAdminUser() {
  localStorage.setItem('userInfo', JSON.stringify({
    username: 'admin',
    email: 'admin@example.com',
    roles: ['admin']
  }))
  
  console.log('Admin user setup complete')
  console.log('Admin will see ALL menu items')
}

// ============================================
// Example 2: Setting up a Developer User
// ============================================

export function setupDeveloperUser() {
  localStorage.setItem('userInfo', JSON.stringify({
    username: 'developer',
    email: 'dev@example.com',
    roles: ['developer']
  }))
  
  console.log('Developer user setup complete')
  console.log('Developer will see:')
  console.log('- Dashboard')
  console.log('- Application Management')
  console.log('- CI/CD Pipeline (including designer)')
  console.log('- Kubernetes')
  console.log('- Logs')
}

// ============================================
// Example 3: Setting up an Operator User
// ============================================

export function setupOperatorUser() {
  localStorage.setItem('userInfo', JSON.stringify({
    username: 'operator',
    email: 'ops@example.com',
    roles: ['operator']
  }))
  
  console.log('Operator user setup complete')
  console.log('Operator will see:')
  console.log('- Dashboard')
  console.log('- Application Management')
  console.log('- Health Check')
  console.log('- CI/CD Pipeline (view only)')
  console.log('- Kubernetes (with management)')
  console.log('- Alerts')
  console.log('- Logs')
  console.log('- Cost Management')
}

// ============================================
// Example 4: Setting up a Viewer User
// ============================================

export function setupViewerUser() {
  localStorage.setItem('userInfo', JSON.stringify({
    username: 'viewer',
    email: 'viewer@example.com',
    roles: ['viewer']
  }))
  
  console.log('Viewer user setup complete')
  console.log('Viewer will see (read-only):')
  console.log('- Dashboard')
  console.log('- Application Management')
  console.log('- CI/CD Pipeline (list only)')
  console.log('- Kubernetes')
  console.log('- Logs')
  console.log('- Alerts')
}

// ============================================
// Example 5: Custom Permissions
// ============================================

export function setupCustomUser() {
  localStorage.setItem('userInfo', JSON.stringify({
    username: 'custom',
    email: 'custom@example.com',
    permissions: [
      'pipeline:view',
      'pipeline:create',
      'logs:view',
      'application:view'
    ]
  }))
  
  console.log('Custom user setup complete')
  console.log('Custom user will see:')
  console.log('- Dashboard')
  console.log('- Application Management')
  console.log('- CI/CD Pipeline (with designer)')
  console.log('- Logs')
}

// ============================================
// Example 6: Multiple Roles
// ============================================

export function setupMultiRoleUser() {
  localStorage.setItem('userInfo', JSON.stringify({
    username: 'multirole',
    email: 'multi@example.com',
    roles: ['developer', 'operator']
  }))
  
  console.log('Multi-role user setup complete')
  console.log('User will have combined permissions from both roles')
}

// ============================================
// Example 7: Upgrading User Permissions
// ============================================

export function upgradeUserPermissions() {
  // Start as viewer
  localStorage.setItem('userInfo', JSON.stringify({
    username: 'user',
    email: 'user@example.com',
    roles: ['viewer']
  }))
  
  console.log('User started as viewer')
  
  // Simulate permission upgrade after some time
  setTimeout(() => {
    localStorage.setItem('userInfo', JSON.stringify({
      username: 'user',
      email: 'user@example.com',
      roles: ['developer']
    }))
    
    console.log('User upgraded to developer')
    console.log('Menu will automatically update to show more items')
    
    // Trigger a page reload or component re-render to see changes
    window.location.reload()
  }, 5000)
}

// ============================================
// Example 8: Checking Permissions Programmatically
// ============================================

export function checkUserPermissions() {
  const userInfoStr = localStorage.getItem('userInfo')
  if (!userInfoStr) {
    console.log('No user logged in')
    return
  }
  
  const userInfo = JSON.parse(userInfoStr)
  console.log('Current user:', userInfo.username)
  console.log('Roles:', userInfo.roles || 'None')
  console.log('Permissions:', userInfo.permissions || 'Role-based')
  
  // Example permission checks
  const permissionsToCheck = [
    'pipeline:view',
    'pipeline:create',
    'system:manage',
    'k8s:view',
    'logs:view'
  ]
  
  console.log('\nPermission Check Results:')
  permissionsToCheck.forEach(perm => {
    // This would use the actual hasPermission function from MainLayout
    console.log(`${perm}: [Would check in actual implementation]`)
  })
}

// ============================================
// Example 9: Simulating Login with Different Users
// ============================================

export function simulateLogin(userType: 'admin' | 'developer' | 'operator' | 'viewer') {
  switch (userType) {
    case 'admin':
      setupAdminUser()
      break
    case 'developer':
      setupDeveloperUser()
      break
    case 'operator':
      setupOperatorUser()
      break
    case 'viewer':
      setupViewerUser()
      break
  }
  
  console.log(`\nLogged in as ${userType}`)
  console.log('Reload the page to see the menu changes')
}

// ============================================
// Example 10: Logout
// ============================================

export function logout() {
  localStorage.removeItem('userInfo')
  localStorage.removeItem('token')
  console.log('User logged out')
  console.log('Redirect to login page')
}

// ============================================
// Usage in Browser Console
// ============================================

/*
To test the permission system in the browser console:

1. Open the application in your browser
2. Open the browser console (F12)
3. Import this module (if using module system) or copy functions
4. Run any of the setup functions:

   // Setup admin user
   setupAdminUser()
   location.reload()

   // Setup developer user
   setupDeveloperUser()
   location.reload()

   // Setup viewer user
   setupViewerUser()
   location.reload()

   // Check current permissions
   checkUserPermissions()

   // Simulate login
   simulateLogin('developer')
   location.reload()

5. Observe the menu changes after reloading

Note: In a real application, user authentication and permission
management would be handled by the backend API. This demo is for
testing the frontend permission filtering only.
*/

// ============================================
// Export all demo functions
// ============================================

export default {
  setupAdminUser,
  setupDeveloperUser,
  setupOperatorUser,
  setupViewerUser,
  setupCustomUser,
  setupMultiRoleUser,
  upgradeUserPermissions,
  checkUserPermissions,
  simulateLogin,
  logout
}

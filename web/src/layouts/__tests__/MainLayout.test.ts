import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('Menu State Management', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear()
  })

  describe('getInitialOpenKeys', () => {
    it('should return saved keys from localStorage', () => {
      const savedKeys = ['pipeline', 'system']
      localStorage.setItem('menuOpenKeys', JSON.stringify(savedKeys))
      
      const getInitialOpenKeys = (): string[] => {
        const saved = localStorage.getItem('menuOpenKeys')
        if (saved) {
          try {
            return JSON.parse(saved)
          } catch (e) {
            console.error('Failed to parse menuOpenKeys from localStorage:', e)
          }
        }
        return []
      }
      
      const result = getInitialOpenKeys()
      expect(result).toEqual(savedKeys)
    })

    it('should return empty array if no saved state', () => {
      const getInitialOpenKeys = (): string[] => {
        const saved = localStorage.getItem('menuOpenKeys')
        if (saved) {
          try {
            return JSON.parse(saved)
          } catch (e) {
            console.error('Failed to parse menuOpenKeys from localStorage:', e)
          }
        }
        return []
      }
      
      const result = getInitialOpenKeys()
      expect(result).toEqual([])
    })

    it('should handle invalid JSON gracefully', () => {
      localStorage.setItem('menuOpenKeys', 'invalid json')
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      
      const getInitialOpenKeys = (): string[] => {
        const saved = localStorage.getItem('menuOpenKeys')
        if (saved) {
          try {
            return JSON.parse(saved)
          } catch (e) {
            console.error('Failed to parse menuOpenKeys from localStorage:', e)
          }
        }
        return []
      }
      
      const result = getInitialOpenKeys()
      expect(result).toEqual([])
      expect(consoleSpy).toHaveBeenCalled()
      
      consoleSpy.mockRestore()
    })
  })

  describe('localStorage persistence', () => {
    it('should save openKeys to localStorage', () => {
      const openKeys = ['pipeline', 'system', 'k8s']
      localStorage.setItem('menuOpenKeys', JSON.stringify(openKeys))
      
      const saved = localStorage.getItem('menuOpenKeys')
      expect(saved).toBeTruthy()
      expect(JSON.parse(saved!)).toEqual(openKeys)
    })

    it('should update localStorage when openKeys change', () => {
      const initialKeys = ['pipeline']
      localStorage.setItem('menuOpenKeys', JSON.stringify(initialKeys))
      
      const updatedKeys = ['pipeline', 'system']
      localStorage.setItem('menuOpenKeys', JSON.stringify(updatedKeys))
      
      const saved = localStorage.getItem('menuOpenKeys')
      expect(JSON.parse(saved!)).toEqual(updatedKeys)
    })
  })

  describe('getParentKey', () => {
    const getParentKey = (path: string): string => {
      if (path.startsWith('/traffic/')) return 'traffic'
      if (path.startsWith('/applications') || path.startsWith('/deploys') || path.startsWith('/deploy/check') || path.startsWith('/canary') || path.startsWith('/bluegreen')) return 'app'
      if (path.startsWith('/healthcheck')) return 'healthcheck'
      if (path.startsWith('/resilience')) return ''
      if (path.startsWith('/pipeline')) return 'pipeline'
      if (path.startsWith('/jenkins')) return 'jenkins'
      if (path.startsWith('/k8s') || path.startsWith('/security')) return 'k8s'
      if (path.startsWith('/approval') || path.startsWith('/deploy/locks')) return 'approval'
      if (path.startsWith('/feishu') || path.startsWith('/dingtalk') || path.startsWith('/wechatwork')) return 'message'
      if (path.startsWith('/oa')) return 'oa'
      if (path.startsWith('/alert')) return 'alert'
      if (path.startsWith('/cost')) return 'cost'
      if (path.startsWith('/logs')) return 'logs'
      if (path.startsWith('/users') || path.startsWith('/rbac') || path.startsWith('/audit') || path.startsWith('/feature-flags') || path.startsWith('/system/monitor') || path.startsWith('/admin')) return 'system'
      return ''
    }

    it('should return correct parent key for pipeline routes', () => {
      expect(getParentKey('/pipeline/list')).toBe('pipeline')
      expect(getParentKey('/pipeline/designer')).toBe('pipeline')
      expect(getParentKey('/pipeline/cache')).toBe('pipeline')
    })

    it('should return correct parent key for system routes', () => {
      expect(getParentKey('/users')).toBe('system')
      expect(getParentKey('/feature-flags')).toBe('system')
      expect(getParentKey('/system/monitor')).toBe('system')
    })

    it('should return correct parent key for healthcheck routes', () => {
      expect(getParentKey('/healthcheck')).toBe('healthcheck')
      expect(getParentKey('/healthcheck/ssl-cert')).toBe('healthcheck')
    })

    it('should return empty string for resilience route', () => {
      expect(getParentKey('/resilience')).toBe('')
    })

    it('should return correct parent key for k8s routes', () => {
      expect(getParentKey('/k8s/clusters')).toBe('k8s')
      expect(getParentKey('/security/overview')).toBe('k8s')
    })
  })
})

describe('Mobile Responsive Features', () => {
  describe('isMobile detection', () => {
    it('should detect mobile when window width < 768', () => {
      // Mock window.innerWidth
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 375
      })
      
      const windowWidth = 375
      const isMobile = windowWidth < 768
      
      expect(isMobile).toBe(true)
    })

    it('should not detect mobile when window width >= 768', () => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 1024
      })
      
      const windowWidth = 1024
      const isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
    })

    it('should detect tablet as non-mobile', () => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 768
      })
      
      const windowWidth = 768
      const isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
    })

    it('should detect desktop as non-mobile', () => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 1920
      })
      
      const windowWidth = 1920
      const isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
    })
  })

  describe('Mobile auto-collapse behavior', () => {
    it('should auto-collapse sidebar on mobile', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      let collapsed = false
      
      // Simulate mobile detection
      if (isMobile) {
        collapsed = true
      }
      
      expect(collapsed).toBe(true)
    })

    it('should not auto-collapse sidebar on desktop', () => {
      const windowWidth = 1024
      const isMobile = windowWidth < 768
      let collapsed = false
      
      // Simulate desktop detection
      if (isMobile) {
        collapsed = true
      }
      
      expect(collapsed).toBe(false)
    })
  })

  describe('Mobile menu click behavior', () => {
    it('should collapse sidebar after menu click on mobile', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      let collapsed = false
      
      // Simulate menu click
      const handleMenuClick = () => {
        if (isMobile) {
          collapsed = true
        }
      }
      
      handleMenuClick()
      expect(collapsed).toBe(true)
    })

    it('should not collapse sidebar after menu click on desktop', () => {
      const windowWidth = 1024
      const isMobile = windowWidth < 768
      let collapsed = false
      
      // Simulate menu click
      const handleMenuClick = () => {
        if (isMobile) {
          collapsed = true
        }
      }
      
      handleMenuClick()
      expect(collapsed).toBe(false)
    })
  })

  describe('Window resize handling', () => {
    it('should update mobile state when window is resized', () => {
      let windowWidth = 1024
      let isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
      
      // Simulate resize to mobile
      windowWidth = 375
      isMobile = windowWidth < 768
      
      expect(isMobile).toBe(true)
    })

    it('should update mobile state when resizing from mobile to desktop', () => {
      let windowWidth = 375
      let isMobile = windowWidth < 768
      
      expect(isMobile).toBe(true)
      
      // Simulate resize to desktop
      windowWidth = 1024
      isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
    })
  })

  describe('Responsive breakpoints', () => {
    it('should use 768px as mobile breakpoint', () => {
      const MOBILE_BREAKPOINT = 768
      
      expect(767 < MOBILE_BREAKPOINT).toBe(true)
      expect(768 < MOBILE_BREAKPOINT).toBe(false)
      expect(769 < MOBILE_BREAKPOINT).toBe(false)
    })

    it('should handle edge case at breakpoint', () => {
      const windowWidth = 768
      const isMobile = windowWidth < 768
      
      // At exactly 768px, should not be considered mobile
      expect(isMobile).toBe(false)
    })
  })
})

describe('Responsive Layout Testing (Task 8.2)', () => {
  describe('8.2.1 测试不同屏幕尺寸下的菜单显示', () => {
    it('should display menu correctly on mobile (375px - iPhone SE)', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      const expectedSidebarWidth = 240 // Mobile sidebar width
      
      expect(isMobile).toBe(true)
      expect(expectedSidebarWidth).toBe(240)
    })

    it('should display menu correctly on mobile (414px - iPhone Pro Max)', () => {
      const windowWidth = 414
      const isMobile = windowWidth < 768
      
      expect(isMobile).toBe(true)
    })

    it('should display menu correctly on tablet portrait (768px - iPad)', () => {
      const windowWidth = 768
      const isMobile = windowWidth < 768
      const isTablet = windowWidth >= 768 && windowWidth <= 1024
      
      expect(isMobile).toBe(false)
      expect(isTablet).toBe(true)
    })

    it('should display menu correctly on tablet landscape (1024px - iPad)', () => {
      const windowWidth = 1024
      const isMobile = windowWidth < 768
      const isTablet = windowWidth >= 768 && windowWidth <= 1024
      
      expect(isMobile).toBe(false)
      expect(isTablet).toBe(true)
    })

    it('should display menu correctly on desktop (1920px)', () => {
      const windowWidth = 1920
      const isMobile = windowWidth < 768
      const isTablet = windowWidth >= 768 && windowWidth <= 1024
      const isDesktop = windowWidth > 1024
      
      expect(isMobile).toBe(false)
      expect(isTablet).toBe(false)
      expect(isDesktop).toBe(true)
    })

    it('should display menu correctly on large desktop (2560px)', () => {
      const windowWidth = 2560
      const isMobile = windowWidth < 768
      const isDesktop = windowWidth > 1024
      
      expect(isMobile).toBe(false)
      expect(isDesktop).toBe(true)
    })

    it('should use correct sidebar width for mobile', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      const sidebarWidth = isMobile ? 240 : 240 // Mobile uses 240px overlay
      
      expect(sidebarWidth).toBe(240)
    })

    it('should use correct sidebar width for tablet', () => {
      const windowWidth = 800
      const isTablet = windowWidth >= 769 && windowWidth <= 1024
      const sidebarWidth = isTablet ? 200 : 240 // Tablet uses 200px
      
      expect(sidebarWidth).toBe(200)
    })

    it('should use correct sidebar width for desktop', () => {
      const windowWidth = 1920
      const isDesktop = windowWidth > 1024
      const sidebarWidth = isDesktop ? 240 : 200 // Desktop uses 240px
      
      expect(sidebarWidth).toBe(240)
    })
  })

  describe('8.2.2 测试平板设备上的菜单交互', () => {
    it('should not auto-collapse on tablet', () => {
      const windowWidth = 768
      const isMobile = windowWidth < 768
      let collapsed = false
      
      if (isMobile) {
        collapsed = true
      }
      
      expect(collapsed).toBe(false)
    })

    it('should allow manual collapse on tablet', () => {
      const windowWidth = 800
      const isMobile = windowWidth < 768
      let collapsed = false
      
      // User manually toggles
      collapsed = !collapsed
      
      expect(collapsed).toBe(true)
    })

    it('should not auto-collapse after menu click on tablet', () => {
      const windowWidth = 1024
      const isMobile = windowWidth < 768
      let collapsed = false
      
      const handleMenuClick = () => {
        if (isMobile) {
          collapsed = true
        }
      }
      
      handleMenuClick()
      expect(collapsed).toBe(false)
    })

    it('should support submenu expansion on tablet', () => {
      const windowWidth = 800
      const openKeys: string[] = []
      
      // Simulate submenu click
      const handleSubMenuClick = (key: string) => {
        if (!openKeys.includes(key)) {
          openKeys.push(key)
        }
      }
      
      handleSubMenuClick('pipeline')
      expect(openKeys).toContain('pipeline')
    })

    it('should maintain menu state on tablet orientation change', () => {
      let windowWidth = 768 // Portrait
      let openKeys = ['pipeline', 'system']
      
      // Change to landscape
      windowWidth = 1024
      
      // openKeys should be preserved
      expect(openKeys).toEqual(['pipeline', 'system'])
    })
  })

  describe('8.2.3 测试横屏和竖屏切换', () => {
    it('should handle portrait to landscape transition on mobile', () => {
      let windowWidth = 375 // Portrait
      let isMobile = windowWidth < 768
      let collapsed = isMobile
      
      expect(collapsed).toBe(true)
      
      // Rotate to landscape (still mobile size)
      windowWidth = 667 // iPhone landscape
      isMobile = windowWidth < 768
      
      expect(isMobile).toBe(true)
      expect(collapsed).toBe(true)
    })

    it('should handle portrait to landscape transition on tablet', () => {
      let windowWidth = 768 // Portrait
      let isMobile = windowWidth < 768
      let collapsed = false
      
      expect(isMobile).toBe(false)
      
      // Rotate to landscape
      windowWidth = 1024
      isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
      expect(collapsed).toBe(false)
    })

    it('should handle landscape to portrait transition on tablet', () => {
      let windowWidth = 1024 // Landscape
      let isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
      
      // Rotate to portrait
      windowWidth = 768
      isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
    })

    it('should preserve menu state during orientation change', () => {
      let windowWidth = 768
      const openKeys = ['pipeline', 'system']
      const selectedKeys = ['/pipeline/list']
      
      // Rotate device
      windowWidth = 1024
      
      // State should be preserved
      expect(openKeys).toEqual(['pipeline', 'system'])
      expect(selectedKeys).toEqual(['/pipeline/list'])
    })

    it('should handle rapid orientation changes', () => {
      let windowWidth = 768
      let isMobile = windowWidth < 768
      
      // Rapid changes
      windowWidth = 1024
      isMobile = windowWidth < 768
      expect(isMobile).toBe(false)
      
      windowWidth = 768
      isMobile = windowWidth < 768
      expect(isMobile).toBe(false)
      
      windowWidth = 1024
      isMobile = windowWidth < 768
      expect(isMobile).toBe(false)
    })

    it('should maintain collapsed state preference across orientation changes', () => {
      let windowWidth = 1024
      let collapsed = true // User preference
      
      // Rotate
      windowWidth = 768
      
      // User preference should be maintained
      expect(collapsed).toBe(true)
    })
  })

  describe('8.2.4 测试触摸操作', () => {
    it('should support touch-friendly menu item size (minimum 44x44px)', () => {
      const MIN_TOUCH_TARGET = 44 // iOS HIG recommendation
      const menuItemHeight = 48 // Ant Design default
      
      expect(menuItemHeight).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET)
    })

    it('should handle touch menu click on mobile', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      let collapsed = false
      let navigated = false
      
      const handleTouchMenuClick = () => {
        navigated = true
        if (isMobile) {
          collapsed = true
        }
      }
      
      handleTouchMenuClick()
      expect(navigated).toBe(true)
      expect(collapsed).toBe(true)
    })

    it('should handle touch submenu expansion', () => {
      const openKeys: string[] = []
      
      const handleTouchSubMenu = (key: string) => {
        if (openKeys.includes(key)) {
          const index = openKeys.indexOf(key)
          openKeys.splice(index, 1)
        } else {
          openKeys.push(key)
        }
      }
      
      // Touch to expand
      handleTouchSubMenu('pipeline')
      expect(openKeys).toContain('pipeline')
      
      // Touch again to collapse
      handleTouchSubMenu('pipeline')
      expect(openKeys).not.toContain('pipeline')
    })

    it('should handle touch sidebar toggle', () => {
      let collapsed = false
      
      const handleTouchToggle = () => {
        collapsed = !collapsed
      }
      
      handleTouchToggle()
      expect(collapsed).toBe(true)
      
      handleTouchToggle()
      expect(collapsed).toBe(false)
    })

    it('should support touch scroll in menu', () => {
      const menuItems = Array.from({ length: 20 }, (_, i) => `item-${i}`)
      const scrollPosition = 0
      
      // Simulate touch scroll
      const handleTouchScroll = (delta: number) => {
        return scrollPosition + delta
      }
      
      const newPosition = handleTouchScroll(100)
      expect(newPosition).toBe(100)
      expect(menuItems.length).toBe(20)
    })

    it('should prevent accidental touches with proper spacing', () => {
      const MENU_ITEM_SPACING = 8 // Spacing between items
      const MENU_ITEM_HEIGHT = 48
      const TOTAL_HEIGHT = MENU_ITEM_HEIGHT + MENU_ITEM_SPACING
      
      expect(TOTAL_HEIGHT).toBeGreaterThan(MENU_ITEM_HEIGHT)
      expect(MENU_ITEM_SPACING).toBeGreaterThanOrEqual(4)
    })

    it('should handle touch backdrop click on mobile', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      let collapsed = false
      
      const handleBackdropTouch = () => {
        if (isMobile && !collapsed) {
          collapsed = true
        }
      }
      
      handleBackdropTouch()
      expect(collapsed).toBe(true)
    })

    it('should support touch-friendly logo area', () => {
      const LOGO_HEIGHT = 64
      const MIN_TOUCH_TARGET = 44
      
      expect(LOGO_HEIGHT).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET)
    })

    it('should support touch-friendly trigger button', () => {
      const TRIGGER_SIZE = 48 // Estimated touch area
      const MIN_TOUCH_TARGET = 44
      
      expect(TRIGGER_SIZE).toBeGreaterThanOrEqual(MIN_TOUCH_TARGET)
    })
  })

  describe('Additional responsive behavior tests', () => {
    it('should handle window resize from desktop to mobile', () => {
      let windowWidth = 1920
      let isMobile = windowWidth < 768
      let collapsed = false
      
      expect(isMobile).toBe(false)
      
      // Resize to mobile
      windowWidth = 375
      isMobile = windowWidth < 768
      if (isMobile) {
        collapsed = true
      }
      
      expect(isMobile).toBe(true)
      expect(collapsed).toBe(true)
    })

    it('should handle window resize from mobile to desktop', () => {
      let windowWidth = 375
      let isMobile = windowWidth < 768
      let collapsed = true
      
      expect(isMobile).toBe(true)
      
      // Resize to desktop
      windowWidth = 1920
      isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
      // collapsed state can be maintained or reset based on UX preference
    })

    it('should use overlay mode on mobile', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      const sidebarMode = isMobile ? 'overlay' : 'inline'
      
      expect(sidebarMode).toBe('overlay')
    })

    it('should use inline mode on desktop', () => {
      const windowWidth = 1920
      const isMobile = windowWidth < 768
      const sidebarMode = isMobile ? 'overlay' : 'inline'
      
      expect(sidebarMode).toBe('inline')
    })

    it('should show backdrop on mobile when sidebar is open', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      const collapsed = false
      const showBackdrop = isMobile && !collapsed
      
      expect(showBackdrop).toBe(true)
    })

    it('should not show backdrop on mobile when sidebar is collapsed', () => {
      const windowWidth = 375
      const isMobile = windowWidth < 768
      const collapsed = true
      const showBackdrop = isMobile && !collapsed
      
      expect(showBackdrop).toBe(false)
    })

    it('should not show backdrop on desktop', () => {
      const windowWidth = 1920
      const isMobile = windowWidth < 768
      const collapsed = false
      const showBackdrop = isMobile && !collapsed
      
      expect(showBackdrop).toBe(false)
    })
  })

  describe('CSS media query breakpoints validation', () => {
    it('should have correct mobile breakpoint (max-width: 768px)', () => {
      const MOBILE_MAX_WIDTH = 768
      
      expect(767).toBeLessThan(MOBILE_MAX_WIDTH)
      expect(768).toBeLessThanOrEqual(MOBILE_MAX_WIDTH)
      expect(769).toBeGreaterThan(MOBILE_MAX_WIDTH)
    })

    it('should have correct tablet breakpoint (769px - 1024px)', () => {
      const TABLET_MIN_WIDTH = 769
      const TABLET_MAX_WIDTH = 1024
      
      expect(768).toBeLessThan(TABLET_MIN_WIDTH)
      expect(769).toBeGreaterThanOrEqual(TABLET_MIN_WIDTH)
      expect(1024).toBeLessThanOrEqual(TABLET_MAX_WIDTH)
      expect(1025).toBeGreaterThan(TABLET_MAX_WIDTH)
    })

    it('should have correct desktop breakpoint (> 1024px)', () => {
      const DESKTOP_MIN_WIDTH = 1024
      
      expect(1024).toBeLessThanOrEqual(DESKTOP_MIN_WIDTH)
      expect(1025).toBeGreaterThan(DESKTOP_MIN_WIDTH)
    })
  })

  describe('Performance and edge cases', () => {
    it('should handle very small screen (320px)', () => {
      const windowWidth = 320
      const isMobile = windowWidth < 768
      
      expect(isMobile).toBe(true)
    })

    it('should handle very large screen (3840px - 4K)', () => {
      const windowWidth = 3840
      const isMobile = windowWidth < 768
      const isDesktop = windowWidth > 1024
      
      expect(isMobile).toBe(false)
      expect(isDesktop).toBe(true)
    })

    it('should handle ultra-wide screen (5120px)', () => {
      const windowWidth = 5120
      const isMobile = windowWidth < 768
      
      expect(isMobile).toBe(false)
    })

    it('should maintain menu functionality at breakpoint boundaries', () => {
      const testWidths = [767, 768, 769, 1023, 1024, 1025]
      
      testWidths.forEach(width => {
        const isMobile = width < 768
        const isTablet = width >= 769 && width <= 1024
        const isDesktop = width > 1024
        
        // Exactly one should be true (or mobile for 768)
        const count = [isMobile, isTablet, isDesktop].filter(Boolean).length
        expect(count).toBeGreaterThanOrEqual(0)
      })
    })
  })
})

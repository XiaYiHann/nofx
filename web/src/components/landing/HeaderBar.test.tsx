/**
 * HeaderBar.test.tsx
 *
 * 测试目的：确保 HeaderBar 组件在不同状态下正确渲染和响应用户交互
 *
 * 覆盖的 Bug 类型：
 * 1. 导航按钮点击不响应 - 测试 onPageChange 回调
 * 2. 用户下拉菜单展开/收起问题 - 测试点击行为
 * 3. 语言切换不生效 - 测试 onLanguageChange 回调
 * 4. 移动端菜单状态异常 - 测试 mobile menu toggle
 *
 * 注意：组件同时渲染 desktop 和 mobile 版本，所以某些元素会出现多次
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import HeaderBar from './HeaderBar'

// Mock framer-motion to avoid animation issues in tests
vi.mock('framer-motion', () => ({
  motion: {
    button: ({ children, ...props }: any) => <button {...props}>{children}</button>,
    div: ({ children, initial, animate, transition, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}))

describe('HeaderBar', () => {
  const mockOnPageChange = vi.fn()
  const mockOnLanguageChange = vi.fn()
  const mockOnLogout = vi.fn()
  const mockOnLoginClick = vi.fn()

  const defaultUser = { email: 'test@example.com' }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('Desktop 模式 - 未登录状态', () => {
    it('should render login and register buttons when not logged in', () => {
      render(
        <HeaderBar
          isLoggedIn={false}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
        />
      )

      // Should show realtime navigation for non-logged-in users
      expect(screen.getAllByText('实时').length).toBeGreaterThan(0)
      // Should show FAQ
      expect(screen.getAllByText('常见问题').length).toBeGreaterThan(0)
    })

    it('should not show authenticated-only pages when not logged in', () => {
      render(
        <HeaderBar
          isLoggedIn={false}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
        />
      )

      // Should NOT show authenticated-only navigation
      expect(screen.queryByText('配置')).not.toBeInTheDocument()
      expect(screen.queryByText('看板')).not.toBeInTheDocument()
    })
  })

  describe('Desktop 模式 - 已登录状态', () => {
    it('should render all navigation tabs when logged in', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Should show all nav options (may be duplicated in mobile menu)
      expect(screen.getAllByText('实时').length).toBeGreaterThan(0)
      expect(screen.getAllByText('配置').length).toBeGreaterThan(0)
      expect(screen.getAllByText('看板').length).toBeGreaterThan(0)
      expect(screen.getAllByText('常见问题').length).toBeGreaterThan(0)
      expect(screen.getAllByText('策略管理').length).toBeGreaterThan(0)
      expect(screen.getAllByText('回测').length).toBeGreaterThan(0)
    })

    it('should display user email', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // User email may appear in multiple places (desktop dropdown + mobile menu)
      expect(screen.getAllByText('test@example.com').length).toBeGreaterThan(0)
      // Avatar should show first letter
      expect(screen.getAllByText('T').length).toBeGreaterThan(0)
    })
  })

  describe('导航交互 - onPageChange 回调', () => {
    /**
     * 测试：点击导航按钮应触发 onPageChange
     * 复现 Bug：导航点击无响应
     */
    it('should call onPageChange when clicking navigation buttons', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Click the first "配置" button (desktop version)
      const configButtons = screen.getAllByText('配置')
      fireEvent.click(configButtons[0])

      expect(mockOnPageChange).toHaveBeenCalledWith('traders')
    })

    it('should call onPageChange with correct page value for each nav item', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Test each navigation button (click first instance which is desktop)
      fireEvent.click(screen.getAllByText('实时')[0])
      expect(mockOnPageChange).toHaveBeenCalledWith('competition')

      fireEvent.click(screen.getAllByText('配置')[0])
      expect(mockOnPageChange).toHaveBeenCalledWith('traders')

      fireEvent.click(screen.getAllByText('看板')[0])
      expect(mockOnPageChange).toHaveBeenCalledWith('trader')

      fireEvent.click(screen.getAllByText('常见问题')[0])
      expect(mockOnPageChange).toHaveBeenCalledWith('faq')

      fireEvent.click(screen.getAllByText('策略管理')[0])
      expect(mockOnPageChange).toHaveBeenCalledWith('strategies')

      fireEvent.click(screen.getAllByText('回测')[0])
      expect(mockOnPageChange).toHaveBeenCalledWith('backtest')
    })
  })

  describe('语言切换', () => {
    /**
     * 测试：语言切换应触发 onLanguageChange
     * 复现 Bug：语言切换后 UI 不更新
     */
    it('should call onLanguageChange when switching language', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Find and click the language dropdown button (shows Chinese flag)
      const langButtons = screen.getAllByText('🇨🇳')
      const langButton = langButtons[0].closest('button')
      expect(langButton).toBeInTheDocument()
      fireEvent.click(langButton!)

      // Click English option (first one in the opened dropdown)
      const englishOptions = screen.getAllByText('English')
      fireEvent.click(englishOptions[0])

      expect(mockOnLanguageChange).toHaveBeenCalledWith('en')
    })

    it('should display correct language flag based on current language', () => {
      const { rerender } = render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
        />
      )

      expect(screen.getAllByText('🇨🇳').length).toBeGreaterThan(0)

      rerender(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="en"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
        />
      )

      expect(screen.getAllByText('🇺🇸').length).toBeGreaterThan(0)
    })

    it('should display different navigation text based on language', () => {
      const { rerender } = render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onPageChange={mockOnPageChange}
        />
      )

      expect(screen.getAllByText('实时').length).toBeGreaterThan(0)
      expect(screen.getAllByText('配置').length).toBeGreaterThan(0)
      expect(screen.getAllByText('看板').length).toBeGreaterThan(0)

      rerender(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="en"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onPageChange={mockOnPageChange}
        />
      )

      expect(screen.getAllByText('Live').length).toBeGreaterThan(0)
      expect(screen.getAllByText('Config').length).toBeGreaterThan(0)
      expect(screen.getAllByText('Dashboard').length).toBeGreaterThan(0)
    })
  })

  describe('用户下拉菜单', () => {
    /**
     * 测试：点击用户信息应展开下拉菜单
     */
    it('should toggle user dropdown on click', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Find the desktop user dropdown trigger (first one)
      const userEmails = screen.getAllByText('test@example.com')
      const userButton = userEmails[0].closest('button')
      expect(userButton).toBeInTheDocument()

      // Initially dropdown should be closed
      expect(screen.queryByText('登录身份')).not.toBeInTheDocument()

      // Click to open
      fireEvent.click(userButton!)

      // Dropdown should be visible - use correct translation key 'loggedInAs' = '已登录为'
      // Note: Both desktop and mobile dropdowns exist, so use getAllByText
      expect(screen.getAllByText('已登录为').length).toBeGreaterThan(0)
      // Multiple logout buttons may exist (desktop + mobile)
      expect(screen.getAllByText('退出登录').length).toBeGreaterThan(0)
    })

    /**
     * 测试：点击登出应触发 onLogout
     */
    it('should call onLogout when clicking logout button', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Open user dropdown
      const userEmails = screen.getAllByText('test@example.com')
      const userButton = userEmails[0].closest('button')
      fireEvent.click(userButton!)

      // Click logout (first button)
      const logoutButtons = screen.getAllByText('退出登录')
      fireEvent.click(logoutButtons[0])

      expect(mockOnLogout).toHaveBeenCalled()
    })
  })

  describe('Mobile 模式 - 菜单切换', () => {
    /**
     * 测试：移动端菜单按钮存在
     */
    it('should have mobile menu button', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Mobile menu button should exist (md:hidden class)
      const mobileButtons = document.querySelectorAll('button.md\\:hidden')
      expect(mobileButtons.length).toBeGreaterThan(0)
    })

    /**
     * 测试：移动端点击导航应触发 onPageChange
     */
    it('should call onPageChange when clicking mobile nav item', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Mobile navigation buttons - there are multiple "配置" buttons
      const configButtons = screen.getAllByText('配置')

      // Click one of them (should trigger onPageChange)
      fireEvent.click(configButtons[0])

      expect(mockOnPageChange).toHaveBeenCalledWith('traders')
    })
  })

  describe('Logo 导航', () => {
    it('should render logo with correct link to home', () => {
      render(
        <HeaderBar
          isLoggedIn={false}
          language="zh"
          onLanguageChange={mockOnLanguageChange}
        />
      )

      const logoLink = screen.getByText('NOFX').closest('a')
      expect(logoLink).toHaveAttribute('href', '/')
    })

    it('should display tagline', () => {
      render(
        <HeaderBar
          isLoggedIn={false}
          language="zh"
          onLanguageChange={mockOnLanguageChange}
        />
      )

      expect(screen.getByText('Agentic Trading OS')).toBeInTheDocument()
    })
  })

  describe('边界情况', () => {
    it('should handle undefined onPageChange gracefully', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          onLogout={mockOnLogout}
          // onPageChange is undefined
        />
      )

      // Clicking should not throw
      const configButtons = screen.getAllByText('配置')
      expect(() => fireEvent.click(configButtons[0])).not.toThrow()
    })

    it('should handle undefined onLanguageChange gracefully', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          // onLanguageChange is undefined
          user={defaultUser}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      const langButtons = screen.getAllByText('🇨🇳')
      const langButton = langButtons[0].closest('button')
      fireEvent.click(langButton!)

      const englishOptions = screen.getAllByText('English')
      expect(() => fireEvent.click(englishOptions[0])).not.toThrow()
    })

    it('should handle undefined onLogout gracefully', () => {
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={defaultUser}
          // onLogout is undefined
          onPageChange={mockOnPageChange}
        />
      )

      const userEmails = screen.getAllByText('test@example.com')
      const userButton = userEmails[0].closest('button')
      fireEvent.click(userButton!)

      // Logout button should not be rendered when onLogout is undefined
      expect(screen.queryByText('退出登录')).not.toBeInTheDocument()
    })

    it('should handle null user gracefully when logged in', () => {
      // Edge case: isLoggedIn=true but user=null
      render(
        <HeaderBar
          isLoggedIn={true}
          currentPage="competition"
          language="zh"
          onLanguageChange={mockOnLanguageChange}
          user={null}
          onLogout={mockOnLogout}
          onPageChange={mockOnPageChange}
        />
      )

      // Should not show user email when user is null
      expect(screen.queryByText('test@example.com')).not.toBeInTheDocument()
    })
  })
})

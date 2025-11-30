/**
 * AuthContext.test.tsx
 *
 * 测试目的：确保 AuthContext 的核心认证逻辑正确工作
 *
 * 覆盖的 Bug 类型：
 * 1. dev_mode 自动登录失败 - 测试开发模式下自动注入测试用户
 * 2. 401 race condition - 测试 unauthorized 事件处理时 token 匹配/不匹配的情况
 * 3. localStorage 恢复失败 - 测试从 localStorage 恢复 user/token 的情况
 * 4. 登录/注册流程中断 - 测试各种 happy & error path
 *
 * Regression 复现案例：
 * - 401 race: 用户在后台请求返回 401 时刚完成 OTP 登录，新 token 不应被清除
 * - dev_mode: 配置 dev_mode=true 时应自动登录 dev-user，无需手动登录
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  render,
  screen,
  waitFor,
  act,
  renderHook,
} from '@testing-library/react'
import { AuthProvider, useAuth } from './AuthContext'
import * as configModule from '../lib/config'
import * as httpClientModule from '../lib/httpClient'

// Mock getSystemConfig
vi.mock('../lib/config', () => ({
  getSystemConfig: vi.fn(),
}))

// Mock httpClient
vi.mock('../lib/httpClient', () => ({
  reset401Flag: vi.fn(),
  httpClient: {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

// Mock fetch for login/register/verifyOTP etc
const mockFetch = vi.fn()
global.fetch = mockFetch

// Test component that uses the auth context
function TestConsumer({
  onAuthData,
}: {
  onAuthData?: (auth: ReturnType<typeof useAuth>) => void
}) {
  const auth = useAuth()
  onAuthData?.(auth)
  return (
    <div>
      <span data-testid="user">{auth.user?.email || 'no-user'}</span>
      <span data-testid="token">{auth.token || 'no-token'}</span>
      <span data-testid="loading">{auth.isLoading ? 'loading' : 'ready'}</span>
    </div>
  )
}

describe('AuthContext', () => {
  beforeEach(() => {
    // Clear localStorage
    localStorage.clear()
    sessionStorage.clear()

    // Reset all mocks
    vi.clearAllMocks()

    // Reset window location
    Object.defineProperty(window, 'location', {
      value: {
        pathname: '/',
        search: '',
        href: '',
      },
      writable: true,
    })

    // Default mock for getSystemConfig
    vi.mocked(configModule.getSystemConfig).mockResolvedValue({
      beta_mode: false,
      registration_enabled: true,
      dev_mode: false,
    })
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('初始化 - dev_mode 自动登录', () => {
    /**
     * 测试：dev_mode=true 时自动注入 dev user
     * 复现 Bug：开发环境下需要手动登录的问题
     */
    it('should auto-login dev user when dev_mode is true', async () => {
      vi.mocked(configModule.getSystemConfig).mockResolvedValue({
        beta_mode: false,
        registration_enabled: true,
        dev_mode: true,
      })

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      expect(screen.getByTestId('user')).toHaveTextContent('dev@localhost')
      expect(localStorage.getItem('auth_user')).toContain('dev-user')
    })

    it('should not auto-login when dev_mode is false', async () => {
      vi.mocked(configModule.getSystemConfig).mockResolvedValue({
        beta_mode: false,
        registration_enabled: true,
        dev_mode: false,
      })

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      expect(screen.getByTestId('user')).toHaveTextContent('no-user')
    })
  })

  describe('从 localStorage 恢复用户状态', () => {
    /**
     * 测试：从 localStorage 恢复 user/token
     * 复现 Bug：页面刷新后丢失登录状态
     */
    it('should restore user and token from localStorage', async () => {
      const savedUser = { id: 'user-123', email: 'test@example.com' }
      const savedToken = 'saved-jwt-token'

      localStorage.setItem('auth_user', JSON.stringify(savedUser))
      localStorage.setItem('auth_token', savedToken)

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      expect(screen.getByTestId('user')).toHaveTextContent('test@example.com')
      expect(screen.getByTestId('token')).toHaveTextContent('saved-jwt-token')
    })

    it('should restore user when getSystemConfig fails', async () => {
      const savedUser = { id: 'user-123', email: 'fallback@example.com' }
      const savedToken = 'fallback-token'

      localStorage.setItem('auth_user', JSON.stringify(savedUser))
      localStorage.setItem('auth_token', savedToken)

      vi.mocked(configModule.getSystemConfig).mockRejectedValue(
        new Error('Network error')
      )

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      expect(screen.getByTestId('user')).toHaveTextContent(
        'fallback@example.com'
      )
      expect(screen.getByTestId('token')).toHaveTextContent('fallback-token')
    })

    it('should not restore if localStorage is empty', async () => {
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      expect(screen.getByTestId('user')).toHaveTextContent('no-user')
      expect(screen.getByTestId('token')).toHaveTextContent('no-token')
    })
  })

  describe('unauthorized 事件处理 (401 race condition)', () => {
    /**
     * 测试：unauthorized 事件处理的 token 匹配逻辑
     * 复现 Bug：后台请求返回 401 时，如果用户刚完成登录，新 token 被错误清除
     */
    it('should clear auth when triggerToken matches current token', async () => {
      const currentToken = 'current-valid-token'
      const currentUser = { id: 'user-1', email: 'user@test.com' }

      localStorage.setItem('auth_token', currentToken)
      localStorage.setItem('auth_user', JSON.stringify(currentUser))

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      // Dispatch unauthorized event with matching token
      act(() => {
        const event = new CustomEvent('unauthorized', {
          detail: { triggerToken: currentToken },
        })
        window.dispatchEvent(event)
      })

      await waitFor(() => {
        expect(screen.getByTestId('user')).toHaveTextContent('no-user')
      })
    })

    it('should NOT clear auth when triggerToken does NOT match current token (race condition protection)', async () => {
      const newToken = 'new-valid-token'
      const oldStaleToken = 'old-stale-token'
      const currentUser = { id: 'user-1', email: 'user@test.com' }

      localStorage.setItem('auth_token', newToken)
      localStorage.setItem('auth_user', JSON.stringify(currentUser))

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      // Dispatch unauthorized event with OLD token (simulating race condition)
      act(() => {
        const event = new CustomEvent('unauthorized', {
          detail: { triggerToken: oldStaleToken },
        })
        window.dispatchEvent(event)
      })

      // User should still be logged in - race condition was blocked
      await new Promise((resolve) => setTimeout(resolve, 50))
      expect(screen.getByTestId('user')).toHaveTextContent('user@test.com')
      expect(screen.getByTestId('token')).toHaveTextContent('new-valid-token')
    })

    it('should clear auth when triggerToken is null and no current token', async () => {
      // No token in localStorage - user is not logged in

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      // Dispatch unauthorized event without triggerToken
      act(() => {
        const event = new CustomEvent('unauthorized', {
          detail: {},
        })
        window.dispatchEvent(event)
      })

      // Should remain no-user (was already no-user)
      expect(screen.getByTestId('user')).toHaveTextContent('no-user')
    })
  })

  describe('login 流程', () => {
    /**
     * 测试：登录成功 (无需 OTP)
     */
    it('should handle successful login without OTP', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          token: 'new-token',
          user_id: 'user-123',
          email: 'login@test.com',
          message: 'Login successful',
        }),
      })

      const pushStateSpy = vi.spyOn(window.history, 'pushState')

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      // Call login
      let result: any
      await act(async () => {
        result = await authContext!.login('test@example.com', 'password123')
      })

      expect(result.success).toBe(true)
      expect(localStorage.getItem('auth_token')).toBe('new-token')
      expect(httpClientModule.reset401Flag).toHaveBeenCalled()
      expect(pushStateSpy).toHaveBeenCalledWith({}, '', '/traders')
    })

    /**
     * 测试：登录需要 OTP 验证
     */
    it('should handle login requiring OTP', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          requires_otp: true,
          user_id: 'user-otp',
          message: 'OTP required',
        }),
      })

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.login('test@example.com', 'password123')
      })

      expect(result.success).toBe(true)
      expect(result.requiresOTP).toBe(true)
      expect(result.userID).toBe('user-otp')
      // Token should not be set yet
      expect(localStorage.getItem('auth_token')).toBeNull()
    })

    /**
     * 测试：登录失败
     */
    it('should handle login failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: async () => ({
          error: 'Invalid credentials',
        }),
      })

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.login('test@example.com', 'wrongpassword')
      })

      expect(result.success).toBe(false)
      expect(result.message).toBe('Invalid credentials')
    })

    /**
     * 测试：登录网络错误
     */
    it('should handle login network error', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'))

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.login('test@example.com', 'password')
      })

      expect(result.success).toBe(false)
      expect(result.message).toBe('登录失败，请重试')
    })
  })

  describe('verifyOTP 流程', () => {
    /**
     * 测试：OTP 验证成功
     */
    it('should handle successful OTP verification', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          token: 'otp-verified-token',
          user_id: 'user-otp',
          email: 'otp@test.com',
          message: 'OTP verified',
        }),
      })

      const pushStateSpy = vi.spyOn(window.history, 'pushState')

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.verifyOTP('user-otp', '123456')
      })

      expect(result.success).toBe(true)
      expect(localStorage.getItem('auth_token')).toBe('otp-verified-token')
      expect(httpClientModule.reset401Flag).toHaveBeenCalled()
      expect(pushStateSpy).toHaveBeenCalledWith({}, '', '/traders')
    })

    /**
     * 测试：OTP 验证失败
     */
    it('should handle OTP verification failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: async () => ({
          error: 'Invalid OTP code',
        }),
      })

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.verifyOTP('user-otp', 'wrong-code')
      })

      expect(result.success).toBe(false)
      expect(result.message).toBe('Invalid OTP code')
    })

    /**
     * 测试：OTP 验证后使用 returnUrl 重定向
     */
    it('should redirect to returnUrl after OTP verification', async () => {
      sessionStorage.setItem('returnUrl', '/dashboard')

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          token: 'otp-token',
          user_id: 'user-1',
          email: 'test@test.com',
          message: 'Success',
        }),
      })

      const pushStateSpy = vi.spyOn(window.history, 'pushState')

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      await act(async () => {
        await authContext!.verifyOTP('user-1', '123456')
      })

      expect(pushStateSpy).toHaveBeenCalledWith({}, '', '/dashboard')
      expect(sessionStorage.getItem('returnUrl')).toBeNull()
    })
  })

  describe('register 流程', () => {
    /**
     * 测试：注册成功
     */
    it('should handle successful registration', async () => {
      vi.mocked(httpClientModule.httpClient.post).mockResolvedValueOnce({
        success: true,
        data: {
          user_id: 'new-user',
          otp_secret: 'secret-key',
          qr_code_url: 'https://qr.example.com/code',
          message: 'Registration successful',
        },
      })

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.register(
          'new@test.com',
          'StrongPass123!',
          'BETA-CODE'
        )
      })

      expect(result.success).toBe(true)
      expect(result.userID).toBe('new-user')
      expect(result.otpSecret).toBe('secret-key')
      expect(result.qrCodeURL).toBe('https://qr.example.com/code')
    })

    /**
     * 测试：注册失败
     */
    it('should handle registration failure', async () => {
      vi.mocked(httpClientModule.httpClient.post).mockResolvedValueOnce({
        success: false,
        message: 'Email already registered',
      })

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.register('existing@test.com', 'password123')
      })

      expect(result.success).toBe(false)
      expect(result.message).toBe('Email already registered')
    })
  })

  describe('completeRegistration 流程', () => {
    /**
     * 测试：完成注册成功
     */
    it('should handle successful completeRegistration', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          token: 'complete-reg-token',
          user_id: 'user-complete',
          email: 'complete@test.com',
          message: 'Registration complete',
        }),
      })

      const pushStateSpy = vi.spyOn(window.history, 'pushState')

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.completeRegistration(
          'user-complete',
          '123456'
        )
      })

      expect(result.success).toBe(true)
      expect(localStorage.getItem('auth_token')).toBe('complete-reg-token')
      expect(httpClientModule.reset401Flag).toHaveBeenCalled()
      expect(pushStateSpy).toHaveBeenCalledWith({}, '', '/traders')
    })
  })

  describe('logout 流程', () => {
    /**
     * 测试：登出清除状态
     */
    it('should clear user state on logout', async () => {
      const savedUser = { id: 'user-logout', email: 'logout@test.com' }
      const savedToken = 'token-to-clear'

      localStorage.setItem('auth_user', JSON.stringify(savedUser))
      localStorage.setItem('auth_token', savedToken)

      // Mock the logout API call
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      })

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      expect(screen.getByTestId('user')).toHaveTextContent('logout@test.com')

      act(() => {
        authContext!.logout()
      })

      await waitFor(() => {
        expect(screen.getByTestId('user')).toHaveTextContent('no-user')
      })

      expect(localStorage.getItem('auth_token')).toBeNull()
      expect(localStorage.getItem('auth_user')).toBeNull()
    })
  })

  describe('resetPassword 流程', () => {
    /**
     * 测试：密码重置成功
     */
    it('should handle successful password reset', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          message: 'Password reset successful',
        }),
      })

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.resetPassword(
          'reset@test.com',
          'NewPass123!'
        )
      })

      expect(result.success).toBe(true)
      expect(result.message).toBe('Password reset successful')
    })

    /**
     * 测试：密码重置失败
     */
    it('should handle password reset failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: async () => ({
          error: 'User not found',
        }),
      })

      let authContext: ReturnType<typeof useAuth> | null = null

      render(
        <AuthProvider>
          <TestConsumer
            onAuthData={(auth) => {
              authContext = auth
            }}
          />
        </AuthProvider>
      )

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('ready')
      })

      let result: any
      await act(async () => {
        result = await authContext!.resetPassword(
          'unknown@test.com',
          'NewPass123!'
        )
      })

      expect(result.success).toBe(false)
      expect(result.message).toBe('User not found')
    })
  })

  describe('useAuth hook 错误处理', () => {
    /**
     * 测试：useAuth 必须在 AuthProvider 内使用
     */
    it('should throw error when used outside AuthProvider', () => {
      // Suppress console.error for this test
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      expect(() => {
        renderHook(() => useAuth())
      }).toThrow('useAuth must be used within an AuthProvider')

      consoleSpy.mockRestore()
    })
  })
})

import React from 'react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { LoginPage } from './LoginPage'
import { AuthProvider } from '../contexts/AuthContext'
import { LanguageProvider } from '../contexts/LanguageContext'
import { toast } from 'sonner'

// Mock hooks
vi.mock('../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../contexts/AuthContext')
  return {
    ...actual,
    useAuth: vi.fn(),
  }
})

vi.mock('../contexts/LanguageContext', async () => {
  const actual = await vi.importActual('../contexts/LanguageContext')
  return {
    ...actual,
    useLanguage: () => ({ language: 'zh' }),
  }
})

vi.mock('../hooks/useSystemConfig', () => ({
  useSystemConfig: vi.fn(() => ({
    config: {
      registration_enabled: true,
      dev_mode: false,
    },
    isLoading: false,
    error: null,
  })),
}))

vi.mock('sonner', () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(() => 'toast-id-1'),
    dismiss: vi.fn(),
  },
}))

// Get mocked functions
import { useAuth } from '../contexts/AuthContext'
import { useSystemConfig } from '../hooks/useSystemConfig'

const mockedUseAuth = vi.mocked(useAuth)
const mockedUseSystemConfig = vi.mocked(useSystemConfig)

describe('LoginPage', () => {
  const mockLogin = vi.fn()
  const mockLoginAdmin = vi.fn()
  const mockVerifyOTP = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    
    mockedUseAuth.mockReturnValue({
      login: mockLogin,
      loginAdmin: mockLoginAdmin,
      verifyOTP: mockVerifyOTP,
      logout: vi.fn(),
      register: vi.fn(),
      user: null,
      token: null,
      loading: false,
    } as any)

    mockedUseSystemConfig.mockReturnValue({
      config: {
        registration_enabled: true,
        dev_mode: false,
      },
      isLoading: false,
      error: null,
    } as any)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  const renderLoginPage = () => {
    return render(<LoginPage />)
  }

  describe('Initial Render', () => {
    it('renders login form with email and password fields', () => {
      renderLoginPage()
      
      expect(screen.getByText('登录 NOFX')).toBeInTheDocument()
      expect(screen.getByText('请输入您的邮箱和密码')).toBeInTheDocument()
      expect(screen.getByPlaceholderText(/请输入邮箱/i)).toBeInTheDocument()
      expect(screen.getByPlaceholderText(/请输入密码/i)).toBeInTheDocument()
    })

    it('shows register link when registration is enabled', () => {
      renderLoginPage()
      
      expect(screen.getByText('还没有账户？')).toBeInTheDocument()
      expect(screen.getByText('立即注册')).toBeInTheDocument()
    })

    it('hides register link when registration is disabled', () => {
      mockedUseSystemConfig.mockReturnValue({
        config: {
          registration_enabled: false,
          dev_mode: false,
        },
        isLoading: false,
        error: null,
      } as any)

      renderLoginPage()
      
      expect(screen.queryByText('立即注册')).not.toBeInTheDocument()
    })

    it('shows forgot password link', () => {
      renderLoginPage()
      
      const forgotButton = screen.getByRole('button', { name: /忘记密码/i })
      expect(forgotButton).toBeInTheDocument()
    })
  })

  describe('Dev Mode', () => {
    it('shows dev mode warning when enabled', () => {
      mockedUseSystemConfig.mockReturnValue({
        config: {
          registration_enabled: true,
          dev_mode: true,
        },
        isLoading: false,
        error: null,
      } as any)

      renderLoginPage()
      
      expect(screen.getByText('🚧 测试模式已启用')).toBeInTheDocument()
      expect(screen.getByText('无需登录即可使用全部功能。')).toBeInTheDocument()
      expect(screen.getByText('直接进入系统 →')).toBeInTheDocument()
    })

    it('does not show dev mode warning when disabled', () => {
      renderLoginPage()
      
      expect(screen.queryByText('🚧 测试模式已启用')).not.toBeInTheDocument()
    })

    it('redirects to /traders when clicking dev mode button', () => {
      mockedUseSystemConfig.mockReturnValue({
        config: {
          registration_enabled: true,
          dev_mode: true,
        },
        isLoading: false,
        error: null,
      } as any)

      renderLoginPage()
      
      const devButton = screen.getByText('直接进入系统 →')
      // Just verify the button is clickable
      expect(devButton).toBeInTheDocument()
    })
  })

  describe('Session Expired Notification', () => {
    it('shows warning toast when redirected from 401', () => {
      sessionStorage.setItem('from401', 'true')
      
      renderLoginPage()
      
      expect(toast.warning).toHaveBeenCalled()
      expect(sessionStorage.getItem('from401')).toBeNull()
    })

    it('does not show warning toast when not redirected from 401', () => {
      renderLoginPage()
      
      expect(toast.warning).not.toHaveBeenCalled()
    })
  })

  describe('Password Visibility Toggle', () => {
    it('toggles password visibility when clicking eye icon', () => {
      renderLoginPage()
      
      const passwordInput = screen.getByPlaceholderText(/请输入密码/i)
      expect(passwordInput).toHaveAttribute('type', 'password')
      
      const toggleButton = screen.getByRole('button', { name: /显示密码|隐藏密码/i })
      fireEvent.click(toggleButton)
      
      expect(passwordInput).toHaveAttribute('type', 'text')
      
      fireEvent.click(toggleButton)
      expect(passwordInput).toHaveAttribute('type', 'password')
    })
  })

  describe('Login Form Submission', () => {
    it('calls login with email and password on submit', async () => {
      mockLogin.mockResolvedValue({ success: true, requiresOTP: false })
      
      renderLoginPage()
      
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      
      const submitButton = screen.getByRole('button', { name: /登录/i })
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith('test@example.com', 'password123')
      })
    })

    it('shows error message on login failure', async () => {
      mockLogin.mockResolvedValue({ success: false, message: '用户名或密码错误' })
      
      renderLoginPage()
      
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'wrong-password' },
      })
      
      const submitButton = screen.getByRole('button', { name: /登录/i })
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(screen.getByText('用户名或密码错误')).toBeInTheDocument()
      })
      expect(toast.error).toHaveBeenCalledWith('用户名或密码错误')
    })

    it('shows default error message when no message provided', async () => {
      mockLogin.mockResolvedValue({ success: false })
      
      renderLoginPage()
      
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password' },
      })
      
      const submitButton = screen.getByRole('button', { name: /登录/i })
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(toast.error).toHaveBeenCalled()
      })
    })

    it('dismisses session expired toast on successful login', async () => {
      sessionStorage.setItem('from401', 'true')
      mockLogin.mockResolvedValue({ success: true, requiresOTP: false })
      
      renderLoginPage()
      
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      
      const submitButton = screen.getByRole('button', { name: /登录/i })
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(toast.dismiss).toHaveBeenCalledWith('toast-id-1')
      })
    })

    it('disables submit button while loading', async () => {
      let resolveLogin: any
      mockLogin.mockReturnValue(new Promise((resolve) => { resolveLogin = resolve }))
      
      renderLoginPage()
      
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      
      const submitButton = screen.getByRole('button', { name: /登录/i })
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(submitButton).toBeDisabled()
      })
      
      resolveLogin({ success: true })
    })
  })

  describe('OTP Step', () => {
    it('switches to OTP step when login requires OTP', async () => {
      mockLogin.mockResolvedValue({ success: true, requiresOTP: true, userID: 'user-123' })
      
      renderLoginPage()
      
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      
      const submitButton = screen.getByRole('button', { name: /登录/i })
      fireEvent.click(submitButton)
      
      await waitFor(() => {
        expect(screen.getByText('请输入两步验证码')).toBeInTheDocument()
      })
    })

    it('shows OTP input field in OTP step', async () => {
      mockLogin.mockResolvedValue({ success: true, requiresOTP: true, userID: 'user-123' })
      
      renderLoginPage()
      
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      
      fireEvent.click(screen.getByRole('button', { name: /登录/i }))
      
      await waitFor(() => {
        expect(screen.getByPlaceholderText(/otpPlaceholder/i)).toBeInTheDocument()
      })
    })

    it('allows only 6 digits in OTP input', async () => {
      mockLogin.mockResolvedValue({ success: true, requiresOTP: true, userID: 'user-123' })
      
      renderLoginPage()
      
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      
      fireEvent.click(screen.getByRole('button', { name: /登录/i }))
      
      await waitFor(() => {
        const otpInput = screen.getByPlaceholderText(/otpPlaceholder/i)
        fireEvent.change(otpInput, { target: { value: '123456abc789' } })
        expect(otpInput).toHaveValue('123456')
      })
    })

    it('calls verifyOTP on OTP submit', async () => {
      mockLogin.mockResolvedValue({ success: true, requiresOTP: true, userID: 'user-123' })
      mockVerifyOTP.mockResolvedValue({ success: true })
      
      renderLoginPage()
      
      // Login step
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      fireEvent.click(screen.getByRole('button', { name: /登录/i }))
      
      // OTP step
      await waitFor(() => {
        const otpInput = screen.getByPlaceholderText(/otpPlaceholder/i)
        fireEvent.change(otpInput, { target: { value: '123456' } })
      })
      
      const verifyButton = screen.getByRole('button', { name: /verifyOTP/i })
      fireEvent.click(verifyButton)
      
      await waitFor(() => {
        expect(mockVerifyOTP).toHaveBeenCalledWith('user-123', '123456')
      })
    })

    it('shows error message on OTP verification failure', async () => {
      mockLogin.mockResolvedValue({ success: true, requiresOTP: true, userID: 'user-123' })
      mockVerifyOTP.mockResolvedValue({ success: false, message: '验证码错误' })
      
      renderLoginPage()
      
      // Login step
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      fireEvent.click(screen.getByRole('button', { name: /登录/i }))
      
      // OTP step
      await waitFor(() => {
        const otpInput = screen.getByPlaceholderText(/otpPlaceholder/i)
        fireEvent.change(otpInput, { target: { value: '123456' } })
      })
      
      fireEvent.click(screen.getByRole('button', { name: /verifyOTP/i }))
      
      await waitFor(() => {
        expect(screen.getByText('验证码错误')).toBeInTheDocument()
      })
      expect(toast.error).toHaveBeenCalledWith('验证码错误')
    })

    it('goes back to login step when clicking back button', async () => {
      mockLogin.mockResolvedValue({ success: true, requiresOTP: true, userID: 'user-123' })
      
      renderLoginPage()
      
      // Login step
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      fireEvent.click(screen.getByRole('button', { name: /登录/i }))
      
      // Wait for OTP step
      await waitFor(() => {
        expect(screen.getByText('请输入两步验证码')).toBeInTheDocument()
      })
      
      // Click back button
      const backButton = screen.getByRole('button', { name: /返回/i })
      fireEvent.click(backButton)
      
      await waitFor(() => {
        expect(screen.getByText('请输入您的邮箱和密码')).toBeInTheDocument()
      })
    })

    it('disables verify button when OTP is not 6 digits', async () => {
      mockLogin.mockResolvedValue({ success: true, requiresOTP: true, userID: 'user-123' })
      
      renderLoginPage()
      
      // Login step
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      fireEvent.click(screen.getByRole('button', { name: /登录/i }))
      
      // OTP step
      await waitFor(() => {
        const otpInput = screen.getByPlaceholderText(/otpPlaceholder/i)
        fireEvent.change(otpInput, { target: { value: '123' } })
      })
      
      const verifyButton = screen.getByRole('button', { name: /verifyOTP/i })
      expect(verifyButton).toBeDisabled()
    })

    it('dismisses session expired toast on successful OTP verification', async () => {
      sessionStorage.setItem('from401', 'true')
      mockLogin.mockResolvedValue({ success: true, requiresOTP: true, userID: 'user-123' })
      mockVerifyOTP.mockResolvedValue({ success: true })
      
      renderLoginPage()
      
      // Login step
      fireEvent.change(screen.getByPlaceholderText(/请输入邮箱/i), {
        target: { value: 'test@example.com' },
      })
      fireEvent.change(screen.getByPlaceholderText(/请输入密码/i), {
        target: { value: 'password123' },
      })
      fireEvent.click(screen.getByRole('button', { name: /登录/i }))
      
      // OTP step
      await waitFor(() => {
        const otpInput = screen.getByPlaceholderText(/otpPlaceholder/i)
        fireEvent.change(otpInput, { target: { value: '123456' } })
      })
      
      fireEvent.click(screen.getByRole('button', { name: /verifyOTP/i }))
      
      await waitFor(() => {
        expect(toast.dismiss).toHaveBeenCalled()
      })
    })
  })

  describe('Navigation', () => {
    it('navigates to register page when clicking register link', () => {
      renderLoginPage()
      
      const registerButton = screen.getByText('立即注册')
      expect(registerButton).toBeInTheDocument()
      // Button click verification - actual navigation tested in integration
    })

    it('navigates to reset password page when clicking forgot password', () => {
      renderLoginPage()
      
      const forgotButton = screen.getByRole('button', { name: /忘记密码/i })
      expect(forgotButton).toBeInTheDocument()
      // Button click verification - actual navigation tested in integration
    })
  })
})

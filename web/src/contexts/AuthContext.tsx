import React, { createContext, useContext, useState, useEffect } from 'react'
import { getSystemConfig } from '../lib/config'
import { reset401Flag, httpClient } from '../lib/httpClient'

interface User {
  id: string
  email: string
}

interface AuthContextType {
  user: User | null
  token: string | null
  login: (
    email: string,
    password: string
  ) => Promise<{
    success: boolean
    message?: string
    requiresOTP?: boolean
    userID?: string
  }>
  loginAdmin: (password: string) => Promise<{
    success: boolean
    message?: string
  }>
  register: (
    email: string,
    password: string,
    betaCode?: string
  ) => Promise<{
    success: boolean
    message?: string
    userID?: string
    qrCodeURL?: string
    otpSecret?: string
    requiresOTP?: boolean
  }>
  completeRegistration: (
    userId: string,
    otpCode: string
  ) => Promise<{
    success: boolean
    message?: string
  }>
  verifyOTP: (
    userId: string,
    otpCode: string
  ) => Promise<{
    success: boolean
    message?: string
  }>
  resetPassword: (
    email: string,
    newPassword: string
  ) => Promise<{ success: boolean; message?: string }>
  logout: () => void
  isLoading: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    // Reset 401 flag on page load to allow fresh 401 handling
    reset401Flag()

    // 先检查是否为管理员模式（使用带缓存的系统配置获取）
    getSystemConfig()
      .then(() => {
        // 不再在管理员模式下模拟登录；统一检查本地存储
        const savedToken = localStorage.getItem('auth_token')
        const savedUser = localStorage.getItem('auth_user')
        if (savedToken && savedUser) {
          setToken(savedToken)
          setUser(JSON.parse(savedUser))
        }

        setIsLoading(false)
      })
      .catch((err) => {
        console.error('Failed to fetch system config:', err)
        // 发生错误时，继续检查本地存储
        const savedToken = localStorage.getItem('auth_token')
        const savedUser = localStorage.getItem('auth_user')

        if (savedToken && savedUser) {
          setToken(savedToken)
          setUser(JSON.parse(savedUser))
        }
        setIsLoading(false)
      })
  }, [])

  // Listen for unauthorized events from httpClient (401 responses)
  useEffect(() => {
    const handleUnauthorized = (event: Event) => {
      // Cast to CustomEvent to access detail
      const customEvent = event as CustomEvent<{ triggerToken?: string }>
      const triggerToken = customEvent.detail?.triggerToken

      // Get current token from localStorage (most reliable source of truth)
      const currentToken = localStorage.getItem('auth_token')

      console.error('[AuthContext] 🚨 Unauthorized event received', {
        triggerToken: triggerToken
          ? `${triggerToken.substring(0, 10)}...`
          : 'null',
        currentToken: currentToken
          ? `${currentToken.substring(0, 10)}...`
          : 'null',
        match: triggerToken === currentToken,
      })

      // Guard logic:
      // If the event carries a triggerToken, we MUST verify it matches the current token.
      // If they don't match, it means the 401 came from a request using an old token.
      if (triggerToken && currentToken && triggerToken !== currentToken) {
        console.error(
          '[AuthContext] 🛑 BLOCKED: Ignoring unauthorized event from stale token'
        )
        return
      }

      console.error('[AuthContext] ✅ CONFIRMED: Clearing auth state')
      // Clear auth state when 401 is detected
      setUser(null)
      setToken(null)
      // Note: localStorage cleanup is already done in httpClient
    }

    window.addEventListener('unauthorized', handleUnauthorized)

    return () => {
      window.removeEventListener('unauthorized', handleUnauthorized)
    }
  }, [])

  const login = async (email: string, password: string) => {
    try {
      const response = await fetch('/api/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, password }),
      })

      const data = await response.json()

      if (response.ok) {
        if (data.requires_otp) {
          // 需要OTP验证
          return {
            success: true,
            requiresOTP: true,
            userID: data.user_id,
            message: data.message,
          }
        } else {
          // Reset 401 flag on successful login
          reset401Flag()

          // 登录成功，保存token和用户信息
          const userInfo = { id: data.user_id, email: data.email }
          localStorage.setItem('auth_token', data.token)
          localStorage.setItem('auth_user', JSON.stringify(userInfo))
          setToken(data.token)
          setUser(userInfo)

          // Check and redirect to returnUrl if exists
          const returnUrl = sessionStorage.getItem('returnUrl')
          if (returnUrl) {
            sessionStorage.removeItem('returnUrl')
            window.history.pushState({}, '', returnUrl)
            window.dispatchEvent(new PopStateEvent('popstate'))
          } else {
            // 跳转到配置页面
            window.history.pushState({}, '', '/traders')
            window.dispatchEvent(new PopStateEvent('popstate'))
          }

          return { success: true, message: data.message }
        }
      } else {
        return { success: false, message: data.error }
      }
    } catch (error) {
      return { success: false, message: '登录失败，请重试' }
    }
  }

  const loginAdmin = async (password: string) => {
    try {
      const response = await fetch('/api/admin-login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      })
      const data = await response.json()
      if (response.ok) {
        // Reset 401 flag on successful login
        reset401Flag()

        const userInfo = {
          id: data.user_id || 'admin',
          email: data.email || 'admin@localhost',
        }
        localStorage.setItem('auth_token', data.token)
        localStorage.setItem('auth_user', JSON.stringify(userInfo))
        setToken(data.token)
        setUser(userInfo)

        // Check and redirect to returnUrl if exists
        const returnUrl = sessionStorage.getItem('returnUrl')
        if (returnUrl) {
          sessionStorage.removeItem('returnUrl')
          window.history.pushState({}, '', returnUrl)
          window.dispatchEvent(new PopStateEvent('popstate'))
        } else {
          // 跳转到仪表盘
          window.history.pushState({}, '', '/dashboard')
          window.dispatchEvent(new PopStateEvent('popstate'))
        }
        return { success: true }
      } else {
        return { success: false, message: data.error || '登录失败' }
      }
    } catch (e) {
      return { success: false, message: '登录失败，请重试' }
    }
  }

  const register = async (
    email: string,
    password: string,
    betaCode?: string
  ) => {
    const requestBody: {
      email: string
      password: string
      beta_code?: string
    } = { email, password }
    if (betaCode) {
      requestBody.beta_code = betaCode
    }

    const result = await httpClient.post<{
      user_id: string
      otp_secret: string
      qr_code_url: string
      message: string
    }>('/api/register', requestBody)

    if (result.success && result.data) {
      return {
        success: true,
        userID: result.data.user_id,
        otpSecret: result.data.otp_secret,
        qrCodeURL: result.data.qr_code_url,
        message: result.message || result.data.message,
      }
    }

    // Only business errors reach here (system/network errors were intercepted)
    return {
      success: false,
      message: result.message || 'Registration failed',
    }
  }

  const resetPassword = async (email: string, newPassword: string) => {
    try {
      const response = await fetch('/api/reset-password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email, new_password: newPassword }),
      })

      const data = await response.json()

      if (response.ok) {
        return { success: true, message: data.message }
      } else {
        return { success: false, message: data.error }
      }
    } catch (error) {
      return { success: false, message: '密码重置失败，请重试' }
    }
  }

  const logout = () => {
    if (token) {
      fetch('/api/logout', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
      }).catch(() => {
        /* ignore errors on logout */
      })
    }
    setUser(null)
    setToken(null)
    localStorage.removeItem('auth_token')
    localStorage.removeItem('auth_user')
  }

  const completeRegistration = async (userId: string, otpCode: string) => {
    try {
      const response = await fetch('/api/complete-registration', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ user_id: userId, otp_code: otpCode }),
      })

      const data = await response.json()

      if (response.ok) {
        // Reset 401 flag on successful registration
        reset401Flag()

        // 注册完成，保存token和用户信息
        const userInfo = { id: data.user_id, email: data.email }
        localStorage.setItem('auth_token', data.token)
        localStorage.setItem('auth_user', JSON.stringify(userInfo))
        setToken(data.token)
        setUser(userInfo)

        // 跳转到配置页面
        window.history.pushState({}, '', '/traders')
        window.dispatchEvent(new PopStateEvent('popstate'))

        return { success: true, message: data.message }
      } else {
        return { success: false, message: data.error }
      }
    } catch (error) {
      return { success: false, message: '验证失败，请重试' }
    }
  }

  const verifyOTP = async (userId: string, otpCode: string) => {
    try {
      const response = await fetch('/api/verify-otp', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ user_id: userId, otp_code: otpCode }),
      })

      const data = await response.json()

      if (response.ok) {
        // Reset 401 flag on successful login
        reset401Flag()

        // 登录成功，保存token和用户信息
        const userInfo = { id: data.user_id, email: data.email }
        localStorage.setItem('auth_token', data.token)
        localStorage.setItem('auth_user', JSON.stringify(userInfo))
        setToken(data.token)
        setUser(userInfo)

        // Check and redirect to returnUrl if exists
        const returnUrl = sessionStorage.getItem('returnUrl')
        if (returnUrl) {
          sessionStorage.removeItem('returnUrl')
          window.history.pushState({}, '', returnUrl)
          window.dispatchEvent(new PopStateEvent('popstate'))
        } else {
          // 跳转到配置页面
          window.history.pushState({}, '', '/traders')
          window.dispatchEvent(new PopStateEvent('popstate'))
        }

        return { success: true, message: data.message }
      } else {
        return { success: false, message: data.error }
      }
    } catch (error) {
      return { success: false, message: '验证失败，请重试' }
    }
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        login,
        loginAdmin,
        register,
        completeRegistration,
        verifyOTP,
        resetPassword,
        logout,
        isLoading,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

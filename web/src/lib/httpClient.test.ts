import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Use vi.hoisted to ensure the mock instance is available for the hoisted vi.mock call
const { mockAxiosInstance } = vi.hoisted(() => {
  return {
    mockAxiosInstance: {
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
      request: vi.fn(),
    },
  }
})

// Mock axios BEFORE importing the module
vi.mock('axios', () => {
  return {
    default: {
      create: vi.fn(() => mockAxiosInstance),
      isAxiosError: vi.fn(() => true),
    },
  }
})

// Import after mocking
import { HttpClient, reset401Flag } from './httpClient'

describe('HttpClient 401 Handling', () => {
  let client: HttpClient

  beforeEach(() => {
    // Reset localStorage
    localStorage.clear()
    sessionStorage.clear()

    // Reset static flag!
    reset401Flag()

    // Reset window location mock
    Object.defineProperty(window, 'location', {
      value: {
        pathname: '/',
        search: '',
        href: '',
      },
      writable: true,
    })

    // Clear mock calls
    vi.clearAllMocks()

    // Create a NEW instance for each test
    client = new HttpClient()
  })

  it('should clear token on 401 if tokens match (normal case)', async () => {
    // Setup initial state
    const token = 'valid-token'
    localStorage.setItem('auth_token', token)

    // Get the error interceptor
    const errorInterceptor =
      mockAxiosInstance.interceptors.response.use.mock.calls[0][1]

    // Simulate 401 error with matching token
    const error = {
      response: {
        status: 401,
        data: { error: 'Unauthorized' },
      },
      config: {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      },
    }

    // Spy on window dispatchEvent
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')

    // Execute interceptor
    // We don't await the result because it returns a never-resolving promise (to halt execution during redirect)
    // But the side effects (clearing storage) happen synchronously before that.
    errorInterceptor(error).catch(() => {})

    // Wait a tick to ensure synchronous code ran
    await new Promise((resolve) => setTimeout(resolve, 0))

    // Assertions
    expect(localStorage.getItem('auth_token')).toBeNull()
    expect(dispatchSpy).toHaveBeenCalledWith(expect.any(Event))
    expect(dispatchSpy.mock.calls[0][0].type).toBe('unauthorized')
  })

  it('should NOT clear token on 401 if tokens mismatch (race condition)', async () => {
    // Setup initial state with NEW token
    const oldToken = 'old-expired-token'
    const newToken = 'new-valid-token'
    localStorage.setItem('auth_token', newToken)

    // Get the error interceptor
    const errorInterceptor =
      mockAxiosInstance.interceptors.response.use.mock.calls[0][1]

    // Simulate 401 error from OLD request
    const error = {
      response: {
        status: 401,
        data: { error: 'Unauthorized' },
      },
      config: {
        headers: {
          Authorization: `Bearer ${oldToken}`,
        },
      },
    }

    // Spy on window dispatchEvent
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')

    // Execute interceptor
    // In the race condition case (if fixed), it should just return/throw and not hang.
    // In the buggy case, it will hang (and clear storage).
    // We use a timeout or just check side effects.
    try {
      errorInterceptor(error).catch(() => {})
    } catch (e) {
      // Ignore errors in race condition test
    }

    // Wait a tick
    await new Promise((resolve) => setTimeout(resolve, 0))

    // Assertions - CRITICAL: Token should still be the NEW one
    // In the current buggy implementation, this fails because it gets cleared
    expect(localStorage.getItem('auth_token')).toBe(newToken)
    expect(dispatchSpy).not.toHaveBeenCalled()
  })

  it('should NOT clear token on 401 if request had no token but new token now exists (pre-login race)', async () => {
    // This tests the scenario where:
    // 1. A request is made without a token (before login)
    // 2. User logs in via OTP, new token is stored
    // 3. The old request returns 401
    // 4. The new token should NOT be cleared

    const newToken = 'newly-acquired-token'
    localStorage.setItem('auth_token', newToken)

    // Get the error interceptor
    const errorInterceptor =
      mockAxiosInstance.interceptors.response.use.mock.calls[0][1]

    // Simulate 401 error from a request that had NO token (undefined Authorization)
    const error = {
      response: {
        status: 401,
        data: { error: 'Unauthorized' },
      },
      config: {
        headers: {
          // No Authorization header - request was made before login
        },
      },
    }

    // Spy on window dispatchEvent
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')

    // Execute interceptor
    try {
      await errorInterceptor(error)
    } catch (e) {
      // Expected to reject
    }

    // Wait a tick
    await new Promise((resolve) => setTimeout(resolve, 0))

    // Assertions - CRITICAL: Token should still be present
    expect(localStorage.getItem('auth_token')).toBe(newToken)
    expect(dispatchSpy).not.toHaveBeenCalled()
  })

  it('should clear token on 401 when no current token exists', async () => {
    // This tests the normal case where user is truly logged out
    // and a 401 should trigger the logout flow

    // No token in localStorage - user is not logged in

    // Get the error interceptor
    const errorInterceptor =
      mockAxiosInstance.interceptors.response.use.mock.calls[0][1]

    // Simulate 401 error
    const error = {
      response: {
        status: 401,
        data: { error: 'Unauthorized' },
      },
      config: {
        headers: {},
      },
    }

    // Spy on window dispatchEvent
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')

    // Execute interceptor
    errorInterceptor(error).catch(() => {})

    // Wait a tick
    await new Promise((resolve) => setTimeout(resolve, 0))

    // When there's no token, it should still dispatch unauthorized event
    expect(dispatchSpy).toHaveBeenCalledWith(expect.any(Event))
    expect(dispatchSpy.mock.calls[0][0].type).toBe('unauthorized')
  })

  it('should handle concurrent 401s during login flow (real scenario simulation)', async () => {
    // This simulates the real-world scenario that caused the bug:
    // 1. User is on login page (no token)
    // 2. Background requests are made (e.g., config checks) - these will get 401
    // 3. User completes OTP verification, new token is stored
    // 4. Background request's 401 response arrives
    // 5. New token should NOT be cleared

    // Get the error interceptor

    const errorInterceptor =
      mockAxiosInstance.interceptors.response.use.mock.calls[0][1]

    // Spy on window dispatchEvent
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')

    // Simulate background request error (made before login, no token)
    const backgroundRequestError = {
      response: {
        status: 401,
        data: { error: 'Unauthorized' },
      },
      config: {
        headers: {
          // No Authorization header - request was made before login
        },
      },
    }

    // Simulate login completing and storing new token
    const newToken = 'fresh-otp-verified-token'
    localStorage.setItem('auth_token', newToken)
    localStorage.setItem(
      'auth_user',
      JSON.stringify({ id: 'user1', email: 'test@example.com' })
    )

    // Now the background request's 401 response arrives
    try {
      await errorInterceptor(backgroundRequestError)
    } catch (e) {
      // Expected to reject
    }

    await new Promise((resolve) => setTimeout(resolve, 0))

    // CRITICAL: Token should still be present
    expect(localStorage.getItem('auth_token')).toBe(newToken)
    expect(localStorage.getItem('auth_user')).not.toBeNull()
    expect(dispatchSpy).not.toHaveBeenCalled()
  })
})

describe('HttpClient request interceptor', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('should attach freshly stored token before SWR-triggered requests fire', () => {
    new HttpClient()
    const requestInterceptor =
      mockAxiosInstance.interceptors.request.use.mock.calls[0][0]

    // Simulate OTP flow writing token to localStorage right before SWR issues request
    const newToken = 'brand-new-token'
    localStorage.setItem('auth_token', newToken)

    const config = requestInterceptor({ headers: {} })

    expect(config.headers.Authorization).toBe(`Bearer ${newToken}`)
  })
})

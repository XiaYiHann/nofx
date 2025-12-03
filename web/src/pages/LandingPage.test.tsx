/**
 * LandingPage.test.tsx
 *
 * 测试目的：确保 LandingPage 的 HeaderBar onPageChange 回调正确处理所有导航项
 *
 * 覆盖的关键功能：
 * 1. 登录状态下点击 News 按钮能正确导航到 /news
 * 2. 其它导航项（competition, traders, etc.）行为不受影响
 * 3. popstate 事件被正确触发
 */

import '@testing-library/jest-dom'
import {
  describe,
  it,
  expect,
  vi,
  beforeEach,
  afterEach,
  type Mock,
} from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

// Capture onPageChange callback from HeaderBar mock
let capturedOnPageChange: ((page: string) => void) | undefined

// Mock HeaderBar to expose onPageChange
vi.mock('../components/landing/HeaderBar', () => ({
  __esModule: true,
  default: ({ onPageChange, isLoggedIn }: any) => {
    capturedOnPageChange = onPageChange
    return (
      <div data-testid="header-bar">
        {isLoggedIn && (
          <>
            <button
              data-testid="nav-news"
              onClick={() => onPageChange?.('news')}
            >
              News
            </button>
            <button
              data-testid="nav-competition"
              onClick={() => onPageChange?.('competition')}
            >
              Competition
            </button>
            <button
              data-testid="nav-traders"
              onClick={() => onPageChange?.('traders')}
            >
              Traders
            </button>
            <button data-testid="nav-faq" onClick={() => onPageChange?.('faq')}>
              FAQ
            </button>
            <button
              data-testid="nav-strategies"
              onClick={() => onPageChange?.('strategies')}
            >
              Strategies
            </button>
            <button
              data-testid="nav-backtest"
              onClick={() => onPageChange?.('backtest')}
            >
              Backtest
            </button>
          </>
        )}
      </div>
    )
  },
}))

// Mock other landing page components
vi.mock('../components/landing/HeroSection', () => ({
  __esModule: true,
  default: () => <div data-testid="hero-section">HeroSection</div>,
}))
vi.mock('../components/landing/AboutSection', () => ({
  __esModule: true,
  default: () => <div data-testid="about-section">AboutSection</div>,
}))
vi.mock('../components/landing/AnimatedSection', () => ({
  __esModule: true,
  default: ({ children }: any) => (
    <div data-testid="animated-section">{children}</div>
  ),
}))
vi.mock('../components/landing/LoginModal', () => ({
  __esModule: true,
  default: () => <div data-testid="login-modal">LoginModal</div>,
}))
vi.mock('../components/landing/FooterSection', () => ({
  __esModule: true,
  default: () => <div data-testid="footer-section">FooterSection</div>,
}))

// Mock framer-motion
vi.mock('framer-motion', () => ({
  motion: {
    button: ({ children, ...props }: any) => (
      <button {...props}>{children}</button>
    ),
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
    h2: ({ children, ...props }: any) => <h2 {...props}>{children}</h2>,
    p: ({ children, ...props }: any) => <p {...props}>{children}</p>,
    a: ({ children, ...props }: any) => <a {...props}>{children}</a>,
  },
}))

// Mock lucide-react
vi.mock('lucide-react', () => ({
  ArrowRight: () => <span data-testid="arrow-right">→</span>,
}))

// Mock hooks
const mockUseAuth = vi.fn()
const mockUseLanguage = vi.fn()

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
}))

vi.mock('../contexts/LanguageContext', () => ({
  useLanguage: () => mockUseLanguage(),
}))

// Mock translations
vi.mock('../i18n/translations', () => ({
  t: (key: string) => key,
}))

// Import LandingPage after mocks are set up
import { LandingPage } from './LandingPage'

describe('LandingPage', () => {
  let pushStateSpy: ReturnType<typeof vi.spyOn>
  let dispatchEventSpy: Mock

  beforeEach(() => {
    vi.clearAllMocks()
    capturedOnPageChange = undefined

    // Spy on window.history.pushState and window.dispatchEvent
    pushStateSpy = vi.spyOn(window.history, 'pushState')
    dispatchEventSpy = vi.fn()
    window.dispatchEvent = dispatchEventSpy

    // Default mock implementations
    mockUseLanguage.mockReturnValue({
      language: 'zh',
      setLanguage: vi.fn(),
    })
  })

  afterEach(() => {
    pushStateSpy.mockRestore()
    vi.resetAllMocks()
  })

  describe('已登录状态下的导航', () => {
    beforeEach(() => {
      mockUseAuth.mockReturnValue({
        user: { email: 'test@example.com' },
        logout: vi.fn(),
      })
    })

    it('should navigate to /news when News button is clicked', () => {
      render(<LandingPage />)

      const newsButton = screen.getByTestId('nav-news')
      fireEvent.click(newsButton)

      // Verify pushState was called with /news
      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/news')

      // Verify popstate event was dispatched
      expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent))
    })

    it('should navigate to /competition when Competition button is clicked', () => {
      render(<LandingPage />)

      const competitionButton = screen.getByTestId('nav-competition')
      fireEvent.click(competitionButton)

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/competition')
      expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent))
    })

    it('should navigate to /traders when Traders button is clicked', () => {
      render(<LandingPage />)

      const tradersButton = screen.getByTestId('nav-traders')
      fireEvent.click(tradersButton)

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/traders')
      expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent))
    })

    it('should navigate to /faq when FAQ button is clicked', () => {
      render(<LandingPage />)

      const faqButton = screen.getByTestId('nav-faq')
      fireEvent.click(faqButton)

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/faq')
      expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent))
    })

    it('should navigate to /strategies when Strategies button is clicked', () => {
      render(<LandingPage />)

      const strategiesButton = screen.getByTestId('nav-strategies')
      fireEvent.click(strategiesButton)

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/strategies')
      expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent))
    })

    it('should navigate to /backtest when Backtest button is clicked', () => {
      render(<LandingPage />)

      const backtestButton = screen.getByTestId('nav-backtest')
      fireEvent.click(backtestButton)

      expect(pushStateSpy).toHaveBeenCalledWith(null, '', '/backtest')
      expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent))
    })
  })

  describe('未登录状态', () => {
    beforeEach(() => {
      mockUseAuth.mockReturnValue({
        user: null,
        logout: vi.fn(),
      })
    })

    it('should render LandingPage without nav buttons when not logged in', () => {
      render(<LandingPage />)

      // HeaderBar should be rendered but without nav buttons (isLoggedIn=false)
      expect(screen.getByTestId('header-bar')).toBeInTheDocument()
      expect(screen.queryByTestId('nav-news')).not.toBeInTheDocument()
    })

    it('should render core page sections', () => {
      render(<LandingPage />)

      expect(screen.getByTestId('hero-section')).toBeInTheDocument()
      expect(screen.getByTestId('about-section')).toBeInTheDocument()
      expect(screen.getByTestId('footer-section')).toBeInTheDocument()
    })
  })

  describe('onPageChange 回调完整性', () => {
    beforeEach(() => {
      mockUseAuth.mockReturnValue({
        user: { email: 'test@example.com' },
        logout: vi.fn(),
      })
    })

    it('should have onPageChange callback passed to HeaderBar', () => {
      render(<LandingPage />)

      expect(capturedOnPageChange).toBeDefined()
      expect(typeof capturedOnPageChange).toBe('function')
    })

    it('should handle all page values correctly', () => {
      render(<LandingPage />)

      const pageRoutes: Array<{ page: string; expectedPath: string }> = [
        { page: 'competition', expectedPath: '/competition' },
        { page: 'traders', expectedPath: '/traders' },
        { page: 'trader', expectedPath: '/dashboard' },
        { page: 'faq', expectedPath: '/faq' },
        { page: 'strategies', expectedPath: '/strategies' },
        { page: 'backtest', expectedPath: '/backtest' },
        { page: 'news', expectedPath: '/news' },
      ]

      pageRoutes.forEach(({ page, expectedPath }) => {
        // Clear spies for each iteration
        pushStateSpy.mockClear()
        dispatchEventSpy.mockClear()

        // Trigger onPageChange
        capturedOnPageChange?.(page)

        expect(pushStateSpy).toHaveBeenCalledWith(null, '', expectedPath)
        expect(dispatchEventSpy).toHaveBeenCalledWith(expect.any(PopStateEvent))
      })
    })
  })
})

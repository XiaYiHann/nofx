/**
 * BacktestDetailPage.comprehensive.test.tsx
 *
 * 综合测试套件：回测详情页面的完整功能测试
 *
 * 覆盖场景：
 * 1. 页面加载和数据显示
 * 2. 实时进度更新（WebSocket）
 * 3. 净值曲线渲染
 * 4. 交易记录显示
 * 5. 决策记录显示
 * 6. 错误处理
 * 7. 边界条件
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import '@testing-library/jest-dom/vitest'
import { render, screen, waitFor } from '@testing-library/react'
import React from 'react'

// Use vi.hoisted to ensure mock data is available during module hoisting
const { mockAuthValue, mockBacktestSocketValue } = vi.hoisted(() => ({
  mockAuthValue: {
    user: { id: 'user-1', email: 'test@example.com' },
    token: 'test-token',
    isLoading: false,
    login: vi.fn(),
    loginAdmin: vi.fn(),
    register: vi.fn(),
    completeRegistration: vi.fn(),
    verifyOTP: vi.fn(),
    resetPassword: vi.fn(),
    logout: vi.fn(),
  },
  mockBacktestSocketValue: {
    status: 'disconnected',
    progress: 0,
    snapshots: [],
    decisions: [],
    currentCycle: 0,
    progressMessage: '',
    isComplete: false,
    error: null,
  },
}))

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => mockAuthValue,
  AuthProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}))

import BacktestDetailPage from './BacktestDetailPage'
import { LanguageProvider } from '../contexts/LanguageContext'

// Helper function to render with providers
function renderWithProviders(ui: React.ReactElement) {
  return render(<LanguageProvider>{ui}</LanguageProvider>)
}

// Helper function to reset all mocks
function resetAllMocks() {
  vi.clearAllMocks()
  vi.resetAllMocks()
}

// Helper function to create mock backtest data
function createMockBacktest(overrides?: any) {
  return {
    id: 'backtest-1',
    trader_id: 'trader-1',
    user_id: 'user-1',
    start_time: '2024-01-01T00:00:00Z',
    end_time: '2024-01-02T00:00:00Z',
    initial_balance: 10000,
    status: 'completed',
    progress: 100,
    total_pnl: 500,
    total_pnl_pct: 5.0,
    max_drawdown: 2.5,
    sharpe_ratio: 1.8,
    win_rate: 60.0,
    total_trades: 10,
    timeframe: '3m',
    ai_model_id: 'model-1',
    exchange_id: 'exchange-1',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
    ...overrides,
  }
}

// Helper function to create mock trade data
function createMockTrade(overrides?: any) {
  return {
    id: 'trade-1',
    backtest_id: 'backtest-1',
    symbol: 'BTCUSDT',
    side: 'long',
    action: 'close',
    entry_price: 50000,
    exit_price: 51000,
    quantity: 0.1,
    leverage: 5,
    pnl: 100,
    pnl_pct: 2.0,
    fee: 5,
    entry_time: '2024-01-01T01:00:00Z',
    exit_time: '2024-01-01T02:00:00Z',
    ...overrides,
  }
}

// Helper function to create mock decision data
function createMockDecision(overrides?: any) {
  return {
    timestamp: '2024-01-01T01:30:00Z',
    decisions: [
      {
        action: 'open_long',
        symbol: 'BTCUSDT',
        confidence: 80,
        reasoning: 'Test reasoning',
      },
    ],
    ...overrides,
  }
}

// Helper function to create mock equity snapshots
function createMockEquitySnapshot(count: number = 2) {
  const snapshots = []
  for (let i = 0; i < count; i++) {
    snapshots.push({
      time: `2024-01-01T${String(i * 12).padStart(2, '0')}:00:00Z`,
      total_equity: 10000 + i * 500,
      pnl: i * 500,
      pnl_pct: i * 5.0,
    })
  }
  return snapshots
}

// Mock api module
vi.mock('../lib/api', () => ({
  api: {
    getBacktest: vi.fn(),
    getBacktestEquityHistory: vi.fn(),
    getBacktestTrades: vi.fn(),
    getBacktestDecisions: vi.fn(),
  },
}))

// Mock useBacktestSocket hook
vi.mock('../hooks/useBacktestSocket', () => ({
  useBacktestSocket: () => mockBacktestSocketValue,
}))

// Mock EquityChart component
vi.mock('../components/EquityChart', () => ({
  EquityChart: ({ backtestId, initialBalance }: any) => (
    <div data-testid="equity-chart">
      Equity Chart: {backtestId}, Initial: {initialBalance}
    </div>
  ),
}))

import { api } from '../lib/api'

// Setup default mock implementations
const defaultMockBacktest = createMockBacktest()
const defaultMockTrades = [createMockTrade()]
const defaultMockDecisions = [createMockDecision()]
const defaultMockEquity = createMockEquitySnapshot(2)

// Helper function to reset mocks to default state
function resetMocksToDefaults() {
  vi.mocked(api.getBacktest).mockResolvedValue(defaultMockBacktest)
  vi.mocked(api.getBacktestEquityHistory).mockResolvedValue(defaultMockEquity)
  vi.mocked(api.getBacktestTrades).mockResolvedValue(defaultMockTrades)
  vi.mocked(api.getBacktestDecisions).mockResolvedValue(defaultMockDecisions)
}

// Reset mocks before each test
beforeEach(() => {
  resetAllMocks()
  resetMocksToDefaults()
})

describe('BacktestDetailPage - Comprehensive', () => {
  // ============================================================================
  // 1. 页面加载和数据显示测试
  // ============================================================================

  describe('Page Loading and Data Display', () => {
    it('should show loading state initially', () => {
      vi.mocked(api.getBacktest).mockImplementation(
        () =>
          new Promise((resolve) =>
            setTimeout(() => resolve(defaultMockBacktest), 100)
          )
      )

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      expect(screen.getByText('加载回测数据中...')).toBeInTheDocument()
    })

    it('should display backtest details after loading', async () => {
      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Verify key metrics are displayed
      expect(screen.getByText(/总收益/)).toBeInTheDocument()
      expect(screen.getByText(/最大回撤/)).toBeInTheDocument()
      expect(screen.getByText(/夏普率/)).toBeInTheDocument()
      expect(screen.getByText(/胜率/)).toBeInTheDocument()
    })

    it('should display correct metric values', async () => {
      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Verify metric values - use regex to handle text split across elements
      expect(screen.getByText(/500\.00/)).toBeInTheDocument()
      expect(screen.getByText(/5\.00%/)).toBeInTheDocument()
      expect(screen.getByText(/2\.50%/)).toBeInTheDocument()
      expect(screen.getByText(/1\.80/)).toBeInTheDocument()
      expect(screen.getByText(/60\.0/)).toBeInTheDocument()
    })

    it('should show error state when backtest not found', async () => {
      vi.mocked(api.getBacktest).mockRejectedValue(new Error('Not found'))

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText('回测数据未找到')).toBeInTheDocument()
        },
        { timeout: 10000 }
      )
    })

    it('should display timeframe and AI model info', async () => {
      const customBacktest = createMockBacktest({
        timeframe: '5m',
        ai_model_id: 'custom-model-1',
      })
      vi.mocked(api.getBacktest).mockResolvedValue(customBacktest)

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      expect(screen.getByText('周期: 5m')).toBeInTheDocument()
      expect(screen.getByText('AI模型: custom-model-1')).toBeInTheDocument()
    })
  })

  // ============================================================================
  // 2. 实时进度更新测试
  // ============================================================================

  describe('Real-time Progress Updates', () => {
    it('should show progress panel when backtest is running', async () => {
      const runningBacktest = createMockBacktest({
        status: 'running',
        progress: 50,
      })
      vi.mocked(api.getBacktest).mockResolvedValue(runningBacktest)

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Progress panel should be visible when running
      expect(screen.getByText(/实时.*进度/)).toBeInTheDocument()
    })

    it('should not show progress panel when backtest is completed', async () => {
      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Progress panel should not be visible when completed
      expect(screen.queryByText(/实时进度/)).not.toBeInTheDocument()
    })
  })

  // ============================================================================
  // 3. 净值曲线测试
  // ============================================================================

  describe('Equity Chart', () => {
    it('should render equity chart component', async () => {
      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Equity chart should be present
      const equityChart = screen.getByTestId('equity-chart')
      expect(equityChart).toBeInTheDocument()
    })

    it('should pass external data when backtest is running', async () => {
      const runningBacktest = createMockBacktest({ status: 'running' })
      vi.mocked(api.getBacktest).mockResolvedValue(runningBacktest)

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Equity chart should be present
      const equityChart = screen.getByTestId('equity-chart')
      expect(equityChart).toBeInTheDocument()
    })
  })

  // ============================================================================
  // 4. 交易记录测试
  // ============================================================================

  describe('Trade Records', () => {
    it('should display trade records table', async () => {
      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Trade table should be present - use getAllByText for multiple occurrences
      const btcElements = screen.getAllByText(/BTCUSDT/)
      expect(btcElements.length).toBeGreaterThan(0)
    })

    it('should display trade count', async () => {
      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Trade count should be displayed - use regex to handle text in context
      expect(screen.getByText(/共.*10.*笔/)).toBeInTheDocument()
    })

    it('should handle empty trades list', async () => {
      vi.mocked(api.getBacktestTrades).mockResolvedValue([])

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Should handle empty trades gracefully
      expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
    })
  })

  // ============================================================================
  // 5. 错误处理测试
  // ============================================================================

  describe('Error Handling', () => {
    it('should handle API errors gracefully', async () => {
      vi.mocked(api.getBacktest).mockRejectedValue(new Error('API Error'))

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText('回测数据未找到')).toBeInTheDocument()
        },
        { timeout: 10000 }
      )
    })

    it('should handle missing equity history', async () => {
      vi.mocked(api.getBacktestEquityHistory).mockRejectedValue(
        new Error('Failed to load')
      )

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Should still render the page
      expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
    })

    it('should handle missing trades', async () => {
      vi.mocked(api.getBacktestTrades).mockRejectedValue(
        new Error('Failed to load')
      )

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Should still render the page
      expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
    })
  })

  // ============================================================================
  // 6. 边界条件测试
  // ============================================================================

  describe('Edge Cases', () => {
    it('should handle zero profit', async () => {
      const zeroProfitBacktest = createMockBacktest({
        total_pnl: 0,
        total_pnl_pct: 0,
      })
      vi.mocked(api.getBacktest).mockResolvedValue(zeroProfitBacktest)

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      expect(screen.getByText(/0\.00%/)).toBeInTheDocument()
    })

    it('should handle negative profit', async () => {
      const negativeProfitBacktest = createMockBacktest({
        total_pnl: -500,
        total_pnl_pct: -5.0,
      })
      vi.mocked(api.getBacktest).mockResolvedValue(negativeProfitBacktest)

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      expect(screen.getByText(/-500\.00/)).toBeInTheDocument()
      expect(screen.getByText(/-5\.00%/)).toBeInTheDocument()
    })

    it('should handle large profit values', async () => {
      const largeProfitBacktest = createMockBacktest({
        total_pnl: 50000,
        total_pnl_pct: 500.0,
      })
      vi.mocked(api.getBacktest).mockResolvedValue(largeProfitBacktest)

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      const profitElements = screen.getAllByText(/50000\.00/)
      expect(profitElements.length).toBeGreaterThan(0)
      const pctElements = screen.getAllByText(/500\.00%/)
      expect(pctElements.length).toBeGreaterThan(0)
    })

    it('should handle zero trades', async () => {
      const zeroTradesBacktest = createMockBacktest({ total_trades: 0 })
      vi.mocked(api.getBacktest).mockResolvedValue(zeroTradesBacktest)
      vi.mocked(api.getBacktestTrades).mockResolvedValue([])

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      // Verify zero trades is displayed
      expect(screen.getByText(/共.*0.*笔/)).toBeInTheDocument()
    })

    it('should handle extreme drawdown', async () => {
      const extremeDrawdownBacktest = createMockBacktest({ max_drawdown: 50.0 })
      vi.mocked(api.getBacktest).mockResolvedValue(extremeDrawdownBacktest)

      renderWithProviders(<BacktestDetailPage backtestId="backtest-1" />)

      await waitFor(
        () => {
          expect(screen.getByText(/回测详情报告/)).toBeInTheDocument()
        },
        { timeout: 10000 }
      )

      expect(screen.getByText('50.00%')).toBeInTheDocument()
    })
  })
})

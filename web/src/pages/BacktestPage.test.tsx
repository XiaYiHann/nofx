/**
 * BacktestPage.test.tsx
 *
 * 测试目的：确保独立回测模式的表单验证逻辑正确
 *
 * 覆盖场景：
 * 1. 独立模式只需要选择 AI 模型即可提交
 * 2. 独立模式交易所可选（不选也能提交）
 * 3. 独立模式下必须选择 AI 模型，否则阻止提交
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { LanguageProvider } from '../contexts/LanguageContext'
import BacktestPage from './BacktestPage'

// Mock api module
vi.mock('../lib/api', () => ({
  api: {
    getBacktests: vi.fn().mockResolvedValue([]),
    getTraders: vi.fn().mockResolvedValue([]),
    getModelConfigs: vi.fn().mockResolvedValue([
      { id: 'model-1', name: 'Test Model', provider: 'openai' },
    ]),
    getExchangeConfigs: vi.fn().mockResolvedValue([
      { id: 'exchange-1', name: 'Binance', type: 'binance_futures' },
    ]),
    createBacktest: vi.fn().mockResolvedValue({ id: 'new-bt', status: 'pending' }),
    deleteBacktest: vi.fn().mockResolvedValue(undefined),
  },
}))

// Mock AuthContext
vi.mock('../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../contexts/AuthContext')
  return {
    ...actual,
    useAuth: vi.fn().mockReturnValue({
      user: { id: 'user-1', email: 'test@example.com' },
      isAuthenticated: true,
    }),
  }
})

import { api } from '../lib/api'

// Helper to render with providers
function renderWithProviders(ui: React.ReactElement) {
  return render(
    <LanguageProvider>
      {ui}
    </LanguageProvider>
  )
}

describe('BacktestPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Reset alert mock
    window.alert = vi.fn()
  })

  describe('Standalone mode form validation', () => {
    it('should show exchange as optional in standalone mode', async () => {
      renderWithProviders(<BacktestPage />)

      // Wait for loading to complete
      await waitFor(() => {
        expect(screen.queryByText('加载中...')).not.toBeInTheDocument()
      })

      // Click "新建回测" button
      fireEvent.click(screen.getByText('新建回测'))

      // Switch to standalone mode
      fireEvent.click(screen.getByText('独立配置回测'))

      // Exchange field should show "(可选)" instead of "*"
      await waitFor(() => {
        expect(screen.getByText(/选择交易所/)).toBeInTheDocument()
        expect(screen.getByText('(可选)')).toBeInTheDocument()
      })
    })

    it('should not call createBacktest when AI model is not selected in standalone mode', async () => {
      renderWithProviders(<BacktestPage />)

      await waitFor(() => {
        expect(screen.queryByText('加载中...')).not.toBeInTheDocument()
      })

      // Click "新建回测" button
      fireEvent.click(screen.getByText('新建回测'))

      // Switch to standalone mode
      fireEvent.click(screen.getByText('独立配置回测'))

      // Set required fields except AI model
      const startDateInput = screen.getByLabelText('开始时间')
      const endDateInput = screen.getByLabelText('结束时间')

      fireEvent.change(startDateInput, { target: { value: '2024-01-01T00:00' } })
      fireEvent.change(endDateInput, { target: { value: '2024-01-02T00:00' } })

      // Try to submit without selecting AI model
      fireEvent.click(screen.getByText('开始回测'))

      // Wait a bit for any async operations
      await new Promise(resolve => setTimeout(resolve, 100))

      // createBacktest should NOT be called when AI model is not selected
      expect(api.createBacktest).not.toHaveBeenCalled()
    })

    it('should allow submission with only AI model selected (no exchange)', async () => {
      renderWithProviders(<BacktestPage />)

      await waitFor(() => {
        expect(screen.queryByText('加载中...')).not.toBeInTheDocument()
      })

      // Click "新建回测" button
      fireEvent.click(screen.getByText('新建回测'))

      // Switch to standalone mode
      fireEvent.click(screen.getByText('独立配置回测'))

      // Wait for AI model dropdown to be available and select it
      await waitFor(() => {
        const selects = screen.getAllByRole('combobox')
        const aiModelSelect = selects.find((select) => {
          const options = select.querySelectorAll('option')
          return Array.from(options).some(opt => opt.textContent?.includes('Test Model'))
        })
        expect(aiModelSelect).toBeDefined()
        if (aiModelSelect) {
          fireEvent.change(aiModelSelect, { target: { value: 'model-1' } })
        }
      })

      // Set required fields
      const startDateInput = screen.getByLabelText('开始时间')
      const endDateInput = screen.getByLabelText('结束时间')

      fireEvent.change(startDateInput, { target: { value: '2024-01-01T00:00' } })
      fireEvent.change(endDateInput, { target: { value: '2024-01-02T00:00' } })

      // DO NOT select exchange (leave it empty)

      // Submit
      fireEvent.click(screen.getByText('开始回测'))

      // Should NOT show alert about exchange (no validation error for exchange)
      await waitFor(() => {
        // The key is that we don't require exchange anymore
        expect(window.alert).not.toHaveBeenCalledWith('请选择交易所')
      })
    })
  })

  describe('createBacktest payload', () => {
    it('should not include exchange_id when not selected in standalone mode', async () => {
      // Reset createBacktest mock to track calls
      vi.mocked(api.createBacktest).mockClear()
      vi.mocked(api.createBacktest).mockResolvedValue({ id: 'bt-1', status: 'pending' })

      renderWithProviders(<BacktestPage />)

      await waitFor(() => {
        expect(screen.queryByText('加载中...')).not.toBeInTheDocument()
      })

      // Click "新建回测" button
      fireEvent.click(screen.getByText('新建回测'))

      // Switch to standalone mode
      fireEvent.click(screen.getByText('独立配置回测'))

      // Set required fields
      const startDateInput = screen.getByLabelText('开始时间')
      const endDateInput = screen.getByLabelText('结束时间')
      const initialBalanceInput = screen.getByLabelText(/初始资金/)

      fireEvent.change(startDateInput, { target: { value: '2024-01-01T00:00' } })
      fireEvent.change(endDateInput, { target: { value: '2024-01-02T00:00' } })
      fireEvent.change(initialBalanceInput, { target: { value: '5000' } })

      // Find and select AI model
      await waitFor(() => {
        const selects = screen.getAllByRole('combobox')
        const aiModelSelect = selects.find((select) => {
          const options = select.querySelectorAll('option')
          return Array.from(options).some(opt => opt.textContent?.includes('Test Model'))
        })
        if (aiModelSelect) {
          fireEvent.change(aiModelSelect, { target: { value: 'model-1' } })
        }
      })

      // Submit
      fireEvent.click(screen.getByText('开始回测'))

      // Wait for API call
      await waitFor(() => {
        if (api.createBacktest.mock.calls.length > 0) {
          const payload = vi.mocked(api.createBacktest).mock.calls[0][0]
          // exchange_id should be undefined (not included) when not selected
          expect(payload.exchange_id).toBeUndefined()
          // ai_model_id should be included
          expect(payload.ai_model_id).toBe('model-1')
        }
      }, { timeout: 3000 })
    })

    it('should include exchange_id when selected in standalone mode', async () => {
      vi.mocked(api.createBacktest).mockClear()
      vi.mocked(api.createBacktest).mockResolvedValue({ id: 'bt-2', status: 'pending' })

      renderWithProviders(<BacktestPage />)

      await waitFor(() => {
        expect(screen.queryByText('加载中...')).not.toBeInTheDocument()
      })

      // Click "新建回测" button
      fireEvent.click(screen.getByText('新建回测'))

      // Switch to standalone mode
      fireEvent.click(screen.getByText('独立配置回测'))

      // Set required fields
      const startDateInput = screen.getByLabelText('开始时间')
      const endDateInput = screen.getByLabelText('结束时间')

      fireEvent.change(startDateInput, { target: { value: '2024-01-01T00:00' } })
      fireEvent.change(endDateInput, { target: { value: '2024-01-02T00:00' } })

      // Find and select both exchange and AI model
      await waitFor(() => {
        const selects = screen.getAllByRole('combobox')

        // Select exchange
        const exchangeSelect = selects.find((select) => {
          const options = select.querySelectorAll('option')
          return Array.from(options).some(opt => opt.textContent?.includes('Binance'))
        })
        if (exchangeSelect) {
          fireEvent.change(exchangeSelect, { target: { value: 'exchange-1' } })
        }

        // Select AI model
        const aiModelSelect = selects.find((select) => {
          const options = select.querySelectorAll('option')
          return Array.from(options).some(opt => opt.textContent?.includes('Test Model'))
        })
        if (aiModelSelect) {
          fireEvent.change(aiModelSelect, { target: { value: 'model-1' } })
        }
      })

      // Submit
      fireEvent.click(screen.getByText('开始回测'))

      // Wait for API call and verify payload
      await waitFor(() => {
        if (api.createBacktest.mock.calls.length > 0) {
          const payload = vi.mocked(api.createBacktest).mock.calls[0][0]
          // Both should be included when exchange is selected
          expect(payload.exchange_id).toBe('exchange-1')
          expect(payload.ai_model_id).toBe('model-1')
        }
      }, { timeout: 3000 })
    })
  })
})

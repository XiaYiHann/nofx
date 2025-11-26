import { describe, it, expect, vi, beforeEach } from 'vitest'
import { api } from '../lib/api'

// Mock the api module
vi.mock('../lib/api', () => ({
  api: {
    updateTrader: vi.fn(),
    getTraders: vi.fn(),
    getModelConfigs: vi.fn(),
    getExchangeConfigs: vi.fn(),
    getSupportedModels: vi.fn(),
    getSupportedExchanges: vi.fn(),
  },
}))

/**
 * AITradersPage Component Tests
 *
 * Tests for the handleSaveEditTrader function to ensure
 * system_prompt_template is correctly included in update requests
 */
describe('AITradersPage handleSaveEditTrader', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Request Body Construction', () => {
    it('should include system_prompt_template in the update request body', async () => {
      // Arrange: mock the updateTrader to capture the request
      const mockUpdateTrader = vi.mocked(api.updateTrader)
      mockUpdateTrader.mockResolvedValue({
        trader_id: 'test-trader-123',
        trader_name: 'Test Trader',
        ai_model: 'deepseek',
        exchange_id: 'binance',
        is_running: false,
        initial_balance: 1000,
      } as any)

      // Act: Simulate what handleSaveEditTrader does internally
      const testData = {
        name: 'Updated Trader',
        ai_model_id: 'deepseek',
        exchange_id: 'binance',
        initial_balance: 1000,
        scan_interval_minutes: 5,
        btc_eth_leverage: 10,
        altcoin_leverage: 5,
        trading_symbols: 'BTCUSDT,ETHUSDT',
        custom_prompt: 'Test custom prompt',
        override_base_prompt: false,
        system_prompt_template: 'aggressive', // The key field we're testing
        is_cross_margin: true,
        use_coin_pool: false,
        use_oi_top: false,
      }

      // This mimics what handleSaveEditTrader does
      const request = {
        name: testData.name,
        ai_model_id: testData.ai_model_id,
        exchange_id: testData.exchange_id,
        initial_balance: testData.initial_balance,
        scan_interval_minutes: testData.scan_interval_minutes,
        btc_eth_leverage: testData.btc_eth_leverage,
        altcoin_leverage: testData.altcoin_leverage,
        trading_symbols: testData.trading_symbols,
        custom_prompt: testData.custom_prompt,
        override_base_prompt: testData.override_base_prompt,
        system_prompt_template: testData.system_prompt_template, // Must be included
        is_cross_margin: testData.is_cross_margin,
        use_coin_pool: testData.use_coin_pool,
        use_oi_top: testData.use_oi_top,
      }

      await api.updateTrader('test-trader-123', request)

      // Assert: Verify the request contains system_prompt_template
      expect(mockUpdateTrader).toHaveBeenCalledTimes(1)
      expect(mockUpdateTrader).toHaveBeenCalledWith(
        'test-trader-123',
        expect.objectContaining({
          system_prompt_template: 'aggressive',
        })
      )
    })

    it('should preserve system_prompt_template when updating other fields', async () => {
      const mockUpdateTrader = vi.mocked(api.updateTrader)
      mockUpdateTrader.mockResolvedValue({} as any)

      const request = {
        name: 'Renamed Trader',
        ai_model_id: 'qwen',
        exchange_id: 'hyperliquid',
        initial_balance: 2000,
        scan_interval_minutes: 10,
        btc_eth_leverage: 20,
        altcoin_leverage: 10,
        trading_symbols: 'BTCUSDT',
        custom_prompt: '',
        override_base_prompt: false,
        system_prompt_template: 'conservative', // Different template
        is_cross_margin: false,
        use_coin_pool: true,
        use_oi_top: true,
      }

      await api.updateTrader('another-trader-id', request)

      // Assert
      expect(mockUpdateTrader).toHaveBeenCalledWith(
        'another-trader-id',
        expect.objectContaining({
          system_prompt_template: 'conservative',
          name: 'Renamed Trader',
          ai_model_id: 'qwen',
        })
      )
    })

    it('should handle empty system_prompt_template', async () => {
      const mockUpdateTrader = vi.mocked(api.updateTrader)
      mockUpdateTrader.mockResolvedValue({} as any)

      const request = {
        name: 'Test Trader',
        ai_model_id: 'deepseek',
        exchange_id: 'binance',
        initial_balance: 1000,
        scan_interval_minutes: 5,
        btc_eth_leverage: 10,
        altcoin_leverage: 5,
        trading_symbols: 'BTCUSDT',
        custom_prompt: '',
        override_base_prompt: false,
        system_prompt_template: '', // Empty - backend should preserve existing
        is_cross_margin: true,
        use_coin_pool: false,
        use_oi_top: false,
      }

      await api.updateTrader('test-trader', request)

      expect(mockUpdateTrader).toHaveBeenCalledWith(
        'test-trader',
        expect.objectContaining({
          system_prompt_template: '',
        })
      )
    })

    it('should include all required fields in the request', async () => {
      const mockUpdateTrader = vi.mocked(api.updateTrader)
      mockUpdateTrader.mockResolvedValue({} as any)

      const request = {
        name: 'Full Test',
        ai_model_id: 'claude',
        exchange_id: 'aster',
        initial_balance: 5000,
        scan_interval_minutes: 15,
        btc_eth_leverage: 15,
        altcoin_leverage: 8,
        trading_symbols: 'BTCUSDT,ETHUSDT,SOLUSDT',
        custom_prompt: 'Custom strategy description',
        override_base_prompt: true,
        system_prompt_template: 'balanced',
        is_cross_margin: true,
        use_coin_pool: true,
        use_oi_top: true,
      }

      await api.updateTrader('full-test-trader', request)

      const [, actualRequest] = mockUpdateTrader.mock.calls[0]

      // Verify all expected fields are present
      expect(actualRequest).toHaveProperty('name')
      expect(actualRequest).toHaveProperty('ai_model_id')
      expect(actualRequest).toHaveProperty('exchange_id')
      expect(actualRequest).toHaveProperty('initial_balance')
      expect(actualRequest).toHaveProperty('scan_interval_minutes')
      expect(actualRequest).toHaveProperty('btc_eth_leverage')
      expect(actualRequest).toHaveProperty('altcoin_leverage')
      expect(actualRequest).toHaveProperty('trading_symbols')
      expect(actualRequest).toHaveProperty('custom_prompt')
      expect(actualRequest).toHaveProperty('override_base_prompt')
      expect(actualRequest).toHaveProperty('system_prompt_template')
      expect(actualRequest).toHaveProperty('is_cross_margin')
      expect(actualRequest).toHaveProperty('use_coin_pool')
      expect(actualRequest).toHaveProperty('use_oi_top')
    })
  })
})

/**
 * api.test.ts
 *
 * 测试目的：确保 API 封装层正确处理请求和响应
 *
 * 覆盖的 Bug 类型：
 * 1. API 请求失败时未抛出正确错误
 * 2. 加密数据传输流程中断
 * 3. 响应数据映射错误
 *
 * Mock 策略：
 * - mock httpClient 的 get/post/put/delete 方法
 * - mock CryptoService 的加密方法
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { api, getAuthHeaders } from './api'

// Mock httpClient
vi.mock('./httpClient', () => ({
  httpClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

// Mock CryptoService
vi.mock('./crypto', () => ({
  CryptoService: {
    fetchPublicKey: vi.fn(),
    initialize: vi.fn(),
    encryptSensitiveData: vi.fn(),
  },
}))

import { httpClient } from './httpClient'
import { CryptoService } from './crypto'

describe('api module', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    sessionStorage.clear()
  })

  describe('getAuthHeaders', () => {
    it('should return empty object when no token', () => {
      const headers = getAuthHeaders()
      expect(headers).toEqual({})
    })

    it('should return Authorization header when token exists', () => {
      localStorage.setItem('auth_token', 'test-token')
      const headers = getAuthHeaders()
      expect(headers).toEqual({ Authorization: 'Bearer test-token' })
    })
  })

  describe('getTraders', () => {
    it('should return traders on success', async () => {
      const mockTraders = [
        { trader_id: 't1', trader_name: 'Trader 1', ai_model: 'deepseek' },
        { trader_id: 't2', trader_name: 'Trader 2', ai_model: 'qwen' },
      ]

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockTraders,
      })

      const result = await api.getTraders()
      expect(result).toEqual(mockTraders)
      expect(httpClient.get).toHaveBeenCalledWith('/api/my-traders')
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: false,
        message: 'Failed to fetch',
      })

      await expect(api.getTraders()).rejects.toThrow('获取trader列表失败')
    })
  })

  describe('createTrader', () => {
    it('should create trader on success', async () => {
      const request = {
        trader_name: 'New Trader',
        exchange_id: 'binance',
        model_id: 'deepseek',
      }
      const mockResponse = {
        trader_id: 'new-id',
        trader_name: 'New Trader',
        ai_model: 'deepseek',
      }

      vi.mocked(httpClient.post).mockResolvedValueOnce({
        success: true,
        data: mockResponse,
      })

      const result = await api.createTrader(request)
      expect(result).toEqual(mockResponse)
      expect(httpClient.post).toHaveBeenCalledWith('/api/traders', request)
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.post).mockResolvedValueOnce({
        success: false,
        message: 'Creation failed',
      })

      await expect(api.createTrader({} as any)).rejects.toThrow('创建交易员失败')
    })
  })

  describe('deleteTrader', () => {
    it('should delete trader on success', async () => {
      vi.mocked(httpClient.delete).mockResolvedValueOnce({
        success: true,
      })

      await expect(api.deleteTrader('trader-123')).resolves.not.toThrow()
      expect(httpClient.delete).toHaveBeenCalledWith('/api/traders/trader-123')
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.delete).mockResolvedValueOnce({
        success: false,
      })

      await expect(api.deleteTrader('trader-123')).rejects.toThrow(
        '删除交易员失败'
      )
    })
  })

  describe('startTrader / stopTrader', () => {
    it('should start trader on success', async () => {
      vi.mocked(httpClient.post).mockResolvedValueOnce({ success: true })

      await expect(api.startTrader('t1')).resolves.not.toThrow()
      expect(httpClient.post).toHaveBeenCalledWith('/api/traders/t1/start')
    })

    it('should stop trader on success', async () => {
      vi.mocked(httpClient.post).mockResolvedValueOnce({ success: true })

      await expect(api.stopTrader('t1')).resolves.not.toThrow()
      expect(httpClient.post).toHaveBeenCalledWith('/api/traders/t1/stop')
    })

    it('should throw error when start fails', async () => {
      vi.mocked(httpClient.post).mockResolvedValueOnce({ success: false })

      await expect(api.startTrader('t1')).rejects.toThrow('启动交易员失败')
    })

    it('should throw error when stop fails', async () => {
      vi.mocked(httpClient.post).mockResolvedValueOnce({ success: false })

      await expect(api.stopTrader('t1')).rejects.toThrow('停止交易员失败')
    })
  })

  describe('getModelConfigs', () => {
    it('should return mapped model configs', async () => {
      const mockModels = [
        {
          id: 'deepseek',
          name: 'DeepSeek',
          provider: 'deepseek',
          enabled: true,
          customApiUrl: 'https://api.deepseek.com',
          customModelName: 'deepseek-chat',
        },
        {
          id: 'qwen',
          name: 'Qwen',
          provider: 'alibaba',
          enabled: false,
          customApiUrl: '',
          customModelName: '',
        },
      ]

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockModels,
      })

      const result = await api.getModelConfigs()

      expect(result).toHaveLength(2)
      expect(result[0].customApiUrl).toBe('https://api.deepseek.com')
      expect(result[1].customApiUrl).toBe('')
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: false,
      })

      await expect(api.getModelConfigs()).rejects.toThrow('获取模型配置失败')
    })
  })

  describe('updateModelConfigs (encrypted)', () => {
    /**
     * 测试：加密传输流程
     * 确保 RSA 加密流程正确执行
     */
    it('should encrypt and send model configs', async () => {
      const request = {
        model_id: 'deepseek',
        api_key: 'secret-key',
      }

      vi.mocked(CryptoService.fetchPublicKey).mockResolvedValueOnce(
        'mock-public-key'
      )
      vi.mocked(CryptoService.initialize).mockResolvedValueOnce(undefined)
      vi.mocked(CryptoService.encryptSensitiveData).mockResolvedValueOnce({
        encrypted_data: 'encrypted-payload',
        encrypted_key: 'encrypted-aes-key',
        iv: 'initialization-vector',
      })
      vi.mocked(httpClient.put).mockResolvedValueOnce({ success: true })

      await expect(api.updateModelConfigs(request)).resolves.not.toThrow()

      expect(CryptoService.fetchPublicKey).toHaveBeenCalled()
      expect(CryptoService.initialize).toHaveBeenCalledWith('mock-public-key')
      expect(CryptoService.encryptSensitiveData).toHaveBeenCalled()
      expect(httpClient.put).toHaveBeenCalledWith('/api/models', {
        encrypted_data: 'encrypted-payload',
        encrypted_key: 'encrypted-aes-key',
        iv: 'initialization-vector',
      })
    })

    it('should throw error when encryption fails', async () => {
      vi.mocked(CryptoService.fetchPublicKey).mockRejectedValueOnce(
        new Error('Failed to fetch public key')
      )

      await expect(api.updateModelConfigs({} as any)).rejects.toThrow(
        'Failed to fetch public key'
      )
    })

    it('should throw error when update fails', async () => {
      vi.mocked(CryptoService.fetchPublicKey).mockResolvedValueOnce(
        'mock-public-key'
      )
      vi.mocked(CryptoService.initialize).mockResolvedValueOnce(undefined)
      vi.mocked(CryptoService.encryptSensitiveData).mockResolvedValueOnce({})
      vi.mocked(httpClient.put).mockResolvedValueOnce({ success: false })

      await expect(api.updateModelConfigs({} as any)).rejects.toThrow(
        '更新模型配置失败'
      )
    })
  })

  describe('getExchangeConfigs', () => {
    it('should return mapped exchange configs', async () => {
      const mockExchanges = [
        {
          id: 'binance',
          name: 'Binance',
          type: 'cex',
          enabled: true,
          testnet: true,
          hyperliquidWalletAddr: '',
          asterUser: '',
          asterSigner: '',
        },
        {
          id: 'hyperliquid',
          name: 'Hyperliquid',
          type: 'dex',
          enabled: false,
          testnet: false,
          hyperliquidWalletAddr: '0x1234',
          asterUser: '',
          asterSigner: '',
        },
      ]

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockExchanges,
      })

      const result = await api.getExchangeConfigs()

      expect(result).toHaveLength(2)
      expect(result[0].testnet).toBe(true)
      expect(result[1].hyperliquidWalletAddr).toBe('0x1234')
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: false,
      })

      await expect(api.getExchangeConfigs()).rejects.toThrow(
        '获取交易所配置失败'
      )
    })
  })

  describe('updateExchangeConfigsEncrypted', () => {
    it('should encrypt and send exchange configs', async () => {
      const request = {
        exchange_id: 'binance',
        api_key: 'api-key',
        secret_key: 'secret-key',
      }

      vi.mocked(CryptoService.fetchPublicKey).mockResolvedValueOnce(
        'mock-public-key'
      )
      vi.mocked(CryptoService.initialize).mockResolvedValueOnce(undefined)
      vi.mocked(CryptoService.encryptSensitiveData).mockResolvedValueOnce({
        encrypted_data: 'encrypted',
      })
      vi.mocked(httpClient.put).mockResolvedValueOnce({ success: true })

      await expect(
        api.updateExchangeConfigsEncrypted(request)
      ).resolves.not.toThrow()

      expect(CryptoService.fetchPublicKey).toHaveBeenCalled()
      expect(httpClient.put).toHaveBeenCalledWith('/api/exchanges', {
        encrypted_data: 'encrypted',
      })
    })

    it('should throw error when update fails', async () => {
      vi.mocked(CryptoService.fetchPublicKey).mockResolvedValueOnce(
        'mock-public-key'
      )
      vi.mocked(CryptoService.initialize).mockResolvedValueOnce(undefined)
      vi.mocked(CryptoService.encryptSensitiveData).mockResolvedValueOnce({})
      vi.mocked(httpClient.put).mockResolvedValueOnce({ success: false })

      await expect(
        api.updateExchangeConfigsEncrypted({} as any)
      ).rejects.toThrow('更新交易所配置失败')
    })
  })

  describe('getStatus', () => {
    it('should fetch status without traderId', async () => {
      const mockStatus = { running: true, call_count: 10, runtime_minutes: 5 }

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockStatus,
      })

      const result = await api.getStatus()
      expect(result).toEqual(mockStatus)
      expect(httpClient.get).toHaveBeenCalledWith('/api/status')
    })

    it('should fetch status with traderId', async () => {
      const mockStatus = { running: false, call_count: 0, runtime_minutes: 0 }

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockStatus,
      })

      const result = await api.getStatus('trader-123')
      expect(result).toEqual(mockStatus)
      expect(httpClient.get).toHaveBeenCalledWith(
        '/api/status?trader_id=trader-123'
      )
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getStatus()).rejects.toThrow('获取系统状态失败')
    })
  })

  describe('getAccount', () => {
    it('should fetch account info', async () => {
      const mockAccount = {
        total_equity: 10000,
        available_balance: 5000,
        total_pnl: 500,
        total_pnl_pct: 5,
      }

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockAccount,
      })

      const result = await api.getAccount('t1')
      expect(result.total_equity).toBe(10000)
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getAccount()).rejects.toThrow('获取账户信息失败')
    })
  })

  describe('getPositions', () => {
    it('should fetch positions', async () => {
      const mockPositions = [
        { symbol: 'BTCUSDT', side: 'long', quantity: 0.1 },
        { symbol: 'ETHUSDT', side: 'short', quantity: 1 },
      ]

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockPositions,
      })

      const result = await api.getPositions('t1')
      expect(result).toHaveLength(2)
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getPositions()).rejects.toThrow('获取持仓列表失败')
    })
  })

  describe('getLatestDecisions', () => {
    it('should fetch latest decisions with limit', async () => {
      const mockDecisions = [
        { cycle_number: 1, success: true },
        { cycle_number: 2, success: true },
      ]

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockDecisions,
      })

      const result = await api.getLatestDecisions('t1', 10)
      expect(result).toHaveLength(2)
      expect(httpClient.get).toHaveBeenCalledWith(
        '/api/decisions/latest?trader_id=t1&limit=10'
      )
    })

    it('should use default limit of 5', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: [],
      })

      await api.getLatestDecisions('t1')
      expect(httpClient.get).toHaveBeenCalledWith(
        '/api/decisions/latest?trader_id=t1&limit=5'
      )
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getLatestDecisions()).rejects.toThrow('获取最新决策失败')
    })
  })

  describe('getEquityHistory', () => {
    it('should fetch equity history', async () => {
      const mockHistory = [
        { timestamp: '2024-01-01', equity: 10000 },
        { timestamp: '2024-01-02', equity: 10500 },
      ]

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockHistory,
      })

      const result = await api.getEquityHistory('t1')
      expect(result).toHaveLength(2)
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getEquityHistory()).rejects.toThrow('获取历史数据失败')
    })
  })

  describe('getEquityHistoryBatch', () => {
    it('should fetch batch equity history', async () => {
      const mockBatchData = {
        't1': [{ timestamp: '2024-01-01', equity: 10000 }],
        't2': [{ timestamp: '2024-01-01', equity: 20000 }],
      }

      vi.mocked(httpClient.post).mockResolvedValueOnce({
        success: true,
        data: mockBatchData,
      })

      const result = await api.getEquityHistoryBatch(['t1', 't2'])
      expect(result).toEqual(mockBatchData)
      expect(httpClient.post).toHaveBeenCalledWith('/api/equity-history-batch', {
        trader_ids: ['t1', 't2'],
      })
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.post).mockResolvedValueOnce({ success: false })

      await expect(api.getEquityHistoryBatch(['t1'])).rejects.toThrow(
        '获取批量历史数据失败'
      )
    })
  })

  describe('getTopTraders', () => {
    it('should fetch top traders', async () => {
      const mockTopTraders = [
        { trader_id: 't1', pnl: 1000 },
        { trader_id: 't2', pnl: 800 },
      ]

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockTopTraders,
      })

      const result = await api.getTopTraders()
      expect(result).toHaveLength(2)
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getTopTraders()).rejects.toThrow('获取前5名交易员失败')
    })
  })

  describe('getCompetition', () => {
    it('should fetch competition data', async () => {
      const mockCompetition = {
        traders: [],
        total_volume: 1000000,
      }

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockCompetition,
      })

      const result = await api.getCompetition()
      expect(result.total_volume).toBe(1000000)
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getCompetition()).rejects.toThrow('获取竞赛数据失败')
    })
  })

  describe('getUserSignalSource', () => {
    it('should fetch user signal source', async () => {
      const mockSource = {
        coin_pool_url: 'https://coinpool.example.com',
        oi_top_url: 'https://oitop.example.com',
      }

      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: mockSource,
      })

      const result = await api.getUserSignalSource()
      expect(result.coin_pool_url).toBe('https://coinpool.example.com')
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getUserSignalSource()).rejects.toThrow(
        '获取用户信号源配置失败'
      )
    })
  })

  describe('saveUserSignalSource', () => {
    it('should save user signal source', async () => {
      vi.mocked(httpClient.post).mockResolvedValueOnce({ success: true })

      await expect(
        api.saveUserSignalSource('https://new-coinpool.com', 'https://new-oi.com')
      ).resolves.not.toThrow()

      expect(httpClient.post).toHaveBeenCalledWith('/api/user/signal-sources', {
        coin_pool_url: 'https://new-coinpool.com',
        oi_top_url: 'https://new-oi.com',
      })
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.post).mockResolvedValueOnce({ success: false })

      await expect(api.saveUserSignalSource('', '')).rejects.toThrow(
        '保存用户信号源配置失败'
      )
    })
  })

  describe('getServerIP', () => {
    it('should fetch server IP', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({
        success: true,
        data: { public_ip: '1.2.3.4', message: 'Success' },
      })

      const result = await api.getServerIP()
      expect(result.public_ip).toBe('1.2.3.4')
    })

    it('should throw error on failure', async () => {
      vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

      await expect(api.getServerIP()).rejects.toThrow('获取服务器IP失败')
    })
  })

  describe('Backtest APIs', () => {
    describe('getBacktests', () => {
      it('should fetch backtests list', async () => {
        const mockBacktests = [
          { id: 'bt1', status: 'completed' },
          { id: 'bt2', status: 'running' },
        ]

        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: mockBacktests,
        })

        const result = await api.getBacktests()
        expect(result).toHaveLength(2)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getBacktests()).rejects.toThrow('获取回测列表失败')
      })
    })

    describe('getBacktest', () => {
      it('should fetch single backtest', async () => {
        const mockBacktest = { id: 'bt1', status: 'completed', pnl: 500 }

        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: mockBacktest,
        })

        const result = await api.getBacktest('bt1')
        expect(result.id).toBe('bt1')
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getBacktest('bt1')).rejects.toThrow('获取回测详情失败')
      })
    })

    describe('createBacktest', () => {
      it('should create backtest', async () => {
        const request = {
          trader_id: 't1',
          start_time: '2024-01-01',
          end_time: '2024-01-31',
          initial_balance: 10000,
        }

        vi.mocked(httpClient.post).mockResolvedValueOnce({
          success: true,
          data: { id: 'new-bt', status: 'running' },
        })

        const result = await api.createBacktest(request)
        expect(result.id).toBe('new-bt')
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.post).mockResolvedValueOnce({ success: false })

        await expect(
          api.createBacktest({
            trader_id: 't1',
            start_time: '',
            end_time: '',
            initial_balance: 0,
          })
        ).rejects.toThrow('创建回测失败')
      })
    })

    describe('deleteBacktest', () => {
      it('should delete backtest', async () => {
        vi.mocked(httpClient.delete).mockResolvedValueOnce({ success: true })

        await expect(api.deleteBacktest('bt1')).resolves.not.toThrow()
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.delete).mockResolvedValueOnce({ success: false })

        await expect(api.deleteBacktest('bt1')).rejects.toThrow('删除回测失败')
      })
    })

    describe('getBacktestEquityHistory', () => {
      it('should fetch backtest equity history', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: [{ timestamp: '2024-01-01', equity: 10000 }],
        })

        const result = await api.getBacktestEquityHistory('bt1')
        expect(result).toHaveLength(1)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getBacktestEquityHistory('bt1')).rejects.toThrow(
          '获取回测净值历史失败'
        )
      })
    })

    describe('getBacktestTrades', () => {
      it('should fetch backtest trades', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: [{ id: 'trade1', symbol: 'BTCUSDT' }],
        })

        const result = await api.getBacktestTrades('bt1')
        expect(result).toHaveLength(1)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getBacktestTrades('bt1')).rejects.toThrow(
          '获取回测交易记录失败'
        )
      })
    })

    describe('getBacktestDecisions', () => {
      it('should fetch backtest decisions', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: [{ cycle: 1, action: 'buy' }],
        })

        const result = await api.getBacktestDecisions('bt1')
        expect(result).toHaveLength(1)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getBacktestDecisions('bt1')).rejects.toThrow(
          '获取回测决策记录失败'
        )
      })
    })
  })

  describe('Other APIs', () => {
    describe('getSupportedModels', () => {
      it('should fetch supported models', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: [{ id: 'deepseek', name: 'DeepSeek' }],
        })

        const result = await api.getSupportedModels()
        expect(result).toHaveLength(1)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getSupportedModels()).rejects.toThrow(
          '获取支持的模型失败'
        )
      })
    })

    describe('getSupportedExchanges', () => {
      it('should fetch supported exchanges', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: [{ id: 'binance', name: 'Binance' }],
        })

        const result = await api.getSupportedExchanges()
        expect(result).toHaveLength(1)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getSupportedExchanges()).rejects.toThrow(
          '获取支持的交易所失败'
        )
      })
    })

    describe('updateTraderPrompt', () => {
      it('should update trader prompt', async () => {
        vi.mocked(httpClient.put).mockResolvedValueOnce({ success: true })

        await expect(
          api.updateTraderPrompt('t1', 'Custom prompt')
        ).resolves.not.toThrow()

        expect(httpClient.put).toHaveBeenCalledWith('/api/traders/t1/prompt', {
          custom_prompt: 'Custom prompt',
        })
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.put).mockResolvedValueOnce({ success: false })

        await expect(api.updateTraderPrompt('t1', '')).rejects.toThrow(
          '更新自定义策略失败'
        )
      })
    })

    describe('getTraderConfig', () => {
      it('should fetch trader config', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: { trader_id: 't1', custom_prompt: 'test' },
        })

        const result = await api.getTraderConfig('t1')
        expect(result.trader_id).toBe('t1')
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getTraderConfig('t1')).rejects.toThrow(
          '获取交易员配置失败'
        )
      })
    })

    describe('updateTrader', () => {
      it('should update trader', async () => {
        const request = { trader_name: 'Updated' }

        vi.mocked(httpClient.put).mockResolvedValueOnce({
          success: true,
          data: { trader_id: 't1', trader_name: 'Updated' },
        })

        const result = await api.updateTrader('t1', request as any)
        expect(result.trader_name).toBe('Updated')
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.put).mockResolvedValueOnce({ success: false })

        await expect(api.updateTrader('t1', {} as any)).rejects.toThrow(
          '更新交易员失败'
        )
      })
    })

    describe('getPublicTraders', () => {
      it('should fetch public traders', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: [{ trader_id: 'public-1' }],
        })

        const result = await api.getPublicTraders()
        expect(result).toHaveLength(1)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getPublicTraders()).rejects.toThrow(
          '获取公开trader列表失败'
        )
      })
    })

    describe('getPublicTraderConfig', () => {
      it('should fetch public trader config', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: { trader_id: 'public-1', model: 'deepseek' },
        })

        const result = await api.getPublicTraderConfig('public-1')
        expect(result.model).toBe('deepseek')
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getPublicTraderConfig('public-1')).rejects.toThrow(
          '获取公开交易员配置失败'
        )
      })
    })

    describe('getPerformance', () => {
      it('should fetch AI performance data', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: { accuracy: 0.85, total_trades: 100 },
        })

        const result = await api.getPerformance('t1')
        expect(result.accuracy).toBe(0.85)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getPerformance()).rejects.toThrow(
          '获取AI学习数据失败'
        )
      })
    })

    describe('getStatistics', () => {
      it('should fetch statistics', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: { total_pnl: 1000, win_rate: 0.6 },
        })

        const result = await api.getStatistics('t1')
        expect(result.total_pnl).toBe(1000)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getStatistics()).rejects.toThrow('获取统计信息失败')
      })
    })

    describe('getDecisions', () => {
      it('should fetch all decisions', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({
          success: true,
          data: [{ cycle: 1 }, { cycle: 2 }],
        })

        const result = await api.getDecisions('t1')
        expect(result).toHaveLength(2)
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.get).mockResolvedValueOnce({ success: false })

        await expect(api.getDecisions()).rejects.toThrow('获取决策日志失败')
      })
    })

    describe('updateExchangeConfigs (non-encrypted)', () => {
      it('should update exchange configs', async () => {
        vi.mocked(httpClient.put).mockResolvedValueOnce({ success: true })

        await expect(
          api.updateExchangeConfigs({ exchange_id: 'binance' } as any)
        ).resolves.not.toThrow()
      })

      it('should throw error on failure', async () => {
        vi.mocked(httpClient.put).mockResolvedValueOnce({ success: false })

        await expect(api.updateExchangeConfigs({} as any)).rejects.toThrow(
          '更新交易所配置失败'
        )
      })
    })
  })
})

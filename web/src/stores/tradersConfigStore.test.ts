/**
 * tradersConfigStore.test.ts
 *
 * 测试目的：确保 Zustand store 的状态管理逻辑正确
 *
 * 覆盖的 Bug 类型：
 * 1. configuredModels/configuredExchanges 过滤逻辑错误
 * 2. loadConfigs 在不同登录状态下的分支处理
 * 3. aster/hyperliquid 特殊交易所的判断逻辑
 *
 * 边界条件：
 * - customApiUrl 为空/有值
 * - apiKey 为空/有值
 * - 特殊交易所 (aster, hyperliquid) 的判断字段
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useTradersConfigStore } from './tradersConfigStore'
import { api } from '../lib/api'

// Mock api module
vi.mock('../lib/api', () => ({
  api: {
    getModelConfigs: vi.fn(),
    getExchangeConfigs: vi.fn(),
    getSupportedModels: vi.fn(),
    getSupportedExchanges: vi.fn(),
    getUserSignalSource: vi.fn(),
  },
}))

describe('useTradersConfigStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Reset store state before each test
    const { result } = renderHook(() => useTradersConfigStore())
    act(() => {
      result.current.reset()
    })
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('初始状态', () => {
    it('should have correct initial state', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      expect(result.current.allModels).toEqual([])
      expect(result.current.allExchanges).toEqual([])
      expect(result.current.supportedModels).toEqual([])
      expect(result.current.supportedExchanges).toEqual([])
      expect(result.current.configuredModels).toEqual([])
      expect(result.current.configuredExchanges).toEqual([])
      expect(result.current.userSignalSource).toEqual({
        coinPoolUrl: '',
        oiTopUrl: '',
      })
    })
  })

  describe('setAllModels - 计算 configuredModels', () => {
    /**
     * 测试：enabled=true 的模型应包含在 configuredModels
     */
    it('should include enabled models in configuredModels', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const models = [
        { id: 'deepseek', name: 'DeepSeek', enabled: true, customApiUrl: '' },
        { id: 'qwen', name: 'Qwen', enabled: false, customApiUrl: '' },
      ]

      act(() => {
        result.current.setAllModels(models as any)
      })

      expect(result.current.allModels).toHaveLength(2)
      expect(result.current.configuredModels).toHaveLength(1)
      expect(result.current.configuredModels[0].id).toBe('deepseek')
    })

    /**
     * 测试：customApiUrl 非空的模型也应包含在 configuredModels (即使 enabled=false)
     */
    it('should include models with non-empty customApiUrl in configuredModels', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const models = [
        {
          id: 'custom1',
          name: 'Custom Model',
          enabled: false,
          customApiUrl: 'https://custom.api.com',
        },
        {
          id: 'disabled',
          name: 'Disabled',
          enabled: false,
          customApiUrl: '',
        },
      ]

      act(() => {
        result.current.setAllModels(models as any)
      })

      expect(result.current.configuredModels).toHaveLength(1)
      expect(result.current.configuredModels[0].id).toBe('custom1')
    })

    /**
     * 测试：customApiUrl 为空白字符串时不应计入
     */
    it('should NOT include models with whitespace-only customApiUrl', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const models = [
        {
          id: 'whitespace',
          name: 'Whitespace',
          enabled: false,
          customApiUrl: '   ',
        },
      ]

      act(() => {
        result.current.setAllModels(models as any)
      })

      expect(result.current.configuredModels).toHaveLength(0)
    })

    /**
     * 测试：既有 enabled 又有 customApiUrl 的情况
     */
    it('should handle both enabled and customApiUrl conditions', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const models = [
        {
          id: 'both',
          name: 'Both',
          enabled: true,
          customApiUrl: 'https://api.com',
        },
        { id: 'enabled-only', name: 'Enabled', enabled: true, customApiUrl: '' },
        {
          id: 'url-only',
          name: 'URL Only',
          enabled: false,
          customApiUrl: 'https://other.com',
        },
        { id: 'neither', name: 'Neither', enabled: false, customApiUrl: '' },
      ]

      act(() => {
        result.current.setAllModels(models as any)
      })

      expect(result.current.configuredModels).toHaveLength(3)
      const ids = result.current.configuredModels.map((m) => m.id)
      expect(ids).toContain('both')
      expect(ids).toContain('enabled-only')
      expect(ids).toContain('url-only')
      expect(ids).not.toContain('neither')
    })
  })

  describe('setAllExchanges - 计算 configuredExchanges', () => {
    /**
     * 测试：aster 交易所通过 asterUser 判断
     */
    it('should configure aster exchange when asterUser is set', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const exchanges = [
        {
          id: 'aster',
          name: 'Aster',
          enabled: false,
          asterUser: 'user123',
          asterSigner: 'signer',
          hyperliquidWalletAddr: '',
          apiKey: '',
        },
      ]

      act(() => {
        result.current.setAllExchanges(exchanges as any)
      })

      expect(result.current.configuredExchanges).toHaveLength(1)
      expect(result.current.configuredExchanges[0].id).toBe('aster')
    })

    /**
     * 测试：aster 交易所 asterUser 为空时不计入
     */
    it('should NOT configure aster exchange when asterUser is empty', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const exchanges = [
        {
          id: 'aster',
          name: 'Aster',
          enabled: true, // enabled 不影响 aster
          asterUser: '',
          asterSigner: 'signer',
          hyperliquidWalletAddr: '',
          apiKey: '',
        },
      ]

      act(() => {
        result.current.setAllExchanges(exchanges as any)
      })

      expect(result.current.configuredExchanges).toHaveLength(0)
    })

    /**
     * 测试：hyperliquid 交易所通过 hyperliquidWalletAddr 判断
     */
    it('should configure hyperliquid exchange when hyperliquidWalletAddr is set', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const exchanges = [
        {
          id: 'hyperliquid',
          name: 'Hyperliquid',
          enabled: false,
          asterUser: '',
          asterSigner: '',
          hyperliquidWalletAddr: '0x1234567890abcdef',
          apiKey: '',
        },
      ]

      act(() => {
        result.current.setAllExchanges(exchanges as any)
      })

      expect(result.current.configuredExchanges).toHaveLength(1)
      expect(result.current.configuredExchanges[0].id).toBe('hyperliquid')
    })

    /**
     * 测试：hyperliquid 交易所钱包地址为空时不计入
     */
    it('should NOT configure hyperliquid exchange when hyperliquidWalletAddr is empty', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const exchanges = [
        {
          id: 'hyperliquid',
          name: 'Hyperliquid',
          enabled: true,
          asterUser: '',
          asterSigner: '',
          hyperliquidWalletAddr: '   ', // whitespace
          apiKey: '',
        },
      ]

      act(() => {
        result.current.setAllExchanges(exchanges as any)
      })

      expect(result.current.configuredExchanges).toHaveLength(0)
    })

    /**
     * 测试：普通交易所通过 enabled 或 apiKey 判断
     */
    it('should configure regular exchange when enabled is true', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const exchanges = [
        {
          id: 'binance',
          name: 'Binance',
          enabled: true,
          asterUser: '',
          asterSigner: '',
          hyperliquidWalletAddr: '',
          apiKey: '',
        },
      ]

      act(() => {
        result.current.setAllExchanges(exchanges as any)
      })

      expect(result.current.configuredExchanges).toHaveLength(1)
      expect(result.current.configuredExchanges[0].id).toBe('binance')
    })

    /**
     * 测试：普通交易所通过 apiKey 判断 (enabled=false)
     */
    it('should configure regular exchange when apiKey is set', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const exchanges = [
        {
          id: 'okx',
          name: 'OKX',
          enabled: false,
          asterUser: '',
          asterSigner: '',
          hyperliquidWalletAddr: '',
          apiKey: 'some-api-key',
        },
      ]

      act(() => {
        result.current.setAllExchanges(exchanges as any)
      })

      expect(result.current.configuredExchanges).toHaveLength(1)
      expect(result.current.configuredExchanges[0].id).toBe('okx')
    })

    /**
     * 测试：混合交易所配置
     */
    it('should handle mixed exchange configurations', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const exchanges = [
        {
          id: 'aster',
          name: 'Aster',
          enabled: false,
          asterUser: 'user1',
          asterSigner: '',
          hyperliquidWalletAddr: '',
          apiKey: '',
        },
        {
          id: 'hyperliquid',
          name: 'Hyperliquid',
          enabled: false,
          asterUser: '',
          asterSigner: '',
          hyperliquidWalletAddr: '0xabc',
          apiKey: '',
        },
        {
          id: 'binance',
          name: 'Binance',
          enabled: true,
          asterUser: '',
          asterSigner: '',
          hyperliquidWalletAddr: '',
          apiKey: '',
        },
        {
          id: 'disabled',
          name: 'Disabled',
          enabled: false,
          asterUser: '',
          asterSigner: '',
          hyperliquidWalletAddr: '',
          apiKey: '',
        },
      ]

      act(() => {
        result.current.setAllExchanges(exchanges as any)
      })

      expect(result.current.configuredExchanges).toHaveLength(3)
      const ids = result.current.configuredExchanges.map((e) => e.id)
      expect(ids).toContain('aster')
      expect(ids).toContain('hyperliquid')
      expect(ids).toContain('binance')
      expect(ids).not.toContain('disabled')
    })
  })

  describe('setSupportedModels / setSupportedExchanges', () => {
    it('should set supported models', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const models = [{ id: 'model1' }, { id: 'model2' }]

      act(() => {
        result.current.setSupportedModels(models as any)
      })

      expect(result.current.supportedModels).toHaveLength(2)
    })

    it('should set supported exchanges', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      const exchanges = [{ id: 'ex1' }, { id: 'ex2' }]

      act(() => {
        result.current.setSupportedExchanges(exchanges as any)
      })

      expect(result.current.supportedExchanges).toHaveLength(2)
    })
  })

  describe('setUserSignalSource', () => {
    it('should set user signal source', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      act(() => {
        result.current.setUserSignalSource({
          coinPoolUrl: 'https://pool.example.com',
          oiTopUrl: 'https://oi.example.com',
        })
      })

      expect(result.current.userSignalSource.coinPoolUrl).toBe(
        'https://pool.example.com'
      )
      expect(result.current.userSignalSource.oiTopUrl).toBe(
        'https://oi.example.com'
      )
    })
  })

  describe('loadConfigs - 未登录状态', () => {
    /**
     * 测试：未登录时只加载 supported models/exchanges
     */
    it('should only load supported configs when user is null', async () => {
      vi.mocked(api.getSupportedModels).mockResolvedValueOnce([
        { id: 'supported1' },
      ] as any)
      vi.mocked(api.getSupportedExchanges).mockResolvedValueOnce([
        { id: 'supported-ex1' },
      ] as any)

      const { result } = renderHook(() => useTradersConfigStore())

      await act(async () => {
        await result.current.loadConfigs(null, null)
      })

      expect(api.getSupportedModels).toHaveBeenCalled()
      expect(api.getSupportedExchanges).toHaveBeenCalled()
      expect(api.getModelConfigs).not.toHaveBeenCalled()
      expect(api.getExchangeConfigs).not.toHaveBeenCalled()

      expect(result.current.supportedModels).toHaveLength(1)
      expect(result.current.supportedExchanges).toHaveLength(1)
    })

    /**
     * 测试：未登录时部分 API 失败不影响其他
     */
    it('should handle partial API failures when not logged in', async () => {
      vi.mocked(api.getSupportedModels).mockRejectedValueOnce(
        new Error('Failed')
      )
      vi.mocked(api.getSupportedExchanges).mockResolvedValueOnce([
        { id: 'ex1' },
      ] as any)

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { result } = renderHook(() => useTradersConfigStore())

      await act(async () => {
        await result.current.loadConfigs(null, null)
      })

      // supportedExchanges should still be loaded
      expect(result.current.supportedExchanges).toHaveLength(1)
      expect(consoleSpy).toHaveBeenCalled()

      consoleSpy.mockRestore()
    })
  })

  describe('loadConfigs - 已登录状态', () => {
    /**
     * 测试：已登录时加载完整配置
     */
    it('should load all configs when user is logged in', async () => {
      const mockModels = [
        { id: 'model1', enabled: true, customApiUrl: '' },
      ]
      const mockExchanges = [
        { id: 'ex1', enabled: true, asterUser: '', hyperliquidWalletAddr: '' },
      ]
      const mockSupportedModels = [{ id: 'supported1' }]
      const mockSupportedExchanges = [{ id: 'supported-ex1' }]

      vi.mocked(api.getModelConfigs).mockResolvedValueOnce(mockModels as any)
      vi.mocked(api.getExchangeConfigs).mockResolvedValueOnce(
        mockExchanges as any
      )
      vi.mocked(api.getSupportedModels).mockResolvedValueOnce(
        mockSupportedModels as any
      )
      vi.mocked(api.getSupportedExchanges).mockResolvedValueOnce(
        mockSupportedExchanges as any
      )
      vi.mocked(api.getUserSignalSource).mockResolvedValueOnce({
        coin_pool_url: 'https://pool.com',
        oi_top_url: 'https://oi.com',
      })

      const { result } = renderHook(() => useTradersConfigStore())

      await act(async () => {
        await result.current.loadConfigs({ id: 'user1' }, 'token')
      })

      expect(api.getModelConfigs).toHaveBeenCalled()
      expect(api.getExchangeConfigs).toHaveBeenCalled()
      expect(api.getSupportedModels).toHaveBeenCalled()
      expect(api.getSupportedExchanges).toHaveBeenCalled()
      expect(api.getUserSignalSource).toHaveBeenCalled()

      expect(result.current.allModels).toHaveLength(1)
      expect(result.current.allExchanges).toHaveLength(1)
      expect(result.current.configuredModels).toHaveLength(1)
      expect(result.current.configuredExchanges).toHaveLength(1)
      expect(result.current.userSignalSource.coinPoolUrl).toBe(
        'https://pool.com'
      )
    })

    /**
     * 测试：已登录时部分 API 失败不影响其他
     */
    it('should handle partial API failures when logged in', async () => {
      vi.mocked(api.getModelConfigs).mockRejectedValueOnce(new Error('Failed'))
      vi.mocked(api.getExchangeConfigs).mockResolvedValueOnce([
        { id: 'ex1', enabled: true },
      ] as any)
      vi.mocked(api.getSupportedModels).mockResolvedValueOnce([
        { id: 's1' },
      ] as any)
      vi.mocked(api.getSupportedExchanges).mockResolvedValueOnce([
        { id: 'se1' },
      ] as any)
      vi.mocked(api.getUserSignalSource).mockRejectedValueOnce(
        new Error('Not configured')
      )

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const consoleLogSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

      const { result } = renderHook(() => useTradersConfigStore())

      await act(async () => {
        await result.current.loadConfigs({ id: 'user1' }, 'token')
      })

      // Other configs should still be loaded
      expect(result.current.allExchanges).toHaveLength(1)
      expect(result.current.supportedModels).toHaveLength(1)
      expect(result.current.supportedExchanges).toHaveLength(1)

      consoleSpy.mockRestore()
      consoleLogSpy.mockRestore()
    })

    /**
     * 测试：测试模式 (user 存在但 token 为 null)
     */
    it('should load full configs in test mode (user exists but token is null)', async () => {
      vi.mocked(api.getModelConfigs).mockResolvedValueOnce([
        { id: 'm1', enabled: true },
      ] as any)
      vi.mocked(api.getExchangeConfigs).mockResolvedValueOnce([
        { id: 'e1', enabled: true },
      ] as any)
      vi.mocked(api.getSupportedModels).mockResolvedValueOnce([])
      vi.mocked(api.getSupportedExchanges).mockResolvedValueOnce([])
      vi.mocked(api.getUserSignalSource).mockRejectedValueOnce(
        new Error('Not set')
      )

      const consoleLogSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

      const { result } = renderHook(() => useTradersConfigStore())

      await act(async () => {
        // user exists but token is null (test mode)
        await result.current.loadConfigs({ id: 'test-user' }, null)
      })

      // Should load full configs, not just supported
      expect(api.getModelConfigs).toHaveBeenCalled()
      expect(api.getExchangeConfigs).toHaveBeenCalled()

      consoleLogSpy.mockRestore()
    })
  })

  describe('reset', () => {
    it('should reset store to initial state', () => {
      const { result } = renderHook(() => useTradersConfigStore())

      // Set some data
      act(() => {
        result.current.setAllModels([{ id: 'm1' }] as any)
        result.current.setAllExchanges([{ id: 'e1' }] as any)
        result.current.setSupportedModels([{ id: 's1' }] as any)
        result.current.setSupportedExchanges([{ id: 'se1' }] as any)
        result.current.setUserSignalSource({
          coinPoolUrl: 'https://test.com',
          oiTopUrl: 'https://oi.com',
        })
      })

      // Verify data was set
      expect(result.current.allModels).toHaveLength(1)

      // Reset
      act(() => {
        result.current.reset()
      })

      // Verify reset
      expect(result.current.allModels).toEqual([])
      expect(result.current.allExchanges).toEqual([])
      expect(result.current.supportedModels).toEqual([])
      expect(result.current.supportedExchanges).toEqual([])
      expect(result.current.configuredModels).toEqual([])
      expect(result.current.configuredExchanges).toEqual([])
      expect(result.current.userSignalSource).toEqual({
        coinPoolUrl: '',
        oiTopUrl: '',
      })
    })
  })
})

import { create } from 'zustand'
import type { AIModel, Exchange } from '../types'
import { api } from '../lib/api'

interface SignalSource {
  coinPoolUrl: string
  oiTopUrl: string
}

interface TradersConfigState {
  // 数据
  allModels: AIModel[]
  allExchanges: Exchange[]
  supportedModels: AIModel[]
  supportedExchanges: Exchange[]
  userSignalSource: SignalSource

  // 计算属性
  configuredModels: AIModel[]
  configuredExchanges: Exchange[]

  // Actions
  setAllModels: (models: AIModel[]) => void
  setAllExchanges: (exchanges: Exchange[]) => void
  setSupportedModels: (models: AIModel[]) => void
  setSupportedExchanges: (exchanges: Exchange[]) => void
  setUserSignalSource: (source: SignalSource) => void

  // 异步加载
  loadConfigs: (user: any, token: string | null) => Promise<void>

  // 重置
  reset: () => void
}

const initialState = {
  allModels: [],
  allExchanges: [],
  supportedModels: [],
  supportedExchanges: [],
  userSignalSource: { coinPoolUrl: '', oiTopUrl: '' },
  configuredModels: [],
  configuredExchanges: [],
}

export const useTradersConfigStore = create<TradersConfigState>((set, get) => ({
  ...initialState,

  setAllModels: (models) => {
    set({ allModels: models })
    // 更新 configuredModels
    const configuredModels = models.filter((m) => {
      return m.enabled || (m.customApiUrl && m.customApiUrl.trim() !== '')
    })
    set({ configuredModels })
  },

  setAllExchanges: (exchanges) => {
    set({ allExchanges: exchanges })
    // 更新 configuredExchanges
    const configuredExchanges = exchanges.filter((e) => {
      if (e.id === 'aster') {
        return e.asterUser && e.asterUser.trim() !== ''
      }
      if (e.id === 'hyperliquid') {
        return e.hyperliquidWalletAddr && e.hyperliquidWalletAddr.trim() !== ''
      }
      // 修复: 添加 enabled 判断,与原始逻辑保持一致
      return e.enabled || (e.apiKey && e.apiKey.trim() !== '')
    })
    set({ configuredExchanges })
  },

  setSupportedModels: (models) => set({ supportedModels: models }),
  setSupportedExchanges: (exchanges) => set({ supportedExchanges: exchanges }),
  setUserSignalSource: (source) => set({ userSignalSource: source }),

  loadConfigs: async (user, _token) => {
    // 未登录且非测试模式时，只加载公开的支持模型和交易所
    // 测试模式下：user 存在但 token 为 null，此时也应加载完整配置
    if (!user) {
      // 使用 Promise.allSettled 确保一个请求失败不影响其他请求
      try {
        const results = await Promise.allSettled([
          api.getSupportedModels(),
          api.getSupportedExchanges(),
        ])

        if (results[0].status === 'fulfilled') {
          get().setSupportedModels(results[0].value)
        } else {
          console.error('Failed to load supported models:', results[0].reason)
        }

        if (results[1].status === 'fulfilled') {
          get().setSupportedExchanges(results[1].value)
        } else {
          console.error(
            'Failed to load supported exchanges:',
            results[1].reason
          )
        }
      } catch (err) {
        console.error('Failed to load supported configs:', err)
      }
      return
    }

    // 登录后（或测试模式下有 user）使用 Promise.allSettled 确保部分失败不影响整体
    try {
      const results = await Promise.allSettled([
        api.getModelConfigs(),
        api.getExchangeConfigs(),
        api.getSupportedModels(),
        api.getSupportedExchanges(),
      ])

      if (results[0].status === 'fulfilled') {
        get().setAllModels(results[0].value)
      } else {
        console.error('Failed to load model configs:', results[0].reason)
      }

      if (results[1].status === 'fulfilled') {
        get().setAllExchanges(results[1].value)
      } else {
        console.error('Failed to load exchange configs:', results[1].reason)
      }

      if (results[2].status === 'fulfilled') {
        get().setSupportedModels(results[2].value)
      } else {
        console.error('Failed to load supported models:', results[2].reason)
      }

      if (results[3].status === 'fulfilled') {
        get().setSupportedExchanges(results[3].value)
      } else {
        console.error('Failed to load supported exchanges:', results[3].reason)
      }

      // 加载用户信号源配置
      try {
        const signalSource = await api.getUserSignalSource()
        get().setUserSignalSource({
          coinPoolUrl: signalSource.coin_pool_url || '',
          oiTopUrl: signalSource.oi_top_url || '',
        })
      } catch (error) {
        console.log('📡 用户信号源配置暂未设置')
      }
    } catch (error) {
      console.error('Failed to load configs:', error)
    }
  },

  reset: () => set(initialState),
}))

import { useEffect, useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { api } from '../lib/api'
import {
  Plus,
  TrendingUp,
  TrendingDown,
  Play,
  Activity,
  Eye,
  Trash2,
  Settings,
} from 'lucide-react'
import type { TraderInfo, IndicatorConfig } from '../types'
import { IndicatorConfigPanel } from '../components/IndicatorConfigPanel'


interface BacktestRun {
  id: string
  trader_id: string
  start_time: string
  end_time: string
  initial_balance: number
  status: string
  progress: number
  total_pnl_pct: number
  total_trades: number
  created_at: string
  ai_model_id?: string
  timeframe?: string
}

const DEFAULT_INDICATOR_CONFIG: IndicatorConfig = {
  indicators: ['ema', 'macd', 'rsi', 'atr', 'volume'],
  timeframes: ['3m', '4h'],
  data_points: {
    '3m': 40,
    '4h': 25,
  },
  parameters: {
    rsi_period: 14,
    ema_period: 20,
    macd_fast: 12,
    macd_slow: 26,
    macd_signal: 9,
    atr_period: 14,
  },
}

export default function BacktestPage() {
  const { user } = useAuth()
  const [backtests, setBacktests] = useState<BacktestRun[]>([])
  const [traders, setTraders] = useState<TraderInfo[]>([])
  const [aiModels, setAiModels] = useState<any[]>([])
  const [exchanges, setExchanges] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateForm, setShowCreateForm] = useState(false)

  // 表单状态
  const [isStandaloneMode, setIsStandaloneMode] = useState(false)
  const [selectedTrader, setSelectedTrader] = useState('')
  const [selectedExchange, setSelectedExchange] = useState('')
  const [btcEthLeverage, setBtcEthLeverage] = useState('5')
  const [altcoinLeverage, setAltcoinLeverage] = useState('5')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [initialBalance, setInitialBalance] = useState('10000')
  const [useTraderConfig, setUseTraderConfig] = useState(true)
  const [creating, setCreating] = useState(false)

  // 高级配置状态
  const [selectedAiModel, setSelectedAiModel] = useState('')
  const [timeframe, setTimeframe] = useState('3m')
  const [dataPoints, setDataPoints] = useState('100')
  const [preheatHours, setPreheatHours] = useState('12')
  const [scanInterval, setScanInterval] = useState('3')
  const [slippage, setSlippage] = useState('10')
  const [tradingSymbols, setTradingSymbols] = useState('')

  // 策略配置状态
  const [indicatorConfig, setIndicatorConfig] = useState<IndicatorConfig>(DEFAULT_INDICATOR_CONFIG)
  const [customPrompt, setCustomPrompt] = useState('')
  const [overrideBasePrompt, setOverrideBasePrompt] = useState(false)
  const [systemPromptTemplate, setSystemPromptTemplate] = useState('default')
  const [promptTemplates, setPromptTemplates] = useState<{ name: string }[]>([])

  useEffect(() => {
    if (user) {
      loadBacktests()
      loadTraders()
      loadAiModels()
      loadTraders()
      loadAiModels()
      loadExchanges()
      loadPromptTemplates()
    }
  }, [user])

  const loadBacktests = async () => {
    try {
      const data = await api.getBacktests()
      setBacktests(data || [])
    } catch (error) {
      console.error('Failed to load backtests:', error)
    } finally {
      setLoading(false)
    }
  }

  const loadTraders = async () => {
    try {
      const data = await api.getTraders()
      setTraders(data || [])
    } catch (error) {
      console.error('Failed to load traders:', error)
    }
  }

  const loadAiModels = async () => {
    try {
      const data = await api.getModelConfigs()
      setAiModels(data || [])
    } catch (error) {
      console.error('Failed to load AI models:', error)
    }
  }

  const loadExchanges = async () => {
    try {
      const data = await api.getExchangeConfigs()
      setExchanges(data || [])
    } catch (error) {
      console.error('Failed to load exchanges:', error)
    }
  }

  const loadPromptTemplates = async () => {
    // 模拟获取模板列表，实际应调用API
    // 由于API不可用，这里使用硬编码列表，与TraderConfigModal一致
    setPromptTemplates([
      { name: 'default' },
      { name: 'adaptive' },
      { name: 'adaptive_relaxed' },
      { name: 'Hansen' },
      { name: 'nof1' },
      { name: 'taro_long_prompts' },
    ])
  }

  // 当选择交易员时，自动填充部分默认值
  const handleTraderChange = (traderId: string) => {
    setSelectedTrader(traderId)
    const trader = traders.find(t => t.trader_id === traderId)
    if (trader) {
      setScanInterval(trader.scan_interval_minutes?.toString() || '3')
    }
  }

  const setTimeRange = (hours: number) => {
    const end = new Date()
    const start = new Date(end.getTime() - hours * 60 * 60 * 1000)

    // Format for datetime-local: YYYY-MM-DDThh:mm
    const toLocalISOString = (date: Date) => {
      const offset = date.getTimezoneOffset() * 60000 // offset in milliseconds
      const localDate = new Date(date.getTime() - offset)
      return localDate.toISOString().slice(0, 16)
    }

    setStartDate(toLocalISOString(start))
    setEndDate(toLocalISOString(end))
  }

  const handleCreateBacktest = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)

    try {
      // 验证
      if (isStandaloneMode) {
        if (!selectedAiModel) {
          alert('请选择 AI 模型')
          setCreating(false)
          return
        }
        if (!selectedExchange) {
          alert('请选择交易所')
          setCreating(false)
          return
        }
      }

      const data = await api.createBacktest({
        trader_id: isStandaloneMode ? undefined : selectedTrader,
        exchange_id: isStandaloneMode ? selectedExchange : undefined,
        start_time: new Date(startDate).toISOString(),
        end_time: new Date(endDate).toISOString(),
        initial_balance: parseFloat(initialBalance),
        use_trader_config: isStandaloneMode ? false : useTraderConfig,
        // 高级配置
        ai_model_id: selectedAiModel || undefined,
        timeframe: timeframe,
        data_points: parseInt(dataPoints),
        preheat_hours: parseInt(preheatHours),
        scan_interval_minutes: parseInt(scanInterval),
        slippage: parseInt(slippage),
        btc_eth_leverage: isStandaloneMode ? parseInt(btcEthLeverage) : undefined,
        altcoin_leverage: isStandaloneMode ? parseInt(altcoinLeverage) : undefined,
        trading_symbols: tradingSymbols || undefined,
        // 策略配置 (独立模式或不使用交易员配置时生效)
        indicator_config: (isStandaloneMode || !useTraderConfig) ? indicatorConfig : undefined,
        custom_prompt: (isStandaloneMode || !useTraderConfig) ? customPrompt : undefined,
        override_base_prompt: (isStandaloneMode || !useTraderConfig) ? overrideBasePrompt : undefined,
        system_prompt_template: (isStandaloneMode || !useTraderConfig) ? systemPromptTemplate : undefined,
      })

      console.log('Backtest created:', data)
      setShowCreateForm(false)
      loadBacktests()

      // 重置表单
      setSelectedTrader('')
      setStartDate('')
      setEndDate('')
      setInitialBalance('10000')
      setUseTraderConfig(true)
      // 重置高级选项
      setSelectedAiModel('')
      setTradingSymbols('')
      setCustomPrompt('')
      setOverrideBasePrompt(false)
      setSystemPromptTemplate('default')
      setIndicatorConfig(DEFAULT_INDICATOR_CONFIG)
    } catch (error: any) {
      console.error('Failed to create backtest:', error)
      alert(error.message || 'Failed to create backtest')
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('确定要删除这个回测吗？')) return

    try {
      await api.deleteBacktest(id)
      loadBacktests()
    } catch (error) {
      console.error('Failed to delete backtest:', error)
      alert('删除失败')
    }
  }

  const handleViewDetails = (id: string) => {
    window.history.pushState({}, '', `/backtest/${id}`)
    window.dispatchEvent(new PopStateEvent('popstate'))
  }

  const handleMockTest = async () => {
    let traderId = ''

    if (traders.length > 0) {
      traderId = traders[0].trader_id
    } else {
      // 尝试自动创建一个 Mock 交易员
      try {
        const [models, exchanges] = await Promise.all([
          api.getModelConfigs(),
          api.getExchangeConfigs(),
        ])

        const enabledModel = Array.isArray(models)
          ? models.find((m: any) => m.enabled)
          : null
        const enabledExchange = Array.isArray(exchanges)
          ? exchanges.find((e: any) => e.enabled)
          : null

        if (!enabledModel || !enabledExchange) {
          alert(
            'Mock 测试需要至少配置一个启用的 AI 模型和交易所。\n请前往 "AI 交易员" 页面配置 API Key。'
          )
          return
        }

        const newTrader = await api.createTrader({
          name: 'Mock Trader',
          ai_model_id: enabledModel.id,
          exchange_id: enabledExchange.id,
          initial_balance: 10000,
          scan_interval_minutes: 5,
          btc_eth_leverage: 1,
          altcoin_leverage: 1,
          trading_symbols: 'BTCUSDT',
          use_coin_pool: false,
          use_oi_top: false,
        })

        traderId = newTrader.trader_id
        loadTraders()
      } catch (err) {
        console.error('Failed to auto-create mock trader:', err)
        alert('无法自动创建 Mock 交易员，请手动创建一个交易员后再试。')
        return
      }

    }

    const now = new Date()
    const endTime = now.toISOString()
    const startTime = new Date(
      now.getTime() - 3 * 24 * 60 * 60 * 1000
    ).toISOString() // 3 days ago

    try {
      await api.createBacktest({
        trader_id: traderId,
        start_time: startTime,
        end_time: endTime,
        initial_balance: 10000,
        use_trader_config: true,
        mock_mode: true,
      })

      const res = await api.getBacktests()
      setBacktests(res)
      alert('Mock 测试已启动')
    } catch (err) {
      console.error(err)
      alert('启动 Mock 测试失败')
    }
  }


  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'text-green-500'
      case 'running':
        return 'text-blue-500'
      case 'failed':
        return 'text-red-500'
      default:
        return 'text-gray-500'
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case 'completed':
        return '已完成'
      case 'running':
        return '运行中'
      case 'failed':
        return '失败'
      case 'pending':
        return '等待中'
      default:
        return status
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-400">加载中...</div>
      </div>
    )
  }

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">策略回测</h1>
          <p className="text-gray-400">使用历史数据验证交易策略的有效性</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleMockTest}
            className="bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 rounded-lg flex items-center gap-2 transition-colors"
          >
            <Play className="w-4 h-4" />
            Mock 测试
          </button>
          <button
            onClick={() => setShowCreateForm(!showCreateForm)}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center gap-2 transition-colors"
          >
            <Plus className="w-4 h-4" />
            新建回测
          </button>
        </div>
      </div>

      {/* 创建表单 */}
      {showCreateForm && (
        <div
          className="rounded-lg p-6 mb-6"
          style={{
            background: 'var(--panel-bg)',
            border: '1px solid var(--panel-border)',
          }}
        >
          <h2 className="text-xl font-bold text-white mb-4">创建新回测</h2>
          <form onSubmit={handleCreateBacktest} className="space-y-6">
            {/* 基础信息 */}
            <div className="space-y-4">
              {/* 模式选择 */}
              <div className="flex gap-4 mb-4">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    checked={!isStandaloneMode}
                    onChange={() => setIsStandaloneMode(false)}
                    className="w-4 h-4 text-blue-600 bg-gray-800 border-gray-700 focus:ring-blue-500"
                  />
                  <span className="text-gray-200">基于现有交易员</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    checked={isStandaloneMode}
                    onChange={() => setIsStandaloneMode(true)}
                    className="w-4 h-4 text-blue-600 bg-gray-800 border-gray-700 focus:ring-blue-500"
                  />
                  <span className="text-gray-200">独立配置回测</span>
                </label>
              </div>

              {!isStandaloneMode ? (
                <div>
                  <label
                    htmlFor="trader-select"
                    className="block text-sm font-medium text-gray-300 mb-2"
                  >
                    选择交易员
                  </label>
                  <select
                    id="trader-select"
                    value={selectedTrader}
                    onChange={(e) => handleTraderChange(e.target.value)}
                    required={!isStandaloneMode}
                    className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"
                  >
                    <option value="">请选择...</option>
                    {traders.map((trader) => (
                      <option key={trader.trader_id} value={trader.trader_id}>
                        {trader.trader_name}
                      </option>
                    ))}
                  </select>
                </div>
              ) : (
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      选择交易所 <span className="text-red-500">*</span>
                    </label>
                    <select
                      value={selectedExchange}
                      onChange={(e) => setSelectedExchange(e.target.value)}
                      required={isStandaloneMode}
                      className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"
                    >
                      <option value="">请选择...</option>
                      {exchanges.map((ex) => (
                        <option key={ex.id} value={ex.id}>
                          {ex.name} ({ex.type})
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      选择 AI 模型 <span className="text-red-500">*</span>
                    </label>
                    <select
                      value={selectedAiModel}
                      onChange={(e) => setSelectedAiModel(e.target.value)}
                      required={isStandaloneMode}
                      className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"
                    >
                      <option value="">请选择...</option>
                      {aiModels.map((model) => (
                        <option key={model.id} value={model.id}>
                          {model.name} ({model.provider})
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      BTC/ETH 杠杆
                    </label>
                    <input
                      type="number"
                      value={btcEthLeverage}
                      onChange={(e) => setBtcEthLeverage(e.target.value)}
                      min="1"
                      max="125"
                      className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      山寨币 杠杆
                    </label>
                    <input
                      type="number"
                      value={altcoinLeverage}
                      onChange={(e) => setAltcoinLeverage(e.target.value)}
                      min="1"
                      max="50"
                      className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"
                    />
                  </div>
                </div>
              )}

              <div className="flex gap-2 justify-end">
                <button
                  type="button"
                  onClick={() => setTimeRange(4)}
                  className="text-xs bg-gray-700 hover:bg-gray-600 text-white px-3 py-1 rounded transition-colors"
                >
                  过去4小时
                </button>
                <button
                  type="button"
                  onClick={() => setTimeRange(24)}
                  className="text-xs bg-gray-700 hover:bg-gray-600 text-white px-3 py-1 rounded transition-colors"
                >
                  过去24小时
                </button>
                <button
                  type="button"
                  onClick={() => setTimeRange(24 * 3)}
                  className="text-xs bg-gray-700 hover:bg-gray-600 text-white px-3 py-1 rounded transition-colors"
                >
                  过去3天
                </button>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label
                    htmlFor="start-date"
                    className="block text-sm font-medium text-gray-300 mb-2"
                  >
                    开始时间
                  </label>
                  <input
                    id="start-date"
                    type="datetime-local"
                    value={startDate}
                    onChange={(e) => setStartDate(e.target.value)}
                    required
                    className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"
                  />
                </div>
                <div>
                  <label
                    htmlFor="end-date"
                    className="block text-sm font-medium text-gray-300 mb-2"
                  >
                    结束时间
                  </label>
                  <input
                    id="end-date"
                    type="datetime-local"
                    value={endDate}
                    onChange={(e) => setEndDate(e.target.value)}
                    required
                    className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"
                  />
                </div>
              </div>

              <div>
                <label
                  htmlFor="initial-balance"
                  className="block text-sm font-medium text-gray-300 mb-2"
                >
                  初始资金 (USDT)
                </label>
                <input
                  id="initial-balance"
                  type="number"
                  value={initialBalance}
                  onChange={(e) => setInitialBalance(e.target.value)}
                  required
                  min="100"
                  step="100"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-white"
                />
              </div>

              {!isStandaloneMode && (
                <div className="flex items-center p-4 bg-gray-800/50 rounded-lg border border-gray-700">
                  <input
                    type="checkbox"
                    id="useTraderConfig"
                    checked={useTraderConfig}
                    onChange={(e) => setUseTraderConfig(e.target.checked)}
                    className="w-5 h-5 text-blue-600 bg-gray-800 border-gray-700 rounded focus:ring-blue-500"
                  />
                  <label
                    htmlFor="useTraderConfig"
                    className="ml-3 text-sm font-medium text-gray-200 cursor-pointer"
                  >
                    使用交易员的配置 (指标、策略、AI模型等)
                    <p className="text-xs text-gray-400 mt-1 font-normal">
                      取消勾选以自定义回测参数、策略和指标配置
                    </p>
                  </label>
                </div>
              )}
            </div>

            {/* 高级配置区域 - 仅当不使用交易员配置时显示 */}
            {(isStandaloneMode || !useTraderConfig) && (
              <div className="space-y-6 animate-fade-in">
                <div className="border-t border-gray-700 pt-6">
                  <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
                    <Settings className="w-5 h-5" />
                    自定义配置
                  </h3>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {/* 左侧：参数设置 */}
                    <div className="space-y-4">
                      <div className="bg-gray-800/30 p-4 rounded-lg border border-gray-700/50">
                        <h4 className="text-sm font-medium text-gray-300 mb-3">基础参数</h4>
                        <div className="space-y-4">
                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              AI 模型 {isStandaloneMode && '(已在上方选择)'}
                            </label>
                            <select
                              value={selectedAiModel}
                              onChange={(e) => setSelectedAiModel(e.target.value)}
                              disabled={isStandaloneMode}
                              className={`w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm ${isStandaloneMode ? 'opacity-50 cursor-not-allowed' : ''}`}
                            >
                              <option value="">{isStandaloneMode ? '已选择' : '默认 (使用交易员模型)'}</option>
                              {!isStandaloneMode && aiModels.map((model) => (
                                <option key={model.id} value={model.id}>
                                  {model.name} ({model.provider})
                                </option>
                              ))}
                            </select>
                          </div>

                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              交易币种
                            </label>
                            <input
                              type="text"
                              value={tradingSymbols}
                              onChange={(e) => setTradingSymbols(e.target.value)}
                              placeholder="BTCUSDT,ETHUSDT"
                              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm"
                            />
                          </div>
                        </div>
                      </div>

                      <div className="bg-gray-800/30 p-4 rounded-lg border border-gray-700/50">
                        <h4 className="text-sm font-medium text-gray-300 mb-3">回测参数</h4>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              K线周期
                            </label>
                            <select
                              value={timeframe}
                              onChange={(e) => setTimeframe(e.target.value)}
                              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm"
                            >
                              <option value="1m">1m</option>
                              <option value="3m">3m</option>
                              <option value="5m">5m</option>
                              <option value="15m">15m</option>
                              <option value="1h">1h</option>
                              <option value="4h">4h</option>
                            </select>
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              扫描间隔 (分)
                            </label>
                            <input
                              type="number"
                              value={scanInterval}
                              onChange={(e) => setScanInterval(e.target.value)}
                              min="1"
                              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm"
                            />
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              预热 (小时)
                            </label>
                            <input
                              type="number"
                              value={preheatHours}
                              onChange={(e) => setPreheatHours(e.target.value)}
                              min="1"
                              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm"
                            />
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              数据点数
                            </label>
                            <input
                              type="number"
                              value={dataPoints}
                              onChange={(e) => setDataPoints(e.target.value)}
                              min="20"
                              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm"
                            />
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              滑点 (bps)
                            </label>
                            <input
                              type="number"
                              value={slippage}
                              onChange={(e) => setSlippage(e.target.value)}
                              min="0"
                              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm"
                            />
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* 右侧：策略与指标 */}
                    <div className="space-y-4">
                      <div className="bg-gray-800/30 p-4 rounded-lg border border-gray-700/50">
                        <h4 className="text-sm font-medium text-gray-300 mb-3">策略配置</h4>
                        <div className="space-y-4">
                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              系统提示词模板
                            </label>
                            <select
                              value={systemPromptTemplate}
                              onChange={(e) => setSystemPromptTemplate(e.target.value)}
                              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm"
                            >
                              {promptTemplates.map((template) => (
                                <option key={template.name} value={template.name}>
                                  {template.name}
                                </option>
                              ))}
                            </select>
                          </div>

                          <div className="flex items-center gap-2">
                            <input
                              type="checkbox"
                              id="overrideBasePrompt"
                              checked={overrideBasePrompt}
                              onChange={(e) => setOverrideBasePrompt(e.target.checked)}
                              className="w-4 h-4 text-blue-600 bg-gray-800 border-gray-700 rounded"
                            />
                            <label htmlFor="overrideBasePrompt" className="text-sm text-gray-300">
                              覆盖基础 Prompt
                            </label>
                          </div>

                          <div>
                            <label className="block text-sm font-medium text-gray-400 mb-1">
                              自定义策略 Prompt
                            </label>
                            <textarea
                              value={customPrompt}
                              onChange={(e) => setCustomPrompt(e.target.value)}
                              rows={4}
                              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm font-mono"
                              placeholder="输入自定义的交易策略提示词..."
                            />
                          </div>
                        </div>
                      </div>

                      <div className="bg-gray-800/30 p-4 rounded-lg border border-gray-700/50">
                        <h4 className="text-sm font-medium text-gray-300 mb-3">指标配置</h4>
                        <IndicatorConfigPanel
                          config={indicatorConfig}
                          onConfigChange={setIndicatorConfig}
                          isEditing={true}
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div className="flex gap-3 pt-4">
              <button
                type="submit"
                disabled={creating}
                className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {creating ? '创建中...' : '开始回测'}
              </button>
              <button
                type="button"
                onClick={() => setShowCreateForm(false)}
                className="border border-gray-700 text-gray-300 hover:bg-gray-800 px-6 py-2 rounded-lg"
              >
                取消
              </button>
            </div>
          </form>
        </div>
      )}

      {/* 回测列表 */}
      <div className="space-y-4">
        {backtests.length === 0 ? (
          <div
            className="text-center py-12 rounded-lg"
            style={{
              background: 'var(--panel-bg)',
              border: '1px solid var(--panel-border)',
            }}
          >
            <Activity className="w-12 h-12 text-gray-600 mx-auto mb-3" />
            <p className="text-gray-400">暂无回测记录</p>
            <p className="text-gray-500 text-sm mt-1">点击"新建回测"开始</p>
          </div>
        ) : (
          backtests.map((backtest) => (
            <div
              key={backtest.id}
              className="rounded-lg p-5 transition-colors"
              style={{
                background: 'var(--panel-bg)',
                border: '1px solid var(--panel-border)',
              }}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <span
                      className={`font-medium ${getStatusColor(backtest.status)}`}
                    >
                      {getStatusText(backtest.status)}
                    </span>
                    {backtest.status === 'running' && (
                      <span className="text-sm text-gray-400">
                        {(backtest.progress * 100).toFixed(1)}%
                      </span>
                    )}
                  </div>

                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <div className="text-gray-400 mb-1">时间范围</div>
                      <div className="text-white">
                        {new Date(backtest.start_time).toLocaleDateString()} -{' '}
                        {new Date(backtest.end_time).toLocaleDateString()}
                      </div>
                    </div>
                    <div>
                      <div className="text-gray-400 mb-1">初始资金</div>
                      <div className="text-white">
                        ${backtest.initial_balance.toLocaleString()}
                      </div>
                    </div>
                    {backtest.status === 'completed' && (
                      <>
                        <div>
                          <div className="text-gray-400 mb-1">总收益</div>
                          <div
                            className={`font-medium ${backtest.total_pnl_pct >= 0
                              ? 'text-green-500'
                              : 'text-red-500'
                              }`}
                          >
                            {backtest.total_pnl_pct >= 0 ? (
                              <TrendingUp className="inline w-4 h-4" />
                            ) : (
                              <TrendingDown className="inline w-4 h-4" />
                            )}{' '}
                            {backtest.total_pnl_pct.toFixed(2)}%
                          </div>
                        </div>
                        <div>
                          <div className="text-gray-400 mb-1">交易次数</div>
                          <div className="text-white">
                            {backtest.total_trades}
                          </div>
                        </div>
                      </>
                    )}
                  </div>

                  <div className="text-xs text-gray-500 mt-3">
                    创建于 {new Date(backtest.created_at).toLocaleString()}
                  </div>
                </div>

                <div className="flex gap-2">
                  {(backtest.status === 'completed' ||
                    backtest.status === 'running') && (
                      <button
                        onClick={() => handleViewDetails(backtest.id)}
                        className="border border-gray-700 text-gray-300 hover:bg-gray-800 p-2 rounded-lg"
                        title="查看详情"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                    )}
                  <button
                    onClick={() => handleDelete(backtest.id)}
                    className="border border-gray-700 text-red-500 hover:bg-gray-800 p-2 rounded-lg"
                    title="删除"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

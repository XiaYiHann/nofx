import { useEffect, useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { api } from '../lib/api'
import { Plus, TrendingUp, TrendingDown, Activity, Trash2, Eye } from 'lucide-react'
import type { TraderInfo } from '../types'

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
}

export default function BacktestPage() {
  const { token } = useAuth()
  const [backtests, setBacktests] = useState<BacktestRun[]>([])
  const [traders, setTraders] = useState<TraderInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateForm, setShowCreateForm] = useState(false)
  
  // 表单状态
  const [selectedTrader, setSelectedTrader] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [initialBalance, setInitialBalance] = useState('10000')
  const [useTraderConfig, setUseTraderConfig] = useState(true)
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    if (token) {
      loadBacktests()
      loadTraders()
    }
  }, [token])

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

  const handleCreateBacktest = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)

    try {
      const data = await api.createBacktest({
        trader_id: selectedTrader,
        start_time: new Date(startDate).toISOString(),
        end_time: new Date(endDate).toISOString(),
        initial_balance: parseFloat(initialBalance),
        use_trader_config: useTraderConfig,
      })

      console.log('Backtest created:', data)
      setShowCreateForm(false)
      loadBacktests()
      
      // 重置表单
      setSelectedTrader('')
      setStartDate('')
      setEndDate('')
      setInitialBalance('10000')
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
        <button
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center gap-2 transition-colors"
        >
          <Plus className="w-4 h-4" />
          新建回测
        </button>
      </div>

      {/* 创建表单 */}
      {showCreateForm && (
        <div className="bg-gray-900 rounded-lg p-6 mb-6 border border-gray-800">
          <h2 className="text-xl font-bold text-white mb-4">创建新回测</h2>
          <form onSubmit={handleCreateBacktest} className="space-y-4">
            <div>
              <label htmlFor="trader-select" className="block text-sm font-medium text-gray-300 mb-2">
                选择交易员
              </label>
              <select
                id="trader-select"
                value={selectedTrader}
                onChange={(e) => setSelectedTrader(e.target.value)}
                required
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

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="start-date" className="block text-sm font-medium text-gray-300 mb-2">
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
                <label htmlFor="end-date" className="block text-sm font-medium text-gray-300 mb-2">
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
              <label htmlFor="initial-balance" className="block text-sm font-medium text-gray-300 mb-2">
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

            <div className="flex items-center">
              <input
                type="checkbox"
                id="useTraderConfig"
                checked={useTraderConfig}
                onChange={(e) => setUseTraderConfig(e.target.checked)}
                className="w-4 h-4 text-blue-600 bg-gray-800 border-gray-700 rounded"
              />
              <label htmlFor="useTraderConfig" className="ml-2 text-sm text-gray-300">
                使用交易员的配置(指标、提示词等)
              </label>
            </div>

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
          <div className="text-center py-12 bg-gray-900 rounded-lg border border-gray-800">
            <Activity className="w-12 h-12 text-gray-600 mx-auto mb-3" />
            <p className="text-gray-400">暂无回测记录</p>
            <p className="text-gray-500 text-sm mt-1">点击"新建回测"开始</p>
          </div>
        ) : (
          backtests.map((backtest) => (
            <div
              key={backtest.id}
              className="bg-gray-900 rounded-lg p-5 border border-gray-800 hover:border-gray-700 transition-colors"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <span className={`font-medium ${getStatusColor(backtest.status)}`}>
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
                            className={`font-medium ${
                              backtest.total_pnl_pct >= 0
                                ? 'text-green-500'
                                : 'text-red-500'
                            }`}
                          >
                            {backtest.total_pnl_pct >= 0 ? <TrendingUp className="inline w-4 h-4" /> : <TrendingDown className="inline w-4 h-4" />}
                            {' '}{backtest.total_pnl_pct.toFixed(2)}%
                          </div>
                        </div>
                        <div>
                          <div className="text-gray-400 mb-1">交易次数</div>
                          <div className="text-white">{backtest.total_trades}</div>
                        </div>
                      </>
                    )}
                  </div>
                  
                  <div className="text-xs text-gray-500 mt-3">
                    创建于 {new Date(backtest.created_at).toLocaleString()}
                  </div>
                </div>

                <div className="flex gap-2">
                  {backtest.status === 'completed' && (
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

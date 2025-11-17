import { useEffect, useState } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { api } from '../lib/api'
import { ArrowLeft } from 'lucide-react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts'

interface BacktestDetail {
  id: string
  trader_id: string
  start_time: string
  end_time: string
  initial_balance: number
  final_equity: number
  total_pnl: number
  total_pnl_pct: number
  max_drawdown: number
  sharpe_ratio: number
  win_rate: number
  total_trades: number
  status: string
  created_at: string
  completed_at: string
}

interface EquitySnapshot {
  time: string
  equity: number
  pnl: number
  pnl_pct: number
}

interface Trade {
  id: number
  symbol: string
  side: string
  action: string
  entry_price: number
  exit_price: number
  quantity: number
  leverage: number
  pnl: number
  pnl_pct: number
  fee: number
  entry_time: string
  exit_time: string
}

export default function BacktestDetailPage({ backtestId }: { backtestId: string }) {
  const { token } = useAuth()
  const [backtest, setBacktest] = useState<BacktestDetail | null>(null)
  const [equityHistory, setEquityHistory] = useState<EquitySnapshot[]>([])
  const [trades, setTrades] = useState<Trade[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (token && backtestId) {
      loadBacktestData()
    }
  }, [token, backtestId])

  const loadBacktestData = async () => {
    try {
      const [backtestData, equityData, tradesData] = await Promise.all([
        api.getBacktest(backtestId),
        api.getBacktestEquityHistory(backtestId),
        api.getBacktestTrades(backtestId),
      ])

      setBacktest(backtestData)
      setEquityHistory(equityData || [])
      setTrades(tradesData || [])
    } catch (error) {
      console.error('Failed to load backtest data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleBack = () => {
    window.history.pushState({}, '', '/backtest')
    window.dispatchEvent(new PopStateEvent('popstate'))
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-400">加载中...</div>
      </div>
    )
  }

  if (!backtest) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-400">回测数据未找到</p>
        <button
          onClick={handleBack}
          className="mt-4 bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg"
        >
          返回列表
        </button>
      </div>
    )
  }

  // 准备图表数据
  const chartData = equityHistory.map((snapshot) => ({
    time: new Date(snapshot.time).toLocaleString('zh-CN', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }),
    equity: snapshot.equity,
    pnl_pct: snapshot.pnl_pct,
  }))

  // 计算统计信息
  const winningTrades = trades.filter((t) => t.pnl > 0).length
  const losingTrades = trades.filter((t) => t.pnl < 0).length

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="mb-6">
        <button
          onClick={handleBack}
          className="mb-4 border border-gray-700 text-gray-300 hover:bg-gray-800 px-4 py-2 rounded-lg flex items-center gap-2"
        >
          <ArrowLeft className="w-4 h-4" />
          返回列表
        </button>
        <h1 className="text-3xl font-bold text-white mb-2">回测详情</h1>
        <p className="text-gray-400">
          {new Date(backtest.start_time).toLocaleDateString()} -{' '}
          {new Date(backtest.end_time).toLocaleDateString()}
        </p>
      </div>

      {/* 核心指标 */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-lg p-4 border border-gray-800">
          <div className="text-gray-400 text-sm mb-1">总收益</div>
          <div
            className={`text-2xl font-bold ${
              backtest.total_pnl_pct >= 0 ? 'text-green-500' : 'text-red-500'
            }`}
          >
            {backtest.total_pnl_pct >= 0 ? '+' : ''}
            {backtest.total_pnl_pct.toFixed(2)}%
          </div>
          <div className="text-gray-500 text-sm mt-1">
            ${backtest.total_pnl.toFixed(2)}
          </div>
        </div>

        <div className="bg-gray-900 rounded-lg p-4 border border-gray-800">
          <div className="text-gray-400 text-sm mb-1">最大回撤</div>
          <div className="text-2xl font-bold text-red-500">
            {backtest.max_drawdown.toFixed(2)}%
          </div>
        </div>

        <div className="bg-gray-900 rounded-lg p-4 border border-gray-800">
          <div className="text-gray-400 text-sm mb-1">夏普率</div>
          <div className="text-2xl font-bold text-white">
            {backtest.sharpe_ratio.toFixed(2)}
          </div>
        </div>

        <div className="bg-gray-900 rounded-lg p-4 border border-gray-800">
          <div className="text-gray-400 text-sm mb-1">胜率</div>
          <div className="text-2xl font-bold text-white">
            {backtest.win_rate.toFixed(1)}%
          </div>
          <div className="text-gray-500 text-sm mt-1">
            {winningTrades}胜 / {losingTrades}负
          </div>
        </div>
      </div>

      {/* 净值曲线 */}
      <div className="bg-gray-900 rounded-lg p-6 border border-gray-800 mb-6">
        <h2 className="text-xl font-bold text-white mb-4">净值曲线</h2>
        <ResponsiveContainer width="100%" height={400}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
            <XAxis dataKey="time" stroke="#9CA3AF" />
            <YAxis stroke="#9CA3AF" />
            <Tooltip
              contentStyle={{
                backgroundColor: '#1F2937',
                border: '1px solid #374151',
                borderRadius: '8px',
              }}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="equity"
              stroke="#3B82F6"
              strokeWidth={2}
              dot={false}
              name="净值"
            />
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* 交易记录 */}
      <div className="bg-gray-900 rounded-lg p-6 border border-gray-800">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-bold text-white">交易记录</h2>
          <div className="text-sm text-gray-400">
            共 {backtest.total_trades} 笔交易
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800">
                <th className="text-left py-3 px-4 text-gray-400 font-medium">币种</th>
                <th className="text-left py-3 px-4 text-gray-400 font-medium">方向</th>
                <th className="text-right py-3 px-4 text-gray-400 font-medium">入场价</th>
                <th className="text-right py-3 px-4 text-gray-400 font-medium">出场价</th>
                <th className="text-right py-3 px-4 text-gray-400 font-medium">数量</th>
                <th className="text-right py-3 px-4 text-gray-400 font-medium">杠杆</th>
                <th className="text-right py-3 px-4 text-gray-400 font-medium">盈亏</th>
                <th className="text-right py-3 px-4 text-gray-400 font-medium">手续费</th>
                <th className="text-left py-3 px-4 text-gray-400 font-medium">时间</th>
              </tr>
            </thead>
            <tbody>
              {trades.map((trade) => (
                <tr key={trade.id} className="border-b border-gray-800 hover:bg-gray-800/50">
                  <td className="py-3 px-4 text-white font-medium">{trade.symbol}</td>
                  <td className="py-3 px-4">
                    <span
                      className={`px-2 py-1 rounded text-xs ${
                        trade.side === 'long'
                          ? 'bg-green-500/20 text-green-500'
                          : 'bg-red-500/20 text-red-500'
                      }`}
                    >
                      {trade.side === 'long' ? '做多' : '做空'}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-right text-gray-300">
                    ${trade.entry_price.toFixed(4)}
                  </td>
                  <td className="py-3 px-4 text-right text-gray-300">
                    ${trade.exit_price.toFixed(4)}
                  </td>
                  <td className="py-3 px-4 text-right text-gray-300">
                    {trade.quantity.toFixed(4)}
                  </td>
                  <td className="py-3 px-4 text-right text-gray-300">
                    {trade.leverage}x
                  </td>
                  <td className="py-3 px-4 text-right">
                    <span
                      className={`font-medium ${
                        trade.pnl >= 0 ? 'text-green-500' : 'text-red-500'
                      }`}
                    >
                      {trade.pnl >= 0 ? '+' : ''}${trade.pnl.toFixed(2)}
                      <span className="text-xs ml-1">
                        ({trade.pnl_pct >= 0 ? '+' : ''}
                        {trade.pnl_pct.toFixed(2)}%)
                      </span>
                    </span>
                  </td>
                  <td className="py-3 px-4 text-right text-gray-400">
                    ${trade.fee.toFixed(2)}
                  </td>
                  <td className="py-3 px-4 text-gray-400">
                    {new Date(trade.entry_time).toLocaleString('zh-CN', {
                      month: 'short',
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {trades.length === 0 && (
          <div className="text-center py-8 text-gray-400">暂无交易记录</div>
        )}
      </div>
    </div>
  )
}

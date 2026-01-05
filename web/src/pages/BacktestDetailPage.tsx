import { useEffect, useState, useCallback } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { api } from '../lib/api'
import {
  ArrowLeft,
  TrendingUp,
  TrendingDown,
  BarChart3,
  Target,
  Wifi,
  WifiOff,
  Activity,
  Brain,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { EquityChart } from '../components/EquityChart'
import { useBacktestSocket } from '../hooks/useBacktestSocket'
import type { Backtest, BacktestTrade, DecisionRecord } from '../types'
import type { EquitySnapshotPayload, DecisionPayload } from '../lib/backtestSocket'

export default function BacktestDetailPage({
  backtestId,
}: {
  backtestId: string
}) {
  const { user } = useAuth()
  const [backtest, setBacktest] = useState<Backtest | null>(null)
  const [trades, setTrades] = useState<BacktestTrade[]>([])
  const [decisions, setDecisions] = useState<DecisionRecord[]>([])
  const [loading, setLoading] = useState(true)

  // 实时决策面板状态
  const [expandedDecisions, setExpandedDecisions] = useState<Set<number>>(new Set())
  const [showLivePanel, setShowLivePanel] = useState(true)

  // WebSocket 连接 - 仅在回测运行时启用
  const isRunning = backtest?.status === 'running'
  const {
    status: wsStatus,
    progress,
    snapshots: wsSnapshots,
    decisions: wsDecisions,
    currentCycle,
    progressMessage,
    isComplete: wsComplete,
    error: wsError,
  } = useBacktestSocket(backtestId, {
    enabled: isRunning,
    autoConnect: true,
    debug: import.meta.env.DEV,
  })

  // 转换 WS 快照格式为 EquityChart 需要的格式
  const externalEquityData = wsSnapshots.map((s: EquitySnapshotPayload) => ({
    cycle: s.cycle,
    timestamp: s.timestamp,
    total_equity: s.total_equity,
    pnl: s.pnl,
    pnl_pct: s.pnl_pct,
  }))

  const loadBacktestData = useCallback(async () => {
    try {
      // 1. 首先获取回测基本信息（关键数据）
      const backtestData = await api.getBacktest(backtestId)
      setBacktest(backtestData)

      // 2. 并行获取其他数据（非关键数据，允许失败）
      const [, tradesData, decisionsData] = await Promise.all([
        api.getBacktestEquityHistory(backtestId).catch((err) => {
          console.warn('Failed to load equity history:', err)
          return []
        }),
        api.getBacktestTrades(backtestId).catch((err) => {
          console.warn('Failed to load trades:', err)
          return []
        }),
        api.getBacktestDecisions(backtestId).catch((err) => {
          console.warn('Failed to load decisions:', err)
          return []
        }),
      ])

      setTrades(tradesData || [])
      setDecisions(decisionsData || [])
    } catch (error) {
      console.error('Failed to load backtest data:', error)
    } finally {
      setLoading(false)
    }
  }, [backtestId])

  useEffect(() => {
    if (user && backtestId) {
      loadBacktestData()
    }
  }, [user, backtestId, loadBacktestData])

  // WebSocket 完成时刷新数据
  useEffect(() => {
    if (wsComplete) {
      loadBacktestData()
    }
  }, [wsComplete, loadBacktestData])

  // 仅在未建立 WebSocket 连接时使用轮询（降级方案）
  useEffect(() => {
    if (backtest?.status === 'running' && wsStatus !== 'connected') {
      const timer = setInterval(() => {
        loadBacktestData()
      }, 5000) // 降低轮询频率（WS 作为主要通道）
      return () => clearInterval(timer)
    }
  }, [backtest?.status, wsStatus, loadBacktestData])

  const handleBack = () => {
    window.history.pushState({}, '', '/backtest')
    window.dispatchEvent(new PopStateEvent('popstate'))
  }

  if (loading) {
    return (
      <div
        className="flex items-center justify-center h-screen"
        style={{ background: '#0B0E11' }}
      >
        <div className="text-center">
          <div
            className="inline-block animate-spin rounded-full h-12 w-12 border-t-2 border-b-2"
            style={{ borderColor: '#F0B90B' }}
          ></div>
          <div className="mt-4" style={{ color: '#848E9C' }}>
            加载回测数据中...
          </div>
        </div>
      </div>
    )
  }

  if (!backtest) {
    return (
      <div className="max-w-7xl mx-auto p-6">
        <div className="text-center py-16">
          <div className="text-6xl mb-4 opacity-30">📊</div>
          <p
            className="text-xl font-semibold mb-2"
            style={{ color: '#EAECEF' }}
          >
            回测数据未找到
          </p>
          <p className="mb-6" style={{ color: '#848E9C' }}>
            该回测可能已被删除或不存在
          </p>
          <button
            onClick={handleBack}
            className="px-6 py-3 rounded-lg font-semibold transition-all hover:scale-105"
            style={{
              background: '#F0B90B',
              color: '#000',
              boxShadow: '0 4px 14px rgba(240, 185, 11, 0.4)',
            }}
          >
            返回列表
          </button>
        </div>
      </div>
    )
  }

  // 计算统计信息
  const winningTrades = trades.filter((t) => t.pnl > 0).length
  const losingTrades = trades.filter((t) => t.pnl < 0).length
  const totalDecisions = decisions.reduce(
    (acc, record) => acc + (record.decisions?.length || 0),
    0
  )

  return (
    <div
      className="max-w-7xl mx-auto p-6 animate-fade-in"
      style={{ background: '#0B0E11', minHeight: '100vh' }}
    >
      {/* 页面头部 */}
      <div className="mb-6">
        <button
          onClick={handleBack}
          className="mb-4 px-4 py-2 rounded-lg flex items-center gap-2 transition-all hover:scale-105"
          style={{
            background: '#1E2329',
            border: '1px solid #2B3139',
            color: '#EAECEF',
          }}
        >
          <ArrowLeft className="w-4 h-4" />
          返回列表
        </button>
        <div
          className="p-6 rounded-xl"
          style={{
            background:
              'linear-gradient(135deg, rgba(240, 185, 11, 0.15) 0%, rgba(252, 213, 53, 0.05) 100%)',
            border: '1px solid rgba(240, 185, 11, 0.2)',
            boxShadow: '0 0 30px rgba(240, 185, 11, 0.15)',
          }}
        >
          <div className="flex items-center justify-between">
            <div>
              <h1
                className="text-2xl sm:text-3xl font-bold mb-2"
                style={{ color: '#EAECEF' }}
              >
                📊 回测详情报告
              </h1>
              <p className="text-sm sm:text-base" style={{ color: '#848E9C' }}>
                {new Date(backtest.start_time).toLocaleDateString()} -{' '}
                {new Date(backtest.end_time).toLocaleDateString()}
              </p>
              <div className="mt-2 text-xs text-gray-500 flex gap-3">
                {backtest.timeframe && <span>周期: {backtest.timeframe}</span>}
                {backtest.ai_model_id && <span>AI模型: {backtest.ai_model_id}</span>}
              </div>
            </div>
            <div className="hidden sm:flex items-center gap-3">
              {/* WebSocket 状态指示器 */}
              {isRunning && (
                <div
                  className="flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium"
                  style={{
                    background:
                      wsStatus === 'connected'
                        ? 'rgba(14, 203, 129, 0.1)'
                        : wsStatus === 'connecting'
                          ? 'rgba(240, 185, 11, 0.1)'
                          : 'rgba(246, 70, 93, 0.1)',
                    color:
                      wsStatus === 'connected'
                        ? '#0ECB81'
                        : wsStatus === 'connecting'
                          ? '#F0B90B'
                          : '#F6465D',
                    border: `1px solid ${
                      wsStatus === 'connected'
                        ? 'rgba(14, 203, 129, 0.2)'
                        : wsStatus === 'connecting'
                          ? 'rgba(240, 185, 11, 0.2)'
                          : 'rgba(246, 70, 93, 0.2)'
                    }`,
                  }}
                >
                  {wsStatus === 'connected' ? (
                    <Wifi className="w-3 h-3" />
                  ) : (
                    <WifiOff className="w-3 h-3" />
                  )}
                  {wsStatus === 'connected'
                    ? '实时推送'
                    : wsStatus === 'connecting'
                      ? '连接中...'
                      : '离线'}
                </div>
              )}
              <div
                className="px-4 py-2 rounded-lg text-sm font-bold"
                style={{
                  background:
                    backtest.status === 'completed'
                      ? 'rgba(14, 203, 129, 0.1)'
                      : 'rgba(240, 185, 11, 0.1)',
                  color:
                    backtest.status === 'completed' ? '#0ECB81' : '#F0B90B',
                  border: `1px solid ${backtest.status === 'completed' ? 'rgba(14, 203, 129, 0.2)' : 'rgba(240, 185, 11, 0.2)'}`,
                }}
              >
                {backtest.status === 'completed' ? '✓ 已完成' : '⏳ 进行中'}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 核心指标卡片 - Binance风格 */}
      <div
        className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4 mb-6 animate-slide-in"
        style={{ animationDelay: '0.1s' }}
      >
        {/* 总收益卡片 */}
        <div
          className="rounded-xl p-4 sm:p-5 relative overflow-hidden group hover:scale-105 transition-transform"
          style={{
            background:
              backtest.total_pnl_pct >= 0
                ? 'linear-gradient(135deg, rgba(14, 203, 129, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)'
                : 'linear-gradient(135deg, rgba(246, 70, 93, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)',
            border: `1px solid ${backtest.total_pnl_pct >= 0 ? 'rgba(14, 203, 129, 0.3)' : 'rgba(246, 70, 93, 0.3)'}`,
            boxShadow: `0 4px 16px ${backtest.total_pnl_pct >= 0 ? 'rgba(14, 203, 129, 0.2)' : 'rgba(246, 70, 93, 0.2)'}`,
          }}
        >
          <div
            className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-20"
            style={{
              background: `radial-gradient(circle, ${backtest.total_pnl_pct >= 0 ? '#0ECB81' : '#F6465D'} 0%, transparent 70%)`,
              filter: 'blur(20px)',
            }}
          />
          <div className="relative">
            <div className="flex items-center gap-2 mb-2">
              {backtest.total_pnl_pct >= 0 ? (
                <TrendingUp className="w-4 h-4" style={{ color: '#0ECB81' }} />
              ) : (
                <TrendingDown
                  className="w-4 h-4"
                  style={{ color: '#F6465D' }}
                />
              )}
              <div
                className="text-xs font-semibold uppercase tracking-wider"
                style={{ color: '#848E9C' }}
              >
                总收益
              </div>
            </div>
            <div
              className="text-2xl sm:text-3xl font-bold mono mb-1"
              style={{
                color:
                  (backtest.total_pnl_pct || 0) >= 0 ? '#0ECB81' : '#F6465D',
              }}
            >
              {(backtest.total_pnl_pct || 0) >= 0 ? '+' : ''}
              {(backtest.total_pnl_pct || 0).toFixed(2)}%
            </div>
            <div className="text-xs" style={{ color: '#848E9C' }}>
              {(backtest.total_pnl || 0) >= 0 ? '+' : ''}$
              {(backtest.total_pnl || 0).toFixed(2)} USDT
            </div>
          </div>
        </div>

        {/* 最大回撤卡片 */}
        <div
          className="rounded-xl p-4 sm:p-5 relative overflow-hidden group hover:scale-105 transition-transform"
          style={{
            background:
              'linear-gradient(135deg, rgba(246, 70, 93, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)',
            border: '1px solid rgba(246, 70, 93, 0.3)',
            boxShadow: '0 4px 16px rgba(246, 70, 93, 0.2)',
          }}
        >
          <div
            className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, #F6465D 0%, transparent 70%)',
              filter: 'blur(20px)',
            }}
          />
          <div className="relative">
            <div className="flex items-center gap-2 mb-2">
              <TrendingDown className="w-4 h-4" style={{ color: '#F6465D' }} />
              <div
                className="text-xs font-semibold uppercase tracking-wider"
                style={{ color: '#848E9C' }}
              >
                最大回撤
              </div>
            </div>
            <div
              className="text-2xl sm:text-3xl font-bold mono"
              style={{ color: '#F6465D' }}
            >
              {(backtest.max_drawdown || 0).toFixed(2)}%
            </div>
          </div>
        </div>

        {/* 夏普率卡片 */}
        <div
          className="rounded-xl p-4 sm:p-5 relative overflow-hidden group hover:scale-105 transition-transform"
          style={{
            background:
              'linear-gradient(135deg, rgba(99, 102, 241, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)',
            border: '1px solid rgba(99, 102, 241, 0.3)',
            boxShadow: '0 4px 16px rgba(99, 102, 241, 0.2)',
          }}
        >
          <div
            className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, #6366F1 0%, transparent 70%)',
              filter: 'blur(20px)',
            }}
          />
          <div className="relative">
            <div className="flex items-center gap-2 mb-2">
              <BarChart3 className="w-4 h-4" style={{ color: '#6366F1' }} />
              <div
                className="text-xs font-semibold uppercase tracking-wider"
                style={{ color: '#848E9C' }}
              >
                夏普率
              </div>
            </div>
            <div
              className="text-2xl sm:text-3xl font-bold mono"
              style={{ color: '#EAECEF' }}
            >
              {(backtest.sharpe_ratio || 0).toFixed(2)}
            </div>
          </div>
        </div>

        {/* 胜率卡片 */}
        <div
          className="rounded-xl p-4 sm:p-5 relative overflow-hidden group hover:scale-105 transition-transform"
          style={{
            background:
              'linear-gradient(135deg, rgba(240, 185, 11, 0.2) 0%, rgba(30, 35, 41, 0.8) 100%)',
            border: '1px solid rgba(240, 185, 11, 0.3)',
            boxShadow: '0 4px 16px rgba(240, 185, 11, 0.2)',
          }}
        >
          <div
            className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-20"
            style={{
              background:
                'radial-gradient(circle, #F0B90B 0%, transparent 70%)',
              filter: 'blur(20px)',
            }}
          />
          <div className="relative">
            <div className="flex items-center gap-2 mb-2">
              <Target className="w-4 h-4" style={{ color: '#F0B90B' }} />
              <div
                className="text-xs font-semibold uppercase tracking-wider"
                style={{ color: '#848E9C' }}
              >
                胜率
              </div>
            </div>
            <div
              className="text-2xl sm:text-3xl font-bold mono"
              style={{ color: '#F0B90B' }}
            >
              {(backtest.win_rate || 0).toFixed(1)}%
            </div>
            <div className="text-xs" style={{ color: '#848E9C' }}>
              {winningTrades}胜 / {losingTrades}负
            </div>
          </div>
        </div>
      </div>

      {/* 实时进度面板 - 仅在运行中显示 */}
      {isRunning && showLivePanel && (
        <div
          className="mb-6 animate-slide-in"
          style={{ animationDelay: '0.15s' }}
        >
          <div
            className="binance-card p-4 sm:p-5"
            style={{
              border: '1px solid rgba(240, 185, 11, 0.3)',
              boxShadow: '0 0 20px rgba(240, 185, 11, 0.1)',
            }}
          >
            {/* 进度条 */}
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-3">
                <Activity
                  className="w-5 h-5 animate-pulse"
                  style={{ color: '#F0B90B' }}
                />
                <span
                  className="text-sm font-semibold"
                  style={{ color: '#EAECEF' }}
                >
                  实时回测进度
                </span>
              </div>
              <div className="flex items-center gap-3">
                <span
                  className="text-sm mono font-bold"
                  style={{ color: '#F0B90B' }}
                >
                  Cycle {currentCycle} ({progress.toFixed(1)}%)
                </span>
                <button
                  onClick={() => setShowLivePanel(false)}
                  className="p-1 rounded hover:bg-gray-700 transition-colors"
                  title="隐藏面板"
                >
                  <ChevronUp className="w-4 h-4" style={{ color: '#848E9C' }} />
                </button>
              </div>
            </div>

            {/* 进度条可视化 */}
            <div
              className="w-full h-2 rounded-full mb-4"
              style={{ background: '#2B3139' }}
            >
              <div
                className="h-full rounded-full transition-all duration-300"
                style={{
                  width: `${Math.min(progress, 100)}%`,
                  background: 'linear-gradient(90deg, #F0B90B 0%, #FCD535 100%)',
                  boxShadow: '0 0 10px rgba(240, 185, 11, 0.5)',
                }}
              />
            </div>

            {/* 最近的 AI 决策 */}
            {wsDecisions.length > 0 && (
              <div className="mt-4">
                <div className="flex items-center gap-2 mb-3">
                  <Brain className="w-4 h-4" style={{ color: '#7C3AED' }} />
                  <span
                    className="text-xs font-semibold uppercase tracking-wider"
                    style={{ color: '#848E9C' }}
                  >
                    最新 AI 决策
                  </span>
                </div>

                {/* 显示最近 3 条决策 */}
                <div className="space-y-2">
                  {wsDecisions.slice(-3).reverse().map((decision: DecisionPayload) => (
                    <div
                      key={decision.cycle}
                      className="rounded-lg p-3"
                      style={{
                        background: 'rgba(124, 58, 237, 0.05)',
                        border: '1px solid rgba(124, 58, 237, 0.1)',
                      }}
                    >
                      <div className="flex items-center justify-between mb-2">
                        <span
                          className="text-xs font-mono"
                          style={{ color: '#848E9C' }}
                        >
                          Cycle {decision.cycle} •{' '}
                          {new Date(decision.timestamp).toLocaleTimeString('zh-CN')}
                        </span>
                        <button
                          onClick={() => {
                            const newSet = new Set(expandedDecisions)
                            if (newSet.has(decision.cycle)) {
                              newSet.delete(decision.cycle)
                            } else {
                              newSet.add(decision.cycle)
                            }
                            setExpandedDecisions(newSet)
                          }}
                          className="p-1 rounded hover:bg-gray-700 transition-colors"
                        >
                          {expandedDecisions.has(decision.cycle) ? (
                            <ChevronUp
                              className="w-4 h-4"
                              style={{ color: '#848E9C' }}
                            />
                          ) : (
                            <ChevronDown
                              className="w-4 h-4"
                              style={{ color: '#848E9C' }}
                            />
                          )}
                        </button>
                      </div>

                      {/* 决策摘要 */}
                      <div className="flex flex-wrap gap-2">
                        {decision.decisions.map((d, i) => (
                          <span
                            key={i}
                            className="px-2 py-1 rounded text-xs font-bold"
                            style={{
                              background:
                                d.action === 'buy' || d.action === 'long'
                                  ? 'rgba(14, 203, 129, 0.1)'
                                  : d.action === 'sell' || d.action === 'short'
                                    ? 'rgba(246, 70, 93, 0.1)'
                                    : 'rgba(132, 142, 156, 0.1)',
                              color:
                                d.action === 'buy' || d.action === 'long'
                                  ? '#0ECB81'
                                  : d.action === 'sell' || d.action === 'short'
                                    ? '#F6465D'
                                    : '#848E9C',
                              border: `1px solid ${
                                d.action === 'buy' || d.action === 'long'
                                  ? 'rgba(14, 203, 129, 0.2)'
                                  : d.action === 'sell' || d.action === 'short'
                                    ? 'rgba(246, 70, 93, 0.2)'
                                    : 'rgba(132, 142, 156, 0.2)'
                              }`,
                            }}
                          >
                            {d.symbol}: {d.action} ({d.confidence}%)
                          </span>
                        ))}
                      </div>

                      {/* 展开的 CoT 推理 */}
                      {expandedDecisions.has(decision.cycle) && decision.cot_trace && (
                        <div
                          className="mt-3 p-3 rounded text-xs"
                          style={{
                            background: '#0B0E11',
                            border: '1px solid #2B3139',
                            maxHeight: '200px',
                            overflowY: 'auto',
                          }}
                        >
                          <div
                            className="font-semibold mb-2"
                            style={{ color: '#7C3AED' }}
                          >
                            🧠 LLM 思维链 (截断)
                          </div>
                          <pre
                            className="whitespace-pre-wrap font-mono"
                            style={{ color: '#EAECEF' }}
                          >
                            {decision.cot_trace}
                          </pre>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 错误提示 */}
            {wsError && (
              <div
                className="mt-3 p-3 rounded text-sm"
                style={{
                  background: 'rgba(246, 70, 93, 0.1)',
                  border: '1px solid rgba(246, 70, 93, 0.2)',
                  color: '#F6465D',
                }}
              >
                ⚠️ {wsError}
              </div>
            )}

            {/* 状态信息 */}
            {progressMessage && (
              <div className="mt-2 text-xs" style={{ color: '#848E9C' }}>
                {progressMessage}
              </div>
            )}
          </div>
        </div>
      )}

      {/* 隐藏时的迷你按钮 */}
      {isRunning && !showLivePanel && (
        <div className="mb-4">
          <button
            onClick={() => setShowLivePanel(true)}
            className="flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium transition-all hover:scale-105"
            style={{
              background: 'rgba(240, 185, 11, 0.1)',
              border: '1px solid rgba(240, 185, 11, 0.2)',
              color: '#F0B90B',
            }}
          >
            <Activity className="w-4 h-4 animate-pulse" />
            显示实时进度 ({progress.toFixed(1)}%)
          </button>
        </div>
      )}

      {/* 净值曲线 - 使用EquityChart组件 */}
      <div className="mb-6 animate-slide-in" style={{ animationDelay: '0.2s' }}>
        <EquityChart
          backtestId={backtestId}
          initialBalance={backtest.initial_balance}
          isBacktest={true}
          externalData={isRunning && externalEquityData.length > 0 ? externalEquityData : undefined}
          playheadCycle={isRunning ? currentCycle : undefined}
          showPlayhead={isRunning && wsStatus === 'connected'}
        />
      </div>

      {/* 交易记录 - Binance风格优化 */}
      <div
        className="binance-card p-4 sm:p-6 animate-slide-in"
        style={{ animationDelay: '0.3s' }}
      >
        <div
          className="flex items-center justify-between mb-5 pb-4"
          style={{ borderBottom: '1px solid #2B3139' }}
        >
          <div className="flex items-center gap-3">
            <div
              className="w-10 h-10 rounded-xl flex items-center justify-center text-xl"
              style={{
                background: 'linear-gradient(135deg, #F0B90B 0%, #FCD535 100%)',
                boxShadow: '0 4px 14px rgba(240, 185, 11, 0.4)',
              }}
            >
              📊
            </div>
            <div>
              <h2
                className="text-lg sm:text-xl font-bold"
                style={{ color: '#EAECEF' }}
              >
                交易记录
              </h2>
              <div className="text-xs sm:text-sm" style={{ color: '#848E9C' }}>
                共 {backtest.total_trades} 笔交易
              </div>
            </div>
          </div>
          {backtest.total_trades > 0 && (
            <div
              className="px-3 py-1 rounded text-xs font-bold"
              style={{
                background: 'rgba(240, 185, 11, 0.1)',
                color: '#F0B90B',
                border: '1px solid rgba(240, 185, 11, 0.2)',
              }}
            >
              {backtest.total_trades} 笔
            </div>
          )}
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: '1px solid #2B3139' }}>
                <th
                  className="text-left py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  币种
                </th>
                <th
                  className="text-left py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  方向
                </th>
                <th
                  className="text-right py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  入场价
                </th>
                <th
                  className="text-right py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  出场价
                </th>
                <th
                  className="text-right py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  数量
                </th>
                <th
                  className="text-right py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  杠杆
                </th>
                <th
                  className="text-right py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  盈亏
                </th>
                <th
                  className="text-right py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  手续费
                </th>
                <th
                  className="text-left py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  时间
                </th>
              </tr>
            </thead>
            <tbody>
              {trades.map((trade, index) => (
                <tr
                  key={trade.id}
                  className="transition-all hover:scale-[1.01]"
                  style={{
                    borderBottom: '1px solid #2B3139',
                    background:
                      index % 2 === 0
                        ? 'transparent'
                        : 'rgba(240, 185, 11, 0.02)',
                  }}
                >
                  <td
                    className="py-3 px-2 sm:px-4 font-mono font-bold"
                    style={{ color: '#EAECEF' }}
                  >
                    {trade.symbol}
                  </td>
                  <td className="py-3 px-2 sm:px-4">
                    <span
                      className="px-2 py-1 rounded text-xs font-bold"
                      style={{
                        background:
                          trade.side === 'long'
                            ? 'rgba(14, 203, 129, 0.1)'
                            : 'rgba(246, 70, 93, 0.1)',
                        color: trade.side === 'long' ? '#0ECB81' : '#F6465D',
                        border: `1px solid ${trade.side === 'long' ? 'rgba(14, 203, 129, 0.2)' : 'rgba(246, 70, 93, 0.2)'}`,
                      }}
                    >
                      {trade.side === 'long' ? '做多' : '做空'}
                    </span>
                  </td>
                  <td
                    className="py-3 px-2 sm:px-4 text-right font-mono"
                    style={{ color: '#EAECEF' }}
                  >
                    ${(trade.entry_price || 0).toFixed(4)}
                  </td>
                  <td
                    className="py-3 px-2 sm:px-4 text-right font-mono"
                    style={{ color: '#EAECEF' }}
                  >
                    ${(trade.exit_price || 0).toFixed(4)}
                  </td>
                  <td
                    className="py-3 px-2 sm:px-4 text-right font-mono"
                    style={{ color: '#848E9C' }}
                  >
                    {(trade.quantity || 0).toFixed(4)}
                  </td>
                  <td
                    className="py-3 px-2 sm:px-4 text-right font-mono font-bold"
                    style={{ color: '#F0B90B' }}
                  >
                    {trade.leverage}x
                  </td>
                  <td className="py-3 px-2 sm:px-4 text-right">
                    <div
                      className="font-mono font-bold"
                      style={{
                        color: (trade.pnl || 0) >= 0 ? '#0ECB81' : '#F6465D',
                      }}
                    >
                      {(trade.pnl || 0) >= 0 ? '+' : ''}$
                      {(trade.pnl || 0).toFixed(2)}
                    </div>
                    <div
                      className="text-xs font-mono"
                      style={{
                        color: (trade.pnl || 0) >= 0 ? '#0ECB81' : '#F6465D',
                        opacity: 0.7,
                      }}
                    >
                      ({(trade.pnl_pct || 0) >= 0 ? '+' : ''}
                      {(trade.pnl_pct || 0).toFixed(2)}%)
                    </div>
                  </td>
                  <td
                    className="py-3 px-2 sm:px-4 text-right font-mono"
                    style={{ color: '#848E9C' }}
                  >
                    ${(trade.fee || 0).toFixed(2)}
                  </td>
                  <td
                    className="py-3 px-2 sm:px-4 font-mono text-xs"
                    style={{ color: '#848E9C' }}
                  >
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
          <div className="text-center py-16">
            <div className="text-6xl mb-4 opacity-30">📈</div>
            <div
              className="text-lg font-semibold mb-2"
              style={{ color: '#EAECEF' }}
            >
              暂无交易记录
            </div>
            <div className="text-sm" style={{ color: '#848E9C' }}>
              该回测还未产生任何交易
            </div>
          </div>
        )}
      </div>

      {/* 决策记录 */}
      <div
        className="binance-card p-4 sm:p-6 animate-slide-in mt-6"
        style={{ animationDelay: '0.4s' }}
      >
        <div
          className="flex items-center justify-between mb-5 pb-4"
          style={{ borderBottom: '1px solid #2B3139' }}
        >
          <div className="flex items-center gap-3">
            <div
              className="w-10 h-10 rounded-xl flex items-center justify-center text-xl"
              style={{
                background: 'linear-gradient(135deg, #7C3AED 0%, #A78BFA 100%)',
                boxShadow: '0 4px 14px rgba(124, 58, 237, 0.4)',
              }}
            >
              🧠
            </div>
            <div>
              <h2
                className="text-lg sm:text-xl font-bold"
                style={{ color: '#EAECEF' }}
              >
                AI 决策记录
              </h2>
              <div className="text-xs sm:text-sm" style={{ color: '#848E9C' }}>
                共 {totalDecisions} 条决策
              </div>
            </div>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: '1px solid #2B3139' }}>
                <th
                  className="text-left py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  时间
                </th>
                <th
                  className="text-left py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  币种
                </th>
                <th
                  className="text-left py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  动作
                </th>
                <th
                  className="text-right py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  置信度
                </th>
                <th
                  className="text-right py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  价格
                </th>
                <th
                  className="text-left py-3 px-2 sm:px-4 font-semibold"
                  style={{ color: '#848E9C' }}
                >
                  理由
                </th>
              </tr>
            </thead>
            <tbody>
              {decisions.flatMap((record, recordIndex) =>
                (record.decisions || []).map((action, actionIndex) => (
                  <tr
                    key={`${recordIndex}-${actionIndex}`}
                    className="transition-all hover:scale-[1.01]"
                    style={{
                      borderBottom: '1px solid #2B3139',
                      background:
                        recordIndex % 2 === 0
                          ? 'transparent'
                          : 'rgba(124, 58, 237, 0.02)',
                    }}
                  >
                    <td
                      className="py-3 px-2 sm:px-4 font-mono text-xs"
                      style={{ color: '#848E9C' }}
                    >
                      {new Date(record.timestamp).toLocaleString('zh-CN', {
                        month: 'short',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </td>
                    <td
                      className="py-3 px-2 sm:px-4 font-mono font-bold"
                      style={{ color: '#EAECEF' }}
                    >
                      {action.symbol}
                    </td>
                    <td className="py-3 px-2 sm:px-4">
                      <span
                        className="px-2 py-1 rounded text-xs font-bold"
                        style={{
                          background:
                            action.action === 'buy' || action.action === 'long'
                              ? 'rgba(14, 203, 129, 0.1)'
                              : action.action === 'sell' ||
                                  action.action === 'short'
                                ? 'rgba(246, 70, 93, 0.1)'
                                : 'rgba(132, 142, 156, 0.1)',
                          color:
                            action.action === 'buy' || action.action === 'long'
                              ? '#0ECB81'
                              : action.action === 'sell' ||
                                  action.action === 'short'
                                ? '#F6465D'
                                : '#848E9C',
                          border: `1px solid ${
                            action.action === 'buy' || action.action === 'long'
                              ? 'rgba(14, 203, 129, 0.2)'
                              : action.action === 'sell' ||
                                  action.action === 'short'
                                ? 'rgba(246, 70, 93, 0.2)'
                                : 'rgba(132, 142, 156, 0.2)'
                          }`,
                        }}
                      >
                        {action.action === 'buy'
                          ? '买入'
                          : action.action === 'sell'
                            ? '卖出'
                            : action.action === 'long'
                              ? '做多'
                              : action.action === 'short'
                                ? '做空'
                                : action.action === 'hold'
                                  ? '持有'
                                  : action.action}
                      </span>
                    </td>
                    <td
                      className="py-3 px-2 sm:px-4 text-right font-mono"
                      style={{ color: '#EAECEF' }}
                    >
                      {action.confidence}%
                    </td>
                    <td
                      className="py-3 px-2 sm:px-4 text-right font-mono"
                      style={{ color: '#EAECEF' }}
                    >
                      ${(action.price || 0).toFixed(4)}
                    </td>
                    <td
                      className="py-3 px-2 sm:px-4 text-xs"
                      style={{ color: '#EAECEF', maxWidth: '300px' }}
                    >
                      {action.reasoning}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {decisions.length === 0 && (
          <div className="text-center py-16">
            <div className="text-6xl mb-4 opacity-30">🧠</div>
            <div
              className="text-lg font-semibold mb-2"
              style={{ color: '#EAECEF' }}
            >
              暂无决策记录
            </div>
            <div className="text-sm" style={{ color: '#848E9C' }}>
              该回测未记录任何决策
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

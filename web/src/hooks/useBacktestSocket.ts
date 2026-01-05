/**
 * useBacktestSocket Hook
 *
 * React Hook 封装，用于在组件中使用回测 WebSocket 连接
 */

import { useEffect, useRef, useState, useCallback } from 'react'
import {
  BacktestSocket,
  type BacktestSocketCallbacks,
  type BacktestSocketOptions,
  type ConnectionStatus,
  type EquitySnapshotPayload,
  type DecisionPayload,
  type ProgressPayload,
  type CompletePayload,
} from '../lib/backtestSocket'

export interface UseBacktestSocketOptions extends BacktestSocketOptions {
  /**
   * 是否自动连接（默认 true）
   */
  autoConnect?: boolean
  /**
   * 是否启用（用于条件连接）
   */
  enabled?: boolean
}

export interface UseBacktestSocketReturn {
  /**
   * 当前连接状态
   */
  status: ConnectionStatus
  /**
   * 当前进度百分比
   */
  progress: number
  /**
   * 实时净值快照列表
   */
  snapshots: EquitySnapshotPayload[]
  /**
   * 实时决策列表
   */
  decisions: DecisionPayload[]
  /**
   * 当前周期
   */
  currentCycle: number
  /**
   * 最新的进度消息
   */
  progressMessage: string
  /**
   * 是否已完成
   */
  isComplete: boolean
  /**
   * 完成时的结果数据
   */
  completeData: CompletePayload | null
  /**
   * 错误信息
   */
  error: string | null
  /**
   * 手动连接
   */
  connect: () => void
  /**
   * 手动断开
   */
  disconnect: () => void
  /**
   * 清空累积数据
   */
  clearData: () => void
}

/**
 * 回测 WebSocket Hook
 *
 * @example
 * ```tsx
 * function BacktestDetail({ backtestId, isRunning }) {
 *   const {
 *     status,
 *     progress,
 *     snapshots,
 *     decisions,
 *     isComplete,
 *   } = useBacktestSocket(backtestId, {
 *     enabled: isRunning,
 *     debug: true,
 *   })
 *
 *   return (
 *     <div>
 *       <div>状态: {status}</div>
 *       <div>进度: {progress}%</div>
 *       <EquityChart externalData={snapshots} />
 *     </div>
 *   )
 * }
 * ```
 */
export function useBacktestSocket(
  backtestId: string,
  options: UseBacktestSocketOptions = {}
): UseBacktestSocketReturn {
  const { autoConnect = true, enabled = true, ...socketOptions } = options

  // 状态
  const [status, setStatus] = useState<ConnectionStatus>('disconnected')
  const [progress, setProgress] = useState(0)
  const [snapshots, setSnapshots] = useState<EquitySnapshotPayload[]>([])
  const [decisions, setDecisions] = useState<DecisionPayload[]>([])
  const [currentCycle, setCurrentCycle] = useState(0)
  const [progressMessage, setProgressMessage] = useState('')
  const [isComplete, setIsComplete] = useState(false)
  const [completeData, setCompleteData] = useState<CompletePayload | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Socket 实例 ref
  const socketRef = useRef<BacktestSocket | null>(null)
  const backtestIdRef = useRef(backtestId)

  // 回调函数
  const callbacks: BacktestSocketCallbacks = {
    onOpen: useCallback(() => {
      setStatus('connected')
      setError(null)
    }, []),

    onProgress: useCallback((payload: ProgressPayload) => {
      setProgress(payload.progress_pct)
      setCurrentCycle(payload.cycle)
      if (payload.message) {
        setProgressMessage(payload.message)
      }
    }, []),

    onEquitySnapshot: useCallback((payload: EquitySnapshotPayload) => {
      setSnapshots((prev) => {
        // 去重：根据 cycle 判断
        const exists = prev.some((s) => s.cycle === payload.cycle)
        if (exists) return prev
        return [...prev, payload]
      })
      setCurrentCycle(payload.cycle)
    }, []),

    onDecision: useCallback((payload: DecisionPayload) => {
      setDecisions((prev) => {
        // 去重：根据 cycle 判断
        const exists = prev.some((d) => d.cycle === payload.cycle)
        if (exists) return prev
        return [...prev, payload]
      })
    }, []),

    onComplete: useCallback((payload: CompletePayload) => {
      setIsComplete(true)
      setCompleteData(payload)
      setProgress(100)
      setStatus('disconnected')
    }, []),

    onError: useCallback((payload: { error: string; message: string }) => {
      setError(payload.message || payload.error)
    }, []),

    onClose: useCallback(() => {
      setStatus('disconnected')
    }, []),

    onReconnect: useCallback((attempt: number) => {
      setStatus('connecting')
      setProgressMessage(`重连中 (${attempt})...`)
    }, []),
  }

  // 连接函数
  const connect = useCallback(() => {
    if (socketRef.current) {
      socketRef.current.disconnect()
    }

    socketRef.current = new BacktestSocket(
      backtestIdRef.current,
      callbacks,
      socketOptions
    )
    socketRef.current.connect()
    setStatus('connecting')
  }, [callbacks, socketOptions])

  // 断开函数
  const disconnect = useCallback(() => {
    socketRef.current?.disconnect()
    socketRef.current = null
  }, [])

  // 清空数据
  const clearData = useCallback(() => {
    setSnapshots([])
    setDecisions([])
    setProgress(0)
    setCurrentCycle(0)
    setProgressMessage('')
    setIsComplete(false)
    setCompleteData(null)
    setError(null)
  }, [])

  // backtestId 变化时重置
  useEffect(() => {
    if (backtestId !== backtestIdRef.current) {
      backtestIdRef.current = backtestId
      clearData()
      disconnect()
    }
  }, [backtestId, clearData, disconnect])

  // 自动连接/断开
  useEffect(() => {
    if (enabled && autoConnect && backtestId) {
      connect()
    }

    return () => {
      disconnect()
    }
  }, [enabled, autoConnect, backtestId, connect, disconnect])

  return {
    status,
    progress,
    snapshots,
    decisions,
    currentCycle,
    progressMessage,
    isComplete,
    completeData,
    error,
    connect,
    disconnect,
    clearData,
  }
}

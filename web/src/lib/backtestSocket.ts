/**
 * Backtest WebSocket Helper
 *
 * 提供回测实时进度推送的 WebSocket 连接管理
 * 支持自动重连、心跳检测、认证 token 处理
 */

// WebSocket 事件类型
export type ProgressEventType =
  | 'progress'
  | 'equity_snapshot'
  | 'decision'
  | 'complete'
  | 'error'

// 进度事件 payload
export interface ProgressPayload {
  cycle: number
  progress_pct: number
  message?: string
}

// 净值快照 payload
export interface EquitySnapshotPayload {
  cycle: number
  timestamp: string
  total_equity: number
  pnl: number
  pnl_pct: number
}

// 决策详情
export interface DecisionItemDetail {
  symbol: string
  action: string
  price: number
  confidence: number
  reasoning: string
}

// 决策事件 payload
export interface DecisionPayload {
  cycle: number
  timestamp: string
  cot_trace?: string // 截断到约2000字符
  decisions: DecisionItemDetail[]
}

// 完成事件 payload
export interface CompletePayload {
  final_equity: number
  total_pnl: number
  total_pnl_pct: number
  max_drawdown: number
  sharpe_ratio: number
  win_rate: number
  total_trades: number
  final_snapshot?: EquitySnapshotPayload
}

// 错误事件 payload
export interface ErrorPayload {
  error: string
  message: string
}

// WebSocket 消息 envelope
export interface BacktestWSMessage<T = unknown> {
  type: ProgressEventType
  backtest_id: string
  timestamp: string
  payload: T
}

// 事件回调类型
export interface BacktestSocketCallbacks {
  onProgress?: (payload: ProgressPayload) => void
  onEquitySnapshot?: (payload: EquitySnapshotPayload) => void
  onDecision?: (payload: DecisionPayload) => void
  onComplete?: (payload: CompletePayload) => void
  onError?: (payload: ErrorPayload) => void
  onOpen?: () => void
  onClose?: (event: CloseEvent) => void
  onReconnect?: (attempt: number) => void
}

// 连接状态
export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

// 配置选项
export interface BacktestSocketOptions {
  // 自动重连相关
  autoReconnect?: boolean
  maxReconnectAttempts?: number
  reconnectInterval?: number // 基础重连间隔（毫秒）
  reconnectBackoff?: number // 重连间隔增长倍数

  // 心跳相关
  heartbeatInterval?: number // 心跳间隔（毫秒）
  heartbeatTimeout?: number // 心跳超时（毫秒）

  // 其他
  debug?: boolean
}

const DEFAULT_OPTIONS: Required<BacktestSocketOptions> = {
  autoReconnect: true,
  maxReconnectAttempts: 5,
  reconnectInterval: 1000,
  reconnectBackoff: 1.5,
  heartbeatInterval: 30000,
  heartbeatTimeout: 10000,
  debug: false,
}

/**
 * BacktestSocket 类
 *
 * 管理单个回测的 WebSocket 连接
 *
 * @example
 * ```tsx
 * const socket = new BacktestSocket(backtestId, {
 *   onProgress: (p) => console.log('进度:', p.progress_pct),
 *   onEquitySnapshot: (s) => setSnapshots(prev => [...prev, s]),
 *   onComplete: (c) => console.log('完成:', c.final_equity),
 * })
 * socket.connect()
 *
 * // 组件卸载时
 * socket.disconnect()
 * ```
 */
export class BacktestSocket {
  private backtestId: string
  private callbacks: BacktestSocketCallbacks
  private options: Required<BacktestSocketOptions>

  private ws: WebSocket | null = null
  private status: ConnectionStatus = 'disconnected'
  private reconnectAttempts = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private heartbeatTimeoutTimer: ReturnType<typeof setTimeout> | null = null
  private manualClose = false

  constructor(
    backtestId: string,
    callbacks: BacktestSocketCallbacks = {},
    options: BacktestSocketOptions = {}
  ) {
    this.backtestId = backtestId
    this.callbacks = callbacks
    this.options = { ...DEFAULT_OPTIONS, ...options }
  }

  /**
   * 获取当前连接状态
   */
  getStatus(): ConnectionStatus {
    return this.status
  }

  /**
   * 获取 WebSocket URL
   */
  private getWsUrl(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host

    // 开发环境下，如果当前页面不是在 8080 端口（后端端口），则尝试连接到 8080
    const wsHost =
      import.meta.env.DEV && window.location.port !== '8080'
        ? window.location.hostname + ':8080'
        : host

    const token = localStorage.getItem('auth_token') || ''
    const tokenParam = token ? `?token=${encodeURIComponent(token)}` : ''

    return `${protocol}//${wsHost}/ws/backtest/${this.backtestId}/progress${tokenParam}`
  }

  /**
   * 连接 WebSocket
   */
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.log('WebSocket 已连接，跳过')
      return
    }

    this.manualClose = false
    this.status = 'connecting'

    try {
      const url = this.getWsUrl()
      this.log('连接 WebSocket:', url)
      this.ws = new WebSocket(url)

      this.ws.onopen = this.handleOpen.bind(this)
      this.ws.onmessage = this.handleMessage.bind(this)
      this.ws.onclose = this.handleClose.bind(this)
      this.ws.onerror = this.handleError.bind(this)
    } catch (err) {
      this.log('WebSocket 创建失败:', err)
      this.status = 'error'
      this.scheduleReconnect()
    }
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    this.manualClose = true
    this.clearTimers()

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }

    this.status = 'disconnected'
    this.reconnectAttempts = 0
  }

  /**
   * 发送心跳
   */
  private sendHeartbeat(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'ping' }))
      this.log('发送心跳 ping')

      // 设置心跳超时
      this.heartbeatTimeoutTimer = setTimeout(() => {
        this.log('心跳超时，重新连接')
        this.ws?.close()
      }, this.options.heartbeatTimeout)
    }
  }

  /**
   * 开始心跳
   */
  private startHeartbeat(): void {
    this.stopHeartbeat()
    this.heartbeatTimer = setInterval(() => {
      this.sendHeartbeat()
    }, this.options.heartbeatInterval)
  }

  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer)
      this.heartbeatTimeoutTimer = null
    }
  }

  /**
   * 处理连接打开
   */
  private handleOpen(): void {
    this.log('WebSocket 连接成功')
    this.status = 'connected'
    this.reconnectAttempts = 0
    this.startHeartbeat()
    this.callbacks.onOpen?.()
  }

  /**
   * 处理收到消息
   */
  private handleMessage(event: MessageEvent): void {
    try {
      const message = JSON.parse(event.data) as BacktestWSMessage

      // 处理 pong 响应
      if ((message as any).type === 'pong') {
        if (this.heartbeatTimeoutTimer) {
          clearTimeout(this.heartbeatTimeoutTimer)
          this.heartbeatTimeoutTimer = null
        }
        this.log('收到心跳 pong')
        return
      }

      this.log('收到消息:', message.type, message.payload)

      // 分发事件
      switch (message.type) {
        case 'progress':
          this.callbacks.onProgress?.(message.payload as ProgressPayload)
          break
        case 'equity_snapshot':
          this.callbacks.onEquitySnapshot?.(message.payload as EquitySnapshotPayload)
          break
        case 'decision':
          this.callbacks.onDecision?.(message.payload as DecisionPayload)
          break
        case 'complete':
          this.callbacks.onComplete?.(message.payload as CompletePayload)
          // 完成后自动断开，不需要重连
          this.manualClose = true
          break
        case 'error':
          this.callbacks.onError?.(message.payload as ErrorPayload)
          break
        default:
          this.log('未知消息类型:', message.type)
      }
    } catch (err) {
      this.log('消息解析失败:', err, event.data)
    }
  }

  /**
   * 处理连接关闭
   */
  private handleClose(event: CloseEvent): void {
    this.log('WebSocket 关闭:', event.code, event.reason)
    this.status = 'disconnected'
    this.stopHeartbeat()
    this.callbacks.onClose?.(event)

    if (!this.manualClose && this.options.autoReconnect) {
      this.scheduleReconnect()
    }
  }

  /**
   * 处理连接错误
   */
  private handleError(event: Event): void {
    this.log('WebSocket 错误:', event)
    this.status = 'error'
  }

  /**
   * 安排重连
   */
  private scheduleReconnect(): void {
    if (this.manualClose) return
    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      this.log('达到最大重连次数，停止重连')
      return
    }

    const delay =
      this.options.reconnectInterval *
      Math.pow(this.options.reconnectBackoff, this.reconnectAttempts)

    this.log(`${delay}ms 后重连 (第 ${this.reconnectAttempts + 1} 次)`)

    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++
      this.callbacks.onReconnect?.(this.reconnectAttempts)
      this.connect()
    }, delay)
  }

  /**
   * 清理所有定时器
   */
  private clearTimers(): void {
    this.stopHeartbeat()
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  /**
   * 日志输出
   */
  private log(...args: unknown[]): void {
    if (this.options.debug) {
      console.log(`[BacktestSocket:${this.backtestId}]`, ...args)
    }
  }
}

/**
 * 创建回测 WebSocket 连接的工厂函数
 *
 * @example
 * ```tsx
 * const socket = createBacktestSocket(backtestId, {
 *   onEquitySnapshot: (s) => setSnapshots(prev => [...prev, s]),
 * })
 * ```
 */
export function createBacktestSocket(
  backtestId: string,
  callbacks: BacktestSocketCallbacks = {},
  options: BacktestSocketOptions = {}
): BacktestSocket {
  return new BacktestSocket(backtestId, callbacks, options)
}

/**
 * 获取回测 WebSocket URL（用于调试）
 */
export function getBacktestWsUrl(backtestId: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const wsHost =
    import.meta.env.DEV && window.location.port === '5173'
      ? window.location.hostname + ':8080'
      : host
  return `${protocol}//${wsHost}/ws/backtest/${backtestId}/progress`
}

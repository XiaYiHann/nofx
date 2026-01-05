import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  BacktestSocket,
  createBacktestSocket,
  getBacktestWsUrl,
  type ProgressPayload,
  type EquitySnapshotPayload,
  type DecisionPayload,
  type CompletePayload,
  type BacktestWSMessage,
} from './backtestSocket'

// Mock WebSocket
class MockWebSocket {
  static instances: MockWebSocket[] = []
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState: number = MockWebSocket.OPEN
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null

  constructor(public url: string) {
    MockWebSocket.instances.push(this)
    // 立即触发 onopen（使用 queueMicrotask 确保 handlers 已绑定）
    queueMicrotask(() => {
      if (this.onopen) {
        this.onopen(new Event('open'))
      }
    })
  }

  send(_data: string): void {
    // Mock send
  }

  close(code?: number, reason?: string): void {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close', { code: code || 1000, reason: reason || '' }))
    }
  }

  simulateMessage(data: unknown): void {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', {
        data: JSON.stringify(data),
      }))
    }
  }

  static clear(): void {
    MockWebSocket.instances = []
  }

  static getLatest(): MockWebSocket | null {
    return MockWebSocket.instances[MockWebSocket.instances.length - 1] || null
  }
}

describe('BacktestSocket', () => {
  beforeEach(() => {
    MockWebSocket.clear()
    vi.stubGlobal('WebSocket', MockWebSocket)

    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: vi.fn().mockReturnValue('test-token'),
        setItem: vi.fn(),
        removeItem: vi.fn(),
      },
      writable: true,
    })

    Object.defineProperty(window, 'location', {
      value: {
        protocol: 'http:',
        host: 'localhost:8080',
        hostname: 'localhost',
        port: '8080',
      },
      writable: true,
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    MockWebSocket.clear()
  })

  describe('getBacktestWsUrl', () => {
    it('should generate correct WebSocket URL', () => {
      const url = getBacktestWsUrl('test-123')
      expect(url).toContain('ws://')
      expect(url).toContain('/ws/backtest/test-123/progress')
    })
  })

  describe('createBacktestSocket', () => {
    it('should create a BacktestSocket instance', () => {
      const socket = createBacktestSocket('test-123')
      expect(socket).toBeInstanceOf(BacktestSocket)
    })
  })

  describe('BacktestSocket class', () => {
    it('should start with disconnected status', () => {
      const socket = new BacktestSocket('test-123')
      expect(socket.getStatus()).toBe('disconnected')
    })

    it('should connect and call onOpen callback', async () => {
      const onOpen = vi.fn()
      const socket = new BacktestSocket('test-123', { onOpen })

      socket.connect()
      await new Promise(resolve => setTimeout(resolve, 5))

      expect(socket.getStatus()).toBe('connected')
      expect(onOpen).toHaveBeenCalled()

      socket.disconnect()
    })

    it('should handle progress events', async () => {
      const onProgress = vi.fn()
      const socket = new BacktestSocket('test-123', { onProgress })

      socket.connect()
      await new Promise(resolve => setTimeout(resolve, 5))

      const mockWs = MockWebSocket.getLatest()
      const progressMessage: BacktestWSMessage<ProgressPayload> = {
        type: 'progress',
        backtest_id: 'test-123',
        timestamp: new Date().toISOString(),
        payload: {
          cycle: 10,
          progress_pct: 50.0,
          message: 'Processing...',
        },
      }

      mockWs?.simulateMessage(progressMessage)

      expect(onProgress).toHaveBeenCalledWith(progressMessage.payload)

      socket.disconnect()
    })

    it('should handle equity_snapshot events', async () => {
      const onEquitySnapshot = vi.fn()
      const socket = new BacktestSocket('test-123', { onEquitySnapshot })

      socket.connect()
      await new Promise(resolve => setTimeout(resolve, 5))

      const mockWs = MockWebSocket.getLatest()
      const snapshotMessage: BacktestWSMessage<EquitySnapshotPayload> = {
        type: 'equity_snapshot',
        backtest_id: 'test-123',
        timestamp: new Date().toISOString(),
        payload: {
          cycle: 10,
          time: new Date().toISOString(),
          total_equity: 10500,
          pnl: 500,
          pnl_pct: 5.0,
        },
      }

      mockWs?.simulateMessage(snapshotMessage)

      expect(onEquitySnapshot).toHaveBeenCalledWith(snapshotMessage.payload)

      socket.disconnect()
    })

    it('should handle decision events', async () => {
      const onDecision = vi.fn()
      const socket = new BacktestSocket('test-123', { onDecision })

      socket.connect()
      await new Promise(resolve => setTimeout(resolve, 5))

      const mockWs = MockWebSocket.getLatest()
      const decisionMessage: BacktestWSMessage<DecisionPayload> = {
        type: 'decision',
        backtest_id: 'test-123',
        timestamp: new Date().toISOString(),
        payload: {
          cycle: 10,
          time: new Date().toISOString(),
          cot_trace: 'Analyzing market conditions...',
          decisions: [
            {
              symbol: 'BTCUSDT',
              action: 'long',
              price: 50000,
              confidence: 85,
              reasoning: 'Strong bullish trend',
            },
          ],
        },
      }

      mockWs?.simulateMessage(decisionMessage)

      expect(onDecision).toHaveBeenCalledWith(decisionMessage.payload)

      socket.disconnect()
    })

    it('should handle complete events and set manualClose', async () => {
      const onComplete = vi.fn()
      const socket = new BacktestSocket('test-123', { onComplete })

      socket.connect()
      await new Promise(resolve => setTimeout(resolve, 5))

      const mockWs = MockWebSocket.getLatest()
      const completeMessage: BacktestWSMessage<CompletePayload> = {
        type: 'complete',
        backtest_id: 'test-123',
        timestamp: new Date().toISOString(),
        payload: {
          final_equity: 12000,
          total_pnl: 2000,
          total_pnl_pct: 20.0,
          max_drawdown: 5.0,
          sharpe_ratio: 1.5,
          win_rate: 60.0,
          total_trades: 50,
        },
      }

      mockWs?.simulateMessage(completeMessage)

      expect(onComplete).toHaveBeenCalledWith(completeMessage.payload)
    })

    it('should disconnect properly', async () => {
      const onClose = vi.fn()
      const socket = new BacktestSocket('test-123', { onClose })

      socket.connect()
      await new Promise(resolve => setTimeout(resolve, 5))

      socket.disconnect()

      expect(socket.getStatus()).toBe('disconnected')
    })

    it('should not reconnect after manual disconnect', async () => {
      const onReconnect = vi.fn()
      const socket = new BacktestSocket(
        'test-123',
        { onReconnect },
        { autoReconnect: true }
      )

      socket.connect()
      await new Promise(resolve => setTimeout(resolve, 5))

      socket.disconnect()
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(onReconnect).not.toHaveBeenCalled()
    })
  })
})

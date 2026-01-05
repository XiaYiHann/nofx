/**
 * 测试工具函数库
 * 提供统一的测试辅助函数，确保测试隔离和一致性
 */

import { render, RenderOptions } from '@testing-library/react'
import { ReactElement } from 'react'
import React from 'react'
import { vi } from 'vitest'
import { LanguageProvider } from '../contexts/LanguageContext'
import { AuthProvider } from '../contexts/AuthContext'
import type { Backtest, BacktestTrade, DecisionRecord } from '../types'

// ============================================================================
// Mock重置函数
// ============================================================================

/**
 * 重置所有mock，确保测试隔离
 * 在每个测试的beforeEach钩子中调用
 */
export function resetAllMocks(): void {
  vi.clearAllMocks()
  vi.resetAllMocks()
}

// ============================================================================
// 测试数据生成器
// ============================================================================

/**
 * 创建默认的mock回测数据
 */
export function createMockBacktest(overrides?: Partial<Backtest>): Backtest {
  return {
    id: 'backtest-1',
    trader_id: 'trader-1',
    user_id: 'user-1',
    start_time: '2024-01-01T00:00:00Z',
    end_time: '2024-01-02T00:00:00Z',
    initial_balance: 10000,
    status: 'completed',
    progress: 100,
    total_pnl: 500,
    total_pnl_pct: 5.0,
    max_drawdown: 2.5,
    sharpe_ratio: 1.8,
    win_rate: 60.0,
    total_trades: 10,
    timeframe: '3m',
    ai_model_id: 'model-1',
    exchange_id: 'exchange-1',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
    ...overrides,
  }
}

/**
 * 创建mock交易记录
 */
export function createMockTrade(overrides?: Partial<BacktestTrade>): BacktestTrade {
  return {
    id: 'trade-1',
    backtest_id: 'backtest-1',
    symbol: 'BTCUSDT',
    side: 'long',
    action: 'close',
    entry_price: 50000,
    exit_price: 51000,
    quantity: 0.1,
    leverage: 5,
    pnl: 100,
    pnl_pct: 2.0,
    fee: 5,
    entry_time: '2024-01-01T01:00:00Z',
    exit_time: '2024-01-01T02:00:00Z',
    ...overrides,
  }
}

/**
 * 创建mock决策记录
 */
export function createMockDecision(overrides?: Partial<DecisionRecord>): DecisionRecord {
  return {
    timestamp: '2024-01-01T01:30:00Z',
    decisions: [
      {
        action: 'open_long',
        symbol: 'BTCUSDT',
        confidence: 80,
        reasoning: 'Test reasoning',
      },
    ],
    ...overrides,
  }
}

/**
 * 创建mock净值历史
 */
export function createMockEquitySnapshot(count: number = 2) {
  const snapshots = []
  for (let i = 0; i < count; i++) {
    snapshots.push({
      time: `2024-01-01T${String(i * 12).padStart(2, '0')}:00:00Z`,
      total_equity: 10000 + i * 500,
      pnl: i * 500,
      pnl_pct: i * 5.0,
    })
  }
  return snapshots
}

// ============================================================================
// 渲染辅助函数
// ============================================================================

/**
 * 自定义render函数，自动包装Context Provider
 */
export function renderWithProviders(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, {
    ...options,
    wrapper: ({ children }) => (
      <LanguageProvider>
        <AuthProvider>{children}</AuthProvider>
      </LanguageProvider>
    ),
  })
}

/**
 * 等待元素出现（带自定义超时）
 */
export async function waitForElement(
  callback: () => HTMLElement | null,
  timeout: number = 5000
): Promise<HTMLElement> {
  const startTime = Date.now()

  while (Date.now() - startTime < timeout) {
    const element = callback()
    if (element) {
      return element
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }

  throw new Error(`Element not found within ${timeout}ms`)
}

/**
 * 等待多个元素出现（带自定义超时）
 */
export async function waitForElements(
  callback: () => HTMLElement[],
  timeout: number = 5000
): Promise<HTMLElement[]> {
  const startTime = Date.now()

  while (Date.now() - startTime < timeout) {
    const elements = callback()
    if (elements.length > 0) {
      return elements
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }

  throw new Error(`No elements found within ${timeout}ms`)
}

// ============================================================================
// 测试辅助函数
// ============================================================================

/**
 * 延迟函数（用于测试异步行为）
 */
export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/**
 * 创建mock用户
 */
export function createMockUser() {
  return {
    id: 'user-1',
    email: 'test@example.com',
    username: 'testuser',
    created_at: '2024-01-01T00:00:00Z',
  }
}

/**
 * 创建mock AI模型配置
 */
export function createMockAIModel() {
  return {
    id: 'model-1',
    name: 'Test Model',
    provider: 'openai',
    model: 'gpt-4',
    api_key: 'sk-test',
    base_url: 'https://api.openai.com/v1',
    created_at: '2024-01-01T00:00:00Z',
  }
}

/**
 * 创建mock交易所配置
 */
export function createMockExchange() {
  return {
    id: 'exchange-1',
    name: 'Binance',
    type: 'binance_futures',
    api_key: 'test-key',
    api_secret: 'test-secret',
    created_at: '2024-01-01T00:00:00Z',
  }
}
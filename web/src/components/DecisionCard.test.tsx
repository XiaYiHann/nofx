import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DecisionCard } from './DecisionCard'

describe('DecisionCard component', () => {
  it('renders auto-scaling metadata when action.auto_scaled is true', () => {
    const mockDecision = {
      timestamp: new Date().toISOString(),
      cycle_number: 1,
      input_prompt: 'mock prompt',
      cot_trace: 'mock cot',
      decision_json: '{}',
      account_state: {
        total_balance: 10000,
        available_balance: 3609.68,
        total_unrealized_profit: 0,
        position_count: 0,
        margin_used_pct: 0,
      },
      positions: [],
      candidate_coins: [],
      decisions: [
        {
          action: 'open_long',
          symbol: 'BTCUSDT',
          quantity: 0.1,
          leverage: 10,
          price: 60000,
          order_id: 12345,
          timestamp: new Date().toISOString(),
          success: true,
          auto_scaled: true,
          original_position_size_usd: 7228.0,
          scaled_position_size_usd: 3609.68,
          required_margin: 360.968,
          available_balance: 3609.68,
          scale_reason: 'Insufficient margin, scaled down',
        },
      ],
      execution_log: [],
      success: true,
    }

    render(<DecisionCard decision={mockDecision as any} language={'en'} />)

    // assert Auto-scaled text is present
    expect(screen.getByText(/Auto-scaled/i)).toBeTruthy()

    // assert original and scaled values are shown together (unique to auto-scale)
    expect(screen.getByText(/7228\.00\s*→\s*3609\.68/)).toBeTruthy()

    // scale reason should appear
    expect(screen.getByText(/Insufficient margin, scaled down/i)).toBeTruthy()
  })
})

package backtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderSimulator_ExecuteMarketOrder(t *testing.T) {
	// 10 bps slippage = 0.1%
	sim := NewOrderSimulator(10)
	marketPrice := 10000.0
	quantity := 1.0
	leverage := 1

	// Test Open Long (Price should increase by 0.1%)
	execPrice, fee, err := sim.ExecuteMarketOrder("long", "open", marketPrice, quantity, leverage)
	assert.NoError(t, err)
	expectedPrice := 10010.0 // 10000 * 1.001
	assert.InDelta(t, expectedPrice, execPrice, 0.000001)
	expectedFee := expectedPrice * quantity * TakerFeeRate
	assert.InDelta(t, expectedFee, fee, 0.000001)

	// Test Close Long (Price should decrease by 0.1%)
	execPrice, fee, err = sim.ExecuteMarketOrder("long", "close", marketPrice, quantity, leverage)
	assert.NoError(t, err)
	expectedPrice = 9990.0 // 10000 * 0.999
	assert.InDelta(t, expectedPrice, execPrice, 0.000001)

	// Test Open Short (Price should decrease by 0.1%)
	execPrice, fee, err = sim.ExecuteMarketOrder("short", "open", marketPrice, quantity, leverage)
	assert.NoError(t, err)
	expectedPrice = 9990.0
	assert.InDelta(t, expectedPrice, execPrice, 0.000001)

	// Test Close Short (Price should increase by 0.1%)
	execPrice, fee, err = sim.ExecuteMarketOrder("short", "close", marketPrice, quantity, leverage)
	assert.NoError(t, err)
	expectedPrice = 10010.0
	assert.InDelta(t, expectedPrice, execPrice, 0.000001)
}

func TestCalculatePositionSize(t *testing.T) {
	equity := 10000.0
	riskPercent := 1.0 // Risk $100
	entryPrice := 100.0
	stopLoss := 90.0 // Risk per unit = $10
	leverage := 10

	// Expected quantity = RiskAmount / PriceRisk = 100 / 10 = 10
	qty := CalculatePositionSize(equity, riskPercent, entryPrice, stopLoss, leverage)
	assert.Equal(t, 10.0, qty)

	// Test Leverage Cap
	// If stop loss is very close, calculated quantity might be huge
	stopLoss = 99.0 // Risk per unit = $1
	// Theoretical qty = 100 / 1 = 100 units
	// Max qty with 10x leverage = (10000 * 10) / 100 = 1000 units
	// So 100 is fine.

	// Let's try a case that hits the cap
	stopLoss = 99.99 // Risk per unit = 0.01
	// Theoretical qty = 100 / 0.01 = 10000 units
	// Max qty = 1000 units
	qty = CalculatePositionSize(equity, riskPercent, entryPrice, stopLoss, leverage)
	assert.Equal(t, 1000.0, qty, "Should be capped by leverage")
}

// TestOrderSimulator_InsufficientMargin 测试保证金不足场景
func TestOrderSimulator_InsufficientMargin(t *testing.T) {
	sim := NewOrderSimulator(10) // 10 bps slippage

	// 场景：可用余额 3609.68 USDT，尝试开仓 7228 USDT
	// 这正是用户报告的实际场景
	tests := []struct {
		name             string
		symbol           string
		side             string
		price            float64
		quantity         float64
		leverage         int
		availableBalance float64
		expectError      bool
		errorContains    string
	}{
		{
			name:             "保证金充足_10x杠杆",
			symbol:           "BTCUSDT",
			side:             "long",
			price:            100.0,
			quantity:         10.0,  // 仓位价值 1000 USDT
			leverage:         10,    // 需要保证金 100 USDT
			availableBalance: 200.0, // 可用 200 USDT
			expectError:      false,
		},
		{
			name:             "保证金不足_用户实际场景",
			symbol:           "ETHUSDT",
			side:             "long",
			price:            100.0,
			quantity:         72.28,   // 仓位价值 7228 USDT
			leverage:         1,       // 需要保证金 7228 USDT
			availableBalance: 3609.68, // 可用 3609.68 USDT
			expectError:      true,
			errorContains:    "insufficient margin",
		},
		{
			name:             "保证金刚好足够_边界",
			symbol:           "BTCUSDT",
			side:             "long",
			price:            100.0,
			quantity:         10.0,  // 仓位价值 1000 USDT
			leverage:         10,    // 需要保证金 100 USDT
			availableBalance: 100.0, // 刚好 100 USDT
			expectError:      false,
		},
		{
			name:             "保证金略微不足_边界",
			symbol:           "BTCUSDT",
			side:             "short",
			price:            100.0,
			quantity:         10.0,  // 仓位价值 1000 USDT
			leverage:         10,    // 需要保证金 100 USDT
			availableBalance: 99.99, // 略微不足
			expectError:      true,
			errorContains:    "insufficient margin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sim.ValidateOrder(tt.symbol, tt.side, "open", tt.quantity, tt.price, tt.availableBalance, tt.leverage)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

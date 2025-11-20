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

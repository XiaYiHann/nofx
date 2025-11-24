package trader

import (
	"fmt"
	"strings"
)

// RiskManager handles risk validation for trades
type RiskManager struct {
	MaxLeverageBTCETH     int
	MaxLeverageAlt        int
	MaxPositionSizeBTCETH float64 // Multiplier of equity (e.g., 10.0)
	MaxPositionSizeAlt    float64 // Multiplier of equity (e.g., 1.5)
	MinRiskRewardRatio    float64
}

// NewRiskManager creates a new RiskManager with default or provided config
func NewRiskManager(btcEthLev, altLev int) *RiskManager {
	return &RiskManager{
		MaxLeverageBTCETH:     btcEthLev,
		MaxLeverageAlt:        altLev,
		MaxPositionSizeBTCETH: 10.0,
		MaxPositionSizeAlt:    1.5,
		MinRiskRewardRatio:    2.0,
	}
}

// CheckLeverage validates if the requested leverage is within limits
func (rm *RiskManager) CheckLeverage(symbol string, leverage int) error {
	maxLev := rm.MaxLeverageAlt
	if isBTC(symbol) || isETH(symbol) {
		maxLev = rm.MaxLeverageBTCETH
	}

	if leverage > maxLev {
		return fmt.Errorf("leverage %d exceeds limit %d for %s", leverage, maxLev, symbol)
	}
	return nil
}

// CheckPositionSize validates if the position size is within limits relative to equity
func (rm *RiskManager) CheckPositionSize(symbol string, sizeUSD, equity float64) error {
	if equity <= 0 {
		return fmt.Errorf("invalid equity: %.2f", equity)
	}

	maxMult := rm.MaxPositionSizeAlt
	if isBTC(symbol) || isETH(symbol) {
		maxMult = rm.MaxPositionSizeBTCETH
	}

	limit := equity * maxMult
	if sizeUSD > limit {
		return fmt.Errorf("position size %.2f exceeds limit %.2f (%.1fx equity) for %s", sizeUSD, limit, maxMult, symbol)
	}
	return nil
}

// CheckRiskReward validates the Risk-Reward ratio
func (rm *RiskManager) CheckRiskReward(entry, stopLoss, takeProfit float64, side string) error {
	var risk, reward float64

	if side == "long" || side == "open_long" {
		risk = entry - stopLoss
		reward = takeProfit - entry
	} else if side == "short" || side == "open_short" {
		risk = stopLoss - entry
		reward = entry - takeProfit
	} else {
		return fmt.Errorf("invalid side: %s", side)
	}

	if risk <= 0 {
		return fmt.Errorf("invalid risk: %.2f (stop loss must be below entry for long, above for short)", risk)
	}
	if reward <= 0 {
		return fmt.Errorf("invalid reward: %.2f (take profit must be above entry for long, below for short)", reward)
	}

	rr := reward / risk
	if rr < rm.MinRiskRewardRatio {
		return fmt.Errorf("risk-reward ratio %.2f is below minimum %.2f", rr, rm.MinRiskRewardRatio)
	}
	return nil
}

// Helper functions
func isBTC(symbol string) bool {
	return strings.Contains(strings.ToUpper(symbol), "BTC")
}

func isETH(symbol string) bool {
	return strings.Contains(strings.ToUpper(symbol), "ETH")
}

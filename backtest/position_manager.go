package backtest

import (
	"fmt"
	"math"
)

// PositionManager 持仓管理器
type PositionManager struct {
	positions        map[string]*Position // symbol -> position
	equity           float64              // 账户净值
	initialBalance   float64              // 初始资金
	availableBalance float64              // 可用资金
}

// NewPositionManager 创建持仓管理器
func NewPositionManager(initialBalance float64) *PositionManager {
	return &PositionManager{
		positions:        make(map[string]*Position),
		equity:           initialBalance,
		initialBalance:   initialBalance,
		availableBalance: initialBalance,
	}
}

// OpenPosition 开仓
func (pm *PositionManager) OpenPosition(symbol, side string, entryPrice, quantity float64, leverage int) error {
	// 检查是否已有持仓
	if _, exists := pm.positions[symbol]; exists {
		return fmt.Errorf("position already exists for %s", symbol)
	}

	// 计算所需保证金
	marginUsed := (entryPrice * quantity) / float64(leverage)

	// 检查可用资金
	if marginUsed > pm.availableBalance {
		return fmt.Errorf("insufficient balance: need %.2f, available %.2f", marginUsed, pm.availableBalance)
	}

	// 创建持仓
	position := &Position{
		Symbol:        symbol,
		Side:          side,
		EntryPrice:    entryPrice,
		Quantity:      quantity,
		Leverage:      leverage,
		MarginUsed:    marginUsed,
		UnrealizedPnL: 0,
	}

	pm.positions[symbol] = position
	pm.availableBalance -= marginUsed

	return nil
}

// ClosePosition 平仓
func (pm *PositionManager) ClosePosition(symbol string, exitPrice float64) (*Trade, error) {
	position, exists := pm.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("no position found for %s", symbol)
	}

	// 计算盈亏
	var pnl float64
	if position.Side == "long" {
		pnl = (exitPrice - position.EntryPrice) * position.Quantity
	} else {
		pnl = (position.EntryPrice - exitPrice) * position.Quantity
	}

	pnlPct := (pnl / position.MarginUsed) * 100

	// 创建交易记录
	trade := &Trade{
		Symbol:     symbol,
		Side:       position.Side,
		Action:     "close",
		EntryPrice: position.EntryPrice,
		ExitPrice:  exitPrice,
		Quantity:   position.Quantity,
		Leverage:   position.Leverage,
		PnL:        pnl,
		PnLPct:     pnlPct,
	}

	// 更新账户
	pm.availableBalance += position.MarginUsed + pnl
	pm.equity += pnl

	// 移除持仓
	delete(pm.positions, symbol)

	return trade, nil
}

// UpdateUnrealizedPnL 更新未实现盈亏
func (pm *PositionManager) UpdateUnrealizedPnL(symbol string, currentPrice float64) error {
	position, exists := pm.positions[symbol]
	if !exists {
		return fmt.Errorf("no position found for %s", symbol)
	}

	var unrealizedPnL float64
	if position.Side == "long" {
		unrealizedPnL = (currentPrice - position.EntryPrice) * position.Quantity
	} else {
		unrealizedPnL = (position.EntryPrice - currentPrice) * position.Quantity
	}

	position.UnrealizedPnL = unrealizedPnL
	return nil
}

// GetEquity 获取账户净值(包含未实现盈亏)
func (pm *PositionManager) GetEquity() float64 {
	totalUnrealizedPnL := 0.0
	for _, position := range pm.positions {
		totalUnrealizedPnL += position.UnrealizedPnL
	}
	return pm.availableBalance + totalUnrealizedPnL + pm.getUsedMargin()
}

// GetPosition 获取持仓
func (pm *PositionManager) GetPosition(symbol string) (*Position, bool) {
	position, exists := pm.positions[symbol]
	return position, exists
}

// HasPosition 检查是否有持仓
func (pm *PositionManager) HasPosition(symbol string) bool {
	_, exists := pm.positions[symbol]
	return exists
}

// GetAvailableBalance 获取可用资金
func (pm *PositionManager) GetAvailableBalance() float64 {
	return pm.availableBalance
}

// getUsedMargin 获取已使用保证金
func (pm *PositionManager) getUsedMargin() float64 {
	totalMargin := 0.0
	for _, position := range pm.positions {
		totalMargin += position.MarginUsed
	}
	return totalMargin
}

// CalculateMaxDrawdown 计算最大回撤
func CalculateMaxDrawdown(equitySnapshots []EquitySnapshot) float64 {
	if len(equitySnapshots) == 0 {
		return 0
	}

	maxEquity := equitySnapshots[0].Equity
	maxDrawdown := 0.0

	for _, snapshot := range equitySnapshots {
		if snapshot.Equity > maxEquity {
			maxEquity = snapshot.Equity
		}

		drawdown := (maxEquity - snapshot.Equity) / maxEquity * 100
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// CalculateSharpeRatio 计算夏普率(简化版,假设无风险利率为0)
func CalculateSharpeRatio(equitySnapshots []EquitySnapshot) float64 {
	if len(equitySnapshots) < 2 {
		return 0
	}

	// 计算收益率序列
	returns := make([]float64, len(equitySnapshots)-1)
	for i := 1; i < len(equitySnapshots); i++ {
		returns[i-1] = (equitySnapshots[i].Equity - equitySnapshots[i-1].Equity) / equitySnapshots[i-1].Equity
	}

	// 计算平均收益率
	var sumReturns float64
	for _, r := range returns {
		sumReturns += r
	}
	avgReturn := sumReturns / float64(len(returns))

	// 计算标准差
	var sumSquaredDiff float64
	for _, r := range returns {
		diff := r - avgReturn
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(returns)))

	if stdDev == 0 {
		return 0
	}

	// 年化(假设252个交易日)
	annualizedReturn := avgReturn * 252
	annualizedStdDev := stdDev * math.Sqrt(252)

	return annualizedReturn / annualizedStdDev
}

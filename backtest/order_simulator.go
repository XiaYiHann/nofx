package backtest

import (
	"fmt"
	"math"
)

const (
	TakerFeeRate = 0.0004 // 0.04% Binance期货taker手续费
)

// OrderSimulator 订单模拟器,模拟订单执行
type OrderSimulator struct {
	slippageBps int // 滑点(基点, 10 = 0.1%)
}

// NewOrderSimulator 创建订单模拟器
func NewOrderSimulator(slippageBps int) *OrderSimulator {
	return &OrderSimulator{
		slippageBps: slippageBps,
	}
}

// ExecuteMarketOrder 执行市价单
// side: "long" or "short"
// action: "open" or "close"
func (os *OrderSimulator) ExecuteMarketOrder(
	side, action string,
	marketPrice, quantity float64,
	leverage int,
) (executionPrice, fee float64, err error) {
	// 应用滑点
	executionPrice = os.applySlippage(marketPrice, side, action)
	
	// 计算手续费(基于名义价值)
	notionalValue := executionPrice * quantity
	fee = notionalValue * TakerFeeRate
	
	return executionPrice, fee, nil
}

// applySlippage 应用滑点
func (os *OrderSimulator) applySlippage(price float64, side, action string) float64 {
	slippagePct := float64(os.slippageBps) / 10000.0
	
	// 开多、平空: 向上滑点
	// 开空、平多: 向下滑点
	if (side == "long" && action == "open") || (side == "short" && action == "close") {
		return price * (1 + slippagePct)
	}
	
	return price * (1 - slippagePct)
}

// CalculatePositionSize 根据风险百分比计算仓位大小
func CalculatePositionSize(
	accountEquity float64,
	riskPercent float64,
	entryPrice float64,
	stopLossPrice float64,
	leverage int,
) float64 {
	// 风险金额
	riskAmount := accountEquity * (riskPercent / 100.0)
	
	// 每单位的风险
	priceRisk := math.Abs(entryPrice - stopLossPrice)
	if priceRisk == 0 {
		return 0
	}
	
	// 计算数量
	quantity := riskAmount / priceRisk
	
	// 确保不超过杠杆限制
	maxQuantity := (accountEquity * float64(leverage)) / entryPrice
	if quantity > maxQuantity {
		quantity = maxQuantity
	}
	
	return quantity
}

// ValidateOrder 验证订单是否合法
func (os *OrderSimulator) ValidateOrder(
	symbol, side, action string,
	quantity, price float64,
	availableBalance float64,
	leverage int,
) error {
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity: %.8f", quantity)
	}
	
	if price <= 0 {
		return fmt.Errorf("invalid price: %.8f", price)
	}
	
	if action == "open" {
		// 检查保证金是否足够
		requiredMargin := (price * quantity) / float64(leverage)
		if requiredMargin > availableBalance {
			return fmt.Errorf("insufficient margin: need %.2f, available %.2f", requiredMargin, availableBalance)
		}
	}
	
	return nil
}

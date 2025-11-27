package testhelpers

import (
	"fmt"
	"nofx/trader"
)

type MockTrader struct {
	Balance   map[string]interface{}
	Positions []map[string]interface{}
	Prices    map[string]float64
}

// Ensure MockTrader implements trader.Trader
var _ trader.Trader = (*MockTrader)(nil)

func NewMockTrader() *MockTrader {
	return &MockTrader{
		Balance: map[string]interface{}{
			"availableBalance":      1000.0,
			"totalWalletBalance":    1000.0,
			"totalUnrealizedProfit": 0.0,
		},
		Positions: []map[string]interface{}{},
		Prices: map[string]float64{
			"BTCUSDT": 50000.0,
			"ETHUSDT": 3000.0,
		},
	}
}

func (m *MockTrader) GetBalance() (map[string]interface{}, error) {
	return m.Balance, nil
}

func (m *MockTrader) GetPositions() ([]map[string]interface{}, error) {
	return m.Positions, nil
}

func (m *MockTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return map[string]interface{}{"orderId": int64(123), "status": "FILLED", "symbol": symbol}, nil
}

func (m *MockTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return map[string]interface{}{"orderId": int64(124), "status": "FILLED", "symbol": symbol}, nil
}

func (m *MockTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	return map[string]interface{}{"orderId": int64(125), "status": "FILLED", "symbol": symbol}, nil
}

func (m *MockTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	return map[string]interface{}{"orderId": int64(126), "status": "FILLED", "symbol": symbol}, nil
}

func (m *MockTrader) SetLeverage(symbol string, leverage int) error {
	return nil
}

func (m *MockTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	return nil
}

func (m *MockTrader) GetMarketPrice(symbol string) (float64, error) {
	if p, ok := m.Prices[symbol]; ok {
		return p, nil
	}
	return 100.0, nil
}

func (m *MockTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	return nil
}

func (m *MockTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	return nil
}

func (m *MockTrader) CancelStopLossOrders(symbol string) error {
	return nil
}

func (m *MockTrader) CancelTakeProfitOrders(symbol string) error {
	return nil
}

func (m *MockTrader) CancelAllOrders(symbol string) error {
	return nil
}

func (m *MockTrader) CancelStopOrders(symbol string) error {
	return nil
}

func (m *MockTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	return fmt.Sprintf("%.3f", quantity), nil
}

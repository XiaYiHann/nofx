package trader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"nofx/decision"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockTraderTestify is a mock implementation of the Trader interface using testify
type MockTraderTestify struct {
	mock.Mock
}

func (m *MockTraderTestify) GetBalance() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTraderTestify) GetPositions() ([]map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockTraderTestify) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	args := m.Called(symbol, quantity, leverage)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTraderTestify) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	args := m.Called(symbol, quantity, leverage)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTraderTestify) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	args := m.Called(symbol, quantity)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTraderTestify) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	args := m.Called(symbol, quantity)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTraderTestify) SetLeverage(symbol string, leverage int) error {
	args := m.Called(symbol, leverage)
	return args.Error(0)
}

func (m *MockTraderTestify) SetMarginMode(symbol string, isCrossMargin bool) error {
	args := m.Called(symbol, isCrossMargin)
	return args.Error(0)
}

func (m *MockTraderTestify) GetMarketPrice(symbol string) (float64, error) {
	args := m.Called(symbol)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTraderTestify) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	args := m.Called(symbol, positionSide, quantity, stopPrice)
	return args.Error(0)
}

func (m *MockTraderTestify) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	args := m.Called(symbol, positionSide, quantity, takeProfitPrice)
	return args.Error(0)
}

func (m *MockTraderTestify) CancelStopLossOrders(symbol string) error {
	args := m.Called(symbol)
	return args.Error(0)
}

func (m *MockTraderTestify) CancelTakeProfitOrders(symbol string) error {
	args := m.Called(symbol)
	return args.Error(0)
}

func (m *MockTraderTestify) CancelAllOrders(symbol string) error {
	args := m.Called(symbol)
	return args.Error(0)
}

func (m *MockTraderTestify) CancelStopOrders(symbol string) error {
	args := m.Called(symbol)
	return args.Error(0)
}

func (m *MockTraderTestify) FormatQuantity(symbol string, quantity float64) (string, error) {
	args := m.Called(symbol, quantity)
	return args.String(0), args.Error(1)
}

func (m *MockTraderTestify) GetOpenOrders(symbol string) ([]map[string]interface{}, error) {
	args := m.Called(symbol)
	if args.Get(0) == nil {
		return []map[string]interface{}{}, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func TestExecuteOpenLong(t *testing.T) {
	mockTrader := new(MockTraderTestify)

	// Mock Market Data Provider
	mockMarketData := func(symbol string, config ...*market.IndicatorConfig) (*market.Data, error) {
		return &market.Data{
			Symbol:       symbol,
			CurrentPrice: 50000.0,
		}, nil
	}

	at := &AutoTrader{
		trader: mockTrader,
		config: AutoTraderConfig{
			IsCrossMargin:   false,
			IndicatorConfig: nil,
		},
		marketDataProvider:    mockMarketData,
		decisionLogger:        logger.NewDecisionLogger("test_logs"),
		positionFirstSeenTime: make(map[string]int64),
	}

	// Mock GetPositions (called before open)
	mockTrader.On("GetPositions").Return([]map[string]interface{}{}, nil)

	// Mock GetBalance
	mockTrader.On("GetBalance").Return(map[string]interface{}{
		"availableBalance": 10000.0,
	}, nil)

	// Mock SetMarginMode
	mockTrader.On("SetMarginMode", "BTCUSDT", false).Return(nil)

	// Mock OpenLong
	mockTrader.On("OpenLong", "BTCUSDT", mock.Anything, 5).Return(map[string]interface{}{
		"orderId": int64(12345),
	}, nil)

	// Mock SetStopLoss/TakeProfit
	mockTrader.On("SetStopLoss", "BTCUSDT", "LONG", mock.Anything, 49000.0).Return(nil)
	mockTrader.On("SetTakeProfit", "BTCUSDT", "LONG", mock.Anything, 55000.0).Return(nil)

	// Decision
	dec := &decision.Decision{
		Action:          "open_long",
		Symbol:          "BTCUSDT",
		Leverage:        5,
		PositionSizeUSD: 1000.0,
		StopLoss:        49000.0,
		TakeProfit:      55000.0,
	}

	actionRecord := &logger.DecisionAction{}

	err := at.executeOpenLongWithRecord(dec, actionRecord)
	if err != nil {
		t.Errorf("executeOpenLongWithRecord failed: %v", err)
	}

	mockTrader.AssertExpectations(t)
}

func TestAnalyzeAndTrade(t *testing.T) {
	// Mock AI Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a valid AI response
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"open_long\", \"leverage\": 5, \"position_size_usd\": 1000, \"stop_loss\": 49000, \"take_profit\": 55000, \"reasoning\": \"Test\"}]\n```",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	mockTrader := new(MockTraderTestify)

	// Mock Market Data Provider
	mockMarketData := func(symbol string, config ...*market.IndicatorConfig) (*market.Data, error) {
		return &market.Data{
			Symbol:       symbol,
			CurrentPrice: 50000.0,
		}, nil
	}

	mcpClient := mcp.New()
	mcpClient.SetCustomAPI(ts.URL, "test-key", "test-model")

	at := &AutoTrader{
		id:     "test-trader",
		trader: mockTrader,
		config: AutoTraderConfig{
			IsCrossMargin:   false,
			IndicatorConfig: nil,
			BTCETHLeverage:  5,
			AltcoinLeverage: 5,
		},
		marketDataProvider:    mockMarketData,
		decisionLogger:        logger.NewDecisionLogger("test_logs"),
		mcpClient:             mcpClient,
		tradingCoins:          []string{"BTCUSDT"}, // Avoid pool API
		initialBalance:        10000.0,
		startTime:             time.Now(),
		systemPromptTemplate:  "default",
		aiModel:               "custom",
		customPrompt:          "test prompt",
		positionFirstSeenTime: make(map[string]int64),
	}

	// Mock calls for getContext
	mockTrader.On("GetBalance").Return(map[string]interface{}{
		"totalWalletBalance":    10000.0,
		"totalUnrealizedProfit": 0.0,
		"availableBalance":      10000.0,
	}, nil)
	mockTrader.On("GetPositions").Return([]map[string]interface{}{}, nil)
	mockTrader.On("GetOpenOrders", "").Return([]map[string]interface{}{}, nil)

	// Mock calls for execution (same as TestExecuteOpenLong)
	mockTrader.On("SetMarginMode", "BTCUSDT", false).Return(nil)
	mockTrader.On("OpenLong", "BTCUSDT", mock.Anything, 5).Return(map[string]interface{}{
		"orderId": int64(12345),
	}, nil)
	mockTrader.On("SetStopLoss", "BTCUSDT", "LONG", mock.Anything, 49000.0).Return(nil)
	mockTrader.On("SetTakeProfit", "BTCUSDT", "LONG", mock.Anything, 55000.0).Return(nil)

	// Run runCycle
	err := at.runCycle()
	if err != nil {
		t.Errorf("runCycle failed: %v", err)
	}

	mockTrader.AssertExpectations(t)
}

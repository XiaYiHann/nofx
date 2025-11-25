package backtest

import (
	"nofx/market"
	"testing"
	"time"
)

func TestCalculateMarketData(t *testing.T) {
	// Create a dummy engine
	engine := &Engine{}

	// Generate dummy klines (enough for indicators)
	klines := make([]market.Kline, 200)
	basePrice := 50000.0
	now := time.Now()

	for i := 0; i < 200; i++ {
		klines[i] = market.Kline{
			OpenTime:  now.Add(time.Duration(i) * 3 * time.Minute).UnixMilli(),
			Open:      basePrice + float64(i),
			High:      basePrice + float64(i) + 10,
			Low:       basePrice + float64(i) - 10,
			Close:     basePrice + float64(i) + 5, // Upward trend
			Volume:    100.0,
			CloseTime: now.Add(time.Duration(i+1) * 3 * time.Minute).UnixMilli(),
		}
	}

	// Call calculateMarketData
	data, err := engine.calculateMarketData("BTCUSDT", klines)
	if err != nil {
		t.Fatalf("calculateMarketData failed: %v", err)
	}

	// Verify indicators are calculated
	if data.CurrentEMA20 == 0 {
		t.Error("CurrentEMA20 should not be 0")
	}
	if data.CurrentMACD == 0 {
		t.Error("CurrentMACD should not be 0")
	}
	if data.CurrentRSI7 == 0 {
		t.Error("CurrentRSI7 should not be 0")
	}

	// Verify TimeframeData
	tfData, ok := data.TimeframeData["3m"]
	if !ok {
		t.Fatal("TimeframeData['3m'] missing")
	}

	if len(tfData.EMA20Values) == 0 {
		t.Error("EMA20Values empty")
	}
	if len(tfData.MACDValues) == 0 {
		t.Error("MACDValues empty")
	}
	if len(tfData.RSI7Values) == 0 {
		t.Error("RSI7Values empty")
	}
	if len(tfData.BollingerUpper) == 0 {
		t.Error("BollingerUpper empty")
	}
}

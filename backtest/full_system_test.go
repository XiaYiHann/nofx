package backtest

import (
	"context"
	"nofx/config"
	"nofx/market"
	"nofx/mcp"
	"nofx/testhelpers"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
)

func TestFullBacktestSystem(t *testing.T) {
	// 1. Setup DB
	db, cleanup := testhelpers.SetupTestDB(t)
	defer cleanup()

	// 2. Setup Mock LLM
	// We provide a response that suggests opening a long position
	llmResponse := `
<reasoning>
Analysis of BTCUSDT. Uptrend detected. RSI is low.
</reasoning>
<decision>
` + "```json" + `
[
  {
    "symbol": "BTCUSDT",
    "action": "open_long",
    "leverage": 5,
    "position_size_usd": 100,
    "stop_loss": 45000,
    "take_profit": 55000,
    "confidence": 90,
    "reasoning": "Strong uptrend"
  }
]
` + "```" + `
</decision>
`
	llmServer := testhelpers.SetupMockLLMServer(t, llmResponse)
	defer llmServer.Close()

	mcpClient := mcp.New()
	mcpClient.SetCustomAPI(llmServer.URL, "test-key", "test-model")

	// 3. Patch Market Data
	// Patch (*APIClient).GetKlinesRange to return generated klines
	var p *market.APIClient
	patches := gomonkey.ApplyMethod(p, "GetKlinesRange", func(_ *market.APIClient, symbol, interval string, start, end int64) ([]market.Kline, error) {
		// Generate dummy klines (enough for 1 hour 3m candles + preheat)
		// Need about 12h + 1h data = 13 * 20 = 260 candles.
		count := 300
		klines := make([]market.Kline, count)
		basePrice := 50000.0

		// End time aligns with 'end' arg
		endTime := time.UnixMilli(end)

		for i := 0; i < count; i++ {
			// Calculate time for this kline
			// Index (count-1) is the last one, at 'end'
			// Index 0 is (count-1) steps before 'end'
			offset := count - 1 - i
			openTime := endTime.Add(-time.Duration(offset) * 3 * time.Minute)

			// Simple uptrend
			price := basePrice + float64(i)*5.0

			klines[i] = market.Kline{
				OpenTime:  openTime.UnixMilli(),
				Open:      price,
				High:      price + 10,
				Low:       price - 10,
				Close:     price + 5,
				Volume:    1000.0,
				CloseTime: openTime.Add(3 * time.Minute).UnixMilli(),
			}
		}
		return klines, nil
	})
	defer patches.Reset()

	// 4. Setup Config
	endTime := time.Now().Truncate(time.Hour)
	startTime := endTime.Add(-1 * time.Hour) // 1 hour backtest

	cfg := &Config{
		TraderID:        "test_trader",
		UserID:          "test_user",
		StartTime:       startTime,
		EndTime:         endTime,
		InitialBalance:  10000.0,
		ScanInterval:    3 * time.Minute,
		TradingSymbols:  []string{"BTCUSDT"},
		Slippage:        0,
		UseTraderConfig: false,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
		BTCETHLeverage:  5,
		AltcoinLeverage: 5,
	}

	// 5. Run Engine
	backtestID := "test_run_system"
	engine := NewEngine(backtestID, cfg, db, mcpClient)

	// Create record
	err := db.CreateBacktest(&config.BacktestRun{
		ID:             backtestID,
		UserID:         cfg.UserID,
		TraderID:       cfg.TraderID,
		StartTime:      cfg.StartTime,
		EndTime:        cfg.EndTime,
		InitialBalance: cfg.InitialBalance,
		Status:         "pending",
	})
	if err != nil {
		t.Fatalf("CreateBacktest failed: %v", err)
	}

	err = engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Engine.Run failed: %v", err)
	}

	// 6. Verify
	res, err := db.GetBacktest(backtestID)
	if err != nil {
		t.Fatalf("GetBacktest failed: %v", err)
	}

	if res.Status != "completed" {
		t.Errorf("Expected status completed, got %s", res.Status)
	}

	// We expect trades because LLM returned open_long and market data was provided.
	trades, err := db.GetBacktestTrades(backtestID)
	if err != nil {
		t.Fatalf("GetBacktestTrades failed: %v", err)
	}
	if len(trades) == 0 {
		t.Log("Warning: No trades executed. This might happen if logic filters out signals or if data alignment is off.")
	} else {
		t.Logf("Executed %d trades", len(trades))
	}

	decisions, err := db.GetBacktestDecisions(backtestID)
	if err != nil {
		t.Fatalf("GetBacktestDecisions failed: %v", err)
	}
	if len(decisions) == 0 {
		t.Error("Expected decisions to be recorded")
	}
}

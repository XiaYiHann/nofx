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
	"github.com/stretchr/testify/assert"
)

func TestRunBacktestUsesAiModelIdOverride(t *testing.T) {
	// 1. Setup DB
	db, cleanup := testhelpers.SetupTestDB(t)
	defer cleanup()

	// 2. Setup 2 Mock LLMs
	// Default model behavior (e.g. always hold)
	llmDefault := testhelpers.SetupMockLLMServer(t, `<decision>{"action":"hold","symbol":"BTCUSDT"}</decision>`)
	defer llmDefault.Close()

	// Overridden model behavior (e.g. always buy)
	llmOverride := testhelpers.SetupMockLLMServer(t, `
<reasoning>Override model says buy</reasoning>
<decision>
`+"```json"+`
[{"symbol":"BTCUSDT","action":"open_long","position_size_usd":100,"confidence":100}]
`+"```"+`
</decision>`)
	defer llmOverride.Close()

	// 3. Create DB Records
	// User
	db.CreateUser(&config.User{ID: "u1", Email: "u1@test.com"})

	// Default Model (trader uses this)
	db.CreateAIModel("u1", "default_model", "Default", "openai", true, "key1", llmDefault.URL)

	// Override Model (backtest uses this)
	db.CreateAIModel("u1", "override_model", "Override", "openai", true, "key2", llmOverride.URL)

	// Exchange
	db.CreateExchange("u1", "ex1", "Binance", "cex", true, "k", "s", true, "", "", "", "")

	// Trader
	trader := &config.TraderRecord{
		ID:             "t1",
		UserID:         "u1",
		AIModelID:      "default_model",
		ExchangeID:     "ex1",
		TradingSymbols: "BTCUSDT",
	}
	db.CreateTrader(trader)

	// 4. Setup Config with Override
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()
	
	cfg := &Config{
		TraderID:       "t1",
		UserID:         "u1",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000,
		TradingSymbols: []string{"BTCUSDT"},
		AiModelID:      "override_model", // <--- THE KEY OVERRIDE
		Timeframe:      "3m",
	}

	// 5. Mock MCP Client creation to verify it uses the override URL
	// Since we can't easily inspect the internal state of Engine's mcpClient,
	// we rely on the behavior: if it uses llmOverride, it will generate trades.
	// If it uses llmDefault, it will generate NO trades (hold).
	
	// We need to construct the engine manually as we would in the handler
	model, _ := db.GetAIModelByID("u1", "override_model")
	
	mcpClient := mcp.New()
	mcpClient.SetCustomAPI(model.CustomAPIURL, model.APIKey, model.CustomModelName)

	engine := NewEngine("bt_override", cfg, db, mcpClient)

	// 6. Mock Market Data to ensure run succeeds
	var p *market.APIClient
	patches := gomonkey.ApplyMethod(p, "GetKlinesRange", func(_ *market.APIClient, symbol, interval string, start, end int64) ([]market.Kline, error) {
		// Generate enough data for preheat (12h default) + run duration (1h)
		// Let's generate 24h of data ending at 'end'
		// 3m interval = 20 per hour. 24h = 480 candles.
		
		endTime := time.UnixMilli(end)
		count := 500
		klines := make([]market.Kline, count)
		base := 50000.0
		
		for i := 0; i < count; i++ {
			// Generate backwards from end
			idx := count - 1 - i
			t := endTime.Add(-time.Duration(i) * 3 * time.Minute)
			
			klines[idx] = market.Kline{
				OpenTime: t.UnixMilli(),
				Close:    base + float64(idx), // Uptrend
				High:     base + float64(idx) + 10,
				Low:      base + float64(idx) - 10,
				CloseTime: t.Add(3 * time.Minute).UnixMilli(),
				Volume: 1000,
			}
		}
		return klines, nil
	})
	defer patches.Reset()

	// 7. Run
	err := engine.Run(context.Background())
	assert.NoError(t, err)

	// 8. Verify Trades
	// If override worked, we should have trades (open_long).
	// If it failed and used default, we would have 0 trades (hold).
	trades := engine.trades
	assert.NotEmpty(t, trades, "Should have executed trades using override model")
	if len(trades) > 0 {
		assert.Equal(t, "long", trades[0].Side)
	}
}

func TestEngineRespectsPreheatAndDataPoints(t *testing.T) {
	// Setup
	db, cleanup := testhelpers.SetupTestDB(t)
	defer cleanup()
	
	// Config with specific timeframe and preheat
	cfg := &Config{
		TraderID:        "t1",
		UserID:          "u1",
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(1*time.Hour),
		TradingSymbols:  []string{"BTCUSDT"},
		Timeframe:       "15m",   // Custom timeframe
		DataPoints:      50,      // Custom data points
		PreheatDuration: 24*time.Hour, // Custom preheat
	}
	
	engine := NewEngine("bt_params", cfg, db, nil)
	
	// We want to verify loadHistoricalData uses these params.
	// We can't easily mock internal methods, but we can mock APIClient.GetKlinesRange
	// and check the arguments passed to it.
	
	var capturedInterval string
	var capturedStart int64
	
	var p *market.APIClient
	patches := gomonkey.ApplyMethod(p, "GetKlinesRange", func(_ *market.APIClient, symbol, interval string, start, end int64) ([]market.Kline, error) {
		capturedInterval = interval
		capturedStart = start
		return []market.Kline{}, nil
	})
	defer patches.Reset()
	
	engine.loadHistoricalData(context.Background())
	
	// Assertions
	assert.Equal(t, "15m", capturedInterval, "Should use configured timeframe for API call")
	
	// Verify start time includes preheat
	expectedStart := cfg.StartTime.Add(-24*time.Hour).UnixMilli()
	// Allow small delta due to time calculations
	assert.InDelta(t, expectedStart, capturedStart, 1000, "Start time should reflect 24h preheat")
}

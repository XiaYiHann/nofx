package backtest

import (
	"context"
	"nofx/config"
	"nofx/market"
	"nofx/mcp"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// loadEnv loads environment variables from .env file for testing
func loadEnv(t *testing.T) {
	// Try to find .env in project root
	envPath := "../.env"
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Logf("⚠️  No .env file found at %s, assuming environment variables are set", envPath)
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			value = strings.Trim(value, `"'`)
			os.Setenv(key, value)
		}
	}
}

// TestFullBacktestIntegration runs a full backtest using real LLM and market data.
// This test requires LLM_API_KEY to be set in .env.
func TestFullBacktestIntegration(t *testing.T) {
	// 1. Load Environment Variables
	loadEnv(t)

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: LLM_API_KEY not set")
	}

	// 2. Setup Database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_backtest.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// 3. Setup MCP Client (GLM-4-Flash)
	mcpClient := mcp.New()
	llmURL := os.Getenv("LLM_API_URL")
	llmModel := os.Getenv("LLM_MODEL")
	if llmModel == "" {
		llmModel = "glm-4-flash"
	}
	
	// Use SetGLMAPIKey as requested
	mcpClient.SetGLMAPIKey(apiKey, llmURL, llmModel)

	// 4. Setup Backtest Config
	// Use a short recent period to ensure data availability and speed
	endTime := time.Now().Truncate(time.Hour)
	startTime := endTime.Add(-4 * time.Hour) // 4 hours backtest

	// Create backtest configuration
	cfg := &Config{
		TraderID:       "test_trader",
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   1 * time.Hour,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10, // 0.1%

		UseTraderConfig: false,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
		BTCETHLeverage:  1,
		AltcoinLeverage: 1,
	}

	// 5. Create Engine
	backtestID := "test_run_" + time.Now().Format("20060102150405")
	engine := NewEngine(backtestID, cfg, db, mcpClient)

	// Create backtest record in DB (Required because Engine updates existing record)
	err = db.CreateBacktest(&config.BacktestRun{
		ID:             backtestID,
		UserID:         cfg.UserID,
		TraderID:       cfg.TraderID,
		StartTime:      cfg.StartTime,
		EndTime:        cfg.EndTime,
		InitialBalance: cfg.InitialBalance,
		Status:         "pending",
	})
	if err != nil {
		t.Fatalf("Failed to create backtest record: %v", err)
	}

	// 6. Run Backtest
	t.Logf("Starting backtest from %s to %s...", startTime, endTime)
	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Fatalf("Backtest failed: %v", err)
	}

	// 7. Verify Results
	result := engine.calculateResult()
	
	t.Logf("Backtest Completed!")
	t.Logf("Final Equity: %.2f", result.FinalEquity)
	t.Logf("Total PnL: %.2f (%.2f%%)", result.TotalPnL, result.TotalPnLPct)
	t.Logf("Total Trades: %d", result.TotalTrades)
	t.Logf("Win Rate: %.2f%%", result.WinRate)

	// Assertions
	if len(result.EquitySnapshots) == 0 {
		t.Error("No equity snapshots recorded")
	}

	// Check if decisions were made (even if no trades executed, decisions should be logged)
	// We need to access engine.decisions, but it's private. 
	// However, we can check the database or just rely on the fact that Run() completed without error.
	// Since we are using a real LLM, we expect *some* interaction.
	
	// Verify database records
	savedResult, err := db.GetBacktest(backtestID)
	if err != nil {
		t.Errorf("Failed to retrieve saved result from DB: %v", err)
	}
	if savedResult == nil {
		t.Error("Saved result is nil")
	} else {
		if savedResult.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", savedResult.Status)
		}
	}
}

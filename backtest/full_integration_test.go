package backtest

import (
	"context"
	"fmt"
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
	// Check if LLM_API_KEY is already set
	if os.Getenv("LLM_API_KEY") != "" {
		t.Logf("✓ LLM_API_KEY already exists in environment")
		return
	}

	// Try multiple possible .env locations
	envPaths := []string{
		".env",          // 当前目录
		"../.env",       // 父目录
		"../../.env",    // 父父目录
		"./.env",        // 显式当前目录
		"../../../.env", // 更上层目录
	}

	// Get current working directory for debugging
	wd, _ := os.Getwd()
	t.Logf("🔍 Current working directory: %s", wd)

	var loaded bool
	var loadedPath string

	for _, envPath := range envPaths {
		// Check if file exists
		if _, err := os.Stat(envPath); err != nil {
			t.Logf("❌ Path not found: %s", envPath)
			continue
		}

		// Try to read file
		content, err := os.ReadFile(envPath)
		if err != nil {
			t.Logf("❌ Failed to read: %s - %v", envPath, err)
			continue
		}

		t.Logf("📄 Successfully read .env from: %s (%d bytes)", envPath, len(content))
		lines := strings.Split(string(content), "\n")
		loadedVars := []string{}

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Remove quotes if present
				value = strings.Trim(value, `"'`)
				os.Setenv(key, value)

				// Log LLM related variables (but hide API key)
				if strings.HasPrefix(key, "LLM_") {
					if key == "LLM_API_KEY" {
						// 安全地掩码 API Key
						var maskedKey string
						if len(value) <= 8 {
							maskedKey = strings.Repeat("*", len(value))
						} else {
							maskedKey = value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
						}
						loadedVars = append(loadedVars, fmt.Sprintf("%s=%s", key, maskedKey))
					} else {
						loadedVars = append(loadedVars, fmt.Sprintf("%s=%s", key, value))
					}
				}
			}
		}

		if len(loadedVars) > 0 {
			t.Logf("✅ Loaded %d environment variables from %s: %s", len(loadedVars), envPath, strings.Join(loadedVars, ", "))
			loaded = true
			loadedPath = envPath
			break
		} else {
			t.Logf("⚠️ File %s exists but no valid environment variables found", envPath)
		}
	}

	if !loaded {
		// List current directory files for debugging
		if files, err := os.ReadDir("."); err == nil {
			var fileNames []string
			for _, file := range files {
				if !file.IsDir() {
					fileNames = append(fileNames, file.Name())
				}
			}
			t.Logf("📁 Current directory files: %s", strings.Join(fileNames, ", "))
		}
		t.Skip("❌ No .env file found from any path and LLM_API_KEY not set in environment, skipping integration test")
	}

	// Verify critical environment variables
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		t.Fatalf("❌ .env file loaded (%s) but LLM_API_KEY not found", loadedPath)
	}

	t.Logf("🎯 Environment validation successful for %s", loadedPath)
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

	// 3. Setup MCP Client with intelligent provider detection
	mcpClient := mcp.New()
	llmURL := os.Getenv("LLM_API_URL")
	llmModel := os.Getenv("LLM_MODEL")
	llmProvider := strings.ToLower(os.Getenv("LLM_PROVIDER"))

	if llmModel == "" {
		llmModel = "glm-4-flash"
	}

	// Auto-detect provider based on URL or explicit setting
	if llmProvider == "" {
		if strings.Contains(llmURL, "bigmodel.cn") {
			llmProvider = "glm"
		} else {
			llmProvider = "deepseek" // default
		}
	}

	// Configure client based on provider
	switch llmProvider {
	case "glm":
		mcpClient.SetGLMAPIKey(apiKey, llmURL, llmModel)
		t.Logf("🔧 [MCP] 配置 GLM 模型: %s", llmModel)
	case "deepseek", "ds":
		mcpClient.SetDeepSeekAPIKey(apiKey, llmURL, llmModel)
		t.Logf("🔧 [MCP] 配置 DeepSeek 模型: %s", llmModel)
	case "qwen":
		mcpClient.SetQwenAPIKey(apiKey, llmURL, llmModel)
		t.Logf("🔧 [MCP] 配置 Qwen 模型: %s", llmModel)
	default:
		mcpClient.SetCustomAPI(llmURL, apiKey, llmModel)
		t.Logf("🔧 [MCP] 配置自定义 API: %s", llmProvider)
	}

	// Show configuration summary
	var maskedKey string
	if len(apiKey) <= 8 {
		maskedKey = strings.Repeat("*", len(apiKey))
	} else {
		maskedKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
	}
	t.Logf("✅ [MCP] %s 配置完成: URL=%s, Model=%s, Key=%s",
		llmProvider, llmURL, llmModel, maskedKey)

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

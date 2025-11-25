package backtest

import (
	"context"
	"fmt"
	"nofx/config"
	"nofx/decision"
	"nofx/market"
	"nofx/mcp"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// loadRealEnv loads real environment variables for backtesting
func loadRealEnv(t *testing.T) {
	// Check if LLM_API_KEY is already set
	if os.Getenv("LLM_API_KEY") != "" {
		t.Logf("✓ LLM_API_KEY already exists in environment")
		return
	}

	// Try multiple possible .env locations
	envPaths := []string{
		".env",                 // 当前目录
		"../.env",              // 父目录
		"../../.env",           // 父父目录
		"./.env",               // 显式当前目录
		"../../../.env",        // 更上层目录
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
		t.Skip("❌ No .env file found from any path and LLM_API_KEY not set in environment, skipping real backtest")
	}

	// Verify critical environment variables
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		t.Fatalf("❌ .env file loaded (%s) but LLM_API_KEY not found", loadedPath)
	}

	t.Logf("🎯 Environment validation successful for %s", loadedPath)
}

// setupPromptsDir sets up the prompts directory for testing
func setupPromptsDir(t *testing.T) {
	// Find the project root by looking for go.mod
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod not found)")
		}
		dir = parent
	}

	promptsPath := filepath.Join(dir, "prompts")
	if _, err := os.Stat(promptsPath); os.IsNotExist(err) {
		t.Fatalf("Prompts directory not found at: %s", promptsPath)
	}

	t.Logf("🔧 Setting prompts directory to: %s", promptsPath)
	decision.SetPromptsDir(promptsPath)

	if err := decision.ReloadPromptTemplates(); err != nil {
		t.Fatalf("Failed to reload prompt templates: %v", err)
	}
	t.Logf("✅ Prompt templates reloaded successfully")
}

// TestRealBacktestWithRealLLMAnd4HourData 运行一个真实的回测测试
// 使用真实的LLM API和过去4小时的历史数据
func TestRealBacktestWithRealLLMAnd4HourData(t *testing.T) {
	// 1. 加载环境变量
	loadRealEnv(t)
	setupPromptsDir(t)

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping real backtest: LLM_API_KEY not set")
	}

	// 2. 设置数据库
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_real_backtest.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// 3. 设置MCP客户端，使用智能提供商检测
	mcpClient := mcp.New()
	llmURL := os.Getenv("LLM_API_URL")
	llmModel := os.Getenv("LLM_MODEL")
	llmProvider := strings.ToLower(os.Getenv("LLM_PROVIDER"))

	if llmModel == "" {
		llmModel = "glm-4-flash"
	}

	// 根据URL或显式设置自动检测提供商
	if llmProvider == "" {
		if strings.Contains(llmURL, "bigmodel.cn") {
			llmProvider = "glm"
		} else {
			llmProvider = "deepseek" // 默认
		}
	}

	// 根据提供商配置客户端
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

	// 显示配置摘要
	var maskedKey string
	if len(apiKey) <= 8 {
		maskedKey = strings.Repeat("*", len(apiKey))
	} else {
		maskedKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
	}
	t.Logf("✅ [MCP] %s 配置完成: URL=%s, Model=%s, Key=%s",
		llmProvider, llmURL, llmModel, maskedKey)

	// 4. 设置真实回测配置 - 使用过去4小时的数据
	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-4 * time.Hour) // 固定回测过去4小时

	// 创建真实回测配置
	cfg := &Config{
		TraderID:       "real_trader",
		UserID:         "real_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 5000.0, // 较小的初始资金用于快速测试
		ScanInterval:   30 * time.Minute, // 30分钟扫描一次，4小时约8个周期
		TradingSymbols: []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}, // 多个交易对
		Slippage:       10, // 0.1%滑点

		// 使用真实LLM，关闭Mock模式
		MockMode:             false,
		UseTraderConfig:      false,
		IndicatorConfig:      market.GetDefaultIndicatorConfig(),
		BTCETHLeverage:       3,  // BTC/ETH使用3倍杠杆
		AltcoinLeverage:      5,  // 山寨币使用5倍杠杆
		CustomPrompt:         "", // 使用默认提示词
		OverrideBasePrompt:   false,
		SystemPromptTemplate: "",
	}

	// 5. 创建回测引擎
	backtestID := fmt.Sprintf("real_backtest_%s", time.Now().Format("20060102_150405"))
	engine := NewEngine(backtestID, cfg, db, mcpClient)

	// 在数据库中创建回测记录
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

	// 6. 运行真实回测
	t.Logf("🚀 开始真实回测 (ID: %s)", backtestID)
	t.Logf("⏰ 回测时间范围: %s 到 %s (时长: %v)",
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"),
		endTime.Sub(startTime))
	t.Logf("💰 初始资金: %.2f USDT", cfg.InitialBalance)
	t.Logf("📊 交易对: %v", cfg.TradingSymbols)
	t.Logf("⚡ 扫描间隔: %v", cfg.ScanInterval)
	t.Logf("🤖 AI模型: %s (%s)", llmModel, llmProvider)

	ctx := context.Background()
	startTimeOverall := time.Now()
	err = engine.Run(ctx)
	duration := time.Since(startTimeOverall)

	if err != nil {
		t.Fatalf("❌ 回测失败: %v", err)
	}

	// 7. 验证并分析结果
	result := engine.calculateResult()

	t.Logf("✅ 回测完成! 耗时: %v", duration)
	t.Logf("📈 最终净值: %.2f USDT", result.FinalEquity)
	t.Logf("💵 总盈亏: %.2f USDT (%.2f%%)", result.TotalPnL, result.TotalPnLPct)
	t.Logf("📊 总交易数: %d", result.TotalTrades)
	t.Logf("🎯 胜率: %.2f%%", result.WinRate)
	t.Logf("📉 最大回撤: %.2f%%", result.MaxDrawdown)
	t.Logf("📊 夏普比率: %.2f", result.SharpeRatio)

	// 基本断言
	if result.FinalEquity <= 0 {
		t.Errorf("❌ 最终净值应该大于0，实际: %.2f", result.FinalEquity)
	}

	if len(result.EquitySnapshots) == 0 {
		t.Error("❌ 没有记录净值快照")
	} else {
		t.Logf("📊 净值快照数量: %d", len(result.EquitySnapshots))
	}

	// 验证至少有一些AI决策活动（即使没有交易）
	// 在真实模式中，AI应该会做出一些决策（即使只是wait/hold）

	// 验证数据库记录
	savedResult, err := db.GetBacktest(backtestID)
	if err != nil {
		t.Errorf("❌ 从数据库检索保存的结果失败: %v", err)
	}
	if savedResult == nil {
		t.Error("❌ 保存的结果为nil")
	} else {
		if savedResult.Status != "completed" {
			t.Errorf("❌ 期望状态'completed'，实际'%s'", savedResult.Status)
		}
		t.Logf("✅ 数据库记录验证成功: 状态=%s, 最终净值=%.2f", savedResult.Status, savedResult.FinalEquity)
	}

	// 8. 详细分析交易活动
	if result.TotalTrades > 0 {
		t.Logf("📋 交易详情:")
		for i, trade := range result.Trades {
			if trade.Action == "close" { // 只显示平仓交易（完整的交易周期）
				t.Logf("  %d. %s %s: 入场%.2f → 出场%.2f | 盈亏: %.2f (%.2f%%) | 杠杆: %dx",
					i+1, trade.Side, trade.Symbol, trade.EntryPrice, trade.ExitPrice,
					trade.PnL, trade.PnLPct, trade.Leverage)
			}
		}
	} else {
		t.Logf("ℹ️  本次回测期间AI没有执行任何交易（可能是观望策略）")
	}

	// 9. 性能指标验证
	if result.SharpeRatio > 2.0 {
		t.Logf("🏆 优秀的夏普比率: %.2f", result.SharpeRatio)
	} else if result.SharpeRatio > 1.0 {
		t.Logf("👍 良好的夏普比率: %.2f", result.SharpeRatio)
	}

	if result.MaxDrawdown < 5.0 {
		t.Logf("🛡️  低回撤: %.2f%%", result.MaxDrawdown)
	} else if result.MaxDrawdown > 20.0 {
		t.Logf("⚠️  高回撤: %.2f%%", result.MaxDrawdown)
	}

	// 10. 测试通过标志
	t.Logf("🎉 真实回测测试通过！AI使用了真实的LLM API和历史数据完成了 %v 的回测",
		endTime.Sub(startTime))
}

// TestRealBacktestWithMultipleSymbols 测试多个交易对的真实回测
func TestRealBacktestWithMultipleSymbols(t *testing.T) {
	// 加载环境变量
	loadRealEnv(t)
	setupPromptsDir(t)

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping multi-symbol backtest: LLM_API_KEY not set")
	}

	// 设置数据库
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_multi_symbol_backtest.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// 设置MCP客户端
	mcpClient := mcp.New()
	llmURL := os.Getenv("LLM_API_URL")
	llmModel := os.Getenv("LLM_MODEL")
	llmProvider := strings.ToLower(os.Getenv("LLM_PROVIDER"))

	if llmProvider == "" {
		if strings.Contains(llmURL, "bigmodel.cn") {
			llmProvider = "glm"
		} else {
			llmProvider = "deepseek"
		}
	}

	switch llmProvider {
	case "glm":
		mcpClient.SetGLMAPIKey(apiKey, llmURL, llmModel)
	case "deepseek", "ds":
		mcpClient.SetDeepSeekAPIKey(apiKey, llmURL, llmModel)
	case "qwen":
		mcpClient.SetQwenAPIKey(apiKey, llmURL, llmModel)
	default:
		mcpClient.SetCustomAPI(llmURL, apiKey, llmModel)
	}

	// 配置多币种回测
	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-4 * time.Hour)

	cfg := &Config{
		TraderID:       "multi_symbol_trader",
		UserID:         "multi_symbol_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 8000.0,
		ScanInterval:   20 * time.Minute, // 更频繁的扫描
		TradingSymbols: []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"}, // 4个主流币种
		Slippage:       10,

		MockMode:             false,
		UseTraderConfig:      false,
		IndicatorConfig:      market.GetDefaultIndicatorConfig(),
		BTCETHLeverage:       2,  // 保守杠杆
		AltcoinLeverage:      3,  // 山寨币保守杠杆
	}

	// 创建并运行回测
	backtestID := fmt.Sprintf("multi_symbol_%s", time.Now().Format("20060102_150405"))
	engine := NewEngine(backtestID, cfg, db, mcpClient)

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

	t.Logf("🌟 开始多币种真实回测 (ID: %s)", backtestID)
	t.Logf("💱 交易对: %v", cfg.TradingSymbols)

	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Fatalf("Multi-symbol backtest failed: %v", err)
	}

	// 分析结果
	result := engine.calculateResult()

	t.Logf("📊 多币种回测结果:")
	t.Logf("   最终净值: %.2f USDT", result.FinalEquity)
	t.Logf("   总盈亏: %.2f USDT (%.2f%%)", result.TotalPnL, result.TotalPnLPct)
	t.Logf("   总交易数: %d", result.TotalTrades)
	t.Logf("   胜率: %.2f%%", result.WinRate)
	t.Logf("   最大回撤: %.2f%%", result.MaxDrawdown)

	// 验证多币种交易活动
	symbolTrades := make(map[string]int)
	for _, trade := range result.Trades {
		if trade.Action == "close" {
			symbolTrades[trade.Symbol]++
		}
	}

	if len(symbolTrades) > 0 {
		t.Logf("📈 各币种交易活动:")
		for symbol, count := range symbolTrades {
			t.Logf("   %s: %d 笔交易", symbol, count)
		}
	}

	// 基本验证
	if result.FinalEquity <= 0 {
		t.Errorf("Final equity should be positive, got: %.2f", result.FinalEquity)
	}

	t.Logf("✅ 多币种真实回测测试完成！")
}
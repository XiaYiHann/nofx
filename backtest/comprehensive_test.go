package backtest

import (
	"context"
	"fmt"
	"nofx/config"
	"nofx/market"
	"nofx/mcp"
	"nofx/mockllm"
	"path/filepath"
	"testing"
	"time"
)

// ============================================================================
// 综合回测测试套件
// 包含真实LLM测试、Mock测试、边界条件测试、性能测试
// ============================================================================

// TestComprehensiveBacktestSuite 运行完整的回测测试套件
func TestComprehensiveBacktestSuite(t *testing.T) {
	// 子测试
	t.Run("MockMode", testBacktestWithMockMode)
	t.Run("RealLLM_NVIDIA", testBacktestWithRealLLM_NVIDIA)
	t.Run("EdgeCases", testBacktestEdgeCases)
	t.Run("Performance", testBacktestPerformance)
	t.Run("MultipleTimeframes", testBacktestMultipleTimeframes)
	t.Run("DifferentLeverages", testBacktestDifferentLeverages)
}

// ============================================================================
// 1. Mock模式测试
// ============================================================================

func testBacktestWithMockMode(t *testing.T) {
	t.Helper()

	// 1. 设置数据库
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_mock_backtest.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 2. 设置MCP客户端 - 使用Mock模式
	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>Mock reasoning for testing</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"open_long\", \"confidence\": 80, \"reasoning\": \"Mock test decision\"}]\n```</decision>",
	)

	// 3. 创建回测配置
	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-2 * time.Hour)

	cfg := &Config{
		TraderID:       "mock_trader",
		UserID:         "mock_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   15 * time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10,
		MockMode:       true, // 启用Mock模式
		Timeframe:      "3m",
		DataPoints:     100,
		BTCETHLeverage: 5,
		AltcoinLeverage: 5,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	// 4. 创建回测引擎
	backtestID := fmt.Sprintf("mock_test_%s", time.Now().Format("20060102_150405"))
	engine := NewEngine(backtestID, cfg, db, mcpClient)

	// 5. 在数据库中创建回测记录
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

	// 6. 运行回测
	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Fatalf("Mock backtest failed: %v", err)
	}

	// 7. 验证结果
	result := engine.calculateResult()

	// 基本断言
	if result.FinalEquity <= 0 {
		t.Errorf("Final equity should be positive, got: %.2f", result.FinalEquity)
	}

	if len(result.EquitySnapshots) == 0 {
		t.Error("No equity snapshots recorded")
	}

	// 验证数据库记录
	savedResult, err := db.GetBacktest(backtestID)
	if err != nil {
		t.Errorf("Failed to retrieve saved result: %v", err)
	}
	if savedResult == nil {
		t.Error("Saved result is nil")
	} else if savedResult.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", savedResult.Status)
	}

	t.Logf("✅ Mock模式测试通过: 最终净值=%.2f, 交易数=%d",
		result.FinalEquity, result.TotalTrades)
}

// ============================================================================
// 2. 真实LLM测试（NVIDIA API）
// ============================================================================

func testBacktestWithRealLLM_NVIDIA(t *testing.T) {
	t.Helper()

	// 硬编码NVIDIA API配置
	apiKey := "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"
	baseURL := "https://integrate.api.nvidia.com/v1"
	model := "qwen/qwen3-next-80b-a3b-instruct"

	// 1. 设置数据库
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_nvidia_backtest.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// 2. 设置MCP客户端 - 使用NVIDIA API
	mcpClient := mcp.New()
	mcpClient.SetCustomAPI(
		baseURL,
		apiKey,
		model,
	)

	// 3. 创建回测配置
	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-1 * time.Hour) // 短时间测试

	cfg := &Config{
		TraderID:       "nvidia_trader",
		UserID:         "nvidia_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 5000.0,
		ScanInterval:   15 * time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10,
		MockMode:       false, // 使用真实LLM
		Timeframe:      "3m",
		DataPoints:     100,
		BTCETHLeverage: 3,
		AltcoinLeverage: 5,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	// 4. 创建回测引擎
	backtestID := fmt.Sprintf("nvidia_test_%s", time.Now().Format("20060102_150405"))
	engine := NewEngine(backtestID, cfg, db, mcpClient)

	// 5. 在数据库中创建回测记录
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

	// 6. 运行回测
	ctx := context.Background()
	startTimeRun := time.Now()
	err = engine.Run(ctx)
	duration := time.Since(startTimeRun)

	if err != nil {
		t.Fatalf("NVIDIA LLM backtest failed: %v", err)
	}

	// 7. 验证结果
	result := engine.calculateResult()

	if result.FinalEquity <= 0 {
		t.Errorf("Final equity should be positive, got: %.2f", result.FinalEquity)
	}

	// 8. 验证决策记录
	decisions, err := db.GetBacktestDecisions(backtestID)
	if err != nil {
		t.Errorf("Failed to get decisions: %v", err)
	}
	if len(decisions) == 0 {
		t.Error("No decisions recorded")
	}

	t.Logf("✅ NVIDIA LLM测试通过: 最终净值=%.2f, 决策数=%d, 耗时=%v",
		result.FinalEquity, len(decisions), duration)
}

// ============================================================================
// 3. 边界条件测试
// ============================================================================

func testBacktestEdgeCases(t *testing.T) {
	t.Helper()

	t.Run("SmallTimeRange", func(t *testing.T) {
		testBacktestWithSmallTimeRange(t)
	})
	t.Run("SmallBalance", func(t *testing.T) {
		testBacktestWithSmallBalance(t)
	})
	t.Run("HighSlippage", func(t *testing.T) {
		testBacktestWithHighSlippage(t)
	})
	t.Run("MultipleSymbols", func(t *testing.T) {
		testBacktestWithMultipleSymbols(t)
	})
}

func testBacktestWithSmallTimeRange(t *testing.T) {
	// 测试极短时间范围（15分钟）
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_small_time.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>Small time range test</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"wait\", \"confidence\": 100}]\n```</decision>",
	)

	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-15 * time.Minute)

	cfg := &Config{
		TraderID:       "small_time_trader",
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   5 * time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10,
		MockMode:       true,
		Timeframe:      "3m",
		DataPoints:     50,
		BTCETHLeverage: 5,
		AltcoinLeverage: 5,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	backtestID := fmt.Sprintf("small_time_%s", time.Now().Format("20060102_150405"))
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

	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Errorf("Small time range backtest failed: %v", err)
	}

	result := engine.calculateResult()
	t.Logf("✅ 小时间范围测试通过: 最终净值=%.2f", result.FinalEquity)
}

func testBacktestWithSmallBalance(t *testing.T) {
	// 测试小资金（100 USDT）
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_small_balance.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>Small balance test</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"wait\", \"confidence\": 100}]\n```</decision>",
	)

	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-1 * time.Hour)

	cfg := &Config{
		TraderID:       "small_balance_trader",
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 100.0, // 小资金
		ScanInterval:   15 * time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10,
		MockMode:       true,
		Timeframe:      "3m",
		DataPoints:     100,
		BTCETHLeverage: 1, // 低杠杆
		AltcoinLeverage: 1,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	backtestID := fmt.Sprintf("small_balance_%s", time.Now().Format("20060102_150405"))
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

	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Errorf("Small balance backtest failed: %v", err)
	}

	result := engine.calculateResult()
	if result.FinalEquity < 0 {
		t.Errorf("Final equity should not be negative, got: %.2f", result.FinalEquity)
	}

	t.Logf("✅ 小资金测试通过: 初始=%.2f, 最终=%.2f", cfg.InitialBalance, result.FinalEquity)
}

func testBacktestWithHighSlippage(t *testing.T) {
	// 测试高滑点环境（100 bps = 1%）
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_high_slippage.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>High slippage test</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"open_long\", \"confidence\": 80}]\n```</decision>",
	)

	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-1 * time.Hour)

	cfg := &Config{
		TraderID:       "high_slippage_trader",
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   15 * time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       100, // 高滑点
		MockMode:       true,
		Timeframe:      "3m",
		DataPoints:     100,
		BTCETHLeverage: 5,
		AltcoinLeverage: 5,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	backtestID := fmt.Sprintf("high_slippage_%s", time.Now().Format("20060102_150405"))
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

	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Errorf("High slippage backtest failed: %v", err)
	}

	result := engine.calculateResult()
	t.Logf("✅ 高滑点测试通过: 最终净值=%.2f, 最大回撤=%.2f%%",
		result.FinalEquity, result.MaxDrawdown)
}

func testBacktestWithMultipleSymbols(t *testing.T) {
	// 测试多币种
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_multi_symbol.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>Multiple symbols test</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"wait\", \"confidence\": 100}]\n```</decision>",
	)

	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-1 * time.Hour)

	cfg := &Config{
		TraderID:       "multi_symbol_trader",
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   15 * time.Minute,
		TradingSymbols: []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}, // 多币种
		Slippage:       10,
		MockMode:       true,
		Timeframe:      "3m",
		DataPoints:     100,
		BTCETHLeverage: 3,
		AltcoinLeverage: 5,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

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

	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Errorf("Multiple symbols backtest failed: %v", err)
	}

	result := engine.calculateResult()
	t.Logf("✅ 多币种测试通过: 最终净值=%.2f, 交易数=%d",
		result.FinalEquity, result.TotalTrades)
}

// ============================================================================
// 4. 性能测试
// ============================================================================

func testBacktestPerformance(t *testing.T) {
	t.Helper()

	t.Run("LongTimeRange", func(t *testing.T) {
		testBacktestWithLongTimeRange(t)
	})
	t.Run("HighFrequency", func(t *testing.T) {
		testBacktestWithHighFrequency(t)
	})
}

func testBacktestWithLongTimeRange(t *testing.T) {
	// 测试长时间范围（24小时）
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_long_time.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>Long time range test</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"wait\", \"confidence\": 100}]\n```</decision>",
	)

	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-24 * time.Hour)

	cfg := &Config{
		TraderID:       "long_time_trader",
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   30 * time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10,
		MockMode:       true,
		Timeframe:      "3m",
		DataPoints:     100,
		BTCETHLeverage: 5,
		AltcoinLeverage: 5,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	backtestID := fmt.Sprintf("long_time_%s", time.Now().Format("20060102_150405"))
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

	ctx := context.Background()
	startTimeRun := time.Now()
	err = engine.Run(ctx)
	duration := time.Since(startTimeRun)

	if err != nil {
		t.Errorf("Long time range backtest failed: %v", err)
	}

	result := engine.calculateResult()
	t.Logf("✅ 长时间范围测试通过: 耗时=%v, 最终净值=%.2f, 净值快照数=%d",
		duration, result.FinalEquity, len(result.EquitySnapshots))
}

func testBacktestWithHighFrequency(t *testing.T) {
	// 测试高频扫描（1分钟间隔）
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_high_freq.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>High frequency test</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"wait\", \"confidence\": 100}]\n```</decision>",
	)

	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-30 * time.Minute)

	cfg := &Config{
		TraderID:       "high_freq_trader",
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   1 * time.Minute, // 高频
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10,
		MockMode:       true,
		Timeframe:      "1m", // 短周期
		DataPoints:     100,
		BTCETHLeverage: 5,
		AltcoinLeverage: 5,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	backtestID := fmt.Sprintf("high_freq_%s", time.Now().Format("20060102_150405"))
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

	ctx := context.Background()
	startTimeRun := time.Now()
	err = engine.Run(ctx)
	duration := time.Since(startTimeRun)

	if err != nil {
		t.Errorf("High frequency backtest failed: %v", err)
	}

	result := engine.calculateResult()
	t.Logf("✅ 高频测试通过: 耗时=%v, 最终净值=%.2f, 周期数=%d",
		duration, result.FinalEquity, len(result.EquitySnapshots))
}

// ============================================================================
// 5. 多时间周期测试
// ============================================================================

func testBacktestMultipleTimeframes(t *testing.T) {
	t.Helper()

	timeframes := []string{"1m", "3m", "5m", "15m", "1h"}

	for _, tf := range timeframes {
		t.Run(tf, func(t *testing.T) {
			testBacktestWithTimeframe(t, tf)
		})
	}
}

func testBacktestWithTimeframe(t *testing.T, timeframe string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, fmt.Sprintf("test_tf_%s.db", timeframe))
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>Timeframe test</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"wait\", \"confidence\": 100}]\n```</decision>",
	)

	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-1 * time.Hour)

	cfg := &Config{
		TraderID:       fmt.Sprintf("tf_%s_trader", timeframe),
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   15 * time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10,
		MockMode:       true,
		Timeframe:      timeframe,
		DataPoints:     100,
		BTCETHLeverage: 5,
		AltcoinLeverage: 5,
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	backtestID := fmt.Sprintf("tf_%s_%s", timeframe, time.Now().Format("20060102_150405"))
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

	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Errorf("Timeframe %s backtest failed: %v", timeframe, err)
	}

	result := engine.calculateResult()
	t.Logf("✅ %s周期测试通过: 最终净值=%.2f", timeframe, result.FinalEquity)
}

// ============================================================================
// 6. 不同杠杆配置测试
// ============================================================================

func testBacktestDifferentLeverages(t *testing.T) {
	t.Helper()

	t.Run("Conservative", func(t *testing.T) {
		testBacktestWithLeverage(t, 1, 2)
	})
	t.Run("Moderate", func(t *testing.T) {
		testBacktestWithLeverage(t, 5, 10)
	})
	t.Run("Aggressive", func(t *testing.T) {
		testBacktestWithLeverage(t, 20, 30)
	})
}

func testBacktestWithLeverage(t *testing.T, btcLeverage, altLeverage int) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, fmt.Sprintf("test_leverage_%d_%d.db", btcLeverage, altLeverage))
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mcpClient := mockllm.NewMCPClientFromMockServer(
		t,
		"<reasoning>Leverage test</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"open_long\", \"confidence\": 80}]\n```</decision>",
	)

	endTime := time.Now().Truncate(time.Minute)
	startTime := endTime.Add(-1 * time.Hour)

	cfg := &Config{
		TraderID:       fmt.Sprintf("leverage_%d_%d_trader", btcLeverage, altLeverage),
		UserID:         "test_user",
		StartTime:      startTime,
		EndTime:        endTime,
		InitialBalance: 10000.0,
		ScanInterval:   15 * time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
		Slippage:       10,
		MockMode:       true,
		Timeframe:      "3m",
		DataPoints:     100,
		BTCETHLeverage: btcLeverage,
		AltcoinLeverage: float64(altLeverage),
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}

	backtestID := fmt.Sprintf("leverage_%d_%d_%s", btcLeverage, altLeverage, time.Now().Format("20060102_150405"))
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

	ctx := context.Background()
	err = engine.Run(ctx)
	if err != nil {
		t.Errorf("Leverage %d/%d backtest failed: %v", btcLeverage, altLeverage, err)
	}

	result := engine.calculateResult()
	t.Logf("✅ 杠杆 %d/%d 测试通过: 最终净值=%.2f, 最大回撤=%.2f%%",
		btcLeverage, altLeverage, result.FinalEquity, result.MaxDrawdown)
}
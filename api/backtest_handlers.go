package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"nofx/backtest"
	"nofx/config"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BacktestConfigRequest 回测配置请求
type BacktestConfigRequest struct {
	TraderID             string                 `json:"trader_id" binding:"required"`
	StartTime            string                 `json:"start_time" binding:"required"`
	EndTime              string                 `json:"end_time" binding:"required"`
	InitialBalance       float64                `json:"initial_balance" binding:"required,gt=0"`
	UseTraderConfig      *bool                  `json:"use_trader_config"`
	MockMode             bool                   `json:"mock_mode"`
	IndicatorConfig      map[string]interface{} `json:"indicator_config,omitempty"`
	CustomPrompt         string                 `json:"custom_prompt,omitempty"`
	SystemPromptTemplate string                 `json:"system_prompt_template,omitempty"`
	TradingSymbols       string                 `json:"trading_symbols,omitempty"`
}

// handleCreateBacktest 创建回测
func (s *Server) handleCreateBacktest(c *gin.Context) {
	userID := c.GetString("user_id")

	var req BacktestConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 解析时间
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_time format"})
		return
	}

	// 验证时间范围
	if !startTime.Before(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_time must be before end_time"})
		return
	}

	if endTime.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_time cannot be in the future"})
		return
	}

	// 默认使用trader配置
	useTraderConfig := true
	if req.UseTraderConfig != nil {
		useTraderConfig = *req.UseTraderConfig
	}

	// 验证trader存在
	trader, aiModel, _, err := s.database.GetTraderConfig(userID, req.TraderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trader not found"})
		return
	}

	// Handle Mock Mode by using CustomPrompt as a storage mechanism
	customPrompt := req.CustomPrompt
	if useTraderConfig {
		customPrompt = trader.CustomPrompt
	}
	if req.MockMode {
		customPrompt = "MOCK_MODE_ALWAYS_LONG"
	}

	// Determine trading symbols
	tradingSymbols := req.TradingSymbols
	if tradingSymbols == "" {
		// 强制使用固定的主流币种进行回测
		tradingSymbols = "BTCUSDT,ETHUSDT,SOLUSDT,BNBUSDT,XRPUSDT,DOGEUSDT"
	}

	// Prepare indicator config JSON string
	var indicatorConfigJSON string
	if req.IndicatorConfig != nil {
		indicatorConfigJSON = fmt.Sprintf("%v", req.IndicatorConfig)
	}

	// 创建回测记录
	backtestID := uuid.New().String()
	backtestRun := &config.BacktestRun{
		ID:                  backtestID,
		UserID:              userID,
		TraderID:            req.TraderID,
		StartTime:           startTime,
		EndTime:             endTime,
		InitialBalance:      req.InitialBalance,
		ScanIntervalMinutes: trader.ScanIntervalMinutes,
		TradingSymbols:      tradingSymbols,
		UseTraderConfig:     useTraderConfig,
		Status:              "pending",
		Progress:            0,
		CreatedAt:           time.Now(),
	}

	// 如果使用trader配置,复制配置
	if useTraderConfig {
		backtestRun.IndicatorConfig = trader.IndicatorConfig
		backtestRun.CustomPrompt = customPrompt
		backtestRun.OverrideBasePrompt = trader.OverrideBasePrompt
		backtestRun.SystemPromptTemplate = trader.SystemPromptTemplate
	} else {
		// 使用自定义配置
		backtestRun.IndicatorConfig = indicatorConfigJSON
		backtestRun.CustomPrompt = customPrompt
		backtestRun.SystemPromptTemplate = req.SystemPromptTemplate
	}

	// 保存到数据库
	if err := s.database.CreateBacktest(backtestRun); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backtest"})
		return
	}

	// 启动异步回测引擎
	go s.runBacktest(backtestID, backtestRun, trader, aiModel)

	c.JSON(http.StatusCreated, gin.H{
		"backtest_id": backtestID,
		"status":      "pending",
		"message":     "Backtest started successfully",
		"config": gin.H{
			"trader_id":         req.TraderID,
			"trader_name":       trader.Name,
			"start_time":        startTime,
			"end_time":          endTime,
			"initial_balance":   req.InitialBalance,
			"use_trader_config": useTraderConfig,
		},
	})
}

// handleGetBacktest 获取回测详情
func (s *Server) handleGetBacktest(c *gin.Context) {
	backtestID := c.Param("id")

	backtest, err := s.database.GetBacktest(backtestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backtest not found"})
		return
	}

	c.JSON(http.StatusOK, backtest)
}

// handleGetBacktestEquityHistory 获取回测净值历史
func (s *Server) handleGetBacktestEquityHistory(c *gin.Context) {
	backtestID := c.Param("id")

	snapshots, err := s.database.GetEquitySnapshots(backtestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get equity history"})
		return
	}

	c.JSON(http.StatusOK, snapshots)
}

// handleGetBacktestTrades 获取回测交易记录
func (s *Server) handleGetBacktestTrades(c *gin.Context) {
	backtestID := c.Param("id")

	trades, err := s.database.GetBacktestTrades(backtestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trades"})
		return
	}

	c.JSON(http.StatusOK, trades)
}

// handleListBacktests 列出回测记录
func (s *Server) handleListBacktests(c *gin.Context) {
	userID := c.GetString("user_id")

	backtests, err := s.database.ListBacktests(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list backtests"})
		return
	}

	c.JSON(http.StatusOK, backtests)
}

// handleDeleteBacktest 删除回测记录
func (s *Server) handleDeleteBacktest(c *gin.Context) {
	backtestID := c.Param("id")

	if err := s.database.DeleteBacktest(backtestID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete backtest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Backtest deleted successfully"})
}

// handleGetBacktestDecisions 获取回测决策记录
func (s *Server) handleGetBacktestDecisions(c *gin.Context) {
	backtestID := c.Param("id")
	if backtestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Backtest ID is required"})
		return
	}

	// 1. 尝试从文件系统读取实时日志（优先）
	// 这种方式可以获取到最新的决策，包括"wait"等不会写入数据库的中间状态
	// 并且格式与实盘交易一致
	logDir := fmt.Sprintf("decision_logs/backtest_%s", backtestID)
	decisionLogger := logger.NewDecisionLogger(logDir)
	fileRecords, err := decisionLogger.GetLatestRecords(100) // 获取最近100条
	if err == nil && len(fileRecords) > 0 {
		// 将文件记录转换为前端需要的格式
		var decisions []config.BacktestDecision
		for _, record := range fileRecords {
			for _, action := range record.Decisions {
				decisions = append(decisions, config.BacktestDecision{
					BacktestID: backtestID,
					Symbol:     action.Symbol,
					Action:     action.Action,
					Price:      action.Price,
					Quantity:   action.Quantity,
					Leverage:   action.Leverage,
					Confidence: action.Confidence,
					Reasoning:  action.Reasoning,
					Timestamp:  action.Timestamp,
				})
			}
		}
		// 如果成功从文件读取到数据，直接返回
		if len(decisions) > 0 {
			c.JSON(http.StatusOK, decisions)
			return
		}
	}

	// 2. 如果文件读取失败或为空，回退到数据库读取
	decisions, err := s.database.GetBacktestDecisions(backtestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get backtest decisions: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, decisions)
}

// runBacktest 异步执行回测
func (s *Server) runBacktest(backtestID string, backtestRun *config.BacktestRun, trader *config.TraderRecord, aiModel *config.AIModelConfig) {
	log.Printf("[Backtest %s] Starting backtest execution", backtestID)

	// 更新状态为运行中
	if err := s.database.UpdateBacktestStatus(backtestID, "running", 0); err != nil {
		log.Printf("[Backtest %s] Failed to update status: %v", backtestID, err)
		return
	}

	// 解析指标配置
	var indicatorConfig *market.IndicatorConfig
	if backtestRun.IndicatorConfig != "" {
		var err error
		indicatorConfig, err = backtest.ParseIndicatorConfig(backtestRun.IndicatorConfig)
		if err != nil {
			log.Printf("[Backtest %s] Failed to parse indicator config: %v", backtestID, err)
			s.database.UpdateBacktestStatus(backtestID, "failed", 0)
			return
		}
	}

	// 解析交易币种
	tradingSymbols := []string{}
	if backtestRun.TradingSymbols != "" {
		tradingSymbols = strings.Split(backtestRun.TradingSymbols, ",")
		for i := range tradingSymbols {
			tradingSymbols[i] = strings.TrimSpace(tradingSymbols[i])
		}
	}

	// 构建回测配置
	cfg := &backtest.Config{
		TraderID:             backtestRun.TraderID,
		UserID:               backtestRun.UserID,
		StartTime:            backtestRun.StartTime,
		EndTime:              backtestRun.EndTime,
		InitialBalance:       backtestRun.InitialBalance,
		ScanInterval:         time.Duration(backtestRun.ScanIntervalMinutes) * time.Minute,
		TradingSymbols:       tradingSymbols,
		Slippage:             10, // 默认10基点(0.1%)滑点
		UseTraderConfig:      backtestRun.UseTraderConfig,
		MockMode:             backtestRun.CustomPrompt == "MOCK_MODE_ALWAYS_LONG", // Detect MockMode from CustomPrompt
		IndicatorConfig:      indicatorConfig,                                     // Use the parsed indicatorConfig
		CustomPrompt:         backtestRun.CustomPrompt,
		OverrideBasePrompt:   backtestRun.OverrideBasePrompt,
		SystemPromptTemplate: backtestRun.SystemPromptTemplate,
		BTCETHLeverage:       trader.BTCETHLeverage,
		AltcoinLeverage:      float64(trader.AltcoinLeverage),
	}

	// 创建MCP客户端
	mcpClient := mcp.New()
	switch aiModel.Provider {
	case "openai", "custom":
		mcpClient.SetCustomAPI(aiModel.CustomAPIURL, aiModel.APIKey, aiModel.CustomModelName)
	case "qwen":
		mcpClient.SetQwenAPIKey(aiModel.APIKey, aiModel.CustomAPIURL, aiModel.CustomModelName)
	case "deepseek":
		mcpClient.SetDeepSeekAPIKey(aiModel.APIKey, aiModel.CustomAPIURL, aiModel.CustomModelName)
	default:
		log.Printf("[Backtest %s] Unknown AI model provider: %s", backtestID, aiModel.Provider)
		s.database.UpdateBacktestStatus(backtestID, "failed", 0)
		return
	}

	// 创建回测引擎
	engine := backtest.NewEngine(backtestID, cfg, s.database, mcpClient)

	// 执行回测
	ctx := context.Background()
	if err := engine.Run(ctx); err != nil {
		log.Printf("[Backtest %s] Execution failed: %v", backtestID, err)
		s.database.UpdateBacktestStatus(backtestID, "failed", 0)
		return
	}

	log.Printf("[Backtest %s] Execution completed successfully", backtestID)
}

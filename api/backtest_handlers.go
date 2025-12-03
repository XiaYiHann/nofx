package api

import (
	"context"
	"encoding/json"
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
	TraderID             string                  `json:"trader_id"` // Optional for standalone mode
	StartTime            string                  `json:"start_time" binding:"required"`
	EndTime              string                  `json:"end_time" binding:"required"`
	InitialBalance       float64                 `json:"initial_balance" binding:"required,gt=0"`
	UseTraderConfig      *bool                   `json:"use_trader_config"`
	MockMode             bool                    `json:"mock_mode"`
	IndicatorConfig      *market.IndicatorConfig `json:"indicator_config,omitempty"`
	CustomPrompt         string                  `json:"custom_prompt,omitempty"`
	SystemPromptTemplate string                  `json:"system_prompt_template,omitempty"`
	TradingSymbols       string                  `json:"trading_symbols,omitempty"`
	// 新增回测配置字段
	AiModelID           string `json:"ai_model_id,omitempty"`
	ExchangeID          string `json:"exchange_id,omitempty"` // Standalone mode only
	Timeframe           string `json:"timeframe,omitempty"`
	DataPoints          int    `json:"data_points,omitempty"`
	PreheatHours        int    `json:"preheat_hours,omitempty"`
	ScanIntervalMinutes int    `json:"scan_interval_minutes,omitempty"`
	Slippage            int    `json:"slippage,omitempty"`
	BTCETHLeverage      int    `json:"btc_eth_leverage,omitempty"` // Standalone mode
	AltcoinLeverage     int    `json:"altcoin_leverage,omitempty"` // Standalone mode
	OverrideBasePrompt  bool   `json:"override_base_prompt,omitempty"`
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

	var trader *config.TraderRecord
	var aiModel *config.AIModelConfig


	if req.TraderID != "" {
		// 验证trader存在
		var exchange *config.ExchangeConfig
		trader, aiModel, exchange, err = s.database.GetTraderConfig(userID, req.TraderID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Trader not found"})
			return
		}
		// Ensure exchange is loaded if needed, though runBacktest doesn't use it directly yet
		_ = exchange
	} else {
		// Standalone Mode
		if req.AiModelID == "" || req.ExchangeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ai_model_id and exchange_id are required for standalone backtest"})
			return
		}

		// Fetch AI Model
		aiModel, err = s.database.GetAIModelByID(userID, req.AiModelID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "AI Model not found"})
			return
		}

		// Find or create placeholder trader for FK constraint
		placeholderID := fmt.Sprintf("standalone_%s", userID)
		trader, _, _, err = s.database.GetTraderConfig(userID, placeholderID)
		if err != nil {
			// Create placeholder trader
			trader = &config.TraderRecord{
				ID:             placeholderID,
				UserID:         userID,
				Name:           "Standalone Backtest",
				ExchangeID:     req.ExchangeID,
				AIModelID:      req.AiModelID,
				InitialBalance: req.InitialBalance,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := s.database.CreateTrader(trader); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create placeholder trader"})
				return
			}
		}

		// Override trader config with request values for this run
		// We create a copy to avoid modifying the cached/DB record
		traderCopy := *trader
		traderCopy.ExchangeID = req.ExchangeID
		traderCopy.AIModelID = req.AiModelID
		if req.BTCETHLeverage > 0 {
			traderCopy.BTCETHLeverage = req.BTCETHLeverage
		} else {
			traderCopy.BTCETHLeverage = 5 // Default
		}
		if req.AltcoinLeverage > 0 {
			traderCopy.AltcoinLeverage = req.AltcoinLeverage
		} else {
			traderCopy.AltcoinLeverage = 5 // Default
		}
		
		trader = &traderCopy
		req.TraderID = placeholderID // Set for BacktestRun FK
		useTraderConfig = false // Force false for standalone
	}

	// 1. 确定基础配置 (从 Trader 继承或使用默认值)
	// 扫描间隔
	scanInterval := trader.ScanIntervalMinutes
	if req.ScanIntervalMinutes > 0 {
		scanInterval = req.ScanIntervalMinutes
	} else if scanInterval <= 0 {
		scanInterval = 3 // Default fallback
	}

	// 交易币种
	tradingSymbols := trader.TradingSymbols
	if req.TradingSymbols != "" {
		tradingSymbols = req.TradingSymbols
	}
	if tradingSymbols == "" {
		tradingSymbols = "BTCUSDT,ETHUSDT,SOLUSDT,BNBUSDT,XRPUSDT,DOGEUSDT"
	}

	// 其他高级参数 (优先使用请求参数，否则使用默认值)
	timeframe := req.Timeframe
	if timeframe == "" {
		timeframe = "3m"
	}
	dataPoints := req.DataPoints
	if dataPoints <= 0 {
		dataPoints = 100
	}
	preheatHours := req.PreheatHours
	if preheatHours <= 0 {
		preheatHours = 12
	}
	slippage := req.Slippage
	if slippage < 0 {
		slippage = 10 // default 10 bps
	}

	// 2. 确定策略配置 (Prompt, Indicator, SystemTemplate)
	var indicatorConfigJSON string
	var customPrompt string
	var systemPromptTemplate string
	var overrideBasePrompt bool

	if useTraderConfig {
		// 沿用 Trader 配置
		indicatorConfigJSON = trader.IndicatorConfig
		customPrompt = trader.CustomPrompt
		systemPromptTemplate = trader.SystemPromptTemplate
		overrideBasePrompt = trader.OverrideBasePrompt
	} else {
		// 使用请求中的自定义配置
		if req.IndicatorConfig != nil {
			jsonBytes, err := json.Marshal(req.IndicatorConfig)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid indicator_config"})
				return
			}
			indicatorConfigJSON = string(jsonBytes)
		}
		customPrompt = req.CustomPrompt
		systemPromptTemplate = req.SystemPromptTemplate
		overrideBasePrompt = req.OverrideBasePrompt
	}

	// Handle Mock Mode
	if req.MockMode {
		customPrompt = "MOCK_MODE_ALWAYS_LONG"
	}

	// 创建回测记录
	backtestID := uuid.New().String()
	backtestRun := &config.BacktestRun{
		ID:                   backtestID,
		UserID:               userID,
		TraderID:             req.TraderID,
		StartTime:            startTime,
		EndTime:              endTime,
		InitialBalance:       req.InitialBalance,
		ScanIntervalMinutes:  scanInterval,
		TradingSymbols:       tradingSymbols,
		UseTraderConfig:      useTraderConfig,
		Status:               "pending",
		Progress:             0,
		CreatedAt:            time.Now(),
		AiModelID:            req.AiModelID,
		Timeframe:            timeframe,
		DataPoints:           dataPoints,
		PreheatHours:         preheatHours,
		Slippage:             slippage,
		IndicatorConfig:      indicatorConfigJSON,
		CustomPrompt:         customPrompt,
		OverrideBasePrompt:   overrideBasePrompt,
		SystemPromptTemplate: systemPromptTemplate,
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

	// 1. 优先尝试从日志文件读取 (实时性更好)
	// 这种方式可以获取到最新的决策，包括"wait"等不会写入数据库的中间状态
	// 并且格式与实盘交易一致
	logDir := fmt.Sprintf("decision_logs/backtest_%s", backtestID)
	decisionLogger := logger.NewDecisionLogger(logDir)
	fileRecords, err := decisionLogger.GetLatestRecords(100) // 获取最近100条
	if err == nil && len(fileRecords) > 0 {
		// 直接返回文件记录，保持与实盘一致的丰富格式(包含CoT、Prompt等)
		c.JSON(http.StatusOK, fileRecords)
		return
	}

	// 2. 如果文件读取失败或为空，回退到数据库读取
	decisions, err := s.database.GetBacktestDecisions(backtestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get backtest decisions: " + err.Error()})
		return
	}

	// 将数据库记录转换为 DecisionRecord 格式，以保持前端接口一致
	var records []logger.DecisionRecord
	for i, d := range decisions {
		records = append(records, logger.DecisionRecord{
			Timestamp:   d.Timestamp,
			CycleNumber: i + 1, // 估算周期号
			Success:     true,
			Decisions: []logger.DecisionAction{
				{
					Action:     d.Action,
					Symbol:     d.Symbol,
					Price:      d.Price,
					Quantity:   d.Quantity,
					Leverage:   d.Leverage,
					Confidence: d.Confidence,
					Reasoning:  d.Reasoning,
					Timestamp:  d.Timestamp,
					Success:    true,
				},
			},
		})
	}

	c.JSON(http.StatusOK, records)
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
		Slippage:             backtestRun.Slippage,
		UseTraderConfig:      backtestRun.UseTraderConfig,
		MockMode:             backtestRun.CustomPrompt == "MOCK_MODE_ALWAYS_LONG", // Detect MockMode from CustomPrompt
		IndicatorConfig:      indicatorConfig,                                     // Use the parsed indicatorConfig
		CustomPrompt:         backtestRun.CustomPrompt,
		OverrideBasePrompt:   backtestRun.OverrideBasePrompt,
		SystemPromptTemplate: backtestRun.SystemPromptTemplate,
		BTCETHLeverage:       trader.BTCETHLeverage,
		AltcoinLeverage:      float64(trader.AltcoinLeverage),
		// 新增配置
		AiModelID:       backtestRun.AiModelID,
		Timeframe:       backtestRun.Timeframe,
		DataPoints:      backtestRun.DataPoints,
		PreheatDuration: time.Duration(backtestRun.PreheatHours) * time.Hour,
	}

	// 确定使用的AI模型 (优先使用回测指定的模型ID)
	var activeAIModel *config.AIModelConfig
	if backtestRun.AiModelID != "" {
		// 尝试获取指定的AI模型
		model, err := s.database.GetAIModelByID(backtestRun.UserID, backtestRun.AiModelID)
		if err != nil {
			log.Printf("[Backtest %s] Failed to get AI model %s: %v, falling back to trader model", 
				backtestID, backtestRun.AiModelID, err)
			activeAIModel = aiModel // Fallback
		} else {
			activeAIModel = model
			log.Printf("[Backtest %s] Using overridden AI model: %s (%s)", backtestID, model.Name, model.Provider)
		}
	} else {
		activeAIModel = aiModel
	}

	// 创建MCP客户端
	mcpClient := mcp.New()
	provider := strings.ToLower(strings.TrimSpace(activeAIModel.Provider))
	switch provider {
	case "openai", "custom":
		mcpClient.SetCustomAPI(activeAIModel.CustomAPIURL, activeAIModel.APIKey, activeAIModel.CustomModelName)
	case "qwen":
		mcpClient.SetQwenAPIKey(activeAIModel.APIKey, activeAIModel.CustomAPIURL, activeAIModel.CustomModelName)
	case "glm":
		mcpClient.SetGLMAPIKey(activeAIModel.APIKey, activeAIModel.CustomAPIURL, activeAIModel.CustomModelName)
	case "deepseek":
		mcpClient.SetDeepSeekAPIKey(activeAIModel.APIKey, activeAIModel.CustomAPIURL, activeAIModel.CustomModelName)
	default:
		log.Printf("[Backtest %s] Unknown AI model provider: '%s' (normalized: '%s')", backtestID, activeAIModel.Provider, provider)
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

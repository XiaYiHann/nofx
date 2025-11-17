package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"nofx/backtest"
	"nofx/config"
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
	IndicatorConfig      map[string]interface{} `json:"indicator_config,omitempty"`
	CustomPrompt         string                 `json:"custom_prompt,omitempty"`
	SystemPromptTemplate string                 `json:"system_prompt_template,omitempty"`
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

	// 创建回测记录
	backtestID := uuid.New().String()
	backtest := &config.BacktestRun{
		ID:                  backtestID,
		UserID:              userID,
		TraderID:            req.TraderID,
		StartTime:           startTime,
		EndTime:             endTime,
		InitialBalance:      req.InitialBalance,
		ScanIntervalMinutes: trader.ScanIntervalMinutes,
		TradingSymbols:      trader.TradingSymbols,
		UseTraderConfig:     useTraderConfig,
		Status:              "pending",
		Progress:            0,
	}

	// 如果使用trader配置,复制配置
	if useTraderConfig {
		backtest.IndicatorConfig = trader.IndicatorConfig
		backtest.CustomPrompt = trader.CustomPrompt
		backtest.OverrideBasePrompt = trader.OverrideBasePrompt
		backtest.SystemPromptTemplate = trader.SystemPromptTemplate
	} else {
		// 使用自定义配置
		if req.IndicatorConfig != nil {
			// 简化处理:转为JSON字符串
			indicatorConfigJSON := fmt.Sprintf("%v", req.IndicatorConfig)
			backtest.IndicatorConfig = indicatorConfigJSON
		}
		backtest.CustomPrompt = req.CustomPrompt
		backtest.SystemPromptTemplate = req.SystemPromptTemplate
	}

	// 保存到数据库
	if err := s.database.CreateBacktest(backtest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backtest"})
		return
	}

	// 启动异步回测引擎
	go s.runBacktest(backtestID, backtest, trader, aiModel)

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
		IndicatorConfig:      indicatorConfig,
		CustomPrompt:         backtestRun.CustomPrompt,
		OverrideBasePrompt:   backtestRun.OverrideBasePrompt,
		SystemPromptTemplate: backtestRun.SystemPromptTemplate,
		BTCETHLeverage:       trader.BTCETHLeverage,
		AltcoinLeverage:      trader.AltcoinLeverage,
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

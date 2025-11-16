package api

import (
	"fmt"
	"net/http"
	"nofx/config"
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
	trader, _, _, err := s.database.GetTraderConfig(userID, req.TraderID)
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

	// TODO: 启动异步回测引擎
	// go s.runBacktest(backtestID, backtest)

	c.JSON(http.StatusCreated, gin.H{
		"backtest_id": backtestID,
		"status":      "pending",
		"message":     "Backtest created successfully (execution not yet implemented)",
		"config": gin.H{
			"trader_id":        req.TraderID,
			"trader_name":      trader.Name,
			"start_time":       startTime,
			"end_time":         endTime,
			"initial_balance":  req.InitialBalance,
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

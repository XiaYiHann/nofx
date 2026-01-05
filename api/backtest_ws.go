package api

import (
	"encoding/json"
	"log"
	"net/http"
	"nofx/auth"
	"nofx/config"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader 配置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源（生产环境应该更严格）
		return true
	},
}

// handleBacktestWSProgress WebSocket 连接处理器
// GET /ws/backtest/:id/progress
func (s *Server) handleBacktestWSProgress(c *gin.Context) {
	backtestID := c.Param("id")
	if backtestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backtest id required"})
		return
	}

	// 认证检查：从 query param 或 header 获取 token
	token := c.Query("token")
	if token == "" {
		// 尝试从 Authorization header 获取
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// 开发模式下允许无 token 连接
	if !s.devMode && token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// 非开发模式验证 token
	if !s.devMode && token != "" {
		if _, err := s.validateToken(token); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
	}

	// 验证回测存在
	backtest, err := s.database.GetBacktest(backtestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backtest not found"})
		return
	}

	// 升级到 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[BacktestWS] Failed to upgrade connection: %v", err)
		return
	}

	// 创建客户端并注册到 Hub
	client := newWSClient(conn, s.backtestHub)
	s.backtestHub.Register(backtestID, client)

	log.Printf("[BacktestWS] Client connected for backtest %s (status: %s)", backtestID, backtest.Status)

	// 发送当前状态快照
	go s.sendInitialSnapshot(client, backtestID, backtest)

	// 启动读写协程
	go client.writePump(backtestID)
	go client.readPump(backtestID)
}

// sendInitialSnapshot 发送初始状态快照
func (s *Server) sendInitialSnapshot(client *wsClient, backtestID string, backtest *config.BacktestRun) {
	// 获取现有净值快照
	equityHistory, err := s.database.GetEquitySnapshots(backtestID)
	if err != nil {
		log.Printf("[BacktestWS] Failed to get equity history: %v", err)
	}

	// 发送每个净值快照
	for i, snapshot := range equityHistory {
		event := NewProgressEvent(backtestID, EventTypeEquitySnapshot, EquitySnapshotPayload{
			Cycle:       i + 1,
			Timestamp:   snapshot.Time.Format("2006-01-02T15:04:05Z07:00"),
			TotalEquity: snapshot.Equity,
			PnL:         snapshot.PnL,
			PnLPct:      snapshot.PnLPct,
		})

		data, _ := jsonMarshal(event)
		select {
		case client.sendChan <- data:
		default:
			// 缓冲区满，跳过
		}
	}

	// 发送当前进度
	progressEvent := NewProgressEvent(backtestID, EventTypeProgress, ProgressPayload{
		ProgressPct: backtest.Progress,
		Cycle:       len(equityHistory),
	})
	data, _ := jsonMarshal(progressEvent)
	select {
	case client.sendChan <- data:
	default:
	}

	// 如果已完成，发送完成事件
	if backtest.Status == "completed" {
		completeEvent := NewProgressEvent(backtestID, EventTypeComplete, CompletePayload{
			FinalEquity: backtest.FinalEquity,
			TotalPnL:    backtest.TotalPnL,
			TotalPnLPct: backtest.TotalPnLPct,
			MaxDrawdown: backtest.MaxDrawdown,
			SharpeRatio: backtest.SharpeRatio,
			WinRate:     backtest.WinRate,
			TotalTrades: backtest.TotalTrades,
		})
		data, _ := jsonMarshal(completeEvent)
		select {
		case client.sendChan <- data:
		default:
		}
	}
}

// BacktestRunInfo 回测运行信息（用于 WS）
type BacktestRunInfo struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	FinalEquity float64 `json:"final_equity"`
	TotalPnL    float64 `json:"total_pnl"`
	TotalPnLPct float64 `json:"total_pnl_pct"`
	MaxDrawdown float64 `json:"max_drawdown"`
	SharpeRatio float64 `json:"sharpe_ratio"`
	WinRate     float64 `json:"win_rate"`
	TotalTrades int     `json:"total_trades"`
}

// validateToken 验证 JWT token（复用现有逻辑）
func (s *Server) validateToken(tokenString string) (string, error) {
	// 使用现有的 auth 包验证 token
	claims, err := auth.ValidateJWT(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// jsonMarshal 辅助函数
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

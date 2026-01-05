package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBacktestHub_RegisterUnregister(t *testing.T) {
	hub := NewBacktestHub()

	// 模拟 WebSocket 连接
	mockConn := &websocket.Conn{}
	client := &wsClient{
		conn:     mockConn,
		sendChan: make(chan []byte, 256),
		hub:      hub,
	}

	// 测试注册
	hub.Register("test-backtest-1", client)
	assert.Equal(t, 1, hub.GetSubscriberCount("test-backtest-1"))

	// 测试重复注册同一个客户端
	hub.Register("test-backtest-1", client)
	assert.Equal(t, 1, hub.GetSubscriberCount("test-backtest-1"))

	// 测试取消注册
	hub.Unregister("test-backtest-1", client)
	assert.Equal(t, 0, hub.GetSubscriberCount("test-backtest-1"))

	// 测试重复取消注册
	hub.Unregister("test-backtest-1", client)
	assert.Equal(t, 0, hub.GetSubscriberCount("test-backtest-1"))
}

func TestBacktestHub_Broadcast(t *testing.T) {
	hub := NewBacktestHub()

	// 创建多个模拟客户端
	clients := make([]*wsClient, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &wsClient{
			conn:     &websocket.Conn{},
			sendChan: make(chan []byte, 256),
			hub:      hub,
		}
		hub.Register("test-backtest-1", clients[i])
	}

	assert.Equal(t, 3, hub.GetSubscriberCount("test-backtest-1"))

	// 测试广播
	event := ProgressEvent{
		Type:       EventTypeProgress,
		BacktestID: "test-backtest-1",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Payload: ProgressPayload{
			ProgressPct: 50.0,
			Cycle:       10,
		},
	}

	hub.Broadcast("test-backtest-1", event)

	// 验证所有客户端都收到消息
	for i, client := range clients {
		select {
		case msg := <-client.sendChan:
			var received ProgressEvent
			err := json.Unmarshal(msg, &received)
			require.NoError(t, err, "Client %d should receive valid JSON", i)
			assert.Equal(t, EventTypeProgress, received.Type)
			assert.Equal(t, "test-backtest-1", received.BacktestID)
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Client %d did not receive message", i)
		}
	}
}

func TestBacktestHub_BroadcastToNonexistentBacktest(t *testing.T) {
	hub := NewBacktestHub()

	// 广播到不存在的 backtest 不应该 panic
	event := ProgressEvent{
		Type:       EventTypeProgress,
		BacktestID: "nonexistent",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Payload:    ProgressPayload{ProgressPct: 50.0, Cycle: 10},
	}

	hub.Broadcast("nonexistent", event)
	// 没有 panic 就是成功
}

func TestBacktestHub_Publish(t *testing.T) {
	hub := NewBacktestHub()

	client := &wsClient{
		conn:     &websocket.Conn{},
		sendChan: make(chan []byte, 256),
		hub:      hub,
	}
	hub.Register("test-backtest-1", client)

	// 测试 Publish 方法（实现 ProgressPublisher 接口）
	event := ProgressEvent{
		Type:       EventTypeEquitySnapshot,
		BacktestID: "test-backtest-1",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Payload: EquitySnapshotPayload{
			Cycle:       5,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			TotalEquity: 10500.0,
			PnL:         500.0,
			PnLPct:      5.0,
		},
	}

	err := hub.Publish(event)
	require.NoError(t, err)

	select {
	case msg := <-client.sendChan:
		var received ProgressEvent
		err := json.Unmarshal(msg, &received)
		require.NoError(t, err)
		assert.Equal(t, EventTypeEquitySnapshot, received.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client did not receive message")
	}
}

func TestNewProgressEvent(t *testing.T) {
	event := NewProgressEvent("backtest-123", EventTypeComplete, CompletePayload{
		FinalEquity: 12000.0,
		TotalPnL:    2000.0,
		TotalPnLPct: 20.0,
		MaxDrawdown: 5.0,
		SharpeRatio: 1.5,
		WinRate:     60.0,
		TotalTrades: 50,
	})

	assert.Equal(t, EventTypeComplete, event.Type)
	assert.Equal(t, "backtest-123", event.BacktestID)
	assert.NotEmpty(t, event.Timestamp)

	payload, ok := event.Payload.(CompletePayload)
	require.True(t, ok)
	assert.Equal(t, 12000.0, payload.FinalEquity)
	assert.Equal(t, 50, payload.TotalTrades)
}

func TestTruncateCoTTrace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 5, "hello... [truncated]"},
		{"empty string", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateCoTTrace(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBacktestWSEndpoint_DevMode 测试开发模式下的 WebSocket 连接
func TestBacktestWSEndpoint_DevMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试服务器
	router := gin.New()
	hub := NewBacktestHub()

	// 模拟 devMode = true 的处理器
	router.GET("/ws/backtest/:id/progress", func(c *gin.Context) {
		backtestID := c.Param("id")
		if backtestID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "backtest id required"})
			return
		}

		// 开发模式允许无 token
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := newWSClient(conn, hub)
		hub.Register(backtestID, client)
		defer hub.Unregister(backtestID, client)

		// 发送测试消息
		event := NewProgressEvent(backtestID, EventTypeProgress, ProgressPayload{
			ProgressPct: 0,
			Cycle:       0,
		})
		data, _ := json.Marshal(event)
		conn.WriteMessage(websocket.TextMessage, data)
	})

	// 创建测试服务器
	server := httptest.NewServer(router)
	defer server.Close()

	// 构建 WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/backtest/test-123/progress"

	// 连接 WebSocket
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	// 读取消息
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var event ProgressEvent
	err = json.Unmarshal(msg, &event)
	require.NoError(t, err)
	assert.Equal(t, EventTypeProgress, event.Type)
	assert.Equal(t, "test-123", event.BacktestID)
}

func TestBacktestHub_ConcurrentAccess(t *testing.T) {
	hub := NewBacktestHub()
	done := make(chan bool)

	// 并发注册/取消注册
	for i := 0; i < 100; i++ {
		go func(id int) {
			client := &wsClient{
				conn:     &websocket.Conn{},
				sendChan: make(chan []byte, 256),
				hub:      hub,
			}
			hub.Register("concurrent-test", client)
			time.Sleep(time.Millisecond)
			hub.Unregister("concurrent-test", client)
			done <- true
		}(i)
	}

	// 并发广播
	for i := 0; i < 50; i++ {
		go func() {
			event := ProgressEvent{
				Type:       EventTypeProgress,
				BacktestID: "concurrent-test",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				Payload:    ProgressPayload{ProgressPct: 50.0, Cycle: 10},
			}
			hub.Broadcast("concurrent-test", event)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 150; i++ {
		<-done
	}

	// 没有 race condition 或 panic 就是成功
}

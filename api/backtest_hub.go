package api

import (
	"encoding/json"
	"log"
	"nofx/backtest"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 使用 backtest 包的类型别名，避免重复定义
type ProgressEventType = backtest.ProgressEventType
type ProgressEvent = backtest.ProgressEvent
type EquitySnapshotPayload = backtest.EquitySnapshotPayload
type DecisionPayload = backtest.DecisionPayload
type DecisionItemDetail = backtest.DecisionItemDetail
type ProgressPayload = backtest.ProgressPayload
type CompletePayload = backtest.CompletePayload

const (
	EventTypeProgress       = backtest.EventTypeProgress
	EventTypeEquitySnapshot = backtest.EventTypeEquitySnapshot
	EventTypeDecision       = backtest.EventTypeDecision
	EventTypeComplete       = backtest.EventTypeComplete
	EventTypeError          = backtest.EventTypeError
)

// ErrorPayload 错误 payload（api 包专用）
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// wsClient WebSocket 客户端连接
type wsClient struct {
	conn     *websocket.Conn
	sendChan chan []byte
	hub      *BacktestHub
	mu       sync.Mutex
}

// BacktestHub 管理 WebSocket 连接订阅
type BacktestHub struct {
	// 按 backtestID 分组的连接
	subscriptions map[string]map[*wsClient]bool
	mu            sync.RWMutex

	// 配置
	maxMessageSize int
	writeTimeout   time.Duration
	readTimeout    time.Duration
	pingInterval   time.Duration
	sendBuffer     int
}

// NewBacktestHub 创建新的 Hub
func NewBacktestHub() *BacktestHub {
	return &BacktestHub{
		subscriptions:  make(map[string]map[*wsClient]bool),
		maxMessageSize: 10 * 1024, // 10KB max CoT trace
		writeTimeout:   10 * time.Second,
		readTimeout:    60 * time.Second,
		pingInterval:   30 * time.Second,
		sendBuffer:     256,
	}
}

// Register 注册客户端到指定 backtestID
func (h *BacktestHub) Register(backtestID string, client *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscriptions[backtestID] == nil {
		h.subscriptions[backtestID] = make(map[*wsClient]bool)
	}
	h.subscriptions[backtestID][client] = true
	log.Printf("[BacktestHub] Client registered for backtest %s (total: %d)", backtestID, len(h.subscriptions[backtestID]))
}

// Unregister 取消注册客户端
func (h *BacktestHub) Unregister(backtestID string, client *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.subscriptions[backtestID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.subscriptions, backtestID)
		}
		log.Printf("[BacktestHub] Client unregistered from backtest %s", backtestID)
	}
}

// Broadcast 广播事件到指定 backtestID 的所有订阅者
func (h *BacktestHub) Broadcast(backtestID string, event ProgressEvent) {
	h.mu.RLock()
	clients, ok := h.subscriptions[backtestID]
	if !ok || len(clients) == 0 {
		h.mu.RUnlock()
		return
	}
	// 复制客户端列表以避免在发送时持有锁
	clientList := make([]*wsClient, 0, len(clients))
	for client := range clients {
		clientList = append(clientList, client)
	}
	h.mu.RUnlock()

	// 序列化消息
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[BacktestHub] Failed to marshal event: %v", err)
		return
	}

	// 非阻塞发送给所有客户端
	for _, client := range clientList {
		select {
		case client.sendChan <- data:
		default:
			// 发送缓冲区满，跳过（避免阻塞）
			log.Printf("[BacktestHub] Client send buffer full, dropping message")
		}
	}
}

// Publish 实现 ProgressPublisher 接口
func (h *BacktestHub) Publish(event ProgressEvent) error {
	h.Broadcast(event.BacktestID, event)
	return nil
}

// GetSubscriberCount 获取指定 backtestID 的订阅者数量
func (h *BacktestHub) GetSubscriberCount(backtestID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.subscriptions[backtestID]; ok {
		return len(clients)
	}
	return 0
}

// newWSClient 创建新的 WebSocket 客户端
func newWSClient(conn *websocket.Conn, hub *BacktestHub) *wsClient {
	return &wsClient{
		conn:     conn,
		sendChan: make(chan []byte, hub.sendBuffer),
		hub:      hub,
	}
}

// writePump 写入协程，处理发送消息和心跳
func (c *wsClient) writePump(backtestID string) {
	ticker := time.NewTicker(c.hub.pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		c.hub.Unregister(backtestID, c)
	}()

	for {
		select {
		case message, ok := <-c.sendChan:
			c.conn.SetWriteDeadline(time.Now().Add(c.hub.writeTimeout))
			if !ok {
				// Hub 关闭了通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[BacktestHub] Write error: %v", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.hub.writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 读取协程，处理客户端消息和关闭
func (c *wsClient) readPump(backtestID string) {
	defer func() {
		c.hub.Unregister(backtestID, c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(int64(c.hub.maxMessageSize))
	c.conn.SetReadDeadline(time.Now().Add(c.hub.readTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.hub.readTimeout))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[BacktestHub] Read error: %v", err)
			}
			break
		}
		// 目前客户端不发送消息，只接收
	}
}

// TruncateCoTTrace 截断 CoT trace 到指定长度
func TruncateCoTTrace(trace string, maxLen int) string {
	if len(trace) <= maxLen {
		return trace
	}
	return trace[:maxLen] + "... [truncated]"
}

// NewProgressEvent 创建进度事件
func NewProgressEvent(backtestID string, eventType ProgressEventType, payload interface{}) ProgressEvent {
	return ProgressEvent{
		Type:       eventType,
		BacktestID: backtestID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Payload:    payload,
	}
}

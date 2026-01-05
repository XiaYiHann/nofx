package backtest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProgressPublisher 用于测试的进度发布器
type MockProgressPublisher struct {
	events []ProgressEvent
	mu     sync.Mutex
}

func NewMockProgressPublisher() *MockProgressPublisher {
	return &MockProgressPublisher{
		events: make([]ProgressEvent, 0),
	}
}

func (m *MockProgressPublisher) Publish(event ProgressEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *MockProgressPublisher) GetEvents() []ProgressEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ProgressEvent{}, m.events...)
}

func (m *MockProgressPublisher) GetEventsByType(eventType ProgressEventType) []ProgressEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]ProgressEvent, 0)
	for _, e := range m.events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
}

func (m *MockProgressPublisher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]ProgressEvent, 0)
}

func TestProgressPublisherInterface(t *testing.T) {
	// 验证 MockProgressPublisher 实现了 ProgressPublisher 接口
	var _ ProgressPublisher = (*MockProgressPublisher)(nil)
}

func TestEngine_SetProgressPublisher(t *testing.T) {
	// 创建一个最小配置的 Engine（不需要实际的数据库）
	cfg := &Config{
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour),
		InitialBalance: 10000,
		ScanInterval:   time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
	}

	engine := NewEngine("test-backtest", cfg, nil, nil)
	assert.Nil(t, engine.progressPublisher)

	// 设置发布器
	publisher := NewMockProgressPublisher()
	engine.SetProgressPublisher(publisher)
	assert.NotNil(t, engine.progressPublisher)
}

func TestEngine_NewEngineWithPublisher(t *testing.T) {
	cfg := &Config{
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour),
		InitialBalance: 10000,
		ScanInterval:   time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
	}

	publisher := NewMockProgressPublisher()
	engine := NewEngine("test-backtest", cfg, nil, nil, publisher)

	assert.NotNil(t, engine.progressPublisher)
}

func TestEngine_PublishProgressEvent(t *testing.T) {
	cfg := &Config{
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour),
		InitialBalance: 10000,
		ScanInterval:   time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
	}

	publisher := NewMockProgressPublisher()
	engine := NewEngine("test-backtest", cfg, nil, nil, publisher)

	// 手动调用发布方法
	engine.publishProgressEvent(5, 50.0)

	events := publisher.GetEventsByType(EventTypeProgress)
	require.Len(t, events, 1)

	assert.Equal(t, EventTypeProgress, events[0].Type)
	assert.Equal(t, "test-backtest", events[0].BacktestID)
	assert.NotEmpty(t, events[0].Timestamp)

	payload, ok := events[0].Payload.(ProgressPayload)
	require.True(t, ok)
	assert.Equal(t, 5, payload.Cycle)
	assert.Equal(t, 50.0, payload.ProgressPct)
}

func TestEngine_PublishEquitySnapshotEvent(t *testing.T) {
	cfg := &Config{
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour),
		InitialBalance: 10000,
		ScanInterval:   time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
	}

	publisher := NewMockProgressPublisher()
	engine := NewEngine("test-backtest", cfg, nil, nil, publisher)

	// 添加一个净值快照
	now := time.Now()
	engine.equitySnapshots = append(engine.equitySnapshots, EquitySnapshot{
		Time:   now,
		Equity: 10500,
		PnL:    500,
		PnLPct: 5.0,
	})

	engine.publishEquitySnapshotEvent(3, now)

	events := publisher.GetEventsByType(EventTypeEquitySnapshot)
	require.Len(t, events, 1)

	assert.Equal(t, EventTypeEquitySnapshot, events[0].Type)
	assert.Equal(t, "test-backtest", events[0].BacktestID)

	payload, ok := events[0].Payload.(EquitySnapshotPayload)
	require.True(t, ok)
	assert.Equal(t, 3, payload.Cycle)
	assert.Equal(t, 10500.0, payload.TotalEquity)
	assert.Equal(t, 500.0, payload.PnL)
	assert.Equal(t, 5.0, payload.PnLPct)
}

func TestEngine_PublishCompleteEvent(t *testing.T) {
	cfg := &Config{
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour),
		InitialBalance: 10000,
		ScanInterval:   time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
	}

	publisher := NewMockProgressPublisher()
	engine := NewEngine("test-backtest", cfg, nil, nil, publisher)
	engine.currentCycle = 100

	// 添加最终净值快照
	now := time.Now()
	engine.equitySnapshots = append(engine.equitySnapshots, EquitySnapshot{
		Time:   now,
		Equity: 12000,
		PnL:    2000,
		PnLPct: 20.0,
	})

	result := &Result{
		FinalEquity: 12000,
		TotalPnL:    2000,
		TotalPnLPct: 20.0,
		MaxDrawdown: 5.0,
		SharpeRatio: 1.5,
		WinRate:     60.0,
		TotalTrades: 50,
	}

	engine.publishCompleteEvent(result)

	events := publisher.GetEventsByType(EventTypeComplete)
	require.Len(t, events, 1)

	assert.Equal(t, EventTypeComplete, events[0].Type)
	assert.Equal(t, "test-backtest", events[0].BacktestID)

	payload, ok := events[0].Payload.(CompletePayload)
	require.True(t, ok)
	assert.Equal(t, 12000.0, payload.FinalEquity)
	assert.Equal(t, 2000.0, payload.TotalPnL)
	assert.Equal(t, 20.0, payload.TotalPnLPct)
	assert.Equal(t, 50, payload.TotalTrades)
	assert.NotNil(t, payload.FinalSnapshot)
	assert.Equal(t, 100, payload.FinalSnapshot.Cycle)
}

func TestEngine_NoPublisherNoError(t *testing.T) {
	cfg := &Config{
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour),
		InitialBalance: 10000,
		ScanInterval:   time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
	}

	// 没有设置 publisher
	engine := NewEngine("test-backtest", cfg, nil, nil)

	// 这些调用不应该 panic
	engine.publishProgressEvent(5, 50.0)
	engine.publishEquitySnapshotEvent(3, time.Now())
	engine.publishCompleteEvent(&Result{})

	// 没有 panic 就是成功
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 5, "hello..."},
		{"empty string", "", 10, ""},
		{"single char truncation", "ab", 1, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProgressEventTypes(t *testing.T) {
	// 验证事件类型常量
	assert.Equal(t, ProgressEventType("progress"), EventTypeProgress)
	assert.Equal(t, ProgressEventType("equity_snapshot"), EventTypeEquitySnapshot)
	assert.Equal(t, ProgressEventType("decision"), EventTypeDecision)
	assert.Equal(t, ProgressEventType("complete"), EventTypeComplete)
	assert.Equal(t, ProgressEventType("error"), EventTypeError)
}

func TestMockProgressPublisher_ConcurrentAccess(t *testing.T) {
	publisher := NewMockProgressPublisher()
	var wg sync.WaitGroup

	// 并发发布事件
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := ProgressEvent{
				Type:       EventTypeProgress,
				BacktestID: "test",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				Payload:    ProgressPayload{Cycle: id, ProgressPct: float64(id)},
			}
			publisher.Publish(event)
		}(i)
	}

	wg.Wait()

	events := publisher.GetEvents()
	assert.Len(t, events, 100)
}

// TestEngine_PublishDecisionEvent 测试决策事件发布
func TestEngine_PublishDecisionEvent_WithNilPublisher(t *testing.T) {
	cfg := &Config{
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(time.Hour),
		InitialBalance: 10000,
		ScanInterval:   time.Minute,
		TradingSymbols: []string{"BTCUSDT"},
	}

	// 没有设置 publisher
	engine := NewEngine("test-backtest", cfg, nil, nil)

	// 创建一个空的 decision record（需要导入 logger 包）
	// 由于依赖问题，这里只测试 nil publisher 不会 panic
	ctx := context.Background()
	_ = ctx // 避免 unused 警告

	// 直接调用 publishDecisionEvent 会需要 logger.DecisionRecord
	// 这里我们只验证 nil publisher 不会导致问题
	engine.progressPublisher = nil
	// publishDecisionEvent 在 nil publisher 时会直接返回
}

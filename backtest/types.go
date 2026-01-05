package backtest

import (
	"nofx/market"
	"time"
)

// ProgressEventType 事件类型
type ProgressEventType string

const (
	EventTypeProgress       ProgressEventType = "progress"
	EventTypeEquitySnapshot ProgressEventType = "equity_snapshot"
	EventTypeDecision       ProgressEventType = "decision"
	EventTypeComplete       ProgressEventType = "complete"
	EventTypeError          ProgressEventType = "error"
)

// ProgressEvent WebSocket 消息结构
type ProgressEvent struct {
	Type       ProgressEventType `json:"type"`
	BacktestID string            `json:"backtest_id"`
	Timestamp  string            `json:"timestamp"`
	Payload    interface{}       `json:"payload"`
}

// ProgressPublisher 进度发布接口
// Engine 在每个 cycle 完成后调用 Publish 发布事件
type ProgressPublisher interface {
	Publish(event ProgressEvent) error
}

// EquitySnapshotPayload 净值快照 payload
type EquitySnapshotPayload struct {
	Cycle       int     `json:"cycle"`
	Timestamp   string  `json:"timestamp"`
	TotalEquity float64 `json:"total_equity"`
	PnL         float64 `json:"pnl"`
	PnLPct      float64 `json:"pnl_pct"`
}

// DecisionPayload 决策 payload
type DecisionPayload struct {
	Cycle        int                  `json:"cycle"`
	Timestamp    string               `json:"timestamp"`
	SystemPrompt string               `json:"system_prompt,omitempty"`
	InputPrompt  string               `json:"input_prompt,omitempty"`
	CoTTrace     string               `json:"cot_trace,omitempty"`
	DecisionJSON string               `json:"decision_json,omitempty"`
	Decisions    []DecisionItemDetail `json:"decisions"`
	AccountState interface{}          `json:"account_state,omitempty"`
	Positions    interface{}          `json:"positions,omitempty"`
}

// DecisionItemDetail 单个决策详情
type DecisionItemDetail struct {
	Action     string  `json:"action"`
	Symbol     string  `json:"symbol"`
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
	Confidence int     `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
	Success    bool    `json:"success"`
}

// ProgressPayload 进度 payload
type ProgressPayload struct {
	ProgressPct float64 `json:"progress_pct"`
	Cycle       int     `json:"cycle"`
}

// CompletePayload 完成 payload
type CompletePayload struct {
	FinalEquity   float64                `json:"final_equity"`
	TotalPnL      float64                `json:"total_pnl"`
	TotalPnLPct   float64                `json:"total_pnl_pct"`
	MaxDrawdown   float64                `json:"max_drawdown"`
	SharpeRatio   float64                `json:"sharpe_ratio"`
	WinRate       float64                `json:"win_rate"`
	TotalTrades   int                    `json:"total_trades"`
	FinalSnapshot *EquitySnapshotPayload `json:"final_snapshot,omitempty"`
}

// Config 回测配置
type Config struct {
	TraderID       string        // 关联的trader ID
	UserID         string        // 用户ID
	StartTime      time.Time     // 回测起始时间
	EndTime        time.Time     // 回测结束时间
	InitialBalance float64       // 初始资金
	ScanInterval   time.Duration // 扫描间隔
	TradingSymbols []string      // 交易币种
	Slippage       int           // 滑点基点(10 = 0.1%)

	// 回测专用覆盖配置
	AiModelID       string        // 覆盖使用的AI模型ID
	Timeframe       string        // K线周期 (e.g. "3m", "15m")
	DataPoints      int           // 指标计算数据点数量
	PreheatDuration time.Duration // 预热时长

	// 配置复用
	UseTraderConfig      bool                    // 是否使用trader配置
	IndicatorConfig      *market.IndicatorConfig // 指标配置
	CustomPrompt         string                  // 自定义提示词
	OverrideBasePrompt   bool                    // 是否覆盖基础提示词
	SystemPromptTemplate string                  // 系统提示词模板
	BTCETHLeverage       int                     // BTC/ETH杠杆
	AltcoinLeverage      float64                 `json:"altcoin_leverage"`
	MockMode             bool                    `json:"mock_mode"` // If true, AI always returns LONG                     // 山寨币杠杆
}

// Result 回测结果
type Result struct {
	FinalEquity float64 // 最终净值
	TotalPnL    float64 // 总盈亏
	TotalPnLPct float64 // 总盈亏百分比
	MaxDrawdown float64 // 最大回撤
	SharpeRatio float64 // 夏普率
	WinRate     float64 // 胜率
	TotalTrades int     // 总交易数

	EquitySnapshots []EquitySnapshot // 净值快照
	Trades          []Trade          // 交易记录
}

// EquitySnapshot 净值快照
type EquitySnapshot struct {
	Time   time.Time
	Equity float64
	PnL    float64
	PnLPct float64
}

// Trade 交易记录
type Trade struct {
	Symbol     string // 币种
	Side       string // long/short
	Action     string // open/close
	EntryPrice float64
	ExitPrice  float64
	Quantity   float64
	Leverage   int
	PnL        float64
	PnLPct     float64
	Fee        float64
	EntryTime  time.Time
	ExitTime   time.Time
}

// Position 持仓
type Position struct {
	Symbol           string
	Side             string // "long" or "short"
	EntryPrice       float64
	MarkPrice        float64 // Current market price
	Quantity         float64
	Leverage         int
	UnrealizedPnL    float64
	UnrealizedPnLPct float64
	LiquidationPrice float64
	MarginUsed       float64
	EntryTime        time.Time // Use time.Time for clarity
}

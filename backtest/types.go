package backtest

import (
	"nofx/market"
	"time"
)

// Config 回测配置
type Config struct {
	TraderID        string                  // 关联的trader ID
	UserID          string                  // 用户ID
	StartTime       time.Time               // 回测起始时间
	EndTime         time.Time               // 回测结束时间
	InitialBalance  float64                 // 初始资金
	ScanInterval    time.Duration           // 扫描间隔
	TradingSymbols  []string                // 交易币种
	Slippage        int                     // 滑点基点(10 = 0.1%)
	
	// 配置复用
	UseTraderConfig      bool                    // 是否使用trader配置
	IndicatorConfig      *market.IndicatorConfig // 指标配置
	CustomPrompt         string                  // 自定义提示词
	OverrideBasePrompt   bool                    // 是否覆盖基础提示词
	SystemPromptTemplate string                  // 系统提示词模板
	BTCETHLeverage       int                     // BTC/ETH杠杆
	AltcoinLeverage      int                     // 山寨币杠杆
}

// Result 回测结果
type Result struct {
	FinalEquity  float64 // 最终净值
	TotalPnL     float64 // 总盈亏
	TotalPnLPct  float64 // 总盈亏百分比
	MaxDrawdown  float64 // 最大回撤
	SharpeRatio  float64 // 夏普率
	WinRate      float64 // 胜率
	TotalTrades  int     // 总交易数
	
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
	Symbol     string  // 币种
	Side       string  // long/short
	Action     string  // open/close
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
	Symbol         string
	Side           string  // long/short
	EntryPrice     float64
	Quantity       float64
	Leverage       int
	MarginUsed     float64
	UnrealizedPnL  float64
	EntryTime      time.Time
}
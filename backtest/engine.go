package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"nofx/config"
	"nofx/decision"
	"nofx/market"
	"nofx/mcp"
	"strings"
	"time"
)

// Engine 回测引擎
type Engine struct {
	config          *Config
	db              *config.Database
	apiClient       *market.APIClient
	mcpClient       *mcp.Client
	timeSimulator   *TimeSimulator
	positionManager *PositionManager
	orderSimulator  *OrderSimulator

	// 回测状态
	backtestID      string
	equitySnapshots []EquitySnapshot
	trades          []Trade

	// 数据缓存
	klineCache map[string][]market.Kline // symbol -> klines
}

// NewEngine 创建回测引擎
func NewEngine(
	backtestID string,
	cfg *Config,
	db *config.Database,
	mcpClient *mcp.Client,
) *Engine {
	return &Engine{
		config:          cfg,
		db:              db,
		apiClient:       market.NewAPIClient(),
		mcpClient:       mcpClient,
		timeSimulator:   NewTimeSimulator(cfg.StartTime, cfg.EndTime, cfg.ScanInterval),
		positionManager: NewPositionManager(cfg.InitialBalance),
		orderSimulator:  NewOrderSimulator(cfg.Slippage),
		backtestID:      backtestID,
		equitySnapshots: []EquitySnapshot{},
		trades:          []Trade{},
		klineCache:      make(map[string][]market.Kline),
	}
}

// Run 执行回测
func (e *Engine) Run(ctx context.Context) error {
	log.Printf("[Backtest %s] Starting backtest from %s to %s",
		e.backtestID, e.config.StartTime, e.config.EndTime)

	// 1. 加载历史数据
	if err := e.loadHistoricalData(ctx); err != nil {
		return fmt.Errorf("failed to load historical data: %w", err)
	}

	// 2. 记录初始净值
	e.recordEquitySnapshot(e.timeSimulator.CurrentTime())

	// 3. 主回测循环
	for e.timeSimulator.HasNext() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		e.timeSimulator.Next()
		currentTime := e.timeSimulator.CurrentTime()

		// 更新进度
		progress := e.timeSimulator.Progress()
		if err := e.updateProgress(progress); err != nil {
			log.Printf("[Backtest %s] Failed to update progress: %v", e.backtestID, err)
		}

		// 获取当前市场数据
		marketDataMap, err := e.getMarketDataAtTime(currentTime)
		if err != nil {
			log.Printf("[Backtest %s] Failed to get market data at %s: %v",
				e.backtestID, currentTime, err)
			continue
		}

		// 更新持仓未实现盈亏
		e.updateUnrealizedPnL(marketDataMap)

		// 获取AI决策
		decisions, err := e.getDecisions(marketDataMap)
		if err != nil {
			log.Printf("[Backtest %s] Failed to get decisions: %v", e.backtestID, err)
			continue
		}

		// 执行交易决策
		if err := e.executeDecisions(decisions, marketDataMap, currentTime); err != nil {
			log.Printf("[Backtest %s] Failed to execute decisions: %v", e.backtestID, err)
		}

		// 记录净值快照
		e.recordEquitySnapshot(currentTime)
	}

	// 4. 计算最终结果
	result := e.calculateResult()

	// 5. 保存结果
	if err := e.saveResult(result); err != nil {
		return fmt.Errorf("failed to save result: %w", err)
	}

	log.Printf("[Backtest %s] Completed. Final Equity: %.2f, Total PnL: %.2f (%.2f%%)",
		e.backtestID, result.FinalEquity, result.TotalPnL, result.TotalPnLPct)

	return nil
}

// loadHistoricalData 加载历史数据
func (e *Engine) loadHistoricalData(ctx context.Context) error {
	log.Printf("[Backtest %s] Loading historical data for %d symbols",
		e.backtestID, len(e.config.TradingSymbols))

	startMs := e.config.StartTime.UnixMilli()
	endMs := e.config.EndTime.UnixMilli()

	// 计算需要的间隔(3m用于主数据)
	interval := "3m"

	for i, symbol := range e.config.TradingSymbols {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Printf("[Backtest %s] Loading data for %s (%d/%d)",
			e.backtestID, symbol, i+1, len(e.config.TradingSymbols))

		klines, err := e.apiClient.GetKlinesRange(symbol, interval, startMs, endMs)
		if err != nil {
			return fmt.Errorf("failed to load klines for %s: %w", symbol, err)
		}

		e.klineCache[symbol] = klines
		log.Printf("[Backtest %s] Loaded %d klines for %s", e.backtestID, len(klines), symbol)
	}

	return nil
}

// getMarketDataAtTime 获取指定时间的市场数据
func (e *Engine) getMarketDataAtTime(t time.Time) (map[string]*market.Data, error) {
	dataMap := make(map[string]*market.Data)
	targetMs := t.UnixMilli()

	for _, symbol := range e.config.TradingSymbols {
		klines, exists := e.klineCache[symbol]
		if !exists {
			continue
		}

		// 找到目标时间之前的最近100根K线
		recentKlines := e.getRecentKlines(klines, targetMs, 100)
		if len(recentKlines) < 20 {
			// 数据不足,跳过
			continue
		}

		// 使用market.Get()计算指标
		marketData, err := e.calculateMarketData(symbol, recentKlines)
		if err != nil {
			log.Printf("[Backtest %s] Failed to calculate market data for %s: %v",
				e.backtestID, symbol, err)
			continue
		}

		dataMap[symbol] = marketData
	}

	return dataMap, nil
}

// getRecentKlines 获取最近N根K线
func (e *Engine) getRecentKlines(klines []market.Kline, targetMs int64, count int) []market.Kline {
	var result []market.Kline

	for i := len(klines) - 1; i >= 0; i-- {
		if klines[i].CloseTime <= targetMs {
			// 向前收集count根
			start := i - count + 1
			if start < 0 {
				start = 0
			}
			result = klines[start : i+1]
			break
		}
	}

	return result
}

// calculateMarketData 计算市场数据(复用market.Get逻辑)
func (e *Engine) calculateMarketData(symbol string, klines []market.Kline) (*market.Data, error) {
	// 简化版本:直接使用K线数据计算基本指标
	// 在实际实现中,应该复用market.Get()的完整逻辑

	if len(klines) == 0 {
		return nil, fmt.Errorf("no klines available")
	}

	lastKline := klines[len(klines)-1]

	// 创建基本的市场数据结构
	data := &market.Data{
		Symbol:        symbol,
		CurrentPrice:  lastKline.Close,
		TimeframeData: make(map[string]*market.TimeframeData),
	}

	// 提取价格和成交量序列
	prices := make([]float64, len(klines))
	volumes := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
		volumes[i] = k.Volume
	}

	// 创建3m时间框架数据
	data.TimeframeData["3m"] = &market.TimeframeData{
		Timeframe:  "3m",
		DataPoints: len(klines),
		MidPrices:  prices,
		Volume:     volumes,
	}

	// TODO: 如果需要完整指标,应该调用market包的计算函数
	// 这里为了简化,只提供基本数据

	return data, nil
}

// getDecisions 获取AI决策
func (e *Engine) getDecisions(marketDataMap map[string]*market.Data) ([]decision.Decision, error) {
	// 构建决策上下文
	ctx := &decision.Context{
		Account: decision.AccountInfo{
			TotalEquity:      e.positionManager.GetEquity(),
			AvailableBalance: e.positionManager.GetAvailableBalance(),
		},
		Positions:       []decision.PositionInfo{},
		CandidateCoins:  []decision.CandidateCoin{},
		MarketDataMap:   marketDataMap,
		BTCETHLeverage:  e.config.BTCETHLeverage,
		AltcoinLeverage: e.config.AltcoinLeverage,
		IndicatorConfig: e.config.IndicatorConfig,
	}

	// 添加持仓信息
	for _, symbol := range e.config.TradingSymbols {
		if pos, exists := e.positionManager.GetPosition(symbol); exists {
			marketData, _ := marketDataMap[symbol]
			currentPrice := 0.0
			if marketData != nil {
				currentPrice = marketData.CurrentPrice
			}

			var pnlPct float64
			if pos.MarginUsed > 0 {
				pnlPct = (pos.UnrealizedPnL / pos.MarginUsed) * 100
			}

			ctx.Positions = append(ctx.Positions, decision.PositionInfo{
				Symbol:           symbol,
				Side:             pos.Side,
				EntryPrice:       pos.EntryPrice,
				MarkPrice:        currentPrice,
				Quantity:         pos.Quantity,
				Leverage:         pos.Leverage,
				UnrealizedPnL:    pos.UnrealizedPnL,
				UnrealizedPnLPct: pnlPct,
				MarginUsed:       pos.MarginUsed,
			})
		} else {
			// 将未持仓的币种作为候选
			ctx.CandidateCoins = append(ctx.CandidateCoins, decision.CandidateCoin{
				Symbol:  symbol,
				Sources: []string{"backtest"},
			})
		}
	}

	// 调用决策引擎
	fullDecision, err := decision.GetFullDecisionWithCustomPrompt(
		ctx,
		e.mcpClient,
		e.config.CustomPrompt,
		e.config.OverrideBasePrompt,
		e.config.SystemPromptTemplate,
	)

	if err != nil {
		return nil, err
	}

	return fullDecision.Decisions, nil
}

// executeDecisions 执行交易决策
func (e *Engine) executeDecisions(
	decisions []decision.Decision,
	marketDataMap map[string]*market.Data,
	currentTime time.Time,
) error {
	for _, dec := range decisions {
		symbol := dec.Symbol
		marketData, exists := marketDataMap[symbol]
		if !exists {
			log.Printf("[Backtest %s] No market data for %s", e.backtestID, symbol)
			continue
		}

		currentPrice := marketData.CurrentPrice

		// 处理不同动作
		switch strings.ToLower(dec.Action) {
		case "open_long", "open_short":
			if err := e.executeOpenPosition(dec, currentPrice, currentTime); err != nil {
				log.Printf("[Backtest %s] Failed to open position %s: %v",
					e.backtestID, symbol, err)
			}

		case "close_long", "close_short":
			if err := e.executeClosePosition(dec, currentPrice, currentTime); err != nil {
				log.Printf("[Backtest %s] Failed to close position %s: %v",
					e.backtestID, symbol, err)
			}
		}
	}

	return nil
}

// executeOpenPosition 执行开仓
func (e *Engine) executeOpenPosition(dec decision.Decision, marketPrice float64, currentTime time.Time) error {
	symbol := dec.Symbol

	// 检查是否已有持仓
	if e.positionManager.HasPosition(symbol) {
		return fmt.Errorf("position already exists")
	}

	// 确定杠杆
	leverage := e.getLeverageForSymbol(symbol)

	// 计算仓位大小(简化版:使用固定百分比)
	accountEquity := e.positionManager.GetEquity()
	riskPercent := 2.0 // 每次交易风险2%

	// 简化计算:使用账户权益的一定比例
	positionValue := accountEquity * (riskPercent / 100.0) * float64(leverage)
	quantity := positionValue / marketPrice

	// 确定方向
	side := "long"
	if strings.Contains(strings.ToLower(dec.Action), "short") {
		side = "short"
	}

	// 执行订单
	executionPrice, fee, err := e.orderSimulator.ExecuteMarketOrder(
		side, "open", marketPrice, quantity, leverage,
	)
	if err != nil {
		return err
	}

	// 开仓
	if err := e.positionManager.OpenPosition(symbol, side, executionPrice, quantity, leverage); err != nil {
		return err
	}

	// 记录交易(开仓)
	trade := Trade{
		Symbol:     symbol,
		Side:       side,
		Action:     "open",
		EntryPrice: executionPrice,
		Quantity:   quantity,
		Leverage:   leverage,
		Fee:        fee,
		EntryTime:  currentTime,
	}
	e.trades = append(e.trades, trade)

	log.Printf("[Backtest %s] Opened %s position: %s @ %.2f (qty: %.4f, leverage: %dx)",
		e.backtestID, side, symbol, executionPrice, quantity, leverage)

	return nil
}

// executeClosePosition 执行平仓
func (e *Engine) executeClosePosition(dec decision.Decision, marketPrice float64, currentTime time.Time) error {
	symbol := dec.Symbol

	pos, exists := e.positionManager.GetPosition(symbol)
	if !exists {
		return fmt.Errorf("no position to close")
	}

	// 执行订单
	executionPrice, fee, err := e.orderSimulator.ExecuteMarketOrder(
		pos.Side, "close", marketPrice, pos.Quantity, pos.Leverage,
	)
	if err != nil {
		return err
	}

	// 平仓
	trade, err := e.positionManager.ClosePosition(symbol, executionPrice)
	if err != nil {
		return err
	}

	// 更新交易记录
	trade.Fee = fee
	trade.ExitTime = currentTime
	e.trades = append(e.trades, *trade)

	log.Printf("[Backtest %s] Closed %s position: %s @ %.2f (PnL: %.2f, %.2f%%)",
		e.backtestID, pos.Side, symbol, executionPrice, trade.PnL, trade.PnLPct)

	return nil
}

// updateUnrealizedPnL 更新未实现盈亏
func (e *Engine) updateUnrealizedPnL(marketDataMap map[string]*market.Data) {
	for symbol, data := range marketDataMap {
		if e.positionManager.HasPosition(symbol) {
			e.positionManager.UpdateUnrealizedPnL(symbol, data.CurrentPrice)
		}
	}
}

// recordEquitySnapshot 记录净值快照
func (e *Engine) recordEquitySnapshot(t time.Time) {
	equity := e.positionManager.GetEquity()
	pnl := equity - e.config.InitialBalance
	pnlPct := (pnl / e.config.InitialBalance) * 100

	snapshot := EquitySnapshot{
		Time:   t,
		Equity: equity,
		PnL:    pnl,
		PnLPct: pnlPct,
	}

	e.equitySnapshots = append(e.equitySnapshots, snapshot)
}

// calculateResult 计算最终结果
func (e *Engine) calculateResult() *Result {
	finalEquity := e.positionManager.GetEquity()
	totalPnL := finalEquity - e.config.InitialBalance
	totalPnLPct := (totalPnL / e.config.InitialBalance) * 100

	maxDrawdown := CalculateMaxDrawdown(e.equitySnapshots)
	sharpeRatio := CalculateSharpeRatio(e.equitySnapshots)

	// 计算胜率
	winCount := 0
	totalTrades := 0
	for _, trade := range e.trades {
		if trade.Action == "close" {
			totalTrades++
			if trade.PnL > 0 {
				winCount++
			}
		}
	}

	winRate := 0.0
	if totalTrades > 0 {
		winRate = (float64(winCount) / float64(totalTrades)) * 100
	}

	return &Result{
		FinalEquity:     finalEquity,
		TotalPnL:        totalPnL,
		TotalPnLPct:     totalPnLPct,
		MaxDrawdown:     maxDrawdown,
		SharpeRatio:     sharpeRatio,
		WinRate:         winRate,
		TotalTrades:     totalTrades,
		EquitySnapshots: e.equitySnapshots,
		Trades:          e.trades,
	}
}

// saveResult 保存结果
func (e *Engine) saveResult(result *Result) error {
	// 保存回测结果
	backtestRun := &config.BacktestRun{
		Status:      "completed",
		FinalEquity: result.FinalEquity,
		TotalPnL:    result.TotalPnL,
		TotalPnLPct: result.TotalPnLPct,
		MaxDrawdown: result.MaxDrawdown,
		SharpeRatio: result.SharpeRatio,
		WinRate:     result.WinRate,
		TotalTrades: result.TotalTrades,
	}

	if err := e.db.SaveBacktestResult(e.backtestID, backtestRun); err != nil {
		return fmt.Errorf("failed to save backtest result: %w", err)
	}

	// 保存净值快照
	dbSnapshots := make([]config.BacktestEquitySnapshot, len(result.EquitySnapshots))
	for i, s := range result.EquitySnapshots {
		dbSnapshots[i] = config.BacktestEquitySnapshot{
			BacktestID: e.backtestID,
			Time:       s.Time,
			Equity:     s.Equity,
			PnL:        s.PnL,
			PnLPct:     s.PnLPct,
		}
	}

	if err := e.db.SaveEquitySnapshots(e.backtestID, dbSnapshots); err != nil {
		return fmt.Errorf("failed to save equity snapshots: %w", err)
	}

	// 保存交易记录
	dbTrades := make([]config.BacktestTrade, len(result.Trades))
	for i, t := range result.Trades {
		dbTrades[i] = config.BacktestTrade{
			BacktestID: e.backtestID,
			Symbol:     t.Symbol,
			Side:       t.Side,
			Action:     t.Action,
			EntryPrice: &t.EntryPrice,
			ExitPrice:  &t.ExitPrice,
			Quantity:   t.Quantity,
			Leverage:   t.Leverage,
			PnL:        t.PnL,
			PnLPct:     t.PnLPct,
			Fee:        t.Fee,
			EntryTime:  &t.EntryTime,
			ExitTime:   &t.ExitTime,
		}
	}

	if err := e.db.SaveBacktestTrades(e.backtestID, dbTrades); err != nil {
		return fmt.Errorf("failed to save trades: %w", err)
	}

	return nil
}

// updateProgress 更新进度
func (e *Engine) updateProgress(progress float64) error {
	return e.db.UpdateBacktestStatus(e.backtestID, "running", progress)
}

// getLeverageForSymbol 获取币种杠杆
func (e *Engine) getLeverageForSymbol(symbol string) int {
	// BTC/ETH 使用较低杠杆
	if strings.Contains(symbol, "BTC") || strings.Contains(symbol, "ETH") {
		if e.config.BTCETHLeverage > 0 {
			return e.config.BTCETHLeverage
		}
		return 10
	}

	// 山寨币使用较高杠杆
	if e.config.AltcoinLeverage > 0 {
		return e.config.AltcoinLeverage
	}
	return 20
}

// ParseIndicatorConfig 解析指标配置JSON字符串
func ParseIndicatorConfig(configJSON string) (*market.IndicatorConfig, error) {
	if configJSON == "" {
		return nil, nil
	}

	var config market.IndicatorConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

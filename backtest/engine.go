package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"nofx/config"
	"nofx/decision"
	"nofx/logger"
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
	decisionLogger  *logger.DecisionLogger

	// 回测状态
	backtestID      string
	equitySnapshots []EquitySnapshot
	trades          []Trade
	decisions       []config.BacktestDecision

	// 增量保存状态
	lastSavedSnapshotIdx int
	lastSavedTradeIdx    int
	lastSavedDecisionIdx int

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
	// 初始化决策日志记录器
	logDir := fmt.Sprintf("decision_logs/backtest_%s", backtestID)
	decisionLogger := logger.NewDecisionLogger(logDir)

	return &Engine{
		config:          cfg,
		db:              db,
		apiClient:       market.NewAPIClient(),
		mcpClient:       mcpClient,
		timeSimulator:   NewTimeSimulator(cfg.StartTime, cfg.EndTime, cfg.ScanInterval),
		positionManager: NewPositionManager(cfg.InitialBalance),
		orderSimulator:  NewOrderSimulator(cfg.Slippage),
		decisionLogger:  decisionLogger,
		backtestID:      backtestID,
		equitySnapshots: []EquitySnapshot{},
		trades:          []Trade{},
		decisions:       []config.BacktestDecision{},
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

	stepCount := 0
	// 3. 主回测循环
	for e.timeSimulator.HasNext() {
		stepCount++
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

		// 记录决策
		for _, dec := range decisions {
			price := 0.0
			if data, ok := marketDataMap[dec.Symbol]; ok {
				price = data.CurrentPrice
			}

			e.decisions = append(e.decisions, config.BacktestDecision{
				BacktestID: e.backtestID,
				Symbol:     dec.Symbol,
				Action:     dec.Action,
				Price:      price,
				Quantity:   dec.PositionSizeUSD,
				Leverage:   dec.Leverage,
				Confidence: float64(dec.Confidence),
				Reasoning:  dec.Reasoning,
				Timestamp:  currentTime,
			})
		}

		// 实时记录决策到文件（用于前端实时显示）
		// 即使没有决策（Wait/Hold），也记录账户状态和持仓，确保前端能实时更新
		record := &logger.DecisionRecord{
			Timestamp:      currentTime,
			CandidateCoins: e.config.TradingSymbols,
			Decisions:      []logger.DecisionAction{},
			AccountState: logger.AccountSnapshot{
				TotalBalance:          e.positionManager.GetEquity(),
				AvailableBalance:      e.positionManager.GetAvailableBalance(),
				TotalUnrealizedProfit: e.positionManager.GetEquity() - e.positionManager.GetAvailableBalance(), // 简化计算
				PositionCount:         len(e.positionManager.GetAllPositions()),
				MarginUsedPct:         0, // 简化
				InitialBalance:        e.config.InitialBalance,
			},
			Positions: []logger.PositionSnapshot{},
		}

		// 填充持仓快照
		for _, pos := range e.positionManager.GetAllPositions() {
			record.Positions = append(record.Positions, logger.PositionSnapshot{
				Symbol:           pos.Symbol,
				Side:             pos.Side,
				PositionAmt:      pos.Quantity,
				EntryPrice:       pos.EntryPrice,
				MarkPrice:        pos.EntryPrice, // 简化，使用入场价作为标记价
				UnrealizedProfit: pos.UnrealizedPnL,
				Leverage:         float64(pos.Leverage),
			})
		}

		// 填充决策动作并打印控制台日志
		if len(decisions) == 0 {
			// 如果没有决策，打印观望日志
			log.Printf("[Backtest %s] Step %d | Time: %s | AI Decision: WAIT/HOLD (No explicit actions)",
				e.backtestID, stepCount, currentTime.Format("2006-01-02 15:04:05"))
		} else {
			for _, dec := range decisions {
				price := 0.0
				if data, ok := marketDataMap[dec.Symbol]; ok {
					price = data.CurrentPrice
				}

				// 1. 添加到文件记录
				record.Decisions = append(record.Decisions, logger.DecisionAction{
					Action:     dec.Action,
					Symbol:     dec.Symbol,
					Quantity:   dec.PositionSizeUSD,
					Leverage:   dec.Leverage,
					Price:      price,
					Confidence: float64(dec.Confidence),
					Reasoning:  dec.Reasoning,
					Timestamp:  currentTime,
					Success:    true,
					Error:      "",
				})

				// 2. 打印控制台日志 (Backend Log)
				// 截取Reasoning前100个字符避免太长
				reasoningShort := dec.Reasoning
				if len(reasoningShort) > 100 {
					reasoningShort = reasoningShort[:100] + "..."
				}
				log.Printf("[Backtest %s] Step %d | Time: %s | AI Decision: %s %s | Conf: %d%% | Reason: %s",
					e.backtestID, stepCount, currentTime.Format("15:04:05"),
					strings.ToUpper(dec.Action), dec.Symbol, dec.Confidence, reasoningShort)
			}
		}

		// 写入日志文件
		if err := e.decisionLogger.LogDecision(record); err != nil {
			log.Printf("[Backtest %s] Failed to log decision to file: %v", e.backtestID, err)
		}

		// 执行交易决策
		if err := e.executeDecisions(decisions, marketDataMap, currentTime); err != nil {
			log.Printf("[Backtest %s] Failed to execute decisions: %v", e.backtestID, err)
		}

		// 记录净值快照
		e.recordEquitySnapshot(currentTime)

		// 6. 增量保存中间结果 (每1个周期保存一次，确保前端能实时看到进度)
		saveInterval := 1
		if stepCount%saveInterval == 0 {
			if err := e.saveIntermediateResults(); err != nil {
				log.Printf("[Backtest %s] Failed to save intermediate results: %v", e.backtestID, err)
			}
		}
	}

	// 强制平掉所有未平仓的持仓（在回测结束时）
	// 这样可以确保所有交易都有完整的记录（entry + exit）
	e.closeAllOpenPositions()

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

// loadHistoricalData 加载历史数据(优先使用缓存)
func (e *Engine) loadHistoricalData(ctx context.Context) error {
	log.Printf("[Backtest %s] Loading historical data for %d symbols",
		e.backtestID, len(e.config.TradingSymbols))

	// 预加载12小时的数据用于计算指标
	preheatDuration := 12 * time.Hour
	startMs := e.config.StartTime.Add(-preheatDuration).UnixMilli()
	endMs := e.config.EndTime.UnixMilli()

	// 计算需要的间隔(3m用于主数据)
	interval := "3m"

	cacheHits := 0
	cacheMisses := 0

	for i, symbol := range e.config.TradingSymbols {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Printf("[Backtest %s] Loading data for %s (%d/%d)",
			e.backtestID, symbol, i+1, len(e.config.TradingSymbols))

		// 1. 优先尝试从缓存读取
		cachedKlines, found, err := e.db.GetKlineCache(symbol, interval, startMs, endMs)
		if err != nil {
			log.Printf("[Backtest %s] ⚠️ 读取缓存失败: %v, 将从API获取", e.backtestID, err)
		}

		var klines []market.Kline

		if found && len(cachedKlines) > 0 {
			// 缓存命中
			klines = cachedKlines
			cacheHits++
			log.Printf("[Backtest %s] ✅ 缓存命中: %s, 共 %d 条K线", e.backtestID, symbol, len(klines))
		} else {
			// 缓存未命中，从API加载
			cacheMisses++
			log.Printf("[Backtest %s] 🌐 缓存未命中，从API加载: %s", e.backtestID, symbol)

			klines, err = e.apiClient.GetKlinesRange(symbol, interval, startMs, endMs)
			if err != nil {
				return fmt.Errorf("failed to load klines for %s: %w", symbol, err)
			}

			// 保存到缓存供下次使用
			if err := e.db.SaveKlineCache(symbol, interval, startMs, endMs, klines); err != nil {
				log.Printf("[Backtest %s] ⚠️ 保存缓存失败: %v", e.backtestID, err)
			}

			log.Printf("[Backtest %s] Loaded %d klines for %s from API", e.backtestID, len(klines), symbol)
		}

		e.klineCache[symbol] = klines
	}

	log.Printf("[Backtest %s] 📊 数据加载完成: 缓存命中 %d, 缓存未命中 %d (命中率: %.1f%%)",
		e.backtestID, cacheHits, cacheMisses,
		float64(cacheHits)/float64(cacheHits+cacheMisses)*100)

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

// getMockDecisions 生成Mock决策(总是做多)
func (e *Engine) getMockDecisions(marketDataMap map[string]*market.Data) ([]decision.Decision, error) {
	var decisions []decision.Decision
	equity := e.positionManager.GetEquity()
	availableBalance := e.positionManager.GetAvailableBalance()

	for symbol, data := range marketDataMap {
		pos, hasPos := e.positionManager.GetPosition(symbol)
		currentPrice := data.CurrentPrice

		if !hasPos {
			// Open Long
			// Calculate reasonable SL/TP for 1:3 risk/reward
			stopLoss := currentPrice * 0.95   // 5% risk
			takeProfit := currentPrice * 1.15 // 15% reward

			// Position size: 10% of equity
			positionSize := equity * 0.1
			if positionSize > availableBalance {
				positionSize = availableBalance * 0.9
			}

			// Ensure minimum position size
			if positionSize < 20 {
				continue // Skip if balance too low
			}

			// Add some indicator info to reasoning to verify data calculation
			indicators := fmt.Sprintf("Price: %.2f", currentPrice)
			if tfData, ok := data.TimeframeData["3m"]; ok && len(tfData.MidPrices) > 0 {
				indicators += fmt.Sprintf(", LastClose: %.2f", tfData.MidPrices[len(tfData.MidPrices)-1])
			}

			decisions = append(decisions, decision.Decision{
				Symbol:          symbol,
				Action:          "open_long",
				Leverage:        1,
				PositionSizeUSD: positionSize,
				StopLoss:        stopLoss,
				TakeProfit:      takeProfit,
				Confidence:      100,
				Reasoning:       fmt.Sprintf("Mock Mode: Opening Long. %s", indicators),
			})
		} else if pos.Side == "short" {
			// Close Short
			decisions = append(decisions, decision.Decision{
				Symbol:     symbol,
				Action:     "close_short",
				Confidence: 100,
				Reasoning:  fmt.Sprintf("Mock Mode: Closing Short at %.2f", currentPrice),
			})
		} else if pos.Side == "long" {
			// Check if we should close the long position
			// Close if profit > 8% or loss > 4%
			if pos.UnrealizedPnLPct > 8.0 {
				decisions = append(decisions, decision.Decision{
					Symbol:     symbol,
					Action:     "close_long",
					Confidence: 100,
					Reasoning:  fmt.Sprintf("Mock Mode: Take Profit - Closing Long. PnL: %.2f%%", pos.UnrealizedPnLPct),
				})
			} else if pos.UnrealizedPnLPct < -4.0 {
				decisions = append(decisions, decision.Decision{
					Symbol:     symbol,
					Action:     "close_long",
					Confidence: 100,
					Reasoning:  fmt.Sprintf("Mock Mode: Stop Loss - Closing Long. PnL: %.2f%%", pos.UnrealizedPnLPct),
				})
			} else {
				// Hold Long
				decisions = append(decisions, decision.Decision{
					Symbol:     symbol,
					Action:     "hold",
					Confidence: 100,
					Reasoning:  fmt.Sprintf("Mock Mode: Holding Long. PnL: %.2f%%", pos.UnrealizedPnLPct),
				})
			}
		}
	}
	return decisions, nil
}

// calculateMarketData 计算市场数据(复用market.Get逻辑)
func (e *Engine) calculateMarketData(symbol string, klines []market.Kline) (*market.Data, error) {
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

	// 使用market包计算完整指标
	// 使用100个数据点以提供足够的上下文给AI
	tfData := market.CalculateTimeframeData(klines, "3m", 100)
	if tfData != nil {
		data.TimeframeData["3m"] = tfData

		// 填充当前指标值 (使用最新一个点的数据)
		if len(tfData.EMA20Values) > 0 {
			data.CurrentEMA20 = tfData.EMA20Values[len(tfData.EMA20Values)-1]
		}
		if len(tfData.MACDValues) > 0 {
			data.CurrentMACD = tfData.MACDValues[len(tfData.MACDValues)-1]
		}
		if len(tfData.RSI7Values) > 0 {
			data.CurrentRSI7 = tfData.RSI7Values[len(tfData.RSI7Values)-1]
		}
	}

	return data, nil
}

// getDecisions 获取AI决策
func (e *Engine) getDecisions(marketDataMap map[string]*market.Data) ([]decision.Decision, error) {
	// 如果开启了Mock模式，直接返回Mock决策
	if e.config.MockMode {
		return e.getMockDecisions(marketDataMap)
	}

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
		AltcoinLeverage: int(e.config.AltcoinLeverage),
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
		3.0, // 回测默认使用 3.0 风险回报比
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
		// Skip special symbols or actions
		if symbol == "ALL" || strings.ToLower(dec.Action) == "wait" {
			continue
		}

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

// closeAllOpenPositions 强制平掉所有未平仓的持仓（回测结束时调用）
func (e *Engine) closeAllOpenPositions() {
	// 获取所有持仓
	positions := e.positionManager.GetAllPositions()
	if len(positions) == 0 {
		return
	}

	log.Printf("[Backtest %s] Closing %d open positions at backtest end", e.backtestID, len(positions))

	// 获取最后时刻的市场数据作为平仓价格
	endTime := e.config.EndTime
	marketDataMap, err := e.getMarketDataAtTime(endTime)
	if err != nil {
		log.Printf("[Backtest %s] Failed to get market data for final close: %v", e.backtestID, err)
		return
	}

	// 逐个平仓
	for _, pos := range positions {
		symbol := pos.Symbol
		data, exists := marketDataMap[symbol]
		if !exists {
			log.Printf("[Backtest %s] No market data for %s, skipping final close", e.backtestID, symbol)
			continue
		}

		// 再次确认持仓存在(虽然是从GetAllPositions获取的)
		_, hasPos := e.positionManager.GetPosition(symbol)
		if !hasPos {
			continue
		}

		// 使用最后的市场价格作为平仓价
		closePrice := data.CurrentPrice

		// 平仓
		trade, err := e.positionManager.ClosePosition(symbol, closePrice)
		if err != nil {
			log.Printf("[Backtest %s] Failed to close position %s: %v", e.backtestID, symbol, err)
			continue
		}

		// 记录交易
		trade.ExitTime = endTime
		trade.Fee = trade.Quantity * closePrice * 0.0004 // 0.04% fee
		e.trades = append(e.trades, *trade)

		log.Printf("[Backtest %s] Force closed %s position: %s @ %.2f (PnL: %.2f, %.2f%%)",
			e.backtestID, pos.Side, symbol, closePrice, trade.PnL, trade.PnLPct)
	}
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

	// 保存决策记录
	if len(e.decisions) > 0 {
		if err := e.db.SaveBacktestDecisions(e.backtestID, e.decisions); err != nil {
			return fmt.Errorf("failed to save decisions: %w", err)
		}
	}

	return nil
}

// saveIntermediateResults 保存中间结果
func (e *Engine) saveIntermediateResults() error {
	// 1. 计算当前统计数据
	result := e.calculateResult()

	// 2. 更新回测统计信息
	backtestRun := &config.BacktestRun{
		FinalEquity: result.FinalEquity,
		TotalPnL:    result.TotalPnL,
		TotalPnLPct: result.TotalPnLPct,
		MaxDrawdown: result.MaxDrawdown,
		SharpeRatio: result.SharpeRatio,
		WinRate:     result.WinRate,
		TotalTrades: result.TotalTrades,
	}
	if err := e.db.UpdateBacktestStats(e.backtestID, backtestRun); err != nil {
		return fmt.Errorf("failed to update backtest stats: %w", err)
	}

	// 3. 增量保存净值快照
	if len(e.equitySnapshots) > e.lastSavedSnapshotIdx {
		newSnapshots := e.equitySnapshots[e.lastSavedSnapshotIdx:]
		dbSnapshots := make([]config.BacktestEquitySnapshot, len(newSnapshots))
		for i, s := range newSnapshots {
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
		e.lastSavedSnapshotIdx = len(e.equitySnapshots)
	}

	// 4. 增量保存交易记录
	if len(e.trades) > e.lastSavedTradeIdx {
		newTrades := e.trades[e.lastSavedTradeIdx:]
		dbTrades := make([]config.BacktestTrade, len(newTrades))
		for i, t := range newTrades {
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
		e.lastSavedTradeIdx = len(e.trades)
	}

	// 5. 增量保存决策记录
	if len(e.decisions) > e.lastSavedDecisionIdx {
		newDecisions := e.decisions[e.lastSavedDecisionIdx:]
		if err := e.db.SaveBacktestDecisions(e.backtestID, newDecisions); err != nil {
			return fmt.Errorf("failed to save decisions: %w", err)
		}
		e.lastSavedDecisionIdx = len(e.decisions)
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
		return int(e.config.AltcoinLeverage)
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

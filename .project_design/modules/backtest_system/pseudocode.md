# 回测系统伪代码

## 核心结构体定义

```go
// 回测引擎主结构
type Engine struct {
    // 配置和依赖
    config          *Config
    db              *config.Database
    apiClient       *market.APIClient
    mcpClient       *mcp.Client

    // 子模块
    timeSimulator   *TimeSimulator
    positionManager *PositionManager
    orderSimulator  *OrderSimulator

    // 状态管理
    backtestID      string
    equitySnapshots []EquitySnapshot
    trades          []Trade
    klineCache      map[string][]market.Kline

    // 运行状态
    isRunning       bool
    progress        float64
    mu              sync.RWMutex
}

// 回测配置结构
type Config struct {
    // 基础配置
    TraderID             string
    UserID               string
    BacktestName         string
    StartTime            time.Time
    EndTime              time.Time
    InitialBalance       float64
    ScanInterval         time.Duration
    TradingSymbols       []string

    // 交易参数
    Slippage             int
    CommissionRate       float64
    UseTraderConfig      bool
    BTCETHLeverage       int
    AltcoinLeverage      int

    // AI配置
    IndicatorConfig      *market.IndicatorConfig
    CustomPrompt         string
    OverrideBasePrompt   bool
    SystemPromptTemplate string

    // 高级选项
    EnableMarginCall     bool
    EnableFundingRate    bool
    DataQuality          string
    MaxParallelRequests  int
}

// 时间模拟器
type TimeSimulator struct {
    startTime     time.Time
    endTime       time.Time
    interval      time.Duration
    currentTime   time.Time
    totalSteps    int
    currentStep   int
}

// 持仓管理器
type PositionManager struct {
    initialBalance  float64
    currentEquity   float64
    availableFunds  float64
    positions       map[string]*Position
    trades          []Trade
    totalFees       float64
    marginUsed      float64
    maxDrawdown     float64
    peakEquity      float64
}

// 订单模拟器
type OrderSimulator struct {
    slippage           int
    commissionRate     float64
    enableMarketImpact bool
    liquidityData      map[string]*LiquidityData
}
```

## 主要方法实现

### 1. 回测引擎初始化

```go
// NewEngine 创建新的回测引擎
func NewEngine(
    backtestID string,
    cfg *Config,
    db *config.Database,
    mcpClient *mcp.Client,
) *Engine {

    engine := &Engine{
        config:          cfg,
        db:              db,
        apiClient:       market.NewAPIClient(),
        mcpClient:       mcpClient,
        backtestID:      backtestID,
        equitySnapshots: make([]EquitySnapshot, 0),
        trades:          make([]Trade, 0),
        klineCache:      make(map[string][]market.Kline),
        isRunning:       false,
    }

    // 初始化子模块
    engine.timeSimulator = NewTimeSimulator(
        cfg.StartTime,
        cfg.EndTime,
        cfg.ScanInterval,
    )

    engine.positionManager = NewPositionManager(cfg.InitialBalance)
    engine.orderSimulator = NewOrderSimulator(
        cfg.Slippage,
        cfg.CommissionRate,
    )

    return engine
}
```

### 2. 主回测执行流程

```go
// Run 执行完整的回测流程
func (e *Engine) Run(ctx context.Context) error {
    log.Printf("[Backtest %s] Starting execution", e.backtestID)

    // 设置运行状态
    e.mu.Lock()
    e.isRunning = true
    e.mu.Unlock()

    defer func() {
        e.mu.Lock()
        e.isRunning = false
        e.mu.Unlock()
    }()

    // 1. 加载历史数据
    if err := e.loadHistoricalData(ctx); err != nil {
        return fmt.Errorf("failed to load historical data: %w", err)
    }

    // 2. 记录初始状态
    e.recordEquitySnapshot(e.timeSimulator.CurrentTime())

    // 3. 主回测循环
    for e.timeSimulator.HasNext() {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        // 推进时间
        e.timeSimulator.Next()
        currentTime := e.timeSimulator.CurrentTime()

        // 更新进度
        progress := e.timeSimulator.Progress()
        if err := e.updateProgress(progress); err != nil {
            log.Printf("[Backtest %s] Failed to update progress: %v", e.backtestID, err)
        }

        // 获取市场数据
        marketDataMap, err := e.getMarketDataAtTime(currentTime)
        if err != nil {
            log.Printf("[Backtest %s] Failed to get market data: %v", e.backtestID, err)
            continue
        }

        // 更新持仓状态
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

    log.Printf("[Backtest %s] Completed successfully", e.backtestID)
    return nil
}
```

### 3. 历史数据加载

```go
// loadHistoricalData 加载历史K线数据
func (e *Engine) loadHistoricalData(ctx context.Context) error {
    log.Printf("[Backtest %s] Loading historical data", e.backtestID)

    // 计算时间范围
    startMs := e.config.StartTime.UnixMilli()
    endMs := e.config.EndTime.UnixMilli()

    // 使用3分钟间隔作为主数据源
    interval := "3m"

    // 并发加载各币种数据
    semaphore := make(chan struct{}, e.config.MaxParallelRequests)
    var wg sync.WaitGroup
    var mu sync.Mutex
    var loadErrors []error

    for i, symbol := range e.config.TradingSymbols {
        wg.Add(1)
        go func(symbol string, index int) {
            defer wg.Done()

            semaphore <- struct{}{} // 获取信号量
            defer func() { <-semaphore }() // 释放信号量

            select {
            case <-ctx.Done():
                return
            default:
            }

            log.Printf("[Backtest %s] Loading %s (%d/%d)",
                e.backtestID, symbol, index+1, len(e.config.TradingSymbols))

            // 调用API获取K线数据
            klines, err := e.apiClient.GetKlinesRange(symbol, interval, startMs, endMs)
            if err != nil {
                mu.Lock()
                loadErrors = append(loadErrors,
                    fmt.Errorf("failed to load %s: %w", symbol, err))
                mu.Unlock()
                return
            }

            // 验证数据质量
            if len(klines) < 100 {
                log.Printf("[Backtest %s] Warning: Insufficient data for %s: %d klines",
                    e.backtestID, symbol, len(klines))
            }

            // 缓存数据
            e.mu.Lock()
            e.klineCache[symbol] = klines
            e.mu.Unlock()

            log.Printf("[Backtest %s] Loaded %d klines for %s",
                e.backtestID, len(klines), symbol)
        }(symbol, i)
    }

    wg.Wait()

    // 检查加载错误
    if len(loadErrors) > 0 {
        log.Printf("[Backtest %s] Data loading errors: %v", e.backtestID, loadErrors)
        // 如果关键币种数据加载失败，返回错误
        for _, err := range loadErrors {
            if strings.Contains(err.Error(), "BTCUSDT") ||
               strings.Contains(err.Error(), "ETHUSDT") {
                return err
            }
        }
    }

    log.Printf("[Backtest %s] Historical data loading completed", e.backtestID)
    return nil
}
```

### 4. 市场数据处理

```go
// getMarketDataAtTime 获取指定时间的市场数据
func (e *Engine) getMarketDataAtTime(t time.Time) (map[string]*market.Data, error) {
    dataMap := make(map[string]*market.Data)
    targetMs := t.UnixMilli()

    e.mu.RLock()
    defer e.mu.RUnlock()

    for _, symbol := range e.config.TradingSymbols {
        klines, exists := e.klineCache[symbol]
        if !exists {
            continue
        }

        // 获取目标时间附近的K线数据
        recentKlines := e.getRecentKlines(klines, targetMs, 100)
        if len(recentKlines) < 20 {
            log.Printf("[Backtest %s] Insufficient data for %s at %s",
                e.backtestID, symbol, t)
            continue
        }

        // 计算技术指标
        marketData, err := e.calculateMarketData(symbol, recentKlines)
        if err != nil {
            log.Printf("[Backtest %s] Failed to calculate market data for %s: %v",
                e.backtestID, symbol, err)
            continue
        }

        dataMap[symbol] = marketData
    }

    if len(dataMap) == 0 {
        return nil, fmt.Errorf("no valid market data available")
    }

    return dataMap, nil
}

// getRecentKlines 获取最近的K线数据
func (e *Engine) getRecentKlines(klines []market.Kline, targetMs int64, count int) []market.Kline {
    var result []market.Kline

    // 从后往前查找目标时间点
    for i := len(klines) - 1; i >= 0; i-- {
        if klines[i].CloseTime <= targetMs {
            // 提取指定数量的K线
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

// calculateMarketData 计算市场数据和技术指标
func (e *Engine) calculateMarketData(symbol string, klines []market.Kline) (*market.Data, error) {
    if len(klines) == 0 {
        return nil, fmt.Errorf("no klines available")
    }

    lastKline := klines[len(klines)-1]

    // 创建基础市场数据结构
    data := &market.Data{
        Symbol:        symbol,
        CurrentPrice:  lastKline.Close,
        TimeframeData: make(map[string]*market.TimeframeData),
    }

    // 提取价格和成交量序列
    prices := make([]float64, len(klines))
    volumes := make([]float64, len(klines))
    highs := make([]float64, len(klines))
    lows := make([]float64, len(klines))

    for i, k := range klines {
        prices[i] = k.Close
        volumes[i] = k.Volume
        highs[i] = k.High
        lows[i] = k.Low
    }

    // 创建3分钟时间框架数据
    data.TimeframeData["3m"] = &market.TimeframeData{
        Timeframe:  "3m",
        DataPoints: len(klines),
        MidPrices:  prices,
        Highs:     highs,
        Lows:      lows,
        Volume:     volumes,
    }

    // 计算技术指标（如果配置了的话）
    if e.config.IndicatorConfig != nil {
        e.calculateTechnicalIndicators(data, klines)
    }

    return data, nil
}

// calculateTechnicalIndicators 计算技术指标
func (e *Engine) calculateTechnicalIndicators(data *market.Data, klines []market.Kline) {
    // 提取价格序列
    closes := make([]float64, len(klines))
    for i, k := range klines {
        closes[i] = k.Close
    }

    // 计算EMA
    if e.config.IndicatorConfig.EnableEMA {
        ema20 := e.calculateEMA(closes, 20)
        ema50 := e.calculateEMA(closes, 50)
        data.EMA20 = &ema20
        data.EMA50 = &ema50
    }

    // 计算MACD
    if e.config.IndicatorConfig.EnableMACD {
        macd := e.calculateMACD(closes)
        data.MACD = &macd
    }

    // 计算RSI
    if e.config.IndicatorConfig.EnableRSI {
        rsi := e.calculateRSI(closes, 14)
        data.RSI14 = &rsi
    }

    // 计算ATR
    if e.config.IndicatorConfig.EnableATR {
        atr := e.calculateATR(klines, 14)
        data.ATR14 = &atr
    }
}
```

### 5. AI决策获取

```go
// getDecisions 获取AI交易决策
func (e *Engine) getDecisions(marketDataMap map[string]*market.Data) ([]decision.Decision, error) {
    // 构建决策上下文
    ctx := &decision.Context{
        Account: decision.AccountInfo{
            TotalEquity:      e.positionManager.GetEquity(),
            AvailableBalance: e.positionManager.GetAvailableBalance(),
            MarginUsed:       e.positionManager.GetMarginUsed(),
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

            // 计算盈亏百分比
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
                EntryTime:        pos.EntryTime,
            })
        } else {
            // 未持仓的币种作为候选
            ctx.CandidateCoins = append(ctx.CandidateCoins, decision.CandidateCoin{
                Symbol:  symbol,
                Sources: []string{"backtest"},
            })
        }
    }

    // 调用AI决策引擎
    fullDecision, err := decision.GetFullDecisionWithCustomPrompt(
        ctx,
        e.mcpClient,
        e.config.CustomPrompt,
        e.config.OverrideBasePrompt,
        e.config.SystemPromptTemplate,
    )

    if err != nil {
        return nil, fmt.Errorf("AI decision failed: %w", err)
    }

    return fullDecision.Decisions, nil
}
```

### 6. 交易决策执行

```go
// executeDecisions 执行交易决策
func (e *Engine) executeDecisions(
    decisions []decision.Decision,
    marketDataMap map[string]*market.Data,
    currentTime time.Time,
) error {

    for _, dec := range decisions {
        // 获取对应的市场数据
        marketData, exists := marketDataMap[dec.Symbol]
        if !exists {
            log.Printf("[Backtest %s] No market data for %s", e.backtestID, dec.Symbol)
            continue
        }

        currentPrice := marketData.CurrentPrice

        // 根据决策类型执行相应操作
        switch strings.ToLower(dec.Action) {
        case "open_long", "open_short":
            if err := e.executeOpenPosition(dec, currentPrice, currentTime); err != nil {
                log.Printf("[Backtest %s] Failed to open position %s: %v",
                    e.backtestID, dec.Symbol, err)
            }

        case "close_long", "close_short":
            if err := e.executeClosePosition(dec, currentPrice, currentTime); err != nil {
                log.Printf("[Backtest %s] Failed to close position %s: %v",
                    e.backtestID, dec.Symbol, err)
            }

        case "wait", "hold":
            // 观望或持仓，无需执行操作
            log.Printf("[Backtest %s] Decision: %s for %s",
                e.backtestID, dec.Action, dec.Symbol)

        default:
            log.Printf("[Backtest %s] Unknown action: %s for %s",
                e.backtestID, dec.Action, dec.Symbol)
        }
    }

    return nil
}
```

### 7. 开仓执行

```go
// executeOpenPosition 执行开仓操作
func (e *Engine) executeOpenPosition(dec decision.Decision, marketPrice float64, currentTime time.Time) error {
    symbol := dec.Symbol

    // 检查是否已有持仓
    if e.positionManager.HasPosition(symbol) {
        return fmt.Errorf("position already exists for %s", symbol)
    }

    // 确定杠杆倍数
    leverage := e.getLeverageForSymbol(symbol)

    // 风险管理：每次交易风险不超过账户的2%
    accountEquity := e.positionManager.GetEquity()
    riskPercent := 2.0

    // 计算仓位价值
    riskAmount := accountEquity * (riskPercent / 100.0)
    positionValue := riskAmount * float64(leverage)
    quantity := positionValue / marketPrice

    // 确定交易方向
    side := "long"
    if strings.Contains(strings.ToLower(dec.Action), "short") {
        side = "short"
    }

    // 模拟订单执行
    executionPrice, fee, err := e.orderSimulator.ExecuteMarketOrder(
        side, "open", marketPrice, quantity, leverage,
    )
    if err != nil {
        return fmt.Errorf("order execution failed: %w", err)
    }

    // 更新持仓管理器
    if err := e.positionManager.OpenPosition(symbol, side, executionPrice, quantity, leverage); err != nil {
        return fmt.Errorf("failed to open position: %w", err)
    }

    // 创建交易记录
    trade := Trade{
        ID:         uuid.New().String(),
        Symbol:     symbol,
        Side:       side,
        Action:     "open",
        EntryPrice: executionPrice,
        Quantity:   quantity,
        Leverage:   leverage,
        Fee:        fee,
        EntryTime:  currentTime,
        DecisionID: dec.ID,
        Reasoning:  dec.Reasoning,
        Confidence: dec.Confidence,
    }

    // 保存交易记录
    e.mu.Lock()
    e.trades = append(e.trades, trade)
    e.mu.Unlock()

    log.Printf("[Backtest %s] Opened %s position: %s @ %.6f (qty: %.6f, lev: %dx, fee: %.6f)",
        e.backtestID, side, symbol, executionPrice, quantity, leverage, fee)

    return nil
}
```

### 8. 平仓执行

```go
// executeClosePosition 执行平仓操作
func (e *Engine) executeClosePosition(dec decision.Decision, marketPrice float64, currentTime time.Time) error {
    symbol := dec.Symbol

    // 获取持仓信息
    pos, exists := e.positionManager.GetPosition(symbol)
    if !exists {
        return fmt.Errorf("no position to close for %s", symbol)
    }

    // 模拟订单执行
    executionPrice, fee, err := e.orderSimulator.ExecuteMarketOrder(
        pos.Side, "close", marketPrice, pos.Quantity, pos.Leverage,
    )
    if err != nil {
        return fmt.Errorf("order execution failed: %w", err)
    }

    // 更新持仓管理器
    trade, err := e.positionManager.ClosePosition(symbol, executionPrice)
    if err != nil {
        return fmt.Errorf("failed to close position: %w", err)
    }

    // 更新交易记录
    trade.Fee = fee
    trade.ExitTime = currentTime
    trade.Duration = e.calculateDuration(pos.EntryTime, currentTime)

    e.mu.Lock()
    e.trades = append(e.trades, *trade)
    e.mu.Unlock()

    log.Printf("[Backtest %s] Closed %s position: %s @ %.6f (PnL: %.6f, %.2f%%, fee: %.6f)",
        e.backtestID, pos.Side, symbol, executionPrice, trade.PnL, trade.PnLPct, fee)

    return nil
}
```

### 9. 持仓管理

```go
// NewPositionManager 创建持仓管理器
func NewPositionManager(initialBalance float64) *PositionManager {
    return &PositionManager{
        initialBalance:  initialBalance,
        currentEquity:   initialBalance,
        availableFunds:  initialBalance,
        positions:       make(map[string]*Position),
        trades:          make([]Trade, 0),
        peakEquity:      initialBalance,
    }
}

// OpenPosition 开仓
func (pm *PositionManager) OpenPosition(symbol, side string, entryPrice, quantity float64, leverage int) error {
    // 计算保证金需求
    positionValue := quantity * entryPrice
    marginRequired := positionValue / float64(leverage)

    // 检查资金是否充足
    if marginRequired > pm.availableFunds {
        return fmt.Errorf("insufficient funds: required %.2f, available %.2f",
            marginRequired, pm.availableFunds)
    }

    // 创建持仓记录
    position := &Position{
        Symbol:         symbol,
        Side:           side,
        EntryPrice:     entryPrice,
        Quantity:       quantity,
        Leverage:       leverage,
        MarginUsed:     marginRequired,
        EntryTime:      time.Now(),
    }

    // 更新状态
    pm.positions[symbol] = position
    pm.availableFunds -= marginRequired
    pm.marginUsed += marginRequired

    log.Printf("Opened position: %s %s @ %.6f, qty: %.6f, lev: %dx, margin: %.2f",
        side, symbol, entryPrice, quantity, leverage, marginRequired)

    return nil
}

// ClosePosition 平仓
func (pm *PositionManager) ClosePosition(symbol, exitPrice float64) (*Trade, error) {
    pos, exists := pm.positions[symbol]
    if !exists {
        return nil, fmt.Errorf("no position found for %s", symbol)
    }

    // 计算盈亏
    priceChange := exitPrice - pos.EntryPrice
    if pos.Side == "short" {
        priceChange = -priceChange
    }

    pnl := priceChange * pos.Quantity
    pnlPct := (pnl / pos.MarginUsed) * 100

    // 创建交易记录
    trade := &Trade{
        Symbol:     symbol,
        Side:       pos.Side,
        Action:     "close",
        EntryPrice: pos.EntryPrice,
        ExitPrice:  exitPrice,
        Quantity:   pos.Quantity,
        Leverage:   pos.Leverage,
        PnL:        pnl,
        PnLPct:     pnlPct,
        EntryTime:  pos.EntryTime,
    }

    // 更新状态
    delete(pm.positions, symbol)
    pm.availableFunds += pos.MarginUsed + pnl
    pm.marginUsed -= pos.MarginUsed
    pm.currentEquity += pnl

    // 更新峰值权益
    if pm.currentEquity > pm.peakEquity {
        pm.peakEquity = pm.currentEquity
    }

    // 计算最大回撤
    drawdown := (pm.peakEquity - pm.currentEquity) / pm.peakEquity * 100
    if drawdown > pm.maxDrawdown {
        pm.maxDrawdown = drawdown
    }

    log.Printf("Closed position: %s %s @ %.6f, PnL: %.6f (%.2f%%)",
        pos.Side, symbol, exitPrice, pnl, pnlPct)

    return trade, nil
}
```

### 10. 订单模拟

```go
// NewOrderSimulator 创建订单模拟器
func NewOrderSimulator(slippage int, commissionRate float64) *OrderSimulator {
    return &OrderSimulator{
        slippage:       slippage,
        commissionRate: commissionRate,
    }
}

// ExecuteMarketOrder 模拟市价单执行
func (os *OrderSimulator) ExecuteMarketOrder(
    side, action string,
    marketPrice, quantity float64,
    leverage int,
) (executionPrice float64, fee float64, err error) {

    // 计算滑点影响
    slippageBps := os.slippage
    slippageRate := float64(slippageBps) / 10000.0

    // 根据交易方向调整滑点
    if (side == "long" && action == "open") || (side == "short" && action == "close") {
        // 买入，价格向上滑
        executionPrice = marketPrice * (1 + slippageRate)
    } else {
        // 卖出，价格向下滑
        executionPrice = marketPrice * (1 - slippageRate)
    }

    // 计算交易金额
    tradeValue := quantity * executionPrice

    // 计算手续费
    fee = tradeValue * os.commissionRate

    return executionPrice, fee, nil
}
```

### 11. 性能计算

```go
// calculateResult 计算回测结果
func (e *Engine) calculateResult() *Result {
    finalEquity := e.positionManager.GetEquity()
    totalPnL := finalEquity - e.config.InitialBalance
    totalPnLPct := (totalPnL / e.config.InitialBalance) * 100

    // 计算风险指标
    maxDrawdown := e.positionManager.maxDrawdown
    sharpeRatio := e.calculateSharpeRatio()

    // 计算交易统计
    winCount, loseCount := 0, 0
    totalTrades := 0

    for _, trade := range e.trades {
        if trade.Action == "close" {
            totalTrades++
            if trade.PnL > 0 {
                winCount++
            } else {
                loseCount++
            }
        }
    }

    winRate := 0.0
    if totalTrades > 0 {
        winRate = (float64(winCount) / float64(totalTrades)) * 100
    }

    // 计算盈利因子
    totalWins, totalLosses := 0.0, 0.0
    for _, trade := range e.trades {
        if trade.Action == "close" {
            if trade.PnL > 0 {
                totalWins += trade.PnL
            } else {
                totalLosses += math.Abs(trade.PnL)
            }
        }
    }

    profitFactor := 0.0
    if totalLosses > 0 {
        profitFactor = totalWins / totalLosses
    }

    return &Result{
        BacktestID:       e.backtestID,
        Config:           e.config,
        FinalEquity:     finalEquity,
        TotalPnL:        totalPnL,
        TotalPnLPct:     totalPnLPct,
        MaxDrawdown:     maxDrawdown,
        SharpeRatio:     sharpeRatio,
        WinRate:         winRate,
        ProfitFactor:    profitFactor,
        TotalTrades:     totalTrades,
        WinningTrades:   winCount,
        LosingTrades:    loseCount,
        EquitySnapshots: e.equitySnapshots,
        Trades:          e.trades,
    }
}

// calculateSharpeRatio 计算夏普比率
func (e *Engine) calculateSharpeRatio() float64 {
    if len(e.equitySnapshots) < 2 {
        return 0.0
    }

    // 计算日收益率序列
    returns := make([]float64, len(e.equitySnapshots)-1)
    for i := 1; i < len(e.equitySnapshots); i++ {
        prevEquity := e.equitySnapshots[i-1].Equity
        currEquity := e.equitySnapshots[i].Equity
        if prevEquity > 0 {
            returns[i-1] = (currEquity - prevEquity) / prevEquity
        }
    }

    // 计算平均收益率和标准差
    meanReturn := 0.0
    for _, r := range returns {
        meanReturn += r
    }
    meanReturn /= float64(len(returns))

    variance := 0.0
    for _, r := range returns {
        variance += math.Pow(r-meanReturn, 2)
    }
    variance /= float64(len(returns))

    stdDev := math.Sqrt(variance)

    // 年化夏普比率 (假设252个交易日)
    if stdDev > 0 {
        return (meanReturn * 252) / (stdDev * math.Sqrt(252))
    }

    return 0.0
}
```

### 12. 结果保存

```go
// saveResult 保存回测结果
func (e *Engine) saveResult(result *Result) error {
    // 1. 更新回测记录状态
    backtestRun := &config.BacktestRun{
        Status:        "completed",
        FinalEquity:  result.FinalEquity,
        TotalPnL:     result.TotalPnL,
        TotalPnLPct:  result.TotalPnLPct,
        MaxDrawdown:  result.MaxDrawdown,
        SharpeRatio:  result.SharpeRatio,
        WinRate:      result.WinRate,
        TotalTrades: result.TotalTrades,
        CompletedAt:  time.Now(),
    }

    if err := e.db.SaveBacktestResult(e.backtestID, backtestRun); err != nil {
        return fmt.Errorf("failed to save backtest result: %w", err)
    }

    // 2. 批量保存净值快照
    if len(result.EquitySnapshots) > 0 {
        dbSnapshots := make([]config.BacktestEquitySnapshot, len(result.EquitySnapshots))
        for i, snapshot := range result.EquitySnapshots {
            dbSnapshots[i] = config.BacktestEquitySnapshot{
                BacktestID: e.backtestID,
                Time:       snapshot.Time,
                Equity:     snapshot.Equity,
                PnL:        snapshot.PnL,
                PnLPct:     snapshot.PnLPct,
            }
        }

        if err := e.db.SaveEquitySnapshots(e.backtestID, dbSnapshots); err != nil {
            return fmt.Errorf("failed to save equity snapshots: %w", err)
        }
    }

    // 3. 批量保存交易记录
    if len(result.Trades) > 0 {
        dbTrades := make([]config.BacktestTrade, len(result.Trades))
        for i, trade := range result.Trades {
            dbTrades[i] = config.BacktestTrade{
                BacktestID:    e.backtestID,
                Symbol:       trade.Symbol,
                Side:         trade.Side,
                Action:       trade.Action,
                EntryPrice:   &trade.EntryPrice,
                ExitPrice:    &trade.ExitPrice,
                Quantity:     trade.Quantity,
                Leverage:     trade.Leverage,
                PnL:          trade.PnL,
                PnLPct:       trade.PnLPct,
                Fee:          trade.Fee,
                EntryTime:    &trade.EntryTime,
                ExitTime:     &trade.ExitTime,
                DecisionID:   &trade.DecisionID,
                Reasoning:    &trade.Reasoning,
                Confidence:   &trade.Confidence,
            }
        }

        if err := e.db.SaveBacktestTrades(e.backtestID, dbTrades); err != nil {
            return fmt.Errorf("failed to save trades: %w", err)
        }
    }

    log.Printf("[Backtest %s] Results saved successfully", e.backtestID)
    return nil
}
```

### 13. 工具函数

```go
// getLeverageForSymbol 获取币种杠杆配置
func (e *Engine) getLeverageForSymbol(symbol string) int {
    // BTC/ETH 使用较低杠杆
    if strings.Contains(symbol, "BTC") || strings.Contains(symbol, "ETH") {
        if e.config.BTCETHLeverage > 0 {
            return e.config.BTCETHLeverage
        }
        return 10 // 默认值
    }

    // 山寨币使用较高杠杆
    if e.config.AltcoinLeverage > 0 {
        return e.config.AltcoinLeverage
    }
    return 20 // 默认值
}

// calculateDuration 计算持仓时长
func (e *Engine) calculateDuration(entryTime, exitTime time.Time) string {
    duration := exitTime.Sub(entryTime)

    hours := int(duration.Hours())
    minutes := int(duration.Minutes()) % 60

    if hours > 24 {
        days := hours / 24
        hours = hours % 24
        return fmt.Sprintf("%dd%dh%dm", days, hours, minutes)
    }

    return fmt.Sprintf("%dh%dm", hours, minutes)
}

// recordEquitySnapshot 记录净值快照
func (e *Engine) recordEquitySnapshot(t time.Time) {
    equity := e.positionManager.GetEquity()
    pnl := equity - e.config.InitialBalance
    pnlPct := (pnl / e.config.InitialBalance) * 100

    // 计算当前回撤
    drawdown := 0.0
    if e.positionManager.peakEquity > 0 {
        drawdown = (e.positionManager.peakEquity - equity) / e.positionManager.peakEquity * 100
    }

    snapshot := EquitySnapshot{
        Time:       t,
        Equity:     equity,
        PnL:        pnl,
        PnLPct:     pnlPct,
        Drawdown:   drawdown,
        UsedMargin: e.positionManager.marginUsed,
        FreeMargin: e.positionManager.availableFunds,
    }

    e.mu.Lock()
    e.equitySnapshots = append(e.equitySnapshots, snapshot)
    e.mu.Unlock()
}

// updateProgress 更新执行进度
func (e *Engine) updateProgress(progress float64) error {
    e.mu.Lock()
    e.progress = progress
    e.mu.Unlock()

    return e.db.UpdateBacktestStatus(e.backtestID, "running", progress)
}
```
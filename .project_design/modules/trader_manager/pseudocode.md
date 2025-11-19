# 交易员管理器伪代码

## 核心结构体定义

```go
// 交易员管理器主结构
type TraderManager struct {
    traders          map[string]*trader.AutoTrader // 交易员实例映射，key为trader ID
    competitionCache *CompetitionCache             // 竞赛数据缓存
    mu               sync.RWMutex                  // 读写锁，保护traders映射
}

// 竞赛数据缓存结构
type CompetitionCache struct {
    data      map[string]interface{} // 缓存的竞赛数据
    timestamp time.Time             // 缓存更新时间
    mu        sync.RWMutex          // 保护缓存的读写锁
}

// 交易员结果传递结构（并发获取用）
type traderResult struct {
    index int                   // 交易员在列表中的索引
    data  map[string]interface{} // 交易员数据
}
```

## 初始化方法

```go
// 创建新的交易员管理器
func NewTraderManager() *TraderManager {
    return &TraderManager{
        traders: make(map[string]*trader.AutoTrader), // 初始化空的交易员映射
        competitionCache: &CompetitionCache{
            data: make(map[string]interface{}), // 初始化空的缓存
        },
    }
}
```

## 数据库加载方法

```go
// 从数据库加载所有交易员配置
func (tm *TraderManager) LoadTradersFromDatabase(database *config.Database) error {
    // 1. 获取写锁，确保加载期间没有并发修改
    tm.mu.Lock()
    defer tm.mu.Unlock()

    // 2. 获取系统中所有用户
    userIDs, err := database.GetAllUsers()
    if err != nil {
        return fmt.Errorf("获取用户列表失败: %w", err)
    }

    log.Printf("📋 发现 %d 个用户，开始加载所有交易员配置...", len(userIDs))

    // 3. 初始化变量收集所有交易员
    var allTraders []*config.TraderRecord
    for _, userID := range userIDs {
        // 获取每个用户的交易员配置
        traders, err := database.GetTraders(userID)
        if err != nil {
            log.Printf("⚠️ 获取用户 %s 的交易员失败: %v", userID, err)
            continue
        }
        log.Printf("📋 用户 %s: %d 个交易员", userID, len(traders))
        allTraders = append(allTraders, traders...)
    }

    log.Printf("📋 总共加载 %d 个交易员配置", len(allTraders))

    // 4. 获取系统级配置（一次性获取，避免重复查询）
    systemConfigs, err := tm.loadSystemConfigs(database)
    if err != nil {
        return fmt.Errorf("加载系统配置失败: %w", err)
    }

    // 5. 为每个交易员创建实例
    for _, traderCfg := range allTraders {
        err := tm.loadSingleTrader(traderCfg, systemConfigs, database)
        if err != nil {
            log.Printf("❌ 添加交易员 %s 失败: %v", traderCfg.Name, err)
            continue
        }
    }

    log.Printf("✓ 成功加载 %d 个交易员到内存", len(tm.traders))
    return nil
}

// 加载系统配置（辅助方法）
func (tm *TraderManager) loadSystemConfigs(database *config.Database) (*SystemConfigs, error) {
    configs := &SystemConfigs{}

    // 获取各项系统配置
    maxDailyLossStr, _ := database.GetSystemConfig("max_daily_loss")
    maxDrawdownStr, _ := database.GetSystemConfig("max_drawdown")
    stopTradingMinutesStr, _ := database.GetSystemConfig("stop_trading_minutes")
    defaultCoinsStr, _ := database.GetSystemConfig("default_coins")

    // 解析数值配置
    configs.MaxDailyLoss = parseFloatOrDefault(maxDailyLossStr, 10.0)
    configs.MaxDrawdown = parseFloatOrDefault(maxDrawdownStr, 20.0)
    configs.StopTradingMinutes = parseIntOrDefault(stopTradingMinutesStr, 60)

    // 解析默认币种列表
    if defaultCoinsStr != "" {
        if err := json.Unmarshal([]byte(defaultCoinsStr), &configs.DefaultCoins); err != nil {
            log.Printf("⚠️ 解析默认币种配置失败: %v，使用空列表", err)
            configs.DefaultCoins = []string{}
        }
    }

    return configs, nil
}
```

## 单个交易员加载方法

```go
// 加载单个交易员（核心逻辑）
func (tm *TraderManager) loadSingleTrader(
    traderCfg *config.TraderRecord,
    systemConfigs *SystemConfigs,
    database *config.Database,
    userID string,
) error {
    // 1. 检查是否已存在
    if _, exists := tm.traders[traderCfg.ID]; exists {
        return fmt.Errorf("trader ID '%s' 已存在", traderCfg.ID)
    }

    // 2. 获取AI模型配置
    aiModelCfg, err := tm.getAIModelConfig(database, userID, traderCfg.AIModelID)
    if err != nil {
        return fmt.Errorf("获取AI模型配置失败: %w", err)
    }

    // 3. 获取交易所配置
    exchangeCfg, err := tm.getExchangeConfig(database, userID, traderCfg.ExchangeID)
    if err != nil {
        return fmt.Errorf("获取交易所配置失败: %w", err)
    }

    // 4. 获取用户信号源配置
    signalSource, err := database.GetUserSignalSource(userID)
    if err != nil {
        log.Printf("🔍 用户 %s 暂未配置信号源", userID)
        signalSource = &config.SignalSource{CoinPoolURL: "", OITopURL: ""}
    }

    // 5. 处理交易币种列表
    tradingCoins := tm.parseTradingCoins(traderCfg.TradingSymbols, systemConfigs.DefaultCoins)

    // 6. 解析指标配置
    indicatorConfig := tm.parseIndicatorConfig(traderCfg.IndicatorConfig)

    // 7. 构建AutoTraderConfig
    traderConfig := tm.buildTraderConfig(
        traderCfg, aiModelCfg, exchangeCfg, signalSource,
        tradingCoins, indicatorConfig, systemConfigs,
    )

    // 8. 创建AutoTrader实例
    at, err := trader.NewAutoTrader(traderConfig, database, userID)
    if err != nil {
        return fmt.Errorf("创建trader失败: %w", err)
    }

    // 9. 设置自定义prompt
    if traderCfg.CustomPrompt != "" {
        at.SetCustomPrompt(traderCfg.CustomPrompt)
        at.SetOverrideBasePrompt(traderCfg.OverrideBasePrompt)
        log.Printf("✓ 已设置自定义交易策略prompt")
    }

    // 10. 添加到管理器
    tm.traders[traderCfg.ID] = at
    log.Printf("✓ Trader '%s' (%s + %s) 已为用户加载到内存",
        traderCfg.Name, aiModelCfg.Provider, exchangeCfg.ID)

    return nil
}
```

## 竞赛数据获取方法

```go
// 获取竞赛数据（带缓存）
func (tm *TraderManager) GetCompetitionData() (map[string]interface{}, error) {
    // 1. 检查缓存是否有效（30秒内）
    tm.competitionCache.mu.RLock()
    if time.Since(tm.competitionCache.timestamp) < 30*time.Second &&
       len(tm.competitionCache.data) > 0 {
        // 返回缓存数据
        cachedData := make(map[string]interface{})
        for k, v := range tm.competitionCache.data {
            cachedData[k] = v
        }
        tm.competitionCache.mu.RUnlock()
        log.Printf("📋 返回竞赛数据缓存 (缓存时间: %.1fs)",
            time.Since(tm.competitionCache.timestamp).Seconds())
        return cachedData, nil
    }
    tm.competitionCache.mu.RUnlock()

    // 2. 缓存过期，重新获取数据
    tm.mu.RLock()
    allTraders := make([]*trader.AutoTrader, 0, len(tm.traders))
    for _, t := range tm.traders {
        allTraders = append(allTraders, t)
    }
    tm.mu.RUnlock()

    log.Printf("🔄 重新获取竞赛数据，交易员数量: %d", len(allTraders))

    // 3. 并发获取交易员数据
    traders := tm.getConcurrentTraderData(allTraders)

    // 4. 按收益率排序（降序）
    sort.Slice(traders, func(i, j int) bool {
        pnlPctI, okI := traders[i]["total_pnl_pct"].(float64)
        pnlPctJ, okJ := traders[j]["total_pnl_pct"].(float64)
        if !okI { pnlPctI = 0 }
        if !okJ { pnlPctJ = 0 }
        return pnlPctI > pnlPctJ
    })

    // 5. 限制返回前50名
    totalCount := len(traders)
    limit := 50
    if len(traders) > limit {
        traders = traders[:limit]
    }

    // 6. 构建结果数据
    comparison := make(map[string]interface{})
    comparison["traders"] = traders
    comparison["count"] = len(traders)
    comparison["total_count"] = totalCount

    // 7. 更新缓存
    tm.competitionCache.mu.Lock()
    tm.competitionCache.data = comparison
    tm.competitionCache.timestamp = time.Now()
    tm.competitionCache.mu.Unlock()

    return comparison, nil
}
```

## 并发数据获取方法

```go
// 并发获取多个交易员的数据
func (tm *TraderManager) getConcurrentTraderData(traders []*trader.AutoTrader) []map[string]interface{} {
    // 1. 创建结果通道
    resultChan := make(chan traderResult, len(traders))

    // 2. 并发获取每个交易员的数据
    for i, t := range traders {
        go func(index int, trader *trader.AutoTrader) {
            // 创建3秒超时的context
            ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            defer cancel()

            // 创建数据通道
            accountChan := make(chan map[string]interface{}, 1)
            errorChan := make(chan error, 1)

            // 启动goroutine获取账户信息
            go func() {
                account, err := trader.GetAccountInfo()
                if err != nil {
                    errorChan <- err
                } else {
                    accountChan <- account
                }
            }()

            // 获取交易员状态
            status := trader.GetStatus()
            var traderData map[string]interface{}

            // 等待结果或超时
            select {
            case account := <-accountChan:
                // 成功获取账户信息
                traderData = map[string]interface{}{
                    "trader_id":       trader.GetID(),
                    "trader_name":     trader.GetName(),
                    "ai_model":        trader.GetAIModel(),
                    "exchange":        trader.GetExchange(),
                    "total_equity":    account["total_equity"],
                    "total_pnl":       account["total_pnl"],
                    "total_pnl_pct":   account["total_pnl_pct"],
                    "position_count":  account["position_count"],
                    "margin_used_pct": account["margin_used_pct"],
                    "is_running":      status["is_running"],
                }
            case err := <-errorChan:
                // 获取账户信息失败
                log.Printf("⚠️ 获取交易员 %s 账户信息失败: %v", trader.GetID(), err)
                traderData = tm.buildErrorTraderData(trader, status, "账户数据获取失败")
            case <-ctx.Done():
                // 超时
                log.Printf("⏰ 获取交易员 %s 账户信息超时", trader.GetID())
                traderData = tm.buildErrorTraderData(trader, status, "获取超时")
            }

            // 发送结果到通道
            resultChan <- traderResult{index: index, data: traderData}
        }(i, t)
    }

    // 3. 收集所有结果
    results := make([]map[string]interface{}, len(traders))
    for i := 0; i < len(traders); i++ {
        result := <-resultChan
        results[result.index] = result.data
    }

    return results
}

// 构建错误状态下的交易员数据
func (tm *TraderManager) buildErrorTraderData(trader *trader.AutoTrader,
    status map[string]interface{}, errorMsg string) map[string]interface{} {
    return map[string]interface{}{
        "trader_id":       trader.GetID(),
        "trader_name":     trader.GetName(),
        "ai_model":        trader.GetAIModel(),
        "exchange":        trader.GetExchange(),
        "total_equity":    0.0,
        "total_pnl":       0.0,
        "total_pnl_pct":   0.0,
        "position_count":  0,
        "margin_used_pct": 0.0,
        "is_running":      status["is_running"],
        "error":           errorMsg,
    }
}
```

## 配置热重载方法

```go
// 热重载指定交易员的技术指标配置
func (tm *TraderManager) ReloadIndicatorConfig(traderID string, newConfig *market.IndicatorConfig) error {
    // 1. 获取读锁查找交易员
    tm.mu.RLock()
    t, exists := tm.traders[traderID]
    tm.mu.RUnlock()

    // 2. 检查交易员是否存在
    if !exists {
        return fmt.Errorf("trader ID '%s' 不存在", traderID)
    }

    // 3. 调用交易员的热重载方法
    t.ReloadIndicatorConfig(newConfig)
    log.Printf("🔄 TraderManager: 已通知 %s 重载配置", traderID)

    return nil
}
```

## 批量操作方法

```go
// 启动所有交易员
func (tm *TraderManager) StartAll() {
    tm.mu.RLock()
    defer tm.mu.RUnlock()

    log.Println("🚀 启动所有Trader...")

    // 为每个交易员启动独立的goroutine
    for id, t := range tm.traders {
        go func(traderID string, at *trader.AutoTrader) {
            log.Printf("▶️  启动 %s...", at.GetName())
            if err := at.Run(); err != nil {
                log.Printf("❌ %s 运行错误: %v", at.GetName(), err)
            }
        }(id, t)
    }
}

// 停止所有交易员
func (tm *TraderManager) StopAll() {
    tm.mu.RLock()
    defer tm.mu.RUnlock()

    log.Println("⏹  停止所有Trader...")

    // 遍历调用每个交易员的停止方法
    for _, t := range tm.traders {
        t.Stop()
    }
}
```

## 工具方法

```go
// 获取交易员实例
func (tm *TraderManager) GetTrader(id string) (*trader.AutoTrader, error) {
    tm.mu.RLock()
    defer tm.mu.RUnlock()

    t, exists := tm.traders[id]
    if !exists {
        return nil, fmt.Errorf("trader ID '%s' 不存在", id)
    }
    return t, nil
}

// 获取所有交易员实例的副本
func (tm *TraderManager) GetAllTraders() map[string]*trader.AutoTrader {
    tm.mu.RLock()
    defer tm.mu.RUnlock()

    result := make(map[string]*trader.AutoTrader)
    for id, t := range tm.traders {
        result[id] = t
    }
    return result
}

// 从内存中移除交易员
func (tm *TraderManager) RemoveTrader(traderID string) {
    tm.mu.Lock()
    defer tm.mu.Unlock()

    if _, exists := tm.traders[traderID]; exists {
        delete(tm.traders, traderID)
        log.Printf("✓ Trader %s 已从内存中移除", traderID)
    }
}

// 辅助函数：字符串转浮点数，带默认值
func parseFloatOrDefault(str string, defaultValue float64) float64 {
    if val, err := strconv.ParseFloat(str, 64); err == nil {
        return val
    }
    return defaultValue
}

// 辅助函数：字符串转整数，带默认值
func parseIntOrDefault(str string, defaultValue int) int {
    if val, err := strconv.Atoi(str); err == nil {
        return val
    }
    return defaultValue
}
```
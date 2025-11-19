# 回测系统模块设计

## 模块概述

回测系统是NOFX系统的历史策略验证组件，用于模拟交易策略在历史数据上的表现。系统支持基于现有交易员配置进行回测，也可以使用自定义参数进行策略验证。通过模拟真实的市场环境、交易执行和风险管理，为用户提供准确的策略性能评估。

## 核心职责

1. **历史数据管理**: 获取、缓存和管理历史K线数据
2. **时间模拟器**: 控制回测时间进程和决策时点
3. **交易模拟器**: 模拟真实的交易执行，包括滑点、手续费等
4. **持仓管理器**: 跟踪模拟持仓状态和资金变化
5. **性能分析器**: 计算各种性能指标和风险指标
6. **结果生成器**: 生成详细的回测报告和可视化数据

## 模块参数定义

### 回测引擎主结构

```go
type Engine struct {
    config          *Config                    // 回测配置
    db              *config.Database            // 数据库实例
    apiClient       *market.APIClient          // 市场数据API客户端
    mcpClient       *mcp.Client                // AI模型客户端
    timeSimulator   *TimeSimulator             // 时间模拟器
    positionManager *PositionManager           // 持仓管理器
    orderSimulator  *OrderSimulator            // 订单模拟器

    // 回测状态
    backtestID      string                     // 回测会话ID
    equitySnapshots []EquitySnapshot          // 净值快照历史
    trades          []Trade                    // 交易记录历史

    // 数据缓存
    klineCache      map[string][]market.Kline  // K线数据缓存 symbol -> klines

    // 运行状态
    isRunning       bool                       // 是否正在运行
    progress        float64                    // 当前进度
    startTime       time.Time                  // 开始时间
    endTime         time.Time                  // 结束时间
    mu              sync.RWMutex               // 读写锁
}
```

### 回测配置结构

```go
type Config struct {
    // 基础配置
    TraderID             string                     // 关联的交易员ID
    UserID               string                     // 用户ID
    BacktestName         string                     // 回测名称
    StartTime            time.Time                  // 回测起始时间
    EndTime              time.Time                  // 回测结束时间
    InitialBalance       float64                    // 初始资金
    ScanInterval         time.Duration              // 扫描间隔
    TradingSymbols       []string                   // 交易币种列表

    // 交易配置
    Slippage             int                        // 滑点基点(10 = 0.1%)
    CommissionRate       float64                    // 手续费率(默认0.1%)
    UseTraderConfig      bool                       // 是否使用交易员配置
    BTCETHLeverage       int                        // BTC/ETH最大杠杆
    AltcoinLeverage      int                        // 山寨币最大杠杆

    // AI配置
    IndicatorConfig      *market.IndicatorConfig    // 技术指标配置
    CustomPrompt         string                     // 自定义提示词
    OverrideBasePrompt   bool                       // 是否覆盖基础提示词
    SystemPromptTemplate string                     // 系统提示词模板

    // 高级配置
    EnableMarginCall     bool                       // 是否启用强平模拟
    EnableFundingRate    bool                       // 是否考虑资金费用
    DataQuality          string                     // 数据质量要求
    MaxParallelRequests  int                        // 最大并发请求数
}
```

### 时间模拟器结构

```go
type TimeSimulator struct {
    startTime     time.Time     // 开始时间
    endTime       time.Time     // 结束时间
    interval      time.Duration // 扫描间隔
    currentTime   time.Time     // 当前时间
    totalSteps    int           // 总步数
    currentStep   int           // 当前步数
}
```

### 持仓管理器结构

```go
type PositionManager struct {
    initialBalance  float64                   // 初始资金
    currentEquity   float64                   // 当前权益
    availableFunds  float64                   // 可用资金
    positions       map[string]*Position     // 持仓映射 symbol -> position
    trades          []Trade                   // 交易记录
    totalFees       float64                   // 总手续费
    marginUsed      float64                   // 已用保证金
    maxDrawdown     float64                   // 最大回撤
    peakEquity      float64                   // 峰值权益
}
```

### 订单模拟器结构

```go
type OrderSimulator struct {
    slippage           int     // 滑点基点
    commissionRate     float64 // 手续费率
    enableMarketImpact bool    // 是否启用市场冲击
    liquidityData      map[string]*LiquidityData // 流动性数据
}
```

### 回测结果结构

```go
type Result struct {
    // 基础指标
    BacktestID       string                     // 回测ID
    Config           *Config                    // 回测配置
    FinalEquity      float64                    // 最终权益
    TotalPnL         float64                    // 总盈亏
    TotalPnLPct      float64                    // 总盈亏百分比

    // 风险指标
    MaxDrawdown      float64                    // 最大回撤
    MaxDrawdownPct   float64                    // 最大回撤百分比
    SharpeRatio      float64                    // 夏普比率
    SortinoRatio     float64                    // 索提诺比率
    CalmarRatio      float64                    // 卡玛比率
    Volatility       float64                    // 收益波动率

    // 交易指标
    WinRate          float64                    // 胜率
    ProfitFactor     float64                    // 盈利因子
    AvgWin           float64                    // 平均盈利
    AvgLoss          float64                    // 平均亏损
    LargestWin       float64                    // 最大盈利
    LargestLoss      float64                    // 最大亏损
    AvgWinDuration   time.Duration              // 平均盈利持仓时间
    AvgLossDuration  time.Duration              // 平均亏损持仓时间

    // 交易统计
    TotalTrades      int                        // 总交易次数
    WinningTrades    int                        // 盈利交易次数
    LosingTrades     int                        // 亏损交易次数
    LongTrades       int                        // 多头交易次数
    ShortTrades      int                        // 空头交易次数

    // 资金统计
    TotalFees        float64                    // 总手续费
    TotalCommission  float64                    // 总交易佣金
    TotalFunding     float64                    // 总资金费用

    // 时间统计
    ExecutionTime    time.Duration              // 执行耗时
    DataPoints       int                        // 数据点数量
    TradingDays      int                        // 交易天数

    // 详细数据
    EquitySnapshots  []EquitySnapshot          // 净值快照历史
    Trades           []Trade                    // 详细交易记录
    DailyReturns     []DailyReturn             // 日收益率序列
    MonthlyReturns   []MonthlyReturn           // 月收益率序列
}
```

## 数据模型定义

### 净值快照结构

```go
type EquitySnapshot struct {
    Time       time.Time `json:"time"`        // 时间戳
    Equity     float64   `json:"equity"`      // 净值
    PnL        float64   `json:"pnl"`         // 盈亏
    PnLPct     float64   `json:"pnl_pct"`     // 盈亏百分比
    Drawdown   float64   `json:"drawdown"`    // 回撤
    UsedMargin float64   `json:"used_margin"` // 已用保证金
    FreeMargin float64   `json:"free_margin"` // 可用保证金
}
```

### 交易记录结构

```go
type Trade struct {
    // 基础信息
    ID            string    `json:"id"`             // 交易ID
    Symbol        string    `json:"symbol"`         // 交易币种
    Side          string    `json:"side"`           // 交易方向 long/short
    Action        string    `json:"action"`         // 操作类型 open/close

    // 价格信息
    EntryPrice    float64   `json:"entry_price"`    // 开仓价格
    ExitPrice     float64   `json:"exit_price"`     // 平仓价格
    StopLoss      float64   `json:"stop_loss"`      // 止损价格
    TakeProfit   float64   `json:"take_profit"`   // 止盈价格

    // 数量信息
    Quantity      float64   `json:"quantity"`       // 交易数量
    Leverage      int       `json:"leverage"`       // 杠杆倍数
    MarginUsed    float64   `json:"margin_used"`    // 保证金使用

    // 盈亏信息
    PnL           float64   `json:"pnl"`            // 盈亏金额
    PnLPct        float64   `json:"pnl_pct"`        // 盈亏百分比
    UnrealizedPnL float64   `json:"unrealized_pnl"` // 未实现盈亏

    // 费用信息
    Fee           float64   `json:"fee"`            // 交易费用
    Commission    float64   `json:"commission"`     // 交易佣金
    Funding       float64   `json:"funding"`        // 资金费用

    // 时间信息
    EntryTime     time.Time `json:"entry_time"`     // 开仓时间
    ExitTime      time.Time `json:"exit_time"`      // 平仓时间
    Duration      string    `json:"duration"`       // 持仓时长

    // AI决策信息
    DecisionID    string    `json:"decision_id"`    // 决策ID
    Reasoning     string    `json:"reasoning"`      // 决策推理
    Confidence    float64   `json:"confidence"`     // 决策置信度
}
```

### type Position struct {

## 核心方法设计

### 1. 回测引擎生命周期方法

#### NewEngine

```go
func NewEngine(
    backtestID string,
    cfg *Config,
    db *config.Database,
    mcpClient *mcp.Client,
) *Engine
```

- **功能**: 创建新的回测引擎实例
- **参数**:
  - `backtestID`: 回测会话唯一标识
  - `cfg`: 回测配置
  - `db`: 数据库实例
  - `mcpClient`: AI模型客户端
- **返回值**: 回测引擎实例
- **内部逻辑**:
  1. 初始化各个子模块
  2. 设置时间模拟器参数
  3. 创建数据缓存
  4. 验证配置参数

#### Run

```go
func (e *Engine) Run(ctx context.Context) error
```

- **功能**: 执行完整的回测流程
- **参数**:
  - `ctx`: 上下文，支持取消操作
- **返回值**: 错误信息
- **内部逻辑**:
  1. 加载历史数据
  2. 初始化净值记录
  3. 执行主回测循环
  4. 计算最终结果
  5. 保存回测结果

#### Stop

```go
func (e *Engine) Stop() error
```

- **功能**: 停止正在执行的回测
- **返回值**: 错误信息
- **内部逻辑**:
  1. 设置停止标志
  2. 保存当前进度
  3. 清理资源

### 2. 数据管理方法

#### loadHistoricalData

```go
func (e *Engine) loadHistoricalData(ctx context.Context) error
```

- **功能**: 加载历史K线数据
- **参数**:
  - `ctx`: 上下文
- **返回值**: 错误信息
- **内部逻辑**:
  1. 计算时间范围
  2. 并发获取各币种数据
  3. 数据质量验证
  4. 缓存到内存

#### getMarketDataAtTime

```go
func (e *Engine) getMarketDataAtTime(t time.Time) (map[string]*market.Data, error)
```

- **功能**: 获取指定时间的市场数据
- **参数**:
  - `t`: 目标时间
- **返回值**: 市场数据映射和错误信息
- **内部逻辑**:
  1. 从缓存中查找K线数据
  2. 计算技术指标
  3. 构建市场数据结构

### 3. 交易执行方法

#### getDecisions

```go
func (e *Engine) getDecisions(marketDataMap map[string]*market.Data) ([]decision.Decision, error)
```

- **功能**: 获取AI交易决策
- **参数**:
  - `marketDataMap`: 市场数据映射
- **返回值**: 决策列表和错误信息
- **内部逻辑**:
  1. 构建决策上下文
  2. 整合账户和持仓信息
  3. 调用AI决策引擎
  4. 解析决策结果

#### executeDecisions

```go
func (e *Engine) executeDecisions(
    decisions []decision.Decision,
    marketDataMap map[string]*market.Data,
    currentTime time.Time,
) error
```

- **功能**: 执行交易决策
- **参数**:
  - `decisions`: 决策列表
  - `marketDataMap`: 市场数据
  - `currentTime`: 当前时间
- **返回值**: 错误信息
- **内部逻辑**:
  1. 遍历所有决策
  2. 执行开仓或平仓操作
  3. 更新持仓状态
  4. 记录交易历史

#### executeOpenPosition

```go
func (e *Engine) executeOpenPosition(dec decision.Decision, marketPrice float64, currentTime time.Time) error
```

- **功能**: 执行开仓操作
- **参数**:
  - `dec`: 交易决策
  - `marketPrice`: 市场价格
  - `currentTime`: 当前时间
- **返回值**: 错误信息
- **内部逻辑**:
  1. 风险检查和验证
  2. 计算仓位大小
  3. 模拟订单执行
  4. 更新持仓管理器

#### executeClosePosition

```go
func (e *Engine) executeClosePosition(dec decision.Decision, marketPrice float64, currentTime time.Time) error
```

- **功能**: 执行平仓操作
- **参数**:
  - `dec`: 交易决策
  - `marketPrice`: 市场价格
  - `currentTime`: 当前时间
- **返回值**: 错误信息
- **内部逻辑**:
  1. 验证持仓存在
  2. 计算盈亏
  3. 模拟平仓执行
  4. 更新交易记录

### 4. 性能分析方法

#### calculateResult

```go
func (e *Engine) calculateResult() *Result
```

- **功能**: 计算回测结果
- **返回值**: 完整的回测结果
- **内部逻辑**:
  1. 计算基础盈亏指标
  2. 计算风险指标
  3. 计算交易统计
  4. 生成时间序列数据

#### calculatePerformanceMetrics

```go
func (e *Engine) calculatePerformanceMetrics(snapshots []EquitySnapshot) *PerformanceMetrics
```

- **功能**: 计算性能指标
- **参数**:
  - `snapshots`: 净值快照序列
- **返回值**: 性能指标结构
- **内部逻辑**:
  1. 计算收益率序列
  2. 计算波动率和相关性
  3. 计算风险调整收益指标
  4. 计算回撤指标

### 5. 时间模拟方法

#### NewTimeSimulator

```go
func NewTimeSimulator(startTime, endTime time.Time, interval time.Duration) *TimeSimulator
```

- **功能**: 创建时间模拟器
- **参数**:
  - `startTime`: 开始时间
  - `endTime`: 结束时间
  - `interval`: 时间间隔
- **返回值**: 时间模拟器实例

#### Next

```go
func (ts *TimeSimulator) Next() bool
```

- **功能**: 推进到下一个时间点
- **返回值**: 是否还有下一个时间点

#### Progress

```go
func (ts *TimeSimulator) Progress() float64
```

- **功能**: 计算当前进度
- **返回值**: 进程百分比(0-1)

### 6. 订单模拟方法

#### ExecuteMarketOrder

```go
func (os *OrderSimulator) ExecuteMarketOrder(
    side, action string,
    marketPrice, quantity float64,
    leverage int,
) (executionPrice float64, fee float64, err error)
```

- **功能**: 模拟市价单执行
- **参数**:
  - `side`: 交易方向
  - `action`: 操作类型
  - `marketPrice`: 市场价格
  - `quantity`: 交易数量
  - `leverage`: 杠杆倍数
- **返回值**: 执行价格、手续费和错误
- **内部逻辑**:
  1. 计算滑点影响
  2. 计算交易手续费
  3. 模拟市场冲击
  4. 返回执行结果

### 7. 持仓管理方法

#### OpenPosition

```go
func (pm *PositionManager) OpenPosition(symbol, side string, entryPrice, quantity float64, leverage int) error
```

- **功能**: 开仓
- **参数**:
  - `symbol`: 币种
  - `side`: 方向
  - `entryPrice`: 开仓价格
  - `quantity`: 数量
  - `leverage`: 杠杆
- **返回值**: 错误信息

#### ClosePosition

```go
func (pm *PositionManager) ClosePosition(symbol, exitPrice float64) (*Trade, error)
```

- **功能**: 平仓
- **参数**:
  - `symbol`: 币种
  - `exitPrice`: 平仓价格
- **返回值**: 交易记录和错误信息

#### UpdateUnrealizedPnL

```go
func (pm *PositionManager) UpdateUnrealizedPnL(symbol, currentPrice float64)
```

- **功能**: 更新未实现盈亏
- **参数**:
  - `symbol`: 币种
  - `currentPrice`: 当前价格

## 数据库集成

### 回测记录表结构

```sql
CREATE TABLE backtest_runs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    trader_id TEXT,
    name TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    initial_balance REAL NOT NULL,
    final_equity REAL,
    total_pnl REAL,
    total_pnl_pct REAL,
    max_drawdown REAL,
    sharpe_ratio REAL,
    win_rate REAL,
    total_trades INTEGER,
    status TEXT DEFAULT 'pending',
    progress REAL DEFAULT 0,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    config_json TEXT
);
```

### 净值快照表结构

```sql
CREATE TABLE backtest_equity_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    backtest_id TEXT NOT NULL,
    time DATETIME NOT NULL,
    equity REAL NOT NULL,
    pnl REAL NOT NULL,
    pnl_pct REAL NOT NULL,
    drawdown REAL,
    used_margin REAL,
    free_margin REAL,
    FOREIGN KEY (backtest_id) REFERENCES backtest_runs(id)
);
```

### 交易记录表结构

```sql
CREATE TABLE backtest_trades (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    backtest_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    action TEXT NOT NULL,
    entry_price REAL,
    exit_price REAL,
    quantity REAL NOT NULL,
    leverage INTEGER,
    pnl REAL,
    pnl_pct REAL,
    fee REAL,
    commission REAL,
    funding REAL,
    entry_time DATETIME,
    exit_time DATETIME,
    duration TEXT,
    decision_id TEXT,
    reasoning TEXT,
    confidence REAL,
    FOREIGN KEY (backtest_id) REFERENCES backtest_runs(id)
);
```

## API接口设计

### 回测管理端点

#### POST /api/backtest

- **功能**: 创建新的回测任务
- **请求体**: `BacktestConfigRequest`
- **响应**: 回测任务信息

#### GET /api/backtest/:id

- **功能**: 获取回测详情
- **路径参数**: `id` - 回测ID
- **响应**: 完整的回测信息

#### GET /api/backtest/:id/equity-history

- **功能**: 获取净值历史
- **路径参数**: `id` - 回测ID
- **响应**: 净值快照列表

#### GET /api/backtest/:id/trades

- **功能**: 获取交易记录
- **路径参数**: `id` - 回测ID
- **响应**: 交易记录列表

#### GET /api/backtest/:id/performance

- **功能**: 获取性能分析
- **路径参数**: `id` - 回测ID
- **响应**: 详细性能指标

#### DELETE /api/backtest/:id

- **功能**: 删除回测记录
- **路径参数**: `id` - 回测ID
- **响应**: 删除结果

#### GET /api/backtest

- **功能**: 列出回测记录
- **查询参数**:
  - `page` - 页码
  - `page_size` - 每页大小
  - `status` - 状态筛选
- **响应**: 分页的回测列表

## 配置参数说明

### 时间配置

- **时间范围**: 支持任意历史时间段，建议不超过1年
- **扫描间隔**: 支持1分钟到1天的各种间隔
- **数据质量**: 可选择不同质量级别的数据

### 交易配置

- **初始资金**: 最小100USDT，最大无限制
- **滑点设置**: 0-50基点，默认10基点(0.1%)
- **手续费率**: 0-1%，默认0.1%
- **杠杆设置**: 支持1-100倍，按币种分类设置

### 风险配置

- **强平模拟**: 可选择是否启用强平模拟
- **资金费用**: 可选择是否考虑资金费用
- **市场冲击**: 可选择是否启用市场冲击模拟

## 性能优化策略

### 数据加载优化

- 并发获取多币种数据
- 智能缓存机制
- 数据压缩存储
- 增量数据更新

### 计算优化

- 预计算技术指标
- 并行决策处理
- 内存中持仓管理
- 批量数据库操作

### 存储优化

- 分页存储净值快照
- 压缩存储交易记录
- 索引优化查询性能
- 定期清理历史数据

## 错误处理策略

### 数据错误

- **数据缺失**: 跳过该时间点，记录警告
- **数据异常**: 使用插值方法修复
- **网络错误**: 自动重试机制
- **API限流**: 智能退避策略

### 计算错误

- **除零错误**: 安全检查和默认值处理
- **数值溢出**: 使用高精度数值类型
- **逻辑错误**: 异常捕获和恢复机制
- **内存不足**: 分批处理和内存监控

### 业务错误

- **配置错误**: 详细错误提示和修复建议
- **资金不足**: 自动调整仓位大小
- **杠杆超限**: 使用最大可用杠杆
- **交易冲突**: 智能处理和优先级排序

## 扩展性设计

### 多交易所支持

- 统一的数据接口
- 可配置的交易所参数
- 自动数据源切换
- 交叉验证机制

### 多策略支持

- 策略模板系统
- 参数化策略配置
- 策略组合测试
- A/B测试框架

### 高级分析功能

- 蒙特卡洛模拟
- 压力测试
- 情景分析
- 风险价值(VaR)计算

### 可视化支持

- 实时图表生成
- 交互式报告
- 导出多种格式
- 自定义仪表板

## 监控和日志

### 执行监控

- 实时进度跟踪
- 性能指标监控
- 资源使用监控
- 错误率统计

### 日志记录

- 结构化日志格式
- 不同级别日志分类
- 日志轮转和归档
- 敏感信息过滤

### 性能指标

- 执行时间统计
- 内存使用监控
- 数据库操作统计
- API调用统计

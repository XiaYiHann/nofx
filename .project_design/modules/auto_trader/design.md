# 自动交易器 (AutoTrader) 模块设计

## 模块概述

自动交易器是NOFX系统的核心执行组件，负责单个交易策略的完整生命周期管理。它集成了AI决策引擎、风险管理、交易执行和性能监控等功能，是连接AI模型与交易所API的桥梁。

## 核心职责

1. **AI决策执行**: 调用AI模型进行交易决策分析
2. **市场数据分析**: 获取和处理多时间框架的市场数据
3. **风险管理**: 执行仓位控制、杠杆管理和止盈止损
4. **交易执行**: 与交易所API交互执行买卖操作
5. **性能监控**: 统计交易表现和记录决策日志
6. **自我学习**: 基于历史表现优化交易策略

## 模块参数定义

### 核心配置结构

```go
type AutoTraderConfig struct {
    // 基础信息
    ID                    string        // 交易员唯一标识
    Name                  string        // 交易员名称
    UserID                string        // 所属用户ID
    AIModel               string        // AI模型类型

    // 交易所配置
    Exchange              string        // 交易所类型
    BinanceAPIKey         string        // 币安API密钥
    BinanceSecretKey      string        // 币安密钥
    HyperliquidPrivateKey string        // Hyperliquid私钥
    HyperliquidWalletAddr string        // Hyperliquid钱包地址
    HyperliquidTestnet    bool          // 是否使用测试网

    // AI模型配置
    UseQwen               bool          // 是否使用Qwen模型
    UseGLM                bool          // 是否使用GLM模型
    DeepSeekKey           string        // DeepSeek API密钥
    QwenKey               string        // Qwen API密钥
    GLMKey                string        // GLM API密钥
    CustomAPIURL          string        // 自定义API URL
    CustomModelName       string        // 自定义模型名称

    // 交易策略配置
    ScanInterval          time.Duration // 扫描间隔
    InitialBalance        float64       // 初始资金
    BTCETHLeverage        int           // BTC/ETH最大杠杆
    AltcoinLeverage       int           // 山寨币最大杠杆
    MaxDailyLoss          float64       // 最大日损失
    MaxDrawdown           float64       // 最大回撤
    StopTradingTime       time.Duration // 停止交易时间
    IsCrossMargin         bool          // 是否全仓模式

    // 币种配置
    DefaultCoins          []string      // 默认币种列表
    TradingCoins          []string      // 交易币种列表
    CoinPoolAPIURL        string        // 币种池API URL

    // 提示词配置
    SystemPromptTemplate  string        // 系统提示词模板
    CustomPrompt          string        // 自定义提示词
    OverrideBasePrompt    bool          // 是否覆盖基础提示词

    // 技术指标配置
    IndicatorConfig       *market.IndicatorConfig
}
```

### 运行时状态结构

```go
type AutoTrader struct {
    // 配置信息
    config        *AutoTraderConfig
    userID        string
    database      *config.Database

    // 交易接口
    trader        trader.Trader

    // 运行状态
    isRunning     bool
    stopChan      chan struct{}
    lastDecision  time.Time

    // 性能统计
    stats         *PerformanceStats

    // AI客户端
    aiClient      AIModelClient

    // 决策历史
    decisionHistory []TradingDecision

    // 自定义配置
    customPrompt     string
    overrideBasePrompt bool

    // 同步锁
    mu              sync.RWMutex
}
```

### 性能统计结构

```go
type PerformanceStats struct {
    TotalTrades        int     // 总交易次数
    WinningTrades      int     // 盈利交易次数
    LosingTrades       int     // 亏损交易次数
    WinRate            float64 // 胜率
    TotalPnL           float64 // 总盈亏
    TotalPnLPct        float64 // 总盈亏百分比
    AvgProfit          float64 // 平均盈利
    AvgLoss            float64 // 平均亏损
    ProfitFactor       float64 // 盈利因子
    SharpeRatio        float64 // 夏普比率
    MaxDrawdown        float64 // 最大回撤
    DailyPnL           float64 // 当日盈亏
    CurrentEquity      float64 // 当前权益
    MarginUsed         float64 // 已用保证金
    PositionCount      int     // 持仓数量
}
```

## 模块间关系

### 依赖关系
- **依赖** `trader.Trader`: 交易所交易接口实现
- **依赖** `config.Database`: 数据存储和查询
- **依赖** `market.MarketData`: 市场数据获取
- **依赖** `AIModelClient`: AI模型API客户端
- **被依赖** `manager.TraderManager`: 交易员管理器

### 数据流向
1. **市场数据**: MarketData → AutoTrader → AI分析
2. **交易决策**: AI模型 → AutoTrader → 交易所API
3. **交易记录**: 交易所API → AutoTrader → Database
4. **性能统计**: AutoTrader → Database → Web界面

## 核心方法设计

### 1. 生命周期管理方法

#### NewAutoTrader
```go
func NewAutoTrader(config AutoTraderConfig, database *config.Database, userID string) (*AutoTrader, error)
```
- **功能**: 创建新的自动交易器实例
- **参数**:
  - `config`: 交易器配置
  - `database`: 数据库实例
  - `userID`: 用户ID
- **返回值**: 交易器实例和错误信息
- **内部逻辑**:
  1. 验证配置参数
  2. 创建交易所交易器实例
  3. 初始化AI客户端
  4. 设置运行时状态
  5. 初始化性能统计

#### Run
```go
func (at *AutoTrader) Run() error
```
- **功能**: 启动自动交易循环
- **返回值**: 错误信息
- **内部逻辑**:
  1. 设置运行状态为true
  2. 启动主交易循环goroutine
  3. 定期执行交易决策
  4. 监听停止信号

#### Stop
```go
func (at *AutoTrader) Stop()
```
- **功能**: 停止自动交易器
- **内部逻辑**:
  1. 发送停止信号
  2. 设置运行状态为false
  3. 等待当前决策完成
  4. 清理资源

### 2. 交易决策方法

#### MakeDecision
```go
func (at *AutoTrader) MakeDecision() error
```
- **功能**: 执行一次完整的交易决策流程
- **返回值**: 错误信息
- **内部逻辑**:
  1. 获取历史性能反馈
  2. 获取账户状态信息
  3. 分析现有持仓
  4. 评估新的交易机会
  5. 调用AI模型进行决策
  6. 执行交易决策
  7. 记录决策日志

#### AnalyzeMarketData
```go
func (at *AutoTrader) AnalyzeMarketData() (map[string]interface{}, error)
```
- **功能**: 分析市场数据并生成AI输入
- **返回值**: 市场分析数据和错误信息
- **内部逻辑**:
  1. 获取多时间框架K线数据
  2. 计算技术指标
  3. 分析市场情绪
  4. 生成结构化数据

### 3. 交易执行方法

#### ExecuteDecision
```go
func (at *AutoTrader) ExecuteDecision(decision *TradingDecision) error
```
- **功能**: 执行AI生成的交易决策
- **参数**:
  - `decision`: 交易决策对象
- **返回值**: 错误信息
- **内部逻辑**:
  1. 风险检查和验证
  2. 计算交易参数
  3. 执行交易操作
  4. 设置止盈止损
  5. 更新持仓记录

#### RiskManagement
```go
func (at *AutoTrader) RiskManagement(decision *TradingDecision) error
```
- **功能**: 风险管理和控制
- **参数**:
  - `decision`: 待执行的交易决策
- **返回值**: 错误信息
- **内部逻辑**:
  1. 检查仓位大小限制
  2. 验证杠杆倍数
  3. 计算止损止盈价格
  4. 检查资金使用率

### 4. 性能监控方法

#### UpdatePerformanceStats
```go
func (at *AutoTrader) UpdatePerformanceStats()
```
- **功能**: 更新性能统计数据
- **内部逻辑**:
  1. 获取当前账户信息
  2. 计算盈亏统计
  3. 更新胜率和风险指标
  4. 保存到数据库

#### GetAccountInfo
```go
func (at *AutoTrader) GetAccountInfo() (map[string]interface{}, error)
```
- **功能**: 获取账户信息
- **返回值**: 账户信息映射和错误信息

#### GetStatus
```go
func (at *AutoTrader) GetStatus() map[string]interface{}
```
- **功能**: 获取交易器运行状态
- **返回值**: 状态信息映射

### 5. 配置管理方法

#### SetCustomPrompt
```go
func (at *AutoTrader) SetCustomPrompt(prompt string)
```
- **功能**: 设置自定义提示词
- **参数**:
  - `prompt`: 自定义提示词内容

#### ReloadIndicatorConfig
```go
func (at *AutoTrader) ReloadIndicatorConfig(newConfig *market.IndicatorConfig)
```
- **功能**: 热重载技术指标配置
- **参数**:
  - `newConfig`: 新的指标配置

### 6. 日志记录方法

#### LogDecision
```go
func (at *AutoTrader) LogDecision(decision *TradingDecision, marketData map[string]interface{}, aiResponse string) error
```
- **功能**: 记录完整的决策过程
- **参数**:
  - `decision`: 交易决策
  - `marketData`: 市场数据快照
  - `aiResponse`: AI响应内容
- **返回值**: 错误信息

#### SaveDecisionToFile
```go
func (at *AutoTrader) SaveDecisionToFile(decisionLog *DecisionLog) error
```
- **功能**: 保存决策日志到文件
- **参数**:
  - `decisionLog`: 决策日志对象
- **返回值**: 错误信息

## AI决策流程设计

### 1. 历史反馈分析
- 获取最近20次交易记录
- 计算胜率、平均盈亏、夏普比率
- 识别表现最好和最差的币种
- 生成历史性能反馈报告

### 2. 账户状态分析
- 获取当前总权益和可用余额
- 统计持仓数量和未实现盈亏
- 计算保证金使用率
- 监控当日盈亏和回撤情况

### 3. 持仓分析
- 获取所有当前持仓信息
- 计算持仓时间和盈亏状态
- 分析技术指标信号
- 评估是否需要调整或平仓

### 4. 市场机会评估
- 获取候选币种列表
- 批量获取市场数据和技术指标
- 计算波动率和趋势强度
- 筛选潜在交易机会

### 5. AI综合决策
- 整合所有分析数据
- 调用AI模型进行综合分析
- 生成结构化交易决策
- 输出决策推理过程

### 6. 执行和记录
- 验证决策可行性
- 执行交易操作
- 设置风险控制参数
- 记录完整决策过程

## 风险控制策略

### 1. 仓位管理
- 单币种仓位限制：山寨币≤1.5倍权益，BTC/ETH≤10倍权益
- 总仓位限制：保证金使用率≤90%
- 动态仓位调整：根据波动率和账户权益调整

### 2. 杠杆控制
- 最大杠杆限制：根据币种类型设定
- 动态杠杆调整：AI根据市场条件自主选择
- 风险评估：高波动市场降低杠杆

### 3. 止盈止损
- 强制止盈止损比例：≥1:2
- 动态调整：根据技术指标和市场条件
- 时间止损：持仓时间过长自动平仓

### 4. 资金管理
- 最大日损失限制：超过限制停止交易
- 最大回撤控制：达到阈值降低风险
- 分散投资：避免过度集中单一币种

## 性能监控指标

### 1. 盈利能力指标
- 总收益率、年化收益率
- 胜率、盈亏比
- 夏普比率、最大回撤
- 平均持仓时间

### 2. 交易活跃度指标
- 交易频率、平均持仓时间
- 仓位利用率、资金周转率
- 决策执行成功率

### 3. 风险控制指标
- VaR（风险价值）
- 最大连续亏损次数
- 杠杆使用率
- 止损执行率

## 错误处理策略

### 1. API调用错误
- 网络超时：重试机制，降级处理
- 限流错误：退避重试，降低调用频率
- 认证错误：停止交易，记录错误

### 2. 数据获取错误
- 市场数据异常：使用缓存数据，标记异常
- 指标计算错误：使用默认值，记录警告
- 账户信息错误：暂停交易，通知用户

### 3. 交易执行错误
- 余额不足：跳过交易，记录原因
- 市场流动性不足：减少交易数量
- 交易所维护：暂停所有交易操作

## 扩展性设计

### 1. AI模型扩展
- 标准化AI接口，支持多种模型
- 可配置的提示词模板
- 支持自定义AI服务端点

### 2. 交易所扩展
- 统一的交易接口抽象
- 可插拔的交易所适配器
- 支持新交易所快速集成

### 3. 策略扩展
- 模块化的策略组件
- 可配置的技术指标
- 支持自定义交易规则

### 4. 监控扩展
- 可定制的性能指标
- 多种通知方式
- 实时监控仪表板
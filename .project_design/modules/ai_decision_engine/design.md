# AI决策引擎模块设计

## 模块概述

AI决策引擎是NOFX系统的智能核心，负责整合市场数据、历史性能反馈和用户配置，通过调用各种AI模型生成交易决策。它支持多种AI模型（DeepSeek、Qwen、GLM等），提供标准化的决策接口，并实现了完整的提示词管理和响应解析功能。

## 核心职责

1. **AI模型集成**: 统一接口支持多种AI模型
2. **市场数据分析**: 处理多时间框架的市场数据
3. **历史反馈整合**: 分析历史交易表现生成反馈
4. **决策生成**: 调用AI模型生成交易决策
5. **结果解析**: 解析AI响应为结构化决策
6. **提示词管理**: 管理和优化AI提示词模板

## 模块参数定义

### AI决策引擎主结构

```go
type AIDecisionEngine struct {
    config              *AIEngineConfig    // 引擎配置
    modelClients        map[string]AIModelClient // AI模型客户端映射
    promptManager       *PromptManager     // 提示词管理器
    dataProcessor       *DataProcessor     // 数据处理器
    responseParser      *ResponseParser    // 响应解析器
    performanceAnalyzer *PerformanceAnalyzer // 性能分析器
    mu                  sync.RWMutex       // 读写锁
}
```

### 引擎配置结构

```go
type AIEngineConfig struct {
    DefaultModel      string            `json:"default_model"`       // 默认AI模型
    Timeout           time.Duration     `json:"timeout"`             // 请求超时时间
    MaxRetries        int               `json:"max_retries"`         // 最大重试次数
    RetryDelay        time.Duration     `json:"retry_delay"`         // 重试延迟
    EnableCache       bool              `json:"enable_cache"`        // 是否启用缓存
    CacheTTL          time.Duration     `json:"cache_ttl"`           // 缓存有效期
    LogLevel          string            `json:"log_level"`           // 日志级别
    CustomHeaders     map[string]string `json:"custom_headers"`      // 自定义请求头
}
```

### 决策请求结构

```go
type DecisionRequest struct {
    TraderID          string                     `json:"trader_id"`           // 交易员ID
    ModelID           string                     `json:"model_id"`            // AI模型ID
    MarketData        *MarketDataSnapshot        `json:"market_data"`         // 市场数据快照
    AccountInfo       *AccountInfoSnapshot       `json:"account_info"`        // 账户信息快照
    Positions         []*PositionSnapshot        `json:"positions"`           // 当前持仓列表
    HistoricalFeedback *HistoricalFeedback       `json:"historical_feedback"` // 历史性能反馈
    CandidateCoins    []string                   `json:"candidate_coins"`     // 候选币种列表
    SystemPrompt      string                     `json:"system_prompt"`       // 系统提示词
    CustomPrompt      string                     `json:"custom_prompt"`       // 自定义提示词
    OverrideBase      bool                       `json:"override_base"`       // 是否覆盖基础提示词
    IndicatorConfig   *market.IndicatorConfig    `json:"indicator_config"`    // 技术指标配置
    RequestTime       time.Time                  `json:"request_time"`        // 请求时间
}
```

### 决策响应结构

```go
type DecisionResponse struct {
    RequestID       string               `json:"request_id"`        // 请求ID
    ModelID         string               `json:"model_id"`          // AI模型ID
    Decision        *TradingDecision     `json:"decision"`          // 交易决策
    Reasoning       string               `json:"reasoning"`         // 决策推理过程
    Confidence      float64              `json:"confidence"`        // 决策置信度
    MarketAnalysis  *MarketAnalysis      `json:"market_analysis"`   // 市场分析结果
    RiskAssessment  *RiskAssessment      `json:"risk_assessment"`   // 风险评估
    ProcessingTime  time.Duration        `json:"processing_time"`   // 处理耗时
    TokenUsage      *TokenUsage          `json:"token_usage"`       // Token使用统计
    Error           error                `json:"error"`             // 错误信息
    ResponseTime    time.Time            `json:"response_time"`     // 响应时间
}
```

### 交易决策结构

```go
type TradingDecision struct {
    Action           string  `json:"action"`            // 决策动作
    Symbol           string  `json:"symbol"`            // 交易币种
    Quantity         float64 `json:"quantity"`          // 交易数量
    Leverage         int     `json:"leverage"`          // 杠杆倍数
    EntryPrice       float64 `json:"entry_price"`       // 建议入场价格
    StopLossPrice    float64 `json:"stop_loss_price"`   // 建议止损价格
    TakeProfitPrice  float64 `json:"take_profit_price"` // 建议止盈价格
    PositionSide     string  `json:"position_side"`     // 持仓方向
    Reason           string  `json:"reason"`            // 决策原因
    ExpectedDuration string  `json:"expected_duration"` // 预期持仓时间
    RiskRewardRatio  float64 `json:"risk_reward_ratio"` // 风险收益比
}
```

### 市场数据快照结构

```go
type MarketDataSnapshot struct {
    Timestamp           time.Time                    `json:"timestamp"`            // 数据时间戳
    Coins               []*CoinMarketData            `json:"coins"`                // 币种市场数据
    TechnicalIndicators map[string]*IndicatorData    `json:"technical_indicators"` // 技术指标数据
    MarketSentiment     *MarketSentimentData         `json:"market_sentiment"`     // 市场情绪数据
    TimeframeData       map[string][]*CandleData     `json:"timeframe_data"`       // 多时间框架数据
    LiquidityInfo       *LiquidityInfo               `json:"liquidity_info"`       // 流动性信息
}
```

### 历史性能反馈结构

```go
type HistoricalFeedback struct {
    OverallPerformance  *OverallPerformanceStats  `json:"overall_performance"`  // 整体性能统计
    RecentTrades        []*RecentTrade            `json:"recent_trades"`         // 最近交易记录
    CoinPerformance     map[string]*CoinStats     `json:"coin_performance"`      // 各币种表现
    TimeAnalysis        *TimeBasedAnalysis        `json:"time_analysis"`         // 时间分析
    RiskMetrics         *RiskMetrics              `json:"risk_metrics"`          // 风险指标
    LearningInsights    []*LearningInsight        `json:"learning_insights"`     // 学习洞察
}
```

## AI模型接口设计

### AI模型客户端接口

```go
type AIModelClient interface {
    // 基础信息
    GetModelID() string
    GetProvider() string
    GetModelName() string

    // 核心功能
    GenerateDecision(request *DecisionRequest) (*DecisionResponse, error)
    TestConnection() error

    // 配置管理
    UpdateConfig(config map[string]interface{}) error
    GetConfig() map[string]interface{}

    // 状态管理
    IsHealthy() bool
    GetRateLimitInfo() *RateLimitInfo
    GetUsageStats() *UsageStats
}
```

### DeepSeek客户端实现

```go
type DeepSeekClient struct {
    client      *http.Client
    apiKey      string
    baseURL     string
    model       string
    config      *DeepSeekConfig
    rateLimiter *RateLimiter
}

type DeepSeekConfig struct {
    APIKey        string        `json:"api_key"`
    BaseURL       string        `json:"base_url"`
    Model         string        `json:"model"`
    MaxTokens     int           `json:"max_tokens"`
    Temperature   float64       `json:"temperature"`
    TopP          float64       `json:"top_p"`
    Timeout       time.Duration `json:"timeout"`
    MaxRetries    int           `json:"max_retries"`
}
```

### Qwen客户端实现

```go
type QwenClient struct {
    client      *http.Client
    apiKey      string
    baseURL     string
    model       string
    config      *QwenConfig
    rateLimiter *RateLimiter
}

type QwenConfig struct {
    APIKey        string        `json:"api_key"`
    BaseURL       string        `json:"base_url"`
    Model         string        `json:"model"`
    MaxTokens     int           `json:"max_tokens"`
    Temperature   float64       `json:"temperature"`
    TopP          float64       `json:"top_p"`
    Timeout       time.Duration `json:"timeout"`
    MaxRetries    int           `json:"max_retries"`
}
```

### GLM客户端实现

```go
type GLMClient struct {
    client      *http.Client
    apiKey      string
    baseURL     string
    model       string
    config      *GLMConfig
    rateLimiter *RateLimiter
}

type GLMConfig struct {
    APIKey        string        `json:"api_key"`
    BaseURL       string        `json:"base_url"`
    Model         string        `json:"model"`
    MaxTokens     int           `json:"max_tokens"`
    Temperature   float64       `json:"temperature"`
    TopP          float64       `json:"top_p"`
    Timeout       time.Duration `json:"timeout"`
    MaxRetries    int           `json:"max_retries"`
}
```

## 核心方法设计

### 1. 决策引擎生命周期方法

#### NewAIDecisionEngine
```go
func NewAIDecisionEngine(config *AIEngineConfig) *AIDecisionEngine
```
- **功能**: 创建新的AI决策引擎实例
- **参数**:
  - `config`: 引擎配置
- **返回值**: 决策引擎实例
- **内部逻辑**:
  1. 初始化模型客户端映射
  2. 创建提示词管理器
  3. 初始化数据处理器
  4. 创建响应解析器
  5. 设置性能分析器

#### RegisterModel
```go
func (engine *AIDecisionEngine) RegisterModel(client AIModelClient) error
```
- **功能**: 注册AI模型客户端
- **参数**:
  - `client`: AI模型客户端
- **返回值**: 错误信息
- **内部逻辑**:
  1. 验证客户端配置
  2. 测试模型连接
  3. 添加到客户端映射
  4. 更新模型列表

#### GenerateDecision
```go
func (engine *AIDecisionEngine) GenerateDecision(request *DecisionRequest) (*DecisionResponse, error)
```
- **功能**: 生成交易决策
- **参数**:
  - `request`: 决策请求
- **返回值**: 决策响应和错误信息
- **内部逻辑**:
  1. 验证请求参数
  2. 构建完整提示词
  3. 获取AI模型客户端
  4. 调用AI模型生成决策
  5. 解析和验证响应
  6. 记录决策日志

### 2. 提示词管理方法

#### BuildPrompt
```go
func (engine *AIDecisionEngine) BuildPrompt(request *DecisionRequest) (string, error)
```
- **功能**: 构建完整的AI提示词
- **参数**:
  - `request`: 决策请求
- **返回值**: 完整提示词和错误信息
- **内部逻辑**:
  1. 构建系统提示词部分
  2. 添加历史性能反馈
  3. 添加账户状态信息
  4. 添加持仓分析数据
  5. 添加市场数据分析
  6. 添加候选币种信息
  7. 添加决策指令和要求

#### FormatMarketData
```go
func (engine *AIDecisionEngine) FormatMarketData(data *MarketDataSnapshot) string
```
- **功能**: 格式化市场数据为提示词格式
- **参数**:
  - `data`: 市场数据快照
- **返回值**: 格式化的市场数据字符串
- **内部逻辑**:
  1. 格式化价格和K线数据
  2. 格式化技术指标数据
  3. 格式化成交量信息
  4. 格式化市场情绪数据

#### FormatHistoricalFeedback
```go
func (engine *AIDecisionEngine) FormatHistoricalFeedback(feedback *HistoricalFeedback) string
```
- **功能**: 格式化历史性能反馈
- **参数**:
  - `feedback`: 历史性能反馈
- **返回值**: 格式化的反馈字符串
- **内部逻辑**:
  1. 格式化整体性能统计
  2. 格式化最近交易记录
  3. 格式化各币种表现
  4. 格式化风险指标

### 3. 响应解析方法

#### ParseDecisionResponse
```go
func (engine *AIDecisionEngine) ParseDecisionResponse(response string, modelID string) (*DecisionResponse, error)
```
- **功能**: 解析AI模型响应
- **参数**:
  - `response`: AI模型原始响应
  - `modelID`: AI模型ID
- **返回值**: 解析后的决策响应和错误信息
- **内部逻辑**:
  1. 提取JSON决策部分
  2. 解析决策结构
  3. 验证决策参数
  4. 计算置信度
  5. 提取推理过程

#### ValidateDecision
```go
func (engine *AIDecisionEngine) ValidateDecision(decision *TradingDecision) error
```
- **功能**: 验证交易决策的有效性
- **参数**:
  - `decision`: 交易决策
- **返回值**: 验证错误
- **内部逻辑**:
  1. 验证决策动作类型
  2. 验证币种格式
  3. 验证数量和价格
  4. 验证风险收益比
  5. 验证止盈止损价格

### 4. 数据处理方法

#### ProcessMarketData
```go
func (engine *AIDecisionEngine) ProcessMarketData(rawData map[string]interface{}) (*MarketDataSnapshot, error)
```
- **功能**: 处理原始市场数据
- **参数**:
  - `rawData`: 原始市场数据
- **返回值**: 处理后的市场数据快照和错误信息
- **内部逻辑**:
  1. 解析币种价格数据
  2. 计算技术指标
  3. 分析市场情绪
  4. 处理多时间框架数据
  5. 计算流动性指标

#### AnalyzeHistoricalPerformance
```go
func (engine *AIDecisionEngine) AnalyzeHistoricalPerformance(traderID string, limit int) (*HistoricalFeedback, error)
```
- **功能**: 分析历史交易性能
- **参数**:
  - `traderID`: 交易员ID
  - `limit`: 分析的记录数量
- **返回值**: 历史性能反馈和错误信息
- **内部逻辑**:
  1. 获取历史交易记录
  2. 计算整体性能统计
  3. 分析各币种表现
  4. 识别成功和失败模式
  5. 生成学习洞察

### 5. 性能分析方法

#### AnalyzeDecisionQuality
```go
func (engine *AIDecisionEngine) AnalyzeDecisionQuality(decision *TradingDecision, outcome *TradeOutcome) *DecisionQuality
```
- **功能**: 分析决策质量
- **参数**:
  - `decision`: 原始决策
  - `outcome`: 交易结果
- **返回值**: 决策质量分析
- **内部逻辑**:
  1. 计算决策准确性
  2. 分析风险控制效果
  3. 评估时机选择
  4. 计算收益表现

#### UpdateModelPerformance
```go
func (engine *AIDecisionEngine) UpdateModelPerformance(modelID string, metrics *PerformanceMetrics) error
```
- **功能**: 更新AI模型性能统计
- **参数**:
  - `modelID`: 模型ID
  - `metrics`: 性能指标
- **返回值**: 错误信息
- **内部逻辑**:
  1. 更新成功率统计
  2. 更新响应时间统计
  3. 更新错误率统计
  4. 更新成本统计

### 6. 缓存管理方法

#### GetCachedDecision
```go
func (engine *AIDecisionEngine) GetCachedDecision(cacheKey string) (*DecisionResponse, bool)
```
- **功能**: 获取缓存的决策
- **参数**:
  - `cacheKey`: 缓存键
- **返回值**: 缓存的决策和是否命中
- **内部逻辑**:
  1. 计算缓存键
  2. 检查缓存是否存在
  3. 验证缓存是否过期
  4. 返回缓存结果

#### CacheDecision
```go
func (engine *AIDecisionEngine) CacheDecision(cacheKey string, response *DecisionResponse) error
```
- **功能**: 缓存决策结果
- **参数**:
  - `cacheKey`: 缓存键
  - `response`: 决策响应
- **返回值**: 错误信息
- **内部逻辑**:
  1. 验证缓存大小限制
  2. 存储到缓存
  3. 设置过期时间
  4. 更新缓存索引

## 提示词模板设计

### 1. 系统提示词模板

```markdown
你是一个专业的加密货币交易AI，具有丰富的市场分析经验和风险管理能力。

## 核心能力
- 深度技术分析（EMA、MACD、RSI、ATR、布林带等）
- 市场情绪和资金流向分析
- 风险收益比计算和仓位管理
- 多时间框架趋势判断

## 交易原则
1. **风险第一**: 永远把风险控制放在首位
2. **纪律性**: 严格执行止盈止损策略
3. **数据驱动**: 基于客观数据做决策，避免情绪化交易
4. **持续学习**: 从历史交易中学习和改进

## 决策要求
- 提供明确的决策逻辑和推理过程
- 确保风险收益比至少为1:2
- 考虑市场流动性和交易成本
- 避免过度交易和频繁调仓
```

### 2. 决策指令模板

```markdown
## 当前任务
基于以下信息，做出交易决策：

### 决策选项
- **wait**: 观望等待，不进行任何交易
- **hold**: 持仓观望，保持现有仓位
- **close_long**: 平掉多头仓位
- **close_short**: 平掉空头仓位
- **open_long**: 开多头仓位
- **open_short**: 开空头仓位

### 输出格式要求
请严格按照以下JSON格式输出决策：

```json
{
  "decision": {
    "action": "wait/hold/open_long/open_short/close_long/close_short",
    "symbol": "BTCUSDT",
    "quantity": 0.1,
    "leverage": 5,
    "entry_price": 45000.0,
    "stop_loss_price": 44000.0,
    "take_profit_price": 47000.0,
    "position_side": "long",
    "reason": "详细解释决策原因",
    "expected_duration": "2-4小时",
    "risk_reward_ratio": 2.0
  },
  "reasoning": "详细的决策推理过程",
  "confidence": 0.85,
  "market_analysis": {
    "trend": "bullish/bearish/neutral",
    "support_level": 44000.0,
    "resistance_level": 47000.0,
    "key_factors": ["因素1", "因素2"]
  },
  "risk_assessment": {
    "risk_level": "low/medium/high",
    "max_loss_percent": 2.0,
    "probability_of_success": 0.7
  }
}
```
```

### 3. 动态提示词构建

```go
// 提示词构建器
type PromptBuilder struct {
    sections []string
    variables map[string]interface{}
}

func (pb *PromptBuilder) AddSection(section string) *PromptBuilder
func (pb *PromptBuilder) AddVariable(key string, value interface{}) *PromptBuilder
func (pb *PromptBuilder) Build() string
```

## 错误处理策略

### 1. AI模型调用错误
- **网络超时**: 自动重试，使用指数退避
- **API限流**: 排队等待，降低调用频率
- **认证失败**: 立即停止，通知管理员
- **模型不可用**: 切换到备用模型

### 2. 响应解析错误
- **JSON格式错误**: 尝试修复格式，使用默认值
- **字段缺失**: 使用合理的默认值
- **数值范围错误**: 限制在合理范围内
- **逻辑冲突**: 记录警告，跳过该决策

### 3. 数据处理错误
- **市场数据缺失**: 使用历史数据补充
- **指标计算错误**: 使用备用计算方法
- **性能分析失败**: 简化分析维度
- **缓存错误**: 直接调用AI模型

## 性能优化

### 1. 响应时间优化
- 并发数据处理
- 预计算常用指标
- 异步AI模型调用
- 响应缓存机制

### 2. 资源使用优化
- 连接池管理
- 内存使用监控
- 定期清理缓存
- 垃圾回收优化

### 3. 成本控制
- Token使用量优化
- 请求合并处理
- 智能缓存策略
- 成本监控和报警

## 扩展性设计

### 1. 新AI模型支持
- 标准化客户端接口
- 插件式模型注册
- 配置驱动的集成
- 自动化测试框架

### 2. 自定义提示词
- 模板化提示词系统
- 变量替换机制
- 多语言支持
- 版本控制管理

### 3. 高级分析功能
- 机器学习模型集成
- 实时数据流处理
- 复杂事件检测
- 预测分析能力

## 监控和日志

### 1. 性能监控
- AI模型响应时间
- 决策准确率统计
- 错误率监控
- 成本使用追踪

### 2. 质量监控
- 决策一致性检查
- 风险控制合规性
- 逻辑冲突检测
- 异常模式识别

### 3. 业务监控
- 交易执行成功率
- 收益表现跟踪
- 用户满意度调研
- 系统可用性监控
# 数据库模块设计

## 模块概述

数据库模块是NOFX系统的数据持久化层，负责所有业务数据的存储、查询、更新和管理。采用SQLite作为数据库引擎，提供轻量级、高性能的数据存储解决方案。模块集成了加密服务来保护敏感信息，并提供了完整的数据库初始化、迁移和管理功能。

## 核心职责

1. **数据持久化**: 所有业务数据的可靠存储
2. **查询服务**: 高效的数据检索和聚合查询
3. **事务管理**: 保证数据一致性和完整性
4. **安全加密**: 敏感数据的加密存储和解密
5. **数据迁移**: 数据库结构升级和数据迁移
6. **备份恢复**: 数据备份和恢复机制

## 模块参数定义

### 数据库主结构

```go
type Database struct {
    db            *sql.DB              // SQLite数据库连接
    cryptoService *crypto.CryptoService // 加密服务实例
    mu            sync.RWMutex         // 读写锁，保护并发访问
    path          string               // 数据库文件路径
}
```

### 数据记录结构

```go
// 用户记录
type UserRecord struct {
    ID           string    `json:"id"`           // 用户唯一标识
    Email        string    `json:"email"`        // 用户邮箱
    PasswordHash string    `json:"password_hash"` // 密码哈希
    IsActive     bool      `json:"is_active"`     // 是否激活
    IsAdmin      bool      `json:"is_admin"`      // 是否管理员
    CreatedAt    time.Time `json:"created_at"`    // 创建时间
    UpdatedAt    time.Time `json:"updated_at"`    // 更新时间
}

// AI模型配置记录
type AIModelConfig struct {
    ID              string            `json:"id"`               // 模型唯一标识
    UserID          string            `json:"user_id"`          // 所属用户ID
    Provider        string            `json:"provider"`         // 供应商(deepseek/qwen/glm/custom)
    ModelName       string            `json:"model_name"`       // 模型名称
    APIKey          string            `json:"api_key"`          // API密钥(加密存储)
    CustomAPIURL    string            `json:"custom_api_url"`   // 自定义API URL
    CustomModelName string            `json:"custom_model_name"`// 自定义模型名称
    Enabled         bool              `json:"enabled"`          // 是否启用
    Config          map[string]interface{} `json:"config"`     // 模型配置参数
    CreatedAt       time.Time         `json:"created_at"`      // 创建时间
    UpdatedAt       time.Time         `json:"updated_at"`      // 更新时间
}

// 交易所配置记录
type ExchangeConfig struct {
    ID                      string    `json:"id"`                        // 交易所唯一标识
    UserID                  string    `json:"user_id"`                  // 所属用户ID
    ExchangeType            string    `json:"exchange_type"`            // 交易所类型
    APIKey                  string    `json:"api_key"`                  // API密钥(加密存储)
    SecretKey               string    `json:"secret_key"`               // 密钥(加密存储)
    HyperliquidWalletAddr   string    `json:"hyperliquid_wallet_addr"`   // Hyperliquid钱包地址
    AsterUser               string    `json:"aster_user"`               // Aster用户地址
    AsterSigner             string    `json:"aster_signer"`             // Aster签名地址
    Testnet                 bool      `json:"testnet"`                  // 是否测试网
    Enabled                 bool      `json:"enabled"`                  // 是否启用
    Config                  map[string]interface{} `json:"config"`     // 交易所配置参数
    CreatedAt               time.Time `json:"created_at"`               // 创建时间
    UpdatedAt               time.Time `json:"updated_at"`               // 更新时间
}

// 交易员配置记录
type TraderRecord struct {
    ID                    string    `json:"id"`                      // 交易员唯一标识
    UserID                string    `json:"user_id"`                 // 所属用户ID
    Name                  string    `json:"name"`                    // 交易员名称
    AIModelID             string    `json:"ai_model_id"`             // AI模型ID
    ExchangeID            string    `json:"exchange_id"`             // 交易所配置ID
    InitialBalance        float64   `json:"initial_balance"`         // 初始资金
    ScanIntervalMinutes   int       `json:"scan_interval_minutes"`    // 扫描间隔(分钟)
    BTCETHLeverage        int       `json:"btc_eth_leverage"`         // BTC/ETH最大杠杆
    AltcoinLeverage       int       `json:"altcoin_leverage"`        // 山寨币最大杠杆
    IsCrossMargin         bool      `json:"is_cross_margin"`         // 是否全仓模式
    IsRunning             bool      `json:"is_running"`              // 是否运行中
    TradingSymbols        string    `json:"trading_symbols"`         // 交易币种列表(逗号分隔)
    UseCoinPool           bool      `json:"use_coin_pool"`           // 是否使用币种池
    CustomPrompt          string    `json:"custom_prompt"`           // 自定义提示词
    OverrideBasePrompt    bool      `json:"override_base_prompt"`     // 是否覆盖基础提示词
    SystemPromptTemplate  string    `json:"system_prompt_template"`   // 系统提示词模板
    IndicatorConfig       string    `json:"indicator_config"`        // 技术指标配置(JSON)
    CreatedAt             time.Time `json:"created_at"`              // 创建时间
    UpdatedAt             time.Time `json:"updated_at"`              // 更新时间
}

// 交易决策记录
type TradingDecision struct {
    ID             string    `json:"id"`              // 决策记录唯一标识
    TraderID       string    `json:"trader_id"`       // 交易员ID
    DecisionType   string    `json:"decision_type"`   // 决策类型
    Symbol         string    `json:"symbol"`          // 交易币种
    Quantity       float64   `json:"quantity"`        // 数量
    Leverage       int       `json:"leverage"`        // 杠杆倍数
    EntryPrice     float64   `json:"entry_price"`     // 入场价格
    StopLossPrice  float64   `json:"stop_loss_price"` // 止损价格
    TakeProfitPrice float64  `json:"take_profit_price"` // 止盈价格
    Reasoning      string    `json:"reasoning"`       // 决策推理过程
    MarketData     string    `json:"market_data"`     // 市场数据快照
    AIResponse     string    `json:"ai_response"`     // AI响应内容
    Status         string    `json:"status"`          // 执行状态
    ErrorMessage   string    `json:"error_message"`   // 错误信息
    CreatedAt      time.Time `json:"created_at"`      // 创建时间
    ExecutedAt     time.Time `json:"executed_at"`     // 执行时间
}

// 持仓记录
type PositionRecord struct {
    ID            string    `json:"id"`             // 持仓记录唯一标识
    TraderID      string    `json:"trader_id"`      // 交易员ID
    Symbol        string    `json:"symbol"`         // 交易币种
    PositionSide  string    `json:"position_side"`  // 持仓方向
    Quantity      float64   `json:"quantity"`       // 持仓数量
    EntryPrice    float64   `json:"entry_price"`    // 开仓价格
    CurrentPrice  float64   `json:"current_price"`  // 当前价格
    UnrealizedPnL float64   `json:"unrealized_pnl"` // 未实现盈亏
    RealizedPnL   float64   `json:"realized_pnl"`   // 已实现盈亏
    MarginUsed    float64   `json:"margin_used"`    // 已用保证金
    Leverage      int       `json:"leverage"`       // 杠杆倍数
    OpenedAt      time.Time `json:"opened_at"`      // 开仓时间
    ClosedAt      time.Time `json:"closed_at"`      // 平仓时间
    Status        string    `json:"status"`         // 持仓状态
}

// 性能统计记录
type PerformanceStats struct {
    ID            string    `json:"id"`             // 性能统计唯一标识
    TraderID      string    `json:"trader_id"`      // 交易员ID
    StatDate      time.Time `json:"stat_date"`      // 统计日期
    TotalEquity   float64   `json:"total_equity"`   // 总权益
    TotalPnL      float64   `json:"total_pnl"`      // 总盈亏
    TotalPnLPct   float64   `json:"total_pnl_pct"`  // 总盈亏百分比
    TotalTrades   int       `json:"total_trades"`   // 总交易次数
    WinningTrades int       `json:"winning_trades"` // 盈利交易次数
    LosingTrades  int       `json:"losing_trades"`  // 亏损交易次数
    WinRate       float64   `json:"win_rate"`      // 胜率
    AvgProfit     float64   `json:"avg_profit"`    // 平均盈利
    AvgLoss       float64   `json:"avg_loss"`      // 平均亏损
    ProfitFactor  float64   `json:"profit_factor"` // 盈利因子
    SharpeRatio   float64   `json:"sharpe_ratio"`  // 夏普比率
    MaxDrawdown   float64   `json:"max_drawdown"`  // 最大回撤
    DailyReturn   float64   `json:"daily_return"`  // 日收益率
    CreatedAt     time.Time `json:"created_at"`     // 创建时间
    UpdatedAt     time.Time `json:"updated_at"`     // 更新时间
}

// 信号源配置记录
type SignalSource struct {
    ID           string            `json:"id"`           // 信号源唯一标识
    UserID       string            `json:"user_id"`      // 所属用户ID
    Name         string            `json:"name"`         // 信号源名称
    CoinPoolURL  string            `json:"coin_pool_url"` // 币种池API URL
    OITopURL     string            `json:"oi_top_url"`   // 持仓量TOP API URL
    Enabled      bool              `json:"enabled"`      // 是否启用
    Config       map[string]interface{} `json:"config"` // 信号源配置
    CreatedAt    time.Time         `json:"created_at"`   // 创建时间
    UpdatedAt    time.Time         `json:"updated_at"`   // 更新时间
}
```

## 数据库表结构

### 1. 用户表 (users)
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_admin BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2. AI模型配置表 (ai_models)
```sql
CREATE TABLE ai_models (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    api_key TEXT NOT NULL, -- 加密存储
    custom_api_url TEXT,
    custom_model_name TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    config TEXT, -- JSON格式
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### 3. 交易所配置表 (exchanges)
```sql
CREATE TABLE exchanges (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    exchange_type TEXT NOT NULL,
    api_key TEXT NOT NULL, -- 加密存储
    secret_key TEXT, -- 加密存储
    hyperliquid_wallet_addr TEXT,
    aster_user TEXT,
    aster_signer TEXT,
    testnet BOOLEAN DEFAULT FALSE,
    enabled BOOLEAN DEFAULT TRUE,
    config TEXT, -- JSON格式
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### 4. 交易员配置表 (traders)
```sql
CREATE TABLE traders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    ai_model_id TEXT NOT NULL,
    exchange_id TEXT NOT NULL,
    initial_balance REAL DEFAULT 1000.0,
    scan_interval_minutes INTEGER DEFAULT 3,
    btc_eth_leverage INTEGER DEFAULT 5,
    altcoin_leverage INTEGER DEFAULT 5,
    is_cross_margin BOOLEAN DEFAULT FALSE,
    is_running BOOLEAN DEFAULT FALSE,
    trading_symbols TEXT, -- 逗号分隔
    use_coin_pool BOOLEAN DEFAULT FALSE,
    custom_prompt TEXT,
    override_base_prompt BOOLEAN DEFAULT FALSE,
    system_prompt_template TEXT,
    indicator_config TEXT, -- JSON格式
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (ai_model_id) REFERENCES ai_models(id),
    FOREIGN KEY (exchange_id) REFERENCES exchanges(id)
);
```

### 5. 交易决策表 (trading_decisions)
```sql
CREATE TABLE trading_decisions (
    id TEXT PRIMARY KEY,
    trader_id TEXT NOT NULL,
    decision_type TEXT NOT NULL,
    symbol TEXT NOT NULL,
    quantity REAL,
    leverage INTEGER,
    entry_price REAL,
    stop_loss_price REAL,
    take_profit_price REAL,
    reasoning TEXT,
    market_data TEXT, -- JSON格式
    ai_response TEXT,
    status TEXT DEFAULT 'pending',
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    executed_at DATETIME,
    FOREIGN KEY (trader_id) REFERENCES traders(id)
);
```

### 6. 持仓记录表 (positions)
```sql
CREATE TABLE positions (
    id TEXT PRIMARY KEY,
    trader_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    position_side TEXT NOT NULL,
    quantity REAL NOT NULL,
    entry_price REAL NOT NULL,
    current_price REAL,
    unrealized_pnl REAL DEFAULT 0.0,
    realized_pnl REAL DEFAULT 0.0,
    margin_used REAL DEFAULT 0.0,
    leverage INTEGER NOT NULL,
    opened_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    closed_at DATETIME,
    status TEXT DEFAULT 'open',
    FOREIGN KEY (trader_id) REFERENCES traders(id)
);
```

### 7. 性能统计表 (performance_stats)
```sql
CREATE TABLE performance_stats (
    id TEXT PRIMARY KEY,
    trader_id TEXT NOT NULL,
    stat_date DATE NOT NULL,
    total_equity REAL DEFAULT 0.0,
    total_pnl REAL DEFAULT 0.0,
    total_pnl_pct REAL DEFAULT 0.0,
    total_trades INTEGER DEFAULT 0,
    winning_trades INTEGER DEFAULT 0,
    losing_trades INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0.0,
    avg_profit REAL DEFAULT 0.0,
    avg_loss REAL DEFAULT 0.0,
    profit_factor REAL DEFAULT 0.0,
    sharpe_ratio REAL DEFAULT 0.0,
    max_drawdown REAL DEFAULT 0.0,
    daily_return REAL DEFAULT 0.0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trader_id) REFERENCES traders(id),
    UNIQUE(trader_id, stat_date)
);
```

### 8. 系统配置表 (system_configs)
```sql
CREATE TABLE system_configs (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    config_type TEXT DEFAULT 'system',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 9. 信号源配置表 (signal_sources)
```sql
CREATE TABLE signal_sources (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    coin_pool_url TEXT,
    oi_top_url TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    config TEXT, -- JSON格式
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### 10. 内测码表 (beta_codes)
```sql
CREATE TABLE beta_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT UNIQUE NOT NULL,
    used_by TEXT, -- 使用该内测码的用户邮箱
    used_at DATETIME,
    is_used BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## 核心方法设计

### 1. 数据库初始化方法

#### NewDatabase
```go
func NewDatabase(dbPath string) (*Database, error)
```
- **功能**: 创建新的数据库实例
- **参数**:
  - `dbPath`: 数据库文件路径
- **返回值**: 数据库实例和错误信息
- **内部逻辑**:
  1. 打开SQLite数据库连接
  2. 设置连接池参数
  3. 执行数据库初始化
  4. 创建必要的表结构
  5. 创建索引优化查询性能

#### InitializeDatabase
```go
func (db *Database) InitializeDatabase() error
```
- **功能**: 初始化数据库结构和基础数据
- **内部逻辑**:
  1. 创建所有数据表
  2. 创建索引
  3. 插入默认系统配置
  4. 创建默认管理员用户（如果需要）

### 2. 用户管理方法

#### CreateUser
```go
func (db *Database) CreateUser(user *UserRecord) error
```
- **功能**: 创建新用户
- **参数**:
  - `user`: 用户记录
- **返回值**: 错误信息

#### GetUserByEmail
```go
func (db *Database) GetUserByEmail(email string) (*UserRecord, error)
```
- **功能**: 根据邮箱获取用户信息
- **参数**:
  - `email`: 用户邮箱
- **返回值**: 用户记录和错误信息

#### GetAllUsers
```go
func (db *Database) GetAllUsers() ([]string, error)
```
- **功能**: 获取所有用户ID列表
- **返回值**: 用户ID列表和错误信息

### 3. AI模型配置方法

#### CreateAIModel
```go
func (db *Database) CreateAIModel(model *AIModelConfig) error
```
- **功能**: 创建AI模型配置
- **参数**:
  - `model`: AI模型配置
- **返回值**: 错误信息
- **特性**: 自动加密API密钥

#### GetAIModels
```go
func (db *Database) GetAIModels(userID string) ([]*AIModelConfig, error)
```
- **功能**: 获取用户的AI模型配置列表
- **参数**:
  - `userID`: 用户ID
- **返回值**: AI模型配置列表和错误信息
- **特性**: 自动解密API密钥

#### UpdateAIModels
```go
func (db *Database) UpdateAIModels(models []*AIModelConfig) error
```
- **功能**: 批量更新AI模型配置
- **参数**:
  - `models`: AI模型配置列表
- **返回值**: 错误信息
- **特性**: 支持事务处理

### 4. 交易所配置方法

#### CreateExchange
```go
func (db *Database) CreateExchange(exchange *ExchangeConfig) error
```
- **功能**: 创建交易所配置
- **参数**:
  - `exchange`: 交易所配置
- **返回值**: 错误信息
- **特性**: 自动加密敏感信息

#### GetExchanges
```go
func (db *Database) GetExchanges(userID string) ([]*ExchangeConfig, error)
```
- **功能**: 获取用户的交易所配置列表
- **参数**:
  - `userID`: 用户ID
- **返回值**: 交易所配置列表和错误信息

#### TestExchangeConnection
```go
func (db *Database) TestExchangeConnection(exchange *ExchangeConfig) error
```
- **功能**: 测试交易所连接
- **参数**:
  - `exchange`: 交易所配置
- **返回值**: 错误信息

### 5. 交易员管理方法

#### CreateTrader
```go
func (db *Database) CreateTrader(trader *TraderRecord) error
```
- **功能**: 创建交易员配置
- **参数**:
  - `trader`: 交易员配置
- **返回值**: 错误信息

#### GetTraders
```go
func (db *Database) GetTraders(userID string) ([]*TraderRecord, error)
```
- **功能**: 获取用户的交易员列表
- **参数**:
  - `userID`: 用户ID
- **返回值**: 交易员列表和错误信息

#### UpdateTrader
```go
func (db *Database) UpdateTrader(trader *TraderRecord) error
```
- **功能**: 更新交易员配置
- **参数**:
  - `trader`: 交易员配置
- **返回值**: 错误信息

#### DeleteTrader
```go
func (db *Database) DeleteTrader(traderID string) error
```
- **功能**: 删除交易员配置
- **参数**:
  - `traderID`: 交易员ID
- **返回值**: 错误信息

### 6. 交易记录方法

#### CreateTradingDecision
```go
func (db *Database) CreateTradingDecision(decision *TradingDecision) error
```
- **功能**: 创建交易决策记录
- **参数**:
  - `decision`: 交易决策
- **返回值**: 错误信息

#### GetLatestDecisions
```go
func (db *Database) GetLatestDecisions(traderID string, limit int) ([]*TradingDecision, error)
```
- **功能**: 获取最新的交易决策记录
- **参数**:
  - `traderID`: 交易员ID
  - `limit`: 记录数量限制
- **返回值**: 决策记录列表和错误信息

### 7. 性能统计方法

#### UpdatePerformanceStats
```go
func (db *Database) UpdatePerformanceStats(traderID string, stats *PerformanceStats) error
```
- **功能**: 更新性能统计数据
- **参数**:
  - `traderID`: 交易员ID
  - `stats`: 性能统计数据
- **返回值**: 错误信息
- **特性**: 使用UPSERT操作

#### GetPerformanceHistory
```go
func (db *Database) GetPerformanceHistory(traderID string, days int) ([]*PerformanceStats, error)
```
- **功能**: 获取性能历史数据
- **参数**:
  - `traderID`: 交易员ID
  - `days`: 天数
- **返回值**: 性能历史列表和错误信息

### 8. 系统配置方法

#### GetSystemConfig
```go
func (db *Database) GetSystemConfig(key string) (string, error)
```
- **功能**: 获取系统配置
- **参数**:
  - `key`: 配置键名
- **返回值**: 配置值和错误信息

#### SetSystemConfig
```go
func (db *Database) SetSystemConfig(key, value string) error
```
- **功能**: 设置系统配置
- **参数**:
  - `key`: 配置键名
  - `value`: 配置值
- **返回值**: 错误信息

### 9. 信号源管理方法

#### GetSignalSource
```go
func (db *Database) GetSignalSource(userID string) (*SignalSource, error)
```
- **功能**: 获取用户信号源配置
- **参数**:
  - `userID`: 用户ID
- **返回值**: 信号源配置和错误信息

#### UpdateSignalSource
```go
func (db *Database) UpdateSignalSource(signalSource *SignalSource) error
```
- **功能**: 更新信号源配置
- **参数**:
  - `signalSource`: 信号源配置
- **返回值**: 错误信息

### 10. 内测码管理方法

#### LoadBetaCodesFromFile
```go
func (db *Database) LoadBetaCodesFromFile(filename string) error
```
- **功能**: 从文件加载内测码
- **参数**:
  - `filename`: 文件路径
- **返回值**: 错误信息

#### ValidateBetaCode
```go
func (db *Database) ValidateBetaCode(code, email string) error
```
- **功能**: 验证并使用内测码
- **参数**:
  - `code`: 内测码
  - `email`: 用户邮箱
- **返回值**: 错误信息

#### GetBetaCodeStats
```go
func (db *Database) GetBetaCodeStats() (total, used int, err error)
```
- **功能**: 获取内测码统计信息
- **返回值**: 总数、已使用数和错误信息

## 加密服务集成

### 1. 数据加密
```go
// 加密敏感数据
func (db *Database) encryptSensitiveData(data string) (string, error)

// 解密敏感数据
func (db *Database) decryptSensitiveData(encryptedData string) (string, error)
```

### 2. 批量加密处理
- API密钥自动加密存储
- 读取时自动解密
- 支持密钥轮换

## 数据库迁移

### 1. 版本管理
```go
type Migration struct {
    Version     string // 迁移版本
    Description string // 迁移描述
    Up          func(*Database) error // 升级函数
    Down        func(*Database) error // 降级函数
}
```

### 2. 自动迁移
- 启动时检查数据库版本
- 自动执行必要的迁移
- 支持回滚操作

## 性能优化

### 1. 索引策略
```sql
-- 用户相关索引
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(is_active);

-- 交易员相关索引
CREATE INDEX idx_traders_user_id ON traders(user_id);
CREATE INDEX idx_traders_running ON traders(is_running);
CREATE INDEX idx_traders_ai_model ON traders(ai_model_id);

-- 决策记录索引
CREATE INDEX idx_decisions_trader_id ON trading_decisions(trader_id);
CREATE INDEX idx_decisions_created_at ON trading_decisions(created_at);

-- 性能统计索引
CREATE INDEX idx_performance_trader_date ON performance_stats(trader_id, stat_date);
```

### 2. 查询优化
- 使用预编译语句
- 批量操作优化
- 连接池管理
- 定期VACUUM操作

### 3. 缓存策略
- 系统配置缓存
- 用户配置缓存
- 热点数据预加载

## 数据安全

### 1. 访问控制
- 用户数据隔离
- 权限验证
- SQL注入防护

### 2. 数据完整性
- 外键约束
- 事务处理
- 数据验证

### 3. 备份恢复
- 定期自动备份
- 增量备份支持
- 恢复验证机制

## 错误处理

### 1. 数据库错误
- 连接失败处理
- 锁超时处理
- 磁盘空间不足处理

### 2. 数据错误
- 约束违反处理
- 数据类型错误处理
- 空值处理

### 3. 并发错误
- 死锁检测和处理
- 乐观锁实现
- 重试机制

## 监控和维护

### 1. 性能监控
- 查询性能统计
- 连接池状态监控
- 数据库大小监控

### 2. 健康检查
- 连接状态检查
- 表结构完整性检查
- 索引有效性检查

### 3. 日志记录
- 慢查询日志
- 错误日志
- 操作审计日志
# 交易员管理器 (TraderManager) 模块设计

## 模块概述

交易员管理器是NOFX系统的核心协调组件，负责管理多个自动交易器实例的生命周期。它提供了统一的接口来创建、启动、停止和监控交易员，同时处理并发数据获取和缓存机制。

## 核心职责

1. **交易员生命周期管理**: 创建、启动、停止、销毁交易员实例
2. **配置管理**: 从数据库加载交易员配置并构建交易器实例
3. **并发控制**: 安全地管理多个交易员的并发操作
4. **数据聚合**: 收集和聚合所有交易员的性能数据
5. **缓存管理**: 提供竞赛数据和性能统计的缓存机制

## 模块参数定义

### 核心结构体参数

```go
type TraderManager struct {
    traders          map[string]*trader.AutoTrader // 交易员实例映射
    competitionCache *CompetitionCache             // 竞赛数据缓存
    mu               sync.RWMutex                  // 读写锁，保证并发安全
}
```

### 竞争缓存参数

```go
type CompetitionCache struct {
    data      map[string]interface{} // 缓存数据
    timestamp time.Time             // 缓存时间戳
    mu        sync.RWMutex          // 缓存读写锁
}
```

## 模块间关系

### 依赖关系
- **依赖** `config.Database`: 获取交易员配置、AI模型配置、交易所配置
- **依赖** `trader.AutoTrader`: 管理的交易员实例
- **依赖** `market.IndicatorConfig`: 技术指标配置
- **被依赖** `api.Server`: API服务器调用管理器方法

### 数据流向
1. **配置加载**: Database → TraderManager → AutoTrader
2. **状态查询**: API Server → TraderManager → AutoTrader
3. **数据聚合**: AutoTrader → TraderManager → API Server

## 核心方法设计

### 1. 交易员加载方法

#### LoadTradersFromDatabase
```go
func (tm *TraderManager) LoadTradersFromDatabase(database *config.Database) error
```
- **功能**: 从数据库加载所有用户的交易员配置到内存
- **参数**:
  - `database`: 数据库实例
- **返回值**: 错误信息
- **内部逻辑**:
  1. 获取所有用户列表
  2. 为每个用户加载交易员配置
  3. 获取AI模型和交易所配置
  4. 构建AutoTrader实例
  5. 添加到内存管理器中

#### LoadUserTraders
```go
func (tm *TraderManager) LoadUserTraders(database *config.Database, userID string) error
```
- **功能**: 为指定用户加载交易员配置
- **参数**:
  - `database`: 数据库实例
  - `userID`: 用户ID
- **返回值**: 错误信息

#### LoadTraderByID
```go
func (tm *TraderManager) LoadTraderByID(database *config.Database, userID, traderID string) error
```
- **功能**: 加载指定ID的单个交易员
- **参数**:
  - `database`: 数据库实例
  - `userID`: 用户ID
  - `traderID`: 交易员ID
- **返回值**: 错误信息

### 2. 交易员管理方法

#### AddTraderFromDB
```go
func (tm *TraderManager) AddTraderFromDB(...) error
```
- **功能**: 从数据库配置添加新交易员
- **参数**: 包含交易员配置、AI模型配置、交易所配置等
- **返回值**: 错误信息

#### RemoveTrader
```go
func (tm *TraderManager) RemoveTrader(traderID string)
```
- **功能**: 从内存中移除交易员
- **参数**:
  - `traderID`: 交易员ID
- **返回值**: 无

### 3. 交易员查询方法

#### GetTrader
```go
func (tm *TraderManager) GetTrader(id string) (*trader.AutoTrader, error)
```
- **功能**: 获取指定ID的交易员实例
- **参数**:
  - `id`: 交易员ID
- **返回值**: 交易员实例和错误信息

#### GetAllTraders
```go
func (tm *TraderManager) GetAllTraders() map[string]*trader.AutoTrader
```
- **功能**: 获取所有交易员实例的副本
- **返回值**: 交易员实例映射

#### GetTraderIDs
```go
func (tm *TraderManager) GetTraderIDs() []string
```
- **功能**: 获取所有交易员ID列表
- **返回值**: 交易员ID字符串切片

### 4. 批量操作方法

#### StartAll
```go
func (tm *TraderManager) StartAll()
```
- **功能**: 启动所有交易员
- **实现**: 并发启动每个交易员

#### StopAll
```go
func (tm *TraderManager) StopAll()
```
- **功能**: 停止所有交易员
- **实现**: 遍历调用每个交易员的停止方法

### 5. 数据聚合方法

#### GetComparisonData
```go
func (tm *TraderManager) GetComparisonData() (map[string]interface{}, error)
```
- **功能**: 获取交易员对比数据
- **返回值**: 包含所有交易员性能数据的映射

#### GetCompetitionData
```go
func (tm *TraderManager) GetCompetitionData() (map[string]interface{}, error)
```
- **功能**: 获取竞赛排行榜数据
- **特性**:
  - 使用缓存机制（30秒有效期）
  - 按收益率排序
  - 限制返回前50名
  - 并发数据获取

#### GetTopTradersData
```go
func (tm *TraderManager) GetTopTradersData() (map[string]interface{}, error)
```
- **功能**: 获取前5名交易员数据
- **特性**: 复用竞赛数据缓存

### 6. 配置管理方法

#### ReloadIndicatorConfig
```go
func (tm *TraderManager) ReloadIndicatorConfig(traderID string, newConfig *market.IndicatorConfig) error
```
- **功能**: 热重载指定交易员的技术指标配置
- **参数**:
  - `traderID`: 交易员ID
  - `newConfig`: 新的指标配置
- **返回值**: 错误信息

## 并发安全设计

### 读写锁使用
- **读操作**: 使用`RLock()`允许多个goroutine同时读取
- **写操作**: 使用`Lock()`确保写操作的原子性
- **缓存访问**: 独立的读写锁保护缓存数据

### 并发数据获取
```go
func (tm *TraderManager) getConcurrentTraderData(traders []*trader.AutoTrader) []map[string]interface{}
```
- **功能**: 并发获取多个交易员的数据
- **特性**:
  - 每个交易员数据获取设置3秒超时
  - 使用context实现超时控制
  - 错误处理和降级策略

## 缓存策略

### 竞赛数据缓存
- **缓存时间**: 30秒
- **缓存内容**: 排行榜数据、交易员性能统计
- **更新策略**: 时间到期自动重新获取
- **并发控制**: 读写锁保护缓存数据

### 缓存失效机制
- 自动过期检查
- 手动缓存清理
- 数据更新时自动刷新

## 错误处理策略

### 配置加载错误
- AI模型不存在 → 跳过该交易员，记录警告
- 交易所配置缺失 → 跳过该交易员，记录警告
- 配置解析失败 → 使用默认配置，记录警告

### 运行时错误
- 交易员获取数据超时 → 返回默认值，标记错误
- 数据格式异常 → 跳过该交易员数据，记录错误
- 网络连接失败 → 使用缓存数据，记录警告

## 性能优化

### 批量操作优化
- 循环外批量查询配置数据
- 减少数据库访问次数
- 缩短锁持有时间

### 并发处理
- 数据获取并发执行
- 异步处理耗时操作
- 使用channel传递结果

### 内存管理
- 使用数据副本避免外部修改
- 及时释放不再使用的资源
- 合理设置缓存大小

## 监控和日志

### 关键操作日志
- 交易员加载/卸载
- 配置更新
- 错误和异常
- 性能指标

### 性能监控
- 缓存命中率
- 数据获取延迟
- 并发操作数量
- 内存使用情况
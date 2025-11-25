# API服务器模块设计

## 模块概述

API服务器是NOFX系统的HTTP接口层，提供完整的RESTful API服务，负责处理前端请求、用户认证、业务逻辑协调和数据返回。它采用Gin框架构建，支持JWT认证、请求验证、错误处理和响应格式化。

## 核心职责

1. **HTTP服务**: 提供RESTful API接口和静态文件服务
2. **用户认证**: JWT令牌生成、验证和刷新
3. **请求路由**: 请求分发和处理器协调
4. **数据验证**: 输入参数验证和格式化
5. **错误处理**: 统一错误响应格式和日志记录
6. **中间件管理**: 认证、日志、CORS等中间件

## 模块参数定义

### 服务器配置结构

```go
type Server struct {
    traderManager   *manager.TraderManager // 交易员管理器引用
    database        *config.Database       // 数据库实例
    cryptoService   *crypto.CryptoService  // 加密服务
    port            int                    // 监听端口
    disableOTP      bool                   // 是否禁用OTP
    engine          *gin.Engine            // Gin引擎实例
    tokenBlacklist  map[string]time.Time   // 令牌黑名单
    blacklistMutex  sync.RWMutex          // 黑名单读写锁
}
```

### 请求响应结构

```go
// API响应基础结构
type APIResponse struct {
    Success   bool        `json:"success"`   // 操作是否成功
    Message   string      `json:"message"`   // 响应消息
    Data      interface{} `json:"data"`      // 响应数据
    Error     string      `json:"error"`     // 错误信息（可选）
    Timestamp int64       `json:"timestamp"` // 响应时间戳
}

// 分页响应结构
type PaginatedResponse struct {
    Success   bool        `json:"success"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data"`
    Pagination Pagination  `json:"pagination"`
    Timestamp int64       `json:"timestamp"`
}

// 分页信息
type Pagination struct {
    Page       int `json:"page"`        // 当前页码
    PageSize   int `json:"page_size"`   // 每页大小
    Total      int `json:"total"`       // 总记录数
    TotalPages int `json:"total_pages"` // 总页数
}
```

### 认证相关结构

```go
// 登录请求
type LoginRequest struct {
    Password string `json:"password" binding:"required"` // 管理员密码
}

// 登录响应
type LoginResponse struct {
    Token     string    `json:"token"`      // JWT令牌
    ExpiresAt time.Time `json:"expires_at"` // 过期时间
}

// JWT Claims
type Claims struct {
    UserID   string    `json:"user_id"`   // 用户ID
    Username string    `json:"username"` // 用户名
    IsAdmin  bool      `json:"is_admin"` // 是否管理员
    IssuedAt time.Time `json:"iat"`      // 签发时间
    ExpiresAt time.Time `json:"exp"`      // 过期时间
}
```

## 模块间关系

### 依赖关系
- **依赖** `manager.TraderManager`: 交易员管理器
- **依赖** `config.Database`: 数据库操作
- **依赖** `crypto.CryptoService`: 加密解密服务
- **被依赖** `main`: 主程序启动和关闭

### 数据流向
1. **请求流**: 前端 → API Server → 业务逻辑层
2. **响应流**: 业务逻辑层 → API Server → 前端
3. **认证流**: 前端 → API Server → JWT验证 → 业务逻辑

## API端点设计

### 1. 认证管理端点

#### POST /api/admin-login
- **功能**: 管理员登录
- **请求体**: `LoginRequest`
- **响应**: `LoginResponse`
- **认证**: 无需认证
- **限流**: 每分钟最多5次尝试

#### POST /api/logout
- **功能**: 用户登出
- **请求体**: 无
- **响应**: `APIResponse`
- **认证**: 需要JWT
- **功能**: 将令牌加入黑名单

#### GET /api/auth/status
- **功能**: 检查认证状态
- **响应**: `APIResponse`包含用户信息
- **认证**: 需要JWT

### 2. AI模型管理端点

#### GET /api/models
- **功能**: 获取AI模型配置列表
- **查询参数**: `enabled` (可选) - 筛选启用状态
- **响应**: `PaginatedResponse`包含AI模型列表
- **认证**: 需要JWT

#### PUT /api/models
- **功能**: 更新AI模型配置
- **请求体**: AI模型配置数组
- **响应**: `APIResponse`
- **认证**: 需要JWT
- **功能**: 批量更新AI模型配置

#### POST /api/models/test
- **功能**: 测试AI模型连接
- **请求体**: AI模型配置
- **响应**: `APIResponse`包含测试结果
- **认证**: 需要JWT

### 3. 交易所管理端点

#### GET /api/exchanges
- **功能**: 获取交易所配置列表
- **查询参数**: `enabled` (可选) - 筛选启用状态
- **响应**: `PaginatedResponse`包含交易所列表
- **认证**: 需要JWT

#### PUT /api/exchanges
- **功能**: 更新交易所配置
- **请求体**: 交易所配置数组
- **响应**: `APIResponse`
- **认证**: 需要JWT
- **加密**: 自动加密API密钥

#### POST /api/exchanges/test
- **功能**: 测试交易所连接
- **请求体**: 交易所配置
- **响应**: `APIResponse`包含测试结果
- **认证**: 需要JWT

### 4. 交易员管理端点

#### GET /api/traders
- **功能**: 获取交易员列表
- **查询参数**:
  - `page` (可选) - 页码，默认1
  - `page_size` (可选) - 每页大小，默认10
  - `status` (可选) - 筛选运行状态
- **响应**: `PaginatedResponse`包含交易员列表
- **认证**: 需要JWT

#### POST /api/traders
- **功能**: 创建新交易员
- **请求体**: 交易员配置
- **响应**: `APIResponse`包含创建的交易员
- **认证**: 需要JWT
- **验证**: 验证AI模型和交易所配置存在

#### GET /api/traders/:id
- **功能**: 获取指定交易员详情
- **路径参数**: `id` - 交易员ID
- **响应**: `APIResponse`包含交易员详情
- **认证**: 需要JWT

#### PUT /api/traders/:id
- **功能**: 更新交易员配置
- **路径参数**: `id` - 交易员ID
- **请求体**: 交易员配置
- **响应**: `APIResponse`
- **认证**: 需要JWT

#### DELETE /api/traders/:id
- **功能**: 删除交易员
- **路径参数**: `id` - 交易员ID
- **响应**: `APIResponse`
- **认证**: 需要JWT
- **安全**: 检查交易员是否在运行

#### POST /api/traders/:id/start
- **功能**: 启动交易员
- **路径参数**: `id` - 交易员ID
- **响应**: `APIResponse`
- **认证**: 需要JWT
- **状态**: 检查配置完整性和依赖项

#### POST /api/traders/:id/stop
- **功能**: 停止交易员
- **路径参数**: `id` - 交易员ID
- **响应**: `APIResponse`
- **认证**: 需要JWT

### 5. 交易数据端点

#### GET /api/status
- **功能**: 获取系统状态
- **查询参数**: `trader_id` (可选) - 指定交易员
- **响应**: `APIResponse`包含系统状态
- **认证**: 需要JWT

#### GET /api/account
- **功能**: 获取账户信息
- **查询参数**: `trader_id` - 交易员ID
- **响应**: `APIResponse`包含账户信息
- **认证**: 需要JWT

#### GET /api/positions
- **功能**: 获取持仓列表
- **查询参数**: `trader_id` - 交易员ID
- **响应**: `APIResponse`包含持仓列表
- **认证**: 需要JWT

#### GET /api/equity-history
- **功能**: 获取权益历史
- **查询参数**:
  - `trader_id` - 交易员ID
  - `days` (可选) - 天数，默认7
- **响应**: `APIResponse`包含权益历史数据
- **认证**: 需要JWT

#### GET /api/decisions/latest
- **功能**: 获取最新决策记录
- **查询参数**:
  - `trader_id` - 交易员ID
  - `limit` (可选) - 记录数，默认5
- **响应**: `APIResponse`包含决策记录
- **认证**: 需要JWT

#### GET /api/statistics
- **功能**: 获取交易统计
- **查询参数**: `trader_id` - 交易员ID
- **响应**: `APIResponse`包含统计数据
- **认证**: 需要JWT

#### GET /api/performance
- **功能**: 获取AI性能分析
- **查询参数**: `trader_id` - 交易员ID
- **响应**: `APIResponse`包含性能分析
- **认证**: 需要JWT

### 6. 竞赛数据端点

#### GET /api/competition
- **功能**: 获取竞赛排行榜
- **查询参数**:
  - `limit` (可选) - 返回数量，默认50
  - `page` (可选) - 页码，默认1
- **响应**: `APIResponse`包含竞赛数据
- **认证**: 需要JWT

#### GET /api/comparison
- **功能**: 获取交易员对比数据
- **查询参数**: `trader_ids` - 交易员ID列表（逗号分隔）
- **响应**: `APIResponse`包含对比数据
- **认证**: 需要JWT

### 7. 信号源管理端点

#### GET /api/signal-sources
- **功能**: 获取信号源配置
- **响应**: `APIResponse`包含信号源列表
- **认证**: 需要JWT

#### PUT /api/signal-sources
- **功能**: 更新信号源配置
- **请求体**: 信号源配置
- **响应**: `APIResponse`
- **认证**: 需要JWT

### 8. 系统配置端点

#### GET /api/config
- **功能**: 获取系统配置
- **响应**: `APIResponse`包含系统配置
- **认证**: 无需认证（公开配置）

#### PUT /api/config
- **功能**: 更新系统配置
- **请求体**: 系统配置
- **响应**: `APIResponse`
- **认证**: 需要JWT

#### GET /api/health
- **功能**: 健康检查
- **响应**: `{"status":"ok"}`
- **认证**: 无需认证

### 9. 静态资源端点

#### GET /
- **功能**: 首页重定向
- **响应**: 重定向到 `/index.html`

#### GET /static/*
- **功能**: 静态文件服务
- **响应**: 静态文件内容
- **缓存**: 设置适当的缓存头

### 10. 实时监控端点 (WebSocket)

#### GET /ws/backtest/:id
- **功能**: 建立WebSocket连接以接收指定回测任务的实时数据。
- **协议**: WebSocket
- **路径参数**: `id` - 回测任务ID
- **认证**: 需要JWT
- **消息流**: 服务器单向推送 `BacktestMessage` 消息。

## 中间件设计

### 1. 认证中间件

```go
func AuthMiddleware() gin.HandlerFunc
```
- **功能**: JWT令牌验证
- **逻辑**:
  1. 从请求头获取Authorization令牌
  2. 验证令牌格式和签名
  3. 检查令牌是否在黑名单中
  4. 解析用户信息到上下文
  5. 拒绝无效或过期令牌

### 2. 管理员权限中间件

```go
func AdminMiddleware() gin.HandlerFunc
```
- **功能**: 管理员权限验证
- **逻辑**:
  1. 检查用户是否已认证
  2. 验证用户是否具有管理员权限
  3. 拒绝非管理员用户访问

### 3. 日志中间件

```go
func LoggerMiddleware() gin.HandlerFunc
```
- **功能**: 请求日志记录
- **逻辑**:
  1. 记录请求时间、方法、路径
  2. 记录响应状态码和耗时
  3. 记录客户端IP和User-Agent
  4. 记录用户ID（如果已认证）

### 4. CORS中间件

```go
func CORSMiddleware() gin.HandlerFunc
```
- **功能**: 跨域请求处理
- **逻辑**:
  1. 设置允许的源、方法、头部
  2. 处理预检请求
  3. 设置CORS相关头部

### 5. 错误处理中间件

```go
func ErrorMiddleware() gin.HandlerFunc
```
- **功能**: 统一错误处理
- **逻辑**:
  1. 捕获panic和错误
  2. 格式化错误响应
  3. 记录错误日志
  4. 返回适当的HTTP状态码

### 6. 限流中间件

```go
func RateLimitMiddleware(requests int, window time.Duration) gin.HandlerFunc
```
- **功能**: API请求限流
- **参数**:
  - `requests`: 允许的请求数
  - `window`: 时间窗口
- **逻辑**:
  1. 基于IP或用户ID进行限流
  2. 使用滑动窗口算法
  3. 返回429状态码当超限时

## 响应格式设计

### 成功响应格式
```json
{
    "success": true,
    "message": "操作成功",
    "data": {
        // 具体数据内容
    },
    "timestamp": 1634567890
}
```

### 错误响应格式
```json
{
    "success": false,
    "message": "操作失败",
    "error": "详细错误信息",
    "timestamp": 1634567890
}
```

### 分页响应格式
```json
{
    "success": true,
    "message": "获取成功",
    "data": [
        // 数据列表
    ],
    "pagination": {
        "page": 1,
        "page_size": 10,
        "total": 100,
        "total_pages": 10
    },
    "timestamp": 1634567890
}
```

## 错误处理策略

### HTTP状态码使用
- `200 OK`: 请求成功
- `201 Created`: 资源创建成功
- `400 Bad Request`: 请求参数错误
- `401 Unauthorized`: 未认证或认证失败
- `403 Forbidden`: 权限不足
- `404 Not Found`: 资源不存在
- `409 Conflict`: 资源冲突
- `422 Unprocessable Entity`: 请求格式正确但语义错误
- `429 Too Many Requests`: 请求过于频繁
- `500 Internal Server Error`: 服务器内部错误

### 错误码设计
```go
const (
    // 通用错误码
    ErrCodeSuccess           = 0     // 成功
    ErrCodeInvalidRequest    = 1001  // 无效请求
    ErrCodeUnauthorized      = 1002  // 未授权
    ErrCodeForbidden         = 1003  // 禁止访问
    ErrCodeNotFound          = 1004  // 资源不存在
    ErrCodeConflict          = 1005  // 资源冲突
    ErrCodeInternalError     = 1006  // 内部错误

    // 业务错误码
    ErrCodeTraderNotFound    = 2001  // 交易员不存在
    ErrCodeTraderRunning     = 2002  // 交易员正在运行
    ErrCodeInvalidConfig     = 2003  // 配置无效
    ErrCodeAIModelFailed     = 2004  // AI模型调用失败
    ErrCodeExchangeError     = 2005  // 交易所错误
)
```

## 安全设计

### 1. 认证安全
- JWT令牌有效期24小时
- 令牌黑名单机制
- 强制HTTPS（生产环境）
- 密码强度验证

### 2. 输入验证
- 参数类型和格式验证
- SQL注入防护
- XSS攻击防护
- 文件上传安全检查

### 3. 访问控制
- 基于角色的权限控制
- API接口权限矩阵
- 敏感操作二次验证
- 操作日志审计

### 4. 数据保护
- 敏感数据加密存储
- 传输层TLS加密
- 密钥轮换机制
- 数据脱敏处理

## 性能优化

### 1. 响应优化
- 启用Gzip压缩
- 设置适当的缓存头
- 减少响应数据大小
- 使用CDN加速静态资源

### 2. 数据库优化
- 连接池管理
- 查询优化和索引
- 分页查询大数据集
- 缓存热点数据

### 3. 并发处理
- 协程池管理
- 请求去重处理
- 限流和降级
- 异步任务处理

## 监控和日志

### 1. 请求监控
- 请求量统计
- 响应时间监控
- 错误率统计
- 热点接口分析

### 2. 性能监控
- CPU和内存使用率
- 数据库连接池状态
- 并发请求数量
- 响应时间分布

### 3. 日志记录
- 结构化日志格式
- 不同级别日志分类
- 日志轮转和归档
- 敏感信息过滤
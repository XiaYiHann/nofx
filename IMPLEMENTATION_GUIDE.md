# 回测系统实现指南

## 分支信息

- **分支名**: `implement/backtest-binance-api`
- **基于**: `dev` 分支
- **目标**: 实现基于 Binance API 的回测系统

## 实现步骤概览

根据 `openspec/changes/implement-backtest-system/tasks.md`,分为 3 个阶段:

### 阶段 1: 核心类型和数据加载 (3-4天)
- [ ] 任务 1: 定义回测类型 (`backtest/types.go`)
- [ ] 任务 2: Binance API 集成 (`backtest/binance_loader.go`)
- [ ] 任务 3: 数据加载和转换 (`backtest/data_loader.go`)
- [ ] 任务 4: 数据缓存和持久化
- [ ] 任务 5: 增量更新机制
- [ ] 任务 6a: API 限流保护
- [ ] 任务 6b: 定时任务
- [ ] 任务 7: 数据验证测试

### 阶段 2: 交易模拟系统 (4-5天)
- [ ] 任务 8: 投资组合管理 (`backtest/portfolio.go`)
- [ ] 任务 9: 风险管理系统
- [ ] 任务 10: 交易模拟器 (`backtest/simulator.go`)
- [ ] 任务 11: 交易执行优化
- [ ] 任务 12: 模拟器测试

### 阶段 3: 回测引擎核心 (3-4天)
- [ ] 任务 13: 主回测引擎 (`backtest/backtest.go`)
- [ ] 任务 14: 回测报告生成
- [ ] 任务 15: API 端点实现
- [ ] 任务 16: Web UI 实现

## 关键文件和接口

### 现有代码结构

```
nofx/
├── market/
│   ├── api_client.go      # 需要扩展: GetHistoricalKlines()
│   ├── types.go           # Kline, Data 等类型定义
│   └── monitor.go
├── decision/
│   ├── engine.go          # 复用: GetFullDecision()
│   └── types.go
├── trader/
│   ├── interface.go       # 需要实现该接口
│   └── binance_trader.go
├── api/
│   └── server.go          # 需要添加回测端点
├── config/
│   └── config.go
└── bootstrap/
    └── bootstrap.go       # 应用初始化
```

### 新增代码结构

```
backtest/
├── types.go              # 回测类型定义
├── binance_loader.go     # Binance API 数据加载
├── data_loader.go        # 数据加载和转换
├── simulator.go          # 交易模拟器 (实现 trader.Trader)
├── portfolio.go          # 投资组合管理
├── backtest.go           # 主回测引擎
├── report.go             # 报告生成
└── cache.go              # 缓存管理 (可选)
```

## 核心概念

### 1. 数据流

```
Binance API
    ↓
binance_loader.GetHistoricalKlines()
    ↓
[]market.Kline (原始 K 线数据)
    ↓
market.Get() (计算技术指标)
    ↓
market.Data (包含所有指标的完整数据)
    ↓
decision.GetFullDecision() (AI 决策)
    ↓
simulator.Execute*() (模拟执易)
    ↓
portfolio.UpdatePnL() (更新账户)
```

### 2. 三级缓存机制

```
内存缓存 (Map)
    ↓ 未命中
SQLite 缓存 (backtest.db)
    ↓ 未命中
Binance API 下载
```

### 3. API 限流 (1200 req/min)

```go
速率限制: 50ms/请求
重试机制: 指数退避 (429 错误)
最大重试: 3 次
```

## 实现检查清单

### 第 1 周

- [ ] 创建 `backtest/types.go` - 定义所有数据结构
- [ ] 创建 `backtest/binance_loader.go` - Binance API 集成
- [ ] 扩展 `market/api_client.go` - 添加 GetHistoricalKlines()
- [ ] 创建 `backtest/data_loader.go` - 数据加载和转换
- [ ] 实现缓存机制和数据持久化

**验证**: 能够成功加载 1 天的 3m 数据并转换为 market.Data

### 第 2 周

- [ ] 创建 `backtest/portfolio.go` - 投资组合管理
- [ ] 创建 `backtest/simulator.go` - 实现 trader.Trader 接口
- [ ] 实现风险管理系统
- [ ] 编写单元测试

**验证**: 能够模拟开仓、平仓和 PnL 计算

### 第 3 周

- [ ] 创建 `backtest/backtest.go` - 主回测引擎
- [ ] 创建 `backtest/report.go` - 报告生成
- [ ] 添加 API 端点 (api/server.go)
- [ ] 前端 UI 集成

**验证**: 完整回测流程可正常运行

## 重要提醒

### 1. 复用现有模块

✅ **必须复用**:
- `market.Get()` - 计算技术指标
- `decision.GetFullDecision()` - AI 决策
- `trader.Trader` 接口 - 定义交易方法
- `github.com/adshao/go-binance/v2` - Binance SDK

❌ **禁止修改**:
- 现有的 `market/`, `decision/`, `trader/` 代码
- 现有的交易执行逻辑
- 现有的配置系统

### 2. 数据一致性

- 确保回测数据与实时交易数据格式完全相同
- 使用相同的技术指标计算方式
- 使用相同的单位和精度

### 3. API 调用注意事项

```go
// ✅ 正确: 遵守速率限制
for symbol := range symbols {
    time.Sleep(50 * time.Millisecond)
    klines, _ := apiClient.GetHistoricalKlines(...)
}

// ❌ 错误: 立即连续调用
for symbol := range symbols {
    klines, _ := apiClient.GetHistoricalKlines(...) // 会被限流!
}
```

### 4. 测试覆盖率

- 单元测试覆盖率 > 80%
- 关键路径的集成测试
- 边界情况测试 (空数据、错误响应等)

## 常用命令

```bash
# 查看当前分支
git branch -v

# 查看提案详情
openspec show implement-backtest-system

# 查看任务列表
cat openspec/changes/implement-backtest-system/tasks.md

# 查看设计文档
cat openspec/changes/implement-backtest-system/design.md

# 运行测试
go test ./backtest/... -v

# 构建项目
go build

# 运行应用
./nofx
```

## 参考资源

- 设计文档: `openspec/changes/implement-backtest-system/design.md`
- Binance API 设计: `openspec/changes/implement-backtest-system/BINANCE_API_DESIGN.md`
- 任务清单: `openspec/changes/implement-backtest-system/tasks.md`
- 规格文档: `openspec/changes/implement-backtest-system/specs/backtest-data/spec.md`

## 问题排查

### 编译错误

如果遇到 `market.Data` 或其他结构体不存在:
1. 检查 `market/types.go` 中的定义
2. 确保使用了正确的导入路径
3. 运行 `go mod tidy`

### API 限流错误

如果遇到 429 错误:
1. 检查速率限制器是否启用
2. 确认重试逻辑是否工作
3. 查看日志输出

### 数据不一致

如果回测数据与实时交易不同:
1. 检查 `market.Get()` 调用是否正确
2. 验证 Kline 数据格式转换
3. 查看 API 响应是否完整

## 沟通和反馈

实现过程中遇到问题:
1. 查阅设计文档和规格文档
2. 检查相关的已实现代码
3. 参考 Binance API 文档: https://binance-docs.github.io/apidocs/futures/en/

祝你实现顺利! 🚀

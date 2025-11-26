# 保证金计算与自动缩放机制

**版本**: v1.0  
**更新日期**: 2025-11-26  
**适用模块**: `trader/auto_trader.go`, `decision/engine.go`, `logger/decision_logger.go`

---

## 概述

本文档说明 NoFx 系统中的保证金计算逻辑、自动缩放机制以及相关的审计记录。

## 问题背景

当 AI 决策建议的 `position_size_usd` 导致所需保证金超过可用余额时，交易会失败。例如：
- **可用余额**: 3609.68 USDT
- **AI 建议仓位**: 7228 USDT (杠杆 1x)
- **所需保证金**: 7228 USDT
- **结果**: 保证金不足错误

## 保证金计算公式

```
所需保证金 = position_size_usd / leverage
预估手续费 = position_size_usd × 0.0004 (Taker 费率 0.04%)
总需求 = 所需保证金 + 预估手续费
```

## 自动缩放机制

### 触发条件

当 `总需求 > 可用余额 × 0.95`（安全边际系数）时，系统会自动缩放仓位。

### 缩放计算

```
最大仓位 = 可用余额 × 0.95 × 杠杆
```

### 最小仓位限制

缩放后的仓位必须 ≥ 12 USDT（交易所最小名义价值 10 USDT + 安全边际）。

### 拒绝条件

如果可用余额不足以开立最小仓位（12 USDT），交易将被拒绝。

## 代码位置

| 功能 | 文件 | 函数 |
|------|------|------|
| 保证金检查与缩放 | `trader/auto_trader.go` | `checkAndScaleMargin()` |
| 开多仓执行 | `trader/auto_trader.go` | `executeOpenLongWithRecord()` |
| 开空仓执行 | `trader/auto_trader.go` | `executeOpenShortWithRecord()` |
| Prompt 约束说明 | `decision/engine.go` | `buildSystemPrompt()` |

## 审计记录

所有自动缩放操作都会记录到 `decision_logs/` 目录，包含以下字段：

```json
{
  "auto_scaled": true,
  "original_position_size_usd": 7228.00,
  "scaled_position_size_usd": 3420.00,
  "required_margin": 342.00,
  "available_balance": 3609.68,
  "scale_reason": "保证金不足，从 7228.00 USDT 缩放到 3420.00 USDT (原需保证金 722.80，可用 3609.68)"
}
```

## Prompt 约束

系统在 System Prompt 中向 AI 明确说明保证金约束：

```
7. **⚠️ 保证金约束**: 开仓所需保证金 = position_size_usd / leverage，**必须 ≤ 可用余额的 95%**（预留手续费）
   - 例: 可用余额 3600 USDT，杠杆 10x → 最大 position_size_usd = 3600 * 0.95 * 10 = 34200 USDT
```

## 测试覆盖

| 测试文件 | 测试用例 |
|----------|----------|
| `trader/auto_trader_test.go` | `TestCheckAndScaleMargin_*` (6个用例) |
| `trader/auto_trader_test.go` | `TestExecuteOpenLong_WithAutoScale` |
| `trader/auto_trader_test.go` | `TestExecuteOpenShort_WithAutoScale` |
| `trader/auto_trader_test.go` | `TestExecuteOpenLong_InsufficientBalance_Rejected` |
| `backtest/order_simulator_test.go` | `TestOrderSimulator_InsufficientMargin` |

运行测试：
```bash
go test ./trader/... -run "TestCheckAndScaleMargin|TestExecuteOpen" -v
go test ./backtest/... -run "TestOrderSimulator_InsufficientMargin" -v
```

## 日志示例

### 自动缩放成功
```
  ⚠️ 自动缩放仓位: 7228.00 → 3420.00 USDT (杠杆 10x, 可用余额 3609.68 USDT)
  ✓ 开仓成功（自动缩放），订单ID: 123456, 数量: 0.0684, 仓位: 7228.00 → 3420.00 USDT
```

### 余额不足被拒绝
```
❌ 可用余额不足以开立最小仓位: 需要≥12.00 USDT，最大可开 5.00 USDT
```

## 配置建议

1. **杠杆设置**: 使用较高杠杆可以减少保证金需求，但增加爆仓风险
2. **仓位大小**: AI 决策时会参考可用余额，但建议在 Prompt 中明确强调保证金限制
3. **安全边际**: 系统默认预留 5% 安全边际，不建议修改

## 相关文档

- [Prompt 编写指南](./prompt-guide.zh-CN.md)
- [回测引擎使用指南](../BACKTEST_USAGE.md)
- [Paper Trading 指南](../PAPER_TRADING_FIXED.md)

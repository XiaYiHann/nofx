### 2025-11-28 22:38:40 CST
**变更摘要**: cherry-pick `e4bfad38`（修复：止盈/止损单在反向平仓后未被撤消），并将该修复集成到当前分支。主要改动包括：在 `Trader` 接口新增 `GetOpenOrders`，为 Binance/Bybit/Hyperliquid/Aster/Lighter(LighterV2) 实现该方法，并在 `AutoTrader.buildTradingContext` 中新增残留挂单清理逻辑；同时更新相关 Mock 与测试用例以保证行为一致性与回归测试覆盖。

**设计思路**:
- 最小范围变更：采用 cherry-pick 单提交而非完整合并 `ares` 分支，避免引入大量冲突与未评审功能。
- 统一抽象与鲁棒性：通过在 `Trader` 接口新增 `GetOpenOrders` 的统一方法，保证跨交易所的订单检查能力，并在无法区分止盈/止损的实现中，使用 `CancelAllOrders` 做安全清理。
- 非阻塞容错：若 `GetOpenOrders` 出错，仅记录警告日志并继续主流程，避免因监控/网络问题影响核心交易决策流程。

**修改意图**:
- 消除"幽灵挂单"与残留止盈/止损单：确保在平仓后不会残留未被撤销的挂单，从而避免错误的后续交易或资金锁定。
- 降低合并风险：通过保持 `dev` 分支的主线实现不变，仅引入必要的修复与接口，降低系统回归风险。
- 保证可测试性：为所有实现与 mock 添加 `GetOpenOrders`，并在单元/集成测试中覆盖此行为，确保回归测试能捕获潜在问题。

---

> 工作区状态（快照）:
> - 未暂存(unstaged): `.github/copilot-instructions.md`, `config/test_rsa_key.pem.pub`
> - 已暂存(staged): 目前无已暂存文件（所有主要变更已经被 commit）
> - 未跟踪(untracked): `.github/prompts/memory.prompt.md`, `docs/merge-plan-ares-to-dev.md`



*附注*: 我已在 `cherry-pick/safe-upstream` 分支上将 `e4bfad38` 的修复应用并运行本地测试（`go test ./trader/...` 与全套 `go test ./...`），所有测试通过。若需要，将该 memory 条目合并到 `dev` 或作为 PR 的说明补充提交。
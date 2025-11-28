### 2025-11-28 23:30:28 CST
**变更摘要**: 添加并验证“测试/开发模式 (--dev)”支持，允许本地绕过登录与 OTP（仅用于本地开发/测试），并在前后端与启动脚本中做了相应兼容改造。

- 关键改动文件（未暂存）：
	- `api/server.go`（后端：auth 中间件、/api/config dev_mode 输出）
	- `main.go`（将 devMode 传入 API Server）
	- `start_local.sh`、`start.sh`、`docker-compose.yml`（启动脚本与部署：解析 --dev 并注入 NOFX_DEV_MODE）
	- `web/src/contexts/AuthContext.tsx`（前端：自动设置 dev 测试用户、保持 token=null）
	- `web/src/App.tsx`（路由保护放宽，只检查 user 不强制 token；SWR 条件调整）
	- `web/src/components/LoginPage.tsx`（显示 dev 模式 Banner 和“直接进入系统”按钮）
	- `web/src/pages/AITradersPage.tsx`、`web/src/stores/tradersConfigStore.ts`、`web/src/lib/config.ts`（其他兼容性调整）
	- 调试/验证产物：`screenshots/dev_mode_traders_page.png`, `screenshots/dev_mode_test_success.png`

**设计思路**:
- 最小入侵与环境控制：通过单一环境变量 `NOFX_DEV_MODE` 控制所有 dev 行为，避免改变生产逻辑或永久凭证。
- 后端作为可信边界：当 devMode 启用时，后端 `authMiddleware` 注入固定测试用户（`dev-user`），并绕过 JWT/OTP 检查；前端仅依赖 `user` 存在性以允许访问（token 可为 null）。
- 前端 UX 与可见性：在 UI 明显提示 dev 模式（黄色 Banner），并提供“直接进入系统”快捷动作，避免误导与混淆。

**修改意图**:
- 降低本地开发与演示的门槛：支持 `./start_local.sh start --dev` / `./start.sh start --dev`，快速进入系统以便调试与演示。
- 安全与范围限定：仅在 `NOFX_DEV_MODE=true` 时生效，明确只用于本地/测试环境，避免生产滥用。
- 提高可复现性：保存验证截图与日志，便于审计、回溯与自动化测试场景构建。

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
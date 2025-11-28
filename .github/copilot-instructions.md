# NOFX AI 代码助理指南

Tip:永远用中文回答

## 一、项目概览

- `main.go` 负责启动整个服务：Gin REST API、`manager/trader_manager.go`、多个 `trader/` 实例以及后台调度协程。
- 所有 HTTP 请求由 `api/server.go` 处理，数据流依次经过 auth 中间件 → `manager` → `trader` → `trader.*` 交换机实现，再通过 `decision/engine.go`、`market`、`mcp/client.go` 调用 AI + 市场数据。
- `market/` 提供统一的多时间框架 K 线和技术指标（TA-Lib），`decision_logs/` 按 `decision_logger.go` 格式写出每次决策的 JSON 记录，`config/database.go` 管理 SQLite 表结构。
- `web/` 前端通过 `web/src/lib/api.ts` 讲 REST 接口封装，`stores/` 使用 Zustand 管理状态，UI 与 `api/` 的 `/api/traders`、`/api/decisions`、`/api/positions` 等端点保持同步。
- '/github/memory.md' 记录了每次变更的设计思路和修改意图，便于后续回顾和理解决策背景。

## 二、运行与验证流程

- 本地开发推荐 `./start_local.sh start`：会自动检查 Go/Node、生成 `config.json` + `secrets/` 密钥、启动 Go 后端（8080）和 Vite 前端（5173）。日志在根目录的 `backend.log`/`frontend.log`。
- Docker/生产部署优先 `./start.sh start`，`./start.sh build` 重建镜像，`./start.sh status|logs` 用于排查。`start.sh` 会构造 `.env`（环境变量见下节）、`config.json`、RSA 密钥。
- 保证测试命令通畅：后端 `go test ./...`，重要集成 `./test_backtest_full.sh`、`./test_binance_testnet.sh`；前端在 `web/` 目录运行 `npm run build`、`npm run test`、`npm run lint`，`npm run dev` 用于本地调试。

## 三、配置与环境依赖

- 改动前确认 `.env` 已配置关键变量：`BINANCE_TESTNET_API_KEY`/`SECRET`, `LLM_API_URL`/`LLM_API_KEY`/`LLM_MODEL`、`DATA_ENCRYPTION_KEY`、`JWT_SECRET`、`NOFX_BACKEND_PORT` 与 `NOFX_FRONTEND_PORT`。这些值直接驱动交易 API、AI 模型调用与 JWT 加密。
- `config.json` 仍是管理前端配置的最终来源（`manager` 读取 `traders`、`models`、`exchanges` 配置），运行时会把 config 写入 SQLite，进程重启后不需重新编辑 JSON。
- `secrets/` 保留 RSA 密钥对，`decision_logs/` 记录 AI 决策详情，`prompts/` 存放可复用的 prompt 模板（`decision/prompt_manager.go` 会加载）。新建表/字段需同步 `config/database.go` + `config/database_test.go`。

## 四、关键代码约定

- `trader/interface.go` 定义 `ExchangeClient` 抽象，`trader/binance_futures.go`、`trader/hyperliquid_trader.go` 与 `trader/aster_trader.go` 共享同一生命周期：`trader_manager` 控制启动/关闭，`auto_trader` 调度决策周期（fetch market → gen prompt → call mcp → risk check → execute order）。
- 所有交易决策都会通过 `decision/engine.go` 结合 `decision/prompt_manager.go`、`mcp/client.go` 与 `market/data.go` 的多时间框架指标生成 Chain-of-Thought。执行前通过 `manager/risk` 约束，在 `logger/decision_logger.go` 记录后并发写入 `decision_logs/{trader_id}`。
- 新的 REST 端点请新增 `api/server.go` 路由、对应 handler，使用 `api/utils.go` 提供的分页/JSON helper。添加数据库逻辑时同步更新 `config/database.go`、`config/database_test.go`、`config/paper_trading_test.go` 的 schema 预置。
- 任何涉及 AI 模型切换、LLM key 管理或 prompt 模板改动都要检查 `mcp/client.go` 和 `decision/prompt_manager.go` 的硬编码列表，并添加对应 `docs/prompt-guide.md` 或 `docs/prompt-guide.zh-CN.md` 注释说明。

## 五、AI/规范协助策略

- 重大能力、新 API、架构调整或性能/安全改动前，先读 `openspec/project.md`、`openspec/AGENTS.md`。所有这类请求都需要遵循 `openspec` 的 change proposal 流程（`openspec spec list --long`、`openspec list`、`proposal.md` + delta spec），否则先给出问题范围和提问。
- 任何涉及规划、提案、模糊需求或大改动的对话，务必打开 `openspec/AGENTS.md` 以及根目录 `AGENTS.md` 中的 “强制阅读”模块，确认是否要创建 spec 或 trigger `openspec validate`。
- 非重大修改直接对代码或文档做变更时，参考 `docs/architecture/README.md` 中的数据流图、`docs/maintainers/README.md` 中的提交约定以及 `docs/getting-started/README.md` 的部署命令保持一致。

## 六、交互与验证重点

- 回归/集成测试： `backtest/engine_test.go`、`backtest/order_simulator_test.go`、`manager/trader_manager_test.go`、`decision/engine_test.go`、`market/*.go` 系列。这些目录新增功能时需补充 `_test.go` 覆盖核心路径。
- UI 变更需同步 `web/src/stores` 与 `web/src/components`，并保持 `web/package.json` 的 lint/format 规则（`npm run lint` + `npm run format:check`）和 Husky 钩子一致。
- 任何涉及部署或密钥的说明，记得在 `BACKTEST_QUICKSTART.md` / `START_GUIDE.md` / `START_SCRIPTS_README.md` 中更新对应步骤，文档引用 `screenshots/` 中的 UI 图例用于快速说明。

## 七、交付建议

- 变更完后运行 `./start_local.sh status` + `go test ./...` + `npm test`，确保前后端联通。常用辅助脚本：`./test_backtest_full.sh`、`scripts/generate_beta_code.sh`。
- 文档更新时引用 `docs/architecture/README.md` 章节（用 `README.md` 的目录结构指导）并保持中文/英文同步。新增文件请添加到 `docs/README.md` 索引。

欢迎根据本指南调整交付节奏，如需补充不清楚的部分，请指出让我们继续迭代。

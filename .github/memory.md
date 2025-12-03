### 2025-12-03 16:06:24 CST

**变更摘要**: 在 `market` 包中新增并集成 10 种技术指标（SMA, VWAP, OBV, Stochastic, Williams %R, CCI, ADX, Parabolic SAR, CMF, Ichimoku）。这些改动涉及：在数据模型中添加字段（`TimeframeData` / `IntradayData`）、在 `data.go` 实现各指标的计算函数并在 `CalculateTimeframeData()` 中集成、补充并完善对应单元测试（`market/data_test.go`），同时更新前端配置界面 (`web/src/components/IndicatorConfigPanel.tsx`) 和 LLM prompt 模板 (`prompts/nof1.txt`)。已运行并通过相关测试与前端构建验证。

**设计思路**:
- 与现有指标实现风格保持一致：采用基于历史切片的滚动计算方式，尽量复用已有的指标计算与校验模式（预热期判断、数据点不足返回安全默认值）。
- 数据结构优先：在 `TimeframeData` / `IntradayData` 中添加数组字段用于保存每一周期指标序列，确保链路中决策引擎、日志和前端均能直接消费同一标准化结构。
- 测试驱动与向后兼容：为每个新增指标添加单元测试并在集成点（CalculateTimeframeData）验证字段存在与长度匹配，保证改动可回滚且不会破坏现有消费者。

**修改意图**:
- 扩展交易策略和 LLM 决策引擎可用的特征集合，提升决策质量与可解释性。
- 通过后端 + 前端 + prompt 三层更新，确保新增指标在 UI 配置、后台计算与决策说明中一致可用。
- 保持高测试覆盖与构建验证，减少引入新指标带来的回归风险。

**工作区快照（临时）**:
- 已修改 (unstaged): `market/data.go`, `market/data_test.go`, `market/types.go`, `prompts/nof1.txt`, `web/src/components/IndicatorConfigPanel.tsx`
- 未跟踪 (untracked): `.vscode/launch.json`, `.vscode/settings.json`, `.vscode/tasks.json`, `BACKTEST_AUDIT_REPORT.md`
- 测试/构建验证：`go test ./market/...` ✅ 通过；`npm run build`（web）✅ 通过。

### 2025-12-01 02:40:00 CST

**变更摘要**: 在 `web` 前端补齐并稳定化大量 Vitest + Testing Library 测试用例 —— 新增/完善多个关键模块的单元与集成测试（包括 `AuthContext`、`api`、`tradersConfigStore`、`HeaderBar`、`App`、`LoginPage`、`RegisterPage`、`EquityChart` 等），目前总计 356 个测试通过；工作区仍有未跟踪的新增测试文件与未暂存的 `web/package.json` / `web/package-lock.json` 修改。

**设计思路**:

- 优先确保关键路径高覆盖（Priority-1 模块目标 ≥80%），以降低回归风险；次序性地补充其他模块以逐步提高整体覆盖率（目标 ≥50%）。
- 在不引入新依赖的前提下，保持测试稳定性：统一 SWR mock 模式、使用 `fireEvent`（项目约束下不使用 `@testing-library/user-event`）、对 i18n 使用实际中文文本断言、对重复 DOM 节点使用 `getAllByText()` 以防偶发断言失败。
- 以可重复、可维护的 mock 策略为中心（SWR 明确 key 比对），避免脆弱测试和不可控的外部依赖。

**修改意图**:

- 快速提升关键模块的测试覆盖并保证稳定性，让 CI 与本地开发都能可靠地捕获回归。
- 修复测试失败的常见根因（翻译 key 与真实文本不一致、SWR key 不精确匹配、多节点断言冲突、异步更新未被 act 包裹等），记录最佳实践以便后续编写测试时复用。
- 在不牺牲稳定性的情况下，向团队展示如何以受控方式逐步提高整体覆盖，下一步将继续补齐剩余模块（如 AI 相关页面、PositionPage、ComparisonChart 等）以提升总体覆盖率。

**工作区快照（临时）**:

- 未暂存 (unstaged): `web/package.json`, `web/package-lock.json`
- 未跟踪 (untracked): 多个新增测试文件（例如 `web/src/components/EquityChart.test.tsx`, `web/src/contexts/AuthContext.test.tsx`, `web/src/components/LoginPage.test.tsx`, `web/src/components/landing/HeaderBar.test.tsx`, `web/src/lib/api.test.ts`, `web/src/stores/tradersConfigStore.test.ts` 等）

### 2025-11-29 17:50:00 CST

**变更摘要**: 确保 News 功能在 fresh clone / 系统重启后可见且稳健 —— 将 News 相关文件纳入版本库、修复前端路由与编译问题、为后端服务与处理器以及前端页面补充全面测试并完成 lint/build/test 验证。

- 已暂存 / 新增（待提交）: `api/news_handler.go`, `api/news_handler_test.go`, `market/news/` (client.go, service.go, types.go, client_test.go, service_test.go), `web/src/pages/NewsPage.tsx`, `web/src/pages/NewsPage.test.tsx`
- 未暂存 / 修改中: `web/src/App.tsx`, `web/src/components/landing/HeaderBar.tsx`, `web/src/components/HeaderBar.tsx`, `web/src/routes/index.tsx`, `web/src/lib/httpClient.ts`, `.github/memory.md` 等（含格式化与集成适配）

**设计思路**:

- 兼容现有路由体系：发现仓库同时存在两套路由实现（`App.tsx` 的老式路由与 `routes/index.tsx` 的 React Router），优先在 `App.tsx` / `landing/HeaderBar.tsx` 中补足 `news` 路由以保证在当前入口（`main.tsx -> App.tsx`）下页面可见；避免大规模迁移引入回归。
- 跨层测试覆盖：在后端采用接口注入（`NewsServiceInterface`）来便于 Mock；为服务层、handler 和前端页面分别添加单元/集成测试，覆盖成功/加载/错误/空结果等场景，确保未来变更不破坏可见性。
- 可观测与可部署：修复编译/格式化问题、运行 lint/build/test 全量验证，确保在 fresh clone + build + run 的流程中功能可见且 CI 可验证。

**修改意图**:

- 立即解决用户反馈：重启或新环境下缺失 News 页面问题（原因：关键文件曾为 untracked 或构建错误），保证 `/news` 在本地/开发环境可访问。
- 防止回归：通过把 News 相关文件纳入版本库并补充端到端测试，降低未来遗失或未提交造成的功能丢失风险。
- 提高维护性：通过接口与测试拆分 (Service interface + handler + frontend) 提升可测试性和可扩展性，后续可更容易地添加/替换新闻来源。


**变更摘要**: 完善 News 功能的测试覆盖，确保前端在 fresh clone + build + run 后功能可见且健壮。

**设计思路**:

- **后端测试**: 为 `handleGetNews` 添加单元测试（`api/news_handler_test.go`），覆盖成功获取所有新闻、按类别筛选、JSON 结构验证、服务错误处理、空结果返回等场景。引入 `NewsServiceInterface` 接口支持 mock 注入，避免测试时依赖真实网络请求。
- **服务层测试**: 为 `market/news.Service` 添加测试（`market/news/service_test.go`），覆盖缓存命中、过期刷新、按类别筛选、日期排序、并发源获取、部分失败容错、并发安全等场景。
- **前端测试**: 为 `NewsPage` 添加 Vitest + testing-library 测试（`web/src/pages/NewsPage.test.tsx`），覆盖加载态骨架屏、成功渲染新闻列表（title/summary/source/score/time）、错误状态友好提示、空结果处理、Tab 切换、页面标题等场景（共 14 个测试用例）。
- **代码重构**: 将 `Server.newsService` 从具体类型 `*news.Service` 改为接口 `NewsServiceInterface`，便于测试时注入 mock 实现。

**修改意图**:

- 确保 News 功能在多种边界条件下都能正常工作，提高系统健壮性。
- 通过测试覆盖关键路径，防止未来重构或修改时引入回归。
- 遵循依赖注入原则，使代码更易测试和维护。

**新增文件**:

- `api/news_handler_test.go`: News API handler 测试（3 个测试函数，覆盖成功/错误/空结果）
- `market/news/service_test.go`: News 服务层测试（9 个测试函数，覆盖缓存/刷新/并发等）
- `web/src/pages/NewsPage.test.tsx`: News 页面前端测试（14 个测试用例）

**修改文件**:

- `api/server.go`: 添加 `NewsServiceInterface` 接口定义，将 `newsService` 字段类型改为接口
- `web/src/pages/NewsPage.tsx`: Prettier 格式化修复

**验证结果**:

- `npm run lint` ✅ 通过
- `npm run build` ✅ 通过
- `npm run test` ✅ 133 tests passed (包括新增的 14 个 NewsPage 测试)
- `go test ./...` ✅ 全部通过（包括新增的 api/news 和 market/news 测试）

---

### 2025-11-29 17:05:00 CST

**变更摘要**: 修复"重启系统后前端未显示 News 页面"问题 —— 解决 TypeScript 编译错误并确保 News 相关文件被正确提交到 git。

**设计思路**:

- **问题分析**: 用户报告重启系统后 News 页面不显示。经调查发现两个问题：1) `NewsPage.tsx` 中有未使用的 `React` 导入，导致 TypeScript 编译失败（`error TS6133`）；2) 所有 News 相关文件（`api/news_handler.go`、`market/news/`、`web/src/pages/NewsPage.tsx`）都是 `untracked` 状态，从未被 git 提交。
- **修复方案**: 移除 `NewsPage.tsx` 中多余的 `React` 导入（React 17+ 不需要显式导入），确保前端能够正确编译。

**修改意图**:

- 确保 News 功能在前端构建时不再因 TypeScript 错误而失败。
- 提醒将 News 相关文件提交到 git（已添加到暂存区），避免未来版本丢失这些功能文件。

**关键变更文件**:

- `web/src/pages/NewsPage.tsx`: 移除未使用的 `React` 导入，修复 TS6133 错误

**待提交文件（已 staged）**:

- `api/news_handler.go`: 新闻 API handler
- `market/news/`: 新闻服务包（client.go, service.go, types.go, client_test.go）
- `web/src/pages/NewsPage.tsx`: 新闻页面组件

---

### 2025-11-29 16:20:00 CST

**变更摘要**: 解决并预防"清空数据库后仍提示邮箱已被注册"的问题 —— 增强数据库重置的可观测性、在注册流程中添加 dev-mode 调试日志、增加可复现的集成测试（删除->注册、WAL checkpoint），并在启动脚本文档中加入"安全重置数据库"步骤。

**设计思路**:

- 保障生产安全：只在 dev-mode 输出额外诊断日志（不记录敏感字段），确保修复不会泄露敏感信息或改变生产语义。
- 可复现 + 可测性：把真实场景（删除用户、WAL 未 checkpoint）变为自动化集成测试，防止未来回归。
- 可观测性优先：在启动时打印实际数据库路径及 WAL/SHM 状态，减少运维与开发人员在重置数据库时的误判。

**修改意图**:

- 帮助开发者与用户安全且确定性地重置数据库（停止服务、删除 main + wal + shm），避免因 WAL 或运行中的进程导致的数据残留。
- 通过测试与日志，能快速定位“邮箱已被注册”类疑难场景并提供可操作的修复建议。
- 在不影响生产安全性的前提下，提升开发和调试效率，减少重复工单与误操作。

**工作区变更快照（staged/unstaged/untracked）**:
- 修改（unstaged/working tree）: `.github/memory.md`, `START_SCRIPTS_README.md`, `api/server.go`, `api/server_integration_test.go`, `config/test_rsa_key.pem.pub`, `main.go`, `web/src/components/HeaderBar.tsx`, `web/src/layouts/MainLayout.tsx`, `web/src/routes/index.tsx`, `web/src/types.ts`
- 新增/未跟踪 (in-progress feature work): `api/news_handler.go`, `market/news/`, `web/src/pages/NewsPage.tsx`, `screenshots/dev_mode_mcp_test.png`

---

### 2025-11-29 16:00:00 CST

**变更摘要**: 修复"邮箱已被注册"问题的调试与文档改进，增加数据库重置指南与注册流程的可观测性。

**设计思路**:

- **问题分析**: 用户报告在清空数据库后仍提示"邮箱已被注册"。经调查确认问题根因不是代码 bug，而是用户可能：1) 未正确删除 WAL 文件 (`config.db-wal`, `config.db-shm`)；2) 服务仍在运行时删除数据库文件；3) Docker 环境中清理了宿主机文件但容器使用的是挂载卷中的不同文件。
- **调试增强**: 在 `main.go` 启动时打印数据库绝对路径和 WAL 文件状态，帮助开发者快速定位实际使用的数据库文件；在 `handleRegister` 中添加 dev-mode 专属日志，记录邮箱查询结果（仅记录用户 ID 和错误类型，不暴露敏感信息）。
- **文档补充**: 在 `START_SCRIPTS_README.md` 新增"安全重置数据库"章节，详细说明 WAL 模式的特性、正确的重置步骤（本地/Docker）、常见问题解答。

**修改意图**:

- **降低用户困惑**: 提供明确的数据库重置指南，避免用户因不了解 SQLite WAL 机制而陷入困境。
- **提高可观测性**: 通过启动日志和 dev-mode 调试日志，使开发者能快速定位数据库和注册流程的问题。
- **测试覆盖**: 新增两个集成测试 (`TestRegisterAfterUserDeletion`, `TestRegisterWithWALCheckpoint`)，验证删除用户后重新注册的行为符合预期。

**关键变更文件**:

- `main.go`: 添加数据库路径和 WAL 文件状态日志
- `api/server.go`: 在 `handleRegister` 添加 dev-mode 调试日志
- `api/server_integration_test.go`: 新增注册/删除/重新注册场景的集成测试
- `START_SCRIPTS_README.md`: 新增"安全重置数据库"章节

---

### 2025-11-29 15:42:00 CST

**变更摘要**: 集成 NewsNow 新闻聚合功能。实现了后端 `market/news` 包以抓取 Hacker News 和 CryptoPanic 数据，并提供 `/api/news` 接口；前端新增 `NewsPage.tsx` 页面，并在导航栏添加入口。

**设计思路**:

- **模块化设计**: 创建独立的 `market/news` 包，定义 `NewsSource` 接口，方便未来扩展更多新闻源（如 Twitter, Bloomberg）。
- **缓存策略**: 在 `NewsService` 中实现内存缓存（TTL 5分钟），避免频繁请求外部 API 导致限流或延迟，同时保证数据相对实时。
- **用户体验一致性**: 前端 `NewsPage` 沿用现有的 Glassmorphism 设计风格和 Tailwind CSS 组件，确保与整体应用视觉风格统一。

**修改意图**:

- **提供实时市场情报**: 满足用户对实时新闻聚合的需求，帮助交易员在应用内直接获取关键市场信息。
- **增强应用粘性**: 通过集成新闻功能，增加用户在应用内的停留时间，使其成为更全面的交易辅助工具。
- **验证开发模式**: 通过在 `--dev` 模式下开发和验证，进一步确认了本地开发环境的便捷性和可靠性。

---

### 2025-11-29 03:01:00 CST

**变更摘要**: 修复 `start_local.sh` 意外进入开发模式的问题，以及 `BacktestPage` 在开发模式下无限加载的 Bug；验证了前端测试套件的稳定性。

**设计思路**:

- **脚本健壮性**: 在 `start_local.sh` 启动函数入口显式初始化 `NOFX_DEV_MODE=false`，确保脚本参数 (`--dev`) 的优先级高于残留的环境变量，防止意外进入测试模式。
- **开发模式兼容性**: 修改 `BacktestPage.tsx` 的数据加载逻辑，将 `useEffect` 依赖从 `token` 改为 `user`。在 Dev Mode 下 `token` 为 null 但 `user` 对象存在，此修改确保了回测列表在两种模式下都能正常加载。
- **多维验证**: 结合了脚本逻辑测试、前端单元测试 (`npm test` 全通过) 和 Chrome DevTools MCP 的端到端导航测试，确保修复的有效性和系统的整体稳定性。

**修改意图**:

- 解决开发者反馈的启动脚本行为异常问题，避免误操作导致的环境混乱。
- 修复前端页面在开发模式下的可用性问题，确保开发体验流畅。
- 确认前端代码库在近期变更后的健康状态，为后续功能开发夯实基础。

---

### 2025-11-29 00:55:00 CST

**变更摘要**: 修复 market 包中由真实 Binance 返回的错误对象导致的测试不稳定性 — 使 API client 在遇到错误对象或非数组响应时返回可断言的友好错误；同时新增 deterministic 单元测试（httptest 模拟）并让外部集成测试默认跳过（需设置 LIVE_TESTS=1 才运行）。

**设计思路**:

- 在不改变默认行为的前提下，增添最小化、可注入的测试点：新增 `NewAPIClientWithBaseURL(baseURL, client)` 来支持测试注入 base URL 或自定义 HTTP client（原有 `NewAPIClient()` 行为不变）。
- 在解析 Klines/Price/ExchangeInfo 时优先检查 HTTP 状态；当 body 无法解析成预期数组时，尝试解析为 Binance 的错误对象 `{code,msg}` 并返回结构化错误，避免未捕获的 unmarshal panic；解析失败时保留原始 body 作为诊断信息。
- 将外部集成测试默认跳过（通过 `LIVE_TESTS=1` 开启），并把关键单元测试改为使用 `httptest.NewServer` 模拟成功/错误场景，保证 CI 的可重复性与确定性。

**修改意图**:

- 消除 CI 随机失败（例如受限地区返回“Service unavailable”或 HTML 错误页面）和 json unmarshal panic，确保 `go test ./...` 在 PR 检查中安全可重复运行。
- 提供更明确的错误上下文（code/msg/body），便于快速定位外部 API 行为变更或网络策略问题。
- 保持向后兼容及最小侵入式修改：只改进错误处理与可测试性，不改变公开 API 语义或现有生产代码预期。

**关键变更文件（工作区快照）**:

- 已修改 (unstaged): `market/api_client.go`, `market/api_integration_test.go`
- 新增未跟踪 (untracked): `market/api_client_test.go`, `screenshots/dev_mode_mcp_test.png`

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
>
> - 未暂存(unstaged): `.github/copilot-instructions.md`, `config/test_rsa_key.pem.pub`
> - 已暂存(staged): 目前无已暂存文件（所有主要变更已经被 commit）
> - 未跟踪(untracked): `.github/prompts/memory.prompt.md`, `docs/merge-plan-ares-to-dev.md`

*附注*: 我已在 `cherry-pick/safe-upstream` 分支上将 `e4bfad38` 的修复应用并运行本地测试（`go test ./trader/...` 与全套 `go test ./...`），所有测试通过。若需要，将该 memory 条目合并到 `dev` 或作为 PR 的说明补充提交。

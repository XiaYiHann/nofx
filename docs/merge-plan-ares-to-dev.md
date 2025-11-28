# 合并计划：origin/ares → dev

> 生成日期: 2025-11-28  
> 分析基础: origin/ares (e4bfad38) vs origin/dev (b5c239a4)

---

## 📊 一、差异与冲突统计

### 1.1 分支概览

| 指标 | 数值 |
|------|------|
| ares 领先 dev 的提交数 | **2** |
| dev 领先 ares 的提交数 | **48** |
| 总变更文件数 | **237** |
| 总行变更 | +57,162 / -7,581 |
| 冲突文件数 | **45** |
| Add/Add 冲突数 | **13** |
| 内容冲突数 | **32** |

### 1.2 ares 分支独有提交

| 提交 Hash | 提交信息 | 风险等级 |
|-----------|----------|----------|
| `e4bfad38` | **fix 止盈止损单未被撤消的反向单** | 🔴 高风险 (交易逻辑修复) |
| `5861f5d6` | ares branch | 🟡 中风险 (分支初始化) |

> **关键发现**: ares 分支的核心价值是 `e4bfad38` 提交，修复了止盈止损单在反向平仓后未被撤销的 bug。这是一个**必须合并**的交易逻辑修复。

---

## 🎯 二、冲突文件清单（按优先级分类）

### 2.1 🔴 P0 - 最高优先级（交易/数据库核心）

| 文件 | 冲突类型 | 风险等级 | 推荐策略 |
|------|----------|----------|----------|
| `trader/auto_trader.go` | 内容冲突 | 🔴 极高 | **Cherry-pick ares 修复 + 保留 dev 新功能** |
| `trader/interface.go` | 自动合并 | 🔴 高 | 接受 ares 新增的 `GetOpenOrders` 接口 |
| `trader/binance_futures.go` | 自动合并 | 🔴 高 | 接受 ares 新增实现 |
| `trader/hyperliquid_trader.go` | 自动合并 | 🔴 高 | 接受 ares 新增实现 |
| `trader/aster_trader.go` | 自动合并 | 🔴 高 | 接受 ares 新增实现 |
| `manager/trader_manager.go` | 内容冲突 | 🔴 高 | 保留 dev 为主，审查差异 |
| `config/database.go` | 内容冲突 | 🔴 高 | **保留 dev** (含 backtest/kline 表结构) |
| `config/database_test.go` | 内容冲突 | 🔴 高 | 保留 dev |

### 2.2 🟠 P1 - 高优先级（API/MCP 核心）

| 文件 | 冲突类型 | 风险等级 | 推荐策略 |
|------|----------|----------|----------|
| `api/server.go` | 内容冲突 | 🟠 高 | **保留 dev** (含策略管理、backtest API) |
| `mcp/client.go` | 内容冲突 | 🟠 高 | 保留 dev (含 builder 模式重构) |
| `decision/engine.go` | 内容冲突 | 🟠 中高 | 保留 dev |
| `market/data.go` | 内容冲突 | 🟠 中 | 保留 dev |
| `market/monitor.go` | 内容冲突 | 🟠 中 | 保留 dev |

### 2.3 🟡 P2 - 中优先级（依赖/配置）

| 文件 | 冲突类型 | 风险等级 | 推荐策略 |
|------|----------|----------|----------|
| `go.mod` | 内容冲突 | 🟡 中 | **保留 dev** + `go mod tidy` |
| `go.sum` | 自动合并 | 🟡 中 | 运行 `go mod tidy` 重新生成 |
| `web/package-lock.json` | 内容冲突 | 🟡 中 | 保留 dev + `npm ci` 重新生成 |
| `docker-compose.yml` | 内容冲突 | 🟡 低 | 保留 dev |
| `start.sh` | 内容冲突 | 🟡 低 | 保留 dev |

### 2.4 🟢 P3 - 低优先级（前端/文档）

| 文件 | 冲突类型 | 推荐策略 |
|------|----------|----------|
| `web/src/App.tsx` | 内容冲突 | 保留 dev |
| `web/src/components/*.tsx` (8个) | 内容/AA冲突 | **保留 dev** |
| `web/src/contexts/AuthContext.tsx` | 内容冲突 | 保留 dev |
| `web/src/lib/api.ts` | 内容冲突 | 保留 dev |
| `web/src/lib/httpClient.ts` | 内容冲突 | 保留 dev |
| `web/src/stores/*.ts` (1个) | AA冲突 | 保留 dev |
| `web/src/layouts/*.tsx` (2个) | AA冲突 | 保留 dev |
| `web/src/routes/index.tsx` | AA冲突 | 保留 dev |
| `web/src/pages/*.tsx` (2个) | 内容/AA冲突 | 保留 dev |
| 文档文件 (docs/, README) | 多种冲突 | 保留 dev 为主 |
| `.env.example` | 内容冲突 | 保留 dev |
| `CHANGELOG.md` | 内容冲突 | 手动合并两边记录 |

### 2.5 ⚠️ 特殊关注：Add/Add 冲突

以下文件两个分支都新增了同路径文件，需要模块 owner 决定保留哪个版本：

| 文件 | 推荐处理 |
|------|----------|
| `Makefile` | 保留 dev (功能更完整) |
| `README.ja.md` | 保留 dev |
| `decision/prompt_manager_test.go` | 保留 dev |
| `trader/hyperliquid_trader_race_test.go` | 保留 dev |
| `web/src/components/HeaderBar.tsx` | 保留 dev |
| `web/src/components/WebCryptoEnvironmentCheck.tsx` | 保留 dev |
| `web/src/components/traders/ExchangeConfigModal.tsx` | 保留 dev |
| `web/src/hooks/useTraderActions.ts` | 保留 dev |
| `web/src/layouts/AuthLayout.tsx` | 保留 dev |
| `web/src/layouts/MainLayout.tsx` | 保留 dev |
| `web/src/pages/TraderDashboard.tsx` | 保留 dev |
| `web/src/routes/index.tsx` | 保留 dev |
| `web/src/stores/tradersConfigStore.ts` | 保留 dev |

### 2.6 ⚠️ 特殊关注：nofx-web-dev 目录

ares 分支引入了一个完整的 `nofx-web-dev/` 目录（独立前端项目），包含：
- 独立的 `package.json`, `pnpm-lock.yaml`
- 完整的 src/ 目录
- specs/ 设计规范文档

**建议**：
1. 在合并前与前端负责人确认是否需要保留此目录
2. 如需保留，应作为独立目录存在，不与 `web/` 合并
3. 如不需要，在合并后删除此目录

---

## 🔧 三、推荐合并策略

### 3.1 核心策略：Cherry-pick + 保留 dev

由于：
- ares 仅有 2 个提交，核心价值是 `e4bfad38` 的交易逻辑修复
- dev 领先 48 个提交，包含大量新功能和重构
- 直接合并会产生 45 个冲突，风险较高

**推荐方案**：**Cherry-pick ares 的修复到 dev**，而非完整合并

### 3.2 需要从 ares 提取的变更

`e4bfad38` 提交包含以下关键修改：

| 文件 | 变更内容 |
|------|----------|
| `trader/interface.go` | +4 行：新增 `GetOpenOrders(symbol string)` 接口 |
| `trader/auto_trader.go` | +40 行：新增清理残留挂单逻辑 |
| `trader/binance_futures.go` | +52 行：实现 `GetOpenOrders` |
| `trader/hyperliquid_trader.go` | +43 行：实现 `GetOpenOrders` |
| `trader/aster_trader.go` | +21 行：实现 `GetOpenOrders` |
| `trader/auto_trader_test.go` | +4 行：测试更新 |

---

## 🧪 四、测试矩阵

### 4.1 必须运行的测试

#### 后端测试
```bash
# 基础测试
go test ./... -cover -v

# 静态检查
go vet ./...
golangci-lint run

# 关键模块专项测试
go test -v ./trader/...
go test -v ./manager/...
go test -v ./config/...
go test -v ./decision/...
go test -v ./api/...
```

#### 前端测试
```bash
cd web

# 依赖安装
npm ci

# Lint 检查
npm run lint

# 单元测试
npm run test

# 构建验证
npm run build
```

#### 集成/回测测试
```bash
# 回测全流程测试
./test_backtest_full.sh

# Binance 测试网验证
./test_binance_testnet.sh
```

### 4.2 特殊验证项

| 验证项 | 测试方法 | 通过标准 |
|--------|----------|----------|
| 止盈止损单清理 | 模拟平仓后检查挂单 | 挂单被正确取消 |
| GetOpenOrders 接口 | 各交易所 API 调用 | 返回正确订单列表 |
| 数据库迁移 | SQLite schema 验证 | 所有表正常创建 |
| API 兼容性 | 前后端联调 | 无 breaking change |

---

## 📋 五、分步操作命令

### 5.1 方案 A：Cherry-pick（推荐）

```bash
# 1. 确保在 dev 分支
git checkout dev
git pull origin dev

# 2. 创建合并分支
git checkout -b fix/ares-tp-sl-cleanup

# 3. Cherry-pick 关键修复
git cherry-pick e4bfad38

# 4. 如有冲突，手动解决后继续
git add .
git cherry-pick --continue

# 5. 运行测试
go test ./trader/... -v
go test ./... -cover

# 6. 推送并创建 PR
git push origin fix/ares-tp-sl-cleanup
```

### 5.2 方案 B：完整合并（高风险）

```bash
# 1. 创建隔离工作区（dry-run）
git fetch origin
git checkout -b merge/ares-into-dev origin/dev

# 2. 执行合并（不提交）
git merge --no-commit --no-ff origin/ares

# 3. 查看冲突状态
git status --porcelain | grep "^UU\|^AA"

# 4. 按优先级解决冲突
# P0: trader/, manager/, config/database.go
# P1: api/server.go, mcp/, decision/
# P2: go.mod, web/package-lock.json
# P3: 前端组件, 文档

# 5. 对于大多数文件，保留 dev 版本
git checkout --ours <file>  # 保留 dev (当前分支)
git checkout --theirs <file>  # 保留 ares (合并来源)

# 6. 特殊处理 trader/ 文件
# 需要手动合并 ares 的 GetOpenOrders 实现到 dev 版本

# 7. 验证合并结果
go mod tidy
go test ./...
cd web && npm ci && npm run lint && npm run test && npm run build

# 8. 如果验证失败，回滚
git merge --abort
```

### 5.3 Dry-run 安全指令

```bash
# 创建完全隔离的 worktree 进行合并测试
git worktree add /tmp/merge-ares-test origin/dev
cd /tmp/merge-ares-test

# 执行合并测试
git merge origin/ares --no-commit || true

# 记录冲突
git status --porcelain > /tmp/conflict-list.txt

# 清理
cd -
git worktree remove /tmp/merge-ares-test
```

---

## 🔄 六、回滚与验证步骤

### 6.1 回滚准备

合并前创建备份点：
```bash
# 记录当前 dev HEAD
git rev-parse HEAD > /tmp/dev-backup-hash.txt

# 或创建备份 tag
git tag backup/pre-ares-merge-$(date +%Y%m%d%H%M%S)
```

### 6.2 回滚操作

```bash
# 方式1：使用记录的 hash
git reset --hard $(cat /tmp/dev-backup-hash.txt)

# 方式2：使用备份 tag
git reset --hard backup/pre-ares-merge-YYYYMMDDHHMMSS

# 方式3：如果已推送，创建 revert commit
git revert -m 1 <merge-commit-hash>
```

### 6.3 验证清单

- [ ] `go test ./... -cover` 全部通过
- [ ] `go vet ./...` 无报错
- [ ] `golangci-lint run` 无新警告
- [ ] `npm ci && npm run lint && npm run test` 通过
- [ ] `npm run build` 成功
- [ ] `./test_backtest_full.sh` 通过
- [ ] 本地启动 `./start_local.sh start` 成功
- [ ] 前后端联调 API 正常

---

## 📝 七、PR 模板

```markdown
## PR: Merge ares branch fixes into dev

### 变更摘要

从 `origin/ares` 分支合并关键修复到 `dev`：

**核心修复**：
- 🐛 修复止盈止损单在反向平仓后未被撤销的 bug (#e4bfad38)
- ✨ 新增 `GetOpenOrders` 接口支持各交易所

**影响范围**：
- `trader/interface.go` - 新增接口定义
- `trader/auto_trader.go` - 清理残留挂单逻辑
- `trader/binance_futures.go` - Binance 实现
- `trader/hyperliquid_trader.go` - Hyperliquid 实现
- `trader/aster_trader.go` - Aster 实现

### 风险评估

| 模块 | 风险等级 | 说明 |
|------|----------|------|
| 交易逻辑 | 🔴 高 | 涉及订单管理，需要充分测试 |
| 接口变更 | 🟡 中 | 新增接口，无 breaking change |

### 测试矩阵

- [ ] `go test ./trader/... -v` ✅
- [ ] `go test ./... -cover` ✅ 
- [ ] `./test_backtest_full.sh` ✅
- [ ] 前端构建 `npm run build` ✅
- [ ] 本地联调验证 ✅

### 校验清单

- [ ] 代码符合 gofmt/golangci-lint 规范
- [ ] 新增/修改的方法有单元测试覆盖
- [ ] 无数据库 schema 变更（或已提供迁移脚本）
- [ ] 无 API breaking change
- [ ] 文档已更新（如需要）

### 必须 Reviewer

- [ ] @交易模块-owner (trader/ 变更)
- [ ] @后端-lead (核心逻辑审查)

### 部署注意事项

⚠️ 合并后需要重启交易服务以应用新的订单清理逻辑
```

---

## 🎯 八、总结与建议

### 8.1 核心结论

1. **ares 分支核心价值**：`e4bfad38` 的止盈止损单清理修复是关键 bug fix
2. **合并风险**：直接合并产生 45 个冲突，dev 有大量新功能，风险较高
3. **推荐策略**：**Cherry-pick** 而非完整合并

### 8.2 执行建议

| 步骤 | 操作 | 负责人 |
|------|------|--------|
| 1 | Cherry-pick `e4bfad38` 到 dev | 开发者 |
| 2 | 解决 `trader/auto_trader.go` 冲突 | 交易模块 owner |
| 3 | 运行完整测试套件 | CI/开发者 |
| 4 | 代码审查 | 交易模块 owner + 后端 lead |
| 5 | 合并到 dev | 有合并权限者 |
| 6 | 验证测试网/回测环境 | QA/开发者 |

### 8.3 风险缓解

- ✅ 优先使用 cherry-pick 降低合并复杂度
- ✅ 必须有交易模块 owner 审查签字
- ✅ 合并前在 backtest 环境验证订单清理逻辑
- ✅ 保留回滚点，出问题可快速回滚

# NOFX 启动脚本说明

## 📁 文件说明

本项目提供了三种启动方式:

### 1. `start_local.sh` - 本地开发模式 ✨推荐开发使用
**不使用 Docker，直接在本地运行**

```bash
# 开发模式启动（前端热重载 + 后端）
./start_local.sh start --dev

# 生产模式启动（构建前端 + 后端）
./start_local.sh start

# 查看状态
./start_local.sh status

# 查看日志
./start_local.sh logs

# 停止服务
./start_local.sh stop

# 重启服务
./start_local.sh restart --dev
```

**特点:**
- ✅ 前端热重载，修改即时生效
- ✅ 直接使用本地 Go 和 Node.js
- ✅ 启动快速
- ✅ 适合开发调试
- ✅ 自动创建包含 Paper Trading 的数据库

**要求:**
- Node.js 18+
- Go 1.21+
- npm

---

### 2. `start_docker.sh` - Docker 部署模式 🐳推荐生产使用
**使用 Docker Compose 部署**

```bash
# 启动服务
./start_docker.sh start

# 查看状态
./start_docker.sh status

# 查看所有日志
./start_docker.sh logs

# 查看后端日志
./start_docker.sh logs nofx

# 停止服务
./start_docker.sh stop

# 重新构建镜像
./start_docker.sh build

# 完全重建（包含删除旧数据库）
./start_docker.sh rebuild-fresh
```

**特点:**
- ✅ 隔离的运行环境
- ✅ 一键部署
- ✅ 适合生产环境
- ✅ 容器化管理
- ✅ 自动创建包含 Paper Trading 的数据库

**要求:**
- Docker
- Docker Compose

#### 🚦 启动后自动测试（Docker 部署）
- 默认关闭，使用 `./start_docker.sh start --with-tests` 或设置环境变量 `NOFX_AUTO_TEST_AFTER_START=true` 即可在容器完全就绪后自动执行 `go test ./config/... ./api/...`。
- 可通过 `--test-packages "./config/... ./api/... ./manager/..."`、`--test-wait 180`、`--go-flags "-count=1 -timeout 8m"` 精细控制测试范围与超时。
- `--allow-test-fail` 仅报警不中断（适合宽松环境或演示），默认会在测试失败时退出并提示使用 `scripts/docker/run_tests_after_start.sh` 复现。
- 同样的参数/环境变量也适用于 `./start.sh start --with-tests`，便于兼容旧脚本或 CI。
- CI 示例：`NOFX_AUTO_TEST_AFTER_START=true ./start_docker.sh start`，或显式 `./start_docker.sh start --with-tests --allow-test-fail`。

**⚠️ 重要提示 - Paper Trading 显示问题:**

如果在 Docker 模式下看不到 Paper Trading 交易所，有两种解决方案:

**方案1: 删除数据库重启（快速）**
```bash
./start_docker.sh stop
rm config.db
./start_docker.sh start
```

**方案2: 完全重建（彻底）**
```bash
./start_docker.sh rebuild-fresh
```

**原因:** Docker 通过 volume 挂载数据库文件。如果 `config.db` 是用旧代码创建的（没有 paper_trading），即使更新代码并重建镜像，数据库文件也不会更新。删除数据库文件后重启，会用新代码创建包含 Paper Trading 的数据库。

---

### 3. `start.sh` - 原始启动脚本（兼容性保留）
**功能最全的脚本，包含更多高级功能**

```bash
# 开发模式
./start.sh start --dev

# 生产模式
./start.sh start

# 查看帮助
./start.sh help
```

同样可以添加 `--with-tests`、`--allow-test-fail` 等旗标，或通过 `NOFX_AUTO_TEST_AFTER_START=true` 控制启动后的自动测试流程。

---

## 🔄 快速选择指南

| 场景 | 推荐脚本 | 命令 |
|------|---------|------|
| 本地开发 | `start_local.sh` | `./start_local.sh start --dev` |
| 测试部署 | `start_docker.sh` | `./start_docker.sh start` |
| 生产部署 | `start_docker.sh` | `./start_docker.sh start` |
| 首次安装 | `setup.sh` (Linux) | `./setup.sh` |

---

## 🎯 Paper Trading 使用说明

**Paper Trading (模拟交易)** 是基于 Binance Testnet 的模拟交易功能，所有三个启动脚本都支持。

### 数据库初始化

首次启动时，系统会自动创建数据库并包含以下交易所:
1. ✅ Binance Futures (真实交易)
2. ✅ Hyperliquid (去中心化)
3. ✅ Aster DEX (去中心化)
4. ✅ **Paper Trading (Binance Testnet)** ⭐模拟交易

### 配置 Paper Trading

1. 启动服务
2. 登录 Web 界面
3. 进入「交易所配置」
4. 选择「Paper Trading (Binance Testnet)」
5. 输入 Binance Testnet API 密钥:
   - 获取地址: https://testnet.binancefuture.com
   - 使用测试资金，无需真实资金
   - 完全模拟真实交易环境

### 验证 Paper Trading 存在

**方法1: 通过 Web 界面**
```
登录 → 交易所配置 → 查看列表
应该看到 4 个交易所（包括 Paper Trading）
```

**方法2: 通过命令行（需要安装 sqlite3）**
```bash
sqlite3 config.db "SELECT id, name FROM exchanges WHERE user_id='default';"
```

应该输出:
```
aster|Aster DEX
binance|Binance Futures
hyperliquid|Hyperliquid
paper_trading|Paper Trading (Binance Testnet)
```

### 故障排除

**问题: 看不到 Paper Trading**

可能原因: 数据库是用旧版本代码创建的

解决方案:

**本地模式:**
```bash
./start_local.sh stop
rm config.db
./start_local.sh start --dev
```

**Docker 模式:**
```bash
./start_docker.sh rebuild-fresh
# 或者
rm config.db && ./start_docker.sh restart
```

**手动添加（不删除数据库）:**
```bash
# 进入容器（Docker 模式）
docker exec -it nofx-trading sh
apk add sqlite
sqlite3 /app/config.db "INSERT OR IGNORE INTO exchanges (id, user_id, name, type, enabled) VALUES ('paper_trading', 'default', 'Paper Trading (Binance Testnet)', 'paper_trading', 0);"
exit

# 或直接在宿主机（如果安装了 sqlite3）
sqlite3 config.db "INSERT OR IGNORE INTO exchanges (id, user_id, name, type, enabled) VALUES ('paper_trading', 'default', 'Paper Trading (Binance Testnet)', 'paper_trading', 0);"
```

---

## 🔄 安全重置数据库

### ⚠️ 重要警告
SQLite 使用 WAL（Write-Ahead Logging）模式来提高性能和数据安全性。重置数据库时，**必须同时删除主文件和 WAL 相关文件**，否则可能导致：
- 旧数据在服务重启后恢复
- 注册时提示"邮箱已被注册"但数据库中实际没有该用户
- 数据不一致问题

### 📋 数据库文件组成
```
config.db      # 主数据库文件
config.db-wal  # WAL 日志文件（Write-Ahead Log）
config.db-shm  # 共享内存文件（Shared Memory）
```

### 🛑 重置数据库步骤

#### 本地开发模式
```bash
# 步骤 1: 停止服务（非常重要！）
./start_local.sh stop

# 步骤 2: 确认服务已停止
ps aux | grep nofx | grep -v grep  # 应该没有输出

# 步骤 3: 删除所有数据库相关文件
rm -f config.db config.db-wal config.db-shm

# 步骤 4: 重新启动服务（会创建新的数据库）
./start_local.sh start --dev
```

#### Docker 部署模式
```bash
# 步骤 1: 停止容器
./start_docker.sh stop

# 步骤 2: 删除宿主机上的数据库文件
rm -f config.db config.db-wal config.db-shm

# 步骤 3: 重新启动
./start_docker.sh start
```

### ⚡ 快速重置（危险操作，会丢失所有数据）
```bash
# 本地模式 - 一键重置
./start_local.sh stop && rm -f config.db* && ./start_local.sh start --dev

# Docker 模式 - 一键重置
./start_docker.sh stop && rm -f config.db* && ./start_docker.sh start
```

### 🔧 仅删除用户数据（保留配置）
如果只想删除用户而不重置整个数据库：
```bash
# 步骤 1: 停止服务
./start_local.sh stop

# 步骤 2: 使用 sqlite3 删除用户（需要安装 sqlite3）
sqlite3 config.db "DELETE FROM users; PRAGMA wal_checkpoint(TRUNCATE);"

# 步骤 3: 删除 WAL 文件（可选，但推荐）
rm -f config.db-wal config.db-shm

# 步骤 4: 重新启动服务
./start_local.sh start --dev
```

### ❓ 常见问题

**Q: 删除了数据库文件，但注册仍然提示"邮箱已被注册"？**

A: 可能的原因：
1. **服务未停止**：在运行的服务进程中，数据仍在内存/WAL 中。请先停止服务。
2. **未删除 WAL 文件**：确保同时删除 `config.db`、`config.db-wal` 和 `config.db-shm`。
3. **Docker 卷问题**：Docker 中数据库可能挂载在不同位置。使用 `docker exec` 检查容器内的实际文件。

**Q: 如何确认服务使用的是哪个数据库文件？**

A: 启动日志会显示数据库路径：
```
📋 初始化配置数据库: /Users/xxx/Code/nofx/config.db
📋 发现 WAL 文件: /Users/xxx/Code/nofx/config.db-wal (2.50 KB)
```

**Q: 如何在不停止服务的情况下检查用户表？**

A: 外部 sqlite3 连接可以读取（但修改可能不会立即被运行中的服务看到）：
```bash
sqlite3 config.db "SELECT email, created_at FROM users;"
```

---

## 📝 注意事项

1. **首次启动**: 所有脚本都会自动设置加密环境（RSA密钥 + 数据加密密钥）
2. **数据库备份**: 每次启动前会自动备份数据库到 `database_backups/`
3. **端口配置**: 
   - 前端: 3000 (可在 .env 中修改 NOFX_FRONTEND_PORT)
   - 后端: 8080 (可在 .env 中修改 NOFX_BACKEND_PORT)
4. **日志位置**:
   - 本地模式: `nofx.log`, `frontend.log`
   - Docker 模式: 通过 `docker compose logs` 查看

---

## 🆘 获取帮助

```bash
# 本地模式
./start_local.sh

# Docker 模式
./start_docker.sh help

# 原始脚本
./start.sh help
```

---

## 🔗 相关文档

- [PAPER_TRADING_FIXED.md](./PAPER_TRADING_FIXED.md) - Paper Trading 修复说明
- [DEVELOPMENT_MODE.md](./DEVELOPMENT_MODE.md) - 开发模式文档
- [docs/DOCKER_BUILD.md](./docs/DOCKER_BUILD.md) - Docker 构建指南
- [docker-compose.yml](./docker-compose.yml) - Docker 配置
- [README.md](./README.md) - 项目主文档

---

## 🐳 Docker 高级操作

除了基本的启动脚本，项目还提供了专用的 Docker 管理脚本：

### 重建镜像

```bash
# 完整重建（运行测试 + 构建 + 标记）
./scripts/docker/rebuild.sh

# 跳过测试快速重建（仅开发时使用）
./scripts/docker/rebuild.sh --skip-tests

# 包含集成测试（需要 LLM_API_KEY）
./scripts/docker/rebuild.sh --with-integration
```

### 清理旧镜像

```bash
# 预览要清理的内容
./scripts/docker/cleanup.sh --dry-run

# 清理旧镜像，保留最近 3 个
./scripts/docker/cleanup.sh

# 完全清理
./scripts/docker/cleanup.sh --all --force
```

### 健康检查

```bash
# 检查所有服务
./scripts/docker/healthcheck.sh

# 仅检查后端
./scripts/docker/healthcheck.sh --backend-only

# 等待启动（最多 120 秒）
./scripts/docker/healthcheck.sh --wait 120
```

详细说明请参考 [docs/DOCKER_BUILD.md](./docs/DOCKER_BUILD.md)。


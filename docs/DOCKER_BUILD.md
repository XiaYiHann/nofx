# NOFX Docker 构建指南

本文档描述了如何安全、可复现地构建和部署 NOFX Docker 镜像。

## 📁 脚本说明

| 脚本 | 用途 |
|------|------|
| `scripts/docker/rebuild.sh` | 完整重建工作流：测试 → 停止 → 构建 → 标记 |
| `scripts/docker/cleanup.sh` | 清理旧镜像/容器（支持 dry-run） |
| `scripts/docker/healthcheck.sh` | 验证服务健康状态 |

## 🚀 快速开始

### 标准重建流程

```bash
# 1. 完整重建（运行单元测试 + 构建）
./scripts/docker/rebuild.sh

# 2. 启动服务
docker compose up -d

# 3. 验证健康状态
./scripts/docker/healthcheck.sh
```

### 镜像标记策略

镜像使用以下格式标记：
```
nofx/backend:<branch>-<commit>-<date>
nofx/frontend:<branch>-<commit>-<date>
```

例如：
```
nofx/backend:cherry-pick-safe-upstream-cf11de0b-20251127
nofx/frontend:cherry-pick-safe-upstream-cf11de0b-20251127
```

## 🔒 安全注意事项

### ⚠️ 绝不提交的内容

以下文件/目录 **绝不能** 提交到 Git：

- `.env` - 环境变量（含密钥）
- `secrets/` - RSA 密钥对
- `config.db` - 可能含加密凭证的数据库
- 任何 `*.key`, `*.pem` 私钥文件

### 镜像安全

- Dockerfile 使用多阶段构建，运行时镜像不含构建工具
- 密钥通过 **运行时环境变量** 注入，不写入镜像层
- `secrets/` 目录通过 Docker volume 挂载，不复制进镜像

### 构建时测试门控

| 环境变量 | 作用 | 默认值 |
|---------|------|--------|
| `LLM_API_KEY` | 存在时启用 LLM 集成测试 | 未设置（跳过） |
| `LIVE_TESTS` | 设为 `1` 启用交易所实测 | 未设置（跳过） |

**CI 安全规则**：
- CI 环境 **不应** 设置 `LIVE_TESTS=1`
- `LLM_API_KEY` 应通过 GitHub Secrets 注入，不裸写

## 📋 命令参考

### rebuild.sh 选项

```bash
./scripts/docker/rebuild.sh [OPTIONS]

选项：
  --with-integration    运行集成测试（需要 LLM_API_KEY）
                        ⚠️ 警告：可能消耗 API 配额
  --skip-tests          跳过所有测试（不推荐用于生产）
  --no-cache            不使用 Docker 缓存构建
  --push                构建后推送到 registry
  -h, --help            显示帮助

环境变量：
  IMAGE_PREFIX          镜像名前缀（默认：nofx）
  REGISTRY              Docker registry（默认：无，仅本地）
  GOPROXY               Go 模块代理
  NPM_REGISTRY          NPM registry
```

### cleanup.sh 选项

```bash
./scripts/docker/cleanup.sh [OPTIONS]

选项：
  --dry-run             预览删除内容，不实际删除
  --force               跳过确认提示
  --keep-latest N       保留最近 N 个镜像（默认：3）
  --all                 删除所有 nofx 镜像
  --containers-only     仅删除容器
  --images-only         仅删除镜像
  --prune               同时执行 docker system prune
  -h, --help            显示帮助
```

### healthcheck.sh 选项

```bash
./scripts/docker/healthcheck.sh [OPTIONS]

选项：
  --backend-only        仅检查后端
  --frontend-only       仅检查前端
  --wait N              等待服务启动的秒数（默认：60）
  --backend-port PORT   后端端口（默认从 .env 读取）
  --frontend-port PORT  前端端口（默认从 .env 读取）
  -v, --verbose         详细输出
  -h, --help            显示帮助
```

## 🔄 常用工作流

### 开发迭代

```bash
# 快速重建（跳过测试，仅开发时使用）
./scripts/docker/rebuild.sh --skip-tests

# 启动并查看日志
docker compose up -d && docker compose logs -f
```

### 生产部署

```bash
# 完整测试 + 构建
./scripts/docker/rebuild.sh

# 清理旧镜像（保留最近 3 个）
./scripts/docker/cleanup.sh --force

# 启动并验证
docker compose up -d
./scripts/docker/healthcheck.sh --wait 120
```

### 回滚到之前版本

```bash
# 1. 查看可用镜像
docker images | grep nofx

# 2. 修改 docker-compose.yml 或使用环境变量
# 例如：
export NOFX_BACKEND_IMAGE=nofx/backend:main-abc1234-20251126
export NOFX_FRONTEND_IMAGE=nofx/frontend:main-abc1234-20251126

# 3. 重启服务
docker compose down
docker compose up -d
```

### 完全清理

```bash
# 预览清理
./scripts/docker/cleanup.sh --all --prune --dry-run

# 执行清理
./scripts/docker/cleanup.sh --all --prune --force
```

## 🤖 CI/CD 集成

### GitHub Actions 示例

参见 `.github/workflows/docker-build.yml`：

```yaml
name: Docker Build

on:
  push:
    branches: [main, dev]
  pull_request:
    branches: [main, dev]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      
      - name: Run Unit Tests
        run: go test ./... -short -count=1
        env:
          # 不设置 LLM_API_KEY 和 LIVE_TESTS，跳过集成测试
          CGO_ENABLED: 1
      
      - name: Build Docker Images
        run: ./scripts/docker/rebuild.sh --skip-tests
        # 测试已在上一步运行
      
      - name: Health Check (Smoke Test)
        run: |
          docker compose up -d
          sleep 30
          ./scripts/docker/healthcheck.sh --wait 60
```

### 安全密钥注入

```yaml
# 仅在需要集成测试时使用
- name: Run Integration Tests
  if: github.event_name == 'schedule'  # 仅定时任务
  run: ./scripts/docker/rebuild.sh --with-integration
  env:
    LLM_API_KEY: ${{ secrets.LLM_API_KEY }}
    # 注意：不设置 LIVE_TESTS=1
```

## 🩺 故障排除

### 构建失败：TA-Lib

如果后端构建失败且错误涉及 TA-Lib：
```bash
# 确保使用正确的 Dockerfile
docker build -f docker/Dockerfile.backend .

# 检查 TA-Lib 编译日志
docker build -f docker/Dockerfile.backend . 2>&1 | grep -A5 "ta-lib"
```

### 健康检查失败

```bash
# 检查容器状态
docker compose ps

# 查看后端日志
docker compose logs nofx | tail -50

# 检查端口占用
lsof -i :8080
lsof -i :3000
```

### 镜像占用过多空间

```bash
# 查看镜像大小
docker images | grep nofx

# 清理并保留最新 2 个
./scripts/docker/cleanup.sh --keep-latest 2 --force

# 完整清理
./scripts/docker/cleanup.sh --all --prune --force
```

## 📚 相关文档

- [START_SCRIPTS_README.md](START_SCRIPTS_README.md) - 启动脚本总览
- [LOCAL_DEV.md](LOCAL_DEV.md) - 本地开发指南
- [.env.example](.env.example) - 环境变量模板

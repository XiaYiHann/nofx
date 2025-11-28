#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX AI Trading System - Docker Startup Script
# Usage: ./start.sh [command]
# ═══════════════════════════════════════════════════════════════

set -e

# ------------------------------------------------------------------------
# Color Definitions
# ------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ------------------------------------------------------------------------
# Utility Functions
# ------------------------------------------------------------------------
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

normalize_bool() {
    local value="${1:-false}"
    # Convert to lowercase using tr for compatibility with Bash 3.2 (macOS)
    value=$(echo "$value" | tr '[:upper:]' '[:lower:]')
    case "$value" in
        1|true|yes|on)
            echo "true"
            ;;
        *)
            echo "false"
            ;;
    esac
}

# ------------------------------------------------------------------------
# Post-start test configuration (shared with start_docker.sh)
# ------------------------------------------------------------------------
AUTO_TEST_SCRIPT="./scripts/docker/run_tests_after_start.sh"
AUTO_TEST_DEFAULT_PACKAGES="./config/... ./api/..."
AUTO_TEST_ENABLED="false"
AUTO_TEST_ALLOW_FAIL="false"
AUTO_TEST_PACKAGES="$AUTO_TEST_DEFAULT_PACKAGES"
AUTO_TEST_WAIT="120"
AUTO_TEST_GOFLAGS="-count=1 -timeout 5m"
reset_auto_test_config() {
    AUTO_TEST_PACKAGES="${NOFX_AUTO_TEST_PACKAGES:-$AUTO_TEST_DEFAULT_PACKAGES}"
    AUTO_TEST_WAIT="${NOFX_AUTO_TEST_WAIT:-120}"
    AUTO_TEST_GOFLAGS="${NOFX_AUTO_TEST_GOFLAGS:--count=1 -timeout 5m}"
    AUTO_TEST_ENABLED="$(normalize_bool "${NOFX_AUTO_TEST_AFTER_START:-false}")"
    AUTO_TEST_ALLOW_FAIL="$(normalize_bool "${NOFX_AUTO_TEST_ALLOW_FAIL:-false}")"
}

parse_auto_test_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --with-tests)
                AUTO_TEST_ENABLED="true"
                shift
                ;;
            --no-tests)
                AUTO_TEST_ENABLED="false"
                shift
                ;;
            --allow-test-fail)
                AUTO_TEST_ALLOW_FAIL="true"
                shift
                ;;
            --test-wait)
                AUTO_TEST_WAIT="${2:-$AUTO_TEST_WAIT}"
                shift 2
                ;;
            --test-packages)
                AUTO_TEST_PACKAGES="${2:-$AUTO_TEST_PACKAGES}"
                shift 2
                ;;
            --go-flags)
                AUTO_TEST_GOFLAGS="${2:-$AUTO_TEST_GOFLAGS}"
                shift 2
                ;;
            --dev)
                # 开发模式参数在此处不处理，但需要识别以避免被忽略
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
}

# 解析 --dev 参数并设置 NOFX_DEV_MODE
parse_dev_mode_arg() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dev)
                export NOFX_DEV_MODE=true
                print_warning "🚧 测试模式已启用 (--dev)"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
}

maybe_run_post_start_tests() {
    if [ "$AUTO_TEST_ENABLED" != "true" ]; then
        return
    fi

    if [ ! -x "$AUTO_TEST_SCRIPT" ]; then
        print_warning "未找到自动测试脚本 $AUTO_TEST_SCRIPT，跳过部署后测试"
        return
    fi

    local opts=("--wait" "$AUTO_TEST_WAIT" "--packages" "$AUTO_TEST_PACKAGES" "--go-flags" "$AUTO_TEST_GOFLAGS")
    if [ "$AUTO_TEST_ALLOW_FAIL" = "true" ]; then
        opts+=("--allow-test-fail")
    fi

    print_info "🚦 自动测试已启用，等待服务就绪后执行 Go 测试"
    if ! "$AUTO_TEST_SCRIPT" "${opts[@]}"; then
        local rerun_cmd="$AUTO_TEST_SCRIPT --wait $AUTO_TEST_WAIT --packages \"$AUTO_TEST_PACKAGES\" --go-flags \"$AUTO_TEST_GOFLAGS\""
        if [ "$AUTO_TEST_ALLOW_FAIL" = "true" ]; then
            rerun_cmd="$rerun_cmd --allow-test-fail"
        fi
        print_error "部署后自动测试失败"
        print_info "可手动重跑: $rerun_cmd"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# Check Docker Availability
# ------------------------------------------------------------------------
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装！请先安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        print_error "Docker 服务未运行！请启动 Docker Desktop 或 Docker 服务。"
        exit 1
    fi

    # Check for compose (docker compose or docker-compose)
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker-compose"
    else
        print_error "Docker Compose 未安装！"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# Configuration & Secrets Generation
# ------------------------------------------------------------------------
check_config() {
    # 1. .env
    if [ ! -f ".env" ]; then
        print_warning ".env 不存在，从模板复制..."
        if [ -f ".env.example" ]; then
            cp .env.example .env
        else
            cat > .env << EOF
# NOFX AI Trading System - Environment Configuration
NOFX_FRONTEND_PORT=3000
NOFX_BACKEND_PORT=8080
NOFX_TIMEZONE=Asia/Shanghai

# Encryption (Auto-generated)
DATA_ENCRYPTION_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)

# Proxy Configuration (Optional - Uncomment if needed)
HTTP_PROXY=http://host.docker.internal:7890
HTTPS_PROXY=http://host.docker.internal:7890
EOF
        fi
        print_success "✓ 已创建 .env 文件"
    fi

    # 2. config.json
    if [ ! -f "config.json" ]; then
        print_warning "config.json 不存在，创建默认配置..."
        if [ -f "config.json.example" ]; then
            cp config.json.example config.json
        else
            cat > config.json << EOF
{
  "system": {
    "lever_rate": 10,
    "leverage_enabled": true,
    "admin_mode": false
  },
  "models": {},
  "exchanges": {}
}
EOF
        fi
        print_success "✓ 已创建 config.json"
    fi

    # 3. Secrets (RSA Keys)
    if [ ! -d "secrets" ]; then
        mkdir -p secrets
        chmod 700 secrets
    fi

    if [ ! -f "secrets/rsa_key" ] || [ ! -f "secrets/rsa_key.pub" ]; then
        print_warning "RSA密钥对不存在，正在生成..."
        openssl genrsa -out secrets/rsa_key 2048 2>/dev/null
        openssl rsa -in secrets/rsa_key -pubout -out secrets/rsa_key.pub 2>/dev/null
        chmod 600 secrets/rsa_key
        print_success "✓ 已生成 RSA 密钥对"
    fi

    # 4. Database File (Ensure it exists for volume mount)
    if [ ! -f "config.db" ]; then
        print_info "创建空数据库文件..."
        touch config.db
        chmod 600 config.db
    fi
    
    # 5. Beta Codes (Ensure it exists for volume mount)
    if [ ! -f "beta_codes.txt" ]; then
        touch beta_codes.txt
    fi

    # 6. Directories
    mkdir -p decision_logs prompts
}

# ------------------------------------------------------------------------
# Commands
# ------------------------------------------------------------------------
start() {
    reset_auto_test_config
    parse_auto_test_args "$@"
    parse_dev_mode_arg "$@"

    check_docker
    check_config
    
    print_info "🚀 正在启动 NOFX (Docker)..."
    
    # 如果启用了开发模式，通过环境变量传递给 docker compose
    if [ "$NOFX_DEV_MODE" = "true" ]; then
        NOFX_DEV_MODE=true $DOCKER_COMPOSE_CMD up -d --remove-orphans
    else
        $DOCKER_COMPOSE_CMD up -d --remove-orphans
    fi

    print_success "✅ 服务已启动！"
    
    # 开发模式下显示额外提示
    if [ "$NOFX_DEV_MODE" = "true" ]; then
        echo ""
        print_warning "⚠️  测试模式注意事项："
        print_warning "   • JWT 鉴权已绕过，前端将自动登录测试用户"
        print_warning "   • 仅用于本地开发和测试，请勿在生产环境使用"
        echo ""
    fi
    
    show_access_info
    maybe_run_post_start_tests
}

stop() {
    check_docker
    print_info "🛑 正在停止服务..."
    $DOCKER_COMPOSE_CMD down
    print_success "✅ 服务已停止"
}

restart() {
    local restart_args=("$@")
    stop
    sleep 1
    start "${restart_args[@]}"
}

logs() {
    check_docker
    $DOCKER_COMPOSE_CMD logs -f --tail=100
}

build() {
    check_docker
    check_config
    print_info "🔨 正在构建 Docker 镜像..."
    $DOCKER_COMPOSE_CMD build --no-cache
    print_success "✅ 构建完成"
}

status() {
    check_docker
    $DOCKER_COMPOSE_CMD ps
}

show_access_info() {
    # Extract ports from .env or use defaults
    local fe_port=$(grep NOFX_FRONTEND_PORT .env | cut -d= -f2 || echo 3000)
    local be_port=$(grep NOFX_BACKEND_PORT .env | cut -d= -f2 || echo 8080)
    
    echo ""
    echo "🌐 Web 界面: http://localhost:${fe_port}"
    echo "🔗 API 端点: http://localhost:${be_port}"
    echo ""
    echo "常用命令:"
    echo "  ./start.sh logs    查看日志"
    echo "  ./start.sh stop    停止服务"
}

show_help() {
    echo "NOFX AI Trading System - Docker 管理脚本"
    echo ""
    echo "用法: ./start.sh [command] [options]"
    echo ""
    echo "命令:"
    echo "  start       启动服务 (默认)"
    echo "  stop        停止服务"
    echo "  restart     重启服务"
    echo "  logs        查看日志"
    echo "  build       重新构建镜像"
    echo "  status      查看容器状态"
    echo "  help        显示此帮助"
    echo ""
    echo "Start/Restart 相关选项:"
    echo "  --dev                  启用测试模式（免登录调试）"
    echo "  --with-tests           启动完成后在容器内执行 go test"
    echo "  --no-tests             显式关闭自动测试 (覆盖 NOFX_AUTO_TEST_AFTER_START)"
    echo "  --allow-test-fail      测试失败仅报警不中断"
    echo "  --test-wait SECONDS    健康检查等待秒数"
    echo "  --test-packages \"PKGS\"  指定测试包"
    echo "  --go-flags \"FLAGS\"     自定义 go test 参数"
    echo ""
    echo "测试模式说明:"
    echo "  使用 --dev 参数启动时，系统将："
    echo "  • 绕过 JWT 鉴权，前端自动登录测试用户"
    echo "  • 仅用于本地开发测试，请勿在生产环境使用"
}

# ------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------
COMMAND="${1:-start}"
shift || true

case "$COMMAND" in
    start)
        start "$@"
        ;;
    stop)
        stop "$@"
        ;;
    restart)
        restart "$@"
        ;;
    logs)
        logs "$@"
        ;;
    build)
        build "$@"
        ;;
    status)
        status "$@"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "未知命令: $COMMAND"
        show_help
        exit 1
        ;;
esac

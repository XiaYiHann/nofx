#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX AI Trading System - Docker Deployment Script
# Docker 部署启动脚本
# Usage: ./start_docker.sh [start|stop|restart|status|logs|build] [--dev]
# ═══════════════════════════════════════════════════════════════

set -e

# ------------------------------------------------------------------------
# Color Definitions
# ------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ------------------------------------------------------------------------
# Utility Functions
# ------------------------------------------------------------------------
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

normalize_bool() {
    local value="${1:-false}"
    case "${value,,}" in
        1|true|yes|on)
            echo "true"
            ;;
        *)
            echo "false"
            ;;
    esac
}

# ------------------------------------------------------------------------
# Post-start test configuration (defaults, can be overridden later)
# ------------------------------------------------------------------------
AUTO_TEST_SCRIPT="./scripts/docker/run_tests_after_start.sh"
AUTO_TEST_DEFAULT_PACKAGES="./config/... ./api/..."
AUTO_TEST_ENABLED="false"
AUTO_TEST_ALLOW_FAIL="false"
AUTO_TEST_PACKAGES="$AUTO_TEST_DEFAULT_PACKAGES"
AUTO_TEST_WAIT="120"
AUTO_TEST_GOFLAGS="-count=1 -timeout 5m"
DEV_MODE_FLAG=""

reset_auto_test_config() {
    AUTO_TEST_PACKAGES="${NOFX_AUTO_TEST_PACKAGES:-$AUTO_TEST_DEFAULT_PACKAGES}"
    AUTO_TEST_WAIT="${NOFX_AUTO_TEST_WAIT:-120}"
    AUTO_TEST_GOFLAGS="${NOFX_AUTO_TEST_GOFLAGS:--count=1 -timeout 5m}"
    AUTO_TEST_ENABLED="$(normalize_bool "${NOFX_AUTO_TEST_AFTER_START:-false}")"
    AUTO_TEST_ALLOW_FAIL="$(normalize_bool "${NOFX_AUTO_TEST_ALLOW_FAIL:-false}")"
}

# ------------------------------------------------------------------------
# Detect Docker Compose Command
# ------------------------------------------------------------------------
detect_docker_compose() {
    if command -v docker compose &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        print_error "Docker Compose 未安装！"
        print_info "请安装 Docker 和 Docker Compose: https://docs.docker.com/get-docker/"
        exit 1
    fi
    print_info "使用 Docker Compose 命令: $DOCKER_COMPOSE"
}

# ------------------------------------------------------------------------
# Check Docker Environment
# ------------------------------------------------------------------------
check_docker() {
    print_info "检查 Docker 环境..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装！"
        print_info "请安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    if ! docker ps &> /dev/null; then
        print_error "Docker daemon 未运行！请启动 Docker"
        exit 1
    fi
    
    detect_docker_compose
    
    print_success "Docker 和 Docker Compose 已安装"
}

# ------------------------------------------------------------------------
# Setup Environment
# ------------------------------------------------------------------------
setup_environment() {
    print_info "检查环境配置..."
    
    # 检查 .env 文件
    if [ ! -f ".env" ]; then
        print_warning ".env 不存在，从模板复制..."
        if [ -f ".env.example" ]; then
            cp .env.example .env
        else
            cat > .env << EOF
NOFX_FRONTEND_PORT=3000
NOFX_BACKEND_PORT=8080
NOFX_TIMEZONE=Asia/Shanghai
DATA_ENCRYPTION_KEY=your_data_encryption_key_here_change_me
JWT_SECRET=your_jwt_secret_here_change_me
NODE_ENV=production
GO_ENV=production
EOF
        fi
        print_info "已创建 .env 文件"
    fi
    
    print_success "环境变量文件存在"
    
    # 检查加密环境
    if [ ! -f "secrets/rsa_key" ] || [ ! -f "secrets/rsa_key.pub" ]; then
        print_warning "RSA密钥对不存在"
        need_setup=true
    fi
    
    if ! grep -q "^DATA_ENCRYPTION_KEY=" .env || grep -q "your_data_encryption_key_here_change_me" .env; then
        print_warning "数据加密密钥未配置"
        need_setup=true
    fi
    
    if ! grep -q "^JWT_SECRET=" .env || grep -q "your_jwt_secret_here_change_me" .env; then
        print_warning "JWT密钥未配置"
        need_setup=true
    fi
    
    if [ "$need_setup" = "true" ]; then
        print_info "🔐 自动设置加密环境..."
        if [ -f "scripts/setup_encryption.sh" ]; then
            echo -e "Y\nn\nn" | bash scripts/setup_encryption.sh
            print_success "加密环境设置完成"
        fi
    else
        print_success "🔐 加密环境已配置"
        print_info "  • RSA密钥对: secrets/rsa_key + secrets/rsa_key.pub"
        print_info "  • 数据加密密钥: .env (DATA_ENCRYPTION_KEY)"
        print_info "  • JWT认证密钥: .env (JWT_SECRET)"
        print_info "  • 加密算法: RSA-OAEP-2048 + AES-256-GCM + HS256"
        print_info "  • 保护数据: API密钥、私钥、Hyperliquid代理钱包、用户认证"
        
        # 修复权限
        if [ -f "secrets/rsa_key" ]; then
            print_warning "修复RSA私钥权限..."
            chmod 600 secrets/rsa_key
        fi
        
        if [ -f ".env" ]; then
            print_warning "修复环境文件权限..."
            chmod 600 .env
        fi
    fi
    
    # 检查 config.json
    if [ ! -f "config.json" ]; then
        if [ -f "config.json.example" ]; then
            cp config.json.example config.json
            print_info "已从示例复制 config.json"
        fi
    fi
    print_success "配置文件存在"
    
    # 检查数据库文件
    if [ ! -f "config.db" ]; then
        print_info "数据库不存在，容器启动时会自动创建"
        print_info "将包含以下交易所: Binance, Hyperliquid, Aster, Paper Trading"
        # 创建空数据库文件，让 Docker 正确挂载
        touch config.db
        chmod 600 config.db
    else
        # 备份现有数据库
        local backup_dir="database_backups"
        mkdir -p "$backup_dir"
        local timestamp=$(date +%Y%m%d_%H%M%S)
        local backup_file="$backup_dir/config.db.$timestamp"
        
        cp config.db "$backup_file"
        chmod 600 "$backup_file"
        print_success "数据库已备份: $backup_file"
        
        # 清理旧备份
        ls -t $backup_dir/config.db.* 2>/dev/null | tail -n +11 | xargs rm -f 2>/dev/null || true
    fi
    print_success "数据库文件存在"
    
    # 检查 beta_codes.txt
    if [ ! -f "beta_codes.txt" ]; then
        touch beta_codes.txt
        print_info "已创建空的 beta_codes.txt (Docker 挂载需要)"
    fi
    
    # 确保必要目录存在
    mkdir -p secrets logs decision_logs prompts database_backups
    chmod 700 secrets
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
# Build Docker Images
# ------------------------------------------------------------------------
build_images() {
    local no_cache=$1
    
    print_info "构建 Docker 镜像..."
    
    # 智能代理检测 (针对中国用户)
    if [ -z "$GOPROXY" ] && grep -q "Asia/Shanghai" .env; then
        print_info "🌐 检测到 Asia/Shanghai 时区，自动设置 Go 代理..."
        export GOPROXY="https://goproxy.cn,direct"
    fi
    
    if [ -z "$NPM_REGISTRY" ] && grep -q "Asia/Shanghai" .env; then
        print_info "🌐 检测到 Asia/Shanghai 时区，自动设置 NPM 镜像..."
        export NPM_REGISTRY="https://registry.npmmirror.com/"
    fi
    
    # 显示构建参数
    if [ -n "$GOPROXY" ]; then
        print_info "🔧 Build Arg: GOPROXY=$GOPROXY"
    fi
    if [ -n "$NPM_REGISTRY" ]; then
        print_info "🔧 Build Arg: NPM_REGISTRY=$NPM_REGISTRY"
    fi
    
    if [ "$no_cache" == "--no-cache" ]; then
        print_warning "使用 --no-cache 重新构建（将花费更长时间）"
        $DOCKER_COMPOSE build --no-cache
    else
        $DOCKER_COMPOSE build
    fi
    
    print_success "Docker 镜像构建完成"
}

# ------------------------------------------------------------------------
# Start Services
# ------------------------------------------------------------------------
start_services() {
    local dev_mode=$1
    
    print_info "正在启动 NOFX AI Trading System..."
    
    # 启动容器
    print_info "启动容器..."
    $DOCKER_COMPOSE up -d
    
    # 等待服务就绪
    print_info "等待服务启动..."
    sleep 5
    
    # 检查容器状态
    if ! docker ps | grep -q "nofx-trading"; then
        print_error "后端容器启动失败"
        print_info "查看日志: $DOCKER_COMPOSE logs nofx"
        exit 1
    fi
    
    if ! docker ps | grep -q "nofx-frontend"; then
        print_error "前端容器启动失败"
        print_info "查看日志: $DOCKER_COMPOSE logs nofx-frontend"
        exit 1
    fi
    
    print_success "服务已启动！"
    
    # 显示访问信息
    local frontend_port=$(grep NOFX_FRONTEND_PORT .env | cut -d= -f2)
    local backend_port=$(grep NOFX_BACKEND_PORT .env | cut -d= -f2)
    frontend_port=${frontend_port:-3000}
    backend_port=${backend_port:-8080}
    
    echo ""
    print_success "🎯 NOFX AI Trading System 已启动（Docker 模式）"
    echo ""
    echo "📱 Web 界面: http://localhost:$frontend_port"
    echo "🔗 API 端点: http://localhost:$backend_port"
    echo ""
    echo "📊 容器状态:"
    docker ps --filter "name=nofx" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    echo ""
    echo "📋 常用命令:"
    echo "  查看服务状态: ./start_docker.sh status"
    echo "  查看所有日志: ./start_docker.sh logs"
    echo "  查看后端日志: ./start_docker.sh logs nofx"
    echo "  查看前端日志: ./start_docker.sh logs nofx-frontend"
    echo "  停止服务: ./start_docker.sh stop"
    echo "  重启服务: ./start_docker.sh restart"
    echo ""
    echo "💡 Paper Trading 已启用！"
    echo "   登录后在交易所配置中可以看到 'Paper Trading (Binance Testnet)'"
    echo ""
    echo "⚠️  重要提示:"
    echo "   • 数据库文件: ./config.db (已挂载到容器)"
    echo "   • 如果看不到 Paper Trading，请删除 config.db 后重启"
    echo "   • 命令: rm config.db && ./start_docker.sh restart"
    echo ""

    maybe_run_post_start_tests
}

# ------------------------------------------------------------------------
# Stop Services
# ------------------------------------------------------------------------
stop_services() {
    print_info "停止服务..."
    
    $DOCKER_COMPOSE down
    
    print_success "所有服务已停止"
}

# ------------------------------------------------------------------------
# Restart Services
# ------------------------------------------------------------------------
restart_services() {
    local dev_mode=$1
    
    print_info "重启服务..."
    
    stop_services
    sleep 2
    setup_environment
    start_services "$dev_mode"
}

# ------------------------------------------------------------------------
# Check Status
# ------------------------------------------------------------------------
check_status() {
    print_info "检查服务状态..."
    
    echo ""
    echo "📊 容器状态:"
    docker ps --filter "name=nofx" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || {
        print_warning "没有运行中的容器"
        return 1
    }
    
    echo ""
    echo "💾 数据库状态:"
    if [ -f "config.db" ]; then
        local db_size=$(du -h config.db | cut -f1)
        print_success "数据库文件存在 (大小: $db_size)"
        
        # 如果安装了 sqlite3，显示交易所列表
        if command -v sqlite3 &> /dev/null; then
            echo ""
            echo "📋 已配置的交易所:"
            sqlite3 config.db "SELECT id, name, type FROM exchanges WHERE user_id='default' ORDER BY id;" 2>/dev/null || true
        fi
    else
        print_warning "数据库文件不存在"
    fi
    
    echo ""
    echo "🔐 加密环境:"
    if [ -f "secrets/rsa_key" ] && [ -f "secrets/rsa_key.pub" ]; then
        print_success "RSA密钥对存在"
    else
        print_warning "RSA密钥对缺失"
    fi
    
    if grep -q "^DATA_ENCRYPTION_KEY=" .env && ! grep -q "your_data_encryption_key_here_change_me" .env; then
        print_success "数据加密密钥已配置"
    else
        print_warning "数据加密密钥未配置"
    fi
    
    echo ""
}

# ------------------------------------------------------------------------
# View Logs
# ------------------------------------------------------------------------
view_logs() {
    local service=$1
    
    if [ -z "$service" ]; then
        print_info "查看所有服务日志 (Ctrl+C 退出)..."
        $DOCKER_COMPOSE logs -f
    else
        print_info "查看 $service 日志 (Ctrl+C 退出)..."
        $DOCKER_COMPOSE logs -f "$service"
    fi
}

# ------------------------------------------------------------------------
# Rebuild with Fresh Database
# ------------------------------------------------------------------------
rebuild_fresh() {
    print_warning "⚠️  此操作将:"
    echo "  1. 停止所有容器"
    echo "  2. 删除现有数据库"
    echo "  3. 重新构建镜像"
    echo "  4. 启动服务（将创建包含 Paper Trading 的新数据库）"
    echo ""
    read -p "确认继续? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "开始重建..."
        
        # 停止服务
        stop_services
        
        # 备份并删除数据库
        if [ -f "config.db" ]; then
            local backup_dir="database_backups"
            mkdir -p "$backup_dir"
            local timestamp=$(date +%Y%m%d_%H%M%S)
            mv config.db "$backup_dir/config.db.before_rebuild.$timestamp"
            print_success "旧数据库已备份"
        fi
        
        # 重新构建
        build_images "--no-cache"
        
        # 启动服务
        setup_environment
        start_services
        
        print_success "重建完成！现在应该可以看到 Paper Trading 了"
    else
        print_info "操作已取消"
    fi
}

# ------------------------------------------------------------------------
# Help
# ------------------------------------------------------------------------
show_help() {
    echo "NOFX Docker 部署脚本"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  start           启动服务"
    echo "  stop            停止服务"
    echo "  restart         重启服务"
    echo "  status          查看状态"
    echo "  logs [service]  查看日志"
    echo "  build           重新构建镜像"
    echo "  update          更新镜像并重启 (保留数据)"
    echo "  rebuild-fresh   删除数据库并重新构建（修复 Paper Trading 缺失问题）"
    echo "  help            显示此帮助"
    echo ""
    echo "Start/Restart 相关选项:"
    echo "  --with-tests           启用部署后自动测试（也可设置 NOFX_AUTO_TEST_AFTER_START=true）"
    echo "  --no-tests             显式关闭自动测试"
    echo "  --allow-test-fail      测试失败仅报警不中断"
    echo "  --test-wait SECONDS    设置健康检查等待秒数"
    echo "  --test-packages \"PKGS\"  指定 go test 包列表"
    echo "  --go-flags \"FLAGS\"     覆盖 go test 额外参数"
    echo ""
    echo "Examples:"
    echo "  $0 start                    # 启动服务"
    echo "  $0 start --with-tests       # 启动后自动在容器内运行 go test"
    echo "  $0 start --with-tests --allow-test-fail"
    echo "  $0 update                   # 更新代码并重启"
    echo "  $0 logs                     # 查看所有日志"
    echo "  $0 logs nofx                # 只查看后端日志"
    echo "  $0 build                    # 重新构建镜像"
    echo "  $0 rebuild-fresh            # 完全重建（包含新数据库）"
    echo ""
    echo "💡 如果看不到 Paper Trading:"
    echo "  方案1: rm config.db && $0 restart"
    echo "  方案2: $0 rebuild-fresh"
    echo ""
}

# ------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------
main() {
    local command=${1:-start}
    shift || true

    reset_auto_test_config
    DEV_MODE_FLAG=""
    local positional_args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dev)
                DEV_MODE_FLAG="--dev"
                shift
                ;;
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
            --)
                shift
                while [[ $# -gt 0 ]]; do
                    positional_args+=("$1")
                    shift
                done
                ;;
            *)
                positional_args+=("$1")
                shift
                ;;
        esac
    done

    check_docker

    case "$command" in
        start)
            setup_environment
            local start_arg="$DEV_MODE_FLAG"
            if [ -z "$start_arg" ] && [ ${#positional_args[@]} -gt 0 ]; then
                start_arg="${positional_args[0]}"
            fi
            start_services "$start_arg"
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services "$DEV_MODE_FLAG"
            ;;
        status)
            check_status
            ;;
        logs)
            local service="${positional_args[0]:-}"
            view_logs "$service"
            ;;
        build)
            local build_arg="${positional_args[0]:-}"
            build_images "$build_arg"
            ;;
        update)
            local build_arg="${positional_args[0]:-}"
            print_info "开始更新流程 (保留数据)..."
            build_images "$build_arg"
            restart_services "$DEV_MODE_FLAG"
            ;;
        rebuild-fresh)
            rebuild_fresh
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

main "$@"

#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX AI Trading System - Local Development Startup Script
# 本地开发模式启动脚本（不使用 Docker）
# Usage: ./start_local.sh [command]
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

# ------------------------------------------------------------------------
# PID 文件路径
# ------------------------------------------------------------------------
BACKEND_PID_FILE=".backend.pid"
FRONTEND_PID_FILE=".frontend.pid"

# ------------------------------------------------------------------------
# 检查依赖
# ------------------------------------------------------------------------
check_dependencies() {
    # 检查 Go
    if ! command -v go &> /dev/null; then
        print_error "Go 未安装！请先安装 Go: https://golang.org/dl/"
        exit 1
    fi

    # 检查 Node.js
    if ! command -v node &> /dev/null; then
        print_error "Node.js 未安装！请先安装 Node.js: https://nodejs.org/"
        exit 1
    fi

    # 检查 npm
    if ! command -v npm &> /dev/null; then
        print_error "npm 未安装！请先安装 npm"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 配置检查
# ------------------------------------------------------------------------
check_config() {
    # 检查 config.json
    if [ ! -f "config.json" ]; then
        print_warning "config.json 不存在，创建默认配置..."
        cat > config.json <<EOF
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
        print_success "✓ 已创建 config.json"
    fi

    # 检查 secrets 目录
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

    # 检查数据库文件
    if [ ! -f "config.db" ]; then
        print_info "创建空数据库文件..."
        touch config.db
        chmod 600 config.db
    fi

    # 检查其他必要文件
    if [ ! -f "beta_codes.txt" ]; then
        touch beta_codes.txt
    fi

    # 创建必要目录
    mkdir -p decision_logs prompts
}

# ------------------------------------------------------------------------
# 安装前端依赖
# ------------------------------------------------------------------------
install_frontend_deps() {
    if [ ! -d "web/node_modules" ]; then
        print_info "📦 安装前端依赖..."
        cd web
        npm install
        cd ..
        print_success "✓ 前端依赖安装完成"
    fi
}

# ------------------------------------------------------------------------
# 启动后端
# ------------------------------------------------------------------------
start_backend() {
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_warning "后端已在运行 (PID: $pid)"
            return
        fi
    fi

    print_info "🚀 启动后端服务..."
    
    # 设置环境变量
    export DATA_ENCRYPTION_KEY=${DATA_ENCRYPTION_KEY:-$(openssl rand -hex 32)}
    export JWT_SECRET=${JWT_SECRET:-$(openssl rand -hex 32)}
    
    # 检查是否为开发模式
    if [ "$NOFX_DEV_MODE" = "true" ]; then
        print_warning "🚧 开发模式已启用 (NOFX_DEV_MODE=true)"
        export DISABLE_OTP=true
    fi
    
    # 在后台启动 Go 服务
    nohup go run . > backend.log 2>&1 &
    local pid=$!
    echo $pid > "$BACKEND_PID_FILE"
    
    # 等待后端启动
    print_info "等待后端服务启动..."
    for i in {1..30}; do
        if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
            print_success "✅ 后端服务已启动 (PID: $pid)"
            return
        fi
        sleep 1
    done
    
    print_error "后端服务启动超时，请检查 backend.log"
}

# ------------------------------------------------------------------------
# 启动前端
# ------------------------------------------------------------------------
start_frontend() {
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_warning "前端已在运行 (PID: $pid)"
            return
        fi
    fi

    print_info "🚀 启动前端开发服务器..."
    
    cd web
    # 在后台启动前端开发服务器
    nohup npm run dev > ../frontend.log 2>&1 &
    local pid=$!
    cd ..
    echo $pid > "$FRONTEND_PID_FILE"
    
    # 等待前端启动
    print_info "等待前端服务启动..."
    sleep 3
    
    print_success "✅ 前端服务已启动 (PID: $pid)"
}

# ------------------------------------------------------------------------
# 停止服务
# ------------------------------------------------------------------------
stop_backend() {
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_info "停止后端服务 (PID: $pid)..."
            kill $pid
            rm "$BACKEND_PID_FILE"
            print_success "✓ 后端服务已停止"
        else
            rm "$BACKEND_PID_FILE"
        fi
    else
        print_warning "后端服务未运行"
    fi
}

stop_frontend() {
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_info "停止前端服务 (PID: $pid)..."
            kill $pid
            rm "$FRONTEND_PID_FILE"
            print_success "✓ 前端服务已停止"
        else
            rm "$FRONTEND_PID_FILE"
        fi
    else
        print_warning "前端服务未运行"
    fi
}

stop() {
    print_info "🛑 停止所有服务..."
    stop_backend
    stop_frontend
}

# ------------------------------------------------------------------------
# 启动所有服务
# ------------------------------------------------------------------------
start() {
    # 默认关闭开发模式，防止环境变量污染
    export NOFX_DEV_MODE=false

    # 解析 --dev 参数
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

    check_dependencies
    check_config
    install_frontend_deps
    
    start_backend
    start_frontend
    
    print_success "✅ 所有服务已启动！"
    
    # 开发模式下显示额外提示
    if [ "$NOFX_DEV_MODE" = "true" ]; then
        echo ""
        print_warning "⚠️  测试模式注意事项："
        print_warning "   • JWT 鉴权已绕过，前端将自动登录测试用户"
        print_warning "   • 仅用于本地开发和测试，请勿在生产环境使用"
        echo ""
    fi
    
    show_access_info
}

# ------------------------------------------------------------------------
# 重启服务
# ------------------------------------------------------------------------
restart() {
    local restart_args=("$@")
    stop
    sleep 2
    start "${restart_args[@]}"
}

# ------------------------------------------------------------------------
# 查看日志
# ------------------------------------------------------------------------
logs() {
    local service=$1
    
    if [ "$service" == "backend" ] || [ "$service" == "be" ]; then
        if [ -f "backend.log" ]; then
            tail -f backend.log
        else
            print_error "backend.log 不存在"
        fi
    elif [ "$service" == "frontend" ] || [ "$service" == "fe" ]; then
        if [ -f "frontend.log" ]; then
            tail -f frontend.log
        else
            print_error "frontend.log 不存在"
        fi
    else
        print_info "同时查看前后端日志..."
        if [ -f "backend.log" ] && [ -f "frontend.log" ]; then
            tail -f backend.log frontend.log
        else
            print_error "日志文件不存在"
        fi
    fi
}

# ------------------------------------------------------------------------
# 查看状态
# ------------------------------------------------------------------------
status() {
    print_info "检查服务状态..."
    
    # 检查后端
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} 后端服务运行中 (PID: $pid)"
        else
            echo -e "${RED}✗${NC} 后端服务未运行 (PID文件存在但进程不存在)"
        fi
    else
        echo -e "${RED}✗${NC} 后端服务未运行"
    fi
    
    # 检查前端
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} 前端服务运行中 (PID: $pid)"
        else
            echo -e "${RED}✗${NC} 前端服务未运行 (PID文件存在但进程不存在)"
        fi
    else
        echo -e "${RED}✗${NC} 前端服务未运行"
    fi
}

# ------------------------------------------------------------------------
# 显示访问信息
# ------------------------------------------------------------------------
show_access_info() {
    echo ""
    echo "🌐 前端开发服务: http://localhost:3000"
    echo "🔗 后端 API: http://localhost:8080"
    echo ""
    echo "常用命令:"
    echo "  ./start_local.sh logs [backend|frontend]  查看日志"
    echo "  ./start_local.sh stop                     停止服务"
    echo "  ./start_local.sh restart                  重启服务"
    echo "  ./start_local.sh status                   查看状态"
    echo ""
    echo "日志文件:"
    echo "  backend.log   - 后端日志"
    echo "  frontend.log  - 前端日志"
}

# ------------------------------------------------------------------------
# 帮助信息
# ------------------------------------------------------------------------
show_help() {
    echo "NOFX AI Trading System - 本地开发启动脚本"
    echo ""
    echo "用法: ./start_local.sh [command] [options]"
    echo ""
    echo "命令:"
    echo "  start [--dev]                启动所有服务 (默认)"
    echo "  stop                         停止所有服务"
    echo "  restart [--dev]              重启所有服务"
    echo "  status                       查看服务状态"
    echo "  logs [backend|frontend]      查看日志"
    echo "  help                         显示此帮助"
    echo ""
    echo "选项:"
    echo "  --dev                        启用测试模式（免登录调试）"
    echo ""
    echo "服务说明:"
    echo "  后端: Go 服务运行在 http://localhost:8080"
    echo "  前端: Vite 开发服务器运行在 http://localhost:5173"
    echo ""
    echo "测试模式说明:"
    echo "  使用 --dev 参数启动时，系统将："
    echo "  • 绕过 JWT 鉴权，前端自动登录测试用户"
    echo "  • 禁用 OTP 验证"
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
        stop
        ;;
    restart)
        restart "$@"
        ;;
    status)
        status
        ;;
    logs)
        logs $1
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

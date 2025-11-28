#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX AI Trading System - Setup & Reset Script
# Usage: ./setup.sh
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
# Confirmation
# ------------------------------------------------------------------------
confirm_action() {
    echo -e "${RED}⚠️  警告: 此操作将执行以下清理:${NC}"
    echo -e "  1. 停止所有相关 Docker 容器"
    echo -e "  2. 删除所有相关 Docker 镜像和容器"
    echo -e "  3. ${RED}删除本地数据库文件 (config.db)${NC}"
    echo -e "  4. 删除本地生成的密钥 (secrets/)"
    echo -e "  5. 删除本地配置文件 (config.json, .env)"
    echo -e "  6. 清理前端构建缓存 (web/node_modules, web/dist)"
    echo -e "  7. 重新构建并启动全新的系统"
    echo ""
    read -p "确定要继续吗? (y/N): " choice
    case "$choice" in 
        y|Y ) echo "开始执行...";;
        * ) echo "操作已取消"; exit 0;;
    esac
}

# ------------------------------------------------------------------------
# Cleanup
# ------------------------------------------------------------------------
cleanup() {
    print_info "🧹 开始清理系统..."

    # 1. Stop Docker containers
    if command -v docker &> /dev/null; then
        print_info "停止 Docker 容器..."
        docker compose down --volumes --remove-orphans || true
    fi

    # 2. Remove files
    print_info "删除本地文件..."
    rm -f config.db
    rm -f config.json
    rm -f .env
    rm -rf secrets
    rm -f backend.log frontend.log
    rm -f .backend.pid .frontend.pid
    
    # 3. Clean frontend
    print_info "清理前端缓存..."
    rm -rf web/node_modules
    rm -rf web/dist
    rm -rf web/.vite

    # 4. Clean Go cache
    if command -v go &> /dev/null; then
        print_info "清理 Go 缓存..."
        go clean -cache
    fi

    print_success "✓ 清理完成"
}

# ------------------------------------------------------------------------
# Rebuild & Start
# ------------------------------------------------------------------------
rebuild_and_start() {
    print_info "🔄 重新构建并启动系统..."
    
    # Ensure start.sh is executable
    chmod +x start.sh

    # Build with no cache
    ./start.sh build

    # Start
    ./start.sh start
}

# ------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------
main() {
    confirm_action
    cleanup
    rebuild_and_start
    
    echo ""
    print_success "🎉 系统重置并重新启动完成！"
    echo "请访问 http://localhost:3000 进行初始化配置。"
}

main

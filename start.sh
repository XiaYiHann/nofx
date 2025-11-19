#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX AI Trading System - Local Development Quick Start Script
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
# Utility Functions: Colored Output
# ------------------------------------------------------------------------
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# ------------------------------------------------------------------------
# Process Management Functions
# ------------------------------------------------------------------------
is_port_in_use() {
    local port=$1
    if command -v lsof &> /dev/null; then
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            return 0
        fi
    elif command -v netstat &> /dev/null; then
        if netstat -tuln 2>/dev/null | grep ":$port " >/dev/null; then
            return 0
        fi
    fi
    return 1
}

find_free_port() {
    local start_port=$1
    local port=$start_port
    while is_port_in_use $port; do
        port=$((port + 1))
    done
    echo $port
}

# ------------------------------------------------------------------------
# Validation: Node.js and npm
# ------------------------------------------------------------------------
check_nodejs() {
    if ! command -v node &> /dev/null; then
        print_error "Node.js 未安装！请先安装 Node.js: https://nodejs.org/"
        print_info "推荐版本: Node.js 18+"
        exit 1
    fi

    if ! command -v npm &> /dev/null; then
        print_error "npm 未安装！请先安装 npm"
        exit 1
    fi

    local node_version=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
    if [ "$node_version" -lt 18 ]; then
        print_warning "推荐使用 Node.js 18+，当前版本: $(node --version)"
    fi

    print_success "Node.js 和 npm 已安装 ($(node --version))"
}

# ------------------------------------------------------------------------
# Validation: Go
# ------------------------------------------------------------------------
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go 未安装！请先安装 Go: https://golang.org/dl/"
        print_info "推荐版本: Go 1.21+"
        exit 1
    fi

    local go_version=$(go version | cut -d' ' -f3 | sed 's/go//')
    print_success "Go 已安装 ($go_version)"
}

# ------------------------------------------------------------------------
# Validation: Environment File (.env)
# ------------------------------------------------------------------------
check_env() {
    if [ ! -f ".env" ]; then
        print_warning ".env 不存在，从模板复制..."
        if [ -f ".env.example" ]; then
            cp .env.example .env
        else
            # 创建基本的.env文件
            cat > .env << EOF
# NOFX AI Trading System - Environment Configuration
# 本地开发环境配置

# 端口配置
NOFX_FRONTEND_PORT=3000
NOFX_BACKEND_PORT=8080

# 加密配置 (首次运行会自动生成)
DATA_ENCRYPTION_KEY=your_data_encryption_key_here_change_me
JWT_SECRET=your_jwt_secret_here_change_me

# 开发模式配置
NODE_ENV=development
GO_ENV=development
EOF
        fi
        print_info "✓ 已创建 .env 文件"
        print_info "💡 请检查 .env 文件中的配置，特别是加密密钥"
    fi
    print_success "环境变量文件存在"
}

# ------------------------------------------------------------------------
# Validation: Encryption Environment (RSA Keys + Data Encryption Key)
# ------------------------------------------------------------------------
check_encryption() {
    local need_setup=false

    print_info "检查加密环境..."

    # 检查RSA密钥对
    if [ ! -f "secrets/rsa_key" ] || [ ! -f "secrets/rsa_key.pub" ]; then
        print_warning "RSA密钥对不存在"
        need_setup=true
    fi

    # 检查数据加密密钥
    if ! grep -q "^DATA_ENCRYPTION_KEY=" .env || grep -q "your_data_encryption_key_here_change_me" .env; then
        print_warning "数据加密密钥未配置或使用默认值"
        need_setup=true
    fi

    # 检查JWT认证密钥
    if ! grep -q "^JWT_SECRET=" .env || grep -q "your_jwt_secret_here_change_me" .env; then
        print_warning "JWT认证密钥未配置或使用默认值"
        need_setup=true
    fi

    # 如果需要设置加密环境，直接自动设置
    if [ "$need_setup" = "true" ]; then
        print_info "🔐 检测到加密环境未配置，正在自动设置..."

        # 检查加密设置脚本是否存在
        if [ -f "scripts/setup_encryption.sh" ]; then
            print_info "加密系统将保护: API密钥、私钥、Hyperliquid代理钱包"
            echo ""

            # 自动运行加密设置脚本
            echo -e "Y\nn\nn" | bash scripts/setup_encryption.sh
            if [ $? -eq 0 ]; then
                echo ""
                print_success "🔐 加密环境设置完成！"
                print_info "  • RSA-2048密钥对已生成"
                print_info "  • AES-256数据加密密钥已配置"
                print_info "  • JWT认证密钥已配置"
                print_info "  • 所有敏感数据现在都受加密保护"
                echo ""
            else
                print_error "加密环境设置失败"
                exit 1
            fi
        else
            print_error "加密设置脚本不存在: scripts/setup_encryption.sh"
            print_info "请手动运行: ./scripts/setup_encryption.sh"
            exit 1
        fi
    else
        print_success "🔐 加密环境已配置"
        print_info "  • RSA密钥对: secrets/rsa_key + secrets/rsa_key.pub"
        print_info "  • 数据加密密钥: .env (DATA_ENCRYPTION_KEY)"
        print_info "  • JWT认证密钥: .env (JWT_SECRET)"
        print_info "  • 加密算法: RSA-OAEP-2048 + AES-256-GCM + HS256"
        print_info "  • 保护数据: API密钥、私钥、Hyperliquid代理钱包、用户认证"

        # 验证密钥文件权限
        if [ -f "secrets/rsa_key" ]; then
            chmod 600 secrets/rsa_key
        fi

        if [ -f ".env" ]; then
            chmod 600 .env
        fi
    fi
}

# ------------------------------------------------------------------------
# Validation: Configuration File (config.json)
# ------------------------------------------------------------------------
check_config() {
    if [ ! -f "config.json" ]; then
        print_warning "config.json 不存在，从模板复制..."
        if [ -f "config.json.example" ]; then
            cp config.json.example config.json
            print_info "✓ 已使用默认配置创建 config.json"
        else
            print_warning "config.json.example 不存在，创建基本配置..."
            cat > config.json << EOF
{
  "system": {
    "lever_rate": 10,
    "leverage_enabled": true,
    "admin_mode": false
  },
  "models": {
    "deepseek": {
      "enabled": false,
      "api_key": "",
      "custom_api_url": "",
      "custom_model_name": "deepseek-chat"
    },
    "qwen": {
      "enabled": false,
      "api_key": "",
      "custom_api_url": "",
      "custom_model_name": "qwen-turbo"
    },
    "claude": {
      "enabled": false,
      "api_key": "",
      "custom_api_url": "",
      "custom_model_name": "claude-3-sonnet-20240229"
    }
  },
  "exchanges": {
    "binance": {
      "enabled": false,
      "api_key": "",
      "secret_key": "",
      "testnet": false
    },
    "hyperliquid": {
      "enabled": false,
      "api_key": "",
      "testnet": true,
      "wallet_addr": "",
      "aster_user": "",
      "aster_signer": "",
      "aster_private_key": ""
    },
    "aster": {
      "enabled": false,
      "api_key": "",
      "testnet": false
    }
  }
}
EOF
        fi
        print_info "💡 如需修改基础设置，可编辑 config.json"
        print_info "💡 模型/交易所/交易员配置请使用Web界面"
    fi
    print_success "配置文件存在"
}

# ------------------------------------------------------------------------
# Validation: Database File (config.db) - Enhanced
# ------------------------------------------------------------------------
check_database() {
    print_info "检查数据库环境..."

    # 处理目录与文件冲突
    if [ -d "config.db" ]; then
        print_warning "⚠️  config.db 是目录而非文件，正在处理..."
        if [ -f "config.db.backup" ]; then
            print_info "发现旧备份文件，先清理..."
            rm -rf config.db.backup
        fi
        mv config.db config.db.backup.$(date +%Y%m%d_%H%M%S).directory 2>/dev/null || true
        print_info "✓ 已将冲突目录重命名为备份，现在创建正确的数据库文件..."
        install -m 600 /dev/null config.db
        print_success "✓ 已创建空数据库文件（权限: 600），系统将在启动时初始化"

    # 创建新的数据库文件
    elif [ ! -f "config.db" ]; then
        print_info "📄 数据库文件不存在，创建新的数据库文件..."
        install -m 600 /dev/null config.db
        print_success "✓ 已创建空数据库文件（权限: 600），系统将在启动时初始化"

    # 验证现有数据库文件
    else
        print_info "🔍 验证现有数据库文件..."

        # 检查文件大小
        local db_size=$(stat -f%z "config.db" 2>/dev/null || stat -c%s "config.db" 2>/dev/null || echo "0")

        if [ "$db_size" -eq 0 ]; then
            print_warning "数据库文件为空，系统将自动初始化"
        elif [ "$db_size" -lt 1024 ]; then
            print_warning "数据库文件过小 (${db_size} bytes)，可能未完全初始化"
        else
            print_success "✓ 数据库文件存在 (${db_size} bytes)"

            # 执行数据库健康检查（如果sqlite3可用）
            perform_database_health_check
        fi

        # 验证文件权限
        local db_perm=$(stat -f "%A" "config.db" 2>/dev/null || stat -c "%a" "config.db" 2>/dev/null)
        if [ "$db_perm" != "600" ]; then
            print_warning "数据库文件权限不安全 ($db_perm)，正在修复..."
            chmod 600 config.db
            print_success "✓ 权限已修复为 600"
        fi
    fi

    # 创建数据库备份（仅在服务启动前）
    if [ "$1" = "startup" ] && [ -f "config.db" ] && [ -s "config.db" ]; then
        create_database_backup
    fi
}

# ------------------------------------------------------------------------
# Database Health Check Functions
# ------------------------------------------------------------------------
perform_database_health_check() {
    if ! command -v sqlite3 &> /dev/null; then
        print_info "💡 sqlite3 未安装，跳过数据库完整性检查"
        print_info "   安装 sqlite3 以启用数据库健康检查: sudo apt install sqlite3"
        return 0
    fi

    print_info "🔍 执行数据库完整性检查..."

    # 检查数据库文件是否可以打开
    if ! sqlite3 config.db "SELECT 1;" >/dev/null 2>&1; then
        print_error "❌ 数据库文件损坏或无法读取"
        print_warning "🔄 正在备份数据库并重建..."

        local backup_name="config.db.corrupted.$(date +%Y%m%d_%H%M%S)"
        mv config.db "$backup_name" 2>/dev/null || true
        print_info "✓ 损坏的数据库已备份为: $backup_name"

        # 创建新的数据库文件
        install -m 600 /dev/null config.db
        print_info "✓ 已创建新的数据库文件"
        return 1
    fi

    # 执行完整性检查
    local integrity_result=$(sqlite3 config.db "PRAGMA integrity_check;" 2>/dev/null | head -1)
    if [ "$integrity_result" = "ok" ]; then
        print_success "✅ 数据库完整性检查通过"

        # 检查关键表是否存在
        check_database_tables
    else
        print_error "❌ 数据库完整性检查失败: $integrity_result"
        print_warning "🔄 建议重建数据库"
        return 1
    fi
}

# 检查数据库表结构
check_database_tables() {
    if ! command -v sqlite3 &> /dev/null; then
        return 0
    fi

    print_info "🔍 检查数据库表结构..."

    # 关键表列表
    local required_tables=("users" "ai_models" "exchanges" "system_config")
    local missing_tables=()

    for table in "${required_tables[@]}"; do
        if ! sqlite3 config.db "SELECT name FROM sqlite_master WHERE type='table' AND name='$table';" 2>/dev/null | grep -q "$table"; then
            missing_tables+=("$table")
        fi
    done

    if [ ${#missing_tables[@]} -eq 0 ]; then
        print_success "✅ 所有必需的数据库表都存在"
    else
        print_warning "⚠️  发现缺失的数据库表: ${missing_tables[*]}"
        print_info "💡 系统启动时会自动创建缺失的表"
    fi
}

# 创建数据库备份
create_database_backup() {
    local backup_dir="database_backups"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$backup_dir/config.db.$timestamp"

    # 创建备份目录
    if [ ! -d "$backup_dir" ]; then
        mkdir -p "$backup_dir"
        chmod 700 "$backup_dir"
        print_info "✓ 创建备份目录: $backup_dir"
    fi

    # 创建备份
    if cp config.db "$backup_file" 2>/dev/null; then
        chmod 600 "$backup_file"
        print_success "✅ 数据库已备份到: $backup_file"

        # 清理旧备份（保留最近10个）
        cleanup_old_backups "$backup_dir" 10
    else
        print_warning "⚠️  数据库备份失败"
    fi
}

# 清理旧备份文件
cleanup_old_backups() {
    local backup_dir="$1"
    local keep_count="$2"

    if [ -d "$backup_dir" ]; then
        local backup_count=$(ls -1 "$backup_dir"/config.db.* 2>/dev/null | wc -l)
        if [ "$backup_count" -gt "$keep_count" ]; then
            local delete_count=$((backup_count - keep_count))
            print_info "🗑️  清理 $delete_count 个旧备份文件..."
            ls -1t "$backup_dir"/config.db.* | tail -n "$delete_count" | xargs rm -f 2>/dev/null || true
            print_success "✓ 备份清理完成"
        fi
    fi
}

# 数据库恢复功能
restore_database() {
    local backup_dir="database_backups"

    if [ ! -d "$backup_dir" ]; then
        print_error "备份目录不存在: $backup_dir"
        return 1
    fi

    print_info "📋 可用的数据库备份:"
    ls -la "$backup_dir"/config.db.* 2>/dev/null | nl || {
        print_error "未找到任何数据库备份"
        return 1
    }

    echo ""
    read -p "请选择要恢复的备份编号 (或输入 'q' 取消): " choice

    if [[ "$choice" =~ ^[qQ]$ ]]; then
        print_info "恢复操作已取消"
        return 0
    fi

    if [[ "$choice" =~ ^[0-9]+$ ]]; then
        local backup_file=$(ls -1t "$backup_dir"/config.db.* | sed -n "${choice}p")
        if [ -n "$backup_file" ] && [ -f "$backup_file" ]; then
            print_warning "即将恢复数据库，当前数据库将被覆盖"
            read -p "确认恢复? [y/N]: " confirm

            if [[ "$confirm" =~ ^[Yy]$ ]]; then
                # 备份当前数据库
                if [ -f "config.db" ]; then
                    mv config.db "config.db.before_restore.$(date +%Y%m%d_%H%M%S)"
                fi

                # 恢复备份
                if cp "$backup_file" config.db; then
                    chmod 600 config.db
                    print_success "✅ 数据库恢复成功"
                    print_info "源备份: $backup_file"
                else
                    print_error "❌ 数据库恢复失败"
                    return 1
                fi
            else
                print_info "恢复操作已取消"
            fi
        else
            print_error "无效的备份编号: $choice"
            return 1
        fi
    else
        print_error "请输入有效的数字编号"
        return 1
    fi
}

# ------------------------------------------------------------------------
# Create Necessary Directories
# ------------------------------------------------------------------------
create_necessary_directories() {
    local directories=("decision_logs" "logs" "temp" "secrets")

    for dir in "${directories[@]}"; do
        if [ ! -d "$dir" ]; then
            if [ "$dir" = "secrets" ]; then
                install -m 700 -d "$dir"
                print_info "✓ 创建私密目录: $dir"
            elif [ "$dir" = "temp" ]; then
                install -m 755 -d "$dir"
                print_info "✓ 创建临时目录: $dir"
            else
                install -m 755 -d "$dir"
                print_info "✓ 创建目录: $dir"
            fi
        fi
    done
}

# ------------------------------------------------------------------------
# Read Environment Variables
# ------------------------------------------------------------------------
read_env_vars() {
    if [ -f ".env" ]; then
        NOFX_FRONTEND_PORT=$(grep "^NOFX_FRONTEND_PORT=" .env 2>/dev/null | cut -d'=' -f2 || echo "3000")
        NOFX_BACKEND_PORT=$(grep "^NOFX_BACKEND_PORT=" .env 2>/dev/null | cut -d'=' -f2 || echo "8080")

        # 去除可能的引号和空格
        NOFX_FRONTEND_PORT=$(echo "$NOFX_FRONTEND_PORT" | tr -d '"' | tr -d "'" | tr -d ' ')
        NOFX_BACKEND_PORT=$(echo "$NOFX_BACKEND_PORT" | tr -d '"' | tr -d "'" | tr -d ' ')

        # 如果为空则使用默认值
        NOFX_FRONTEND_PORT=${NOFX_FRONTEND_PORT:-3000}
        NOFX_BACKEND_PORT=${NOFX_BACKEND_PORT:-8080}
    else
        NOFX_FRONTEND_PORT=3000
        NOFX_BACKEND_PORT=8080
    fi

    # 检查端口是否被占用，如果被占用则强制清理
    if is_port_in_use $NOFX_FRONTEND_PORT; then
        print_warning "端口 $NOFX_FRONTEND_PORT 被占用，正在清理..."
        lsof -ti:$NOFX_FRONTEND_PORT | xargs kill -9 2>/dev/null || true
        sleep 1
        if is_port_in_use $NOFX_FRONTEND_PORT; then
             print_error "无法清理端口 $NOFX_FRONTEND_PORT"
             exit 1
        fi
        print_success "端口 $NOFX_FRONTEND_PORT 已清理"
    fi

    if is_port_in_use $NOFX_BACKEND_PORT; then
        print_warning "端口 $NOFX_BACKEND_PORT 被占用，正在清理..."
        lsof -ti:$NOFX_BACKEND_PORT | xargs kill -9 2>/dev/null || true
        sleep 1
        if is_port_in_use $NOFX_BACKEND_PORT; then
             print_error "无法清理端口 $NOFX_BACKEND_PORT"
             exit 1
        fi
        print_success "端口 $NOFX_BACKEND_PORT 已清理"
    fi
}

# ------------------------------------------------------------------------
# Frontend Setup and Build
# --------
setup_frontend() {
    print_info "检查前端环境..."
    cd web

    if [ ! -d "node_modules" ]; then
        print_info "安装前端依赖..."
        npm install
    else
        print_info "前端依赖已安装，检查更新..."
        npm install --silent
    fi

    cd ..
    print_success "前端环境准备完成"
}

# ------------------------------------------------------------------------
# Service Management: Start
# ------------------------------------------------------------------------
start() {
    print_info "🚀 正在启动 NOFX AI Trading System (本地开发模式)..."

    # 读取环境变量
    read_env_vars

    # 创建必要的目录
    create_necessary_directories

    # 执行数据库检查（传入启动标志以触发备份）
    check_database "startup"

    # 设置前端环境
    setup_frontend

    # 构建前端（生产模式）
    if [ "$1" != "--dev" ]; then
        print_info "🔨 构建前端..."
        if ! cd web; then
            print_error "❌ 无法进入 web 目录"
            exit 1
        fi

        if ! npm run build; then
            print_error "❌ 前端构建失败"
            cd ..
            exit 1
        fi

        cd ..
        print_success "✅ 前端构建完成"
    fi

    # 启动后端服务
    print_info "🚀 启动后端服务..."

    # 设置开发模式环境变量
    if [ "$1" == "--dev" ]; then
        export DISABLE_OTP=true
        export NOFX_DEV_MODE=true
        print_info "🚫 开发模式：已禁用2FA验证，已开启详细日志"
    fi

    # 导出端口配置，确保后端能获取到动态分配的端口
    export NOFX_BACKEND_PORT
    export NOFX_FRONTEND_PORT

    # 清理旧的PID文件
    rm -f nofx.pid frontend.pid

    # 启动后端（使用二进制文件或源码运行）
    if [ -f "nofx" ]; then
        print_info "📦 使用编译后的二进制文件启动后端..."
        nohup ./nofx > nofx.log 2>&1 &
        BACKEND_PID=$!
        echo $BACKEND_PID > nofx.pid
    else
        print_info "🔧 使用源码启动后端..."
        if ! command -v go &> /dev/null; then
            print_error "❌ Go 未安装，无法启动后端服务"
            exit 1
        fi

        nohup go run . > nofx.log 2>&1 &
        BACKEND_PID=$!
        echo $BACKEND_PID > nofx.pid
    fi

    # 验证后端启动
    sleep 3
    if ! kill -0 $BACKEND_PID 2>/dev/null; then
        print_error "❌ 后端启动失败，请检查日志: tail -f nofx.log"
        rm -f nofx.pid
        exit 1
    fi
    print_success "✅ 后端服务已启动 (PID: $BACKEND_PID)"

    # 启动前端开发服务器（开发模式）
    if [ "$1" == "--dev" ]; then
        print_info "🔧 启动前端开发服务器..."
        cd web

        # 检查Vite是否可用
        if ! npm list vite &>/dev/null; then
            print_error "❌ Vite 未安装，请先运行: npm install"
            cd ..
            kill $BACKEND_PID 2>/dev/null
            rm -f nofx.pid
            exit 1
        fi

        # Ensure Vite dev server proxy target is aligned with backend
        export VITE_API_URL="http://localhost:${NOFX_BACKEND_PORT}"
        nohup npm run dev > ../frontend.log 2>&1 &
        FRONTEND_PID=$!
        echo $FRONTEND_PID > ../frontend.pid
        cd ..

        # 验证前端启动
        sleep 5
        if ! kill -0 $FRONTEND_PID 2>/dev/null; then
            print_error "❌ 前端开发服务器启动失败，请检查日志: tail -f frontend.log"
            kill $BACKEND_PID 2>/dev/null
            rm -f nofx.pid frontend.pid
            exit 1
        fi
        print_success "✅ 前端开发服务器已启动 (PID: $FRONTEND_PID)"

        # 等待前端热加载完成
        sleep 2
        print_success "🎉 开发服务器已启动！"
    else
        print_success "🎉 生产服务器已启动！"
    fi

    # 最终健康检查
    perform_startup_health_check

    # 显示启动信息
    display_startup_info "$1"
}

# 服务启动健康检查
perform_startup_health_check() {
    print_info "🔍 执行启动健康检查..."

    # 检查后端API
    local max_attempts=10
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -s "http://localhost:${NOFX_BACKEND_PORT}/api/health" >/dev/null 2>&1; then
            print_success "✅ 后端API健康检查通过"
            break
        elif [ $attempt -eq $max_attempts ]; then
            print_warning "⚠️  后端API健康检查超时，但服务可能仍在启动中"
            print_info "   请检查后端日志: tail -f nofx.log"
            break
        else
            print_info "   等待后端API响应... ($attempt/$max_attempts)"
            sleep 2
            attempt=$((attempt + 1))
        fi
    done
}

# 显示启动信息
display_startup_info() {
    local mode="$1"

    echo ""
    print_success "🎯 NOFX AI Trading System 启动完成！"
    echo ""

    if [ "$mode" == "--dev" ]; then
        echo "📱 前端开发服务器: http://localhost:${NOFX_FRONTEND_PORT}"
    else
        echo "🌐 Web 界面: http://localhost:${NOFX_FRONTEND_PORT}"
    fi

    echo "🔗 API 端点: http://localhost:${NOFX_BACKEND_PORT}"
    echo ""

    echo "📊 服务状态:"
    if [ -f "nofx.pid" ]; then
        local backend_pid=$(cat nofx.pid)
        if kill -0 $backend_pid 2>/dev/null; then
            echo "  ✅ 后端服务运行中 (PID: $backend_pid)"
        fi
    fi

    if [ "$mode" == "--dev" ] && [ -f "frontend.pid" ]; then
        local frontend_pid=$(cat frontend.pid)
        if kill -0 $frontend_pid 2>/dev/null; then
            echo "  ✅ 前端开发服务器运行中 (PID: $frontend_pid)"
        fi
    fi

    echo ""
    echo "📋 常用命令:"
    echo "  查看服务状态: ./start.sh status"
    echo "  查看后端日志: tail -f nofx.log"

    if [ "$mode" == "--dev" ]; then
        echo "  查看前端日志: tail -f frontend.log"
    fi

    echo "  停止服务: ./start.sh stop"
    echo "  重启服务: ./start.sh restart"
    echo "  数据库恢复: ./start.sh restore-db"

    echo ""
    if [ "$mode" == "--dev" ]; then
        print_info "💡 开发模式支持热重载，修改代码后会自动更新"
    fi
    print_info "💡 数据库已自动备份到 ./database_backups/ 目录"
}

# ------------------------------------------------------------------------
# Service Management: Stop (Enhanced)
# ------------------------------------------------------------------------
stop() {
    print_info "正在停止所有 NOFX 服务..."

    local stopped_backend=false
    local stopped_frontend=false
    local forced_kill=false

    # 1. 使用PID文件停止进程（如果存在）
    print_info "检查PID文件..."

    # 停止后端
    if [ -f "nofx.pid" ]; then
        local backend_pid=$(cat nofx.pid)
        if kill -0 $backend_pid 2>/dev/null; then
            print_info "终止后端进程 (PID: $backend_pid)..."
            if kill $backend_pid 2>/dev/null; then
                # 等待进程优雅退出
                local count=0
                while kill -0 $backend_pid 2>/dev/null && [ $count -lt 10 ]; do
                    sleep 1
                    count=$((count + 1))
                done

                if kill -0 $backend_pid 2>/dev/null; then
                    print_warning "后端进程未响应SIGTERM，使用SIGKILL强制终止..."
                    kill -9 $backend_pid 2>/dev/null
                    forced_kill=true
                fi
                stopped_backend=true
                print_success "后端服务已停止"
            else
                print_warning "无法终止后端进程 $backend_pid"
            fi
        else
            print_info "后端进程 $backend_pid 已不存在，清理PID文件"
        fi
        rm -f nofx.pid
    else
        print_info "后端PID文件不存在"
    fi

    # 停止前端开发服务器
    if [ -f "frontend.pid" ]; then
        local frontend_pid=$(cat frontend.pid)
        if kill -0 $frontend_pid 2>/dev/null; then
            print_info "终止前端开发服务器 (PID: $frontend_pid)..."
            if kill $frontend_pid 2>/dev/null; then
                # 等待进程优雅退出
                local count=0
                while kill -0 $frontend_pid 2>/dev/null && [ $count -lt 5 ]; do
                    sleep 1
                    count=$((count + 1))
                done

                if kill -0 $frontend_pid 2>/dev/null; then
                    print_warning "前端进程未响应SIGTERM，使用SIGKILL强制终止..."
                    kill -9 $frontend_pid 2>/dev/null
                    forced_kill=true
                fi
                stopped_frontend=true
                print_success "前端开发服务器已停止"
            else
                print_warning "无法终止前端进程 $frontend_pid"
            fi
        else
            print_info "前端进程 $frontend_pid 已不存在，清理PID文件"
        fi
        rm -f frontend.pid
    else
        print_info "前端PID文件不存在"
    fi

    # 2. 端口扫描检测并终止残留进程
    print_info "扫描端口占用情况..."

    # 检查后端端口
    read_env_vars  # 确保端口变量已设置
    if is_port_in_use $NOFX_BACKEND_PORT; then
        print_warning "发现端口 $NOFX_BACKEND_PORT 仍被占用，查找占用进程..."
        local port_pids=$(lsof -ti:$NOFX_BACKEND_PORT 2>/dev/null)
        if [ -n "$port_pids" ]; then
            for pid in $port_pids; do
                local process_name=$(ps -p $pid -o comm= 2>/dev/null)
                print_info "终止占用端口的进程: $pid ($process_name)"
                kill $pid 2>/dev/null || true
                sleep 1
                if kill -0 $pid 2>/dev/null; then
                    print_warning "进程 $pid 未响应，强制终止..."
                    kill -9 $pid 2>/dev/null || true
                    forced_kill=true
                fi
                stopped_backend=true
            done
        fi
    fi

    # 3. 进程名匹配兜底（更全面的模式匹配）
    print_info "执行进程名匹配清理..."

    # 定义进程模式数组
    local process_patterns=(
        "go run \."                    # go run 命令
        "\./nofx"                       # nofx 二进制文件
        "npm run dev"                   # npm dev 命令
        "vite.*--port"                  # vite 开发服务器
        "node.*vite"                    # node vite 进程
    )

    for pattern in "${process_patterns[@]}"; do
        local pids=$(pgrep -f "$pattern" 2>/dev/null || true)
        if [ -n "$pids" ]; then
            for pid in $pids; do
                # 排除当前的shell和编辑器进程
                if [ $pid != $$ ] && ps -p $pid > /dev/null 2>&1; then
                    local cmd=$(ps -p $pid -o command= 2>/dev/null | head -c 100)
                    print_info "终止匹配进程: $pid ($cmd...)"
                    kill $pid 2>/dev/null || true
                    sleep 1
                    if kill -0 $pid 2>/dev/null; then
                        print_warning "进程 $pid 未响应，强制终止..."
                        kill -9 $pid 2>/dev/null || true
                        forced_kill=true
                    fi
                fi
            done
        fi
    done

    # 4. 最终清理和验证
    print_info "执行最终清理..."

    # 清理可能的残留PID文件
    rm -f nofx.pid frontend.pid nofx.log frontend.log

    # 清理临时文件
    find . -name "*.tmp" -delete 2>/dev/null || true
    find . -name ".#*" -delete 2>/dev/null || true

    # 5. 最终验证
    sleep 2
    print_info "验证服务停止状态..."

    local backend_running=false
    local frontend_running=false

    # 检查后端是否还在运行
    if is_port_in_use $NOFX_BACKEND_PORT; then
        backend_running=true
        print_error "⚠️  后端端口 $NOFX_BACKEND_PORT 仍被占用"
        local remaining_pids=$(lsof -ti:$NOFX_BACKEND_PORT 2>/dev/null)
        if [ -n "$remaining_pids" ]; then
            print_error "占用进程: $remaining_pids"
        fi
    else
        print_success "✅ 后端服务已完全停止"
    fi

    # 检查前端是否还在运行
    if is_port_in_use $NOFX_FRONTEND_PORT; then
        frontend_running=true
        print_error "⚠️  前端端口 $NOFX_FRONTEND_PORT 仍被占用"
        local remaining_pids=$(lsof -ti:$NOFX_FRONTEND_PORT 2>/dev/null)
        if [ -n "$remaining_pids" ]; then
            print_error "占用进程: $remaining_pids"
        fi
    else
        print_success "✅ 前端服务已完全停止"
    fi

    # 总结报告
    echo ""
    if [ "$backend_running" = false ] && [ "$frontend_running" = false ]; then
        print_success "🎉 所有 NOFX 服务已成功停止！"
        if [ "$forced_kill" = true ]; then
            print_info "部分进程需要强制终止 (SIGKILL)"
        fi
    else
        print_error "❌ 部分服务未能完全停止"
        print_info "请手动检查上述进程并终止"
        print_info "或者尝试: sudo lsof -ti:$NOFX_BACKEND_PORT | xargs sudo kill -9"
        return 1
    fi
}

# ------------------------------------------------------------------------
# Service Management: Restart
# ------------------------------------------------------------------------
restart() {
    stop
    sleep 1
    start "$1"
}

# ------------------------------------------------------------------------
# Monitoring: Logs
# ------------------------------------------------------------------------
logs() {
    if [ -z "$2" ] || [ "$2" == "backend" ] || [ "$2" == "all" ]; then
        if [ -f "nofx.log" ]; then
            print_info "=== 后端日志 ==="
            tail -f nofx.log
        else
            print_warning "后端日志文件不存在"
        fi
    fi
}

# ------------------------------------------------------------------------
# Monitoring: Status
# ------------------------------------------------------------------------
status() {
    read_env_vars

    print_info "服务状态:"

    # 检查后端
    if [ -f "nofx.pid" ]; then
        local backend_pid=$(cat nofx.pid)
        if kill -0 $backend_pid 2>/dev/null; then
            print_success "后端运行中 (PID: $backend_pid)"
        else
            print_error "后端进程不存在"
            rm -f nofx.pid
        fi
    else
        print_warning "后端未启动"
    fi

    # 检查前端开发服务器
    if [ -f "frontend.pid" ]; then
        local frontend_pid=$(cat frontend.pid)
        if kill -0 $frontend_pid 2>/dev/null; then
            print_success "前端开发服务器运行中 (PID: $frontend_pid)"
        else
            print_error "前端开发服务器进程不存在"
            rm -f frontend.pid
        fi
    else
        if [ "$1" != "--prod" ]; then
            print_info "前端开发服务器未启动"
        fi
    fi

    echo ""
    print_info "端口检查:"
    if is_port_in_use $NOFX_BACKEND_PORT; then
        print_success "后端端口 $NOFX_BACKEND_PORT 正在使用"
    else
        print_warning "后端端口 $NOFX_BACKEND_PORT 未使用"
    fi

    if is_port_in_use $NOFX_FRONTEND_PORT; then
        print_success "前端端口 $NOFX_FRONTEND_PORT 正在使用"
    else
        print_warning "前端端口 $NOFX_FRONTEND_PORT 未使用"
    fi

    echo ""
    print_info "健康检查:"
    if curl -s "http://localhost:${NOFX_BACKEND_PORT}/api/health" >/dev/null; then
        local health=$(curl -s "http://localhost:${NOFX_BACKEND_PORT}/api/health" | jq '.' 2>/dev/null || echo "{}")
        print_success "后端API响应正常"
        echo "$health" | jq '.' 2>/dev/null || echo "后端API正常运行"
    else
        print_error "后端API未响应"
    fi
}

# ------------------------------------------------------------------------
# Build: Production Build
# --------
build() {
    print_info "开始生产构建..."

    # 构建前端
    print_info "构建前端..."
    cd web
    npm run build
    cd ..

    # 构建后端
    print_info "构建后端..."
    go build -o nofx .

    print_success "构建完成！"
    print_info "前端: web/dist/"
    print_info "后端: nofx"
}

# ------------------------------------------------------------------------
# Development: Clean
# --------
clean() {
    print_info "清理构建文件和日志..."

    # 停止服务
    stop

    # 清理文件
    rm -f nofx nofx.log frontend.log
    rm -f nofx.pid frontend.pid
    rm -rf web/dist

    print_success "清理完成"
}

# ------------------------------------------------------------------------
# Encryption: Manual Setup
# ------------------------------------------------------------------------
setup_encryption_manual() {
    print_info "🔐 手动设置加密环境"

    if [ -f "scripts/setup_encryption.sh" ]; then
        bash scripts/setup_encryption.sh
    else
        print_error "加密设置脚本不存在: scripts/setup_encryption.sh"
        print_info "请确保项目文件完整"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# Help: Usage Information
# ------------------------------------------------------------------------
show_help() {
    echo "NOFX AI Trading System - 增强版本地开发管理脚本"
    echo ""
    echo "用法: ./start.sh [command] [options]"
    echo ""
    echo "📋 基础命令:"
    echo "  start [--dev]       启动服务（默认：生产模式，--dev：开发模式）"
    echo "  stop                停止服务"
    echo "  restart [--dev]     重启服务"
    echo "  status [--prod]      查看服务状态"
    echo "  logs [service]      查看日志（backend/all）"
    echo "  build               构建生产版本"
    echo "  clean               清理构建文件和日志"
    echo "  help                显示此帮助信息"
    echo ""
    echo "🔧 高级命令:"
    echo "  setup-encryption    设置加密环境（RSA密钥+数据加密）"
    echo "  restore-db          恢复数据库备份"
    echo ""
    echo "🔍 模式说明:"
    echo "  生产模式: 构建前端静态文件，启动Go后端服务器"
    echo "  开发模式: 启动前端开发服务器(Vite) + Go后端服务器"
    echo ""
    echo "💡 使用示例:"
    echo "  ./start.sh start --dev      # 开发模式启动"
    echo "  ./start.sh start             # 生产模式启动"
    echo "  ./start.sh logs backend      # 查看后端日志"
    echo "  ./start.sh status            # 查看状态"
    echo "  ./start.sh build             # 构建生产版本"
    echo "  ./start.sh restore-db        # 恢复数据库"
    echo ""
    echo "🔐 关于加密:"
    echo "  系统自动检测加密环境，首次运行时会自动设置"
    echo "  手动设置: ./scripts/setup_encryption.sh"
    echo ""
    echo "💾 数据库管理:"
    echo "  自动备份: 每次启动前自动备份数据库"
    echo "  备份位置: ./database_backups/"
    echo "  保留策略: 自动保留最近10个备份"
    echo "  健康检查: 自动验证数据库完整性"
    echo "  sqlite3 可选: 安装后可启用高级健康检查"
}

# ------------------------------------------------------------------------
# Main: Command Dispatcher
# ------------------------------------------------------------------------
main() {
    # 检查基本依赖
    check_nodejs
    check_go

    case "${1:-start}" in
        start)
            print_info "🔍 执行启动前检查..."
            check_env
            check_encryption
            check_config
            # 数据库检查将在 start 函数内部执行，以便触发备份
            start "$2"
            ;;
        stop)
            print_info "🛑 正在停止服务..."
            stop
            ;;
        restart)
            print_info "🔄 正在重启服务..."
            restart "$2"
            ;;
        status)
            print_info "📊 检查服务状态..."
            status "$2"
            ;;
        logs)
            logs "$@"
            ;;
        build)
            print_info "🔨 执行生产构建..."
            check_env
            check_encryption
            check_config
            setup_frontend
            build
            ;;
        clean)
            print_info "🧹 清理项目文件..."
            clean
            ;;
        setup-encryption)
            print_info "🔐 设置加密环境..."
            setup_encryption_manual
            ;;
        restore-db)
            print_info "💾 恢复数据库..."
            restore_database
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "❌ 未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# Execute Main
main "$@"
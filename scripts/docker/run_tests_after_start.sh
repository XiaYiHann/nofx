#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX Docker Post-Start Test Runner
#
# 等待容器健康后，在后端容器内执行 Go 单元测试，确保配置/接口可
# 正常运行。默认仅运行 ./config/... 与 ./api/...，可通过参数或环境
# 变量覆盖。
#
# Usage:
#   ./scripts/docker/run_tests_after_start.sh [OPTIONS]
#
# Options:
#   --wait SECONDS              覆盖等待健康检查的秒数 (默认: $NOFX_AUTO_TEST_WAIT or 120)
#   --packages "LIST"          指定 Go 测试包 (默认: "./config/... ./api/...")
#   --go-flags "FLAGS"         自定义 go test 额外参数 (默认: "-count=1 -timeout 5m")
#   --allow-test-fail          测试失败仅报警不退出 (CI 请慎用)
#   --backend-only-health      仅检查后端健康
#   --frontend-only-health     仅检查前端健康
#   --skip-health              跳过健康检查（假定服务已就绪）
#   -h, --help                 查看帮助
#
# 环境变量：
#   NOFX_AUTO_TEST_AFTER_START=true|false  # start 脚本判断是否调用本脚本
#   NOFX_AUTO_TEST_WAIT=120                # 默认等待时长
#   NOFX_AUTO_TEST_PACKAGES="./config/... ./api/..."
#   NOFX_AUTO_TEST_GOFLAGS="-count=1 -timeout 5m"
#   NOFX_AUTO_TEST_ALLOW_FAIL=true|false   # 默认失败策略
# ═══════════════════════════════════════════════════════════════

set -euo pipefail

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
# Paths & Defaults
# ------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

DEFAULT_PACKAGES="./config/... ./api/..."
DEFAULT_WAIT="120"
DEFAULT_GOFLAGS="-count=1 -timeout 5m"

PACKAGES="${NOFX_AUTO_TEST_PACKAGES:-$DEFAULT_PACKAGES}"
WAIT_SECONDS="${NOFX_AUTO_TEST_WAIT:-$DEFAULT_WAIT}"
GO_TEST_FLAGS="${NOFX_AUTO_TEST_GOFLAGS:-$DEFAULT_GOFLAGS}"
ALLOW_FAIL="$(normalize_bool "${NOFX_AUTO_TEST_ALLOW_FAIL:-false}")"
HEALTH_BACKEND_ONLY="false"
HEALTH_FRONTEND_ONLY="false"
SKIP_HEALTH="false"

# ------------------------------------------------------------------------
# Argument Parsing
# ------------------------------------------------------------------------
show_help() {
        cat <<'EOF'
NOFX Docker Post-Start Test Runner

Usage: ./scripts/docker/run_tests_after_start.sh [OPTIONS]

Options:
    --wait SECONDS              覆盖等待健康检查的秒数 (默认 120 或 NOFX_AUTO_TEST_WAIT)
    --packages "LIST"          指定 go test 包列表 (默认 "./config/... ./api/...")
    --go-flags "FLAGS"         自定义 go test 额外参数 (默认 "-count=1 -timeout 5m")
    --allow-test-fail          测试失败仅报警不中断
    --backend-only-health      健康检查仅验证后端
    --frontend-only-health     健康检查仅验证前端
    --skip-health              跳过健康检查 (假定服务已好)
    -h, --help                 显示本帮助

环境变量 (start.sh/start_docker.sh 会自动传入):
    NOFX_AUTO_TEST_AFTER_START=true|false
    NOFX_AUTO_TEST_WAIT=120
    NOFX_AUTO_TEST_PACKAGES="./config/... ./api/..."
    NOFX_AUTO_TEST_GOFLAGS="-count=1 -timeout 5m"
    NOFX_AUTO_TEST_ALLOW_FAIL=true|false

示例:
    # 手动在部署后执行默认测试
    ./scripts/docker/run_tests_after_start.sh

    # 加长等待时间并扩大测试范围
    ./scripts/docker/run_tests_after_start.sh --wait 180 --packages "./config/... ./api/... ./manager/..."

    # 仅检查后端并允许失败不中断
    ./scripts/docker/run_tests_after_start.sh --backend-only-health --allow-test-fail
EOF
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --wait)
            WAIT_SECONDS="${2:-$WAIT_SECONDS}"
            shift 2
            ;;
        --packages)
            PACKAGES="${2:-$PACKAGES}"
            shift 2
            ;;
        --go-flags)
            GO_TEST_FLAGS="${2:-$GO_TEST_FLAGS}"
            shift 2
            ;;
        --allow-test-fail)
            ALLOW_FAIL="true"
            shift
            ;;
        --backend-only-health)
            HEALTH_BACKEND_ONLY="true"
            shift
            ;;
        --frontend-only-health)
            HEALTH_FRONTEND_ONLY="true"
            shift
            ;;
        --skip-health)
            SKIP_HEALTH="true"
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_warning "未知选项: $1"
            shift
            ;;
    esac
done

if [ -z "$PACKAGES" ]; then
    print_warning "未指定测试包，默认使用 $DEFAULT_PACKAGES"
    PACKAGES="$DEFAULT_PACKAGES"
fi

# ------------------------------------------------------------------------
# Docker Helpers
# ------------------------------------------------------------------------
detect_compose_cmd() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker-compose"
    else
        print_error "Docker Compose 未安装"
        exit 1
    fi
}

ensure_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装"
        exit 1
    fi

    if ! docker ps &> /dev/null; then
        print_error "Docker daemon 未运行，请先启动 Docker"
        exit 1
    fi

    detect_compose_cmd
}

# ------------------------------------------------------------------------
# Health Check Wrapper
# ------------------------------------------------------------------------
run_health_check() {
    if [ "$SKIP_HEALTH" = "true" ]; then
        print_warning "跳过健康检查 (--skip-health)"
        return
    fi

    local health_script="$PROJECT_ROOT/scripts/docker/healthcheck.sh"
    if [ ! -x "$health_script" ]; then
        print_warning "未找到 $health_script，跳过健康检查"
        return
    fi

    local args=("--wait" "$WAIT_SECONDS")
    [ "$HEALTH_BACKEND_ONLY" = "true" ] && args+=("--backend-only")
    [ "$HEALTH_FRONTEND_ONLY" = "true" ] && args+=("--frontend-only")

    print_info "等待服务健康 (healthcheck --wait $WAIT_SECONDS) ..."
    if ! "$health_script" "${args[@]}"; then
        print_error "健康检查失败，可重试: $health_script ${args[*]}"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# Go Test Runner
# ------------------------------------------------------------------------
run_go_tests() {
    local container_cmd="set -euo pipefail; cd /app; go test ${GO_TEST_FLAGS} ${PACKAGES}"
    print_info "🚦 正在运行部署后健康测试 (go test ${PACKAGES})"
    if ! $DOCKER_COMPOSE_CMD exec -T nofx /bin/bash -c "$container_cmd"; then
        if [ "$ALLOW_FAIL" = "true" ]; then
            print_warning "Go 测试失败，但根据配置继续。"
            print_info "可手动重跑: $DOCKER_COMPOSE_CMD exec -T nofx /bin/bash -c \"$container_cmd\""
            return
        fi
        print_error "Go 测试失败"
        print_info "可手动重跑: $DOCKER_COMPOSE_CMD exec -T nofx /bin/bash -c \"$container_cmd\""
        exit 1
    fi
    print_success "部署后测试通过"
}

# ------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------
ensure_docker
run_health_check
run_go_tests

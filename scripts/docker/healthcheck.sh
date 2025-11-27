#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX Docker Health Check Script
#
# Verifies that backend and frontend services are healthy.
#
# Usage:
#   ./scripts/docker/healthcheck.sh [OPTIONS]
#
# Options:
#   --backend-only        Only check backend health
#   --frontend-only       Only check frontend health
#   --wait N              Wait up to N seconds for services (default: 60)
#   --backend-port PORT   Backend port (default: 8080 or from .env)
#   --frontend-port PORT  Frontend port (default: 3000 or from .env)
#   -v, --verbose         Verbose output
#   -h, --help            Show this help message
#
# Exit Codes:
#   0 - All checks passed
#   1 - One or more checks failed
#   2 - Services not running
#
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
print_success() { echo -e "${GREEN}[✓]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[!]${NC} $1"; }
print_error() { echo -e "${RED}[✗]${NC} $1"; }

# ------------------------------------------------------------------------
# Configuration
# ------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Default options
BACKEND_ONLY=false
FRONTEND_ONLY=false
WAIT_SECONDS=60
VERBOSE=false

# Load ports from .env if available
if [ -f "$PROJECT_ROOT/.env" ]; then
    BACKEND_PORT=$(grep -E "^NOFX_BACKEND_PORT=" "$PROJECT_ROOT/.env" | cut -d= -f2 || echo "8080")
    FRONTEND_PORT=$(grep -E "^NOFX_FRONTEND_PORT=" "$PROJECT_ROOT/.env" | cut -d= -f2 || echo "3000")
else
    BACKEND_PORT=8080
    FRONTEND_PORT=3000
fi

# Track failures
FAILURES=0

# ------------------------------------------------------------------------
# Parse Arguments
# ------------------------------------------------------------------------
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --backend-only)
                BACKEND_ONLY=true
                shift
                ;;
            --frontend-only)
                FRONTEND_ONLY=true
                shift
                ;;
            --wait)
                WAIT_SECONDS="${2:-60}"
                shift 2
                ;;
            --backend-port)
                BACKEND_PORT="${2:-8080}"
                shift 2
                ;;
            --frontend-port)
                FRONTEND_PORT="${2:-3000}"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    cat << EOF
NOFX Docker Health Check Script

Usage: ./scripts/docker/healthcheck.sh [OPTIONS]

Options:
  --backend-only        Only check backend health
  --frontend-only       Only check frontend health
  --wait N              Wait up to N seconds for services (default: 60)
  --backend-port PORT   Backend port (default: 8080 or from .env)
  --frontend-port PORT  Frontend port (default: 3000 or from .env)
  -v, --verbose         Verbose output
  -h, --help            Show this help message

Exit Codes:
  0 - All checks passed
  1 - One or more checks failed
  2 - Services not running

Examples:
  # Check all services
  ./scripts/docker/healthcheck.sh

  # Wait up to 2 minutes for startup
  ./scripts/docker/healthcheck.sh --wait 120

  # Check only backend with verbose output
  ./scripts/docker/healthcheck.sh --backend-only --verbose

  # Custom ports
  ./scripts/docker/healthcheck.sh --backend-port 9090 --frontend-port 8000
EOF
}

# ------------------------------------------------------------------------
# Wait for Port
# ------------------------------------------------------------------------
wait_for_port() {
    local port=$1
    local name=$2
    local elapsed=0
    
    print_info "Waiting for $name on port $port..."
    
    while [ $elapsed -lt $WAIT_SECONDS ]; do
        if nc -z localhost "$port" 2>/dev/null; then
            [ "$VERBOSE" = true ] && print_info "$name port $port is open"
            return 0
        fi
        sleep 1
        ((elapsed++))
        [ "$VERBOSE" = true ] && echo -ne "\r  Waiting... ${elapsed}s / ${WAIT_SECONDS}s"
    done
    
    [ "$VERBOSE" = true ] && echo ""
    return 1
}

# ------------------------------------------------------------------------
# Check Backend
# ------------------------------------------------------------------------
check_backend() {
    if [ "$FRONTEND_ONLY" = true ]; then
        return 0
    fi
    
    print_info "Checking backend health (port $BACKEND_PORT)..."
    
    # Wait for port to be available
    if ! wait_for_port "$BACKEND_PORT" "Backend"; then
        print_error "Backend port $BACKEND_PORT not responding after ${WAIT_SECONDS}s"
        ((FAILURES++))
        return 1
    fi
    
    # Check /api/health endpoint
    local response
    local http_code
    
    response=$(curl -s -w "\n%{http_code}" "http://localhost:${BACKEND_PORT}/api/health" 2>/dev/null || echo -e "\n000")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        print_success "Backend /api/health returned 200 OK"
        [ "$VERBOSE" = true ] && echo "  Response: $body"
        return 0
    else
        print_error "Backend /api/health returned HTTP $http_code"
        [ "$VERBOSE" = true ] && echo "  Response: $body"
        ((FAILURES++))
        return 1
    fi
}

# ------------------------------------------------------------------------
# Check Frontend
# ------------------------------------------------------------------------
check_frontend() {
    if [ "$BACKEND_ONLY" = true ]; then
        return 0
    fi
    
    print_info "Checking frontend health (port $FRONTEND_PORT)..."
    
    # Wait for port to be available
    if ! wait_for_port "$FRONTEND_PORT" "Frontend"; then
        print_error "Frontend port $FRONTEND_PORT not responding after ${WAIT_SECONDS}s"
        ((FAILURES++))
        return 1
    fi
    
    # Check root endpoint (should return HTML)
    local response
    local http_code
    
    response=$(curl -s -w "\n%{http_code}" "http://localhost:${FRONTEND_PORT}/" 2>/dev/null || echo -e "\n000")
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Frontend / returned 200 OK"
        
        # Additional check: verify it's actually HTML
        body=$(echo "$response" | sed '$d')
        if echo "$body" | grep -qi "<!DOCTYPE\|<html"; then
            [ "$VERBOSE" = true ] && print_success "  Response contains valid HTML"
        else
            print_warning "  Response may not be valid HTML"
        fi
        return 0
    else
        print_error "Frontend / returned HTTP $http_code"
        ((FAILURES++))
        return 1
    fi
}

# ------------------------------------------------------------------------
# Check Docker Containers
# ------------------------------------------------------------------------
check_containers() {
    print_info "Checking Docker container status..."
    
    local backend_running=false
    local frontend_running=false
    
    # Check backend container
    if docker ps --format '{{.Names}}' | grep -q "nofx-trading\|nofx$"; then
        backend_running=true
        print_success "Backend container is running"
    else
        print_warning "Backend container not found or not running"
    fi
    
    # Check frontend container
    if docker ps --format '{{.Names}}' | grep -q "nofx-frontend"; then
        frontend_running=true
        print_success "Frontend container is running"
    else
        print_warning "Frontend container not found or not running"
    fi
    
    # Check Docker healthcheck status
    if [ "$VERBOSE" = true ]; then
        echo ""
        print_info "Docker health status:"
        docker ps --filter "name=nofx" --format "table {{.Names}}\t{{.Status}}" 2>/dev/null || true
    fi
    
    if [ "$backend_running" = false ] && [ "$frontend_running" = false ]; then
        print_error "No NOFX containers running!"
        print_info "Start services with: docker compose up -d"
        exit 2
    fi
}

# ------------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------------
print_summary() {
    echo ""
    echo "══════════════════════════════════════════════════════════════"
    
    if [ $FAILURES -eq 0 ]; then
        echo -e "  ${GREEN}All health checks passed!${NC}"
    else
        echo -e "  ${RED}${FAILURES} health check(s) failed${NC}"
    fi
    
    echo "══════════════════════════════════════════════════════════════"
    echo ""
    echo "Service URLs:"
    [ "$FRONTEND_ONLY" != true ] && echo "  Backend API:  http://localhost:${BACKEND_PORT}"
    [ "$BACKEND_ONLY" != true ] && echo "  Frontend UI:  http://localhost:${FRONTEND_PORT}"
    echo ""
}

# ------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------
main() {
    parse_args "$@"
    
    echo "══════════════════════════════════════════════════════════════"
    echo "            NOFX Docker Health Check"
    echo "══════════════════════════════════════════════════════════════"
    echo ""
    
    # Check containers first
    check_containers
    echo ""
    
    # Health checks
    check_backend || true
    check_frontend || true
    
    # Summary
    print_summary
    
    exit $FAILURES
}

main "$@"

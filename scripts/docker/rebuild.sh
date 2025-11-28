#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX Docker Rebuild Script
# 
# Safely stops old containers, runs tests, and builds new images
# with proper tagging strategy.
#
# Usage:
#   ./scripts/docker/rebuild.sh [OPTIONS]
#
# Options:
#   --with-integration    Run integration tests (requires LLM_API_KEY)
#   --skip-tests          Skip all tests during build
#   --no-cache            Build without Docker cache
#   --push                Push images to registry after build
#   -h, --help            Show this help message
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
CYAN='\033[0;36m'
NC='\033[0m'

# ------------------------------------------------------------------------
# Utility Functions
# ------------------------------------------------------------------------
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_step() { echo -e "${CYAN}[STEP]${NC} $1"; }

# ------------------------------------------------------------------------
# Configuration
# ------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# Default options
WITH_INTEGRATION=false
SKIP_TESTS=false
NO_CACHE=""
PUSH_IMAGES=false

# Image naming
IMAGE_PREFIX="${IMAGE_PREFIX:-nofx}"
REGISTRY="${REGISTRY:-}"

# ------------------------------------------------------------------------
# Parse Arguments
# ------------------------------------------------------------------------
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --with-integration)
                WITH_INTEGRATION=true
                shift
                ;;
            --skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            --no-cache)
                NO_CACHE="--no-cache"
                shift
                ;;
            --push)
                PUSH_IMAGES=true
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
NOFX Docker Rebuild Script

Usage: ./scripts/docker/rebuild.sh [OPTIONS]

Options:
  --with-integration    Run integration tests (requires LLM_API_KEY, LIVE_TESTS)
                        ⚠️  WARNING: May consume API quotas or make real requests
  --skip-tests          Skip all tests during build
  --no-cache            Build without Docker cache
  --push                Push images to registry after build
  -h, --help            Show this help message

Environment Variables:
  IMAGE_PREFIX          Image name prefix (default: nofx)
  REGISTRY              Docker registry (default: none, local only)
  GOPROXY               Go module proxy (default: https://proxy.golang.org,direct)
  NPM_REGISTRY          NPM registry (default: https://registry.npmjs.org/)

Examples:
  # Standard rebuild with unit tests
  ./scripts/docker/rebuild.sh

  # Full rebuild including integration tests
  ./scripts/docker/rebuild.sh --with-integration

  # Quick rebuild without tests (not recommended for production)
  ./scripts/docker/rebuild.sh --skip-tests

  # Clean rebuild
  ./scripts/docker/rebuild.sh --no-cache

Security Notes:
  - Never commit .env or secrets/ to git
  - Integration tests (--with-integration) may:
    * Consume LLM API quotas (LLM_API_KEY)
    * Make real exchange API calls (LIVE_TESTS=1)
  - CI should NOT enable LIVE_TESTS=1 or expose real credentials
EOF
}

# ------------------------------------------------------------------------
# Generate Image Tag
# ------------------------------------------------------------------------
generate_tag() {
    local branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    local commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    local date=$(date +%Y%m%d)
    
    # Sanitize branch name for Docker tag (replace / with -)
    branch=$(echo "$branch" | tr '/' '-' | tr -cd '[:alnum:]-')
    
    echo "${branch}-${commit}-${date}"
}

# ------------------------------------------------------------------------
# Security Checks
# ------------------------------------------------------------------------
check_secrets() {
    print_step "Checking for secrets leakage..."
    
    # Check if .env is in .gitignore
    if [ -f ".env" ]; then
        if ! grep -q "^\.env$" .gitignore 2>/dev/null; then
            print_warning ".env exists but may not be in .gitignore!"
        fi
    fi
    
    # Check if secrets/ is in .gitignore
    if [ -d "secrets" ]; then
        if ! grep -q "^secrets" .gitignore 2>/dev/null; then
            print_warning "secrets/ directory may not be in .gitignore!"
        fi
    fi
    
    # Check for accidentally staged secrets
    if git diff --cached --name-only 2>/dev/null | grep -qE "^(\.env|secrets/|.*\.key|.*\.pem)$"; then
        print_error "DANGER: Secrets appear to be staged for commit!"
        print_error "Run: git reset HEAD .env secrets/"
        exit 1
    fi
    
    print_success "No obvious secrets leakage detected"
}

# ------------------------------------------------------------------------
# Run Tests
# ------------------------------------------------------------------------
run_tests() {
    if [ "$SKIP_TESTS" = true ]; then
        print_warning "Skipping tests (--skip-tests specified)"
        return 0
    fi
    
    print_step "Running unit tests..."
    
    # Check if Go is available
    if ! command -v go &> /dev/null; then
        print_warning "Go not installed locally, tests will run in Docker build"
        return 0
    fi
    
    # Run short tests (skip integration/E2E by default)
    # These tests should NOT require external services
    print_info "Running: go test ./... -short -count=1"
    
    # Temporarily unset integration test triggers
    (
        unset LLM_API_KEY
        unset LIVE_TESTS
        go test ./... -short -count=1 -timeout 5m
    ) || {
        print_error "Unit tests failed!"
        exit 1
    }
    
    print_success "Unit tests passed"
    
    # Integration tests (optional)
    if [ "$WITH_INTEGRATION" = true ]; then
        print_warning "Running integration tests (--with-integration)..."
        print_warning "⚠️  This may consume API quotas or make real requests!"
        
        if [ -z "${LLM_API_KEY:-}" ]; then
            print_error "LLM_API_KEY not set, cannot run LLM integration tests"
            print_info "Set LLM_API_KEY in your environment or .env file"
            exit 1
        fi
        
        # Load .env if exists
        if [ -f ".env" ]; then
            set -a
            source .env
            set +a
        fi
        
        print_info "Running: go test ./decision/... -v -run TestRealLLM -count=1"
        RUN_REAL_LLM_TESTS=true go test ./decision/... -v -run "TestRealLLM" -count=1 -timeout 10m || {
            print_error "Integration tests failed!"
            exit 1
        }
        
        print_success "Integration tests passed"
    fi
}

# ------------------------------------------------------------------------
# Stop Existing Containers
# ------------------------------------------------------------------------
stop_containers() {
    print_step "Stopping existing containers..."
    
    # Check for compose
    if docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    else
        print_warning "Docker Compose not found, skipping compose down"
        return 0
    fi
    
    # Stop with timeout
    $COMPOSE_CMD down --timeout 30 || true
    
    # Also stop any containers with our naming pattern
    docker ps -aq --filter "name=nofx" | xargs -r docker stop 2>/dev/null || true
    
    print_success "Containers stopped"
}

# ------------------------------------------------------------------------
# Build Images
# ------------------------------------------------------------------------
build_images() {
    local tag=$(generate_tag)
    local backend_image="${IMAGE_PREFIX}/backend:${tag}"
    local frontend_image="${IMAGE_PREFIX}/frontend:${tag}"
    
    # Add registry prefix if specified
    if [ -n "$REGISTRY" ]; then
        backend_image="${REGISTRY}/${backend_image}"
        frontend_image="${REGISTRY}/${frontend_image}"
    fi
    
    print_step "Building images with tag: ${tag}"
    echo ""
    print_info "Backend:  ${backend_image}"
    print_info "Frontend: ${frontend_image}"
    echo ""
    
    # Build backend
    print_step "Building backend image..."
    docker build \
        ${NO_CACHE} \
        --build-arg GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}" \
        -f docker/Dockerfile.backend \
        -t "${backend_image}" \
        -t "${IMAGE_PREFIX}/backend:latest" \
        . || {
        print_error "Backend build failed!"
        exit 1
    }
    print_success "Backend image built: ${backend_image}"
    
    # Build frontend
    print_step "Building frontend image..."
    docker build \
        ${NO_CACHE} \
        --build-arg NPM_REGISTRY="${NPM_REGISTRY:-https://registry.npmjs.org/}" \
        -f docker/Dockerfile.frontend \
        -t "${frontend_image}" \
        -t "${IMAGE_PREFIX}/frontend:latest" \
        . || {
        print_error "Frontend build failed!"
        exit 1
    }
    print_success "Frontend image built: ${frontend_image}"
    
    # Output tags for downstream use
    echo ""
    echo "══════════════════════════════════════════════════════════════"
    echo -e "${GREEN}Build Complete!${NC}"
    echo "══════════════════════════════════════════════════════════════"
    echo ""
    echo "Images built:"
    echo "  Backend:  ${backend_image}"
    echo "  Frontend: ${frontend_image}"
    echo ""
    echo "Also tagged as :latest"
    echo ""
    
    # Export for scripts
    export NOFX_BACKEND_IMAGE="${backend_image}"
    export NOFX_FRONTEND_IMAGE="${frontend_image}"
    export NOFX_IMAGE_TAG="${tag}"
    
    # Save to file for CI
    echo "NOFX_BACKEND_IMAGE=${backend_image}" > .build-info
    echo "NOFX_FRONTEND_IMAGE=${frontend_image}" >> .build-info
    echo "NOFX_IMAGE_TAG=${tag}" >> .build-info
    
    # Push if requested
    if [ "$PUSH_IMAGES" = true ]; then
        print_step "Pushing images to registry..."
        docker push "${backend_image}"
        docker push "${frontend_image}"
        print_success "Images pushed"
    fi
}

# ------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------
main() {
    parse_args "$@"
    
    echo "══════════════════════════════════════════════════════════════"
    echo "            NOFX Docker Rebuild Script"
    echo "══════════════════════════════════════════════════════════════"
    echo ""
    
    # Pre-flight checks
    check_secrets
    
    # Run tests
    run_tests
    
    # Stop existing
    stop_containers
    
    # Build new images
    build_images
    
    echo ""
    print_info "To start services with new images:"
    echo "  docker compose up -d"
    echo ""
    print_info "To verify health:"
    echo "  ./scripts/docker/healthcheck.sh"
    echo ""
}

main "$@"

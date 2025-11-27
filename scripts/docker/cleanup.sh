#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX Docker Cleanup Script
#
# Safely removes old containers and images with safeguards.
#
# Usage:
#   ./scripts/docker/cleanup.sh [OPTIONS]
#
# Options:
#   --dry-run             Show what would be deleted without deleting
#   --force               Skip confirmation prompts
#   --keep-latest N       Keep the N most recent images (default: 3)
#   --all                 Remove ALL nofx images (use with caution)
#   --containers-only     Only remove containers, not images
#   --images-only         Only remove images, not containers
#   --prune               Also run docker system prune
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
print_dry_run() { echo -e "${CYAN}[DRY-RUN]${NC} Would: $1"; }

# ------------------------------------------------------------------------
# Configuration
# ------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Default options
DRY_RUN=false
FORCE=false
KEEP_LATEST=3
REMOVE_ALL=false
CONTAINERS_ONLY=false
IMAGES_ONLY=false
RUN_PRUNE=false

# Image patterns to match
IMAGE_PATTERNS=("nofx/backend" "nofx/frontend" "nofx-trading" "nofx-frontend")

# ------------------------------------------------------------------------
# Parse Arguments
# ------------------------------------------------------------------------
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            --keep-latest)
                KEEP_LATEST="${2:-3}"
                shift 2
                ;;
            --all)
                REMOVE_ALL=true
                KEEP_LATEST=0
                shift
                ;;
            --containers-only)
                CONTAINERS_ONLY=true
                shift
                ;;
            --images-only)
                IMAGES_ONLY=true
                shift
                ;;
            --prune)
                RUN_PRUNE=true
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
NOFX Docker Cleanup Script

Usage: ./scripts/docker/cleanup.sh [OPTIONS]

Options:
  --dry-run             Show what would be deleted without deleting
  --force               Skip confirmation prompts
  --keep-latest N       Keep the N most recent images (default: 3)
  --all                 Remove ALL nofx images (use with caution)
  --containers-only     Only remove containers, not images
  --images-only         Only remove images, not containers
  --prune               Also run docker system prune
  -h, --help            Show this help message

Examples:
  # Preview what would be cleaned up
  ./scripts/docker/cleanup.sh --dry-run

  # Remove old images, keeping 3 most recent
  ./scripts/docker/cleanup.sh

  # Remove all images without confirmation
  ./scripts/docker/cleanup.sh --all --force

  # Only stop/remove containers
  ./scripts/docker/cleanup.sh --containers-only

  # Full cleanup including docker prune
  ./scripts/docker/cleanup.sh --prune --force

Safety:
  - Default mode keeps 3 most recent images
  - Always prompts for confirmation unless --force
  - --dry-run shows exactly what would be deleted
  - Running containers are stopped before removal
EOF
}

# ------------------------------------------------------------------------
# Confirmation
# ------------------------------------------------------------------------
confirm() {
    local message="$1"
    if [ "$FORCE" = true ]; then
        return 0
    fi
    if [ "$DRY_RUN" = true ]; then
        return 0
    fi
    
    echo -e "${YELLOW}${message}${NC}"
    read -p "Continue? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Aborted by user"
        exit 0
    fi
}

# ------------------------------------------------------------------------
# List NOFX Resources
# ------------------------------------------------------------------------
list_containers() {
    # Find containers matching our patterns
    local containers=()
    
    # By name
    for name in "nofx-trading" "nofx-frontend" "nofx"; do
        local found=$(docker ps -aq --filter "name=^${name}$" 2>/dev/null || true)
        if [ -n "$found" ]; then
            containers+=($found)
        fi
    done
    
    # By image prefix
    for pattern in "${IMAGE_PATTERNS[@]}"; do
        local found=$(docker ps -aq --filter "ancestor=${pattern}" 2>/dev/null || true)
        if [ -n "$found" ]; then
            containers+=($found)
        fi
    done
    
    # Deduplicate and output
    printf '%s\n' "${containers[@]}" | sort -u
}

list_images() {
    local images=()
    
    for pattern in "${IMAGE_PATTERNS[@]}"; do
        local found=$(docker images --format "{{.ID}}\t{{.Repository}}:{{.Tag}}\t{{.CreatedAt}}" \
            --filter "reference=${pattern}*" 2>/dev/null || true)
        if [ -n "$found" ]; then
            images+=("$found")
        fi
    done
    
    # Also match any image with nofx in the name
    local nofx_images=$(docker images --format "{{.ID}}\t{{.Repository}}:{{.Tag}}\t{{.CreatedAt}}" \
        | grep -i "nofx" || true)
    if [ -n "$nofx_images" ]; then
        images+=("$nofx_images")
    fi
    
    # Sort by creation time (newest first) and deduplicate
    printf '%s\n' "${images[@]}" | sort -t$'\t' -k3 -r | cut -f1,2 | sort -u
}

# ------------------------------------------------------------------------
# Clean Containers
# ------------------------------------------------------------------------
clean_containers() {
    if [ "$IMAGES_ONLY" = true ]; then
        return 0
    fi
    
    print_info "Finding NOFX containers..."
    
    local containers=$(list_containers)
    
    if [ -z "$containers" ]; then
        print_info "No NOFX containers found"
        return 0
    fi
    
    echo ""
    print_info "Found containers:"
    for cid in $containers; do
        local cname=$(docker inspect --format '{{.Name}}' "$cid" 2>/dev/null | tr -d '/')
        local cimage=$(docker inspect --format '{{.Config.Image}}' "$cid" 2>/dev/null)
        local cstatus=$(docker inspect --format '{{.State.Status}}' "$cid" 2>/dev/null)
        echo "  - $cid ($cname) [$cimage] - $cstatus"
    done
    echo ""
    
    if [ "$DRY_RUN" = true ]; then
        print_dry_run "Stop and remove ${#containers[@]} container(s)"
        return 0
    fi
    
    confirm "This will stop and remove the above containers."
    
    for cid in $containers; do
        print_info "Stopping container: $cid"
        docker stop "$cid" --time 30 2>/dev/null || true
        print_info "Removing container: $cid"
        docker rm -f "$cid" 2>/dev/null || true
    done
    
    print_success "Containers cleaned"
}

# ------------------------------------------------------------------------
# Clean Images
# ------------------------------------------------------------------------
clean_images() {
    if [ "$CONTAINERS_ONLY" = true ]; then
        return 0
    fi
    
    print_info "Finding NOFX images..."
    
    # Get all matching images with their info
    local all_images=$(docker images --format "{{.ID}}\t{{.Repository}}:{{.Tag}}\t{{.CreatedAt}}" \
        | grep -i "nofx" | sort -t$'\t' -k3 -r || true)
    
    if [ -z "$all_images" ]; then
        print_info "No NOFX images found"
        return 0
    fi
    
    # Split into backend and frontend lists
    local backend_images=$(echo "$all_images" | grep -i "backend" || true)
    local frontend_images=$(echo "$all_images" | grep -i "frontend" || true)
    local other_images=$(echo "$all_images" | grep -iv "backend\|frontend" || true)
    
    # Determine what to keep and what to remove
    local to_remove=()
    local to_keep=()
    
    # Process backend images
    local count=0
    while IFS=$'\t' read -r id name created; do
        if [ -z "$id" ]; then continue; fi
        if [ "$REMOVE_ALL" = true ] || [ $count -ge $KEEP_LATEST ]; then
            to_remove+=("$id:$name")
        else
            to_keep+=("$id:$name")
        fi
        ((count++)) || true
    done <<< "$backend_images"
    
    # Process frontend images
    count=0
    while IFS=$'\t' read -r id name created; do
        if [ -z "$id" ]; then continue; fi
        if [ "$REMOVE_ALL" = true ] || [ $count -ge $KEEP_LATEST ]; then
            to_remove+=("$id:$name")
        else
            to_keep+=("$id:$name")
        fi
        ((count++)) || true
    done <<< "$frontend_images"
    
    # Other images (always remove if --all, otherwise keep)
    while IFS=$'\t' read -r id name created; do
        if [ -z "$id" ]; then continue; fi
        if [ "$REMOVE_ALL" = true ]; then
            to_remove+=("$id:$name")
        else
            to_keep+=("$id:$name")
        fi
    done <<< "$other_images"
    
    echo ""
    if [ ${#to_keep[@]} -gt 0 ]; then
        print_info "Keeping (most recent $KEEP_LATEST per type):"
        for item in "${to_keep[@]}"; do
            echo "  ✓ ${item#*:}"
        done
    fi
    
    if [ ${#to_remove[@]} -eq 0 ]; then
        print_info "No images to remove"
        return 0
    fi
    
    echo ""
    print_warning "To be removed:"
    for item in "${to_remove[@]}"; do
        echo "  ✗ ${item#*:}"
    done
    echo ""
    
    if [ "$DRY_RUN" = true ]; then
        print_dry_run "Remove ${#to_remove[@]} image(s)"
        return 0
    fi
    
    confirm "This will remove ${#to_remove[@]} image(s)."
    
    for item in "${to_remove[@]}"; do
        local id="${item%%:*}"
        local name="${item#*:}"
        print_info "Removing image: $name"
        docker rmi -f "$id" 2>/dev/null || print_warning "Could not remove $name (may be in use)"
    done
    
    print_success "Images cleaned"
}

# ------------------------------------------------------------------------
# Docker Prune
# ------------------------------------------------------------------------
run_docker_prune() {
    if [ "$RUN_PRUNE" != true ]; then
        return 0
    fi
    
    print_info "Running docker system prune..."
    
    if [ "$DRY_RUN" = true ]; then
        print_dry_run "docker system prune -f"
        return 0
    fi
    
    if [ "$FORCE" != true ]; then
        confirm "This will remove unused Docker resources (networks, volumes, build cache)."
    fi
    
    docker system prune -f
    print_success "Docker system pruned"
}

# ------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------
main() {
    parse_args "$@"
    
    echo "══════════════════════════════════════════════════════════════"
    echo "            NOFX Docker Cleanup Script"
    echo "══════════════════════════════════════════════════════════════"
    echo ""
    
    if [ "$DRY_RUN" = true ]; then
        print_warning "DRY-RUN MODE - No changes will be made"
        echo ""
    fi
    
    # Check docker availability
    if ! command -v docker &> /dev/null; then
        print_error "Docker not found!"
        exit 1
    fi
    
    # Clean resources
    clean_containers
    clean_images
    run_docker_prune
    
    echo ""
    print_success "Cleanup complete!"
    
    if [ "$DRY_RUN" = true ]; then
        echo ""
        print_info "Run without --dry-run to actually clean up"
    fi
}

main "$@"

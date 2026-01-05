#!/bin/bash

# NVIDIA Qwen E2E Test Script
# This script automates the process of testing the NVIDIA Qwen backtest functionality.

set -e

# Configuration
API_URL="http://localhost:8080/api"
EMAIL="test_e2e_$(date +%s)@example.com"
PASSWORD="password123"
MODEL_ID="nvidia-qwen"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 1. Check Health
log_info "Checking backend health..."
if ! curl -s "${API_URL}/health" | grep -q 'ok'; then
    log_error "Backend is not healthy or not running on :8080"
    exit 1
fi
log_success "Backend is healthy."

# 2. Register (Optional, handled in case of non-dev mode)
log_info "Registering test user: ${EMAIL}..."
REGISTER_RES=$(curl -s -X POST "${API_URL}/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"${EMAIL}\", \"password\": \"${PASSWORD}\"}")

if echo "$REGISTER_RES" | grep -q 'error'; then
    log_info "Registration note: $(echo "$REGISTER_RES" | jq -r '.error // "Already registered or error skipped"')"
else
    log_success "User registered successfully."
fi

# 3. Login
log_info "Logging in..."
LOGIN_RES=$(curl -s -X POST "${API_URL}/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"${EMAIL}\", \"password\": \"${PASSWORD}\"}")

USER_ID=$(echo "$LOGIN_RES" | jq -r '.user_id')
if [ "$USER_ID" == "null" ] || [ -z "$USER_ID" ]; then
    log_error "Login failed: $LOGIN_RES"
    exit 1
fi

# 4. Verify OTP (Since disableOTP=true, we can just login again or if it requires OTP, we might need a workaround)
# In dev mode, we can skip this and just use dev-user if we want, but let's try to get a token if possible.
# Wait, handleVerifyOTP is required if handleLogin returns requires_otp=true.
# But for now, let's see if we can just use dev-user bypass by NOT providing a token.
# Actually, let's try to get a token via handleVerifyOTP with some dummy code if disableOTP is on?
# No, let's just use the devMode bypass for simplicity in this E2E test script.

# If we are in dev mode, we don't need token.
# Let's check if we can get a token with a dummy code.
log_info "Attempting to get token with dummy OTP (since disableOTP is likely on)..."
VERIFY_RES=$(curl -s -X POST "${API_URL}/verify-otp" \
    -H "Content-Type: application/json" \
    -d "{\"user_id\": \"${USER_ID}\", \"otp_code\": \"123456\"}")

TOKEN=$(echo "$VERIFY_RES" | jq -r '.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    log_info "Could not get token via OTP (expected if not in dev mode or setup not complete)."
    log_info "Falling back to dev-user bypass (no token)."
    AUTH_HEADER=""
else
    log_success "Token obtained successfully."
    AUTH_HEADER="Authorization: Bearer ${TOKEN}"
fi

# 5. Create Backtest
log_info "Creating backtest with model: ${MODEL_ID}..."
# Use a short time range for fast testing (e.g., 2 hours ago to 1 hour ago)
END_TIME=$(date -u -v-1M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "1 minute ago" +%Y-%m-%dT%H:%M:%SZ)
START_TIME=$(date -u -v-31M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "31 minutes ago" +%Y-%m-%dT%H:%M:%SZ)

BACKTEST_REQ=$(cat <<EOF
{
    "start_time": "${START_TIME}",
    "end_time": "${END_TIME}",
    "initial_balance": 10000,
    "ai_model_id": "${MODEL_ID}",
    "use_trader_config": false,
    "indicator_config": {},
    "symbols": ["BTCUSDT"]
}
EOF
)

CREATE_RES=$(curl -s -X POST "${API_URL}/backtest" \
    -H "Content-Type: application/json" \
    -H "$AUTH_HEADER" \
    -d "$BACKTEST_REQ")

BACKTEST_ID=$(echo "$CREATE_RES" | jq -r '.backtest_id')

if [ "$BACKTEST_ID" == "null" ] || [ -z "$BACKTEST_ID" ]; then
    log_error "Failed to create backtest: $CREATE_RES"
    exit 1
fi

log_success "Backtest created! ID: ${BACKTEST_ID}"

# 6. Poll Status
log_info "Polling backtest status..."
MAX_ATTEMPTS=60
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    STATUS_RES=$(curl -s -X GET "${API_URL}/backtest/${BACKTEST_ID}" \
        -H "$AUTH_HEADER")
    
    STATUS=$(echo "$STATUS_RES" | jq -r '.status')
    PROGRESS=$(echo "$STATUS_RES" | jq -r '.progress')
    
    log_info "Progress: ${PROGRESS}% (Status: ${STATUS})"
    
    if [ "$STATUS" == "completed" ]; then
        log_success "Backtest completed!"
        break
    elif [ "$STATUS" == "failed" ]; then
        ERROR=$(echo "$STATUS_RES" | jq -r '.error')
        log_error "Backtest failed: ${ERROR}"
        exit 1
    fi
    
    sleep 5
    ATTEMPT=$((ATTEMPT+1))
done

if [ $ATTEMPT -ge $MAX_ATTEMPTS ]; then
    log_error "Backtest timed out."
    exit 1
fi

# 7. Check Decisions
log_info "Checking backtest decisions..."
DECISIONS_RES=$(curl -s -X GET "${API_URL}/backtest/${BACKTEST_ID}/decisions" \
    -H "$AUTH_HEADER")

DECISION_COUNT=$(echo "$DECISIONS_RES" | jq '. | length')
log_info "Found ${DECISION_COUNT} decisions."

if [ "$DECISION_COUNT" -gt 0 ]; then
    log_success "E2E Test Passed: Backtest generated decisions with NVIDIA Qwen!"
else
    log_warning "Backtest finished but generated no decisions. This might be normal for short periods, but check logs if unexpected."
fi

# Show final stats
echo -e "\n${GREEN}=== Final Summary ===${NC}"
curl -s -X GET "${API_URL}/backtest/${BACKTEST_ID}" -H "$AUTH_HEADER" | jq '{id, status, total_trades, final_equity, total_pnl_pct}'

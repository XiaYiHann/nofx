package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"nofx/config"
	"nofx/crypto"
	"nofx/manager"
	"nofx/testhelpers"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// setupBacktestTest creates a test server with a logged-in user and AI model
func setupBacktestTest(t *testing.T) (*Server, string, string, *config.Database, func()) {
	db, cleanup := testhelpers.SetupTestDB(t)

	os.Setenv("DATA_ENCRYPTION_KEY", "test-encryption-key-for-backtest-tests")

	tmpKey := t.TempDir() + "/test_key"
	cs, err := crypto.NewCryptoService(tmpKey)
	require.NoError(t, err)
	db.SetCryptoService(cs)

	tm := manager.NewTraderManager()
	server := NewServer(tm, db, cs, 0, true) // disableOTP=true

	// Enable registration and create a test user
	db.SetSystemConfig("registration_enabled", "true")

	regBody := map[string]string{
		"email":    "backtest_test@example.com",
		"password": "password123",
	}
	w := testhelpers.PerformRequest(server.router, "POST", "/api/register", regBody)
	require.Equal(t, http.StatusOK, w.Code, "Register failed: %s", w.Body.String())

	var regResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &regResp)
	require.NoError(t, err)
	token, ok := regResp["token"].(string)
	require.True(t, ok, "Token not found in response")

	userID, ok := regResp["user_id"].(string)
	require.True(t, ok, "user_id not found in response")

	return server, token, userID, db, func() {
		os.Unsetenv("DATA_ENCRYPTION_KEY")
		cleanup()
	}
}

// performAuthRequest performs a request with JWT auth header
func performAuthRequest(server *Server, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, path, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	server.router.ServeHTTP(w, req)
	return w
}

// createTestAIModelInDB creates a test AI model directly in database and returns its ID
func createTestAIModelInDB(t *testing.T, db *config.Database, userID string) string {
	aiModelID := "test-model-" + time.Now().Format("20060102150405")
	err := db.CreateAIModel(userID, aiModelID, "Test AI Model", "openai", true, "test-api-key", "https://api.openai.com/v1")
	require.NoError(t, err, "Failed to create AI model in database")
	return aiModelID
}

// createTestExchangeInDB creates a test exchange directly in database and returns its ID
func createTestExchangeInDB(t *testing.T, db *config.Database, userID string) string {
	exchangeID := "test-exchange-" + time.Now().Format("20060102150405")
	err := db.CreateExchange(userID, exchangeID, "Test Exchange", "binance_futures_testnet", true, "test-api-key", "test-api-secret", false, "", "", "", "")
	require.NoError(t, err, "Failed to create exchange in database")
	return exchangeID
}

func TestCreateBacktest_StandaloneWithAIModelOnly(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestTest(t)
	defer cleanup()

	// Create AI model directly in database
	aiModelID := createTestAIModelInDB(t, db, userID)

	// Test: standalone backtest with only ai_model_id (no exchange_id)
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"ai_model_id":     aiModelID,
		"start_time":      startTime,
		"end_time":        endTime,
		"initial_balance": 10000.0,
		// No trader_id (standalone mode)
		// No exchange_id (should be optional now)
	}

	w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusCreated, w.Code, "Create backtest failed: %s", w.Body.String())

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Verify response contains backtest_id
	backtestID, ok := resp["backtest_id"].(string)
	require.True(t, ok, "backtest_id not found in response")
	require.NotEmpty(t, backtestID)

	// Verify status
	status, ok := resp["status"].(string)
	require.True(t, ok, "status not found in response")
	require.Equal(t, "pending", status)
}

func TestCreateBacktest_StandaloneMissingAIModel(t *testing.T) {
	server, token, _, _, cleanup := setupBacktestTest(t)
	defer cleanup()

	// Test: standalone backtest without ai_model_id (should fail)
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"start_time":      startTime,
		"end_time":        endTime,
		"initial_balance": 10000.0,
		// No trader_id (standalone mode)
		// No ai_model_id (should fail)
	}

	w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 Bad Request")

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	errorMsg, ok := resp["error"].(string)
	require.True(t, ok, "error message not found in response")
	require.Contains(t, errorMsg, "ai_model_id is required")
}

func TestCreateBacktest_StandaloneWithExchange(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestTest(t)
	defer cleanup()

	// Create AI model and exchange directly in database
	aiModelID := createTestAIModelInDB(t, db, userID)
	exchangeID := createTestExchangeInDB(t, db, userID)

	// Test: standalone backtest with both ai_model_id and exchange_id
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"ai_model_id":     aiModelID,
		"exchange_id":     exchangeID,
		"start_time":      startTime,
		"end_time":        endTime,
		"initial_balance": 10000.0,
		// No trader_id (standalone mode)
	}

	w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusCreated, w.Code, "Create backtest failed: %s", w.Body.String())

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	backtestID, ok := resp["backtest_id"].(string)
	require.True(t, ok, "backtest_id not found in response")
	require.NotEmpty(t, backtestID)
}

func TestCreateBacktest_TraderBasedNotRegressed(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestTest(t)
	defer cleanup()

	// Create AI model and exchange directly in database
	aiModelID := createTestAIModelInDB(t, db, userID)
	exchangeID := createTestExchangeInDB(t, db, userID)

	// Create a trader via API
	traderBody := map[string]interface{}{
		"name":                  "Test Trader",
		"ai_model_id":           aiModelID,
		"exchange_id":           exchangeID,
		"initial_balance":       10000.0,
		"scan_interval_minutes": 5,
		"btc_eth_leverage":      5,
		"altcoin_leverage":      3,
		"trading_symbols":       "BTCUSDT",
	}
	w := performAuthRequest(server, "POST", "/api/traders", token, traderBody)
	require.Equal(t, http.StatusCreated, w.Code, "Create trader failed: %s", w.Body.String())

	var traderResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &traderResp)
	require.NoError(t, err)
	traderID, ok := traderResp["trader_id"].(string)
	require.True(t, ok, "trader_id not found in response")

	// Test: trader-based backtest (should still work)
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"trader_id":         traderID,
		"start_time":        startTime,
		"end_time":          endTime,
		"initial_balance":   10000.0,
		"use_trader_config": true,
	}

	w = performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusCreated, w.Code, "Create trader-based backtest failed: %s", w.Body.String())

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	backtestID, ok := resp["backtest_id"].(string)
	require.True(t, ok, "backtest_id not found in response")
	require.NotEmpty(t, backtestID)
}

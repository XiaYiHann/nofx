package api

import (
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// 回测API综合测试套件
// 测试所有回测相关的API端点
// ============================================================================

// TestBacktestAPIComprehensiveSuite 运行完整的回测API测试
func TestBacktestAPIComprehensiveSuite(t *testing.T) {
	t.Run("CreateBacktest", testCreateBacktestAPI)
	t.Run("GetBacktests", testGetBacktestsAPI)
	t.Run("GetBacktestDetail", testGetBacktestDetailAPI)
	t.Run("DeleteBacktest", testDeleteBacktestAPI)
	t.Run("GetBacktestEquityHistory", testGetBacktestEquityHistoryAPI)
	t.Run("GetBacktestTrades", testGetBacktestTradesAPI)
	t.Run("GetBacktestDecisions", testGetBacktestDecisionsAPI)
	t.Run("BacktestValidation", testBacktestValidation)
	t.Run("BacktestErrorHandling", testBacktestErrorHandling)
}

// setupBacktestAPITest 创建测试服务器和认证token
func setupBacktestAPITest(t *testing.T) (*Server, string, string, *config.Database, func()) {
	db, cleanup := testhelpers.SetupTestDB(t)

	os.Setenv("DATA_ENCRYPTION_KEY", "test-encryption-key-for-api-tests")

	tmpKey := t.TempDir() + "/test_key"
	cs, err := crypto.NewCryptoService(tmpKey)
	require.NoError(t, err)
	db.SetCryptoService(cs)

	tm := manager.NewTraderManager()
	server := NewServer(tm, db, cs, 0, true)

	// 启用注册
	db.SetSystemConfig("registration_enabled", "true")

	// 创建测试用户
	regBody := map[string]string{
		"email":    "backtest_api_test@example.com",
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

// createTestAIModel 创建测试AI模型
func createTestAIModel(t *testing.T, db *config.Database, userID string) string {
	aiModelID := "test-model-" + time.Now().Format("20060102150405")
	err := db.CreateAIModel(userID, aiModelID, "Test AI Model", "openai", true, "test-api-key", "https://api.openai.com/v1")
	require.NoError(t, err)
	return aiModelID
}

// createTestExchange 创建测试交易所
func createTestExchange(t *testing.T, db *config.Database, userID string) string {
	exchangeID := "test-exchange-" + time.Now().Format("20060102150405")
	err := db.CreateExchange(userID, exchangeID, "Test Exchange", "binance_futures_testnet", true, "test-api-key", "test-api-secret", false, "", "", "", "")
	require.NoError(t, err)
	return exchangeID
}

// createTestTrader 创建测试交易员
func createTestTrader(t *testing.T, server *Server, token, aiModelID, exchangeID string) string {
	traderBody := map[string]interface{}{
		"name":                  "Test Trader",
		"ai_model_id":           aiModelID,
		"exchange_id":           exchangeID,
		"initial_balance":       10000.0,
		"scan_interval_minutes": 5,
		"btc_eth_leverage":      5,
		"altcoin_leverage":      5,
		"trading_symbols":       "BTCUSDT",
	}
	w := performAuthRequest(server, "POST", "/api/traders", token, traderBody)
	require.Equal(t, http.StatusCreated, w.Code, "Create trader failed: %s", w.Body.String())

	var traderResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &traderResp)
	require.NoError(t, err)
	traderID, ok := traderResp["trader_id"].(string)
	require.True(t, ok)
	return traderID
}

// ============================================================================
// 1. 创建回测API测试
// ============================================================================

func testCreateBacktestAPI(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	aiModelID := createTestAIModel(t, db, userID)
	exchangeID := createTestExchange(t, db, userID)

	t.Run("StandaloneMode", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)

		backtestBody := map[string]interface{}{
			"ai_model_id":     aiModelID,
			"start_time":      startTime,
			"end_time":        endTime,
			"initial_balance": 10000.0,
			"timeframe":       "3m",
			"scan_interval_minutes": 15,
			"btc_eth_leverage": 5,
			"altcoin_leverage": 10,
			"trading_symbols":  "BTCUSDT,ETHUSDT",
		}

		w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
		assert.Equal(t, http.StatusCreated, w.Code, "Create standalone backtest failed: %s", w.Body.String())

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		backtestID, ok := resp["backtest_id"].(string)
		assert.True(t, ok, "backtest_id not found")
		assert.NotEmpty(t, backtestID)

		status, ok := resp["status"].(string)
		assert.True(t, ok, "status not found")
		assert.Equal(t, "pending", status)
	})

	t.Run("TraderBasedMode", func(t *testing.T) {
		traderID := createTestTrader(t, server, token, aiModelID, exchangeID)

		now := time.Now()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)

		backtestBody := map[string]interface{}{
			"trader_id":       traderID,
			"start_time":      startTime,
			"end_time":        endTime,
			"initial_balance": 10000.0,
			"use_trader_config": true,
		}

		w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
		assert.Equal(t, http.StatusCreated, w.Code, "Create trader-based backtest failed: %s", w.Body.String())

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		backtestID, ok := resp["backtest_id"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, backtestID)
	})

	t.Run("MissingAIModel", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)

		backtestBody := map[string]interface{}{
			"start_time":      startTime,
			"end_time":        endTime,
			"initial_balance": 10000.0,
		}

		w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 Bad Request")

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		errorMsg, ok := resp["error"].(string)
		assert.True(t, ok, "error message not found")
		assert.Contains(t, errorMsg, "ai_model_id")
	})

	t.Run("InvalidTimeRange", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(24 * time.Hour).Format(time.RFC3339) // 未来时间
		endTime := now.Format(time.RFC3339)

		backtestBody := map[string]interface{}{
			"ai_model_id":     aiModelID,
			"start_time":      startTime,
			"end_time":        endTime,
			"initial_balance": 10000.0,
		}

		w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 Bad Request for invalid time range")
	})
}

// ============================================================================
// 2. 获取回测列表API测试
// ============================================================================

func testGetBacktestsAPI(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	aiModelID := createTestAIModel(t, db, userID)

	// 创建几个回测
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	for i := 0; i < 3; i++ {
		backtestBody := map[string]interface{}{
			"ai_model_id":     aiModelID,
			"start_time":      startTime,
			"end_time":        endTime,
			"initial_balance": 10000.0,
		}
		performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	}

	// 获取回测列表
	w := performAuthRequest(server, "GET", "/api/backtests", token, nil)
	assert.Equal(t, http.StatusOK, w.Code, "Get backtests failed: %s", w.Body.String())

	var resp []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(resp), 3, "Expected at least 3 backtests")
}

// ============================================================================
// 3. 获取回测详情API测试
// ============================================================================

func testGetBacktestDetailAPI(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	aiModelID := createTestAIModel(t, db, userID)

	// 创建回测
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"ai_model_id":     aiModelID,
		"start_time":      startTime,
		"end_time":        endTime,
		"initial_balance": 10000.0,
	}
	w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	backtestID, _ := createResp["backtest_id"].(string)

	// 获取详情
	w = performAuthRequest(server, "GET", "/api/backtest/"+backtestID, token, nil)
	assert.Equal(t, http.StatusOK, w.Code, "Get backtest detail failed: %s", w.Body.String())

	var detailResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &detailResp)
	require.NoError(t, err)

	assert.Equal(t, backtestID, detailResp["id"])
	assert.Equal(t, "pending", detailResp["status"])
	assert.Equal(t, 10000.0, detailResp["initial_balance"])
}

// ============================================================================
// 4. 删除回测API测试
// ============================================================================

func testDeleteBacktestAPI(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	aiModelID := createTestAIModel(t, db, userID)

	// 创建回测
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"ai_model_id":     aiModelID,
		"start_time":      startTime,
		"end_time":        endTime,
		"initial_balance": 10000.0,
	}
	w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	backtestID, _ := createResp["backtest_id"].(string)

	// 删除回测
	w = performAuthRequest(server, "DELETE", "/api/backtest/"+backtestID, token, nil)
	assert.Equal(t, http.StatusOK, w.Code, "Delete backtest failed: %s", w.Body.String())

	// 验证已删除
	w = performAuthRequest(server, "GET", "/api/backtest/"+backtestID, token, nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "Expected 404 after deletion")
}

// ============================================================================
// 5. 获取净值历史API测试
// ============================================================================

func testGetBacktestEquityHistoryAPI(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	aiModelID := createTestAIModel(t, db, userID)

	// 创建回测
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"ai_model_id":     aiModelID,
		"start_time":      startTime,
		"end_time":        endTime,
		"initial_balance": 10000.0,
	}
	w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	backtestID, _ := createResp["backtest_id"].(string)

	// 获取净值历史
	w = performAuthRequest(server, "GET", "/api/backtest/"+backtestID+"/equity", token, nil)
	// 可能返回空数组（回测未开始）
	assert.Equal(t, http.StatusOK, w.Code, "Get equity history failed: %s", w.Body.String())

	var historyResp []interface{}
	err = json.Unmarshal(w.Body.Bytes(), &historyResp)
	require.NoError(t, err)
}

// ============================================================================
// 6. 获取交易记录API测试
// ============================================================================

func testGetBacktestTradesAPI(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	aiModelID := createTestAIModel(t, db, userID)

	// 创建回测
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"ai_model_id":     aiModelID,
		"start_time":      startTime,
		"end_time":        endTime,
		"initial_balance": 10000.0,
	}
	w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	backtestID, _ := createResp["backtest_id"].(string)

	// 获取交易记录
	w = performAuthRequest(server, "GET", "/api/backtest/"+backtestID+"/trades", token, nil)
	assert.Equal(t, http.StatusOK, w.Code, "Get trades failed: %s", w.Body.String())

	var tradesResp []interface{}
	err = json.Unmarshal(w.Body.Bytes(), &tradesResp)
	require.NoError(t, err)
}

// ============================================================================
// 7. 获取决策记录API测试
// ============================================================================

func testGetBacktestDecisionsAPI(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	aiModelID := createTestAIModel(t, db, userID)

	// 创建回测
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Format(time.RFC3339)

	backtestBody := map[string]interface{}{
		"ai_model_id":     aiModelID,
		"start_time":      startTime,
		"end_time":        endTime,
		"initial_balance": 10000.0,
	}
	w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	backtestID, _ := createResp["backtest_id"].(string)

	// 获取决策记录
	w = performAuthRequest(server, "GET", "/api/backtest/"+backtestID+"/decisions", token, nil)
	assert.Equal(t, http.StatusOK, w.Code, "Get decisions failed: %s", w.Body.String())

	var decisionsResp []interface{}
	err = json.Unmarshal(w.Body.Bytes(), &decisionsResp)
	require.NoError(t, err)
}

// ============================================================================
// 8. 回测验证测试
// ============================================================================

func testBacktestValidation(t *testing.T) {
	server, token, userID, db, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	aiModelID := createTestAIModel(t, db, userID)

	t.Run("NegativeBalance", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)

		backtestBody := map[string]interface{}{
			"ai_model_id":     aiModelID,
			"start_time":      startTime,
			"end_time":        endTime,
			"initial_balance": -100.0, // 负数
		}

		w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 for negative balance")
	})

	t.Run("ZeroBalance", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)

		backtestBody := map[string]interface{}{
			"ai_model_id":     aiModelID,
			"start_time":      startTime,
			"end_time":        endTime,
			"initial_balance": 0.0, // 零
		}

		w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 for zero balance")
	})

	t.Run("InvalidLeverage", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
		endTime := now.Format(time.RFC3339)

		backtestBody := map[string]interface{}{
			"ai_model_id":     aiModelID,
			"start_time":      startTime,
			"end_time":        endTime,
			"initial_balance": 10000.0,
			"btc_eth_leverage": 200, // 超过限制
		}

		w := performAuthRequest(server, "POST", "/api/backtest", token, backtestBody)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 for invalid leverage")
	})
}

// ============================================================================
// 9. 错误处理测试
// ============================================================================

func testBacktestErrorHandling(t *testing.T) {
	server, _, _, _, cleanup := setupBacktestAPITest(t)
	defer cleanup()

	t.Run("Unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/backtests", nil)
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Expected 401 Unauthorized")
	})

	t.Run("InvalidBacktestID", func(t *testing.T) {
		// 需要有效的token
		server, token, _, _, cleanup := setupBacktestAPITest(t)
		defer cleanup()

		w := performAuthRequest(server, "GET", "/api/backtest/invalid-id", token, nil)
		assert.Equal(t, http.StatusNotFound, w.Code, "Expected 404 for invalid backtest ID")
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		server, token, _, _, cleanup := setupBacktestAPITest(t)
		defer cleanup()

		w := performAuthRequest(server, "POST", "/api/backtest", token, "invalid json")
		assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 for malformed JSON")
	})
}
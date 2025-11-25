package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nofx/auth"
	"nofx/config"
	"nofx/crypto"
	"nofx/manager"
	"nofx/testhelpers"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
)

const defaultDataEncryptionKey = "0123456789abcdef0123456789abcdef"

func TestServerRealLoginAndProtectedEndpoint(t *testing.T) {
	t.Helper()
	testhelpers.LoadDotEnv(t)

	if os.Getenv("DATA_ENCRYPTION_KEY") == "" {
		os.Setenv("DATA_ENCRYPTION_KEY", defaultDataEncryptionKey)
	}

	cryptoService, err := crypto.NewCryptoService("secrets/rsa_key")
	require.NoError(t, err)

	dbPath := filepath.Join(t.TempDir(), "config.db")
	database, err := config.NewDatabase(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	database.SetCryptoService(cryptoService)

	auth.SetJWTSecret("test-jwt-secret")
	traderManager := manager.NewTraderManager()
	server := NewServer(traderManager, database, cryptoService, 0, true)

	password := "RealTestPass#123"
	passwordHash, err := auth.HashPassword(password)
	require.NoError(t, err)

	// Generate OTP secret for the test user
	otpSecret, err := auth.GenerateOTPSecret()
	require.NoError(t, err)

	user := &config.User{
		ID:           "api-test-user",
		Email:        "api-test@example.com",
		PasswordHash: passwordHash,
		OTPSecret:    otpSecret,
		OTPVerified:  true,
	}
	require.NoError(t, database.CreateUser(user))

	aiModelID := "api-test-model"
	require.NoError(t, database.CreateAIModel(user.ID, aiModelID, "API Test Model", "deepseek", true, "sk-test", ""))

	exchangeID := "api-test-exchange"
	require.NoError(t, database.CreateExchange(user.ID, exchangeID, "API Test Exchange", "binance", true, "binance-key", "binance-secret", true, "", "", "", ""))

	traderRecord := &config.TraderRecord{
		ID:                   "api-trader",
		UserID:               user.ID,
		Name:                 "API Real Trader",
		AIModelID:            aiModelID,
		ExchangeID:           exchangeID,
		InitialBalance:       1000,
		ScanIntervalMinutes:  3,
		BTCETHLeverage:       5,
		AltcoinLeverage:      3,
		TradingSymbols:       "BTCUSDT",
		SystemPromptTemplate: "default",
		IsCrossMargin:        true,
	}
	require.NoError(t, database.CreateTrader(traderRecord))

	// Step 1: Login (returns requires_otp: true)
	loginPayload := map[string]string{"email": user.Email, "password": password}
	body, err := json.Marshal(loginPayload)
	require.NoError(t, err)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	server.router.ServeHTTP(loginResp, loginReq)
	require.Equal(t, http.StatusOK, loginResp.Code)

	var loginStepData struct {
		UserID      string `json:"user_id"`
		RequiresOTP bool   `json:"requires_otp"`
	}
	require.NoError(t, json.Unmarshal(loginResp.Body.Bytes(), &loginStepData))
	require.True(t, loginStepData.RequiresOTP, "Login should require OTP verification")
	require.Equal(t, user.ID, loginStepData.UserID)

	// Step 2: Verify OTP to get token
	otpCode, err := totp.GenerateCode(otpSecret, time.Now())
	require.NoError(t, err)

	verifyPayload := map[string]string{"user_id": user.ID, "otp_code": otpCode}
	verifyBody, err := json.Marshal(verifyPayload)
	require.NoError(t, err)

	verifyReq := httptest.NewRequest(http.MethodPost, "/api/verify-otp", bytes.NewReader(verifyBody))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyResp := httptest.NewRecorder()
	server.router.ServeHTTP(verifyResp, verifyReq)
	require.Equal(t, http.StatusOK, verifyResp.Code)

	var loginData struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	require.NoError(t, json.Unmarshal(verifyResp.Body.Bytes(), &loginData))
	require.NotEmpty(t, loginData.Token)
	require.Equal(t, user.ID, loginData.UserID)
	require.Equal(t, user.Email, loginData.Email)

	tradersReq := httptest.NewRequest(http.MethodGet, "/api/my-traders", nil)
	tradersReq.Header.Set("Authorization", "Bearer "+loginData.Token)
	tradersResp := httptest.NewRecorder()
	server.router.ServeHTTP(tradersResp, tradersReq)
	require.Equal(t, http.StatusOK, tradersResp.Code)

	var traderList []struct {
		TraderID       string  `json:"trader_id"`
		AIModel        string  `json:"ai_model"`
		ExchangeID     string  `json:"exchange_id"`
		InitialBalance float64 `json:"initial_balance"`
	}
	require.NoError(t, json.Unmarshal(tradersResp.Body.Bytes(), &traderList))
	require.Len(t, traderList, 1)
	require.Equal(t, traderRecord.ID, traderList[0].TraderID)
	require.Equal(t, traderRecord.AIModelID, traderList[0].AIModel)
	require.Equal(t, traderRecord.ExchangeID, traderList[0].ExchangeID)
	require.Equal(t, traderRecord.InitialBalance, traderList[0].InitialBalance)
}

// TestUpdateTraderSystemPromptTemplatePersistence tests that system_prompt_template
// is correctly persisted when updating a trader via PUT /api/traders/:id
func TestUpdateTraderSystemPromptTemplatePersistence(t *testing.T) {
	t.Helper()
	testhelpers.LoadDotEnv(t)

	if os.Getenv("DATA_ENCRYPTION_KEY") == "" {
		os.Setenv("DATA_ENCRYPTION_KEY", defaultDataEncryptionKey)
	}

	cryptoService, err := crypto.NewCryptoService("secrets/rsa_key")
	require.NoError(t, err)

	dbPath := filepath.Join(t.TempDir(), "config.db")
	database, err := config.NewDatabase(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	database.SetCryptoService(cryptoService)

	auth.SetJWTSecret("test-jwt-secret-update")
	traderManager := manager.NewTraderManager()
	server := NewServer(traderManager, database, cryptoService, 0, true)

	// Create test user with OTP secret
	password := "UpdateTestPass#123"
	passwordHash, err := auth.HashPassword(password)
	require.NoError(t, err)

	otpSecret, err := auth.GenerateOTPSecret()
	require.NoError(t, err)

	user := &config.User{
		ID:           "update-test-user",
		Email:        "update-test@example.com",
		PasswordHash: passwordHash,
		OTPSecret:    otpSecret,
		OTPVerified:  true,
	}
	require.NoError(t, database.CreateUser(user))

	// Create AI model and exchange
	aiModelID := "update-test-model"
	require.NoError(t, database.CreateAIModel(user.ID, aiModelID, "Update Test Model", "deepseek", true, "sk-test", ""))

	exchangeID := "update-test-exchange"
	require.NoError(t, database.CreateExchange(user.ID, exchangeID, "Update Test Exchange", "binance", true, "binance-key", "binance-secret", true, "", "", "", ""))

	// Create trader with default system_prompt_template
	traderRecord := &config.TraderRecord{
		ID:                   "update-test-trader",
		UserID:               user.ID,
		Name:                 "Update Test Trader",
		AIModelID:            aiModelID,
		ExchangeID:           exchangeID,
		InitialBalance:       1000,
		ScanIntervalMinutes:  5,
		BTCETHLeverage:       10,
		AltcoinLeverage:      5,
		TradingSymbols:       "BTCUSDT",
		SystemPromptTemplate: "default", // Initial value
		IsCrossMargin:        true,
	}
	require.NoError(t, database.CreateTrader(traderRecord))

	// Step 1: Login (returns requires_otp: true)
	loginPayload := map[string]string{"email": user.Email, "password": password}
	body, err := json.Marshal(loginPayload)
	require.NoError(t, err)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	server.router.ServeHTTP(loginResp, loginReq)
	require.Equal(t, http.StatusOK, loginResp.Code)

	var loginStepData struct {
		UserID      string `json:"user_id"`
		RequiresOTP bool   `json:"requires_otp"`
	}
	require.NoError(t, json.Unmarshal(loginResp.Body.Bytes(), &loginStepData))
	require.True(t, loginStepData.RequiresOTP, "Login should require OTP verification")

	// Step 2: Verify OTP to get token
	otpCode, err := totp.GenerateCode(otpSecret, time.Now())
	require.NoError(t, err)

	verifyPayload := map[string]string{"user_id": user.ID, "otp_code": otpCode}
	verifyBody, err := json.Marshal(verifyPayload)
	require.NoError(t, err)

	verifyReq := httptest.NewRequest(http.MethodPost, "/api/verify-otp", bytes.NewReader(verifyBody))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyResp := httptest.NewRecorder()
	server.router.ServeHTTP(verifyResp, verifyReq)
	require.Equal(t, http.StatusOK, verifyResp.Code)

	var loginData struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(verifyResp.Body.Bytes(), &loginData))
	require.NotEmpty(t, loginData.Token)

	// Update trader with new system_prompt_template = "aggressive"
	updatePayload := map[string]interface{}{
		"name":                   "Update Test Trader",
		"ai_model_id":            aiModelID,
		"exchange_id":            exchangeID,
		"initial_balance":        1000,
		"scan_interval_minutes":  5,
		"btc_eth_leverage":       10,
		"altcoin_leverage":       5,
		"trading_symbols":        "BTCUSDT",
		"system_prompt_template": "aggressive", // New value
		"is_cross_margin":        true,
	}
	updateBody, err := json.Marshal(updatePayload)
	require.NoError(t, err)

	updateReq := httptest.NewRequest(http.MethodPut, "/api/traders/"+traderRecord.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+loginData.Token)
	updateResp := httptest.NewRecorder()
	server.router.ServeHTTP(updateResp, updateReq)
	require.Equal(t, http.StatusOK, updateResp.Code, "Update should succeed, got: %s", updateResp.Body.String())

	// Verify by calling GET /api/traders/:id/config
	configReq := httptest.NewRequest(http.MethodGet, "/api/traders/"+traderRecord.ID+"/config", nil)
	configReq.Header.Set("Authorization", "Bearer "+loginData.Token)
	configResp := httptest.NewRecorder()
	server.router.ServeHTTP(configResp, configReq)
	require.Equal(t, http.StatusOK, configResp.Code, "Get config should succeed, got: %s", configResp.Body.String())

	var configData map[string]interface{}
	require.NoError(t, json.Unmarshal(configResp.Body.Bytes(), &configData))

	// Assert: system_prompt_template should be "aggressive"
	systemPromptTemplate, ok := configData["system_prompt_template"].(string)
	require.True(t, ok, "system_prompt_template should be a string in response")
	require.Equal(t, "aggressive", systemPromptTemplate, "system_prompt_template should be updated to 'aggressive'")

	// Also verify directly from database
	traders, err := database.GetTraders(user.ID)
	require.NoError(t, err)
	require.Len(t, traders, 1)
	require.Equal(t, "aggressive", traders[0].SystemPromptTemplate, "Database should have updated system_prompt_template")
}

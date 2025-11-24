package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nofx/auth"
	"nofx/config"
	"nofx/crypto"
	"nofx/manager"
	"nofx/testhelpers"

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

	user := &config.User{
		ID:           "api-test-user",
		Email:        "api-test@example.com",
		PasswordHash: passwordHash,
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

	loginPayload := map[string]string{"email": user.Email, "password": password}
	body, err := json.Marshal(loginPayload)
	require.NoError(t, err)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	server.router.ServeHTTP(loginResp, loginReq)
	require.Equal(t, http.StatusOK, loginResp.Code)

	var loginData struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	require.NoError(t, json.Unmarshal(loginResp.Body.Bytes(), &loginData))
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

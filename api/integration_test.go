package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"nofx/crypto"
	"nofx/manager"
	"nofx/testhelpers"
	"testing"
)

func TestAPIIntegration(t *testing.T) {
	// Setup dependencies
	db, cleanup := testhelpers.SetupTestDB(t)
	defer cleanup()

	// Create crypto service
	tmpKey := t.TempDir() + "/test_key"
	cs, err := crypto.NewCryptoService(tmpKey)
	if err != nil {
		t.Fatal(err)
	}
	db.SetCryptoService(cs)

	tm := manager.NewTraderManager()
	server := NewServer(tm, db, cs, 0, true) // 0 port, disableOTP=true

	// Helper to perform requests
	perform := func(method, path string, body interface{}) *httptest.ResponseRecorder {
		return testhelpers.PerformRequest(server.router, method, path, body)
	}

	t.Run("HealthCheck", func(t *testing.T) {
		w := perform("GET", "/api/health", nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("PublicConfig", func(t *testing.T) {
		w := perform("GET", "/api/config", nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if _, ok := resp["default_coins"]; !ok {
			t.Error("Expected default_coins in response")
		}
	})

	t.Run("RegisterAndLogin", func(t *testing.T) {
		// Enable registration
		db.SetSystemConfig("registration_enabled", "true")

		// Register
		regBody := map[string]string{
			"email": "api_test@example.com",
			"password": "password123",
		}
		w := perform("POST", "/api/register", regBody)
		if w.Code != http.StatusOK {
			t.Logf("Register failed: %s", w.Body.String())
			t.Fatalf("Register failed with code %d", w.Code)
		}

		var regResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &regResp)
		if _, ok := regResp["token"]; !ok {
			t.Error("Token should be present in registration response")
		}

		// Login
		loginBody := map[string]string{
			"email": "api_test@example.com",
			"password": "password123",
		}
		wLogin := perform("POST", "/api/login", loginBody)
		if wLogin.Code != http.StatusOK {
			t.Errorf("Login failed: %d", wLogin.Code)
		}
	})
}

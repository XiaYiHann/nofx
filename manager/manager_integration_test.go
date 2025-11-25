package manager

import (
	"nofx/config"
	"nofx/crypto"
	"os"
	"testing"
)

func setupTestDB(t *testing.T) (*config.Database, func()) {
	// Create temp file for DB
	tmpfile, err := os.CreateTemp("", "test_nofx_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	dbPath := tmpfile.Name()
	tmpfile.Close()

	// Create temp file for RSA key
	rsaKeyFile, err := os.CreateTemp("", "test_rsa_*.pem")
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to create temp RSA key file: %v", err)
	}
	rsaKeyPath := rsaKeyFile.Name()
	rsaKeyFile.Close()
	os.Remove(rsaKeyPath) // Remove it so NewCryptoService generates a new one

	// Set env var for data key
	os.Setenv("DATA_ENCRYPTION_KEY", "test-key-123456789012345678901234")

	// Init DB
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("Failed to init database: %v", err)
	}

	// Init Crypto
	cs, err := crypto.NewCryptoService(rsaKeyPath)
	if err != nil {
		db.Close()
		os.Remove(dbPath)
		os.Remove(rsaKeyPath)
		t.Fatalf("Failed to init crypto service: %v", err)
	}
	db.SetCryptoService(cs)

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
		os.Remove(rsaKeyPath)
		os.Unsetenv("DATA_ENCRYPTION_KEY")
	}

	return db, cleanup
}

func TestAddTraderFromDB(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tm := NewTraderManager()

	// Setup test data
	userID := "user1"
	traderID := "trader1"

	traderCfg := &config.TraderRecord{
		ID:              traderID,
		UserID:          userID,
		Name:            "Test Trader",
		ExchangeID:      "binance1",
		AIModelID:       "deepseek1",
		IsRunning:       false,
		InitialBalance:  1000.0,
		BTCETHLeverage:  5,
		AltcoinLeverage: 5,
	}

	aiModelCfg := &config.AIModelConfig{
		ID:       "deepseek1",
		Provider: "deepseek",
		APIKey:   "sk-test",
	}

	exchangeCfg := &config.ExchangeConfig{
		ID:        "binance",
		Type:      "binance",
		APIKey:    "apikey",
		SecretKey: "secret",
	}

	err := tm.AddTraderFromDB(traderCfg, aiModelCfg, exchangeCfg, "http://pool", "http://oi", 10.0, 20.0, 60, []string{"BTCUSDT"}, db, userID)
	if err != nil {
		t.Fatalf("AddTraderFromDB failed: %v", err)
	}

	// Verify trader exists
	trader, err := tm.GetTrader(traderID)
	if err != nil {
		t.Errorf("GetTrader failed: %v", err)
	}
	if trader == nil {
		t.Error("Trader should not be nil")
	}
}

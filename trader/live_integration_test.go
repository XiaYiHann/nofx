package trader

import (
	"os"
	"testing"
)

func TestLiveIntegration(t *testing.T) {
	if os.Getenv("LIVE_TESTS") != "1" {
		t.Skip("Skipping live integration test. Set LIVE_TESTS=1 to run.")
	}

	t.Run("ConnectBinancePublic", func(t *testing.T) {
		// Use dummy keys. Public endpoints should work or return specific error.
		// Note: NewFuturesTrader attempts to set DualSidePosition which requires valid keys.
		// It logs error but returns the instance.
		ft := NewFuturesTrader("dummy_key", "dummy_secret", "test_user")

		// GetMarketPrice calls ListPricesService which is public on Binance Futures
		price, err := ft.GetMarketPrice("BTCUSDT")
		if err != nil {
			t.Fatalf("GetMarketPrice failed (check network connection): %v", err)
		}

		t.Logf("Current BTCUSDT Price: %.2f", price)
		if price <= 0 {
			t.Errorf("Price should be positive, got %.2f", price)
		}
	})
}

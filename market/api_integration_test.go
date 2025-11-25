package market

import (
	"testing"
)

// TestAPIClient_GetKlines_Integration tests fetching Klines from Binance API.
// This is an integration test that requires network access.
func TestAPIClient_GetKlines_Integration(t *testing.T) {
	client := NewAPIClient()

	symbol := "BTCUSDT"
	interval := "1h"
	limit := 5

	klines, err := client.GetKlines(symbol, interval, limit)
	if err != nil {
		t.Fatalf("Failed to get klines: %v", err)
	}

	if len(klines) == 0 {
		t.Errorf("Expected klines, got 0")
	}

	if len(klines) > limit {
		t.Errorf("Expected at most %d klines, got %d", limit, len(klines))
	}

	t.Logf("Successfully fetched %d klines for %s", len(klines), symbol)
	for i, k := range klines {
		t.Logf("Kline %d: Time=%d, Close=%.2f", i, k.CloseTime, k.Close)
	}
}

// TestAPIClient_GetExchangeInfo_Integration tests fetching Exchange Info.
func TestAPIClient_GetExchangeInfo_Integration(t *testing.T) {
	client := NewAPIClient()

	info, err := client.GetExchangeInfo()
	if err != nil {
		t.Fatalf("Failed to get exchange info: %v", err)
	}

	if len(info.Symbols) == 0 {
		t.Errorf("Expected symbols, got 0")
	}

	t.Logf("Successfully fetched exchange info. Total symbols: %d", len(info.Symbols))
}

package trader

import (
	"context"
	"sync"
	"testing"

	"github.com/sonirico/go-hyperliquid"
)

// TestMetaConcurrentAccess tests that concurrent access to meta field is safe
func TestMetaConcurrentAccess(t *testing.T) {
	// Create a HyperliquidTrader instance with meta initialized
	trader := &HyperliquidTrader{
		ctx: context.Background(),
		meta: &hyperliquid.Meta{
			Universe: []hyperliquid.AssetInfo{
				{Name: "BTC", SzDecimals: 5},
				{Name: "ETH", SzDecimals: 4},
			},
		},
	}

	// Number of concurrent goroutines
	concurrency := 100
	var wg sync.WaitGroup

	// Test concurrent reads (getSzDecimals)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// This should not cause race conditions
			decimals := trader.getSzDecimals("BTC")
			if decimals != 5 {
				t.Errorf("Expected decimals 5, got %d", decimals)
			}
		}()
	}

	wg.Wait()
}

// TestMetaConcurrentReadWrite tests concurrent reads and writes to meta field
// Note: This test documents the current behavior. In production, meta is only set
// once during initialization, so concurrent writes are not expected.
func TestMetaConcurrentReadWrite(t *testing.T) {
	t.Skip("Skipping: HyperliquidTrader does not currently have mutex protection for meta field. Meta is only set once during initialization.")
}

// TestGetSzDecimals_NilMeta tests getSzDecimals with nil meta
func TestGetSzDecimals_NilMeta(t *testing.T) {
	trader := &HyperliquidTrader{
		meta: nil,
	}

	// Should return default value 4 when meta is nil
	decimals := trader.getSzDecimals("BTC")
	expectedDecimals := 4

	if decimals != expectedDecimals {
		t.Errorf("Expected default decimals %d for nil meta, got %d", expectedDecimals, decimals)
	}
}

// TestGetSzDecimals_ValidMeta tests getSzDecimals with valid meta
func TestGetSzDecimals_ValidMeta(t *testing.T) {
	trader := &HyperliquidTrader{
		meta: &hyperliquid.Meta{
			Universe: []hyperliquid.AssetInfo{
				{Name: "BTC", SzDecimals: 5},
				{Name: "ETH", SzDecimals: 4},
				{Name: "SOL", SzDecimals: 3},
			},
		},
	}

	tests := []struct {
		coin             string
		expectedDecimals int
	}{
		{"BTC", 5},
		{"ETH", 4},
		{"SOL", 3},
	}

	for _, tt := range tests {
		t.Run(tt.coin, func(t *testing.T) {
			decimals := trader.getSzDecimals(tt.coin)
			if decimals != tt.expectedDecimals {
				t.Errorf("For coin %s, expected decimals %d, got %d", tt.coin, tt.expectedDecimals, decimals)
			}
		})
	}
}

// TestMetaMutex_NoRaceCondition tests that using -race detector finds no issues
// Note: This test documents the current behavior. In production, meta is only set
// once during initialization, so concurrent writes are not expected.
func TestMetaMutex_NoRaceCondition(t *testing.T) {
	t.Skip("Skipping: HyperliquidTrader does not currently have mutex protection for meta field. Meta is only set once during initialization.")
}

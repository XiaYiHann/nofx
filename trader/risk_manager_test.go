package trader

import (
	"testing"
)

func TestCheckLeverage(t *testing.T) {
	rm := NewRiskManager(50, 20) // BTC/ETH 50x, Alt 20x

	tests := []struct {
		name     string
		symbol   string
		leverage int
		wantErr  bool
	}{
		{"BTC within limit", "BTCUSDT", 50, false},
		{"BTC exceeds limit", "BTCUSDT", 51, true},
		{"ETH within limit", "ETHUSDT", 50, false},
		{"ETH exceeds limit", "ETHUSDT", 51, true},
		{"Alt within limit", "SOLUSDT", 20, false},
		{"Alt exceeds limit", "SOLUSDT", 21, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := rm.CheckLeverage(tt.symbol, tt.leverage); (err != nil) != tt.wantErr {
				t.Errorf("RiskManager.CheckLeverage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPositionSize(t *testing.T) {
	rm := NewRiskManager(50, 20)
	equity := 1000.0

	tests := []struct {
		name    string
		symbol  string
		sizeUSD float64
		wantErr bool
	}{
		{"BTC within limit (10x)", "BTCUSDT", 10000.0, false},
		{"BTC exceeds limit", "BTCUSDT", 10001.0, true},
		{"Alt within limit (1.5x)", "SOLUSDT", 1500.0, false},
		{"Alt exceeds limit", "SOLUSDT", 1501.0, true},
		{"Zero equity", "BTCUSDT", 100.0, true}, // Should fail or handle gracefully (test setup uses 1000.0, but we can override)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentEquity := equity
			if tt.name == "Zero equity" {
				currentEquity = 0
			}
			if err := rm.CheckPositionSize(tt.symbol, tt.sizeUSD, currentEquity); (err != nil) != tt.wantErr {
				t.Errorf("RiskManager.CheckPositionSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckRiskReward(t *testing.T) {
	rm := NewRiskManager(50, 20)
	rm.MinRiskRewardRatio = 2.0

	tests := []struct {
		name       string
		entry      float64
		stopLoss   float64
		takeProfit float64
		side       string
		wantErr    bool
	}{
		{"Long Valid R:R 2.0", 100, 90, 120, "long", false}, // Risk 10, Reward 20 -> 2.0
		{"Long Invalid R:R 1.5", 100, 90, 115, "long", true}, // Risk 10, Reward 15 -> 1.5
		{"Short Valid R:R 2.0", 100, 110, 80, "short", false}, // Risk 10, Reward 20 -> 2.0
		{"Short Invalid R:R 1.0", 100, 110, 90, "short", true}, // Risk 10, Reward 10 -> 1.0
		{"Invalid Side", 100, 90, 120, "invalid", true},
		{"Long Invalid StopLoss", 100, 110, 120, "long", true}, // SL > Entry
		{"Short Invalid StopLoss", 100, 90, 80, "short", true}, // SL < Entry
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := rm.CheckRiskReward(tt.entry, tt.stopLoss, tt.takeProfit, tt.side); (err != nil) != tt.wantErr {
				t.Errorf("RiskManager.CheckRiskReward() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

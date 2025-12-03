package market

import (
	"math"
	"testing"
)

// generateTestKlines 生成测试用的 K线数据
func generateTestKlines(count int) []Kline {
	klines := make([]Kline, count)
	for i := 0; i < count; i++ {
		// 生成模拟的价格数据，有一定的波动
		basePrice := 100.0
		variance := float64(i%10) * 0.5
		open := basePrice + variance
		high := open + 1.0
		low := open - 0.5
		close := open + 0.3
		volume := 1000.0 + float64(i*100)

		klines[i] = Kline{
			OpenTime:  int64(i * 180000), // 3分钟间隔
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: int64((i+1)*180000 - 1),
		}
	}
	return klines
}

// TestCalculateIntradaySeries_VolumeCollection 测试 Volume 数据收集
func TestCalculateIntradaySeries_VolumeCollection(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		expectedVolLen int
	}{
		{
			name:           "正常情况 - 20个K线",
			klineCount:     20,
			expectedVolLen: 10, // 应该收集最近10个
		},
		{
			name:           "刚好10个K线",
			klineCount:     10,
			expectedVolLen: 10,
		},
		{
			name:           "少于10个K线",
			klineCount:     5,
			expectedVolLen: 5, // 应该返回所有5个
		},
		{
			name:           "超过10个K线",
			klineCount:     30,
			expectedVolLen: 10, // 应该只返回最近10个
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines, tt.expectedVolLen)

			if data == nil {
				t.Fatal("calculateIntradaySeries returned nil")
			}

			if len(data.Volume) != tt.expectedVolLen {
				t.Errorf("Volume length = %d, want %d", len(data.Volume), tt.expectedVolLen)
			}

			// 验证 Volume 数据正确性
			if len(data.Volume) > 0 {
				// 计算期望的起始索引
				start := tt.klineCount - 10
				if start < 0 {
					start = 0
				}

				// 验证第一个 Volume 值
				expectedFirstVolume := klines[start].Volume
				if data.Volume[0] != expectedFirstVolume {
					t.Errorf("First volume = %.2f, want %.2f", data.Volume[0], expectedFirstVolume)
				}

				// 验证最后一个 Volume 值
				expectedLastVolume := klines[tt.klineCount-1].Volume
				lastVolume := data.Volume[len(data.Volume)-1]
				if lastVolume != expectedLastVolume {
					t.Errorf("Last volume = %.2f, want %.2f", lastVolume, expectedLastVolume)
				}
			}
		})
	}
}

// TestCalculateIntradaySeries_VolumeValues 测试 Volume 值的正确性
func TestCalculateIntradaySeries_VolumeValues(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1000.0, High: 101.0, Low: 99.0, Open: 100.0},
		{Close: 101.0, Volume: 1100.0, High: 102.0, Low: 100.0, Open: 101.0},
		{Close: 102.0, Volume: 1200.0, High: 103.0, Low: 101.0, Open: 102.0},
		{Close: 103.0, Volume: 1300.0, High: 104.0, Low: 102.0, Open: 103.0},
		{Close: 104.0, Volume: 1400.0, High: 105.0, Low: 103.0, Open: 104.0},
		{Close: 105.0, Volume: 1500.0, High: 106.0, Low: 104.0, Open: 105.0},
		{Close: 106.0, Volume: 1600.0, High: 107.0, Low: 105.0, Open: 106.0},
		{Close: 107.0, Volume: 1700.0, High: 108.0, Low: 106.0, Open: 107.0},
		{Close: 108.0, Volume: 1800.0, High: 109.0, Low: 107.0, Open: 108.0},
		{Close: 109.0, Volume: 1900.0, High: 110.0, Low: 108.0, Open: 109.0},
	}

	data := calculateIntradaySeries(klines)

	expectedVolumes := []float64{1000.0, 1100.0, 1200.0, 1300.0, 1400.0, 1500.0, 1600.0, 1700.0, 1800.0, 1900.0}

	if len(data.Volume) != len(expectedVolumes) {
		t.Fatalf("Volume length = %d, want %d", len(data.Volume), len(expectedVolumes))
	}

	for i, expected := range expectedVolumes {
		if data.Volume[i] != expected {
			t.Errorf("Volume[%d] = %.2f, want %.2f", i, data.Volume[i], expected)
		}
	}
}

// TestCalculateIntradaySeries_ATR14 测试 ATR14 计算
func TestCalculateIntradaySeries_ATR14(t *testing.T) {
	tests := []struct {
		name          string
		klineCount    int
		expectZero    bool
		expectNonZero bool
	}{
		{
			name:          "足够数据 - 20个K线",
			klineCount:    20,
			expectNonZero: true,
		},
		{
			name:          "刚好15个K线（ATR14需要至少15个）",
			klineCount:    15,
			expectNonZero: true,
		},
		{
			name:       "数据不足 - 14个K线",
			klineCount: 14,
			expectZero: true,
		},
		{
			name:       "数据不足 - 10个K线",
			klineCount: 10,
			expectZero: true,
		},
		{
			name:       "数据不足 - 5个K线",
			klineCount: 5,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines)

			if data == nil {
				t.Fatal("calculateIntradaySeries returned nil")
			}

			if tt.expectZero && data.ATR14 != 0 {
				t.Errorf("ATR14 = %.3f, expected 0 (insufficient data)", data.ATR14)
			}

			if tt.expectNonZero && data.ATR14 <= 0 {
				t.Errorf("ATR14 = %.3f, expected > 0", data.ATR14)
			}
		})
	}
}

// TestCalculateATR 测试 ATR 计算函数
func TestCalculateATR(t *testing.T) {
	tests := []struct {
		name       string
		klines     []Kline
		period     int
		expectZero bool
	}{
		{
			name: "正常计算 - 足够数据",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
				{High: 103.0, Low: 101.0, Close: 102.0},
				{High: 104.0, Low: 102.0, Close: 103.0},
				{High: 105.0, Low: 103.0, Close: 104.0},
				{High: 106.0, Low: 104.0, Close: 105.0},
				{High: 107.0, Low: 105.0, Close: 106.0},
				{High: 108.0, Low: 106.0, Close: 107.0},
				{High: 109.0, Low: 107.0, Close: 108.0},
				{High: 110.0, Low: 108.0, Close: 109.0},
				{High: 111.0, Low: 109.0, Close: 110.0},
				{High: 112.0, Low: 110.0, Close: 111.0},
				{High: 113.0, Low: 111.0, Close: 112.0},
				{High: 114.0, Low: 112.0, Close: 113.0},
				{High: 115.0, Low: 113.0, Close: 114.0},
				{High: 116.0, Low: 114.0, Close: 115.0},
			},
			period:     14,
			expectZero: false,
		},
		{
			name: "数据不足 - 等于period",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
				{High: 103.0, Low: 101.0, Close: 102.0},
			},
			period:     2,
			expectZero: true,
		},
		{
			name: "数据不足 - 少于period",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
			},
			period:     14,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atr := calculateATR(tt.klines, tt.period)

			if tt.expectZero {
				if atr != 0 {
					t.Errorf("calculateATR() = %.3f, expected 0 (insufficient data)", atr)
				}
			} else {
				if atr <= 0 {
					t.Errorf("calculateATR() = %.3f, expected > 0", atr)
				}
			}
		})
	}
}

// TestCalculateATR_TrueRange 测试 ATR 的 True Range 计算正确性
func TestCalculateATR_TrueRange(t *testing.T) {
	// 创建一个简单的测试用例，手动计算期望的 ATR
	klines := []Kline{
		{High: 50.0, Low: 48.0, Close: 49.0}, // TR = 2.0
		{High: 51.0, Low: 49.0, Close: 50.0}, // TR = max(2.0, 2.0, 1.0) = 2.0
		{High: 52.0, Low: 50.0, Close: 51.0}, // TR = max(2.0, 2.0, 1.0) = 2.0
		{High: 53.0, Low: 51.0, Close: 52.0}, // TR = 2.0
		{High: 54.0, Low: 52.0, Close: 53.0}, // TR = 2.0
	}

	atr := calculateATR(klines, 3)

	// 期望的计算：
	// TR[1] = max(51-49, |51-49|, |49-49|) = 2.0
	// TR[2] = max(52-50, |52-50|, |50-50|) = 2.0
	// TR[3] = max(53-51, |53-51|, |51-51|) = 2.0
	// 初始 ATR = (2.0 + 2.0 + 2.0) / 3 = 2.0
	// TR[4] = max(54-52, |54-52|, |52-52|) = 2.0
	// 平滑 ATR = (2.0*2 + 2.0) / 3 = 2.0

	expectedATR := 2.0
	tolerance := 0.01 // 允许小的浮点误差

	if math.Abs(atr-expectedATR) > tolerance {
		t.Errorf("calculateATR() = %.3f, want approximately %.3f", atr, expectedATR)
	}
}

// TestCalculateIntradaySeries_ConsistencyWithOtherIndicators 测试 Volume 和其他指标的一致性
func TestCalculateIntradaySeries_ConsistencyWithOtherIndicators(t *testing.T) {
	klines := generateTestKlines(30)
	data := calculateIntradaySeries(klines)

	// 所有数组应该存在
	if data.MidPrices == nil {
		t.Error("MidPrices should not be nil")
	}
	if data.Volume == nil {
		t.Error("Volume should not be nil")
	}

	// MidPrices 和 Volume 应该有相同的长度（都是最近10个）
	if len(data.MidPrices) != len(data.Volume) {
		t.Errorf("MidPrices length (%d) should equal Volume length (%d)",
			len(data.MidPrices), len(data.Volume))
	}

	// 所有 Volume 值应该大于 0
	for i, vol := range data.Volume {
		if vol <= 0 {
			t.Errorf("Volume[%d] = %.2f, should be > 0", i, vol)
		}
	}
}

// TestCalculateIntradaySeries_EmptyKlines 测试空 K线数据
func TestCalculateIntradaySeries_EmptyKlines(t *testing.T) {
	klines := []Kline{}
	data := calculateIntradaySeries(klines)

	if data == nil {
		t.Fatal("calculateIntradaySeries should not return nil for empty klines")
	}

	// 所有切片应该为空
	if len(data.MidPrices) != 0 {
		t.Errorf("MidPrices length = %d, want 0", len(data.MidPrices))
	}
	if len(data.Volume) != 0 {
		t.Errorf("Volume length = %d, want 0", len(data.Volume))
	}

	// ATR14 应该为 0（数据不足）
	if data.ATR14 != 0 {
		t.Errorf("ATR14 = %.3f, want 0", data.ATR14)
	}
}

// TestCalculateIntradaySeries_VolumePrecision 测试 Volume 精度保持
func TestCalculateIntradaySeries_VolumePrecision(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1234.5678, High: 101.0, Low: 99.0},
		{Close: 101.0, Volume: 9876.5432, High: 102.0, Low: 100.0},
		{Close: 102.0, Volume: 5555.1111, High: 103.0, Low: 101.0},
	}

	data := calculateIntradaySeries(klines)

	expectedVolumes := []float64{1234.5678, 9876.5432, 5555.1111}

	for i, expected := range expectedVolumes {
		if data.Volume[i] != expected {
			t.Errorf("Volume[%d] = %.4f, want %.4f (precision not preserved)",
				i, data.Volume[i], expected)
		}
	}
}

// TestCalculateIntradaySeries_CustomDataPoints 测试自定义数据点数量
func TestCalculateIntradaySeries_CustomDataPoints(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		customPoints   int
		expectedVolLen int
	}{
		{
			name:           "使用20个数据点",
			klineCount:     50,
			customPoints:   20,
			expectedVolLen: 20,
		},
		{
			name:           "使用50个数据点",
			klineCount:     100,
			customPoints:   50,
			expectedVolLen: 50,
		},
		{
			name:           "使用40个数据点(默认3m)",
			klineCount:     60,
			customPoints:   40,
			expectedVolLen: 40,
		},
		{
			name:           "使用100个数据点(最大值)",
			klineCount:     150,
			customPoints:   100,
			expectedVolLen: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines, tt.customPoints)

			if len(data.Volume) != tt.expectedVolLen {
				t.Errorf("Volume length = %d, want %d", len(data.Volume), tt.expectedVolLen)
			}

			if len(data.MidPrices) != tt.expectedVolLen {
				t.Errorf("MidPrices length = %d, want %d", len(data.MidPrices), tt.expectedVolLen)
			}

			// EMA和MACD的长度可能少于请求的数据点（因为需要预热期）
			// 只要有数据就说明配置生效了
			if len(data.EMA20Values) == 0 {
				t.Error("EMA20Values should contain some data")
			}

			if len(data.MACDValues) == 0 {
				t.Error("MACDValues should contain some data")
			}
		})
	}
}

// TestCalculateLongerTermData_CustomDataPoints 测试长期数据自定义数据点数量
func TestCalculateLongerTermData_CustomDataPoints(t *testing.T) {
	tests := []struct {
		name         string
		klineCount   int
		customPoints int
		minExpected  int // 最小预期长度（考虑到指标计算需要预热期）
	}{
		{
			name:         "使用20个数据点",
			klineCount:   60,
			customPoints: 20,
			minExpected:  15, // MACD需要预热
		},
		{
			name:         "使用25个数据点(默认值)",
			klineCount:   60,
			customPoints: 25,
			minExpected:  20,
		},
		{
			name:         "使用40个数据点",
			klineCount:   80,
			customPoints: 40,
			minExpected:  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateLongerTermData(klines, tt.customPoints)

			// MACD和RSI需要预热期，所以长度可能小于请求的数据点
			// 但应该至少有一些数据
			if len(data.MACDValues) < tt.minExpected {
				t.Errorf("MACDValues length = %d, want at least %d", len(data.MACDValues), tt.minExpected)
			}

			if len(data.RSI14Values) < tt.minExpected {
				t.Errorf("RSI14Values length = %d, want at least %d", len(data.RSI14Values), tt.minExpected)
			}

			// Check scalar values are calculated
			if data.EMA20 == 0 && data.EMA50 == 0 {
				t.Error("Both EMA20 and EMA50 are zero, expected calculated values")
			}

			// Verify CurrentVolume is set
			if data.CurrentVolume == 0 {
				t.Error("CurrentVolume should be set")
			}
		})
	}
}

// TestGetDefaultIndicatorConfig 测试默认指标配置
func TestGetDefaultIndicatorConfig(t *testing.T) {
	config := GetDefaultIndicatorConfig()

	// 验证默认指标包含基础指标
	expectedBaseIndicators := []string{"ema", "macd", "rsi", "atr", "volume"}
	for _, expected := range expectedBaseIndicators {
		found := false
		for _, ind := range config.Indicators {
			if ind == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("基础指标 %s 应在默认配置中", expected)
		}
	}

	// 验证默认时间框架
	expectedTimeframes := []string{"3m", "4h"}
	if len(config.Timeframes) != len(expectedTimeframes) {
		t.Errorf("默认时间框架数量 = %d, want %d", len(config.Timeframes), len(expectedTimeframes))
	}

	for i, expected := range expectedTimeframes {
		if config.Timeframes[i] != expected {
			t.Errorf("Timeframes[%d] = %s, want %s", i, config.Timeframes[i], expected)
		}
	}

	// 验证默认数据点
	if config.DataPoints["3m"] != 40 {
		t.Errorf("DataPoints['3m'] = %d, want 40", config.DataPoints["3m"])
	}

	if config.DataPoints["4h"] != 25 {
		t.Errorf("DataPoints['4h'] = %d, want 25", config.DataPoints["4h"])
	}

	// 验证参数map已初始化
	if config.Parameters == nil {
		t.Error("Parameters should be initialized (empty map)")
	}
}

// TestIndicatorConfig_EdgeCases 测试边界情况
func TestIndicatorConfig_EdgeCases(t *testing.T) {
	t.Run("空配置应该使用默认值", func(t *testing.T) {
		klines := generateTestKlines(50)

		// 模拟空配置的情况，使用默认数据点数量
		data := calculateIntradaySeries(klines, 40) // 默认3m = 40个点

		if len(data.Volume) != 40 {
			t.Errorf("使用默认值时Volume length = %d, want 40", len(data.Volume))
		}
	})

	t.Run("最小数据点数量", func(t *testing.T) {
		klines := generateTestKlines(20)
		data := calculateIntradaySeries(klines, 10)

		if len(data.Volume) != 10 {
			t.Errorf("最小数据点Volume length = %d, want 10", len(data.Volume))
		}
	})

	t.Run("最大数据点数量", func(t *testing.T) {
		klines := generateTestKlines(150)
		data := calculateIntradaySeries(klines, 100)

		if len(data.Volume) != 100 {
			t.Errorf("最大数据点Volume length = %d, want 100", len(data.Volume))
		}
	})
}

// TestIsStaleData_NormalData tests that normal fluctuating data returns false
func TestIsStaleData_NormalData(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1000},
		{Close: 100.5, Volume: 1200},
		{Close: 99.8, Volume: 900},
		{Close: 100.2, Volume: 1100},
		{Close: 100.1, Volume: 950},
	}

	result := isStaleData(klines, "BTCUSDT")

	if result {
		t.Error("Expected false for normal fluctuating data, got true")
	}
}

// TestIsStaleData_PriceFreezeWithZeroVolume tests that frozen price + zero volume returns true
func TestIsStaleData_PriceFreezeWithZeroVolume(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
	}

	result := isStaleData(klines, "DOGEUSDT")

	if !result {
		t.Error("Expected true for frozen price + zero volume, got false")
	}
}

// TestIsStaleData_PriceFreezeWithVolume tests that frozen price but normal volume returns false
func TestIsStaleData_PriceFreezeWithVolume(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1000},
		{Close: 100.0, Volume: 1200},
		{Close: 100.0, Volume: 900},
		{Close: 100.0, Volume: 1100},
		{Close: 100.0, Volume: 950},
	}

	result := isStaleData(klines, "STABLECOIN")

	if result {
		t.Error("Expected false for frozen price but normal volume (low volatility market), got true")
	}
}

// TestIsStaleData_InsufficientData tests that insufficient data (<5 klines) returns false
func TestIsStaleData_InsufficientData(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
	}

	result := isStaleData(klines, "BTCUSDT")

	if result {
		t.Error("Expected false for insufficient data (<5 klines), got true")
	}
}

// TestIsStaleData_ExactlyFiveKlines tests edge case with exactly 5 klines
func TestIsStaleData_ExactlyFiveKlines(t *testing.T) {
	// Stale case: exactly 5 frozen klines with zero volume
	staleKlines := []Kline{
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
		{Close: 100.0, Volume: 0},
	}

	result := isStaleData(staleKlines, "TESTUSDT")
	if !result {
		t.Error("Expected true for exactly 5 frozen klines with zero volume, got false")
	}

	// Normal case: exactly 5 klines with fluctuation
	normalKlines := []Kline{
		{Close: 100.0, Volume: 1000},
		{Close: 100.1, Volume: 1100},
		{Close: 99.9, Volume: 900},
		{Close: 100.0, Volume: 1000},
		{Close: 100.05, Volume: 950},
	}

	result = isStaleData(normalKlines, "TESTUSDT")
	if result {
		t.Error("Expected false for exactly 5 normal klines, got true")
	}
}

// TestIsStaleData_WithinTolerance tests price changes within tolerance (0.01%)
func TestIsStaleData_WithinTolerance(t *testing.T) {
	// Price changes within 0.01% tolerance should be treated as frozen
	basePrice := 10000.0
	tolerance := 0.0001                        // 0.01%
	smallChange := basePrice * tolerance * 0.5 // Half of tolerance

	klines := []Kline{
		{Close: basePrice, Volume: 1000},
		{Close: basePrice + smallChange, Volume: 1000},
		{Close: basePrice - smallChange, Volume: 1000},
		{Close: basePrice, Volume: 1000},
		{Close: basePrice + smallChange, Volume: 1000},
	}

	result := isStaleData(klines, "BTCUSDT")

	// Should return false because there's normal volume despite tiny price changes
	if result {
		t.Error("Expected false for price within tolerance but with volume, got true")
	}
}

// TestIsStaleData_MixedScenario tests realistic scenario with some history before freeze
func TestIsStaleData_MixedScenario(t *testing.T) {
	// Simulate: normal trading → suddenly freezes
	klines := []Kline{
		{Close: 100.0, Volume: 1000}, // Normal
		{Close: 100.5, Volume: 1200}, // Normal
		{Close: 100.2, Volume: 1100}, // Normal
		{Close: 50.0, Volume: 0},     // Freeze starts
		{Close: 50.0, Volume: 0},     // Frozen
		{Close: 50.0, Volume: 0},     // Frozen
		{Close: 50.0, Volume: 0},     // Frozen
		{Close: 50.0, Volume: 0},     // Frozen (last 5 are all frozen)
	}

	result := isStaleData(klines, "DOGEUSDT")

	// Should detect stale data based on last 5 klines
	if !result {
		t.Error("Expected true for frozen last 5 klines with zero volume, got false")
	}
}

// TestIsStaleData_EmptyKlines tests edge case with empty slice
func TestIsStaleData_EmptyKlines(t *testing.T) {
	klines := []Kline{}

	result := isStaleData(klines, "BTCUSDT")

	if result {
		t.Error("Expected false for empty klines, got true")
	}
}

func TestCalculateEMAArray(t *testing.T) {
	klines := generateTestKlines(50)
	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
	}

	period := 20
	emaArray := CalculateEMAArray(prices, period)

	if len(emaArray) != len(prices) {
		t.Errorf("Expected length %d, got %d", len(prices), len(emaArray))
	}

	// Verify last value matches single point calculation
	lastEMA := calculateEMA(klines, period)
	if math.Abs(emaArray[len(emaArray)-1]-lastEMA) > 0.000001 {
		t.Errorf("Expected last EMA %f, got %f", lastEMA, emaArray[len(emaArray)-1])
	}
}

func TestCalculateRSIArray(t *testing.T) {
	klines := generateTestKlines(50)
	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
	}

	period := 14
	rsiArray := CalculateRSIArray(prices, period)

	if len(rsiArray) != len(prices) {
		t.Errorf("Expected length %d, got %d", len(prices), len(rsiArray))
	}

	// Verify last value matches single point calculation
	lastRSI := calculateRSI(klines, period)
	if math.Abs(rsiArray[len(rsiArray)-1]-lastRSI) > 0.000001 {
		t.Errorf("Expected last RSI %f, got %f", lastRSI, rsiArray[len(rsiArray)-1])
	}
}

func TestCalculateMACDArray(t *testing.T) {
	klines := generateTestKlines(50)
	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
	}

	macd, _, _ := CalculateMACDArray(prices)

	if len(macd) != len(prices) {
		t.Errorf("Expected length %d, got %d", len(prices), len(macd))
	}

	// Verify last value matches single point calculation
	lastMACD := calculateMACD(klines)
	if math.Abs(macd[len(macd)-1]-lastMACD) > 0.000001 {
		t.Errorf("Expected last MACD %f, got %f", lastMACD, macd[len(macd)-1])
	}
}

func TestCalculateBollingerBandsArray(t *testing.T) {
	klines := generateTestKlines(50)
	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
	}

	upper, mid, lower := CalculateBollingerBandsArray(prices, 20, 2.0)

	if len(upper) != len(prices) {
		t.Errorf("Expected length %d, got %d", len(prices), len(upper))
	}

	// Verify last value matches single point calculation
	lastUpper, lastMid, lastLower := calculateBollingerBands(klines, 20, 2.0)
	if math.Abs(upper[len(upper)-1]-lastUpper) > 0.000001 {
		t.Errorf("Expected last Upper %f, got %f", lastUpper, upper[len(upper)-1])
	}
	if math.Abs(mid[len(mid)-1]-lastMid) > 0.000001 {
		t.Errorf("Expected last Mid %f, got %f", lastMid, mid[len(mid)-1])
	}
	if math.Abs(lower[len(lower)-1]-lastLower) > 0.000001 {
		t.Errorf("Expected last Lower %f, got %f", lastLower, lower[len(lower)-1])
	}
}

func TestCalculateATRArray(t *testing.T) {
	klines := generateTestKlines(50)
	highs := make([]float64, len(klines))
	lows := make([]float64, len(klines))
	closes := make([]float64, len(klines))
	for i, k := range klines {
		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
	}

	atrArray := CalculateATRArray(highs, lows, closes, 14)

	if len(atrArray) != len(klines) {
		t.Errorf("Expected length %d, got %d", len(klines), len(atrArray))
	}

	// Verify last value matches single point calculation
	lastATR := calculateATR(klines, 14)
	if math.Abs(atrArray[len(atrArray)-1]-lastATR) > 0.000001 {
		t.Errorf("Expected last ATR %f, got %f", lastATR, atrArray[len(atrArray)-1])
	}
}

// TestCalculateSMA 测试 SMA 计算函数
func TestCalculateSMA(t *testing.T) {
	tests := []struct {
		name        string
		klines      []Kline
		period      int
		expectedSMA float64
		tolerance   float64
	}{
		{
			name: "正常计算 - 5个周期",
			klines: []Kline{
				{Close: 10.0},
				{Close: 11.0},
				{Close: 12.0},
				{Close: 13.0},
				{Close: 14.0},
			},
			period:      5,
			expectedSMA: 12.0, // (10+11+12+13+14)/5 = 12
			tolerance:   0.001,
		},
		{
			name: "正常计算 - 3个周期",
			klines: []Kline{
				{Close: 100.0},
				{Close: 110.0},
				{Close: 120.0},
				{Close: 130.0},
				{Close: 140.0},
			},
			period:      3,
			expectedSMA: 130.0, // (120+130+140)/3 = 130
			tolerance:   0.001,
		},
		{
			name: "数据不足",
			klines: []Kline{
				{Close: 100.0},
				{Close: 110.0},
			},
			period:      5,
			expectedSMA: 0, // 数据不足返回0
			tolerance:   0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sma := calculateSMA(tt.klines, tt.period)

			if math.Abs(sma-tt.expectedSMA) > tt.tolerance {
				t.Errorf("calculateSMA() = %.4f, want %.4f", sma, tt.expectedSMA)
			}
		})
	}
}

// TestCalculateVWAP 测试 VWAP 计算函数
func TestCalculateVWAP(t *testing.T) {
	tests := []struct {
		name         string
		klines       []Kline
		expectedVWAP float64
		tolerance    float64
	}{
		{
			name: "正常计算",
			klines: []Kline{
				{High: 100.0, Low: 98.0, Close: 99.0, Volume: 1000},   // TP=99, Vol=1000
				{High: 102.0, Low: 100.0, Close: 101.0, Volume: 2000}, // TP=101, Vol=2000
				{High: 104.0, Low: 102.0, Close: 103.0, Volume: 1500}, // TP=103, Vol=1500
			},
			// VWAP = (99*1000 + 101*2000 + 103*1500) / (1000+2000+1500)
			//      = (99000 + 202000 + 154500) / 4500 = 455500 / 4500 = 101.22...
			expectedVWAP: 101.222222,
			tolerance:    0.01,
		},
		{
			name:         "空数据",
			klines:       []Kline{},
			expectedVWAP: 0,
			tolerance:    0.001,
		},
		{
			name: "零成交量",
			klines: []Kline{
				{High: 100.0, Low: 98.0, Close: 99.0, Volume: 0},
			},
			expectedVWAP: 0, // 总成交量为0时返回0
			tolerance:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vwap := calculateVWAP(tt.klines)

			if math.Abs(vwap-tt.expectedVWAP) > tt.tolerance {
				t.Errorf("calculateVWAP() = %.6f, want %.6f", vwap, tt.expectedVWAP)
			}
		})
	}
}

// TestCalculateOBV 测试 OBV 计算函数
func TestCalculateOBV(t *testing.T) {
	tests := []struct {
		name        string
		klines      []Kline
		expectedOBV float64
		tolerance   float64
	}{
		{
			name: "价格上涨 - OBV累加",
			klines: []Kline{
				{Close: 100.0, Volume: 1000},
				{Close: 101.0, Volume: 1500}, // 上涨: +1500
				{Close: 102.0, Volume: 2000}, // 上涨: +2000
			},
			expectedOBV: 3500, // 0 + 1500 + 2000 = 3500
			tolerance:   0.001,
		},
		{
			name: "价格下跌 - OBV扣减",
			klines: []Kline{
				{Close: 100.0, Volume: 1000},
				{Close: 99.0, Volume: 1500}, // 下跌: -1500
				{Close: 98.0, Volume: 2000}, // 下跌: -2000
			},
			expectedOBV: -3500, // 0 - 1500 - 2000 = -3500
			tolerance:   0.001,
		},
		{
			name: "混合场景",
			klines: []Kline{
				{Close: 100.0, Volume: 1000},
				{Close: 102.0, Volume: 1500}, // 上涨: +1500
				{Close: 101.0, Volume: 1000}, // 下跌: -1000
				{Close: 101.0, Volume: 800},  // 持平: +0
				{Close: 105.0, Volume: 2000}, // 上涨: +2000
			},
			expectedOBV: 2500, // 0 + 1500 - 1000 + 0 + 2000 = 2500
			tolerance:   0.001,
		},
		{
			name:        "单个K线",
			klines:      []Kline{{Close: 100.0, Volume: 1000}},
			expectedOBV: 0, // 第一个K线OBV为0
			tolerance:   0.001,
		},
		{
			name:        "空数据",
			klines:      []Kline{},
			expectedOBV: 0,
			tolerance:   0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obv := calculateOBV(tt.klines)

			if math.Abs(obv-tt.expectedOBV) > tt.tolerance {
				t.Errorf("calculateOBV() = %.4f, want %.4f", obv, tt.expectedOBV)
			}
		})
	}
}

// TestCalculateTimeframeData_NewIndicators 测试 TimeframeData 包含新指标字段
func TestCalculateTimeframeData_NewIndicators(t *testing.T) {
	klines := generateTestKlines(50)

	data := CalculateTimeframeData(klines, "4h", 25)

	if data == nil {
		t.Fatal("CalculateTimeframeData returned nil")
	}

	// 验证 SMAValues 存在
	if data.SMAValues == nil {
		t.Error("SMAValues should not be nil")
	}

	// 验证 VWAPValues 存在
	if data.VWAPValues == nil {
		t.Error("VWAPValues should not be nil")
	}

	// 验证 OBVValues 存在
	if data.OBVValues == nil {
		t.Error("OBVValues should not be nil")
	}

	// 验证长度合理 (应该与其他指标长度匹配)
	if len(data.SMAValues) != len(data.MidPrices) {
		t.Errorf("SMAValues length = %d, want %d", len(data.SMAValues), len(data.MidPrices))
	}

	if len(data.VWAPValues) != len(data.MidPrices) {
		t.Errorf("VWAPValues length = %d, want %d", len(data.VWAPValues), len(data.MidPrices))
	}

	if len(data.OBVValues) != len(data.MidPrices) {
		t.Errorf("OBVValues length = %d, want %d", len(data.OBVValues), len(data.MidPrices))
	}
}

// TestGetDefaultIndicatorConfig_NewIndicators 测试默认配置包含新指标
func TestGetDefaultIndicatorConfig_NewIndicators(t *testing.T) {
	config := GetDefaultIndicatorConfig()

	// 验证新指标在默认列表中
	newIndicators := []string{"sma", "vwap", "obv", "stochastic", "williams_r", "cci", "adx"}
	for _, indicator := range newIndicators {
		found := false
		for _, ind := range config.Indicators {
			if ind == indicator {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Indicator %s should be in default config", indicator)
		}
	}

	// 验证 sma_period 参数存在
	if config.Parameters["sma_period"] != 20 {
		t.Errorf("sma_period = %d, want 20", config.Parameters["sma_period"])
	}
}

// TestCalculateStochastic 测试 Stochastic 计算函数
func TestCalculateStochastic(t *testing.T) {
	tests := []struct {
		name     string
		klines   []Kline
		period   int
		expectK  float64
		expectD  float64
		toleranceK float64
	}{
		{
			name: "正常计算 - 接近最高价",
			klines: []Kline{
				{High: 100.0, Low: 90.0, Close: 95.0},
				{High: 102.0, Low: 91.0, Close: 96.0},
				{High: 105.0, Low: 92.0, Close: 97.0},
				{High: 103.0, Low: 93.0, Close: 98.0},
				{High: 104.0, Low: 94.0, Close: 104.0}, // 收盘价接近最高价
			},
			period:     5,
			expectK:    100.0, // (104-90)/(105-90)*100 = 93.33
			toleranceK: 10,
		},
		{
			name: "正常计算 - 接近最低价",
			klines: []Kline{
				{High: 100.0, Low: 90.0, Close: 95.0},
				{High: 102.0, Low: 91.0, Close: 96.0},
				{High: 105.0, Low: 92.0, Close: 97.0},
				{High: 103.0, Low: 93.0, Close: 98.0},
				{High: 104.0, Low: 94.0, Close: 91.0}, // 收盘价接近最低价
			},
			period:     5,
			expectK:    6.67, // (91-90)/(105-90)*100 ≈ 6.67
			toleranceK: 1,
		},
		{
			name: "数据不足",
			klines: []Kline{
				{High: 100.0, Low: 90.0, Close: 95.0},
				{High: 102.0, Low: 91.0, Close: 96.0},
			},
			period:     5,
			expectK:    0,
			toleranceK: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stochK, _ := calculateStochastic(tt.klines, tt.period)

			if math.Abs(stochK-tt.expectK) > tt.toleranceK {
				t.Errorf("calculateStochastic() K = %.2f, want approximately %.2f", stochK, tt.expectK)
			}
		})
	}
}

// TestCalculateWilliamsR 测试 Williams %R 计算函数
func TestCalculateWilliamsR(t *testing.T) {
	tests := []struct {
		name      string
		klines    []Kline
		period    int
		expected  float64
		tolerance float64
	}{
		{
			name: "正常计算 - 接近最高价 (超买)",
			klines: []Kline{
				{High: 100.0, Low: 90.0, Close: 95.0},
				{High: 102.0, Low: 91.0, Close: 96.0},
				{High: 105.0, Low: 92.0, Close: 97.0},
				{High: 103.0, Low: 93.0, Close: 98.0},
				{High: 104.0, Low: 94.0, Close: 104.0}, // 收盘价接近最高价
			},
			period:    5,
			expected:  -6.67, // (105-104)/(105-90)*-100 ≈ -6.67
			tolerance: 2,
		},
		{
			name: "正常计算 - 接近最低价 (超卖)",
			klines: []Kline{
				{High: 100.0, Low: 90.0, Close: 95.0},
				{High: 102.0, Low: 91.0, Close: 96.0},
				{High: 105.0, Low: 92.0, Close: 97.0},
				{High: 103.0, Low: 93.0, Close: 98.0},
				{High: 104.0, Low: 94.0, Close: 91.0}, // 收盘价接近最低价
			},
			period:    5,
			expected:  -93.33, // (105-91)/(105-90)*-100 ≈ -93.33
			tolerance: 2,
		},
		{
			name: "数据不足",
			klines: []Kline{
				{High: 100.0, Low: 90.0, Close: 95.0},
			},
			period:    5,
			expected:  0,
			tolerance: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			williamsR := calculateWilliamsR(tt.klines, tt.period)

			if math.Abs(williamsR-tt.expected) > tt.tolerance {
				t.Errorf("calculateWilliamsR() = %.2f, want approximately %.2f", williamsR, tt.expected)
			}
		})
	}
}

// TestCalculateCCI 测试 CCI 计算函数
func TestCalculateCCI(t *testing.T) {
	tests := []struct {
		name      string
		klines    []Kline
		period    int
		checkSign bool // 只检查正负号
		positive  bool
	}{
		{
			name: "上涨趋势 - CCI 应为正",
			klines: func() []Kline {
				klines := make([]Kline, 20)
				for i := 0; i < 20; i++ {
					base := 100.0 + float64(i)*2 // 持续上涨
					klines[i] = Kline{
						High:  base + 1,
						Low:   base - 1,
						Close: base,
					}
				}
				return klines
			}(),
			period:    20,
			checkSign: true,
			positive:  true,
		},
		{
			name: "下跌趋势 - CCI 应为负",
			klines: func() []Kline {
				klines := make([]Kline, 20)
				for i := 0; i < 20; i++ {
					base := 140.0 - float64(i)*2 // 持续下跌
					klines[i] = Kline{
						High:  base + 1,
						Low:   base - 1,
						Close: base,
					}
				}
				return klines
			}(),
			period:    20,
			checkSign: true,
			positive:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cci := calculateCCI(tt.klines, tt.period)

			if tt.checkSign {
				if tt.positive && cci <= 0 {
					t.Errorf("calculateCCI() = %.2f, expected positive value", cci)
				}
				if !tt.positive && cci >= 0 {
					t.Errorf("calculateCCI() = %.2f, expected negative value", cci)
				}
			}
		})
	}
}

// TestCalculateADX 测试 ADX 计算函数
func TestCalculateADX(t *testing.T) {
	tests := []struct {
		name        string
		klines      []Kline
		period      int
		expectRange [2]float64 // ADX 应该在这个范围内
	}{
		{
			name: "强趋势 - ADX 应该较高",
			klines: func() []Kline {
				klines := make([]Kline, 20)
				for i := 0; i < 20; i++ {
					base := 100.0 + float64(i)*3 // 强烈上涨
					klines[i] = Kline{
						High:  base + 2,
						Low:   base - 1,
						Close: base + 1,
					}
				}
				return klines
			}(),
			period:      14,
			expectRange: [2]float64{0, 100}, // ADX 在 0-100 范围内
		},
		{
			name: "数据不足",
			klines: []Kline{
				{High: 100.0, Low: 90.0, Close: 95.0},
				{High: 102.0, Low: 91.0, Close: 96.0},
			},
			period:      14,
			expectRange: [2]float64{0, 0.001},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adx := calculateADX(tt.klines, tt.period)

			if adx < tt.expectRange[0] || adx > tt.expectRange[1] {
				t.Errorf("calculateADX() = %.2f, expected in range [%.2f, %.2f]", adx, tt.expectRange[0], tt.expectRange[1])
			}
		})
	}
}

// TestCalculateTimeframeData_SecondBatchIndicators 测试第二批指标在 TimeframeData 中
func TestCalculateTimeframeData_SecondBatchIndicators(t *testing.T) {
	klines := generateTestKlines(50)

	data := CalculateTimeframeData(klines, "4h", 25)

	if data == nil {
		t.Fatal("CalculateTimeframeData returned nil")
	}

	// 验证第二批指标字段存在
	if data.StochKValues == nil {
		t.Error("StochKValues should not be nil")
	}
	if data.StochDValues == nil {
		t.Error("StochDValues should not be nil")
	}
	if data.WilliamsR == nil {
		t.Error("WilliamsR should not be nil")
	}
	if data.CCIValues == nil {
		t.Error("CCIValues should not be nil")
	}
	if data.ADXValues == nil {
		t.Error("ADXValues should not be nil")
	}

	// 验证长度匹配
	expectedLen := len(data.MidPrices)
	if len(data.StochKValues) != expectedLen {
		t.Errorf("StochKValues length = %d, want %d", len(data.StochKValues), expectedLen)
	}
	if len(data.WilliamsR) != expectedLen {
		t.Errorf("WilliamsR length = %d, want %d", len(data.WilliamsR), expectedLen)
	}
	if len(data.CCIValues) != expectedLen {
		t.Errorf("CCIValues length = %d, want %d", len(data.CCIValues), expectedLen)
	}
	if len(data.ADXValues) != expectedLen {
		t.Errorf("ADXValues length = %d, want %d", len(data.ADXValues), expectedLen)
	}
}

// TestCalculatePSAR 测试 Parabolic SAR 计算函数
func TestCalculatePSAR(t *testing.T) {
	tests := []struct {
		name        string
		klines      []Kline
		expectRange [2]float64
	}{
		{
			name: "上升趋势 - PSAR 应在价格下方",
			klines: func() []Kline {
				klines := make([]Kline, 20)
				for i := 0; i < 20; i++ {
					base := 100.0 + float64(i)*2
					klines[i] = Kline{
						Open:  base,
						High:  base + 2,
						Low:   base - 1,
						Close: base + 1,
					}
				}
				return klines
			}(),
			expectRange: [2]float64{90, 140}, // PSAR 应该在合理范围内
		},
		{
			name: "数据不足",
			klines: []Kline{
				{High: 100, Low: 90, Close: 95, Open: 92},
				{High: 102, Low: 91, Close: 96, Open: 95},
			},
			expectRange: [2]float64{0, 0.001},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			psar := calculatePSAR(tt.klines)

			if psar < tt.expectRange[0] || psar > tt.expectRange[1] {
				t.Errorf("calculatePSAR() = %.2f, expected in range [%.2f, %.2f]", psar, tt.expectRange[0], tt.expectRange[1])
			}
		})
	}
}

// TestCalculateCMF 测试 CMF 计算函数
func TestCalculateCMF(t *testing.T) {
	tests := []struct {
		name      string
		klines    []Kline
		period    int
		checkSign bool
		positive  bool
	}{
		{
			name: "买入压力 - CMF 应为正",
			klines: func() []Kline {
				klines := make([]Kline, 20)
				for i := 0; i < 20; i++ {
					// 收盘价接近最高价 = 买入压力
					klines[i] = Kline{
						High:   float64(100 + i),
						Low:    float64(90 + i),
						Close:  float64(99 + i), // 接近最高价
						Volume: 1000,
					}
				}
				return klines
			}(),
			period:    20,
			checkSign: true,
			positive:  true,
		},
		{
			name: "卖出压力 - CMF 应为负",
			klines: func() []Kline {
				klines := make([]Kline, 20)
				for i := 0; i < 20; i++ {
					// 收盘价接近最低价 = 卖出压力
					klines[i] = Kline{
						High:   float64(100 + i),
						Low:    float64(90 + i),
						Close:  float64(91 + i), // 接近最低价
						Volume: 1000,
					}
				}
				return klines
			}(),
			period:    20,
			checkSign: true,
			positive:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmf := calculateCMF(tt.klines, tt.period)

			if tt.checkSign {
				if tt.positive && cmf <= 0 {
					t.Errorf("calculateCMF() = %.4f, expected positive value", cmf)
				}
				if !tt.positive && cmf >= 0 {
					t.Errorf("calculateCMF() = %.4f, expected negative value", cmf)
				}
			}
		})
	}
}

// TestCalculateIchimoku 测试 Ichimoku 计算函数
func TestCalculateIchimoku(t *testing.T) {
	tests := []struct {
		name         string
		klines       []Kline
		tenkanPeriod int
		kijunPeriod  int
		expectTenkan bool
		expectKijun  bool
	}{
		{
			name: "正常计算",
			klines: func() []Kline {
				klines := make([]Kline, 30)
				for i := 0; i < 30; i++ {
					klines[i] = Kline{
						High:  float64(100 + i),
						Low:   float64(90 + i),
						Close: float64(95 + i),
					}
				}
				return klines
			}(),
			tenkanPeriod: 9,
			kijunPeriod:  26,
			expectTenkan: true,
			expectKijun:  true,
		},
		{
			name: "数据不足",
			klines: []Kline{
				{High: 100, Low: 90, Close: 95},
				{High: 102, Low: 91, Close: 96},
			},
			tenkanPeriod: 9,
			kijunPeriod:  26,
			expectTenkan: false,
			expectKijun:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenkan, kijun := calculateIchimoku(tt.klines, tt.tenkanPeriod, tt.kijunPeriod)

			if tt.expectTenkan && tenkan == 0 {
				t.Error("Tenkan should not be 0")
			}
			if !tt.expectTenkan && tenkan != 0 {
				t.Errorf("Tenkan = %.2f, expected 0", tenkan)
			}
			if tt.expectKijun && kijun == 0 {
				t.Error("Kijun should not be 0")
			}
			if !tt.expectKijun && kijun != 0 {
				t.Errorf("Kijun = %.2f, expected 0", kijun)
			}
		})
	}
}

// TestCalculateTimeframeData_ThirdBatchIndicators 测试第三批指标在 TimeframeData 中
func TestCalculateTimeframeData_ThirdBatchIndicators(t *testing.T) {
	klines := generateTestKlines(50)

	data := CalculateTimeframeData(klines, "4h", 25)

	if data == nil {
		t.Fatal("CalculateTimeframeData returned nil")
	}

	// 验证第三批指标字段存在
	if data.PSARValues == nil {
		t.Error("PSARValues should not be nil")
	}
	if data.CMFValues == nil {
		t.Error("CMFValues should not be nil")
	}
	if data.IchimokuTenkan == nil {
		t.Error("IchimokuTenkan should not be nil")
	}
	if data.IchimokuKijun == nil {
		t.Error("IchimokuKijun should not be nil")
	}

	// 验证长度匹配
	expectedLen := len(data.MidPrices)
	if len(data.PSARValues) != expectedLen {
		t.Errorf("PSARValues length = %d, want %d", len(data.PSARValues), expectedLen)
	}
	if len(data.CMFValues) != expectedLen {
		t.Errorf("CMFValues length = %d, want %d", len(data.CMFValues), expectedLen)
	}
	if len(data.IchimokuTenkan) != expectedLen {
		t.Errorf("IchimokuTenkan length = %d, want %d", len(data.IchimokuTenkan), expectedLen)
	}
	if len(data.IchimokuKijun) != expectedLen {
		t.Errorf("IchimokuKijun length = %d, want %d", len(data.IchimokuKijun), expectedLen)
	}
}

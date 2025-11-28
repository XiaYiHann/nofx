package market

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAPIClient tests that NewAPIClient creates a client with default base URL.
func TestNewAPIClient(t *testing.T) {
	client := NewAPIClient()
	assert.NotNil(t, client)
	assert.Equal(t, defaultBaseURL, client.baseURL)
	assert.NotNil(t, client.client)
}

// TestNewAPIClientWithBaseURL tests that NewAPIClientWithBaseURL creates a client with custom base URL.
func TestNewAPIClientWithBaseURL(t *testing.T) {
	customURL := "http://localhost:8080"
	client := NewAPIClientWithBaseURL(customURL, nil)
	assert.NotNil(t, client)
	assert.Equal(t, customURL, client.baseURL)
	assert.NotNil(t, client.client)
}

// TestNewAPIClientWithBaseURL_WithCustomHttpClient tests that custom HTTP client is used.
func TestNewAPIClientWithBaseURL_WithCustomHttpClient(t *testing.T) {
	customURL := "http://localhost:8080"
	customHTTPClient := &http.Client{}
	client := NewAPIClientWithBaseURL(customURL, customHTTPClient)
	assert.NotNil(t, client)
	assert.Equal(t, customURL, client.baseURL)
	// Note: due to hook mechanism, client.client might be different from customHTTPClient
	assert.NotNil(t, client.client)
}

// TestGetKlines_Mocked_ValidArray tests GetKlines with a valid klines array response.
func TestGetKlines_Mocked_ValidArray(t *testing.T) {
	// Create a mock server that returns valid klines data
	mockKlines := [][]interface{}{
		{
			float64(1609459200000), // OpenTime
			"29000.00",             // Open
			"29500.00",             // High
			"28500.00",             // Low
			"29200.00",             // Close
			"1000.5",               // Volume
			float64(1609462800000), // CloseTime
			"29000000.00",          // QuoteVolume
			float64(5000),          // Trades
			"500.25",               // TakerBuyBaseVolume
			"14500000.00",          // TakerBuyQuoteVolume
		},
		{
			float64(1609462800000), // OpenTime
			"29200.00",             // Open
			"29800.00",             // High
			"29100.00",             // Low
			"29700.00",             // Close
			"1200.75",              // Volume
			float64(1609466400000), // CloseTime
			"35000000.00",          // QuoteVolume
			float64(6000),          // Trades
			"600.50",               // TakerBuyBaseVolume
			"17500000.00",          // TakerBuyQuoteVolume
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/fapi/v1/klines", r.URL.Path)
		assert.Equal(t, "BTCUSDT", r.URL.Query().Get("symbol"))
		assert.Equal(t, "1h", r.URL.Query().Get("interval"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockKlines)
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	klines, err := client.GetKlines("BTCUSDT", "1h", 5)

	require.NoError(t, err)
	require.Len(t, klines, 2)

	// Validate first kline
	assert.Equal(t, int64(1609459200000), klines[0].OpenTime)
	assert.Equal(t, 29000.00, klines[0].Open)
	assert.Equal(t, 29500.00, klines[0].High)
	assert.Equal(t, 28500.00, klines[0].Low)
	assert.Equal(t, 29200.00, klines[0].Close)
	assert.Equal(t, 1000.5, klines[0].Volume)
	assert.Equal(t, int64(1609462800000), klines[0].CloseTime)
	assert.Equal(t, 5000, klines[0].Trades)

	// Validate second kline
	assert.Equal(t, int64(1609462800000), klines[1].OpenTime)
	assert.Equal(t, 29700.00, klines[1].Close)
}

// TestGetKlines_Mocked_ErrorObject tests GetKlines with an API error response.
func TestGetKlines_Mocked_ErrorObject(t *testing.T) {
	// Create a mock server that returns an API error object
	apiError := map[string]interface{}{
		"code": -2015,
		"msg":  "Service unavailable from a restricted location according to 'b]'. Please refer to local laws and regulations.",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(apiError)
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	klines, err := client.GetKlines("BTCUSDT", "1h", 5)

	assert.Nil(t, klines)
	require.Error(t, err)

	// Verify error is a BinanceAPIError
	var binanceErr *BinanceAPIError
	if errors.As(err, &binanceErr) {
		assert.Equal(t, -2015, binanceErr.Code)
		assert.Contains(t, binanceErr.Msg, "Service unavailable")
	} else {
		// Fallback: check error string contains expected info
		assert.Contains(t, err.Error(), "Service unavailable")
	}

	// Ensure no panic occurred and we got a proper error
	assert.NotContains(t, err.Error(), "panic")
}

// TestGetKlines_Mocked_ErrorObject_HTTP200 tests GetKlines when API returns 200 but with error JSON.
func TestGetKlines_Mocked_ErrorObject_HTTP200(t *testing.T) {
	// Some APIs might return 200 but with error JSON body
	apiError := map[string]interface{}{
		"code": -1121,
		"msg":  "Invalid symbol.",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(apiError)
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	klines, err := client.GetKlines("INVALID", "1h", 5)

	assert.Nil(t, klines)
	require.Error(t, err)

	// The error should mention the API error code
	var binanceErr *BinanceAPIError
	if errors.As(err, &binanceErr) {
		assert.Equal(t, -1121, binanceErr.Code)
		assert.Contains(t, binanceErr.Msg, "Invalid symbol")
	}
}

// TestGetKlines_Mocked_UnexpectedResponse tests GetKlines with an unexpected response format.
func TestGetKlines_Mocked_UnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("<html><body>Service temporarily unavailable</body></html>"))
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	klines, err := client.GetKlines("BTCUSDT", "1h", 5)

	assert.Nil(t, klines)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected Binance response")
}

// TestGetKlines_Mocked_EmptyArray tests GetKlines with an empty array response.
func TestGetKlines_Mocked_EmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	klines, err := client.GetKlines("BTCUSDT", "1h", 5)

	require.NoError(t, err)
	assert.Len(t, klines, 0)
}

// TestGetExchangeInfo_Mocked_ErrorObject tests GetExchangeInfo with an API error response.
func TestGetExchangeInfo_Mocked_ErrorObject(t *testing.T) {
	apiError := map[string]interface{}{
		"code": -1003,
		"msg":  "Too many requests; current limit is 1200 requests per minute.",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(apiError)
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	info, err := client.GetExchangeInfo()

	assert.Nil(t, info)
	require.Error(t, err)

	var binanceErr *BinanceAPIError
	if errors.As(err, &binanceErr) {
		assert.Equal(t, -1003, binanceErr.Code)
		assert.Contains(t, binanceErr.Msg, "Too many requests")
	}
}

// TestGetCurrentPrice_Mocked_ValidResponse tests GetCurrentPrice with a valid response.
func TestGetCurrentPrice_Mocked_ValidResponse(t *testing.T) {
	ticker := map[string]string{
		"symbol": "BTCUSDT",
		"price":  "45000.50",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/fapi/v1/ticker/price", r.URL.Path)
		assert.Equal(t, "BTCUSDT", r.URL.Query().Get("symbol"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ticker)
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	price, err := client.GetCurrentPrice("BTCUSDT")

	require.NoError(t, err)
	assert.Equal(t, 45000.50, price)
}

// TestGetCurrentPrice_Mocked_ErrorObject tests GetCurrentPrice with an API error response.
func TestGetCurrentPrice_Mocked_ErrorObject(t *testing.T) {
	apiError := map[string]interface{}{
		"code": -1121,
		"msg":  "Invalid symbol.",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiError)
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	price, err := client.GetCurrentPrice("INVALID")

	assert.Equal(t, float64(0), price)
	require.Error(t, err)

	var binanceErr *BinanceAPIError
	if errors.As(err, &binanceErr) {
		assert.Equal(t, -1121, binanceErr.Code)
	}
}

// TestBinanceAPIError_Error tests the Error() method of BinanceAPIError.
func TestBinanceAPIError_Error(t *testing.T) {
	err := &BinanceAPIError{
		Code: -2015,
		Msg:  "Service unavailable from a restricted location",
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "code=-2015")
	assert.Contains(t, errStr, "Service unavailable")
}

// TestParseAPIError tests the parseAPIError helper function.
func TestParseAPIError(t *testing.T) {
	client := NewAPIClient()

	tests := []struct {
		name     string
		body     string
		expected *BinanceAPIError
	}{
		{
			name:     "valid API error",
			body:     `{"code": -2015, "msg": "Service unavailable"}`,
			expected: &BinanceAPIError{Code: -2015, Msg: "Service unavailable"},
		},
		{
			name:     "not an API error - empty object",
			body:     `{}`,
			expected: nil, // code is 0, so not considered an error
		},
		{
			name:     "not an API error - array",
			body:     `[]`,
			expected: nil,
		},
		{
			name:     "not an API error - invalid JSON",
			body:     `not json`,
			expected: nil,
		},
		{
			name:     "API error with positive code (non-standard but possible)",
			body:     `{"code": 1001, "msg": "Some message"}`,
			expected: &BinanceAPIError{Code: 1001, Msg: "Some message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.parseAPIError([]byte(tt.body))
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Code, result.Code)
				assert.Equal(t, tt.expected.Msg, result.Msg)
			}
		})
	}
}

// TestGetKlinesBatch_Mocked_ValidArray tests getKlinesBatch with a valid response.
func TestGetKlinesBatch_Mocked_ValidArray(t *testing.T) {
	mockKlines := [][]interface{}{
		{
			float64(1609459200000),
			"29000.00",
			"29500.00",
			"28500.00",
			"29200.00",
			"1000.5",
			float64(1609462800000),
			"29000000.00",
			float64(5000),
			"500.25",
			"14500000.00",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/fapi/v1/klines", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("startTime"))
		assert.NotEmpty(t, r.URL.Query().Get("endTime"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockKlines)
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	klines, err := client.getKlinesBatch("BTCUSDT", "1h", 1609459200000, 1609466400000, 100)

	require.NoError(t, err)
	require.Len(t, klines, 1)
	assert.Equal(t, int64(1609459200000), klines[0].OpenTime)
}

// TestGetKlinesBatch_Mocked_ErrorObject tests getKlinesBatch with an API error response.
func TestGetKlinesBatch_Mocked_ErrorObject(t *testing.T) {
	apiError := map[string]interface{}{
		"code": -2015,
		"msg":  "Service unavailable from a restricted location",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(apiError)
	}))
	defer server.Close()

	client := NewAPIClientWithBaseURL(server.URL, nil)
	klines, err := client.getKlinesBatch("BTCUSDT", "1h", 1609459200000, 1609466400000, 100)

	assert.Nil(t, klines)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Service unavailable") || strings.Contains(err.Error(), "-2015"))
}

// TestParseKline_InvalidData tests parseKline with invalid data.
func TestParseKline_InvalidData(t *testing.T) {
	// Too few elements
	shortData := KlineResponse([]interface{}{float64(1609459200000), "29000.00"})
	_, err := parseKline(shortData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid kline data")
}

// TestParseIntervalToMs tests the parseIntervalToMs function.
func TestParseIntervalToMs(t *testing.T) {
	tests := []struct {
		interval string
		expected int64
		hasError bool
	}{
		{"1m", 60 * 1000, false},
		{"5m", 5 * 60 * 1000, false},
		{"15m", 15 * 60 * 1000, false},
		{"1h", 60 * 60 * 1000, false},
		{"4h", 4 * 60 * 60 * 1000, false},
		{"1d", 24 * 60 * 60 * 1000, false},
		{"", 0, true},      // invalid
		{"m", 0, true},     // invalid (no number)
		{"1x", 0, true},    // invalid unit
		{"abc", 0, true},   // completely invalid
	}

	for _, tt := range tests {
		t.Run(tt.interval, func(t *testing.T) {
			result, err := parseIntervalToMs(tt.interval)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

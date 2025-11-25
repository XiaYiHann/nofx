package trader

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBinanceSpotAPI performs a complete integration test of the Binance Spot API
// using the Testnet. It covers:
// 1. Connectivity (Ping, Time)
// 2. Account Information
// 3. Order Placement (Test & Real)
// 4. Order Query
// 5. Order Cancellation
func TestBinanceSpotAPI(t *testing.T) {
	// 1. Load Environment Variables
	// Try to load from .env file in project root
	_ = godotenv.Load("../.env")

	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		t.Skip("Skipping Binance Spot API test: BINANCE_API_KEY or BINANCE_SECRET_KEY not set")
	}

	// 2. Initialize Client
	// Use Testnet Base URL
	binance.UseTestnet = true
	client := binance.NewClient(apiKey, secretKey)

	ctx := context.Background()

	// 3. Connectivity Tests
	t.Run("Connectivity", func(t *testing.T) {
		// Ping
		err := client.NewPingService().Do(ctx)
		assert.NoError(t, err, "Ping failed")

		// Server Time
		serverTime, err := client.NewServerTimeService().Do(ctx)
		assert.NoError(t, err, "Get Server Time failed")
		t.Logf("Server Time: %d", serverTime)
	})

	// 4. Account Information
	t.Run("Account Info", func(t *testing.T) {
		res, err := client.NewGetAccountService().Do(ctx)
		require.NoError(t, err, "Get Account failed")

		t.Logf("Can Trade: %v", res.CanTrade)
		t.Logf("Account Type: %s", res.AccountType)

		// Print non-zero balances
		for _, bal := range res.Balances {
			free, _ := fmt.Sscanf(bal.Free, "%f", new(float64))
			locked, _ := fmt.Sscanf(bal.Locked, "%f", new(float64))
			if free > 0 || locked > 0 {
				t.Logf("Balance %s: Free=%s, Locked=%s", bal.Asset, bal.Free, bal.Locked)
			}
		}
	})

	// 5. Order Operations
	t.Run("Order Operations", func(t *testing.T) {
		symbol := "BTCUSDT"

		// 5.1 Test Order (Validation only, no execution)
		err := client.NewCreateOrderService().
			Symbol(symbol).
			Side(binance.SideTypeBuy).
			Type(binance.OrderTypeLimit).
			TimeInForce(binance.TimeInForceTypeGTC).
			Quantity("0.01").
			Price("50000").
			Test(ctx)
		assert.NoError(t, err, "Test Order failed")
		t.Log("Test Order (Validation) passed")

		// 5.2 Place Real Order (Limit Buy at low price to avoid execution)
		// Note: On Testnet, prices might be different. Using a safe low price.
		price := "40000"
		quantity := "0.001"

		order, err := client.NewCreateOrderService().
			Symbol(symbol).
			Side(binance.SideTypeBuy).
			Type(binance.OrderTypeLimit).
			TimeInForce(binance.TimeInForceTypeGTC).
			Quantity(quantity).
			Price(price).
			Do(ctx)

		require.NoError(t, err, "Place Order failed")
		t.Logf("Order Placed: ID=%d, Symbol=%s, Price=%s, Qty=%s, Status=%s",
			order.OrderID, order.Symbol, order.Price, order.OrigQuantity, order.Status)

		// 5.3 Query Order
		// Wait a bit to ensure propagation
		time.Sleep(1 * time.Second)

		queryOrder, err := client.NewGetOrderService().
			Symbol(symbol).
			OrderID(order.OrderID).
			Do(ctx)

		assert.NoError(t, err, "Query Order failed")
		if err == nil {
			assert.Equal(t, order.OrderID, queryOrder.OrderID)
			// Parse prices to float for comparison to handle "40000" vs "40000.00000000"
			expectedPrice, _ := strconv.ParseFloat(price, 64)
			actualPrice, _ := strconv.ParseFloat(queryOrder.Price, 64)
			assert.Equal(t, expectedPrice, actualPrice, "Price mismatch")
			t.Logf("Order Queried: Status=%s", queryOrder.Status)
		}

		// 5.4 Cancel Order
		cancelRes, err := client.NewCancelOrderService().
			Symbol(symbol).
			OrderID(order.OrderID).
			Do(ctx)

		assert.NoError(t, err, "Cancel Order failed")
		if err == nil {
			t.Logf("Order Cancelled: ID=%d, Status=%s", cancelRes.OrderID, cancelRes.Status)
		}

		// 5.5 Verify Cancellation
		time.Sleep(1 * time.Second)
		finalOrder, err := client.NewGetOrderService().
			Symbol(symbol).
			OrderID(order.OrderID).
			Do(ctx)

		assert.NoError(t, err, "Query Final Order failed")
		if err == nil {
			assert.Equal(t, binance.OrderStatusTypeCanceled, finalOrder.Status)
			t.Log("Order Cancellation Verified")
		}
	})
}

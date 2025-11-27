# Testing Strategy

This project uses a comprehensive test suite covering unit, integration, and E2E tests.

## Running Tests

To run all tests:
```bash
go test ./...
```

## Test Structure

- **Unit Tests**: Located alongside code (e.g., `*_test.go`). Cover logic and boundaries.
- **Integration Tests**:
  - `api/integration_test.go`: Tests API endpoints using a real router and in-memory DB.
  - `manager/trader_manager_test.go`: Tests trader lifecycle management with DB persistence.
- **E2E Tests**:
  - `backtest/full_system_test.go`: Runs a complete backtest loop with mocked Market Data and LLM.

## Live Integration Tests

Tests that interact with real external APIs (e.g. Binance) are skipped by default to ensure CI stability and security.
To run them, set the `LIVE_TESTS` environment variable:

```bash
LIVE_TESTS=1 go test ./trader/... -v
```

## Helpers & Mocks

The `testhelpers` package provides reusable test infrastructure:
- `SetupTestDB`: Creates a clean, temporary SQLite DB for each test with auto-cleanup.
- `SetupMockLLMServer`: Mocks LLM API responses for deterministic testing.
- `MockTrader`: Mocks exchange interactions.
- `PerformRequest`: Helper for testing Gin API routes.

We use `gomonkey` for patching functions (e.g., `market.NewAPIClient`) where dependency injection is difficult.

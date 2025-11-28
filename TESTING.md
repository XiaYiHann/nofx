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

## Real LLM Integration Tests

Integration tests involving LLM calls (e.g., `decision/engine_integration_test.go`) use a **Mock LLM Server** by default. This ensures tests are fast, deterministic, and do not consume API quotas.

To run these tests against a **Real LLM API**:

1. Set `RUN_REAL_LLM_TESTS=true`
2. Provide `LLM_API_KEY` (and optionally `LLM_PROVIDER`, `LLM_MODEL`, `LLM_API_URL`)

```bash
# Example: Run real LLM tests with DeepSeek
export LLM_API_KEY="your-api-key"
export RUN_REAL_LLM_TESTS=true
go test ./decision/... -v -run TestRealLLM
```

**Note:** CI does NOT run real LLM tests by default. You must manually trigger the `integration-tests` workflow or use the `rebuild.sh --with-integration` script.

## Helpers & Mocks

The `testhelpers` package provides reusable test infrastructure:

- `SetupTestDB`: Creates a clean, temporary SQLite DB for each test with auto-cleanup.
- `SetupMockLLMServer`: Mocks LLM API responses for deterministic testing.
- `MockTrader`: Mocks exchange interactions.
- `PerformRequest`: Helper for testing Gin API routes.

We use `gomonkey` for patching functions (e.g., `market.NewAPIClient`) where dependency injection is difficult.

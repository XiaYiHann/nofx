# Backtest System Status Audit Report

## Summary
The backtest system is **Mature and Functional** for single-user, single-strategy scenarios. The core engine correctly implements time simulation, data loading (with caching), order execution (with slippage), and position management. Frontend integration is complete with creation, monitoring, and detailed result visualization.

Key components are well-tested via unit tests (Backend: `backtest` package; Frontend: `web` Vitest suite). However, the system relies heavily on simplified heuristics for "Mock Mode" (always long), which limits the ability to verify *strategy* performance without spending real LLM tokens.

## Evidence
- **Backend Compilation**: `go build ./backtest/...` passed.
- **Frontend Tests**: `npm test` passed (16 files, 356 tests).
- **Key Files Verified**:
  - `backtest/engine.go`: Implements full event loop, data preheating (12h), and graceful handling of missing data.
  - `api/backtest_handlers.go`: correctly maps API requests to Engine config.
  - `web/src/pages/BacktestPage.tsx`: Implements polling and status visualization.
- **Test Coverage**:
  - `backtest/full_system_test.go` provides a strong integration test baseline using `mockllm` and `gomonkey`.
  - Frontend tests cover API error handling and UI state management (auth, navigation).

## Gaps & Risks

### 1. Mock Mode Fidelity (Risk: High)
- **Location**: `backtest/engine.go` (`getMockDecisions`)
- **Evidence**: Code uses a hardcoded heuristic (Open Long if no pos; Close if >8% profit or <-4% loss).
- **Impact**: Developers cannot test "Short" logic or complex reasoning flows without spending real money/tokens on LLM calls.
- **Risk**: Bugs in "Short" position management or complex order types might remain undetected in CI.

### 2. Slippage & Fee Modeling (Gap: Medium)
- **Location**: `backtest/order_simulator.go`
- **Evidence**: Uses fixed `slippageBps`.
- **Impact**: Does not account for market volatility or liquidity depth.
- **Risk**: Backtest results might be overly optimistic for large position sizes.

### 3. Concurrency & Scalability (Risk: Medium)
- **Location**: `config/database.go` (SQLite WAL mode)
- **Evidence**: While WAL is enabled, heavy concurrent backtests might still contend for database locks or saturate disk I/O.
- **Risk**: System performance degradation during simultaneous multi-user backtests.

## Actionable Next Steps

### Priority 1: Enhance Mock Decision Engine (Scriptable Scenarios)
- **Goal**: Allow testing specific market scenarios (e.g., "Short Squeeze", "Chop") without real LLM.
- **Files**: `backtest/engine.go`, `backtest/types.go`
- **Instruction**: Add a `MockScenario` config to `BacktestRun`. If set to `random_walk` or `trend_following`, `getMockDecisions` should behave accordingly. Better yet, allow loading a JSON "Decision Script" to force specific decisions at specific timestamps.
- **Acceptance**: A test case that forces a "Short" entry and successfully verifies the PnL calculation for a short position.

### Priority 2: Deterministic End-to-End Regression Test
- **Goal**: Ensure no regressions in the full "Create -> Run -> Result" flow.
- **Files**: `backtest/full_integration_test.go` (New File)
- **Instruction**: Create a test that spins up the *actual* Gin server (on a random port), points it to a temp DB, creates a backtest via HTTP POST, waits for completion, and asserts on the returned JSON.
- **Acceptance**: `go test -run TestE2EBacktestFlow` passes and verifies the API contract.

### Priority 3: Performance Benchmarking
- **Goal**: Verify system stability under load.
- **Instruction**: Create a script to trigger 5 concurrent backtests (mock mode) and measure CPU/Memory usage and DB lock contention.
- **Acceptance**: System completes 5 concurrent runs within X% time of a single run * 5.

## Reproduction Commands

### Verify Backend Build
```bash
go build ./backtest/...
```

### Verify Frontend
```bash
cd web && npm ci && npm test
```

### Run Backend Integration Tests (Mock LLM)
```bash
go test -v -run TestFullBacktestSystem ./backtest
```

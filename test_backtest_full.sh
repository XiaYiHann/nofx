#!/bin/bash

# Check if .env exists
if [ ! -f .env ]; then
    echo "⚠️  .env file not found!"
    echo "Please create .env with LLM_API_KEY set."
    exit 1
fi

# Run the full backtest integration test
echo "🚀 Running Full Backtest Integration Test..."
echo "This test uses REAL data from Binance and REAL LLM (GLM-4-Flash)."
echo "It may take a minute or two."
echo "---------------------------------------------------"

go test -v -timeout 300s -run TestFullBacktestIntegration ./backtest

if [ $? -eq 0 ]; then
    echo "---------------------------------------------------"
    echo "✅ Test Passed Successfully!"
else
    echo "---------------------------------------------------"
    echo "❌ Test Failed!"
fi

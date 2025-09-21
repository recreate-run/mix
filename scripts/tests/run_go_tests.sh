#!/bin/bash
# Script to run Go tests for the Mix application
set -e

echo "🔍 Running Go tests..."

# Navigate to the Go backend directory
cd "$(dirname "$0")/../../mix_agent"

# Set environment variables for tests
export TEST_MODE=true

# Run tests with verbose output and generate coverage
go test -v ./internal/... -coverprofile=coverage.out

# Display coverage summary
echo "📊 Test coverage summary:"
go tool cover -func=coverage.out

echo "✅ Go tests completed"
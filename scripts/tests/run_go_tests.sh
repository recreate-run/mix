#!/bin/bash
# Script to run Go tests for the Mix application
set -e

# Parse command line arguments
UNIT_ONLY=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --unit-only)
            UNIT_ONLY=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --unit-only    Run only stable unit tests (excludes integration tests)"
            echo "  --verbose      Enable verbose output"
            echo "  -h, --help     Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo "🔍 Running Go tests..."

# Navigate to the Go backend directory
cd "$(dirname "$0")/../../mix_agent"

# Set environment variables for tests
export TEST_MODE=true

# Determine test command based on options
if [ "$UNIT_ONLY" = true ]; then
    echo "📝 Running unit tests only (excluding integration tests)"
    # Run stable unit tests only - these modules have solid test coverage
    TEST_PACKAGES="./internal/credentials ./internal/session ./internal/message ./internal/preferences ./internal/llm/agent ./internal/config ./internal/llm/tools"
    echo "🎯 Testing packages: credentials, session, message, preferences, agent, config, tools"
else
    echo "🧪 Running all tests including integration tests"
    TEST_PACKAGES="./internal/..."
fi

# Set verbosity
VERBOSITY=""
if [ "$VERBOSE" = true ] || [ "$UNIT_ONLY" = true ]; then
    VERBOSITY="-v"
fi

# Run tests with coverage
echo "⚡ Executing tests..."
if go test $VERBOSITY $TEST_PACKAGES -coverprofile=coverage.out; then
    echo "✅ All tests passed!"

    # Display coverage summary
    echo ""
    echo "📊 Test coverage summary:"
    go tool cover -func=coverage.out

    if [ "$UNIT_ONLY" = true ]; then
        echo ""
        echo "📈 Core module coverage:"
        go tool cover -func=coverage.out | grep -E "(credentials|session|message|preferences|agent|config|tools)" | head -25

        # Calculate and display total coverage for tested modules
        TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        echo ""
        echo "🎯 Total Coverage: $TOTAL_COVERAGE"
    fi

    echo ""
    echo "✅ Go tests completed successfully"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi
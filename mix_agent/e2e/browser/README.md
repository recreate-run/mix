# Browser Tool E2E Tests

This directory contains end-to-end tests for the Browser tool integration.

## What These Tests Do

These E2E tests verify the Browser tool works correctly in a real environment:

1. **TestBrowserE2EFullWorkflow**: Tests a complete user workflow
   - Creates a session
   - Sends a message requesting browser action
   - Waits for agent to process the message
   - Verifies browser tool was invoked
   - Checks for screenshot file
   - Cleans up the session

2. **TestBrowserE2ESessionIsolation**: Tests session isolation
   - Creates two separate sessions
   - Sends different browser requests to each
   - Verifies each session has isolated file storage
   - Cleans up both sessions

## Difference from Integration Tests

| Aspect | E2E Tests (here) | Integration Tests |
|--------|------------------|-------------------|
| **Location** | `/mix_agent/e2e/browser/` | `/mix_agent/internal/http/integration_tests/` |
| **Server** | Real server (task dev) | Test server (httptest.Server) |
| **Database** | Real database | Isolated test database |
| **Services** | Real local-browser-service | Mocked browser client |
| **Purpose** | Test real user workflows | Test API contracts |
| **Speed** | Slower | Faster |
| **When to run** | Before deployment | During development |

## Prerequisites

Before running these tests, you must have:

1. **Mix agent server running:**
   ```bash
   task dev
   ```

2. **Local browser service running:**
   ```bash
   cd browser-service
   go run ./cmd/server
   ```

3. **Valid API key configured:**
   - In `.env` file: `ANTHROPIC_API_KEY=sk-ant-...`
   - Or via: `mix auth add anthropic`

4. **Server accessible at:**
   - Default: `http://localhost:3020`
   - Override: `export E2E_SERVER_URL=http://localhost:3000`

## Running the Tests

```bash
# Run all browser E2E tests
go test ./mix_agent/e2e/browser/... -v

# Run specific test
go test ./mix_agent/e2e/browser/... -v -run TestBrowserE2EFullWorkflow

# Skip E2E tests (if server not running)
export SKIP_E2E_TESTS=1
go test ./mix_agent/e2e/browser/... -v
```

## Test Behavior

The tests will **automatically skip** if:
- `SKIP_E2E_TESTS` environment variable is set
- Server is not running at the expected URL
- Local browser service is not running at `localhost:8081`
- Health check endpoint returns non-200 status

Example output when skipping:
```
=== RUN   TestBrowserE2EFullWorkflow
    browser_e2e_test.go:73: Skipping E2E test: server not running at http://localhost:3020: dial tcp: connection refused
--- SKIP: TestBrowserE2EFullWorkflow (0.00s)
```

## Configuration

Environment variables:
- `E2E_SERVER_URL` - Server URL (default: `http://localhost:3020`)
- `SKIP_E2E_TESTS` - Set to skip all E2E tests
- Request timeout is hardcoded to 60 seconds

## Test Flow

Each test follows this pattern:

1. **Pre-flight checks:**
   - Check if E2E tests are disabled
   - Check if server is running
   - Check if browser-service is running

2. **Execute test scenario:**
   - Make real HTTP requests to actual server
   - Wait for agent to process messages
   - Verify expected outcomes

3. **Cleanup:**
   - Delete test sessions
   - Clean up any created resources

## Debugging

If tests fail:

1. **Check server is running:**
   ```bash
   curl http://localhost:3020/health
   ```

2. **Check local-browser-service is running:**
   ```bash
   curl http://localhost:8081/health
   ```

3. **Check logs:**
   ```bash
   task tail-log
   ```

4. **Run with verbose output:**
   ```bash
   go test ./mix_agent/e2e/browser/... -v -run TestBrowserE2EFullWorkflow
   ```

## When to Run These Tests

Run E2E tests:
- Before creating a pull request
- Before deploying to production
- After significant browser tool changes
- When verifying bug fixes

Don't run E2E tests:
- During rapid development (use integration tests instead)
- In CI without proper setup (set SKIP_E2E_TESTS=1)
- When testing isolated components

## Adding New Tests

When adding new browser E2E tests:

1. Follow the naming convention: `TestBrowserE2E<Scenario>`
2. Start with skip checks:
   ```go
   func TestBrowserE2ENewScenario(t *testing.T) {
       skipIfE2EDisabled(t)
       skipIfServerNotRunning(t)
       skipIfLocalBrowserServiceNotRunning(t)
       // ... test implementation
   }
   ```
3. Always clean up resources (delete sessions, etc.)
4. Use the helper functions: `makeRequest`, `parseJSONResponse`, `waitForProcessing`
5. Log test progress with `t.Log()` for debugging

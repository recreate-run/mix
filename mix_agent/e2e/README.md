# End-to-End (E2E) Tests

This directory contains **true end-to-end tests** that test the application as a whole.

## E2E Tests vs Integration Tests

### E2E Tests (this directory: `/mix_agent/e2e/`)
- ✅ Test the **actual running application**
- ✅ Use **real server** (not httptest.Server)
- ✅ Use **real database** (the application's database)
- ✅ Test **real user workflows** from start to finish
- ✅ May test through **frontend UI** or REST API
- ✅ Run **separately** from unit/integration tests
- ❌ **Slower** and require full application stack

**Example:** Start server with `task dev`, then run tests against `http://localhost:3020`

### Integration Tests (`/mix_agent/internal/http/integration_tests/`)
- ✅ Test **component interactions** within the app
- ✅ Use **test HTTP server** (httptest.Server)
- ✅ Use **isolated test database** (fresh for each test)
- ✅ Test **API contracts** and component integration
- ✅ Run as part of **regular test suite**
- ✅ **Faster** and don't require running server

**Example:** `go test ./mix_agent/internal/http/integration_tests/...` creates test server automatically

## Running E2E Tests

### Prerequisites

1. **Start the application:**
   ```bash
   task dev  # Starts both frontend and backend
   ```

2. **Start browser-service:**
   ```bash
   cd browser-service
   go run ./cmd/server
   ```

3. **Ensure API key is configured:**
   - Either in `.env` file: `ANTHROPIC_API_KEY=sk-ant-...`
   - Or via: `mix auth add anthropic`

### Run E2E Tests

```bash
# Run all E2E tests
go test ./mix_agent/e2e/... -v

# Run specific E2E test
go test ./mix_agent/e2e/browser/... -v -run TestBrowserE2EFullWorkflow

# Skip E2E tests
export SKIP_E2E_TESTS=1
go test ./mix_agent/e2e/... -v
```

### Configuration

Environment variables:
- `E2E_SERVER_URL` - Server URL (default: `http://localhost:3020`)
- `SKIP_E2E_TESTS` - Set to skip all E2E tests
- `E2E_TIMEOUT` - Request timeout (default: 60s)

## Test Structure

```
e2e/
├── README.md                    # This file
├── browser/
│   └── browser_e2e_test.go     # Browser tool E2E tests
└── [future test suites]/
```

## Writing E2E Tests

E2E tests should:

1. **Start with skipIfE2EDisabled()** to respect SKIP_E2E_TESTS
2. **Check server is running** with skipIfServerNotRunning()
3. **Test complete user workflows** from start to finish
4. **Use real HTTP requests** to the actual server
5. **Clean up after themselves** (delete sessions, etc.)
6. **Be independent** (don't depend on other tests)

Example:
```go
func TestMyE2EWorkflow(t *testing.T) {
    skipIfE2EDisabled(t)
    skipIfServerNotRunning(t)

    // 1. Create session via API
    // 2. Perform user actions
    // 3. Verify results
    // 4. Cleanup
}
```

## Continuous Integration

In CI environments:
- Set `SKIP_E2E_TESTS=1` if server isn't running
- Or start server in CI and run E2E tests as separate step
- E2E tests are typically slower, consider running them separately

## When to Use E2E Tests

Use E2E tests for:
- ✅ Critical user workflows
- ✅ Full stack integration verification
- ✅ Smoke tests for deployments
- ✅ Testing external service integration

Don't use E2E tests for:
- ❌ Unit-level logic testing
- ❌ Fast feedback during development
- ❌ Testing all edge cases (use unit tests)
- ❌ Component isolation testing

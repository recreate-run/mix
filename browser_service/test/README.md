# Test Utilities

## Test Helpers

### `helpers.go`

Contains shared test utilities for integration and e2e tests:

- **`startTestServer(t, ctx)`** - Starts a WebSocket server on a random port for testing
  - Returns: server instance, WebSocket URL, cleanup function
  - Used by: `e2e_test.go`

**Note:** The `pkg/client/client_test.go` file contains a duplicate of `startTestServer` because Go's import rules prevent it from importing from the `test` package. This is acceptable since:
1. The function is well-tested and stable
2. Go convention discourages cross-package test utilities
3. The duplication is only ~40 lines

## Test Skip Pattern

Each integration test file contains:
```go
func skipIfIntegrationTestsDisabled(t *testing.T) {
    if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
        t.Skip("Skipping integration test")
    }
}
```

This pattern is intentionally duplicated across test files because:
1. It's only 4 lines per file
2. Consolidating it would create import cycles
3. Go convention favors simple duplication over complex imports for such trivial code

**Files with this pattern:**
- `internal/browser/browser_integration_test.go`
- `internal/server/websocket_integration_test.go`
- `pkg/client/client_test.go`
- `test/e2e_test.go`

## Running Tests

```bash
# All tests
task test

# Skip integration tests (faster, no browser)
SKIP_INTEGRATION_TESTS=1 go test ./...

# Specific test package
go test -v ./test
```

# Session Cleanup Race Condition

## Issue Discovery

During parallel E2E test execution, we encountered recurring errors:

```
level=ERROR msg="failed to process events: failed to get session: sql: no rows in result set"
```

**Context:** When running 3 browser E2E tests concurrently, tests would delete their sessions during cleanup while background event processors were still attempting to access them.

## Root Cause

The event processing system runs in background goroutines that poll for session events. When a test completes:

1. Test calls `DELETE /api/sessions/:id`
2. Session row is immediately removed from the database
3. Background event processor wakes up, tries to fetch session
4. Database returns `sql.ErrNoRows`
5. Error is logged and propagated (causing noise/confusion)

This is a **resource lifecycle mismatch** - background processors don't know when sessions are deleted.

## Solution 1: Context Cancellation + WaitGroup (Rejected)

### Approach
Add lifecycle management to sessions:

```go
type Session struct {
    // ... existing fields
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
}
```

**Implementation Steps:**
1. Store context/cancel in Session struct
2. All background goroutines track themselves via WaitGroup
3. On DELETE, call `cancel()` then `wg.Wait(timeout)`
4. Only delete from DB after goroutines finish

**Pros:**
- Explicit lifecycle control
- Guaranteed cleanup ordering
- Immediate shutdown signal

**Cons:**
- Significant complexity (3+ files to modify)
- Threading context through codebase
- WaitGroup tracking everywhere
- Over-engineering for the actual problem
- Doesn't match Go idioms (resources should handle their own cleanup)

## Solution 2: Graceful Error Handling (Chosen)

### Approach
Treat missing sessions as normal, not exceptional:

```go
// In internal/llm/callbacks/executor.go
session, err := s.sessionService.Get(ctx, event.SessionID)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        // Session deleted - exit silently
        return nil
    }
    return fmt.Errorf("failed to get session: %w", err)
}
```

**Implementation:**
- Single file change (`internal/llm/callbacks/executor.go`)
- Check for `sql.ErrNoRows` specifically
- Exit gracefully instead of logging error

**Pros:**
- Minimal code change (~3 lines)
- Defensive programming pattern
- Background processes naturally terminate
- No struct modifications
- Follows "fail gracefully" principle

**Cons:**
- Doesn't provide immediate shutdown (goroutines finish their current iteration)
- Small delay between deletion and processor exit

## Decision Rationale

**Why Solution 2 is better:**

1. **Simplicity** - The "race" is actually normal behavior. Sessions can be deleted while background work runs. The bug is that we crash instead of handling it.

2. **Go Idioms** - Resources should be resilient to dependencies disappearing. Goroutines polling for work should handle "work no longer exists."

3. **Scope** - We're fixing a parallel testing issue, not building infrastructure for graceful shutdown (which may be needed later, but not now).

4. **YAGNI Principle** - Context cancellation adds complexity we don't currently need. If we later require immediate shutdown or resource cleanup, we can add it then.

## Implementation Status

**Status:** Not yet implemented
**Estimated Effort:** 5 minutes
**Files to Change:** 1 (`internal/llm/callbacks/executor.go`)

## Testing Verification

After implementing, verify with:

```bash
cd mix_agent
go test -v -tags=e2e -run "TestBrowserE2E(ActionSequenceSuccess|FormFilling|ElementSelectionByIndex)$" \
  ./e2e/browser/... -timeout 5m -parallel 3
```

**Expected Result:** No more "sql: no rows in result set" errors in logs during parallel execution.

## Related Issues

This pattern should be applied anywhere we access sessions asynchronously:
- Event processors
- SSE streaming handlers
- Background notification systems

**Search for similar patterns:**
```bash
grep -r "sessionService.Get" internal/
```

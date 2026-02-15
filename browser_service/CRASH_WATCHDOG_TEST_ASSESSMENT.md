# Crash Watchdog Test Quality Assessment

## Executive Summary

Phase 6 Crash Watchdog implementation includes **real infrastructure** for browser health monitoring and event streaming, but the test suite has significant gaps. Only 1 out of 6 tests actually verifies the watchdog detects problems.

## Infrastructure Built (Verified Working)

### Event System ✅
- `internal/browser/events/broker.go` - Generic pub/sub event broker
- `internal/browser/events/events.go` - Event type definitions
- WebSocket event streaming from server to clients
- Client event subscription API (`SubscribeToEvents()`)

### Crash Watchdog ✅
- `internal/browser/watchdog/crash.go` - Monitoring implementation
- Network request tracking with 10s timeout
- Health check loop (5s interval, 10s initial delay)
- Target crash event listeners
- Multi-tab registration

## Test Quality Analysis

### Real Tests (Actually Verify Functionality)

**TestCrashWatchdogDetectsNetworkTimeout** ✅ **VERIFIED**
- Triggers actual 15-second HTTP request via JavaScript fetch
- Subscribes to WebSocket event stream
- Waits for and validates NetworkTimeout event
- Verifies event contains: `elapsed_seconds ≥ 10`, correct URL, method, resource type
- **Proof**: Test output shows actual event captured with 13.97 elapsed seconds
- **Status**: This is a proper integration test

**TestCrashWatchdogEventsStopAfterDisconnect** ✅ **DECENT**
- Verifies events stop flowing after client disconnect
- Tests event channel lifecycle management

### Passive Tests (Don't Verify Core Functionality)

**TestCrashWatchdogHealthCheckPasses** ⚠️ **PASSIVE**
- Waits 21 seconds and checks NO BrowserUnresponsive events occur
- **Problem**: Could pass even if health check loop isn't running at all
- **Missing**: No verification that health check loop executed
- **Should**: Add instrumentation or force an unresponsive state

**TestCrashWatchdogWithMultipleTabs** ⚠️ **SUPERFICIAL**
- Creates 3 tabs, waits 16 seconds, checks browser works
- **Problem**: Doesn't verify CDP listeners were registered for each tab
- **Missing**: No evidence watchdog is monitoring all tabs
- **Should**: Trigger events on different tabs and verify detection

**TestCrashWatchdogNetworkRequestTracking** ⚠️ **PASSIVE**
- Navigates pages, checks no timeout events
- **Problem**: Doesn't verify requests are added/removed from tracking map
- **Missing**: No inspection of internal watchdog state
- **Should**: Add accessor to verify tracking map is populated

**TestCrashWatchdogStartsAndStops** ⚠️ **SUPERFICIAL**
- Waits and verifies browser is responsive
- **Problem**: No evidence monitoring loop is running
- **Missing**: Can't distinguish from watchdog not starting at all
- **Should**: Use instrumentation or ticker verification

## Critical Gaps

### Unimplemented Tests
1. **Target Crash Detection** - No test navigates to `chrome://crash` or verifies TargetCrash events
2. **Browser Unresponsive** - No test that actually makes browser unresponsive and verifies detection
3. **Internal State Verification** - No tests inspect watchdog's network request map or listener registry

### Architectural Limitations
- Watchdog state is private with no test accessors
- Can't verify monitoring loop tick count or health check executions
- Can't inspect which targets have registered listeners

## Recommendations

### Immediate Improvements
1. Add test instrumentation package with counters for health checks executed
2. Implement test that triggers actual browser unresponsiveness
3. Add accessor methods to expose watchdog state for testing (test-only builds)
4. Create test that verifies network request map is populated during navigation

### Future Work
- Add metrics/telemetry for health check execution count
- Implement chrome://crash test with proper target recovery verification
- Add stress test with many concurrent slow requests
- Test watchdog behavior during browser process crashes (not just tab crashes)

## Conclusion

The crash watchdog **infrastructure is real and functional** (proven by NetworkTimeout test). However, test coverage is **incomplete**. Current test suite validates event streaming works but doesn't comprehensively verify all monitoring capabilities. The watchdog may work correctly, but tests don't prove it.

**Recommendation**: Add instrumentation before claiming Phase 6 is fully verified.

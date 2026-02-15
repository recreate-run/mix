# Fallback Removal Plan: Simplify Coordinate-Based Clicking

**Date**: 2026-02-15
**Status**: Ready for Implementation

## Executive Summary

mix_agent currently has redundant fallback logic for coordinate-based clicking that should be removed. Both service mode (local browser) and tunnel mode (remote browser) now support native coordinate clicking internally, making the fallback unnecessary.

## Current Redundant Architecture

### Fallback #1: Service Mode Double Conversion

```
mix_agent (browser.go)
  ↓ Checks: client implements coordinateClicker? YES
  ↓ Calls: client.ClickAt(450, 300)
  ↓
browser-service (handler.go)
  ↓ Receives: ClickAt(450, 300)
  ↓ Calls: Context.ClickAtCoordinate()
  ↓ Internally: ReadPage → find element → ClickByBackendID
```

**Problem**: No conversion needed in mix_agent, browser-service handles it.

### Fallback #2: Tunnel Mode Local Conversion

```
mix_agent (browser.go)
  ↓ Checks: client implements coordinateClicker? NO
  ↓ Fallback: backendIDFromCoordinate() [local conversion]
  ↓ Network: ReadPage() to get elements
  ↓ Searches: locally for element at coordinates
  ↓ Calls: client.ClickByBackendID(123)
  ↓
tunnel (tunnel_client_wrapper.go)
  ↓ Sends: DOM.click with backend ID
```

**Problem**: TunnelClientWrapper ALREADY has `ClickAt()` that uses CDP's `Input.dispatchMouseEvent`. We're converting coordinates to backend IDs unnecessarily.

**Tunnel already supports:**
```go
func (t *TunnelClientWrapper) ClickAt(ctx, x, y, button, clickCount, duration, tabID) {
    // Uses CDP: Input.dispatchMouseEvent
    // No conversion needed!
}
```

## Simplified Architecture (After Cleanup)

```
mix_agent (browser.go)
  ↓ Always calls: ClickAt(450, 300)
  ↓
  ├─ Service Mode → browser-service.ClickAt()
  │                  ↓ handler.go → Context.ClickAtCoordinate()
  │                  ↓ Internal: ReadPage + search + ClickByBackendID
  │
  └─ Tunnel Mode → tunnel.ClickAt()
                    ↓ Uses CDP: Input.dispatchMouseEvent
```

**Result**: Simple, direct calls. Each client handles coordinates internally.

## Code Changes Required

### 1. Remove coordinateClicker Interface

**File**: `mix_agent/internal/llm/tools/browser/browser.go`

**Delete** (around line 543):
```go
type coordinateClicker interface {
    ClickAt(ctx context.Context, x, y float64, button string, clickCount int, duration *int, tabID ...string) error
}
```

### 2. Remove clickByCoordinate Fallback Logic

**File**: `mix_agent/internal/llm/tools/browser/browser.go`

**Delete** `clickByCoordinate()` function (around lines 623-651):
```go
func (b *browserTool) clickByCoordinate(ctx, client, sessionID, tabID, coordinate, button, clickCount, duration, repeat) error {
    // REMOVE: This checks coordinateClicker interface
    // REMOVE: Falls back to backendIDFromCoordinate conversion
}
```

### 3. Simplify Click Handler Methods

**File**: `mix_agent/internal/llm/tools/browser/browser.go`

**Replace** coordinate-based click logic in:
- `handleLeftClick()` (around line 663)
- `handleRightClick()` (around line 733)
- `handleDoubleClick()` (around line 787)
- `handleTripleClick()` (around line 841)

**Before** (lines 684-688):
```go
if params.Coordinate != nil {
    if err := b.clickByCoordinate(ctx, client, sessionID, params.TabID, *params.Coordinate, mouseButtonLeft, 1, nil, 1); err != nil {
        return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
    }
    return interfaces.NewTextResponse("Successfully clicked coordinate")
}
```

**After**:
```go
if params.Coordinate != nil {
    if err := client.ClickAt(ctx, params.Coordinate.X, params.Coordinate.Y, mouseButtonLeft, 1, nil, params.TabID); err != nil {
        return interfaces.NewTextErrorResponse(fmt.Sprintf("Click failed: %v", err))
    }
    return interfaces.NewTextResponse("Successfully clicked coordinate")
}
```

### 4. Remove coordinateClicker Type Assertions

**File**: `mix_agent/internal/llm/tools/browser/browser.go`

**Delete** all checks like (appears ~8 times):
```go
if _, ok := client.(coordinateClicker); ok {
    // coordinate conversion logic
}
```

These appear in:
- Ref-based clicking (lines 668, 738, 792, 846)
- Index-based clicking error messages (lines 692, 762, 816, 870)

**Replace** with direct method calls to the appropriate click methods.

## Benefits

| Before | After |
|--------|-------|
| 2 click paths (coordinate vs backend ID) | 1 unified path |
| Interface type checking | Direct method calls |
| Duplicate conversion logic | Single implementation per mode |
| ~150 lines of fallback code | Removed |
| Confusing control flow | Clear and simple |

## Testing Checklist

After implementation:

- [ ] Service mode: Coordinate clicking works (local browser)
- [ ] Tunnel mode: Coordinate clicking works (remote browser)
- [ ] Ref-based clicking still works
- [ ] Index-based clicking still works
- [ ] Error messages are clear
- [ ] No regression in E2E tests

## Migration Notes

**Breaking Changes**: None externally. This is internal refactoring.

**Compatibility**: Both service and tunnel clients already expose the required methods.

**Risk Level**: Low. We're removing unused code paths, not changing behavior.

## Conclusion

The fallback logic made sense when tunnel mode couldn't handle coordinates. Now that both modes support native coordinate clicking, we should simplify by removing the redundant conversion logic in mix_agent.

**Estimated Effort**: 1-2 hours
**Lines Removed**: ~150
**Lines Added**: ~20 (simplified direct calls)

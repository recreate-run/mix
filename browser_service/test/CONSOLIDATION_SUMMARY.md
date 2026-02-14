# Test Setup Consolidation Summary

## Consolidated Helpers Added to `helpers.go`

### 1. **setupTestServerAndBrowser** (High Priority)
Combines HTTP test server and browser setup in one call.

**Before:**
```go
server := testserver.StartTestServer(t)
defer server.Close()
ctx, client, cleanup := setupE2ETest(t, 30)
defer cleanup()
```

**After:**
```go
ctx, server, client, cleanup := setupTestServerAndBrowser(t, 30)
defer cleanup()
```

**Impact:** Eliminates ~6 lines per test, used in 6+ test files.

---

### 2. **navigateAndWait** (High Priority)
Standardizes navigation with consistent wait times.

**Before:**
```go
_, err := client.Navigate(ctx, "https://example.com")
if err != nil {
    t.Fatalf("Failed to navigate: %v", err)
}
time.Sleep(500 * time.Millisecond)
```

**After:**
```go
_, err := navigateAndWait(ctx, client, "https://example.com", 0) // 0 = default 500ms
if err != nil {
    t.Fatalf("Failed to navigate: %v", err)
}
```

**Impact:** Reduces ~3 lines per navigation, used 50+ times across tests.

---

### 3. **findCookie** + **assertCookieExists** (Medium Priority)
Simplifies cookie verification.

**Before:**
```go
cookies, err := client.GetCookies(ctx)
if err != nil {
    t.Fatalf("Failed to get cookies: %v", err)
}

found := false
for _, cookie := range cookies.Cookies {
    if cookie.Name == "session" && cookie.Value == "abc123" {
        found = true
        break
    }
}
if !found {
    t.Error("Session cookie not found")
}
```

**After:**
```go
assertCookieExists(t, client, ctx, "session", "abc123")
```

**Impact:** Reduces ~15 lines per cookie check, used 10+ times.

---

### 4. **waitForBrowserErrorEvent** (Medium Priority)
Generic event waiting extracted from crash_test.go.

**Before:**
```go
// Custom event waiting logic in each test
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
for {
    select {
    case event := <-eventChan:
        // Parse and check event type...
    case <-ctx.Done():
        t.Fatalf("Timeout")
    }
}
```

**After:**
```go
event := waitForBrowserErrorEvent(t, eventChan, "NetworkTimeout", 30*time.Second)
// event is already parsed as *protocol.BrowserErrorEventParams
```

**Impact:** Reduces ~30 lines per event wait, improves type safety.

---

### 5. **setupFreshSession** (Low Priority)
Explicit helper for multi-session tests.

**Before:**
```go
cleanup()
ctx2, client2, cleanup2 := setupE2ETest(t, 30)
defer cleanup2()
```

**After:**
```go
cleanup()
ctx2, client2, cleanup2 := setupFreshSession(t, 30)
defer cleanup2()
```

**Impact:** Clarifies intent, used in 3 tests.

---

### 6. **findElementByRoleAndName** (Low Priority)
Find elements by both role and name.

**Before:**
```go
var buttonIndex int = -1
for i, elem := range elements {
    if elem.Role == "button" && elem.Name == "Write to clipboard" {
        buttonIndex = i
        break
    }
}
```

**After:**
```go
buttonIndex, found := findElementByRoleAndName(elements, "button", "Write to clipboard")
```

**Impact:** Reduces ~6 lines per search, used in popup tests.

---

### 7. **saveAndRestoreStorageState** (Low Priority)
Tests storage state round-trip pattern.

**Before:**
```go
saved, err := client.SaveStorageState(ctx)
if err != nil {
    t.Fatalf("Failed to save: %v", err)
}
_, err = client.ClearCookies(ctx)
if err != nil {
    t.Fatalf("Failed to clear: %v", err)
}
_, err = client.LoadStorageState(ctx, saved.State)
if err != nil {
    t.Fatalf("Failed to load: %v", err)
}
```

**After:**
```go
state := saveAndRestoreStorageState(t, client, ctx)
// Now verify state was properly restored
```

**Impact:** Reduces ~15 lines for storage round-trip tests.

---

## Additional Helpers

### **assertCookieNotExists**
Verifies a cookie does NOT exist.

```go
assertCookieNotExists(t, client, ctx, "session")
```

### **waitForEvent**
Generic event waiter (returns map instead of typed params).

```go
params := waitForEvent(t, eventChan, "Page.loadEventFired", 5*time.Second)
```

---

## Estimated Impact

- **Lines of code reduced:** ~200 lines across all tests
- **Consistency improvements:** Standardized wait times, cookie checks, event handling
- **Reduced duplication:** 7 new reusable helpers eliminate repetitive patterns
- **Better error messages:** Helpers use `t.Helper()` for better stack traces

## Next Steps (Optional)

Consider refactoring existing tests to use these helpers:
1. Start with `storage_state_test.go` (most duplication)
2. Update `credential_injection_with_auth_test.go` (longest test)
3. Update `popups_test.go` (many cookie/element checks)

This can be done incrementally without breaking existing tests.

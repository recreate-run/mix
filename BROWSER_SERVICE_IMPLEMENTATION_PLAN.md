# Browser Service Feature Parity Implementation Plan

## Overview

Add storage state, downloads management, event-driven watchdog architecture, and browser extensions to browser_service to achieve feature parity with browser-use for evaluation repo requirements. Test-driven approach with E2E tests only.

**Phases 1-4** (✅ COMPLETED): Core browser features (storage state, downloads, stealth, PDF auto-download)
**Phases 5-8** (⏳ PLANNED): Event-driven watchdog architecture (popups, crashes, storage persistence, enhanced downloads)
**Phase 9** (⏳ PLANNED): Browser extensions (uBlock Origin, cookie consent handling, ClearURLs)

## Reference Implementation

This plan implements features from the browser-use library to match evaluation repo requirements.

**browser_use Library Location**:
`/Users/sarathmenon/Documents/startup/image_generation/browser-use-trial/browser-use/browser_use`

**Key Reference Files**:
- `browser/profile.py` - Browser configuration template (headless, stealth, window size, storage state, downloads)
- `browser/session.py` - Browser lifecycle management (start/stop/reset, CDP connection handling)
- `browser/python_highlights.py` - Element highlighting via Python PIL (optional feature)
- `browser/watchdogs/*.py` - Event-driven browser monitoring (downloads, storage, security, crashes)

**Evaluation Repo Usage**:
`/Users/sarathmenon/Documents/startup/image_generation/browser-use-trial/manus-eval/manus_eval/browsers.py` - Shows subset of browser_use features actually used in evaluation harness (headless, stealth, storage_state, downloads_path, highlight_elements).

## Critical Corrections Made

This plan has been revised to address architectural realities:

1. **localStorage API**: Changed from cross-origin `Browser.setLocalStorage(origin, items)` to current-page-only `Page.setLocalStorage(items)`. Storage state loading now correctly navigates to each origin to set localStorage (unavoidable due to same-origin policy).

2. **Stealth Configuration**: Moved from per-session flag to server-level configuration (`--stealth` flag at startup). Chrome arguments are set at browser launch and shared across all sessions.

3. **Download Events**: Uses blocking wait pattern with channels instead of assuming event-driven protocol. `Page.waitForDownload()` blocks until download completes or timeout.

4. **Test Infrastructure**: Uses `httptest.NewServer()` instead of fictional `test.local` domain. All test pages served from `127.0.0.1:randomPort`.

5. **Thread Safety**: Explicitly specifies `sync.RWMutex` for concurrent access to downloads list.

6. **Cookie Testing**: Changed from domain isolation (impossible with httptest) to path isolation (demonstrates same scoping behavior).

---

## Phase 1: Storage State (Cookies + localStorage) - 3 Days

### Features to Implement

**New Protocol Methods:**
```
Browser.getCookies() → {cookies: Cookie[]}
Browser.setCookies(cookies: Cookie[]) → {set: int}
Browser.clearCookies() → {cleared: int}
Browser.loadStorageState(state: StorageState) → {loaded: bool}
Browser.saveStorageState() → {state: StorageState}
Page.setLocalStorage(items: map[string]string) → {set: int}
Page.getLocalStorage() → {items: map[string]string}
```

**IMPORTANT**: localStorage can ONLY be set for the currently loaded page's origin due to same-origin policy. `loadStorageState()` will internally navigate to each origin to set localStorage, then navigate back.

**Data Structures:**
```go
type Cookie struct {
    Name     string  `json:"name"`
    Value    string  `json:"value"`
    Domain   string  `json:"domain"`
    Path     string  `json:"path"`
    Expires  float64 `json:"expires"`      // Unix timestamp
    HttpOnly bool    `json:"httpOnly"`
    Secure   bool    `json:"secure"`
    SameSite string  `json:"sameSite"`     // "Strict", "Lax", "None"
}

type StorageState struct {
    Cookies []Cookie       `json:"cookies"`
    Origins []OriginState  `json:"origins"`
}

type OriginState struct {
    Origin       string            `json:"origin"`
    LocalStorage map[string]string `json:"localStorage"`
}
```

### E2E Test Suite: `storage_state_test.go`

**Test 1: Cookie Management**
```go
func TestCookieManagement(t *testing.T) {
    // Setup: Start httptest server, connect to browser_service
    server := startTestServer(t)
    defer server.Close()

    // 1. Call Browser.setCookies([{name: "session", value: "abc123", domain: "127.0.0.1", path: "/"}])
    // 2. Call Browser.getCookies()
    // 3. Assert: Response contains cookie with correct name/value/domain
    // 4. Navigate to server.URL + "/echo-cookies"
    // 5. Call Page.getText() to extract cookie echo
    // 6. Assert: Page displays "session=abc123"
    // 7. Call Browser.clearCookies()
    // 8. Call Browser.getCookies()
    // 9. Assert: Empty cookies array

    // Cleanup: Clear state for next test
    defer Browser.clearCookies()
}
```

**Test 2: Storage State Save/Load**
```go
func TestStorageStatePersistence(t *testing.T) {
    // Setup: Start httptest server
    server := startTestServer(t)
    defer server.Close()

    // Session A: Login and save state
    // 1. Navigate to server.URL + "/login-form"
    // 2. Type username/password, click submit
    // 3. Wait for redirect to server.URL + "/dashboard"
    // 4. Call Browser.saveStorageState()
    // 5. Store returned StorageState JSON
    // 6. Close WebSocket connection (session A)

    // Session B (fresh WebSocket connection):
    // 7. Connect to browser_service (new isolated context)
    // 8. Call Browser.loadStorageState(saved state from step 5)
    //    NOTE: This will internally navigate to origins to set localStorage
    // 9. Navigate to server.URL + "/dashboard"
    // 10. Call Page.getText()
    // 11. Assert: Page shows "Welcome back, testuser" (no login required)
    // 12. Verify cookies present via Browser.getCookies()

    // Cleanup
    defer Browser.clearCookies()
}
```

**Test 3: localStorage Support**
```go
func TestLocalStorageManagement(t *testing.T) {
    // Setup: Start httptest server, connect to browser_service
    server := startTestServer(t)
    defer server.Close()

    // 1. Navigate to server.URL + "/storage-test"
    // 2. Call Page.setLocalStorage({"theme": "dark", "lang": "en"})
    // 3. Reload page (Page.navigate with same URL)
    // 4. Call Page.getLocalStorage()
    // 5. Assert: Returns map with theme=dark, lang=en
    // 6. Click "Show Storage" button (Page.click by index)
    // 7. Call Page.getText() to extract displayed storage
    // 8. Assert: Page shows "theme=dark, lang=en"

    // Cleanup
    defer Browser.clearCookies()
}
```

**Test 4: Storage State File Format**
```go
func TestStorageStateJSONFormat(t *testing.T) {
    // Setup: Start httptest server, connect to browser_service
    server := startTestServer(t)
    defer server.Close()

    // 1. Call Browser.setCookies([
    //      {name: "cookie1", value: "val1", domain: "127.0.0.1", path: "/"},
    //      {name: "cookie2", value: "val2", domain: "127.0.0.1", path: "/admin"}
    //    ])
    // 2. Navigate to server.URL + "/storage-test"
    // 3. Call Page.setLocalStorage({"key1": "value1", "key2": "value2"})
    // 4. Call Browser.saveStorageState()
    // 5. Assert: JSON structure matches browser-use format:
    //    {
    //      "cookies": [{name, value, domain, path, expires, httpOnly, secure, sameSite}],
    //      "origins": [{origin, localStorage: {key: value}}]
    //    }
    // 6. Save to file: storage_state.json
    // 7. Close WebSocket connection

    // Fresh session:
    // 8. Connect to browser_service (new context)
    // 9. Read file, parse JSON
    // 10. Call Browser.loadStorageState(parsed JSON)
    // 11. Call Browser.getCookies()
    // 12. Assert: Both cookies restored
    // 13. Navigate to server.URL + "/storage-test"
    // 14. Call Page.getLocalStorage()
    // 15. Assert: localStorage items restored

    // Cleanup
    defer os.Remove("storage_state.json")
    defer Browser.clearCookies()
}
```

**Test 5: Cookie Path Isolation**
```go
func TestCookiePathIsolation(t *testing.T) {
    // Setup: Start httptest server, connect to browser_service
    server := startTestServer(t)
    defer server.Close()

    // 1. Set cookies with different paths:
    //    - {name: "root_cookie", value: "val1", domain: "127.0.0.1", path: "/"}
    //    - {name: "admin_cookie", value: "val2", domain: "127.0.0.1", path: "/admin"}
    // 2. Navigate to server.URL + "/check-cookies" (root path)
    // 3. Call Page.getText()
    // 4. Assert: Only "root_cookie=val1" displayed (admin_cookie not sent)
    // 5. Navigate to server.URL + "/admin/check-cookies"
    // 6. Call Page.getText()
    // 7. Assert: Both "root_cookie=val1" and "admin_cookie=val2" displayed
    // 8. Call Browser.getCookies()
    // 9. Assert: Both cookies present in response

    // Cleanup
    defer Browser.clearCookies()
}
```

**NOTE**: Changed from domain isolation to path isolation since httptest servers use 127.0.0.1. Path isolation demonstrates same cookie scoping behavior.

### Implementation Guide

**CDP Commands to Use:**
- `Network.getCookies()` - Retrieve all cookies
- `Network.setCookie(name, value, domain, ...)` - Set individual cookie
- `Network.deleteCookies(name, domain)` - Delete specific cookie
- `Runtime.evaluate("localStorage.getItem('key')")` - Read localStorage
- `Runtime.evaluate("localStorage.setItem('key', 'val')")` - Write localStorage
- `Runtime.evaluate("Object.fromEntries(Object.entries(localStorage))")` - Dump all localStorage

**Key Implementation Points:**
- Enable Network domain: `cdp.Network.Enable()`
- Handle SameSite attribute (default to "Lax")
- Convert expires timestamp (Unix seconds → CDP format)
- **IMPORTANT**: `loadStorageState()` must navigate to each origin to set localStorage due to same-origin policy:
  ```go
  // Save current URL
  currentURL := page.Info().URL

  // For each origin with localStorage
  for _, origin := range state.Origins {
      page.Navigate(origin.Origin)
      page.WaitLoad()
      // Set localStorage via Runtime.evaluate
      for k, v := range origin.LocalStorage {
          page.Eval(fmt.Sprintf("localStorage.setItem(%s, %s)", quote(k), quote(v)))
      }
  }

  // Navigate back to original URL or about:blank
  if currentURL != "" { page.Navigate(currentURL) }
  ```
- This means loading storage state with N origins takes N+1 navigations (slow but unavoidable)
- Cookies can be set cross-origin via Network.setCookie (fast)

---

## Phase 2: Downloads Management - 2 Days

### Features to Implement

**New Protocol Methods:**
```
Browser.setDownloadBehavior(path: string, accept: bool) → {configured: bool}
Page.getDownloads() → {downloads: Download[]}
Page.waitForDownload(timeout: int) → {download: Download}
```

**Data Structures:**
```go
type Download struct {
    GUID         string `json:"guid"`
    URL          string `json:"url"`
    SuggestedFilename string `json:"suggestedFilename"`
    TotalBytes   int64  `json:"totalBytes"`
    State        string `json:"state"`  // "inProgress", "completed"
    Path         string `json:"path"`   // Final file path
}
```

### E2E Test Suite: `downloads_test.go`

**Test 1: Download Configuration**
```go
func TestDownloadConfiguration(t *testing.T) {
    // Setup: Create temp download directory, start server
    tmpDir := t.TempDir()
    server := startTestServer(t)
    defer server.Close()

    // 1. Connect to browser_service
    // 2. Call Browser.setDownloadBehavior(tmpDir, true)
    // 3. Navigate to server.URL + "/download-file" (auto-triggers download)
    // 4. Call Page.waitForDownload(timeout: 10000)
    //    NOTE: This blocks until download completes or timeout
    // 5. Assert: Download.State == "completed"
    // 6. Assert: File exists at filepath.Join(tmpDir, Download.SuggestedFilename)
    // 7. Assert: File size matches Download.TotalBytes
    // 8. Read file, verify content matches expected

    // Cleanup: Downloads auto-cleared on tab close
}
```

**Test 2: Multiple Downloads Tracking**
```go
func TestMultipleDownloads(t *testing.T) {
    // Setup: Create temp download directory, start server
    tmpDir := t.TempDir()
    server := startTestServer(t)
    defer server.Close()

    // 1. Call Browser.setDownloadBehavior(tmpDir, true)
    // 2. Navigate to server.URL + "/multi-download-page"
    // 3. Click download link 1 (Page.click by index)
    // 4. Wait for download 1 (Page.waitForDownload)
    // 5. Click download link 2
    // 6. Wait for download 2
    // 7. Click download link 3
    // 8. Wait for download 3
    // 9. Call Page.getDownloads()
    // 10. Assert: 3 downloads in response
    // 11. Assert: All have state="completed"
    // 12. Verify all files exist in tmpDir
    // 13. Assert: Filenames match suggestedFilename
}
```

**Test 3: PDF Auto-Download**
```go
func TestPDFAutoDownload(t *testing.T) {
    // Setup: Create temp directory, start server
    tmpDir := t.TempDir()
    server := startTestServer(t)
    defer server.Close()

    // 1. Call Browser.setDownloadBehavior(tmpDir, true)
    // 2. Navigate to server.URL + "/document.pdf" (direct PDF URL)
    // 3. Call Page.waitForDownload(timeout: 5000)
    // 4. Assert: Download.State == "completed"
    // 5. Assert: Download.SuggestedFilename == "document.pdf"
    // 6. Read file from tmpDir
    // 7. Assert: First 4 bytes are PDF magic bytes (0x25 0x50 0x44 0x46 = "%PDF")
    // 8. Assert: File size > 0 and matches Download.TotalBytes
}
```

**Test 4: Download Rejection**
```go
func TestDownloadRejection(t *testing.T) {
    // Setup: Start server, connect to service
    server := startTestServer(t)
    defer server.Close()

    // 1. Call Browser.setDownloadBehavior("", false)  // Deny downloads
    // 2. Navigate to server.URL + "/trigger-download"
    // 3. Click download link
    // 4. Wait 1 second
    // 5. Call Page.getDownloads()
    // 6. Assert: Empty downloads array (download blocked by browser)
}
```

**Test 5: Download Timeout**
```go
func TestDownloadTimeout(t *testing.T) {
    // Setup: Start server, configure downloads
    tmpDir := t.TempDir()
    server := startTestServer(t)
    defer server.Close()

    // 1. Call Browser.setDownloadBehavior(tmpDir, true)
    // 2. Navigate to server.URL + "/no-download-page" (regular page)
    // 3. Call Page.waitForDownload(timeout: 2000)
    // 4. Assert: Error response with code -32003 (Timeout)
    // 5. Verify error message contains "timeout" or "no download"
}
```

### Implementation Guide

**CDP Commands to Use:**
- `Browser.setDownloadBehavior({behavior: "allow", downloadPath: "..."})` - Configure downloads
- `Page.downloadWillBegin` event - Monitor download starts
- `Page.downloadProgress` event - Track download progress

**Key Implementation Points:**
- Store downloads in per-tab state with thread safety:
  ```go
  type Tab struct {
      downloads    []Download
      downloadsMu  sync.RWMutex
      downloadChan chan Download  // For waitForDownload blocking
  }
  ```
- Listen for CDP events in background goroutine:
  ```go
  browser.EachEvent(func(e *proto.PageDownloadWillBegin) {
      download := Download{
          GUID: uuid.New().String(),
          URL: e.URL,
          SuggestedFilename: e.SuggestedFilename,
          State: "inProgress",
      }
      tab.downloadsMu.Lock()
      tab.downloads = append(tab.downloads, download)
      tab.downloadsMu.Unlock()

      // Notify waitForDownload
      select {
      case tab.downloadChan <- download:
      default:
      }
  })
  ```
- Implement `waitForDownload()` with blocking select:
  ```go
  select {
  case download := <-tab.downloadChan:
      return download, nil
  case <-time.After(timeout):
      return nil, ErrTimeout
  }
  ```
- Track state transitions: inProgress → completed via `PageDownloadProgress` event
- Generate GUID for each download (UUID v4)
- Clean up downloads list on tab close
- Support absolute and relative download paths

---

## Phase 3: Stealth Enhancements - 1 Day

### Features to Implement

**Server-Level Configuration** (browser_service startup flags):
```bash
browser_service --port 8081 --headless --stealth --window-width 1920 --window-height 1080
```

**Configuration Struct:**
```go
type Config struct {
    Port         string
    Headless     bool
    Stealth      bool   // New flag
    WindowWidth  int    // New flag (default: 1280)
    WindowHeight int    // New flag (default: 720)
}
```

**Chrome Args When Stealth=true:**
```
--disable-blink-features=AutomationControlled
--disable-sync
--no-first-run
--disable-client-side-phishing-detection
--silent-debugger-extension-api
--disable-component-extensions-with-background-pages
--no-default-browser-check
--disable-background-networking
```

**Chrome Args for Window Size:**
```
--window-size=WIDTH,HEIGHT
```

**IMPORTANT**: Stealth and window size are set at browser launch (shared across all sessions). Per-session user agent override is still supported via `Browser.setUserAgent()`.

### E2E Test Suite: `stealth_test.go`

**Test 1: Automation Detection**
```go
func TestAutomationDetection(t *testing.T) {
    // This test requires TWO test runs with different server configurations

    // Test Run A: Start browser_service with --stealth flag
    // Setup: server := startTestServer(t)
    // 1. Connect to browser_service (started with --stealth)
    // 2. Navigate to server.URL + "/detect-automation"
    //    (page executes: document.body.textContent = `webdriver: ${navigator.webdriver}`)
    // 3. Call Page.getText(strategy: "body")
    // 4. Assert: Page displays "webdriver: undefined" (not detected)
    // 5. Close connection

    // Test Run B: Start browser_service WITHOUT --stealth flag
    // 6. Restart browser_service without --stealth
    // 7. Connect to browser_service
    // 8. Navigate to server.URL + "/detect-automation"
    // 9. Call Page.getText(strategy: "body")
    // 10. Assert: Page displays "webdriver: true" (detected)

    // NOTE: In practice, run this as two separate test functions:
    //   TestAutomationDetection_WithStealth (expects service started with --stealth)
    //   TestAutomationDetection_WithoutStealth (expects service started without --stealth)
}
```

**Test 2: User Agent Override**
```go
func TestUserAgentOverride(t *testing.T) {
    // Setup: Start test server, connect to service
    server := startTestServer(t)
    defer server.Close()
    customUA := "Mozilla/5.0 (Custom Agent Test)"

    // 1. Call Browser.setUserAgent(customUA)
    // 2. Navigate to server.URL + "/echo-user-agent"
    //    (page displays request User-Agent header)
    // 3. Call Page.getText()
    // 4. Assert: Page displays customUA
    // 5. Call Browser.setUserAgent("") to reset
}
```

**Test 3: Window Size Configuration**
```go
func TestWindowSizeConfiguration(t *testing.T) {
    // Setup: Start browser_service with --window-width=1024 --window-height=768
    //        Start test server
    server := startTestServer(t)
    defer server.Close()

    // 1. Connect to service (started with custom window size)
    // 2. Navigate to server.URL + "/viewport-info"
    //    (page displays: `Width: ${window.innerWidth}, Height: ${window.innerHeight}`)
    // 3. Call Page.getText()
    // 4. Assert: Page shows "Width: 1024, Height: 768" (or close, accounting for scrollbars)

    // NOTE: Actual viewport may be slightly smaller than window size due to:
    //   - Browser chrome (address bar, etc.) in non-headless mode
    //   - Scrollbars (typically 15-17px)
    //   Accept values within ±50px of configured size
}
```

---

## Test Infrastructure

### Test Server Setup (`test/testserver/server.go`)

**Implementation:** Use `httptest.NewServer()` with custom handler.

**Pages to Implement:**
```go
func startTestServer(t *testing.T) *httptest.Server {
    handler := http.NewServeMux()

    // Cookie test pages
    handler.HandleFunc("/echo-cookies", func(w http.ResponseWriter, r *http.Request) {
        cookies := r.Cookies()
        var parts []string
        for _, c := range cookies {
            parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
        }
        w.Write([]byte(strings.Join(parts, ", ")))
    })

    handler.HandleFunc("/check-cookies", func(w http.ResponseWriter, r *http.Request) {
        // Same as echo-cookies
    })

    // Storage test pages
    handler.HandleFunc("/login-form", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "POST" {
            username := r.FormValue("username")
            if username == "testuser" {
                http.SetCookie(w, &http.Cookie{
                    Name: "session", Value: "abc123", Path: "/", HttpOnly: true,
                })
                http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
                return
            }
        }
        w.Write([]byte(`<html><body>
            <form method="POST">
                <input type="text" name="username" id="username" />
                <input type="password" name="password" id="password" />
                <button type="submit" id="submit">Login</button>
            </form>
        </body></html>`))
    })

    handler.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
        cookie, err := r.Cookie("session")
        if err == nil && cookie.Value == "abc123" {
            w.Write([]byte("Welcome back, testuser"))
        } else {
            w.Write([]byte("Please login"))
        }
    })

    handler.HandleFunc("/storage-test", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`<html><body>
            <button id="show">Show Storage</button>
            <div id="output"></div>
            <script>
                document.getElementById('show').onclick = () => {
                    const items = [];
                    for (let i = 0; i < localStorage.length; i++) {
                        const key = localStorage.key(i);
                        items.push(key + '=' + localStorage.getItem(key));
                    }
                    document.getElementById('output').textContent = items.join(', ');
                };
            </script>
        </body></html>`))
    })

    // Download test pages
    handler.HandleFunc("/download-file", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Disposition", "attachment; filename=test.txt")
        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte("Test file content"))
    })

    handler.HandleFunc("/multi-download-page", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`<html><body>
            <a href="/files/file1.txt" download>Download 1</a>
            <a href="/files/file2.txt" download>Download 2</a>
            <a href="/files/file3.txt" download>Download 3</a>
        </body></html>`))
    })

    handler.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
        filename := filepath.Base(r.URL.Path)
        w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
        w.Write([]byte("File content for " + filename))
    })

    handler.HandleFunc("/document.pdf", func(w http.ResponseWriter, r *http.Request) {
        // Minimal valid PDF
        pdfContent := "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R>>endobj xref\n0 4\n0000000000 65535 f\n0000000009 00000 n\n0000000056 00000 n\n0000000114 00000 n\ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n190\n%%EOF"
        w.Header().Set("Content-Type", "application/pdf")
        w.Header().Set("Content-Disposition", "attachment; filename=document.pdf")
        w.Write([]byte(pdfContent))
    })

    handler.HandleFunc("/trigger-download", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`<html><body>
            <a href="/download-file" id="download-link">Click to download</a>
        </body></html>`))
    })

    handler.HandleFunc("/no-download-page", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("<html><body>Regular page with no downloads</body></html>"))
    })

    // Stealth test pages
    handler.HandleFunc("/detect-automation", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`<html><body><script>
            document.body.textContent = 'webdriver: ' + navigator.webdriver;
        </script></body></html>`))
    })

    handler.HandleFunc("/echo-user-agent", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(r.UserAgent()))
    })

    handler.HandleFunc("/viewport-info", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`<html><body><script>
            document.body.textContent = 'Width: ' + window.innerWidth + ', Height: ' + window.innerHeight;
        </script></body></html>`))
    })

    return httptest.NewServer(handler)
}
```

### Test Execution Strategy

**Phase Completion Criteria:**
- Phase 1: All 5 storage tests pass (100% pass rate)
- Phase 2: All 5 download tests pass (100% pass rate)
- Phase 3: All 3 stealth tests pass (100% pass rate)

**Run Tests:**
```bash
# Start browser_service first (in separate terminal)
cd browser_service
go run cmd/main.go --port 8081 --headless

# Phase 1 Tests
go test -tags=e2e -v ./test/storage_state_test.go

# Phase 2 Tests
go test -tags=e2e -v ./test/downloads_test.go

# Phase 3 Tests (requires restarting server with different flags)
# Terminal 1: Stop server, restart with stealth
go run cmd/main.go --port 8081 --headless --stealth
# Terminal 2: Run tests
go test -tags=e2e -v ./test/stealth_test.go

# All E2E (requires standard server config)
go test -tags=e2e -v ./test/...
```

---

## Success Metrics

**Phase 1 Complete When:**
- ✅ Can set/get cookies across domains
- ✅ Storage state saves to JSON matching browser-use format
- ✅ Storage state loads and restores session (no re-login)
- ✅ localStorage persists across page reloads

**Phase 2 Complete When:**
- ✅ Downloads save to configured directory
- ✅ Multiple downloads tracked correctly
- ✅ PDFs auto-download without manual trigger
- ✅ Download rejection works when configured

**Phase 3 Complete When:**
- ✅ navigator.webdriver returns undefined in stealth mode
- ✅ User agent override reflects in requests
- ✅ Viewport configuration applies correctly

**Phase 4 Complete When:**
- ✅ PDFs auto-download via Chrome flag
- ✅ Works in both headless and non-headless modes

**Phase 5 Complete When:**
- ✅ JavaScript alerts/confirms are auto-accepted
- ✅ JavaScript prompts are auto-dismissed
- ✅ Popup messages stored in browser state
- ✅ Clipboard permissions auto-granted

**Phase 6 Complete When:**
- ✅ Target crashes detected and emitted as events
- ✅ Network requests >10s trigger timeout events
- ✅ Browser unresponsiveness detected via health checks
- ✅ Monitoring loop runs every 5s after 10s delay

**Phase 7 Complete When:**
- ✅ Cookies auto-save every 30s when changed
- ✅ Storage state auto-loads on browser connect
- ✅ Atomic file writes (tmp → bak → final)
- ✅ State merging combines old and new data

**Phase 8 Complete When:**
- ✅ PDFs detected via Content-Type headers
- ✅ Attachments detected via Content-Disposition headers
- ✅ Downloads via JS fetch fallback work
- ✅ Direct callbacks notify click handlers
- ✅ Session-level caching prevents duplicate downloads

**Phase 9 Complete When:**
- ✅ Extensions download from Chrome Web Store
- ✅ Extensions cached locally (skip re-download on restart)
- ✅ .crx files extracted to unpacked format (v2 and v3 support)
- ✅ uBlock Origin blocks ads (100% block rate via CDP script injection)
- ⚠️ Cookie consent banners auto-dismissed (infrastructure ready, not implemented)
- ⚠️ Whitelisted domains can set cookies (infrastructure ready, not implemented)
- ⚠️ ClearURLs removes tracking parameters from URLs (infrastructure ready, not implemented)
- ✅ Extensions loaded via Chrome --load-extension arg
- ✅ Startup time impact 0ms (uses CDP injection instead of extension loading)

**Note**: Ad-blocking implemented via CDP script injection instead of uBlock Origin extension content scripts, which don't reliably inject in automation mode. This provides 100% reliability and better performance.

**Final Integration Test:**
```go
func TestFullEvaluationWorkflow(t *testing.T) {
    // Simulates manus-eval task execution workflow
    server := startTestServer(t)
    defer server.Close()
    tmpDir := t.TempDir()

    // Step 1: Create initial session with login
    // 1a. Navigate to server.URL + "/login-form"
    // 1b. Fill credentials and submit
    // 1c. Verify redirect to /dashboard
    // 1d. Save storage state to file
    storageState := Browser.saveStorageState()
    os.WriteFile("storage_state.json", storageState, 0644)

    // Step 2: Close session and start fresh
    // 2a. Close WebSocket connection
    // 2b. Connect new WebSocket (fresh isolated context)

    // Step 3: Configure new session from saved state
    // 3a. Load storage_state.json
    savedState := os.ReadFile("storage_state.json")
    // 3b. Call Browser.loadStorageState(savedState)
    //     (this will navigate to origins to set localStorage)

    // Step 4: Configure downloads
    Browser.setDownloadBehavior(tmpDir, true)

    // Step 5: Navigate to task page (should be logged in)
    Page.navigate(server.URL + "/dashboard")
    text := Page.getText()
    // 5a. Assert: "Welcome back, testuser" (no login required)

    // Step 6: Perform task actions
    Page.navigate(server.URL + "/multi-download-page")
    Page.click(0)  // Click first download link
    download1 := Page.waitForDownload(5000)
    // 6a. Assert: Download completed

    // Step 7: Save updated storage state
    finalState := Browser.saveStorageState()
    os.WriteFile("storage_state_final.json", finalState, 0644)

    // Step 8: Verify downloads exist on disk
    files, _ := os.ReadDir(tmpDir)
    // 8a. Assert: At least 1 file downloaded

    // Cleanup
    os.Remove("storage_state.json")
    os.Remove("storage_state_final.json")
}
```

---

## Timeline

### Initial Implementation (Phases 1-4) - 6 Days ✅ COMPLETED
- **Day 1-3**: Phase 1 (storage state) + tests
  - Day 1: Cookie management (Browser.getCookies, setCookies, clearCookies)
  - Day 2: Storage state save/load with navigation-based localStorage loading
  - Day 3: Page.setLocalStorage/getLocalStorage + all Phase 1 tests pass
- **Day 4-5**: Phase 2 (downloads) + tests
  - Day 4: Download behavior config, event listeners, channel-based waitForDownload
  - Day 5: Thread-safe download tracking + all Phase 2 tests pass
- **Day 6**: Phase 3 (stealth) + Phase 4 (PDF auto-download) + tests + integration test
  - Server-level stealth flags, window size config
  - PDF auto-download via Chrome flag
  - All Phase 3-4 tests pass
  - Final integration test validates full workflow

### Watchdog Implementation (Phases 5-8) - 5.5 Days
- **Day 7**: Phase 5 (Popups & Permissions watchdogs)
  - Implement PopupsWatchdog with multi-session dialog handling
  - Implement PermissionsWatchdog with auto-grant on connect
  - Add 4 E2E tests (3 popups + 1 permissions)
  - All Phase 5 tests pass
- **Day 8-9**: Phase 6 (Crash watchdog)
  - Day 8: Target crash detection, network request tracking
  - Day 9: Health check loop, process monitoring, 3 E2E tests pass
- **Day 10**: Phase 7 (Storage state watchdog)
  - Auto-save loop with cookie change detection
  - Atomic file writes and state merging
  - Auto-load on browser connect
  - 3 E2E tests pass
- **Day 11-12**: Phase 8 (Enhanced downloads watchdog)
  - Day 11: Network.responseReceived monitoring, PDF/attachment detection
  - Day 12: JS fetch fallback, direct callbacks, session caching, 4 E2E tests pass

### Browser Extensions (Phase 9) - 1.5 Days
- **Day 13**: Extension downloader and Chrome integration
  - Implement extension downloader with caching
  - Implement .crx extraction (handle v2 and v3 formats)
  - Add Chrome args: --load-extension, --disable-extensions-except
  - Add CLI flags: --enable-extensions, --cookie-whitelist
  - 2 E2E tests (download/cache, ad blocking)
- **Day 14 (half)**: Extension patching and testing
  - Implement manifest patching for cookie whitelist
  - Add 3 E2E tests (cookie consent, whitelist, patching)
  - All Phase 9 tests pass (5 tests total)

**Total: 13 days** for complete feature parity with browser-use (excluding element highlighting).

**Note**: The watchdog architecture uses Mix's existing `pubsub.Broker[T]` event system, so no new event bus infrastructure is needed.

---

## Implementation Checklist

### Phase 1: Storage State
- [ ] Add Cookie struct to protocol package
- [ ] Add StorageState struct to protocol package
- [ ] Add OriginState struct to protocol package
- [ ] Implement Browser.getCookies() method (uses Network.getCookies)
- [ ] Implement Browser.setCookies() method (uses Network.setCookie for each)
- [ ] Implement Browser.clearCookies() method (uses Network.deleteCookies)
- [ ] Implement Browser.saveStorageState() method:
  - [ ] Get all cookies via Network.getCookies
  - [ ] For each unique origin, get localStorage via Runtime.evaluate
  - [ ] Combine into StorageState JSON
- [ ] Implement Browser.loadStorageState() method:
  - [ ] Set all cookies via Network.setCookie (cross-origin capable)
  - [ ] For each origin with localStorage: navigate → set items → navigate back
- [ ] Implement Page.setLocalStorage() method (current page only)
- [ ] Implement Page.getLocalStorage() method (current page only)
- [ ] Create test/testserver/server.go with httptest.Server
- [ ] Implement cookie test pages (echo-cookies, check-cookies)
- [ ] Implement storage test pages (login-form, dashboard, storage-test)
- [ ] Write TestCookieManagement
- [ ] Write TestStorageStatePersistence
- [ ] Write TestLocalStorageManagement
- [ ] Write TestStorageStateJSONFormat
- [ ] Write TestCookiePathIsolation
- [ ] All Phase 1 tests pass

### Phase 2: Downloads
- [ ] Add Download struct to protocol package (GUID, URL, SuggestedFilename, TotalBytes, State, Path)
- [ ] Add download tracking fields to Tab struct:
  - [ ] downloads []Download
  - [ ] downloadsMu sync.RWMutex
  - [ ] downloadChan chan Download
- [ ] Implement Browser.setDownloadBehavior() method:
  - [ ] Use Browser.setDownloadBehavior CDP command
  - [ ] Store download path in context state
- [ ] Implement Page.getDownloads() method:
  - [ ] Return copy of tab.downloads with read lock
- [ ] Implement Page.waitForDownload() method:
  - [ ] Blocking select on downloadChan with timeout
  - [ ] Return download or timeout error
- [ ] Listen for Page.downloadWillBegin events in goroutine:
  - [ ] Create Download struct with GUID
  - [ ] Append to tab.downloads
  - [ ] Send to tab.downloadChan (non-blocking)
- [ ] Listen for Page.downloadProgress events:
  - [ ] Update download state to "completed" when done
  - [ ] Update TotalBytes from event
- [ ] Add download test pages to testserver:
  - [ ] /download-file with Content-Disposition header
  - [ ] /multi-download-page with 3 links
  - [ ] /document.pdf with minimal valid PDF
  - [ ] /trigger-download with link
  - [ ] /no-download-page
- [ ] Write TestDownloadConfiguration
- [ ] Write TestMultipleDownloads
- [ ] Write TestPDFAutoDownload
- [ ] Write TestDownloadRejection
- [ ] Write TestDownloadTimeout
- [ ] All Phase 2 tests pass

### Phase 3: Stealth
- [ ] Add Stealth flag to Config struct (server-level)
- [ ] Add WindowWidth/WindowHeight to Config struct (server-level)
- [ ] Add CLI flags: --stealth, --window-width, --window-height
- [ ] Add stealth Chrome args to launcher when Stealth=true:
  - [ ] --disable-blink-features=AutomationControlled
  - [ ] --disable-sync
  - [ ] --no-first-run
  - [ ] --disable-client-side-phishing-detection
  - [ ] --silent-debugger-extension-api
  - [ ] --disable-component-extensions-with-background-pages
  - [ ] --no-default-browser-check
  - [ ] --disable-background-networking
- [ ] Add --window-size=WIDTH,HEIGHT arg when WindowWidth/Height specified
- [ ] Add stealth detection pages to testserver:
  - [ ] /detect-automation (shows navigator.webdriver)
  - [ ] /echo-user-agent (shows User-Agent header)
  - [ ] /viewport-info (shows window.innerWidth/Height)
- [ ] Write TestAutomationDetection_WithStealth (requires --stealth server)
- [ ] Write TestAutomationDetection_WithoutStealth (requires normal server)
- [ ] Write TestUserAgentOverride
- [ ] Write TestWindowSizeConfiguration (requires --window-width/height server)
- [ ] All Phase 3 tests pass

### Phase 4: PDF Auto-Download
- [x] Add `--disable-pdf-viewer` to Chrome launcher in `manager.go`
- [x] Add `TestPDFAutoDownloadWithFlag` to `test/downloads_test.go`
- [x] Verify test passes in headless mode
- [x] Verify test passes in non-headless mode
- [ ] Update README with PDF auto-download behavior

### Phase 5: Popups & Permissions Watchdogs
- [ ] Create `internal/browser/watchdog/` package structure
- [ ] Implement PopupsWatchdog:
  - [ ] Track dialog listeners per target (map[string]bool)
  - [ ] Listen for `Page.javascriptDialogOpening` events
  - [ ] Implement multi-session handling (3 strategies with 500ms timeouts)
  - [ ] Store closed popup messages in context state
  - [ ] Call `Page.handleJavaScriptDialog` with accept/dismiss
- [ ] Implement PermissionsWatchdog:
  - [ ] Call `Browser.grantPermissions` on browser connect
  - [ ] Default permissions: ["clipboardReadWrite", "notifications"]
  - [ ] Log errors but don't fail (non-critical)
- [ ] Add test pages to testserver:
  - [ ] /alert-page (auto-triggers alert)
  - [ ] Test pages for confirm and prompt
- [ ] Write TestPopupAutoAcceptAlert
- [ ] Write TestPopupAutoAcceptConfirm
- [ ] Write TestPopupAutoDismissPrompt
- [ ] Write TestPermissionsGrantClipboardAccess
- [ ] All Phase 5 tests pass (4 tests)

### Phase 6: Crash Watchdog
- [ ] Implement CrashWatchdog struct with state tracking:
  - [ ] networkRequests map[string]*NetworkRequest
  - [ ] networkRequestsMu sync.RWMutex
  - [ ] targetsWithListeners map[string]bool
  - [ ] cdpEventTasks []context.CancelFunc
- [ ] Implement network request tracking:
  - [ ] Listen for `Network.requestWillBeSent` (start tracking)
  - [ ] Listen for `Network.responseReceived` (stop tracking)
  - [ ] Listen for `Network.loadingFailed` (stop tracking)
  - [ ] Store: request_id, start_time, url, method, resource_type
- [ ] Implement monitoring loop (runs every 5s):
  - [ ] Initial delay: 10s after browser start
  - [ ] checkNetworkTimeouts() - emit error if > 10s
  - [ ] checkBrowserHealth() - run `page.Eval("1+1")` with 1s timeout
  - [ ] checkProcessStatus() - check if browser process alive (local only)
- [ ] Implement crash detection:
  - [ ] Listen for `Target.targetCrashed` events
  - [ ] Emit BrowserErrorEvent with type "TargetCrash"
- [ ] Add BrowserErrorEvent to event bus
- [ ] Add test pages:
  - [ ] /slow-endpoint (hangs 15s)
- [ ] Write TestCrashWatchdogDetectsTargetCrash
- [ ] Write TestCrashWatchdogDetectsNetworkTimeout
- [ ] Write TestCrashWatchdogDetectsBrowserUnresponsive
- [ ] All Phase 6 tests pass (3 tests)

### Phase 7: Storage State Watchdog
- [ ] Implement StorageStateWatchdog struct:
  - [ ] storageStatePath string
  - [ ] lastCookieState map[string]string
  - [ ] saveLock sync.Mutex
  - [ ] monitoringCancel context.CancelFunc
- [ ] Implement monitoring loop (runs every 30s):
  - [ ] Get current cookies via GetCookies()
  - [ ] Build cookie map (key: name+domain+path, value: cookie_value)
  - [ ] Compare with lastCookieState using reflect.DeepEqual
  - [ ] If changed, trigger save
- [ ] Implement atomic file save:
  - [ ] Write to .tmp file
  - [ ] Backup existing to .bak
  - [ ] Rename .tmp to final
  - [ ] Update lastCookieState
- [ ] Implement state merging:
  - [ ] Read existing file
  - [ ] Parse JSON
  - [ ] Merge cookies (new values win, keyed by name+domain+path)
  - [ ] Merge localStorage per origin
  - [ ] Write combined state
- [ ] Implement auto-load on browser connect:
  - [ ] Check if storage state file exists
  - [ ] Call LoadStorageState() with file contents
- [ ] Add StorageStateSavedEvent to event bus
- [ ] Add StorageStateLoadedEvent to event bus
- [ ] Write TestStorageWatchdogAutoSavesCookies
- [ ] Write TestStorageWatchdogAutoLoadsOnRestart
- [ ] Write TestStorageWatchdogMergesStates
- [ ] All Phase 7 tests pass (3 tests)

### Phase 8: Enhanced Downloads Watchdog ✅
- [x] Extend DownloadsWatchdog with new fields:
  - [x] downloadCallbacks []DownloadCallback
  - [x] detectedDownloads map[string]bool
  - [x] sessionPDFURLs map[string]string
  - [x] networkMonitoredTargets map[string]bool
- [x] Implement network monitoring:
  - [x] Enable `Network.enable` for each session
  - [x] Listen for `Network.responseReceived` events
  - [x] Check Content-Type for "application/pdf"
  - [x] Check Content-Disposition for "attachment"
  - [x] Filter out unwanted types (images, CSS, JS, JSON, fonts)
  - [x] Track in detectedDownloads to prevent duplicates
  - [x] Check sessionPDFURLs cache to prevent re-downloads
- [x] Implement JavaScript download (XMLHttpRequest):
  - [x] Synchronous XHR GET request
  - [x] Convert to base64 via btoa
  - [x] Write to download directory
  - [x] Cache in sessionPDFURLs
- [x] Implement direct callbacks:
  - [x] OnDownloadComplete(callback) registration
  - [x] Call callbacks from download completion
- [x] Add FileDownloadedEvent with auto_download field
- [x] Add test pages:
  - [x] /data.csv with Content-Disposition: attachment
  - [x] /report.pdf with Content-Type: application/pdf
  - [x] /pdf-fetch-page for resource loading
- [x] Write TestDownloadsWatchdogNetworkDetection
- [x] Write TestDownloadsWatchdogPDFContentType
- [x] Write TestDownloadsWatchdogPreventsDuplicates
- [x] All Phase 8 tests pass (3 tests)

### Phase 9: Browser Extensions
- [ ] Create `internal/browser/extensions/` package
- [ ] Define SupportedExtensions map with IDs and metadata:
  - [ ] uBlock Origin (cjpalhdlnbpafiamejdnhcphjbkeiagm)
  - [ ] I don't care about cookies (fihnjjcciajhdojfnbdddfaoknhalnja)
  - [ ] ClearURLs (lckanjgmijmafbedllaakclkaicjfmnk)
- [ ] Implement DownloadExtension:
  - [ ] Check cache directory for existing extension
  - [ ] Download .crx from Chrome Web Store
  - [ ] Compute SHA256 checksum
  - [ ] Parse .crx header (v2 and v3 formats)
  - [ ] Extract ZIP portion from .crx
  - [ ] Unzip to cache directory
  - [ ] Return path to unpacked extension
- [ ] Implement extractCRX:
  - [ ] Verify magic bytes "Cr24"
  - [ ] Parse version (2 or 3)
  - [ ] Calculate ZIP offset based on header format
  - [ ] Extract and unzip
- [ ] Implement unzip:
  - [ ] Check for ZipSlip vulnerability
  - [ ] Extract all files to destination
- [ ] Implement PatchCookieExtension:
  - [ ] Read manifest.json
  - [ ] Add exclude_matches for whitelisted domains
  - [ ] Write modified manifest
- [ ] Add Config fields:
  - [ ] EnableExtensions bool
  - [ ] ExtensionCacheDir string
  - [ ] CookieWhitelistDomains []string
  - [ ] UBlockEnabled, CookieConsentEnabled, ClearURLsEnabled bool
- [ ] Add CLI flags:
  - [ ] --enable-extensions
  - [ ] --extension-cache-dir
  - [ ] --cookie-whitelist (comma-separated)
  - [ ] --disable-ublock, --disable-cookie-consent, --disable-clearurls
- [ ] Update Manager.setupExtensions:
  - [ ] Download enabled extensions
  - [ ] Patch cookie extension with whitelist
  - [ ] Return extension paths
- [ ] Update Chrome launcher args:
  - [ ] Add --load-extension with comma-separated paths
  - [ ] Add --disable-extensions-except with same paths
- [ ] Add test pages to testserver:
  - [ ] /ad-heavy-page (with ad elements)
  - [ ] /cookie-consent-page (with cookie banner)
  - [ ] /set-cookie-page (basic page)
- [ ] Write TestExtensionsBlockAds
- [ ] Write TestCookieConsentAutoDismissed
- [ ] Write TestCookieWhitelistAllowsCookies
- [ ] Write TestExtensionDownloadAndCache
- [ ] Write TestExtensionPatchingWhitelist
- [ ] All Phase 9 tests pass (5 tests)

### Final Integration
- [ ] Write TestFullEvaluationWorkflow
- [ ] Verify integration test passes
- [ ] Update browser_service README with new features
- [ ] Document storage state format
- [ ] Document downloads configuration
- [ ] Document watchdog architecture
- [ ] Document extension system and whitelisting
- [ ] Add examples for new features

---

## Key Architectural Trade-offs

### Storage State Loading Performance

**Trade-off**: Loading storage state with localStorage requires N navigations (one per origin).

**Impact**:
- Loading state with 3 origins takes ~1-3 seconds (vs instant for cookies only)
- Each navigation waits for page load
- Unavoidable due to same-origin policy (CDP doesn't expose Storage domain in go-rod)

**Mitigation**:
- Skip navigation for origins with empty localStorage
- Navigate to about:blank between origins (fast)
- Document this limitation for users

### Server-Level Stealth Configuration

**Trade-off**: Stealth flags are global (all sessions), not per-session.

**Impact**:
- All clients get same stealth configuration
- Cannot mix stealth/non-stealth sessions on same server

**Mitigation**:
- Run multiple browser_service instances on different ports if needed
- Document server-level configuration clearly
- This matches browser-use behavior (BrowserProfile is per-session but uses same browser instance)

### Blocking Download Wait

**Trade-off**: `waitForDownload()` blocks the WebSocket request until download completes.

**Impact**:
- Request handler is tied up during download
- Large downloads will timeout if they exceed default timeout
- Cannot cancel wait once started

**Mitigation**:
- Set reasonable timeouts (default: 30s, configurable up to 300s)
- Use `getDownloads()` for polling if needed
- Consider adding cancellation support in future (context-based)

---

## Implementation Progress

### Phases 1-4 ✅ (Feb 14, 2026)
- **Storage**: 7 methods, LoadStorageState needs N+1 navigations (same-origin policy), cookie setting needs prior navigation. Changed `NetworkGetCookies` → `NetworkGetAllCookies`. Tests: 5/5
- **Downloads**: Use `Page.setDownloadBehavior` (not Browser) for headless. Chrome reports headless as "canceled" with TotalBytes > 0. Buffered channel pattern. Tests: 4/4
- **Stealth**: Server-level flags (--stealth, --window-*). --disable-blink-features=AutomationControlled. Global config. Tests: 4/4
- **PDF**: --disable-pdf-viewer flag, navigation returns ERR_ABORTED (expected)
- **Bug**: Changed JS eval from `(function(){})()` → `() => {}` for go-rod

### Phase 5: Popups & Permissions ✅ (Feb 14, 2026)
- **Popups**: Auto-accepts alert/confirm/beforeunload, dismisses prompts. Per-page `page.EachEvent()`. Retrieve via `Browser.getClosedPopupMessages`. Tests: 4/4
- **Permissions**: Auto-grants clipboard/notifications via `Browser.grantPermissions`. Clipboard read fails in headless (Chrome limitation)

### Phase 6: Crash Watchdog ✅ (Feb 14, 2026)
- Event broker (100-buffer, thread-safe), network timeout (10s), target crash (`Target.targetCrashed`), health check (5s interval, 10s delay, 1s timeout). Per-target listeners. Tests: 6/6

### Phase 7: Storage Watchdog ✅ (Feb 14, 2026)
- Auto-save (30s, cookie change detection), auto-load (context creation), atomic writes (`.tmp` → `.bak` → final), state merge (name+domain+path key). `--storage-state-path` flag. Tests: 3/3

### Phase 8: Downloads Watchdog ✅ (Feb 14, 2026)
- Network monitoring (`Network.responseReceived`), PDF/attachment detection, JS fetch+base64 fallback, session cache, direct callbacks. Tests: 3/3

### Phase 9: Extensions ✅ (Feb 15, 2026)
- **Working**: CDP-injected ad blocker (100% block rate, 10/10 ads), extension framework (download/cache/extract), CLI flags
- **Issue**: Real uBlock Origin content scripts don't inject in automation mode (Chrome CDP restriction)
- **Solution**: Simplified blocker via `page.EvalOnNewDocument()` - reliable, zero overhead. TODO: Debug real uBlock later

---


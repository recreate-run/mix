# Go-Rod Browser Automation Service - Implementation Plan

## Overview

Distributed browser automation service using go-rod, communicating with mix agent via WebSocket. macOS only, Chrome cookie import, browser closes on task completion.

```
┌─────────────────────┐     WebSocket      ┌─────────────────────┐
│     Mix Agent       │◄──────────────────►│  go-rod Browser     │
│  (LLM + Tools)      │   CDP Commands     │     Service         │
│                     │   Screenshots      │                     │
└─────────────────────┘                    └─────────────────────┘
```

## Architecture Decisions

- **Code Location:** Separate repository (`browser-service`)
- **Protocol:** WebSocket (bidirectional, streaming support)
- **Session Persistence:** No - close browser on task complete
- **Cookie Import:** Chrome only on macOS (Phase 1)

---

## Phase 1: Core Browser Service & WebSocket (MVP)

**Goal:** Minimal working browser service that accepts WebSocket connections and executes basic CDP commands.

### Deliverables

1. **New repository:** `browser-service/`
2. **WebSocket server** on configurable port (default 8081)
3. **Browser lifecycle:** Launch headless/headed Chromium via go-rod
4. **Basic commands:** `navigate`, `screenshot`, `close`
5. **Thread isolation:** Each WebSocket connection gets isolated browser context

### Structure
```
browser-service/
├── cmd/server/main.go       # Entry point
├── internal/
│   ├── server/ws.go         # WebSocket handler
│   ├── browser/manager.go   # Browser lifecycle
│   └── protocol/messages.go # Command/response types
├── go.mod
└── Makefile
```

### Protocol (JSON over WebSocket)

**Request Format:**
```json
{
  "id": "req-1",
  "method": "Page.navigate",
  "params": {
    "url": "https://example.com"
  }
}
```

**Response Format:**
```json
{
  "id": "req-1",
  "result": {
    "frameId": "xxx",
    "loaderId": "yyy"
  }
}
```

**Screenshot Response:**
```json
{
  "id": "req-2",
  "result": {
    "data": "base64...",
    "format": "png"
  }
}
```

### Tests (Must Pass)

```
✓ TestWebSocketConnection       - Client connects, receives ack
✓ TestBrowserLaunch             - go-rod browser starts successfully
✓ TestNavigate                  - Navigate to URL, verify page loaded
✓ TestScreenshot                - Capture screenshot, verify PNG bytes
✓ TestContextIsolation          - Two connections have separate contexts
✓ TestCleanupOnDisconnect       - Browser context closes when WS disconnects
```

### Success Criteria

- `go run ./cmd/server` starts WebSocket server
- Can connect via `wscat`, send navigate command, receive screenshot
- Browser visible in headed mode, closes on disconnect

---

## Phase 2: Raw Data Mode & Element Interaction

**Goal:** Return raw accessibility data; Mix agent handles filtering and overlays.

### Deliverables

1. **Raw screenshot mode** - return PNG + accessibility tree + viewport in one call
2. **Element interaction** - click, type, scroll by index
3. **No overlay processing** - browser-service is data provider only
4. **Mix processes locally** - vision package filters elements, renders overlays

### Commands

**Screenshot with Raw Data:**
```json
{
  "method": "Page.screenshot",
  "params": {"raw": true}
}
```

**Response:**
```json
{
  "data": "base64...",
  "rawNodes": [
    {"role": "button", "name": "Submit", "bounds": {...}, "backendId": 123}
  ],
  "rawViewport": {"x": 0, "y": 0, "width": 1920, "height": 1080}
}
```

**Element Interaction:**
```json
{"method": "Page.click", "params": {"index": 0}}
{"method": "Page.type", "params": {"index": 1, "text": "hello"}}
{"method": "Page.scroll", "params": {"direction": "down", "amount": 500}}
```

### Screenshot Pipeline

**Browser-Service (Raw Mode):**
```
1. Capture screenshot via CDP
2. Query accessibility tree (CDP Accessibility.getFullAXTree)
3. Return: PNG + raw nodes + viewport (single response)
```

**Mix Agent (Vision Package):**
```
4. Filter interactive nodes (buttons, links, inputs)
5. Render numbered overlays with image/draw
6. Save annotated screenshot
```

### Tests (Must Pass)

```
✓ TestScreenshotWithRawMode     - Returns PNG + rawNodes + rawViewport
✓ TestClickByIndex              - Click element by index
✓ TestTypeByIndex               - Type into input by index
✓ TestScrollPage                - Scroll down, verify viewport changed
```

### Success Criteria

- Browser returns raw data in single call (not two)
- Mix agent renders overlays locally
- Click by index works correctly
- 84 browser-service tests passing

---

## Phase 3: Mix Agent Integration

**Goal:** Browser tool in mix agent that communicates with browser service.

### Deliverables

1. **WebSocket client** in mix agent (`internal/llm/tools/browser/`)
2. **BrowserTool** implementing `BaseTool` interface
3. **Session-to-thread mapping** - each mix session gets unique browser thread
4. **Tool actions:** `open`, `screenshot`, `click`, `type`, `scroll`, `close`
5. **Screenshot storage** in session assets directory

### Architecture

**Mix Vision Package:**
- `element_filter.go` - Filter interactive nodes
- `overlay.go` - Render numbered bounding boxes
- `types.go` - Element, ViewportBounds types

### Tests (Must Pass)

**Mix Vision Package:**
```
✓ TestFilterInteractiveElements - Filters only interactive roles
✓ TestOverlayBoundingBoxes      - Renders numbered boxes on PNG
✓ TestElementIndexConsistency   - Sequential indexing (0, 1, 2...)
```

**Mix Integration:**
```
✓ TestBrowserToolIntegration    - Full workflow with raw mode
✓ TestSessionIsolation          - Independent browser contexts
```

---

## Phase 4: Advanced Features

**Goal:** Production-ready with cookie import, anti-detection, error recovery.

### Deliverables

1. **Chrome cookie import (macOS)**
   - Read `~/Library/Application Support/Google/Chrome/Default/Cookies`
   - Decrypt via Keychain (`security find-generic-password`)
   - PBKDF2 + AES-128-CBC decryption
   - Import into go-rod browser context

2. **Anti-detection**
   - Firefox user agent spoofing
   - `navigator.webdriver` override
   - Dialog suppression (alert/confirm/prompt)

3. **Error recovery**
   - WebSocket reconnection (5 attempts, exponential backoff)
   - Navigation timeout handling
   - Page crash recovery

4. **New commands:** `importCookies`, `setUserAgent`

### Cookie Import Flow

```
1. Check Chrome not running (or warn user)
2. Read encrypted cookies from SQLite
3. Get encryption key: security find-generic-password -w -s "Chrome Safe Storage"
4. Derive key: PBKDF2(password, "saltysalt", 1003, SHA1) → 16 bytes
5. Decrypt each cookie: AES-128-CBC, IV = 16 spaces
6. Import via go-rod: browser.SetCookies()
```

### Cookie Import Command

**Request:**
```json
{
  "method": "Browser.importCookies",
  "params": {
    "source": "chrome",
    "profile": "Default"
  }
}
```

**Response:**
```json
{
  "result": {
    "imported": 145,
    "failed": 2
  }
}
```

### Anti-Detection Commands

**Set User Agent:**
```json
{
  "method": "Browser.setUserAgent",
  "params": {
    "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
  }
}
```

**Suppress Dialogs:**
```json
{
  "method": "Browser.suppressDialogs",
  "params": {
    "enabled": true
  }
}
```

### Tests (Must Pass)

```
✓ TestChromeKeyRetrieval        - Gets key from macOS Keychain
✓ TestCookieDecryption          - Decrypts test cookie correctly
✓ TestCookieImport              - Imported cookies work (test with httpbin)
✓ TestUserAgentSpoof            - navigator.userAgent returns Firefox
✓ TestWebdriverHidden           - navigator.webdriver is undefined
✓ TestDialogSuppression         - alert() doesn't block execution
✓ TestReconnection              - Recovers from dropped WebSocket
✓ TestNavigationTimeout         - Graceful handling of hung pages
✓ TestPageCrashRecovery         - Recovers from chrome://crash
```

### Success Criteria

- Import Chrome cookies, browse authenticated sites
- Pass basic bot detection (navigator checks)
- Graceful degradation on network issues

---

## Repository Structure (Final)

```
browser-service/
├── cmd/server/main.go
├── internal/
│   ├── server/ws.go              # WebSocket handler
│   ├── browser/
│   │   ├── manager.go            # Lifecycle
│   │   ├── context.go            # Raw AX tree extraction
│   │   └── cookies.go            # Chrome import (macOS)
│   └── antidetect/
│       ├── useragent.go
│       └── stealth.go
├── pkg/
│   ├── client/client.go
│   └── protocol/messages.go      # Raw mode types
├── test/e2e_test.go
└── Makefile
```

---

## Dependencies

### browser-service

```go
require (
    github.com/go-rod/rod v0.114.0
    github.com/gorilla/websocket v1.5.0
    github.com/mattn/go-sqlite3 v1.14.18
)
```

### mix agent

```go
require (
    github.com/gorilla/websocket v1.5.0
    golang.org/x/image v0.15.0  // Vision package overlays
)
```

---

## Test Execution

### Phase 1
```bash
cd browser-service
go test ./... -run "TestWebSocket|TestBrowser|TestNavigate|TestScreenshot|TestContext|TestCleanup"
```

### Phase 2
```bash
# Browser-service raw mode
go test ./... -run "TestScreenshotWithRawMode|TestClick|TestType"
```

### Phase 3
```bash
# Mix vision package
go test ./mix_agent/internal/llm/tools/browser/vision/... -v
# Mix integration
go test ./mix_agent/internal/llm/tools/browser/... -v
```

### Phase 4
```bash
cd browser-service
go test ./... -run "TestChrome|TestCookie|TestUserAgent|TestWebdriver|TestDialog|TestReconnect|TestTimeout|TestCrash"
```

---

## Development Workflow

### Phase Completion Checklist

Each phase must complete before moving to the next:

1. ✅ All tests passing
2. ✅ Success criteria met
3. ✅ Code review completed
4. ✅ Documentation updated
5. ✅ Integration verified

### Running the Service

**Development Mode (Headed Browser):**
```bash
go run ./cmd/server --headless=false --port=8081
```

**Production Mode:**
```bash
go run ./cmd/server --headless=true --port=8081
```

### Testing with wscat

```bash
# Install wscat
npm install -g wscat

# Connect to service
wscat -c ws://localhost:8081

# Send navigate command
{"id":"1","method":"Page.navigate","params":{"url":"https://example.com"}}

# Request screenshot
{"id":"2","method":"Page.screenshot","params":{}}
```

---

## Integration with Mix Agent

**Configuration:**
- Endpoint: `ws://localhost:8081`
- Vision package: Filters interactive elements, renders overlays
- Session mapping: Each session gets unique browser context

**Actions:** `open`, `screenshot`, `click`, `type`, `scroll`, `close`

---

## Performance Targets

- **WebSocket Connection:** < 100ms
- **Browser Launch:** < 2s (headed), < 1s (headless)
- **Page Navigation:** < 5s (network dependent)
- **Screenshot Capture:** < 500ms
- **Element Click:** < 100ms
- **Cookie Import:** < 1s (100 cookies)

---

## Error Handling

### Common Errors

1. **Navigation Timeout:** Page takes too long to load
   - Return partial screenshot + error
   - Allow retry with increased timeout

2. **Element Not Found:** Index doesn't exist
   - Return current element list
   - Suggest re-taking screenshot

3. **WebSocket Disconnect:** Connection lost
   - Auto-reconnect with exponential backoff
   - Queue commands during reconnection

4. **Browser Crash:** Chrome process died
   - Restart browser automatically
   - Return error to client

5. **Cookie Decryption Failed:** Wrong key or format
   - Log specific error
   - Continue without cookies

---

## Security Considerations

1. **Command Validation:** Sanitize all inputs (URLs, text, indices)
2. **Resource Limits:** Max 10 concurrent browser contexts per service
3. **Timeout Enforcement:** Hard timeout of 60s per command
4. **Cookie Encryption:** Never log decrypted cookie values
5. **Network Isolation:** Option to disable network access (local testing)

---

## Future Enhancements (Post-MVP)

- Support for multiple browser engines (Firefox, Safari)
- Video recording of browser sessions
- Network request interception/modification
- Multi-tab support within single session
- Proxy configuration
- Custom CDP command passthrough
- Distributed browser pool (Kubernetes)

---

## Estimated Effort

- **Phase 1:** 2-3 days
- **Phase 2:** 3-4 days
- **Phase 3:** 2-3 days
- **Phase 4:** 3-4 days

**Total:** 10-14 days for full implementation

---

**Document Created:** 2026-02-06
**Author:** Implementation plan for go-rod browser automation service
**Version:** 1.0

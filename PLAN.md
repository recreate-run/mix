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

## Phase 2: Vision System & Element Interaction

**Goal:** Screenshot overlays with numbered bounding boxes, element interaction by index.

### Deliverables

1. **Accessibility tree extraction** via CDP `Accessibility.getFullAXTree`
2. **Interactive element filtering** (buttons, links, inputs, etc.)
3. **Bounding box overlay** on screenshots using `image/draw`
4. **Element indexing** - numbered labels `[0]`, `[1]`, etc.
5. **New commands:** `getElements`, `click`, `type`, `scroll`

### New Commands

**Get Elements:**
```json
{
  "method": "Page.getElements"
}
```

**Response:**
```json
{
  "elements": [
    {
      "index": 0,
      "role": "button",
      "name": "Submit",
      "bounds": {"x": 100, "y": 200, "width": 80, "height": 40}
    }
  ]
}
```

**Click Element:**
```json
{
  "method": "Page.click",
  "params": {"index": 0}
}
```

**Type Text:**
```json
{
  "method": "Page.type",
  "params": {"index": 1, "text": "hello"}
}
```

**Scroll Page:**
```json
{
  "method": "Page.scroll",
  "params": {"direction": "down", "amount": 500}
}
```

### Annotated Screenshot Pipeline

```
1. Capture raw screenshot
2. Query accessibility tree
3. Filter interactive nodes (role: button, link, textbox, etc.)
4. Calculate screen coordinates from AX bounds
5. Draw numbered boxes using Go image library
6. Return annotated PNG + element list
```

### Tests (Must Pass)

```
✓ TestAccessibilityTree         - Extract AX tree from test page
✓ TestElementFiltering          - Filter returns only interactive elements
✓ TestBoundingBoxOverlay        - Screenshot has visible numbered boxes
✓ TestClickByIndex              - Click element [0], verify action occurred
✓ TestTypeByIndex               - Type into input [1], verify value
✓ TestScrollPage                - Scroll down, verify viewport changed
✓ TestElementIndexConsistency   - Same page returns same indices
```

### Success Criteria

- Screenshot shows `[0] Button`, `[1] Input` boxes
- LLM can say "click element 0" and it works
- Element list JSON matches visual overlay

---

## Phase 3: Mix Agent Integration

**Goal:** Browser tool in mix agent that communicates with browser service.

### Deliverables

1. **WebSocket client** in mix agent (`internal/llm/tools/browser/`)
2. **BrowserTool** implementing `BaseTool` interface
3. **Session-to-thread mapping** - each mix session gets unique browser thread
4. **Tool actions:** `open`, `screenshot`, `click`, `type`, `scroll`, `close`
5. **Screenshot storage** in session assets directory

### Tool Interface

```go
// internal/llm/tools/browser/browser.go
type BrowserTool struct {
    wsEndpoint string
    clients    map[string]*websocket.Conn  // sessionID -> connection
}

func (b *BrowserTool) Info() ToolInfo {
    return ToolInfo{
        Name: "browser",
        Description: "...",  // Load from .md file
        Parameters: map[string]any{
            "action": {
                "type": "string",
                "enum": ["open", "screenshot", "click", "type", "scroll", "close"]
            },
            "url": {"type": "string"},
            "index": {"type": "integer"},
            "text": {"type": "string"},
        },
    }
}

func (b *BrowserTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
    // 1. Extract sessionID from context
    // 2. Get or create WebSocket connection
    // 3. Send command to browser service
    // 4. Wait for response
    // 5. Save screenshot if applicable
    // 6. Return ToolResponse
}
```

### Tool Response Format

**Screenshot Action:**
```json
{
  "type": "image",
  "path": "/sessions/{id}/assets/screenshot_001.png",
  "elements": [
    {"index": 0, "name": "Submit"},
    {"index": 1, "name": "Email input"}
  ]
}
```

**Click/Type Action:**
```json
{
  "type": "text",
  "content": "Clicked element [0]: Submit button"
}
```

### Tests (Must Pass)

```
✓ TestBrowserToolRegistration   - Tool appears in CoderAgentTools()
✓ TestOpenAction                - "open https://example.com" launches browser
✓ TestScreenshotAction          - Returns valid image path in session storage
✓ TestClickAction               - Click by index succeeds
✓ TestTypeAction                - Type text into element
✓ TestCloseAction               - Browser closes, connection cleaned up
✓ TestSessionIsolation          - Two sessions have independent browsers
✓ TestAutoCleanup               - Browser closes when session ends
✓ TestE2EAgentFlow              - Full flow: open → screenshot → click → screenshot
```

### Success Criteria

- Agent can: "Open google.com, take a screenshot, click the search box, type 'hello', take another screenshot"
- Screenshots saved to session assets, viewable in mix UI
- Browser closes automatically when task completes

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
browser-service/                    # New repo
├── cmd/server/main.go
├── internal/
│   ├── server/
│   │   ├── ws.go                  # WebSocket handler
│   │   └── protocol.go            # Message types
│   ├── browser/
│   │   ├── manager.go             # Lifecycle management
│   │   ├── vision.go              # AX tree + overlays
│   │   └── cookies.go             # Chrome import (macOS)
│   └── antidetect/
│       ├── useragent.go
│       └── stealth.go
├── pkg/client/                     # Go client library
│   └── client.go
├── test/
│   ├── e2e_test.go
│   └── testdata/
├── docs/
│   └── IMPLEMENTATION_PLAN.md     # This file
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Dependencies

### browser-service

```go
require (
    github.com/go-rod/rod v0.114.0
    github.com/gorilla/websocket v1.5.0
    golang.org/x/image v0.15.0
    github.com/mattn/go-sqlite3 v1.14.18
    github.com/stretchr/testify v1.8.4
)
```

### mix agent additions

```go
require (
    github.com/gorilla/websocket v1.5.0
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
go test ./... -run "TestAccessibility|TestElement|TestBoundingBox|TestClick|TestType|TestScroll"
```

### Phase 3 (in mix agent)
```bash
cd mix_agent
go test ./internal/llm/tools/browser/...
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

### Configuration

Add to mix agent config:
```yaml
browser:
  enabled: true
  endpoint: "ws://localhost:8081"
  timeout: 30000
  headless: false
```

### Tool Registration

```go
// internal/llm/agent/tools.go
func CoderAgentTools(...) []tools.BaseTool {
    return []tools.BaseTool{
        // existing tools...
        tools.NewBrowserTool(permissions, config.Browser.Endpoint),
    }
}
```

### Tool Description

Create `internal/config/prompts/tools/browser.md`:
```markdown
# Browser Tool

Automate web browsing tasks using a headless browser with vision capabilities.

## Actions

- **open**: Navigate to a URL
- **screenshot**: Capture current page with numbered element overlays
- **click**: Click an element by index
- **type**: Type text into an input element by index
- **scroll**: Scroll the page
- **close**: Close the browser session

## Examples

Open a website and take a screenshot:
{"action": "open", "url": "https://google.com"}
{"action": "screenshot"}

Click the search box and type:
{"action": "click", "index": 0}
{"action": "type", "index": 0, "text": "hello world"}
```

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

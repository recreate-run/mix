# Browser Providers

Mix supports three browser provider modes. Each session specifies its browser mode when created.

## 1. Electron Embedded Browser

**Mode**: `"electron-embedded-browser"` | **Status**: ✅ Production

```
┌──────────────┐                        ┌──────────────┐
│  Mix Agent   │ ─────WebSocket────────▶│ Electron App │
│ (Go Backend) │                        │  (Chromium)  │
└──────────────┘                        └──────────────┘
```

Communicates with Electron desktop app's embedded Chromium browser via WebSocket tunnels. Use for desktop applications requiring embedded browser control.

## 2. Local Browser Service (Default)

**Mode**: `"local-browser-service"` | **Status**: ✅ Production

```
┌──────────────┐                        ┌──────────────────┐
│  Mix Agent   │ ─────WebSocket────────▶│ Browser Service  │
│ (Go Backend) │                        │   (GoRod+CDP)    │
└──────────────┘                        └──────────────────┘
```

Connects to external browser-service process via WebSocket. Requires `BROWSER_SERVICE_URL` env var. Default mode for local development and testing.

## 3. Remote CDP WebSocket

**Mode**: `"remote-cdp-websocket"` | **Status**: ⏳ Planned

```
┌──────────────┐                        ┌──────────────────┐
│  Mix Agent   │ ──CDP WebSocket───────▶│  Cloud Browser   │
│ (Go Backend) │                        │ (BrowserStack...)│
└──────────────┘                        └──────────────────┘
```

Direct CDP connection to cloud browser providers. Requires `CDPURL` env var. Not yet implemented.

## Configuration

```bash
# Browser service URL (required for local-browser-service mode)
# Use HTTP URL - automatically converted to WebSocket URL internally
export BROWSER_SERVICE_URL="http://localhost:8091"
```

Sessions specify their browser mode when created via API or CLI `--browser-mode` flag (defaults to `local-browser-service`).

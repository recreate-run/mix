# Browser Providers

Mix supports three browser provider modes. Set via `BROWSER_MODE` env var or per-session override.

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
# Global default
export BROWSER_MODE="local-browser-service"

# Local service requires URL
export BROWSER_SERVICE_URL="ws://localhost:9222"
```

Per-session override available via `Session.BrowserMode` field.

# Browser Service

Go-based browser automation service using go-rod with WebSocket API for distributed LLM agent control.

## Quick Start

```bash
# Install dependencies
task install

# Start development server with hot reload
task dev

# In another terminal, test the WebSocket connection
task ws-test
```

## Documentation

- [CLAUDE.md](./CLAUDE.md) - Development guide and conventions
- [docs/IMPLEMENTATION_PLAN.md](./docs/IMPLEMENTATION_PLAN.md) - Phased implementation plan

## Architecture

WebSocket-based browser automation service that provides:

- **Isolated browser contexts** per WebSocket connection
- **Vision-based automation** with screenshot overlays and numbered elements
- **Chrome cookie import** for authenticated browsing (macOS)
- **Anti-detection** features (user agent spoofing, dialog suppression)

```
┌─────────────────────┐     WebSocket      ┌─────────────────────┐
│     Mix Agent       │◄──────────────────►│  go-rod Browser     │
│  (LLM + Tools)      │   CDP Commands     │     Service         │
│                     │   Screenshots      │                     │
└─────────────────────┘                    └─────────────────────┘
```

## Development

See [CLAUDE.md](./CLAUDE.md) for detailed development instructions.

Common commands:

```bash
task dev              # Start with hot reload
task dev-headless     # Start in headless mode (for E2E tests)
task build            # Build binary
task test             # Run all tests
task test-phase1      # Run Phase 1 tests only
task lint             # Run linters
task fmt              # Format code
```

### Headless Mode

Use headless mode for E2E testing or CI/CD environments:

```bash
task dev-headless                              # Headless with hot reload
task dev-with-args -- --headless --port 9000   # Custom arguments
```

**Note:** Headless mode is set at browser launch (server-level), not per-session. All WebSocket clients share the same browser instance.

## Testing

```bash
# Run all tests
task test

# Run phase-specific tests
task test-phase1  # WebSocket + Browser basics
task test-phase2  # Vision + Element interaction
task test-phase4  # Cookies + Anti-detection

# Manual WebSocket testing
task ws-test
```

## WebSocket Protocol

Connect to `ws://localhost:8081` and send JSON-RPC style messages:

```json
{
  "id": "req-1",
  "method": "Page.navigate",
  "params": {
    "url": "https://example.com"
  }
}
```

See [docs/IMPLEMENTATION_PLAN.md](./docs/IMPLEMENTATION_PLAN.md) for full protocol specification.

## Implementation Status

- [x] Project setup (Taskfile, hot reload, CLAUDE.md)
- [ ] Phase 1: Core WebSocket + Browser
- [ ] Phase 2: Vision System + Element Interaction
- [ ] Phase 3: Mix Agent Integration
- [ ] Phase 4: Advanced Features (Cookies, Anti-detection)

## License

MIT

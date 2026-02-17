# Research Notes

This document tracks research findings and architectural decisions for Mix, including approaches we've chosen not to pursue.

---

## JavaScript Injection in Browser Tool

**Date:** 2026-02-16
**Decision:** NOT implementing JavaScript injection capability
**Status:** Rejected

### Findings

Research into AI browser automation tools (Puppeteer MCP, Playwright MCP, agent-browser CLI) shows JavaScript execution is common but introduces severe security risks:

1. **Prompt Injection Attacks**: Malicious websites can embed hidden instructions that trick the LLM into executing harmful code (credential theft, session hijacking)
2. **Privilege Escalation**: Injected scripts run with full user privileges, accessing cookies, localStorage, and auth tokens
3. **Invisible Execution**: Unlike clicks (visually verifiable), scripts execute without observable feedback
4. **Irreversible Actions**: Can trigger destructive operations (data deletion, account closure) without safeguards

### Rationale

Our click/type approach prioritizes:
- **Observable actions** - Each operation visible in screenshots
- **Sandboxed execution** - No access to JavaScript context or credentials
- **User verification** - Critical actions require explicit approval
- **Security-first design** - Attack surface minimized for early-stage product

Industry trend shows tools with JS execution face "serious prompt injection and credential risks" (Browserless 2026 report). Anthropic's Computer Use deliberately avoids page-level JS injection, using screen-based automation instead.

**Conclusion:** Security benefits outweigh convenience gains. Current primitives sufficient for automation needs.

---

## Mocked Browser in Agent Integration Tests

**Date:** 2026-02-17
**Decision:** NOT implementing mocked-browser + real-LLM integration tests
**Status:** Rejected

### Context

When testing the browser agent (e.g. `read_page` with `filter: "links"`), a natural middle-ground test tier would mock the `BrowserClient` interface to return hardcoded accessibility nodes while making real LLM calls. This would be faster than full e2e tests and avoid needing a running browser service.

### Findings

We validated this approach against our existing test suite and found the trade-offs unfavourable:

1. **Still slow**: Real LLM API calls dominate latency regardless of browser mocking. A 3-turn agent interaction costs ~9–15 seconds even with instant mock responses — not fast enough to run frequently in CI.

2. **Still flaky**: LLM non-determinism means the agent may choose `find` over `read_page`, add extra navigation steps, or vary its phrasing. Assertions must be loose enough to tolerate this, reducing test value.

3. **Hides real failures**: The most valuable signal from our e2e tests came from a real browser issue — `Viewport: 0x0` causing 32 agent messages and a failed run. A mocked browser returns valid dimensions by definition, silently masking exactly the class of failures worth catching.

4. **Maintenance overhead**: Mocking `BrowserClient` requires faking `Navigate`, `ReadPage`, tab management, and viewport data. This mock layer must be kept in sync with the real interface as it evolves, adding friction with no additional coverage.

### Rationale

The existing two-tier strategy covers the meaningful cases:

- **E2e tests** (`mix_agent/e2e/`) use a real LLM, real browser service, and real agent loop — catching integration failures across the full stack.
- **Tool-level tests** (`browser_integration_test.go`) call the tool handler directly with a mock WebSocket server — fast, deterministic, no LLM cost.

A mocked-browser middle tier occupies the worst position: it pays LLM cost without real-world fidelity, and pays mock maintenance cost without the speed of direct tool tests.

**Conclusion:** The two existing test tiers provide sufficient coverage. Mocked-browser agent tests add complexity without meaningful gain.

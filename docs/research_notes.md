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

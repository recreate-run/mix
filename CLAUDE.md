# Claude Code Integration

## Development Commands

```bash
make dev          # Development (starts both frontend and backend.  This autoreloads and auto compiles.  Don't ever stop the server)
make build        # Production build (we rarely need this)
make clean        # Clean build artifacts (we rarely need this)
make tail-log     # Reads the current log file (last 100 lines of code)
```

* **Do NOT build the program yourself to check for errors—ever.** All output is written to `dev.log`. Run `make tail-log` to view it.
* **Do NOT stop the dev server.** It stays running, auto-compiles, and auto-reloads via the Go `air` package, logging to `dev.log`.
* **Run `make` from the project’s top-level directory.** If it fails, you probably weren’t there.
* You **MUST check the tail-log** after finishing each task

## Architecture

1. Backend - Go
2. Frontend - Tauri 2.0 app withh react

## Tech Stack

* **ALWAYS use TanStack Query for data fetching**

## Code style

1. As this is an early-stage startup, YOU MUST prioritize simple, readable code with minimal abstraction—avoid premature optimization. Strive for elegant, minimal solutions that reduce complexity.Focus on clear implementation that’s easy to understand and iterate on as the product evolves.
2. NEVER mock LLM API calls
3. DO NOT use preserve backward compatibility unless the user specifically requests it
4. Do not handle errors (eg. API failures) gracefully, raise exceptions immediately.


## Notes

This project uses `.mix/commands/` for custom slash commands (independent from Claude Code's `.claude/commands/` system).
Users can add `.md` files to `.mix/commands/` or `~/.mix/commands/` to create custom commands.
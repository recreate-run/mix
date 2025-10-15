# Claude Code Integration

## Development Commands

<bash_commands>
make dev          # start both frontend and backend (autoreloads and auto compiles)
make tail-log     # Reads the current log file (last 100 lines of code)
make clean
make help
make frontend-typecheck     # Always use this for frontend typechecking

</bash_commands>

- Do NOT build the program yourself to check for errors—ever. All output is written to `dev.log`. Run `make tail-log` to view it.
- Do NOT stop the dev server. It stays running, auto-compiles, and auto-reloads via the Go `air` package, logging to `dev.log`.
- Run `make` from the project's top-level directory. If it fails, you probably weren't there.
- You MUST check the tail-log after finishing each task
- ALWAYS update mix_agent/internal/http/rest_docs.go when modifying any backend API endpoints, request/response schemas, or validation rules
- Before writing database migrations, ALWAYS search for tool-specific patterns first (e.g., "goose sqlite foreign key") - don't write migrations based on general SQL knowledge alone

## Architecture

1. Backend - Golang
2. Frontend - React app with tanstack router

## Tech Stack

- ALWAYS use TanStack Query for data fetching
- ALWAYS use uv for Python package management and virtual environments
- Database query generation: Uses sqlc v1.29.0 (pinned in go.mod). Run `make sqlc-generate` after modifying SQL queries in `mix_agent/internal/db/sql/`

## Code style

1. As this is an early-stage startup, YOU MUST prioritize simple, readable code with minimal abstraction—avoid premature optimization. Strive for elegant, minimal solutions that reduce complexity.Focus on clear implementation that's easy to understand and iterate on as the product evolves.
2. NEVER mock LLM API calls
3. DO NOT preserve backward compatibility unless the user specifically requests it
4. Do not handle errors (eg. API failures) gracefully, raise exceptions immediately.

## Go Error Handling

Fail fast for business logic. Ignore non-critical operations.

Return errors immediately:

- API calls, database operations, file I/O
- User-facing operations
- Business logic failures

Use `_` to explicitly ignore:

- `defer file.Close()` - cleanup
- `os.Setenv/Unsetenv` - env operations
- `resp.Body.Close()` - HTTP cleanup
- `server.Shutdown()` - graceful shutdown
- `json.Encode()` after headers sent
- Analytics/telemetry tracking

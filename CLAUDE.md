# Claude Code Integration

## Development Commands

<bash_commands>
task dev          # start both frontend and backend (autoreloads and auto compiles)
task tail-log     # Reads the current log file (last 100 lines of code)
task clean
task              # Show all available tasks (default)
task frontend-typecheck     # Always use this for frontend typechecking

</bash_commands>

- Do NOT build the program yourself to check for errors—ever. All output is written to `dev.log`. Run `task tail-log` to view it.
- Do NOT stop the dev server. It stays running, auto-compiles, and auto-reloads via the Go `air` package, logging to `dev.log`.
- Run `task` from the project's top-level directory. If it fails, you probably weren't there.
- You MUST check the tail-log after finishing each task
- ALWAYS update mix_agent/internal/http/rest_docs.go when modifying any backend API endpoints, request/response schemas, or validation rules
- When adding new API fields/types, update the OpenAPI spec in rest_docs.go, then wait for the user to regenerate the SDK (version bump in package.json) - NEVER create local TypeScript type augmentations
- Before writing database migrations, ALWAYS search for tool-specific patterns first (e.g., "goose sqlite foreign key") - don't write migrations based on general SQL knowledge alone

## Architecture

1. Backend - Golang
2. Frontend - React app with tanstack router

## Tech Stack

- ALWAYS use TanStack Query for data fetching
- ALWAYS use uv for Python package management and virtual environments
- Database query generation: Uses sqlc v1.29.0 (pinned in go.mod). Run `task sqlc-generate` after modifying SQL queries in `mix_agent/internal/db/sql/`

## Code style

1. As this is an early-stage startup, YOU MUST prioritize simple, readable code with minimal abstraction—avoid premature optimization. Strive for elegant, minimal solutions that reduce complexity.Focus on clear implementation that's easy to understand and iterate on as the product evolves.
2. DO NOT preserve backward compatibility unless the user specifically requests it
3. Do not handle errors (eg. API failures) gracefully, raise exceptions immediately.
4. ALWAYS extract repeated strings (3+ occurrences) into named constants; use stdlib constants (`http.Method*`, `http.Status*`) instead of string literals
5. ALWAYS add `t.Helper()` as the first line in any test helper function that takes `*testing.T` as a parameter

## Go specific rules

- Context Everywhere: All async operations take `context.Context` as first parameter for cancellation and timeouts.
- Error Handling: Always handle errors explicitly. Use `fmt.Errorf("...: %w", err)` for error wrapping.
- Typed Structures: Never use `map[string]interface{}` for structured data with known fields. Define typed structs in appropriate packages (protocol, browser, internal).
- Structured Errors: Use custom error types from `internal/errors` package. Never return errors as maps or parse error strings.
- Named Constants: Define constants in `internal/constants` for magic strings (timeouts, buffer sizes) and magic numbers. Never hardcode these values.


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

- ALWAYS use context-aware functions (`http.NewRequestWithContext`, `exec.CommandContext`, `db.QueryRowContext`) - NEVER use non-context versions
- NEVER return `(nil, nil)` - use sentinel errors like `var ErrNotFound = errors.New("not found")` for "not found" cases
- ALWAYS use `%w` in `fmt.Errorf("msg: %w", err)`, `errors.Is(err, target)` for comparisons, and `errors.As(err, &target)` for type assertions - NEVER use `%v`, `==`, or direct type assertions on errors

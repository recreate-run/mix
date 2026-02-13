# AGENTS.md Rules Compliance Audit Report

Date: February 13, 2026  
Repository: `mix`  
Scope: Static scan of `mix_agent`, `mix_dev_tool`, and selected `docs` files, with emphasis on enforceable AGENTS.md rules.  
Method: Pattern-based repo scan plus manual spot checks of representative hits. No code changes were made.

## Audit Summary

This audit found several concrete rule violations, primarily in backend Go error handling patterns and broad use of untyped maps. The most impactful issues are:

1. Error-string parsing (`strings.Contains(err.Error(), ...)`) instead of structured error handling.
2. Direct non-`nil` error comparisons (`err != someError`) instead of `errors.Is`.
3. Widespread use of `map[string]interface{}` / `map[string]any` where AGENTS.md requires typed structs for known fields.
4. Cleanup calls using `defer x.Close()` without explicit `_ =` ignore where AGENTS.md explicitly asks for `_` on cleanup/env operations.

Some important checks passed:

1. No `http.NewRequest(...)` usage was found.
2. No `exec.Command(...)` usage was found.
3. No violations were found for the `t.Helper()` first-line rule in test helpers scanned.
4. No clear `%v`/`%s` wrapping of `err` in `fmt.Errorf` (without `%w`) was detected in non-test internal Go code.

## Findings by Rule

### 1. Structured Errors: Error-string parsing used

AGENTS.md says: use structured errors and avoid parsing error strings.

Violations found in non-test internal code (9 occurrences total of `strings.Contains(err.Error(), ...)`). Representative examples:

- `mix_agent/internal/http/rest_sessions.go:367`
- `mix_agent/internal/http/rest_sessions.go:563`
- `mix_agent/internal/llm/provider/openai.go:302`
- `mix_agent/internal/llm/provider/openai.go:308`
- `mix_agent/internal/llm/provider/openai.go:449`
- `mix_agent/internal/llm/provider/openai.go:457`
- `mix_agent/internal/llm/provider/openai.go:511`
- `mix_agent/internal/llm/provider/anthropic.go:426`
- `mix_agent/internal/llm/provider/anthropic.go:716`

Why this is a violation:

- Logic depends on text substrings in error messages, which is brittle and conflicts with AGENTS.md guidance to use structured/custom errors and `errors.Is`/`errors.As`.

### 2. Error comparisons: direct comparison against non-`nil` errors

AGENTS.md says: use `errors.Is(err, target)` for comparisons, never direct `==`/`!=` comparisons on errors.

Violations found:

- `mix_agent/internal/http/server.go:249` (`err != http.ErrServerClosed`)
- `mix_agent/internal/llm/provider/openai_oauth.go:182` (`err != http.ErrServerClosed`)

Why this is a violation:

- These should use `!errors.Is(err, http.ErrServerClosed)` for robust wrapped-error behavior.

### 3. Typed structures rule: heavy use of untyped maps

AGENTS.md says: do not use `map[string]interface{}` for known structured data; define typed structs.

Counts found:

- Non-test internal occurrences (excluding `rest_docs.go`): **163**
- E2E occurrences: **89**

Representative production/internal examples:

- `mix_agent/internal/http/server.go:84`
- `mix_agent/internal/http/server.go:114`
- `mix_agent/internal/http/rest_preferences.go:71`
- `mix_agent/internal/http/rest_preferences.go:98`
- `mix_agent/internal/http/rest_preferences.go:145`
- `mix_agent/internal/http/rest_tools.go:141`
- `mix_agent/internal/http/rest_system.go:107`
- `mix_agent/internal/browser/tunnel/client.go:69`
- `mix_agent/internal/llm/provider/gemini.go:594`
- `mix_agent/internal/llm/provider/gemini.go:600`

Important context:

- Some `map[string]any` usage in tool-schema definitions may be intentional due to dynamic JSON schema construction.
- `mix_agent/internal/http/rest_docs.go` contains extensive map-based OpenAPI construction and was excluded from the 163 count, since it is schema-document assembly by design.

### 4. Cleanup/env explicit ignore rule not consistently followed

AGENTS.md says to explicitly ignore specific non-critical operations with `_`, including cleanup (`defer file.Close()`), `os.Setenv/Unsetenv`, etc.

#### 4a. `defer ...Close()` without explicit `_ =`

Non-test internal hits: **13**

Examples:

- `mix_agent/internal/commands/builtin.go:351`
- `mix_agent/internal/http/rest_system.go:140`
- `mix_agent/internal/llm/tools/browser/browser_analyze.go:380`
- `mix_agent/internal/db/oauth_credentials.sql.go:149`
- `mix_agent/internal/db/oauth_credentials.sql.go:194`
- `mix_agent/internal/db/files.sql.go:127`
- `mix_agent/internal/db/messages.sql.go:127`

Notes:

- Several hits are sqlc-generated files under `mix_agent/internal/db/*.sql.go`.

#### 4b. `os.Setenv/Unsetenv` handled via `if err := ...` instead of `_ =`

Hits: **9** (mostly tests)

Examples:

- `mix_agent/internal/config/config_test.go:255`
- `mix_agent/internal/config/config_test.go:260`
- `mix_agent/internal/config/config_test.go:267`
- `mix_agent/internal/config/config_test.go:278`
- `mix_agent/internal/config/config_test.go:305`
- `mix_agent/internal/commands/registry_test.go:212`

Why this is a violation:

- AGENTS.md explicitly asks to use `_` for these env operations.

### 5. HTTP method literals instead of stdlib constants

AGENTS.md says to use stdlib constants (e.g., `http.Method*`) instead of hardcoded strings.

Detected examples:

- `mix_agent/internal/http/rest_common_test.go:477` (`"GET"`)
- `mix_agent/internal/http/rest_common_test.go:483` (`"POST"`)

Scope note:

- These are test-only instances in the scan output.

### 6. Frontend data-fetching rule (“always use TanStack Query”)

AGENTS.md says to always use TanStack Query for data fetching.

Observed direct `fetch()` in frontend/docs:

- `mix_dev_tool/src/hooks/useCsv.ts:73`
- `mix_dev_tool/src/hooks/use-media-download.ts:52`
- `mix_dev_tool/src/vite-console-forward-plugin.ts:145`
- `docs/components/copy-markdown.tsx:15`

Assessment:

- `mix_dev_tool/src/hooks/useCsv.ts` is used inside a `useQuery` query function and aligns with TanStack Query usage.
- `mix_dev_tool/src/hooks/use-media-download.ts` performs download fetch in a callback and does not use TanStack Query; this is the strongest likely violation in app code.
- `mix_dev_tool/src/vite-console-forward-plugin.ts` is Vite plugin/runtime forwarding logic (infrastructure path, not typical app state data fetching).
- `docs/components/copy-markdown.tsx` is docs-site UI behavior; applicability depends on whether AGENTS rules are intended to govern docs app equally.

## Rules Checked With No Violations Found

1. Context-aware request creation (`http.NewRequestWithContext` vs `http.NewRequest`): no `http.NewRequest(` found.
2. Context-aware command execution (`exec.CommandContext` vs `exec.Command`): no `exec.Command(` found.
3. Test helper rule (`t.Helper()` as first line): no violations found in scanned helper functions.
4. Error wrapping `%w`: no clear non-test `fmt.Errorf(...err...)` cases without `%w` were detected.
5. `return (nil, nil)` rule: no direct `return nil, nil` found. (Matches were `return nil, nil, fmt.Errorf(...)`, which are not violations of this specific rule.)

## Limitations

1. This is a static audit; it does not validate runtime behavior.
2. Some AGENTS.md rules are intentionally broad or subjective (for example, “simple readable code,” “minimal abstraction”), so this report focuses on objectively detectable patterns.
3. Counts include likely intentional dynamic-schema patterns in some LLM/tooling code; those are still listed because AGENTS.md is written as strict policy.

## Prioritization Recommendation

1. Fix structured error handling first (replace string parsing and direct non-`nil` comparisons with typed errors plus `errors.Is`/`errors.As`).
2. Address high-traffic production map usages by introducing typed response/request structs in HTTP handlers and provider glue code.
3. Normalize cleanup/env patterns to match explicit `_` conventions in AGENTS.md.
4. Decide policy scope for frontend docs/plugin fetch patterns, then enforce consistently.


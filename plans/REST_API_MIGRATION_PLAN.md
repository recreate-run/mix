# Mix REST API Migration Plan

## Overview

Replace the JSON-RPC `/rpc` endpoint with proper REST endpoints. Remove all backward compatibility and legacy JSON-RPC code to create a clean, minimal HTTP API.

## Current State Analysis - CRITICAL CORRECTION

- **Existing**: JSON-RPC `/rpc` endpoint handling all operations
- **Available**: Complete business logic in `api/query.go` - all 17 handlers fully implemented  
- **BLOCKER**: Global session state (`currentSessionID` in `App` struct) prevents stateless REST migration
- **Dependencies**: 11+ operations depend on server-side session state that must be removed first
- **Goal**: Remove global session state dependencies, then extract existing logic into REST wrappers

## Migration Strategy

### Stateless Design Principle

Every endpoint requires explicit session ID - no server-side "current session" state:

- **No `/sessions/current` endpoint** - always specify session ID
- **No `/sessions/{id}/select` endpoint** - no concept of "selecting" sessions
- **Every operation is explicit** - client specifies exactly which session to operate on

### Stateful Operations Requiring Removal/Modification

**Current Global State Dependencies** (`app.go:32`):

- `currentSessionID string` field in App struct
- `SetCurrentSession()`, `GetCurrentSession()`, `GetCurrentSessionID()` methods

**JSON-RPC Operations to DELETE entirely:**

- `sessions.current` (`query.go:174`) - Returns global current session
- `sessions.select` (`query.go:176`) - Sets global current session

**Operations to MODIFY (remove state dependency):**

- `messages.send` (`query.go:806`) - Currently calls `SetCurrentSession()` before processing
- `sessions.delete` (`query.go:1119-1120`) - Currently prevents deleting "current" session
- `sessions.create` (`query.go:527`) - Currently has optional `setCurrent` flag
- SSE streaming (`sse.go:102`) - Currently calls `SetCurrentSession()`
- Built-in commands (`commands/builtin.go:285,359`) - Currently depend on "current session" concept

**Integration Points Requiring Updates:**

- Asset server working directory: Use `WORKING_DIR` environment variable instead of global session state

### Implementation Approach

**Extract and Wrap**: Create REST wrappers around existing business logic

#### What Already Exists in `api/query.go`

- **`handleSessionsList()`** - Complete logic for listing sessions
- **`handleSessionsCreate()`** - Complete logic for creating sessions  
- **`handleMessagesSend()`** - Complete auth, validation, and agent processing
- **All 17 handlers fully implemented** with validation, error handling, and app integration

#### What to Build (17 REST wrappers)

Simple HTTP wrappers that extract path/query parameters and call existing handlers:

```go
func (h *RESTHandler) GetSession(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    // Extract existing logic from handleSessionsGet() 
    // Convert response to HTTP format
}
```

#### Implementation Steps

1. **Create REST wrappers** - New file `mix_agent/internal/http/rest.go` (~200 lines)
2. **Extract logic** - Move core logic from `handle*()` functions to shared functions  
3. **Remove JSON-RPC protocol** - Delete `/rpc` endpoint, keep business logic temporarily
4. **Wire up routes** - Register 17 REST endpoints in `server.go` using Go 1.22+ mux patterns

## Complete REST API Routes

### Sessions Management

```go
// Session CRUD operations - completely stateless
mux.HandleFunc("GET /api/sessions", restHandler.ListSessions)
mux.HandleFunc("POST /api/sessions", restHandler.CreateSession)
mux.HandleFunc("GET /api/sessions/{id}", restHandler.GetSession)
mux.HandleFunc("DELETE /api/sessions/{id}", restHandler.DeleteSession)
mux.HandleFunc("POST /api/sessions/{id}/fork", restHandler.ForkSession)
```

### Messages & Communication

```go
// Message operations - session ID required for all operations
mux.HandleFunc("POST /api/sessions/{id}/messages", restHandler.SendMessage)
mux.HandleFunc("GET /api/sessions/{id}/messages", restHandler.ListMessages)

// Agent control - session ID required
mux.HandleFunc("POST /api/sessions/{id}/cancel", restHandler.CancelAgent)

// Global message history (optional pagination via query params)
mux.HandleFunc("GET /api/messages/history", restHandler.GetMessageHistory)
```

### Tools & Extensions

```go
// MCP and tool management
mux.HandleFunc("GET /api/mcp", restHandler.ListMCPServers)
mux.HandleFunc("GET /api/commands", restHandler.ListCommands)
mux.HandleFunc("GET /api/commands/{name}", restHandler.GetCommand)
```

### Authentication & Permissions

```go
// Authentication
mux.HandleFunc("POST /api/auth/login", restHandler.AuthLogin)
mux.HandleFunc("POST /api/auth/apikey", restHandler.SetAPIKey)

// Permission management
mux.HandleFunc("POST /api/permissions/{id}/grant", restHandler.GrantPermission)
mux.HandleFunc("POST /api/permissions/{id}/deny", restHandler.DenyPermission)
```

### Media & Assets (Keep Existing)

```go
// Keep existing asset endpoints (already REST-compliant)
mux.HandleFunc("GET /api/file-types", HandleFileTypes(app))
mux.HandleFunc("POST /api/video/export-url", HandleURLVideoExport)
mux.HandleFunc("GET /api/gsap_animations", HandleGSAPAnimationsList(app))
mux.HandleFunc("GET /api/gsap_animations/{name}", HandleGSAPAnimationSchema(app))

// Asset serving (already working)
mux.HandleFunc("/input/", HandleInputAssets(app))  
mux.HandleFunc("/output/", HandleOutputAssets(app))
mux.HandleFunc("/gsap_animations/", HandleGSAPAnimationFiles(app))
```

### System & Health

```go
// System endpoints
mux.HandleFunc("GET /health", healthCheck)
mux.HandleFunc("GET /stream", streamHandler)
mux.HandleFunc("GET /stream/{path...}", streamSubHandler)
```

## HTTP Method & Status Code Standards

### Request Methods

- `GET` - Retrieve resources (idempotent)
- `POST` - Create resources or trigger actions
- `PUT` - Update/replace resources (idempotent)
- `DELETE` - Remove resources (idempotent)

### Response Status Codes

- `200 OK` - Successful retrieval
- `201 Created` - Successful creation
- `204 No Content` - Successful deletion
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `404 Not Found` - Resource not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server errors

## Implementation Benefits

### Developer Experience

- **Familiar Patterns**: Standard REST conventions
- **Self-Documenting**: URLs describe operations
- **Tool Support**: Works with curl, Postman, OpenAPI generators
- **HTTP Semantics**: Proper caching, idempotency, status codes

### Architecture Improvements

- **Minimal Complexity**: Remove JSON-RPC abstraction layer
- **Direct Routing**: Go 1.22+ mux patterns with path parameters
- **Type Safety**: Strong typing without JSON-RPC conversion
- **Performance**: No JSON marshaling overhead

## Complexity Assessment

**High complexity** - Architectural changes required BEFORE simple wrapping:

**Phase 1 Prerequisites** (blocking - must complete first):

- **Remove global state dependencies**: ~100 lines changed across 6 files
- **11+ operations currently depend on `currentSessionID`**: Requires careful refactoring
- **SSE streaming integration redesign**: Real-time features depend on session state  
- **Command system updates**: Built-in commands require session context removal
- **Race condition elimination**: Concurrent request safety improvements

**Phase 2-4 Implementation** (after prerequisites):

- **~200 lines of wrapper code**: HTTP parameter extraction and response conversion
- **Logic refactoring**: Extract core logic from existing handlers to shared functions
- **Response format conversion**: Transform existing responses to proper HTTP responses
- **Testing required**: All 17 endpoints need verification

**Critical insight**: Cannot create REST wrappers until stateful dependencies are removed first. This is NOT a "simple wrapper" migration - it requires fundamental architectural changes to eliminate server-side session state.

## Breaking Changes

- **Complete**: Removes `/rpc` endpoint entirely
- **No Backward Compatibility**: Clean break from JSON-RPC
- **Frontend Impact**: Will require complete frontend rewrite (ignoring per requirements)

## Files to Modify

1. **CREATE** `mix_agent/internal/http/rest.go` - REST wrappers around extracted business logic
2. **MODIFY** `mix_agent/internal/http/server.go` - Remove `/rpc`, add REST route registration
3. **REFACTOR** `mix_agent/internal/api/query.go` - Extract business logic to shared functions
4. **UPDATE** `mix_agent/README.md` - Replace JSON-RPC examples with REST examples

## Implementation Strategy - PREREQUISITES ADDED

**Phase 1: REMOVE GLOBAL SESSION STATE** (blocking prerequisite)

- Remove `currentSessionID` field from App struct (`app.go:32`)
- Delete `SetCurrentSession()`, `GetCurrentSession()`, `GetCurrentSessionID()` methods (`app.go:155-199`)
- Remove `sessions.current` and `sessions.select` JSON-RPC handlers entirely (`query.go:174,176`)
- Modify `messages.send` to remove `SetCurrentSession()` call (`query.go:806`)
- Update SSE streaming to remove global state dependency (`sse.go:102`)
- Update built-in commands to remove current session dependencies (`commands/builtin.go`)
- **Asset server**: Use `WORKING_DIR` environment variable instead of global session state calls

**Phase 2**: Extract business logic from JSON-RPC handlers to shared functions  
**Phase 3**: Create REST wrappers that call shared functions
**Phase 4**: Remove JSON-RPC protocol layer (`/rpc` endpoint and `QueryRequest`/`QueryResponse` types)

## Route Organization Principles

- **No String Matching**: Use Go 1.22+ mux patterns exclusively
- **Resource-Based**: Group by entity type (`/api/sessions`, `/api/messages`)
- **Action Suffixes**: Use sub-resources for actions (`/sessions/{id}/fork`)
- **Query Parameters**: For filtering and pagination (`?limit=50&offset=100`)
- **Path Parameters**: For resource identification (`{id}`, `{name}`)

## Summary

**Key Decision**: Complete statelessness - no "current session" concept anywhere in the API

**Implementation Reality**: This migration extracts and wraps existing business logic rather than building from scratch. All validation, authentication, and core functionality already exists in `api/query.go` and just needs HTTP wrappers to replace JSON-RPC protocol wrappers.

This creates a clean, modern REST API that follows HTTP standards while leveraging all existing, tested business logic.

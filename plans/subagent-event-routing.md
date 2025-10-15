# Subagent Event Routing Plan

## Goal
Route subagent events to parent main session using context-scoped routing with explicit `RouteTo` field.

## Core Principle
- `SessionID` = what the event is **about** (unchanged semantics)
- `RouteTo` = where to **send** the event (new explicit routing)

---

## Changes

### 1. Context Helpers
**File**: `mix_agent/internal/llm/agent/agent.go`
**Location**: Add at package level (after line 24, before line 26)

```go
type contextKey string
const sessionRoutingKey contextKey = "session_routing"

type sessionRouting struct {
    RouteTo  string // Where events should be sent
    Origin   string // Where events originated from
}

func withSessionRouting(ctx context.Context, routeTo, origin string) context.Context {
    return context.WithValue(ctx, sessionRoutingKey, sessionRouting{RouteTo: routeTo, Origin: origin})
}

func getSessionRouting(ctx context.Context) (routeTo, origin string) {
    if r, ok := ctx.Value(sessionRoutingKey).(sessionRouting); ok {
        return r.RouteTo, r.Origin
    }
    return "", ""
}
```

### 2. Agent Event Schema
**File**: `mix_agent/internal/llm/agent/agent.go`
**Location**: Line 60-78 (modify existing `AgentEvent` struct)

```go
type AgentEvent struct {
    Type    AgentEventType
    Message message.Message
    Error   error

    // Routing fields
    SessionID string // What this event is about (provenance/origin)
    RouteTo   string // NEW: Where to send this event (destination for SSE)

    // When summarizing
    Progress  string
    Done      bool

    // When thinking
    Thinking string

    // When streaming content
    Content string

    // When executing tools
    ToolCallID string
}
```

### 3. Populate Routing Context
**File**: `mix_agent/internal/llm/agent/agent.go`
**Location**: In `RunWithPlanMode()` method, insert AFTER line 299 (after `context.WithCancel`), BEFORE line 301 (`activeContexts.Store`)

```go
func (a *agent) RunWithPlanMode(ctx context.Context, sessionID string, content string, planMode bool, attachments ...message.Attachment) (<-chan AgentEvent, error) {
    // ... existing validation ...

    genCtx, cancel := context.WithCancel(ctx)  // Line 299

    // NEW: Set up routing context BEFORE storing cancel func
    sess, err := a.sessions.Get(genCtx, sessionID)
    if err != nil {
        cancel()
        return nil, err
    }

    // Route subagent events to parent, otherwise to self
    routeTo := sessionID
    if sess.ParentSessionID != "" {
        routeTo = sess.ParentSessionID
    }
    genCtx = withSessionRouting(genCtx, routeTo, sessionID)
    // END NEW CODE

    // Store cancel function for potential cancellation
    a.activeContexts.Store(sessionID, cancel)  // Line 301

    // ... rest continues unchanged ...
}
```

### 4. Fix Event Forwarding Filter (CRITICAL)
**File**: `mix_agent/internal/llm/agent/agent.go`
**Location**: Line 365 in `RunWithPlanMode()` method (inside subscription goroutine)

**Problem**: Without this fix, subagent events are filtered out before reaching SSE broadcast logic.

**Replace this line**:
```go
if (event.Payload.SessionID == sessionID || event.Payload.Message.SessionID == sessionID) && !event.Payload.Done {
```

**With**:
```go
// Forward events that originated from OR are routed to this session
if (event.Payload.SessionID == sessionID ||
    event.Payload.RouteTo == sessionID ||
    event.Payload.Message.SessionID == sessionID) && !event.Payload.Done {
```

**Why**: The original filter rejects events where `SessionID != sessionID`. Subagent events have `SessionID = "subagent-456"` but `RouteTo = "main-123"`. The new condition allows events routed to this session to pass through.

### 5. Auto-Populate RouteTo in Publish
**File**: `mix_agent/internal/llm/agent/agent.go`
**Location**: Add new method after `Update()` method (around line 844)

**Note**: This method shadows the embedded `pubsub.Broker[AgentEvent].Publish()` method, which is intentional. Go will call this version on `agent` instances, which then delegates to the broker.

```go
// Publish overrides the embedded Broker.Publish to auto-populate routing fields from context
func (a *agent) Publish(ctx context.Context, t pubsub.EventType, event AgentEvent) error {
    routeTo, origin := getSessionRouting(ctx)

    // Auto-populate from context if not already set
    if event.RouteTo == "" && routeTo != "" {
        event.RouteTo = routeTo
    }
    if event.SessionID == "" && origin != "" {
        event.SessionID = origin
    }

    return a.Broker.Publish(ctx, t, event)
}
```

### 6. SSE Routing Decision
**File**: `mix_agent/internal/http/rest_messages_broadcast.go`
**Location**: Modify `BroadcastAgentEventToSSE()` function at line 11

Add routing logic at the TOP of the function, then replace ALL occurrences of the `sessionID` parameter with `targetSessionID`:

```go
func BroadcastAgentEventToSSE(sessionID string, event agent.AgentEvent) {
    // NEW: Determine routing target from event
    targetSessionID := event.RouteTo
    if targetSessionID == "" {
        targetSessionID = sessionID // Fallback: use function parameter
    }
    // END NEW CODE

    switch event.Type {
    case agent.AgentEventTypeThinking:
        // CHANGED: sessionID → targetSessionID
        registry.BroadcastEvent(targetSessionID, "thinking", ThinkingEvent{
            Type: "thinking",
            Content: event.Thinking,
        })

    case agent.AgentEventTypeContentDelta:
        if event.Content != "" {
            // CHANGED: sessionID → targetSessionID
            registry.BroadcastEvent(targetSessionID, "content", ContentEvent{
                Type: "content",
                Content: event.Content,
            })
        }

    case agent.AgentEventTypeToolParameterDelta:
        // CHANGED: sessionID → targetSessionID
        registry.BroadcastEvent(targetSessionID, "tool_parameter_delta", ToolParameterDeltaEvent{
            Type:       "tool_parameter_delta",
            ToolCallID: event.ToolCallID,
            Input:      event.Content,
        })

    // ... continue for ALL remaining cases (11 total BroadcastEvent calls to update)
    // Replace sessionID → targetSessionID in every registry.BroadcastEvent() call
    }
}
```

**Critical**: You must replace `sessionID` with `targetSessionID` in **all 11 `registry.BroadcastEvent()` calls** throughout the switch statement (lines 15, 20, 25, 41, 48, 54, 86, 88, 92, 100, 111).

---

## Implementation Checklist

- [ ] Add context helper functions (`withSessionRouting`, `getSessionRouting`) at package level
- [ ] Add `RouteTo string` field to `AgentEvent` struct (line 60)
- [ ] Populate routing context in `RunWithPlanMode()` after line 299
- [ ] **Fix event forwarding filter at line 365 (CRITICAL - without this, events won't reach SSE)**
- [ ] Add `agent.Publish()` method that shadows embedded broker method
- [ ] Update `BroadcastAgentEventToSSE()` to route based on `event.RouteTo`
- [ ] Replace all 11 occurrences of `sessionID` → `targetSessionID` in broadcast calls
- [ ] Test: Subagent events appear in parent session SSE stream
- [ ] Test: Main session events still work (RouteTo == SessionID)

---

## Event Flow

### Without Fix (Broken)
```
Subagent publishes: SessionID="sub-456", RouteTo="main-123"
  → Main agent filter: SessionID == "main-123"? NO ❌
  → Event DROPPED before reaching SSE
```

### With Fix (Working)
```
Subagent publishes: SessionID="sub-456", RouteTo="main-123"
  → Main agent filter: RouteTo == "main-123"? YES ✅
  → Event forwarded to main agent's channel
  → BroadcastAgentEventToSSE uses RouteTo="main-123"
  → registry.BroadcastEvent("main-123", ...)
  → SSE clients connected to main-123 receive event
```

---

## Why This Works

| Aspect | Benefit |
|--------|---------|
| **Context-scoped** | Automatic cleanup, no memory leaks |
| **Explicit routing** | RouteTo field is self-documenting |
| **Semantic clarity** | SessionID = provenance, RouteTo = destination |
| **Zero global state** | No sync.Map, no lifecycle management |
| **Request-scoped** | Each RunWithPlanMode() execution isolated |
| **Two-layer routing** | Filter at subscription + route at broadcast = complete solution |

---

## Expected Impact

- **LOC**: ~55 lines (context helpers + routing logic + filter fix)
- **Breaking Changes**: None (additive only, backward compatible)
- **Critical Changes**: 2 (event filter + SSE broadcast)
- **Complexity**: Minimal - context is idiomatic Go

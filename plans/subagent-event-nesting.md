# Subagent Event Nesting Implementation Plan

## Goal

Display all subagent events nested under their spawning task tool in a ladder view on the frontend.

## Solution

Use context-based `ParentToolCallID` propagation to link subagent events to their parent tool without frontend state management.

---

## Backend Changes

### 1. Add ParentToolCallID to AgentEvent

**File**: `mix_agent/internal/llm/agent/agent.go:80`

```go
type AgentEvent struct {
    Type    AgentEventType
    Message message.Message
    Error   error

    // Routing fields
    SessionID        string // What this event is about (provenance/origin)
    RouteTo          string // Where to send this event (destination for SSE)
    ParentToolCallID string // NEW: Which tool call spawned this subagent session

    // ... rest unchanged
}
```

### 2. Add Context Helpers for Tool Tracking

**File**: `mix_agent/internal/llm/agent/agent.go` (modify existing context helpers around line 30)

```go
// Update existing sessionRouting struct to include ParentToolCallID
type sessionRouting struct {
    RouteTo          string
    Origin           string
    ParentToolCallID string  // NEW
}

// Add helper to wrap context with tool call ID for subagent event tracking
// Package-private since task-tool.go is in the same package
func withToolContext(ctx context.Context, toolCallID string) context.Context {
    routing := sessionRouting{ParentToolCallID: toolCallID}

    // Preserve existing routing fields
    if r, ok := ctx.Value(sessionRoutingKey).(sessionRouting); ok {
        routing.RouteTo = r.RouteTo
        routing.Origin = r.Origin
    }

    return context.WithValue(ctx, sessionRoutingKey, routing)
}

// Update existing getSessionRouting to return all three fields
func getSessionRouting(ctx context.Context) (routeTo, origin, parentToolCallID string) {
    if r, ok := ctx.Value(sessionRoutingKey).(sessionRouting); ok {
        return r.RouteTo, r.Origin, r.ParentToolCallID
    }
    return "", "", ""
}

// Update existing withSessionRouting to preserve parentToolCallID
func withSessionRouting(ctx context.Context, routeTo, origin string) context.Context {
    routing := sessionRouting{RouteTo: routeTo, Origin: origin}

    // Preserve existing parentToolCallID if present
    if r, ok := ctx.Value(sessionRoutingKey).(sessionRouting); ok {
        routing.ParentToolCallID = r.ParentToolCallID
    }

    return context.WithValue(ctx, sessionRoutingKey, routing)
}
```

### 3. Set Tool Context When Executing Task Tool

**File**: `mix_agent/internal/llm/agent/task-tool.go:104-105` (in `Run` method)

Wrap the context with the tool call ID before spawning the subagent:

```go
// Line 104 - ADD THIS (new line before agent.Run call)
toolCtx := withToolContext(ctx, call.ID)

// Line 105 - MODIFY THIS (change ctx to toolCtx)
done, err := agent.Run(toolCtx, subSession.ID, params.Prompt)
```

**Complete context:**

```go
subSession, err := b.sessions.Create(ctx, "Subagent: "+params.Description, "", "default", session.SessionTypeSubagent, session.SubagentType(params.SubagentType), sessionID)
if err != nil {
    return tools.ToolResponse{}, fmt.Errorf("error creating session: %s", err)
}

// NEW: Wrap context with tool call ID for event tracking
toolCtx := withToolContext(ctx, call.ID)

// MODIFIED: Use wrapped context
done, err := agent.Run(toolCtx, subSession.ID, params.Prompt)
if err != nil {
    return tools.ToolResponse{}, fmt.Errorf("error generating agent: %s", err)
}
```

**Note:** No import needed - `withToolContext` is package-private in the same `agent` package.

### 4. Auto-Populate ParentToolCallID in Publish

**File**: `mix_agent/internal/llm/agent/agent.go` (update existing `Publish` method around line 844)

```go
func (a *agent) Publish(ctx context.Context, t pubsub.EventType, event AgentEvent) error {
    routeTo, origin, parentToolCallID := getSessionRouting(ctx)  // Updated signature

    if event.RouteTo == "" && routeTo != "" {
        event.RouteTo = routeTo
    }
    if event.SessionID == "" && origin != "" {
        event.SessionID = origin
    }
    if event.ParentToolCallID == "" && parentToolCallID != "" {
        event.ParentToolCallID = parentToolCallID  // NEW
    }

    return a.Broker.Publish(ctx, t, event)
}
```

### 5. Add parentToolCallId to SSE Events

**File**: `mix_agent/internal/http/sse_events.go`

```go
type ThinkingEvent struct {
    Type             string `json:"type"`
    Content          string `json:"content"`
    ParentToolCallID string `json:"parentToolCallId,omitempty"`  // NEW
}

type ToolEvent struct {
    Type             string `json:"type"`
    Name             string `json:"name"`
    Input            string `json:"input"`
    ID               string `json:"id"`
    Status           string `json:"status"`
    ParentToolCallID string `json:"parentToolCallId,omitempty"`  // NEW
}

type ContentEvent struct {
    Type             string `json:"type"`
    Content          string `json:"content"`
    ParentToolCallID string `json:"parentToolCallId,omitempty"`  // NEW
}

type ToolParameterDeltaEvent struct {
    Type             string `json:"type"`
    ToolCallID       string `json:"toolCallId"`
    Input            string `json:"input"`
    ParentToolCallID string `json:"parentToolCallId,omitempty"`  // NEW
}

type ToolExecutionStartEvent struct {
    Type             string `json:"type"`
    ToolName         string `json:"toolName"`
    Progress         string `json:"progress"`
    ToolCallID       string `json:"toolCallId"`
    ParentToolCallID string `json:"parentToolCallId,omitempty"`  // NEW
}

type ToolExecutionCompleteEvent struct {
    Type             string `json:"type"`
    ToolName         string `json:"toolName"`
    Progress         string `json:"progress"`
    Success          bool   `json:"success"`
    ToolCallID       string `json:"toolCallId"`
    ParentToolCallID string `json:"parentToolCallId,omitempty"`  // NEW
}

type CompleteEvent struct {
    Type              string `json:"type"`
    Content           string `json:"content,omitempty"`
    MessageID         string `json:"messageId,omitempty"`
    Done              bool   `json:"done"`
    Reasoning         string `json:"reasoning,omitempty"`
    ReasoningDuration int64  `json:"reasoningDuration,omitempty"`
    ParentToolCallID  string `json:"parentToolCallId,omitempty"`  // NEW
}

type ErrorEvent struct {
    Error            string `json:"error"`
    Type             string `json:"type,omitempty"`
    RetryAfter       int    `json:"retryAfter,omitempty"`
    Attempt          int    `json:"attempt,omitempty"`
    MaxAttempts      int    `json:"maxAttempts,omitempty"`
    ParentToolCallID string `json:"parentToolCallId,omitempty"`  // NEW
}

type SummarizeEvent struct {
    Type             string `json:"type"`
    Progress         string `json:"progress"`
    Done             bool   `json:"done"`
    ParentToolCallID string `json:"parentToolCallId,omitempty"`  // NEW
}

type PermissionEvent struct {
    Type             string      `json:"type"`
    ID               string      `json:"id"`
    SessionID        string      `json:"sessionId"`
    ToolName         string      `json:"toolName"`
    Description      string      `json:"description"`
    Action           string      `json:"action"`
    Path             string      `json:"path"`
    Params           interface{} `json:"params"`
    ParentToolCallID string      `json:"parentToolCallId,omitempty"`  // NEW
}
```

### 6. Pass ParentToolCallID in Broadcast Events

**File**: `mix_agent/internal/http/rest_messages_broadcast.go:11`

```go
func BroadcastAgentEventToSSE(sessionID string, event agent.AgentEvent) {
    targetSessionID := event.RouteTo
    if targetSessionID == "" {
        targetSessionID = sessionID
    }
    parentToolCallID := event.ParentToolCallID  // NEW

    switch event.Type {
    case agent.AgentEventTypeThinking:
        registry.BroadcastEvent(targetSessionID, "thinking", ThinkingEvent{
            Type:             "thinking",
            Content:          event.Thinking,
            ParentToolCallID: parentToolCallID,  // NEW
        })

    case agent.AgentEventTypeToolParameterDelta:
        registry.BroadcastEvent(targetSessionID, "tool_parameter_delta", ToolParameterDeltaEvent{
            Type:             "tool_parameter_delta",
            ToolCallID:       event.ToolCallID,
            Input:            event.Content,
            ParentToolCallID: parentToolCallID,  // NEW
        })

    case agent.AgentEventTypeToolExecutionStart:
        toolName := extractToolNameFromProgress(event.Progress)
        registry.BroadcastEvent(targetSessionID, "tool_execution_start", ToolExecutionStartEvent{
            Type:             "tool_execution_start",
            ToolName:         toolName,
            Progress:         event.Progress,
            ToolCallID:       event.ToolCallID,
            ParentToolCallID: parentToolCallID,  // NEW
        })

    case agent.AgentEventTypeToolExecutionComplete:
        toolName := extractToolNameFromProgress(event.Progress)
        success := !strings.Contains(strings.ToLower(event.Progress), "error")
        registry.BroadcastEvent(targetSessionID, "tool_execution_complete", ToolExecutionCompleteEvent{
            Type:             "tool_execution_complete",
            ToolName:         toolName,
            Progress:         event.Progress,
            Success:          success,
            ToolCallID:       event.ToolCallID,
            ParentToolCallID: parentToolCallID,  // NEW
        })

    case agent.AgentEventTypeContentDelta:
        if event.Content != "" {
            registry.BroadcastEvent(targetSessionID, "content", ContentEvent{
                Type:             "content",
                Content:          event.Content,
                ParentToolCallID: parentToolCallID,  // NEW
            })
        }

    case agent.AgentEventTypeResponse:
        // Stream tool calls
        toolCalls := event.Message.ToolCalls()
        for _, toolCall := range toolCalls {
            status := "running"
            if toolCall.Finished {
                status = "completed"
            }
            registry.BroadcastEvent(targetSessionID, "tool", ToolEvent{
                Type:             "tool",
                Name:             toolCall.Name,
                Input:            toolCall.Input,
                ID:               toolCall.ID,
                Status:           status,
                ParentToolCallID: parentToolCallID,  // NEW
            })
        }

        // Send completion event for final events
        if event.Done {
            if event.Message.FinishReason() == "permission_denied" {
                registry.BroadcastEvent(targetSessionID, "error", ErrorEvent{
                    Error:            "Permission denied",
                    ParentToolCallID: parentToolCallID,  // NEW
                })
            } else {
                content := event.Message.Content().String()
                reasoningContent := event.Message.ReasoningContent()
                registry.BroadcastEvent(targetSessionID, "complete", CompleteEvent{
                    Type:              "complete",
                    Content:           content,
                    MessageID:         event.Message.ID,
                    Done:              true,
                    Reasoning:         reasoningContent.String(),
                    ReasoningDuration: reasoningContent.Duration,
                    ParentToolCallID:  parentToolCallID,  // NEW
                })
            }
        }

    case agent.AgentEventTypeError:
        errMsg := event.Error.Error()
        if strings.Contains(errMsg, "rate_limit_error") {
            // Extract retry information
            retryAfter := 60
            attempt := 1
            maxAttempts := 8
            if strings.Contains(errMsg, "Retrying due to rate limit") {
                var currentAttempt, totalAttempts int
                _, err := fmt.Sscanf(errMsg, "Retrying due to rate limit... attempt %d of %d", &currentAttempt, &totalAttempts)
                if err == nil && currentAttempt > 0 && totalAttempts > 0 {
                    attempt = currentAttempt
                    maxAttempts = totalAttempts
                }
            }
            registry.BroadcastEvent(targetSessionID, "rate_limit_error", ErrorEvent{
                Error:            "This request would exceed your account's rate limit. The application will automatically retry.",
                Type:             "rate_limit_error",
                RetryAfter:       retryAfter,
                Attempt:          attempt,
                MaxAttempts:      maxAttempts,
                ParentToolCallID: parentToolCallID,  // NEW
            })
        } else {
            registry.BroadcastEvent(targetSessionID, "error", ErrorEvent{
                Error:            errMsg,
                ParentToolCallID: parentToolCallID,  // NEW
            })
        }

    case agent.AgentEventTypeSummarize:
        registry.BroadcastEvent(targetSessionID, "summarize", SummarizeEvent{
            Type:             "summarize",
            Progress:         event.Progress,
            Done:             event.Done,
            ParentToolCallID: parentToolCallID,  // NEW
        })
    }
}
```

**Note on PermissionEvent:** Permission events are broadcasted through a separate code path (not through `BroadcastAgentEventToSSE`). If permission requests need to be nested under task tools, you'll need to:

1. Track the tool context when permission is requested
2. Include `parentToolCallId` when broadcasting permission events from the permission handler
3. Pass tool context through the permission service layer

This is outside the scope of the current plan but may be needed for complete subagent visibility.

---

## Frontend Changes

### 1. Add parentToolCallId to TimelineEntry

**File**: `mix_web_demo/src/types/message.ts:20`

```typescript
export type TimelineEntry =
    | {
        type: "thinking";
        timestamp: number;
        content: string;
        id: string;
        parentToolCallId?: string;  // NEW
      }
    | {
        type: "tool";
        timestamp: number;
        content: ToolCall;
        id: string;
        parentToolCallId?: string;  // NEW
      }
    | {
        type: "content";
        timestamp: number;
        content: string;
        id: string;
        parentToolCallId?: string;  // NEW
      };
```

### 2. Capture parentToolCallId from SSE Events

**File**: `mix_web_demo/src/hooks/usePersistentSSE.ts`

Update event handlers that create timeline entries:

```typescript
// In thinking event handler
case "thinking":
    const thinkingEntry: TimelineEntry = {
        type: "thinking",
        timestamp: Date.now(),
        content: thinkingEvent.data.content,
        id: crypto.randomUUID(),
        parentToolCallId: thinkingEvent.data.parentToolCallId,  // NEW
    };

// In tool event handler
case "tool":
    const toolEntry: TimelineEntry = {
        type: "tool",
        timestamp: Date.now(),
        content: toolCall,
        id: toolCall.id,
        parentToolCallId: toolEvent.data.parentToolCallId,  // NEW
    };

// In content event handler
case "content":
    const contentEntry: TimelineEntry = {
        type: "content",
        timestamp: Date.now(),
        content: contentEvent.data.content,
        id: crypto.randomUUID(),
        parentToolCallId: contentEvent.data.parentToolCallId,  // NEW
    };
```

### 3. Group and Nest Timeline Entries

**File**: `mix_web_demo/src/components/conversation-display.tsx:133`

Replace `renderTimelineEntries` function:

```typescript
const renderTimelineEntries = (timeline: TimelineEntry[], isNested = false) => {
    if (!timeline || timeline.length === 0) return null;

    let entriesToRender = timeline;
    let nestedMap = new Map<string, TimelineEntry[]>();

    // Only group by parentToolCallId at the top level
    if (!isNested) {
        const topLevelEntries: TimelineEntry[] = [];

        for (const entry of timeline) {
            if (entry.parentToolCallId) {
                // This is a subagent event - group under parent tool
                const nested = nestedMap.get(entry.parentToolCallId) || [];
                nested.push(entry);
                nestedMap.set(entry.parentToolCallId, nested);
            } else {
                // Top-level entry
                topLevelEntries.push(entry);
            }
        }

        entriesToRender = topLevelEntries;
    }
    // else: Already grouped - render as-is

    // Group consecutive thinking entries
    const groupedEntries: Array<
        | { type: "thinking"; entries: string[]; timestamps: number[] }
        | { type: "tool"; entry: TimelineEntry; nestedEntries?: TimelineEntry[] }
        | { type: "content"; entry: TimelineEntry }
    > = [];

    for (const entry of entriesToRender) {
        if (entry.type === "thinking") {
            const lastGroup = groupedEntries[groupedEntries.length - 1];
            if (lastGroup && lastGroup.type === "thinking") {
                lastGroup.entries.push(entry.content);
                lastGroup.timestamps.push(entry.timestamp);
            } else {
                groupedEntries.push({
                    type: "thinking",
                    entries: [entry.content],
                    timestamps: [entry.timestamp],
                });
            }
        } else if (entry.type === "tool") {
            groupedEntries.push({
                type: "tool",
                entry,
                nestedEntries: nestedMap.get(entry.content.id),
            });
        } else {
            groupedEntries.push({ type: "content", entry });
        }
    }

    return groupedEntries.map((group, index) => {
        if (group.type === "thinking") {
            const totalContent = group.entries.join("");
            const duration =
                group.timestamps.length > 1
                    ? Math.round(
                          (group.timestamps[group.timestamps.length - 1] -
                              group.timestamps[0]) /
                              1000,
                      )
                    : 0;

            return (
                <AIReasoning
                    className="mb-4 w-full"
                    duration={duration > 0 ? duration : undefined}
                    isStreaming={false}
                    key={`thinking-${group.timestamps[0]}`}
                >
                    <AIReasoningTrigger />
                    <AIReasoningContent>{totalContent}</AIReasoningContent>
                </AIReasoning>
            );
        }

        if (group.type === "content") {
            return (
                <div className="mb-4" key={`content-${group.entry.id}`}>
                    <ResponseRenderer content={group.entry.content as string} />
                </div>
            );
        }

        // Tool with potential nested subagent events
        const toolCall = group.entry.content as ToolCall;
        const hasNestedEvents = group.nestedEntries && group.nestedEntries.length > 0;

        return (
            <AIToolLadder key={`tool-${group.entry.id}`}>
                <AIToolStep isLast={true} status={toolCall.status} stepNumber={1}>
                    <AIToolHeader
                        description={toolCall.description}
                        name={toolCall.name}
                        status={toolCall.status}
                        toolCall={toolCall}
                    />
                    <AIToolContent toolCall={toolCall} />

                    {/* Nested subagent events */}
                    {hasNestedEvents && (
                        <div className="mt-4 ml-4 border-l-2 border-muted pl-4">
                            <div className="text-xs text-muted-foreground mb-2 font-medium">
                                Subagent Activity
                            </div>
                            {renderTimelineEntries(group.nestedEntries, true)}
                        </div>
                    )}
                </AIToolStep>
            </AIToolLadder>
        );
    });
};
```

---

## Critical Implementation Notes

### Package Structure (Backend)

`withToolContext` is **package-private** (lowercase) because:

- Both `agent.go` and `task-tool.go` are in the **same package** (`agent`)
- No cross-package access is needed
- Using lowercase follows Go conventions for internal package functions
- No import statement needed in `task-tool.go`

The implementation reuses the existing `sessionRoutingKey` constant by extending the `sessionRouting` struct to include `ParentToolCallID`.

### Recursive Rendering Fix (Frontend)

The `renderTimelineEntries` function includes an `isNested` parameter to prevent a recursive grouping bug. Without this:

- Nested entries still have `parentToolCallId` set (e.g., "task-123")
- Recursive call tries to group them again by `parentToolCallId`
- They get placed into `nestedMap["task-123"]`, but tool "task-123" doesn't exist in the nested list
- **Result**: Nested events never render

**Solution**: Pass `isNested=true` when recursively rendering to skip the grouping logic.

### Complete Event Coverage

All relevant SSE event types must include `parentToolCallId` for consistency:

- `ThinkingEvent` - Subagent reasoning blocks
- `ContentEvent` - Subagent content generation
- `ToolEvent` - Tool calls from subagents
- `ToolParameterDeltaEvent` - Real-time tool parameter streaming
- `ToolExecutionStartEvent` - Tool execution start markers
- `ToolExecutionCompleteEvent` - Tool execution completion
- `CompleteEvent` - Marks subagent completion
- `ErrorEvent` - Tracks errors from subagents
- `SummarizeEvent` - Shows subagent progress updates
- `PermissionEvent` - Permission requests from subagent tools

**Excluded events:**

- `HeartbeatEvent` - Connection keepalive (not user-facing)
- `SessionCreatedEvent` - Session lifecycle (not scoped to tool calls)
- `ConnectedEvent` - SSE connection status (not scoped to tool calls)

This ensures all subagent activity appears nested under the parent task tool.

---

## Testing Checklist

### Backend Tests

- [ ] Verify `ParentToolCallID` propagates through context from task tool to subagent
- [ ] Confirm `withToolContext` correctly preserves existing RouteTo/Origin fields
- [ ] Ensure `withSessionRouting` correctly preserves existing ParentToolCallID field
- [ ] Verify `getSessionRouting` returns all three fields (routeTo, origin, parentToolCallID)
- [ ] Check SSE events include `parentToolCallId` for subagent events
- [ ] Ensure main session events have empty/missing `parentToolCallId`
- [ ] Test that multiple nested subagents maintain correct tool call IDs

### Frontend Tests

- [ ] Verify timeline entries capture `parentToolCallId` from SSE events
- [ ] Check subagent events nest under correct task tool in UI
- [ ] Test multiple concurrent subagents render in separate nesting groups
- [ ] Verify regular (non-task) tools render unchanged (no nesting)
- [ ] Confirm nested events render (not lost to recursive grouping bug)
- [ ] Test that error/complete/summarize events from subagents appear nested
- [ ] Verify thinking blocks from subagents appear nested under task tool
- [ ] Check content deltas from subagents appear nested
- [ ] Test tool parameter streaming in nested context

### End-to-End Tests

- [ ] Create task tool → verify subagent thinking/tools appear nested in real-time
- [ ] Cancel task tool → verify nested events stop streaming
- [ ] Reconnect SSE → verify nested events resume correctly with parentToolCallId
- [ ] Test with subagent that makes multiple tool calls
- [ ] Verify cost rollup still works correctly with nested events

---

## Benefits

- **Stateless frontend**: No session-level maps or state tracking
- **Survives reconnections**: Every event is self-describing
- **No database changes**: Pure context-based propagation
- **Recursive nesting**: Subagents can spawn sub-subagents naturally
- **Single source of truth**: Backend controls the relationship

---

## Session List Filtering (Follow-up)

Subagent sessions are filtered from the session list so they don't appear as clickable sessions in the UI.

### Backend Changes

**1. Filter REST Endpoint**

- File: `mix_agent/internal/http/rest_sessions.go:69`
- Skip subagent sessions: `if s.SessionType == "subagent" { continue }`

**2. Filter SSE Broadcast**

- File: `mix_agent/internal/http/sse.go:258`
- Skip `session_created` for subagents: `if sessionEvent.Payload.SessionType == session.SessionTypeSubagent { continue }`

**Result**: Only main and forked sessions appear in the session list. Subagent events still render nested under their parent task tool via `parentToolCallId`.

---

## Implementation Summary (2025-01-15)

**Status**: ✅ Complete

**Changes Made**:

1. **Backend** (Go): Added `ParentToolCallID` field to `AgentEvent` struct, extended context helpers (`withToolContext`, `withSessionRouting`, `getSessionRouting`), auto-populated field in `Publish()` method, and propagated to all 10 SSE event types
2. **OpenAPI**: Updated all SSE event schemas in `rest_docs.go` to include optional `parentToolCallId` field
3. **Frontend** (TypeScript): Added `parentToolCallId` to `TimelineEntry` type, captured from SSE events in `usePersistentSSE.ts`, implemented recursive nested rendering in `conversation-display.tsx` with `isNested` parameter

---

## Broker Isolation Issue

**Problem**: Each `NewAgent()` creates isolated `Broker[AgentEvent]` instance. Subagent publishes to Broker B, main session subscribes to Broker A → events never cross brokers.

**Solution**: Share parent agent's broker with subagent instead of creating new broker.

### Implementation

**1. Add broker parameter to agent constructor**

```go
// File: mix_agent/internal/llm/agent/builder.go

// Keep existing for backward compatibility
func NewAgent(...) (Service, error) {
    return NewAgentWithBroker(..., nil)
}

// New variant accepting optional broker
func NewAgentWithBroker(name string, sessions session.Service, messages message.Service,
    tools tools.Service, config Config, permissions permission.Service,
    broker *pubsub.Broker[AgentEvent]) (Service, error) {

    // Use provided broker or create new one
    if broker == nil {
        broker = pubsub.NewBroker[AgentEvent]()
    }

    return &agent{
        Broker: broker, // Use shared/new broker
        // ... rest of fields
    }
}
```

**2. Extract parent broker in task tool**

```go
// File: mix_agent/internal/llm/agent/task-tool.go

type taskTool struct {
    sessions    session.Service
    messages    message.Service
    permissions permission.Service
    parentAgent Service // Store parent agent reference
}

func (b *taskTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
    // ... existing validation ...

    // Create subagent with parent's broker
    parentBroker := b.parentAgent.(*agent).Broker // Extract broker from parent
    subAgent, err := NewAgentWithBroker("sub", b.sessions, b.messages, agentTools,
        session.DefaultConfig(), b.permissions, parentBroker)

    // ... rest unchanged ...
}
```

**3. Pass parent agent when building task tool**

```go
// File: mix_agent/internal/llm/agent/builder.go (in Build method)

taskTool := &taskTool{
    sessions:    a.sessions,
    messages:    a.messages,
    permissions: a.permissions,
    parentAgent: &a, // Pass agent reference
}
```

### Why Shared Broker > Re-publishing

| Aspect | Re-publishing | Shared Broker |
|--------|--------------|---------------|
| Event copies | 2 (original + forwarded) | 1 (single publish) |
| Goroutines | Extra forwarding loop | None |
| Memory | 2x events | 1x events |
| Complexity | Subscribe/forward/cleanup | Just pass reference |
| Error handling | Forwarding can fail | Direct publish |

### Result

- Subagent publishes to parent's broker → main session receives events automatically
- No event forwarding, no subscription management, no goroutine cleanup
- Single pub/sub system for parent and all subagents

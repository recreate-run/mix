# SSE Event Naming Refactor Plan

## Problem

The current `"tool"` event is overloaded - it's used for both "tool declared" and "parameters complete" by checking a `status` field. This is confusing and error-prone.

## Current vs Proposed Naming

### Current (Confusing)
```
1. "tool" (status="running")          → Tool declared by LLM
2. "tool_parameter_delta"             → Parameters streaming
3. "tool" (status="completed")        → Parameters complete
4. "tool_execution_start"             → Backend starts execution
5. "tool_execution_complete"          → Backend finishes execution
```

### Proposed (Clear)
```
1. "tool_use_start"                   → Tool declared by LLM
2. "tool_use_parameter_delta"         → Parameters streaming
3. "tool_use_complete"                → Parameters complete
4. "tool_execution_start"             → Backend starts execution
5. "tool_execution_complete"          → Backend finishes execution
```

## Changes Required

### Backend Files

1. **`mix_agent/internal/http/rest_messages_broadcast.go`**
   - Line 51-70: Split `AgentEventTypeResponse` case into two separate SSE events
     - Send `"tool_use_start"` when tool is first added (Finished=false)
     - Send `"tool_use_complete"` when parameters complete (Finished=true)
   - Line 41-49: Rename `"tool_parameter_delta"` → `"tool_use_parameter_delta"`

2. **`mix_agent/internal/http/sse_events.go`**
   - Remove: `ToolEvent` struct (with status field)
   - Add: `ToolUseStartEvent` struct
   - Add: `ToolUseCompleteEvent` struct
   - Rename: `ToolParameterDeltaEvent` → `ToolUseParameterDeltaEvent`

### Frontend Files

Search for and update all SSE event handlers:
- `event.type === "tool"` → Split into `"tool_use_start"` and `"tool_use_complete"`
- `event.status === "completed"` → Remove status checks, use event type
- `"tool_parameter_delta"` → `"tool_use_parameter_delta"`

Likely files:
- Event handling hooks
- SSE stream processors
- Type definitions

## Implementation Steps

1. **Update backend event types** (sse_events.go)
2. **Update backend event broadcasting** (rest_messages_broadcast.go)
3. **Update frontend to handle new event types**
4. **Remove old event types and status field**
5. **Update documentation**

## Breaking Change Strategy

Since this is a breaking change:
- Frontend and backend MUST be deployed together
- No backward compatibility needed (early stage startup per CLAUDE.md)

## Testing

After changes:
1. Verify SSE stream with browser DevTools
2. Test tool execution lifecycle shows all 5 events correctly
3. Test subagent tool events route correctly with new event types

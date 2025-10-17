# Callback Context Exclusion - Implementation Plan

## Problem Statement

All callback execution results and injected messages are included in the agent's conversation context. This can bloat the context window, leak implementation details, and increase token costs.

**Goal:** Allow callbacks to execute and save results WITHOUT including them in the agent's conversation history, while maintaining visibility in the UI.

---

## Approach 1: Metadata-Based Filtering (Per-Message Control)

### Architecture

Messages tagged with `exclude_from_context` metadata are filtered when loading conversation history. UI shows all messages regardless of flag.

```
Callback → Create message with metadata → Save to DB → UI displays
                                             ↓
Agent loads history → Filter excluded → Send to LLM
```

### Implementation Files

1. **`mix_agent/internal/llm/interfaces/callback.go`** - Add `ExcludeFromContext bool`, add validation
2. **`mix_agent/internal/llm/callbacks/executor.go`** - Add metadata to saved messages
3. **`mix_agent/internal/llm/agent/agent.go`** - Add filtering logic
4. **`mix_agent/internal/http/rest_docs.go`** - Update API schema
5. **`mix_dev_tool/src/components/right-sidebar.tsx`** - Add UI control

### Key Constraint

`send_message` callbacks **cannot** be excluded from context. They inject conversation messages that the agent must see for proper flow detection.

```go
// Validation in callback.go
if c.Type == CallbackTypeSendMessage && c.ExcludeFromContext {
    return fmt.Errorf("send_message callbacks cannot be excluded from context")
}
```

### Applicable Types

- ✅ **bash_script** - Can be excluded (stdout/stderr are metadata)
- ✅ **sub_agent** - Can be excluded (result summary is metadata)
- ❌ **send_message** - Cannot be excluded (creates conversation content)

### Pros & Cons

| Pros | Cons |
|------|------|
| Simple (2-3 hours) | Stores all messages in DB |
| Per-callback control | Must load then filter |
| Backward compatible | Metadata coupling |
| No schema changes | - |

---

## Approach 2: Architectural Separation (Out-of-Band Storage)

### Architecture

Callback results stored as metadata on parent tool messages or in separate table. Only `send_message` creates standalone conversation messages.

```
Callback → Attach to tool_result metadata → Store
              ↓                               ↓
Agent loads → Only conversation messages → UI merges both
```

### Implementation Files

1. **`mix_agent/internal/message/message.go`** - Add metadata structure
2. **`mix_agent/internal/llm/callbacks/executor.go`** - Attach instead of create
3. **`mix_agent/internal/http/rest_messages.go`** - Serialize from metadata
4. **Database** (optional) - Separate table with foreign key

### Pros & Cons

| Pros | Cons |
|------|------|
| Clean separation | Schema changes (6-8 hours) |
| Better performance | UI merges two sources |
| Fully async | Less flexible later |
| Scalable | Breaking change |

---

## Recommendation

**Start with Approach 1.** Migrate to Approach 2 later if callback volume justifies architectural investment.

---

## Implementation Checklist (Approach 1)

### Backend
- [ ] Add `ExcludeFromContext bool` to `CallbackConfig` (callback.go)
- [ ] Add validation: reject `excludeFromContext: true` for `send_message` type
- [ ] Update `saveCallbackResultMessage()` to include metadata (executor.go)
- [ ] Add `shouldExcludeFromContext()` helper (agent.go)
- [ ] Modify `loadConversationHistory()` to filter (agent.go)
- [ ] Update OpenAPI spec - only for bash_script/sub_agent (rest_docs.go)

### Frontend
- [ ] Add `excludeFromContext` checkbox to callback form (right-sidebar.tsx)
- [ ] Disable checkbox when type is `send_message`
- [ ] Update SDK to v0.8.1
- [ ] Show "excluded from context" badge in UI

### Testing
- [ ] Excluded callbacks don't appear in agent context
- [ ] Non-excluded callbacks appear normally
- [ ] UI displays all callbacks regardless of flag
- [ ] Validation rejects `send_message` with `excludeFromContext: true`
- [ ] Backward compatibility (defaults to `false`)

---

## API Schema Change

**File:** `mix_agent/internal/http/rest_docs.go`

Add to `CallbackConfig` schema:

```go
"excludeFromContext": map[string]interface{}{
    "type":        "boolean",
    "description": "Exclude callback results from agent context. Only applies to bash_script and sub_agent types. Not allowed for send_message.",
    "default":     false,
},
```

---

## Performance Impact

**Message Load:**
- Load 120 messages → Filter to 100 → Send 100 to LLM
- Overhead: O(n) in-memory filter (~1ms for typical sessions)

**Async Savings:**
- Callback result saves happen in background
- **Speedup:** 50-500ms per tool execution with callbacks

---

## Configuration

| Setting | Default | Reason |
|---------|---------|--------|
| `excludeFromContext` | `false` | Backward compatible |
| UI badge visibility | Always show | User awareness |
| Summary inclusion | Follow same flag | Consistency |

---

## Edge Cases

1. **Summary mode:** Excluded messages also excluded from summaries
2. **Export:** Document whether callback results are included in exports
3. **Migration:** Existing callbacks without metadata default to included
4. **Metadata tampering:** Server-controlled, not client-modifiable

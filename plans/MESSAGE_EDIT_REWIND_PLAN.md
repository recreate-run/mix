# Message Edit & Rewind Implementation Plan

## Overview

Replace the current "fork session" functionality with "edit message in-place" that rewinds the conversation to a specific point, allowing users to edit and resubmit messages without creating new sessions.

## Problem Statement

**Current Behavior (Fork):**
- Clicking pencil icon creates a new session
- Copies messages up to fork point
- Navigates to new session URL
- Creates session sprawl and loses context

**Desired Behavior (Edit/Rewind):**
- Clicking pencil icon enables edit mode
- Deletes messages after edit point
- Pre-populates input with original message
- Stays in same session
- Ensures LLM receives correct context

## Core Requirements

1. ✅ Backend must be source of truth (for LLM context correctness)
2. ✅ Delete messages after edit point (hard delete)
3. ✅ Clean up media files from deleted messages
4. ✅ Pre-populate input with message being edited
5. ✅ Stay in same session (no navigation)
6. ✅ Support canceling edit (Escape key)
7. ✅ Visual feedback during edit mode

## Architecture

### Two-Phase Flow

**Phase 1: Click Edit (Frontend Only)**
- Optimistically hide messages after edit point
- Pre-populate input field
- Show edit mode UI indicator
- Store pre-edit state for cancel

**Phase 2: Submit (Backend Sync)**
- Call rewind endpoint to delete messages
- Clean up associated media files
- Submit new edited message
- LLM generates response with correct context

## Implementation Tasks

### Backend Changes

#### 1. New Rewind Endpoint

**File:** `mix_agent/internal/http/rest_sessions.go`

**New types:**
```go
type RewindSessionRequest struct {
    MessageIndex int64 `json:"messageIndex"` // Delete messages AFTER this index
    CleanupMedia bool  `json:"cleanupMedia"` // Whether to clean up media files
}

type RewindSessionResponse struct {
    ID                    string    `json:"id"`
    Title                 string    `json:"title"`
    UserMessageCount      int64     `json:"userMessageCount"`
    AssistantMessageCount int64     `json:"assistantMessageCount"`
    ToolCallCount         int64     `json:"toolCallCount"`
    PromptTokens          int64     `json:"promptTokens"`
    CompletionTokens      int64     `json:"completionTokens"`
    Cost                  float64   `json:"cost"`
    CreatedAt             time.Time `json:"createdAt"`
}
```

**New handler:**
```go
func (h *SessionHandler) HandleRewindSession(w http.ResponseWriter, r *http.Request)
```

**Endpoint:** `POST /api/sessions/{id}/rewind`

**Logic:**
1. Validate request (sessionID, messageIndex >= 0)
2. If cleanupMedia=true, get messages that will be deleted
3. Call `Messages.DeleteAfterIndex(ctx, sessionID, messageIndex)`
4. Call `cleanupMediaFromMessages(deletedMessages, sessionStorageDir)`
5. Return updated session metadata

#### 2. Message Service Method

**File:** `mix_agent/internal/message/message.go`

**Add to Service interface:**
```go
DeleteAfterIndex(ctx context.Context, sessionID string, messageIndex int64) error
```

**Implementation:**
```go
func (s *service) DeleteAfterIndex(ctx context.Context, sessionID string, messageIndex int64) error {
    messages, err := s.List(ctx, sessionID)
    if err != nil {
        return fmt.Errorf("failed to list messages: %w", err)
    }

    if messageIndex >= int64(len(messages)) {
        return nil // No messages to delete
    }

    // Delete messages after messageIndex (inclusive means keep 0..messageIndex)
    for i := int(messageIndex) + 1; i < len(messages); i++ {
        if err := s.Delete(ctx, messages[i].ID); err != nil {
            return fmt.Errorf("failed to delete message at index %d: %w", i, err)
        }
    }

    return nil
}
```

#### 3. Media Cleanup Utility

**File:** `mix_agent/internal/session/media_cleanup.go` (NEW)

**Functions:**
```go
func CleanupMediaFromMessages(messages []message.Message, sessionStorageDir string) error
func extractLocalFilePath(url string) string
func extractMediaPathFromToolResult(toolResult message.ToolResult) string
```

**Logic:**
- Parse message parts for media references (ImageURLContent, BinaryContent, URIContent)
- Parse tool results for media_output paths
- Delete files from session storage directory
- Log errors but don't fail (orphaned files are acceptable)

#### 4. Register Route

**File:** `mix_agent/internal/http/server.go`

```go
mux.HandleFunc("POST /api/sessions/{id}/rewind", sessionHandler.HandleRewindSession)
```

#### 5. Update API Documentation

**File:** `mix_agent/internal/http/rest_docs.go`

Add rewind endpoint documentation with request/response schemas.

#### 6. Integration Tests

**File:** `mix_agent/internal/http/session_rewind_integration_test.go` (NEW)

**Test cases:**
1. Rewind to middle message (delete last 3 of 6 messages)
2. Rewind with media cleanup enabled
3. Rewind with image attachments (verify files deleted)
4. Rewind with tool-generated media (verify cleanup)
5. Rewind with missing media files (should not fail)
6. Rewind to last message (no deletion)
7. Rewind to first message (delete all but 1)
8. Rewind with invalid messageIndex (negative)
9. Rewind non-existent session (404 error)
10. Verify LLM context correctness after rewind

### Frontend Changes

#### 1. New Rewind Hook

**File:** `mix_playground/src/hooks/useRewindSession.ts` (NEW)

```typescript
interface RewindSessionParams {
  sessionId: string;
  messageIndex: number;
  cleanupMedia?: boolean;
}

export function useRewindSession() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: RewindSessionParams) => {
      return await mix.sessions.rewind({
        id: params.sessionId,
        requestBody: {
          messageIndex: params.messageIndex,
          cleanupMedia: params.cleanupMedia ?? true,
        },
      });
    },
    onSuccess: (data, variables) => {
      invalidateSessionCaches(queryClient, variables.sessionId);
    },
  });
}
```

#### 2. Update ChatApp Component

**File:** `mix_playground/src/components/chat-app.tsx`

**New state:**
```typescript
const [editingMessageIndex, setEditingMessageIndex] = useState<number | null>(null);
const [preEditMessages, setPreEditMessages] = useState<UIMessage[]>([]);
```

**New handlers:**
```typescript
const handleEditMessage = (messageIndex: number) => {
  // Store pre-edit state
  // Pre-populate input with message content
  // Restore attachments
  // Hide messages after edit point
  // Focus input
}

const handleCancelEdit = () => {
  // Restore original messages
  // Clear edit state
  // Clear input
}

const rewindAndSubmit = async (messageIndex: number, newContent: string, attachments: Attachment[]) => {
  // Call rewind endpoint
  // Submit new message via normal flow
}
```

**Update handleSubmit:**
```typescript
if (editingMessageIndex !== null) {
  // Edit mode - rewind then submit
  await rewindAndSubmit(editingMessageIndex, text, attachments);
  setEditingMessageIndex(null);
} else {
  // Normal submission
  await normalSubmitFlow();
}
```

**Add Escape key listener:**
```typescript
useEffect(() => {
  const handleEscape = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && editingMessageIndex !== null) {
      handleCancelEdit();
    }
  };
  window.addEventListener('keydown', handleEscape);
  return () => window.removeEventListener('keydown', handleEscape);
}, [editingMessageIndex]);
```

**Add edit mode banner:**
```typescript
{editingMessageIndex !== null && (
  <div className="fixed top-0 z-50 bg-yellow-500/20 border-b border-yellow-500 px-4 py-2">
    <span>✏️ Editing message {editingMessageIndex + 1}. Press Enter to submit or Esc to cancel.</span>
    <Button onClick={handleCancelEdit}>Cancel</Button>
  </div>
)}
```

#### 3. Update ConversationDisplay Component

**File:** `mix_playground/src/components/conversation-display.tsx`

**Update props:**
```typescript
interface ConversationDisplayProps {
  // ... existing props
  onEditMessage?: (index: number) => void; // Renamed from onForkMessage
  editingMessageIndex?: number | null; // New prop for visual indicator
}
```

**Update button callback:**
```typescript
{onEditMessage && (
  <Button
    onClick={() => onEditMessage(index)}
    title="Edit this message"
  >
    <Pencil className="size-4" />
  </Button>
)}
```

**Add visual indicator:**
```typescript
<AIMessage
  className={editingMessageIndex === index ? 'ring-2 ring-primary' : ''}
>
```

#### 4. Update SDK (if needed)

**File:** `mix_sdk/openapi.json`

Add rewind endpoint schema and regenerate TypeScript SDK.

### Database Changes

**None required** - Using existing `DeleteMessage` query for hard delete.

## Edge Cases

### 1. Edit Last Message
```typescript
// messageIndex = last index
// Delete only that message and create new one
rewindIndex = messageIndex - 1;
```

### 2. Edit While Streaming
```typescript
if (sseStream.processing) {
  await sseStream.cancel();
}
// Then proceed with edit
```

### 3. Media Cleanup Failures
- Log warning but don't fail request
- Orphaned files acceptable (cleanup later with maintenance job)
- Message deletion succeeds regardless

### 4. Concurrent Edits
- Last write wins (backend is source of truth)
- Frontend refreshes on focus to get latest state
- No conflict resolution needed

### 5. Edit Empty Session
```typescript
if (messages.length === 0) {
  // No-op or disable edit button
}
```

### 6. Network Failures
```typescript
try {
  await rewindAndSubmit(...);
} catch (error) {
  // Restore pre-edit state
  setMessages(preEditMessages);
  showErrorToast('Failed to update message');
}
```

## File Changes Summary

### Backend (Go)
1. ✅ `mix_agent/internal/http/rest_sessions.go` - Add HandleRewindSession
2. ✅ `mix_agent/internal/message/message.go` - Add DeleteAfterIndex method
3. ✅ `mix_agent/internal/session/media_cleanup.go` - NEW file for media cleanup
4. ✅ `mix_agent/internal/http/server.go` - Register rewind route
5. ✅ `mix_agent/internal/http/rest_docs.go` - Update API docs
6. ✅ `mix_agent/internal/http/session_rewind_integration_test.go` - NEW integration tests

### Frontend (TypeScript/React)
1. ✅ `mix_playground/src/hooks/useRewindSession.ts` - NEW rewind hook
2. ✅ `mix_playground/src/components/chat-app.tsx` - Add edit mode logic
3. ✅ `mix_playground/src/components/conversation-display.tsx` - Update props and button

### Configuration
1. ✅ `mix_sdk/openapi.json` - Add rewind endpoint schema
2. ✅ Regenerate TypeScript SDK

## Testing Strategy

### Unit Tests
- Message service DeleteAfterIndex
- Media cleanup functions
- Frontend state management

### Integration Tests
- Full rewind flow (backend)
- Media cleanup with various message types
- Error handling and rollback

### Manual Testing
1. Edit middle message → verify messages deleted
2. Edit with images → verify files cleaned up
3. Edit with tool-generated media → verify cleanup
4. Press Escape → verify cancel works
5. Edit during streaming → verify stream cancels
6. Network error → verify state rollback
7. Submit edited message → verify LLM context correct

## Success Criteria

1. ✅ User can click pencil icon to edit any user message
2. ✅ Messages after edit point are hidden immediately
3. ✅ Input is pre-populated with original message
4. ✅ User can cancel with Escape key
5. ✅ Submitting edited message deletes old messages from DB
6. ✅ Media files are cleaned up from disk
7. ✅ LLM receives correct context (only messages before edit point)
8. ✅ Session stays on same URL (no navigation)
9. ✅ Visual feedback during edit mode
10. ✅ Error handling with state rollback

## Migration Notes

### Existing Fork Sessions
- Old forked sessions remain in database
- Can optionally add cleanup script to identify and archive orphaned forks
- No immediate action required

### API Backwards Compatibility
- New `/rewind` endpoint doesn't affect existing endpoints
- Frontend changes are internal (no breaking API changes)
- Old fork endpoint can be deprecated in future release

## Timeline Estimate

### Phase 1: Backend (2-3 hours)
- Rewind endpoint: 1 hour
- Media cleanup: 1 hour
- Tests: 1 hour

### Phase 2: Frontend (2-3 hours)
- Hook and state management: 1 hour
- UI updates: 1 hour
- Edge cases and polish: 1 hour

### Phase 3: Testing & Refinement (1 hour)
- Manual testing
- Bug fixes
- Documentation

**Total: 5-7 hours**

## Future Enhancements

1. **Undo Rewind:** Store rewind history for 24 hours, allow undo
2. **Soft Delete:** Archive messages instead of hard delete (with expiration)
3. **Diff View:** Show what changed between original and edited message
4. **Bulk Edit:** Edit multiple messages in sequence
5. **Version History:** Track all versions of edited messages
6. **Media Preview:** Show media that will be deleted before confirming
7. **Keyboard Shortcuts:** Quick edit with Ctrl+E or similar

## Security Considerations

1. ✅ Validate session ownership (existing auth checks)
2. ✅ Validate messageIndex bounds
3. ✅ Prevent path traversal in media cleanup
4. ✅ Rate limit rewind endpoint to prevent abuse
5. ✅ Audit log for message deletions (optional)

## Performance Considerations

1. ✅ Batch delete messages (current Delete method already efficient)
2. ✅ Media cleanup runs asynchronously (doesn't block response)
3. ✅ Frontend optimistic updates (instant UX)
4. ✅ Cache invalidation is targeted (only affected session)

## Rollback Plan

If critical issues arise:
1. Keep old fork endpoint active
2. Add feature flag to toggle between fork/rewind
3. Frontend can fall back to fork behavior
4. No data loss (messages are deleted intentionally)
5. Can restore from backups if needed

---

**Status:** Ready for implementation
**Priority:** High
**Risk Level:** Medium (destructive operation, but intentional)
**Dependencies:** None (uses existing infrastructure)
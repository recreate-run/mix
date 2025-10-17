# Eliminate Reload After Streaming Completes

## Problem Statement

Currently, after streaming completes, the frontend:
1. Invalidates React Query cache
2. Refetches all messages from the database via `GET /api/sessions/{id}/messages`
3. Shows visible loading/reload state (bad UX)
4. Clears streaming UI after refetch completes

This creates an unnecessary round-trip to the database when the backend already has all the message data.

## Root Cause

The SSE `complete` event only sends the `messageId`, not the full message object:

```go
// Current: sse.go:374
CompleteEvent{
  MessageID: event.Message.ID,  // Only ID, not full object
  Content: content,
  Reasoning: reasoning,
  ReasoningDuration: reasoningDuration
}
```

Without the full message objects (user + assistant), the frontend can't update the React Query cache directly and must refetch.

## Solution: Send Full Message Objects via SSE

### Approach

Instead of refetching after streaming, send complete message objects via SSE:

1. **New Event: `user_message_created`** - Sent immediately after user message is saved to DB
2. **Enhanced Event: `complete`** - Include full assistant message object

This allows the frontend to update the React Query cache directly using `setQueryData`.

## Implementation Plan

### Backend Changes

#### 1. Define New SSE Event Types (mix_agent/internal/http/sse.go)

```go
// UserMessageCreatedEvent - sent after user message is saved to DB
type UserMessageCreatedEvent struct {
	Type      string      `json:"type"`
	Message   MessageData `json:"message"`  // Full message object with ID
}

// CompleteEvent - enhanced to include full assistant message
type CompleteEvent struct {
	Type              string      `json:"type"`
	Content           string      `json:"content"`
	MessageID         string      `json:"messageId"`
	Message           MessageData `json:"message"`  // NEW: Full assistant message object
	Done              bool        `json:"done"`
	Reasoning         string      `json:"reasoning,omitempty"`
	ReasoningDuration int64       `json:"reasoningDuration,omitempty"`
}
```

#### 2. Send user_message_created Event (mix_agent/internal/llm/agent/coder.go)

After saving the user message to the database:

```go
// After userMsg is saved (around line where we have userMsg)
userMessageData := h.convertMessageToData(userMsg)
if err := sseWriter.WriteEvent("user_message_created", UserMessageCreatedEvent{
	Type:    "user_message_created",
	Message: userMessageData,
}); err != nil {
	return err
}
```

#### 3. Send Full Assistant Message in complete Event (mix_agent/internal/http/sse.go)

Modify the complete event broadcast:

```go
// Find where we send complete event (around line 374)
assistantMessageData := h.convertMessageToData(event.Message)
if err := sseWriter.WriteEvent("complete", CompleteEvent{
	Type:              "complete",
	Content:           content,
	MessageID:         event.Message.ID,
	Message:           assistantMessageData,  // NEW: Full message
	Done:              true,
	Reasoning:         reasoning,
	ReasoningDuration: reasoningDuration,
}); err != nil {
	return err
}
```

#### 4. Create Message Conversion Helper (if doesn't exist)

Reuse existing `convertMessagesToData` logic from rest_messages.go:

```go
func (h *MessageHandler) convertMessageToData(msg message.Message) MessageData {
	// Extract from existing convertMessagesToData logic
	// Returns single MessageData with ID, content, tool calls, reasoning, etc.
}
```

### Frontend Changes

#### 1. Add New Event Handler (mix_dev_tool/src/hooks/usePersistentSSE.ts)

```typescript
case "user_message_created": {
	const userMessageEvent = event as SSEUserMessageCreatedEvent;
	const message = userMessageEvent.data.message;

	// Update React Query cache with user message
	queryClient.setQueryData<MessageData[]>(
		CACHE_KEYS.sessionMessages(sessionId),
		(old = []) => [...old, message]
	);

	// Clear pending user message (it's now in cache)
	setState((prev) => ({
		...prev,
		pendingUserMessage: null,
	}));
	break;
}
```

#### 2. Enhance complete Event Handler (mix_dev_tool/src/hooks/usePersistentSSE.ts)

```typescript
case "complete": {
	const completeEvent = event as SSECompleteEvent;

	// Update React Query cache with assistant message
	if (completeEvent.data.message) {
		queryClient.setQueryData<MessageData[]>(
			CACHE_KEYS.sessionMessages(sessionId),
			(old = []) => [...old, completeEvent.data.message]
		);
	}

	setState((prev) => ({
		...prev,
		reasoning: completeEvent.data.reasoning || null,
		reasoningDuration: completeEvent.data.reasoningDuration || null,
		completed: true,
		processing: false,
		assistantMessageId: completeEvent.data.messageId || null,
	}));
	break;
}
```

#### 3. Remove Cache Invalidation (mix_dev_tool/src/components/chat-app.tsx)

**DELETE** the invalidation useEffect (lines 133-152):

```typescript
// DELETE THIS ENTIRE EFFECT
useEffect(() => {
	if (
		sseStream.completed &&
		!sseStream.processing &&
		(sseStream.finalContent || sseStream.toolCalls.length > 0) &&
		session?.id
	) {
		// First, invalidate cache to fetch fresh messages from backend
		queryClient.invalidateQueries({
			queryKey: CACHE_KEYS.sessionMessages(session.id),
		});
	}
}, [...]);
```

#### 4. Simplify Streaming Cleanup (mix_dev_tool/src/components/chat-app.tsx)

Modify the second useEffect (lines 155-171) to trigger immediately after completion:

```typescript
// Clear streaming UI immediately after completion (no need to wait for refetch)
useEffect(() => {
	if (
		sseStream.completed &&
		!sseStream.processing &&
		(sseStream.finalContent || sseStream.toolCalls.length > 0)
	) {
		// Cache already updated via SSE - safe to clear streaming UI
		sseStream.clearStreamingContent();
	}
}, [
	sseStream.completed,
	sseStream.processing,
	sseStream.finalContent,
	sseStream.toolCalls.length,
	sseStream.clearStreamingContent,
]);
```

#### 5. Add TypeScript Types

Update SDK types or create local types:

```typescript
interface SSEUserMessageCreatedEvent {
	type: "user_message_created";
	data: {
		message: MessageData;
	};
}

interface SSECompleteEvent {
	type: "complete";
	data: {
		content: string;
		messageId: string;
		message: MessageData;  // NEW
		done: boolean;
		reasoning?: string;
		reasoningDuration?: number;
	};
}
```

## Benefits

1. **No Reload** - Eliminates visible reload/flicker after streaming
2. **Faster** - No unnecessary API call to refetch messages
3. **Simpler** - Fewer useEffects, cleaner logic
4. **Better UX** - Smooth transition from streaming to final messages
5. **Net Code Reduction** - Remove invalidation logic, add direct cache updates

## Migration Notes

- **Backward Compatible**: Old clients will ignore new events, still refetch (works but slower)
- **No Database Changes**: Only SSE event structure changes
- **Type Safety**: Ensure SDK types are updated for new event structure

## Testing Checklist

- [ ] User message appears immediately after sending (no pending state)
- [ ] Streaming content displays correctly
- [ ] Assistant message appears without reload after streaming completes
- [ ] No flash/flicker during transition
- [ ] Tab switching during streaming works correctly (no duplicates)
- [ ] Error cases handled properly (failed message send, etc.)
- [ ] Multiple rapid messages work correctly
- [ ] Cache stays in sync with database

## Files to Modify

### Backend
- `mix_agent/internal/http/sse.go` - Event type definitions and broadcasting
- `mix_agent/internal/llm/agent/coder.go` - Send user_message_created event
- `mix_agent/internal/http/rest_messages.go` - Reuse message conversion logic

### Frontend
- `mix_dev_tool/src/hooks/usePersistentSSE.ts` - Handle new events, update cache
- `mix_dev_tool/src/components/chat-app.tsx` - Remove invalidation logic
- SDK types (if we control them) - Add new event types

## Estimated Lines of Code

- **Backend**: +30 lines (new event types, emission)
- **Frontend**: +25 lines (event handlers, cache updates)
- **Removed**: -40 lines (invalidation logic, waiting for refetch)
- **Net**: +15 lines (complexity reduced significantly)

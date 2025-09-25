# API Structure Refactoring Plan: Eliminate JSON-String Encoding

## Overview
The current API accepts JSON-stringified structured data as a simple string parameter. This creates semantic mismatch between the documented interface and actual usage, introduces unnecessary parsing overhead, and creates potential failure points.

**Goal**: Align API interface with actual data structure to eliminate JSON encoding/decoding pattern.

## Current Problem
- **API Contract**: `{ "content": "string" }`
- **Actual Usage**: `{ "content": JSON.stringify(messageData) }`
- **Result**: Backend must parse JSON from string, creating complexity and error vectors

## Target Architecture
- **New API Contract**: `{ "text": "string", "media": ["string"], "apps": ["string"], "plan_mode": boolean }`
- **Data Flow**: Direct object sending without JSON stringification
- **Benefits**: Zero parsing overhead, type safety, self-documenting API

---

## Implementation Plan

### Phase 1: Backend API Structure (Foundation)

#### 1.1 Update Request Schema
**File**: `mix_agent/internal/http/rest_messages.go`
- **Lines 110-112**: Replace `SendMessageRequest` struct:
  ```go
  // Current
  type SendMessageRequest struct {
      Content string `json:"content"`
  }

  // New
  type SendMessageRequest struct {
      Text     string   `json:"text"`
      Media    []string `json:"media"`
      Apps     []string `json:"apps"`
      PlanMode bool     `json:"plan_mode"`
  }
  ```

- **Line 168**: Update handler logic:
  ```go
  // Current: UserInput: req.Content,
  // New: UserInput: req.Text,
  ```

- **Lines 177-178**: Update slash command parsing:
  ```go
  // Current: Uses req.Content for slash commands
  // New: Use req.Text directly
  ```

#### 1.2 Update OpenAPI Documentation
**File**: `mix_agent/internal/http/rest_docs.go`
- **Lines 198-201**: Replace schema definition:
  ```go
  // Current: Single "content" string field
  // New: Structured object with text, media, apps, plan_mode fields
  ```

### Phase 2: Remove Legacy Parsing (Cleanup)

#### 2.1 Eliminate SSE JSON Parsing Functions
**File**: `mix_agent/internal/http/sse.go`
- **Lines 243-244**: **DELETE** `extractText()` function entirely
- **Lines 252-254**: **DELETE** `parseMessageContent()` function entirely
- Update SSE handler to work directly with structured request data

### Phase 3: Frontend Simplification (Final)

#### 3.1 Direct Object Sending
**File**: `mix_playground/src/hooks/usePersistentSSE.ts`
- **Line 602**: Remove JSON.stringify:
  ```typescript
  // Current: await sendMessage(JSON.stringify(messageData));
  // New: await sendMessage(messageData);
  ```

---

## Implementation Sequence

### Step 1: Backend Foundation
1. Update `SendMessageRequest` struct (`rest_messages.go:110-112`)
2. Update request body schema (`rest_docs.go:198-201`)
3. Update handler logic (`rest_messages.go:168, 177-178`)

### Step 2: Remove Parsing Layer
1. Delete `extractText()` function (`sse.go:243-244`)
2. Delete `parseMessageContent()` function (`sse.go:252-254`)
3. Update SSE handlers to use direct structured data

### Step 3: Frontend Cleanup
1. Remove `JSON.stringify()` call (`usePersistentSSE.ts:602`)
2. Verify TypeScript SDK regenerates correctly

---

## Benefits After Refactoring

### Performance
- **Zero JSON parsing overhead** on backend
- **Reduced payload size** (no JSON escaping)
- **Fewer CPU cycles** per message

### Maintainability
- **Self-documenting API** (schema matches actual usage)
- **Type safety** throughout the pipeline
- **Eliminates semantic contracts** (hidden expectations)

### Reliability
- **No JSON parsing failures** from malformed strings
- **Proper validation** at API boundary
- **Clear error messages** for invalid data

---

## Risk Assessment

### Low Risk
- Changes are surgical and well-defined
- Existing data structures already match target schema
- No database schema changes required

### Migration Strategy
- **No backward compatibility needed** (per project requirements)
- **Atomic deployment** (all changes deploy together)
- **TypeScript SDK auto-regenerates** from OpenAPI spec

---

## Validation Checklist

### Backend Tests
- [ ] Message sending works with structured request
- [ ] SSE streaming works without JSON parsing
- [ ] Slash commands work with structured text field

### Frontend Tests
- [ ] `usePersistentSSE` sends structured objects
- [ ] TypeScript compilation passes
- [ ] Message history loads correctly

### API Documentation
- [ ] OpenAPI spec reflects actual structure
- [ ] Generated SDK matches new interface
- [ ] API examples show correct usage

---

## Files Modified

### Backend
- `mix_agent/internal/http/rest_messages.go` (lines 110-112, 168, 177-178)
- `mix_agent/internal/http/rest_docs.go` (lines 198-201)
- `mix_agent/internal/http/sse.go` (remove lines 243-244, 252-254)

### Frontend
- `mix_playground/src/hooks/usePersistentSSE.ts` (line 602)

### Auto-Generated
- TypeScript SDK will regenerate from updated OpenAPI spec
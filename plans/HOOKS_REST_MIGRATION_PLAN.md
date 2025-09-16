# Tauri App Hooks REST API Migration Plan

## Overview

This plan outlines the migration of all 16 hooks in `tauri_app/src/hooks/` from the legacy JSON-RPC system to the new Mix TypeScript SDK with REST API endpoints. The migration will replace all `rpcCall()` instances with type-safe SDK calls while maintaining existing functionality and user experience.

## Current State Analysis

### Hook Inventory (16 Total Files)

#### **RPC-Dependent Hooks (10 hooks requiring migration):**

1. `useSession.ts` - Session CRUD operations
2. `useSessionsList.ts` - Session listing and management
3. `useForkSession.ts` - Session forking functionality
4. `useSessionMessages.ts` - Session message loading
5. `useMessages.ts` - Message sending
6. `useMessageHistory.ts` - Cross-session message history
7. `useMessageHistoryNavigation.ts` - Message history UI navigation
8. `usePersistentSSE.ts` - Real-time streaming with RPC control calls
9. `useMCPList.ts` - MCP server management
10. `usePermissions.ts` - Runtime permission handling (limited RPC usage)

#### **Non-RPC Hooks (6 hooks - no migration needed):**

1. `use-copy-to-clipboard.ts` - Pure utility hook
2. `use-mobile.ts` - Responsive design hook
3. `useFileSystem.ts` - Tauri filesystem APIs only
4. `useFileReference.ts` - File selection state management
5. `useFileTypes.ts` - HTTP fetch to existing REST endpoint
6. `useOpenApps.ts` - Tauri native APIs only

### Current RPC Architecture

#### **Core Infrastructure:**

- **RPC Client**: `/lib/rpc.ts` with single `rpcCall<T>()` function
- **Protocol**: JSON-RPC over HTTP POST to `localhost:8088/rpc`
- **Cache**: TanStack Query with hierarchical cache keys
- **Error Handling**: RPC-specific error format and propagation

#### **RPC Endpoints Currently Used:**

- **Sessions**: `sessions.create`, `sessions.list`, `sessions.get`, `sessions.select` (REMOVED), `sessions.delete`, `sessions.fork`
- **Messages**: `messages.send`, `messages.list`, `messages.history`  
- **System**: `mcp.list`, `agent.cancel`, `permission.grant`, `permission.deny`

## REST API Endpoint Mapping

Based on the REST API migration plan, here's the complete RPC → REST mapping:

### Sessions API Migration

- `sessions.create` → `mix.sessions.create()` (see `mix-typescript-sdk/README.md:142`)
- `sessions.list` → `mix.sessions.list()` (see `mix-typescript-sdk/README.md:141`)
- `sessions.get` → `mix.sessions.get()` (see `mix-typescript-sdk/README.md:144`)
- `sessions.delete` → `mix.sessions.delete()` (see `mix-typescript-sdk/README.md:143`)
- `sessions.fork` → `mix.sessions.fork()` (see `mix-typescript-sdk/README.md:145`)
- `sessions.select` → **REMOVED - Stateless Design**

### Messages API Migration  

- `messages.send` → `mix.messages.send()` (see `mix-typescript-sdk/README.md:131`)
- `messages.list` → `mix.messages.getSession()` (see `mix-typescript-sdk/README.md:130`)
- `messages.history` → `mix.messages.getHistory()` (see `mix-typescript-sdk/README.md:128`)

### System API Migration

- `mcp.list` → `mix.system.listMcpServers()` (see `mix-typescript-sdk/README.md:151`)
- `agent.cancel` → `mix.messages.cancelProcessing()` (see `mix-typescript-sdk/README.md:129`)
- `permission.grant` → `mix.permissions.grant()` (see `mix-typescript-sdk/README.md:137`)
- `permission.deny` → `mix.permissions.deny()` (see `mix-typescript-sdk/README.md:136`)

## Hook-by-Hook Migration Plan

### Phase 1: Core Session Management Hooks

#### 1. `useSession.ts` - Session CRUD Operations

**Current RPC Usage:**

- Line 15: `rpcCall('sessions.create', {title, sessionStorageDirectory})`
- Line 28: `rpcCall('sessions.get', {id: sessionId})`

**Migration Changes:**

- Replace RPC calls with SDK methods from `mix-typescript-sdk`
- Update response handling for REST format
- Maintain existing TanStack Query integration at lines 11-35
- Update cache keys defined in `/lib/cache-keys.ts`

#### 2. `useSessionsList.ts` - Session List Management

**Current RPC Usage:**

- Line 18: `rpcCall('sessions.list', {})`
- Line 34: `rpcCall('sessions.select', {id: sessionId})`
- Line 52: `rpcCall('sessions.delete', {id: sessionId})`

**Migration Changes:**

- **BREAKING**: Remove `useSelectSession()` hook entirely (lines 30-45)
- Update session list query at line 18 to use SDK
- Update delete mutation at line 52 to use SDK
- Remove global "current session" state management
- Update optimistic update logic at lines 55-70
- Update cache invalidation patterns referencing `/lib/session-cache.ts`

#### 3. `useForkSession.ts` - Session Forking

**Current RPC Usage:**

- Line 12: `rpcCall('sessions.fork', {sourceSessionId, messageIndex, title})`

**Migration Changes:**

- Replace RPC call at line 12 with SDK method
- Update parameter structure - SDK may have different parameter format
- Update response handling at lines 15-20
- Maintain existing cache invalidation using `/lib/session-cache.ts`

### Phase 2: Message Management Hooks

#### 4. `useSessionMessages.ts` - Session Message Loading

**Current RPC Usage:**

- Line 25: `rpcCall('messages.list', {sessionId})`

**Migration Changes:**

- Replace RPC call at line 25 with SDK method
- Update response data extraction at lines 28-32
- Maintain existing message transformation logic at lines 35-50
- Update cache keys referencing `/lib/cache-keys.ts`

#### 5. `useMessages.ts` - Message Sending  

**Current RPC Usage:**

- Line 14: `rpcCall('messages.send', {content, sessionId})`

**Migration Changes:**

- Replace RPC call at line 14 with SDK method
- Update parameter structure - check SDK documentation for exact format
- Maintain cache invalidation logic at lines 18-22
- Update error handling at lines 25-30

#### 6. `useMessageHistory.ts` - Message History with Pagination

**Current RPC Usage:**

- Line 28: `rpcCall('messages.history', {limit, offset})`

**Migration Changes:**

- Replace RPC call at line 28 with SDK method
- Update infinite query configuration at lines 15-35
- Maintain existing pagination logic at lines 40-55
- Update response data extraction at lines 30-38

#### 7. `useMessageHistoryNavigation.ts` - Message History Navigation

**Current Dependencies:**

- Uses `useMessageHistory` internally at line 8
- No direct RPC calls

**Migration Changes:**

- Test integration with updated `useMessageHistory` hook
- Verify attachment reconstruction logic at lines 45-60
- Update data access patterns if response format changes

### Phase 3: Real-time Communication & Permissions

#### 8. `usePersistentSSE.ts` - Complex Real-time Streaming

**Current RPC Usage:**

- Line 145: `rpcCall('agent.cancel', {sessionId})`
- Line 267: `rpcCall('permission.grant', {id})`
- Line 275: `rpcCall('permission.deny', {id})`

**Migration Changes:**

- Replace RPC control calls at lines 145, 267, 275 with SDK methods
- Verify SSE endpoint compatibility - check backend SSE implementation
- Update message sending logic if needed
- Maintain complex state management for tool calls (lines 180-250)
- Update permission handling workflow (lines 260-285)

#### 9. `useMCPList.ts` - MCP Server Management

**Current RPC Usage:**

- Line 12: `rpcCall('mcp.list', {})`

**Migration Changes:**

- Replace RPC call at line 12 with SDK method
- Update response data handling at lines 15-18
- Maintain existing caching strategy with TanStack Query

#### 10. `usePermissions.ts` - System Permissions (Mixed Usage)

**Current Usage:**

- Primarily uses Tauri native APIs for macOS permissions (lines 15-80)
- Limited RPC usage for runtime permissions (check for any rpcCall instances)

**Migration Changes:**

- Review any runtime permission RPC calls in the file
- Update to use SDK permission methods if found
- Maintain native macOS permission handling unchanged

### Phase 4: Infrastructure Updates

#### Core Infrastructure Changes

**1. Replace RPC Client (`/lib/rpc.ts`):**

- Install Mix TypeScript SDK package
- Create SDK client instance configuration
- Update backend URL configuration in `/utils/backendUrl.ts`
- Add SDK error handling patterns

**2. Update Cache Management (`/lib/session-cache.ts`):**

- Update cache key structure for REST endpoints
- Modify cache invalidation utilities at lines 15-45
- Update cache key definitions in `/lib/cache-keys.ts`

**3. Update Error Handling:**

- Replace RPC error format with SDK error handling
- Update error types and message extraction
- Update error handling patterns across all hooks

## Implementation Phases

### Phase 1: Foundation

- Install Mix TypeScript SDK package
- Configure SDK client with proper server URL and error handling
- Create SDK wrapper utilities for common patterns
- Update development environment and dependencies

### Phase 2: Core Sessions

- Migrate `useSession.ts` - Basic session CRUD
- Migrate `useForkSession.ts` - Session forking
- Migrate `useMCPList.ts` - Simple system endpoint
- Test core session functionality

### Phase 3: Stateless Migration

- Remove global session state from `useSessionsList.ts`
- Update all session selection usage to explicit sessionId parameters
- Update components that depend on current session concept
- **BREAKING**: Remove `useSelectSession` hook entirely

### Phase 4: Messages

- Migrate `useMessages.ts` - Message sending
- Migrate `useSessionMessages.ts` - Message loading
- Migrate `useMessageHistory.ts` - History with pagination
- Test message history navigation integration

### Phase 5: Real-time Integration

- Migrate `usePersistentSSE.ts` - Complex streaming integration
- Update permission handling workflow
- Test SSE + REST API integration
- Verify tool execution and streaming functionality

### Phase 6: Cleanup & Testing

- Remove legacy RPC client code
- Update error handling consistency
- Comprehensive testing of all migrated functionality
- Performance testing and optimization

## Technical Implementation Details

### SDK Configuration

- Configure SDK client with proper server URL (reference `/utils/backendUrl.ts`)
- Set up retry configuration as needed
- Configure error handling patterns
- Reference SDK documentation in `mix-typescript-sdk/README.md:335-353` for configuration options

### Error Handling Migration

- Migrate from RPC error format to SDK error handling
- Update error types and message extraction patterns
- Reference SDK error handling documentation in `mix-typescript-sdk/README.md:257-327`
- Import error classes from `mix-typescript-sdk/models/errors`

### Cache Key Migration

- Update cache keys in `/lib/cache-keys.ts`
- Modify cache invalidation utilities in `/lib/session-cache.ts`
- Update cache key structure from RPC-based to REST-based patterns

## Breaking Changes & Migration Impact

### High Impact Breaking Changes

#### 1. **Stateless Session Management**

**Impact:** Complete removal of "current session" concept
**Changes Required:**

- Remove `useSelectSession()` hook entirely
- Update all components to pass explicit `sessionId` parameters
- Remove global session state from components
- Update routing/navigation to include session context

**Components Affected:**

- Session navigation components
- Message input components  
- Tool execution displays
- Any components assuming "current session" context

#### 2. **Parameter Structure Changes**

**Impact:** Method signatures change for some operations
**Files Affected:**

- `useMessages.ts` (line 14) - message sending parameter structure
- Check SDK documentation for exact parameter formats

#### 3. **Session Fork Changes**

**Impact:** Parameter structure changes
**Files Affected:**

- `useForkSession.ts` (line 12) - fork operation parameters
- Verify SDK method signature against current RPC parameters

#### 3. **Response Format Changes**

**Impact:** Data extraction patterns change
**Changes Required:**

- SDK automatically unwraps `response.data`
- Remove manual response unwrapping
- Update TypeScript types for new response format

### Medium Impact Changes

#### 1. **Cache Management**

- Cache key updates
- Invalidation pattern adjustments
- Response format handling

#### 2. **Error Handling**

- New error types and structure
- Error message extraction changes
- Error propagation updates

#### 3. **SSE Integration**

- Endpoint URL changes
- Permission handling integration
- Mixed REST/SSE communication patterns

### Low Impact Changes

#### 1. **Simple RPC Replacements**

- Direct method swaps
- Minimal logic changes
- Straightforward testing

## Success Criteria

### Functional Requirements

- [ ] All 10 RPC-dependent hooks successfully migrated
- [ ] All existing functionality preserved
- [ ] Real-time streaming continues to work
- [ ] Message history and pagination work correctly
- [ ] Session management operates without global state
- [ ] Error handling maintains user experience

### Technical Requirements

- [ ] Zero RPC calls remaining in hook code
- [ ] All TanStack Query caches work correctly
- [ ] TypeScript types are properly updated
- [ ] Error handling follows consistent patterns
- [ ] Performance matches or improves over RPC implementation

### Quality Requirements

- [ ] Comprehensive test coverage for all migrated hooks
- [ ] No regression in user experience
- [ ] Clean removal of legacy RPC infrastructure
- [ ] Consistent code patterns across all hooks

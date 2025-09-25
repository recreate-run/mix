# True Concurrency Architecture Plan v2.0

## Executive Summary

Remove artificial session-wide blocking while preserving essential session isolation. Enable maximum concurrency by only serializing operations that actually have logical dependencies.

## Problem Analysis

**Root Issue**: `activeRequests.LoadOrStore()` in `mix_agent/internal/llm/agent/agent.go:248-250` blocks ALL operations per session when 95% of operations have no conflicts and can run concurrently.

**Session Resources Are Already Isolated**:

- File operations: Isolated to `/storage/{session-id}/`
- Shell processes: Separate per session with own working directories
- Conversation context: Properly scoped per session
- Cross-session operations: Already fully parallel

## New Architecture: Per-Session Message Queuing

### **Core Principle**

Replace **session-wide blocking** with **per-session message queuing** + **dependency-aware execution**.

**Message Queuing Scope**: Each session has its own independent message queue. Cross-session messages process in full parallel with zero interaction between sessions.

```go
// REMOVE: Session-wide blocking
activeRequests sync.Map  // DELETE ENTIRELY

// ADD: Per-session message processing
type SessionManager struct {
    sessions sync.Map // sessionID → *SessionContext
}

type SessionContext struct {
    MessageQueue    chan MessageRequest  // ONE queue per session
    ActiveExecution sync.WaitGroup       // Track concurrent operations
}
```

### **Session State Management (Keep Unchanged)**

```go
type SessionContext struct {
    // Essential state (PRESERVE)
    SessionID       string
    WorkingDir      string
    ShellProcess    *exec.Cmd
    ConversationCtx context.Context
    LLMProviders    map[string]Provider

    // NEW: Concurrency management
    MessageQueue    chan MessageRequest
    ActiveExecution sync.WaitGroup
}
```

## Concurrency Matrix

### **Operations That Can Run Concurrently**

#### **Across Sessions (Already Works)**

- ✅ All operations (complete isolation)

#### **Within Same Session (NEW)**

- ✅ **Multiple file reads** (`storage/{session}/file1.txt`, `storage/{session}/file2.txt`)
- ✅ **Independent file writes** (different files, no read-after-write dependencies)
- ✅ **Search operations** (grep, glob, find)
- ✅ **Web fetches/searches** (stateless HTTP requests)
- ✅ **Database reads** (no session-specific state)
- ✅ **LLM API calls** (for independent purposes)
- ✅ **Shell commands** (separate processes can run in parallel)
- ✅ **Independent tool calls** within same message execution

#### **Example Concurrent Execution Within Session**

```
User Message: "Analyze these 5 files and search for patterns"
├── File read: file1.txt     } Can run in parallel
├── File read: file2.txt     }
├── File read: file3.txt     }
├── Grep search: pattern1    }
└── Grep search: pattern2    }
```

### **Operations That Must Be Serialized**

#### **Message Processing Order**

```
User Message A → [Process completely] → User Message B
```

**Why**: User expects messages processed in sequence for conversation flow.

#### **Tool Dependency Chains Within Message**

```
Tool Call 1: Read file content
Tool Call 2: Modify file based on content from Tool Call 1  ← Depends on Tool Call 1
```

#### **Shell State Dependencies**

```
Tool Call 1: cd /some/directory
Tool Call 2: ls  ← Depends on directory change from Tool Call 1
```

### **Operations That DON'T Conflict (Corrected Understanding)**

#### **File Operations**

- ❌ **OLD**: "File writes conflict within session"
- ✅ **NEW**: File writes to different files can run concurrently
- ⚠️ **ONLY SERIALIZE**: Read-after-write to same file within same message

#### **Shell Commands**

- ❌ **OLD**: "Shell commands conflict within session"
- ✅ **NEW**: Independent shell commands can run in parallel processes
- ⚠️ **ONLY SERIALIZE**: Commands that depend on previous shell state changes

## Implementation Plan

### **Phase 1: Remove Session-Wide Blocking**

1. **Delete entirely**:

   ```go
   // REMOVE from agent.go
   activeRequests sync.Map
   ErrSessionBusy error
   IsSessionBusy() method
   IsBusy() method
   ```

2. **Remove blocking logic**:
   - `mix_agent/internal/llm/agent/agent.go:248-250`
   - `mix_agent/internal/http/rest_messages.go` - ErrSessionBusy handling
   - `mix_agent/internal/http/sse.go` - "Failed to start agent" responses

### **Phase 2: Implement Message-Level Queuing**

1. **Add to session context**:

   ```go
   type SessionManager struct {
       sessions sync.Map // sessionID → *SessionContext
   }

   type SessionContext struct {
       MessageQueue chan MessageRequest
       cancel       context.CancelFunc
   }
   ```

2. **Message processing**:
   - Queue user messages per session
   - Process messages sequentially
   - Allow concurrent tool execution within message

### **Phase 3: Dependency-Aware Tool Execution**

1. **Tool dependency analysis**:
   - Identify tool calls that can run in parallel
   - Execute independent tools concurrently
   - Serialize only dependent chains

2. **Context management**:
   - Request-level contexts for cancellation
   - Session-level context for resource management

### **Phase 4: Remove Legacy Components**

1. **Agent factory complexity**:
   - Simplify to stateless request handlers with session context
   - Remove agent caching/cleanup mechanisms
   - Eliminate race conditions in agent creation

## How This Architecture Eliminates Previous Issues

### **Session Cancellation**
- Message-level cancellation with request contexts
- Individual message processing can be cancelled without affecting session

### **Agent Creation Race Conditions**
- No agent caching - direct session context lookup via SessionManager
- Session context lifecycle managed atomically

### **Memory Leaks**
- No long-lived agents to accumulate in memory
- Sessions cleaned up on explicit close or TTL with stateless request processing

### **Context Management Complexity**
- Clear separation: Session context (long-lived resources) vs Request context (short-lived operations)
- Simplified hierarchy eliminates complex context juggling

## Expected Performance Gains

### **Throughput Improvements**

- **Cross-session**: Already optimal (no change)
- **Within-session**: 5-10x improvement for independent operations
- **Tool execution**: Parallel processing where dependencies allow

### **User Experience**

- ❌ No more "session busy" errors ever
- ✅ Multiple file operations execute simultaneously
- ✅ Search operations don't block file operations
- ✅ Web fetches don't block local operations

### **Resource Utilization**

- CPU: Better utilization of multi-core systems
- I/O: Parallel file and network operations
- Memory: Simplified architecture, reduced overhead

## Testing Strategy

### **Concurrency Tests**

```bash
# Test 1: Concurrent file operations within session
curl -X POST /session/123/message -d '{"query": "read file1.txt, file2.txt, and file3.txt"}'
# Should: All file reads execute in parallel

# Test 2: Independent tool calls
curl -X POST /session/123/message -d '{"query": "search for X and grep for Y"}'
# Should: Search and grep execute concurrently

# Test 3: Cross-session parallelism (already works)
curl -X POST /session/123/message & curl -X POST /session/456/message
# Should: Both sessions process simultaneously
```

### **Dependency Tests**

```bash
# Test 4: Tool dependency chains
curl -X POST /session/123/message -d '{"query": "cd /tmp && ls"}'
# Should: cd completes before ls executes

# Test 5: Message ordering
curl -X POST /session/123/message -d '{"query": "create file.txt"}' &
curl -X POST /session/123/message -d '{"query": "read file.txt"}'
# Should: First message completes before second starts
```

### **Stress Tests**

- 100 concurrent sessions with rapid message sending
- Multiple large file operations per session
- Complex tool dependency chains
- Memory usage monitoring over time

## Implementation Files

### **Core Changes**

- `mix_agent/internal/app/app.go` - Replace agent factory with session manager
- `mix_agent/internal/llm/agent/agent.go` - Remove session blocking, add message queuing
- `mix_agent/internal/http/rest_messages.go` - Remove ErrSessionBusy handling
- `mix_agent/internal/http/sse.go` - Remove blocking logic

### **New Components**

- `mix_agent/internal/session/manager.go` - Session context management
- `mix_agent/internal/concurrency/executor.go` - Dependency-aware tool execution

### **Remove Completely**

- All `ErrSessionBusy` error handling
- `activeRequests` sync.Map and related methods
- Agent creation/cleanup race condition workarounds
- Complex context hierarchies

## Success Metrics

### **Functional**

- ✅ Zero "session busy" errors
- ✅ Message ordering preserved per session
- ✅ Tool dependencies respected
- ✅ Session isolation maintained

### **Performance**

- ✅ 5-10x faster for concurrent operations within sessions
- ✅ Linear scaling with concurrent sessions
- ✅ Reduced memory usage (simpler architecture)
- ✅ Better CPU/I/O utilization

## Migration Strategy

1. **Feature flag**: Toggle between old and new concurrency models
2. **Gradual rollout**: Test with subset of sessions first
3. **Monitoring**: Track error rates, performance metrics
4. **Rollback plan**: Instant revert to old model if issues arise

This architecture achieves true concurrency while maintaining all essential functionality through proper dependency analysis rather than blanket blocking.

## Concurrency Integration Test Plan

### **Test Overview**

Create a minimal but comprehensive integration test to validate the concurrency architecture implementation. The test will follow existing patterns from `mix_agent/internal/http/integration_tests/` while focusing specifically on concurrent operations.

### **Test File Structure**

**Location**: `mix_agent/internal/http/integration_tests/rest_concurrency_integration_test.go`

**Test Functions**:
```go
func TestConcurrentToolExecutionWithinSession(t *testing.T)
func TestConcurrentToolExecutionAcrossSessions(t *testing.T)
func TestConcurrentSessionOperations(t *testing.T)
func TestConcurrentMessageProcessing(t *testing.T)
func TestSessionIsolationUnderLoad(t *testing.T)
```

### **Test Scenarios**

#### **Test 1: Concurrent Tool Execution Within Session**
```go
// Validates that multiple tool calls in the same message execute in parallel
// Expected behavior: Tools run concurrently, total time < sum of individual times

sessionID := createTestSession()
start := time.Now()

// Send message requesting multiple independent tools
message := map[string]interface{}{
    "text": "Read file1.txt, search for 'test' in file2.txt, and list *.go files",
}

response := makeJSONRequest(t, server, "POST",
    "/api/sessions/"+sessionID+"/messages", message)

duration := time.Since(start)
validateToolConcurrency(t, response, duration)
```

#### **Test 2: Concurrent Tool Execution Across Sessions**
```go
// Validates that different sessions can execute tools simultaneously
// Expected behavior: No session blocking, both complete successfully

var wg sync.WaitGroup
results := make(chan TestResult, 2)

// Session 1: File operations
wg.Add(1)
go func() {
    defer wg.Done()
    session1 := createTestSession()
    result := executeToolsInSession(session1, "Read large-file1.txt")
    results <- TestResult{SessionID: session1, Duration: result.Duration}
}()

// Session 2: Search operations
wg.Add(1)
go func() {
    defer wg.Done()
    session2 := createTestSession()
    result := executeToolsInSession(session2, "Search for patterns in codebase")
    results <- TestResult{SessionID: session2, Duration: result.Duration}
}()

wg.Wait()
validateNoCrossSessionBlocking(t, results)
```

#### **Test 3: Concurrent Session Operations**
```go
// Validates that session management operations are thread-safe
// Expected behavior: No race conditions in session creation/deletion

var wg sync.WaitGroup
sessionIDs := make(chan string, 10)

// Concurrent session creation
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(index int) {
        defer wg.Done()
        sessionID := createTestSessionWithTitle(fmt.Sprintf("Concurrent Test %d", index))
        sessionIDs <- sessionID
    }(i)
}

wg.Wait()
close(sessionIDs)

// Validate all sessions created successfully
validateSessionCreation(t, sessionIDs)
```

#### **Test 4: Session Isolation Under Load**
```go
// Validates that concurrent operations don't cross-contaminate session state
// Expected behavior: Each session maintains isolated state

sessions := createMultipleTestSessions(5)
var wg sync.WaitGroup

// Each session performs unique operations
for i, sessionID := range sessions {
    wg.Add(1)
    go func(sid string, index int) {
        defer wg.Done()

        // Create unique file in session storage
        fileName := fmt.Sprintf("session_%d_data.txt", index)
        content := fmt.Sprintf("Data for session %d", index)

        createFileInSession(sid, fileName, content)
        readResult := readFileFromSession(sid, fileName)

        // Validate isolation
        if readResult != content {
            t.Errorf("Session isolation failed: expected %s, got %s", content, readResult)
        }
    }(sessionID, i)
}

wg.Wait()
validateSessionIsolation(t, sessions)
```

### **Test Implementation Utilities**

#### **Helper Functions**
```go
// Follows existing integration test patterns
func createTestSessionForConcurrency() string {
    return createTestSession("Concurrency Test Session")
}

func executeParallelRequests(sessions []string, requests []TestRequest) []TestResponse {
    var wg sync.WaitGroup
    results := make([]TestResponse, len(requests))

    for i, req := range requests {
        wg.Add(1)
        go func(index int, request TestRequest) {
            defer wg.Done()
            results[index] = executeRequest(request)
        }(i, req)
    }

    wg.Wait()
    return results
}

func validateConcurrencyMetrics(t *testing.T, responses []TestResponse) {
    // Check for absence of "session busy" errors
    // Validate response times indicate parallel execution
    // Ensure all operations completed successfully
}
```

### **Validation Criteria**

#### **Success Metrics**
- ✅ **Zero Session Busy Errors**: No `ErrSessionBusy` or equivalent blocking
- ✅ **Parallel Execution Evidence**: Total execution time < sum of individual tool times
- ✅ **Session Isolation**: Each session's operations don't affect others
- ✅ **Data Consistency**: All concurrent operations complete with correct results
- ✅ **Resource Cleanup**: No memory leaks or hanging goroutines

#### **Performance Assertions**
```go
func assertParallelExecution(t *testing.T, toolTimes []time.Duration, totalTime time.Duration) {
    sumOfTools := time.Duration(0)
    for _, duration := range toolTimes {
        sumOfTools += duration
    }

    // Allow for some overhead, but should be significantly faster than sequential
    maxAllowedTime := time.Duration(float64(sumOfTools) * 0.7) // 30% faster than sequential

    if totalTime > maxAllowedTime {
        t.Errorf("Expected parallel execution: total %v should be < 70%% of sequential %v",
                totalTime, sumOfTools)
    }
}
```

### **Integration with CI/CD**

#### **Test Execution**
```bash
# Run concurrency tests specifically
go test -v ./mix_agent/internal/http/integration_tests -run TestConcurrent

# Run with race detection
go test -race -v ./mix_agent/internal/http/integration_tests -run TestConcurrent

# Stress testing
go test -v ./mix_agent/internal/http/integration_tests -run TestConcurrent -count=10
```

#### **Performance Benchmarking**
```go
func BenchmarkConcurrentToolExecution(b *testing.B) {
    server := setupIntegrationTestServer(b)
    defer server.Close()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sessionID := createTestSession()
        executeConcurrentTools(sessionID, standardToolSet)
    }
}
```

This integration test plan provides comprehensive validation of the concurrency architecture while following established patterns from the existing test suite. The tests are designed to be minimal yet thorough, focusing on the core concurrency benefits while ensuring system stability.

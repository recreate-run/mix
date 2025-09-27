# Mix Agent Test Coverage Improvement Plan

## Current State Assessment

Current mix_agent test coverage is extremely limited, with most packages showing 0% coverage. Only the integration tests in `mix/internal/http/integration_tests` have decent coverage (73.3%), while the `mix/internal/llm/agent` package has minimal coverage (9.2%).

Key issues:
1. Most core functionality lacks unit tests
2. Critical flows like authentication, credentials management, and user preferences have no tests
3. Reliance on integration tests without proper unit test coverage

## Test Coverage Goals

| Milestone | Overall Coverage Target | Timeline |
|-----------|-------------------------|----------|
| 1         | 25%                     | 2 weeks  |
| 2         | 50%                     | 4 weeks  |
| 3         | 75%                     | 8 weeks  |
| Final     | 85%+                    | 12 weeks |

## Priority Areas

Based on analysis of the codebase, these are the high-priority areas requiring test coverage, ordered by importance:

1. **Authentication & Credentials Management** (0% current)
   - API key management
   - OAuth authentication flows
   - Credential storage and encryption

2. **User Preferences** (0% current)
   - Preference storage
   - Model settings
   - Provider selection

3. **Agent Core Functionality** (9.2% current)
   - Message handling
   - Context management
   - Tool execution

4. **HTTP API Endpoints** (partial coverage via integration tests)
   - REST API handlers
   - SSE streaming
   - Error handling

5. **Database Operations** (0% current)
   - Session persistence
   - Message storage
   - Preference management

## Detailed Implementation Plan

### Milestone 1 (0% → 25%)

#### Authentication & Credentials

1. **API Key Management Tests**
   - Test file: `credentials/service_test.go`
   - Test cases:
     - Store and retrieve API keys
     - Validate API key formats for different providers
     - Test encryption/decryption
     - Test cache operations
     - Test error handling for invalid keys

2. **OAuth Credential Tests**
   - Test file: `credentials/oauth_test.go`
   - Test cases:
     - Store and retrieve OAuth credentials
     - Test token expiration logic
     - Test refresh token operations
     - Test provider-specific handling

#### User Preferences

1. **Preferences Service Tests**
   - Test file: `preferences/service_test.go`
   - Test cases:
     - Create default preferences
     - Get/update preferences
     - Test caching functionality
     - Test agent config retrieval
     - Test provider preference handling

### Milestone 2 (25% → 50%)

#### Agent Core Functionality

1. **Message Handling Tests**
   - Test file: `internal/llm/agent/message_handler_test.go`
   - Test cases:
     - Message creation and retrieval
     - Content streaming
     - Attachment handling
     - Error scenarios

2. **Context Management Tests**
   - Test file: `internal/llm/agent/context_test.go`
   - Test cases:
     - Context creation
     - Context updates
     - History management
     - Token counting and truncation

3. **Tool Execution Tests**
   - Test file: `internal/llm/tools/tools_test.go`
   - Test cases:
     - Tool registration and discovery
     - Parameter validation
     - Execution flow
     - Error handling

### Milestone 3 (50% → 75%)

#### HTTP API Endpoints

1. **REST API Handler Tests**
   - Test files:
     - `internal/http/handler_sessions_test.go`
     - `internal/http/handler_messages_test.go`
     - `internal/http/handler_files_test.go`
   - Test cases:
     - Request validation
     - Response formatting
     - Error responses
     - Authentication middleware

2. **SSE Streaming Tests**
   - Test file: `internal/http/stream_handler_test.go`
   - Test cases:
     - Connection establishment
     - Event streaming
     - Client disconnection
     - Error propagation

#### Database Operations

1. **Session Persistence Tests**
   - Test file: `internal/session/service_test.go`
   - Test cases:
     - Session creation
     - Session retrieval
     - Session updates
     - Session deletion

2. **Message Storage Tests**
   - Test file: `internal/message/service_test.go`
   - Test cases:
     - Message saving
     - Message retrieval
     - Message updating
     - Message querying

### Final Milestone (75% → 85%+)

1. **Edge Cases and Error Handling**
   - Comprehensive tests for error paths
   - Recovery mechanisms
   - Timeout handling
   - Concurrency issues

2. **Performance Tests**
   - Response times
   - Resource usage
   - Concurrency handling
   - Cache effectiveness

## Testing Infrastructure Requirements

1. **Mock Framework**
   - Use `testify/mock` for interface mocking
   - Create mock implementations for external dependencies

2. **Test Database**
   - Use SQLite in-memory database for tests
   - Implement test fixtures for common scenarios

3. **Test HTTP Server**
   - Use `httptest` package for HTTP handler testing
   - Create test utilities for common HTTP operations

4. **Test Environment**
   - Create a test environment configuration
   - Use environment variables for test settings

## Test Development Guidelines

1. **Test Structure**
   - Use table-driven tests for multiple test cases
   - Use subtests for related test scenarios
   - Use descriptive test names

2. **Mocking Strategy**
   - Mock external dependencies (database, HTTP clients, etc.)
   - Use interface-based design for testability
   - Avoid mocking concrete types when possible

3. **Coverage Goals**
   - Aim for 100% coverage of critical paths
   - Prioritize testing error handling
   - Use coverage reports to identify untested code

4. **Test Documentation**
   - Document test assumptions
   - Document test fixtures
   - Document test setup/teardown

## Implementation Steps

1. **Setup Testing Infrastructure**
   - Create mock implementations for core interfaces
   - Set up test database
   - Create test utilities

2. **Implement High-Priority Tests**
   - Start with authentication and credentials
   - Move to preferences and agent core
   - Then address HTTP API and database operations

3. **Establish CI Integration**
   - Add test coverage reporting to CI
   - Set up test failure notifications
   - Configure test timeouts

4. **Documentation and Knowledge Sharing**
   - Document testing approach
   - Share testing best practices
   - Train team on testing methodology

## Next Steps

1. Begin with creating mock implementations for core interfaces
2. Implement the first test file for credentials service
3. Configure test coverage reporting
4. Establish regular testing progress reviews
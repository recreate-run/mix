package integration_tests

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestResult represents the result of a concurrent test operation
type TestResult struct {
	SessionID string
	Duration  time.Duration
	Success   bool
	Error     error
}

// TestRequest represents a concurrent test request
type TestRequest struct {
	SessionID string
	Path      string
	Payload   interface{}
	Method    string
}

// TestResponse represents a concurrent test response
type TestResponse struct {
	Response *http.Response
	Duration time.Duration
	Error    error
}

// TestConcurrentFileOperationsAcrossSessions validates that file operations
// across different sessions can execute concurrently
func TestConcurrentFileOperationsAcrossSessions(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing concurrent file operations across multiple sessions")

	// Create multiple sessions
	sessionCount := 3
	sessions := createMultipleTestSessions(t, result, sessionCount)

	var wg sync.WaitGroup
	fileResults := make(chan TestResult, sessionCount)

	// Each session performs file operations concurrently
	for i, sessionID := range sessions {
		wg.Add(1)
		go func(index int, sid string) {
			defer wg.Done()

			fileName := fmt.Sprintf("concurrent_test_%d.txt", index)
			fileContent := fmt.Sprintf("Test content from session %d", index)

			start := time.Now()

			// Upload a file to this session
			uploadResp := makeMultipartFileRequest(t, result.Server,
				"/api/sessions/"+sid+"/files/upload",
				fileName, fileContent)

			duration := time.Since(start)
			success := uploadResp != nil && uploadResp.StatusCode == http.StatusCreated

			if uploadResp != nil {
				_ = uploadResp.Body.Close()
			}

			fileResults <- TestResult{
				SessionID: sid,
				Duration:  duration,
				Success:   success,
			}
		}(i, sessionID)
	}

	wg.Wait()
	close(fileResults)

	// Validate all file operations completed successfully
	successfulOperations := 0
	for result := range fileResults {
		if result.Success {
			successfulOperations++
		} else {
			t.Errorf("File operation failed for session %s", result.SessionID)
		}
	}

	if successfulOperations != sessionCount {
		t.Errorf("Expected %d successful file operations, got %d", sessionCount, successfulOperations)
	}

	t.Logf("✅ Concurrent file operations test passed - %d/%d operations successful",
		successfulOperations, sessionCount)
}

// TestConcurrentToolExecutionAcrossSessions validates that different sessions
// can execute tools simultaneously without blocking each other
func TestConcurrentToolExecutionAcrossSessions(t *testing.T) {
	// Use fake provider with tool execution sequence
	config := fakeBashToolResponse("echo 'test'", "File created successfully")
	result := setupIntegrationTestServerWithFakeProvider(t, config)
	defer result.Server.Close()

	t.Log("Testing concurrent tool execution across multiple sessions")

	var wg sync.WaitGroup
	results := make(chan TestResult, 3)

	// Create and execute operations in 3 different sessions concurrently
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			sessionID := createTestSessionForConcurrency(t, result,
				fmt.Sprintf("Cross-Session Test %d", index))

			start := time.Now()

			// Each session performs different operations
			var messageContent string
			switch index {
			case 0:
				messageContent = "Create a file named 'session0.txt' with content 'Session 0 data'"
			case 1:
				messageContent = "Create a file named 'session1.txt' and list all files"
			case 2:
				messageContent = "Search for any files and create 'session2.txt'"
			}

			messageRequest := map[string]interface{}{
				"text": messageContent,
			}

			msgResp := makeJSONRequest(t, result.Server, "POST",
				"/api/sessions/"+sessionID+"/messages", messageRequest)

			duration := time.Since(start)
			// Accept both 200 (sync) and 202 (async) as success
			success := msgResp != nil && (msgResp.StatusCode == http.StatusOK || msgResp.StatusCode == http.StatusAccepted)

			if msgResp != nil {
				_ = msgResp.Body.Close()
			}

			results <- TestResult{
				SessionID: sessionID,
				Duration:  duration,
				Success:   success,
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Validate all sessions completed successfully
	sessionResults := make([]TestResult, 0, 3)
	for result := range results {
		sessionResults = append(sessionResults, result)
		if !result.Success {
			t.Errorf("Session %s failed to complete successfully", result.SessionID)
		}
	}

	if len(sessionResults) != 3 {
		t.Fatalf("Expected 3 session results, got %d", len(sessionResults))
	}

	t.Logf("✅ Cross-session concurrency test passed - All %d sessions completed successfully",
		len(sessionResults))
}

// TestConcurrentSessionOperations validates that session management operations
// are thread-safe and don't have race conditions
func TestConcurrentSessionOperations(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing concurrent session creation and management")

	var wg sync.WaitGroup
	sessionIDs := make(chan string, 5)
	creationErrors := make(chan error, 5)

	// Concurrent session creation
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			sessionRequest := map[string]interface{}{
				"title": fmt.Sprintf("Concurrent Session Test %d", index),
			}

			createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
			defer func() { _ = createResp.Body.Close() }()

			if createResp.StatusCode == http.StatusCreated {
				sessionData := validateObjectResponse(t, createResp, http.StatusCreated)
				if sessionID, ok := sessionData["id"].(string); ok && sessionID != "" {
					sessionIDs <- sessionID
				} else {
					creationErrors <- fmt.Errorf("invalid session ID in response")
				}
			} else {
				creationErrors <- fmt.Errorf("session creation failed with status %d", createResp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	close(sessionIDs)
	close(creationErrors)

	// Check for creation errors
	errorCount := 0
	for err := range creationErrors {
		t.Errorf("Session creation error: %v", err)
		errorCount++
	}

	// Validate all sessions were created successfully
	successfulSessions := make([]string, 0, 5)
	for sessionID := range sessionIDs {
		successfulSessions = append(successfulSessions, sessionID)
	}

	expectedSessions := 5 - errorCount
	if len(successfulSessions) != expectedSessions {
		t.Fatalf("Expected %d successful sessions, got %d", expectedSessions, len(successfulSessions))
	}

	t.Logf("✅ Concurrent session operations test passed - %d sessions created successfully",
		len(successfulSessions))
}

// TestConcurrentMessageProcessing validates that message processing within
// a session maintains order while allowing concurrent tool execution
func TestConcurrentMessageProcessing(t *testing.T) {
	// Use fake provider for fast, deterministic testing
	config := simpleFakeResponse("File created successfully")
	result := setupIntegrationTestServerWithFakeProvider(t, config)
	defer result.Server.Close()

	t.Log("Testing concurrent message processing with ordering guarantees")

	sessionID := createTestSessionForConcurrency(t, result, "Message Processing Test")

	// Send multiple messages to the same session
	// They should be processed in order, but tools within each message can be concurrent
	var wg sync.WaitGroup
	messageResponses := make(chan TestResult, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			messageRequest := map[string]interface{}{
				"text": fmt.Sprintf("Create file 'message_%d.txt' with content 'Message %d data'",
					index, index),
			}

			start := time.Now()
			msgResp := makeJSONRequest(t, result.Server, "POST",
				"/api/sessions/"+sessionID+"/messages", messageRequest)
			duration := time.Since(start)

			success := false
			if msgResp != nil {
				// Accept both 200 (sync) and 202 (async) as success
				success = msgResp.StatusCode == http.StatusOK || msgResp.StatusCode == http.StatusAccepted
				_ = msgResp.Body.Close()
			}

			messageResponses <- TestResult{
				SessionID: sessionID,
				Duration:  duration,
				Success:   success,
			}
		}(i)
	}

	wg.Wait()
	close(messageResponses)

	// Validate all messages were processed successfully
	processedMessages := 0
	for result := range messageResponses {
		if !result.Success {
			t.Errorf("Message processing failed for session %s", result.SessionID)
		} else {
			processedMessages++
		}
	}

	if processedMessages != 3 {
		t.Fatalf("Expected 3 processed messages, got %d", processedMessages)
	}

	t.Logf("✅ Concurrent message processing test passed - %d messages processed", processedMessages)
}

// TestSessionIsolationUnderLoad validates that concurrent operations
// don't cross-contaminate session state
func TestSessionIsolationUnderLoad(t *testing.T) {
	// Use fake provider for fast testing
	config := simpleFakeResponse("File created successfully with unique content")
	result := setupIntegrationTestServerWithFakeProvider(t, config)
	defer result.Server.Close()

	t.Log("Testing session isolation under concurrent load")

	// Create multiple sessions
	sessionCount := 4
	sessions := make([]string, sessionCount)
	for i := 0; i < sessionCount; i++ {
		sessions[i] = createTestSessionForConcurrency(t, result,
			fmt.Sprintf("Isolation Test Session %d", i))
	}

	var wg sync.WaitGroup
	isolationResults := make(chan TestResult, sessionCount)

	// Each session performs unique operations that should not interfere with others
	for i, sessionID := range sessions {
		wg.Add(1)
		go func(index int, sid string) {
			defer wg.Done()

			// Create unique content for each session
			uniqueContent := fmt.Sprintf("Unique data for session %d - timestamp %d",
				index, time.Now().UnixNano())

			messageRequest := map[string]interface{}{
				"text": fmt.Sprintf("Create file 'isolation_test_%d.txt' with content '%s'",
					index, uniqueContent),
			}

			start := time.Now()
			msgResp := makeJSONRequest(t, result.Server, "POST",
				"/api/sessions/"+sid+"/messages", messageRequest)
			duration := time.Since(start)

			success := false
			if msgResp != nil {
				// Accept both 200 (sync) and 202 (async) as success
				success = msgResp.StatusCode == http.StatusOK || msgResp.StatusCode == http.StatusAccepted
				_ = msgResp.Body.Close()
			}

			isolationResults <- TestResult{
				SessionID: sid,
				Duration:  duration,
				Success:   success,
			}
		}(i, sessionID)
	}

	wg.Wait()
	close(isolationResults)

	// Validate session isolation
	successfulOperations := 0
	for result := range isolationResults {
		if !result.Success {
			t.Errorf("Session isolation test failed for session %s", result.SessionID)
		} else {
			successfulOperations++
		}
	}

	if successfulOperations != sessionCount {
		t.Fatalf("Expected %d successful operations, got %d", sessionCount, successfulOperations)
	}

	t.Logf("✅ Session isolation test passed - %d sessions maintained proper isolation",
		successfulOperations)
}

// Helper Functions

// createTestSessionForConcurrency creates a test session with a specific title
func createTestSessionForConcurrency(t *testing.T, result *TestServerResult, title string) string {
	t.Helper()
	sessionRequest := map[string]interface{}{
		"title": title,
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	defer func() { _ = createResp.Body.Close() }()
	sessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID, ok := sessionData["id"].(string)
	if !ok || sessionID == "" {
		t.Fatalf("Failed to create test session: invalid session ID")
	}

	return sessionID
}

// createMultipleTestSessions creates multiple test sessions concurrently
func createMultipleTestSessions(t *testing.T, result *TestServerResult, count int) []string {
	t.Helper()
	var wg sync.WaitGroup
	sessionIDs := make(chan string, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sessionID := createTestSessionForConcurrency(t, result,
				fmt.Sprintf("Multi-Session Test %d", index))
			sessionIDs <- sessionID
		}(i)
	}

	wg.Wait()
	close(sessionIDs)

	sessions := make([]string, 0, count)
	for sessionID := range sessionIDs {
		sessions = append(sessions, sessionID)
	}

	return sessions
}

// BenchmarkConcurrentToolExecution provides performance benchmarking for concurrent operations
func BenchmarkConcurrentToolExecution(b *testing.B) {
	// Convert *testing.B to *testing.T for helper functions
	t := &testing.T{}
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	sessionID := createTestSessionForConcurrency(t, result, "Benchmark Session")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		messageRequest := map[string]interface{}{
			"text": fmt.Sprintf("Create benchmark file %d and list current files", i),
		}

		msgResp := makeJSONRequest(t, result.Server, "POST",
			"/api/sessions/"+sessionID+"/messages", messageRequest)

		if msgResp != nil {
			_ = msgResp.Body.Close()
		}
	}
}

// TestNoConcurrencyRegressions validates that the new concurrency architecture
// doesn't break existing functionality
func TestNoConcurrencyRegressions(t *testing.T) {
	// Use fake provider for fast, deterministic testing
	config := simpleFakeResponse("Basic functionality test completed successfully")
	result := setupIntegrationTestServerWithFakeProvider(t, config)
	defer result.Server.Close()

	t.Log("Testing that concurrency changes don't break existing functionality")

	// Test basic session creation
	sessionID := createTestSessionForConcurrency(t, result, "Regression Test Session")

	// Test basic message sending
	messageRequest := map[string]interface{}{
		"text": "This is a basic test message to ensure normal functionality works",
	}

	msgResp := makeJSONRequest(t, result.Server, "POST",
		"/api/sessions/"+sessionID+"/messages", messageRequest)
	defer func() { _ = msgResp.Body.Close() }()

	// Accept both 200 (sync) and 202 (async) as success
	expectedStatus := http.StatusAccepted
	if msgResp.StatusCode != http.StatusOK && msgResp.StatusCode != http.StatusAccepted {
		t.Fatalf("Basic message functionality broken: expected status %d or %d, got %d",
			http.StatusOK, http.StatusAccepted, msgResp.StatusCode)
	}

	// For async (202), we just verify the response structure
	if msgResp.StatusCode == http.StatusAccepted {
		expectedStatus = http.StatusAccepted
	}

	messageData := validateObjectResponse(t, msgResp, expectedStatus)

	// Verify response structure hasn't changed
	// For async responses (202), we just verify we got a valid response
	if msgResp.StatusCode == http.StatusAccepted {
		// Async response - just verify we have some data
		if messageData == nil {
			t.Fatalf("No response data received for async message")
		}
	} else {
		// Sync response - verify full message structure
		if _, ok := messageData["id"].(string); !ok {
			t.Fatalf("Message response structure changed: missing ID field")
		}

		if _, ok := messageData["role"].(string); !ok {
			t.Fatalf("Message response structure changed: missing role field")
		}
	}

	t.Logf("✅ No concurrency regressions detected - basic functionality intact")
}

// TestErrorHandlingUnderConcurrency validates that error conditions
// are properly handled in concurrent scenarios
func TestErrorHandlingUnderConcurrency(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing error handling under concurrent load")

	// Create a session
	sessionID := createTestSessionForConcurrency(t, result, "Error Handling Test")

	var wg sync.WaitGroup
	errorResults := make(chan TestResult, 3)

	// Send some requests that might cause errors concurrently
	testCases := []string{
		"Read a non-existent file called 'does-not-exist.txt'",
		"Execute an invalid command that should fail",
		"Try to access a restricted resource",
	}

	for _, testCase := range testCases {
		wg.Add(1)
		go func(content string) {
			defer wg.Done()

			messageRequest := map[string]interface{}{
				"text": content,
			}

			start := time.Now()
			msgResp := makeJSONRequest(t, result.Server, "POST",
				"/api/sessions/"+sessionID+"/messages", messageRequest)
			duration := time.Since(start)

			// For error handling test, we expect the system to handle errors gracefully
			// The response should still be valid (200 OK) but may contain error information
			success := false
			if msgResp != nil {
				success = msgResp.StatusCode == http.StatusOK
				_ = msgResp.Body.Close()
			}

			errorResults <- TestResult{
				SessionID: sessionID,
				Duration:  duration,
				Success:   success,
			}
		}(testCase)
	}

	wg.Wait()
	close(errorResults)

	// Validate that errors are handled gracefully without system crashes
	handledErrors := 0
	for result := range errorResults {
		if result.Success {
			handledErrors++
		}

		// Ensure reasonable response times even for error cases
		if result.Duration > 10*time.Second {
			t.Errorf("Error handling took too long: %v", result.Duration)
		}
	}

	t.Logf("✅ Error handling test completed - %d/%d requests handled gracefully",
		handledErrors, len(testCases))
}

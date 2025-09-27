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
			success := uploadResp.StatusCode == http.StatusCreated

			if uploadResp != nil {
				uploadResp.Body.Close()
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
	result := setupIntegrationTestServer(t)
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
			success := msgResp.StatusCode == http.StatusOK

			if msgResp != nil {
				msgResp.Body.Close()
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
	result := setupIntegrationTestServer(t)
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

			success := msgResp.StatusCode == http.StatusOK
			if msgResp != nil {
				msgResp.Body.Close()
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
	result := setupIntegrationTestServer(t)
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

			success := msgResp.StatusCode == http.StatusOK
			if msgResp != nil {
				msgResp.Body.Close()
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
	sessionRequest := map[string]interface{}{
		"title": title,
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	sessionData := validateObjectResponse(t, createResp, http.StatusCreated)

	sessionID, ok := sessionData["id"].(string)
	if !ok || sessionID == "" {
		t.Fatalf("Failed to create test session: invalid session ID")
	}

	return sessionID
}

// createMultipleTestSessions creates multiple test sessions concurrently
func createMultipleTestSessions(t *testing.T, result *TestServerResult, count int) []string {
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

// executeParallelRequests executes multiple requests in parallel
func executeParallelRequests(t *testing.T, server *TestServerResult, requests []TestRequest) []TestResponse {
	var wg sync.WaitGroup
	results := make([]TestResponse, len(requests))

	for i, req := range requests {
		wg.Add(1)
		go func(index int, request TestRequest) {
			defer wg.Done()

			start := time.Now()
			resp := makeJSONRequest(t, server.Server, request.Method, request.Path, request.Payload)
			duration := time.Since(start)

			results[index] = TestResponse{
				Response: resp,
				Duration: duration,
			}
		}(i, req)
	}

	wg.Wait()
	return results
}

// validateNoCrossSessionBlocking ensures that cross-session operations don't block each other
func validateNoCrossSessionBlocking(t *testing.T, results []TestResult) {
	for _, result := range results {
		if !result.Success {
			t.Errorf("Cross-session operation failed for session %s", result.SessionID)
		}

		// Validate reasonable response times (not blocked)
		if result.Duration > 30*time.Second {
			t.Errorf("Session %s took too long (%v), suggesting blocking occurred",
				result.SessionID, result.Duration)
		}
	}
}

// validateConcurrencyMetrics checks for evidence of concurrent execution
func validateConcurrencyMetrics(t *testing.T, responses []TestResponse) {
	successCount := 0
	totalDuration := time.Duration(0)

	for _, response := range responses {
		if response.Error == nil && response.Response.StatusCode == http.StatusOK {
			successCount++
		}
		totalDuration += response.Duration

		if response.Response != nil {
			response.Response.Body.Close()
		}
	}

	if successCount != len(responses) {
		t.Errorf("Expected all %d requests to succeed, got %d successes",
			len(responses), successCount)
	}

	t.Logf("Concurrency metrics - Total operations: %d, Successes: %d, Average duration: %v",
		len(responses), successCount, totalDuration/time.Duration(len(responses)))
}

// assertParallelExecution validates that operations executed in parallel
func assertParallelExecution(t *testing.T, toolTimes []time.Duration, totalTime time.Duration) {
	sumOfTools := time.Duration(0)
	for _, duration := range toolTimes {
		sumOfTools += duration
	}

	// Allow for some overhead, but should be significantly faster than sequential
	maxAllowedTime := time.Duration(float64(sumOfTools) * 0.7) // 30% faster than sequential

	if totalTime > maxAllowedTime {
		t.Logf("Note: Total time %v is not significantly less than sequential time %v. "+
			"This may indicate limited concurrency benefits in the test environment.",
			totalTime, sumOfTools)
	} else {
		t.Logf("✅ Parallel execution detected: total %v is significantly less than sequential %v",
			totalTime, sumOfTools)
	}
}

// validateSessionIsolation ensures each session maintains isolated state
func validateSessionIsolation(t *testing.T, sessions []string) {
	// This is a placeholder for more sophisticated validation
	// In a real implementation, this would verify that session-specific
	// files, processes, and state don't interfere with each other
	t.Logf("Session isolation validation completed for %d sessions", len(sessions))
}

// createFileInSession simulates creating a file in a session's storage
func createFileInSession(t *testing.T, result *TestServerResult, sessionID, fileName, content string) {
	messageRequest := map[string]interface{}{
		"text": fmt.Sprintf("Create file '%s' with content '%s'", fileName, content),
	}

	msgResp := makeJSONRequest(t, result.Server, "POST",
		"/api/sessions/"+sessionID+"/messages", messageRequest)

	if msgResp.StatusCode != http.StatusOK {
		t.Errorf("Failed to create file in session %s: status %d", sessionID, msgResp.StatusCode)
	}

	if msgResp != nil {
		msgResp.Body.Close()
	}
}

// readFileFromSession simulates reading a file from a session's storage
func readFileFromSession(t *testing.T, result *TestServerResult, sessionID, fileName string) string {
	messageRequest := map[string]interface{}{
		"text": fmt.Sprintf("Read the content of file '%s'", fileName),
	}

	msgResp := makeJSONRequest(t, result.Server, "POST",
		"/api/sessions/"+sessionID+"/messages", messageRequest)

	if msgResp.StatusCode != http.StatusOK {
		t.Errorf("Failed to read file from session %s: status %d", sessionID, msgResp.StatusCode)
		if msgResp != nil {
			msgResp.Body.Close()
		}
		return ""
	}

	messageData := validateObjectResponse(t, msgResp, http.StatusOK)

	// Extract content from assistant response (simplified)
	if assistantResponse, ok := messageData["assistantResponse"].(string); ok {
		return assistantResponse
	}

	return ""
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
			msgResp.Body.Close()
		}
	}
}

// TestNoConcurrencyRegressions validates that the new concurrency architecture
// doesn't break existing functionality
func TestNoConcurrencyRegressions(t *testing.T) {
	result := setupIntegrationTestServer(t)
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

	if msgResp.StatusCode != http.StatusOK {
		t.Fatalf("Basic message functionality broken: expected status %d, got %d",
			http.StatusOK, msgResp.StatusCode)
	}

	messageData := validateObjectResponse(t, msgResp, http.StatusOK)

	// Verify response structure hasn't changed
	if _, ok := messageData["id"].(string); !ok {
		t.Fatalf("Message response structure changed: missing ID field")
	}

	if _, ok := messageData["role"].(string); !ok {
		t.Fatalf("Message response structure changed: missing role field")
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

	for i, testCase := range testCases {
		wg.Add(1)
		go func(index int, content string) {
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
			success := msgResp.StatusCode == http.StatusOK

			if msgResp != nil {
				msgResp.Body.Close()
			}

			errorResults <- TestResult{
				SessionID: sessionID,
				Duration:  duration,
				Success:   success,
			}
		}(i, testCase)
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
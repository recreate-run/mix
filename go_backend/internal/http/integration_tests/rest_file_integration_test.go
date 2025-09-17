package integration_tests

import (
	"net/http"
	"testing"
)

// Test 20: File Upload and List - POST /api/sessions/{id}/files/upload + GET /api/sessions/{id}/files
func TestRESTFileUploadAndList(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file upload and listing functionality")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "File Upload Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Upload a test file
	testContent := "This is test file content for upload testing"
	testFilename := "test-upload.txt"

	uploadResp := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+sessionID+"/files/upload",
		testFilename,
		testContent)

	uploadData := validateObjectResponse(t, uploadResp, http.StatusCreated)

	// Validate upload response
	uploadedFilename, ok := uploadData["name"].(string)
	if !ok || uploadedFilename != testFilename {
		t.Fatalf("Expected uploaded filename '%s', got %v", testFilename, uploadedFilename)
	}

	uploadedSize, ok := uploadData["size"].(float64)
	if !ok || int(uploadedSize) != len(testContent) {
		t.Fatalf("Expected uploaded file size %d, got %v", len(testContent), uploadedSize)
	}

	// List files in the session
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/files", nil)
	filesList := validateArrayResponse(t, listResp, http.StatusOK)

	// Verify the uploaded file appears in the list
	found := false
	for _, fileItem := range filesList {
		fileObj, ok := fileItem.(map[string]interface{})
		if !ok {
			continue
		}
		if fileObj["name"].(string) == testFilename {
			found = true
			// Verify file properties in list
			if int(fileObj["size"].(float64)) != len(testContent) {
				t.Fatalf("Expected file size %d in list, got %v", len(testContent), fileObj["size"])
			}
			break
		}
	}

	if !found {
		t.Fatalf("Uploaded file '%s' not found in session file list", testFilename)
	}

	t.Logf("✅ File upload and list test passed - Uploaded and listed file: %s", testFilename)
}

// Test 21: File Serving - Upload and download file, verify content
func TestRESTFileServing(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file serving functionality")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "File Serving Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Upload a test file with specific content
	testContent := "This is test content for file serving validation.\nIt has multiple lines.\nAnd special characters: !@#$%^&*()"
	testFilename := "test-serve.txt"

	uploadResp := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+sessionID+"/files/upload",
		testFilename,
		testContent)

	validateObjectResponse(t, uploadResp, http.StatusCreated)

	// Download the file
	downloadResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionID+"/files/"+testFilename, nil)

	if downloadResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d for file download, got %d", http.StatusOK, downloadResp.StatusCode)
	}

	// Read the downloaded content
	downloadedContent := make([]byte, len(testContent)+100) // Extra buffer to detect size issues
	n, err := downloadResp.Body.Read(downloadedContent)
	downloadResp.Body.Close()

	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Failed to read downloaded file content: %v", err)
	}

	// Verify content matches exactly
	actualContent := string(downloadedContent[:n])
	if actualContent != testContent {
		t.Fatalf("Downloaded content doesn't match uploaded content.\nExpected: %q\nActual: %q", testContent, actualContent)
	}

	// Verify content type
	contentType := downloadResp.Header.Get("Content-Type")
	if contentType == "" {
		t.Logf("Warning: No Content-Type header set for downloaded file")
	}

	// Test downloading non-existent file (should return 404)
	nonExistentResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionID+"/files/non-existent.txt", nil)

	if nonExistentResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code %d for non-existent file, got %d", http.StatusNotFound, nonExistentResp.StatusCode)
	}

	t.Logf("✅ File serving test passed - Downloaded file content matches: %s", testFilename)
}

// Test 22: File Path Security - Test path traversal prevention
func TestRESTFilePathSecurity(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file path security and traversal prevention")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "Security Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Note: Multipart form uploads automatically sanitize filenames via Go's multipart library
	// This is an additional security layer - dangerous path components are stripped
	// We verify this behavior but know it's not our os.Root protection

	// Test that dangerous filenames get sanitized by multipart library (expected behavior)
	dangerousFilename := "../etc/passwd"
	testContent := "This should not be written outside session directory"

	t.Logf("Testing multipart filename sanitization for: %s", dangerousFilename)

	uploadResp := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+sessionID+"/files/upload",
		dangerousFilename,
		testContent)

	// Multipart library sanitizes this to just "passwd", so upload should succeed
	if uploadResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code %d after multipart sanitization, got %d",
			http.StatusCreated, uploadResp.StatusCode)
	}
	uploadResp.Body.Close()

	// Verify the file was created with sanitized name "passwd"
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/files", nil)
	filesList := validateArrayResponse(t, listResp, http.StatusOK)

	found := false
	for _, fileItem := range filesList {
		fileObj := fileItem.(map[string]interface{})
		if fileObj["name"].(string) == "passwd" { // Sanitized filename
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected to find sanitized filename 'passwd' in file list")
	}

	// Test dangerous file access paths (should be rejected)
	dangerousAccessPaths := []string{
		"../other-session-file.txt",
		"../../config.json",
		"../../../../../etc/passwd",
		"./../sensitive.txt",
	}

	for _, dangerousPath := range dangerousAccessPaths {
		t.Logf("Testing file access rejection for dangerous path: %s", dangerousPath)

		accessResp := makeJSONRequest(t, result.Server, "GET",
			"/api/sessions/"+sessionID+"/files/"+dangerousPath, nil)

		// Should be rejected - either 400 (validation error) or 404 (router rejects path)
		// Both are acceptable security responses
		if accessResp.StatusCode != http.StatusBadRequest && accessResp.StatusCode != http.StatusNotFound {
			t.Fatalf("Expected status code %d or %d for dangerous path '%s', got %d",
				http.StatusBadRequest, http.StatusNotFound, dangerousPath, accessResp.StatusCode)
		}
		accessResp.Body.Close()
	}

	// Test that normal filenames still work (positive test)
	normalFilename := "normal-file.txt"
	normalContent := "This is a normal file"

	normalUploadResp := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+sessionID+"/files/upload",
		normalFilename,
		normalContent)

	if normalUploadResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code %d for normal filename, got %d",
			http.StatusCreated, normalUploadResp.StatusCode)
	}
	normalUploadResp.Body.Close()

	// Verify normal file can be accessed
	normalAccessResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionID+"/files/"+normalFilename, nil)

	if normalAccessResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d for normal file access, got %d",
			http.StatusOK, normalAccessResp.StatusCode)
	}
	normalAccessResp.Body.Close()

	t.Logf("✅ File path security test passed - All dangerous paths properly rejected")
}

// Test 23: File Session Isolation - Verify files are isolated between sessions
func TestRESTFileSessionIsolation(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file session isolation")

	// Create two separate sessions
	session1Request := map[string]interface{}{
		"title": "Isolation Test Session 1",
	}
	session2Request := map[string]interface{}{
		"title": "Isolation Test Session 2",
	}

	// Create session 1
	createResp1 := makeJSONRequest(t, result.Server, "POST", "/api/sessions", session1Request)
	session1Data := validateObjectResponse(t, createResp1, http.StatusCreated)
	session1ID := session1Data["id"].(string)

	// Create session 2
	createResp2 := makeJSONRequest(t, result.Server, "POST", "/api/sessions", session2Request)
	session2Data := validateObjectResponse(t, createResp2, http.StatusCreated)
	session2ID := session2Data["id"].(string)

	// Upload a file to session 1
	session1FileName := "session1-private.txt"
	session1Content := "This file belongs to session 1 and should not be accessible from session 2"

	uploadResp1 := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+session1ID+"/files/upload",
		session1FileName,
		session1Content)

	validateObjectResponse(t, uploadResp1, http.StatusCreated)

	// Upload a file to session 2 (different file, same name to test isolation)
	session2FileName := "session1-private.txt" // Same filename as session 1
	session2Content := "This file belongs to session 2 with the same filename"

	uploadResp2 := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+session2ID+"/files/upload",
		session2FileName,
		session2Content)

	validateObjectResponse(t, uploadResp2, http.StatusCreated)

	// Verify session 1 can access its own file
	access1Resp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+session1ID+"/files/"+session1FileName, nil)

	if access1Resp.StatusCode != http.StatusOK {
		t.Fatalf("Session 1 should be able to access its own file, got status %d", access1Resp.StatusCode)
	}

	// Read content to verify it's session 1's file
	content1 := make([]byte, len(session1Content)+10)
	n1, _ := access1Resp.Body.Read(content1)
	access1Resp.Body.Close()
	actualContent1 := string(content1[:n1])

	if actualContent1 != session1Content {
		t.Fatalf("Session 1 file content mismatch. Expected: %q, Got: %q", session1Content, actualContent1)
	}

	// Verify session 2 can access its own file
	access2Resp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+session2ID+"/files/"+session2FileName, nil)

	if access2Resp.StatusCode != http.StatusOK {
		t.Fatalf("Session 2 should be able to access its own file, got status %d", access2Resp.StatusCode)
	}

	// Read content to verify it's session 2's file
	content2 := make([]byte, len(session2Content)+10)
	n2, _ := access2Resp.Body.Read(content2)
	access2Resp.Body.Close()
	actualContent2 := string(content2[:n2])

	if actualContent2 != session2Content {
		t.Fatalf("Session 2 file content mismatch. Expected: %q, Got: %q", session2Content, actualContent2)
	}

	// Test file listing isolation - session 1 should only see its own files
	list1Resp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+session1ID+"/files", nil)
	files1List := validateArrayResponse(t, list1Resp, http.StatusOK)

	// Session 1 should see exactly 1 file
	if len(files1List) != 1 {
		t.Fatalf("Session 1 should see exactly 1 file, got %d", len(files1List))
	}

	// Test file listing isolation - session 2 should only see its own files
	list2Resp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+session2ID+"/files", nil)
	files2List := validateArrayResponse(t, list2Resp, http.StatusOK)

	// Session 2 should see exactly 1 file
	if len(files2List) != 1 {
		t.Fatalf("Session 2 should see exactly 1 file, got %d", len(files2List))
	}

	// Test cross-session access attempt (using wrong session IDs)
	// This tests the session validation, not just file existence
	wrongSessionResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/wrong-session-id/files/"+session1FileName, nil)

	// Should fail with validation error (invalid session ID format returns 400 Bad Request)
	if wrongSessionResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid session ID format, got %d",
			http.StatusBadRequest, wrongSessionResp.StatusCode)
	}
	wrongSessionResp.Body.Close()

	t.Logf("✅ File session isolation test passed - Sessions properly isolated")
}
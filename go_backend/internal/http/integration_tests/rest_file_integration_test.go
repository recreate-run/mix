package integration_tests

import (
	"net/http"
	"testing"

	"mix/internal/storage"
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

// Test 21: File Upload and Serving - Upload and download file, verify content (round-trip test)
func TestRESTFileUploadAndServing(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file upload and serving round-trip functionality")

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

	t.Logf("✅ File upload and serving round-trip test passed - Downloaded file content matches: %s", testFilename)
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

// Test 23: File Shared Storage - Verify files are shared between sessions via uploads directory
func TestRESTFileSharedStorage(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file shared storage via uploads directory")

	// Create two separate sessions
	session1Request := map[string]interface{}{
		"title": "Shared Storage Test Session 1",
	}
	session2Request := map[string]interface{}{
		"title": "Shared Storage Test Session 2",
	}

	// Create session 1
	createResp1 := makeJSONRequest(t, result.Server, "POST", "/api/sessions", session1Request)
	session1Data := validateObjectResponse(t, createResp1, http.StatusCreated)
	session1ID := session1Data["id"].(string)

	// Create session 2
	createResp2 := makeJSONRequest(t, result.Server, "POST", "/api/sessions", session2Request)
	session2Data := validateObjectResponse(t, createResp2, http.StatusCreated)
	session2ID := session2Data["id"].(string)

	// Upload a file via session 1
	sharedFileName := "shared-file.txt"
	sharedContent := "This file is uploaded via session 1 but should be accessible from session 2"

	uploadResp1 := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+session1ID+"/files/upload",
		sharedFileName,
		sharedContent)

	validateObjectResponse(t, uploadResp1, http.StatusCreated)

	// Verify session 1 can access the uploaded file
	access1Resp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+session1ID+"/files/"+sharedFileName, nil)

	if access1Resp.StatusCode != http.StatusOK {
		t.Fatalf("Session 1 should be able to access uploaded file, got status %d", access1Resp.StatusCode)
	}

	// Read content to verify it's correct
	content1 := make([]byte, len(sharedContent)+10)
	n1, _ := access1Resp.Body.Read(content1)
	access1Resp.Body.Close()
	actualContent1 := string(content1[:n1])

	if actualContent1 != sharedContent {
		t.Fatalf("Session 1 file content mismatch. Expected: %q, Got: %q", sharedContent, actualContent1)
	}

	// Verify session 2 can also access the same file (since it's in shared uploads)
	access2Resp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+session2ID+"/files/"+sharedFileName, nil)

	if access2Resp.StatusCode != http.StatusOK {
		t.Fatalf("Session 2 should be able to access shared file, got status %d", access2Resp.StatusCode)
	}

	// Read content to verify it's the same file
	content2 := make([]byte, len(sharedContent)+10)
	n2, _ := access2Resp.Body.Read(content2)
	access2Resp.Body.Close()
	actualContent2 := string(content2[:n2])

	if actualContent2 != sharedContent {
		t.Fatalf("Session 2 file content mismatch. Expected: %q, Got: %q", sharedContent, actualContent2)
	}

	// Test file listing - both sessions should see the same files from uploads directory
	list1Resp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+session1ID+"/files", nil)
	files1List := validateArrayResponse(t, list1Resp, http.StatusOK)

	list2Resp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+session2ID+"/files", nil)
	files2List := validateArrayResponse(t, list2Resp, http.StatusOK)

	// Both sessions should see the same number of files (from shared uploads directory)
	if len(files1List) != len(files2List) {
		t.Fatalf("Both sessions should see same file count, got %d vs %d", len(files1List), len(files2List))
	}

	// Both should see at least the file we uploaded
	found1, found2 := false, false
	for _, fileItem := range files1List {
		fileObj := fileItem.(map[string]interface{})
		if fileObj["name"].(string) == sharedFileName {
			found1 = true
			break
		}
	}
	for _, fileItem := range files2List {
		fileObj := fileItem.(map[string]interface{})
		if fileObj["name"].(string) == sharedFileName {
			found2 = true
			break
		}
	}

	if !found1 || !found2 {
		t.Fatalf("Both sessions should see the shared file in listings, found1=%t, found2=%t", found1, found2)
	}

	// Test cross-session access attempt with invalid session ID (should still fail validation)
	wrongSessionResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/wrong-session-id/files/"+sharedFileName, nil)

	// Should fail with validation error (invalid session ID format returns 400 Bad Request)
	if wrongSessionResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid session ID format, got %d",
			http.StatusBadRequest, wrongSessionResp.StatusCode)
	}
	wrongSessionResp.Body.Close()

	t.Logf("✅ File shared storage test passed - Files properly shared via uploads directory")
}

// Test 24: File Deletion - DELETE /api/sessions/{id}/files/{filename}
func TestRESTFileDeletion(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing file deletion functionality")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "File Deletion Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Upload a test file
	testContent := "This is test content for deletion testing"
	testFilename := "test-delete.txt"

	uploadResp := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+sessionID+"/files/upload",
		testFilename,
		testContent)

	validateObjectResponse(t, uploadResp, http.StatusCreated)

	// Verify file exists before deletion
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/files", nil)
	filesList := validateArrayResponse(t, listResp, http.StatusOK)

	fileExists := false
	for _, fileItem := range filesList {
		fileObj := fileItem.(map[string]interface{})
		if fileObj["name"].(string) == testFilename {
			fileExists = true
			break
		}
	}
	if !fileExists {
		t.Fatalf("File %s should exist before deletion", testFilename)
	}

	// Delete the file
	deleteResp := makeJSONRequest(t, result.Server, "DELETE",
		"/api/sessions/"+sessionID+"/files/"+testFilename, nil)

	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status code %d for file deletion, got %d", http.StatusNoContent, deleteResp.StatusCode)
	}

	// Verify file is gone from file list
	listAfterDeleteResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/files", nil)
	filesAfterDelete := validateArrayResponse(t, listAfterDeleteResp, http.StatusOK)

	for _, fileItem := range filesAfterDelete {
		fileObj := fileItem.(map[string]interface{})
		if fileObj["name"].(string) == testFilename {
			t.Fatalf("File %s should not exist after deletion", testFilename)
		}
	}

	// Verify file cannot be accessed directly (should return 404)
	accessResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionID+"/files/"+testFilename, nil)

	if accessResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code %d when accessing deleted file, got %d", http.StatusNotFound, accessResp.StatusCode)
	}

	// Test deleting non-existent file (should return 404)
	nonExistentDeleteResp := makeJSONRequest(t, result.Server, "DELETE",
		"/api/sessions/"+sessionID+"/files/non-existent-file.txt", nil)

	if nonExistentDeleteResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status code %d when deleting non-existent file, got %d", http.StatusNotFound, nonExistentDeleteResp.StatusCode)
	}

	// Test deleting from non-existent session (should return 400)
	invalidSessionDeleteResp := makeJSONRequest(t, result.Server, "DELETE",
		"/api/sessions/invalid-session-id/files/"+testFilename, nil)

	if invalidSessionDeleteResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d when deleting from invalid session, got %d", http.StatusBadRequest, invalidSessionDeleteResp.StatusCode)
	}

	t.Logf("✅ File deletion test passed - File properly deleted: %s", testFilename)
}

// Test 25: Thumbnail Generation - GET /api/sessions/{id}/files/{filename}?thumb=...
func TestRESTFileThumbnailGeneration(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing thumbnail generation functionality")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "Thumbnail Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Upload a text file and test thumbnail on non-image (should return 400)
	textContent := "This is not an image file"
	textFilename := "test-text.txt"

	textUploadResp := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+sessionID+"/files/upload",
		textFilename,
		textContent)

	validateObjectResponse(t, textUploadResp, http.StatusCreated)

	// Try to generate thumbnail from text file (should fail)
	textThumbResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionID+"/files/"+textFilename+"?thumb=100", nil)

	if textThumbResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for thumbnail on non-image, got %d", http.StatusBadRequest, textThumbResp.StatusCode)
	}
	textThumbResp.Body.Close()

	// Test invalid thumbnail parameter on text file (should return 400)
	invalidThumbResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionID+"/files/"+textFilename+"?thumb=invalid", nil)

	if invalidThumbResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid thumbnail param, got %d", http.StatusBadRequest, invalidThumbResp.StatusCode)
	}
	invalidThumbResp.Body.Close()

	// Test accessing file without thumbnail params (should work normally)
	normalResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionID+"/files/"+textFilename, nil)

	if normalResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d for normal file access, got %d", http.StatusOK, normalResp.StatusCode)
	}
	normalResp.Body.Close()

	t.Logf("✅ Thumbnail generation test passed - Thumbnail parameter validation working")
}

// Test 26: Large File Handling - Test large file operations
func TestRESTLargeFileHandling(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing large file handling functionality")

	// Create a session first
	sessionRequest := map[string]interface{}{
		"title": "Large File Test Session",
	}

	createResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionRequest)
	createdSessionData := validateObjectResponse(t, createResp, http.StatusCreated)
	sessionID := createdSessionData["id"].(string)

	// Test uploading a reasonably large file (1MB)
	// Create 1MB of data for testing
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256) // Fill with varying data
	}
	largeFilename := "large-test-file.bin"

	t.Logf("Testing upload of %d KB file", len(largeContent)/1024)

	largeUploadResp := makeMultipartFileRequestFromBytes(t, result.Server,
		"/api/sessions/"+sessionID+"/files/upload",
		largeFilename,
		largeContent)

	if largeUploadResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code %d for large file upload, got %d", http.StatusCreated, largeUploadResp.StatusCode)
	}

	uploadData := validateObjectResponse(t, largeUploadResp, http.StatusCreated)

	// Verify upload response contains correct size
	uploadedSize, ok := uploadData["size"].(float64)
	if !ok || int(uploadedSize) != len(largeContent) {
		t.Fatalf("Expected uploaded file size %d, got %v", len(largeContent), uploadedSize)
	}

	// Verify the large file appears in file list
	listResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/files", nil)
	filesList := validateArrayResponse(t, listResp, http.StatusOK)

	found := false
	for _, fileItem := range filesList {
		fileObj := fileItem.(map[string]interface{})
		if fileObj["name"].(string) == largeFilename {
			found = true
			// Verify size in list matches
			if int(fileObj["size"].(float64)) != len(largeContent) {
				t.Fatalf("Expected file size %d in list, got %v", len(largeContent), fileObj["size"])
			}
			break
		}
	}

	if !found {
		t.Fatalf("Large file '%s' not found in session file list", largeFilename)
	}

	// Try to access the large file (verify headers, don't download full content)
	accessResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionID+"/files/"+largeFilename, nil)

	if accessResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d when accessing large file, got %d", http.StatusOK, accessResp.StatusCode)
	}
	accessResp.Body.Close()

	// Verify we can delete the large file
	deleteResp := makeJSONRequest(t, result.Server, "DELETE",
		"/api/sessions/"+sessionID+"/files/"+largeFilename, nil)

	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status code %d for large file deletion, got %d", http.StatusNoContent, deleteResp.StatusCode)
	}

	// Verify file is gone after deletion
	listAfterDeleteResp := makeJSONRequest(t, result.Server, "GET", "/api/sessions/"+sessionID+"/files", nil)
	filesAfterDelete := validateArrayResponse(t, listAfterDeleteResp, http.StatusOK)

	for _, fileItem := range filesAfterDelete {
		fileObj := fileItem.(map[string]interface{})
		if fileObj["name"].(string) == largeFilename {
			t.Fatalf("Large file %s should not exist after deletion", largeFilename)
		}
	}

	t.Logf("✅ Large file handling test passed - 1MB file uploaded, listed, accessed, and deleted successfully")
}

// Test 27: Session Isolated File Serving - Test session-specific storage access
func TestRESTSessionIsolatedFileServing(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing session-isolated file serving functionality")

	// Create two sessions
	sessionARequest := map[string]interface{}{
		"title": "Session A for File Isolation Test",
	}
	sessionBRequest := map[string]interface{}{
		"title": "Session B for File Isolation Test",
	}

	// Create session A
	createARsep := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionARequest)
	sessionAData := validateObjectResponse(t, createARsep, http.StatusCreated)
	sessionAID := sessionAData["id"].(string)

	// Create session B
	createBResp := makeJSONRequest(t, result.Server, "POST", "/api/sessions", sessionBRequest)
	sessionBData := validateObjectResponse(t, createBResp, http.StatusCreated)
	sessionBID := sessionBData["id"].(string)

	// Manually create a file in session A's storage directory
	testContent := "This is session A's private file content.\nIt should only be accessible by session A."
	testFilename := "session-private.txt"

	// Get session A's storage root and create the file
	sessionARoot, err := storage.GetSessionRoot(sessionAID, result.App.StorageConfig)
	if err != nil {
		t.Fatalf("Failed to get session A storage root: %v", err)
	}
	defer sessionARoot.Close()

	// Create the file in session A's directory
	file, err := sessionARoot.Create(testFilename)
	if err != nil {
		t.Fatalf("Failed to create file in session A storage: %v", err)
	}
	_, err = file.WriteString(testContent)
	file.Close()
	if err != nil {
		t.Fatalf("Failed to write content to session A file: %v", err)
	}

	// Test 1: Session A should be able to access its own file
	accessAResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionAID+"/files/"+testFilename, nil)

	if accessAResp.StatusCode != http.StatusOK {
		t.Fatalf("Session A should be able to access its own file, got status %d", accessAResp.StatusCode)
	}

	// Verify content integrity
	content := make([]byte, len(testContent)+10)
	n, err := accessAResp.Body.Read(content)
	accessAResp.Body.Close()
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Failed to read session A file content: %v", err)
	}

	actualContent := string(content[:n])
	if actualContent != testContent {
		t.Fatalf("Session A file content mismatch.\nExpected: %q\nActual: %q", testContent, actualContent)
	}

	// Test 2: Session B should NOT be able to access session A's file
	accessBResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionBID+"/files/"+testFilename, nil)

	if accessBResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Session B should not be able to access session A's file, got status %d", accessBResp.StatusCode)
	}
	accessBResp.Body.Close()

	// Test 3: Verify shared uploads still work for both sessions
	// Upload a file to shared storage via session A
	sharedContent := "This is shared content accessible by all sessions"
	sharedFilename := "shared-test.txt"

	uploadResp := makeMultipartFileRequest(t, result.Server,
		"/api/sessions/"+sessionAID+"/files/upload",
		sharedFilename,
		sharedContent)

	validateObjectResponse(t, uploadResp, http.StatusCreated)

	// Both sessions should be able to access the shared file
	sharedAccessAResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionAID+"/files/"+sharedFilename, nil)
	if sharedAccessAResp.StatusCode != http.StatusOK {
		t.Fatalf("Session A should access shared file, got status %d", sharedAccessAResp.StatusCode)
	}
	sharedAccessAResp.Body.Close()

	sharedAccessBResp := makeJSONRequest(t, result.Server, "GET",
		"/api/sessions/"+sessionBID+"/files/"+sharedFilename, nil)
	if sharedAccessBResp.StatusCode != http.StatusOK {
		t.Fatalf("Session B should access shared file, got status %d", sharedAccessBResp.StatusCode)
	}
	sharedAccessBResp.Body.Close()

	t.Logf("✅ Session isolated file serving test passed - Session isolation working correctly")
}
//go:build integration
// +build integration

package browser

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"

	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/session"
)

func TestBrowserTool_AnalyzeScreenshot_Integration(t *testing.T) {
	// Skip if GEMINI_API_KEY not set
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	// Skip if browser service not available
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}

	// Initialize context
	ctx := context.Background()

	// Initialize test database with migrations
	testConfigDir := t.TempDir()
	_ = os.Setenv("_CONFIG_DIR", testConfigDir)
	_ = os.Setenv("_DATA_DIR", testConfigDir)
	_ = os.Setenv("VITE_BACKEND_URL", "http://localhost:3020")

	// Load config (creates database structure)
	_, err := config.Load(testConfigDir, false, false)
	require.NoError(t, err)

	// Connect to database (runs migrations)
	dbConn, err := db.Connect(ctx, ".mix")
	require.NoError(t, err)
	defer dbConn.Close()

	// Initialize API credentials service
	err = config.InitAPICredentials(dbConn)
	require.NoError(t, err)

	// Store Gemini API key in credentials service
	apiCredService := config.GetAPICredentials()
	require.NotNil(t, apiCredService)

	err = apiCredService.StoreAPIKey(ctx, models.ProviderGemini, apiKey)
	require.NoError(t, err)

	// Start mock browser server
	mockServer := startMockBrowserServer(t)
	defer mockServer.Close()

	// Create browser tool
	sessionConfig := session.Config{
		BasePath: t.TempDir(),
	}

	tool := NewBrowserTool(
		nil,              // permissions not needed for test
		&MockSessionService{}, // sessions
		mockServer.wsURL,
		sessionConfig,
		"local-browser-service",
		nil, // client factory not needed
		nil, // connection manager
		nil, // tunnel registry getter
		"",  // baseURL not needed for test
	)

	// Create context with session info
	ctx = context.WithValue(ctx, interfaces.SessionIDContextKey, "test-session")
	ctx = context.WithValue(ctx, interfaces.SessionStorageContextKey, sessionConfig.BasePath)
	ctx = context.WithValue(ctx, interfaces.MessageIDContextKey, "test-message")

	// Test 1: Text analysis of screenshot
	t.Run("TextAnalysis", func(t *testing.T) {
		params := BrowserParams{
			Action: ActionAnalyzeScreenshot,
			TabID:  "tab-1",
			Prompt: "Describe what you see in this screenshot in one sentence.",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := interfaces.ToolCall{
			ID:    "test-call-1",
			Input: string(paramsJSON),
		}

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		if response.IsError {
			t.Fatalf("Expected no error but got: %s", response.Content)
		}
		require.NotEmpty(t, response.Content)

		t.Logf("Text analysis response: %s", response.Content)
	})

	// Test 2: Bounding box detection (text format validation)
	t.Run("BoundingBox", func(t *testing.T) {
		params := BrowserParams{
			Action: ActionAnalyzeScreenshot,
			TabID:  "tab-1",
			Prompt: "Give me the bounding box coordinates for any button you can find. Use normalized coordinates in the range [0, 1000]. If there are no buttons, return an empty array.",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := interfaces.ToolCall{
			ID:    "test-call-2",
			Input: string(paramsJSON),
		}

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		require.False(t, response.IsError, "Expected no error, got: %s", response.Content)
		require.NotEmpty(t, response.Content)

		// Validate text format (should match read_page style)
		require.Contains(t, response.Content, "Found", "Should contain 'Found' in response")
		require.Contains(t, response.Content, "element(s)", "Should contain 'element(s)' in response")

		// For non-empty results, validate coordinate format
		if strings.Contains(response.Content, "- ") {
			// Should have format: - element_type (x=123,y=456)
			require.Regexp(t, `- \w+ \(x=\d+,y=\d+\)`, response.Content, "Should match read_page coordinate format")
		}

		t.Logf("Bounding box response:\n%s", response.Content)
	})

	// Test 3: Missing prompt parameter
	t.Run("MissingPrompt", func(t *testing.T) {
		params := BrowserParams{
			Action: ActionAnalyzeScreenshot,
			TabID:  "tab-1",
			// Prompt is missing
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := interfaces.ToolCall{
			ID:    "test-call-3",
			Input: string(paramsJSON),
		}

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "missing prompt parameter")

		t.Logf("Error response: %s", response.Content)
	})

	// Test 4: Missing tab ID
	t.Run("MissingTabID", func(t *testing.T) {
		params := BrowserParams{
			Action: ActionAnalyzeScreenshot,
			Prompt: "Describe the page",
			// TabID is missing
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := interfaces.ToolCall{
			ID:    "test-call-4",
			Input: string(paramsJSON),
		}

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		response, err := tool.Run(ctx, call)
		require.NoError(t, err)
		require.True(t, response.IsError)
		require.Contains(t, response.Content, "tabId")

		t.Logf("Error response: %s", response.Content)
	})
}

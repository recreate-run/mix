//go:build integration
// +build integration

package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/stretchr/testify/require"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/session"
)

// mockBrowserServerWithImage creates a mock browser server that returns a specific test image
func startMockBrowserServerWithImage(t *testing.T, screenshotBase64 string) *mockBrowserServer {
	t.Helper()

	mbs := &mockBrowserServer{
		connections: make(map[*websocket.Conn]bool),
		tabs:        make(map[string]map[string]any),
		tabCounter:  0,
	}

	// Create initial default tab
	mbs.tabs["tab-1"] = map[string]any{
		"id":       "tab-1",
		"url":      "about:blank",
		"title":    "New Tab",
		"isActive": true,
	}
	mbs.activeTabID = "tab-1"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Failed to upgrade connection: %v", err)
			return
		}

		mbs.mu.Lock()
		mbs.connections[conn] = true
		mbs.mu.Unlock()
		defer func() {
			mbs.mu.Lock()
			delete(mbs.connections, conn)
			mbs.mu.Unlock()
			_ = conn.Close()
		}()

		// Handle WebSocket messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			mbs.mu.Lock()
			mbs.requestCount++
			mbs.mu.Unlock()

			// Parse request
			var req browserprotocol.Request
			if err := json.Unmarshal(message, &req); err != nil {
				continue
			}

			// Generate response based on method
			var response browserprotocol.Response
			response.ID = req.ID

			switch req.Method {
			case "Page.navigate":
				response.Result = map[string]any{
					"frameId": "frame-123",
				}

			case "Page.screenshot":
				// Return the provided test image
				response.Result = map[string]any{
					"data":   screenshotBase64,
					"format": "png",
				}

			case "Browser.close":
				response.Result = map[string]any{
					"success": true,
				}

			default:
				response.Result = map[string]any{}
			}

			// Send response
			respData, _ := json.Marshal(response)
			if err := conn.WriteMessage(websocket.TextMessage, respData); err != nil {
				return
			}
		}
	})

	mbs.server = httptest.NewServer(handler)
	mbs.wsURL = "ws://" + mbs.server.Listener.Addr().String()

	return mbs
}

func TestBrowserTool_AnalyzeScreenshot_WithRealImage_Integration(t *testing.T) {
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

	// Load test image
	imageData, err := os.ReadFile("testdata/taxonomy_button.png")
	require.NoError(t, err, "Failed to read test image")

	// Encode to base64
	screenshotBase64 := base64.StdEncoding.EncodeToString(imageData)

	// Start mock browser server with test image
	mockServer := startMockBrowserServerWithImage(t, screenshotBase64)
	defer mockServer.Close()

	// Create browser tool
	sessionConfig := session.Config{
		BasePath: t.TempDir(),
	}

	tool := NewBrowserTool(
		nil, // permissions not needed for test
		mockServer.wsURL,
		sessionConfig,
		"service",
		nil, // client factory not needed
		nil, // connection manager
		nil, // tunnel registry getter
	)

	// Create context with session info
	ctx = context.WithValue(ctx, interfaces.SessionIDContextKey, "test-session")
	ctx = context.WithValue(ctx, interfaces.SessionStorageContextKey, sessionConfig.BasePath)
	ctx = context.WithValue(ctx, interfaces.MessageIDContextKey, "test-message")

	// Test: Bounding box detection with real image
	t.Run("BoundingBoxWithRealImage", func(t *testing.T) {
		params := BrowserParams{
			Action: ActionAnalyzeScreenshot,
			TabID:  "tab-1",
			Prompt: "Give me the bounding box coordinates for the taxonomy button in the contents sidebar. Use normalized coordinates in the range [0, 1000].",
		}

		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		call := interfaces.ToolCall{
			ID:    "test-call-real-image",
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

		// Validate JSON structure
		var boundingBox []map[string]interface{}
		err = json.Unmarshal([]byte(response.Content), &boundingBox)
		require.NoError(t, err, "Response should be valid JSON")
		require.NotEmpty(t, boundingBox, "Should have at least one bounding box")

		// Validate box_2d structure
		box2d, ok := boundingBox[0]["box_2d"]
		require.True(t, ok, "Should have box_2d key")

		coords, ok := box2d.([]interface{})
		require.True(t, ok, "box_2d should be an array")
		require.Equal(t, 4, len(coords), "Should have 4 coordinates")

		// Validate coordinate range
		for i, coord := range coords {
			coordFloat, ok := coord.(float64)
			require.True(t, ok, "Coordinate should be a number")
			require.GreaterOrEqual(t, coordFloat, 0.0, "Coordinate %d should be >= 0", i)
			require.LessOrEqual(t, coordFloat, 1000.0, "Coordinate %d should be <= 1000", i)
		}

		// Log the coordinates for manual verification
		t.Logf("Detected taxonomy button bounding box: %s", response.Content)
		t.Logf("Expected coordinates approximately: [398, 37, 418, 104]")
		t.Logf("Actual coordinates: [%.0f, %.0f, %.0f, %.0f]",
			coords[0].(float64), coords[1].(float64), coords[2].(float64), coords[3].(float64))

		// Validate coordinates are in reasonable range (not exact match due to Gemini variance)
		// X1 should be around 398 (normalized to [0,1000])
		x1 := coords[0].(float64)
		require.Greater(t, x1, 300.0, "X1 should be roughly in the middle-right area")
		require.Less(t, x1, 500.0, "X1 should be roughly in the middle-right area")
	})
}

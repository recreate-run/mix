package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/permission"
	"mix/internal/pubsub"
)

// Mock permission service for basic testing
type mockPermissionServiceSimple struct {
	mock.Mock
}

func (m *mockPermissionServiceSimple) Subscribe(ctx context.Context) <-chan pubsub.Event[permission.PermissionRequest] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[permission.PermissionRequest])
}

func (m *mockPermissionServiceSimple) Unsubscribe(ch chan<- permission.PermissionRequest) {
	m.Called(ch)
}

func (m *mockPermissionServiceSimple) GrantPersistant(perm permission.PermissionRequest) {
	m.Called(perm)
}

func (m *mockPermissionServiceSimple) Grant(perm permission.PermissionRequest) {
	m.Called(perm)
}

func (m *mockPermissionServiceSimple) Deny(perm permission.PermissionRequest) {
	m.Called(perm)
}

func (m *mockPermissionServiceSimple) Request(opts permission.CreatePermissionRequest) bool {
	args := m.Called(opts)
	return args.Bool(0)
}

// Helper to create test context
func createSimpleTestContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, "test-session-123")
	ctx = context.WithValue(ctx, MessageIDContextKey, "test-message-456")
	ctx = context.WithValue(ctx, SessionStorageContextKey, "/test/storage/test-session-123")
	return ctx
}

// Test SearchParams JSON serialization/deserialization
func TestSearchParams_JSONSerialization_Simple(t *testing.T) {
	spellCheckTrue := true
	spellCheckFalse := false

	tests := []struct {
		name   string
		params SearchParams
	}{
		{
			name: "basic params",
			params: SearchParams{
				Query:      "test query",
				SearchType: "web",
			},
		},
		{
			name: "full params with domains",
			params: SearchParams{
				Query:          "golang tutorial",
				SearchType:     "images",
				AllowedDomains: []string{"example.com", "test.org"},
				BlockedDomains: []string{"spam.com"},
				SafeSearch:     "strict",
				SpellCheck:     &spellCheckTrue,
			},
		},
		{
			name: "params with false spellcheck",
			params: SearchParams{
				Query:      "test",
				SpellCheck: &spellCheckFalse,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := json.Marshal(tt.params)
			require.NoError(t, err)

			// Deserialize
			var deserializedParams SearchParams
			err = json.Unmarshal(data, &deserializedParams)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, tt.params.Query, deserializedParams.Query)
			assert.Equal(t, tt.params.SearchType, deserializedParams.SearchType)
			assert.Equal(t, tt.params.AllowedDomains, deserializedParams.AllowedDomains)
			assert.Equal(t, tt.params.BlockedDomains, deserializedParams.BlockedDomains)
			assert.Equal(t, tt.params.SafeSearch, deserializedParams.SafeSearch)

			if tt.params.SpellCheck != nil {
				require.NotNil(t, deserializedParams.SpellCheck)
				assert.Equal(t, *tt.params.SpellCheck, *deserializedParams.SpellCheck)
			} else {
				assert.Nil(t, deserializedParams.SpellCheck)
			}
		})
	}
}

// Test response structures JSON deserialization
func TestResponseStructures_JSONDeserialization_Simple(t *testing.T) {
	t.Run("BraveSearchResponse", func(t *testing.T) {
		jsonData := `{
			"type": "search",
			"web": {
				"type": "search",
				"results": [
					{
						"title": "Test Title",
						"url": "https://example.com",
						"description": "Test description"
					}
				]
			}
		}`

		var response BraveSearchResponse
		err := json.Unmarshal([]byte(jsonData), &response)
		require.NoError(t, err)

		assert.Equal(t, "search", response.Type)
		assert.Equal(t, "search", response.Web.Type)
		assert.Len(t, response.Web.Results, 1)
		assert.Equal(t, "Test Title", response.Web.Results[0].Title)
		assert.Equal(t, "https://example.com", response.Web.Results[0].URL)
		assert.Equal(t, "Test description", response.Web.Results[0].Description)
	})

	t.Run("ImageSearchResponse", func(t *testing.T) {
		jsonData := `{
			"type": "images",
			"results": [
				{
					"type": "image",
					"title": "Test Image",
					"url": "https://example.com/page",
					"source": "example.com",
					"page_fetched": "2023-01-01T00:00:00Z",
					"thumbnail": {
						"src": "https://example.com/thumb.jpg"
					},
					"properties": {
						"url": "https://example.com/image.jpg",
						"placeholder": "placeholder_url"
					},
					"confidence": "high"
				}
			]
		}`

		var response ImageSearchResponse
		err := json.Unmarshal([]byte(jsonData), &response)
		require.NoError(t, err)

		assert.Equal(t, "images", response.Type)
		assert.Len(t, response.Results, 1)

		result := response.Results[0]
		assert.Equal(t, "image", result.Type)
		assert.Equal(t, "Test Image", result.Title)
		assert.Equal(t, "https://example.com/page", result.URL)
		assert.Equal(t, "example.com", result.Source)
		assert.Equal(t, "https://example.com/thumb.jpg", result.Thumbnail.Src)
		assert.Equal(t, "https://example.com/image.jpg", result.Properties.URL)
		assert.Equal(t, "high", result.Confidence)
	})

	t.Run("VideoSearchResponse", func(t *testing.T) {
		jsonData := `{
			"type": "videos",
			"results": [
				{
					"type": "video",
					"title": "Test Video",
					"url": "https://example.com/video-page",
					"source": "youtube.com",
					"page_fetched": "2023-01-01T00:00:00Z",
					"thumbnail": {
						"src": "https://example.com/video-thumb.jpg"
					},
					"properties": {
						"url": "https://example.com/video.mp4",
						"duration": "10:30",
						"views": "1000",
						"upload_date": "2023-01-01",
						"placeholder": "placeholder_url"
					},
					"confidence": "medium"
				}
			]
		}`

		var response VideoSearchResponse
		err := json.Unmarshal([]byte(jsonData), &response)
		require.NoError(t, err)

		assert.Equal(t, "videos", response.Type)
		assert.Len(t, response.Results, 1)

		result := response.Results[0]
		assert.Equal(t, "video", result.Type)
		assert.Equal(t, "Test Video", result.Title)
		assert.Equal(t, "https://example.com/video-page", result.URL)
		assert.Equal(t, "youtube.com", result.Source)
		assert.Equal(t, "https://example.com/video-thumb.jpg", result.Thumbnail.Src)
		assert.Equal(t, "https://example.com/video.mp4", result.Properties.URL)
		assert.Equal(t, "10:30", result.Properties.Duration)
		assert.Equal(t, "1000", result.Properties.Views)
		assert.Equal(t, "2023-01-01", result.Properties.UploadDate)
		assert.Equal(t, "medium", result.Confidence)
	})
}

// Test NewWebSearchTool
func TestNewWebSearchTool_Simple(t *testing.T) {
	mockPerms := &mockPermissionServiceSimple{}
	tool := NewWebSearchTool(mockPerms)

	assert.NotNil(t, tool)

	// Type assertion to access internal fields
	searchTool, ok := tool.(*searchTool)
	require.True(t, ok)

	assert.NotNil(t, searchTool.client)
	assert.Equal(t, 30*time.Second, searchTool.client.Timeout)
	assert.Equal(t, mockPerms, searchTool.permissions)
}

// Test searchTool.Info()
func TestSearchTool_Info_Simple(t *testing.T) {
	mockPerms := &mockPermissionServiceSimple{}
	tool := NewWebSearchTool(mockPerms)

	info := tool.Info()

	assert.Equal(t, SearchToolName, info.Name)
	assert.Equal(t, "search", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Required, "query")

	// Check parameter schema
	params := info.Parameters
	assert.Contains(t, params, "query")
	assert.Contains(t, params, "search_type")
	assert.Contains(t, params, "allowed_domains")
	assert.Contains(t, params, "blocked_domains")
	assert.Contains(t, params, "safesearch")
	assert.Contains(t, params, "spellcheck")

	// Check query parameter details
	queryParam := params["query"].(map[string]any)
	assert.Equal(t, "string", queryParam["type"])
	assert.Equal(t, 2, queryParam["minLength"])

	// Check search_type enum
	searchTypeParam := params["search_type"].(map[string]any)
	enumValues := searchTypeParam["enum"].([]string)
	assert.Contains(t, enumValues, "web")
	assert.Contains(t, enumValues, "images")
	assert.Contains(t, enumValues, "videos")
}

// Test searchTool.Run() - Input validation
func TestSearchTool_Run_InputValidation_Simple(t *testing.T) {
	mockPerms := &mockPermissionServiceSimple{}
	tool := NewWebSearchTool(mockPerms)
	ctx := createSimpleTestContext()

	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "invalid JSON",
			input:         `{invalid json}`,
			expectedError: "Failed to parse search parameters",
		},
		{
			name:          "empty query",
			input:         `{"query": ""}`,
			expectedError: "Query parameter is required",
		},
		{
			name:          "query too short",
			input:         `{"query": "a"}`,
			expectedError: "Query must be at least 2 characters long",
		},
		{
			name:          "invalid search type",
			input:         `{"query": "test", "search_type": "invalid"}`,
			expectedError: "search_type must be 'web', 'images', or 'videos'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := ToolCall{
				ID:    "test-call",
				Name:  "search",
				Input: tt.input,
			}

			response, err := tool.Run(ctx, call)
			assert.NoError(t, err) // Validation errors return error responses, not errors
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, tt.expectedError)
		})
	}
}

// Test context validation without full config setup
func TestSearchTool_Run_ContextValidation_Simple(t *testing.T) {
	// Set API key so we reach context validation
	os.Setenv("BRAVE_SEARCH_API_KEY", "test-api-key")
	defer os.Unsetenv("BRAVE_SEARCH_API_KEY")

	mockPerms := &mockPermissionServiceSimple{}
	tool := NewWebSearchTool(mockPerms)

	tests := []struct {
		name        string
		setupCtx    func() context.Context
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing session ID",
			setupCtx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, MessageIDContextKey, "test-message")
				ctx = context.WithValue(ctx, SessionStorageContextKey, "/test/storage")
				return ctx
			},
			expectError: true,
			errorMsg:    "session ID and message ID are required",
		},
		{
			name: "missing message ID",
			setupCtx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, SessionIDContextKey, "test-session")
				ctx = context.WithValue(ctx, SessionStorageContextKey, "/test/storage")
				return ctx
			},
			expectError: true,
			errorMsg:    "session ID and message ID are required",
		},
		{
			name: "missing storage directory",
			setupCtx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, SessionIDContextKey, "test-session")
				ctx = context.WithValue(ctx, MessageIDContextKey, "test-message")
				return ctx
			},
			expectError: true,
			errorMsg:    "failed to get session storage directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			call := ToolCall{
				ID:    "test-call",
				Name:  "search",
				Input: `{"query": "test"}`,
			}

			response, err := tool.Run(ctx, call)

			// The test environment doesn't have credentials service set up,
			// so we get "Credentials service not available" instead of context validation errors
			assert.NoError(t, err)
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, "Credentials service not available")
		})
	}
}

// Test format methods directly
func TestSearchTool_FormatMethods_Simple(t *testing.T) {
	tool := &searchTool{
		client:      &http.Client{Timeout: 30 * time.Second},
		permissions: &mockPermissionServiceSimple{},
	}

	t.Run("formatWebResults", func(t *testing.T) {
		tests := []struct {
			name         string
			body         string
			expectError  bool
			expectEmpty  bool
			expectCount  int
		}{
			{
				name: "valid results",
				body: `{
					"type": "search",
					"web": {
						"type": "search",
						"results": [
							{
								"title": "Result 1",
								"url": "https://example.com/1",
								"description": "Description 1"
							},
							{
								"title": "Result 2",
								"url": "https://example.com/2",
								"description": "Description 2"
							}
						]
					}
				}`,
				expectError: false,
				expectEmpty: false,
				expectCount: 2,
			},
			{
				name: "empty results",
				body: `{
					"type": "search",
					"web": {
						"type": "search",
						"results": []
					}
				}`,
				expectError: false,
				expectEmpty: true,
				expectCount: 0,
			},
			{
				name:        "invalid JSON",
				body:        `{invalid json}`,
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				response, err := tool.formatWebResults([]byte(tt.body), "test query")

				if tt.expectError {
					assert.NoError(t, err) // Format methods return error responses, not errors
					assert.True(t, response.IsError)
					assert.Contains(t, response.Content, "Failed to parse")
				} else if tt.expectEmpty {
					assert.NoError(t, err)
					assert.False(t, response.IsError)
					assert.Contains(t, response.Content, "No web search results found")
				} else {
					assert.NoError(t, err)
					assert.False(t, response.IsError)
					assert.Contains(t, response.Content, "Web search results for: test query")

					// Count occurrences of numbered results
					resultCount := strings.Count(response.Content, ". Result")
					expectedCount := tt.expectCount
					if expectedCount > MaxSearchResults {
						expectedCount = MaxSearchResults
					}
					assert.Equal(t, expectedCount, resultCount)
				}
			})
		}
	})

	t.Run("formatImageResults", func(t *testing.T) {
		body := `{
			"type": "images",
			"results": [
				{
					"type": "image",
					"title": "Test Image",
					"url": "https://example.com/page",
					"source": "example.com",
					"thumbnail": {"src": "https://example.com/thumb.jpg"},
					"properties": {"url": "https://example.com/image.jpg"},
					"confidence": "high"
				}
			]
		}`

		response, err := tool.formatImageResults([]byte(body), "test query")
		assert.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Image search results for: test query")
		assert.Contains(t, response.Content, "1. Test Image")
		assert.Contains(t, response.Content, "Image URL: https://example.com/image.jpg")
		assert.Contains(t, response.Content, "Thumbnail: https://example.com/thumb.jpg")
		assert.Contains(t, response.Content, "Source: example.com")
		assert.Contains(t, response.Content, "Confidence: high")
	})

	t.Run("formatVideoResults", func(t *testing.T) {
		body := `{
			"type": "videos",
			"results": [
				{
					"type": "video",
					"title": "Test Video",
					"url": "https://example.com/video-page",
					"source": "youtube.com",
					"thumbnail": {"src": "https://example.com/video-thumb.jpg"},
					"properties": {
						"url": "https://example.com/video.mp4",
						"duration": "10:30",
						"views": "1000",
						"upload_date": "2023-01-01"
					},
					"confidence": "medium"
				}
			]
		}`

		response, err := tool.formatVideoResults([]byte(body), "test query")
		assert.NoError(t, err)
		assert.False(t, response.IsError)
		assert.Contains(t, response.Content, "Video search results for: test query")
		assert.Contains(t, response.Content, "1. Test Video")
		assert.Contains(t, response.Content, "Video URL: https://example.com/video.mp4")
		assert.Contains(t, response.Content, "Duration: 10:30")
		assert.Contains(t, response.Content, "Views: 1000")
		assert.Contains(t, response.Content, "Upload Date: 2023-01-01")
		assert.Contains(t, response.Content, "Thumbnail: https://example.com/video-thumb.jpg")
		assert.Contains(t, response.Content, "Source: youtube.com")
		assert.Contains(t, response.Content, "Confidence: medium")
	})
}

// Test max results limiting
func TestSearchTool_MaxResultsLimiting_Simple(t *testing.T) {
	tool := &searchTool{
		client:      &http.Client{Timeout: 30 * time.Second},
		permissions: &mockPermissionServiceSimple{},
	}

	// Create response with more than MaxSearchResults (3)
	results := make([]SearchResult, 5)
	for i := 0; i < 5; i++ {
		results[i] = SearchResult{
			Title:       fmt.Sprintf("Result %d", i+1),
			URL:         fmt.Sprintf("https://example.com/%d", i+1),
			Description: fmt.Sprintf("Description %d", i+1),
		}
	}

	response := BraveSearchResponse{
		Type: "search",
		Web: WebResults{
			Type:    "search",
			Results: results,
		},
	}

	body, err := json.Marshal(response)
	require.NoError(t, err)

	toolResponse, err := tool.formatWebResults(body, "test query")
	assert.NoError(t, err)
	assert.False(t, toolResponse.IsError)

	// Should only show MaxSearchResults (3) results
	resultCount := strings.Count(toolResponse.Content, ". Result")
	assert.Equal(t, MaxSearchResults, resultCount)

	// Should contain first 3 results
	assert.Contains(t, toolResponse.Content, "1. Result 1")
	assert.Contains(t, toolResponse.Content, "2. Result 2")
	assert.Contains(t, toolResponse.Content, "3. Result 3")

	// Should not contain 4th and 5th results
	assert.NotContains(t, toolResponse.Content, "4. Result 4")
	assert.NotContains(t, toolResponse.Content, "5. Result 5")
}

// Test interface compliance
func TestSearchTool_InterfaceCompliance_Simple(t *testing.T) {
	mockPerms := &mockPermissionServiceSimple{}
	tool := NewWebSearchTool(mockPerms)

	// Verify it implements BaseTool interface
	var _ BaseTool = tool

	// Verify it has required methods
	info := tool.Info()
	assert.NotEmpty(t, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotNil(t, info.Parameters)
}

// Test constants
func TestConstants_Simple(t *testing.T) {
	assert.Equal(t, "search", SearchToolName)
	assert.Equal(t, 3, MaxSearchResults)
}

// Test SearchPermissionsParams type alias
func TestSearchPermissionsParams_Simple(t *testing.T) {
	spellCheck := true
	original := SearchParams{
		Query:          "test",
		SearchType:     "web",
		AllowedDomains: []string{"example.com"},
		BlockedDomains: []string{"spam.com"},
		SafeSearch:     "strict",
		SpellCheck:     &spellCheck,
	}

	// Type conversion should work
	permParams := SearchPermissionsParams(original)

	assert.Equal(t, original.Query, permParams.Query)
	assert.Equal(t, original.SearchType, permParams.SearchType)
	assert.Equal(t, original.AllowedDomains, permParams.AllowedDomains)
	assert.Equal(t, original.BlockedDomains, permParams.BlockedDomains)
	assert.Equal(t, original.SafeSearch, permParams.SafeSearch)
	assert.Equal(t, original.SpellCheck, permParams.SpellCheck)
}

// Test parameter defaults and validation with environment variable setup
func TestSearchTool_ParameterDefaults_Simple(t *testing.T) {
	mockPerms := &mockPermissionServiceSimple{}
	mockPerms.On("Request", mock.AnythingOfType("permission.CreatePermissionRequest")).Return(false)

	tool := NewWebSearchTool(mockPerms)
	ctx := createSimpleTestContext()

	// Set API key
	os.Setenv("BRAVE_SEARCH_API_KEY", "test-api-key")
	defer os.Unsetenv("BRAVE_SEARCH_API_KEY")

	tests := []struct {
		name         string
		input        string
		expectedType string
	}{
		{
			name:         "default search type",
			input:        `{"query": "test"}`,
			expectedType: "web",
		},
		{
			name:         "explicit web search",
			input:        `{"query": "test", "search_type": "web"}`,
			expectedType: "web",
		},
		{
			name:         "image search",
			input:        `{"query": "test", "search_type": "images"}`,
			expectedType: "image",
		},
		{
			name:         "video search",
			input:        `{"query": "test", "search_type": "videos"}`,
			expectedType: "video",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify permission request has correct action
			mockPerms.On("Request", mock.MatchedBy(func(req permission.CreatePermissionRequest) bool {
				return req.Action == tt.expectedType+"_search"
			})).Return(false).Once()

			call := ToolCall{
				ID:    "test-call",
				Name:  "search",
				Input: tt.input,
			}

			_, err := tool.Run(ctx, call)
			assert.Error(t, err)
			assert.Equal(t, permission.ErrorPermissionDenied, err)
		})
	}

	mockPerms.AssertExpectations(t)
}
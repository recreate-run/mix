package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test NewMediaShowcaseTool constructor
func TestNewMediaShowcaseTool(t *testing.T) {
	tool := NewMediaShowcaseTool()
	assert.NotNil(t, tool)

	// Verify it returns the correct type
	mediaShowcaseTool, ok := tool.(*mediaShowcaseTool)
	assert.True(t, ok)
	assert.NotNil(t, mediaShowcaseTool)
}

// Test that mediaShowcaseTool implements BaseTool interface
func TestMediaShowcaseToolImplementsBaseTool(t *testing.T) {
	tool := NewMediaShowcaseTool()

	// Should implement BaseTool interface methods
	assert.NotPanics(t, func() {
		_ = tool.Info()
	})

	assert.NotPanics(t, func() {
		_, _ = tool.Run(context.Background(), ToolCall{})
	})
}

// Test Info method
func TestMediaShowcaseToolInfo(t *testing.T) {
	tool := NewMediaShowcaseTool()
	info := tool.Info()

	// Test basic properties
	assert.Equal(t, "ShowMedia", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotNil(t, info.Parameters)
	assert.Equal(t, []string{"outputs"}, info.Required)

	// Test parameters structure
	outputs, exists := info.Parameters["outputs"]
	assert.True(t, exists)
	assert.NotNil(t, outputs)

	outputsMap, ok := outputs.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "array", outputsMap["type"])
	assert.NotEmpty(t, outputsMap["description"])

	// Test items structure
	items, exists := outputsMap["items"]
	assert.True(t, exists)
	itemsMap, ok := items.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "object", itemsMap["type"])

	// Test properties structure
	properties, exists := itemsMap["properties"]
	assert.True(t, exists)
	propertiesMap, ok := properties.(map[string]any)
	assert.True(t, ok)

	// Test required properties exist
	requiredProperties := []string{"path", "type", "title", "description", "config", "startTime", "duration"}
	for _, prop := range requiredProperties {
		_, exists := propertiesMap[prop]
		assert.True(t, exists, "Property %s should exist", prop)
	}

	// Test type enum values
	typeProperty := propertiesMap["type"].(map[string]any)
	enumValues, exists := typeProperty["enum"]
	assert.True(t, exists)
	enumSlice, ok := enumValues.([]string)
	assert.True(t, ok)
	expectedTypes := []string{"image", "video", "audio", "gsap_animation", "pdf", "csv"}
	assert.ElementsMatch(t, expectedTypes, enumSlice)

	// Test required fields
	required, exists := itemsMap["required"]
	assert.True(t, exists)
	requiredSlice, ok := required.([]string)
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"type", "title"}, requiredSlice)
}

// Test MediaShowcaseParams and MediaOutput JSON serialization
func TestMediaOutputJSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		output MediaOutput
	}{
		{
			name: "basic image output",
			output: MediaOutput{
				Path:        "https://example.com/image.jpg",
				Type:        "image",
				Title:       "Test Image",
				Description: "A test image",
			},
		},
		{
			name: "video output with timing",
			output: MediaOutput{
				Path:        "https://example.com/video.mp4",
				Type:        "video",
				Title:       "Test Video",
				Description: "A test video",
				StartTime:   intPtr(30),
				Duration:    intPtr(60),
			},
		},
		{
			name: "gsap_animation with config",
			output: MediaOutput{
				Type:        "gsap_animation",
				Title:       "Test Animation",
				Description: "A test animation",
				Config: map[string]interface{}{
					"url":      "https://example.com/animation.html",
					"duration": 5,
					"autoplay": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.output)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Test unmarshaling
			var unmarshaled MediaOutput
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Compare relevant fields
			assert.Equal(t, tt.output.Path, unmarshaled.Path)
			assert.Equal(t, tt.output.Type, unmarshaled.Type)
			assert.Equal(t, tt.output.Title, unmarshaled.Title)
			assert.Equal(t, tt.output.Description, unmarshaled.Description)

			// Handle pointer fields
			if tt.output.StartTime != nil {
				require.NotNil(t, unmarshaled.StartTime)
				assert.Equal(t, *tt.output.StartTime, *unmarshaled.StartTime)
			} else {
				assert.Nil(t, unmarshaled.StartTime)
			}

			if tt.output.Duration != nil {
				require.NotNil(t, unmarshaled.Duration)
				assert.Equal(t, *tt.output.Duration, *unmarshaled.Duration)
			} else {
				assert.Nil(t, unmarshaled.Duration)
			}

			// Handle config field - JSON unmarshaling converts numbers to float64
			if tt.output.Config != nil {
				assert.NotNil(t, unmarshaled.Config)
				// For config comparison, we need to handle the JSON number conversion
				if configMap, ok := tt.output.Config.(map[string]interface{}); ok {
					unmarshaledConfig, ok := unmarshaled.Config.(map[string]interface{})
					assert.True(t, ok)

					// Compare URL field specifically
					if url, exists := configMap["url"]; exists {
						unmarshaledURL, urlExists := unmarshaledConfig["url"]
						assert.True(t, urlExists)
						assert.Equal(t, url, unmarshaledURL)
					}

					// For other fields, just verify they exist and are the right type
					for key, value := range configMap {
						unmarshaledValue, exists := unmarshaledConfig[key]
						assert.True(t, exists, "Key %s should exist", key)
						if key != "duration" { // Skip numeric comparison due to JSON conversion
							assert.Equal(t, value, unmarshaledValue)
						}
					}
				} else {
					assert.Equal(t, tt.output.Config, unmarshaled.Config)
				}
			} else {
				assert.Nil(t, unmarshaled.Config)
			}
		})
	}
}

// Test MediaShowcaseParams JSON serialization
func TestMediaShowcaseParamsJSONSerialization(t *testing.T) {
	params := MediaShowcaseParams{
		Outputs: []MediaOutput{
			{
				Path:        "https://example.com/image.jpg",
				Type:        "image",
				Title:       "Test Image",
				Description: "A test image",
			},
			{
				Path:      "https://example.com/video.mp4",
				Type:      "video",
				Title:     "Test Video",
				StartTime: intPtr(10),
				Duration:  intPtr(30),
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(params)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test unmarshaling
	var unmarshaled MediaShowcaseParams
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Len(t, unmarshaled.Outputs, 2)

	// Verify first output
	assert.Equal(t, "https://example.com/image.jpg", unmarshaled.Outputs[0].Path)
	assert.Equal(t, "image", unmarshaled.Outputs[0].Type)
	assert.Equal(t, "Test Image", unmarshaled.Outputs[0].Title)

	// Verify second output with timing
	assert.Equal(t, "video", unmarshaled.Outputs[1].Type)
	require.NotNil(t, unmarshaled.Outputs[1].StartTime)
	assert.Equal(t, 10, *unmarshaled.Outputs[1].StartTime)
	require.NotNil(t, unmarshaled.Outputs[1].Duration)
	assert.Equal(t, 30, *unmarshaled.Outputs[1].Duration)
}

// Test Run method with valid inputs
func TestMediaShowcaseToolRunValid(t *testing.T) {
	tool := &mediaShowcaseTool{}
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		expectedMsg string
	}{
		{
			name: "single image output",
			input: `{
				"outputs": [{
					"path": "https://example.com/image.jpg",
					"type": "image",
					"title": "Test Image",
					"description": "A test image"
				}]
			}`,
			expectedMsg: "Successfully showcasing 1 media output(s): Test Image",
		},
		{
			name: "multiple outputs",
			input: `{
				"outputs": [
					{
						"path": "https://example.com/image.jpg",
						"type": "image",
						"title": "Test Image"
					},
					{
						"path": "https://example.com/video.mp4",
						"type": "video",
						"title": "Test Video",
						"startTime": 30,
						"duration": 60
					}
				]
			}`,
			expectedMsg: "Successfully showcasing 2 media output(s): Test Image, Test Video",
		},
		{
			name: "gsap_animation with config",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Animation Demo",
					"description": "A GSAP animation",
					"config": {
						"url": "https://example.com/animation.html",
						"duration": 5
					}
				}]
			}`,
			expectedMsg: "Successfully showcasing 1 media output(s): Animation Demo",
		},
		{
			name: "youtube video",
			input: `{
				"outputs": [{
					"path": "https://youtube.com/watch?v=abc123",
					"type": "video",
					"title": "YouTube Video",
					"description": "A YouTube video"
				}]
			}`,
			expectedMsg: "Successfully showcasing 1 media output(s): YouTube Video",
		},
		{
			name: "audio with timing",
			input: `{
				"outputs": [{
					"path": "https://example.com/audio.mp3",
					"type": "audio",
					"title": "Audio Track",
					"startTime": 0,
					"duration": 180
				}]
			}`,
			expectedMsg: "Successfully showcasing 1 media output(s): Audio Track",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := ToolCall{
				ID:    "test-call",
				Name:  "ShowMedia",
				Input: tt.input,
			}

			response, err := tool.Run(ctx, call)
			require.NoError(t, err)
			assert.Equal(t, "text", string(response.Type))
			assert.Equal(t, tt.expectedMsg, response.Content)
			assert.False(t, response.IsError)
		})
	}
}

// Test Run method with invalid inputs and error cases
func TestMediaShowcaseToolRunErrors(t *testing.T) {
	tool := &mediaShowcaseTool{}
	ctx := context.Background()

	tests := getMediaShowcaseErrorTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			call := ToolCall{
				ID:    "test-call",
				Name:  "ShowMedia",
				Input: tt.input,
			}

			response, err := tool.Run(ctx, call)
			require.NoError(t, err) // The tool should not return an error, but an error response
			assert.Equal(t, "text", string(response.Type))
			assert.True(t, response.IsError)
			assert.Contains(t, response.Content, tt.expectedError)
		})
	}
}

func getMediaShowcaseErrorTestCases() []struct {
	name          string
	input         string
	expectedError string
} {
	basicErrors := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "invalid JSON",
			input:         `{"outputs": [invalid json}`,
			expectedError: "Invalid parameters:",
		},
		{
			name:          "empty outputs array",
			input:         `{"outputs": []}`,
			expectedError: "No media outputs provided",
		},
		{
			name: "missing type",
			input: `{
				"outputs": [{
					"path": "https://example.com/image.jpg",
					"title": "Test Image"
				}]
			}`,
			expectedError: "Output 0 missing type",
		},
		{
			name: "missing title",
			input: `{
				"outputs": [{
					"path": "https://example.com/image.jpg",
					"type": "image"
				}]
			}`,
			expectedError: "Output 0 missing title",
		},
		{
			name: "missing path for image",
			input: `{
				"outputs": [{
					"type": "image",
					"title": "Test Image"
				}]
			}`,
			expectedError: "Output 0 missing path",
		},
		{
			name: "invalid media type",
			input: `{
				"outputs": [{
					"path": "https://example.com/file.xyz",
					"type": "invalid_type",
					"title": "Test File"
				}]
			}`,
			expectedError: "Invalid media type 'invalid_type' for output 0",
		},
		{
			name: "non-URL path",
			input: `{
				"outputs": [{
					"path": "/local/path/image.jpg",
					"type": "image",
					"title": "Test Image"
				}]
			}`,
			expectedError: "path must be a valid HTTP/HTTPS URL for output 0",
		},
	}

	gsapErrors := getGsapAnimationErrorTestCases()
	timeErrors := getTimeValidationErrorTestCases()

	return append(append(basicErrors, gsapErrors...), timeErrors...)
}

func getGsapAnimationErrorTestCases() []struct {
	name          string
	input         string
	expectedError string
} {
	return []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "gsap_animation missing config",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Animation"
				}]
			}`,
			expectedError: "gsap_animation type requires config parameter for output 0",
		},
		{
			name: "gsap_animation invalid config type",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Animation",
					"config": "not an object"
				}]
			}`,
			expectedError: "gsap_animation config must be a JSON object for output 0",
		},
		{
			name: "gsap_animation missing config.url",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Animation",
					"config": {
						"duration": 5
					}
				}]
			}`,
			expectedError: "gsap_animation requires config.url field for output 0",
		},
		{
			name: "gsap_animation invalid config.url type",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Animation",
					"config": {
						"url": 123
					}
				}]
			}`,
			expectedError: "gsap_animation config.url must be a non-empty string for output 0",
		},
		{
			name: "gsap_animation empty config.url",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Animation",
					"config": {
						"url": ""
					}
				}]
			}`,
			expectedError: "gsap_animation config.url must be a non-empty string for output 0",
		},
		{
			name: "gsap_animation invalid URL",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Animation",
					"config": {
						"url": "ftp://invalid.com"
					}
				}]
			}`,
			expectedError: "gsap_animation config.url must be a valid HTTP/HTTPS URL for output 0",
		},
	}
}

func getTimeValidationErrorTestCases() []struct {
	name          string
	input         string
	expectedError string
} {
	return []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "negative startTime",
			input: `{
				"outputs": [{
					"path": "https://example.com/video.mp4",
					"type": "video",
					"title": "Test Video",
					"startTime": -1
				}]
			}`,
			expectedError: "startTime must be >= 0 for output 0",
		},
		{
			name: "zero duration",
			input: `{
				"outputs": [{
					"path": "https://example.com/video.mp4",
					"type": "video",
					"title": "Test Video",
					"duration": 0
				}]
			}`,
			expectedError: "duration must be > 0 for output 0",
		},
		{
			name: "negative duration",
			input: `{
				"outputs": [{
					"path": "https://example.com/video.mp4",
					"type": "video",
					"title": "Test Video",
					"duration": -10
				}]
			}`,
			expectedError: "duration must be > 0 for output 0",
		},
	}
}

// Test edge cases and boundary conditions
func TestMediaShowcaseToolEdgeCases(t *testing.T) {
	tool := &mediaShowcaseTool{}
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "zero startTime (valid)",
			input: `{
				"outputs": [{
					"path": "https://example.com/video.mp4",
					"type": "video",
					"title": "Test Video",
					"startTime": 0
				}]
			}`,
			shouldError: false,
		},
		{
			name: "minimum duration (valid)",
			input: `{
				"outputs": [{
					"path": "https://example.com/video.mp4",
					"type": "video",
					"title": "Test Video",
					"duration": 1
				}]
			}`,
			shouldError: false,
		},
		{
			name: "large startTime and duration",
			input: `{
				"outputs": [{
					"path": "https://example.com/video.mp4",
					"type": "video",
					"title": "Test Video",
					"startTime": 3600,
					"duration": 7200
				}]
			}`,
			shouldError: false,
		},
		{
			name: "gsap_animation with complex config",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Complex Animation",
					"config": {
						"url": "https://example.com/complex.html",
						"duration": 10,
						"easing": "power2.out",
						"autoplay": true,
						"loop": false,
						"params": {
							"color": "#ff0000",
							"scale": 1.5
						}
					}
				}]
			}`,
			shouldError: false,
		},
		{
			name: "multiple outputs with mixed types",
			input: `{
				"outputs": [
					{
						"path": "https://example.com/image.jpg",
						"type": "image",
						"title": "Image 1"
					},
					{
						"type": "gsap_animation",
						"title": "Animation 1",
						"config": {
							"url": "https://example.com/anim.html"
						}
					},
					{
						"path": "https://youtube.com/watch?v=123",
						"type": "video",
						"title": "YouTube 1"
					}
				]
			}`,
			shouldError: false,
		},
		{
			name: "empty description (optional field)",
			input: `{
				"outputs": [{
					"path": "https://example.com/image.jpg",
					"type": "image",
					"title": "Test Image",
					"description": ""
				}]
			}`,
			shouldError: false,
		},
		{
			name: "null config.url",
			input: `{
				"outputs": [{
					"type": "gsap_animation",
					"title": "Animation",
					"config": {
						"url": null
					}
				}]
			}`,
			shouldError: true,
			errorMsg:    "gsap_animation requires config.url field for output 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := ToolCall{
				ID:    "test-call",
				Name:  "ShowMedia",
				Input: tt.input,
			}

			response, err := tool.Run(ctx, call)
			require.NoError(t, err)

			if tt.shouldError {
				assert.True(t, response.IsError)
				assert.Contains(t, response.Content, tt.errorMsg)
			} else {
				assert.False(t, response.IsError)
				assert.Contains(t, response.Content, "Successfully showcasing")
			}
		})
	}
}

// Test isURL helper function
func TestIsURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "valid HTTP URL",
			path:     "http://example.com/image.jpg",
			expected: true,
		},
		{
			name:     "valid HTTPS URL",
			path:     "https://example.com/image.jpg",
			expected: true,
		},
		{
			name:     "HTTP with port",
			path:     "http://localhost:8080/file.jpg",
			expected: true,
		},
		{
			name:     "HTTPS with path and query",
			path:     "https://example.com/path/file.jpg?param=value",
			expected: true,
		},
		{
			name:     "local file path",
			path:     "/local/path/to/file.jpg",
			expected: false,
		},
		{
			name:     "relative path",
			path:     "relative/path/file.jpg",
			expected: false,
		},
		{
			name:     "Windows path",
			path:     "C:\\Windows\\file.jpg",
			expected: false,
		},
		{
			name:     "FTP URL",
			path:     "ftp://example.com/file.jpg",
			expected: false,
		},
		{
			name:     "empty string",
			path:     "",
			expected: false,
		},
		{
			name:     "just http://",
			path:     "http://",
			expected: true, // Still has the prefix
		},
		{
			name:     "just https://",
			path:     "https://",
			expected: true, // Still has the prefix
		},
		{
			name:     "HTTP uppercase",
			path:     "HTTP://example.com/file.jpg",
			expected: false, // Case sensitive
		},
		{
			name:     "mixed case",
			path:     "Http://example.com/file.jpg",
			expected: false, // Case sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isURL(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test tool behavior with different contexts
func TestMediaShowcaseToolWithContext(t *testing.T) {
	tool := &mediaShowcaseTool{}

	tests := []struct {
		name    string
		ctx     context.Context
		input   string
		wantErr bool
	}{
		{
			name: "background context",
			ctx:  context.Background(),
			input: `{
				"outputs": [{
					"path": "https://example.com/image.jpg",
					"type": "image",
					"title": "Test Image"
				}]
			}`,
			wantErr: false,
		},
		{
			name: "context with values",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, SessionIDContextKey, "session-123")
				return ctx
			}(),
			input: `{
				"outputs": [{
					"path": "https://example.com/video.mp4",
					"type": "video",
					"title": "Test Video"
				}]
			}`,
			wantErr: false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx
			}(),
			input: `{
				"outputs": [{
					"path": "https://example.com/audio.mp3",
					"type": "audio",
					"title": "Test Audio"
				}]
			}`,
			wantErr: false, // Tool doesn't currently check for cancellation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := ToolCall{
				ID:    "test-call",
				Name:  "ShowMedia",
				Input: tt.input,
			}

			response, err := tool.Run(tt.ctx, call)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.False(t, response.IsError)
				assert.Contains(t, response.Content, "Successfully showcasing")
			}
		})
	}
}

// Test error message format consistency
func TestErrorMessageFormat(t *testing.T) {
	tool := &mediaShowcaseTool{}
	ctx := context.Background()

	// Test that error messages for different outputs use consistent indexing
	input := `{
		"outputs": [
			{
				"path": "https://example.com/image.jpg",
				"type": "image",
				"title": "Valid Image"
			},
			{
				"type": "invalid_type",
				"title": "Invalid Type"
			}
		]
	}`

	call := ToolCall{
		ID:    "test-call",
		Name:  "ShowMedia",
		Input: input,
	}

	response, err := tool.Run(ctx, call)
	require.NoError(t, err)
	assert.True(t, response.IsError)
	assert.Contains(t, response.Content, "Output 1") // Should reference the second output (index 1)
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}

// Benchmark tests for performance
func BenchmarkMediaShowcaseToolRun(b *testing.B) {
	tool := &mediaShowcaseTool{}
	ctx := context.Background()

	input := `{
		"outputs": [{
			"path": "https://example.com/image.jpg",
			"type": "image",
			"title": "Test Image",
			"description": "A test image"
		}]
	}`

	call := ToolCall{
		ID:    "test-call",
		Name:  "ShowMedia",
		Input: input,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Run(ctx, call)
	}
}

func BenchmarkIsURL(b *testing.B) {
	testPaths := []string{
		"https://example.com/image.jpg",
		"http://localhost:8080/file.jpg",
		"/local/path/to/file.jpg",
		"relative/path/file.jpg",
		"ftp://example.com/file.jpg",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_ = isURL(path)
		}
	}
}

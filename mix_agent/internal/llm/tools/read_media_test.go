package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/llm/interfaces"
	"mix/internal/message"
)

// MockProvider is a mock implementation of interfaces.Provider for testing
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) SendMessages(ctx context.Context, messages []message.Message, options interface{}) (*message.Message, error) {
	args := m.Called(ctx, messages, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*message.Message), args.Error(1)
}

func (m *MockProvider) SendMessagesStream(ctx context.Context, messages []message.Message, options interface{}, responseChannel chan<- message.Message) error {
	args := m.Called(ctx, messages, options, responseChannel)
	return args.Error(0)
}

func (m *MockProvider) GetMaxTokens() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockProvider) SetMaxTokens(maxTokens int) {
	m.Called(maxTokens)
}

func (m *MockProvider) GetModel() string {
	args := m.Called()
	return args.String(0)
}

// Test struct creation and JSON serialization
func TestReadMediaParams_JSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		params ReadMediaParams
	}{
		{
			name: "complete params",
			params: ReadMediaParams{
				FilePath:      "/path/to/image.jpg",
				DirectoryPath: "",
				MediaType:     "image",
				Prompt:        "Analyze this image",
				Recursive:     true,
				WordCount:     300,
				AudioMode:     "",
				VideoMode:     "",
			},
		},
		{
			name: "audio params",
			params: ReadMediaParams{
				DirectoryPath: "/path/to/audio",
				MediaType:     "audio",
				Prompt:        "Transcribe this audio",
				Recursive:     false,
				WordCount:     500,
				AudioMode:     "transcript",
			},
		},
		{
			name: "video params",
			params: ReadMediaParams{
				FilePath:  "/path/to/video.mp4",
				MediaType: "video",
				Prompt:    "Describe this video",
				VideoMode: "description",
			},
		},
		{
			name: "minimal params",
			params: ReadMediaParams{
				FilePath:  "/minimal.png",
				MediaType: "image",
				Prompt:    "Analyze",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.params)
			require.NoError(t, err)

			// Test JSON unmarshaling
			var unmarshaled ReadMediaParams
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Verify fields are preserved
			assert.Equal(t, tt.params.FilePath, unmarshaled.FilePath)
			assert.Equal(t, tt.params.DirectoryPath, unmarshaled.DirectoryPath)
			assert.Equal(t, tt.params.MediaType, unmarshaled.MediaType)
			assert.Equal(t, tt.params.Prompt, unmarshaled.Prompt)
			assert.Equal(t, tt.params.Recursive, unmarshaled.Recursive)
			assert.Equal(t, tt.params.WordCount, unmarshaled.WordCount)
			assert.Equal(t, tt.params.AudioMode, unmarshaled.AudioMode)
			assert.Equal(t, tt.params.VideoMode, unmarshaled.VideoMode)
		})
	}
}

func TestReadMediaResult_JSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		result ReadMediaResult
	}{
		{
			name: "successful result",
			result: ReadMediaResult{
				FilePath:  "/path/to/file.jpg",
				MediaType: "image",
				Analysis:  "This is a detailed analysis of the image.",
				Error:     "",
			},
		},
		{
			name: "error result",
			result: ReadMediaResult{
				FilePath:  "/path/to/file.mp3",
				MediaType: "audio",
				Analysis:  "",
				Error:     "Failed to process audio file",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			var unmarshaled ReadMediaResult
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, tt.result, unmarshaled)
		})
	}
}

func TestReadMediaResponse_JSONSerialization(t *testing.T) {
	response := ReadMediaResponse{
		Results: []ReadMediaResult{
			{
				FilePath:  "/file1.jpg",
				MediaType: "image",
				Analysis:  "Analysis 1",
			},
			{
				FilePath:  "/file2.mp4",
				MediaType: "video",
				Analysis:  "Analysis 2",
			},
		},
		Summary: "Successfully analyzed 2 files.",
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var unmarshaled ReadMediaResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response, unmarshaled)
}

// Test NewReadMediaTool function
func TestNewReadMediaTool(t *testing.T) {
	tool := NewReadMediaTool()
	assert.NotNil(t, tool)
	assert.Implements(t, (*interfaces.BaseTool)(nil), tool)
}

// Test tool Info method
func TestReadMediaTool_Info(t *testing.T) {
	tool := &readMediaTool{}
	info := tool.Info()

	assert.Equal(t, ReadMediaToolName, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotEmpty(t, info.Parameters)
	assert.Contains(t, info.Required, "media_type")
	assert.Contains(t, info.Required, "prompt")

	// Test parameter definitions
	params := info.Parameters
	assert.Contains(t, params, "file_path")
	assert.Contains(t, params, "directory_path")
	assert.Contains(t, params, "media_type")
	assert.Contains(t, params, "prompt")
	assert.Contains(t, params, "recursive")
	assert.Contains(t, params, "word_count")
	assert.Contains(t, params, "audio_mode")
	assert.Contains(t, params, "video_mode")

	// Test media_type enum values
	mediaTypeParam := params["media_type"].(map[string]any)
	enumValues := mediaTypeParam["enum"].([]string)
	assert.Contains(t, enumValues, "image")
	assert.Contains(t, enumValues, "audio")
	assert.Contains(t, enumValues, "video")

	// Test word_count limits
	wordCountParam := params["word_count"].(map[string]any)
	assert.Equal(t, MinWordCount, wordCountParam["minimum"])
	assert.Equal(t, MaxWordCount, wordCountParam["maximum"])
	assert.Equal(t, DefaultWordCount, wordCountParam["default"])
}

// Test parameter validation
func TestReadMediaTool_ValidateParams(t *testing.T) {
	tool := &readMediaTool{}

	tests := []struct {
		name        string
		params      ReadMediaParams
		expectError bool
		errorContains string
	}{
		{
			name: "valid image params",
			params: ReadMediaParams{
				FilePath:  "/absolute/path/image.jpg",
				MediaType: "image",
				Prompt:    "Analyze this image",
			},
			expectError: false,
		},
		{
			name: "valid audio params with transcript mode",
			params: ReadMediaParams{
				FilePath:  "/absolute/path/audio.mp3",
				MediaType: "audio",
				Prompt:    "Transcribe this audio",
				AudioMode: "transcript",
			},
			expectError: false,
		},
		{
			name: "valid video params",
			params: ReadMediaParams{
				FilePath:  "/absolute/path/video.mp4",
				MediaType: "video",
				Prompt:    "Describe this video",
				VideoMode: "description",
			},
			expectError: false,
		},
		{
			name: "valid directory params",
			params: ReadMediaParams{
				DirectoryPath: "/absolute/path/images",
				MediaType:     "image",
				Prompt:        "Analyze images",
				Recursive:     true,
			},
			expectError: false,
		},
		{
			name: "both file and directory path",
			params: ReadMediaParams{
				FilePath:      "/path/file.jpg",
				DirectoryPath: "/path/dir",
				MediaType:     "image",
				Prompt:        "Analyze",
			},
			expectError:   true,
			errorContains: "cannot specify both file_path and directory_path",
		},
		{
			name: "no file or directory path",
			params: ReadMediaParams{
				MediaType: "image",
				Prompt:    "Analyze",
			},
			expectError:   true,
			errorContains: "must specify either file_path or directory_path",
		},
		{
			name: "invalid media type",
			params: ReadMediaParams{
				FilePath:  "/path/file.txt",
				MediaType: "document",
				Prompt:    "Analyze",
			},
			expectError:   true,
			errorContains: "media_type must be 'image', 'audio', or 'video'",
		},
		{
			name: "audio without audio_mode",
			params: ReadMediaParams{
				FilePath:  "/path/audio.mp3",
				MediaType: "audio",
				Prompt:    "Analyze",
			},
			expectError:   true,
			errorContains: "audio_mode is required when media_type is 'audio'",
		},
		{
			name: "video without video_mode",
			params: ReadMediaParams{
				FilePath:  "/path/video.mp4",
				MediaType: "video",
				Prompt:    "Analyze",
			},
			expectError:   true,
			errorContains: "video_mode is required when media_type is 'video'",
		},
		{
			name: "invalid audio mode",
			params: ReadMediaParams{
				FilePath:  "/path/audio.mp3",
				MediaType: "audio",
				Prompt:    "Analyze",
				AudioMode: "invalid",
			},
			expectError:   true,
			errorContains: "audio_mode must be 'transcript' or 'description'",
		},
		{
			name: "invalid video mode",
			params: ReadMediaParams{
				FilePath:  "/path/video.mp4",
				MediaType: "video",
				Prompt:    "Analyze",
				VideoMode: "invalid",
			},
			expectError:   true,
			errorContains: "video_mode must be 'description'",
		},
		{
			name: "word count too low",
			params: ReadMediaParams{
				FilePath:  "/path/image.jpg",
				MediaType: "image",
				Prompt:    "Analyze",
				WordCount: MinWordCount - 1,
			},
			expectError:   true,
			errorContains: fmt.Sprintf("word_count must be between %d and %d", MinWordCount, MaxWordCount),
		},
		{
			name: "word count too high",
			params: ReadMediaParams{
				FilePath:  "/path/image.jpg",
				MediaType: "image",
				Prompt:    "Analyze",
				WordCount: MaxWordCount + 1,
			},
			expectError:   true,
			errorContains: fmt.Sprintf("word_count must be between %d and %d", MinWordCount, MaxWordCount),
		},
		{
			name: "relative file path",
			params: ReadMediaParams{
				FilePath:  "relative/path/image.jpg",
				MediaType: "image",
				Prompt:    "Analyze",
			},
			expectError:   true,
			errorContains: "file_path and directory_path must be absolute paths",
		},
		{
			name: "relative directory path",
			params: ReadMediaParams{
				DirectoryPath: "relative/path",
				MediaType:     "image",
				Prompt:        "Analyze",
			},
			expectError:   true,
			errorContains: "file_path and directory_path must be absolute paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.validateParams(tt.params)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test supported file detection
func TestReadMediaTool_IsSupportedFile(t *testing.T) {
	tool := &readMediaTool{}

	tests := []struct {
		name      string
		filePath  string
		mediaType string
		expected  bool
	}{
		// Image files
		{"jpg image", "/path/file.jpg", "image", true},
		{"jpeg image", "/path/file.jpeg", "image", true},
		{"png image", "/path/file.PNG", "image", true}, // Case insensitive
		{"gif image", "/path/file.gif", "image", true},
		{"webp image", "/path/file.webp", "image", true},
		{"bmp image", "/path/file.bmp", "image", true},
		{"unsupported image", "/path/file.tiff", "image", false},

		// Audio files
		{"mp3 audio", "/path/file.mp3", "audio", true},
		{"wav audio", "/path/file.WAV", "audio", true}, // Case insensitive
		{"m4a audio", "/path/file.m4a", "audio", true},
		{"aac audio", "/path/file.aac", "audio", true},
		{"ogg audio", "/path/file.ogg", "audio", true},
		{"flac audio", "/path/file.flac", "audio", true},
		{"unsupported audio", "/path/file.mid", "audio", false},

		// Video files
		{"mp4 video", "/path/file.mp4", "video", true},
		{"avi video", "/path/file.AVI", "video", true}, // Case insensitive
		{"mov video", "/path/file.mov", "video", true},
		{"wmv video", "/path/file.wmv", "video", true},
		{"flv video", "/path/file.flv", "video", true},
		{"webm video", "/path/file.webm", "video", true},
		{"mkv video", "/path/file.mkv", "video", true},
		{"unsupported video", "/path/file.3gp", "video", false},

		// Cross-type checks
		{"image as audio", "/path/file.jpg", "audio", false},
		{"audio as video", "/path/file.mp3", "video", false},
		{"video as image", "/path/file.mp4", "image", false},

		// Edge cases
		{"no extension", "/path/file", "image", false},
		{"empty path", "", "image", false},
		{"just extension", ".jpg", "image", true},
		{"multiple extensions", "/path/file.backup.jpg", "image", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.isSupportedFile(tt.filePath, tt.mediaType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test file discovery in directories
func TestReadMediaTool_FindSupportedFiles(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir := t.TempDir()

	// Create test directory structure
	imageDir := filepath.Join(tempDir, "images")
	audioDir := filepath.Join(tempDir, "audio")
	nestedDir := filepath.Join(imageDir, "nested")

	require.NoError(t, os.MkdirAll(imageDir, 0755))
	require.NoError(t, os.MkdirAll(audioDir, 0755))
	require.NoError(t, os.MkdirAll(nestedDir, 0755))

	// Create test files
	testFiles := map[string]string{
		filepath.Join(imageDir, "image1.jpg"):     "",
		filepath.Join(imageDir, "image2.png"):     "",
		filepath.Join(imageDir, "document.txt"):   "", // Should be ignored
		filepath.Join(audioDir, "audio1.mp3"):     "",
		filepath.Join(audioDir, "audio2.wav"):     "",
		filepath.Join(nestedDir, "nested.jpg"):    "",
		filepath.Join(nestedDir, "nested.gif"):    "",
	}

	for filePath := range testFiles {
		file, err := os.Create(filePath)
		require.NoError(t, err)
		file.Close()
	}

	tool := &readMediaTool{}

	tests := []struct {
		name         string
		dirPath      string
		mediaType    string
		recursive    bool
		expectedCount int
		expectedFiles []string
	}{
		{
			name:         "image files non-recursive",
			dirPath:      imageDir,
			mediaType:    "image",
			recursive:    false,
			expectedCount: 2,
			expectedFiles: []string{"image1.jpg", "image2.png"},
		},
		{
			name:         "image files recursive",
			dirPath:      imageDir,
			mediaType:    "image",
			recursive:    true,
			expectedCount: 4,
			expectedFiles: []string{"image1.jpg", "image2.png", "nested.jpg", "nested.gif"},
		},
		{
			name:         "audio files",
			dirPath:      audioDir,
			mediaType:    "audio",
			recursive:    false,
			expectedCount: 2,
			expectedFiles: []string{"audio1.mp3", "audio2.wav"},
		},
		{
			name:         "no video files",
			dirPath:      tempDir,
			mediaType:    "video",
			recursive:    true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := tool.findSupportedFiles(tt.dirPath, tt.mediaType, tt.recursive)
			require.NoError(t, err)

			assert.Len(t, files, tt.expectedCount)

			// Check that expected files are found
			for _, expectedFile := range tt.expectedFiles {
				found := false
				for _, actualFile := range files {
					if strings.Contains(actualFile, expectedFile) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected file %s not found in results", expectedFile)
			}
		})
	}
}

// Test getFilesToProcess method
func TestReadMediaTool_GetFilesToProcess(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	imageFile := filepath.Join(tempDir, "test.jpg")
	audioFile := filepath.Join(tempDir, "test.mp3")
	textFile := filepath.Join(tempDir, "test.txt")

	for _, filePath := range []string{imageFile, audioFile, textFile} {
		file, err := os.Create(filePath)
		require.NoError(t, err)
		file.Close()
	}

	tool := &readMediaTool{}

	tests := []struct {
		name          string
		params        ReadMediaParams
		expectedCount int
		expectError   bool
	}{
		{
			name: "single supported file",
			params: ReadMediaParams{
				FilePath:  imageFile,
				MediaType: "image",
			},
			expectedCount: 1,
		},
		{
			name: "single unsupported file",
			params: ReadMediaParams{
				FilePath:  textFile,
				MediaType: "image",
			},
			expectedCount: 0,
		},
		{
			name: "directory with supported files",
			params: ReadMediaParams{
				DirectoryPath: tempDir,
				MediaType:     "image",
			},
			expectedCount: 1,
		},
		{
			name: "nonexistent file",
			params: ReadMediaParams{
				FilePath:  "/nonexistent/file.jpg",
				MediaType: "image",
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := tool.getFilesToProcess(tt.params)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, files, tt.expectedCount)
			}
		})
	}
}

// Test readFileContent method
func TestReadMediaTool_ReadFileContent(t *testing.T) {
	tool := &readMediaTool{}

	// Create a temporary file with test content
	tempFile, err := os.CreateTemp("", "test_image*.jpg")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	testContent := []byte("fake image content")
	_, err = tempFile.Write(testContent)
	require.NoError(t, err)
	tempFile.Close()

	t.Run("successful read", func(t *testing.T) {
		data, mimeType, err := tool.readFileContent(tempFile.Name())
		require.NoError(t, err)
		assert.Equal(t, testContent, data)
		assert.Contains(t, mimeType, "image")
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, _, err := tool.readFileContent("/nonexistent/file.jpg")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file")
	})
}

// Test MIME type detection
func TestReadMediaTool_MIMETypeDetection(t *testing.T) {
	tool := &readMediaTool{}

	tests := []struct {
		filename     string
		expectedType string
	}{
		{"test.jpg", "image"},
		{"test.png", "image"},
		{"test.mp3", "audio"},
		{"test.wav", "audio"},
		{"test.mp4", "video"},
		{"test.avi", "video"},
		{"test.unknown", "application"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			// Create temporary file
			tempFile, err := os.CreateTemp("", tt.filename)
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())

			tempFile.Write([]byte("test content"))
			tempFile.Close()

			_, mimeType, err := tool.readFileContent(tempFile.Name())
			require.NoError(t, err)
			assert.Contains(t, mimeType, tt.expectedType)
		})
	}
}

// Test buildAnalysisPrompt method
func TestReadMediaTool_BuildAnalysisPrompt(t *testing.T) {
	tool := &readMediaTool{}

	tests := []struct {
		name     string
		params   ReadMediaParams
		contains []string
	}{
		{
			name: "image analysis",
			params: ReadMediaParams{
				MediaType: "image",
				Prompt:    "What do you see?",
				WordCount: 150,
			},
			contains: []string{"Analyze this image", "What do you see?", "150 words"},
		},
		{
			name: "audio transcript",
			params: ReadMediaParams{
				MediaType: "audio",
				AudioMode: "transcript",
				Prompt:    "Convert to text",
			},
			contains: []string{"accurate transcription", "Convert to text", "200 words"}, // Default word count
		},
		{
			name: "audio description",
			params: ReadMediaParams{
				MediaType: "audio",
				AudioMode: "description",
				Prompt:    "Describe the sound",
				WordCount: 300,
			},
			contains: []string{"Analyze this audio", "Describe the sound", "300 words"},
		},
		{
			name: "video analysis",
			params: ReadMediaParams{
				MediaType: "video",
				Prompt:    "Summarize the content",
				WordCount: 400,
			},
			contains: []string{"Analyze this video", "visual and audio elements", "Summarize the content", "400 words"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := tool.buildAnalysisPrompt(tt.params)

			for _, expectedText := range tt.contains {
				assert.Contains(t, prompt, expectedText)
			}
		})
	}
}

// Test generateSummary method
func TestReadMediaTool_GenerateSummary(t *testing.T) {
	tool := &readMediaTool{}

	tests := []struct {
		name     string
		results  []ReadMediaResult
		expected string
	}{
		{
			name:     "no results",
			results:  []ReadMediaResult{},
			expected: "No files were analyzed.",
		},
		{
			name: "all successful",
			results: []ReadMediaResult{
				{FilePath: "/file1.jpg", Analysis: "analysis1"},
				{FilePath: "/file2.png", Analysis: "analysis2"},
			},
			expected: "Successfully analyzed 2 file(s).",
		},
		{
			name: "all failed",
			results: []ReadMediaResult{
				{FilePath: "/file1.jpg", Error: "error1"},
				{FilePath: "/file2.png", Error: "error2"},
			},
			expected: "Analyzed 0 file(s) successfully, 2 failed.",
		},
		{
			name: "mixed results",
			results: []ReadMediaResult{
				{FilePath: "/file1.jpg", Analysis: "analysis1"},
				{FilePath: "/file2.png", Error: "error1"},
				{FilePath: "/file3.gif", Analysis: "analysis2"},
			},
			expected: "Analyzed 2 file(s) successfully, 1 failed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tool.generateSummary(tt.results)
			assert.Equal(t, tt.expected, summary)
		})
	}
}

// Test contains helper function
func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "item does not exist",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "grape",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
		{
			name:     "case sensitive",
			slice:    []string{"Apple", "Banana"},
			item:     "apple",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test constants
func TestReadMediaConstants(t *testing.T) {
	assert.Equal(t, "ReadMedia", ReadMediaToolName)
	assert.Equal(t, 200, DefaultWordCount)
	assert.Equal(t, 1000, MaxWordCount)
	assert.Equal(t, 50, MinWordCount)

	// Test supported file type constants
	assert.Contains(t, supportedImageTypes, ".jpg")
	assert.Contains(t, supportedImageTypes, ".png")
	assert.Contains(t, supportedAudioTypes, ".mp3")
	assert.Contains(t, supportedAudioTypes, ".wav")
	assert.Contains(t, supportedVideoTypes, ".mp4")
	assert.Contains(t, supportedVideoTypes, ".avi")
}

// Test interface compliance
func TestReadMediaTool_InterfaceCompliance(t *testing.T) {
	tool := NewReadMediaTool()

	// Test that tool implements BaseTool interface
	_, ok := tool.(interfaces.BaseTool)
	assert.True(t, ok, "ReadMedia tool should implement BaseTool interface")

	// Test Info method returns valid ToolInfo
	info := tool.Info()
	assert.NotEmpty(t, info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotEmpty(t, info.Parameters)
	assert.NotEmpty(t, info.Required)
}

// Test Run method with mocked dependencies
func TestReadMediaTool_Run_MockedDependencies(t *testing.T) {
	// This test focuses on the Run method flow without hitting external dependencies
	// We'll test parameter validation, context handling, and response generation

	tool := &readMediaTool{}

	// Test invalid JSON input
	t.Run("invalid JSON input", func(t *testing.T) {
		ctx := context.Background()
		call := interfaces.ToolCall{
			ID:    "test-1",
			Name:  "ReadMedia",
			Input: "invalid json",
		}

		response, err := tool.Run(ctx, call)
		assert.NoError(t, err) // Error is captured in response, not returned
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "error parsing parameters")
	})

	// Test parameter validation errors
	t.Run("parameter validation error", func(t *testing.T) {
		ctx := context.Background()
		params := ReadMediaParams{
			MediaType: "invalid",
			Prompt:    "test",
		}

		paramsJSON, _ := json.Marshal(params)
		call := interfaces.ToolCall{
			ID:    "test-2",
			Name:  "ReadMedia",
			Input: string(paramsJSON),
		}

		response, err := tool.Run(ctx, call)
		assert.NoError(t, err)
		assert.True(t, response.IsError)
		assert.Contains(t, response.Content, "media_type must be")
	})

	// Test missing context values
	t.Run("missing session context", func(t *testing.T) {
		ctx := context.Background()
		params := ReadMediaParams{
			FilePath:  "/absolute/path/test.jpg",
			MediaType: "image",
			Prompt:    "test prompt",
		}

		paramsJSON, _ := json.Marshal(params)
		call := interfaces.ToolCall{
			ID:    "test-3",
			Name:  "ReadMedia",
			Input: string(paramsJSON),
		}

		_, err := tool.Run(ctx, call)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID and message ID are required")
	})
}

// Test error handling scenarios
func TestReadMediaTool_ErrorHandling(t *testing.T) {
	tool := &readMediaTool{}

	// Test directory that doesn't exist
	t.Run("nonexistent directory", func(t *testing.T) {
		files, err := tool.findSupportedFiles("/nonexistent/directory", "image", false)
		assert.Error(t, err)
		assert.Empty(t, files)
	})

	// Test permission denied scenarios (if possible in test environment)
	t.Run("file read permissions", func(t *testing.T) {
		// Create a file and remove read permissions (Unix-like systems)
		tempFile, err := os.CreateTemp("", "no_read_perm*.jpg")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		tempFile.Close()

		// Try to remove read permissions (may not work on all systems)
		if err := os.Chmod(tempFile.Name(), 0000); err == nil {
			defer os.Chmod(tempFile.Name(), 0644) // Restore for cleanup

			_, _, err := tool.readFileContent(tempFile.Name())
			assert.Error(t, err)
		}
	})
}

// Benchmark tests for performance
func BenchmarkReadMediaTool_IsSupportedFile(b *testing.B) {
	tool := &readMediaTool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.isSupportedFile("/path/to/file.jpg", "image")
	}
}

func BenchmarkReadMediaTool_ValidateParams(b *testing.B) {
	tool := &readMediaTool{}
	params := ReadMediaParams{
		FilePath:  "/absolute/path/test.jpg",
		MediaType: "image",
		Prompt:    "Analyze this image",
		WordCount: 200,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.validateParams(params)
	}
}

// Test edge cases with special characters and Unicode
func TestReadMediaTool_UnicodeAndSpecialChars(t *testing.T) {
	tool := &readMediaTool{}

	tests := []struct {
		name     string
		filepath string
		expected bool
	}{
		{
			name:     "unicode filename",
			filepath: "/path/测试图片.jpg",
			expected: true,
		},
		{
			name:     "filename with spaces",
			filepath: "/path with spaces/my image.png",
			expected: true,
		},
		{
			name:     "filename with special chars",
			filepath: "/path/image@#$%^&*().jpg",
			expected: true,
		},
		{
			name:     "very long filename",
			filepath: "/path/" + strings.Repeat("a", 200) + ".jpg",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.isSupportedFile(tt.filepath, "image")
			assert.Equal(t, tt.expected, result)
		})
	}
}
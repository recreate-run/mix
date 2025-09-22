package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"mix/internal/config"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/llm/provider"
	"mix/internal/logging"
	"mix/internal/message"
	"mix/internal/permission"
)

type MultimodalAnalyzerParams struct {
	FilePath      string `json:"file_path,omitempty"`
	DirectoryPath string `json:"directory_path,omitempty"`
	AnalysisType  string `json:"analysis_type"`
	Prompt        string `json:"prompt"`
	Recursive     bool   `json:"recursive,omitempty"`
	WordCount     int    `json:"word_count,omitempty"`
	AudioMode     string `json:"audio_mode,omitempty"`
	VideoMode     string `json:"video_mode,omitempty"`
}

type multimodalAnalyzerTool struct {
	permissions permission.Service
}

type MultimodalAnalysisResult struct {
	FilePath     string `json:"file_path"`
	AnalysisType string `json:"analysis_type"`
	Analysis     string `json:"analysis"`
	Error        string `json:"error,omitempty"`
}

type MultimodalAnalysisResponse struct {
	Results []MultimodalAnalysisResult `json:"results"`
	Summary string                     `json:"summary"`
}

const (
	MultimodalAnalyzerToolName = "multimodal_analyzer"
	DefaultWordCount           = 200
	MaxWordCount               = 1000
	MinWordCount               = 50
)

var supportedImageTypes = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
var supportedAudioTypes = []string{".mp3", ".wav", ".m4a", ".aac", ".ogg", ".flac"}
var supportedVideoTypes = []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv"}

func NewMultimodalAnalyzerTool(permissions permission.Service) BaseTool {
	return &multimodalAnalyzerTool{
		permissions: permissions,
	}
}

func (m *multimodalAnalyzerTool) Info() ToolInfo {
	return ToolInfo{
		Name:        MultimodalAnalyzerToolName,
		Description: LoadToolDescription("multimodal_analyzer"),
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to single file for analysis",
			},
			"directory_path": map[string]any{
				"type":        "string",
				"description": "Path to directory for batch processing",
			},
			"analysis_type": map[string]any{
				"type":        "string",
				"enum":        []string{"image", "audio", "video"},
				"description": "Type of media analysis to perform",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Analysis prompt for the media content",
			},
			"recursive": map[string]any{
				"type":        "boolean",
				"default":     false,
				"description": "Process directories recursively",
			},
			"word_count": map[string]any{
				"type":        "integer",
				"minimum":     MinWordCount,
				"maximum":     MaxWordCount,
				"default":     DefaultWordCount,
				"description": "Target word count for analysis",
			},
			"audio_mode": map[string]any{
				"type":        "string",
				"enum":        []string{"transcript", "description"},
				"description": "Audio analysis mode (required for audio type)",
			},
			"video_mode": map[string]any{
				"type":        "string",
				"enum":        []string{"description"},
				"description": "Video analysis mode (required for video type)",
			},
		},
		Required: []string{"analysis_type", "prompt"},
	}
}

func (m *multimodalAnalyzerTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params MultimodalAnalyzerParams
	logging.Debug("multimodal analyzer tool params", "params", call.Input)

	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	// Validate required parameters
	if err := m.validateParams(params); err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	// Get session context
	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for multimodal analysis")
	}

	// Determine files to process
	files, err := m.getFilesToProcess(params)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error determining files to process: %s", err)), nil
	}

	if len(files) == 0 {
		return NewTextErrorResponse("no supported files found for analysis"), nil
	}

	// Process files
	results, err := m.processFiles(ctx, sessionID, messageID, files, params)
	if err != nil {
		return ToolResponse{}, err
	}

	// Create response
	response := MultimodalAnalysisResponse{
		Results: results,
		Summary: m.generateSummary(results),
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error formatting response: %s", err)), nil
	}

	return NewTextResponse(string(responseJSON)), nil
}

func (m *multimodalAnalyzerTool) validateParams(params MultimodalAnalyzerParams) error {
	// Validate mutually exclusive file/directory paths
	if params.FilePath != "" && params.DirectoryPath != "" {
		return fmt.Errorf("cannot specify both file_path and directory_path")
	}

	if params.FilePath == "" && params.DirectoryPath == "" {
		return fmt.Errorf("must specify either file_path or directory_path")
	}

	// Validate analysis type
	if params.AnalysisType != "image" && params.AnalysisType != "audio" && params.AnalysisType != "video" {
		return fmt.Errorf("analysis_type must be 'image', 'audio', or 'video'")
	}

	// Validate type-specific requirements
	if params.AnalysisType == "audio" && params.AudioMode == "" {
		return fmt.Errorf("audio_mode is required when analysis_type is 'audio'")
	}

	if params.AnalysisType == "video" && params.VideoMode == "" {
		return fmt.Errorf("video_mode is required when analysis_type is 'video'")
	}

	// Validate audio mode
	if params.AudioMode != "" && params.AudioMode != "transcript" && params.AudioMode != "description" {
		return fmt.Errorf("audio_mode must be 'transcript' or 'description'")
	}

	// Validate video mode
	if params.VideoMode != "" && params.VideoMode != "description" {
		return fmt.Errorf("video_mode must be 'description'")
	}

	// Validate word count
	if params.WordCount != 0 && (params.WordCount < MinWordCount || params.WordCount > MaxWordCount) {
		return fmt.Errorf("word_count must be between %d and %d", MinWordCount, MaxWordCount)
	}

	// Validate paths are absolute
	targetPath := params.FilePath
	if targetPath == "" {
		targetPath = params.DirectoryPath
	}
	if !filepath.IsAbs(targetPath) {
		return fmt.Errorf("file_path and directory_path must be absolute paths")
	}

	return nil
}

func (m *multimodalAnalyzerTool) getFilesToProcess(params MultimodalAnalyzerParams) ([]string, error) {
	var files []string

	if params.FilePath != "" {
		// Single file
		if m.isSupportedFile(params.FilePath, params.AnalysisType) {
			files = append(files, params.FilePath)
		}
	} else {
		// Directory processing
		var err error
		files, err = m.findSupportedFiles(params.DirectoryPath, params.AnalysisType, params.Recursive)
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (m *multimodalAnalyzerTool) findSupportedFiles(dirPath string, analysisType string, recursive bool) ([]string, error) {
	var files []string
	var supportedExts []string

	switch analysisType {
	case "image":
		supportedExts = supportedImageTypes
	case "audio":
		supportedExts = supportedAudioTypes
	case "video":
		supportedExts = supportedVideoTypes
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if !recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, supportedExt := range supportedExts {
			if ext == supportedExt {
				files = append(files, path)
				break
			}
		}

		return nil
	}

	err := filepath.Walk(dirPath, walkFn)
	return files, err
}

func (m *multimodalAnalyzerTool) isSupportedFile(filePath string, analysisType string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch analysisType {
	case "image":
		for _, supportedExt := range supportedImageTypes {
			if ext == supportedExt {
				return true
			}
		}
	case "audio":
		for _, supportedExt := range supportedAudioTypes {
			if ext == supportedExt {
				return true
			}
		}
	case "video":
		for _, supportedExt := range supportedVideoTypes {
			if ext == supportedExt {
				return true
			}
		}
	}

	return false
}

func (m *multimodalAnalyzerTool) processFiles(ctx context.Context, sessionID, messageID string, files []string, params MultimodalAnalyzerParams) ([]MultimodalAnalysisResult, error) {
	var results []MultimodalAnalysisResult

	for _, filePath := range files {
		result := m.analyzeFile(ctx, sessionID, messageID, filePath, params)
		results = append(results, result)
	}

	return results, nil
}

func (m *multimodalAnalyzerTool) analyzeFile(ctx context.Context, sessionID, messageID, filePath string, params MultimodalAnalyzerParams) MultimodalAnalysisResult {
	result := MultimodalAnalysisResult{
		FilePath:     filePath,
		AnalysisType: params.AnalysisType,
	}

	// Request permission to read the file
	p := m.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        filePath,
			ToolName:    MultimodalAnalyzerToolName,
			Action:      fmt.Sprintf("Analyze %s file: %s", params.AnalysisType, filePath),
			Description: fmt.Sprintf("Read and analyze %s file using Gemini AI: %s", params.AnalysisType, filePath),
			Params:      params,
		},
	)

	if !p {
		result.Error = "Permission denied for file access"
		return result
	}

	// Read file content
	fileData, mimeType, err := m.readFileContent(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("Error reading file: %s", err)
		return result
	}

	// Create Gemini provider
	geminiProvider, err := m.createGeminiProvider()
	if err != nil {
		result.Error = fmt.Sprintf("Error creating Gemini provider: %s", err)
		return result
	}

	// Prepare analysis prompt
	analysisPrompt := m.buildAnalysisPrompt(params)

	// Create message with binary content
	userMessage := message.Message{
		Role: message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: analysisPrompt},
			message.BinaryContent{
				Path:     filePath,
				MIMEType: mimeType,
				Data:     fileData,
			},
		},
	}

	// Send to Gemini for analysis
	analysis, err := m.sendToGemini(ctx, geminiProvider, userMessage)
	if err != nil {
		result.Error = fmt.Sprintf("Error during Gemini analysis: %s", err)
		return result
	}

	result.Analysis = analysis
	return result
}

func (m *multimodalAnalyzerTool) readFileContent(filePath string) ([]byte, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType == "" {
		// Fallback MIME type detection
		ext := strings.ToLower(filepath.Ext(filePath))
		switch {
		case contains(supportedImageTypes, ext):
			mimeType = "image/" + strings.TrimPrefix(ext, ".")
		case contains(supportedAudioTypes, ext):
			mimeType = "audio/" + strings.TrimPrefix(ext, ".")
		case contains(supportedVideoTypes, ext):
			mimeType = "video/" + strings.TrimPrefix(ext, ".")
		default:
			mimeType = "application/octet-stream"
		}
	}

	return data, mimeType, nil
}

func (m *multimodalAnalyzerTool) createGeminiProvider() (interfaces.Provider, error) {
	// Get API credentials
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		return nil, fmt.Errorf("API credentials service not available")
	}

	ctx := context.Background()
	apiKey, err := credentialsService.GetAPIKey(ctx, models.ProviderGemini)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gemini API key: %w", err)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("Gemini API key not configured")
	}

	// Create Gemini provider
	geminiProvider, err := provider.NewProvider(
		models.ProviderGemini,
		provider.WithAPIKey(apiKey),
		provider.WithModel(models.GeminiModels[models.Gemini25]), // Use Gemini 2.5 Pro for multimodal
		provider.WithMaxTokens(4096),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini provider: %w", err)
	}

	return geminiProvider, nil
}

func (m *multimodalAnalyzerTool) buildAnalysisPrompt(params MultimodalAnalyzerParams) string {
	wordCount := params.WordCount
	if wordCount == 0 {
		wordCount = DefaultWordCount
	}

	var promptPrefix string
	switch params.AnalysisType {
	case "image":
		promptPrefix = "Analyze this image and "
	case "audio":
		if params.AudioMode == "transcript" {
			promptPrefix = "Provide an accurate transcription of this audio content. "
		} else {
			promptPrefix = "Analyze this audio content and describe "
		}
	case "video":
		promptPrefix = "Analyze this video content, describing both visual and audio elements. "
	}

	return fmt.Sprintf("%s%s\n\nPlease provide approximately %d words in your response.", promptPrefix, params.Prompt, wordCount)
}

func (m *multimodalAnalyzerTool) sendToGemini(ctx context.Context, geminiProvider interfaces.Provider, userMessage message.Message) (string, error) {
	messages := []message.Message{userMessage}

	// Send message to Gemini
	response, err := geminiProvider.SendMessages(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("failed to send message to Gemini: %w", err)
	}

	// Extract text content from response
	if response == nil {
		return "", fmt.Errorf("no response received from Gemini")
	}

	return response.Content, nil
}

func (m *multimodalAnalyzerTool) generateSummary(results []MultimodalAnalysisResult) string {
	if len(results) == 0 {
		return "No files were analyzed."
	}

	successCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Error == "" {
			successCount++
		} else {
			errorCount++
		}
	}

	if errorCount == 0 {
		return fmt.Sprintf("Successfully analyzed %d file(s).", successCount)
	}

	return fmt.Sprintf("Analyzed %d file(s) successfully, %d failed.", successCount, errorCount)
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mix/internal/config"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/llm/provider"
	"mix/internal/logging"
	"mix/internal/message"
)

type ReadMediaParams struct {
	FilePath      string `json:"file_path,omitempty"`
	DirectoryPath string `json:"directory_path,omitempty"`
	MediaType     string `json:"media_type"`
	Prompt        string `json:"prompt"`
	Recursive     bool   `json:"recursive,omitempty"`
	AudioMode     string `json:"audio_mode,omitempty"`
	VideoMode     string `json:"video_mode,omitempty"`
	PdfPages      string `json:"pdf_pages,omitempty"`
	VideoInterval string `json:"video_interval,omitempty"`
}

type readMediaTool struct{}

type ReadMediaResult struct {
	FilePath  string `json:"file_path"`
	MediaType string `json:"media_type"`
	Analysis  string `json:"analysis"`
	Error     string `json:"error,omitempty"`
}

type ReadMediaResponse struct {
	Results []ReadMediaResult `json:"results"`
	Summary string            `json:"summary"`
}

const (
	ReadMediaToolName = "ReadMedia"
)

var supportedImageTypes = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
var supportedAudioTypes = []string{".mp3", ".wav", ".m4a", ".aac", ".ogg", ".flac"}
var supportedVideoTypes = []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv"}
var supportedPDFTypes = []string{".pdf"}

func NewReadMediaTool() BaseTool {
	return &readMediaTool{}
}

func (r *readMediaTool) Info() ToolInfo {
	return ToolInfo{
		Name:        ReadMediaToolName,
		Description: LoadToolDescription("read_media"),
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Path to single file for analysis",
			},
			"directory_path": map[string]any{
				"type":        "string",
				"description": "Path to directory for batch processing",
			},
			"media_type": map[string]any{
				"type":        "string",
				"enum":        []string{"image", "audio", "video", "pdf"},
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
			"pdf_pages": map[string]any{
				"type":        "string",
				"description": "PDF page selection: single page '5' or ranges '1-3,7,10-12' (PDF only)",
			},
			"video_interval": map[string]any{
				"type":        "string",
				"description": "Video time interval: timestamps '00:20:50-00:26:10' or '20:50-26:10' (video only)",
			},
		},
		Required: []string{"media_type", "prompt"},
	}
}

func (r *readMediaTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ReadMediaParams
	logging.Debug("multimodal analyzer tool params", "params", call.Input)

	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	// Validate required parameters
	if err := r.validateParams(params); err != nil {
		return NewTextErrorResponse(err.Error()), nil
	}

	// Check if Gemini API key is configured before processing any files
	if err := r.validateGeminiAPIKey(ctx); err != nil {
		logging.Error("Gemini API key not configured")
		return NewTextErrorResponse("Configuration required: Gemini API key is not configured. Please configure your Gemini API key in Settings to use image analysis features. This is a configuration requirement and no alternative tools or approaches should be attempted."), nil
	}

	// Get session context
	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for multimodal analysis")
	}

	// Determine files to process
	files, err := r.getFilesToProcess(params)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error determining files to process: %s", err)), nil
	}

	if len(files) == 0 {
		return NewTextErrorResponse("no supported files found for analysis"), nil
	}

	// Process files
	results, err := r.processFiles(ctx, sessionID, messageID, files, params)
	if err != nil {
		return ToolResponse{}, err
	}

	// Create response
	response := ReadMediaResponse{
		Results: results,
		Summary: r.generateSummary(results),
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error formatting response: %s", err)), nil
	}

	return NewTextResponse(string(responseJSON)), nil
}

func (r *readMediaTool) validateParams(params ReadMediaParams) error {
	// Validate mutually exclusive file/directory paths
	if params.FilePath != "" && params.DirectoryPath != "" {
		return fmt.Errorf("cannot specify both file_path and directory_path")
	}

	if params.FilePath == "" && params.DirectoryPath == "" {
		return fmt.Errorf("must specify either file_path or directory_path")
	}

	// Validate analysis type
	if params.MediaType != "image" && params.MediaType != "audio" && params.MediaType != "video" && params.MediaType != "pdf" {
		return fmt.Errorf("media_type must be 'image', 'audio', 'video', or 'pdf'")
	}

	// Validate type-specific requirements
	if params.MediaType == "audio" && params.AudioMode == "" {
		return fmt.Errorf("audio_mode is required when media_type is 'audio'")
	}

	if params.MediaType == "video" && params.VideoMode == "" {
		return fmt.Errorf("video_mode is required when media_type is 'video'")
	}

	// Validate audio mode
	if params.AudioMode != "" && params.AudioMode != "transcript" && params.AudioMode != "description" {
		return fmt.Errorf("audio_mode must be 'transcript' or 'description'")
	}

	// Validate video mode
	if params.VideoMode != "" && params.VideoMode != "description" {
		return fmt.Errorf("video_mode must be 'description'")
	}

	// Validate PDF pages parameter
	if params.PdfPages != "" {
		if params.MediaType != "pdf" {
			return fmt.Errorf("pdf_pages parameter can only be used with media_type 'pdf'")
		}
		if err := ValidatePageSelection(params.PdfPages); err != nil {
			return fmt.Errorf("invalid pdf_pages parameter: %w", err)
		}
	}

	// Validate video interval parameter
	if params.VideoInterval != "" {
		if params.MediaType != "video" {
			return fmt.Errorf("video_interval parameter can only be used with media_type 'video'")
		}
		// Validate the interval format by trying to parse it
		if _, _, err := parseVideoInterval(params.VideoInterval); err != nil {
			return fmt.Errorf("invalid video_interval parameter: %w", err)
		}
	}

	// Validate paths are absolute (skip for URLs)
	targetPath := params.FilePath
	if targetPath == "" {
		targetPath = params.DirectoryPath
	}

	// Skip absolute path validation for URLs
	if params.FilePath != "" && isURL(params.FilePath) {
		// URLs are supported for all media types
		return nil
	}

	if !filepath.IsAbs(targetPath) {
		return fmt.Errorf("file_path and directory_path must be absolute paths")
	}

	return nil
}

func (r *readMediaTool) validateGeminiAPIKey(ctx context.Context) error {
	// Get API credentials
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		return fmt.Errorf("API credentials service not available")
	}

	apiKey, err := credentialsService.GetAPIKey(ctx, models.ProviderGemini)
	if err != nil {
		return fmt.Errorf("failed to get Gemini API key: %w", err)
	}

	if apiKey == "" {
		return fmt.Errorf("Gemini API key not configured")
	}

	return nil
}

func (r *readMediaTool) getFilesToProcess(params ReadMediaParams) ([]string, error) {
	var files []string

	if params.FilePath != "" {
		// Check if it's a URL (YouTube, etc.)
		if isURL(params.FilePath) {
			// For URLs, just return the URL as the "file" to process
			files = append(files, params.FilePath)
		} else {
			// Single local file
			if r.isSupportedFile(params.FilePath, params.MediaType) {
				files = append(files, params.FilePath)
			}
		}
	} else {
		// Directory processing
		var err error
		files, err = r.findSupportedFiles(params.DirectoryPath, params.MediaType, params.Recursive)
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (r *readMediaTool) findSupportedFiles(dirPath string, mediaType string, recursive bool) ([]string, error) {
	var files []string
	var supportedExts []string

	switch mediaType {
	case "image":
		supportedExts = supportedImageTypes
	case "audio":
		supportedExts = supportedAudioTypes
	case "video":
		supportedExts = supportedVideoTypes
	case "pdf":
		supportedExts = supportedPDFTypes
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

func (r *readMediaTool) isSupportedFile(filePath string, mediaType string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch mediaType {
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
	case "pdf":
		for _, supportedExt := range supportedPDFTypes {
			if ext == supportedExt {
				return true
			}
		}
	}

	return false
}

func (r *readMediaTool) processFiles(ctx context.Context, sessionID, messageID string, files []string, params ReadMediaParams) ([]ReadMediaResult, error) {
	var results []ReadMediaResult

	for _, filePath := range files {
		result := r.analyzeFile(ctx, sessionID, messageID, filePath, params)
		results = append(results, result)
	}

	return results, nil
}

func (r *readMediaTool) analyzeFile(ctx context.Context, sessionID, messageID, filePath string, params ReadMediaParams) ReadMediaResult {
	result := ReadMediaResult{
		FilePath:  filePath,
		MediaType: params.MediaType,
	}

	// Create Gemini provider (API key already validated)
	geminiProvider, err := r.createGeminiProvider()
	if err != nil {
		result.Error = fmt.Sprintf("Unexpected error creating Gemini provider: %s", err)
		return result
	}

	// Prepare analysis prompt
	analysisPrompt := r.buildAnalysisPrompt(params)

	// Parse video interval if provided
	var startOffset, endOffset string
	var videoTruncated bool
	if params.VideoInterval != "" {
		var err error
		startOffset, endOffset, err = parseVideoInterval(params.VideoInterval)
		if err != nil {
			result.Error = fmt.Sprintf("Error parsing video interval: %s", err)
			return result
		}
	} else if params.MediaType == "video" {
		// Auto-truncate videos to first 10 minutes when no interval specified
		startOffset = "0s"
		endOffset = "600s" // 10 minutes = 600 seconds
		videoTruncated = true
	}

	// Append truncation notice if video was auto-truncated
	if videoTruncated {
		analysisPrompt += "\n\nNote: This video has been truncated to the first 10 minutes."
	}

	// Create message with appropriate content type
	var userMessage message.Message
	if isURL(filePath) && isYouTubeURL(filePath) {
		// For YouTube URLs, use URIContent with native Gemini support
		userMessage = message.Message{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: analysisPrompt},
				message.URIContent{
					URI:         filePath,
					MIMEType:    "video/mp4", // YouTube videos are treated as MP4 by Gemini
					StartOffset: startOffset,
					EndOffset:   endOffset,
				},
			},
		}
	} else if isURL(filePath) {
		// For all other URLs, download to memory and use BinaryContent
		fileData, mimeType, err := r.downloadURLToMemory(filePath)
		if err != nil {
			result.Error = fmt.Sprintf("Error downloading from URL: %s", err)
			return result
		}

		// For PDF files, extract pages (with auto-truncation if > 10 pages and no range specified)
		var wasTruncated bool
		if params.MediaType == "pdf" {
			extractedData, truncated, err := ExtractPDFPages(fileData, params.PdfPages)
			if err != nil {
				result.Error = fmt.Sprintf("Error extracting PDF pages from URL: %s", err)
				return result
			}
			fileData = extractedData
			wasTruncated = truncated
		}

		// Append truncation notice to prompt if PDF was auto-truncated
		if wasTruncated {
			analysisPrompt += "\n\nNote: This PDF has been truncated at the tenth page."
		}

		userMessage = message.Message{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: analysisPrompt},
				message.BinaryContent{
					Path:        filePath,
					MIMEType:    mimeType,
					Data:        fileData,
					StartOffset: startOffset,
					EndOffset:   endOffset,
				},
			},
		}
	} else {
		// For local files, read content and use BinaryContent
		fileData, mimeType, err := r.readFileContent(filePath)
		if err != nil {
			result.Error = fmt.Sprintf("Error reading file: %s", err)
			return result
		}

		// For PDF files, extract pages (with auto-truncation if > 10 pages and no range specified)
		var wasTruncated bool
		if params.MediaType == "pdf" {
			extractedData, truncated, err := ExtractPDFPages(fileData, params.PdfPages)
			if err != nil {
				result.Error = fmt.Sprintf("Error extracting PDF pages: %s", err)
				return result
			}
			fileData = extractedData
			wasTruncated = truncated
		}

		// Append truncation notice to prompt if PDF was auto-truncated
		if wasTruncated {
			analysisPrompt += "\n\nNote: This PDF has been truncated at the tenth page."
		}

		userMessage = message.Message{
			Role: message.User,
			Parts: []message.ContentPart{
				message.TextContent{Text: analysisPrompt},
				message.BinaryContent{
					Path:        filePath,
					MIMEType:    mimeType,
					Data:        fileData,
					StartOffset: startOffset,
					EndOffset:   endOffset,
				},
			},
		}
	}

	// Send to Gemini for analysis
	analysis, err := r.sendToGemini(ctx, geminiProvider, userMessage)
	if err != nil {
		result.Error = fmt.Sprintf("Error during Gemini analysis: %s", err)
		return result
	}

	result.Analysis = analysis

	return result
}

func (r *readMediaTool) readFileContent(filePath string) ([]byte, string, error) {
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
		case contains(supportedPDFTypes, ext):
			mimeType = "application/pdf"
		default:
			mimeType = "application/octet-stream"
		}
	}

	return data, mimeType, nil
}

func (r *readMediaTool) downloadURLToMemory(url string) ([]byte, string, error) {
	// Download the media file from URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download from URL: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read data: %w", err)
	}

	// Get MIME type from Content-Type header
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		// Fallback to extension-based detection
		ext := strings.ToLower(filepath.Ext(url))
		switch {
		case contains(supportedImageTypes, ext):
			mimeType = "image/" + strings.TrimPrefix(ext, ".")
		case contains(supportedAudioTypes, ext):
			mimeType = "audio/" + strings.TrimPrefix(ext, ".")
		case contains(supportedVideoTypes, ext):
			mimeType = "video/" + strings.TrimPrefix(ext, ".")
		case contains(supportedPDFTypes, ext):
			mimeType = "application/pdf"
		default:
			mimeType = "application/octet-stream"
		}
	}

	return data, mimeType, nil
}

func (r *readMediaTool) createGeminiProvider() (interfaces.Provider, error) {
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

func (r *readMediaTool) buildAnalysisPrompt(params ReadMediaParams) string {
	var promptPrefix string
	switch params.MediaType {
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
	case "pdf":
		promptPrefix = "Analyze this PDF document, including both text content and visual elements such as charts, diagrams, and tables. "
	}

	return fmt.Sprintf("%s%s", promptPrefix, params.Prompt)
}

func (r *readMediaTool) sendToGemini(ctx context.Context, geminiProvider interfaces.Provider, userMessage message.Message) (string, error) {
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

func (r *readMediaTool) generateSummary(results []ReadMediaResult) string {
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

// Helper function to check if a URL is a YouTube URL
func isYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
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

// parseTimestamp converts a timestamp string (HH:MM:SS or MM:SS) to seconds
func parseTimestamp(timestamp string) (int, error) {
	parts := strings.Split(timestamp, ":")

	var hours, minutes, seconds int
	var err error

	switch len(parts) {
	case 2:
		// MM:SS format
		minutes, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes: %w", err)
		}
		seconds, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds: %w", err)
		}
	case 3:
		// HH:MM:SS format
		hours, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid hours: %w", err)
		}
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes: %w", err)
		}
		seconds, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds: %w", err)
		}
	default:
		return 0, fmt.Errorf("timestamp must be in HH:MM:SS or MM:SS format")
	}

	// Validate ranges
	if minutes < 0 || minutes >= 60 {
		return 0, fmt.Errorf("minutes must be between 0 and 59")
	}
	if seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("seconds must be between 0 and 59")
	}
	if hours < 0 {
		return 0, fmt.Errorf("hours must be non-negative")
	}

	totalSeconds := hours*3600 + minutes*60 + seconds
	return totalSeconds, nil
}

// parseVideoInterval parses a video interval string and returns start/end offsets in Gemini format
func parseVideoInterval(interval string) (string, string, error) {
	parts := strings.Split(interval, "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("video_interval must be in format 'start-end' (e.g., '00:20:50-00:26:10')")
	}

	startTimestamp := strings.TrimSpace(parts[0])
	endTimestamp := strings.TrimSpace(parts[1])

	startSeconds, err := parseTimestamp(startTimestamp)
	if err != nil {
		return "", "", fmt.Errorf("invalid start timestamp: %w", err)
	}

	endSeconds, err := parseTimestamp(endTimestamp)
	if err != nil {
		return "", "", fmt.Errorf("invalid end timestamp: %w", err)
	}

	if startSeconds >= endSeconds {
		return "", "", fmt.Errorf("start timestamp must be before end timestamp")
	}

	// Format as Gemini API expects: "1250s", "1570s"
	return fmt.Sprintf("%ds", startSeconds), fmt.Sprintf("%ds", endSeconds), nil
}

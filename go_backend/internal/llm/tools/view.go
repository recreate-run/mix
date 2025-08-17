package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mix/internal/logging"
	"mix/internal/permission"
)

type ViewParams struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

type viewTool struct {
	permissions permission.Service
}

type ViewResponseMetadata struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

const (
	ViewToolName     = "view"
	DefaultReadLimit = 2000
	MaxLineLength    = 2000
)

func NewViewTool(permissions permission.Service) BaseTool {
	return &viewTool{
		permissions: permissions,
	}
}

func (v *viewTool) Info() ToolInfo {
	return ToolInfo{
		Name:        ViewToolName,
		Description: LoadToolDescription("view"),
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to read",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "The line number to start reading from (0-based)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "The number of lines to read (defaults to 2000)",
			},
		},
		Required: []string{"file_path"},
	}
}

// Run implements Tool.
func (v *viewTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ViewParams
	logging.Debug("view tool params", "params", call.Input)
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	if params.FilePath == "" {
		return NewTextErrorResponse("file_path is required"), nil
	}

	// Require absolute paths only
	filePath := params.FilePath
	if !filepath.IsAbs(filePath) {
		return NewTextErrorResponse("file_path must be an absolute path, not a relative path"), nil
	}

	// Check permissions before reading the file
	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for reading a file")
	}

	// Request permission to read the file
	p := v.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        filePath,
			ToolName:    ViewToolName,
			Action:      fmt.Sprintf("Read file: %s", filePath),
			Description: fmt.Sprintf("Read file: %s", filePath),
			Params: ViewParams{
				FilePath: filePath,
				Offset:   params.Offset,
				Limit:    params.Limit,
			},
		},
	)
	if !p {
		return ToolResponse{}, permission.ErrorPermissionDenied
	}

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to offer suggestions for similarly named files
			dir := filepath.Dir(filePath)
			base := filepath.Base(filePath)

			dirEntries, dirErr := os.ReadDir(dir)
			if dirErr == nil {
				var suggestions []string
				for _, entry := range dirEntries {
					if strings.Contains(strings.ToLower(entry.Name()), strings.ToLower(base)) ||
						strings.Contains(strings.ToLower(base), strings.ToLower(entry.Name())) {
						suggestions = append(suggestions, filepath.Join(dir, entry.Name()))
						if len(suggestions) >= 3 {
							break
						}
					}
				}

				if len(suggestions) > 0 {
					return NewTextErrorResponse(fmt.Sprintf("File not found: %s\n\nDid you mean one of these?\n%s",
						filePath, strings.Join(suggestions, "\n"))), nil
				}
			}

			return NewTextErrorResponse(fmt.Sprintf("File not found: %s", filePath)), nil
		}
		return ToolResponse{}, fmt.Errorf("error accessing file: %w", err)
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		return NewTextErrorResponse(fmt.Sprintf("Path is a directory, not a file: %s", filePath)), nil
	}

	// Set default limit if not provided
	if params.Limit <= 0 {
		params.Limit = DefaultReadLimit
	}

	// Check if it's an image file
	isImage, imageType := isImageFile(filePath)
	if isImage {
		// Get image file info
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("error getting image file info: %w", err)
		}

		// Return text description instead of base64 data to avoid context overflow
		imageDescription := fmt.Sprintf("Image file (%s) at %s\nFile size: %d bytes\n",
			imageType, filePath, fileInfo.Size())

		recordFileRead(filePath)
		return WithResponseMetadata(
			NewTextResponse(imageDescription),
			ViewResponseMetadata{
				FilePath: filePath,
				Content:  imageDescription,
			},
		), nil
	}

	// Check if it's a video file
	isVideo, videoType := isVideoFile(filePath)
	if isVideo {
		// Get video file info
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("error getting video file info: %w", err)
		}

		// Return text description instead of video data to avoid context overflow
		videoDescription := fmt.Sprintf("Video file (%s) at %s\nFile size: %d bytes\n",
			videoType, filePath, fileInfo.Size())

		recordFileRead(filePath)
		return WithResponseMetadata(
			NewTextResponse(videoDescription),
			ViewResponseMetadata{
				FilePath: filePath,
				Content:  videoDescription,
			},
		), nil
	}

	// Check if it's an audio file
	isAudio, audioType := isAudioFile(filePath)
	if isAudio {
		// Get audio file info
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("error getting audio file info: %w", err)
		}

		// Return text description instead of audio data to avoid context overflow
		audioDescription := fmt.Sprintf("Audio file (%s) at %s\nFile size: %d bytes\n",
			audioType, filePath, fileInfo.Size())

		recordFileRead(filePath)
		return WithResponseMetadata(
			NewTextResponse(audioDescription),
			ViewResponseMetadata{
				FilePath: filePath,
				Content:  audioDescription,
			},
		), nil
	}

	// Read the file content
	content, lineCount, err := readTextFile(filePath, params.Offset, params.Limit)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error reading file: %w", err)
	}

	// Handle empty files
	if content == "" && lineCount == 0 {
		output := "<file>\n<system-reminder>\nFile exists but has empty contents.\n</system-reminder>\n</file>\n"
		recordFileRead(filePath)
		return WithResponseMetadata(
			NewTextResponse(output),
			ViewResponseMetadata{
				FilePath: filePath,
				Content:  "",
			},
		), nil
	}

	// LSP functionality removed
	output := "<file>\n"
	// Format the output with line numbers
	output += addLineNumbers(content, params.Offset+1)

	// Add a note if the content was truncated
	if lineCount > params.Offset+len(strings.Split(content, "\n")) {
		output += fmt.Sprintf("\n\n(File has more lines. Use 'offset' parameter to read beyond line %d)",
			params.Offset+len(strings.Split(content, "\n")))
	}
	output += "\n</file>\n"
	// LSP diagnostics functionality removed
	recordFileRead(filePath)
	return WithResponseMetadata(
		NewTextResponse(output),
		ViewResponseMetadata{
			FilePath: filePath,
			Content:  content,
		},
	), nil
}

func addLineNumbers(content string, startLine int) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")

	var result []string
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")

		lineNum := i + startLine
		// Use cat -n format: right-aligned line number followed by tab and content
		result = append(result, fmt.Sprintf("%6d\t%s", lineNum, line))
	}

	return strings.Join(result, "\n")
}

func readTextFile(filePath string, offset, limit int) (string, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	lineCount := 0

	scanner := NewLineScanner(file)
	if offset > 0 {
		for lineCount < offset && scanner.Scan() {
			lineCount++
		}
		if err = scanner.Err(); err != nil {
			return "", 0, err
		}
	}

	if offset == 0 {
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return "", 0, err
		}
	}

	var lines []string
	lineCount = offset

	for scanner.Scan() && len(lines) < limit {
		lineCount++
		lineText := scanner.Text()
		if len(lineText) > MaxLineLength {
			lineText = lineText[:MaxLineLength] + "..."
		}
		lines = append(lines, lineText)
	}

	// Continue scanning to get total line count
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	return strings.Join(lines, "\n"), lineCount, nil
}

func isImageFile(filePath string) (bool, string) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return true, "JPEG"
	case ".png":
		return true, "PNG"
	case ".gif":
		return true, "GIF"
	case ".bmp":
		return true, "BMP"
	case ".svg":
		return true, "SVG"
	case ".webp":
		return true, "WebP"
	default:
		return false, ""
	}
}

func isVideoFile(filePath string) (bool, string) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp4":
		return true, "MP4"
	case ".mov":
		return true, "MOV"
	case ".avi":
		return true, "AVI"
	case ".mkv":
		return true, "MKV"
	case ".webm":
		return true, "WebM"
	case ".wmv":
		return true, "WMV"
	case ".m4v":
		return true, "M4V"
	case ".flv":
		return true, "FLV"
	default:
		return false, ""
	}
}

func isAudioFile(filePath string) (bool, string) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".wav":
		return true, "WAV"
	case ".mp3":
		return true, "MP3"
	case ".flac":
		return true, "FLAC"
	case ".ogg":
		return true, "OGG"
	case ".aac":
		return true, "AAC"
	case ".m4a":
		return true, "M4A"
	case ".wma":
		return true, "WMA"
	default:
		return false, ""
	}
}

type LineScanner struct {
	scanner *bufio.Scanner
}

func NewLineScanner(r io.Reader) *LineScanner {
	return &LineScanner{
		scanner: bufio.NewScanner(r),
	}
}

func (s *LineScanner) Scan() bool {
	return s.scanner.Scan()
}

func (s *LineScanner) Text() string {
	return s.scanner.Text()
}

func (s *LineScanner) Err() error {
	return s.scanner.Err()
}

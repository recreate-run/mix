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

type ReadTextParams struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

type readTextTool struct {
	permissions permission.Service
}

type ReadTextResponseMetadata struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

const (
	ReadTextToolName = "ReadText"
	DefaultReadLimit = 2000
	MaxLineLength    = 2000
)

func NewReadTextTool(permissions permission.Service) BaseTool {
	return &readTextTool{
		permissions: permissions,
	}
}

func (r *readTextTool) Info() ToolInfo {
	return ToolInfo{
		Name:        ReadTextToolName,
		Description: LoadToolDescription("read_text"),
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
func (r *readTextTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ReadTextParams
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
	p := r.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        filePath,
			ToolName:    ReadTextToolName,
			Action:      fmt.Sprintf("Read file: %s", filePath),
			Description: fmt.Sprintf("Read file: %s", filePath),
			Params: ReadTextParams{
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

	// Check if it's a binary file and reject it
	if isBinaryFile(filePath) {
		return NewTextErrorResponse(fmt.Sprintf("Cannot read binary file: %s", filePath)), nil
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
			ReadTextResponseMetadata{
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
		ReadTextResponseMetadata{
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

func isBinaryFile(filePath string) bool {
	// Check by file extension first
	ext := strings.ToLower(filepath.Ext(filePath))
	binaryExtensions := []string{
		// Images
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".ico", ".tiff", ".tif",
		// Videos
		".mp4", ".mov", ".avi", ".mkv", ".webm", ".wmv", ".m4v", ".flv", ".3gp", ".ogv",
		// Audio
		".wav", ".mp3", ".flac", ".ogg", ".aac", ".m4a", ".wma", ".opus",
		// Archives
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
		// Executables
		".exe", ".bin", ".app", ".dmg", ".pkg", ".deb", ".rpm",
		// Documents
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		// Other binary formats
		".db", ".sqlite", ".sqlite3", ".dat", ".bin", ".o", ".so", ".dylib", ".dll",
	}

	for _, binaryExt := range binaryExtensions {
		if ext == binaryExt {
			return true
		}
	}

	// If extension check passes, sample the file content
	return isBinaryContent(filePath)
}

func isBinaryContent(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false // If we can't open it, let the main function handle the error
	}
	defer file.Close()

	// Read first 512 bytes to check for binary content
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false // If we can't read it, let the main function handle the error
	}

	// Check for null bytes (common indicator of binary content)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	// Check for high percentage of non-printable characters
	nonPrintable := 0
	for i := 0; i < n; i++ {
		b := buffer[i]
		// Consider printable: ASCII 32-126, tab (9), newline (10), carriage return (13)
		if !(b >= 32 && b <= 126) && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
	}

	// If more than 30% non-printable characters, consider it binary
	if n > 0 && float64(nonPrintable)/float64(n) > 0.30 {
		return true
	}

	return false
}

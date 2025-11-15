package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"mix/internal/permission"
)

type GrepParams struct {
	Pattern         string `json:"pattern"`
	Path            string `json:"path"`
	Glob            string `json:"glob"`
	Type            string `json:"type"`
	OutputMode      string `json:"output_mode"`
	CaseInsensitive bool   `json:"-i"`
	ShowLineNumbers bool   `json:"-n"`
	ContextAfter    int    `json:"-A"`
	ContextBefore   int    `json:"-B"`
	ContextAround   int    `json:"-C"`
	Multiline       bool   `json:"multiline"`
	HeadLimit       int    `json:"head_limit"`
}

type GrepResponseMetadata struct {
	NumberOfMatches int  `json:"number_of_matches"`
	Truncated       bool `json:"truncated"`
}

type grepTool struct {
	permissions permission.Service
}

const (
	GrepToolName               = "Grep"
	outputModeFilesWithMatches = "files_with_matches"
	outputModeContent          = "content"
	outputModeCount            = "count"
	truncatedResultsMessage    = "\n\n(Results truncated to head_limit. Refine your search for complete results.)"
)

func NewGrepTool(permissions permission.Service) BaseTool {
	return &grepTool{
		permissions: permissions,
	}
}

func (g *grepTool) Info() ToolInfo {
	return ToolInfo{
		Name:        GrepToolName,
		Description: LoadToolDescription("grep"),
		Parameters: map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The regex pattern to search for in file contents",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory to search in (rg PATH). Defaults to current working directory.",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Glob pattern to filter files (e.g. \"*.js\", \"*.{ts,tsx}\") - maps to rg --glob",
			},
			"type": map[string]any{
				"type":        "string",
				"description": "File type to search (rg --type). Common types: js, py, rust, go, java, etc. More efficient than include for standard file types.",
			},
			"output_mode": map[string]any{
				"type":        "string",
				"description": "Output mode: \"content\" shows matching lines (supports -A/-B/-C context, -n line numbers, head_limit), \"files_with_matches\" shows file paths (supports head_limit), \"count\" shows match counts (supports head_limit). Defaults to \"files_with_matches\".",
				"enum":        []string{outputModeContent, outputModeFilesWithMatches, outputModeCount},
			},
			"-i": map[string]any{
				"type":        "boolean",
				"description": "Case insensitive search (rg -i)",
			},
			"-n": map[string]any{
				"type":        "boolean",
				"description": "Show line numbers in output (rg -n). Requires output_mode: \"content\", ignored otherwise.",
			},
			"-A": map[string]any{
				"type":        "number",
				"description": "Number of lines to show after each match (rg -A). Requires output_mode: \"content\", ignored otherwise.",
			},
			"-B": map[string]any{
				"type":        "number",
				"description": "Number of lines to show before each match (rg -B). Requires output_mode: \"content\", ignored otherwise.",
			},
			"-C": map[string]any{
				"type":        "number",
				"description": "Number of lines to show before and after each match (rg -C). Requires output_mode: \"content\", ignored otherwise.",
			},
			"multiline": map[string]any{
				"type":        "boolean",
				"description": "Enable multiline mode where . matches newlines and patterns can span lines (rg -U --multiline-dotall). Default: false.",
			},
			"head_limit": map[string]any{
				"type":        "number",
				"description": "Limit output to first N lines/entries, equivalent to \"| head -N\". Works across all output modes: content (limits output lines), files_with_matches (limits file paths), count (limits count entries). When unspecified, shows all results from ripgrep.",
			},
		},
		Required: []string{"pattern"},
	}
}

func (g *grepTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params GrepParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	if params.Pattern == "" {
		return NewTextErrorResponse("pattern is required"), nil
	}

	// Default output mode
	if params.OutputMode == "" {
		params.OutputMode = outputModeFilesWithMatches
	}

	searchPath := params.Path
	if searchPath == "" {
		var err error
		searchPath, err = GetSessionStorageDirectory(ctx)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("failed to get session storage directory: %w", err)
		}
	}

	// Check permissions before searching files
	sessionID, _ := GetContextValues(ctx)
	if sessionID == "" {
		return ToolResponse{}, fmt.Errorf("session ID is required for searching files")
	}

	// Request permission to search files
	p := g.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        searchPath,
			ToolName:    GrepToolName,
			Action:      fmt.Sprintf("Search files with pattern: %s", params.Pattern),
			Description: fmt.Sprintf("Search files in %s with pattern: %s", searchPath, params.Pattern),
			Params:      params,
		},
	)
	if !p {
		return ToolResponse{}, permission.ErrPermissionDenied
	}

	output, count, truncated, err := searchFilesAdvanced(ctx, params, searchPath)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error searching files: %w", err)
	}

	return WithResponseMetadata(
		NewTextResponse(output),
		GrepResponseMetadata{
			NumberOfMatches: count,
			Truncated:       truncated,
		},
	), nil
}

func searchFilesAdvanced(ctx context.Context, params GrepParams, searchPath string) (output string, count int, truncated bool, err error) {
	_, err = exec.LookPath("rg")
	if err != nil {
		return "", 0, false, fmt.Errorf("ripgrep not found: %w", err)
	}

	args := buildRipgrepArgs(params)
	args = append(args, searchPath)

	cmd := exec.CommandContext(ctx, "rg", args...)
	outputBytes, cmdErr := cmd.Output()
	if cmdErr != nil {
		var exitErr *exec.ExitError
		if errors.As(cmdErr, &exitErr) && exitErr.ExitCode() == 1 {
			return "No matches found", 0, false, nil
		}
		return "", 0, false, cmdErr
	}

	outputStr := string(outputBytes)
	if outputStr == "" {
		return "No matches found", 0, false, nil
	}

	// Process output based on mode
	switch params.OutputMode {
	case outputModeContent:
		return processContentOutput(outputStr, params)
	case outputModeFilesWithMatches:
		return processFilesOutput(outputStr, params)
	case outputModeCount:
		return processCountOutput(outputStr, params)
	default:
		return "", 0, false, fmt.Errorf("unknown output mode: %s", params.OutputMode)
	}
}

func buildRipgrepArgs(params GrepParams) []string {
	args := []string{}

	// Output format based on mode
	switch params.OutputMode {
	case outputModeFilesWithMatches:
		args = append(args, "-l") // --files-with-matches
	case outputModeCount:
		args = append(args, "-c") // --count
	case outputModeContent:
		args = append(args, "-H") // Include filename
		if params.ShowLineNumbers {
			args = append(args, "-n") // Line numbers
		}
		// Context lines
		if params.ContextAround > 0 {
			args = append(args, fmt.Sprintf("-C%d", params.ContextAround))
		} else {
			if params.ContextBefore > 0 {
				args = append(args, fmt.Sprintf("-B%d", params.ContextBefore))
			}
			if params.ContextAfter > 0 {
				args = append(args, fmt.Sprintf("-A%d", params.ContextAfter))
			}
		}
	}

	// Case insensitive
	if params.CaseInsensitive {
		args = append(args, "-i")
	}

	// Multiline mode
	if params.Multiline {
		args = append(args, "-U", "--multiline-dotall")
	}

	// File filtering
	if params.Glob != "" {
		args = append(args, "--glob", params.Glob)
	}
	if params.Type != "" {
		args = append(args, "--type", params.Type)
	}

	// Pattern
	args = append(args, params.Pattern)

	return args
}

func processContentOutput(output string, params GrepParams) (result string, count int, truncated bool, err error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Apply head_limit to output lines
	truncated = false
	if params.HeadLimit > 0 && len(lines) > params.HeadLimit {
		lines = lines[:params.HeadLimit]
		truncated = true
	}

	// Count unique files in the output
	fileSet := make(map[string]bool)
	for _, line := range lines {
		if line == "" || line == "--" {
			continue
		}
		// Extract filename (before first : or -)
		var filename string
		if idx := strings.Index(line, ":"); idx != -1 {
			filename = line[:idx]
		} else if idx := strings.Index(line, "-"); idx != -1 {
			filename = line[:idx]
		}
		if filename != "" {
			fileSet[filename] = true
		}
	}

	result = strings.Join(lines, "\n")
	if truncated {
		result += truncatedResultsMessage
	}

	return result, len(fileSet), truncated, nil
}

func processFilesOutput(output string, params GrepParams) (outputStr string, count int, truncated bool, err error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Parse file paths and get modification times
	type fileWithTime struct {
		path    string
		modTime time.Time
	}

	filesWithTimes := []fileWithTime{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		fileInfo, err := os.Stat(line)
		if err != nil {
			continue
		}
		filesWithTimes = append(filesWithTimes, fileWithTime{
			path:    line,
			modTime: fileInfo.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(filesWithTimes, func(i, j int) bool {
		return filesWithTimes[i].modTime.After(filesWithTimes[j].modTime)
	})

	// Apply head_limit
	truncated = false
	if params.HeadLimit > 0 && len(filesWithTimes) > params.HeadLimit {
		filesWithTimes = filesWithTimes[:params.HeadLimit]
		truncated = true
	}

	// Build output
	result := make([]string, len(filesWithTimes))
	for i, f := range filesWithTimes {
		result[i] = f.path
	}

	outputStr = strings.Join(result, "\n")
	if truncated {
		outputStr += truncatedResultsMessage
	}

	return outputStr, len(filesWithTimes), truncated, nil
}

func processCountOutput(output string, params GrepParams) (outputStr string, count int, truncated bool, err error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Parse count entries and get modification times
	type countEntry struct {
		path    string
		count   int
		modTime time.Time
	}

	entries := []countEntry{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: file:count
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		filePath := parts[0]
		count, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		entries = append(entries, countEntry{
			path:    filePath,
			count:   count,
			modTime: fileInfo.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime.After(entries[j].modTime)
	})

	// Apply head_limit
	truncated = false
	if params.HeadLimit > 0 && len(entries) > params.HeadLimit {
		entries = entries[:params.HeadLimit]
		truncated = true
	}

	// Build output
	result := make([]string, len(entries))
	totalCount := 0
	for i, e := range entries {
		result[i] = fmt.Sprintf("%s:%d", e.path, e.count)
		totalCount += e.count
	}

	outputStr = strings.Join(result, "\n")
	if truncated {
		outputStr += truncatedResultsMessage
	}

	return outputStr, len(entries), truncated, nil
}

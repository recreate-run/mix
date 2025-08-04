package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"mix/internal/fileutil"
)

type GrepParams struct {
	Pattern     string `json:"pattern"`
	Path        string `json:"path"`
	Include     string `json:"include"`
	LiteralText bool   `json:"literal_text"`
}

type grepMatch struct {
	path     string
	modTime  time.Time
	lineNum  int
	lineText string
}

type GrepResponseMetadata struct {
	NumberOfMatches int  `json:"number_of_matches"`
	Truncated       bool `json:"truncated"`
}

type grepTool struct{}

const (
	GrepToolName = "grep"
)

func NewGrepTool() BaseTool {
	return &grepTool{}
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
				"description": "The directory to search in. Defaults to the current working directory.",
			},
			"include": map[string]any{
				"type":        "string",
				"description": "File pattern to include in the search (e.g. \"*.js\", \"*.{ts,tsx}\")",
			},
			"literal_text": map[string]any{
				"type":        "boolean",
				"description": "If true, the pattern will be treated as literal text with special regex characters escaped. Default is false.",
			},
		},
		Required: []string{"pattern"},
	}
}

// escapeRegexPattern escapes special regex characters so they're treated as literal characters
func escapeRegexPattern(pattern string) string {
	specialChars := []string{"\\", ".", "+", "*", "?", "(", ")", "[", "]", "{", "}", "^", "$", "|"}
	escaped := pattern

	for _, char := range specialChars {
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}

	return escaped
}

func (g *grepTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params GrepParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	if params.Pattern == "" {
		return NewTextErrorResponse("pattern is required"), nil
	}

	// If literal_text is true, escape the pattern
	searchPattern := params.Pattern
	if params.LiteralText {
		searchPattern = escapeRegexPattern(params.Pattern)
	}

	searchPath := params.Path
	if searchPath == "" {
		var err error
		searchPath, err = GetWorkingDirectory(ctx)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	matches, truncated, err := searchFiles(searchPattern, searchPath, params.Include, 100)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error searching files: %w", err)
	}

	var output string
	if len(matches) == 0 {
		output = "No files found"
	} else {
		output = fmt.Sprintf("Found %d matches\n", len(matches))

		currentFile := ""
		for _, match := range matches {
			if currentFile != match.path {
				if currentFile != "" {
					output += "\n"
				}
				currentFile = match.path
				output += fmt.Sprintf("%s:\n", match.path)
			}
			if match.lineNum > 0 {
				output += fmt.Sprintf("  Line %d: %s\n", match.lineNum, match.lineText)
			} else {
				output += fmt.Sprintf("  %s\n", match.path)
			}
		}

		if truncated {
			output += "\n(Results are truncated. Consider using a more specific path or pattern.)"
		}
	}

	return WithResponseMetadata(
		NewTextResponse(output),
		GrepResponseMetadata{
			NumberOfMatches: len(matches),
			Truncated:       truncated,
		},
	), nil
}

func searchFiles(pattern, rootPath, include string, limit int) ([]grepMatch, bool, error) {
	matches, err := searchWithRipgrep(pattern, rootPath, include)
	if err != nil {
		matches, err = searchFilesWithRegex(pattern, rootPath, include)
		if err != nil {
			return nil, false, err
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	truncated := len(matches) > limit
	if truncated {
		matches = matches[:limit]
	}

	return matches, truncated, nil
}

func searchWithRipgrep(pattern, path, include string) ([]grepMatch, error) {
	_, err := exec.LookPath("rg")
	if err != nil {
		return nil, fmt.Errorf("ripgrep not found: %w", err)
	}

	// Use -n to show line numbers and include the matched line
	args := []string{"-H", "-n", pattern}
	if include != "" {
		args = append(args, "--glob", include)
	}
	args = append(args, path)

	cmd := exec.Command("rg", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []grepMatch{}, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	matches := make([]grepMatch, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse ripgrep output format: file:line:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		filePath := parts[0]
		lineNum, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		lineText := parts[2]

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue // Skip files we can't access
		}

		matches = append(matches, grepMatch{
			path:     filePath,
			modTime:  fileInfo.ModTime(),
			lineNum:  lineNum,
			lineText: lineText,
		})
	}

	return matches, nil
}

func searchFilesWithRegex(pattern, rootPath, include string) ([]grepMatch, error) {
	matches := []grepMatch{}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var includePattern *regexp.Regexp
	if include != "" {
		regexPattern := globToRegex(include)
		includePattern, err = regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid include pattern: %w", err)
		}
	}

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil // Skip directories
		}

		if fileutil.SkipHidden(path) {
			return nil
		}

		if includePattern != nil && !includePattern.MatchString(path) {
			return nil
		}

		match, lineNum, lineText, err := fileContainsPattern(path, regex)
		if err != nil {
			return nil // Skip files we can't read
		}

		if match {
			matches = append(matches, grepMatch{
				path:     path,
				modTime:  info.ModTime(),
				lineNum:  lineNum,
				lineText: lineText,
			})

			if len(matches) >= 200 {
				return filepath.SkipAll
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func fileContainsPattern(filePath string, pattern *regexp.Regexp) (bool, int, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, 0, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if pattern.MatchString(line) {
			return true, lineNum, line, nil
		}
	}

	return false, 0, "", scanner.Err()
}

func globToRegex(glob string) string {
	regexPattern := strings.ReplaceAll(glob, ".", "\\.")
	regexPattern = strings.ReplaceAll(regexPattern, "*", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "?", ".")

	re := regexp.MustCompile(`\{([^}]+)\}`)
	regexPattern = re.ReplaceAllStringFunc(regexPattern, func(match string) string {
		inner := match[1 : len(match)-1]
		return "(" + strings.ReplaceAll(inner, ",", "|") + ")"
	})

	return regexPattern
}

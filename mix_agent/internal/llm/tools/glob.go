package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mix/internal/fileutil"
	"mix/internal/logging"
)

const (
	GlobToolName = "Glob"
)

type GlobParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

type GlobResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

type globTool struct{}

func NewGlobTool() BaseTool {
	return &globTool{}
}

func (g *globTool) Info() ToolInfo {
	return ToolInfo{
		Name:        GlobToolName,
		Description: LoadToolDescription("glob"),
		Parameters: map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The glob pattern to match files against",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The directory to search in. Defaults to the current working directory.",
			},
		},
		Required: []string{"pattern"},
	}
}

func (g *globTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params GlobParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	if params.Pattern == "" {
		return NewTextErrorResponse("pattern is required"), nil
	}

	searchPath := params.Path
	if searchPath == "" {
		var err error
		searchPath, err = GetSessionStorageDirectory(ctx)
		if err != nil {
			return ToolResponse{}, fmt.Errorf("failed to get session storage directory: %w", err)
		}
	}

	files, truncated, err := globFiles(params.Pattern, searchPath, 100)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error finding files: %w", err)
	}

	var output string
	if len(files) == 0 {
		output = "No files found"
	} else {
		output = strings.Join(files, "\n")
		if truncated {
			output += "\n\n(Results are truncated. Consider using a more specific path or pattern.)"
		}
	}

	return WithResponseMetadata(
		NewTextResponse(output),
		GlobResponseMetadata{
			NumberOfFiles: len(files),
			Truncated:     truncated,
		},
	), nil
}

func globFiles(pattern, searchPath string, limit int) ([]string, bool, error) {
	// Check if search path exists
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return nil, false, fmt.Errorf("directory does not exist: %s", searchPath)
	}

	cmdRg := fileutil.GetRgCmd(pattern)
	if cmdRg != nil {
		cmdRg.Dir = searchPath
		matches, truncated, err := runRipgrep(cmdRg, searchPath, limit)
		if err == nil {
			return matches, truncated, nil
		}
		logging.Warn(fmt.Sprintf("Ripgrep execution failed: %v. Falling back to doublestar.", err))
	}

	return fileutil.GlobWithDoublestar(pattern, searchPath, limit)
}

func runRipgrep(cmd *exec.Cmd, searchRoot string, limit int) ([]string, bool, error) {
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("ripgrep: %w\n%s", err, out)
	}

	var matches []string
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) == 0 {
			continue
		}
		absPath := string(p)
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(searchRoot, absPath)
		}
		if fileutil.SkipHidden(absPath) {
			continue
		}
		matches = append(matches, absPath)
	}

	// Get file modification times and sort by most recent first
	type fileWithTime struct {
		path    string
		modTime time.Time
	}

	filesWithTimes := make([]fileWithTime, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		filesWithTimes = append(filesWithTimes, fileWithTime{
			path:    path,
			modTime: info.ModTime(),
		})
	}

	sort.SliceStable(filesWithTimes, func(i, j int) bool {
		return filesWithTimes[i].modTime.After(filesWithTimes[j].modTime)
	})

	// Apply limit and extract paths
	result := make([]string, 0, len(filesWithTimes))
	limitToApply := len(filesWithTimes)
	if limit > 0 && limit < len(filesWithTimes) {
		limitToApply = limit
	}
	for i := 0; i < limitToApply; i++ {
		result = append(result, filesWithTimes[i].path)
	}

	return result, limit > 0 && len(filesWithTimes) > limit, nil
}

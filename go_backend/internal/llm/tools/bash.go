package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mix/internal/config"
	"mix/internal/llm/models"
	"mix/internal/llm/tools/shell"
	"mix/internal/logging"
	"mix/internal/permission"
)

type BashParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type BashPermissionsParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type BashResponseMetadata struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}
type bashTool struct {
	permissions permission.Service
}

const (
	BashToolName = "bash"

	DefaultTimeout  = 1 * 60 * 1000  // 1 minutes in milliseconds
	MaxTimeout      = 10 * 60 * 1000 // 10 minutes in milliseconds
	MaxOutputLength = 30000
)

var bannedCommands = []string{
	"alias", "curlie", "wget", "axel", "aria2c",
	"nc", "telnet", "lynx", "w3m", "links", "httpie", "xh",
	"http-prompt", "chrome", "firefox", "safari",
}

var safeReadOnlyCommands = []string{
	"ls", "echo", "pwd", "date", "cal", "uptime", "whoami", "id", "groups", "env", "printenv", "set", "unset", "which", "type", "whereis",
	"whatis", "uname", "hostname", "df", "du", "free", "top", "ps", "kill", "killall", "nice", "nohup", "time", "timeout",

	"git status", "git log", "git diff", "git show", "git branch", "git tag", "git remote", "git ls-files", "git ls-remote",
	"git rev-parse", "git config --get", "git config --list", "git describe", "git blame", "git grep", "git shortlog",

	"go version", "go help", "go list", "go env", "go doc", "go vet", "go fmt", "go mod", "go test", "go build", "go run", "go install", "go clean",
}

func bashDescription() string {
	return LoadToolDescription("bash")
}

func NewBashTool(permission permission.Service) BaseTool {
	return &bashTool{
		permissions: permission,
	}
}

func (b *bashTool) Info() ToolInfo {
	return ToolInfo{
		Name:        BashToolName,
		Description: bashDescription(),
		Parameters: map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The command to execute",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Optional timeout in milliseconds (max 600000)",
			},
		},
		Required: []string{"command"},
	}
}

func (b *bashTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params BashParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	if params.Timeout > MaxTimeout {
		params.Timeout = MaxTimeout
	} else if params.Timeout <= 0 {
		params.Timeout = DefaultTimeout
	}

	if params.Command == "" {
		return NewTextErrorResponse("missing command"), nil
	}

	baseCmd := strings.Fields(params.Command)[0]
	for _, banned := range bannedCommands {
		if strings.EqualFold(baseCmd, banned) {
			return NewTextErrorResponse(fmt.Sprintf("command '%s' is not allowed", baseCmd)), nil
		}
	}

	isSafeReadOnly := false
	cmdLower := strings.ToLower(params.Command)

	for _, safe := range safeReadOnlyCommands {
		if strings.HasPrefix(cmdLower, strings.ToLower(safe)) {
			if len(cmdLower) == len(safe) || cmdLower[len(safe)] == ' ' || cmdLower[len(safe)] == '-' {
				isSafeReadOnly = true
				break
			}
		}
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a new file")
	}
	startTime := time.Now()
	sessionStorageDir, err := GetSessionStorageDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get session storage directory: %w", err)
	}

	if !isSafeReadOnly {
		p := b.permissions.Request(
			permission.CreatePermissionRequest{
				SessionID:   sessionID,
				Path:        sessionStorageDir,
				ToolName:    BashToolName,
				Action:      fmt.Sprintf("Execute command: %s", params.Command),
				Description: fmt.Sprintf("Execute command: %s", params.Command),
				Params: BashPermissionsParams{
					Command: params.Command,
				},
			},
		)
		if !p {
			return ToolResponse{}, permission.ErrorPermissionDenied
		}
	}

	// Check for multimodal analyzer commands and handle them specially
	if strings.Contains(params.Command, "multimodal-analyzer") {
		// Check auth first
		if authResponse := b.checkMultimodalAnalyzerAuth(); authResponse != nil {
			return *authResponse, nil
		}

		// Convert any session URLs to local paths
		params.Command = b.convertSessionURLsToLocalPaths(ctx, params.Command)
	}

	shell := shell.GetPersistentShell(sessionStorageDir)
	
	// For multimodal analyzer commands, ensure Gemini API key is available in environment
	if strings.Contains(params.Command, "multimodal-analyzer") {
		b.ensureGeminiAPIKeyInEnvironment(ctx, shell)
	}
	
	stdout, stderr, exitCode, interrupted, err := shell.Exec(ctx, params.Command, params.Timeout)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error executing command: %w", err)
	}

	stdout = truncateOutput(stdout)
	stderr = truncateOutput(stderr)

	errorMessage := stderr
	if interrupted {
		if errorMessage != "" {
			errorMessage += "\n"
		}
		errorMessage += "Command was aborted before completion"
	} else if exitCode != 0 {
		if errorMessage != "" {
			errorMessage += "\n"
		}
		errorMessage += fmt.Sprintf("Exit code %d", exitCode)
	}

	hasBothOutputs := stdout != "" && stderr != ""

	if hasBothOutputs {
		stdout += "\n"
	}

	if errorMessage != "" {
		stdout += "\n" + errorMessage
	}

	metadata := BashResponseMetadata{
		StartTime: startTime.UnixMilli(),
		EndTime:   time.Now().UnixMilli(),
	}
	if stdout == "" {
		return WithResponseMetadata(NewTextResponse("no output"), metadata), nil
	}
	return WithResponseMetadata(NewTextResponse(stdout), metadata), nil
}

func truncateOutput(content string) string {
	if len(content) <= MaxOutputLength {
		return content
	}

	halfLength := MaxOutputLength / 2
	start := content[:halfLength]
	end := content[len(content)-halfLength:]

	truncatedLinesCount := countLines(content[halfLength : len(content)-halfLength])
	return fmt.Sprintf("%s\n\n... [%d lines truncated] ...\n\n%s", start, truncatedLinesCount, end)
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}

// checkMultimodalAnalyzerAuth checks if the multimodal analyzer can authenticate with Gemini API
func (b *bashTool) checkMultimodalAnalyzerAuth() *ToolResponse {
	logging.Info("Checking multimodal analyzer authentication")
	
	// First check database for Gemini API key
	credentialsService := config.GetAPICredentials()
	if credentialsService != nil {
		ctx := context.Background()
		apiKey, err := credentialsService.GetAPIKey(ctx, models.ProviderGemini)
		if err == nil && apiKey != "" {
			logging.Info("Found Gemini API key in database for multimodal analyzer")
			return nil // API key is available from database, proceed with execution
		}
	}
	
	// Fallback: Check if GEMINI_API_KEY environment variable is set
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		logging.Info("Found Gemini API key in environment for multimodal analyzer")
		return nil // API key is available from environment, proceed with execution
	}

	// Create helpful error message with instructions
	errorMsg := `❌ Multimodal Analyzer Authentication Required

The multimodal analyzer needs a Gemini API key to analyze media files.

🔧 How to fix this:

Option 1 - Set API key through the UI (Recommended):
1. Get a Gemini API key from Google AI Studio:
   https://makersuite.google.com/app/apikey

2. Use the login command in the chat:
   /login gemini

3. Enter your Gemini API key when prompted

Option 2 - Set environment variable:
1. Set the environment variable:
   export GEMINI_API_KEY="your_api_key_here"

2. Or add it to your shell profile (~/.bashrc, ~/.zshrc):
   echo 'export GEMINI_API_KEY="your_api_key_here"' >> ~/.bashrc

3. Restart your terminal or run:
   source ~/.bashrc

Once the API key is set, you can use the multimodal analyzer to analyze images, audio, and video files.`

	response := NewTextErrorResponse(errorMsg)
	return &response
}

// ensureGeminiAPIKeyInEnvironment sets the GEMINI_API_KEY environment variable in the shell
// if it's available in the database but not in the current environment
func (b *bashTool) ensureGeminiAPIKeyInEnvironment(ctx context.Context, shell *shell.PersistentShell) {
	// Check if GEMINI_API_KEY is already set in environment
	if os.Getenv("GEMINI_API_KEY") != "" {
		return // Already set, nothing to do
	}
	
	// Try to get API key from database
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		return
	}
	
	apiKey, err := credentialsService.GetAPIKey(ctx, models.ProviderGemini)
	if err != nil || apiKey == "" {
		return // No key in database
	}
	
	// Set the environment variable in the shell
	exportCmd := fmt.Sprintf("export GEMINI_API_KEY='%s'", apiKey)
	_, _, _, _, execErr := shell.Exec(ctx, exportCmd, 5000) // 5 second timeout
	if execErr != nil {
		logging.Error("Failed to set GEMINI_API_KEY in shell environment", "error", execErr)
	} else {
		logging.Info("Set Gemini API key from database in shell environment for multimodal analyzer")
	}
}

// convertSessionURLsToLocalPaths converts localhost session URLs to local storage paths
func (b *bashTool) convertSessionURLsToLocalPaths(ctx context.Context, command string) string {
	// Regex to match localhost session file URLs
	// Pattern: http://localhost:8088/api/sessions/{sessionId}/files/{filename}
	urlPattern := regexp.MustCompile(`http://localhost:8088/api/sessions/([^/]+)/files/([^"\s]+)`)

	return urlPattern.ReplaceAllStringFunc(command, func(match string) string {
		matches := urlPattern.FindStringSubmatch(match)
		if len(matches) != 3 {
			return match // Return original if parsing fails
		}

		filename := matches[2]

		// Get session storage directory for this session
		sessionStorageDir, err := GetSessionStorageDirectory(ctx)
		if err != nil {
			logging.Warn("Failed to get session storage directory for URL conversion", "error", err)
			return match // Return original URL if can't get storage dir
		}

		// Construct local file path
		localPath := filepath.Join(sessionStorageDir, filename)

		logging.Info("Converted session URL to local path", "url", match, "localPath", localPath)
		return localPath
	})
}

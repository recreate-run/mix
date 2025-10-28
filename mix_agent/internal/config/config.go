// Package config manages application configuration for embedded binary use.
package config

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"mix/internal/credentials"
	"mix/internal/database"
	"mix/internal/llm/models"
	"mix/internal/logging"
	"mix/internal/preferences"
)

//go:embed all:prompts
var embeddedPrompts embed.FS

// MCPType defines the type of MCP (Model Control Protocol) server.
type MCPType string

// Supported MCP types
const (
	MCPStdio MCPType = "stdio"
	MCPSse   MCPType = "sse"
)

// MCPServer defines the configuration for a Model Control Protocol server.
type MCPServer struct {
	Command      string            `json:"command"`
	Env          []string          `json:"env"`
	Args         []string          `json:"args"`
	Type         MCPType           `json:"type"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers"`
	AllowedTools []string          `json:"allowedTools,omitempty"`
	DeniedTools  []string          `json:"deniedTools,omitempty"`
}

type AgentName string

const (
	AgentMain AgentName = "main" // Single main agent for embedded use
	AgentSub  AgentName = "sub"  // Sub-agent for tool dispatch tasks
)

// Agent defines configuration for different LLM models and their token limits.
type Agent struct {
	Model           models.ModelID `json:"model"`
	MaxTokens       int64          `json:"maxTokens"`
	ReasoningEffort string         `json:"reasoningEffort"` // For openai models low,medium,heigh
}

// Provider struct removed - providers now managed by database API credentials service

// Data defines storage configuration.
type Data struct {
	Directory string `json:"directory,omitempty"`
}

// Removed LSP configs for embedded binary

// ShellConfig defines the configuration for the shell used by the bash tool.
type ShellConfig struct {
	Path string   `json:"path,omitempty"`
	Args []string `json:"args,omitempty"`
}

// Config is the simplified configuration structure for embedded binary.
type Config struct {
	Data       Data                 `json:"data"`
	Database   database.Config      `json:"database"`
	WorkingDir string               `json:"wd,omitempty"`
	PromptsDir string               `json:"promptsDir,omitempty"`
	MCPServers map[string]MCPServer `json:"mcpServers,omitempty"`
	// Providers removed - managed by database API credentials service
	Agents           map[AgentName]Agent `json:"agents,omitempty"`
	Debug            bool                `json:"debug,omitempty"`
	Shell            ShellConfig         `json:"shell,omitempty"`
	SkipPermissions  bool                `json:"skipPermissions,omitempty"`
	AnalyticsEnabled bool                `json:"analyticsEnabled,omitempty"`
}

// Application constants
const (
	defaultDataDirectory = ".mix"
	defaultLogLevel      = "info"
	appName              = "mix"

	MaxTokensFallbackDefault = 4096
)

// Global configuration instance
var cfg *Config

// Mutex to protect concurrent access to cfg
var cfgMutex sync.RWMutex

// Global user preferences service
var userPreferencesService preferences.Service

// Global API credentials service
var apiCredentialsService *credentials.APICredentialsService

// Load initializes the configuration from environment variables only.
// Agent configurations are now loaded from database via UserPreferencesService.
// If debug is true, debug mode is enabled and log level is set to debug.
// If skipPermissions is true, all permission prompts will be bypassed.
func Load(sessionStorageDir string, debug bool, skipPermissions bool) (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	// Get home directory for data directory
	homeDir, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	promptsDir := filepath.Join(homeDir.HomeDir, ".mix", "prompts")

	cfg = &Config{
		Data: Data{
			Directory: defaultDataDirectory,
		},
		Database:   loadDatabaseConfig(),
		WorkingDir: sessionStorageDir,
		PromptsDir: promptsDir,
		MCPServers: getDefaultMCPServers(),
		// Providers removed - managed by database API credentials service
		Agents:          make(map[AgentName]Agent), // Keep for legacy compatibility but unused
		SkipPermissions: skipPermissions,
		Debug:           debug,
		Shell: ShellConfig{
			Path: getShellPath(),
			Args: []string{"-l"},
		},
		AnalyticsEnabled: getAnalyticsEnabled(),
	}

	// Providers now managed entirely through database API credentials service

	// Apply default values for MCP servers
	applyDefaultValues()

	// Ensure embedded .mix directory structure is written to home directory
	if err := ensureEmbeddedDataDirectory(); err != nil {
		return cfg, fmt.Errorf("failed to initialize embedded data directory: %w", err)
	}

	// Configure logging
	setupLogging(debug)

	// No validation needed - providers and agents managed by database services

	return cfg, nil
}

// Helper functions for simplified configuration loading

// getShellPath gets the shell path from environment or returns default
func getShellPath() string {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return "/bin/bash"
	}
	return shellPath
}

// getAnalyticsEnabled checks if analytics are enabled from environment
func getAnalyticsEnabled() bool {
	analyticsEnabled := os.Getenv("MIX_ANALYTICS_ENABLED")
	if analyticsEnabled != "" {
		return analyticsEnabled == "true" || analyticsEnabled == "1"
	}
	return true // Default to true for backward compatibility
}

// getDefaultMCPServers returns the default MCP server configurations
func getDefaultMCPServers() map[string]MCPServer {
	return map[string]MCPServer{
		"blender": {
			Type:         MCPStdio,
			Command:      "uvx",
			Args:         []string{"blender-mcp"},
			Env:          []string{},
			AllowedTools: []string{"execute_blender_code"},
		},
	}
}

// loadDatabaseConfig loads database configuration from environment variables
func loadDatabaseConfig() database.Config {
	dbType := strings.ToLower(getEnvOrDefault("MIX_DB_TYPE", string(database.ProviderSQLite)))

	config := database.Config{
		Type: database.ProviderType(dbType),
		SQLite: database.SQLiteConfig{
			DataDir:  getEnvOrDefault("MIX_DB_SQLITE_DATA_DIR", defaultDataDirectory),
			Filename: getEnvOrDefault("MIX_DB_SQLITE_FILENAME", "mix.db"),
		},
		Turso: database.TursoConfig{
			URL:       os.Getenv("MIX_DB_TURSO_URL"),
			AuthToken: os.Getenv("MIX_DB_TURSO_AUTH_TOKEN"),
		},
	}

	// Validate the database type
	switch config.Type {
	case database.ProviderSQLite, database.ProviderTurso:
		// Valid types
	default:
		logging.Debug("Invalid database type, defaulting to SQLite", "type", dbType)
		config.Type = database.ProviderSQLite
	}

	return config
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// initializeProviders removed - providers now managed by database API credentials service

// setProviderDefaults removed - providers now initialized directly from environment

// setupLogging configures the application logger
func setupLogging(debug bool) {
	defaultLevel := slog.LevelInfo
	if debug {
		defaultLevel = slog.LevelDebug
	}

	if os.Getenv("_DEV_DEBUG") == "true" {
		loggingFile := fmt.Sprintf("%s/%s", cfg.Data.Directory, "debug.log")
		messagesPath := fmt.Sprintf("%s/%s", cfg.Data.Directory, "messages")

		// Create directories and files if they don't exist
		if _, err := os.Stat(loggingFile); os.IsNotExist(err) {
			if err := os.MkdirAll(cfg.Data.Directory, 0o755); err == nil {
				if _, err := os.Create(loggingFile); err != nil {
					panic(fmt.Sprintf("failed to create logging file: %v", err))
				}
			}
		}

		if _, err := os.Stat(messagesPath); os.IsNotExist(err) {
			if err := os.MkdirAll(messagesPath, 0o756); err != nil {
				panic(fmt.Sprintf("failed to create messages directory: %v", err))
			}
		}

		if sloggingFileWriter, err := os.OpenFile(loggingFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666); err == nil {
			logger := slog.New(slog.NewTextHandler(sloggingFileWriter, &slog.HandlerOptions{
				Level: defaultLevel,
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}))
			slog.SetDefault(logger)
			return
		}
	}

	// Default console logging
	logger := slog.New(slog.NewTextHandler(logging.NewWriter(), &slog.HandlerOptions{
		Level: defaultLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	slog.SetDefault(logger)
}

// ensureEmbeddedDataDirectory ensures an empty .mix directory exists in the home directory
func ensureEmbeddedDataDirectory() error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Target .mix directory in home
	targetMixDir := filepath.Join(homeDir, ".mix")

	// Create empty .mix directory if it doesn't exist
	if err := os.MkdirAll(targetMixDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .mix directory: %w", err)
	}

	return nil
}

func applyDefaultValues() {
	// Set default MCP type if not specified
	cfgMutex.Lock()
	for k, v := range cfg.MCPServers {
		if v.Type == "" {
			v.Type = MCPStdio
			cfg.MCPServers[k] = v
		}
	}
	cfgMutex.Unlock()
}

// Get returns the current configuration.
// It's safe to call this function multiple times.
func Get() *Config {
	return cfg
}

// ResetForTesting resets the configuration for testing purposes
func ResetForTesting() {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	cfg = nil
}

// InitUserPreferences initializes the user preferences service with database connection
// This should be called after database connection is established
func InitUserPreferences(database *sql.DB) error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	userPreferencesService = preferences.NewUserPreferencesService(database)
	return nil
}

// GetUserPreferences returns the user preferences service
func GetUserPreferences() preferences.Service {
	cfgMutex.RLock()
	defer cfgMutex.RUnlock()
	return userPreferencesService
}

// InitAPICredentials initializes the API credentials service with database connection
// This should be called after database connection is established
func InitAPICredentials(database *sql.DB) error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	// Generate encryption key for API key storage
	encryptionKey, err := credentials.GenerateEncryptionKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	apiCredentialsService = credentials.NewAPICredentialsService(database, encryptionKey)
	return nil
}

// GetAPICredentials returns the API credentials service
func GetAPICredentials() *credentials.APICredentialsService {
	cfgMutex.RLock()
	defer cfgMutex.RUnlock()
	return apiCredentialsService
}

// GetAgentFromDatabase returns agent configuration from database
func GetAgentFromDatabase(ctx context.Context, agentName AgentName) (Agent, error) {
	if userPreferencesService == nil {
		return Agent{}, fmt.Errorf("user preferences service not initialized")
	}

	// Convert config agent name to preferences agent name
	var prefAgentName preferences.AgentName
	switch agentName {
	case AgentMain:
		prefAgentName = preferences.AgentMain
	case AgentSub:
		prefAgentName = preferences.AgentSub
	default:
		return Agent{}, fmt.Errorf("unknown agent name: %s", agentName)
	}

	prefAgent, err := userPreferencesService.GetAgentConfig(ctx, prefAgentName)
	if err != nil {
		return Agent{}, err
	}

	// Convert preferences agent to config agent
	return Agent{
		Model:           prefAgent.Model,
		MaxTokens:       prefAgent.MaxTokens,
		ReasoningEffort: prefAgent.ReasoningEffort,
	}, nil
}

// GetEmbeddedPrompts returns the embedded prompts filesystem
func GetEmbeddedPrompts() embed.FS {
	return embeddedPrompts
}

// PromptsDirectory returns the prompts directory from the configuration.
func PromptsDirectory() (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config not loaded")
	}
	return cfg.PromptsDir, nil
}

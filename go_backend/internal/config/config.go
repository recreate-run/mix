// Package config manages application configuration for embedded binary use.
package config

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"mix/internal/llm/models"
	"mix/internal/logging"

	"github.com/spf13/viper"
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

// Provider defines configuration for an LLM provider.
type Provider struct {
	APIKey   string `json:"apiKey"`
	Disabled bool   `json:"disabled"`
}

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
	Data            Data                              `json:"data"`
	WorkingDir      string                            `json:"wd,omitempty"`
	PromptsDir      string                            `json:"promptsDir,omitempty"`
	MCPServers      map[string]MCPServer              `json:"mcpServers,omitempty"`
	Providers       map[models.ModelProvider]Provider `json:"providers,omitempty"`
	Agents          map[AgentName]Agent               `json:"agents,omitempty"`
	Debug           bool                              `json:"debug,omitempty"`
	ContextPaths    []string                          `json:"contextPaths,omitempty"`
	Shell           ShellConfig                       `json:"shell,omitempty"`
	SkipPermissions bool                              `json:"skipPermissions,omitempty"`
}

// Application constants
const (
	defaultDataDirectory = ".mix"
	defaultLogLevel      = "info"
	appName              = "mix"

	MaxTokensFallbackDefault = 4096
)

var defaultContextPaths = []string{
	"MIX.md",
}

// getDefaultConfig returns the hardcoded default configuration
func getDefaultConfig() *Config {
	return &Config{
		Data: Data{
			Directory: ".mix",
		},
		ContextPaths: []string{"MIX.md"},
		Shell: ShellConfig{
			Path: "",
			Args: []string{"-l"},
		},
		Debug:           false,
		SkipPermissions: false,
		MCPServers:      make(map[string]MCPServer),
		Providers: map[models.ModelProvider]Provider{
			models.ProviderAnthropic: {
				APIKey:   "",
				Disabled: false,
			},
		},
		Agents: map[AgentName]Agent{
			AgentMain: {
				Model:     "claude-4-sonnet",
				MaxTokens: 4096,
			},
			AgentSub: {
				Model:     "claude-4-sonnet",
				MaxTokens: 2048,
			},
		},
	}
}

// Global configuration instance
var cfg *Config

// Mutex to protect concurrent access to cfg
var cfgMutex sync.RWMutex

// Load initializes the configuration from environment variables and config files.
// If debug is true, debug mode is enabled and log level is set to debug.
// If skipPermissions is true, all permission prompts will be bypassed.
// It returns an error if configuration loading fails.
func Load(workingDir string, debug bool, skipPermissions bool) (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	configureViper()
	setDefaults(debug)

	// Ensure config file exists in home directory
	if err := ensureConfigFile(); err != nil {
		return nil, fmt.Errorf("failed to initialize config file: %w", err)
	}

	// Read global config
	if err := readConfig(viper.ReadInConfig()); err != nil {
		return nil, err
	}

	// Load and merge local config
	mergeLocalConfig(workingDir)

	// Get prompts directory from config with default expansion
	promptsDir := viper.GetString("promptsDir")
	if promptsDir == "" {
		homeDir, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		promptsDir = filepath.Join(homeDir.HomeDir, ".mix", "prompts")
	} else if strings.HasPrefix(promptsDir, "~/") {
		// Expand ~ to home directory
		homeDir, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		promptsDir = filepath.Join(homeDir.HomeDir, promptsDir[2:])
	}

	cfg = &Config{
		WorkingDir:      workingDir,
		PromptsDir:      promptsDir,
		MCPServers:      make(map[string]MCPServer),
		Providers:       make(map[models.ModelProvider]Provider),
		SkipPermissions: skipPermissions,
	}

	setProviderDefaults()

	// Apply configuration to the struct
	if err := viper.Unmarshal(cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Restore prompts directory after viper unmarshal (which overwrites with empty default)
	cfg.PromptsDir = promptsDir

	applyDefaultValues()
	
	// Ensure embedded .mix directory structure is written to home directory
	if err := ensureEmbeddedDataDirectory(); err != nil {
		return cfg, fmt.Errorf("failed to initialize embedded data directory: %w", err)
	}
	
	// Prompts directory no longer needed - all prompts are embedded
	defaultLevel := slog.LevelInfo
	if cfg.Debug {
		defaultLevel = slog.LevelDebug
	}
	if os.Getenv("_DEV_DEBUG") == "true" {
		loggingFile := fmt.Sprintf("%s/%s", cfg.Data.Directory, "debug.log")
		messagesPath := fmt.Sprintf("%s/%s", cfg.Data.Directory, "messages")

		// if file does not exist create it
		if _, err := os.Stat(loggingFile); os.IsNotExist(err) {
			if err := os.MkdirAll(cfg.Data.Directory, 0o755); err != nil {
				return cfg, fmt.Errorf("failed to create directory: %w", err)
			}
			if _, err := os.Create(loggingFile); err != nil {
				return cfg, fmt.Errorf("failed to create log file: %w", err)
			}
		}

		if _, err := os.Stat(messagesPath); os.IsNotExist(err) {
			if err := os.MkdirAll(messagesPath, 0o756); err != nil {
				return cfg, fmt.Errorf("failed to create directory: %w", err)
			}
		}
		// Message directory setting removed for embedded binary

		sloggingFileWriter, err := os.OpenFile(loggingFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return cfg, fmt.Errorf("failed to open log file: %w", err)
		}
		// Configure logger without timestamps
		logger := slog.New(slog.NewTextHandler(sloggingFileWriter, &slog.HandlerOptions{
			Level: defaultLevel,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Remove the time attribute
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		}))
		slog.SetDefault(logger)
	} else {
		// Configure logger without timestamps
		logger := slog.New(slog.NewTextHandler(logging.NewWriter(), &slog.HandlerOptions{
			Level: defaultLevel,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Remove the time attribute
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		}))
		slog.SetDefault(logger)
	}

	// Validate configuration
	if err := Validate(); err != nil {
		return cfg, fmt.Errorf("config validation failed: %w", err)
	}

	if cfg.Agents == nil {
		cfg.Agents = make(map[AgentName]Agent)
	}

	// Require explicit agent configuration
	cfgMutex.RLock()
	_, mainExists := cfg.Agents[AgentMain]
	_, subExists := cfg.Agents[AgentSub]
	cfgMutex.RUnlock()

	if !mainExists {
		return cfg, fmt.Errorf("main agent not configured - please specify model in configuration file")
	}
	if !subExists {
		return cfg, fmt.Errorf("sub agent not configured - please specify model in configuration file")
	}
	return cfg, nil
}

// configureViper sets up viper's configuration paths and environment variables.
func configureViper() {
	viper.SetConfigName(fmt.Sprintf(".%s", appName))
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(fmt.Sprintf("$XDG_CONFIG_HOME/%s", appName))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.config/%s", appName))
	viper.SetEnvPrefix(strings.ToUpper(appName))
	viper.AutomaticEnv()
}

// setDefaults configures default values for embedded binary configuration.
func setDefaults(debug bool) {
	viper.SetDefault("data.directory", defaultDataDirectory)
	viper.SetDefault("contextPaths", defaultContextPaths)
	viper.SetDefault("promptsDir", "")

	// Set default shell from environment or fallback to /bin/bash
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/bash"
	}
	viper.SetDefault("shell.path", shellPath)
	viper.SetDefault("shell.args", []string{"-l"})

	if debug {
		viper.SetDefault("debug", true)
		viper.Set("log.level", "debug")
	} else {
		viper.SetDefault("debug", false)
		viper.SetDefault("log.level", defaultLogLevel)
	}
}

// setProviderDefaults configures LLM provider defaults for embedded binary.
func setProviderDefaults() {

	if apiKey := os.Getenv("AZURE_OPENAI_ENDPOINT"); apiKey != "" {
		// api-key may be empty when using Entra ID credentials – that's okay
		viper.SetDefault("providers.azure.apiKey", os.Getenv("AZURE_OPENAI_API_KEY"))
	}
}

// hasAWSCredentials checks if AWS credentials are available in the environment.
func hasAWSCredentials() bool {
	// Check for explicit AWS credentials
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}

	// Check for AWS profile
	if os.Getenv("AWS_PROFILE") != "" || os.Getenv("AWS_DEFAULT_PROFILE") != "" {
		return true
	}

	// Check for AWS region
	if os.Getenv("AWS_REGION") != "" || os.Getenv("AWS_DEFAULT_REGION") != "" {
		return true
	}

	// Check if running on EC2 with instance profile
	if os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" ||
		os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "" {
		return true
	}

	return false
}

// hasVertexAICredentials checks if VertexAI credentials are available in the environment.
func hasVertexAICredentials() bool {
	// Check for explicit VertexAI parameters
	if os.Getenv("VERTEXAI_PROJECT") != "" && os.Getenv("VERTEXAI_LOCATION") != "" {
		return true
	}
	// Check for Google Cloud project and location
	if os.Getenv("GOOGLE_CLOUD_PROJECT") != "" && (os.Getenv("GOOGLE_CLOUD_REGION") != "" || os.Getenv("GOOGLE_CLOUD_LOCATION") != "") {
		return true
	}
	return false
}

// readConfig handles the result of reading a configuration file.
func readConfig(err error) error {
	if err == nil {
		return nil
	}

	// It's okay if the config file doesn't exist
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		return nil
	}

	return fmt.Errorf("failed to read config: %w", err)
}

// writeEmbeddedResource and writeEmbeddedDirectory functions removed - no longer needed
// as prompts are now loaded directly from embedded filesystem

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

// ensureConfigFile creates a .mix.json file in the home directory if it doesn't exist.
func ensureConfigFile() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configFile := filepath.Join(homeDir, fmt.Sprintf(".%s.json", appName))

	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil {
		// File exists, nothing to do
		return nil
	} else if !os.IsNotExist(err) {
		// Some other error occurred
		return fmt.Errorf("failed to check config file: %w", err)
	}

	// File doesn't exist, create it with hardcoded default config
	defaultCfg := getDefaultConfig()
	configData, err := json.MarshalIndent(defaultCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(configFile, configData, 0o644); err != nil {
		return fmt.Errorf("failed to create config file %s: %w", configFile, err)
	}

	return nil
}

// mergeLocalConfig loads and merges configuration from the local directory.
func mergeLocalConfig(workingDir string) {
	local := viper.New()
	local.SetConfigName(fmt.Sprintf(".%s", appName))
	local.SetConfigType("json")
	local.AddConfigPath(workingDir)

	// Merge local config if it exists
	if err := local.ReadInConfig(); err == nil {
		viper.MergeConfigMap(local.AllSettings())
	}
}

// applyDefaultValues sets default values for configuration fields that need processing.
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

// It validates model IDs and providers, ensuring they are supported.
func validateAgent(cfg *Config, name AgentName, agent Agent) error {
	// Check if model exists
	model, modelExists := models.SupportedModels[agent.Model]
	if !modelExists {
		return fmt.Errorf("unsupported model %s configured for agent %s", agent.Model, name)
	}

	// Check if provider for the model is configured
	provider := model.Provider
	cfgMutex.RLock()
	providerCfg, providerExists := cfg.Providers[provider]
	cfgMutex.RUnlock()

	if !providerExists {
		// Provider not configured, check if we have environment variables
		apiKey := getProviderAPIKey(provider)
		if apiKey == "" && provider != "anthropic" && provider != "openai" {
			return fmt.Errorf("provider %s not configured for agent %s (model %s) and no API key found in environment", provider, name, agent.Model)
		}
		// Add provider - with API key from environment or empty for OAuth-supported providers
		cfgMutex.Lock()
		cfg.Providers[provider] = Provider{
			APIKey: apiKey,
		}
		cfgMutex.Unlock()
		if apiKey != "" {
			logging.Info("added provider from environment", "provider", provider)
		} else {
			logging.Info("added provider without API key (OAuth-supported)", "provider", provider)
		}
	} else if providerCfg.Disabled {
		return fmt.Errorf("provider %s is disabled for agent %s (model %s)", provider, name, agent.Model)
	} else if providerCfg.APIKey == "" && provider != "anthropic" && provider != "openai" {
		return fmt.Errorf("provider %s has no API key configured for agent %s (model %s)", provider, name, agent.Model)
	}

	logging.Info("Selected provider", "agent", name, "model", agent.Model, "provider", provider)

	// Validate max tokens
	if agent.MaxTokens <= 0 {
		logging.Warn("invalid max tokens, setting to default",
			"agent", name,
			"model", agent.Model,
			"max_tokens", agent.MaxTokens)

		// Update the agent with default max tokens
		cfgMutex.Lock()
		updatedAgent := cfg.Agents[name]
		if model.DefaultMaxTokens > 0 {
			updatedAgent.MaxTokens = model.DefaultMaxTokens
		} else {
			updatedAgent.MaxTokens = MaxTokensFallbackDefault
		}
		cfg.Agents[name] = updatedAgent
		cfgMutex.Unlock()
	} else if model.ContextWindow > 0 && agent.MaxTokens > model.ContextWindow/2 {
		// Ensure max tokens doesn't exceed half the context window (reasonable limit)
		logging.Warn("max tokens exceeds half the context window, adjusting",
			"agent", name,
			"model", agent.Model,
			"max_tokens", agent.MaxTokens,
			"context_window", model.ContextWindow)

		// Update the agent with adjusted max tokens
		cfgMutex.Lock()
		updatedAgent := cfg.Agents[name]
		updatedAgent.MaxTokens = model.ContextWindow / 2
		cfg.Agents[name] = updatedAgent
		cfgMutex.Unlock()
	}

	// Validate reasoning effort for models that support reasoning
	if model.CanReason && provider == models.ProviderOpenAI || provider == models.ProviderLocal {
		if agent.ReasoningEffort == "" {
			// Set default reasoning effort for models that support it
			logging.Info("setting default reasoning effort for model that supports reasoning",
				"agent", name,
				"model", agent.Model)

			// Update the agent with default reasoning effort
			cfgMutex.Lock()
			updatedAgent := cfg.Agents[name]
			updatedAgent.ReasoningEffort = "medium"
			cfg.Agents[name] = updatedAgent
			cfgMutex.Unlock()
		} else {
			// Check if reasoning effort is valid (low, medium, high)
			effort := strings.ToLower(agent.ReasoningEffort)
			if effort != "low" && effort != "medium" && effort != "high" {
				logging.Warn("invalid reasoning effort, setting to medium",
					"agent", name,
					"model", agent.Model,
					"reasoning_effort", agent.ReasoningEffort)

				// Update the agent with valid reasoning effort
				cfgMutex.Lock()
				updatedAgent := cfg.Agents[name]
				updatedAgent.ReasoningEffort = "medium"
				cfg.Agents[name] = updatedAgent
				cfgMutex.Unlock()
			}
		}
	} else if !model.CanReason && agent.ReasoningEffort != "" {
		// Model doesn't support reasoning but reasoning effort is set
		logging.Warn("model doesn't support reasoning but reasoning effort is set, ignoring",
			"agent", name,
			"model", agent.Model,
			"reasoning_effort", agent.ReasoningEffort)

		// Update the agent to remove reasoning effort
		cfgMutex.Lock()
		updatedAgent := cfg.Agents[name]
		updatedAgent.ReasoningEffort = ""
		cfg.Agents[name] = updatedAgent
		cfgMutex.Unlock()
	}

	return nil
}

// Validate checks if the configuration is valid and applies defaults where needed.
func Validate() error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Validate agent models
	for name, agent := range cfg.Agents {
		if err := validateAgent(cfg, name, agent); err != nil {
			return err
		}
	}

	// Validate providers
	cfgMutex.Lock()
	for provider, providerCfg := range cfg.Providers {
		// Skip API key validation for providers that support OAuth authentication
		if providerCfg.APIKey == "" && !providerCfg.Disabled && provider != "anthropic" && provider != "openai" {
			fmt.Printf("provider has no API key, marking as disabled %s", provider)
			logging.Warn("provider has no API key, marking as disabled", "provider", provider)
			providerCfg.Disabled = true
			cfg.Providers[provider] = providerCfg
		}
	}
	cfgMutex.Unlock()

	// Removed LSP validation for embedded binary

	return nil
}

// getProviderAPIKey gets the API key for providers from environment variables
func getProviderAPIKey(provider models.ModelProvider) string {
	switch provider {
	case models.ProviderAnthropic:
		return os.Getenv("ANTHROPIC_API_KEY")
	case models.ProviderOpenAI:
		return os.Getenv("OPENAI_API_KEY")
	case models.ProviderGemini:
		return os.Getenv("GEMINI_API_KEY")
	case models.ProviderGROQ:
		return os.Getenv("GROQ_API_KEY")
	case models.ProviderAzure:
		return os.Getenv("AZURE_OPENAI_API_KEY")
	case models.ProviderOpenRouter:
		return os.Getenv("OPENROUTER_API_KEY")
	case models.ProviderBedrock:
		if hasAWSCredentials() {
			return "aws-credentials-available"
		}
	case models.ProviderVertexAI:
		if hasVertexAICredentials() {
			return "vertex-ai-credentials-available"
		}
	}
	return ""
}

func updateCfgFile(updateCfg func(config *Config)) error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Get the config file path
	configFile := viper.ConfigFileUsed()
	var configData []byte
	if configFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configFile = filepath.Join(homeDir, fmt.Sprintf(".%s.json", appName))
		logging.Info("config file not found, creating new one", "path", configFile)
		configData = []byte(`{}`)
	} else {
		// Read the existing config file
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		configData = data
	}

	// Parse the JSON
	var userCfg *Config
	if err := json.Unmarshal(configData, &userCfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	updateCfg(userCfg)

	// Write the updated config back to file
	updatedData, err := json.MarshalIndent(userCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get returns the current configuration.
// It's safe to call this function multiple times.
func Get() *Config {
	return cfg
}

// GetEmbeddedPrompts returns the embedded prompts filesystem
func GetEmbeddedPrompts() embed.FS {
	return embeddedPrompts
}

// LaunchDirectory returns the current launch directory from the configuration.
func LaunchDirectory() (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config not loaded")
	}
	return cfg.WorkingDir, nil
}

// PromptsDirectory returns the prompts directory from the configuration.
func PromptsDirectory() (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config not loaded")
	}
	return cfg.PromptsDir, nil
}

func UpdateAgentModel(agentName AgentName, modelID models.ModelID) error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	model, ok := models.SupportedModels[modelID]
	if !ok {
		return fmt.Errorf("model %s not supported", modelID)
	}

	cfgMutex.Lock()
	existingAgentCfg := cfg.Agents[agentName]

	maxTokens := existingAgentCfg.MaxTokens
	if model.DefaultMaxTokens > 0 {
		maxTokens = model.DefaultMaxTokens
	}

	newAgentCfg := Agent{
		Model:           modelID,
		MaxTokens:       maxTokens,
		ReasoningEffort: existingAgentCfg.ReasoningEffort,
	}
	cfg.Agents[agentName] = newAgentCfg
	cfgMutex.Unlock()

	if err := validateAgent(cfg, agentName, newAgentCfg); err != nil {
		// revert config update on failure
		cfgMutex.Lock()
		cfg.Agents[agentName] = existingAgentCfg
		cfgMutex.Unlock()
		return fmt.Errorf("failed to update agent model: %w", err)
	}

	return updateCfgFile(func(config *Config) {
		if config.Agents == nil {
			config.Agents = make(map[AgentName]Agent)
		}
		config.Agents[agentName] = newAgentCfg
	})
}

// Removed UpdateTheme function for embedded binary

// Removed GitHub token loading for embedded binary

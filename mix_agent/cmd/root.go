package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mix/internal/app"
	"mix/internal/browser"
	"mix/internal/config"
	"mix/internal/constants"
	"mix/internal/database"
	"mix/internal/format"
	httphandlers "mix/internal/http"
	"mix/internal/llm/agent"
	"mix/internal/logging"
	"mix/internal/version"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mix",
	Short: "AI assistant for software development with CLI and HTTP API",
	Long: `Mix is a powerful AI assistant that helps with software development tasks.
It provides both CLI-only mode for direct prompt processing and an HTTP API 
for AI capabilities, file operations, and MCP integration to assist in video generation 
and content creation workflows.`,
	Example: `
  # CLI mode with prompt (direct output)
  mix -p "Explain the use of context in Go"

  # CLI mode with JSON output format
  mix -p "Explain the use of context in Go" -f json

  # Start HTTP API server
  mix --http-port 8080

  # Run with debug logging
  mix -d -p "Your prompt here"

  # Print version
  mix -v
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If the help flag is set, show the help message
		if cmd.Flag("help").Changed {
			_ = cmd.Help()
			return nil
		}
		if cmd.Flag("version").Changed {
			fmt.Println(version.Version)
			return nil
		}

		// Load the config
		debug, _ := cmd.Flags().GetBool("debug")
		cwd, _ := cmd.Flags().GetString("cwd")
		prompt, _ := cmd.Flags().GetString("prompt")
		outputFormat, _ := cmd.Flags().GetString("output-format")
		quiet, _ := cmd.Flags().GetBool("quiet")
		query, _ := cmd.Flags().GetString("query")
		httpPort, _ := cmd.Flags().GetInt("http-port")
		httpHost, _ := cmd.Flags().GetString("http-host")
		skipPermissions, _ := cmd.Flags().GetBool("dangerously-skip-permissions")
		browserMode, _ := cmd.Flags().GetString("browser-mode")
		cdpURL, _ := cmd.Flags().GetString("cdp-url")

		// Load .env file if it exists (ignore error if file doesn't exist)
		_ = godotenv.Load()

		// Validate format option
		if !format.IsValid(outputFormat) {
			return fmt.Errorf("invalid format option: %s\n%s", outputFormat, format.GetHelpText())
		}

		// Determine working directory: use --cwd if provided, otherwise current directory
		if cwd == "" {
			var err error
			cwd, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}
		}

		// Only change directory if --cwd was explicitly provided
		if cmd.Flag("cwd").Changed {
			err := os.Chdir(cwd)
			if err != nil {
				return fmt.Errorf("failed to change directory: %w", err)
			}
		}

		cfg, err := config.Load(cwd, debug, skipPermissions)
		if err != nil {
			return err
		}

		// Create main context for the application
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create database manager with configuration from environment
		dbManager, err := database.NewManager(cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to create database manager: %w", err)
		}

		dbCtx, dbCancel := context.WithTimeout(ctx, constants.DatabaseConnectionTimeout)
		defer dbCancel()

		err = dbManager.Connect(dbCtx)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer func() { _ = dbManager.Close() }()

		appInstance, err := app.New(ctx, dbManager.GetDB())
		if err != nil {
			logging.Error("Failed to create app", "error", err)
			return err
		}
		defer appInstance.Shutdown()

		// Initialize MCP tools early for both modes
		initMCPTools(ctx, appInstance)

		// HTTP server mode (blocks, no other modes)
		if httpPort > 0 {
			// Override BaseURL if it doesn't match the HTTP server port
			// This ensures screenshot URLs and file URLs use the correct server address
			if httpHost == "" {
				httpHost = "localhost"
			}
			expectedBaseURL := fmt.Sprintf("http://%s:%d", httpHost, httpPort)
			if appInstance.BaseURL != expectedBaseURL {
				logging.Warn("BaseURL mismatch detected",
					"configured", appInstance.BaseURL,
					"expected", expectedBaseURL,
					"action", "overriding to match HTTP server")
				appInstance.BaseURL = expectedBaseURL
			}

			// Set up signal handling for graceful shutdown during hot reload
			// This ensures the HTTP server releases the port before the new process starts
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigChan
				logging.Info("Received shutdown signal, initiating graceful shutdown")
				cancel() // Cancel the context to trigger server shutdown
			}()

			return httphandlers.StartServer(ctx, appInstance, httpHost, httpPort)
		}

		// Query mode (structured data output)
		if query != "" {
			return runQuery(ctx, appInstance, query, outputFormat)
		}

		// CLI-only mode (when prompt provided)
		if prompt != "" {
			return appInstance.RunNonInteractive(ctx, prompt, outputFormat, browserMode, cdpURL, quiet)
		}

		// Default: Show help when no mode is specified
		_ = cmd.Help()
		return fmt.Errorf("no mode specified - use --prompt for CLI mode or --http-port for server mode")
	},
}

func initMCPTools(ctx context.Context, appInstance *app.App) {
	go func() {
		defer logging.RecoverPanic("MCP-goroutine", nil)

		// Create a context with timeout for the initial MCP tools fetch
		ctxWithTimeout, cancel := context.WithTimeout(ctx, constants.MCPInitTimeout)
		defer cancel()

		// Set this up once with proper error handling
		// Create temporary manager for initial MCP setup
		tempManager := agent.NewMCPClientManager()
		defer tempManager.Close()
		agent.GetMcpTools(ctxWithTimeout, appInstance.Permissions, tempManager)
	}()
}

func runQuery(ctx context.Context, appInstance *app.App, queryType, outputFormat string) error {
	handler := httphandlers.NewCLIQueryHandler(appInstance)

	// JSON-RPC mode is no longer supported - removed for simplicity
	if queryType == "json" {
		return fmt.Errorf("JSON-RPC mode is no longer supported. Use specific query types: %v",
			handler.GetSupportedQueryTypes())
	}

	result, err := handler.HandleQueryType(ctx, queryType)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	// Format output
	if outputFormat == "json" {
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		// For text output, pretty print
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(jsonBytes))
	}

	return nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("help", "h", false, "Help")
	rootCmd.Flags().BoolP("version", "v", false, "Version")
	rootCmd.Flags().BoolP("debug", "d", false, "Debug")
	rootCmd.Flags().StringP("cwd", "c", "", "Current working directory")

	// CLI-only mode flags
	rootCmd.Flags().StringP("prompt", "p", "", "Run in CLI mode with this prompt")
	rootCmd.Flags().StringP("output-format", "f", format.Text.String(),
		"Output format for CLI-only mode (text, json)")
	rootCmd.Flags().BoolP("quiet", "q", false, "Hide spinner in CLI-only mode")
	rootCmd.Flags().String("browser-mode", browser.DefaultMode,
		"Browser mode for CLI-only mode (electron-embedded-browser, local-browser-service, remote-cdp-websocket)")
	rootCmd.Flags().String("cdp-url", "",
		"CDP WebSocket URL (required when browser-mode is remote-cdp-websocket)")

	// Data query flags
	rootCmd.Flags().String("query", "", "Query structured data: sessions, tools, mcp, commands")

	// HTTP server flags
	rootCmd.Flags().Int("http-port", 0, "Start HTTP REST API server on this port (0 = disabled)")
	rootCmd.Flags().String("http-host", "localhost", "HTTP server host")

	// Permission flags
	rootCmd.Flags().Bool("dangerously-skip-permissions", false, "Skip all permission prompts (DANGEROUS - use only in trusted environments)")

	// Register custom validation for the format flag
	_ = rootCmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return format.SupportedFormats, cobra.ShellCompDirectiveNoFileComp
	})

	// Add subcommands
	rootCmd.AddCommand(authCmd)
}

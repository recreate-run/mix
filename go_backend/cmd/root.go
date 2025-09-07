package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/format"
	httphandlers "mix/internal/http"
	"mix/internal/llm/agent"
	"mix/internal/logging"
	"mix/internal/version"

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
			cmd.Help()
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

		// Validate format option
		if !format.IsValid(outputFormat) {
			return fmt.Errorf("invalid format option: %s\n%s", outputFormat, format.GetHelpText())
		}

		// Determine working directory: use --cwd if provided, otherwise current directory
		if cwd == "" {
			var err error
			cwd, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %v", err)
			}
		}

		// Only change directory if --cwd was explicitly provided
		if cmd.Flag("cwd").Changed {
			err := os.Chdir(cwd)
			if err != nil {
				return fmt.Errorf("failed to change directory: %v", err)
			}
		}

		_, err := config.Load(cwd, debug, skipPermissions)
		if err != nil {
			return err
		}

		// Create main context for the application
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Connect DB with timeout, this will also run migrations
		dbCtx, dbCancel := context.WithTimeout(ctx, db.DBConnectionTimeout)
		defer dbCancel()
		conn, err := db.Connect(dbCtx)
		if err != nil {
			return err
		}

		app, err := app.New(ctx, conn)
		if err != nil {
			logging.Error("Failed to create app: %v", err)
			return err
		}
		defer app.Shutdown()

		// Initialize MCP tools early for both modes
		initMCPTools(ctx, app)

		// HTTP server mode (blocks, no other modes)
		if httpPort > 0 {
			return httphandlers.StartServer(ctx, app, httpHost, httpPort)
		}

		// Query mode (structured data output)
		if query != "" {
			return runQuery(ctx, app, query, outputFormat)
		}

		// CLI-only mode (when prompt provided)
		if prompt != "" {
			return app.RunNonInteractive(ctx, prompt, outputFormat, quiet)
		}

		// Default: Show help when no mode is specified
		cmd.Help()
		return fmt.Errorf("no mode specified - use --prompt for CLI mode or --http-port for server mode")
	},
}

func initMCPTools(ctx context.Context, app *app.App) {
	go func() {
		defer logging.RecoverPanic("MCP-goroutine", nil)

		// Create a context with timeout for the initial MCP tools fetch
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Set this up once with proper error handling
		// Create temporary manager for initial MCP setup
		tempManager := agent.NewMCPClientManager()
		defer tempManager.Close()
		agent.GetMcpTools(ctxWithTimeout, app.Permissions, tempManager)
	}()
}

func runQuery(ctx context.Context, app *app.App, queryType, outputFormat string) error {
	handler := httphandlers.NewCLIQueryHandler(app)

	// JSON-RPC mode is no longer supported - removed for simplicity
	if queryType == "json" {
		return fmt.Errorf("JSON-RPC mode is no longer supported. Use specific query types: %v", 
			handler.GetSupportedQueryTypes())
	}

	result, err := handler.HandleQueryType(ctx, queryType)
	if err != nil {
		return fmt.Errorf("query error: %s", err.Error())
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

	// Data query flags
	rootCmd.Flags().String("query", "", "Query structured data: sessions, tools, mcp, commands")

	// HTTP server flags
	rootCmd.Flags().Int("http-port", 0, "Start HTTP REST API server on this port (0 = disabled)")
	rootCmd.Flags().String("http-host", "localhost", "HTTP server host")

	// Permission flags
	rootCmd.Flags().Bool("dangerously-skip-permissions", false, "Skip all permission prompts (DANGEROUS - use only in trusted environments)")

	// Register custom validation for the format flag
	rootCmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return format.SupportedFormats, cobra.ShellCompDirectiveNoFileComp
	})

	// Add subcommands
	rootCmd.AddCommand(authCmd)
}

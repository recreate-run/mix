package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mix/internal/api"
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

		if cwd != "" {
			err := os.Chdir(cwd)
			if err != nil {
				return fmt.Errorf("failed to change directory: %v", err)
			}
		}
		if cwd == "" {
			c, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %v", err)
			}
			cwd = c
		}
		_, err := config.Load(cwd, debug, skipPermissions)
		if err != nil {
			return err
		}

		// Connect DB, this will also run migrations
		conn, err := db.Connect()
		if err != nil {
			return err
		}

		// Create main context for the application
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

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
			return startHTTPServer(ctx, app, httpHost, httpPort)
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
		logging.Info("MCP message handling goroutine exiting")
	}()
}

func runQuery(ctx context.Context, app *app.App, queryType, outputFormat string) error {
	handler := api.NewQueryHandler(app)

	// Special case: if queryType is "json", read JSON-RPC requests from stdin
	if queryType == "json" {
		return handleJSONRPCFromStdin(ctx, handler, outputFormat)
	}

	response := handler.HandleQueryType(ctx, queryType)

	if response.Error != nil {
		return fmt.Errorf("query error: %s", response.Error.Message)
	}

	// Format output
	if outputFormat == "json" {
		jsonBytes, err := json.Marshal(response.Result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		// For text output, pretty print
		jsonBytes, err := json.MarshalIndent(response.Result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(jsonBytes))
	}

	return nil
}

// hasStdinData checks if stdin has data available without blocking
func hasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is a pipe/file (has data) or if it's coming from terminal
	return (stat.Mode()&os.ModeCharDevice) == 0 && stat.Size() > 0
}

func handleJSONRPCFromStdin(ctx context.Context, handler *api.QueryHandler, outputFormat string) error {
	// Check if stdin has data before trying to read
	if !hasStdinData() {
		return fmt.Errorf(`no JSON-RPC input provided

Usage examples:
  echo '{"method": "sessions.list", "id": 1}' | %s --query json --output-format json
  echo '{"method": "sessions.create", "params": {"title": "New Session"}, "id": 1}' | %s --query json --output-format json
  
Available methods: sessions.list, sessions.create, sessions.select, sessions.delete, tools.list, mcp.list, commands.list`,
			os.Args[0], os.Args[0])
	}

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON-RPC request
		var request api.QueryRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			// Output error response
			errorResponse := &api.QueryResponse{
				Error: &api.QueryError{
					Code:    -32700,
					Message: "Parse error: " + err.Error(),
				},
				ID: nil,
			}
			outputJSONRPCResponse(errorResponse, outputFormat)
			continue
		}

		// Handle the request
		response := handler.Handle(ctx, &request)
		outputJSONRPCResponse(response, outputFormat)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %w", err)
	}

	return nil
}

func outputJSONRPCResponse(response *api.QueryResponse, outputFormat string) {
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		// Fallback error response
		fallbackResponse := &api.QueryResponse{
			Error: &api.QueryError{
				Code:    -32603,
				Message: "Internal error: " + err.Error(),
			},
			ID: response.ID,
		}
		jsonBytes, _ = json.Marshal(fallbackResponse)
	}

	fmt.Println(string(jsonBytes))
}

// SSE handler functions moved to internal/http/sse.go

func startHTTPServer(ctx context.Context, app *app.App, host string, port int) error {
	handler := api.NewQueryHandler(app)

	// Create dedicated HTTP mux
	mux := http.NewServeMux()

	// Add debug endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Mix HTTP JSON-RPC Server\nPath: %s\nMethod: %s\n", r.URL.Path, r.Method)
	})

	// Add SSE streaming endpoint
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		httphandlers.HandleSSEStream(ctx, handler, w, r)
	})

	// Add message queue endpoint for persistent SSE
	mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
		// Handle stream endpoints
		if strings.HasSuffix(r.URL.Path, "/message") {
			httphandlers.HandleMessageQueue(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Only accept POST requests
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			errorResponse := &api.QueryResponse{
				Error: &api.QueryError{
					Code:    -32700,
					Message: "Parse error: " + err.Error(),
				},
			}
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		// Parse JSON-RPC request
		var request api.QueryRequest
		if err := json.Unmarshal(body, &request); err != nil {
			errorResponse := &api.QueryResponse{
				Error: &api.QueryError{
					Code:    -32700,
					Message: "Parse error: " + err.Error(),
				},
			}
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		// Log the incoming request
		logging.Debug("HTTP Request: method=%s\n", request.Method)
		logging.Debug("HTTP Request Body: %s\n", string(body))

		// Handle the request
		response := handler.Handle(ctx, &request)

		// Log the response
		if responseJSON, err := json.Marshal(response); err == nil {
			logging.Debug("HTTP Response: %s\n", string(responseJSON))
		} else {
			logging.Debug("HTTP Response: failed to marshal response: %v\n", err)
		}

		// Send response
		json.NewEncoder(w).Encode(response)
	})

	addr := host + ":" + strconv.Itoa(port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  15 * time.Minute, // Prevent 60-second drops
	}

	// Immediate feedback to user
	logging.Info("Starting HTTP JSON-RPC server", "address", addr)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		logging.Info("Shutting down HTTP server")
		server.Shutdown(context.Background())
	}()

	// Start server and provide ready confirmation
	logging.Info("HTTP JSON-RPC server ready", "address", addr)
	logging.Info("Send JSON-RPC requests", "url", fmt.Sprintf("http://%s/rpc", addr))
	logging.Info("Press Ctrl+C to stop")

	// Start server and block (this will block until server shuts down)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %v", err)
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
	rootCmd.Flags().Int("http-port", 0, "Start HTTP JSON-RPC server on this port (0 = disabled)")
	rootCmd.Flags().String("http-host", "localhost", "HTTP server host")

	// Permission flags
	rootCmd.Flags().Bool("dangerously-skip-permissions", false, "Skip all permission prompts (DANGEROUS - use only in trusted environments)")

	// Register custom validation for the format flag
	rootCmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return format.SupportedFormats, cobra.ShellCompDirectiveNoFileComp
	})

	// Add auth subcommand
	rootCmd.AddCommand(authCmd)
}

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"mix/internal/app"
	"mix/internal/auth"
	"mix/internal/config"
	"mix/internal/logging"
	"mix/internal/version"
)

// StartServer starts the HTTP REST API server with all configured routes
func StartServer(ctx context.Context, a *app.App, host string, port int) error {
	// Initialize tunnel registry if not already set
	if a.TunnelRegistry == nil {
		a.TunnelRegistry = NewTunnelRegistry()
	}
	tunnelRegistry := a.TunnelRegistry.(*TunnelRegistry)

	// Create REST handlers
	sessionHandler := NewSessionHandler(a)
	messageHandler := NewMessageHandler(a)
	systemHandler := NewSystemHandler(a)
	preferencesHandler := NewPreferencesHandler(a)
	authHandler := NewAuthHandler(a)
	toolsHandler := NewToolsHandler(a)
	systemInfoHandler := NewSystemInfoHandler(a)

	// Create session-aware asset handler
	fileHandler := NewFileHandler(a)
	sessionAssetHandler := NewSessionAssetHandler(a, a.StorageConfig)

	// Initialize and start the OAuth token refresh service
	credentialsService := config.GetAPICredentials()
	if credentialsService != nil {
		// Check every 30 minutes for tokens that need refresh (uses 35-minute expiry buffer)
		tokenRefreshService := auth.NewTokenRefreshService(credentialsService, 30*time.Minute)

		// Set the service in the auth handler so endpoints can use it
		authHandler.SetTokenRefreshService(tokenRefreshService)

		// Start the background refresh service
		go tokenRefreshService.Start(ctx)
	} else {
		logging.Warn("Credentials service not available - OAuth token refresh service disabled")
	}

	// Create dedicated HTTP mux with CORS middleware
	mux := http.NewServeMux()

	// CORS middleware wrapper
	corsMiddleware := func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers for all requests
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Continue with the actual handler
			handler.ServeHTTP(w, r)
		})
	}

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return detailed health information
		health := map[string]interface{}{
			"status":      "ok",
			"timestamp":   time.Now().Format(time.RFC3339),
			"version":     version.Version,
			"environment": os.Getenv("ENV"),
			"services": map[string]string{
				"backend":  "healthy",
				"database": "connected",
			},
		}
		if err := json.NewEncoder(w).Encode(health); err != nil {
			logging.Error("Failed to encode health response", "error", err)
		}
	})

	// Add documentation endpoint
	mux.HandleFunc("GET /doc", HandleDocumentation)

	// Add SSE streaming endpoint
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		HandleSSEStream(ctx, a, w, r)
	})

	// WebSocket tunnel endpoint for browser connections
	mux.HandleFunc("GET /api/v1/tunnel/cdp/session/{sessionId}", tunnelRegistry.HandleTunnelConnection)

	// Active tunnels endpoint
	mux.HandleFunc("GET /api/v1/tunnel/active", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		activeTunnels := tunnelRegistry.GetActiveTunnels()
		response := map[string]interface{}{
			"active_tunnels": activeTunnels,
			"count":          len(activeTunnels),
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logging.Error("Failed to encode active tunnels response", "error", err)
		}
	})

	// Test command endpoint for E2E testing
	mux.HandleFunc("POST /api/v1/tunnel/test-command/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.PathValue("sessionId")
		if sessionID == "" {
			http.Error(w, "Missing sessionId", http.StatusBadRequest)
			return
		}

		// Parse request body
		var requestBody struct {
			ID     interface{} `json:"id"`     // Allow client to specify ID
			Method string      `json:"method"`
			Params interface{} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// Use provided ID or generate a simple one
		commandID := requestBody.ID
		if commandID == nil {
			commandID = int(time.Now().Unix()) // Use Unix timestamp (smaller number)
		}

		// Create CDP command
		command := CDPRequest{
			ID:     commandID,
			Method: requestBody.Method,
			Params: requestBody.Params,
		}

		// Send command and wait for response
		response, err := tunnelRegistry.SendCommandToTunnel(sessionID, command)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to send command: %v", err), http.StatusInternalServerError)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logging.Error("Failed to encode test command response", "error", err)
		}
	})

	// Add URL video export endpoint (new Playwright-based export)

	// NEW SESSION-BASED FILE ENDPOINTS (replaces /input/ and /output/)

	// Session file management endpoints
	mux.HandleFunc("POST /api/sessions/{id}/files/upload", fileHandler.HandleUploadFile)
	mux.HandleFunc("GET /api/sessions/{id}/files", fileHandler.HandleListFiles)
	mux.HandleFunc("GET /api/sessions/{id}/files/{filename}", sessionAssetHandler.HandleServeFile)
	mux.HandleFunc("DELETE /api/sessions/{id}/files/{filename}", fileHandler.HandleDeleteFile)

	// REST API Endpoints

	// Session endpoints
	mux.HandleFunc("GET /api/sessions", sessionHandler.HandleListSessions)
	mux.HandleFunc("GET /api/sessions/{id}", sessionHandler.HandleGetSession)
	mux.HandleFunc("POST /api/sessions", sessionHandler.HandleCreateSession)
	mux.HandleFunc("PATCH /api/sessions/{id}/callbacks", sessionHandler.HandleUpdateSessionCallbacks)
	mux.HandleFunc("POST /api/sessions/{id}/rewind", sessionHandler.HandleRewindSession)
	mux.HandleFunc("DELETE /api/sessions/{id}", sessionHandler.HandleDeleteSession)

	// Message endpoints
	mux.HandleFunc("POST /api/sessions/{id}/messages", messageHandler.HandleSendMessage)
	mux.HandleFunc("GET /api/sessions/{id}/messages", messageHandler.HandleListSessionMessages)
	mux.HandleFunc("GET /api/sessions/{id}/export", messageHandler.HandleExportSession)
	mux.HandleFunc("GET /api/messages/history", messageHandler.HandleMessageHistory)
	mux.HandleFunc("POST /api/sessions/{id}/cancel", messageHandler.HandleCancelAgent)

	// System endpoints
	mux.HandleFunc("GET /api/mcp", systemHandler.HandleListMCPServers)
	mux.HandleFunc("GET /api/commands", systemHandler.HandleListCommands)
	mux.HandleFunc("GET /api/commands/{name}", systemHandler.HandleGetCommand)
	mux.HandleFunc("POST /api/permissions/{id}/grant", systemHandler.HandleGrantPermission)
	mux.HandleFunc("POST /api/permissions/{id}/deny", systemHandler.HandleDenyPermission)
	mux.HandleFunc("POST /api/notifications/{id}/respond", systemHandler.HandleNotificationRespond)
	mux.HandleFunc("GET /api/system/info", systemInfoHandler.HandleGetSystemInfo)

	// User preferences endpoints
	mux.HandleFunc("GET /api/preferences", preferencesHandler.HandleGetPreferences)
	mux.HandleFunc("POST /api/preferences", preferencesHandler.HandleUpdatePreferences)
	mux.HandleFunc("GET /api/preferences/providers", preferencesHandler.HandleGetAvailableProviders)
	mux.HandleFunc("POST /api/preferences/reset", preferencesHandler.HandleResetPreferences)

	// Authentication management endpoints
	mux.HandleFunc("POST /api/auth/api-key", authHandler.HandleStoreAPIKey)
	mux.HandleFunc("DELETE /api/auth/{provider}", authHandler.HandleDeleteCredentials)
	mux.HandleFunc("GET /api/auth/status", authHandler.HandleAuthStatus)
	mux.HandleFunc("GET /api/auth/validate", authHandler.HandleValidatePreferredProvider)
	mux.HandleFunc("POST /api/auth/oauth/{provider}", authHandler.HandleStartOAuth)
	mux.HandleFunc("POST /api/auth/oauth-callback", authHandler.HandleOAuthCallback)

	// OAuth token management endpoints
	mux.HandleFunc("POST /internal/auth/refresh-tokens", authHandler.HandleRefreshTokens)
	mux.HandleFunc("GET /health/auth", authHandler.HandleOAuthHealth)

	// Tools management endpoints
	// mux.HandleFunc("POST /api/tools/credentials", toolsHandler.HandleStoreToolAPIKey)
	// mux.HandleFunc("DELETE /api/tools/credentials/{tool_type}/{provider}", toolsHandler.HandleDeleteToolCredential)
	mux.HandleFunc("GET /api/tools/credentials-status", toolsHandler.HandleToolCredentialsStatus)
	mux.HandleFunc("GET /api/tools", toolsHandler.HandleListLLMTools)

	addr := host + ":" + strconv.Itoa(port)
	server := &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux), // Apply CORS middleware to all routes
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  15 * time.Minute, // Prevent 60-second drops
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		logging.Info("Shutting down HTTP server")
		_ = server.Shutdown(context.Background())
	}()

	// Start server and block (this will block until server shuts down)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}

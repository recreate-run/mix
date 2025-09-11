package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mix/internal/app"
	"mix/internal/logging"
	"mix/internal/version"
)

// StartServer starts the HTTP REST API server with all configured routes
func StartServer(ctx context.Context, app *app.App, host string, port int) error {
	// Create REST handlers
	sessionHandler := NewSessionHandler(app)
	messageHandler := NewMessageHandler(app)
	systemHandler := NewSystemHandler(app)
	preferencesHandler := NewPreferencesHandler(app)
	authHandler := NewAuthHandler(app)

	// Create dedicated HTTP mux with CORS middleware
	mux := http.NewServeMux()
	
	// CORS middleware wrapper
	corsMiddleware := func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers for all requests
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			// Continue with the actual handler
			handler.ServeHTTP(w, r)
		})
	}

	// Add debug endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Mix HTTP REST API Server\\nPath: %s\\nMethod: %s\\n", r.URL.Path, r.Method)
	})
	
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
		json.NewEncoder(w).Encode(health)
	})

	// Add documentation endpoint  
	mux.HandleFunc("GET /doc", HandleDocumentation)

	// Add SSE streaming endpoint
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		HandleSSEStream(ctx, app, w, r)
	})

	// Add message queue endpoint for persistent SSE
	mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
		// Handle stream endpoints
		if strings.HasSuffix(r.URL.Path, "/message") {
			HandleMessageQueue(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// Add URL video export endpoint (new Playwright-based export)
	mux.HandleFunc("/api/video/export-url", HandleURLVideoExport)

	// Add file types endpoint
	mux.HandleFunc("/api/file-types", HandleFileTypes(app))

	// Add asset serving endpoints for media files
	mux.HandleFunc("/input/", HandleInputAssets(app))
	mux.HandleFunc("/output/", HandleOutputAssets(app))
	
	// Add GSAP animation endpoints with clean, explicit routing
	mux.HandleFunc("/api/gsap_animations/", func(w http.ResponseWriter, r *http.Request) {
		animationName := strings.TrimPrefix(r.URL.Path, "/api/gsap_animations/")
		if animationName == "" {
			app.AssetServer.ServeGSAPAnimationsList(w, r)
		} else {
			app.AssetServer.ServeGSAPAnimationSchema(w, r, animationName)
		}
	})
	mux.HandleFunc("/api/gsap_animations", func(w http.ResponseWriter, r *http.Request) {
		app.AssetServer.ServeGSAPAnimationsList(w, r)
	})
	mux.HandleFunc("/gsap_animations/", func(w http.ResponseWriter, r *http.Request) {
		app.AssetServer.ServeGSAPAnimationFiles(w, r)
	})

	// REST API Endpoints
	
	// Session endpoints
	mux.HandleFunc("GET /api/sessions", sessionHandler.HandleListSessions)
	mux.HandleFunc("GET /api/sessions/{id}", sessionHandler.HandleGetSession)
	mux.HandleFunc("POST /api/sessions", sessionHandler.HandleCreateSession)
	mux.HandleFunc("POST /api/sessions/{id}/fork", sessionHandler.HandleForkSession)
	mux.HandleFunc("DELETE /api/sessions/{id}", sessionHandler.HandleDeleteSession)
	
	// Message endpoints
	mux.HandleFunc("POST /api/sessions/{id}/messages", messageHandler.HandleSendMessage)
	mux.HandleFunc("GET /api/sessions/{id}/messages", messageHandler.HandleListSessionMessages)
	mux.HandleFunc("GET /api/messages/history", messageHandler.HandleMessageHistory)
	mux.HandleFunc("POST /api/sessions/{id}/cancel", messageHandler.HandleCancelAgent)
	
	// System endpoints
	mux.HandleFunc("POST /api/auth/login", systemHandler.HandleAuthLogin)
	mux.HandleFunc("POST /api/auth/apikey", systemHandler.HandleSetAPIKey)
	mux.HandleFunc("GET /api/mcp", systemHandler.HandleListMCPServers)
	mux.HandleFunc("GET /api/commands", systemHandler.HandleListCommands)
	mux.HandleFunc("GET /api/commands/{name}", systemHandler.HandleGetCommand)
	mux.HandleFunc("POST /api/permissions/{id}/grant", systemHandler.HandleGrantPermission)
	mux.HandleFunc("POST /api/permissions/{id}/deny", systemHandler.HandleDenyPermission)
	
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

	addr := host + ":" + strconv.Itoa(port)
	server := &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux), // Apply CORS middleware to all routes
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  15 * time.Minute, // Prevent 60-second drops
	}

	// Immediate feedback to user
	logging.Info("Starting HTTP REST API server", "address", addr)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		logging.Info("Shutting down HTTP server")
		server.Shutdown(context.Background())
	}()

	// Start server and block (this will block until server shuts down)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %v", err)
	}

	return nil
}
package http

import (
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
	"mix/internal/logging"
	"mix/internal/version"
)

// StartServer starts the HTTP JSON-RPC server with all configured routes
func StartServer(ctx context.Context, app *app.App, host string, port int) error {
	handler := api.NewQueryHandler(app)

	// Create dedicated HTTP mux
	mux := http.NewServeMux()

	// Add debug endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Mix HTTP JSON-RPC Server\\nPath: %s\\nMethod: %s\\n", r.URL.Path, r.Method)
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

	// Add SSE streaming endpoint
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		HandleSSEStream(ctx, handler, w, r)
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
		logging.Debug("HTTP Request: method=%s\\n", request.Method)
		logging.Debug("HTTP Request Body: %s\\n", string(body))

		// Handle the request
		response := handler.Handle(ctx, &request)

		// Log the response
		if responseJSON, err := json.Marshal(response); err == nil {
			logging.Debug("HTTP Response: %s\\n", string(responseJSON))
		} else {
			logging.Debug("HTTP Response: failed to marshal response: %v\\n", err)
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

	// Start server and block (this will block until server shuts down)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %v", err)
	}

	return nil
}
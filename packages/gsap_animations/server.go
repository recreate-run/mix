package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Animation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Directory   string `json:"directory"`
}

func main() {
	startServer(context.Background())
}

// startServer starts the GSAP server with graceful shutdown support
func startServer(ctx context.Context) error {
	// Initialize storage directory
	if err := InitializeStorage(); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	port := os.Getenv("GSAP_PORT")
	if port == "" {
		port = "8089"
	}

	mux := http.NewServeMux()

	// Enable CORS for all endpoints
	handler := corsMiddleware(mux)

	// API endpoints
	mux.HandleFunc("/animations", listAnimations)
	mux.HandleFunc("/animations/", handleAnimationRequest)
	mux.HandleFunc("/shared/", serveSharedAsset)
	mux.HandleFunc("/storage/", ServeStorageFiles)
	mux.HandleFunc("/export", handleExport)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	log.Printf("GSAP server starting on port %s", port)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
			errChan <- err
		}
	}()

	// Wait for either context cancellation or server error
	select {
	case <-ctx.Done():
		// Normal shutdown requested
	case err := <-errChan:
		// Server failed to start
		return err
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
		return err
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getAnimationsDir() string {
	// Get directory from environment variable or use current directory
	if dir := os.Getenv("GSAP_ANIMATIONS_DIR"); dir != "" {
		return dir
	}
	// Default to current directory where server.go is located
	dir, _ := os.Getwd()
	return dir
}

func listAnimations(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	animationsDir := getAnimationsDir()
	entries, err := os.ReadDir(animationsDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read animations directory: %v", err), http.StatusInternalServerError)
		return
	}

	var animations []Animation
	for _, entry := range entries {
		// Skip non-directories and special directories
		if !entry.IsDir() || entry.Name() == "shared" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Read schema.json
		schemaPath := filepath.Join(animationsDir, entry.Name(), "schema.json")
		schemaData, err := os.ReadFile(schemaPath)
		if err != nil {
			continue // Skip animations without schema
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaData, &schema); err != nil {
			continue // Skip invalid schemas
		}

		name, _ := schema["name"].(string)
		description, _ := schema["description"].(string)

		animations = append(animations, Animation{
			Name:        name,
			Description: description,
			Directory:   entry.Name(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(animations)
}

func handleAnimationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /animations/{name}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/animations/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	animationName := parts[0]
	action := parts[1]

	// Validate animation exists
	animationsDir := getAnimationsDir()
	animationPath := filepath.Join(animationsDir, animationName)

	if _, err := os.Stat(animationPath); os.IsNotExist(err) {
		http.Error(w, "Animation not found", http.StatusNotFound)
		return
	}

	switch action {
	case "schema":
		serveSchema(w, animationPath)
	case "preview":
		servePreview(w, animationPath)
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}

func serveSchema(w http.ResponseWriter, animationPath string) {
	schemaPath := filepath.Join(animationPath, "schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Schema not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to read schema: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func servePreview(w http.ResponseWriter, animationPath string) {
	indexPath := filepath.Join(animationPath, "index.html")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Preview not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to read preview: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Rewrite paths to use our server
	html := string(data)
	html = strings.ReplaceAll(html, `href="../shared/`, `href="/shared/`)
	html = strings.ReplaceAll(html, `src="../shared/`, `src="/shared/`)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func serveSharedAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get file path
	filePath := strings.TrimPrefix(r.URL.Path, "/shared/")
	if filePath == "" || strings.Contains(filePath, "..") {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Read and serve file
	animationsDir := getAnimationsDir()
	fullPath := filepath.Join(animationsDir, "shared", filePath)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	// Get file info for Content-Length and Last-Modified headers
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file info: %v", err), http.StatusInternalServerError)
		return
	}

	// Set content type based on extension
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(filePath, ".js"):
		contentType = "application/javascript"
	case strings.HasSuffix(filePath, ".css"):
		contentType = "text/css"
	case strings.HasSuffix(filePath, ".json"):
		contentType = "application/json"
	case strings.HasSuffix(filePath, ".html"):
		contentType = "text/html"
	case strings.HasSuffix(filePath, ".mp4"):
		contentType = "video/mp4"
	case strings.HasSuffix(filePath, ".webm"):
		contentType = "video/webm"
	case strings.HasSuffix(filePath, ".mov"):
		contentType = "video/quicktime"
	case strings.HasSuffix(filePath, ".avi"):
		contentType = "video/x-msvideo"
	}

	// Set content headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(filePath)))

	// Use http.ServeContent for better range support (needed for videos)
	http.ServeContent(w, r, filePath, fileInfo.ModTime(), file)
}

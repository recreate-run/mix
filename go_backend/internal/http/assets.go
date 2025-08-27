package http

import (
	"encoding/json"
	"net/http"

	"mix/internal/app"
)

// HandleInputAssets handles GET /input/*
// Serves input media files (images, videos, audio, text) through the AssetServer
func HandleInputAssets(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app.AssetServer.ServeHTTP(w, r)
	}
}

// HandleOutputAssets handles GET /output/*
// Serves output media files (generated content, exports) through the AssetServer
func HandleOutputAssets(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app.AssetServer.ServeHTTP(w, r)
	}
}

// HandleFileTypes handles GET /api/file-types
// Returns supported file types from the asset server
func HandleFileTypes(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get supported file types from asset server
		fileTypes := app.AssetServer.GetSupportedFileTypes()

		// Set JSON content type
		w.Header().Set("Content-Type", "application/json")

		// Marshal and send response
		jsonBytes, err := json.Marshal(fileTypes)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(jsonBytes)
	}
}
package http

import (
	"context"
	"encoding/json"
	"net/http"

	"mix/internal/api"
	"mix/internal/app"
	"mix/internal/video"
)

// VideoExportRequest represents the JSON request body for video export
type VideoExportRequest struct {
	Config     interface{} `json:"config"`
	OutputPath string      `json:"outputPath"`
	SessionID  string      `json:"sessionId,omitempty"`
}

// VideoExportResponse represents the JSON response for video export
type VideoExportResponse struct {
	OutputPath string `json:"outputPath"`
	Message    string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HandleVideoExport handles POST /api/video/export requests
func HandleVideoExport(ctx context.Context, handler *api.QueryHandler, w http.ResponseWriter, r *http.Request) {
	// Set JSON content type
	w.Header().Set("Content-Type", "application/json")
	
	// Only accept POST requests
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only POST method is allowed",
		})
		return
	}

	// Parse request body
	var req VideoExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid_json",
			Message: "Failed to parse JSON body: " + err.Error(),
		})
		return
	}

	// Validate required parameters
	if req.Config == nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "missing_config",
			Message: "Missing required parameter: config",
		})
		return
	}

	if req.OutputPath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "missing_output_path",
			Message: "Missing required parameter: outputPath",
		})
		return
	}

	// Convert config to JSON string
	configBytes, err := json.Marshal(req.Config)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "invalid_config",
			Message: "Invalid config format: " + err.Error(),
		})
		return
	}

	// Get app instance from handler
	appInstance := getAppFromHandler(handler)

	// Export the video using the video service
	exportResp, err := appInstance.Video.Export(video.ExportRequest{
		ConfigJSON: string(configBytes),
		OutputPath: req.OutputPath,
		SessionID:  req.SessionID,
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "export_failed",
			Message: "Video export failed: " + err.Error(),
		})
		return
	}

	// Return successful response
	response := VideoExportResponse{
		OutputPath: req.OutputPath,
		Message:    exportResp.Message,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Helper function to get app from handler
func getAppFromHandler(handler *api.QueryHandler) *app.App {
	return handler.GetApp()
}


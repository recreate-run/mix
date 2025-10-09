package http

import (
	"encoding/json"
	"net/http"

	"mix/internal/app"
)

type SystemInfoHandler struct {
	app *app.App
}

func NewSystemInfoHandler(app *app.App) *SystemInfoHandler {
	return &SystemInfoHandler{app: app}
}

type SystemInfoResponse struct {
	StorageBasePath string `json:"storageBasePath"`
}

// HandleGetSystemInfo returns system information including storage configuration
func (h *SystemInfoHandler) HandleGetSystemInfo(w http.ResponseWriter, r *http.Request) {
	response := SystemInfoResponse{
		StorageBasePath: h.app.StorageConfig.BasePath,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

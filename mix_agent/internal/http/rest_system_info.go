package http

import (
	"encoding/json"
	"net/http"

	"mix/internal/app"
	"mix/internal/constants"
)

type SystemInfoHandler struct {
	app *app.App
}

func NewSystemInfoHandler(a *app.App) *SystemInfoHandler {
	return &SystemInfoHandler{app: a}
}

type SystemInfoResponse struct {
	StorageBasePath string `json:"storageBasePath"`
}

// HandleGetSystemInfo returns system information including storage configuration
func (h *SystemInfoHandler) HandleGetSystemInfo(w http.ResponseWriter, r *http.Request) {
	response := SystemInfoResponse{
		StorageBasePath: h.app.StorageConfig.BasePath,
	}

	w.Header().Set("Content-Type", constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

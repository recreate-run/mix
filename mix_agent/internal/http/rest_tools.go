package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"mix/internal/app"
	"mix/internal/config"
	"mix/internal/constants"
	"mix/internal/llm/models"
	"mix/internal/logging"
	"mix/internal/tools"
)

// ToolsHandler handles REST endpoints for tool credential management
type ToolsHandler struct {
	app *app.App
}

// NewToolsHandler creates a new tools handler
func NewToolsHandler(a *app.App) *ToolsHandler {
	return &ToolsHandler{
		app: a,
	}
}

// StoreToolAPIKeyRequest represents the request body for storing a tool API key
type StoreToolAPIKeyRequest struct {
	ToolType string `json:"tool_type"`
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
}

// ToolAuthStatus represents authentication status for a tool
type ToolAuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	DisplayName   string `json:"display_name"`
	Description   string `json:"description"`
	APIKeyFormat  string `json:"api_key_format"`
	RequiresKey   bool   `json:"api_key_required"`
	Provider      string `json:"provider,omitempty"`
}

// ToolsStatusResponse represents the authentication status for all tools
type ToolsStatusResponse struct {
	Categories map[string]ToolCategoryStatus `json:"categories"`
}

// ToolsStatusResponseArrayFormat converts ToolsStatusResponse to an SDK-compatible array format
type ToolsStatusResponseArrayFormat struct {
	Categories map[string]ToolCategoryStatusArrayFormat `json:"categories"`
}

// ToolCategoryStatusArrayFormat represents the status of tools in a category with tools as array
type ToolCategoryStatusArrayFormat struct {
	DisplayName string           `json:"display_name"`
	Description string           `json:"description"`
	Icon        string           `json:"icon"`
	Tools       []ToolAuthStatus `json:"tools"`
}

// ToolCategoryStatus represents the status of tools in a category
type ToolCategoryStatus struct {
	DisplayName string                    `json:"display_name"`
	Description string                    `json:"description"`
	Icon        string                    `json:"icon"`
	Tools       map[string]ToolAuthStatus `json:"tools"`
}

// HandleStoreToolAPIKey handles POST /api/tools/credentials
func (h *ToolsHandler) HandleStoreToolAPIKey(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	var request StoreToolAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON request", "INVALID_JSON")
		return
	}

	// Validate request
	if request.ToolType == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Tool type is required", "MISSING_TOOL_TYPE")
		return
	}

	if request.Provider == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider is required", "MISSING_PROVIDER")
		return
	}

	if request.APIKey == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "API key is required", "MISSING_API_KEY")
		return
	}

	// Validate tool type and provider
	registry := tools.GetRegistry()
	toolType := tools.ToolType(request.ToolType)
	provider := tools.ToolProvider(request.Provider)

	tool, exists := registry.GetTool(toolType, provider)
	if !exists {
		WriteErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Tool %s/%s not supported", toolType, provider), "INVALID_TOOL")
		return
	}

	// Get API credentials service
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Credentials service not available", "CREDENTIALS_SERVICE_UNAVAILABLE")
		return
	}

	// Store the API key using simplified provider approach
	if err := credentialsService.StoreAPIKey(r.Context(), models.ModelProvider(string(provider)), request.APIKey); err != nil {
		// Track failed authentication attempt
		if h.app.Analytics != nil {
			_ = h.app.Analytics.TrackProviderAuth(r.Context(), fmt.Sprintf("%s_%s", toolType, provider), false, "api_key")
		}
		WriteErrorResponse(w, http.StatusBadRequest, err.Error(), "TOOL_API_KEY_STORAGE_FAILED")
		return
	}

	// Track successful authentication
	if h.app.Analytics != nil {
		_ = h.app.Analytics.TrackProviderAuth(r.Context(), fmt.Sprintf("%s_%s", toolType, provider), true, "api_key")
	}

	// Return success response
	response := StoreToolAPIKeyResponse{
		Status:   "success",
		ToolType: request.ToolType,
		Provider: request.Provider,
		Message:  fmt.Sprintf("%s API key stored successfully", tool.DisplayName),
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleDeleteToolCredential handles DELETE /api/tools/credentials/{tool_type}/{provider}
func (h *ToolsHandler) HandleDeleteToolCredential(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodDelete {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	// Extract tool type and provider from URL path
	pathParts := splitPath(r.URL.Path)
	if len(pathParts) < 5 || pathParts[0] != "api" || pathParts[1] != "tools" || pathParts[2] != "credentials" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid URL format", "INVALID_URL")
		return
	}

	toolTypeStr := pathParts[3]
	if len(pathParts) < 5 {
		WriteErrorResponse(w, http.StatusBadRequest, "Provider is required", "MISSING_PROVIDER")
		return
	}
	providerStr := pathParts[4]

	toolType := tools.ToolType(toolTypeStr)
	provider := tools.ToolProvider(providerStr)

	// Validate tool exists
	registry := tools.GetRegistry()
	_, exists := registry.GetTool(toolType, provider)
	if !exists {
		WriteErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Tool %s/%s not supported", toolType, provider), "INVALID_TOOL")
		return
	}

	// Get API credentials service
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Credentials service not available", "CREDENTIALS_SERVICE_UNAVAILABLE")
		return
	}

	// Delete the credential using simplified provider approach
	if err := credentialsService.DeleteAPIKey(r.Context(), models.ModelProvider(string(provider))); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to delete tool credential", "TOOL_CREDENTIAL_DELETE_FAILED")
		return
	}

	// Return success response
	response := DeleteToolCredentialResponse{
		Status:  "success",
		Message: "Tool credential deleted successfully",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// HandleToolCredentialsStatus handles GET /api/tools/credentials-status
// Returns authentication/credential status for external tool integrations (Brave Search, Gemini, etc.)
func (h *ToolsHandler) HandleToolCredentialsStatus(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	status := h.checkAllToolsStatus(r.Context())

	// Convert to array format for SDK compatibility
	arrayFormat := h.convertToArrayFormat(status)
	WriteJSONResponse(w, http.StatusOK, arrayFormat)
}

// LLMToolInfo represents information about an LLM tool
type LLMToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Required    []string       `json:"required"`
}

// LLMToolsListResponse represents the list of all LLM tools available
type LLMToolsListResponse struct {
	Tools []LLMToolInfo `json:"tools"`
}

// HandleListLLMTools handles GET /api/tools
// Returns the list of all LLM tools that Claude can invoke (Bash, Edit, Read, Write, etc.)
// This list is dynamically extracted from the actual tools available to the agent.
func (h *ToolsHandler) HandleListLLMTools(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, constants.MethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	// Get tools dynamically from the agent
	agentTools := h.app.CoderAgent.GetTools()

	// Convert agent tools to API response format
	llmTools := make([]LLMToolInfo, 0, len(agentTools))
	for _, tool := range agentTools {
		info := tool.Info()
		llmTools = append(llmTools, LLMToolInfo{
			Name:        info.Name,
			Description: info.Description,
			Parameters:  info.Parameters,
			Required:    info.Required,
		})
	}

	response := LLMToolsListResponse{
		Tools: llmTools,
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// convertToArrayFormat converts the map-based tools structure to an array-based format for SDK compatibility
func (h *ToolsHandler) convertToArrayFormat(status ToolsStatusResponse) ToolsStatusResponseArrayFormat {
	arrayFormat := ToolsStatusResponseArrayFormat{
		Categories: make(map[string]ToolCategoryStatusArrayFormat),
	}

	for categoryID, category := range status.Categories {
		arrayCategory := ToolCategoryStatusArrayFormat{
			DisplayName: category.DisplayName,
			Description: category.Description,
			Icon:        category.Icon,
			Tools:       make([]ToolAuthStatus, 0, len(category.Tools)),
		}

		// Convert map of tools to array of tools
		for providerID, tool := range category.Tools {
			// Add provider ID to the tool object
			tool.Provider = providerID
			arrayCategory.Tools = append(arrayCategory.Tools, tool)
		}

		arrayFormat.Categories[categoryID] = arrayCategory
	}

	return arrayFormat
}

// checkAllToolsStatus checks authentication status for all available tools
func (h *ToolsHandler) checkAllToolsStatus(ctx context.Context) ToolsStatusResponse {
	status := ToolsStatusResponse{
		Categories: make(map[string]ToolCategoryStatus),
	}

	// Get credentials service
	credentialsService := config.GetAPICredentials()
	registry := tools.GetRegistry()

	// Get all categories
	categories := registry.GetAllCategories()

	for _, category := range categories {
		categoryStatus := ToolCategoryStatus{
			DisplayName: category.DisplayName,
			Description: category.Description,
			Icon:        category.Icon,
			Tools:       make(map[string]ToolAuthStatus),
		}

		// Check each tool in the category
		for _, tool := range category.Tools {
			toolStatus := ToolAuthStatus{
				DisplayName:   tool.DisplayName,
				Description:   tool.Description,
				APIKeyFormat:  tool.APIKeyFormat,
				RequiresKey:   tool.RequiresKey,
				Authenticated: false,
			}

			// Check if tool has API key (if required)
			if tool.RequiresKey && credentialsService != nil {
				hasKey, err := credentialsService.HasAPIKey(ctx, models.ModelProvider(string(tool.Provider)))
				if err != nil {
					logging.Error("Failed to check tool API key", "tool", tool.Provider, "error", err)
				} else {
					toolStatus.Authenticated = hasKey
				}
			} else if !tool.RequiresKey {
				// Tool doesn't require API key, so it's always "authenticated"
				toolStatus.Authenticated = true
			}

			categoryStatus.Tools[string(tool.Provider)] = toolStatus
		}

		status.Categories[string(category.Type)] = categoryStatus
	}

	return status
}

// splitPath splits URL path into components, removing empty strings
func splitPath(path string) []string {
	parts := make([]string, 0)
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

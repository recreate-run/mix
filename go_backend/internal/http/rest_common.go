package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"mix/internal/logging"
)

// REST Response wrapper for consistent JSON responses
type RESTResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Error   *RESTError  `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// REST Error structure
type RESTError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Standard error types
const (
	ErrorTypeBadRequest     = "bad_request"
	ErrorTypeNotFound       = "not_found"
	ErrorTypeInternalError  = "internal_error"
	ErrorTypeUnauthorized   = "unauthorized"
	ErrorTypeValidation     = "validation_error"
)

// HTTP status code mapping for different error types
var errorStatusMap = map[string]int{
	ErrorTypeBadRequest:    http.StatusBadRequest,
	ErrorTypeNotFound:      http.StatusNotFound,
	ErrorTypeInternalError: http.StatusInternalServerError,
	ErrorTypeUnauthorized:  http.StatusUnauthorized,
	ErrorTypeValidation:    http.StatusBadRequest,
}

// sendJSONResponse sends a standardized JSON response
func sendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	response := RESTResponse{Data: data}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error("Failed to encode JSON response", "error", err)
	}
}

// sendErrorResponse sends a standardized error response
func sendErrorResponse(w http.ResponseWriter, errorType string, message string) {
	status := errorStatusMap[errorType]
	if status == 0 {
		status = http.StatusInternalServerError
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	response := RESTResponse{
		Error: &RESTError{
			Code:    status,
			Message: message,
			Type:    errorType,
		},
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error("Failed to encode error response", "error", err)
	}
}

// sendValidationError sends a validation error with 400 status
func sendValidationError(w http.ResponseWriter, field string, message string) {
	fullMessage := fmt.Sprintf("Validation error for field '%s': %s", field, message)
	sendErrorResponse(w, ErrorTypeValidation, fullMessage)
}

// sendNotFoundError sends a 404 error
func sendNotFoundError(w http.ResponseWriter, resource string, id string) {
	message := fmt.Sprintf("%s with ID '%s' not found", resource, id)
	sendErrorResponse(w, ErrorTypeNotFound, message)
}

// sendInternalError sends a 500 error
func sendInternalError(w http.ResponseWriter, operation string, err error) {
	message := fmt.Sprintf("Internal error during %s: %s", operation, err.Error())
	sendErrorResponse(w, ErrorTypeInternalError, message)
	logging.Error("Internal error", "operation", operation, "error", err)
}

// sendUnauthorizedError sends a 401 error
func sendUnauthorizedError(w http.ResponseWriter, message string) {
	sendErrorResponse(w, ErrorTypeUnauthorized, message)
}

// parseIntParam extracts and validates integer path parameters
func parseIntParam(param string, paramName string) (int64, error) {
	if param == "" {
		return 0, fmt.Errorf("%s is required", paramName)
	}
	
	value, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", paramName)
	}
	
	return value, nil
}

// parseJSONBody parses JSON request body into the provided struct
func parseJSONBody(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict parsing
	
	return decoder.Decode(target)
}

// setCORSHeaders sets CORS headers for REST endpoints
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// handleCORSPreflight handles OPTIONS requests for CORS
func handleCORSPreflight(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == "OPTIONS" {
		setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}
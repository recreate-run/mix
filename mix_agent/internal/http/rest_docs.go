package http

import (
	"encoding/json"
	"net/http"
)

// HandleDocumentation serves OpenAPI 3.1 specification as JSON
func HandleDocumentation(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	serveOpenAPISpec(w, r)
}

// serveOpenAPISpec serves the OpenAPI 3.1 specification as JSON
func serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := getOpenAPISpec()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(spec); err != nil {
		sendInternalError(w, "generating OpenAPI spec", err)
	}
}

// OpenAPI 3.1 specification structures with proper field ordering
type OpenAPISpec struct {
	OpenAPI           string                 `json:"openapi"`
	Info              OpenAPIInfo            `json:"info"`
	Servers           []OpenAPIServer        `json:"servers"`
	XSpeakeasyRetries map[string]interface{} `json:"x-speakeasy-retries"`
	Paths             map[string]interface{} `json:"paths"`
	Components        OpenAPIComponents      `json:"components"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type OpenAPIComponents struct {
	Schemas map[string]interface{} `json:"schemas"`
}

// getOpenAPISpec returns the complete OpenAPI 3.1 specification with proper field ordering
func getOpenAPISpec() OpenAPISpec {
	return OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       "Mix REST API",
			Description: "REST API for the Mix application - session management, messaging, and system operations",
			Version:     "1.0.0",
		},
		Servers: []OpenAPIServer{
			{
				URL:         "http://localhost:8088",
				Description: "Development server",
			},
		},
		XSpeakeasyRetries: map[string]interface{}{
			"strategy": "backoff",
			"backoff": map[string]interface{}{
				"initialInterval": 500,    // 500ms
				"maxInterval":     60000,  // 60 seconds  
				"maxElapsedTime":  600000, // 10 minutes (shorter for dev environment)
				"exponent":        1.5,    // exponential backoff
			},
			"statusCodes": []string{
				"5XX", // All server errors
				"408", // Request timeout
				"429", // Too many requests
			},
			"retryConnectionErrors": true,
		},
		Paths: map[string]interface{}{
			// Session Management Endpoints
			"/api/sessions": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "listSessions",
					"summary":     "List all sessions",
					"description": "Retrieve a list of all available sessions with their metadata",
					"tags":        []string{"Sessions"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("array", getSessionDataSchema(), "List of sessions"),
						"401": createErrorResponse("Unauthorized - authentication required"),
						"500": createErrorResponse("Internal server error"),
					},
				},
				"post": map[string]interface{}{
					"operationId":  "createSession",
					"summary":     "Create a new session",
					"description": "Create a new session with required title and optional custom system prompt. Session automatically gets isolated storage directory.",
					"tags":        []string{"Sessions"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"title"},
						"properties": map[string]interface{}{
							"title": map[string]interface{}{
								"type":        "string",
								"description": "Title for the session",
							},
							"customSystemPrompt": map[string]interface{}{
								"type":        "string",
								"description": "Custom system prompt content. Size limits apply based on promptMode: 100KB (102,400 bytes) for replace mode, 50KB (51,200 bytes) for append mode. Ignored in default mode. Supports environment variable substitution with $<variable> syntax.",
								"maxLength":   102400,
								"example":     "You are a helpful assistant specialized in $<domain>. Always be concise and accurate.",
							},
							"promptMode": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"default", "append", "replace"},
								"default":     "default",
								"description": "Custom prompt handling mode:\n- 'default': Use base system prompt only (customSystemPrompt ignored)\n- 'append': Append customSystemPrompt to base system prompt (50KB limit)\n- 'replace': Replace base system prompt with customSystemPrompt (100KB limit)",
								"example":     "append",
							},
						},
					}),
					"responses": map[string]interface{}{
						"201": createSuccessResponse("object", getSessionDataSchema(), "Created session"),
						"400": map[string]interface{}{
							"description": "Invalid request data",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ErrorResponse",
									},
									"examples": map[string]interface{}{
										"missing_title": map[string]interface{}{
											"summary": "Missing required title",
											"value": map[string]interface{}{
												"error": map[string]interface{}{
													"code":    400,
													"message": "title is required",
													"type":    "validation_error",
												},
											},
										},
										"invalid_prompt_mode": map[string]interface{}{
											"summary": "Invalid prompt mode",
											"value": map[string]interface{}{
												"error": map[string]interface{}{
													"code":    400,
													"message": "promptMode must be 'default', 'append', or 'replace'",
													"type":    "validation_error",
												},
											},
										},
										"prompt_size_exceeded_replace": map[string]interface{}{
											"summary": "Custom prompt size exceeds replace mode limit",
											"value": map[string]interface{}{
												"error": map[string]interface{}{
													"code":    400,
													"message": "Custom prompt size (150KB) exceeds replace mode limit of 100KB",
													"type":    "validation_error",
												},
											},
										},
										"prompt_size_exceeded_append": map[string]interface{}{
											"summary": "Custom prompt size exceeds append mode limit",
											"value": map[string]interface{}{
												"error": map[string]interface{}{
													"code":    400,
													"message": "Custom prompt size (75KB) exceeds append mode limit of 50KB",
													"type":    "validation_error",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/api/sessions/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "getSession",
					"summary":     "Get a specific session",
					"description": "Retrieve detailed information about a specific session",
					"tags":        []string{"Sessions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", getSessionDataSchema(), "Session details"),
						"404": createErrorResponse("Session not found"),
					},
				},
				"delete": map[string]interface{}{
					"operationId":  "deleteSession",
					"summary":     "Delete a session",
					"description": "Permanently delete a session and all its data",
					"tags":        []string{"Sessions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
					},
					"responses": map[string]interface{}{
						"204": map[string]interface{}{
							"description": "Session deleted successfully",
						},
						"404": createErrorResponse("Session not found"),
					},
				},
			},
			"/api/sessions/{id}/fork": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "forkSession",
					"summary":     "Fork a session",
					"description": "Create a new session based on an existing session, copying messages up to a specified index",
					"tags":        []string{"Sessions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Source session ID to fork from"),
					},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"messageIndex"},
						"properties": map[string]interface{}{
							"messageIndex": map[string]interface{}{
								"type":        "integer",
								"minimum":     0,
								"description": "Index of the last message to include in the fork (0-based)",
							},
							"title": map[string]interface{}{
								"type":        "string",
								"description": "Optional title for the forked session (defaults to 'Forked Session')",
							},
						},
					}),
					"responses": map[string]interface{}{
						"201": createSuccessResponse("object", getSessionDataSchema(), "Forked session"),
						"400": createErrorResponse("Invalid request - messageIndex must be >= 0"),
						"404": createErrorResponse("Source session not found"),
					},
				},
			},
			"/api/sessions/{id}/rewind": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "rewindSession",
					"summary":     "Rewind a session",
					"description": "Delete messages after a specified message in the current session, optionally cleaning up media files created after that point",
					"tags":        []string{"Sessions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID to rewind"),
					},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"messageId"},
						"properties": map[string]interface{}{
							"messageId": map[string]interface{}{
								"type":        "string",
								"description": "ID of the last message to keep. All messages after this message will be deleted.",
							},
							"cleanupMedia": map[string]interface{}{
								"type":        "boolean",
								"default":     true,
								"description": "Whether to clean up media files created after the rewind point (based on file timestamp)",
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", getSessionDataSchema(), "Session rewound successfully"),
						"400": createErrorResponse("Invalid request - messageId is required"),
						"404": createErrorResponse("Session or message not found"),
					},
				},
			},
			"/api/sessions/{id}/export": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "exportSession",
					"summary":     "Export session transcript",
					"description": "Export complete session transcript with all messages, tool calls, reasoning, and metadata as JSON",
					"tags":        []string{"Sessions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID to export"),
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Session transcript exported successfully",
							"headers": map[string]interface{}{
								"Content-Disposition": map[string]interface{}{
									"description": "Suggests filename for download",
									"schema": map[string]interface{}{
										"type":    "string",
										"example": "attachment; filename=session_abc123_transcript.json",
									},
								},
							},
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ExportSession",
									},
								},
							},
						},
						"404": createErrorResponse("Session not found"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			// Message Operations
			"/api/sessions/{id}/messages": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "sendMessage",
					"summary":     "Send a message to session",
					"description": "Send a user message to a specific session for AI processing",
					"tags":        []string{"Messages"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
					},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"text"},
						"properties": map[string]interface{}{
							"text": map[string]interface{}{
								"type":        "string",
								"description": "The text content of the message",
							},
							"plan_mode": map[string]interface{}{
								"type":        "boolean",
								"description": "Whether the message is in planning mode",
								"default":     false,
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", getBackendMessageSchema(), "Message sent and processed"),
						"400": createErrorResponse("Invalid message data"),
						"404": createErrorResponse("Session not found"),
					},
				},
				"get": map[string]interface{}{
					"operationId":  "getSessionMessages",
					"summary":     "List session messages",
					"description": "Retrieve all messages from a specific session",
					"tags":        []string{"Messages"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("array", getBackendMessageSchema(), "List of session messages"),
						"404": createErrorResponse("Session not found"),
					},
				},
			},
			"/api/sessions/{id}/cancel": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "cancelSessionProcessing",
					"summary":     "Cancel agent processing",
					"description": "Cancel any ongoing agent processing in the specified session",
					"tags":        []string{"Messages"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"cancelled": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether cancellation was successful",
								},
							},
						}, "Cancellation status"),
						"404": createErrorResponse("Session not found"),
					},
				},
			},
			"/api/messages/history": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "getMessageHistory",
					"summary":     "Get global message history",
					"description": "Retrieve message history across all sessions with optional pagination",
					"tags":        []string{"Messages"},
					"parameters": []map[string]interface{}{
						{
							"name":        "limit",
							"in":          "query",
							"description": "Maximum number of messages to return",
							"schema": map[string]interface{}{
								"type":    "integer",
								"default": 50,
								"minimum": 1,
								"maximum": 1000,
							},
						},
						{
							"name":        "offset",
							"in":          "query",
							"description": "Number of messages to skip",
							"schema": map[string]interface{}{
								"type":    "integer",
								"default": 0,
								"minimum": 0,
							},
						},
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("array", getBackendMessageSchema(), "Message history"),
						"401": createErrorResponse("Unauthorized - authentication required"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			// System Operations
			"/api/mcp": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "listMcpServers",
					"summary":     "List MCP servers",
					"description": "Retrieve list of available Model Context Protocol (MCP) servers",
					"tags":        []string{"System"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("array", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "MCP server name",
								},
								"connected": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether the MCP server is currently connected",
								},
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Server connection status (e.g., 'connected', 'failed', 'disconnected')",
								},
								"tools": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"name": map[string]interface{}{
												"type":        "string",
												"description": "Tool name",
											},
											"description": map[string]interface{}{
												"type":        "string",
												"description": "Tool description",
											},
										},
										"required": []string{"name", "description"},
									},
									"description": "List of tools provided by this MCP server (null if server is not connected)",
									"nullable":    true,
								},
							},
							"required": []string{"name", "connected", "status"},
						}, "List of MCP servers"),
						"401": createErrorResponse("Unauthorized - authentication required"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/commands": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "listCommands",
					"summary":     "List available commands",
					"description": "Retrieve list of all available commands",
					"tags":        []string{"System"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("array", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Command name",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "Command description",
								},
							},
						}, "List of commands"),
						"401": createErrorResponse("Unauthorized - authentication required"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/commands/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "getCommand",
					"summary":     "Get specific command",
					"description": "Retrieve details about a specific command",
					"tags":        []string{"System"},
					"parameters": []map[string]interface{}{
						createPathParameter("name", "Command name"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Command name",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "Command description",
								},
								"usage": map[string]interface{}{
									"type":        "string",
									"description": "Command usage instructions",
								},
							},
						}, "Command details"),
						"404": createErrorResponse("Command not found"),
					},
				},
			},
			"/api/permissions/{id}/grant": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "grantPermission",
					"summary":     "Grant permission",
					"description": "Grant a specific permission",
					"tags":        []string{"Permissions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Permission ID"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"granted": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether permission was granted",
								},
							},
						}, "Permission grant status"),
						"401": createErrorResponse("Unauthorized - authentication required"),
						"404": createErrorResponse("Permission not found"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/permissions/{id}/deny": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "denyPermission",
					"summary":     "Deny permission",
					"description": "Deny a specific permission",
					"tags":        []string{"Permissions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Permission ID"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"denied": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether permission was denied",
								},
							},
						}, "Permission deny status"),
						"401": createErrorResponse("Unauthorized - authentication required"),
						"404": createErrorResponse("Permission not found"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/auth/api-key": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "storeApiKey",
					"summary":     "Store API key",
					"description": "Store API key for direct authentication with a specific provider",
					"tags":        []string{"Authentication"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"provider", "api_key"},
						"properties": map[string]interface{}{
							"provider": map[string]interface{}{
								"type":        "string",
								"description": "Provider name (anthropic, openai, openrouter, gemini, brave)",
								"enum":        []string{"anthropic", "openai", "openrouter", "gemini", "brave"},
							},
							"api_key": map[string]interface{}{
								"type":        "string",
								"description": "API key for authentication",
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Operation status (success)",
								},
								"provider": map[string]interface{}{
									"type":        "string",
									"description": "Provider name",
								},
								"message": map[string]interface{}{
									"type":        "string",
									"description": "Success message",
								},
							},
						}, "API key stored status"),
						"400": createErrorResponse("Invalid request data or API key format"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/auth/status": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "getAuthStatus",
					"summary":     "Get authentication status",
					"description": "Get authentication status for all supported providers",
					"tags":        []string{"Authentication"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"providers": map[string]interface{}{
									"type": "object",
									"additionalProperties": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"authenticated": map[string]interface{}{
												"type":        "boolean",
												"description": "Whether provider is authenticated",
											},
											"auth_method": map[string]interface{}{
												"type":        "string",
												"description": "Authentication method (oauth, api_key, none)",
												"enum":        []string{"oauth", "api_key", "none"},
											},
											"display_name": map[string]interface{}{
												"type":        "string",
												"description": "User-friendly provider name",
											},
										},
									},
									"description": "Map of provider authentication status",
								},
							},
						}, "Authentication status for all providers"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/auth/validate": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "validatePreferredProvider",
					"summary":     "Validate preferred provider",
					"description": "Check if the user's preferred provider is authenticated",
					"tags":        []string{"Authentication"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"valid": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether preferred provider is authenticated",
								},
								"provider": map[string]interface{}{
									"type":        "string",
									"description": "Preferred provider name",
								},
								"auth_method": map[string]interface{}{
									"type":        "string",
									"description": "Authentication method used",
									"enum":        []string{"oauth", "api_key", "none"},
								},
								"message": map[string]interface{}{
									"type":        "string",
									"description": "Status message",
								},
							},
						}, "Preferred provider validation status"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/auth/oauth/{provider}": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "startOAuthFlow",
					"summary":     "Start OAuth authentication",
					"description": "Initiate OAuth authentication flow for a specific provider",
					"tags":        []string{"Authentication"},
					"parameters": []map[string]interface{}{
						createPathParameter("provider", "Provider name (currently only 'anthropic')"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"auth_url": map[string]interface{}{
									"type":        "string",
									"description": "OAuth authorization URL to redirect to",
								},
								"state": map[string]interface{}{
									"type":        "string",
									"description": "OAuth state token for verification",
								},
								"message": map[string]interface{}{
									"type":        "string",
									"description": "Instructions for completing OAuth flow",
								},
							},
						}, "OAuth authorization information"),
						"400": createErrorResponse("Invalid provider or OAuth not supported"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/auth/oauth-callback": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "handleOAuthCallback",
					"summary":     "Handle OAuth callback",
					"description": "Process OAuth callback and exchange code for access token",
					"tags":        []string{"Authentication"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"provider", "code", "state"},
						"properties": map[string]interface{}{
							"provider": map[string]interface{}{
								"type":        "string",
								"description": "Provider name (anthropic)",
							},
							"code": map[string]interface{}{
								"type":        "string",
								"description": "Authorization code from OAuth provider",
							},
							"state": map[string]interface{}{
								"type":        "string",
								"description": "OAuth state for verification",
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Operation status",
								},
								"provider": map[string]interface{}{
									"type":        "string",
									"description": "Provider name",
								},
								"message": map[string]interface{}{
									"type":        "string",
									"description": "Status message",
								},
								"expires_in": map[string]interface{}{
									"type":        "integer",
									"description": "Seconds until token expiration",
								},
							},
						}, "OAuth completion status"),
						"400": createErrorResponse("Invalid request parameters"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/auth/{provider}": map[string]interface{}{
				"delete": map[string]interface{}{
					"operationId":  "deleteCredentials",
					"summary":     "Delete provider credentials",
					"description": "Delete stored API key and/or OAuth credentials for a provider",
					"tags":        []string{"Authentication"},
					"parameters": []map[string]interface{}{
						createPathParameter("provider", "Provider name (anthropic, openai, openrouter, gemini, brave)"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Operation status",
								},
								"provider": map[string]interface{}{
									"type":        "string",
									"description": "Provider name",
								},
								"message": map[string]interface{}{
									"type":        "string",
									"description": "Status message",
								},
							},
						}, "Credentials deletion status"),
						"400": createErrorResponse("Invalid provider"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/preferences": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "getPreferences",
					"summary":     "Get user preferences",
					"description": "Retrieve current user preferences including model and provider settings",
					"tags":        []string{"Preferences"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"preferences": map[string]interface{}{
									"type": "object",
									"description": "User preferences (null if no preferences exist)",
									"nullable": true,
									"properties": map[string]interface{}{
										"preferred_provider": map[string]interface{}{
											"type":        "string",
											"description": "Preferred AI provider (anthropic, openai, openrouter)",
										},
										"main_agent_model": map[string]interface{}{
											"type":        "string",
											"description": "Main agent model ID",
										},
										"main_agent_max_tokens": map[string]interface{}{
											"type":        "integer",
											"description": "Maximum tokens for main agent responses",
										},
										"main_agent_reasoning_effort": map[string]interface{}{
											"type":        "string",
											"description": "Reasoning effort setting for main agent",
										},
										"sub_agent_model": map[string]interface{}{
											"type":        "string",
											"description": "Sub agent model ID",
										},
										"sub_agent_max_tokens": map[string]interface{}{
											"type":        "integer",
											"description": "Maximum tokens for sub agent responses",
										},
										"sub_agent_reasoning_effort": map[string]interface{}{
											"type":        "string",
											"description": "Reasoning effort setting for sub agent",
										},
										"created_at": map[string]interface{}{
											"type":        "integer",
											"description": "Unix timestamp when preferences were created",
										},
										"updated_at": map[string]interface{}{
											"type":        "integer",
											"description": "Unix timestamp of last update",
										},
									},
								},
								"available_providers": map[string]interface{}{
									"type": "object",
									"description": "Map of available AI providers and their models",
									"additionalProperties": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"display_name": map[string]interface{}{
												"type":        "string",
												"description": "User-friendly provider name",
											},
											"models": map[string]interface{}{
												"type":        "array",
												"items":       map[string]interface{}{"type": "string"},
												"description": "Available models from this provider",
											},
										},
									},
								},
							},
							"required": []string{"available_providers"},
						}, "User preferences and available providers"),
						"500": createErrorResponse("Internal server error"),
					},
				},
				"post": map[string]interface{}{
					"operationId":  "updatePreferences",
					"summary":     "Update user preferences",
					"description": "Update user preferences including model and provider settings",
					"tags":        []string{"Preferences"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"preferred_provider": map[string]interface{}{
								"type":        "string",
								"description": "Preferred AI provider (anthropic, openai, openrouter)",
							},
							"main_agent_model": map[string]interface{}{
								"type":        "string",
								"description": "Main agent model ID",
							},
							"main_agent_max_tokens": map[string]interface{}{
								"type":        "integer",
								"description": "Maximum tokens for main agent responses",
							},
							"main_agent_reasoning_effort": map[string]interface{}{
								"type":        "string",
								"description": "Reasoning effort setting for main agent",
							},
							"sub_agent_model": map[string]interface{}{
								"type":        "string",
								"description": "Sub agent model ID",
							},
							"sub_agent_max_tokens": map[string]interface{}{
								"type":        "integer",
								"description": "Maximum tokens for sub agent responses",
							},
							"sub_agent_reasoning_effort": map[string]interface{}{
								"type":        "string",
								"description": "Reasoning effort setting for sub agent",
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"preferred_provider": map[string]interface{}{
									"type":        "string",
									"description": "Preferred AI provider",
								},
								"main_agent_model": map[string]interface{}{
									"type":        "string",
									"description": "Main agent model ID",
								},
								"main_agent_max_tokens": map[string]interface{}{
									"type":        "integer",
									"description": "Maximum tokens for main agent",
								},
								"main_agent_reasoning_effort": map[string]interface{}{
									"type":        "string",
									"description": "Reasoning effort for main agent",
								},
								"sub_agent_model": map[string]interface{}{
									"type":        "string",
									"description": "Sub agent model ID",
								},
								"sub_agent_max_tokens": map[string]interface{}{
									"type":        "integer",
									"description": "Maximum tokens for sub agent",
								},
								"sub_agent_reasoning_effort": map[string]interface{}{
									"type":        "string",
									"description": "Reasoning effort for sub agent",
								},
								"created_at": map[string]interface{}{
									"type":        "integer",
									"description": "Creation timestamp",
								},
								"updated_at": map[string]interface{}{
									"type":        "integer",
									"description": "Last update timestamp",
								},
							},
						}, "Updated preferences"),
						"400": createErrorResponse("Invalid request parameters"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/preferences/providers": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "getAvailableProviders",
					"summary":     "Get available providers",
					"description": "Retrieve list of available AI providers and their supported models",
					"tags":        []string{"Preferences"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"additionalProperties": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"display_name": map[string]interface{}{
										"type":        "string",
										"description": "Provider display name",
									},
									"models": map[string]interface{}{
										"type":        "array",
										"items":       map[string]interface{}{"type": "string"},
										"description": "Available models for this provider",
									},
								},
							},
							"description": "Map of available providers and their models",
						}, "Available providers"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/preferences/reset": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "resetPreferences",
					"summary":     "Reset preferences",
					"description": "Reset user preferences to default values",
					"tags":        []string{"Preferences"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"preferred_provider": map[string]interface{}{
									"type":        "string",
									"description": "Reset preferred provider",
								},
								"main_agent_model": map[string]interface{}{
									"type":        "string",
									"description": "Reset main agent model",
								},
								"main_agent_max_tokens": map[string]interface{}{
									"type":        "integer",
									"description": "Reset main agent max tokens",
								},
								"main_agent_reasoning_effort": map[string]interface{}{
									"type":        "string",
									"description": "Reset main agent reasoning effort",
								},
								"sub_agent_model": map[string]interface{}{
									"type":        "string",
									"description": "Reset sub agent model",
								},
								"sub_agent_max_tokens": map[string]interface{}{
									"type":        "integer",
									"description": "Reset sub agent max tokens",
								},
								"sub_agent_reasoning_effort": map[string]interface{}{
									"type":        "string",
									"description": "Reset sub agent reasoning effort",
								},
								"created_at": map[string]interface{}{
									"type":        "integer",
									"description": "Creation timestamp",
								},
								"updated_at": map[string]interface{}{
									"type":        "integer",
									"description": "Reset timestamp",
								},
							},
						}, "Reset preferences"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			// Session File Management Endpoints
			"/api/sessions/{id}/files/upload": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "uploadSessionFile",
					"summary":     "Upload file to session",
					"description": "Upload a file to session-specific storage directory",
					"tags":        []string{"Files"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"multipart/form-data": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"file": map[string]interface{}{
											"type":        "string",
											"format":      "binary",
											"description": "File to upload",
										},
									},
									"required": []string{"file"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": createSuccessResponse("object", getFileInfoSchema(), "File uploaded successfully"),
						"400": createErrorResponse("Invalid file or session ID"),
						"404": createErrorResponse("Session not found"),
						"413": createErrorResponse("File too large (max 32MB)"),
					},
				},
			},
			"/api/sessions/{id}/files": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "listSessionFiles",
					"summary":     "List session files",
					"description": "List all files in session storage directory",
					"tags":        []string{"Files"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("array", getFileInfoSchema(), "List of files in session"),
						"404": createErrorResponse("Session not found"),
					},
				},
			},
			"/api/sessions/{id}/files/{filename}": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "getSessionFile",
					"summary":     "Get session file",
					"description": "Download or serve a specific file from session storage. Supports thumbnail generation with ?thumb parameter.",
					"tags":        []string{"Files"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
						createPathParameter("filename", "Filename to retrieve"),
						{
							"name":        "thumb",
							"in":          "query",
							"description": "Thumbnail specification: '100' (box), 'w100' (width), 'h100' (height)",
							"schema": map[string]interface{}{
								"type":    "string",
								"pattern": "^(\\d+|w\\d+|h\\d+)$",
							},
						},
						{
							"name":        "time",
							"in":          "query",
							"description": "Time offset in seconds for video thumbnails (default: 1.0)",
							"schema": map[string]interface{}{
								"type":    "number",
								"minimum": 0,
								"maximum": 86400,
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "File content",
							"content": map[string]interface{}{
								"*/*": map[string]interface{}{
									"schema": map[string]interface{}{
										"type":        "string",
										"format":      "binary",
										"description": "File content with appropriate MIME type",
									},
								},
							},
						},
						"400": createErrorResponse("Invalid filename or thumbnail parameters"),
						"404": createErrorResponse("Session or file not found"),
					},
				},
				"delete": map[string]interface{}{
					"operationId":  "deleteSessionFile",
					"summary":     "Delete session file",
					"description": "Delete a specific file from session storage. Only files are supported - directories cannot be deleted.",
					"tags":        []string{"Files"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
						createPathParameter("filename", "Filename to delete"),
					},
					"responses": map[string]interface{}{
						"204": map[string]interface{}{
							"description": "File deleted successfully",
						},
						"400": createErrorResponse("Bad request - attempted to delete a directory"),
						"404": createErrorResponse("Session or file not found"),
					},
				},
			},
			// Tools & Agents API Endpoints
			"/api/tools/status": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "getToolsStatus",
					"summary":     "Get tools status",
					"description": "Get status and authentication information for all available tools and categories",
					"tags":        []string{"Tools"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"categories": map[string]interface{}{
									"type": "object",
									"additionalProperties": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"display_name": map[string]interface{}{
												"type":        "string",
												"description": "User-friendly category name",
											},
											"tools": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"provider": map[string]interface{}{
															"type":        "string",
															"description": "Tool provider name",
														},
														"display_name": map[string]interface{}{
															"type":        "string",
															"description": "User-friendly tool name",
														},
														"description": map[string]interface{}{
															"type":        "string",
															"description": "Tool description",
														},
														"authenticated": map[string]interface{}{
															"type":        "boolean",
															"description": "Whether tool is authenticated and ready to use",
														},
														"api_key_required": map[string]interface{}{
															"type":        "boolean",
															"description": "Whether tool requires API key authentication",
														},
													},
												},
												"description": "Available tools in this category",
											},
										},
									},
									"description": "Map of tool categories and their tools",
								},
							},
						}, "Tools status and authentication information"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/stream": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "streamEvents",
					"summary":     "Server-Sent Events stream for real-time updates",
					"description": "Establishes a persistent SSE connection for receiving real-time updates during message processing. Connection remains open for multiple messages and includes proper reconnection support with Last-Event-ID header.",
					"tags":        []string{"Streaming"},
					"parameters": []map[string]interface{}{
						{
							"name":        "sessionId",
							"in":          "query",
							"required":    true,
							"description": "Session ID to stream events for",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "Last-Event-ID",
							"in":          "header",
							"required":    false,
							"description": "Last received event ID for reconnection and event replay",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "SSE event stream",
							"headers": map[string]interface{}{
								"Content-Type": map[string]interface{}{
									"description": "Server-sent events MIME type",
									"schema": map[string]interface{}{
										"type":    "string",
										"example": "text/event-stream",
									},
								},
								"Cache-Control": map[string]interface{}{
									"description": "Prevents caching of the event stream",
									"schema": map[string]interface{}{
										"type":    "string",
										"example": "no-cache",
									},
								},
								"Connection": map[string]interface{}{
									"description": "Keep connection alive for streaming",
									"schema": map[string]interface{}{
										"type":    "string",
										"example": "keep-alive",
									},
								},
							},
							"content": map[string]interface{}{
								"text/event-stream": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/SSEEventStream",
									},
								},
							},
						},
						"404": createErrorResponse("Session not found"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/stream/{id}/message": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "sendStreamingMessage",
					"summary":     "Send message via streaming pipeline",
					"description": "Send a message to a session via the streaming pipeline. This endpoint integrates with active SSE connections to broadcast real-time processing events including thinking, content, tool execution, and completion events.",
					"tags":        []string{"Streaming"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID to send message to"),
					},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"content"},
						"properties": map[string]interface{}{
							"content": map[string]interface{}{
								"type":        "string",
								"description": "Message content to send for processing",
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"$ref": "#/components/schemas/StreamMessageResponse",
						}, "Message broadcasted to streaming pipeline. Real-time processing events will be sent via active SSE connections."),
						"400": createErrorResponse("Invalid message content"),
						"404": createErrorResponse("Session not found"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId":  "healthCheck",
					"summary":     "Health check",
					"description": "Check server health and status",
					"tags":        []string{"System"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Health status",
								},
								"timestamp": map[string]interface{}{
									"type":        "string",
									"description": "Current timestamp",
								},
								"version": map[string]interface{}{
									"type":        "string",
									"description": "Application version",
								},
							},
						}, "Health information"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
		},
		Components: OpenAPIComponents{
			Schemas: map[string]interface{}{
				"CoreToolName": map[string]interface{}{
					"type": "string",
					"enum": []string{
						"bash", "ReadText", "glob", "ReadMedia", "grep", "write", "edit",
						"python_execution", "search", "todo_write", "exit_plan_mode",
						"show_media", "task",
					},
					"description": "Core built-in tool names",
				},
				"ToolName": map[string]interface{}{
					"anyOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/CoreToolName"},
						{
							"type":        "string",
							"pattern":     "^[a-zA-Z0-9_]+_[a-zA-Z0-9_]+$",
							"description": "MCP tool names following pattern: {serverName}_{toolName}",
							"example":     "brave_search",
						},
					},
					"description": "Tool name - either a core tool or MCP tool following {serverName}_{toolName} pattern",
				},
				"ErrorResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"error": map[string]interface{}{
							"$ref": "#/components/schemas/RESTError",
						},
					},
					"required": []string{"error"},
				},
				"RESTError": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "integer",
							"description": "HTTP status code",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Error message",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Error type",
							"enum":        []string{"bad_request", "not_found", "internal_error", "unauthorized", "validation_error"},
						},
					},
					"required": []string{"code", "message", "type"},
				},
				"MessageRole": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"user", "assistant", "tool"},
					"description": "Message role",
				},
				"SessionData": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique session identifier",
						},
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Session title",
						},
						"userMessageCount": map[string]interface{}{
							"type":        "integer",
							"description": "Number of user messages in session",
						},
						"assistantMessageCount": map[string]interface{}{
							"type":        "integer",
							"description": "Number of assistant messages in session",
						},
						"toolCallCount": map[string]interface{}{
							"type":        "integer",
							"description": "Number of tool calls made in session",
						},
						"promptTokens": map[string]interface{}{
							"type":        "integer",
							"description": "Total prompt tokens used",
						},
						"completionTokens": map[string]interface{}{
							"type":        "integer",
							"description": "Total completion tokens used",
						},
						"cost": map[string]interface{}{
							"type":        "number",
							"format":      "double",
							"description": "Total cost of session",
						},
						"createdAt": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Session creation timestamp",
						},
						"firstUserMessage": map[string]interface{}{
							"type":        "string",
							"description": "First user message (optional)",
						},
					},
					"required": []string{"id", "title", "userMessageCount", "assistantMessageCount", "toolCallCount", "promptTokens", "completionTokens", "cost", "createdAt"},
				},
				"MessageData": map[string]interface{}{
					"type": "object",
					"description": "Message data structure for user input",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{
							"type":        "string",
							"description": "The text content of the message",
						},
						"plan_mode": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the message is in planning mode",
							"default":     false,
						},
					},
					"required": []string{"text"},
				},
				"BackendMessage": map[string]interface{}{
					"type": "object",
					"description": "Backend message structure representing a complete message exchange",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique message identifier",
						},
						"sessionId": map[string]interface{}{
							"type":        "string",
							"description": "Session identifier",
						},
						"role": map[string]interface{}{
							"type":        "string",
							"description": "Message role (user, assistant, tool)",
						},
						"userInput": map[string]interface{}{
							"type":        "string",
							"description": "User's input message",
						},
						"assistantResponse": map[string]interface{}{
							"type":        "string",
							"description": "Assistant's response message (optional)",
						},
						"toolCalls": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/ToolCallData",
							},
							"description": "Tool calls made during message processing",
						},
						"reasoning": map[string]interface{}{
							"type":        "string",
							"description": "Reasoning process (optional)",
						},
						"reasoningDuration": map[string]interface{}{
							"type":        "integer",
							"description": "Reasoning duration in milliseconds (optional)",
						},
					},
					"required": []string{"id", "sessionId", "role", "userInput"},
				},
				"ToolCallData": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique tool call identifier",
						},
						"name": map[string]interface{}{
							"$ref":        "#/components/schemas/ToolName",
							"description": "Tool name",
						},
						"input": map[string]interface{}{
							"type":        "string",
							"description": "Tool input parameters",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Tool type",
						},
						"finished": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether tool call has finished",
						},
						"result": map[string]interface{}{
							"type":        "string",
							"description": "Tool execution result (optional)",
						},
						"isError": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether tool call resulted in error (optional)",
						},
					},
					"required": []string{"id", "name", "input", "type", "finished"},
				},
				"FileInfo": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "File name",
						},
						"size": map[string]interface{}{
							"type":        "integer",
							"format":      "int64",
							"description": "File size in bytes",
						},
						"modified": map[string]interface{}{
							"type":        "integer",
							"format":      "int64",
							"description": "Last modified timestamp (Unix time)",
						},
						"isDir": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether this is a directory",
						},
						"url": map[string]interface{}{
							"type":        "string",
							"description": "Static URL to access the file",
						},
					},
					"required": []string{"name", "size", "modified", "isDir", "url"},
				},
				"SSEBaseEvent": map[string]interface{}{
					"type":        "object",
					"description": "Base SSE event with standard fields",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique sequential event identifier for ordering and reconnection",
							"example":     "1234567890",
						},
						"event": map[string]interface{}{
							"type":        "string",
							"description": "Event type identifier",
							"enum":        []string{"connected", "heartbeat", "error", "complete", "thinking", "content", "tool", "tool_execution_start", "tool_execution_complete", "permission", "summarize"},
						},
						"retry": map[string]interface{}{
							"type":        "integer",
							"description": "Client retry interval in milliseconds",
							"example":     30000,
						},
					},
					"required": []string{"id", "event"},
				},
				"SSEEventStream": map[string]interface{}{
					"type":         "object",
					"description":  "Server-Sent Event stream with discriminated event types",
					"discriminator": map[string]interface{}{
						"propertyName": "event",
						"mapping": map[string]interface{}{
							"connected":              "#/components/schemas/SSEConnectedEvent",
							"heartbeat":              "#/components/schemas/SSEHeartbeatEvent",
							"error":                  "#/components/schemas/SSEErrorEvent",
							"complete":               "#/components/schemas/SSECompleteEvent",
							"thinking":               "#/components/schemas/SSEThinkingEvent",
							"content":                "#/components/schemas/SSEContentEvent",
							"tool":                   "#/components/schemas/SSEToolEvent",
							"tool_execution_start":   "#/components/schemas/SSEToolExecutionStartEvent",
							"tool_execution_complete": "#/components/schemas/SSEToolExecutionCompleteEvent",
							"permission":             "#/components/schemas/SSEPermissionEvent",
							"summarize":              "#/components/schemas/SSESummarizeEvent",
						},
					},
					"oneOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEConnectedEvent"},
						{"$ref": "#/components/schemas/SSEHeartbeatEvent"},
						{"$ref": "#/components/schemas/SSEErrorEvent"},
						{"$ref": "#/components/schemas/SSECompleteEvent"},
						{"$ref": "#/components/schemas/SSEThinkingEvent"},
						{"$ref": "#/components/schemas/SSEContentEvent"},
						{"$ref": "#/components/schemas/SSEToolEvent"},
						{"$ref": "#/components/schemas/SSEToolExecutionStartEvent"},
						{"$ref": "#/components/schemas/SSEToolExecutionCompleteEvent"},
						{"$ref": "#/components/schemas/SSEPermissionEvent"},
						{"$ref": "#/components/schemas/SSESummarizeEvent"},
					},
				},
				"SSEConnectedEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"sessionId": map[string]interface{}{
											"type":        "string",
											"description": "Session identifier for the connected stream",
										},
									},
									"required": []string{"sessionId"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEHeartbeatEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Heartbeat type",
											"example":     "ping",
										},
									},
									"required": []string{"type"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEErrorEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"error": map[string]interface{}{
											"type":        "string",
											"description": "Error message description",
										},
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Error type classification",
										},
										"retryAfter": map[string]interface{}{
											"type":        "integer",
											"description": "Milliseconds to wait before retry",
										},
										"attempt": map[string]interface{}{
											"type":        "integer",
											"description": "Current retry attempt number",
										},
										"maxAttempts": map[string]interface{}{
											"type":        "integer",
											"description": "Maximum number of retry attempts",
										},
									},
									"required": []string{"error"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSECompleteEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Completion type",
										},
										"content": map[string]interface{}{
											"type":        "string",
											"description": "Final response content",
										},
										"messageId": map[string]interface{}{
											"type":        "string",
											"description": "Completed message identifier",
										},
										"done": map[string]interface{}{
											"type":        "boolean",
											"description": "Indicates message processing completion",
										},
										"reasoning": map[string]interface{}{
											"type":        "string",
											"description": "Optional reasoning content",
										},
										"reasoningDuration": map[string]interface{}{
											"type":        "integer",
											"description": "Duration of reasoning process in milliseconds",
										},
									},
									"required": []string{"type", "done"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEThinkingEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Thinking event type",
										},
										"content": map[string]interface{}{
											"type":        "string",
											"description": "Thinking or reasoning content",
										},
									},
									"required": []string{"type", "content"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEContentEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Content event type",
										},
										"content": map[string]interface{}{
											"type":        "string",
											"description": "Streaming content delta",
										},
									},
									"required": []string{"type", "content"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEToolEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Tool event type",
										},
										"name": map[string]interface{}{
											"$ref":        "#/components/schemas/ToolName",
											"description": "Tool name being executed",
										},
										"input": map[string]interface{}{
											"type":        "string",
											"description": "Tool input parameters",
										},
										"id": map[string]interface{}{
											"type":        "string",
											"description": "Tool execution identifier",
										},
										"status": map[string]interface{}{
											"type":        "string",
											"description": "Tool execution status",
										},
									},
									"required": []string{"type", "name", "input", "id", "status"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEToolExecutionStartEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Tool execution start event type",
										},
										"toolName": map[string]interface{}{
											"$ref":        "#/components/schemas/ToolName",
											"description": "Name of the tool being executed",
										},
										"progress": map[string]interface{}{
											"type":        "string",
											"description": "Execution progress description",
										},
										"toolCallId": map[string]interface{}{
											"type":        "string",
											"description": "Tool call identifier",
										},
									},
									"required": []string{"type", "toolName", "progress", "toolCallId"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEToolExecutionCompleteEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Tool execution complete event type",
										},
										"toolName": map[string]interface{}{
											"$ref":        "#/components/schemas/ToolName",
											"description": "Name of the completed tool",
										},
										"progress": map[string]interface{}{
											"type":        "string",
											"description": "Final execution progress description",
										},
										"success": map[string]interface{}{
											"type":        "boolean",
											"description": "Indicates if tool execution succeeded",
										},
										"toolCallId": map[string]interface{}{
											"type":        "string",
											"description": "Tool call identifier",
										},
									},
									"required": []string{"type", "toolName", "progress", "success", "toolCallId"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEPermissionEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Permission event type",
										},
										"id": map[string]interface{}{
											"type":        "string",
											"description": "Permission request identifier",
										},
										"sessionId": map[string]interface{}{
											"type":        "string",
											"description": "Session identifier for the permission request",
										},
										"toolName": map[string]interface{}{
											"$ref":        "#/components/schemas/ToolName",
											"description": "Tool requiring permission",
										},
										"description": map[string]interface{}{
											"type":        "string",
											"description": "Human-readable permission description",
										},
										"action": map[string]interface{}{
											"type":        "string",
											"description": "Requested action description",
										},
										"path": map[string]interface{}{
											"type":        "string",
											"description": "File path for permission request",
										},
										"params": map[string]interface{}{
											"type":        "object",
											"description": "Additional parameters for the permission request",
										},
									},
									"required": []string{"type", "id", "sessionId", "toolName", "description", "action"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"StreamMessageResponse": map[string]interface{}{
					"description": "Response from streaming message endpoint indicating broadcast status",
					"type": "object",
					"properties": map[string]interface{}{
						"sessionId": map[string]interface{}{
							"type":        "string",
							"description": "Session identifier",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"broadcasted"},
							"description": "Broadcast status",
						},
					},
					"required": []string{"status", "sessionId"},
				},
				"SSESummarizeEvent": map[string]interface{}{
					"allOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEBaseEvent"},
						{
							"type": "object",
							"properties": map[string]interface{}{
								"data": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Summarization event type",
										},
										"progress": map[string]interface{}{
											"type":        "string",
											"description": "Summarization progress description",
										},
										"done": map[string]interface{}{
											"type":        "boolean",
											"description": "Indicates if summarization is complete",
										},
									},
									"required": []string{"type", "progress", "done"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
			},
		},
	}
}

// Helper functions for OpenAPI schema generation
func createPathParameter(name, description string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"in":          "path",
		"required":    true,
		"description": description,
		"schema": map[string]interface{}{
			"type": "string",
		},
	}
}

func createRequestBody(schema map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"required": true,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": schema,
			},
		},
	}
}

func createSuccessResponse(dataType string, schema map[string]interface{}, description string) map[string]interface{} {
	var responseSchema map[string]interface{}
	if dataType == "array" {
		responseSchema = map[string]interface{}{
			"type":  "array",
			"items": schema,
		}
	} else {
		responseSchema = schema
	}

	return map[string]interface{}{
		"description": description,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": responseSchema,
			},
		},
	}
}

func createErrorResponse(description string) map[string]interface{} {
	return map[string]interface{}{
		"description": description,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": "#/components/schemas/ErrorResponse",
				},
			},
		},
	}
}

func getSessionDataSchema() map[string]interface{} {
	return map[string]interface{}{
		"$ref": "#/components/schemas/SessionData",
	}
}

func getMessageDataSchema() map[string]interface{} {
	return map[string]interface{}{
		"$ref": "#/components/schemas/MessageData",
	}
}

func getToolCallDataSchema() map[string]interface{} {
	return map[string]interface{}{
		"$ref": "#/components/schemas/ToolCallData",
	}
}

func getFileInfoSchema() map[string]interface{} {
	return map[string]interface{}{
		"$ref": "#/components/schemas/FileInfo",
	}
}

func getBackendMessageSchema() map[string]interface{} {
	return map[string]interface{}{
		"$ref": "#/components/schemas/BackendMessage",
	}
}
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
					"description": "Create a new session with required title and optional working directory",
					"tags":        []string{"Sessions"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"title"},
						"properties": map[string]interface{}{
							"title": map[string]interface{}{
								"type":        "string",
								"description": "Title for the session",
							},
							"workingDirectory": map[string]interface{}{
								"type":        "string",
								"description": "Optional working directory path",
							},
						},
					}),
					"responses": map[string]interface{}{
						"201": createSuccessResponse("object", getSessionDataSchema(), "Created session"),
						"400": createErrorResponse("Invalid request data"),
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
								"minimum":     1,
								"description": "Index of the last message to include in the fork (1-based)",
							},
							"title": map[string]interface{}{
								"type":        "string",
								"description": "Optional title for the forked session (defaults to 'Forked Session')",
							},
						},
					}),
					"responses": map[string]interface{}{
						"201": createSuccessResponse("object", getSessionDataSchema(), "Forked session"),
						"400": createErrorResponse("Invalid request - messageIndex must be > 0"),
						"404": createErrorResponse("Source session not found"),
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
						"required": []string{"content"},
						"properties": map[string]interface{}{
							"content": map[string]interface{}{
								"type":        "string",
								"description": "Message content to send",
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", getMessageDataSchema(), "Message sent and processed"),
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
						"200": createSuccessResponse("array", getMessageDataSchema(), "List of session messages"),
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
						"200": createSuccessResponse("array", getMessageDataSchema(), "Message history"),
						"401": createErrorResponse("Unauthorized - authentication required"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			// System Operations
			"/api/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "initiateOAuthLogin",
					"summary":     "OAuth authentication",
					"description": "Initiate OAuth authentication flow",
					"tags":        []string{"Authentication"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"authUrl": map[string]interface{}{
									"type":        "string",
									"description": "OAuth authorization URL",
								},
							},
						}, "Authentication URL"),
						"401": createErrorResponse("Unauthorized - authentication failed"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/auth/apikey": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId":  "setApiKey",
					"summary":     "Set API key",
					"description": "Set API key for direct authentication",
					"tags":        []string{"Authentication"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type": "object",
						"required": []string{"apiKey"},
						"properties": map[string]interface{}{
							"apiKey": map[string]interface{}{
								"type":        "string",
								"description": "API key for authentication",
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"success": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether API key was set successfully",
								},
							},
						}, "API key set status"),
						"400": createErrorResponse("Invalid API key"),
						"401": createErrorResponse("Unauthorized - authentication failed"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
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
								"description": "Provider name (anthropic, openai, openrouter)",
								"enum":        []string{"anthropic", "openai", "openrouter"},
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
						createPathParameter("provider", "Provider name (anthropic, openai, openrouter)"),
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
						}, "User preferences"),
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
						"workingDirectory": map[string]interface{}{
							"type":        "string",
							"description": "Working directory path (optional)",
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
							"$ref": "#/components/schemas/MessageRole",
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
							"type":        "string",
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
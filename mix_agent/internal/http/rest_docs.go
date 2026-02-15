package http

import (
	"encoding/json"
	"net/http"

	"mix/internal/constants"
)

// HandleDocumentation serves OpenAPI 3.1 specification as JSON
func HandleDocumentation(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if handleCORSPreflight(w, r) {
		return
	}

	serveOpenAPISpec(w)
}

// serveOpenAPISpec serves the OpenAPI 3.1 specification as JSON
func serveOpenAPISpec(w http.ResponseWriter) {
	spec := getOpenAPISpec()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(spec)
}

// OpenAPI 3.1 specification structures with proper field ordering
type OpenAPISpec struct {
	OpenAPI           string                 `json:"openapi"`
	Info              OpenAPIInfo            `json:"info"`
	XSpeakeasyRetries map[string]interface{} `json:"x-speakeasy-retries"`
	Paths             map[string]interface{} `json:"paths"`
	Components        OpenAPIComponents      `json:"components"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type OpenAPIComponents struct {
	Schemas map[string]interface{} `json:"schemas"`
}

// getOpenAPISpec returns the complete OpenAPI 3.1 specification with proper field ordering
//
//nolint:funlen // OpenAPI spec is necessarily long
func getOpenAPISpec() OpenAPISpec {
	return OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       "Mix REST API",
			Description: "REST API for the Mix application - session management, messaging, and system operations",
			Version:     "1.0.0",
		},
		XSpeakeasyRetries: map[string]interface{}{
			"strategy": "backoff",
			"backoff": map[string]interface{}{
				"initialInterval": constants.RetryInitialInterval,
				"maxInterval":     constants.RetryMaxInterval,
				"maxElapsedTime":  constants.RetryMaxElapsedTime,
				"exponent":        constants.RetryBackoffExponent,
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
					"operationId": "listSessions",
					"summary":     "List all sessions",
					"description": "Retrieve a list of all available sessions with their metadata",
					"tags":        []string{"Sessions"},
					"parameters": []map[string]interface{}{
						{
							"in":          "query",
							"name":        "includeSubagents",
							"schema":      map[string]interface{}{"type": "boolean", "default": false},
							"required":    false,
							"description": "Include subagent sessions in response (default: false, subagent sessions are hidden)",
						},
					},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("array", getSessionDataSchema(), "List of sessions"),
						"401": createErrorResponse("Unauthorized - authentication required"),
						"500": createErrorResponse("Internal server error"),
					},
				},
				"post": map[string]interface{}{
					"operationId": "createSession",
					"summary":     "Create a new session",
					"description": "Create a new session with required title and optional custom system prompt. Session automatically gets isolated storage directory. Supports session-level callbacks for automated actions after tool execution.",
					"tags":        []string{"Sessions"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type":     "object",
						"required": []string{"title", "browserMode"},
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
							"sessionType": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"main"},
								"default":     "main",
								"description": "Session type. API can only create 'main' sessions. Subagent sessions are created automatically by the task delegation system.",
								"example":     "main",
							},
							"subagentType": map[string]interface{}{
								"type":        "string",
								"description": "Subagent type - must not be set for API-created sessions. This field is reserved for programmatic subagent creation.",
								"example":     "",
							},
							"browserMode": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"electron-embedded-browser", "local-browser-service", "remote-cdp-websocket"},
								"description": "Browser automation mode (required):\n- 'electron-embedded-browser': Electron app with embedded Chromium browser\n- 'local-browser-service': Local browser-service (GoRod-based)\n- 'remote-cdp-websocket': Remote CDP WebSocket URL (cloud browser providers)",
								"example":     "local-browser-service",
							},
							"cdpUrl": map[string]interface{}{
								"type":        "string",
								"description": "CDP WebSocket URL for remote browser connections. Required when browserMode is 'remote-cdp-websocket'. Must start with 'ws://' or 'wss://'.",
								"example":     "wss://connect.browserbase.com/v1/sessions/abc123",
							},
							"callbacks": map[string]interface{}{
								"type":        "array",
								"description": "Session-level callbacks that execute after tool completion. Environment variables available: CALLBACK_TOOL_RESULT, CALLBACK_TOOL_NAME, CALLBACK_TOOL_ID, CALLBACK_SESSION_ID",
								"items": map[string]interface{}{
									"$ref": "#/components/schemas/Callback",
								},
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
										"invalid_session_type": map[string]interface{}{
											"summary": "Invalid session type",
											"value": map[string]interface{}{
												"error": map[string]interface{}{
													"code":    400,
													"message": "API can only create main sessions. Subagent sessions are created automatically.",
													"type":    "validation_error",
												},
											},
										},
										"subagent_type_not_allowed": map[string]interface{}{
											"summary": "Subagent type not allowed for API-created sessions",
											"value": map[string]interface{}{
												"error": map[string]interface{}{
													"code":    400,
													"message": "subagentType cannot be set for API-created sessions. Subagent sessions are created programmatically by the task delegation system.",
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
					"operationId": "getSession",
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
					"operationId": "deleteSession",
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
			"/api/sessions/{id}/callbacks": map[string]interface{}{
				"patch": map[string]interface{}{
					"operationId": "updateSessionCallbacks",
					"summary":     "Update session callbacks",
					"description": "Update the callback configurations for a session. Callbacks execute automatically after tool completion. Pass an empty array to clear all callbacks.",
					"tags":        []string{"Sessions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID to update"),
					},
					"requestBody": createRequestBody(map[string]interface{}{
						"type":     "object",
						"required": []string{"callbacks"},
						"properties": map[string]interface{}{
							"callbacks": map[string]interface{}{
								"type":        "array",
								"description": "Session-level callbacks that execute after tool completion. Environment variables available: CALLBACK_TOOL_RESULT, CALLBACK_TOOL_NAME, CALLBACK_TOOL_ID, CALLBACK_SESSION_ID",
								"items": map[string]interface{}{
									"$ref": "#/components/schemas/Callback",
								},
							},
						},
					}),
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", getSessionDataSchema(), "Session callbacks updated successfully"),
						"400": createErrorResponse("Invalid request - validation error in callbacks array"),
						"404": createErrorResponse("Session not found"),
					},
				},
			},
			"/api/sessions/{id}/rewind": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "rewindSession",
					"summary":     "Rewind a session",
					"description": "Delete messages after a specified message in the current session, optionally cleaning up media files created after that point",
					"tags":        []string{"Sessions"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID to rewind"),
					},
					"requestBody": createRequestBody(map[string]interface{}{
						"type":     "object",
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
					"operationId": "exportSession",
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
					"operationId": "sendMessage",
					"summary":     "Send a message to session (async)",
					"description": "Send a user message to a specific session for AI processing. Returns immediately with 202 Accepted. All results stream via Server-Sent Events (SSE) connection.",
					"tags":        []string{"Messages"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Session ID"),
					},
					"requestBody": createRequestBody(map[string]interface{}{
						"type":     "object",
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
							"thinking_level": map[string]interface{}{
								"type":        "string",
								"description": "Thinking level: off (disabled), basic (4k tokens), medium (10k tokens), maximum (32k tokens). If not provided, determined by keywords in message.",
								"enum":        []string{"off", "basic", "medium", "maximum"},
								"nullable":    true,
								"example":     "medium",
							},
							"max_steps": map[string]interface{}{
								"type":        "integer",
								"description": "Maximum tool call iterations for this message. If not provided, unlimited iterations allowed.",
								"minimum":     1,
								"example":     25,
							},
						},
					}),
					"responses": map[string]interface{}{
						"202": map[string]interface{}{
							"description": "Message accepted for processing. Agent runs asynchronously and streams results via SSE.",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type":        "string",
												"description": "Processing status",
												"example":     "processing",
											},
											"sessionId": map[string]interface{}{
												"type":        "string",
												"description": "Session ID for the processing task",
											},
										},
										"required": []string{"status", "sessionId"},
									},
								},
							},
						},
						"400": createErrorResponse("Invalid message data"),
						"404": createErrorResponse("Session not found"),
					},
				},
				"get": map[string]interface{}{
					"operationId": "getSessionMessages",
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
					"operationId": "cancelSessionProcessing",
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
					"operationId": "getMessageHistory",
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
					"operationId": "listMcpServers",
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
					"operationId": "listCommands",
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
					"operationId": "getCommand",
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
			"/api/system/info": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "getSystemInfo",
					"summary":     "Get system information",
					"description": "Retrieve system information including storage configuration",
					"tags":        []string{"System"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"storageBasePath": map[string]interface{}{
									"type":        "string",
									"description": "Absolute path to the storage base directory",
								},
							},
							"required": []string{"storageBasePath"},
						}, "System information"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/permissions/{id}/grant": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "grantPermission",
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
					"operationId": "denyPermission",
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
			"/api/notifications/{id}/respond": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "respondToNotification",
					"summary":     "Respond to notification",
					"description": "Send user's response to a notification request",
					"tags":        []string{"Notifications"},
					"parameters": []map[string]interface{}{
						createPathParameter("id", "Notification ID"),
					},
					"requestBody": createRequestBody(map[string]interface{}{
						"type":     "object",
						"required": []string{"type"},
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type":        "string",
								"description": "Response type",
								"enum":        []string{"acknowledge", "text", "choice"},
							},
							"value": map[string]interface{}{
								"type":        "string",
								"description": "User's text input or selected choice (optional for acknowledge type)",
							},
						},
					}),
					"responses": map[string]interface{}{
						"204": map[string]interface{}{
							"description": "Notification response accepted",
						},
						"401": createErrorResponse("Unauthorized - authentication required"),
						"404": createErrorResponse("Notification not found or already responded"),
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/auth/api-key": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "storeApiKey",
					"summary":     "Store API key",
					"description": "Store API key for direct authentication with a specific provider",
					"tags":        []string{"Authentication"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type":     "object",
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
					"operationId": "getAuthStatus",
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
					"operationId": "validatePreferredProvider",
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
					"operationId": "startOAuthFlow",
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
					"operationId": "handleOAuthCallback",
					"summary":     "Handle OAuth callback",
					"description": "Process OAuth callback and exchange code for access token",
					"tags":        []string{"Authentication"},
					"requestBody": createRequestBody(map[string]interface{}{
						"type":     "object",
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
					"operationId": "deleteCredentials",
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
			"/internal/auth/refresh-tokens": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "refreshOAuthTokens",
					"summary":     "Manually refresh OAuth tokens",
					"description": "Manually trigger OAuth token refresh for all expired tokens. Normally tokens are refreshed automatically by the background service every 30 minutes.",
					"tags":        []string{"Authentication", "Internal"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Operation status",
									"example":     "success",
								},
								"message": map[string]interface{}{
									"type":        "string",
									"description": "Status message",
									"example":     "Token refresh triggered successfully",
								},
							},
						}, "Token refresh triggered"),
						"500": createErrorResponse("Token refresh service not available or internal error"),
					},
				},
			},
			"/health/auth": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "getOAuthHealth",
					"summary":     "Get OAuth authentication health",
					"description": "Get health status of all OAuth credentials. Background service refreshes tokens 35 minutes before expiry. API calls mark tokens expired 5 minutes before expiry. Health statuses: 'healthy' (tokens valid, >5min remaining), 'degraded' (some tokens within 5min of expiry but refreshable), 'unhealthy' (tokens expired without refresh capability)",
					"tags":        []string{"Health", "Authentication"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Overall health status",
									"enum":        []string{"healthy", "degraded", "unhealthy"},
									"example":     "healthy",
								},
								"providers": map[string]interface{}{
									"type": "object",
									"additionalProperties": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"provider": map[string]interface{}{
												"type":        "string",
												"description": "Provider name",
											},
											"status": map[string]interface{}{
												"type":        "string",
												"description": "Token status",
												"enum":        []string{"active", "expired", "expired_no_refresh", "error", "not_found"},
											},
											"expires_at": map[string]interface{}{
												"type":        "string",
												"format":      "date-time",
												"description": "Token expiration time",
											},
											"expires_in": map[string]interface{}{
												"type":        "string",
												"description": "Human-readable time until expiration",
												"example":     "2h30m15s",
											},
											"last_refresh": map[string]interface{}{
												"type":        "string",
												"format":      "date-time",
												"description": "Last time token was refreshed",
											},
											"error": map[string]interface{}{
												"type":        "string",
												"description": "Error message if status is 'error'",
											},
										},
										"required": []string{"provider", "status"},
									},
									"description": "Map of provider OAuth health status",
								},
								"timestamp": map[string]interface{}{
									"type":        "string",
									"format":      "date-time",
									"description": "Health check timestamp",
								},
							},
							"required": []string{"status", "providers", "timestamp"},
						}, "OAuth health status"),
						"500": createErrorResponse("Health check service not available or internal error"),
					},
				},
			},
			"/api/preferences": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "getPreferences",
					"summary":     "Get user preferences",
					"description": "Retrieve current user preferences including model and provider settings",
					"tags":        []string{"Preferences"},
					"responses": map[string]interface{}{
						"200": createSuccessResponse("object", map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"preferences": map[string]interface{}{
									"$ref":        "#/components/schemas/UserPreferencesResponse",
									"description": "User preferences (null if no preferences exist)",
									"nullable":    true,
								},
								"available_providers": map[string]interface{}{
									"type":        "object",
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
					"operationId": "updatePreferences",
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
					"operationId": "getAvailableProviders",
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
					"operationId": "resetPreferences",
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
					"operationId": "uploadSessionFile",
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
					"operationId": "listSessionFiles",
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
					"operationId": "getSessionFile",
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
					"operationId": "deleteSessionFile",
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
					"operationId": "getToolsStatus",
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
					"operationId": "streamEvents",
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
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "healthCheck",
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
			// Tools Management Endpoints
			"/api/tools": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "listLLMTools",
					"summary":     "List LLM tools",
					"description": "Returns the list of all LLM tools that Claude can invoke. The list is dynamically extracted from the actual tools registered in CoderAgentTools() (agent/tools.go), ensuring it always reflects the current tool availability. Typical tools include: Bash, Edit, Read, Write, Grep, Glob, WebFetch, WebSearch, ReadMedia, TodoWrite, ExitPlanMode, and Task. This endpoint is useful for creating tool callbacks or understanding available agent capabilities.",
					"tags":        []string{"Tools"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of LLM tools",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"tools": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"name": map[string]interface{}{
															"type":        "string",
															"description": "Tool name",
															"example":     "Bash",
														},
														"description": map[string]interface{}{
															"type":        "string",
															"description": "Tool description",
															"example":     "Execute bash commands in a persistent shell session",
														},
														"parameters": map[string]interface{}{
															"type":        "object",
															"description": "Tool parameter schema",
														},
														"required": map[string]interface{}{
															"type":        "array",
															"items":       map[string]interface{}{"type": "string"},
															"description": "Required parameters",
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"500": createErrorResponse("Internal server error"),
					},
				},
			},
			"/api/tools/credentials-status": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "getToolCredentialsStatus",
					"summary":     "Get tool credentials status",
					"description": "Returns authentication/credential status for external tool integrations (Brave Search, Gemini Vision, etc.). This endpoint checks if API keys are configured for tools that require external service credentials.",
					"tags":        []string{"Tools"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Tool credentials status",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"categories": map[string]interface{}{
												"type":        "object",
												"description": "Tool categories grouped by type",
												"additionalProperties": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"display_name": map[string]interface{}{
															"type":        "string",
															"description": "Category display name",
															"example":     "Web Search",
														},
														"description": map[string]interface{}{
															"type":        "string",
															"description": "Category description",
															"example":     "Search the web for real-time information",
														},
														"icon": map[string]interface{}{
															"type":        "string",
															"description": "Category icon",
															"example":     "🔍",
														},
														"tools": map[string]interface{}{
															"type":        "array",
															"description": "Tools in this category",
															"items": map[string]interface{}{
																"type": "object",
																"properties": map[string]interface{}{
																	"authenticated": map[string]interface{}{
																		"type":        "boolean",
																		"description": "Whether the tool has valid credentials",
																		"example":     true,
																	},
																	"display_name": map[string]interface{}{
																		"type":        "string",
																		"description": "Tool display name",
																		"example":     "Brave Search",
																	},
																	"description": map[string]interface{}{
																		"type":        "string",
																		"description": "Tool description",
																		"example":     "Privacy-focused web search with real-time results",
																	},
																	"api_key_format": map[string]interface{}{
																		"type":        "string",
																		"description": "Expected API key format",
																		"example":     "BSA...",
																	},
																	"api_key_required": map[string]interface{}{
																		"type":        "boolean",
																		"description": "Whether an API key is required",
																		"example":     true,
																	},
																	"provider": map[string]interface{}{
																		"type":        "string",
																		"description": "Provider identifier",
																		"example":     "brave",
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
							},
						},
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
						"Bash", "ReadText", "Glob", "ReadMedia", "Grep", "Write", "Edit", "Search", "TodoWrite", "ExitPlanMode",
						"Show", "Task",
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
				"Callback": map[string]interface{}{
					"type":        "object",
					"description": "Session-level callback configuration that executes after tool completion",
					"required":    []string{"toolName", "type"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Human-readable name for this callback (optional, defaults to 'Callback #XXXX')",
							"example":     "Log Output",
						},
						"toolName": map[string]interface{}{
							"type":        "string",
							"description": "Tool to attach callback to (e.g., 'show', 'bash', '*' for all tools)",
							"example":     "*",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"bash_script", "sub_agent", "send_message"},
							"description": "Callback type: 'bash_script' for shell commands, 'sub_agent' for spawning sub-agents, 'send_message' for injecting messages",
						},
						"bashCommand": map[string]interface{}{
							"type":        "string",
							"description": "Bash command to execute (required for bash_script type). Has access to environment variables.",
						},
						"bashTimeout": map[string]interface{}{
							"type":        "integer",
							"description": "Timeout in milliseconds for bash execution (default: 120000)",
							"default":     120000,
						},
						"subAgentPrompt": map[string]interface{}{
							"type":        "string",
							"description": "Prompt for the sub-agent (required for sub_agent type). Tool execution context is automatically appended.",
						},
						"subAgentType": map[string]interface{}{
							"type":        "string",
							"description": "Type of sub-agent to spawn (default: 'general-purpose')",
							"default":     "general-purpose",
						},
						"includeFullHistory": map[string]interface{}{
							"type":        "boolean",
							"description": "Include full conversation history in sub-agent context (not yet implemented)",
							"default":     false,
						},
						"messageContent": map[string]interface{}{
							"type":        "string",
							"description": "Message content to inject into the conversation (required for send_message type). Will be sent as a User message.",
							"example":     "Please review the changes and run tests",
						},
						"excludeFromContext": map[string]interface{}{
							"type":        "boolean",
							"description": "Exclude callback results from agent context. Only applies to bash_script and sub_agent types. Not allowed for send_message.",
							"default":     false,
						},
					},
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
						"parentSessionId": map[string]interface{}{
							"type":        "string",
							"description": "Parent session ID for subagent sessions (null for main sessions)",
						},
						"parentToolCallId": map[string]interface{}{
							"type":        "string",
							"description": "ID of the tool call that spawned this subagent session (null for non-subagent sessions)",
						},
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Session title",
						},
						"sessionType": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"main", "subagent"},
							"description": "Session type:\n- 'main': Root-level user interactions\n- 'subagent': Delegated task workers",
						},
						"subagentType": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"general-purpose"},
							"description": "Subagent specialization type (only present for subagent sessions)",
						},
						"browserMode": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"electron-embedded-browser", "local-browser-service", "remote-cdp-websocket"},
							"description": "Browser automation mode:\n- 'electron-embedded-browser': Electron app with embedded Chromium browser\n- 'local-browser-service': Local browser-service (GoRod-based)\n- 'remote-cdp-websocket': Remote CDP WebSocket URL (cloud browser providers)",
						},
						"cdpUrl": map[string]interface{}{
							"type":        "string",
							"description": "CDP WebSocket URL for remote browser connections (only present when browserMode is 'remote-cdp-websocket')",
							"example":     "wss://connect.browserbase.com/v1/sessions/abc123",
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
							"description": "Total cost of session (for subagent sessions, costs are also accumulated in parent session)",
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
						"callbacks": map[string]interface{}{
							"type":        "array",
							"description": "Session-level callback configurations (optional)",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/Callback",
							},
						},
					},
					"required": []string{"id", "title", "sessionType", "browserMode", "userMessageCount", "assistantMessageCount", "toolCallCount", "promptTokens", "completionTokens", "cost", "createdAt"},
				},
				"MessageData": map[string]interface{}{
					"type":        "object",
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
					"type":        "object",
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
						"callbackResults": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/CallbackResultData",
							},
							"description": "Callback execution results (optional)",
						},
						"reasoning": map[string]interface{}{
							"type":        "string",
							"description": "Reasoning process (optional)",
						},
						"reasoningDuration": map[string]interface{}{
							"type":        "integer",
							"description": "Reasoning duration in milliseconds (optional)",
						},
						"inputTokens": map[string]interface{}{
							"type":        "integer",
							"description": "Input tokens used for this message (includes cache creation)",
						},
						"outputTokens": map[string]interface{}{
							"type":        "integer",
							"description": "Output tokens generated for this message (includes cache reads)",
						},
						"cacheCreationTokens": map[string]interface{}{
							"type":        "integer",
							"description": "Tokens used for prompt cache creation (optional)",
						},
						"cacheReadTokens": map[string]interface{}{
							"type":        "integer",
							"description": "Tokens read from prompt cache (optional)",
						},
						"cost": map[string]interface{}{
							"type":        "number",
							"format":      "double",
							"description": "Cost for this specific message in USD",
						},
						"model": map[string]interface{}{
							"type":        "string",
							"description": "Model used for this message (e.g., 'claude-sonnet-4')",
						},
					},
					"required": []string{"id", "sessionId", "role", "userInput"},
				},
				"ExportSession": map[string]interface{}{
					"type":        "object",
					"description": "Comprehensive session export with all messages, tool calls, and metadata",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Session identifier",
						},
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Session title",
						},
						"userMessageCount": map[string]interface{}{
							"type":        "integer",
							"description": "Number of user messages",
						},
						"assistantMessageCount": map[string]interface{}{
							"type":        "integer",
							"description": "Number of assistant messages",
						},
						"toolCallCount": map[string]interface{}{
							"type":        "integer",
							"description": "Total number of tool calls",
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
						"updatedAt": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Session last update timestamp",
						},
						"messages": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/ExportMessage",
							},
							"description": "Complete list of messages with full details",
						},
					},
					"required": []string{"id", "title", "messages"},
				},
				"ExportMessage": map[string]interface{}{
					"type":        "object",
					"description": "Complete message information for export",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Message identifier",
						},
						"role": map[string]interface{}{
							"type":        "string",
							"description": "Message role (user, assistant, tool)",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Message content",
						},
						"toolCalls": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/ExportToolCall",
							},
							"description": "Tool calls with complete information",
						},
						"reasoning": map[string]interface{}{
							"type":        "string",
							"description": "Reasoning content (optional)",
						},
						"reasoningDuration": map[string]interface{}{
							"type":        "integer",
							"description": "Reasoning duration in milliseconds (optional)",
						},
						"model": map[string]interface{}{
							"type":        "string",
							"description": "Model used for this message (optional)",
						},
						"finishReason": map[string]interface{}{
							"type":        "string",
							"description": "Completion finish reason (optional)",
						},
						"createdAt": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Message creation timestamp",
						},
						"updatedAt": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Message update timestamp",
						},
					},
					"required": []string{"id", "role", "content", "createdAt", "updatedAt"},
				},
				"ExportToolCall": map[string]interface{}{
					"type":        "object",
					"description": "Complete tool call information for export",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Tool call identifier",
						},
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Tool name",
						},
						"input": map[string]interface{}{
							"type":        "string",
							"description": "Tool input as JSON string",
						},
						"inputJson": map[string]interface{}{
							"type":        "object",
							"description": "Parsed tool input (optional)",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Tool type",
						},
						"finished": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether tool execution finished",
						},
						"result": map[string]interface{}{
							"type":        "string",
							"description": "Tool execution result (optional)",
						},
						"screenshotUrls": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Screenshot URLs captured during tool execution (optional)",
						},
					},
					"required": []string{"id", "name", "input", "type", "finished"},
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
						"screenshotUrls": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Screenshot URLs captured during tool execution (optional)",
						},
					},
					"required": []string{"id", "name", "input", "type", "finished"},
				},
				"CallbackResultData": map[string]interface{}{
					"type":        "object",
					"description": "Callback execution result information",
					"properties": map[string]interface{}{
						"tool_call_id": map[string]interface{}{
							"type":        "string",
							"description": "ID of the tool call that triggered this callback",
						},
						"tool_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the tool that triggered callback",
						},
						"callback_name": map[string]interface{}{
							"type":        "string",
							"description": "Human-readable name of the callback (optional)",
						},
						"callback_type": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"bash_script", "sub_agent", "send_message"},
							"description": "Type of callback executed",
						},
						"stdout": map[string]interface{}{
							"type":        "string",
							"description": "Standard output from bash callback (optional)",
						},
						"stderr": map[string]interface{}{
							"type":        "string",
							"description": "Standard error from bash callback (optional)",
						},
						"exit_code": map[string]interface{}{
							"type":        "integer",
							"description": "Exit code from bash callback (optional)",
						},
						"subagent_id": map[string]interface{}{
							"type":        "string",
							"description": "ID of spawned sub-agent session (optional)",
						},
						"subagent_result": map[string]interface{}{
							"type":        "string",
							"description": "Result from sub-agent execution (optional)",
						},
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether callback executed successfully",
						},
						"error": map[string]interface{}{
							"type":        "string",
							"description": "Error message if callback failed (optional)",
						},
						"exclude_from_context": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether this callback result is excluded from agent context (optional)",
						},
					},
					"required": []string{"tool_call_id", "tool_name", "callback_type", "success"},
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
							"enum":        []string{"connected", "heartbeat", "error", "complete", "thinking", "content", "tool_use_start", "tool_use_parameter_streaming_complete", "tool_use_parameter_delta", "tool_execution_start", "tool_execution_complete", "permission", "notification", "user_message_created", "session_created", "session_deleted"},
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
					"type":        "object",
					"description": "Server-Sent Event stream with discriminated event types",
					"discriminator": map[string]interface{}{
						"propertyName": "event",
						"mapping": map[string]interface{}{
							"connected":                             "#/components/schemas/SSEConnectedEvent",
							"heartbeat":                             "#/components/schemas/SSEHeartbeatEvent",
							"error":                                 "#/components/schemas/SSEErrorEvent",
							"complete":                              "#/components/schemas/SSECompleteEvent",
							"thinking":                              "#/components/schemas/SSEThinkingEvent",
							"content":                               "#/components/schemas/SSEContentEvent",
							"tool_use_start":                        "#/components/schemas/SSEToolUseStartEvent",
							"tool_use_parameter_streaming_complete": "#/components/schemas/SSEToolUseParameterStreamingCompleteEvent",
							"tool_use_parameter_delta":              "#/components/schemas/SSEToolUseParameterDeltaEvent",
							"tool_execution_start":                  "#/components/schemas/SSEToolExecutionStartEvent",
							"tool_execution_complete":               "#/components/schemas/SSEToolExecutionCompleteEvent",
							"permission":                            "#/components/schemas/SSEPermissionEvent",
							"notification":                          "#/components/schemas/SSENotificationEvent",
							"user_message_created":                  "#/components/schemas/SSEUserMessageCreatedEvent",
							"session_created":                       "#/components/schemas/SSESessionCreatedEvent",
							"session_deleted":                       "#/components/schemas/SSESessionDeletedEvent",
						},
					},
					"oneOf": []map[string]interface{}{
						{"$ref": "#/components/schemas/SSEConnectedEvent"},
						{"$ref": "#/components/schemas/SSEHeartbeatEvent"},
						{"$ref": "#/components/schemas/SSEErrorEvent"},
						{"$ref": "#/components/schemas/SSECompleteEvent"},
						{"$ref": "#/components/schemas/SSEThinkingEvent"},
						{"$ref": "#/components/schemas/SSEContentEvent"},
						{"$ref": "#/components/schemas/SSEToolUseStartEvent"},
						{"$ref": "#/components/schemas/SSEToolUseParameterStreamingCompleteEvent"},
						{"$ref": "#/components/schemas/SSEToolUseParameterDeltaEvent"},
						{"$ref": "#/components/schemas/SSEToolExecutionStartEvent"},
						{"$ref": "#/components/schemas/SSEToolExecutionCompleteEvent"},
						{"$ref": "#/components/schemas/SSEPermissionEvent"},
						{"$ref": "#/components/schemas/SSENotificationEvent"},
						{"$ref": "#/components/schemas/SSEUserMessageCreatedEvent"},
						{"$ref": "#/components/schemas/SSESessionCreatedEvent"},
						{"$ref": "#/components/schemas/SSESessionDeletedEvent"},
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
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
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
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
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
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
										},
										"assistantMessageId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the assistant message this thinking belongs to",
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
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
										},
										"assistantMessageId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the assistant message this content belongs to",
										},
									},
									"required": []string{"type", "content"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEToolUseStartEvent": map[string]interface{}{
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
											"description": "Tool use start event type",
										},
										"name": map[string]interface{}{
											"$ref":        "#/components/schemas/ToolName",
											"description": "Tool name declared for execution",
										},
										"id": map[string]interface{}{
											"type":        "string",
											"description": "Tool call identifier",
										},
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
										},
										"assistantMessageId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the assistant message this tool belongs to",
										},
									},
									"required": []string{"type", "name", "id"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEToolUseParameterStreamingCompleteEvent": map[string]interface{}{
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
											"description": "Tool use parameter streaming complete event type",
										},
										"name": map[string]interface{}{
											"$ref":        "#/components/schemas/ToolName",
											"description": "Tool name being executed",
										},
										"input": map[string]interface{}{
											"type":        "string",
											"description": "Complete JSON-encoded tool input parameters",
										},
										"id": map[string]interface{}{
											"type":        "string",
											"description": "Tool call identifier",
										},
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
										},
										"assistantMessageId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the assistant message this tool belongs to",
										},
									},
									"required": []string{"type", "name", "input", "id"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEToolUseParameterDeltaEvent": map[string]interface{}{
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
											"description": "Tool use parameter delta event type",
										},
										"toolCallId": map[string]interface{}{
											"type":        "string",
											"description": "Tool call identifier for correlation",
										},
										"input": map[string]interface{}{
											"type":        "string",
											"description": "Partial JSON parameter delta - may not be parseable until complete",
										},
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
										},
										"assistantMessageId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the assistant message this tool parameter delta belongs to",
										},
									},
									"required": []string{"type", "toolCallId", "input"},
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
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
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
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
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
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
										},
									},
									"required": []string{"type", "id", "sessionId", "toolName", "description", "action"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSENotificationEvent": map[string]interface{}{
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
											"description": "Notification event type",
										},
										"id": map[string]interface{}{
											"type":        "string",
											"description": "Notification identifier",
										},
										"sessionId": map[string]interface{}{
											"type":        "string",
											"description": "Session identifier for the notification",
										},
										"notificationType": map[string]interface{}{
											"type":        "string",
											"enum":        []string{"info", "warning", "error", "question"},
											"description": "Type of notification",
										},
										"title": map[string]interface{}{
											"type":        "string",
											"description": "Notification title",
										},
										"message": map[string]interface{}{
											"type":        "string",
											"description": "Notification message content",
										},
										"responseType": map[string]interface{}{
											"type":        "string",
											"enum":        []string{"acknowledge", "text", "choice"},
											"description": "Expected response type from user",
										},
										"choices": map[string]interface{}{
											"type":        "array",
											"items":       map[string]interface{}{"type": "string"},
											"description": "Available choices (required when responseType is 'choice')",
										},
										"timeout": map[string]interface{}{
											"type":        "integer",
											"description": "Timeout in seconds for user response",
										},
										"createdAt": map[string]interface{}{
											"type":        "integer",
											"description": "Unix timestamp when notification was created",
										},
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
										},
									},
									"required": []string{"type", "id", "sessionId", "notificationType", "title", "message", "responseType", "timeout", "createdAt"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSEUserMessageCreatedEvent": map[string]interface{}{
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
											"description": "User message created event type",
											"example":     "user_message_created",
										},
										"messageId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the created user message",
										},
										"content": map[string]interface{}{
											"type":        "string",
											"description": "Content of the user message",
										},
										"parentToolCallId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the parent tool call that spawned this subagent (for nested events)",
										},
									},
									"required": []string{"type", "messageId", "content"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSESessionCreatedEvent": map[string]interface{}{
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
											"description": "Event type",
											"example":     "session_created",
										},
										"sessionId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the newly created session",
										},
										"title": map[string]interface{}{
											"type":        "string",
											"description": "Title of the newly created session",
										},
										"createdAt": map[string]interface{}{
											"type":        "integer",
											"format":      "int64",
											"description": "Unix timestamp when the session was created",
										},
									},
									"required": []string{"type", "sessionId", "title", "createdAt"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				"SSESessionDeletedEvent": map[string]interface{}{
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
											"description": "Event type",
											"example":     "session_deleted",
										},
										"sessionId": map[string]interface{}{
											"type":        "string",
											"description": "ID of the deleted session",
										},
									},
									"required": []string{"type", "sessionId"},
								},
							},
							"required": []string{"data"},
						},
					},
				},
				// New typed response schemas (replacing map[string]interface{} in handlers)
				"StoreToolAPIKeyResponse": map[string]interface{}{
					"type":        "object",
					"description": "Success response when storing a tool API key",
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Operation status",
							"example":     "success",
						},
						"tool_type": map[string]interface{}{
							"type":        "string",
							"description": "Tool type identifier",
							"example":     "web_search",
						},
						"provider": map[string]interface{}{
							"type":        "string",
							"description": "Provider identifier",
							"example":     "brave",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Success message",
							"example":     "Brave Search API key stored successfully",
						},
					},
					"required": []string{"status", "tool_type", "provider", "message"},
				},
				"DeleteToolCredentialResponse": map[string]interface{}{
					"type":        "object",
					"description": "Success response when deleting a tool credential",
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Operation status",
							"example":     "success",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Success message",
							"example":     "Tool credential deleted successfully",
						},
					},
					"required": []string{"status", "message"},
				},
				"SystemHealthResponse": map[string]interface{}{
					"type":        "object",
					"description": "System health check response",
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Overall system status",
							"example":     "ok",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Health check timestamp",
						},
						"version": map[string]interface{}{
							"type":        "string",
							"description": "Application version",
							"example":     "1.0.0",
						},
						"environment": map[string]interface{}{
							"type":        "string",
							"description": "Environment name",
							"example":     "production",
						},
						"services": map[string]interface{}{
							"$ref": "#/components/schemas/HealthServices",
						},
					},
					"required": []string{"status", "timestamp", "version", "environment", "services"},
				},
				"HealthServices": map[string]interface{}{
					"type":        "object",
					"description": "Backend services health status",
					"properties": map[string]interface{}{
						"backend": map[string]interface{}{
							"type":        "string",
							"description": "Backend service status",
							"example":     "healthy",
						},
						"database": map[string]interface{}{
							"type":        "string",
							"description": "Database connection status",
							"example":     "connected",
						},
					},
					"required": []string{"backend", "database"},
				},
				"ActiveTunnelsResponse": map[string]interface{}{
					"type":        "object",
					"description": "Active WebSocket tunnels response",
					"properties": map[string]interface{}{
						"active_tunnels": map[string]interface{}{
							"type":        "array",
							"description": "List of active tunnel session IDs",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of active tunnels",
							"example":     2,
						},
					},
					"required": []string{"active_tunnels", "count"},
				},
				"SetAPIKeyResponse": map[string]interface{}{
					"type":        "object",
					"description": "Success response when setting an API key",
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Operation status",
							"example":     "success",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Success message",
							"example":     "API key set successfully. You can now use the application.",
						},
					},
					"required": []string{"status", "message"},
				},
				"PermissionResponse": map[string]interface{}{
					"type":        "object",
					"description": "Permission operation response",
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Permission status (granted or denied)",
							"enum":        []string{"granted", "denied"},
							"example":     "granted",
						},
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Permission request ID",
							"example":     "perm_abc123",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Status message",
							"example":     "Permission granted successfully",
						},
					},
					"required": []string{"status", "id", "message"},
				},
				"PreferencesWithProviders": map[string]interface{}{
					"type":        "object",
					"description": "User preferences with available provider metadata",
					"properties": map[string]interface{}{
						"preferences": map[string]interface{}{
							"type":        "object",
							"description": "User preferences (null if not set)",
							"nullable":    true,
							"$ref":        "#/components/schemas/UserPreferencesResponse",
						},
						"available_providers": map[string]interface{}{
							"type":        "object",
							"description": "Available LLM providers with their metadata",
							"additionalProperties": map[string]interface{}{
								"$ref": "#/components/schemas/ProviderInfo",
							},
						},
					},
					"required": []string{"preferences", "available_providers"},
				},
				"ProviderInfo": map[string]interface{}{
					"type":        "object",
					"description": "LLM provider metadata",
					"properties": map[string]interface{}{
						"display_name": map[string]interface{}{
							"type":        "string",
							"description": "Provider display name",
							"example":     "Anthropic",
						},
						"models": map[string]interface{}{
							"type":        "object",
							"description": "Available models for this provider (dynamic structure)",
						},
					},
					"required": []string{"display_name", "models"},
				},
				"UserPreferencesResponse": getUserPreferencesResponseSchema(),
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

func getUserPreferencesResponseSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": "User preferences configuration",
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
	}
}

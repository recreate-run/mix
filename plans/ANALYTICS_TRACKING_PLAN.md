# Analytics Event Tracking Plan

## Executive Summary
This document outlines comprehensive event tracking for Mix Agent to improve product analytics, user behavior understanding, and feature usage monitoring.

## Currently Tracked Events ✅

### 1. Message Events
- **`user_message`** - Every user prompt/input
  - Properties: `session_id`, `message_id`, `content`, `model`, `content_length`, `is_truncated`
  - Location: `message/tracking_service.go:44`

- **`agent_response`** - Every assistant response
  - Properties: `session_id`, `message_id`, `content`, `model`, `provider`
  - Enhanced for OpenRouter: `thinking_enabled`, `thinking_length`, `response_time_ms`, `token_usage_*`, `cost`
  - Location: `message/tracking_service.go:58`, `message/tracking_service.go:134`

### 2. Tool Events
- **`tool_call`** - Every tool execution
  - Properties: `session_id`, `message_id`, `tool_name`, `tool_input`, `tool_id`, `success`, `error`
  - Location: `message/tracking_service.go:109`, `message/tracking_service.go:124`

### 3. Authentication Events
- **`provider_auth`** - Authentication events for providers
  - Properties: `provider`, `success`, `auth_method` (api_key, oauth, delete_credentials)
  - Locations:
    - API Key: `rest_auth.go:96,108,116`
    - OAuth: `rest_auth.go:381,392,409,419,441,460`
    - Deletion: `rest_auth.go:168,176`
    - Tool credentials: `rest_tools.go:128,136`

### 4. Frontend Events
- **`mix_playground_initialized`** - App startup
  - Properties: `version`, `timestamp`, `app_type`, `app_platform`, `app_version`
  - Location: `mix_playground/src/lib/posthog.ts:43`

## Missing Events to Track 🚨

### 1. Session Lifecycle Events

#### `session_created`
- **When**: User creates a new session
- **Location**: `rest_sessions.go:184`
- **Properties**:
  - `session_id`
  - `title`
  - `has_custom_prompt` (boolean)
  - `prompt_mode` (default, append, replace)
  - `custom_prompt_length` (if applicable)

#### `session_deleted`
- **When**: User deletes a session
- **Location**: `rest_sessions.go:294`
- **Properties**:
  - `session_id`
  - `session_age_seconds` (created_at to deletion time)
  - `message_count`
  - `total_cost`


#### `session_rewound`
- **When**: User rewinds session to a previous message
- **Location**: `rest_sessions.go:377`
- **Properties**:
  - `session_id`
  - `rewind_to_message_id`
  - `messages_deleted_count`
  - `cleanup_media` (boolean)

### 2. Configuration & Preferences Events

#### `preferences_updated`
- **When**: User updates their preferences
- **Location**: `rest_preferences.go:245-278`
- **Properties**:
  - `fields_changed` (array: e.g., ["preferred_provider", "main_agent_model"])
  - `preferred_provider` (if changed)
  - `main_agent_model` (if changed)
  - `main_agent_max_tokens` (if changed)
  - `main_agent_reasoning_effort` (if changed)
  - `sub_agent_model` (if changed)
  - `sub_agent_max_tokens` (if changed)
  - `sub_agent_reasoning_effort` (if changed)

#### `preferences_reset`
- **When**: User resets preferences to defaults
- **Location**: `rest_preferences.go:342`
- **Properties**:
  - `previous_provider`
  - `previous_main_model`
  - `reset_to_defaults` (boolean: true)

### 3. File Operations Events

#### `file_uploaded`
- **When**: User uploads a file to session
- **Location**: `rest_files.go:69`
- **Properties**:
  - `session_id`
  - `file_size_bytes`
  - `file_type` (extension)
  - `file_name_sanitized` (boolean - was filename modified)
  - `is_media` (boolean - image/video)

#### `file_deleted`
- **When**: User deletes a file from session
- **Location**: `rest_files.go:217`
- **Properties**:
  - `session_id`
  - `file_name`
  - `file_existed` (boolean)

### 4. Export & Download Events

#### `session_exported`
- **When**: User exports session transcript
- **Location**: `rest_messages.go:382`
- **Properties**:
  - `session_id`
  - `message_count`
  - `export_format` (json)
  - `session_age_seconds`
  - `total_cost`
  - `total_tokens`

#### `video_exported`
- **When**: User exports animated video
- **Location**: `gsap_animations/export.go:73`
- **Properties**:
  - `url`
  - `fps`
  - `aspect_ratio`
  - `height`
  - `duration`
  - `uploaded_to_s3` (boolean)
  - `export_duration_ms`

### 5. Agent Control Events

#### `agent_cancelled`
- **When**: User cancels ongoing agent processing
- **Location**: `rest_messages.go:371`
- **Properties**:
  - `session_id`
  - `was_processing` (boolean)
  - `cancel_reason` (user_initiated)

### 6. Permission Events

#### `permission_granted`
- **When**: User grants a permission
- **Location**: `rest_system.go:302`
- **Properties**:
  - `permission_id`
  - `permission_type`
  - `session_id` (if applicable)

#### `permission_denied`
- **When**: User denies a permission
- **Location**: `rest_system.go:332`
- **Properties**:
  - `permission_id`
  - `permission_type`
  - `session_id` (if applicable)

### 7. Tool Credential Events

#### `tool_credential_deleted`
- **When**: User deletes tool credentials
- **Location**: `rest_tools.go:195`
- **Properties**:
  - `tool_type`
  - `provider`
  - `had_credentials` (boolean)

### 8. Error & Failure Events

#### `api_error`
- **When**: API errors occur
- **Locations**: Various error handlers
- **Properties**:
  - `error_type` (validation, internal, not_found, unauthorized)
  - `error_code` (e.g., INVALID_JSON, MISSING_PROVIDER)
  - `endpoint`
  - `http_method`
  - `http_status`
  - `session_id` (if applicable)

#### `export_error`
- **When**: Export operations fail
- **Properties**:
  - `export_type` (session, video)
  - `error_message`
  - `session_id` (if applicable)

#### `file_upload_error`
- **When**: File upload fails
- **Properties**:
  - `session_id`
  - `error_reason`
  - `file_size_bytes`
  - `file_type`

### 9. Performance & Usage Metrics

#### `session_duration_completed`
- **When**: Session is deleted or considered complete
- **Properties**:
  - `session_id`
  - `duration_seconds` (created_at to completion)
  - `message_count`
  - `tool_call_count`
  - `total_cost`
  - `total_tokens_used`

#### `feature_usage`
- **When**: Specific features are used
- **Properties**:
  - `feature_name` (custom_system_prompt, mcp_server, etc.)
  - `session_id`
  - `feature_config` (relevant configuration)

### 10. MCP Server Events

#### `mcp_server_connected`
- **When**: MCP server is successfully connected
- **Properties**:
  - `server_name`
  - `server_type`
  - `tool_count`

#### `mcp_server_failed`
- **When**: MCP server connection fails
- **Properties**:
  - `server_name`
  - `error_message`

## Implementation Priority

### Phase 1 (High Priority) 🔴
1. **Session Lifecycle Events** - Critical for understanding user engagement
   - `session_created`
   - `session_deleted`
   - `session_forked`
   - `session_rewound`

2. **Error Events** - Essential for debugging and reliability
   - `api_error`
   - `export_error`
   - `file_upload_error`

### Phase 2 (Medium Priority) 🟡
3. **File Operations** - Important for feature usage
   - `file_uploaded`
   - `file_deleted`

4. **Export Events** - Key feature tracking
   - `session_exported`
   - `video_exported`

5. **Agent Control** - User interaction tracking
   - `agent_cancelled`

### Phase 3 (Lower Priority) 🟢
6. **Preferences** - Configuration tracking
   - `preferences_updated`
   - `preferences_reset`

7. **Permissions** - Security & UX tracking
   - `permission_granted`
   - `permission_denied`

8. **Tool Credentials** - Completion of auth tracking
   - `tool_credential_deleted`

9. **MCP Server Events** - Advanced feature tracking
   - `mcp_server_connected`
   - `mcp_server_failed`

10. **Performance Metrics** - Deep analytics
    - `session_duration_completed`
    - `feature_usage`

## Analytics Service Updates Needed

### 1. Add New Event Types to Analytics Service
Location: `mix_agent/internal/analytics/analytics.go`

```go
const (
    // Existing events...
    EventUserMessage    = "user_message"
    EventAgentResponse  = "agent_response"
    EventToolCall       = "tool_call"
    EventProviderAuth   = "provider_auth"

    // New session events
    EventSessionCreated = "session_created"
    EventSessionDeleted = "session_deleted"
    EventSessionForked  = "session_forked"
    EventSessionRewound = "session_rewound"

    // File events
    EventFileUploaded   = "file_uploaded"
    EventFileDeleted    = "file_deleted"

    // Export events
    EventSessionExported = "session_exported"
    EventVideoExported   = "video_exported"

    // Control events
    EventAgentCancelled  = "agent_cancelled"

    // Permission events
    EventPermissionGranted = "permission_granted"
    EventPermissionDenied  = "permission_denied"

    // Error events
    EventAPIError        = "api_error"
    EventExportError     = "export_error"
    EventFileUploadError = "file_upload_error"

    // Config events
    EventPreferencesUpdated = "preferences_updated"
    EventPreferencesReset   = "preferences_reset"
)
```

### 2. Add New Tracking Methods to Service Interface

```go
type Service interface {
    // Existing methods...

    // Session lifecycle
    TrackSessionCreated(ctx context.Context, sessionID, title string, hasCustomPrompt bool, promptMode string) error
    TrackSessionDeleted(ctx context.Context, sessionID string, ageSeconds int64, messageCount int, cost float64) error
    TrackSessionForked(ctx context.Context, sourceSessionID, newSessionID string, messageIndex int, messagesCopied int) error
    TrackSessionRewound(ctx context.Context, sessionID, messageID string, messagesDeleted int, cleanupMedia bool) error

    // File operations
    TrackFileUploaded(ctx context.Context, sessionID string, fileSizeBytes int64, fileType string, sanitized bool) error
    TrackFileDeleted(ctx context.Context, sessionID, fileName string, existed bool) error

    // Export
    TrackSessionExported(ctx context.Context, sessionID string, messageCount int, cost float64, totalTokens int64) error
    TrackVideoExported(ctx context.Context, url string, fps int, aspectRatio string, duration float64, uploadedToS3 bool) error

    // Control
    TrackAgentCancelled(ctx context.Context, sessionID string) error

    // Permissions
    TrackPermissionGranted(ctx context.Context, permissionID, permissionType string) error
    TrackPermissionDenied(ctx context.Context, permissionID, permissionType string) error

    // Errors
    TrackAPIError(ctx context.Context, errorType, errorCode, endpoint, httpMethod string, httpStatus int) error

    // Config
    TrackPreferencesUpdated(ctx context.Context, fieldsChanged []string, updates map[string]interface{}) error
    TrackPreferencesReset(ctx context.Context, previousProvider, previousModel string) error
}
```

## Expected Benefits

1. **Product Development**
   - Understand which features are most used
   - Identify pain points and friction
   - Prioritize feature development

2. **User Behavior**
   - Session patterns (length, message count, cost)
   - Export and file usage patterns
   - Model and provider preferences

3. **Performance & Reliability**
   - Error tracking and debugging
   - API reliability metrics
   - Export success rates

4. **Business Metrics**
   - Cost tracking per session
   - Feature adoption rates
   - User retention indicators

## Privacy & Compliance

- All content is truncated to 10k characters
- No PII tracking (using anonymous user IDs)
- Error messages sanitized
- Opt-out capability via `USER_NAME` env var check

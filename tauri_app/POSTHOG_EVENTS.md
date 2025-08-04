# PostHog Analytics Events

This document lists all the analytics events tracked in the Recreate application using PostHog.

## Existing Events

### Application Events
- `tauri_app_initialized` - Fired when the app starts up
  - Properties: `version`, `timestamp`

### Chat Events
- `message_submitted` - User sends a message
  - Properties: `message_length`, `message_content`, `has_media`, `media_count`, `session_id`, `has_file_references`, `timestamp`
  
- `response_received` - AI response received
  - Properties: `session_id`, `response_length`, `response_content`, `processing_time_ms`, `tool_count`, `timestamp`

- `tools_used` - When AI tools are executed
  - Properties: `session_id`, `tool_count`, `tools`, `tool_details`, `message_response_length`

- `error_occurred` - When errors happen during chat
  - Properties: `session_id`, `error_message`, `error_type`, `last_user_message`, `timestamp`, `tools_in_progress`

- `session_created` - When new chat sessions are created
  - Properties: `session_id`, `creation_time`, `previous_messages_count`, `client_id`

## New Events

### File/Folder Interactions
- `folder_selected` - User selects a parent folder
  - Properties: `folder_path`, `timestamp`
  
- `folder_cleared` - User clears folder selection
  - Properties: `timestamp`
  
- `file_referenced` - User references a file in a message
  - Properties: `file_path`, `file_name`, `is_directory`, `timestamp`
  
- `file_navigation` - User navigates between folders
  - Properties: `action` (enter_folder/navigate_back/navigate_to_root), `folder_path`, `folder_name`, `items_count`, `from_folder`, `to_folder`, `timestamp`
  
- `file_attachment_added` - User adds a file attachment
  - Properties: `file_type` (file/folder), `file_path`, `file_name`, `file_extension`, `timestamp`
  
- `file_attachment_removed` - User removes a file attachment
  - Properties: `file_path`, `file_name`, `file_type` (file/folder), `timestamp`

### Application Interactions
- `app_referenced` - User references an app in a message
  - Properties: `app_name`, `app_id`, `timestamp`
  
- `app_attachment_added` - User attaches an app
  - Properties: `app_name`, `app_id`, `timestamp`
  
- `app_attachment_removed` - User removes an app attachment
  - Properties: `app_name`, `app_id`, `timestamp`

### UI Interactions
- `slash_command_opened` - Slash command menu is opened
  - Properties: `timestamp`
  
- `slash_command_executed` - User executes a slash command
  - Properties: `command_id`, `command_name`, `timestamp`
  
- `command_menu_opened` - Command menu is opened
  - Properties: `trigger_method`, `timestamp`
  
- `command_menu_closed` - Command menu is closed
  - Properties: `method` (escape_key/close_button), `timestamp`
  
- `command_menu_navigation` - User navigates in command menu
  - Properties: `action` (back_to_commands/view_permissions), `from`, `method`, `timestamp`
  
- `command_executed` - User executes a command
  - Properties: `command`, `timestamp`

### Message History
- `history_navigation` - User navigates message history
  - Properties: `direction` (up/down), `method`, `timestamp`
  
- `session_started` - New session started
  - Properties: `timestamp`
  
- `session_duration` - Track session duration
  - Properties: `duration_ms`, `timestamp`

### Permission Interactions
- `permission_requested` - User requests a system permission
  - Properties: `permission_type`, `permission_label`, `timestamp`

### Performance Metrics
- `response_latency` - Measure response time
  - Properties: `response_time_ms`, `session_id`, `tool_count`, `response_length`, `timestamp`
  
- `tool_execution_time` - Measure tool execution time
  - Properties: `tool_name`, `tool_id`, `execution_time_ms`, `status`, `has_error`, `session_id`, `timestamp`
  
- `app_load_time` - Measure app startup time
  - Properties: `load_time_ms`, `timestamp`

## How to Add New Events

To add a new event to track:

1. Import the tracking function:
   ```typescript
   import { safeTrackEvent } from '@/lib/posthog';
   ```

2. Track an event:
   ```typescript
   safeTrackEvent('event_name', {
     property1: value1,
     property2: value2,
     timestamp: new Date().toISOString()
   });
   ```

3. Add the event to this documentation with a description and its properties.
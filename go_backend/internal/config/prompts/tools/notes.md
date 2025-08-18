Automates macOS Notes app operations via AppleScript integration.

This tool provides programmatic access to the Notes app for reading and managing notes content.

## Prerequisites

- macOS Notes app must be installed and running
- The tool uses AppleScript automation 
- Requires appropriate system permissions for application automation
- A note must be selected in the Notes app for most operations

## Available Operations

### Note Access Operations

**get_current_note**
- Gets information about the currently selected note in Notes app
- Args: None
- Returns: Complete note information including content, metadata, and properties

**get_current_note_html**
- Gets only the HTML body content of the currently selected note
- Args: None  
- Returns: Raw HTML string including embedded images and formatting

## Available Data

The tool returns comprehensive note information:

- **id**: Unique identifier of the note (x-coredata URL format)
- **name**: The note title (normally the first line of content)
- **body**: Full HTML content of the note
- **plaintext**: Plain text version of the note content
- **creation_date**: When the note was created
- **modification_date**: When the note was last modified  
- **password_protected**: Whether the note is password protected
- **shared**: Whether the note is shared with others
- **container**: The folder/account containing the note

## Usage Examples

```json
// Get currently selected note with full metadata
{"operation": "get_current_note"}

// Get only HTML content of current note
{"operation": "get_current_note_html"}
```

## Response Format

**get_current_note response:**
```json
{
  "id": "x-coredata://BBD93077-5E69-4BE5-BC04-63CA76D6AF64/ICNote/p797",
  "name": "My Important Note",
  "body": "<div>HTML formatted content...</div>",
  "plaintext": "Plain text content of the note...",
  "creation_date": "2025-07-26T10:30:00Z",
  "modification_date": "2025-07-26T11:45:00Z", 
  "password_protected": false,
  "shared": false,
  "container": "General"
}
```

**get_current_note_html response:**
```html
<div><h1>My Note Title</h1></div>
<div><br></div>
<div>Note content with <strong>formatting</strong></div>
<div><img src="data:image/jpeg;base64,/9j/4AAQ..."/></div>
```

## Typical Workflow

1. **Select** → Choose a note in the Notes app
2. **Read** → Use get_current_note for full metadata or get_current_note_html for just HTML content
3. **Process** → Analyze or work with the note content as needed

## Important Notes

- Notes app must be running and have a note selected
- The tool accesses the currently selected note (first item in selection)
- Follows fail-fast error handling - exceptions propagate immediately
- Operations are read-only - no note modification capabilities currently
- Date parsing may vary based on system locale settings
- HTML body content preserves formatting and embedded content
- get_current_note_html returns raw HTML including base64-encoded images
- get_current_note provides structured metadata along with content

## Error Handling

Common error patterns:
- "No note is currently selected" - No note selected in Notes app
- "Notes got an error" - Notes app communication failure
- "Invalid note info response" - Unexpected data format from AppleScript
- "AppleScript execution failed" - System-level automation issue

All errors include descriptive messages to help with debugging and resolution.

## Security Considerations

- Tool requires macOS Automation permissions for Notes app access
- Only reads existing note content - does not create or modify notes
- Respects Notes app privacy settings and restrictions
- Permission requests are logged and user-controlled
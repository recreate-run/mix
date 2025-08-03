package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"mix/internal/permission"
	"mix/internal/utils"
)

type NotesParams struct {
	Operation string      `json:"operation"`
	Args      interface{} `json:"args"`
}

type NoteInfo struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Body              string    `json:"body"`
	Plaintext         string    `json:"plaintext"`
	CreationDate      time.Time `json:"creation_date"`
	ModificationDate  time.Time `json:"modification_date"`
	PasswordProtected bool      `json:"password_protected"`
	Shared            bool      `json:"shared"`
	Container         string    `json:"container"`
}

type notesTool struct {
	permissions permission.Service
}

const (
	NotesToolName = "notes"
)

func notesDescription() string {
	return LoadToolDescription("notes")
}

func NewNotesTool(permission permission.Service, bashTool BaseTool) BaseTool {
	return &notesTool{
		permissions: permission,
	}
}

func (n *notesTool) Info() ToolInfo {
	return ToolInfo{
		Name:        NotesToolName,
		Description: notesDescription(),
		Parameters: map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "The operation to perform (get_current_note, get_current_note_html)",
			},
			"args": map[string]any{
				"type":        "object",
				"description": "Operation-specific arguments",
			},
		},
		Required: []string{"operation"},
	}
}

func (n *notesTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params NotesParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}

	if params.Operation == "" {
		return NewTextErrorResponse("missing operation"), nil
	}

	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return ToolResponse{}, fmt.Errorf("session ID and message ID are required for notes operations")
	}

	workingDir, err := GetWorkingDirectory(ctx)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	
	granted := n.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        workingDir,
			ToolName:    NotesToolName,
			Action:      params.Operation,
			Description: fmt.Sprintf("Execute Notes operation: %s", params.Operation),
			Params:      params,
		},
	)
	if !granted {
		return ToolResponse{}, permission.ErrorPermissionDenied
	}

	var result interface{}

	switch params.Operation {
	case "get_current_note":
		result, err = n.getCurrentNote(ctx)
	case "get_current_note_html":
		result, err = n.getCurrentNoteHTML(ctx)
	default:
		return NewTextErrorResponse(fmt.Sprintf("unknown operation: %s", params.Operation)), nil
	}

	if err != nil {
		return ToolResponse{}, fmt.Errorf("notes operation failed: %w", err)
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return NewTextErrorResponse("failed to serialize result"), nil
	}

	return NewTextResponse(string(resultJSON)), nil
}

func (n *notesTool) getCurrentNote(ctx context.Context) (*NoteInfo, error) {
	// First, get the selection
	script := `tell application "Notes"
		set selectedNotes to selection
		if (count of selectedNotes) > 0 then
			set currentNote to item 1 of selectedNotes
			set noteContainer to container of currentNote
			return {name of currentNote, plaintext of currentNote, body of currentNote, id of currentNote, creation date of currentNote, modification date of currentNote, password protected of currentNote, shared of currentNote, name of noteContainer}
		else
			error "No note selected"
		end if
	end tell`

	result, err := utils.ExecuteAppleScript(ctx, script)
	if err != nil {
		// Log the actual AppleScript error for debugging
		log.Printf("[Notes Tool] AppleScript error in getCurrentNote: %v", err)

		if strings.Contains(err.Error(), "No note selected") {
			return nil, fmt.Errorf("no note is currently selected in Notes app")
		}
		// Return the original error with context
		return nil, fmt.Errorf("failed to get current note from Notes app: %w", err)
	}

	// Parse the comma-separated result
	// AppleScript returns: name, plaintext, body, id, creation_date, modification_date, password_protected, shared, container
	parts := strings.Split(result, ", ")
	if len(parts) < 9 {
		return nil, fmt.Errorf("invalid note info response: expected 9 parts, got %d", len(parts))
	}

	// Parse dates (AppleScript format: "date \"Tuesday, July 26, 2025 at 10:30:00 AM\"")
	creationDate, _ := time.Parse("Monday, January 2, 2006 at 3:04:05 PM", strings.Trim(parts[4], "date \""))
	modificationDate, _ := time.Parse("Monday, January 2, 2006 at 3:04:05 PM", strings.Trim(parts[5], "date \""))

	// Parse boolean values
	passwordProtected := parts[6] == "true"
	shared := parts[7] == "true"

	return &NoteInfo{
		ID:                parts[3],
		Name:              parts[0],
		Plaintext:         parts[1],
		Body:              parts[2],
		CreationDate:      creationDate,
		ModificationDate:  modificationDate,
		PasswordProtected: passwordProtected,
		Shared:            shared,
		Container:         parts[8],
	}, nil
}

func (n *notesTool) getCurrentNoteHTML(ctx context.Context) (string, error) {
	// Get only the HTML body content
	script := `tell application "Notes"
		set selectedNotes to selection
		if (count of selectedNotes) > 0 then
			set currentNote to item 1 of selectedNotes
			return body of currentNote
		else
			error "No note selected"
		end if
	end tell`

	result, err := utils.ExecuteAppleScript(ctx, script)
	if err != nil {
		// Log the actual AppleScript error for debugging
		log.Printf("[Notes Tool] AppleScript error in getCurrentNoteHTML: %v", err)

		if strings.Contains(err.Error(), "No note selected") {
			return "", fmt.Errorf("no note is currently selected in Notes app")
		}
		// Return the original error with context
		return "", fmt.Errorf("failed to get current note HTML from Notes app: %w", err)
	}

	return result, nil
}

// parseArgs is a helper function to parse arguments into the appropriate struct
func (n *notesTool) parseArgs(args interface{}, target interface{}) error {
	if args == nil {
		return nil
	}

	argBytes, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	if err := json.Unmarshal(argBytes, target); err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	return nil
}

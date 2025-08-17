package permission

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mix/internal/config"
	"mix/internal/logging"
	"mix/internal/pubsub"
	"mix/internal/session"

	"github.com/google/uuid"
)

var ErrorPermissionDenied = errors.New("permission denied")

type CreatePermissionRequest struct {
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type Service interface {
	pubsub.Suscriber[PermissionRequest]
	GrantPersistant(permission PermissionRequest)
	Grant(permission PermissionRequest)
	Deny(permission PermissionRequest)
	Request(opts CreatePermissionRequest) bool
}

type permissionService struct {
	*pubsub.Broker[PermissionRequest]

	sessionPermissions []PermissionRequest
	pendingRequests    sync.Map
	sessions          session.Service
}

func (s *permissionService) GrantPersistant(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- true
	}
	s.sessionPermissions = append(s.sessionPermissions, permission)
}

func (s *permissionService) Grant(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- true
	}
}

func (s *permissionService) Deny(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- false
	}
}

// isPathWithinSessionRoot checks if the given path is accessible within the session working directory using os.Root
func (s *permissionService) isPathWithinSessionRoot(sessionID, requestedPath string) bool {
	// Get session working directory
	sess, err := s.sessions.Get(context.Background(), sessionID)
	if err != nil {
		logging.Error("Failed to get session", "sessionID", sessionID, "error", err)
		return false
	}

	if sess.WorkingDirectory == "" {
		logging.Info("Session has no working directory", "sessionID", sessionID)
		return false
	}

	// Clean and make absolute paths for comparison
	absSessionDir, err := filepath.Abs(filepath.Clean(sess.WorkingDirectory))
	if err != nil {
		logging.Error("Failed to get absolute path for session working dir", "workingDirectory", sess.WorkingDirectory, "error", err)
		return false
	}

	absRequestedPath, err := filepath.Abs(filepath.Clean(requestedPath))
	if err != nil {
		logging.Error("Failed to get absolute path for requested path", "requestedPath", requestedPath, "error", err)
		return false
	}

	// Calculate relative path from session directory to requested path
	relPath, err := filepath.Rel(absSessionDir, absRequestedPath)
	if err != nil {
		logging.Debug("Failed to calculate relative path", "sessionDir", absSessionDir, "requestedPath", absRequestedPath, "error", err)
		return false
	}

	// If relative path starts with "..", then absRequestedPath is outside absSessionDir
	if filepath.IsAbs(relPath) || relPath == ".." || filepath.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		logging.Debug("Path is outside session working directory", "relPath", relPath)
		return false
	}

	// Create root filesystem view for session working directory
	rootFS, err := os.OpenRoot(sess.WorkingDirectory)
	if err != nil {
		logging.Error("Failed to create root filesystem for session directory", "workingDirectory", sess.WorkingDirectory, "error", err)
		return false
	}
	defer rootFS.Close()

	// Try to access the requested path through the root using relative path
	// This will fail if the path involves path traversal or doesn't exist
	_, err = rootFS.Stat(relPath)
	if err != nil {
		logging.Debug("Path not accessible within session root", "relPath", relPath, "error", err)
		return false
	}

	return true // Path is accessible within session working directory
}


func (s *permissionService) Request(opts CreatePermissionRequest) bool {
	logging.Info("Permission request", "sessionID", opts.SessionID, "toolName", opts.ToolName, "action", opts.Action, "path", opts.Path)

	dir := filepath.Dir(opts.Path)
	if dir == "." {
		// Get session working directory for relative paths
		sess, err := s.sessions.Get(context.Background(), opts.SessionID)
		if err != nil {
			logging.Error("Failed to get session for relative path resolution", "sessionID", opts.SessionID, "error", err)
			return false // Deny if we can't get session info
		}
		if sess.WorkingDirectory == "" {
			logging.Error("Session has no working directory for relative path resolution", "sessionID", opts.SessionID)
			return false // Deny if no working directory set
		}
		dir = sess.WorkingDirectory
	}

	// Check if path is within session working directory using os.Root
	if s.isPathWithinSessionRoot(opts.SessionID, dir) {
		// Path is within session working directory
		if config.Get().SkipPermissions {
			logging.Info("Path is within session working directory, permissions skipped", "path", dir)
			return true
		}
		// Still require permission even within session directory if not skipped
		logging.Info("Path is within session working directory, requesting permission", "path", dir)
	} else {
		// Path is outside session working directory - always require permission
		logging.Info("Path is outside session working directory, requiring permission", "path", dir)
		// Continue to permission request flow below
	}
	permission := PermissionRequest{
		ID:          uuid.New().String(),
		Path:        dir,
		SessionID:   opts.SessionID,
		ToolName:    opts.ToolName,
		Description: opts.Description,
		Action:      opts.Action,
		Params:      opts.Params,
	}

	for _, p := range s.sessionPermissions {
		if p.ToolName == permission.ToolName && p.Action == permission.Action && p.SessionID == permission.SessionID && p.Path == permission.Path {
			logging.Info("Found existing permission", "toolName", permission.ToolName, "action", permission.Action, "sessionID", permission.SessionID)
			return true
		}
	}

	respCh := make(chan bool, 1)

	s.pendingRequests.Store(permission.ID, respCh)
	defer s.pendingRequests.Delete(permission.ID)

	logging.Info("Publishing permission request for approval", "permissionID", permission.ID)
	fmt.Printf("PERMISSION: Publishing event to %d subscribers\n", s.GetSubscriberCount())
	if err := s.Publish(context.Background(), pubsub.CreatedEvent, permission); err != nil {
		logging.Error("Failed to publish permission request", "permissionID", permission.ID, "error", err)
		return false
	}
	fmt.Printf("PERMISSION: Event published successfully\n")

	// Wait for the response with a timeout (30 seconds)
	select {
	case resp := <-respCh:
		logging.Info("Permission responded", "permissionID", permission.ID, "approved", resp)
		return resp
	case <-time.After(30 * time.Second):
		logging.Info("Permission request timed out after 30 seconds, denying", "permissionID", permission.ID)
		return false
	}
}

func NewPermissionService(sessions session.Service) Service {
	return &permissionService{
		Broker:             pubsub.NewBroker[PermissionRequest](),
		sessionPermissions: make([]PermissionRequest, 0),
		sessions:          sessions,
	}
}

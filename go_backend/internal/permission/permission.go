package permission

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"mix/internal/config"
	"mix/internal/pubsub"

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

func (s *permissionService) Request(opts CreatePermissionRequest) bool {
	log.Printf("Permission request: SessionID=%s, ToolName=%s, Action=%s, Path=%s",
		opts.SessionID, opts.ToolName, opts.Action, opts.Path)

	if config.Get().SkipPermissions {
		log.Printf("Permissions globally skipped via --dangerously-skip-permissions flag")
		return true
	}

	dir := filepath.Dir(opts.Path)
	if dir == "." {
		var err error
		dir, err = config.LaunchDirectory()
		if err != nil {
			panic(fmt.Sprintf("failed to get launch directory for permission check: %v", err))
		}
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
			log.Printf("Found existing permission for %s:%s in session %s", permission.ToolName, permission.Action, permission.SessionID)
			return true
		}
	}

	respCh := make(chan bool, 1)

	s.pendingRequests.Store(permission.ID, respCh)
	defer s.pendingRequests.Delete(permission.ID)

	log.Printf("Publishing permission request %s for approval", permission.ID)
	if err := s.Publish(context.Background(), pubsub.CreatedEvent, permission); err != nil {
		log.Printf("Failed to publish permission request %s: %v", permission.ID, err)
		return false
	}

	// Wait for the response with a timeout (30 seconds)
	select {
	case resp := <-respCh:
		log.Printf("Permission %s responded: %t", permission.ID, resp)
		return resp
	case <-time.After(30 * time.Second):
		log.Printf("Permission request %s timed out after 30 seconds, denying", permission.ID)
		return false
	}
}

func NewPermissionService() Service {
	return &permissionService{
		Broker:             pubsub.NewBroker[PermissionRequest](),
		sessionPermissions: make([]PermissionRequest, 0),
	}
}

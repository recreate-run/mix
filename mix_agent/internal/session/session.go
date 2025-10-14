package session

import (
	"context"
	"database/sql"
	"fmt"

	"mix/internal/db"
	"mix/internal/llm/tools/shell"
	"mix/internal/pubsub"

	"github.com/google/uuid"
)

// SessionType represents the type/category of a session
type SessionType string

const (
	SessionTypeMain     SessionType = "main"     // Regular user session
	SessionTypeSubagent SessionType = "subagent" // Task tool worker session
	SessionTypeForked   SessionType = "forked"   // User-forked session
)

// String returns the string representation of SessionType
func (s SessionType) String() string {
	return string(s)
}

// IsValidSessionType checks if the given string is a valid SessionType
func IsValidSessionType(s string) bool {
	switch SessionType(s) {
	case SessionTypeMain, SessionTypeSubagent, SessionTypeForked:
		return true
	default:
		return false
	}
}

// SubagentType represents the specialization of a subagent session
type SubagentType string

const (
	SubagentTypeGeneralPurpose SubagentType = "general-purpose" // Full toolset subagent
	// Future types can be added here: research, code-review, etc.
)

// String returns the string representation of SubagentType
func (s SubagentType) String() string {
	return string(s)
}

// IsValidSubagentType checks if the given string is a valid SubagentType
func IsValidSubagentType(s string) bool {
	switch SubagentType(s) {
	case SubagentTypeGeneralPurpose:
		return true
	default:
		return false
	}
}

// Context key for suppressing session event publishing (used for internal/sub-agent sessions)
type contextKey string

const suppressPublishKey contextKey = "suppress_session_publish"

// WithSuppressPublish returns a context that suppresses session event publishing
func WithSuppressPublish(ctx context.Context) context.Context {
	return context.WithValue(ctx, suppressPublishKey, true)
}

// shouldPublish checks if session events should be published based on context
func shouldPublish(ctx context.Context) bool {
	suppress, _ := ctx.Value(suppressPublishKey).(bool)
	return !suppress
}

type Session struct {
	ID                    string
	ParentSessionID       string
	Title                 string
	UserMessageCount      int64
	AssistantMessageCount int64
	ToolCallCount         int64
	PromptTokens          int64
	CompletionTokens      int64
	SummaryMessageID      string
	CustomSystemPrompt    string
	PromptMode            string
	SessionType           SessionType  // Type-safe session category
	SubagentType          SubagentType // Type-safe subagent specialization
	Cost                  float64
	CreatedAt             int64
	UpdatedAt             int64
}

// Simplified Service interface for embedded binary
type Service interface {
	pubsub.Suscriber[Session]
	Create(ctx context.Context, title string, customSystemPrompt string, promptMode string, sessionType SessionType, subagentType SubagentType, parentSessionID string) (Session, error)
	Fork(ctx context.Context, sourceSessionID string, title string) (Session, error)
	Get(ctx context.Context, id string) (Session, error)
	List(ctx context.Context) ([]Session, error)
	ListWithContent(ctx context.Context) ([]db.ListSessionsWithContentRow, error)
	Save(ctx context.Context, session Session) (Session, error)
	IncrementCost(ctx context.Context, sessionID string, costDelta float64) error
	Delete(ctx context.Context, id string) error
}

type service struct {
	*pubsub.Broker[Session]
	q             db.Querier
	storageConfig Config
}

func (s *service) Create(ctx context.Context, title string, customSystemPrompt string, promptMode string, sessionType SessionType, subagentType SubagentType, parentSessionID string) (Session, error) {
	// Default to 'main' session type if not specified
	if sessionType == "" {
		sessionType = SessionTypeMain
	}

	// Validate subagent type if specified
	if subagentType != "" && !IsValidSubagentType(subagentType.String()) {
		return Session{}, fmt.Errorf("invalid subagent type: %s", subagentType)
	}

	// Validate session hierarchy constraints BEFORE creating any resources
	if parentSessionID != "" {
		parentSession, err := s.Get(ctx, parentSessionID)
		if err != nil {
			return Session{}, fmt.Errorf("parent session not found: %w", err)
		}

		// Main sessions must be roots (no parent allowed)
		if sessionType == SessionTypeMain {
			return Session{}, fmt.Errorf("main sessions cannot have a parent session")
		}

		// Subagent sessions must have a main session as parent
		if sessionType == SessionTypeSubagent {
			if parentSession.SessionType != SessionTypeMain {
				return Session{}, fmt.Errorf("subagent sessions can only be created from main sessions, got parent type: %s", parentSession.SessionType)
			}
		}

		// Forked sessions should use Fork() method, not Create()
		if sessionType == SessionTypeForked {
			return Session{}, fmt.Errorf("forked sessions must be created using Fork() method, not Create()")
		}
	} else {
		// Sessions without parent must be main sessions
		if sessionType != SessionTypeMain {
			return Session{}, fmt.Errorf("%s sessions must have a parent session", sessionType)
		}
	}

	sessionID := uuid.New().String()

	// FAIL IMMEDIATELY if we cannot create storage directory
	// This prevents database inconsistency where session exists but has no storage
	if err := CreateSessionDirectory(sessionID, s.storageConfig); err != nil {
		return Session{}, fmt.Errorf("CRITICAL: session storage directory creation failed, aborting session creation: %w", err)
	}

	// Convert typed values to strings for database layer
	sessionTypeStr := sessionType.String()
	subagentTypeStr := subagentType.String()

	// Only create database entry AFTER storage directory is confirmed to exist
	dbSession, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		ID:                 sessionID,
		ParentSessionID:    sql.NullString{String: parentSessionID, Valid: parentSessionID != ""},
		Title:              title,
		CustomSystemPrompt: sql.NullString{String: customSystemPrompt, Valid: customSystemPrompt != ""},
		PromptMode:         sql.NullString{String: promptMode, Valid: promptMode != ""},
		SessionType:        sessionTypeStr,
		SubagentType:       sql.NullString{String: subagentTypeStr, Valid: subagentTypeStr != ""},
	})
	if err != nil {
		// If DB creation fails after directory creation, we have an orphaned directory
		// This is better than the reverse (session in DB with no directory)
		return Session{}, fmt.Errorf("session database creation failed after storage directory was created: %w", err)
	}

	session, err := s.fromCreatedSessionRow(dbSession)
	if err != nil {
		return Session{}, fmt.Errorf("session data conversion failed: %w", err)
	}

	if shouldPublish(ctx) {
		err = s.Publish(ctx, pubsub.CreatedEvent, session)
		if err != nil {
			return Session{}, fmt.Errorf("session event publication failed: %w", err)
		}
	}
	return session, nil
}

func (s *service) Fork(ctx context.Context, sourceSessionID string, title string) (Session, error) {
	// Verify source session exists
	sourceSession, err := s.Get(ctx, sourceSessionID)
	if err != nil {
		return Session{}, err
	}

	// Validate fork hierarchy constraints - cannot fork from subagent sessions
	if sourceSession.SessionType == SessionTypeSubagent {
		return Session{}, fmt.Errorf("cannot fork from subagent sessions - subagents are delegated work contexts not meant for user interaction")
	}

	sessionID := uuid.New().String()

	// FAIL IMMEDIATELY if we cannot create storage directory for forked session
	if err := CreateSessionDirectory(sessionID, s.storageConfig); err != nil {
		return Session{}, fmt.Errorf("CRITICAL: forked session storage directory creation failed, aborting fork: %w", err)
	}

	dbSession, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		ID:                 sessionID,
		ParentSessionID:    sql.NullString{String: sourceSessionID, Valid: true},
		Title:              title,
		CustomSystemPrompt: sql.NullString{Valid: false}, // Forked sessions use default prompt
		PromptMode:         sql.NullString{String: "default", Valid: true},
		SessionType:        SessionTypeForked.String(), // Type-safe constant
		SubagentType:       sql.NullString{Valid: false}, // Not a subagent
	})
	if err != nil {
		return Session{}, err
	}
	session, err := s.fromCreatedSessionRow(dbSession)
	if err != nil {
		return Session{}, err
	}

	if shouldPublish(ctx) {
		err = s.Publish(ctx, pubsub.CreatedEvent, session)
		if err != nil {
			return Session{}, err
		}
	}
	return session, nil
}

// Removed complex session creation methods for embedded binary

func (s *service) Delete(ctx context.Context, id string) error {
	session, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Delete session from database
	err = s.q.DeleteSession(ctx, session.ID)
	if err != nil {
		return err
	}

	// Clean up session shell first (before deleting storage directory)
	sessionStorageDir := GetSessionStoragePath(session.ID, s.storageConfig)
	if err := shell.CleanupSessionShell(sessionStorageDir); err != nil {
		return fmt.Errorf("failed to cleanup session shell for %s: %w", session.ID, err)
	}

	// Delete session storage directory and all files
	if err := DeleteSessionDirectory(session.ID, s.storageConfig); err != nil {
		// Log error but don't fail the operation - database cleanup succeeded
		fmt.Printf("Failed to delete session storage directory for %s: %v\n", session.ID, err)
	}

	if shouldPublish(ctx) {
		err = s.Publish(ctx, pubsub.DeletedEvent, session)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *service) Get(ctx context.Context, id string) (Session, error) {
	dbSession, err := s.q.GetSessionByID(ctx, id)
	if err != nil {
		return Session{}, err
	}
	return s.fromGetSessionByIDRow(dbSession)
}

func (s *service) List(ctx context.Context) ([]Session, error) {
	dbSessions, err := s.q.ListSessionsMetadata(ctx)
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, len(dbSessions))
	for i, dbSession := range dbSessions {
		session, err := s.fromListSessionsMetadataRow(dbSession)
		if err != nil {
			return nil, err
		}
		sessions[i] = session
	}
	return sessions, nil
}

func (s *service) ListWithContent(ctx context.Context) ([]db.ListSessionsWithContentRow, error) {
	return s.q.ListSessionsWithContent(ctx)
}

func (s *service) Save(ctx context.Context, session Session) (Session, error) {
	// Immutability: SessionType, SubagentType, and ParentSessionID cannot be changed after creation.
	// Enforced at application level - UpdateSession SQL query excludes these fields from SET clause.
	// Direct database access could bypass this constraint.
	_, err := s.q.UpdateSession(ctx, db.UpdateSessionParams{
		ID:                 session.ID,
		Title:              session.Title,
		CustomSystemPrompt: sql.NullString{String: session.CustomSystemPrompt, Valid: session.CustomSystemPrompt != ""},
		PromptMode:         sql.NullString{String: session.PromptMode, Valid: session.PromptMode != ""},
		PromptTokens:       session.PromptTokens,
		CompletionTokens:   session.CompletionTokens,
		SummaryMessageID: sql.NullString{
			String: session.SummaryMessageID,
			Valid:  session.SummaryMessageID != "",
		},
		Cost: session.Cost,
	})
	if err != nil {
		return Session{}, err
	}

	// Get fresh session data (includes updated fields + message counts)
	updatedSession, err := s.Get(ctx, session.ID)
	if err != nil {
		return Session{}, err
	}

	if shouldPublish(ctx) {
		err = s.Publish(ctx, pubsub.UpdatedEvent, updatedSession)
		if err != nil {
			return Session{}, err
		}
	}
	return updatedSession, nil
}

func (s *service) IncrementCost(ctx context.Context, sessionID string, costDelta float64) error {
	return s.q.IncrementSessionCost(ctx, db.IncrementSessionCostParams{
		ID:   sessionID,
		Cost: costDelta,
	})
}

// Removed List method for embedded binary

// Conversion methods for different query return types


func (s *service) fromGetSessionByIDRow(item db.GetSessionByIDRow) (Session, error) {
	return Session{
		ID:                    item.ID,
		ParentSessionID:       item.ParentSessionID.String,
		Title:                 item.Title,
		UserMessageCount:      item.UserMessageCount,
		AssistantMessageCount: item.AssistantMessageCount,
		ToolCallCount:         item.ToolCallCount,
		PromptTokens:          item.PromptTokens,
		CompletionTokens:      item.CompletionTokens,
		SummaryMessageID:      item.SummaryMessageID.String,
		CustomSystemPrompt:    item.CustomSystemPrompt.String,
		PromptMode:            item.PromptMode.String,
		SessionType:           SessionType(item.SessionType),        // Convert string to type
		SubagentType:          SubagentType(item.SubagentType.String), // Convert string to type
		Cost:                  item.Cost,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}, nil
}

func (s *service) fromListSessionsMetadataRow(item db.ListSessionsMetadataRow) (Session, error) {
	return Session{
		ID:                    item.ID,
		ParentSessionID:       item.ParentSessionID.String,
		Title:                 item.Title,
		UserMessageCount:      item.UserMessageCount,
		AssistantMessageCount: item.AssistantMessageCount,
		ToolCallCount:         item.ToolCallCount,
		PromptTokens:          item.PromptTokens,
		CompletionTokens:      item.CompletionTokens,
		SummaryMessageID:      item.SummaryMessageID.String,
		CustomSystemPrompt:    item.CustomSystemPrompt.String,
		PromptMode:            item.PromptMode.String,
		SessionType:           SessionType(item.SessionType),        // Convert string to type
		SubagentType:          SubagentType(item.SubagentType.String), // Convert string to type
		Cost:                  item.Cost,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}, nil
}

func (s *service) fromCreatedSessionRow(item db.CreateSessionRow) (Session, error) {
	return Session{
		ID:                    item.ID,
		ParentSessionID:       item.ParentSessionID.String,
		Title:                 item.Title,
		UserMessageCount:      0, // New sessions always have 0 messages
		AssistantMessageCount: 0, // New sessions always have 0 messages
		ToolCallCount:         0, // New sessions always have 0 messages
		PromptTokens:          item.PromptTokens,
		CompletionTokens:      item.CompletionTokens,
		SummaryMessageID:      item.SummaryMessageID.String,
		CustomSystemPrompt:    item.CustomSystemPrompt.String,
		PromptMode:            item.PromptMode.String,
		SessionType:           SessionType(item.SessionType),        // Convert string to type
		SubagentType:          SubagentType(item.SubagentType.String), // Convert string to type
		Cost:                  item.Cost,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}, nil
}

func NewService(q db.Querier, storageConfig Config) Service {
	broker := pubsub.NewBroker[Session]()
	return &service{
		Broker:        broker,
		q:             q,
		storageConfig: storageConfig,
	}
}

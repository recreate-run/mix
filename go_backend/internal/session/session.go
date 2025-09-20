package session

import (
	"context"
	"database/sql"
	"fmt"

	"mix/internal/db"
	"mix/internal/llm/tools/shell"
	"mix/internal/pubsub"
	"mix/internal/storage"

	"github.com/google/uuid"
)

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
	Cost                  float64
	CreatedAt             int64
	UpdatedAt             int64
}

// Simplified Service interface for embedded binary
type Service interface {
	pubsub.Suscriber[Session]
	Create(ctx context.Context, title string) (Session, error)
	Fork(ctx context.Context, sourceSessionID string, title string) (Session, error)
	Get(ctx context.Context, id string) (Session, error)
	List(ctx context.Context) ([]Session, error)
	ListWithContent(ctx context.Context) ([]db.ListSessionsWithContentRow, error)
	Save(ctx context.Context, session Session) (Session, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	*pubsub.Broker[Session]
	q             db.Querier
	storageConfig storage.Config
}

func (s *service) Create(ctx context.Context, title string) (Session, error) {
	sessionID := uuid.New().String()
	
	// FAIL IMMEDIATELY if we cannot create storage directory
	// This prevents database inconsistency where session exists but has no storage
	if err := storage.CreateSessionDirectory(sessionID, s.storageConfig); err != nil {
		return Session{}, fmt.Errorf("CRITICAL: session storage directory creation failed, aborting session creation: %w", err)
	}

	// Only create database entry AFTER storage directory is confirmed to exist
	dbSession, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		ID:    sessionID,
		Title: title,
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

	err = s.Publish(ctx, pubsub.CreatedEvent, session)
	if err != nil {
		return Session{}, fmt.Errorf("session event publication failed: %w", err)
	}
	return session, nil
}

func (s *service) Fork(ctx context.Context, sourceSessionID string, title string) (Session, error) {
	// Verify source session exists
	_, err := s.Get(ctx, sourceSessionID)
	if err != nil {
		return Session{}, err
	}

	sessionID := uuid.New().String()
	
	// FAIL IMMEDIATELY if we cannot create storage directory for forked session
	if err := storage.CreateSessionDirectory(sessionID, s.storageConfig); err != nil {
		return Session{}, fmt.Errorf("CRITICAL: forked session storage directory creation failed, aborting fork: %w", err)
	}

	dbSession, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		ID:              sessionID,
		ParentSessionID: sql.NullString{String: sourceSessionID, Valid: true},
		Title:           title,
	})
	if err != nil {
		return Session{}, err
	}
	session, err := s.fromCreatedSessionRow(dbSession)
	if err != nil {
		return Session{}, err
	}

	err = s.Publish(ctx, pubsub.CreatedEvent, session)
	if err != nil {
		return Session{}, err
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
	sessionStorageDir := storage.GetSessionStoragePath(session.ID, s.storageConfig)
	if err := shell.CleanupSessionShell(sessionStorageDir); err != nil {
		return fmt.Errorf("failed to cleanup session shell for %s: %w", session.ID, err)
	}

	// Delete session storage directory and all files
	if err := storage.DeleteSessionDirectory(session.ID, s.storageConfig); err != nil {
		// Log error but don't fail the operation - database cleanup succeeded
		fmt.Printf("Failed to delete session storage directory for %s: %v\n", session.ID, err)
	}

	err = s.Publish(ctx, pubsub.DeletedEvent, session)
	if err != nil {
		return err
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
	dbSession, err := s.q.UpdateSession(ctx, db.UpdateSessionParams{
		ID:               session.ID,
		Title:            session.Title,
		PromptTokens:     session.PromptTokens,
		CompletionTokens: session.CompletionTokens,
		SummaryMessageID: sql.NullString{
			String: session.SummaryMessageID,
			Valid:  session.SummaryMessageID != "",
		},
		Cost: session.Cost,
	})
	if err != nil {
		return Session{}, err
	}
	session, err = s.fromUpdateSessionRowWithCounts(ctx, dbSession)
	if err != nil {
		return Session{}, err
	}
	err = s.Publish(ctx, pubsub.UpdatedEvent, session)
	if err != nil {
		return Session{}, err
	}
	return session, nil
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
		Cost:                  item.Cost,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}, nil
}

func (s *service) fromUpdateSessionRowWithCounts(ctx context.Context, item db.UpdateSessionRow) (Session, error) {
	// Get accurate counts by querying the full session data
	fullSession, err := s.q.GetSessionByID(ctx, item.ID)
	if err != nil {
		return Session{}, err
	}

	return Session{
		ID:                    item.ID,
		ParentSessionID:       item.ParentSessionID.String,
		Title:                 item.Title,
		UserMessageCount:      fullSession.UserMessageCount,      // Get real counts
		AssistantMessageCount: fullSession.AssistantMessageCount, // Get real counts
		ToolCallCount:         fullSession.ToolCallCount,         // Get real counts
		PromptTokens:          item.PromptTokens,
		CompletionTokens:      item.CompletionTokens,
		SummaryMessageID:      item.SummaryMessageID.String,
		Cost:                  item.Cost,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}, nil
}

func NewService(q db.Querier, storageConfig storage.Config) Service {
	broker := pubsub.NewBroker[Session]()
	return &service{
		Broker:        broker,
		q:             q,
		storageConfig: storageConfig,
	}
}

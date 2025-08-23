package session

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"mix/internal/db"
	"mix/internal/pubsub"

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
	WorkingDirectory      string
}

// Simplified Service interface for embedded binary
type Service interface {
	pubsub.Suscriber[Session]
	Create(ctx context.Context, title string, workingDirectory string) (Session, error)
	Fork(ctx context.Context, sourceSessionID string, title string) (Session, error)
	Get(ctx context.Context, id string) (Session, error)
	List(ctx context.Context) ([]Session, error)
	ListWithFirstMessage(ctx context.Context) ([]db.ListSessionsWithFirstMessageRow, error)
	Save(ctx context.Context, session Session) (Session, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	*pubsub.Broker[Session]
	q db.Querier
}

func (s *service) Create(ctx context.Context, title string, workingDirectory string) (Session, error) {
	dbSession, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		ID:               uuid.New().String(),
		Title:            title,
		WorkingDirectory: sql.NullString{String: workingDirectory, Valid: true},
	})
	if err != nil {
		return Session{}, err
	}
	session, err := s.fromCreatedSessionRow(dbSession)
	if err != nil {
		return Session{}, err
	}

	// Create input directory structure in session's working directory
	inputDir := filepath.Join(workingDirectory, "input")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		return Session{}, fmt.Errorf("failed to create input directory: %w", err)
	}

	inputSubdirs := []string{"images", "videos", "audios", "text"}
	for _, subdir := range inputSubdirs {
		subdirPath := filepath.Join(inputDir, subdir)
		if err := os.MkdirAll(subdirPath, 0o755); err != nil {
			return Session{}, fmt.Errorf("failed to create input subdirectory %s: %w", subdir, err)
		}
	}

	// Create output directory for generated videos
	outputDir := filepath.Join(workingDirectory, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Session{}, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create MIX.md file if it doesn't exist
	mixFilePath := filepath.Join(workingDirectory, "MIX.md")
	if _, err := os.Stat(mixFilePath); os.IsNotExist(err) {
		mixContent := "Sample MIX.md"
		if err := os.WriteFile(mixFilePath, []byte(mixContent), 0o644); err != nil {
			return Session{}, fmt.Errorf("failed to create MIX.md file: %w", err)
		}
	}

	err = s.Publish(ctx, pubsub.CreatedEvent, session)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *service) Fork(ctx context.Context, sourceSessionID string, title string) (Session, error) {
	sourceSession, err := s.Get(ctx, sourceSessionID)
	if err != nil {
		return Session{}, err
	}

	dbSession, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		ID:               uuid.New().String(),
		ParentSessionID:  sql.NullString{String: sourceSessionID, Valid: true},
		Title:            title,
		WorkingDirectory: sql.NullString{String: sourceSession.WorkingDirectory, Valid: true},
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

	err = s.q.DeleteSession(ctx, session.ID)
	if err != nil {
		return err
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
	dbSessions, err := s.q.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, len(dbSessions))
	for i, dbSession := range dbSessions {
		session, err := s.fromListSessionsRow(dbSession)
		if err != nil {
			return nil, err
		}
		sessions[i] = session
	}
	return sessions, nil
}

func (s *service) ListWithFirstMessage(ctx context.Context) ([]db.ListSessionsWithFirstMessageRow, error) {
	return s.q.ListSessionsWithFirstMessage(ctx)
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

// validateWorkingDirectory ensures working directory is valid
func validateWorkingDirectory(wd sql.NullString, sessionID string) error {
	if !wd.Valid {
		return fmt.Errorf("session %s has invalid working directory", sessionID)
	}
	return nil
}

func (s *service) fromGetSessionByIDRow(item db.GetSessionByIDRow) (Session, error) {
	if err := validateWorkingDirectory(item.WorkingDirectory, item.ID); err != nil {
		return Session{}, err
	}
	
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
		WorkingDirectory:      item.WorkingDirectory.String,
	}, nil
}

func (s *service) fromListSessionsRow(item db.ListSessionsRow) (Session, error) {
	if err := validateWorkingDirectory(item.WorkingDirectory, item.ID); err != nil {
		return Session{}, err
	}
	
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
		WorkingDirectory:      item.WorkingDirectory.String,
	}, nil
}

func (s *service) fromCreatedSessionRow(item db.CreateSessionRow) (Session, error) {
	if err := validateWorkingDirectory(item.WorkingDirectory, item.ID); err != nil {
		return Session{}, err
	}
	
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
		WorkingDirectory:      item.WorkingDirectory.String,
	}, nil
}

func (s *service) fromUpdateSessionRowWithCounts(ctx context.Context, item db.UpdateSessionRow) (Session, error) {
	if err := validateWorkingDirectory(item.WorkingDirectory, item.ID); err != nil {
		return Session{}, err
	}
	
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
		WorkingDirectory:      item.WorkingDirectory.String,
	}, nil
}


func NewService(q db.Querier) Service {
	broker := pubsub.NewBroker[Session]()
	return &service{
		Broker: broker,
		q:      q,
	}
}

package session

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"mix/internal/db"
	"mix/internal/pubsub"

	"github.com/google/uuid"
)

type Session struct {
	ID               string
	ParentSessionID  string
	Title            string
	MessageCount     int64
	PromptTokens     int64
	CompletionTokens int64
	SummaryMessageID string
	Cost             float64
	CreatedAt        int64
	UpdatedAt        int64
	WorkingDirectory string
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
	var workingDirValue sql.NullString
	if workingDirectory != "" {
		workingDirValue = sql.NullString{String: workingDirectory, Valid: true}
	}

	dbSession, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		ID:               uuid.New().String(),
		Title:            title,
		WorkingDirectory: workingDirValue,
	})
	if err != nil {
		return Session{}, err
	}
	session := s.fromDBItem(dbSession)

	// Create input directory structure in session's working directory
	if workingDirectory != "" {
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

		// Create output directory for Remotion videos
		outputDir := filepath.Join(workingDirectory, "output")
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return Session{}, fmt.Errorf("failed to create output directory: %w", err)
		}

		// Setup Remotion project by cloning template repository
		remotionProjectDir := filepath.Join(workingDirectory, "remotion_project")
		if err := s.setupRemotionProject(remotionProjectDir); err != nil {
			return Session{}, fmt.Errorf("failed to setup Remotion project: %w", err)
		}

		// Create MIX.md file if it doesn't exist
		mixFilePath := filepath.Join(workingDirectory, "MIX.md")
		if _, err := os.Stat(mixFilePath); os.IsNotExist(err) {
			mixContent := "Sample MIX.md"
			if err := os.WriteFile(mixFilePath, []byte(mixContent), 0o644); err != nil {
				return Session{}, fmt.Errorf("failed to create MIX.md file: %w", err)
			}
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

	var parentSessionID sql.NullString
	parentSessionID = sql.NullString{String: sourceSessionID, Valid: true}

	var workingDirValue sql.NullString
	if sourceSession.WorkingDirectory != "" {
		workingDirValue = sql.NullString{String: sourceSession.WorkingDirectory, Valid: true}
	}

	dbSession, err := s.q.CreateSession(ctx, db.CreateSessionParams{
		ID:               uuid.New().String(),
		ParentSessionID:  parentSessionID,
		Title:            title,
		WorkingDirectory: workingDirValue,
	})
	if err != nil {
		return Session{}, err
	}
	session := s.fromDBItem(dbSession)
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
	return s.fromDBItem(dbSession), nil
}

func (s *service) List(ctx context.Context) ([]Session, error) {
	dbSessions, err := s.q.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, len(dbSessions))
	for i, dbSession := range dbSessions {
		sessions[i] = s.fromDBItem(dbSession)
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
	session = s.fromDBItem(dbSession)
	err = s.Publish(ctx, pubsub.UpdatedEvent, session)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

// Removed List method for embedded binary

func (s service) fromDBItem(item db.Session) Session {
	return Session{
		ID:               item.ID,
		ParentSessionID:  item.ParentSessionID.String,
		Title:            item.Title,
		MessageCount:     item.MessageCount,
		PromptTokens:     item.PromptTokens,
		CompletionTokens: item.CompletionTokens,
		SummaryMessageID: item.SummaryMessageID.String,
		Cost:             item.Cost,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
		WorkingDirectory: item.WorkingDirectory.String,
	}
}

func (s *service) setupRemotionProject(projectDir string) error {
	// Skip if project already exists
	if _, err := os.Stat(projectDir); err == nil {
		return nil
	}

	// Get absolute path to script (from project root)
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	
	// Determine project root directory
	projectRoot := wd
	if filepath.Base(wd) == "go_backend" {
		projectRoot = filepath.Dir(wd)
	}
	
	scriptPath := filepath.Join(projectRoot, "go_backend", "scripts", "setup_remotion_project.sh")
	
	// Get session workspace directory (parent of projectDir) and project name
	sessionWorkspace := filepath.Dir(projectDir)
	remotionDirName := filepath.Base(projectDir)

	// Execute setup script in session workspace with relative project directory
	cmd := exec.Command("bash", scriptPath, remotionDirName)
	cmd.Dir = sessionWorkspace // Execute in session workspace
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup Remotion project: %w", err)
	}

	return nil
}

func NewService(q db.Querier) Service {
	broker := pubsub.NewBroker[Session]()
	return &service{
		broker,
		q,
	}
}

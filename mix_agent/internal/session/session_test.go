package session

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"

	"mix/internal/db"
	"mix/internal/pubsub"
)

// Test helper functions
func createTestService(t *testing.T) (*service, *db.MockQuerier) {
	t.Helper()
	mockQuerier := &db.MockQuerier{}

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "session_test_*")
	require.NoError(t, err)

	storageConfig := Config{
		BasePath: tempDir,
	}

	broker := pubsub.NewBroker[Session]()
	svc := &service{
		Broker:        broker,
		q:             mockQuerier,
		storageConfig: storageConfig,
	}

	// Clean up temp directory after test
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	return svc, mockQuerier
}

func createTestSession() Session {
	return Session{
		ID:                    uuid.New().String(),
		Title:                 "Test Session",
		UserMessageCount:      5,
		AssistantMessageCount: 3,
		ToolCallCount:         2,
		PromptTokens:          100,
		CompletionTokens:      200,
		CustomSystemPrompt:    "test prompt",
		PromptMode:            "standard",
		Cost:                  1.5,
		CreatedAt:             time.Now().Unix(),
		UpdatedAt:             time.Now().Unix(),
	}
}

func createTestCreateSessionRow() db.CreateSessionRow {
	return db.CreateSessionRow{
		ID:                 uuid.New().String(),
		ParentSessionID:    sql.NullString{String: "", Valid: false},
		Title:              "Test Session",
		PromptTokens:       0,
		CompletionTokens:   0,
		CustomSystemPrompt: sql.NullString{String: "test prompt", Valid: true},
		PromptMode:         sql.NullString{String: "standard", Valid: true},
		Cost:               0,
		CreatedAt:          time.Now().Unix(),
		UpdatedAt:          time.Now().Unix(),
	}
}

func createTestGetSessionByIDRow() db.GetSessionByIDRow {
	return db.GetSessionByIDRow{
		ID:                    uuid.New().String(),
		ParentSessionID:       sql.NullString{String: "", Valid: false},
		Title:                 "Test Session",
		UserMessageCount:      5,
		AssistantMessageCount: 3,
		ToolCallCount:         2,
		PromptTokens:          100,
		CompletionTokens:      200,
		CustomSystemPrompt:    sql.NullString{String: "test prompt", Valid: true},
		PromptMode:            sql.NullString{String: "standard", Valid: true},
		Cost:                  1.5,
		CreatedAt:             time.Now().Unix(),
		UpdatedAt:             time.Now().Unix(),
	}
}

// Test Create method
func TestCreate(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	title := "Test Session"
	customSystemPrompt := "test prompt"
	promptMode := "standard"

	createRow := createTestCreateSessionRow()

	mockQuerier.On("CreateSession", mock.Anything, mock.AnythingOfType("db.CreateSessionParams")).
		Return(createRow, nil)

	session, err := svc.Create(context.Background(), title, customSystemPrompt, promptMode, SessionTypeMain, "", "", "")

	require.NoError(t, err)
	assert.Equal(t, createRow.ID, session.ID)
	assert.Equal(t, title, session.Title)
	assert.Equal(t, customSystemPrompt, session.CustomSystemPrompt)
	assert.Equal(t, promptMode, session.PromptMode)

	mockQuerier.AssertExpectations(t)
}

// Test Fork method
func TestFork(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	sourceSessionID := uuid.New().String()
	title := "Forked Session"

	sourceSession := createTestGetSessionByIDRow()
	sourceSession.ID = sourceSessionID

	createRow := createTestCreateSessionRow()
	createRow.ParentSessionID = sql.NullString{String: sourceSessionID, Valid: true}
	createRow.Title = title

	mockQuerier.On("GetSessionByID", mock.Anything, sourceSessionID).
		Return(sourceSession, nil)
	mockQuerier.On("CreateSession", mock.Anything, mock.AnythingOfType("db.CreateSessionParams")).
		Return(createRow, nil)

	session, err := svc.Fork(context.Background(), sourceSessionID, title)

	require.NoError(t, err)
	assert.Equal(t, createRow.ID, session.ID)
	assert.Equal(t, title, session.Title)
	assert.Equal(t, sourceSessionID, session.ParentSessionID)

	mockQuerier.AssertExpectations(t)
}

// Test Get method
func TestGet(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	sessionID := uuid.New().String()
	getRow := createTestGetSessionByIDRow()
	getRow.ID = sessionID

	mockQuerier.On("GetSessionByID", mock.Anything, sessionID).
		Return(getRow, nil)

	session, err := svc.Get(context.Background(), sessionID)

	require.NoError(t, err)
	assert.Equal(t, getRow.ID, session.ID)
	assert.Equal(t, getRow.Title, session.Title)
	assert.Equal(t, getRow.UserMessageCount, session.UserMessageCount)

	mockQuerier.AssertExpectations(t)
}

// Test List method
func TestList(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	listRows := []db.ListSessionsMetadataRow{
		{
			ID:                    uuid.New().String(),
			ParentSessionID:       sql.NullString{String: "", Valid: false},
			Title:                 "Session 1",
			UserMessageCount:      5,
			AssistantMessageCount: 3,
			ToolCallCount:         2,
			PromptTokens:          100,
			CompletionTokens:      200,
			CustomSystemPrompt:    sql.NullString{String: "", Valid: false},
			PromptMode:            sql.NullString{String: "", Valid: false},
			Cost:                  1.5,
			CreatedAt:             time.Now().Unix(),
			UpdatedAt:             time.Now().Unix(),
		},
	}

	mockQuerier.On("ListSessionsMetadata", mock.Anything).
		Return(listRows, nil)

	sessions, err := svc.List(context.Background())

	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, listRows[0].ID, sessions[0].ID)

	mockQuerier.AssertExpectations(t)
}

// Test ListWithContent method
func TestListWithContent(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	contentRows := []db.ListSessionsWithContentRow{
		{
			ID:    uuid.New().String(),
			Title: "Session 1",
		},
	}

	mockQuerier.On("ListSessionsWithContent", mock.Anything).
		Return(contentRows, nil)

	sessions, err := svc.ListWithContent(context.Background())

	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, contentRows[0].ID, sessions[0].ID)

	mockQuerier.AssertExpectations(t)
}

// Test Save method
func TestSave(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	session := createTestSession()
	updateRow := db.UpdateSessionRow{
		ID:                 session.ID,
		ParentSessionID:    sql.NullString{String: session.ParentSessionID, Valid: false},
		Title:              session.Title,
		PromptTokens:       session.PromptTokens,
		CompletionTokens:   session.CompletionTokens,
		CustomSystemPrompt: sql.NullString{String: session.CustomSystemPrompt, Valid: true},
		PromptMode:         sql.NullString{String: session.PromptMode, Valid: true},
		Cost:               session.Cost,
		CreatedAt:          session.CreatedAt,
		UpdatedAt:          session.UpdatedAt,
	}

	getRow := createTestGetSessionByIDRow()
	getRow.ID = session.ID       // Ensure getRow has the same ID as the session being saved
	getRow.Title = session.Title // Match the title too for consistency

	mockQuerier.On("UpdateSession", mock.Anything, mock.AnythingOfType("db.UpdateSessionParams")).
		Return(updateRow, nil)
	mockQuerier.On("GetSessionByID", mock.Anything, session.ID).
		Return(getRow, nil)

	updatedSession, err := svc.Save(context.Background(), session)

	require.NoError(t, err)
	assert.Equal(t, session.ID, updatedSession.ID)
	assert.Equal(t, session.Title, updatedSession.Title)

	mockQuerier.AssertExpectations(t)
}

// Test Delete method
func TestDelete(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	sessionID := uuid.New().String()
	getRow := createTestGetSessionByIDRow()
	getRow.ID = sessionID

	// Create session directory for cleanup test
	sessionDir := filepath.Join(svc.storageConfig.BasePath, sessionID)
	err := os.MkdirAll(sessionDir, 0o750)
	require.NoError(t, err)

	mockQuerier.On("GetSessionByID", mock.Anything, sessionID).
		Return(getRow, nil)
	mockQuerier.On("DeleteSession", mock.Anything, sessionID).
		Return(nil)

	err = svc.Delete(context.Background(), sessionID)

	require.NoError(t, err)

	// Verify directory was deleted
	_, err = os.Stat(sessionDir)
	assert.True(t, os.IsNotExist(err))

	mockQuerier.AssertExpectations(t)
}

// Test conversion method fromGetSessionByIDRow
func TestFromGetSessionByIDRow(t *testing.T) {
	svc, _ := createTestService(t)

	dbRow := createTestGetSessionByIDRow()

	session, err := svc.fromGetSessionByIDRow(dbRow)

	require.NoError(t, err)
	assert.Equal(t, dbRow.ID, session.ID)
	assert.Equal(t, dbRow.Title, session.Title)
	assert.Equal(t, dbRow.UserMessageCount, session.UserMessageCount)
	assert.Equal(t, dbRow.CustomSystemPrompt.String, session.CustomSystemPrompt)
}

// Test conversion method fromListSessionsMetadataRow
func TestFromListSessionsMetadataRow(t *testing.T) {
	svc, _ := createTestService(t)

	dbRow := db.ListSessionsMetadataRow{
		ID:                    uuid.New().String(),
		ParentSessionID:       sql.NullString{String: "", Valid: false},
		Title:                 "Test Session",
		UserMessageCount:      5,
		AssistantMessageCount: 3,
		ToolCallCount:         2,
		PromptTokens:          100,
		CompletionTokens:      200,
		CustomSystemPrompt:    sql.NullString{String: "test prompt", Valid: true},
		PromptMode:            sql.NullString{String: "standard", Valid: true},
		Cost:                  1.5,
		CreatedAt:             time.Now().Unix(),
		UpdatedAt:             time.Now().Unix(),
	}

	session := svc.fromListSessionsMetadataRow(dbRow)

	assert.Equal(t, dbRow.ID, session.ID)
	assert.Equal(t, dbRow.Title, session.Title)
	assert.Equal(t, dbRow.UserMessageCount, session.UserMessageCount)
}

// Test conversion method fromCreatedSessionRow
func TestFromCreatedSessionRow(t *testing.T) {
	svc, _ := createTestService(t)

	dbRow := createTestCreateSessionRow()

	session := svc.fromCreatedSessionRow(dbRow)

	assert.Equal(t, dbRow.ID, session.ID)
	assert.Equal(t, dbRow.Title, session.Title)
	assert.Equal(t, int64(0), session.UserMessageCount) // New sessions have 0 messages
	assert.Equal(t, dbRow.CustomSystemPrompt.String, session.CustomSystemPrompt)
}

// Test NewService function
func TestNewService(t *testing.T) {
	mockQuerier := &db.MockQuerier{}
	storageConfig := Config{BasePath: "/tmp/test"}

	service := NewService(mockQuerier, storageConfig)

	assert.NotNil(t, service)

	// Test that it implements the Service interface
	var _ = service
}

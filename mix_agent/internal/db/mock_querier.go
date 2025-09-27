package db

import (
	"context"
	"database/sql"

	"github.com/stretchr/testify/mock"
)

// MockQuerier is a mock implementation of the Querier interface for testing
type MockQuerier struct {
	mock.Mock
}

// API Credentials methods
func (m *MockQuerier) UpsertAPICredential(ctx context.Context, arg UpsertAPICredentialParams) (ApiCredential, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(ApiCredential), args.Error(1)
}

func (m *MockQuerier) GetAPICredential(ctx context.Context, provider string) (ApiCredential, error) {
	args := m.Called(ctx, provider)
	return args.Get(0).(ApiCredential), args.Error(1)
}

func (m *MockQuerier) HasAPICredential(ctx context.Context, provider string) (int64, error) {
	args := m.Called(ctx, provider)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQuerier) DeleteAPICredential(ctx context.Context, provider string) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockQuerier) ListAPICredentials(ctx context.Context) ([]ApiCredential, error) {
	args := m.Called(ctx)
	return args.Get(0).([]ApiCredential), args.Error(1)
}

func (m *MockQuerier) DeleteAllAPICredentials(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQuerier) CreateAPICredential(ctx context.Context, arg CreateAPICredentialParams) (ApiCredential, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(ApiCredential), args.Error(1)
}

func (m *MockQuerier) UpdateAPICredential(ctx context.Context, arg UpdateAPICredentialParams) (ApiCredential, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(ApiCredential), args.Error(1)
}

// OAuth Credentials methods
func (m *MockQuerier) UpsertOAuthCredential(ctx context.Context, arg UpsertOAuthCredentialParams) (OauthCredential, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(OauthCredential), args.Error(1)
}

func (m *MockQuerier) GetOAuthCredential(ctx context.Context, provider string) (OauthCredential, error) {
	args := m.Called(ctx, provider)
	return args.Get(0).(OauthCredential), args.Error(1)
}

func (m *MockQuerier) HasOAuthCredential(ctx context.Context, provider string) (int64, error) {
	args := m.Called(ctx, provider)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQuerier) DeleteOAuthCredential(ctx context.Context, provider string) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockQuerier) ListOAuthCredentials(ctx context.Context) ([]OauthCredential, error) {
	args := m.Called(ctx)
	return args.Get(0).([]OauthCredential), args.Error(1)
}

func (m *MockQuerier) DeleteAllOAuthCredentials(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQuerier) ListExpiredOAuthCredentials(ctx context.Context) ([]OauthCredential, error) {
	args := m.Called(ctx)
	return args.Get(0).([]OauthCredential), args.Error(1)
}

func (m *MockQuerier) CreateOAuthCredential(ctx context.Context, arg CreateOAuthCredentialParams) (OauthCredential, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(OauthCredential), args.Error(1)
}

func (m *MockQuerier) UpdateOAuthCredential(ctx context.Context, arg UpdateOAuthCredentialParams) (OauthCredential, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(OauthCredential), args.Error(1)
}

// Files methods
func (m *MockQuerier) CreateFile(ctx context.Context, arg CreateFileParams) (File, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(File), args.Error(1)
}

func (m *MockQuerier) GetFile(ctx context.Context, id string) (File, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(File), args.Error(1)
}

func (m *MockQuerier) GetFileByPathAndSession(ctx context.Context, arg GetFileByPathAndSessionParams) (File, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(File), args.Error(1)
}

func (m *MockQuerier) UpdateFile(ctx context.Context, arg UpdateFileParams) (File, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(File), args.Error(1)
}

func (m *MockQuerier) DeleteFile(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) ListFilesByPath(ctx context.Context, path string) ([]File, error) {
	args := m.Called(ctx, path)
	return args.Get(0).([]File), args.Error(1)
}

func (m *MockQuerier) ListFilesBySession(ctx context.Context, sessionID string) ([]File, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]File), args.Error(1)
}

func (m *MockQuerier) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]File, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]File), args.Error(1)
}

// Messages methods
func (m *MockQuerier) CreateMessage(ctx context.Context, arg CreateMessageParams) (Message, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(Message), args.Error(1)
}

func (m *MockQuerier) GetMessage(ctx context.Context, id string) (Message, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Message), args.Error(1)
}

func (m *MockQuerier) UpdateMessage(ctx context.Context, arg UpdateMessageParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) DeleteMessage(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) ListMessagesBySession(ctx context.Context, sessionID string) ([]Message, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockQuerier) ListMessagesForFork(ctx context.Context, arg ListMessagesForForkParams) ([]Message, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockQuerier) ListUserMessageHistory(ctx context.Context, arg ListUserMessageHistoryParams) ([]Message, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]Message), args.Error(1)
}

// Sessions methods
func (m *MockQuerier) CreateSession(ctx context.Context, arg CreateSessionParams) (CreateSessionRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(CreateSessionRow), args.Error(1)
}

func (m *MockQuerier) GetSessionByID(ctx context.Context, id string) (GetSessionByIDRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(GetSessionByIDRow), args.Error(1)
}

func (m *MockQuerier) UpdateSession(ctx context.Context, arg UpdateSessionParams) (UpdateSessionRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(UpdateSessionRow), args.Error(1)
}

func (m *MockQuerier) DeleteSession(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) ListSessionsMetadata(ctx context.Context) ([]ListSessionsMetadataRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]ListSessionsMetadataRow), args.Error(1)
}

func (m *MockQuerier) ListSessionsWithContent(ctx context.Context) ([]ListSessionsWithContentRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]ListSessionsWithContentRow), args.Error(1)
}

// User Preferences methods
func (m *MockQuerier) CreateUserPreferences(ctx context.Context, arg CreateUserPreferencesParams) (UserPreference, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(UserPreference), args.Error(1)
}

func (m *MockQuerier) GetUserPreferences(ctx context.Context) (UserPreference, error) {
	args := m.Called(ctx)
	return args.Get(0).(UserPreference), args.Error(1)
}

func (m *MockQuerier) UpdateUserPreferences(ctx context.Context, arg UpdateUserPreferencesParams) (UserPreference, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(UserPreference), args.Error(1)
}

func (m *MockQuerier) UpdateMainAgentModel(ctx context.Context, arg UpdateMainAgentModelParams) (UserPreference, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(UserPreference), args.Error(1)
}

func (m *MockQuerier) UpdateSubAgentModel(ctx context.Context, arg UpdateSubAgentModelParams) (UserPreference, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(UserPreference), args.Error(1)
}

func (m *MockQuerier) UpdateUserPreferredProvider(ctx context.Context, preferredProvider sql.NullString) (UserPreference, error) {
	args := m.Called(ctx, preferredProvider)
	return args.Get(0).(UserPreference), args.Error(1)
}

func (m *MockQuerier) ResetUserPreferencesToDefaults(ctx context.Context) (UserPreference, error) {
	args := m.Called(ctx)
	return args.Get(0).(UserPreference), args.Error(1)
}
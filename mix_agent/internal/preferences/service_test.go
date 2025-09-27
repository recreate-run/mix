package preferences

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/db"
	"mix/internal/llm/models"
)

// Test helper functions
func CreateTestService(t *testing.T, mockQueries *db.MockQuerier) *UserPreferencesService {
	return NewUserPreferencesServiceWithQuerierAndPreload(mockQueries, false)
}

func CreateTestUserPreference() db.UserPreference {
	return db.UserPreference{
		PreferredProvider:        sql.NullString{String: "anthropic", Valid: true},
		MainAgentModel:           sql.NullString{String: "claude-4-sonnet", Valid: true},
		MainAgentMaxTokens:       sql.NullInt64{Int64: 4096, Valid: true},
		MainAgentReasoningEffort: sql.NullString{String: "medium", Valid: true},
		SubAgentModel:            sql.NullString{String: "claude-4-sonnet", Valid: true},
		SubAgentMaxTokens:        sql.NullInt64{Int64: 2048, Valid: true},
		SubAgentReasoningEffort:  sql.NullString{String: "low", Valid: true},
		CreatedAt:                time.Now().Unix(),
		UpdatedAt:                time.Now().Unix(),
	}
}

// Test GetUserPreferences from cache
func TestGetUserPreferencesFromCache(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	prefs, err := service.GetUserPreferences(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, &testPrefs, prefs)

	// Should not call DB
	mockQueries.AssertExpectations(t)
}

// Test GetUserPreferences from database
func TestGetUserPreferencesFromDB(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	testPrefs := CreateTestUserPreference()
	mockQueries.On("GetUserPreferences", mock.Anything).Return(testPrefs, nil)

	prefs, err := service.GetUserPreferences(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, &testPrefs, prefs)

	// Check cache is populated
	cached, found := service.preferencesCache.Load("default_user")
	assert.True(t, found)
	assert.Equal(t, &testPrefs, cached)

	mockQueries.AssertExpectations(t)
}

// Test GetUserPreferences not found
func TestGetUserPreferencesNotFound(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("GetUserPreferences", mock.Anything).Return(db.UserPreference{}, sql.ErrNoRows)

	prefs, err := service.GetUserPreferences(context.Background())
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, prefs)

	mockQueries.AssertExpectations(t)
}

// Test CreateDefaultUserPreferences
func TestCreateDefaultUserPreferences(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	expectedPrefs := CreateTestUserPreference()
	mockQueries.On("CreateUserPreferences", mock.Anything, mock.AnythingOfType("db.CreateUserPreferencesParams")).
		Return(expectedPrefs, nil)

	prefs, err := service.CreateDefaultUserPreferences(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, &expectedPrefs, prefs)

	// Check cache is populated
	cached, found := service.preferencesCache.Load("default_user")
	assert.True(t, found)
	assert.Equal(t, &expectedPrefs, cached)

	mockQueries.AssertExpectations(t)
}

// Test GetOrCreateUserPreferences when preferences exist
func TestGetOrCreateUserPreferencesExisting(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	prefs, err := service.GetOrCreateUserPreferences(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, &testPrefs, prefs)

	mockQueries.AssertExpectations(t)
}

// Test GetOrCreateUserPreferences when preferences don't exist
func TestGetOrCreateUserPreferencesCreate(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("GetUserPreferences", mock.Anything).Return(db.UserPreference{}, sql.ErrNoRows)

	createdPrefs := CreateTestUserPreference()
	mockQueries.On("CreateUserPreferences", mock.Anything, mock.AnythingOfType("db.CreateUserPreferencesParams")).
		Return(createdPrefs, nil)

	prefs, err := service.GetOrCreateUserPreferences(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, &createdPrefs, prefs)

	mockQueries.AssertExpectations(t)
}

// Test UpdateMainAgentPreferences
func TestUpdateMainAgentPreferences(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	updatedPrefs := CreateTestUserPreference()
	mockQueries.On("UpdateMainAgentModel", mock.Anything, mock.AnythingOfType("db.UpdateMainAgentModelParams")).
		Return(updatedPrefs, nil)

	err := service.UpdateMainAgentPreferences(context.Background(), "claude-4-opus", 8192, "high")
	assert.NoError(t, err)

	// Check cache is cleared
	_, found := service.preferencesCache.Load("default_user")
	assert.False(t, found)

	mockQueries.AssertExpectations(t)
}

// Test UpdateSubAgentPreferences
func TestUpdateSubAgentPreferences(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	updatedPrefs := CreateTestUserPreference()
	mockQueries.On("UpdateSubAgentModel", mock.Anything, mock.AnythingOfType("db.UpdateSubAgentModelParams")).
		Return(updatedPrefs, nil)

	err := service.UpdateSubAgentPreferences(context.Background(), "claude-4-haiku", 1024, "low")
	assert.NoError(t, err)

	// Check cache is cleared
	_, found := service.preferencesCache.Load("default_user")
	assert.False(t, found)

	mockQueries.AssertExpectations(t)
}

// Test UpdatePreferredProvider
func TestUpdatePreferredProvider(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	updatedPrefs := CreateTestUserPreference()
	mockQueries.On("UpdateUserPreferredProvider", mock.Anything, mock.AnythingOfType("sql.NullString")).
		Return(updatedPrefs, nil)

	err := service.UpdatePreferredProvider(context.Background(), models.ProviderOpenAI)
	assert.NoError(t, err)

	// Check cache is cleared
	_, found := service.preferencesCache.Load("default_user")
	assert.False(t, found)

	mockQueries.AssertExpectations(t)
}

// Test GetAgentConfig for main agent
func TestGetAgentConfigMain(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	agent, err := service.GetAgentConfig(context.Background(), AgentMain)
	assert.NoError(t, err)
	assert.Equal(t, models.ModelID("claude-4-sonnet"), agent.Model)
	assert.Equal(t, int64(4096), agent.MaxTokens)
	assert.Equal(t, "medium", agent.ReasoningEffort)

	mockQueries.AssertExpectations(t)
}

// Test GetAgentConfig for sub agent
func TestGetAgentConfigSub(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	agent, err := service.GetAgentConfig(context.Background(), AgentSub)
	assert.NoError(t, err)
	assert.Equal(t, models.ModelID("claude-4-sonnet"), agent.Model)
	assert.Equal(t, int64(2048), agent.MaxTokens)
	assert.Equal(t, "low", agent.ReasoningEffort)

	mockQueries.AssertExpectations(t)
}

// Test GetAgentConfig with unknown agent
func TestGetAgentConfigUnknown(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	_, err := service.GetAgentConfig(context.Background(), "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent name")

	mockQueries.AssertExpectations(t)
}

// Test GetPreferredProvider
func TestGetPreferredProvider(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	provider, err := service.GetPreferredProvider(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, models.ModelProvider("anthropic"), provider)

	mockQueries.AssertExpectations(t)
}

// Test GetPreferredProvider with empty preference
func TestGetPreferredProviderEmpty(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	testPrefs := CreateTestUserPreference()
	testPrefs.PreferredProvider = sql.NullString{String: "", Valid: false}
	service.preferencesCache.Store("default_user", &testPrefs)

	provider, err := service.GetPreferredProvider(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, models.ProviderAnthropic, provider) // Default

	mockQueries.AssertExpectations(t)
}

// Test PreloadPreferences success
func TestPreloadPreferences(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	testPrefs := CreateTestUserPreference()
	mockQueries.On("GetUserPreferences", mock.Anything).Return(testPrefs, nil)

	service.PreloadPreferences(context.Background())

	// Check cache is populated
	cached, found := service.preferencesCache.Load("default_user")
	assert.True(t, found)
	assert.Equal(t, &testPrefs, cached)

	mockQueries.AssertExpectations(t)
}

// Test PreloadPreferences not found
func TestPreloadPreferencesNotFound(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("GetUserPreferences", mock.Anything).Return(db.UserPreference{}, sql.ErrNoRows)

	service.PreloadPreferences(context.Background())

	// Check cache is not populated
	_, found := service.preferencesCache.Load("default_user")
	assert.False(t, found)

	mockQueries.AssertExpectations(t)
}

// Test PreloadPreferences error
func TestPreloadPreferencesError(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	mockQueries.On("GetUserPreferences", mock.Anything).Return(db.UserPreference{}, errors.New("database error"))

	service.PreloadPreferences(context.Background())

	// Check cache is not populated
	_, found := service.preferencesCache.Load("default_user")
	assert.False(t, found)

	mockQueries.AssertExpectations(t)
}

// Test ClearCache
func TestClearCache(t *testing.T) {
	mockQueries := &db.MockQuerier{}
	service := CreateTestService(t, mockQueries)

	// Pre-populate cache
	testPrefs := CreateTestUserPreference()
	service.preferencesCache.Store("default_user", &testPrefs)

	service.ClearCache()

	// Check cache is cleared
	_, found := service.preferencesCache.Load("default_user")
	assert.False(t, found)
}

// Test service initialization with preload enabled
func TestNewUserPreferencesServiceWithPreload(t *testing.T) {
	mockQueries := &db.MockQuerier{}

	// Mock the preload call that will happen in the background goroutine
	mockQueries.On("GetUserPreferences", mock.Anything).Return(db.UserPreference{}, sql.ErrNoRows)

	service := NewUserPreferencesServiceWithQuerierAndPreload(mockQueries, true)
	require.NotNil(t, service)
	assert.Equal(t, mockQueries, service.queries)

	// Give time for the background goroutine to complete
	time.Sleep(100 * time.Millisecond)
	mockQueries.AssertExpectations(t)
}

// Test service initialization without preload
func TestNewUserPreferencesServiceWithoutPreload(t *testing.T) {
	mockQueries := &db.MockQuerier{}

	service := NewUserPreferencesServiceWithQuerierAndPreload(mockQueries, false)
	require.NotNil(t, service)
	assert.Equal(t, mockQueries, service.queries)
}
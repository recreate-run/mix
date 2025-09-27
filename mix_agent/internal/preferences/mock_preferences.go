package preferences

import (
	"context"

	"github.com/stretchr/testify/mock"

	"mix/internal/db"
	"mix/internal/llm/models"
)

// MockService implements Service interface for testing
type MockService struct {
	mock.Mock
}

// Ensure MockService implements Service interface
var _ Service = (*MockService)(nil)

func (m *MockService) GetUserPreferences(ctx context.Context) (*db.UserPreference, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.UserPreference), args.Error(1)
}

func (m *MockService) CreateDefaultUserPreferences(ctx context.Context) (*db.UserPreference, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.UserPreference), args.Error(1)
}

func (m *MockService) GetOrCreateUserPreferences(ctx context.Context) (*db.UserPreference, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.UserPreference), args.Error(1)
}

func (m *MockService) UpdateMainAgentPreferences(ctx context.Context, modelID models.ModelID, maxTokens int64, reasoningEffort string) error {
	args := m.Called(ctx, modelID, maxTokens, reasoningEffort)
	return args.Error(0)
}

func (m *MockService) UpdateSubAgentPreferences(ctx context.Context, modelID models.ModelID, maxTokens int64, reasoningEffort string) error {
	args := m.Called(ctx, modelID, maxTokens, reasoningEffort)
	return args.Error(0)
}

func (m *MockService) UpdatePreferredProvider(ctx context.Context, provider models.ModelProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockService) GetAgentConfig(ctx context.Context, agentName AgentName) (Agent, error) {
	args := m.Called(ctx, agentName)
	return args.Get(0).(Agent), args.Error(1)
}

func (m *MockService) GetPreferredProvider(ctx context.Context) (models.ModelProvider, error) {
	args := m.Called(ctx)
	return args.Get(0).(models.ModelProvider), args.Error(1)
}

func (m *MockService) PreloadPreferences(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockService) ClearCache() {
	m.Called()
}
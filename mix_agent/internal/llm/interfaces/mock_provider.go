package interfaces

import (
	"context"

	"github.com/stretchr/testify/mock"

	"mix/internal/llm/models"
	"mix/internal/message"
)

// MockProvider implements interfaces.Provider for testing
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Model() models.Model {
	args := m.Called()
	return args.Get(0).(models.Model)
}

func (m *MockProvider) SendMessages(ctx context.Context, messages []message.Message, tools []BaseTool) (*ProviderResponse, error) {
	args := m.Called(ctx, messages, tools)
	return args.Get(0).(*ProviderResponse), args.Error(1)
}

func (m *MockProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []BaseTool) <-chan ProviderEvent {
	args := m.Called(ctx, messages, tools)
	return args.Get(0).(<-chan ProviderEvent)
}
package message

import (
	"context"

	"github.com/stretchr/testify/mock"

	"mix/internal/pubsub"
)

// MockService implements message.Service for testing
type MockService struct {
	mock.Mock
}

func (m *MockService) Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error) {
	args := m.Called(ctx, sessionID, params)
	return args.Get(0).(Message), args.Error(1)
}

func (m *MockService) Update(ctx context.Context, message Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockService) Get(ctx context.Context, id string) (Message, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Message), args.Error(1)
}

func (m *MockService) List(ctx context.Context, sessionID string) ([]Message, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) ListUserMessageHistory(ctx context.Context, limit, offset int64) ([]Message, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockService) CopyMessagesToSession(ctx context.Context, sourceSessionID, targetSessionID string, messageIndex int64) error {
	args := m.Called(ctx, sourceSessionID, targetSessionID, messageIndex)
	return args.Error(0)
}

func (m *MockService) Subscribe(ctx context.Context) <-chan pubsub.Event[Message] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[Message])
}
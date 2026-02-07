package session

import (
	"context"

	"github.com/stretchr/testify/mock"

	"mix/internal/db"
	"mix/internal/pubsub"
)

// MockService implements session.Service for testing
type MockService struct {
	mock.Mock
}

func (m *MockService) Create(ctx context.Context, title, customSystemPrompt, promptMode string, sessionType SessionType, subagentType SubagentType, parentSessionID, parentToolCallID string) (Session, error) {
	args := m.Called(ctx, title, customSystemPrompt, promptMode, sessionType, subagentType, parentSessionID, parentToolCallID)
	return args.Get(0).(Session), args.Error(1)
}

func (m *MockService) Get(ctx context.Context, id string) (Session, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Session), args.Error(1)
}

func (m *MockService) List(ctx context.Context) ([]Session, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Session), args.Error(1)
}

func (m *MockService) ListWithContent(ctx context.Context) ([]db.ListSessionsWithContentRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ListSessionsWithContentRow), args.Error(1)
}

func (m *MockService) Save(ctx context.Context, sess Session) (Session, error) {
	args := m.Called(ctx, sess)
	var result Session
	if args.Get(0) != nil {
		result = args.Get(0).(Session)
	}
	return result, args.Error(1)
}

func (m *MockService) IncrementCost(ctx context.Context, sessionID string, costDelta float64) error {
	args := m.Called(ctx, sessionID, costDelta)
	return args.Error(0)
}

func (m *MockService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) Subscribe(ctx context.Context) <-chan pubsub.Event[Session] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[Session])
}

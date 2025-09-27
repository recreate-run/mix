package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"mix/internal/config"
	"mix/internal/db"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/message"
	"mix/internal/pubsub"
	"mix/internal/session"
)

// Mock implementations for testing

// MockSessionService implements session.Service
type MockSessionService struct {
	mock.Mock
}

func (m *MockSessionService) Create(ctx context.Context, title string, customSystemPrompt string, promptMode string) (session.Session, error) {
	args := m.Called(ctx, title, customSystemPrompt, promptMode)
	return args.Get(0).(session.Session), args.Error(1)
}

func (m *MockSessionService) Fork(ctx context.Context, sourceSessionID string, title string) (session.Session, error) {
	args := m.Called(ctx, sourceSessionID, title)
	return args.Get(0).(session.Session), args.Error(1)
}

func (m *MockSessionService) Get(ctx context.Context, id string) (session.Session, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(session.Session), args.Error(1)
}

func (m *MockSessionService) List(ctx context.Context) ([]session.Session, error) {
	args := m.Called(ctx)
	return args.Get(0).([]session.Session), args.Error(1)
}

func (m *MockSessionService) ListWithContent(ctx context.Context) ([]db.ListSessionsWithContentRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.ListSessionsWithContentRow), args.Error(1)
}

func (m *MockSessionService) Save(ctx context.Context, sess session.Session) (session.Session, error) {
	args := m.Called(ctx, sess)
	var result session.Session
	if args.Get(0) != nil {
		result = args.Get(0).(session.Session)
	}
	return result, args.Error(1)
}

func (m *MockSessionService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionService) Subscribe(ctx context.Context) <-chan pubsub.Event[session.Session] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[session.Session])
}

// MockMessageService implements message.Service
type MockMessageService struct {
	mock.Mock
}

func (m *MockMessageService) Create(ctx context.Context, sessionID string, params message.CreateMessageParams) (message.Message, error) {
	args := m.Called(ctx, sessionID, params)
	return args.Get(0).(message.Message), args.Error(1)
}

func (m *MockMessageService) Update(ctx context.Context, message message.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageService) Get(ctx context.Context, id string) (message.Message, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(message.Message), args.Error(1)
}

func (m *MockMessageService) List(ctx context.Context, sessionID string) ([]message.Message, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]message.Message), args.Error(1)
}

func (m *MockMessageService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMessageService) ListUserMessageHistory(ctx context.Context, limit, offset int64) ([]message.Message, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]message.Message), args.Error(1)
}

func (m *MockMessageService) CopyMessagesToSession(ctx context.Context, sourceSessionID, targetSessionID string, messageIndex int64) error {
	args := m.Called(ctx, sourceSessionID, targetSessionID, messageIndex)
	return args.Error(0)
}

func (m *MockMessageService) Subscribe(ctx context.Context) <-chan pubsub.Event[message.Message] {
	args := m.Called(ctx)
	return args.Get(0).(<-chan pubsub.Event[message.Message])
}

// MockProvider implements interfaces.Provider
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Model() models.Model {
	args := m.Called()
	return args.Get(0).(models.Model)
}

func (m *MockProvider) SendMessages(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) (*interfaces.ProviderResponse, error) {
	args := m.Called(ctx, messages, tools)
	return args.Get(0).(*interfaces.ProviderResponse), args.Error(1)
}

func (m *MockProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) <-chan interfaces.ProviderEvent {
	args := m.Called(ctx, messages, tools)
	return args.Get(0).(<-chan interfaces.ProviderEvent)
}

// MockTool implements interfaces.BaseTool
type MockTool struct {
	mock.Mock
}

func (m *MockTool) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) Parameters() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	args := m.Called(ctx, input)
	return args.Get(0), args.Error(1)
}

// Test helper functions
func CreateTestAgent(t *testing.T, mockSessions *MockSessionService, mockMessages *MockMessageService, mockProvider *MockProvider) *agent {
	agentTools := []interfaces.BaseTool{}
	storageConfig := session.Config{}

	// Create agent manually instead of using NewAgent to avoid provider creation
	ctx, cancel := context.WithCancel(context.Background())
	accumulator := NewMessageAccumulator(mockMessages)

	return &agent{
		Broker:            pubsub.NewBroker[AgentEvent](),
		agentName:         config.AgentMain,
		provider:          mockProvider,
		messages:          mockMessages,
		sessions:          mockSessions,
		storageConfig:     storageConfig,
		tools:             agentTools,
		titleProvider:     mockProvider,
		summarizeProvider: mockProvider,
		accumulator:       accumulator,
		ctx:               ctx,
		cancel:            cancel,
	}
}

func CreateTestSession() session.Session {
	return session.Session{
		ID:                    "test-session-123",
		Title:                 "Test Session",
		UserMessageCount:      0,
		AssistantMessageCount: 0,
		ToolCallCount:         0,
		SummaryMessageID:      "",
		CreatedAt:             time.Now().Unix(),
		UpdatedAt:             time.Now().Unix(),
	}
}

func CreateTestMessage() message.Message {
	return message.Message{
		ID:        "test-message-123",
		SessionID: "test-session-123",
		Role:      message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello, world!"},
		},
		CreatedAt: time.Now().Unix(),
	}
}

func CreateTestModel() models.Model {
	return models.Model{
		ID:               "claude-4-sonnet",
		Provider:         models.ProviderAnthropic,
		Name:             "Claude 4 Sonnet",
		APIModel:         "claude-3-sonnet",
		ContextWindow:    4096,
		DefaultMaxTokens: 4096,
		CostPer1MIn:      3.0,
		CostPer1MOut:     15.0,
	}
}

// Basic functionality tests

// Test Model method
func TestModel(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	testModel := CreateTestModel()
	mockProvider.On("Model").Return(testModel)

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	model := agent.Model()
	assert.Equal(t, testModel, model)

	mockProvider.AssertExpectations(t)
}

// Test Cancel method
func TestCancel(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := "test-session-123"

	// Set up a context to cancel
	_, cancel := context.WithCancel(context.Background())
	agent.activeContexts.Store(sessionID, cancel)

	// Test cancel
	agent.Cancel(sessionID)

	// Check that context was cancelled (should not exist in map anymore)
	_, exists := agent.activeContexts.Load(sessionID)
	assert.False(t, exists)
}

// Test Cancel with summarize context
func TestCancelSummarize(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := "test-session-123"

	// Set up a summarize context to cancel
	_, cancel := context.WithCancel(context.Background())
	agent.activeContexts.Store(sessionID+"-summarize", cancel)

	// Test cancel
	agent.Cancel(sessionID)

	// Check that summarize context was cancelled
	_, exists := agent.activeContexts.Load(sessionID + "-summarize")
	assert.False(t, exists)
}

// Test generateTitle method
func TestGenerateTitle(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := "test-session-123"
	content := "Hello, world!"
	testSession := CreateTestSession()

	// Mock session retrieval
	mockSessions.On("Get", mock.Anything, sessionID).Return(testSession, nil)

	// Mock session save after title update
	mockSessions.On("Save", mock.Anything, mock.AnythingOfType("session.Session")).Return(testSession, nil)

	// Mock provider response for title generation
	mockProvider.On("SendMessages", mock.Anything, mock.AnythingOfType("[]message.Message"), mock.AnythingOfType("[]interfaces.BaseTool")).
		Return(&interfaces.ProviderResponse{
			Content: "Test Title",
			FinishReason: message.FinishReasonEndTurn,
		}, nil)

	err := agent.generateTitle(context.Background(), sessionID, content)
	assert.NoError(t, err)

	mockSessions.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

// Test generateTitle with empty content
func TestGenerateTitleEmptyContent(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := "test-session-123"
	content := ""

	err := agent.generateTitle(context.Background(), sessionID, content)
	assert.NoError(t, err)

	// Should not call any mocks
	mockSessions.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

// Test generateTitle without title provider
func TestGenerateTitleNoProvider(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)
	agent.titleProvider = nil // Remove title provider

	sessionID := "test-session-123"
	content := "Hello, world!"

	err := agent.generateTitle(context.Background(), sessionID, content)
	assert.NoError(t, err)

	// Should not call any mocks
	mockSessions.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

// Test ClearAllSessionProviders
func TestClearAllSessionProviders(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	// Mock the Model method that will be called during cleanup
	testModel := CreateTestModel()
	mockProvider.On("Model").Return(testModel)

	// Add some session providers
	agent.sessionProviders.Store("session-1", mockProvider)
	agent.sessionProviders.Store("session-2", mockProvider)

	agent.ClearAllSessionProviders()

	// Check that all session providers are cleared
	_, exists1 := agent.sessionProviders.Load("session-1")
	_, exists2 := agent.sessionProviders.Load("session-2")
	assert.False(t, exists1)
	assert.False(t, exists2)
}

// Test error handling
func TestErr(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	testError := errors.New("test error")
	event := agent.err(testError)

	assert.Equal(t, AgentEventTypeError, event.Type)
	assert.Equal(t, testError, event.Error)
}

// Test Shutdown
func TestShutdown(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	// Should not panic
	agent.Shutdown()

	// Context should be cancelled
	select {
	case <-agent.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be cancelled")
	}
}

// Test createUserMessage basic functionality
func TestCreateUserMessage(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := "test-session-123"
	content := "Hello, world!"
	attachmentParts := []message.ContentPart{}

	testMessage := CreateTestMessage()
	mockMessages.On("Create", mock.Anything, sessionID, mock.AnythingOfType("message.CreateMessageParams")).
		Return(testMessage, nil)

	userMsg, err := agent.createUserMessage(context.Background(), sessionID, content, attachmentParts)
	assert.NoError(t, err)
	assert.Equal(t, testMessage, userMsg)

	mockMessages.AssertExpectations(t)
}

// Test createUserMessage with plan mode
func TestCreateUserMessagePlanMode(t *testing.T) {
	mockSessions := &MockSessionService{}
	mockMessages := &MockMessageService{}
	mockProvider := &MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := "test-session-123"
	content := "Hello, world!"
	attachmentParts := []message.ContentPart{}

	// Create context with plan mode
	ctx := context.WithValue(context.Background(), "plan_mode", true)

	testMessage := CreateTestMessage()
	mockMessages.On("Create", mock.Anything, sessionID, mock.AnythingOfType("message.CreateMessageParams")).
		Return(testMessage, nil)

	userMsg, err := agent.createUserMessage(ctx, sessionID, content, attachmentParts)
	assert.NoError(t, err)
	assert.Equal(t, testMessage, userMsg)

	mockMessages.AssertExpectations(t)
}
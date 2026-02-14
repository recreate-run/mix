package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/config"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/message"
	"mix/internal/pubsub"
	"mix/internal/session"
)

const (
	testSessionID      = "test-session-123"
	testMessageContent = "Hello, world!"
)

// Test helper functions using mocks from their respective packages

// Test helper functions
func CreateTestAgent(t *testing.T, mockSessions *session.MockService, mockMessages *message.MockService, mockProvider *interfaces.MockProvider) *agent {
	t.Helper()
	agentTools := []interfaces.BaseTool{}
	storageConfig := session.Config{}

	// Create agent manually instead of using NewAgent to avoid provider creation
	ctx, cancel := context.WithCancel(context.Background())
	accumulator := NewMessageAccumulator(mockMessages)

	return &agent{
		broker:        pubsub.NewBroker[AgentEvent](),
		agentName:     config.AgentMain,
		provider:      mockProvider,
		messages:      mockMessages,
		sessions:      mockSessions,
		storageConfig: storageConfig,
		tools:         agentTools,
		titleProvider: mockProvider,
		accumulator:   accumulator,
		ctx:           ctx,
		cancel:        cancel,
	}
}

func CreateTestSession() session.Session {
	return session.Session{
		ID:                    testSessionID,
		Title:                 "Test Session",
		UserMessageCount:      0,
		AssistantMessageCount: 0,
		ToolCallCount:         0,
		CreatedAt:             time.Now().Unix(),
		UpdatedAt:             time.Now().Unix(),
	}
}

func CreateTestMessage() message.Message {
	return message.Message{
		ID:        "test-message-123",
		SessionID: testSessionID,
		Role:      message.User,
		Parts: []message.ContentPart{
			message.TextContent{Text: testMessageContent},
		},
		CreatedAt: time.Now().Unix(),
	}
}

func CreateTestModel() models.Model {
	return models.Model{
		ID:               "claude-sonnet-4-5",
		Provider:         models.ProviderAnthropic,
		Name:             "Claude 4.5 Sonnet",
		APIModel:         "claude-sonnet-4-5-20250929",
		ContextWindow:    200000,
		DefaultMaxTokens: 50000,
		CostPer1MIn:      3.0,
		CostPer1MOut:     15.0,
	}
}

// Basic functionality tests

// Test Model method
func TestModel(t *testing.T) {
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

	testModel := CreateTestModel()
	mockProvider.On("Model").Return(testModel)

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	model := agent.Model()
	assert.Equal(t, testModel, model)

	mockProvider.AssertExpectations(t)
}

// Test Cancel method
func TestCancel(t *testing.T) {
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := testSessionID

	// Set up a context to cancel
	_, cancel := context.WithCancel(context.Background())
	agent.activeContexts.Store(sessionID, cancel)

	// Test cancel
	agent.Cancel(sessionID)

	// Check that context was cancelled (should not exist in map anymore)
	_, exists := agent.activeContexts.Load(sessionID)
	assert.False(t, exists)
}

// Test generateTitle method
func TestGenerateTitle(t *testing.T) {
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := testSessionID
	content := testMessageContent
	testSession := CreateTestSession()

	// Mock session retrieval
	mockSessions.On("Get", mock.Anything, sessionID).Return(testSession, nil)

	// Mock session save after title update
	mockSessions.On("Save", mock.Anything, mock.AnythingOfType("session.Session")).Return(testSession, nil)

	// Mock provider response for title generation
	mockProvider.On("SendMessages", mock.Anything, mock.AnythingOfType("[]message.Message"), mock.AnythingOfType("[]interfaces.BaseTool")).
		Return(&interfaces.ProviderResponse{
			Content:      "Test Title",
			FinishReason: message.FinishReasonEndTurn,
		}, nil)

	err := agent.generateTitle(context.Background(), sessionID, content)
	require.NoError(t, err)

	mockSessions.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

// Test generateTitle with empty content
func TestGenerateTitleEmptyContent(t *testing.T) {
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := testSessionID
	content := ""

	err := agent.generateTitle(context.Background(), sessionID, content)
	require.NoError(t, err)

	// Should not call any mocks
	mockSessions.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

// Test generateTitle without title provider
func TestGenerateTitleNoProvider(t *testing.T) {
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)
	agent.titleProvider = nil // Remove title provider

	sessionID := testSessionID
	content := testMessageContent

	err := agent.generateTitle(context.Background(), sessionID, content)
	require.NoError(t, err)

	// Should not call any mocks
	mockSessions.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

// Test ClearAllSessionProviders
func TestClearAllSessionProviders(t *testing.T) {
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

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
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	testError := errors.New("test error")
	event := agent.err(testError)

	assert.Equal(t, AgentEventTypeError, event.Type)
	assert.Equal(t, testError, event.Error)
}

// Test Shutdown
func TestShutdown(t *testing.T) {
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

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
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := testSessionID
	content := testMessageContent
	attachmentParts := []message.ContentPart{}

	testMessage := CreateTestMessage()
	mockMessages.On("Create", mock.Anything, sessionID, mock.AnythingOfType("message.CreateMessageParams")).
		Return(testMessage, nil)

	userMsg, err := agent.createUserMessage(context.Background(), sessionID, content, attachmentParts)
	require.NoError(t, err)
	assert.Equal(t, testMessage, userMsg)

	mockMessages.AssertExpectations(t)
}

// Test createUserMessage with plan mode
func TestCreateUserMessagePlanMode(t *testing.T) {
	mockSessions := &session.MockService{}
	mockMessages := &message.MockService{}
	mockProvider := &interfaces.MockProvider{}

	agent := CreateTestAgent(t, mockSessions, mockMessages, mockProvider)

	sessionID := testSessionID
	content := testMessageContent
	attachmentParts := []message.ContentPart{}

	// Create context with plan mode
	ctx := context.WithValue(context.Background(), interfaces.PlanModeContextKey, true)

	testMessage := CreateTestMessage()
	mockMessages.On("Create", mock.Anything, sessionID, mock.AnythingOfType("message.CreateMessageParams")).
		Return(testMessage, nil)

	userMsg, err := agent.createUserMessage(ctx, sessionID, content, attachmentParts)
	require.NoError(t, err)
	assert.Equal(t, testMessage, userMsg)

	mockMessages.AssertExpectations(t)
}

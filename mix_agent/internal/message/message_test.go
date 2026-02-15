package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"mix/internal/db"
	"mix/internal/llm/models"
	"mix/internal/pubsub"
)

// Test helper functions
func createTestService(t *testing.T) (*service, *db.MockQuerier) {
	t.Helper()
	mockQuerier := &db.MockQuerier{}
	broker := pubsub.NewBroker[Message]()
	svc := &service{
		Broker: broker,
		q:      mockQuerier,
	}
	return svc, mockQuerier
}

func createTestMessage() Message {
	return Message{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Role:      User,
		Parts: []ContentPart{
			TextContent{Text: "Hello, world!"},
		},
		Model:     models.ModelID("claude-sonnet-4-5"),
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
}

func createTestDBMessage() db.Message {
	parts, _ := json.Marshal([]partWrapper{
		{Type: textType, Data: TextContent{Text: "Hello, world!"}},
	})
	return db.Message{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Role:      string(User),
		Parts:     string(parts),
		Model:     sql.NullString{String: "claude-sonnet-4-5", Valid: true},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
}

// Test Create method
func TestCreate(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	sessionID := uuid.New().String()
	params := CreateMessageParams{
		Role: User,
		Parts: []ContentPart{
			TextContent{Text: "Test message"},
		},
		Model: models.ModelID("claude-sonnet-4-5"),
	}

	dbMessage := createTestDBMessage()
	dbMessage.SessionID = sessionID

	mockQuerier.On("CreateMessage", mock.Anything, mock.AnythingOfType("db.CreateMessageParams")).
		Return(dbMessage, nil)

	message, err := svc.Create(context.Background(), sessionID, params)

	require.NoError(t, err)
	assert.Equal(t, dbMessage.SessionID, message.SessionID)
	assert.Equal(t, User, message.Role)

	mockQuerier.AssertExpectations(t)
}

// Test Create with Assistant role (no finish part added)
func TestCreateAssistant(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	sessionID := uuid.New().String()
	params := CreateMessageParams{
		Role: Assistant,
		Parts: []ContentPart{
			TextContent{Text: "Assistant response"},
		},
		Model: models.ModelID("claude-sonnet-4-5"),
	}

	dbMessage := createTestDBMessage()
	dbMessage.Role = string(Assistant)

	mockQuerier.On("CreateMessage", mock.Anything, mock.AnythingOfType("db.CreateMessageParams")).
		Return(dbMessage, nil)

	message, err := svc.Create(context.Background(), sessionID, params)

	require.NoError(t, err)
	assert.Equal(t, Assistant, message.Role)

	mockQuerier.AssertExpectations(t)
}

// Test Update method
func TestUpdate(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	message := createTestMessage()
	message.Parts = append(message.Parts, Finish{Reason: FinishReasonEndTurn, Time: time.Now().Unix()})

	mockQuerier.On("UpdateMessage", mock.Anything, mock.AnythingOfType("db.UpdateMessageParams")).
		Return(nil)

	err := svc.Update(context.Background(), message)

	require.NoError(t, err)

	mockQuerier.AssertExpectations(t)
}

// Test Get method
func TestGet(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	messageID := uuid.New().String()
	dbMessage := createTestDBMessage()
	dbMessage.ID = messageID

	mockQuerier.On("GetMessage", mock.Anything, messageID).
		Return(dbMessage, nil)

	message, err := svc.Get(context.Background(), messageID)

	require.NoError(t, err)
	assert.Equal(t, messageID, message.ID)

	mockQuerier.AssertExpectations(t)
}

// Test List method
func TestList(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	sessionID := uuid.New().String()
	dbMessages := []db.Message{createTestDBMessage(), createTestDBMessage()}

	mockQuerier.On("ListMessagesBySession", mock.Anything, sessionID).
		Return(dbMessages, nil)

	messages, err := svc.List(context.Background(), sessionID)

	require.NoError(t, err)
	assert.Len(t, messages, 2)

	mockQuerier.AssertExpectations(t)
}

// Test Delete method
func TestDelete(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	messageID := uuid.New().String()
	dbMessage := createTestDBMessage()
	dbMessage.ID = messageID

	mockQuerier.On("GetMessage", mock.Anything, messageID).
		Return(dbMessage, nil)
	mockQuerier.On("DeleteMessage", mock.Anything, messageID).
		Return(nil)

	err := svc.Delete(context.Background(), messageID)

	require.NoError(t, err)

	mockQuerier.AssertExpectations(t)
}

// Test ListUserMessageHistory method
func TestListUserMessageHistory(t *testing.T) {
	svc, mockQuerier := createTestService(t)

	limit := int64(10)
	offset := int64(0)
	dbMessages := []db.Message{createTestDBMessage()}

	mockQuerier.On("ListUserMessageHistory", mock.Anything, mock.AnythingOfType("db.ListUserMessageHistoryParams")).
		Return(dbMessages, nil)

	messages, err := svc.ListUserMessageHistory(context.Background(), limit, offset)

	require.NoError(t, err)
	assert.Len(t, messages, 1)

	mockQuerier.AssertExpectations(t)
}

// Test marshallParts function
func TestMarshallParts(t *testing.T) {
	parts := []ContentPart{
		TextContent{Text: "Hello"},
		ToolCall{ID: "tool-1", Name: "test_tool", Input: "test input"},
		Finish{Reason: FinishReasonEndTurn, Time: time.Now().Unix()},
	}

	data, err := marshallParts(parts)

	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON and can be unmarshalled back
	unmarshalled, err := unmarshallParts(data)
	require.NoError(t, err)
	assert.Len(t, unmarshalled, 3)
}

// Test unmarshallParts function
func TestUnmarshallParts(t *testing.T) {
	originalParts := []ContentPart{
		TextContent{Text: "Hello"},
		ToolCall{ID: "tool-1", Name: "test_tool"},
	}

	data, err := marshallParts(originalParts)
	require.NoError(t, err)

	parts, err := unmarshallParts(data)

	require.NoError(t, err)
	assert.Len(t, parts, 2)

	textContent, ok := parts[0].(TextContent)
	assert.True(t, ok)
	assert.Equal(t, "Hello", textContent.Text)

	toolCall, ok := parts[1].(ToolCall)
	assert.True(t, ok)
	assert.Equal(t, "tool-1", toolCall.ID)
}

// Test unmarshallParts with unknown type
func TestUnmarshallPartsUnknownType(t *testing.T) {
	invalidData := `[{"type":"unknown_type","data":{}}]`

	_, err := unmarshallParts([]byte(invalidData))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown part type")
}

// Test Message.Content method
func TestMessageContent(t *testing.T) {
	message := Message{
		Parts: []ContentPart{
			ToolCall{ID: "tool-1"},
			TextContent{Text: "Hello, world!"},
		},
	}

	content := message.Content()

	assert.Equal(t, "Hello, world!", content.Text)
}

// Test Message.ToolCalls method
func TestMessageToolCalls(t *testing.T) {
	message := Message{
		Parts: []ContentPart{
			TextContent{Text: "Hello"},
			ToolCall{ID: "tool-1", Name: "test_tool"},
			ToolCall{ID: "tool-2", Name: "another_tool"},
		},
	}

	toolCalls := message.ToolCalls()

	assert.Len(t, toolCalls, 2)
	assert.Equal(t, "tool-1", toolCalls[0].ID)
	assert.Equal(t, "tool-2", toolCalls[1].ID)
}

// Test Message.IsFinished method
func TestMessageIsFinished(t *testing.T) {
	message := Message{
		Parts: []ContentPart{
			TextContent{Text: "Hello"},
			Finish{Reason: FinishReasonEndTurn},
		},
	}

	assert.True(t, message.IsFinished())

	messageUnfinished := Message{
		Parts: []ContentPart{
			TextContent{Text: "Hello"},
		},
	}

	assert.False(t, messageUnfinished.IsFinished())
}

// Test Message.FinishReason method
func TestMessageFinishReason(t *testing.T) {
	message := Message{
		Parts: []ContentPart{
			TextContent{Text: "Hello"},
			Finish{Reason: FinishReasonMaxTokens},
		},
	}

	reason := message.FinishReason()

	assert.Equal(t, FinishReasonMaxTokens, reason)
}

// Test Message.AppendContent method
func TestMessageAppendContent(t *testing.T) {
	message := Message{
		Parts: []ContentPart{
			TextContent{Text: "Hello"},
		},
	}

	message.AppendContent(" world!")

	content := message.Content()
	assert.Equal(t, "Hello world!", content.Text)
}

// Test Message.AddToolCall method
func TestMessageAddToolCall(t *testing.T) {
	message := Message{Parts: []ContentPart{}}
	toolCall := ToolCall{ID: "tool-1", Name: "test_tool"}

	message.AddToolCall(toolCall)

	toolCalls := message.ToolCalls()
	assert.Len(t, toolCalls, 1)
	assert.Equal(t, "tool-1", toolCalls[0].ID)
}

// Test Message.AddFinish method
func TestMessageAddFinish(t *testing.T) {
	message := Message{Parts: []ContentPart{}}

	message.AddFinish(FinishReasonEndTurn)

	assert.True(t, message.IsFinished())
	assert.Equal(t, FinishReasonEndTurn, message.FinishReason())
}

// Test Message.ThinkingBlocks method
func TestMessageThinkingBlocks(t *testing.T) {
	message := Message{
		Parts: []ContentPart{
			TextContent{Text: "Hello"},
			ThinkingBlockContent{Thinking: "Let me think...", Signature: "sig1"},
			ThinkingBlockContent{Thinking: "Another thought", Signature: "sig2"},
		},
	}

	blocks := message.ThinkingBlocks()

	assert.Len(t, blocks, 2)
	assert.Equal(t, "Let me think...", blocks[0].Thinking)
	assert.Equal(t, "Another thought", blocks[1].Thinking)
}

// Test Message.HasThinkingBlocks method
func TestMessageHasThinkingBlocks(t *testing.T) {
	messageWithThinking := Message{
		Parts: []ContentPart{
			TextContent{Text: "Hello"},
			ThinkingBlockContent{Thinking: "I'm thinking..."},
		},
	}

	messageWithoutThinking := Message{
		Parts: []ContentPart{
			TextContent{Text: "Hello"},
		},
	}

	assert.True(t, messageWithThinking.HasThinkingBlocks())
	assert.False(t, messageWithoutThinking.HasThinkingBlocks())
}

// Test Message.RateLimitInfo method
func TestMessageRateLimitInfo(t *testing.T) {
	// Test message with rate limit error
	rateLimitMessage := Message{
		Parts: []ContentPart{
			TextContent{Text: "rate_limit_error: too many requests"},
			Finish{Reason: "error"},
		},
	}

	info := rateLimitMessage.RateLimitInfo()
	assert.NotNil(t, info)
	assert.Equal(t, 60, info.RetryAfter)

	// Test message without rate limit error
	normalMessage := Message{
		Parts: []ContentPart{
			TextContent{Text: "Normal response"},
			Finish{Reason: FinishReasonEndTurn},
		},
	}

	info = normalMessage.RateLimitInfo()
	assert.Nil(t, info)
}

// Test NewService function
func TestNewService(t *testing.T) {
	mockQuerier := &db.MockQuerier{}

	service := NewService(mockQuerier)

	assert.NotNil(t, service)

	// Test that it implements the Service interface
	var _ = service
}

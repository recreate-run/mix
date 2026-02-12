package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"mix/internal/db"
	"mix/internal/llm/models"
	"mix/internal/pubsub"

	"github.com/google/uuid"
)

type CreateMessageParams struct {
	Role  MessageRole
	Parts []ContentPart
	Model models.ModelID
}

type Service interface {
	pubsub.Suscriber[Message]
	Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error)
	Update(ctx context.Context, message Message) error
	Get(ctx context.Context, id string) (Message, error)
	List(ctx context.Context, sessionID string) ([]Message, error)
	Delete(ctx context.Context, id string) error
	ListUserMessageHistory(ctx context.Context, limit, offset int64) ([]Message, error)
	DeleteAfterIndex(ctx context.Context, sessionID string, messageIndex int64) error
}

type service struct {
	*pubsub.Broker[Message]
	q db.Querier
}

func NewService(q db.Querier) Service {
	return &service{
		Broker: pubsub.NewBroker[Message](),
		q:      q,
	}
}

func (s *service) Delete(ctx context.Context, id string) error {
	message, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	err = s.q.DeleteMessage(ctx, message.ID)
	if err != nil {
		return err
	}
	err = s.Publish(ctx, pubsub.DeletedEvent, message)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error) {
	if params.Role != Assistant {
		params.Parts = append(params.Parts, Finish{
			Reason: "stop",
		})
	}
	partsJSON, err := marshallParts(params.Parts)
	if err != nil {
		return Message{}, err
	}
	dbMessage, err := s.q.CreateMessage(ctx, db.CreateMessageParams{
		ID:                  uuid.New().String(),
		SessionID:           sessionID,
		Role:                string(params.Role),
		Parts:               string(partsJSON),
		Model:               sql.NullString{String: string(params.Model), Valid: true},
		InputTokens:         sql.NullInt64{Int64: 0, Valid: true},
		OutputTokens:        sql.NullInt64{Int64: 0, Valid: true},
		CacheCreationTokens: sql.NullInt64{Int64: 0, Valid: true},
		CacheReadTokens:     sql.NullInt64{Int64: 0, Valid: true},
		Cost:                sql.NullFloat64{Float64: 0, Valid: true},
	})
	if err != nil {
		return Message{}, err
	}
	message, err := s.fromDBItem(dbMessage)
	if err != nil {
		return Message{}, err
	}
	err = s.Publish(ctx, pubsub.CreatedEvent, message)
	if err != nil {
		return Message{}, err
	}
	return message, nil
}

func (s *service) Update(ctx context.Context, message Message) error {
	parts, err := marshallParts(message.Parts)
	if err != nil {
		return err
	}
	finishedAt := sql.NullInt64{}
	if f := message.FinishPart(); f != nil {
		finishedAt.Int64 = f.Time
		finishedAt.Valid = true
	}
	err = s.q.UpdateMessage(ctx, db.UpdateMessageParams{
		ID:                  message.ID,
		Parts:               string(parts),
		FinishedAt:          finishedAt,
		InputTokens:         sql.NullInt64{Int64: message.InputTokens, Valid: true},
		OutputTokens:        sql.NullInt64{Int64: message.OutputTokens, Valid: true},
		CacheCreationTokens: sql.NullInt64{Int64: message.CacheCreationTokens, Valid: true},
		CacheReadTokens:     sql.NullInt64{Int64: message.CacheReadTokens, Valid: true},
		Cost:                sql.NullFloat64{Float64: message.Cost, Valid: true},
	})
	if err != nil {
		return err
	}
	message.UpdatedAt = time.Now().Unix()
	err = s.Publish(ctx, pubsub.UpdatedEvent, message)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) Get(ctx context.Context, id string) (Message, error) {
	dbMessage, err := s.q.GetMessage(ctx, id)
	if err != nil {
		return Message{}, err
	}
	return s.fromDBItem(dbMessage)
}

func (s *service) List(ctx context.Context, sessionID string) ([]Message, error) {
	dbMessages, err := s.q.ListMessagesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	messages := make([]Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i], err = s.fromDBItem(dbMessage)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

func (s *service) ListUserMessageHistory(ctx context.Context, limit, offset int64) ([]Message, error) {
	dbMessages, err := s.q.ListUserMessageHistory(ctx, db.ListUserMessageHistoryParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	messages := make([]Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i], err = s.fromDBItem(dbMessage)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

func (s *service) fromDBItem(item db.Message) (Message, error) {
	parts, err := unmarshallParts([]byte(item.Parts))
	if err != nil {
		return Message{}, err
	}
	return Message{
		ID:                  item.ID,
		SessionID:           item.SessionID,
		Role:                MessageRole(item.Role),
		Parts:               parts,
		Model:               models.ModelID(item.Model.String),
		InputTokens:         item.InputTokens.Int64,
		OutputTokens:        item.OutputTokens.Int64,
		CacheCreationTokens: item.CacheCreationTokens.Int64,
		CacheReadTokens:     item.CacheReadTokens.Int64,
		Cost:                item.Cost.Float64,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}, nil
}

type partType string

const (
	reasoningType        partType = "reasoning"
	thinkingBlockType    partType = "thinking_block"
	redactedThinkingType partType = "redacted_thinking"
	textType             partType = "text"
	imageURLType         partType = "image_url"
	binaryType           partType = "binary"
	uriType              partType = "uri"
	toolCallType         partType = "tool_call"
	toolResultType       partType = "tool_result"
	callbackResultType   partType = "callback_result"
	finishType           partType = "finish"
)

type partWrapper struct {
	Type partType    `json:"type"`
	Data ContentPart `json:"data"`
}

func marshallParts(parts []ContentPart) ([]byte, error) {
	wrappedParts := make([]partWrapper, len(parts))

	for i, part := range parts {
		var typ partType

		switch part.(type) {
		case ReasoningContent:
			typ = reasoningType
		case ThinkingBlockContent:
			typ = thinkingBlockType
		case RedactedThinkingContent:
			typ = redactedThinkingType
		case TextContent:
			typ = textType
		case ImageURLContent:
			typ = imageURLType
		case BinaryContent:
			typ = binaryType
		case URIContent:
			typ = uriType
		case ToolCall:
			typ = toolCallType
		case ToolResult:
			typ = toolResultType
		case CallbackResult:
			typ = callbackResultType
		case Finish:
			typ = finishType
		case ThinkingBlock:
			typ = thinkingBlockType
		default:
			return nil, fmt.Errorf("unknown part type: %T", part)
		}

		wrappedParts[i] = partWrapper{
			Type: typ,
			Data: part,
		}
	}
	return json.Marshal(wrappedParts)
}

func unmarshallParts(data []byte) ([]ContentPart, error) {
	temp := []json.RawMessage{}

	if err := json.Unmarshal(data, &temp); err != nil {
		return nil, err
	}

	parts := make([]ContentPart, 0)

	for _, rawPart := range temp {
		var wrapper struct {
			Type partType        `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(rawPart, &wrapper); err != nil {
			return nil, err
		}

		switch wrapper.Type {
		case reasoningType:
			part := ReasoningContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case thinkingBlockType:
			part := ThinkingBlockContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case redactedThinkingType:
			part := RedactedThinkingContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case textType:
			part := TextContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case imageURLType:
			part := ImageURLContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case binaryType:
			part := BinaryContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case uriType:
			part := URIContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case toolCallType:
			part := ToolCall{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case toolResultType:
			part := ToolResult{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case callbackResultType:
			part := CallbackResult{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case finishType:
			part := Finish{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		default:
			return nil, fmt.Errorf("unknown part type: %s", wrapper.Type)
		}
	}
	return parts, nil
}

// DeleteAfterIndex deletes all messages in a session after the specified index (0-based)
// The message at messageIndex is kept, all messages after it are deleted
// If messageIndex >= message count, no messages are deleted
func (s *service) DeleteAfterIndex(ctx context.Context, sessionID string, messageIndex int64) error {
	// Get all messages for the session
	messages, err := s.List(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to list messages for deletion: %w", err)
	}

	// Validate messageIndex - if at or beyond end, nothing to delete
	if messageIndex >= int64(len(messages)) {
		return nil
	}

	// Delete messages after the specified index (messageIndex is inclusive - we keep messages 0..messageIndex)
	for i := int(messageIndex) + 1; i < len(messages); i++ {
		if err := s.Delete(ctx, messages[i].ID); err != nil {
			return fmt.Errorf("failed to delete message at index %d (id=%s): %w", i, messages[i].ID, err)
		}
	}

	return nil
}

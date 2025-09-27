package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mix/internal/logging"
	"mix/internal/message"
)

// MessageUpdater is a minimal interface for message updates
type MessageUpdater interface {
	Update(ctx context.Context, message message.Message) error
}

// MessageAccumulator holds in-memory state for messages during streaming
// to avoid hitting the database for every delta event
type MessageAccumulator struct {
	mu       sync.RWMutex
	messages map[string]*AccumulatedMessage
	
	// Dependencies
	messageUpdater MessageUpdater
	
	// Shutdown handling
	ctx    context.Context
	cancel context.CancelFunc
}

// AccumulatedMessage represents the in-memory state of a streaming message
type AccumulatedMessage struct {
	Message       *message.Message
	LastUpdated   time.Time
	IsDirty       bool
	IsFinalized   bool
	mu            sync.Mutex
}

// NewMessageAccumulator creates a new message accumulator
func NewMessageAccumulator(messageUpdater MessageUpdater) *MessageAccumulator {
	ctx, cancel := context.WithCancel(context.Background())
	
	ma := &MessageAccumulator{
		messages:       make(map[string]*AccumulatedMessage),
		messageUpdater: messageUpdater,
		ctx:            ctx,
		cancel:         cancel,
	}
	
	// Note: No periodic flushing - messages are only flushed on:
	// 1. Finalization (completion/cancellation/error)
	// 2. Tool events (immediate flush)
	// 3. Shutdown (cleanup)
	
	return ma
}

// Store adds or updates a message in the accumulator
func (ma *MessageAccumulator) Store(msg *message.Message) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	
	accumulated, exists := ma.messages[msg.ID]
	if !exists {
		accumulated = &AccumulatedMessage{
			Message:     msg,
			LastUpdated: time.Now(),
			IsDirty:     true,
		}
		ma.messages[msg.ID] = accumulated
		logging.Debug(fmt.Sprintf("MessageAccumulator: Stored new message %s in accumulator", msg.ID))
	} else {
		accumulated.mu.Lock()
		accumulated.Message = msg
		accumulated.LastUpdated = time.Now()
		accumulated.IsDirty = true
		accumulated.mu.Unlock()
		logging.Debug(fmt.Sprintf("MessageAccumulator: Updated message %s in accumulator", msg.ID))
	}
}

// Get retrieves a message from the accumulator
func (ma *MessageAccumulator) Get(messageID string) (*message.Message, bool) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	
	accumulated, exists := ma.messages[messageID]
	if !exists {
		return nil, false
	}
	
	accumulated.mu.Lock()
	defer accumulated.mu.Unlock()
	
	// Return a copy to avoid concurrent modification
	msgCopy := *accumulated.Message
	return &msgCopy, true
}

// UpdateThinking appends thinking content without DB update
func (ma *MessageAccumulator) UpdateThinking(messageID string, delta string) error {
	ma.mu.RLock()
	accumulated, exists := ma.messages[messageID]
	ma.mu.RUnlock()
	
	if !exists {
		return nil // Message not in accumulator, skip
	}
	
	accumulated.mu.Lock()
	defer accumulated.mu.Unlock()
	
	// AppendReasoningContent already handles the delta appending
	// We just need to mark as dirty
	accumulated.LastUpdated = time.Now()
	accumulated.IsDirty = true
	
	logging.Debug(fmt.Sprintf("MessageAccumulator: Accumulated thinking delta for message %s (no DB write)", messageID))
	
	return nil
}

// UpdateContent appends content delta without DB update
func (ma *MessageAccumulator) UpdateContent(messageID string, delta string) error {
	ma.mu.RLock()
	accumulated, exists := ma.messages[messageID]
	ma.mu.RUnlock()
	
	if !exists {
		return nil // Message not in accumulator, skip
	}
	
	accumulated.mu.Lock()
	defer accumulated.mu.Unlock()
	
	// AppendContent already handles the delta appending
	// We just need to mark as dirty
	accumulated.LastUpdated = time.Now()
	accumulated.IsDirty = true
	
	logging.Debug(fmt.Sprintf("MessageAccumulator: Accumulated content delta for message %s (no DB write)", messageID))
	
	return nil
}

// FlushMessage immediately flushes a specific message to the database
func (ma *MessageAccumulator) FlushMessage(messageID string) error {
	ma.mu.RLock()
	accumulated, exists := ma.messages[messageID]
	ma.mu.RUnlock()
	
	if !exists {
		return nil // Nothing to flush
	}
	
	accumulated.mu.Lock()
	defer accumulated.mu.Unlock()
	
	if !accumulated.IsDirty {
		return nil // No changes to flush
	}
	
	// Update in database
	logging.Info(fmt.Sprintf("MessageAccumulator: Flushing message %s to database (manual flush)", accumulated.Message.ID))
	if err := ma.messageUpdater.Update(context.Background(), *accumulated.Message); err != nil {
		logging.Error(fmt.Sprintf("MessageAccumulator: Failed to flush message %s: %v", accumulated.Message.ID, err))
		return err
	}
	
	accumulated.IsDirty = false
	logging.Debug(fmt.Sprintf("MessageAccumulator: Successfully flushed message %s", accumulated.Message.ID))
	return nil
}

// FinalizeMessage marks a message as complete and flushes it
func (ma *MessageAccumulator) FinalizeMessage(messageID string, finishReason message.FinishReason) error {
	ma.mu.RLock()
	accumulated, exists := ma.messages[messageID]
	ma.mu.RUnlock()
	
	if !exists {
		return nil
	}
	
	accumulated.mu.Lock()
	defer accumulated.mu.Unlock()
	
	// Update finish reason and finalize
	// Note: AddFinish is already called by the agent before finalizing
	accumulated.IsFinalized = true
	accumulated.IsDirty = true
	
	// Always flush finalized messages immediately
	logging.Info(fmt.Sprintf("MessageAccumulator: Finalizing message %s with reason %v - flushing to database", messageID, finishReason))
	if err := ma.messageUpdater.Update(context.Background(), *accumulated.Message); err != nil {
		logging.Error(fmt.Sprintf("MessageAccumulator: Failed to finalize message %s: %v", messageID, err))
		return err
	}
	
	accumulated.IsDirty = false
	logging.Debug(fmt.Sprintf("MessageAccumulator: Successfully finalized message %s", messageID))
	
	// Remove from accumulator after a delay to handle late events
	go func() {
		time.Sleep(5 * time.Second)
		ma.mu.Lock()
		delete(ma.messages, messageID)
		ma.mu.Unlock()
		logging.Debug(fmt.Sprintf("MessageAccumulator: Removed finalized message %s from accumulator", messageID))
	}()
	
	return nil
}

// Note: We removed periodic flushing. Messages are only flushed on:
// - Finalization (completion/cancellation/error) 
// - Tool events (immediate flush)
// - Shutdown (cleanup)

// flushAllMessages flushes all messages regardless of dirty state
func (ma *MessageAccumulator) flushAllMessages() {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	
	if len(ma.messages) > 0 {
		logging.Info(fmt.Sprintf("MessageAccumulator: Flushing all %d messages on shutdown", len(ma.messages)))
	}
	
	flushedCount := 0
	for id, accumulated := range ma.messages {
		accumulated.mu.Lock()
		if accumulated.IsDirty {
			logging.Info(fmt.Sprintf("MessageAccumulator: Shutdown flush of message %s to database", id))
			if err := ma.messageUpdater.Update(context.Background(), *accumulated.Message); err != nil {
				logging.Error(fmt.Sprintf("MessageAccumulator: Failed to flush message %s during shutdown: %v", id, err))
			} else {
				accumulated.IsDirty = false
				flushedCount++
			}
		}
		accumulated.mu.Unlock()
	}
	
	if flushedCount > 0 {
		logging.Info(fmt.Sprintf("MessageAccumulator: Shutdown flush completed - flushed %d messages", flushedCount))
	}
}

// Shutdown gracefully shuts down the accumulator
func (ma *MessageAccumulator) Shutdown() {
	logging.Info("MessageAccumulator: Shutting down")
	
	// Flush all pending messages before shutdown
	ma.flushAllMessages()
	
	ma.cancel()
}
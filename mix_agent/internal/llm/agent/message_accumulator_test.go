package agent

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mix/internal/message"
)

// mockMessageUpdater implements MessageUpdater for testing
type mockMessageUpdater struct {
	mu          sync.Mutex
	updateCount map[string]int
	messages    map[string]*message.Message
}

func newMockMessageUpdater() *mockMessageUpdater {
	return &mockMessageUpdater{
		updateCount: make(map[string]int),
		messages:    make(map[string]*message.Message),
	}
}

func (m *mockMessageUpdater) Update(ctx context.Context, msg message.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.updateCount[msg.ID]++
	msgCopy := msg
	m.messages[msg.ID] = &msgCopy
	return nil
}

func (m *mockMessageUpdater) GetUpdateCount(msgID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCount[msgID]
}

// countingMessageUpdater wraps mock to count total operations
type countingMessageUpdater struct {
	*mockMessageUpdater
	totalUpdates atomic.Int64
}

func newCountingMessageUpdater() *countingMessageUpdater {
	return &countingMessageUpdater{
		mockMessageUpdater: newMockMessageUpdater(),
	}
}

func (c *countingMessageUpdater) Update(ctx context.Context, msg message.Message) error {
	c.totalUpdates.Add(1)
	return c.mockMessageUpdater.Update(ctx, msg)
}

func (c *countingMessageUpdater) GetTotalUpdateCount() int64 {
	return c.totalUpdates.Load()
}

// TestNoPeriodicFlush verifies that messages are NOT flushed periodically
func TestNoPeriodicFlush(t *testing.T) {
	mockService := newMockMessageUpdater()
	accumulator := NewMessageAccumulator(mockService)
	defer accumulator.Shutdown()

	msg := &message.Message{
		ID:        "test-no-flush",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts:     []message.ContentPart{},
	}

	// Store and update message
	accumulator.Store(msg)
	
	// Add multiple deltas over time
	for i := 0; i < 10; i++ {
		msg.AppendContent("delta ")
		if err := accumulator.UpdateContent(msg.ID, "delta "); err != nil {
			t.Logf("UpdateContent failed: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for what would have been multiple flush intervals
	t.Log("Waited 1 second - with old periodic flush this would have triggered 2 flushes")

	// Should have NO DB writes (no periodic flush)
	if count := mockService.GetUpdateCount(msg.ID); count != 0 {
		t.Errorf("Expected 0 DB updates (no periodic flush), but got %d", count)
	}

	// Now finalize the message
	msg.AddFinish(message.FinishReasonEndTurn)
	if err := accumulator.FinalizeMessage(msg.ID, message.FinishReasonEndTurn); err != nil {
		t.Logf("FinalizeMessage failed: %v", err)
	}

	// Should have exactly 1 DB write (on finalization)
	if count := mockService.GetUpdateCount(msg.ID); count != 1 {
		t.Errorf("Expected 1 DB update (on finalization), but got %d", count)
	}
	
	t.Log("✓ Confirmed: No periodic flushes, only flush on finalization")
}

// TestFlushOnlyOnSpecificEvents verifies flush happens only on specific events
func TestFlushOnlyOnSpecificEvents(t *testing.T) {
	mockService := newCountingMessageUpdater()
	accumulator := NewMessageAccumulator(mockService)

	t.Log("Testing flush behavior - should only flush on specific events")

	// Test 1: Normal streaming (no flush)
	msg1 := &message.Message{
		ID:        "msg-streaming",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts:     []message.ContentPart{},
	}
	accumulator.Store(msg1)
	
	// Simulate rapid streaming
	for i := 0; i < 50; i++ {
		msg1.AppendContent("x")
		if err := accumulator.UpdateContent(msg1.ID, "x"); err != nil {
			t.Logf("UpdateContent failed: %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)
	streamingWrites := mockService.GetTotalUpdateCount()
	t.Logf("After streaming 50 deltas: %d DB writes (expected 0)", streamingWrites)

	if streamingWrites != 0 {
		t.Errorf("Expected 0 DB writes during streaming, got %d", streamingWrites)
	}

	// Test 2: Tool event (immediate flush)
	msg2 := &message.Message{
		ID:        "msg-tool",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts:     []message.ContentPart{
			message.TextContent{Text: "Before tool"},
		},
	}
	accumulator.Store(msg2)

	// Simulate tool use - triggers immediate flush
	if err := accumulator.FlushMessage(msg2.ID); err != nil {
		t.Logf("FlushMessage failed: %v", err)
	}
	
	toolWrites := mockService.GetTotalUpdateCount() - streamingWrites
	t.Logf("After tool event: %d new DB writes (expected 1)", toolWrites)
	
	if toolWrites != 1 {
		t.Errorf("Expected 1 DB write for tool event, got %d", toolWrites)
	}

	// Test 3: Finalization (immediate flush)  
	msg3 := &message.Message{
		ID:        "msg-final",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts:     []message.ContentPart{},
	}
	accumulator.Store(msg3)
	msg3.AppendContent("Final message")
	if err := accumulator.UpdateContent(msg3.ID, "Final message"); err != nil {
		t.Logf("UpdateContent failed: %v", err)
	}

	// Finalize
	msg3.AddFinish(message.FinishReasonEndTurn)
	if err := accumulator.FinalizeMessage(msg3.ID, message.FinishReasonEndTurn); err != nil {
		t.Logf("FinalizeMessage failed: %v", err)
	}
	
	finalWrites := mockService.GetTotalUpdateCount() - streamingWrites - toolWrites
	t.Logf("After finalization: %d new DB writes (expected 1)", finalWrites)
	
	if finalWrites != 1 {
		t.Errorf("Expected 1 DB write for finalization, got %d", finalWrites)
	}

	t.Log("✓ Confirmed: Flushes only on tool events and finalization")
	
	// Shutdown will flush any remaining dirty messages
	accumulator.Shutdown()
}

// TestLongStreamingWithoutFlush simulates a very long stream without periodic flushes
func TestLongStreamingWithoutFlush(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long streaming test")
	}

	mockService := newCountingMessageUpdater()
	accumulator := NewMessageAccumulator(mockService)
	defer accumulator.Shutdown()

	msg := &message.Message{
		ID:        "long-stream",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts:     []message.ContentPart{},
	}
	
	accumulator.Store(msg)
	
	// Simulate streaming for 3 seconds
	start := time.Now()
	deltaCount := 0
	
	t.Log("Starting 3-second streaming simulation...")
	for time.Since(start) < 3*time.Second {
		msg.AppendContent(".")
		if err := accumulator.UpdateContent(msg.ID, "."); err != nil {
			t.Logf("UpdateContent failed: %v", err)
		}
		deltaCount++
		time.Sleep(10 * time.Millisecond)
	}
	
	// Check no DB writes during streaming
	dbWritesDuring := mockService.GetTotalUpdateCount()
	t.Logf("During 3-second streaming (%d deltas): %d DB writes", deltaCount, dbWritesDuring)
	
	if dbWritesDuring != 0 {
		t.Errorf("Expected 0 DB writes during long streaming, got %d", dbWritesDuring)
	}
	
	// Finalize - this is when the DB write happens
	msg.AddFinish(message.FinishReasonEndTurn)
	if err := accumulator.FinalizeMessage(msg.ID, message.FinishReasonEndTurn); err != nil {
		t.Logf("FinalizeMessage failed: %v", err)
	}
	
	// Should have exactly 1 DB write
	finalWrites := mockService.GetTotalUpdateCount()
	t.Logf("After finalization: %d total DB writes", finalWrites)
	
	if finalWrites != 1 {
		t.Errorf("Expected 1 DB write total, got %d", finalWrites)
	}
	
	t.Logf("✓ Streamed %d deltas over 3 seconds with only 1 DB write on completion", deltaCount)
}

// TestCancellationScenario tests that cancelled messages are properly saved
func TestCancellationScenario(t *testing.T) {
	mockService := newMockMessageUpdater()
	accumulator := NewMessageAccumulator(mockService)
	defer accumulator.Shutdown()

	msg := &message.Message{
		ID:        "cancelled-msg",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts:     []message.ContentPart{},
	}
	
	accumulator.Store(msg)
	
	// Simulate partial generation
	msg.AppendReasoningContent("I'm thinking about this problem...")
	if err := accumulator.UpdateThinking(msg.ID, "I'm thinking about this problem..."); err != nil {
		t.Logf("UpdateThinking failed: %v", err)
	}

	msg.AppendContent("Here's my partial respo-")
	if err := accumulator.UpdateContent(msg.ID, "Here's my partial respo-"); err != nil {
		t.Logf("UpdateContent failed: %v", err)
	}

	// User cancels
	msg.AddFinish(message.FinishReasonCanceled)
	if err := accumulator.FinalizeMessage(msg.ID, message.FinishReasonCanceled); err != nil {
		t.Logf("FinalizeMessage failed: %v", err)
	}
	
	// Should save the partial content
	if count := mockService.GetUpdateCount(msg.ID); count != 1 {
		t.Errorf("Expected 1 DB write for cancelled message, got %d", count)
	}
	
	t.Log("✓ Cancelled message saved with partial content")
}
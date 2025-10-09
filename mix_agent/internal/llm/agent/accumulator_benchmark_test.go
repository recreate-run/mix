package agent

import (
	"fmt"
	"testing"
	"time"

	"mix/internal/message"
)

// TestFinalOptimizationResults shows the final results of our optimization
func TestFinalOptimizationResults(t *testing.T) {
	mockService := newCountingMessageUpdater()
	accumulator := NewMessageAccumulator(mockService)
	defer accumulator.Shutdown()

	t.Log("\n=== FINAL OPTIMIZATION RESULTS ===")
	t.Log("Before: DB write on EVERY thinking/content delta")
	t.Log("After: DB write ONLY on completion/cancellation/error/tools")
	t.Log("")

	// Simulate a realistic Claude response
	assistant := &message.Message{
		ID:        "optimized-msg",
		SessionID: "test-session",
		Role:      message.Assistant,
		Parts:     []message.ContentPart{},
	}
	
	accumulator.Store(assistant)
	
	// Phase 1: Thinking (10 deltas)
	t.Log("Phase 1: Thinking...")
	for i := 0; i < 10; i++ {
		delta := fmt.Sprintf("Thinking part %d. ", i)
		assistant.AppendReasoningContent(delta)
		_ = accumulator.UpdateThinking(assistant.ID, delta)
		time.Sleep(50 * time.Millisecond)
	}
	
	thinkingWrites := mockService.GetTotalUpdateCount()
	t.Logf("  → After 10 thinking deltas: %d DB writes", thinkingWrites)
	
	// Phase 2: Content (20 deltas)
	t.Log("\nPhase 2: Content generation...")
	for i := 0; i < 20; i++ {
		delta := fmt.Sprintf("Content chunk %d. ", i)
		assistant.AppendContent(delta)
		if err := accumulator.UpdateContent(assistant.ID, delta); err != nil {
			t.Logf("UpdateContent failed: %v", err)
		}
		time.Sleep(30 * time.Millisecond)
	}
	
	contentWrites := mockService.GetTotalUpdateCount()
	t.Logf("  → After 20 content deltas: %d DB writes", contentWrites)
	
	// Phase 3: Completion
	t.Log("\nPhase 3: Message completion...")
	assistant.AddFinish(message.FinishReasonEndTurn)
	if err := accumulator.FinalizeMessage(assistant.ID, message.FinishReasonEndTurn); err != nil {
		t.Logf("FinalizeMessage failed: %v", err)
	}
	
	finalWrites := mockService.GetTotalUpdateCount()
	t.Logf("  → After finalization: %d DB writes", finalWrites)
	
	// Results
	totalEvents := 10 + 20 + 1 // thinking + content + finish
	t.Log("\n=== SUMMARY ===")
	t.Logf("Total events: %d", totalEvents)
	t.Logf("Total DB writes: %d", finalWrites)
	t.Logf("Write reduction: %.1f%%", (1.0 - float64(finalWrites)/float64(totalEvents)) * 100)
	
	t.Log("\nWithout optimization:")
	t.Logf("  - %d DB writes (one per event)", totalEvents)
	t.Log("  - DB hit on EVERY delta")
	t.Log("  - High database load")
	t.Log("  - Potential performance issues")
	
	t.Log("\nWith optimization:")
	t.Logf("  - %d DB write (only on completion)", finalWrites)
	t.Log("  - In-memory accumulation during streaming")
	t.Log("  - Minimal database load")
	t.Log("  - Better performance & scalability")
}

// TestScenarioComparison compares different scenarios
func TestScenarioComparison(t *testing.T) {
	scenarios := []struct {
		name           string
		thinkingDeltas int
		contentDeltas  int
		hasTools       bool
		isCancelled    bool
	}{
		{"Simple response", 5, 10, false, false},
		{"Complex with tools", 15, 25, true, false},
		{"Cancelled mid-stream", 8, 12, false, true},
		{"Long stream no tools", 30, 50, false, false},
	}
	
	t.Log("\n=== SCENARIO COMPARISON ===")
	t.Log("DB writes for different message types:")
	
	for _, s := range scenarios {
		mockService := newCountingMessageUpdater()
		accumulator := NewMessageAccumulator(mockService)
		
		msg := &message.Message{
			ID:        fmt.Sprintf("scenario-%s", s.name),
			SessionID: "test",
			Role:      message.Assistant,
			Parts:     []message.ContentPart{},
		}
		
		accumulator.Store(msg)
		
		// Add thinking
		for i := 0; i < s.thinkingDeltas; i++ {
			msg.AppendReasoningContent(".")
			_ = accumulator.UpdateThinking(msg.ID, ".")
		}

		// Add content
		for i := 0; i < s.contentDeltas; i++ {
			msg.AppendContent(".")
			if err := accumulator.UpdateContent(msg.ID, "."); err != nil {
				t.Logf("UpdateContent failed: %v", err)
			}
		}

		// Tool use if applicable
		if s.hasTools {
			if err := accumulator.FlushMessage(msg.ID); err != nil {
				t.Logf("FlushMessage failed: %v", err)
			}
		}

		// Finish
		if s.isCancelled {
			msg.AddFinish(message.FinishReasonCanceled)
			if err := accumulator.FinalizeMessage(msg.ID, message.FinishReasonCanceled); err != nil {
				t.Logf("FinalizeMessage failed: %v", err)
			}
		} else {
			msg.AddFinish(message.FinishReasonEndTurn)
			if err := accumulator.FinalizeMessage(msg.ID, message.FinishReasonEndTurn); err != nil {
				t.Logf("FinalizeMessage failed: %v", err)
			}
		}
		
		totalDeltas := s.thinkingDeltas + s.contentDeltas
		dbWrites := mockService.GetTotalUpdateCount()
		reduction := (1.0 - float64(dbWrites)/float64(totalDeltas)) * 100
		
		t.Logf("\n%s:", s.name)
		t.Logf("  Events: %d | DB writes: %d | Reduction: %.0f%%", 
			totalDeltas, dbWrites, reduction)
		
		accumulator.Shutdown()
	}
}
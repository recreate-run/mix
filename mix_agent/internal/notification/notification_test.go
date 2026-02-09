package notification

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"mix/internal/db"
	"mix/internal/pubsub"
	"mix/internal/session"
)

// mockSessionService for testing
type mockSessionService struct{}

func (m *mockSessionService) Create(ctx context.Context, title, customSystemPrompt, promptMode string, sessionType session.SessionType, subagentType session.SubagentType, parentSessionID, parentToolCallID string) (session.Session, error) {
	return session.Session{ID: "test-session-id"}, nil
}

func (m *mockSessionService) Get(ctx context.Context, id string) (session.Session, error) {
	return session.Session{ID: id}, nil
}

func (m *mockSessionService) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockSessionService) List(ctx context.Context) ([]session.Session, error) {
	return []session.Session{{ID: "test-session-id"}}, nil
}

func (m *mockSessionService) ListWithContent(ctx context.Context) ([]db.ListSessionsWithContentRow, error) {
	return []db.ListSessionsWithContentRow{}, nil
}

func (m *mockSessionService) Save(ctx context.Context, sess session.Session) (session.Session, error) {
	return sess, nil
}

func (m *mockSessionService) IncrementCost(ctx context.Context, sessionID string, costDelta float64) error {
	return nil
}

func (m *mockSessionService) Subscribe(ctx context.Context) <-chan pubsub.Event[session.Session] {
	ch := make(chan pubsub.Event[session.Session])
	return ch
}

func TestNotificationService_Request_Acknowledge(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	// Use channel to coordinate notification ID
	notifIDChan := make(chan string, 1)

	go func() {
		notifID := <-notifIDChan
		time.Sleep(100 * time.Millisecond)
		
		err := svc.Respond(notifID, NotificationResponse{
			ID:   notifID,
			Type: "acknowledge",
		})
		if err != nil {
			t.Errorf("Respond failed: %v", err)
		}
	}()

	// Subscribe to get the notification ID
	subChan := svc.Subscribe(context.Background())

	go func() {
		event := <-subChan
		notifIDChan <- event.Payload.ID
	}()

	// Create notification request
	resp, err := svc.Request(CreateNotificationRequest{
		SessionID:    "test-session",
		Type:         NotificationTypeInfo,
		Title:        "Test Notification",
		Message:      "Please acknowledge",
		ResponseType: ResponseTypeAcknowledge,
		Timeout:      5,
	})

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.Type != "acknowledge" {
		t.Errorf("Expected response type 'acknowledge', got '%s'", resp.Type)
	}
}

func TestNotificationService_Request_Text(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	expectedText := "User's text response"
	notifIDChan := make(chan string, 1)

	go func() {
		notifID := <-notifIDChan
		time.Sleep(100 * time.Millisecond)
		
		err := svc.Respond(notifID, NotificationResponse{
			ID:    notifID,
			Type:  "text",
			Value: expectedText,
		})
		if err != nil {
			t.Errorf("Respond failed: %v", err)
		}
	}()

	// Subscribe to get the notification ID
	subChan := svc.Subscribe(context.Background())

	go func() {
		event := <-subChan
		notifIDChan <- event.Payload.ID
	}()

	// Create notification request
	resp, err := svc.Request(CreateNotificationRequest{
		SessionID:    "test-session",
		Type:         NotificationTypeQuestion,
		Title:        "Test Question",
		Message:      "Please enter your name",
		ResponseType: ResponseTypeText,
		Timeout:      2,
	})

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.Type != "text" {
		t.Errorf("Expected response type 'text', got '%s'", resp.Type)
	}

	if resp.Value != expectedText {
		t.Errorf("Expected value '%s', got '%s'", expectedText, resp.Value)
	}
}

func TestNotificationService_Request_Choice(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	choices := []string{"Option A", "Option B", "Option C"}
	expectedChoice := "Option B"
	notifIDChan := make(chan string, 1)

	go func() {
		notifID := <-notifIDChan
		time.Sleep(100 * time.Millisecond)
		
		err := svc.Respond(notifID, NotificationResponse{
			ID:    notifID,
			Type:  "choice",
			Value: expectedChoice,
		})
		if err != nil {
			t.Errorf("Respond failed: %v", err)
		}
	}()

	// Subscribe to get the notification ID
	subChan := svc.Subscribe(context.Background())

	go func() {
		event := <-subChan
		notifIDChan <- event.Payload.ID
	}()

	// Create notification request
	resp, err := svc.Request(CreateNotificationRequest{
		SessionID:    "test-session",
		Type:         NotificationTypeQuestion,
		Title:        "Choose an option",
		Message:      "Please select one",
		ResponseType: ResponseTypeChoice,
		Choices:      choices,
		Timeout:      5,
	})

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.Type != "choice" {
		t.Errorf("Expected response type 'choice', got '%s'", resp.Type)
	}

	if resp.Value != expectedChoice {
		t.Errorf("Expected choice '%s', got '%s'", expectedChoice, resp.Value)
	}
}

func TestNotificationService_Request_Timeout(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	// Create notification with short timeout and don't respond
	_, err := svc.Request(CreateNotificationRequest{
		SessionID:    "test-session",
		Type:         NotificationTypeInfo,
		Title:        "Test Timeout",
		Message:      "This will timeout",
		ResponseType: ResponseTypeAcknowledge,
		Timeout:      1,
	})

	if !errors.Is(err, ErrNotificationTimeout) {
		t.Fatalf("Expected ErrNotificationTimeout, got: %v", err)
	}
}

func TestNotificationService_Request_InvalidTimeout(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	tests := []struct {
		name    string
		timeout int
		wantErr bool
	}{
		{"timeout zero uses default", 0, false},
		{"timeout too high", 301, true},
		{"timeout valid min", 1, false},
		{"timeout valid max", 300, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			
			done := make(chan error, 1)
			go func() {
				_, err := svc.Request(CreateNotificationRequest{
					SessionID:    "test-session",
					Type:         NotificationTypeInfo,
					Title:        "Test",
					Message:      "Test message",
					ResponseType: ResponseTypeAcknowledge,
					Timeout:      tt.timeout,
				})
				done <- err
			}()

			select {
			case err := <-done:
				if tt.wantErr && !errors.Is(err, ErrInvalidTimeout) {
					t.Errorf("Expected ErrInvalidTimeout, got: %v", err)
				}
				if !tt.wantErr && err != nil && !errors.Is(err, ErrNotificationTimeout) {
					t.Errorf("Expected timeout or nil error, got: %v", err)
				}
			case <-time.After(2 * time.Second):
				// For valid timeouts, we expect this to timeout waiting for response
				if tt.wantErr {
					t.Error("Test should have failed with invalid timeout")
				}
			}
		})
	}
}

func TestNotificationService_Request_MissingChoices(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	_, err := svc.Request(CreateNotificationRequest{
		SessionID:    "test-session",
		Type:         NotificationTypeQuestion,
		Title:        "Choose",
		Message:      "No choices provided",
		ResponseType: ResponseTypeChoice,
		Choices:      []string{},
		Timeout:      5,
	})

	if err == nil {
		t.Fatal("Expected error for missing choices, got nil")
	}
}

func TestNotificationService_Respond_NotFound(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	err := svc.Respond("non-existent-id", NotificationResponse{
		ID:   "non-existent-id",
		Type: "acknowledge",
	})

	if !errors.Is(err, ErrNotificationNotFound) {
		t.Fatalf("Expected ErrNotificationNotFound, got: %v", err)
	}
}

func TestNotificationService_PubSub(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	subChan := svc.Subscribe(context.Background())

	notifIDChan := make(chan string, 1)

	go func() {
		notifID := <-notifIDChan
		time.Sleep(100 * time.Millisecond)
		
		err := svc.Respond(notifID, NotificationResponse{
			ID:   notifID,
			Type: "acknowledge",
		})
		if err != nil {
			t.Errorf("Respond failed: %v", err)
		}
	}()

	// Listen for notification event
	go func() {
		event := <-subChan
		if event.Payload.Title != "Test PubSub" {
			t.Errorf("Expected title 'Test PubSub', got '%s'", event.Payload.Title)
		}
		if event.Payload.SessionID != "test-session" {
			t.Errorf("Expected sessionID 'test-session', got '%s'", event.Payload.SessionID)
		}
		notifIDChan <- event.Payload.ID
	}()

	// Create notification request
	_, err := svc.Request(CreateNotificationRequest{
		SessionID:    "test-session",
		Type:         NotificationTypeInfo,
		Title:        "Test PubSub",
		Message:      "Testing publish/subscribe",
		ResponseType: ResponseTypeAcknowledge,
		Timeout:      5,
	})

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

func TestNotificationService_ConcurrentRequests(t *testing.T) {
	t.Helper()
	svc := NewService(&mockSessionService{})

	// Create a shared subscription channel for all requests
	subChan := svc.Subscribe(context.Background())
	
	// Map to store notification IDs and their response channels
	notifMap := sync.Map{}

	// Background goroutine to handle all notification events
	go func() {
		for event := range subChan {
			notifID := event.Payload.ID
			// Store the notification ID in map
			respChan := make(chan bool, 1)
			notifMap.Store(notifID, respChan)
			
			// Respond after delay
			go func(id string, ch chan bool) {
				<-ch // Wait for signal
				time.Sleep(50 * time.Millisecond)
				_ = svc.Respond(id, NotificationResponse{
					ID:   id,
					Type: "acknowledge",
				})
			}(notifID, respChan)
		}
	}()

	numRequests := 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	errChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()

			// Start request
			done := make(chan error, 1)
			go func() {
				_, err := svc.Request(CreateNotificationRequest{
					SessionID:    "test-session",
					Type:         NotificationTypeInfo,
					Title:        "Concurrent test",
					Message:      "Testing concurrent requests",
					ResponseType: ResponseTypeAcknowledge,
					Timeout:      5,
				})
				done <- err
			}()

			// Give some time for notification to be published
			time.Sleep(100 * time.Millisecond)

			// Signal responders
			notifMap.Range(func(key, value any) bool {
				ch := value.(chan bool)
				select {
				case ch <- true:
				default:
				}
				return true
			})

			// Wait for request to complete
			err := <-done
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

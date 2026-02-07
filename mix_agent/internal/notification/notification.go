package notification

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"mix/internal/pubsub"
	"mix/internal/session"

	"github.com/google/uuid"
)

const (
	DefaultTimeout = 60 * time.Second
	MaxTimeout     = 300 * time.Second
)

var (
	ErrNotificationTimeout  = errors.New("notification request timed out")
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidTimeout       = errors.New("timeout must be between 1 and 300 seconds")
)

// NotificationType represents the severity/type of notification
type NotificationType string

const (
	NotificationTypeInfo     NotificationType = "info"
	NotificationTypeWarning  NotificationType = "warning"
	NotificationTypeError    NotificationType = "error"
	NotificationTypeQuestion NotificationType = "question"
)

// NotificationResponseType defines how the user should respond
type NotificationResponseType string

const (
	ResponseTypeAcknowledge NotificationResponseType = "acknowledge"
	ResponseTypeText        NotificationResponseType = "text"
	ResponseTypeChoice      NotificationResponseType = "choice"
)

// CreateNotificationRequest is the input for creating a notification
type CreateNotificationRequest struct {
	SessionID    string                   `json:"sessionId"`
	Type         NotificationType         `json:"type"`
	Title        string                   `json:"title"`
	Message      string                   `json:"message"`
	ResponseType NotificationResponseType `json:"responseType"`
	Choices      []string                 `json:"choices,omitempty"`
	Timeout      int                      `json:"timeout,omitempty"` // Seconds, 0 uses default
}

// NotificationRequest is the full notification sent via SSE
type NotificationRequest struct {
	ID           string                   `json:"id"`
	SessionID    string                   `json:"sessionId"`
	Type         NotificationType         `json:"type"`
	Title        string                   `json:"title"`
	Message      string                   `json:"message"`
	ResponseType NotificationResponseType `json:"responseType"`
	Choices      []string                 `json:"choices,omitempty"`
	Timeout      int                      `json:"timeout"` // Seconds
	CreatedAt    time.Time                `json:"createdAt"`
}

// NotificationResponse is the user's response
type NotificationResponse struct {
	ID    string `json:"id"`
	Type  string `json:"type"`            // "acknowledge" | "text" | "choice"
	Value string `json:"value,omitempty"` // User's text input or selected choice
}

// Service defines the notification service interface
type Service interface {
	pubsub.Suscriber[NotificationRequest]
	Request(opts CreateNotificationRequest) (NotificationResponse, error)
	Respond(notificationID string, response NotificationResponse) error
}

type notificationService struct {
	*pubsub.Broker[NotificationRequest]
	pendingRequests sync.Map // ID → chan NotificationResponse
	sessions        session.Service
}

// Request creates a notification and BLOCKS until user responds or timeout
func (s *notificationService) Request(opts CreateNotificationRequest) (NotificationResponse, error) {
	// Validate timeout
	timeout := DefaultTimeout
	if opts.Timeout > 0 {
		if opts.Timeout < 1 || opts.Timeout > 300 {
			return NotificationResponse{}, fmt.Errorf("%w: got %d seconds", ErrInvalidTimeout, opts.Timeout)
		}
		timeout = time.Duration(opts.Timeout) * time.Second
	}

	// Validate choices for choice type
	if opts.ResponseType == ResponseTypeChoice && len(opts.Choices) == 0 {
		return NotificationResponse{}, errors.New("choices required for choice response type")
	}

	notification := NotificationRequest{
		ID:           uuid.New().String(),
		SessionID:    opts.SessionID,
		Type:         opts.Type,
		Title:        opts.Title,
		Message:      opts.Message,
		ResponseType: opts.ResponseType,
		Choices:      opts.Choices,
		Timeout:      int(timeout.Seconds()),
		CreatedAt:    time.Now(),
	}

	respCh := make(chan NotificationResponse, 1)
	s.pendingRequests.Store(notification.ID, respCh)
	defer s.pendingRequests.Delete(notification.ID)

	// Publish to SSE subscribers
	if err := s.Publish(context.Background(), pubsub.CreatedEvent, notification); err != nil {
		return NotificationResponse{}, fmt.Errorf("failed to publish notification: %w", err)
	}

	// BLOCKING: Wait for response or timeout
	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(timeout):
		return NotificationResponse{}, ErrNotificationTimeout
	}
}

// Respond handles a user's response to a notification
func (s *notificationService) Respond(notificationID string, response NotificationResponse) error {
	respChRaw, ok := s.pendingRequests.Load(notificationID)
	if !ok {
		return ErrNotificationNotFound
	}

	respCh := respChRaw.(chan NotificationResponse)
	select {
	case respCh <- response:
		return nil
	default:
		// Channel already closed or full (should not happen with buffered channel)
		return ErrNotificationNotFound
	}
}

// NewService creates a new notification service
func NewService(sessions session.Service) Service {
	return &notificationService{
		Broker:   pubsub.NewBroker[NotificationRequest](),
		sessions: sessions,
	}
}

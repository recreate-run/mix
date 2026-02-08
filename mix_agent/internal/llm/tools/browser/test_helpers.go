package browser

import (
	"errors"

	browserpkg "mix/internal/browser"
)

// ErrMockFactoryNotConfigured is returned when mockClientFactory is called without test-specific configuration
var ErrMockFactoryNotConfigured = errors.New("mock client factory not configured for this test")

// mockClientFactory creates a mock client factory for tests
func mockClientFactory(sessionID string) (browserpkg.Client, error) {
	// Return error for tests that don't provide their own factory
	// Individual tests must provide their own factory if they need browser client functionality
	return nil, ErrMockFactoryNotConfigured
}

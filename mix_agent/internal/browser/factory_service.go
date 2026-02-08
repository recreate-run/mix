package browser

import (
	"mix/internal/browser/service"
)

// newServiceClient creates a service client
func newServiceClient(connectionManager service.ConnectionManager, sessionID string) Client {
	return service.NewClient(connectionManager, sessionID)
}

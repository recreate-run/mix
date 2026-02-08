package browser

import (
	"mix/internal/browser/tunnel"
)

// newTunnelClient creates a tunnel client
func newTunnelClient(registry interface{}, sessionID string) Client {
	return tunnel.NewClient(registry, sessionID)
}

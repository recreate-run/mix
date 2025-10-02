package storage

import (
	"fmt"
)

// NewProvider creates a new storage provider based on the configuration
func NewProvider(cfg Config) (Provider, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid storage configuration: %w", err)
	}

	// Create provider based on type
	switch cfg.Type {
	case ProviderTypeLocal:
		return NewLocalProvider(cfg)
	case ProviderTypeSupabase:
		return NewSupabaseProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage provider type: %s", cfg.Type)
	}
}

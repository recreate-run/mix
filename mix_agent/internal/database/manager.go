package database

import (
	"context"
	"database/sql"
	"fmt"
)

// Manager manages database connections through providers
type Manager struct {
	provider Provider
	config   Config
}

// NewManager creates a new database manager with the specified configuration
func NewManager(config Config) (*Manager, error) {
	var provider Provider

	switch config.Type {
	case ProviderSQLite:
		provider = NewSQLiteProvider(config.SQLite)
	case ProviderTurso:
		provider = NewTursoProvider(config.Turso)
	default:
		return nil, fmt.Errorf("unsupported database provider type: %s", config.Type)
	}

	return &Manager{
		provider: provider,
		config:   config,
	}, nil
}

// Connect establishes a database connection and runs migrations
func (m *Manager) Connect(ctx context.Context) error {
	// Connect to the database
	if err := m.provider.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := m.provider.RunMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// GetDB returns the underlying sql.DB instance
func (m *Manager) GetDB() *sql.DB {
	return m.provider.GetDB()
}

// Close closes the database connection
func (m *Manager) Close() error {
	return m.provider.Close()
}

// Ping checks if the database connection is alive
func (m *Manager) Ping(ctx context.Context) error {
	return m.provider.Ping(ctx)
}

// BeginTx starts a transaction
func (m *Manager) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return m.provider.BeginTx(ctx, opts)
}

// Type returns the provider type
func (m *Manager) Type() ProviderType {
	return m.provider.Type()
}

// GetProvider returns the underlying provider (for testing or advanced usage)
func (m *Manager) GetProvider() Provider {
	return m.provider
}

package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pressly/goose/v3"
	_ "github.com/tursodatabase/libsql-client-go/libsql"

	"mix/internal/db"
	"mix/internal/logging"
)

// TursoProvider implements Provider interface for Turso databases
type TursoProvider struct {
	config TursoConfig
	db     *sql.DB
}

// NewTursoProvider creates a new Turso provider
func NewTursoProvider(config TursoConfig) *TursoProvider {
	return &TursoProvider{
		config: config,
	}
}

// Connect establishes a connection to the Turso database
func (p *TursoProvider) Connect(ctx context.Context) error {
	if p.config.URL == "" {
		return fmt.Errorf("turso URL is required")
	}
	if p.config.AuthToken == "" {
		return fmt.Errorf("turso auth token is required")
	}

	// Construct connection string with auth token
	connectionString := p.config.URL + "?authToken=" + p.config.AuthToken

	// Connect to Turso using libsql driver
	logging.Info("Connecting to Turso database", "url", p.config.URL)
	db, err := sql.Open("libsql", connectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to Turso: %w", err)
	}

	// Store the connection
	p.db = db

	// Verify connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err = p.db.PingContext(pingCtx); err != nil {
		_ = p.db.Close() // Ignore close error in cleanup path
		p.db = nil
		return fmt.Errorf("failed to ping Turso database: %w", err)
	}

	// Note: No need to set SQLite pragmas for Turso as they are handled by the service
	logging.Debug("Successfully connected to Turso database", "url", p.config.URL)

	return nil
}

// Close closes the database connection
func (p *TursoProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// GetDB returns the underlying sql.DB instance
func (p *TursoProvider) GetDB() *sql.DB {
	return p.db
}

// Ping checks if the database connection is alive
func (p *TursoProvider) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.PingContext(ctx)
}

// BeginTx starts a transaction
func (p *TursoProvider) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return p.db.BeginTx(ctx, opts)
}

// Type returns the provider type
func (p *TursoProvider) Type() ProviderType {
	return ProviderTurso
}

// RunMigrations executes database migrations for Turso
func (p *TursoProvider) RunMigrations(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Set embedded filesystem for migrations (same as SQLite)
	goose.SetBaseFS(db.FS)

	// Turso uses SQLite dialect for migrations
	if err := goose.SetDialect("sqlite3"); err != nil {
		logging.Error("Failed to set dialect", "error", err)
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Run migrations with timeout
	migrationCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := goose.UpContext(migrationCtx, p.db, "migrations"); err != nil {
		logging.Error("Failed to apply migrations to Turso", "error", err)
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	logging.Debug("Successfully applied migrations to Turso database")
	return nil
}

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/pressly/goose/v3"

	"mix/internal/db"
	"mix/internal/logging"
)

// SQLiteProvider implements Provider interface for SQLite databases
type SQLiteProvider struct {
	config SQLiteConfig
	db     *sql.DB
}

// NewSQLiteProvider creates a new SQLite provider
func NewSQLiteProvider(config SQLiteConfig) *SQLiteProvider {
	return &SQLiteProvider{
		config: config,
	}
}

// Connect establishes a connection to the SQLite database
// This wraps the existing logic from internal/db/connect.go exactly
func (p *SQLiteProvider) Connect(ctx context.Context) error {
	if p.config.DataDir == "" {
		return fmt.Errorf("data.dir is not set")
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(p.config.DataDir, 0o700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Construct database path
	filename := p.config.Filename
	if filename == "" {
		filename = "mix.db"
	}
	dbPath := filepath.Join(p.config.DataDir, filename)

	// Open the SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Store the connection
	p.db = db

	// Verify connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err = p.db.PingContext(pingCtx); err != nil {
		_ = p.db.Close() // Ignore close error in cleanup path
		p.db = nil
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set pragmas for better performance (copied from original connect.go)
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA page_size = 4096;",
		"PRAGMA cache_size = -8000;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, pragma := range pragmas {
		pragmaCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if _, err = p.db.ExecContext(pragmaCtx, pragma); err != nil {
			logging.Error("Failed to set pragma", pragma, err)
		} else {
			logging.Debug("Set pragma", "pragma", pragma)
		}
		cancel()
	}

	return nil
}

// Close closes the database connection
func (p *SQLiteProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// GetDB returns the underlying sql.DB instance
func (p *SQLiteProvider) GetDB() *sql.DB {
	return p.db
}

// Ping checks if the database connection is alive
func (p *SQLiteProvider) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.PingContext(ctx)
}

// BeginTx starts a transaction
func (p *SQLiteProvider) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return p.db.BeginTx(ctx, opts)
}

// Type returns the provider type
func (p *SQLiteProvider) Type() ProviderType {
	return ProviderSQLite
}

// RunMigrations executes database migrations
// This wraps the existing migration logic from internal/db/connect.go exactly
func (p *SQLiteProvider) RunMigrations(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Set embedded filesystem for migrations (from original connect.go)
	goose.SetBaseFS(db.FS)

	// Set SQLite dialect
	if err := goose.SetDialect("sqlite3"); err != nil {
		logging.Error("Failed to set dialect", "error", err)
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Run migrations with timeout
	migrationCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := goose.UpContext(migrationCtx, p.db, "migrations"); err != nil {
		logging.Error("Failed to apply migrations", "error", err)
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
